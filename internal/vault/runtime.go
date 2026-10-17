package vault

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oneworkspacestudio/my-vault/internal/payload"
	"github.com/oneworkspacestudio/my-vault/internal/platform"
)

type Runtime struct {
	root                                string
	config                              Config
	sso                                 SSOConfig
	ssoWarning                          string
	runtimeFolder, engineSHA, engineURL string
	engineAddress                       atomic.Value
	gate                                sync.RWMutex
	busy                                atomic.Bool
	stopping                            atomic.Bool
	operations                          sync.Mutex
	job                                 *platform.Job
	command                             *exec.Cmd
	held                                *os.File
	done                                chan error
	managementURL, capability           string
	cancel                              context.CancelFunc
}

func randomToken() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}

func Run(ctx context.Context, root string, c Config, open bool, onReady ...func()) error {
	if e := c.Validate(); e != nil {
		return e
	}
	if e := os.MkdirAll(root, 0700); e != nil {
		return e
	}
	if e := platform.ProtectDir(root); e != nil {
		return e
	}
	lock, e := platform.Lock(filepath.Join(root, "runtime.lock"))
	if e != nil {
		return errors.New("My Vault already owns this data folder; use manage to open it")
	}
	defer lock.Close()
	folder, sha, e := payload.Extract(root)
	if e != nil {
		return e
	}
	if e = c.Save(root); e != nil {
		return e
	}
	_ = os.Remove(filepath.Join(root, "myvault-pending.json"))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	rt := &Runtime{root: root, config: c, runtimeFolder: folder, engineSHA: sha, cancel: cancel, capability: randomToken()}
	if e = rt.startEngine(ctx); e != nil {
		return e
	}
	defer func() { rt.gate.Lock(); defer rt.gate.Unlock(); rt.stopEngine() }()
	local, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		return e
	}
	defer local.Close()
	rt.managementURL = "http://" + local.Addr().String()
	manager := &http.Server{Handler: rt.managementHandler(), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 16}
	defer manager.Close()
	go manager.Serve(local)
	if e = platform.SaveJSON(filepath.Join(root, "control.json"), map[string]any{"url": rt.managementURL, "token": rt.capability, "pid": os.Getpid()}); e != nil {
		return e
	}
	defer os.Remove(filepath.Join(root, "control.json"))
	listener, e := net.Listen("tcp4", c.Listen)
	if e != nil {
		return e
	}
	server := &http.Server{Handler: rt.publicHandler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 2 * time.Minute, WriteTimeout: 5 * time.Minute, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 1 << 20}
	cert, key, _, e := localCertificate(root)
	if e != nil {
		listener.Close()
		return e
	}
	secureListener, e := net.Listen("tcp4", c.LocalTLSListen())
	if e != nil {
		listener.Close()
		return fmt.Errorf("local HTTPS port is unavailable: %w", e)
	}
	secureServer := &http.Server{Handler: rt.publicHandler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 2 * time.Minute, WriteTimeout: 5 * time.Minute, IdleTimeout: 90 * time.Second, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
	defer secureServer.Close()
	errs := make(chan error, 2)
	go func() { errs <- server.Serve(listener) }()
	go func() { errs <- secureServer.ServeTLS(secureListener, cert, key) }()
	if open {
		platform.OpenBrowser(rt.managementURL + "/#" + rt.capability)
	}
	go rt.schedule(ctx)
	if len(onReady) > 0 {
		onReady[0]()
	}
	watch := time.NewTicker(time.Second)
	defer watch.Stop()
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case e = <-errs:
			cancel()
			break loop
		case <-watch.C:
			if rt.busy.Load() {
				continue
			}
			rt.gate.RLock()
			ended := false
			if rt.command != nil {
				select {
				case <-rt.done:
					ended = true
				default:
				}
			}
			rt.gate.RUnlock()
			if ended {
				e = errors.New("Vaultwarden exited; encrypted state is retained. Open local recovery or restart My Vault")
				cancel()
				break loop
			}
		}
	}
	rt.stopping.Store(true)
	rt.busy.Store(true)
	shutdown, stop := context.WithTimeout(context.Background(), 30*time.Second)
	defer stop()
	_ = server.Shutdown(shutdown)
	_ = secureServer.Shutdown(shutdown)
	// The native Windows engine can be terminated only after inbound work stops;
	// SQLite WAL recovery and VACUUM INTO protect the later cold snapshot.
	return e
}

func (rt *Runtime) startEngine(ctx context.Context) error {
	// An optional identity configuration must never prevent independent access.
	sso, ssoErr := loadSSO(rt.root)
	rt.ssoWarning = ""
	if ssoErr != nil {
		sso = SSOConfig{}
		rt.ssoWarning = "MyAccounts integration needs repair. Independent vault sign-in remains available."
	}
	rt.sso = sso
	port, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		return e
	}
	address := port.Addr().(*net.TCPAddr).Port
	port.Close()
	engine := filepath.Join(rt.root, "engine")
	if e = os.MkdirAll(engine, 0700); e != nil {
		return e
	}
	rt.engineURL = fmt.Sprintf("http://127.0.0.1:%d", address)
	rt.engineAddress.Store(rt.engineURL)
	held, e := platform.LockedBinary(filepath.Join(rt.runtimeFolder, "vaultwarden.exe"), rt.engineSHA)
	if e != nil {
		return e
	}
	job, e := platform.NewJob()
	if e != nil {
		held.Close()
		return e
	}
	cmd := exec.Command(filepath.Join(rt.runtimeFolder, "vaultwarden.exe"))
	cmd.Dir = engine
	// An explicit environment prevents a parent's ADMIN_TOKEN, DOMAIN, DATABASE_URL,
	// SMTP or SSO settings from silently weakening this independently managed vault.
	for _, key := range []string{"SystemRoot", "WINDIR", "SystemDrive", "PATH", "TEMP", "TMP", "COMSPEC", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "PROGRAMDATA"} {
		if value := os.Getenv(key); value != "" {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	cmd.Env = append(cmd.Env, "DATA_FOLDER="+engine, "DATABASE_URL=sqlite://"+filepath.ToSlash(filepath.Join(engine, "db.sqlite3")), "CONFIG_FILE="+filepath.Join(engine, "myvault-disabled-config.json"), "WEB_VAULT_FOLDER="+filepath.Join(rt.runtimeFolder, "web-vault"), "DOMAIN="+rt.config.Origin, "ROCKET_ADDRESS=127.0.0.1", fmt.Sprintf("ROCKET_PORT=%d", address), "ROCKET_WORKERS=4", "ADMIN_TOKEN=", "DISABLE_ADMIN_TOKEN=false", "SIGNUPS_ALLOWED=false", "INVITATIONS_ALLOWED=false", "ORG_CREATION_USERS=all", "SIGNUPS_VERIFY=false", "IP_HEADER=X-Real-IP", "IP_HEADER_TRUSTED_PROXIES=127.0.0.1/32", "ICON_SERVICE=internal", "DISABLE_ICON_DOWNLOAD=true", "ENABLE_WEBSOCKET=true", "PUSH_ENABLED=false", "SHOW_PASSWORD_HINT=false", "LOG_LEVEL=warn", "EXTENDED_LOGGING=false")
	cmd.Env = append(cmd.Env, sso.engineEnvironment()...)
	// Registration is a local-owner, time-limited email allowlist, not unrestricted
	// signup. Enabling it never changes master-password derivation or encryption.
	if rt.config.EnrollmentEmail != "" && time.Now().Before(rt.config.EnrollmentUntil) {
		cmd.Env = append(cmd.Env, "SIGNUPS_ALLOWED=true")
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if e = job.Start(cmd); e != nil {
		held.Close()
		job.Close()
		return e
	}
	rt.job = job
	rt.command = cmd
	rt.held = held
	done := make(chan error, 1)
	rt.done = done
	go func() { done <- cmd.Wait() }()
	deadline := time.NewTimer(90 * time.Second)
	defer deadline.Stop()
	client := &http.Client{Timeout: 2 * time.Second}
	for {
		r, e := client.Get(rt.engineURL + rt.config.Base() + "/alive")
		if e == nil {
			r.Body.Close()
			if r.StatusCode == 200 {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			rt.stopEngine()
			return ctx.Err()
		case <-rt.done:
			rt.stopEngine()
			return errors.New("Vaultwarden stopped during startup; data was retained")
		case <-deadline.C:
			rt.stopEngine()
			return errors.New("Vaultwarden did not become healthy")
		case <-time.After(300 * time.Millisecond):
		}
	}
}
func (rt *Runtime) stopEngine() {
	if rt.job != nil {
		rt.job.Close()
		rt.job = nil
	}
	if rt.command != nil {
		_ = rt.command.Process.Kill()
		select {
		case <-rt.done:
		case <-time.After(10 * time.Second):
		}
		rt.command = nil
	}
	if rt.held != nil {
		rt.held.Close()
		rt.held = nil
	}
}

func (rt *Runtime) address() string {
	if current := rt.engineAddress.Load(); current != nil {
		return current.(string)
	}
	return rt.engineURL
}
func (rt *Runtime) publicHandler() http.Handler {
	publicConfig := rt.config
	target, _ := url.Parse(rt.address())
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		http.Error(w, "Vault service unavailable. Your data are retained; open My Vault locally for recovery.", 503)
	}
	director := proxy.Director
	external, _ := url.Parse(publicConfig.Origin)
	proxy.Director = func(r *http.Request) {
		director(r)
		active, _ := url.Parse(rt.address())
		r.URL.Scheme = active.Scheme
		r.URL.Host = active.Host
		r.Host = external.Host
		r.Header.Del("Forwarded")
		r.Header.Del("X-Forwarded-For")
		r.Header.Set("X-Forwarded-Proto", external.Scheme)
		r.Header.Set("X-Forwarded-Host", external.Host)
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		// Conservative per-proxy limits: never trust a browser-supplied IP header.
		r.Header.Set("X-Real-IP", ip)
		r.Header.Del("X-MyVault-Proxy")
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
			resp.Header.Set("Cache-Control", "no-store")
		}
		local, _ := resp.Request.Context().Value(localClientKey{}).(string)
		if local == "" || resp.Request.URL.Path != publicConfig.Base()+"/api/config" || resp.StatusCode != 200 {
			return nil
		}
		defer resp.Body.Close()
		var conf map[string]any
		if e := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&conf); e != nil {
			return e
		}
		if env, ok := conf["environment"].(map[string]any); ok {
			for key, value := range env {
				if text, ok := value.(string); ok && strings.HasPrefix(text, publicConfig.Origin) {
					env[key] = local + strings.TrimPrefix(text, publicConfig.Origin)
				}
			}
		}
		body, e := json.Marshal(conf)
		if e != nil {
			return e
		}
		resp.Body = io.NopCloser(bytes.NewReader(body))
		resp.ContentLength = int64(len(body))
		resp.Header.Set("Content-Length", fmt.Sprint(len(body)))
		resp.Header.Set("Cache-Control", "no-store")
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		localTLSHost := "localhost:" + strings.Split(publicConfig.LocalTLSListen(), ":")[1]
		if r.Host != external.Host && r.Host != publicConfig.Listen && r.Host != "localhost:"+strings.Split(publicConfig.Listen, ":")[1] && !(r.TLS != nil && (r.Host == localTLSHost || r.Host == publicConfig.LocalTLSListen())) {
			http.Error(w, "Host rejected", 403)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, publicConfig.Base())
		if path == "" {
			path = "/"
		}
		if pathpkg.Clean(path) != path && pathpkg.Clean(path)+"/" != path {
			http.Error(w, "Noncanonical path rejected", 400)
			return
		}
		localHost := r.Host == publicConfig.Listen || r.Host == "localhost:"+strings.Split(publicConfig.Listen, ":")[1] || (r.TLS != nil && (r.Host == localTLSHost || r.Host == publicConfig.LocalTLSListen()))
		if localHost {
			r = r.WithContext(context.WithValue(r.Context(), localClientKey{}, publicConfig.LocalURL()))
		}
		if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/diagnostics") || strings.HasPrefix(path, "/myvault-disabled-config") {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/healthz" {
			if rt.busy.Load() {
				http.Error(w, "maintenance", 503)
				return
			}
			resp, e := (&http.Client{Timeout: 3 * time.Second}).Get(rt.address() + publicConfig.Base() + "/alive")
			if e != nil {
				http.Error(w, "engine unavailable", 503)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				http.Error(w, "database unavailable", 503)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"status":"ok","independentLogin":true}`)
			return
		}
		if path == "/_myvault/recovery" {
			serveRecovery(w, r, publicConfig.Origin)
			return
		}
		if path == "/_myvault/client-config" && r.Method == http.MethodGet {
			rt.gate.RLock()
			enabled := rt.sso.Enabled
			rt.gate.RUnlock()
			// OIDC browser binding belongs to the dedicated public origin. Local
			// recovery keeps independent login rather than starting a callback
			// on another origin which cannot carry that browser-binding cookie.
			if localHost && !localVaultHostnameForSSO(external.Hostname()) {
				enabled = false
			}
			w.Header().Set("Cache-Control", "no-store")
			// The native engine's canonical SSO identifier selects its own
			// enrollment/password-policy routes. A display label here makes the
			// client request a nonexistent organization policy and log out.
			writeJSON(w, map[string]any{"ssoEnabled": enabled, "ssoPath": "#/sso?identifier=00000000-01DC-01DC-01DC-000000000000", "independentPath": "?login=independent#/login"})
			return
		}
		if localHost && r.TLS == nil && r.Method == "GET" && path == "/" {
			http.Redirect(w, r, publicConfig.LocalURL()+"/", http.StatusTemporaryRedirect)
			return
		}
		if rt.busy.Load() {
			w.Header().Set("Retry-After", "10")
			http.Error(w, "Encrypted backup or maintenance in progress. Retry shortly.", 503)
			return
		}
		// WebSockets are notification-only; engine termination closes them during a
		// cold backup. All ordinary API/file requests drain before the engine stops.
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			proxy.ServeHTTP(w, r)
			return
		}
		rt.gate.RLock()
		defer rt.gate.RUnlock()
		if rt.busy.Load() {
			http.Error(w, "Maintenance", 503)
			return
		}
		if strings.Contains(path, "/accounts/register") && r.Method != "GET" && r.Method != "HEAD" {
			expectedOrigin := "http://" + r.Host
			if r.TLS != nil {
				expectedOrigin = "https://" + r.Host
			}
			if !localHost || r.Header.Get("Origin") != expectedOrigin || rt.config.EnrollmentEmail == "" || time.Now().After(rt.config.EnrollmentUntil) {
				http.Error(w, "Account creation is closed. The local owner can open enrollment in My Vault.", 403)
				return
			}
			data, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 256<<10))
			if e != nil {
				http.Error(w, "Registration request too large", 413)
				return
			}
			var form struct {
				Email string `json:"email"`
			}
			if json.Unmarshal(data, &form) != nil || !strings.EqualFold(form.Email, rt.config.EnrollmentEmail) {
				http.Error(w, "This enrollment window is for the email selected by the local owner.", 403)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(data))
			r.ContentLength = int64(len(data))
		}
		proxy.ServeHTTP(w, r)
	})
}

type localClientKey struct{}

func (rt *Runtime) maintenance(ctx context.Context, action func() error) error {
	if rt.stopping.Load() {
		return errors.New("My Vault is stopping")
	}
	if !rt.busy.CompareAndSwap(false, true) {
		return errors.New("another maintenance operation is running")
	}
	defer rt.busy.Store(false)
	rt.gate.Lock()
	defer rt.gate.Unlock()
	if rt.stopping.Load() {
		return errors.New("My Vault is stopping")
	}
	rt.stopEngine()
	e := action()
	restart := rt.startEngine(ctx)
	if e != nil {
		return e
	}
	return restart
}
func (rt *Runtime) schedule(ctx context.Context) {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			rt.scheduledTick(ctx)
		}
	}
}

func (rt *Runtime) scheduledTick(ctx context.Context) {
	rt.operations.Lock()
	defer rt.operations.Unlock()
	if rt.stopping.Load() {
		return
	}
	rt.gate.RLock()
	c := rt.config
	rt.gate.RUnlock()
	if c.EnrollmentEmail != "" && time.Now().After(c.EnrollmentUntil) {
		_ = rt.maintenance(ctx, func() error {
			rt.config.EnrollmentEmail = ""
			rt.config.EnrollmentUntil = time.Time{}
			return rt.config.Save(rt.root)
		})
	}
	if c.BackupEveryHours == 0 {
		return
	}
	s := readStatus(rt.root)
	if len(s.Backups) > 0 && time.Since(s.Backups[len(s.Backups)-1].Created) < time.Duration(c.BackupEveryHours)*time.Hour {
		return
	}
	e := rt.maintenance(ctx, func() error {
		_, e := BackupStopped(rt.root, filepath.Join(c.BackupFolder, "myvault-"+time.Now().UTC().Format("20060102T150405")+".age"), "", c.BackupRecipient)
		return e
	})
	if e != nil {
		s = readStatus(rt.root)
		s.LastError = "Scheduled backup failed. Existing backups were retained; open the local owner tools."
		_ = writeStatus(rt.root, s)
	}
}
