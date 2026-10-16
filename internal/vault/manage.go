package vault

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/oneworkspacestudio/my-vault/internal/platform"
	"github.com/oneworkspacestudio/my-vault/web"
)

type localRequest struct {
	Password  string `json:"password"`
	Confirm   string `json:"confirm"`
	Identity  string `json:"identity"`
	Recipient string `json:"recipient"`
	Path      string `json:"path"`
	Email     string `json:"email"`
	Origin    string `json:"origin"`
	Hours     int    `json:"hours"`
	Keep      int    `json:"keep"`
}

func (rt *Runtime) managementHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.FS(web.Files)))
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		rt.gate.RLock()
		defer rt.gate.RUnlock()
		status := readStatus(rt.root)
		writeJSON(w, map[string]any{"recovery": status, "settings": rt.config, "vaultURL": rt.config.Origin, "localVaultURL": rt.config.LocalURL() + "/", "busy": rt.busy.Load(), "dataFolder": rt.root})
	})
	mux.HandleFunc("POST /api/{action}", func(w http.ResponseWriter, r *http.Request) {
		rt.operations.Lock()
		defer rt.operations.Unlock()
		if rt.stopping.Load() {
			http.Error(w, "My Vault is stopping", 503)
			return
		}
		var input localRequest
		if e := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&input); e != nil {
			http.Error(w, "Invalid request", 400)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		var result any = map[string]bool{"ok": true}
		var e error
		switch r.PathValue("action") {
		case "trust-local":
			if input.Confirm != "trust-localhost" {
				e = errors.New("explicit local certificate trust confirmation is required")
				break
			}
			e = platform.TrustLocalCertificate(filepath.Join(rt.root, "local-tls", "localhost.cer"))
			if e == nil {
				result = map[string]string{"message": "Local certificate trusted for this Windows account. Open the secure local vault."}
			}
		case "backup":
			if input.Password != input.Confirm {
				e = errors.New("backup passphrases must match")
				break
			}
			path := input.Path
			if path == "" {
				path = filepath.Join(rt.root, "backups", "myvault-"+time.Now().UTC().Format("20060102T150405")+".age")
			}
			e = rt.maintenance(ctx, func() error {
				record, err := BackupStopped(rt.root, path, input.Password, input.Recipient)
				result = record
				return err
			})
		case "verify":
			if !rt.busy.CompareAndSwap(false, true) {
				e = errors.New("another operation is running")
				break
			}
			defer rt.busy.Store(false)
			var stage string
			stage, e = VerifyBackup(rt.root, input.Path, input.Password, input.Identity)
			if e == nil {
				os.RemoveAll(stage)
				s := readStatus(rt.root)
				fd, err := digestFile(input.Path)
				if err != nil {
					e = err
					break
				}
				for i := range s.Backups {
					if s.Backups[i].SHA256 == fd.SHA256 {
						s.Backups[i].Verified = true
						s.Backups[i].Verification = "decrypted; manifest and SQLite integrity verified"
					}
				}
				e = writeStatus(rt.root, s)
			}
		case "recovery-key":
			var key *age.X25519Identity
			key, e = age.GenerateX25519Identity()
			if e == nil {
				result = map[string]string{"identity": key.String(), "recipient": key.Recipient().String(), "warning": "Save this key outside the server. My Vault does not retain it."}
			}
		case "enroll":
			address, err := mail.ParseAddress(input.Email)
			if err != nil || address.Address != input.Email || strings.ContainsAny(input.Email, "\r\n,") {
				e = errors.New("enter one valid email address")
				break
			}
			e = rt.maintenance(ctx, func() error {
				rt.config.EnrollmentEmail = strings.ToLower(input.Email)
				rt.config.EnrollmentUntil = time.Now().Add(10 * time.Minute)
				return rt.config.Save(rt.root)
			})
		case "close-enrollment":
			e = rt.maintenance(ctx, func() error {
				rt.config.EnrollmentEmail = ""
				rt.config.EnrollmentUntil = time.Time{}
				return rt.config.Save(rt.root)
			})
		case "schedule":
			rt.gate.Lock()
			defer rt.gate.Unlock()
			next := rt.config
			next.BackupFolder = input.Path
			next.BackupRecipient = input.Recipient
			next.BackupEveryHours = input.Hours
			next.Retention = input.Keep
			e = next.Validate()
			if e == nil {
				e = next.Save(rt.root)
			}
			if e == nil {
				rt.config = next
			}
		case "origin":
			rt.gate.RLock()
			next := rt.config
			rt.gate.RUnlock()
			next.Origin = strings.TrimRight(input.Origin, "/")
			e = next.Validate()
			if e == nil {
				e = platform.SaveJSON(filepath.Join(rt.root, "myvault-pending.json"), next)
			}
			if e == nil {
				result = map[string]string{"message": "HTTPS address saved. Configure your proxy, then restart My Vault to apply it. Keep direct local access for recovery."}
			}
		case "rotate":
			if !rt.busy.CompareAndSwap(false, true) {
				e = errors.New("another operation is running")
				break
			}
			defer rt.busy.Store(false)
			e = RotateVerified(rt.root, input.Keep)
		case "export":
			var data []byte
			var info os.FileInfo
			info, e = os.Stat(input.Path)
			if e == nil && (info.Size() > 64<<20 || !info.Mode().IsRegular()) {
				e = errors.New("select a regular encrypted JSON file of at most 64 MiB")
			}
			if e == nil {
				data, e = os.ReadFile(input.Path)
			}
			if e == nil {
				e = RecordExport(rt.root, data)
			}
		case "stop":
			go func() { time.Sleep(300 * time.Millisecond); rt.cancel() }()
		default:
			e = errors.New("unknown operation")
		}
		// Secrets stay in this request's memory only. No request bodies or arbitrary
		// upstream payloads are written to logs, events or telemetry.
		if e != nil {
			http.Error(w, e.Error(), 400)
			return
		}
		writeJSON(w, result)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		if "http://"+r.Host != rt.managementURL {
			http.Error(w, "Host rejected", 403)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+rt.capability)) != 1 {
				http.Error(w, "Open MyVault.exe manage on this computer to authorize local recovery tools.", 401)
				return
			}
			if r.Method != "GET" && r.Header.Get("Origin") != rt.managementURL {
				http.Error(w, "Origin rejected", 403)
				return
			}
			if r.Method != "GET" && r.Header.Get("Content-Type") != "application/json" {
				http.Error(w, "JSON required", 415)
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}
func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value)
}

func Control(root, action string) error {
	var c struct {
		URL   string `json:"url"`
		Token string `json:"token"`
	}
	if e := platform.LoadJSON(filepath.Join(root, "control.json"), &c); e != nil {
		return errors.New("My Vault is not running from this data folder; start it first")
	}
	if !strings.HasPrefix(c.URL, "http://127.0.0.1:") || len(c.Token) != 64 {
		return errors.New("invalid local control record")
	}
	if action == "manage" {
		platform.OpenBrowser(c.URL + "/#" + c.Token)
		return nil
	}
	req, e := http.NewRequest("POST", c.URL+"/api/"+action, strings.NewReader("{}"))
	if e != nil {
		return e
	}
	req.Header.Set("Origin", c.URL)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, e := (&http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("local owner operation refused")
	}
	return nil
}

var recoveryPage = template.Must(template.New("recovery").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>My Vault · Recovery &amp; Portability</title><link rel="stylesheet" href="{{.Base}}/my-vault.css"><main class="myvault-guide"><h1>Recovery &amp; Portability</h1><p>Your vault remains yours when the server is gone.</p><h2>Three different safety nets</h2><ul><li><b>Server backup:</b> an age-encrypted copy of accounts, encrypted vault database, attachments and server keys. Restore with MyVault.exe on a fresh computer and your independent backup password/key.</li><li><b>Portable vault export:</b> choose Tools → Export vault → encrypted JSON → Password protected in the compatible client. Use a separate strong export password. Account restricted exports alone cannot recover a lost account.</li><li><b>Offline client cache:</b> first sign in and sync, then lock (do not log out). Disconnect the device and test unlocking. Logging out, reinstalling, changing devices or losing app data can remove this cache.</li></ul><h2>Set up before you need it</h2><ol><li>Install an official Bitwarden extension, desktop or mobile client and configure your self-hosted URL.</li><li>Sync, lock and test offline unlocking on two devices. Do not assume offline editing/sync.</li><li>Enable compatible MFA and keep its recovery code outside this server.</li><li>Make a password-protected export; import it into a fresh independent account and inspect the restored items.</li><li>Make and verify an encrypted full server backup on independent storage.</li><li>Keep your master password, backup/export passwords or age identity, MFA recovery codes and restore instructions independently.</li></ol><h2>Restore without One Workspace or MyAccounts</h2><ol><li>Obtain the verified MyVault.exe and a copy of your .age backup.</li><li>On a clean Windows PC, run <code>MyVault.exe restore --data C:\MyVault-Recovered --input X:\myvault.age --origin http://localhost:8914</code>.</li><li>Enter the independent backup key/passphrase in the local restore wizard.</li><li>Run <code>MyVault.exe run --data C:\MyVault-Recovered</code> and log in with your vault email/master password and MFA.</li><li>For a portable export instead, create a fresh independent compatible account, choose Tools → Import data → Bitwarden JSON and enter the export password.</li><li>Check item counts, notes and login values; retrieve separately backed-up attachments and organisation items.</li></ol><h2>Know the limits</h2><p>JSON export is not a full server backup. Attachments, organisation-owned items, Sends, passkeys, SSH keys and authenticator data vary by format and client version; see the shipped acceptance and client guides. An exported file's presence is not proof it will import. The web vault is not an offline client.</p><h2>Backup status and owner tools</h2><p>Run <code>MyVault.exe manage --data YOUR_PRIVATE_FOLDER</code> on the server to see last encrypted backup, integrity verification, portable-export record, retention and restore tools. This page exposes no private backup paths or account status.</p><p><a href="{{.Origin}}/">Return to My Vault →</a></p><p>Compatible clients perform vault decryption. MyAccounts and this companion cannot recover an unknown master password from encrypted vault data.</p></main></html>`))

func serveRecovery(w http.ResponseWriter, r *http.Request, origin string) {
	u := strings.TrimRight(origin, "/")
	base := ""
	if at := strings.Index(strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://"), "/"); at >= 0 {
		raw := strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
		base = raw[at:]
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = recoveryPage.Execute(w, map[string]string{"Origin": u, "Base": base})
}

func WaitStopped(root string) error {
	for i := 0; i < 100; i++ {
		lock, e := platform.Lock(filepath.Join(root, "runtime.lock"))
		if e == nil {
			lock.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("My Vault is still stopping; data remain at the configured location")
}
