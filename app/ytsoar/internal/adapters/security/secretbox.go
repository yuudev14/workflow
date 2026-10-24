package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
)

// SecretBox seals secrets with AES-256-GCM. A sealed value is base64(nonce + ciphertext).
type SecretBox struct {
	aead cipher.AEAD
}

func NewSecretBox(base64Key string) (*SecretBox, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil || len(key) != 32 {
		return nil, errors.New("YTSOAR_ENCRYPTION_KEY must be 32 bytes in base64")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &SecretBox{aead: aead}, nil
}

func (b *SecretBox) Seal(plain string) (string, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b.aead.Seal(nonce, nonce, []byte(plain), nil)), nil
}

func (b *SecretBox) Open(sealed string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(sealed)
	n := b.aead.NonceSize()
	if err != nil || len(raw) < n {
		return "", errors.New("secret is not sealed")
	}
	plain, err := b.aead.Open(nil, raw[:n], raw[n:], nil)
	if err != nil {
		return "", errors.New("secret does not open with this key")
	}
	return string(plain), nil
}
