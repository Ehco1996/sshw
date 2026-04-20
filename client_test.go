package sshw

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestParseSigner_rsa(t *testing.T) {
	t.Parallel()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}
	pemBytes := pem.EncodeToMemory(block)

	s, err := parseSigner(pemBytes, "")
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("nil signer")
	}
}

func TestParseSigner_invalidPEM(t *testing.T) {
	t.Parallel()
	_, err := parseSigner([]byte("not a pem block"), "")
	if err == nil {
		t.Fatal("expected error")
	}
}
