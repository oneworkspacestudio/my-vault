package vault

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

func TestLocalHTTPSLeafIsRestrictedAndStable(t *testing.T) {
	root := t.TempDir()
	cert, key, thumb, e := localCertificate(root)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tls.LoadX509KeyPair(cert, key); e != nil {
		t.Fatal(e)
	}
	data, _ := os.ReadFile(cert)
	block, _ := pem.Decode(data)
	leaf, e := x509.ParseCertificate(block.Bytes)
	if e != nil {
		t.Fatal(e)
	}
	if leaf.IsCA || leaf.KeyUsage&x509.KeyUsageCertSign != 0 || leaf.VerifyHostname("localhost") != nil || leaf.VerifyHostname("127.0.0.1") != nil || leaf.VerifyHostname("example.com") == nil {
		t.Fatal("local certificate privileges exceed localhost")
	}
	_, _, again, e := localCertificate(root)
	if e != nil || again != thumb {
		t.Fatal("startup silently rotated certificate")
	}
	c := Config{Listen: "127.0.0.1:8914", Origin: "https://vault.example.test"}
	if c.LocalURL() != "https://localhost:8915" {
		t.Fatal(c.LocalURL())
	}
}
