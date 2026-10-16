package vault

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRemoteConfigurationRequiresTLSAndLoopback(t *testing.T) {
	valid := Config{Schema: 1, Listen: "127.0.0.1:8914", Origin: "https://vault.example.test", Retention: 7}
	if e := valid.Validate(); e != nil {
		t.Fatal(e)
	}
	cases := []Config{valid, valid, valid, valid, valid, valid}
	cases[0].Origin = "http://vault.example.test"
	cases[1].Listen = "0.0.0.0:8914"
	cases[2].Origin = "https://user:password@vault.example.test"
	cases[3].Origin = "https://vault.example.test/#secret"
	cases[4].Retention = 1
	cases[5].BackupEveryHours = 24
	for _, c := range cases {
		if c.Validate() == nil {
			t.Fatal("unsafe configuration accepted")
		}
	}
}
func TestVaultCannotShareCanonicalWorkspaceOrigin(t *testing.T) {
	for _, origin := range []string{"https://WORKSPACE.example.test:443", "https://workspace.example.test"} {
		c := Config{Schema: 1, Listen: "127.0.0.1:8914", Origin: origin, WorkspaceOrigin: "https://workspace.example.test", Retention: 7}
		if c.Validate() == nil {
			t.Fatal("same browser origin accepted", origin)
		}
	}
}

func TestOwnerManagementOriginCapabilityAndNoLeak(t *testing.T) {
	rt := &Runtime{root: t.TempDir(), config: Config{Schema: 1, Listen: "127.0.0.1:8914", Origin: "https://localhost:8914", Retention: 7}, managementURL: "http://127.0.0.1:19001", capability: strings.Repeat("c", 64)}
	handler := rt.managementHandler()
	for _, item := range []struct {
		host, origin, token, method, path string
		want                              int
	}{
		{"evil.example", "", "", "GET", "/api/status", 403},
		{"127.0.0.1:19001", "", "", "GET", "/api/status", 401},
		{"127.0.0.1:19001", "https://evil.example", rt.capability, "POST", "/api/recovery-key", 403},
		{"127.0.0.1:19001", rt.managementURL, rt.capability, "GET", "/api/status", 200},
	} {
		r := httptest.NewRequest(item.method, "http://"+item.host+item.path, strings.NewReader("{}"))
		r.Host = item.host
		r.Header.Set("Origin", item.origin)
		r.Header.Set("Authorization", "Bearer "+item.token)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != item.want {
			t.Fatalf("%s: got %d want %d", item.path, w.Code, item.want)
		}
		if strings.Contains(w.Body.String(), rt.capability) {
			t.Fatal("management capability reflected")
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("private response cacheable")
		}
	}
}

func TestPublicAdminAndClosedSignup(t *testing.T) {
	rt := &Runtime{config: Config{Schema: 1, Listen: "127.0.0.1:8914", Origin: "https://vault.example.test", Retention: 7}, engineURL: "http://127.0.0.1:1"}
	handler := rt.publicHandler()
	for _, path := range []string{"/admin", "/admin/users", "/diagnostics", "/myvault-disabled-config.json"} {
		r := httptest.NewRequest("GET", "https://vault.example.test"+path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 404 {
			t.Fatal("admin endpoint exposed", path, w.Code)
		}
	}
	r := httptest.NewRequest("POST", "https://vault.example.test/identity/accounts/register/finish", strings.NewReader(`{"email":"person@example.test"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatal("public signup open")
	}
}
