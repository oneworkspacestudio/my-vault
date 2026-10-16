package vault

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func (c Config) LocalTLSListen() string {
	h, p, _ := net.SplitHostPort(c.Listen)
	n, _ := strconv.Atoi(p)
	return net.JoinHostPort(h, strconv.Itoa(n+1))
}
func (c Config) LocalURL() string {
	_, p, _ := net.SplitHostPort(c.LocalTLSListen())
	return "https://localhost:" + p + c.Base()
}

// A self-signed leaf restricted to localhost, not a certificate authority.
// Trust is a distinct owner action; startup never modifies a certificate store.
func localCertificate(root string) (cert, key, thumbprint string, err error) {
	folder := filepath.Join(root, "local-tls")
	if err = os.MkdirAll(folder, 0700); err != nil {
		return
	}
	cert = filepath.Join(folder, "localhost.pem")
	key = filepath.Join(folder, "localhost-key.pem")
	derPath := filepath.Join(folder, "localhost.cer")
	data, e := os.ReadFile(cert)
	if e == nil {
		block, _ := pem.Decode(data)
		if block == nil {
			err = errors.New("local HTTPS certificate is invalid")
			return
		}
		parsed, pe := x509.ParseCertificate(block.Bytes)
		if pe != nil {
			err = pe
			return
		}
		if parsed.IsCA || parsed.VerifyHostname("localhost") != nil {
			err = errors.New("unexpected local HTTPS certificate")
			return
		}
		sum := sha1.Sum(block.Bytes)
		thumbprint = hex.EncodeToString(sum[:])
		return
	}
	if !os.IsNotExist(e) {
		err = e
		return
	}
	if _, e = os.Stat(key); !os.IsNotExist(e) {
		err = errors.New("incomplete local HTTPS state; retain it and repair the certificate files")
		return
	}
	private, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		err = e
		return
	}
	serial, e := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if e != nil {
		err = e
		return
	}
	template := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: "My Vault localhost"}, NotBefore: time.Now().Add(-5 * time.Minute), NotAfter: time.Now().AddDate(1, 0, 0), DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true, IsCA: false}
	der, e := x509.CreateCertificate(rand.Reader, template, template, &private.PublicKey, private)
	if e != nil {
		err = e
		return
	}
	secret, e := x509.MarshalPKCS8PrivateKey(private)
	if e != nil {
		err = e
		return
	}
	if err = os.WriteFile(key, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: secret}), 0600); err != nil {
		return
	}
	if err = os.WriteFile(derPath, der, 0600); err != nil {
		return
	}
	err = os.WriteFile(cert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600)
	sum := sha1.Sum(der)
	thumbprint = hex.EncodeToString(sum[:])
	return
}
