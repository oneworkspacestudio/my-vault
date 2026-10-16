package vault

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oneworkspacestudio/my-vault/internal/platform"
)

// Optional upstream OIDC authentication. Encryption and independent password
// login remain entirely in the compatible client/server implementation.
// This file is outside backups: restored vaults start without an IdP dependency.
type SSOConfig struct {
	Enabled      bool   `json:"enabled"`
	Issuer       string `json:"issuer,omitempty"`
	ClientID     string `json:"clientId,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
}

func (s SSOConfig) Validate() error {
	if !s.Enabled {
		return nil
	}
	u, e := url.Parse(s.Issuer)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || strings.ContainsAny(s.Issuer, "\r\n\\") || strings.HasSuffix(s.Issuer, "/.well-known/openid-configuration") {
		return errors.New("use the exact MyAccounts issuer address, without a discovery suffix")
	}
	local := strings.EqualFold(u.Hostname(), "localhost") || (net.ParseIP(u.Hostname()) != nil && net.ParseIP(u.Hostname()).IsLoopback())
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return errors.New("MyAccounts requires HTTPS; HTTP is allowed only for a loopback identity server")
	}
	if s.ClientID == "" || s.ClientSecret == "" || len(s.ClientID) > 256 || len(s.ClientSecret) > 4096 || strings.ContainsAny(s.ClientID+s.ClientSecret, "\r\n\x00") {
		return errors.New("a valid MyAccounts client identifier and secret are required")
	}
	return nil
}

func loadSSO(root string) (SSOConfig, error) {
	var s SSOConfig
	path := filepath.Join(root, "myvault-sso.json")
	if e := platform.NoReparse(path); e != nil && !os.IsNotExist(e) {
		return s, e
	}
	e := platform.LoadJSON(path, &s)
	if os.IsNotExist(e) {
		return s, nil
	}
	if e != nil {
		return s, errors.New("MyAccounts configuration is invalid; disable the optional integration in local owner tools")
	}
	return s, s.Validate()
}

func (s SSOConfig) checkDiscovery(ctx context.Context) error {
	if e := s.Validate(); e != nil || !s.Enabled {
		return e
	}
	u, _ := url.Parse(s.Issuer)
	client := &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(s.Issuer, "/")+"/.well-known/openid-configuration", nil)
	r, e := client.Do(req)
	if e != nil {
		return errors.New("MyAccounts discovery is unavailable or its HTTPS certificate is not trusted; independent vault sign-in is available")
	}
	defer r.Body.Close()
	var d struct {
		Issuer     string   `json:"issuer"`
		Auth       string   `json:"authorization_endpoint"`
		Token      string   `json:"token_endpoint"`
		Keys       string   `json:"jwks_uri"`
		Algorithms []string `json:"id_token_signing_alg_values_supported"`
	}
	if r.StatusCode != 200 || json.NewDecoder(io.LimitReader(r.Body, 256<<10)).Decode(&d) != nil || d.Issuer != s.Issuer {
		return errors.New("MyAccounts discovery did not match the configured issuer")
	}
	for _, raw := range []string{d.Auth, d.Token, d.Keys} {
		endpoint, e := url.Parse(raw)
		if e != nil || endpoint.Host == "" || endpoint.User != nil || endpoint.Fragment != "" || !sameBrowserOrigin(u, endpoint) {
			return errors.New("MyAccounts discovery endpoints must use the issuer's secure origin")
		}
	}
	for _, alg := range d.Algorithms {
		if alg == "RS256" {
			return nil
		}
	}
	return errors.New("MyAccounts must advertise standard RS256 ID tokens")
}

func (s SSOConfig) engineEnvironment() []string {
	if !s.Enabled {
		return []string{"SSO_ENABLED=false"}
	}
	return []string{"SSO_ENABLED=true", "SSO_ONLY=false", "SSO_SIGNUPS_ALLOWED=false", "SSO_SIGNUPS_MATCH_EMAIL=true", "SSO_ALLOW_UNKNOWN_EMAIL_VERIFICATION=false", "SSO_PKCE=true", "SSO_DEBUG_TOKENS=false", "SSO_AUTH_ONLY_NOT_SESSION=true", "SSO_SCOPES=email profile", "SSO_CLIENT_CACHE_EXPIRATION=0", "SSO_AUTHORITY=" + s.Issuer, "SSO_CLIENT_ID=" + s.ClientID, "SSO_CLIENT_SECRET=" + s.ClientSecret}
}
