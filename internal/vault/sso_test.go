package vault

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oneworkspacestudio/my-vault/internal/platform"
)

func TestSSOConfigurationSecurity(t *testing.T) {
	s := SSOConfig{Enabled: true, Issuer: "https://accounts.example.test", ClientID: "workspace-vault", ClientSecret: "synthetic-client-secret"}
	if e := s.Validate(); e != nil {
		t.Fatal(e)
	}
	for _, issuer := range []string{"http://accounts.example.test", "https://user:pass@accounts.example.test", "https://accounts.example.test/#bad", "https://accounts.example.test/.well-known/openid-configuration", "file:///private"} {
		bad := s
		bad.Issuer = issuer
		if bad.Validate() == nil {
			t.Fatal("unsafe issuer accepted", issuer)
		}
	}
	env := strings.Join(s.engineEnvironment(), "\n")
	for _, required := range []string{"SSO_ONLY=false", "SSO_SIGNUPS_ALLOWED=false", "SSO_ALLOW_UNKNOWN_EMAIL_VERIFICATION=false", "SSO_PKCE=true", "SSO_DEBUG_TOKENS=false"} {
		if !strings.Contains(env, required) {
			t.Fatal("SSO lost required protection", required)
		}
	}
	if strings.Contains(strings.Join((SSOConfig{}).engineEnvironment(), "\n"), s.ClientSecret) {
		t.Fatal("disabled SSO retained secret")
	}
}

func TestSSODiscoveryAndPublicConfiguration(t *testing.T) {
	var issuer string
	badEndpoint := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys := issuer + "/keys"
		if badEndpoint {
			keys = "http://external.example.test/keys"
		}
		json.NewEncoder(w).Encode(map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": keys, "id_token_signing_alg_values_supported": []string{"RS256"}})
	}))
	defer server.Close()
	issuer = server.URL
	s := SSOConfig{Enabled: true, Issuer: issuer, ClientID: "workspace-vault", ClientSecret: "synthetic-never-public"}
	if e := s.checkDiscovery(context.Background()); e != nil {
		t.Fatal(e)
	}
	badEndpoint = true
	if s.checkDiscovery(context.Background()) == nil {
		t.Fatal("external discovery endpoint accepted")
	}
	rt := &Runtime{config: Config{Origin: "https://vault.example.test", Listen: "127.0.0.1:8914"}, engineURL: "http://127.0.0.1:1", sso: s}
	w := httptest.NewRecorder()
	rt.publicHandler().ServeHTTP(w, httptest.NewRequest("GET", "https://vault.example.test/_myvault/client-config", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"ssoEnabled":true`) {
		t.Fatal("public SSO entry unavailable")
	}
	if strings.Contains(w.Body.String(), s.ClientSecret) || strings.Contains(w.Body.String(), s.Issuer) || strings.Contains(w.Body.String(), s.ClientID) {
		t.Fatal("SSO private configuration leaked")
	}
}

func TestSSOMissingAndExplicitOptOut(t *testing.T) {
	root := t.TempDir()
	if s, e := loadSSO(root); e != nil || s.Enabled {
		t.Fatal("missing optional identity blocks independent vault")
	}
	if e := platform.SaveJSON(filepath.Join(root, "myvault-sso.json"), SSOConfig{Enabled: false}); e != nil {
		t.Fatal(e)
	}
	if s, e := loadSSO(root); e != nil || s.Enabled {
		t.Fatal("explicit independent mode lost")
	}
}
