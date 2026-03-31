package auth

import (
	"encoding/base64"
	"strconv"
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

func TestTokenManagerParsesLegacyTokenDuringMigration(t *testing.T) {
	manager := NewTokenManager("test-secret")
	expiresAt := time.Now().UTC().Add(time.Minute)
	payload := strings.Join([]string{"user_legacy", "sess_legacy", strconv.FormatInt(expiresAt.Unix(), 10)}, ".")
	token := base64.RawURLEncoding.EncodeToString([]byte(payload + "." + manager.sign(payload)))
	claims, err := manager.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("parse legacy token: %v", err)
	}
	if claims.UserID != "user_legacy" || claims.SessionID != "sess_legacy" {
		t.Fatalf("unexpected legacy claims: %#v", claims)
	}
}

func TestValidateCSRF(t *testing.T) {
	req := newRequestWithCookie(CSRFCookieName, "token-1")
	req.Header.Set(CSRFHeaderName, "token-1")
	if !ValidateCSRF(req) {
		t.Fatal("expected csrf to validate")
	}
}
