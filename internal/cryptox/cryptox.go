package cryptox

import (
	"crypto/rand"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

var oidX25519 = asn1.ObjectIdentifier{1, 3, 101, 110}

type pkixPublicKey struct {
	Algo      pkix.AlgorithmIdentifier
	BitString asn1.BitString
}

type pkcs8PrivateKey struct {
	Version    int
	Algo       pkix.AlgorithmIdentifier
	PrivateKey []byte
}

func LoadPublicKey(path string) (*[32]byte, error) {
	der, err := decodePEM(path)
	if err != nil {
		return nil, err
	}
	var pub pkixPublicKey
	if _, err := asn1.Unmarshal(der, &pub); err != nil {
		return nil, fmt.Errorf("parse public key %s: %w", path, err)
	}
	if !pub.Algo.Algorithm.Equal(oidX25519) {
		return nil, fmt.Errorf("%s is not an X25519 public key", path)
	}
	if len(pub.BitString.Bytes) != 32 {
		return nil, fmt.Errorf("%s: invalid X25519 public key length", path)
	}
	var out [32]byte
	copy(out[:], pub.BitString.Bytes)
	return &out, nil
}

func LoadPrivateKey(path string) (*[32]byte, error) {
	der, err := decodePEM(path)
	if err != nil {
		return nil, err
	}
	var priv pkcs8PrivateKey
	if _, err := asn1.Unmarshal(der, &priv); err != nil {
		return nil, fmt.Errorf("parse private key %s: %w", path, err)
	}
	if !priv.Algo.Algorithm.Equal(oidX25519) {
		return nil, fmt.Errorf("%s is not an X25519 private key", path)
	}
	var raw []byte
	if _, err := asn1.Unmarshal(priv.PrivateKey, &raw); err != nil {
		return nil, fmt.Errorf("parse private key %s: %w", path, err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("%s: invalid X25519 private key length", path)
	}
	var out [32]byte
	copy(out[:], raw)
	return &out, nil
}

func decodePEM(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("%s: no PEM data found", path)
	}
	return block.Bytes, nil
}

func Encrypt(pub *[32]byte, plaintext []byte) ([]byte, error) {
	return box.SealAnonymous(nil, plaintext, pub, rand.Reader)
}

func Decrypt(priv *[32]byte, ciphertext []byte) ([]byte, error) {
	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, priv)

	plaintext, ok := box.OpenAnonymous(nil, ciphertext, &pub, priv)
	if !ok {
		return nil, errors.New("decrypt: authentication failed (wrong key?)")
	}
	return plaintext, nil
}
