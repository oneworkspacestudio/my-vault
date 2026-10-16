package vault

import (
	"errors"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/oneworkspacestudio/my-vault/internal/platform"
)

type Config struct {
	Schema           int       `json:"schema"`
	Listen           string    `json:"listen"`
	Origin           string    `json:"origin"`
	WorkspaceOrigin  string    `json:"workspaceOrigin,omitempty"`
	BackupFolder     string    `json:"backupFolder,omitempty"`
	BackupRecipient  string    `json:"backupRecipient,omitempty"`
	BackupEveryHours int       `json:"backupEveryHours,omitempty"`
	Retention        int       `json:"retention"`
	EnrollmentEmail  string    `json:"enrollmentEmail,omitempty"`
	EnrollmentUntil  time.Time `json:"enrollmentUntil,omitempty"`
}

func Load(root, listen, origin string) (Config, error) {
	c := Config{Schema: 1, Listen: "127.0.0.1:8914", Origin: "https://localhost:8915", Retention: 7}
	path := filepath.Join(root, "myvault.json")
	if e := platform.LoadJSON(path, &c); e != nil && !os.IsNotExist(e) {
		return c, e
	}
	if e := platform.LoadJSON(filepath.Join(root, "myvault-pending.json"), &c); e != nil && !os.IsNotExist(e) {
		return c, e
	}
	if listen != "" {
		c.Listen = listen
		if _, e := os.Stat(path); os.IsNotExist(e) {
			c.Origin = c.LocalURL()
		}
	}
	if origin != "" {
		c.Origin = strings.TrimRight(origin, "/")
	}
	if managed := os.Getenv("MYVAULT_WORKSPACE_ORIGIN"); managed != "" {
		c.WorkspaceOrigin = managed
	}
	if c.WorkspaceOrigin == "" {
		var parent struct {
			Origin string `json:"origin"`
		}
		if platform.LoadJSON(filepath.Join(filepath.Dir(root), "workspace.json"), &parent) == nil {
			c.WorkspaceOrigin = parent.Origin
		}
	}
	return c, c.Validate()
}
func (c Config) Validate() error {
	host, port, e := net.SplitHostPort(c.Listen)
	if e != nil || net.ParseIP(host) == nil || net.ParseIP(host).To4() == nil || !net.ParseIP(host).IsLoopback() || port == "0" {
		return errors.New("vault must listen on a fixed loopback address")
	}
	n, pe := strconv.Atoi(port)
	if pe != nil || n < 1 || n > 65534 {
		return errors.New("vault port must leave the following port available for local HTTPS")
	}
	u, e := url.Parse(c.Origin)
	if e != nil || u.User != nil || u.Host == "" || u.RawQuery != "" || u.Fragment != "" || strings.ContainsAny(u.Path, "\\\r\n") || strings.Contains(u.Path, "..") {
		return errors.New("invalid public vault URL")
	}
	if u.Scheme != "https" {
		return errors.New("vault clients require HTTPS, including localhost; use the secure local vault address")
	}
	for _, letter := range u.Hostname() {
		if letter > 127 {
			return errors.New("use an ASCII or punycode vault hostname")
		}
	}
	if workspace, e := url.Parse(c.WorkspaceOrigin); e == nil && workspace.Host != "" && sameBrowserOrigin(workspace, u) {
		return errors.New("use a separate vault origin, not the shared Workspace address")
	}
	if c.Schema != 1 || c.Retention < 2 || c.Retention > 365 {
		return errors.New("unsupported settings or retention; retain between 2 and 365 verified backups")
	}
	if c.BackupRecipient != "" {
		if _, e = age.ParseX25519Recipient(c.BackupRecipient); e != nil {
			return errors.New("invalid age backup recipient")
		}
	}
	if c.BackupEveryHours != 0 && (c.BackupEveryHours < 1 || c.BackupEveryHours > 168 || c.BackupRecipient == "" || !filepath.IsAbs(c.BackupFolder)) {
		return errors.New("scheduled backups need an independent age recipient, absolute folder and 1–168 hour interval")
	}
	return nil
}
func sameBrowserOrigin(a, b *url.URL) bool {
	port := func(u *url.URL) string {
		if u.Port() != "" {
			return u.Port()
		}
		if strings.EqualFold(u.Scheme, "https") {
			return "443"
		}
		return "80"
	}
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(strings.TrimSuffix(a.Hostname(), "."), strings.TrimSuffix(b.Hostname(), ".")) && port(a) == port(b)
}
func (c Config) Base() string { u, _ := url.Parse(c.Origin); return strings.TrimRight(u.Path, "/") }
func (c Config) Save(root string) error {
	if e := c.Validate(); e != nil {
		return e
	}
	return platform.SaveJSON(filepath.Join(root, "myvault.json"), c)
}
