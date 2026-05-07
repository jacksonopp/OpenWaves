package keystore

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

// LoadOrGenerate loads an existing RSA key pair for the given username from keysDir,
// or generates a new RSA-2048 key pair and writes it to disk if either file is missing.
// Returns the private key and the public key in PEM format.
func LoadOrGenerate(username, keysDir string) (*rsa.PrivateKey, string, error) {
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return nil, "", fmt.Errorf("keystore: create keys dir: %w", err)
	}

	privPath := filepath.Join(keysDir, username+".pem")
	pubPath := filepath.Join(keysDir, username+".pub.pem")

	_, errPriv := os.Stat(privPath)
	_, errPub := os.Stat(pubPath)

	if errPriv == nil && errPub == nil {
		return loadKeyPair(privPath, pubPath)
	}

	return generateAndSave(privPath, pubPath)
}

func loadKeyPair(privPath, pubPath string) (*rsa.PrivateKey, string, error) {
	privBytes, err := os.ReadFile(privPath)
	if err != nil {
		return nil, "", fmt.Errorf("keystore: read private key: %w", err)
	}

	block, _ := pem.Decode(privBytes)
	if block == nil {
		return nil, "", fmt.Errorf("keystore: failed to decode private key PEM")
	}

	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, "", fmt.Errorf("keystore: parse private key: %w", err)
	}

	privKey, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, "", fmt.Errorf("keystore: private key is not RSA")
	}

	pubPEM, err := os.ReadFile(pubPath)
	if err != nil {
		return nil, "", fmt.Errorf("keystore: read public key: %w", err)
	}

	return privKey, string(pubPEM), nil
}

func generateAndSave(privPath, pubPath string) (*rsa.PrivateKey, string, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", fmt.Errorf("keystore: generate RSA key: %w", err)
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, "", fmt.Errorf("keystore: marshal private key: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, "", fmt.Errorf("keystore: marshal public key: %w", err)
	}

	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		return nil, "", fmt.Errorf("keystore: write private key: %w", err)
	}

	if err := os.WriteFile(pubPath, pubPEM, 0600); err != nil {
		return nil, "", fmt.Errorf("keystore: write public key: %w", err)
	}

	return privKey, string(pubPEM), nil
}
