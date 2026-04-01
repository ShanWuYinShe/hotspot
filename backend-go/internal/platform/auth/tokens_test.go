package auth

import (
	"strings"
	"testing"
	"time"
)

func TestTokenManagerMintAndParse(t *testing.T) {
	manager := NewTokenManager("test-secret")
	token, expiresAt, err := manager.MintAccessToken("user_1", "sess_1", time.Minute)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	if strings.Count(token, ".") != 2 {
		t.Fatalf("expected JWT token format, got %q", token)
	}
	claims, err := manager.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.UserID != "user_1" || claims.SessionID != "sess_1" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
	if claims.ExpiresAt.Unix() != expiresAt.Unix() {
		t.Fatalf("unexpected expiry: %v != %v", claims.ExpiresAt, expiresAt)
	}
}

func TestTokenManagerRejectsLegacyStyleToken(t *testing.T) {
	manager := NewTokenManager("test-secret")
	token := "dXNlcl9sZWdhY3kuc2Vzc19sZWdhY3kuMTcxMjAwMDAwMC5zaWc"
	if _, err := manager.ParseAccessToken(token); err == nil || err.Error() != "invalid token" {
		t.Fatalf("expected invalid legacy token, got %v", err)
	}
}

func TestValidateCSRF(t *testing.T) {
	req := newRequestWithCookie(CSRFCookieName, "token-1")
	req.Header.Set(CSRFHeaderName, "token-1")
	if !ValidateCSRF(req) {
		t.Fatal("expected csrf to validate")
	}
}
