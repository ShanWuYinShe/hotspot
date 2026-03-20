package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type TokenManager struct {
	secret []byte
}

type AccessClaims struct {
	UserID    string
	SessionID string
	ExpiresAt time.Time
}

func NewTokenManager(secret string) *TokenManager {
	if secret == "" {
		secret = "dev-secret-change-me"
	}
	return &TokenManager{secret: []byte(secret)}
}

func (m *TokenManager) MintAccessToken(userID, sessionID string, ttl time.Duration) (string, time.Time, error) {
	expiresAt := time.Now().UTC().Add(ttl)
	payload := strings.Join([]string{userID, sessionID, strconv.FormatInt(expiresAt.Unix(), 10)}, ".")
	sig := m.sign(payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "." + sig)), expiresAt, nil
}

func (m *TokenManager) ParseAccessToken(token string) (AccessClaims, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return AccessClaims{}, errors.New("invalid token")
	}
	parts := strings.Split(string(decoded), ".")
	if len(parts) != 4 {
		return AccessClaims{}, errors.New("invalid token")
	}
	payload := strings.Join(parts[:3], ".")
	if !hmac.Equal([]byte(parts[3]), []byte(m.sign(payload))) {
		return AccessClaims{}, errors.New("invalid token")
	}
	expiresUnix, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return AccessClaims{}, errors.New("invalid token")
	}
	expiresAt := time.Unix(expiresUnix, 0).UTC()
	if time.Now().UTC().After(expiresAt) {
		return AccessClaims{}, errors.New("expired token")
	}
	return AccessClaims{UserID: parts[0], SessionID: parts[1], ExpiresAt: expiresAt}, nil
}

func GenerateOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func GenerateCSRFToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func NewSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("sess_%s", hex.EncodeToString(buf)), nil
}

func (m *TokenManager) sign(payload string) string {
	h := hmac.New(sha256.New, m.secret)
	_, _ = h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}
