package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenManager 负责生成和解析 access/refresh token。
type TokenManager interface {
	GenerateTokens(userID, sessionID string) (accessToken, refreshToken string, accessExp, refreshExp time.Time, err error)
	ParseAccessToken(token string) (userID, sessionID string, exp time.Time, err error)
	ParseRefreshToken(token string) (userID, sessionID string, exp time.Time, err error)
}

type jwtTokenManager struct {
	secret              []byte
	accessTokenTTL      time.Duration
	refreshTokenTTL     time.Duration
}

// NewJWTTokenManager 使用 HMAC-SHA 方式创建 Token 管理器。
func NewJWTTokenManager(secret string, accessTTL, refreshTTL time.Duration) TokenManager {
	return &jwtTokenManager{
		secret:          []byte(secret),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

// tokenClaims 为 JWT 载荷，仅存 user_id 和 session_id，遵循 spec 要求。
type tokenClaims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

func (m *jwtTokenManager) GenerateTokens(userID, sessionID string) (string, string, time.Time, time.Time, error) {
	now := time.Now().UTC()
	accessExp := now.Add(m.accessTokenTTL)
	refreshExp := now.Add(m.refreshTokenTTL)

	accessClaims := &tokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	refreshClaims := &tokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(m.secret)
	if err != nil {
		return "", "", time.Time{}, time.Time{}, err
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(m.secret)
	if err != nil {
		return "", "", time.Time{}, time.Time{}, err
	}
	return accessToken, refreshToken, accessExp, refreshExp, nil
}

func (m *jwtTokenManager) parse(tokenStr string) (*tokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &tokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*tokenClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func (m *jwtTokenManager) ParseAccessToken(token string) (string, string, time.Time, error) {
	c, err := m.parse(token)
	if err != nil {
		return "", "", time.Time{}, err
	}
	return c.UserID, c.SessionID, c.ExpiresAt.Time, nil
}

func (m *jwtTokenManager) ParseRefreshToken(token string) (string, string, time.Time, error) {
	c, err := m.parse(token)
	if err != nil {
		return "", "", time.Time{}, err
	}
	return c.UserID, c.SessionID, c.ExpiresAt.Time, nil
}
