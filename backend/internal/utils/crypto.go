package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

func EncryptText(text, key string) (string, error) {
	if key == "" {
		return "", errors.New("empty key")
	}
	sum := make([]byte, 32)
	copy(sum, []byte(key))
	block, err := aes.NewCipher(sum)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(text), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func DecryptText(text, key string) (string, error) {
	if key == "" {
		return "", errors.New("empty key")
	}
	raw, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		return "", err
	}
	sum := make([]byte, 32)
	copy(sum, []byte(key))
	block, err := aes.NewCipher(sum)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("invalid ciphertext")
	}
	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
