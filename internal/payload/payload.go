package payload

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/oneworkspacestudio/my-vault/internal/platform"
)

//go:embed runtime.zip
var archive []byte

//go:embed manifest.json
var manifest []byte

type File struct {
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}
type Manifest struct {
	Schema int             `json:"schema"`
	Engine string          `json:"engine"`
	Web    string          `json:"web"`
	Files  map[string]File `json:"files"`
}

func Info() (Manifest, error) {
	var m Manifest
	e := json.Unmarshal(manifest, &m)
	if e == nil && (m.Schema != 1 || len(m.Files) < 2) {
		e = errors.New("invalid embedded runtime manifest")
	}
	return m, e
}

func Extract(root string) (string, string, error) {
	m, e := Info()
	if e != nil {
		return "", "", e
	}
	sum := sha256.Sum256(manifest)
	folder := filepath.Join(root, "runtime", hex.EncodeToString(sum[:8]))
	valid := func() error {
		for rel, expected := range m.Files {
			p, e := platform.Inside(folder, rel)
			if e != nil {
				return e
			}
			if e = platform.NoReparse(p); e != nil {
				return e
			}
			f, e := os.Open(p)
			if e != nil {
				return e
			}
			h := sha256.New()
			n, e := io.Copy(h, f)
			f.Close()
			if e != nil || n != expected.Size || hex.EncodeToString(h.Sum(nil)) != expected.SHA256 {
				return errors.New("runtime checksum mismatch; repair using the original verified MyVault.exe")
			}
		}
		return nil
	}
	if _, e = os.Stat(folder); e == nil {
		if e = valid(); e != nil {
			return "", "", e
		}
		return folder, m.Files["vaultwarden.exe"].SHA256, nil
	}
	if e = os.MkdirAll(filepath.Dir(folder), 0700); e != nil {
		return "", "", e
	}
	stage, e := os.MkdirTemp(filepath.Dir(folder), ".extract-")
	if e != nil {
		return "", "", e
	}
	defer os.RemoveAll(stage)
	z, e := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if e != nil {
		return "", "", e
	}
	if len(z.File) != len(m.Files) {
		return "", "", errors.New("runtime file count mismatch")
	}
	for _, f := range z.File {
		expected, ok := m.Files[f.Name]
		if !ok || uint64(expected.Size) != f.UncompressedSize64 {
			return "", "", errors.New("unexpected runtime member")
		}
		p, e := platform.Inside(stage, f.Name)
		if e != nil {
			return "", "", e
		}
		if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			return "", "", e
		}
		r, e := f.Open()
		if e != nil {
			return "", "", e
		}
		out, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			r.Close()
			return "", "", e
		}
		h := sha256.New()
		n, e := io.Copy(io.MultiWriter(out, h), io.LimitReader(r, expected.Size+1))
		r.Close()
		ce := out.Close()
		if e != nil || ce != nil || n != expected.Size || hex.EncodeToString(h.Sum(nil)) != expected.SHA256 {
			return "", "", errors.New("runtime extraction digest mismatch")
		}
	}
	if e = os.Rename(stage, folder); e != nil {
		return "", "", e
	}
	return folder, m.Files["vaultwarden.exe"].SHA256, nil
}
