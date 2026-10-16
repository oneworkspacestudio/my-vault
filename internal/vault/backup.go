package vault

import (
	"archive/zip"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/oneworkspacestudio/my-vault/internal/platform"
	_ "modernc.org/sqlite"
)

type BackupRecord struct {
	File         string    `json:"file"`
	Created      time.Time `json:"created"`
	SHA256       string    `json:"sha256"`
	Verified     bool      `json:"verified"`
	Verification string    `json:"verification"`
}
type RecoveryStatus struct {
	Backups        []BackupRecord `json:"backups"`
	ExportDate     time.Time      `json:"exportDate,omitempty"`
	ExportSHA256   string         `json:"exportSha256,omitempty"`
	ExportImported bool           `json:"exportImported"`
	LastError      string         `json:"lastError,omitempty"`
}
type FileDigest struct {
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}
type BackupManifest struct {
	Schema  int                   `json:"schema"`
	Created time.Time             `json:"created"`
	Files   map[string]FileDigest `json:"files"`
	Engine  string                `json:"engine"`
}

func readStatus(root string) RecoveryStatus {
	var s RecoveryStatus
	_ = platform.LoadJSON(filepath.Join(root, "recovery.json"), &s)
	return s
}
func writeStatus(root string, s RecoveryStatus) error {
	return platform.SaveJSON(filepath.Join(root, "recovery.json"), s)
}
func digestFile(path string) (FileDigest, error) {
	f, e := os.Open(path)
	if e != nil {
		return FileDigest{}, e
	}
	defer f.Close()
	h := sha256.New()
	n, e := io.Copy(h, f)
	return FileDigest{n, hex.EncodeToString(h.Sum(nil))}, e
}

func sqliteSnapshot(source, target string) error {
	// Called only with the engine stopped. SQLite reads/replays the committed WAL;
	// VACUUM INTO produces a standalone database, not a live db-only file copy.
	db, e := sql.Open("sqlite", source)
	if e != nil {
		return e
	}
	defer db.Close()
	if _, e = db.Exec("PRAGMA busy_timeout=5000"); e != nil {
		return e
	}
	var result string
	if e = db.QueryRow("PRAGMA integrity_check").Scan(&result); e != nil || result != "ok" {
		return errors.New("source SQLite integrity check failed; original data retained")
	}
	_, e = db.Exec("VACUUM INTO '" + strings.ReplaceAll(target, "'", "''") + "'")
	return e
}
func checkDatabase(path string) error {
	db, e := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if e != nil {
		return e
	}
	defer db.Close()
	var result string
	if e = db.QueryRow("PRAGMA integrity_check").Scan(&result); e != nil || result != "ok" {
		return errors.New("backup SQLite integrity verification failed")
	}
	return nil
}

// BackupStopped requires exclusive installation ownership and a stopped engine.
// recipient supports offsite scheduled backups without retaining a decryption key.
func BackupStopped(root, output, password, recipient string) (record BackupRecord, err error) {
	if !filepath.IsAbs(output) || !strings.HasSuffix(strings.ToLower(output), ".age") {
		return record, errors.New("choose a new absolute .age backup filename")
	}
	if recipient == "" && len(password) < 16 {
		return record, errors.New("use an independent backup passphrase of at least 16 characters")
	}
	var r age.Recipient
	if recipient != "" {
		r, err = age.ParseX25519Recipient(recipient)
	} else {
		r, err = age.NewScryptRecipient(password)
	}
	if err != nil || r == nil {
		return record, errors.New("a backup passphrase or valid age recipient is required")
	}
	stage, err := os.MkdirTemp(root, ".snapshot-")
	if err != nil {
		return record, err
	}
	defer os.RemoveAll(stage)
	if err = platform.ProtectDir(stage); err != nil {
		return record, err
	}
	source := filepath.Join(root, "engine")
	if _, err = os.Stat(filepath.Join(source, "db.sqlite3")); err != nil {
		return record, errors.New("there is no initialized vault database to back up")
	}
	if err = os.MkdirAll(filepath.Join(stage, "engine"), 0700); err != nil {
		return record, err
	}
	if err = sqliteSnapshot(filepath.Join(source, "db.sqlite3"), filepath.Join(stage, "engine", "db.sqlite3")); err != nil {
		return record, err
	}
	err = filepath.WalkDir(source, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if e = platform.NoReparse(path); e != nil {
			return e
		}
		rel, _ := filepath.Rel(source, path)
		if d.IsDir() {
			if rel == "icon_cache" || rel == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == "db.sqlite3" || strings.HasPrefix(rel, "db.sqlite3-") || strings.HasSuffix(rel, ".log") {
			return nil
		}
		return platform.CopyFile(path, filepath.Join(stage, "engine", rel))
	})
	if err != nil {
		return record, err
	}
	if err = platform.CopyFile(filepath.Join(root, "myvault.json"), filepath.Join(stage, "myvault.json")); err != nil {
		return record, err
	}
	m := BackupManifest{Schema: 1, Created: time.Now().UTC(), Files: map[string]FileDigest{}, Engine: "1.37.4"}
	err = filepath.WalkDir(stage, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(stage, path)
		if rel == ".owner-sid" {
			return nil
		}
		fd, e := digestFile(path)
		if e == nil {
			m.Files[filepath.ToSlash(rel)] = fd
		}
		return e
	})
	if err != nil {
		return record, err
	}
	if err = platform.SaveJSON(filepath.Join(stage, "manifest.json"), m); err != nil {
		return record, err
	}
	if err = os.MkdirAll(filepath.Dir(output), 0700); err != nil {
		return record, err
	}
	file, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return record, err
	}
	complete := false
	defer func() {
		file.Close()
		if !complete {
			_ = os.Remove(output)
		}
	}()
	encrypted, err := age.Encrypt(file, r)
	if err != nil {
		return record, err
	}
	archive := zip.NewWriter(encrypted)
	err = filepath.WalkDir(stage, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(stage, path)
		if rel == ".owner-sid" {
			return nil
		}
		w, e := archive.Create(filepath.ToSlash(rel))
		if e != nil {
			return e
		}
		in, e := os.Open(path)
		if e != nil {
			return e
		}
		defer in.Close()
		_, e = io.Copy(w, in)
		return e
	})
	if e := archive.Close(); err == nil {
		err = e
	}
	if e := encrypted.Close(); err == nil {
		err = e
	}
	if e := file.Sync(); err == nil {
		err = e
	}
	if e := file.Close(); err == nil {
		err = e
	}
	if err != nil {
		return record, err
	}
	complete = true
	fd, err := digestFile(output)
	if err != nil {
		return record, err
	}
	record = BackupRecord{File: output, Created: m.Created, SHA256: fd.SHA256, Verification: "encrypted; independent-key verification required"}
	if recipient == "" {
		checked, e := VerifyBackup(root, output, password, "")
		if e != nil {
			return record, e
		}
		os.RemoveAll(checked)
		record.Verified = true
		record.Verification = "decrypted, every file hash checked and SQLite integrity_check passed"
	}
	status := readStatus(root)
	status.Backups = append(status.Backups, record)
	status.LastError = ""
	return record, writeStatus(root, status)
}

// VerifyBackup returns a private, verified extraction. The caller removes it.
// The .zip plaintext is never stored on removable/offsite output storage.
func VerifyBackup(root, input, password, identity string) (stage string, err error) {
	var id age.Identity
	if identity != "" {
		id, err = age.ParseX25519Identity(strings.TrimSpace(identity))
	} else {
		id, err = age.NewScryptIdentity(password)
	}
	if err != nil {
		return "", errors.New("invalid independent backup key or passphrase")
	}
	in, err := os.Open(input)
	if err != nil {
		return "", err
	}
	defer in.Close()
	decrypted, err := age.Decrypt(in, id)
	if err != nil {
		return "", errors.New("backup could not be decrypted with that key or passphrase")
	}
	stage, err = os.MkdirTemp(root, ".verify-")
	if err != nil {
		return "", err
	}
	success := false
	cleanupStage := stage
	defer func() {
		if !success {
			os.RemoveAll(cleanupStage)
		}
	}()
	if err = platform.ProtectDir(stage); err != nil {
		return "", err
	}
	temp, err := os.CreateTemp(stage, "archive-*.zip")
	if err != nil {
		return "", err
	}
	n, err := io.Copy(temp, io.LimitReader(decrypted, 64<<30+1))
	if e := temp.Close(); err == nil {
		err = e
	}
	if err != nil || n > 64<<30 {
		return "", errors.New("invalid, truncated or oversized encrypted backup")
	}
	z, err := zip.OpenReader(temp.Name())
	if err != nil {
		return "", errors.New("invalid backup archive")
	}
	defer z.Close()
	var manifest BackupManifest
	for _, f := range z.File {
		if f.Name == "manifest.json" {
			if f.UncompressedSize64 > 16<<20 {
				return "", errors.New("oversized manifest")
			}
			r, e := f.Open()
			if e != nil {
				return "", e
			}
			err = json.NewDecoder(io.LimitReader(r, 16<<20)).Decode(&manifest)
			r.Close()
			if err != nil {
				return "", err
			}
		}
	}
	if manifest.Schema != 1 || len(manifest.Files) < 2 || len(manifest.Files) > 100000 || len(z.File) != len(manifest.Files)+1 {
		return "", errors.New("invalid backup manifest")
	}
	var total int64
	seen := map[string]bool{}
	for _, f := range z.File {
		if f.Name == "manifest.json" {
			continue
		}
		fd, ok := manifest.Files[f.Name]
		key := strings.ToLower(f.Name)
		if !ok || seen[key] || fd.Size < 0 || fd.Size > 2<<30 || uint64(fd.Size) != f.UncompressedSize64 || !(f.Name == "myvault.json" || strings.HasPrefix(f.Name, "engine/")) {
			return "", errors.New("unexpected, duplicate or oversized archive member")
		}
		seen[key] = true
		total += fd.Size
		if total > 64<<30 {
			return "", errors.New("backup extraction limit exceeded")
		}
		target, e := platform.Inside(stage, f.Name)
		if e != nil {
			return "", e
		}
		if e = os.MkdirAll(filepath.Dir(target), 0700); e != nil {
			return "", e
		}
		r, e := f.Open()
		if e != nil {
			return "", e
		}
		out, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			r.Close()
			return "", e
		}
		h := sha256.New()
		n, e := io.Copy(io.MultiWriter(out, h), io.LimitReader(r, fd.Size+1))
		r.Close()
		ce := out.Close()
		if e != nil || ce != nil || n != fd.Size || hex.EncodeToString(h.Sum(nil)) != fd.SHA256 {
			return "", errors.New("backup file integrity check failed")
		}
	}
	if err = checkDatabase(filepath.Join(stage, "engine", "db.sqlite3")); err != nil {
		return "", err
	}
	z.Close()
	if err = os.Remove(temp.Name()); err != nil {
		return "", err
	}
	success = true
	return stage, nil
}

func Restore(root, input, password, identity, listen, origin string) error {
	if listen == "" {
		listen = "127.0.0.1:8914"
	}
	if origin == "" {
		_, port, e := net.SplitHostPort(listen)
		if e != nil {
			return e
		}
		number, _ := strconv.Atoi(port)
		origin = "https://localhost:" + strconv.Itoa(number+1)
	}
	if e := platform.RequireNewDir(root); e != nil {
		return errors.New("restore needs an empty new data folder; existing vaults are never overwritten")
	}
	if e := platform.ProtectDir(root); e != nil {
		return e
	}
	lock, e := platform.Lock(filepath.Join(root, "runtime.lock"))
	if e != nil {
		return e
	}
	defer lock.Close()
	stage, e := VerifyBackup(root, input, password, identity)
	if e != nil {
		return e
	}
	defer os.RemoveAll(stage)
	c, e := Load(stage, listen, origin)
	if e != nil {
		return e
	}
	c.EnrollmentEmail = ""
	c.EnrollmentUntil = time.Time{}
	c.BackupEveryHours = 0
	if e = os.Rename(filepath.Join(stage, "engine"), filepath.Join(root, "engine")); e != nil {
		return e
	}
	if e = c.Save(root); e != nil {
		return e
	}
	return platform.SaveJSON(filepath.Join(root, "restored.json"), map[string]any{"at": time.Now().UTC(), "integrityVerified": true, "accountUnlockVerified": false, "source": "independent age-encrypted backup"})
}

func RotateVerified(root string, keep int) error {
	// Explicit only: never remove unverified archives or the last two known-good
	// records. Refuse altered paths/files instead of guessing they are disposable.
	s := readStatus(root)
	verified := 0
	for _, r := range s.Backups {
		if r.Verified {
			fd, e := digestFile(r.File)
			if e != nil || fd.SHA256 != r.SHA256 {
				return errors.New("a verified backup changed or is unavailable; nothing was deleted")
			}
			verified++
		}
	}
	if keep < 2 {
		return errors.New("retain at least two verified backups")
	}
	removed := map[string]bool{}
	for _, r := range s.Backups {
		if !r.Verified || verified <= keep {
			continue
		}
		fd, e := digestFile(r.File)
		if e != nil || fd.SHA256 != r.SHA256 {
			return errors.New("backup changed or unavailable; retention stopped")
		}
		if e = os.Remove(r.File); e != nil {
			return e
		}
		removed[r.File] = true
		verified--
	}
	kept := s.Backups[:0]
	for _, r := range s.Backups {
		if !removed[r.File] {
			kept = append(kept, r)
		}
	}
	s.Backups = kept
	return writeStatus(root, s)
}

func RecordExport(root string, data []byte) error {
	var e struct {
		Encrypted         bool   `json:"encrypted"`
		PasswordProtected bool   `json:"passwordProtected"`
		Data              string `json:"data"`
	}
	if json.Unmarshal(data, &e) != nil || !e.Encrypted || !e.PasswordProtected || e.Data == "" {
		return errors.New("select a password-protected encrypted Bitwarden JSON export; account-restricted and plaintext exports are not accepted")
	}
	sum := sha256.Sum256(data)
	s := readStatus(root)
	s.ExportSHA256 = fmt.Sprintf("%x", sum)
	s.ExportDate = time.Now().UTC()
	s.ExportImported = false
	return writeStatus(root, s)
}
