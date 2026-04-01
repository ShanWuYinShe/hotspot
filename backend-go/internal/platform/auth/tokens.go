package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	secret []byte
}

type AccessClaims struct {
	UserID    string
	SessionID string
	ExpiresAt time.Time
}

type jwtAccessClaims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

func NewTokenManager(secret string) *TokenManager {
	if secret == "" {
		secret = "dev-secret-change-me"
	}
	return &TokenManager{secret: []byte(secret)}
}

func (m *TokenManager) MintAccessToken(userID, sessionID string, ttl time.Duration) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)
	claims := jwtAccessClaims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

func (m *TokenManager) ParseAccessToken(token string) (AccessClaims, error) {
	claims, err := m.parseJWTAccessToken(token)
	if err == nil {
		return claims, nil
	}
	if errors.Is(err, jwt.ErrTokenExpired) {
		return AccessClaims{}, errors.New("expired token")
	}
	return AccessClaims{}, errors.New("invalid token")
}

func (m *TokenManager) parseJWTAccessToken(token string) (AccessClaims, error) {
	claims := &jwtAccessClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("invalid token")
		}
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return AccessClaims{}, err
	}
	if !parsed.Valid || claims.Subject == "" || claims.SessionID == "" || claims.ExpiresAt == nil {
		return AccessClaims{}, errors.New("invalid token")
	}
	return AccessClaims{UserID: claims.Subject, SessionID: claims.SessionID, ExpiresAt: claims.ExpiresAt.Time.UTC()}, nil
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
