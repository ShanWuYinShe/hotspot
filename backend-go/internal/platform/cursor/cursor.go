package cursor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"

	"hotspot/backend-go/internal/platform/config"
)

var ErrInvalidCursor = errors.New("cursor: invalid cursor")

func Encode(namespace string, payload any) (string, error) {
	blob, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	block, err := newBlock(namespace)
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
	sealed := gcm.Seal(nonce, nonce, blob, nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func Decode(namespace string, value string, out any) error {
	if value == "" {
		return nil
	}
	blob, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return ErrInvalidCursor
	}
	block, err := newBlock(namespace)
	if err != nil {
		return ErrInvalidCursor
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ErrInvalidCursor
	}
	if len(blob) < gcm.NonceSize() {
		return ErrInvalidCursor
	}
	nonce, ciphertext := blob[:gcm.NonceSize()], blob[gcm.NonceSize():]
	payload, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ErrInvalidCursor
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return ErrInvalidCursor
	}
	return nil
}

func newBlock(namespace string) (cipher.Block, error) {
	secret := config.Load().AuthTokenSecret
	if secret == "" {
		secret = "dev-secret-change-me"
	}
	key := sha256.Sum256([]byte(namespace + ":" + secret))
	return aes.NewCipher(key[:])
}
