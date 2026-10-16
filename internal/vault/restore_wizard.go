package vault

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/oneworkspacestudio/my-vault/internal/platform"
	"github.com/oneworkspacestudio/my-vault/web"
)

// The restore wizard runs without any original account/issuer/server. No vault
// master password is collected, only the independently held backup secret.
func RestoreWizard(ctx context.Context, root, input, listen, origin string) error {
	if listen == "" {
		listen = "127.0.0.1:8914"
	}
	if origin == "" {
		_, port, e := net.SplitHostPort(listen)
		if e != nil {
			return e
		}
		n, e := strconv.Atoi(port)
		if e != nil {
			return e
		}
		origin = "https://localhost:" + strconv.Itoa(n+1)
	}
	if e := (Config{Schema: 1, Listen: listen, Origin: origin, Retention: 7}).Validate(); e != nil {
		return e
	}
	listener, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		return e
	}
	defer listener.Close()
	local := "http://" + listener.Addr().String()
	capability := randomToken()
	done := make(chan struct{}, 1)
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.FS(web.Files)))
	mux.HandleFunc("GET /api/restore-info", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"dataFolder": root, "input": input, "origin": origin})
	})
	mux.HandleFunc("POST /api/restore", func(w http.ResponseWriter, r *http.Request) {
		var p localRequest
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&p) != nil {
			http.Error(w, "Invalid request", 400)
			return
		}
		archive := input
		if p.Path != "" {
			archive = p.Path
		}
		targetOrigin := origin
		if p.Origin != "" {
			targetOrigin = p.Origin
		}
		if e := Restore(root, archive, p.Password, p.Identity, listen, targetOrigin); e != nil {
			http.Error(w, e.Error(), 400)
			return
		}
		writeJSON(w, map[string]string{"message": "Backup restored and integrity verified. Close this window and start My Vault using this data folder. Then check your independent vault login and restored items."})
		go func() {
			time.Sleep(3 * time.Second)
			select {
			case done <- struct{}{}:
			default:
			}
		}()
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'")
		if "http://"+r.Host != local {
			http.Error(w, "Host rejected", 403)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+capability)) != 1 {
				http.Error(w, "Authorization required", 401)
				return
			}
			if r.Method != "GET" && (r.Header.Get("Origin") != local || r.Header.Get("Content-Type") != "application/json") {
				http.Error(w, "Origin or content type rejected", 403)
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	defer server.Close()
	go server.Serve(listener)
	platform.OpenBrowser(local + "/restore.html#" + capability)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
