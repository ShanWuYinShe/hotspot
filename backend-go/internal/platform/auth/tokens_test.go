package auth

import (
	"testing"
	"time"
)

func TestTokenManagerMintAndParse(t *testing.T) {
	manager := NewTokenManager("test-secret")
	token, expiresAt, err := manager.MintAccessToken("user_1", "sess_1", time.Minute)
	if err != nil {
		t.Fatalf("mint token: %v", err)
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

func TestValidateCSRF(t *testing.T) {
	req := newRequestWithCookie(CSRFCookieName, "token-1")
	req.Header.Set(CSRFHeaderName, "token-1")
	if !ValidateCSRF(req) {
		t.Fatal("expected csrf to validate")
	}
}
