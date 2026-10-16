package vault

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/oneworkspacestudio/my-vault/internal/platform"
)

func fixture(t *testing.T) (string, *sql.DB) {
	t.Helper()
	root := t.TempDir()
	if e := platform.ProtectDir(root); e != nil {
		t.Fatal(e)
	}
	c := Config{Schema: 1, Listen: "127.0.0.1:18914", Origin: "https://localhost:18914", Retention: 7}
	if e := c.Save(root); e != nil {
		t.Fatal(e)
	}
	if e := os.MkdirAll(filepath.Join(root, "engine", "attachments", "synthetic"), 0700); e != nil {
		t.Fatal(e)
	}
	db, e := sql.Open("sqlite", filepath.Join(root, "engine", "db.sqlite3"))
	if e != nil {
		t.Fatal(e)
	}
	_, e = db.Exec("PRAGMA journal_mode=WAL; PRAGMA wal_autocheckpoint=0; CREATE TABLE items(id TEXT PRIMARY KEY, content TEXT); INSERT INTO items VALUES('synthetic','sensitive-fixture-marker');")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(root, "engine", "attachments", "synthetic", "file"), []byte("synthetic encrypted attachment bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(root, "engine", "rsa_key.pem"), []byte("synthetic server signing material"), 0600); e != nil {
		t.Fatal(e)
	}
	return root, db
}

func TestIndependentEncryptedColdBackupAndRestore(t *testing.T) {
	root, db := fixture(t)
	defer db.Close()
	file := filepath.Join(t.TempDir(), "vault.age")
	password := "synthetic independent backup passphrase"
	record, e := BackupStopped(root, file, password, "")
	if e != nil {
		t.Fatal(e)
	}
	if !record.Verified {
		t.Fatal("passphrase archive not verified")
	}
	encrypted, e := os.ReadFile(file)
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Contains(encrypted, []byte("sensitive-fixture-marker")) || bytes.Contains(encrypted, []byte("synthetic server")) {
		t.Fatal("unencrypted backup data")
	}
	if _, e = VerifyBackup(root, file, "incorrect independent password", ""); e == nil {
		t.Fatal("wrong passphrase accepted")
	}
	// The restore consumes only the independent archive and password. No source
	// server/issuer/control record is available through the target path.
	destination := filepath.Join(t.TempDir(), "recovered")
	if e = Restore(destination, file, password, "", "127.0.0.1:18915", "https://localhost:18915"); e != nil {
		t.Fatal(e)
	}
	restored, e := sql.Open("sqlite", filepath.Join(destination, "engine", "db.sqlite3"))
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	var item string
	if e = restored.QueryRow("SELECT content FROM items WHERE id='synthetic'").Scan(&item); e != nil || item != "sensitive-fixture-marker" {
		t.Fatal("WAL item did not survive restore", e)
	}
	attachment, e := os.ReadFile(filepath.Join(destination, "engine", "attachments", "synthetic", "file"))
	if e != nil || string(attachment) != "synthetic encrypted attachment bytes" {
		t.Fatal("attachment not restored")
	}
	c, e := Load(destination, "", "")
	if e != nil || c.Origin != "https://localhost:18915" || c.BackupEveryHours != 0 || c.EnrollmentEmail != "" {
		t.Fatal("restore did not isolate new hosting settings", e)
	}
	if e = Restore(destination, file, password, "", "", ""); e == nil {
		t.Fatal("overwrote existing restored installation")
	}
	encrypted[len(encrypted)-10] ^= 0xff
	corrupt := filepath.Join(t.TempDir(), "corrupt.age")
	os.WriteFile(corrupt, encrypted, 0600)
	if _, e = VerifyBackup(root, corrupt, password, ""); e == nil {
		t.Fatal("corrupted archive accepted")
	}
	leftovers, _ := filepath.Glob(filepath.Join(root, ".verify-*"))
	if len(leftovers) != 0 {
		t.Fatal("verification plaintext retained")
	}
}

func TestRecipientBackupNeedsIndependentKey(t *testing.T) {
	root, db := fixture(t)
	defer db.Close()
	key, e := age.GenerateX25519Identity()
	if e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(t.TempDir(), "offsite.age")
	r, e := BackupStopped(root, file, "", key.Recipient().String())
	if e != nil {
		t.Fatal(e)
	}
	if r.Verified {
		t.Fatal("recipient-only encryption claimed decrypt verification")
	}
	stage, e := VerifyBackup(root, file, "", key.String())
	if e != nil {
		t.Fatal(e)
	}
	os.RemoveAll(stage)
	data, _ := os.ReadFile(filepath.Join(root, "myvault.json"))
	if bytes.Contains(data, []byte(key.String())) {
		t.Fatal("private recovery identity persisted")
	}
}

func TestExportFormatDoesNotClaimRecovery(t *testing.T) {
	root := t.TempDir()
	for _, data := range []string{`{"encrypted":false,"passwordProtected":false,"data":"plain"}`, `{"encrypted":true,"passwordProtected":false,"data":"restricted"}`} {
		if RecordExport(root, []byte(data)) == nil {
			t.Fatal("nonportable export accepted")
		}
	}
	if e := RecordExport(root, []byte(`{"encrypted":true,"passwordProtected":true,"data":"synthetic-ciphertext"}`)); e != nil {
		t.Fatal(e)
	}
	s := readStatus(root)
	if s.ExportImported || s.ExportSHA256 == "" {
		t.Fatal("format check claimed successful import")
	}
	saved, _ := os.ReadFile(filepath.Join(root, "recovery.json"))
	if strings.Contains(string(saved), "synthetic-ciphertext") {
		t.Fatal("export content retained")
	}
}
