package hls

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// Sign computes an RSA-PKCS1v15 signature over the SHA-256 hash of data.
func Sign(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	h := sha256.Sum256(data)
	return rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
}

// Verify checks an RSA-PKCS1v15 signature against data using the given public key PEM.
func Verify(publicKeyPEM string, data []byte, sig []byte) error {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return errors.New("hls: failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return errors.New("hls: public key is not RSA")
	}

	h := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, h[:], sig)
}
