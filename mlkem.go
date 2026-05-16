package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/open-quantum-safe/liboqs-go/oqs"
)

const kemAlgorithm = "ML-KEM-768"

type KEMSession struct {
	kem       oqs.KeyEncapsulation
	publicKey []byte
	secretKey []byte
}

func newKEMSession() (*KEMSession, error) {
	kem := oqs.KeyEncapsulation{}
	if err := kem.Init(kemAlgorithm, nil); err != nil {
		return nil, fmt.Errorf("ML-KEM init failed: %w", err)
	}

	pubKey, err := kem.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("ML-KEM keygen failed: %w", err)
	}

	return &KEMSession{
		kem:       kem,
		publicKey: pubKey,
	}, nil
}

func (s *KEMSession) encapsulate(peerPublicKey []byte) (ciphertext []byte, sharedSecret []byte, err error) {
	kem := oqs.KeyEncapsulation{}
	if err := kem.Init(kemAlgorithm, nil); err != nil {
		return nil, nil, fmt.Errorf("ML-KEM encap init failed: %w", err)
	}
	defer kem.Clean()

	ct, ss, err := kem.EncapSecret(peerPublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("ML-KEM encapsulation failed: %w", err)
	}
	return ct, ss, nil
}

func (s *KEMSession) decapsulate(ciphertext []byte) ([]byte, error) {
	ss, err := s.kem.DecapSecret(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("ML-KEM decapsulation failed: %w", err)
	}
	return ss, nil
}

func (s *KEMSession) clean() {
	s.kem.Clean()
}

// deriveSessionKey converts a raw shared secret into a 32-byte AES key
func deriveSessionKey(sharedSecret []byte, room string) []byte {
	h := sha256.Sum256(append(sharedSecret, []byte(room)...))
	return h[:]
}

// sharedSecretToHex encodes shared secret for transport
func sharedSecretToHex(ss []byte) string {
	return hex.EncodeToString(ss)
}

// hexToBytes decodes hex string back to bytes
func hexToBytes(s string) ([]byte, error) {
	return hex.DecodeString(s)
}
