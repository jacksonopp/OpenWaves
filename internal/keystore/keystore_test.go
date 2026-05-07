package keystore

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrGenerate_Creates(t *testing.T) {
	dir := t.TempDir()
	key, pubPEM, err := LoadOrGenerate("alice", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil private key")
	}
	if pubPEM == "" {
		t.Fatal("expected non-empty public key PEM")
	}

	// Verify files were created
	if _, err := os.Stat(filepath.Join(dir, "alice.pem")); err != nil {
		t.Errorf("private key file not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "alice.pub.pem")); err != nil {
		t.Errorf("public key file not created: %v", err)
	}
}

func TestLoadOrGenerate_Idempotent(t *testing.T) {
	dir := t.TempDir()

	_, pub1, err := LoadOrGenerate("bob", dir)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	_, pub2, err := LoadOrGenerate("bob", dir)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	if pub1 != pub2 {
		t.Errorf("expected same public key PEM on second call\ngot1: %s\ngot2: %s", pub1, pub2)
	}
}

func TestLoadOrGenerate_LoadsExisting(t *testing.T) {
	dir := t.TempDir()

	// Generate a known key pair manually
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	if err := os.WriteFile(filepath.Join(dir, "carol.pem"), privPEM, 0600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "carol.pub.pem"), pubPEM, 0600); err != nil {
		t.Fatalf("write public key: %v", err)
	}

	loadedKey, loadedPubPEM, err := LoadOrGenerate("carol", dir)
	if err != nil {
		t.Fatalf("LoadOrGenerate error: %v", err)
	}

	if loadedPubPEM != string(pubPEM) {
		t.Errorf("public key PEM mismatch\nwant: %s\ngot:  %s", pubPEM, loadedPubPEM)
	}

	if loadedKey.PublicKey.N.Cmp(privKey.PublicKey.N) != 0 {
		t.Error("loaded private key does not match original")
	}
}
