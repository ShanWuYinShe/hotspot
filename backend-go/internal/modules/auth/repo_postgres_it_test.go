package auth

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

func TestPostgresIT_RegisterCreatesAccount(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)

		account, err := repo.Register(ctx, RegisterParams{
			LoginID:  "alice@example.com",
			Nickname: "Alice",
			Password: "secret123",
			Defaults: ProfileDefaults{
				LocationName: "Test City",
				Latitude:     35.0,
				Longitude:    112.0,
				CreatedAt:    time.Now().UTC(),
			},
			RegisterIP:       "203.0.113.10",
			RegisterLocation: "US",
		})
		if err != nil {
			t.Fatalf("Register: %v", err)
		}
		if account.UserID == "" {
			t.Fatal("expected non-empty user ID")
		}
		if account.LoginID != "alice@example.com" {
			t.Fatalf("login_id: got %q, want %q", account.LoginID, "alice@example.com")
		}
		if account.Nickname != "Alice" {
			t.Fatalf("nickname: got %q, want %q", account.Nickname, "Alice")
		}
		if account.UserType != "normal" {
			t.Fatalf("user_type: got %q, want %q", account.UserType, "normal")
		}
	})
}

func TestPostgresIT_RegisterRejectsDuplicateLoginID(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		params := RegisterParams{
			LoginID:  "bob@example.com",
			Nickname: "Bob",
			Password: "secret123",
			Defaults: ProfileDefaults{
				LocationName: "Test City",
				Latitude:     35.0,
				Longitude:    112.0,
				CreatedAt:    time.Now().UTC(),
			},
			RegisterIP:       "203.0.113.10",
			RegisterLocation: "US",
		}
		if _, err := repo.Register(ctx, params); err != nil {
			t.Fatalf("first Register: %v", err)
		}
		if _, err := repo.Register(ctx, params); err != ErrLoginIDConflict {
			t.Fatalf("duplicate Register: got %v, want ErrLoginIDConflict", err)
		}
	})
}

func TestPostgresIT_AuthenticateVerifiesPassword(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		_, err := repo.Register(ctx, RegisterParams{
			LoginID:  "carol@example.com",
			Nickname: "Carol",
			Password: "correct-horse-battery",
			Defaults: ProfileDefaults{
				LocationName: "Test City",
				Latitude:     35.0,
				Longitude:    112.0,
				CreatedAt:    time.Now().UTC(),
			},
			RegisterIP:       "203.0.113.10",
			RegisterLocation: "US",
		})
		if err != nil {
			t.Fatalf("Register: %v", err)
		}

		account, err := repo.Authenticate(ctx, "carol@example.com", "correct-horse-battery")
		if err != nil {
			t.Fatalf("Authenticate: %v", err)
		}
		if account.LoginID != "carol@example.com" {
			t.Fatalf("login_id: got %q", account.LoginID)
		}

		_, err = repo.Authenticate(ctx, "carol@example.com", "wrong")
		if err != ErrInvalidCredentials {
			t.Fatalf("wrong password: got %v, want ErrInvalidCredentials", err)
		}
	})
}

func TestPostgresIT_CreateGuestCreatesTempAccount(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		account, err := repo.CreateGuest(ctx, GuestParams{
			Nickname:   "Guest123",
			Defaults:   ProfileDefaults{CreatedAt: time.Now().UTC()},
			RegisterIP: "10.0.0.1",
		})
		if err != nil {
			t.Fatalf("CreateGuest: %v", err)
		}
		if account.UserID == "" {
			t.Fatal("expected non-empty user ID")
		}
		if account.UserType != "temp" {
			t.Fatalf("user_type: got %q, want %q", account.UserType, "temp")
		}
		if account.LoginID != "" {
			t.Fatalf("guest should have empty login_id, got %q", account.LoginID)
		}
	})
}

func TestPostgresIT_UpdateLoginMetaPersists(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		account, err := repo.Register(ctx, RegisterParams{
			LoginID:  "dave@example.com",
			Nickname: "Dave",
			Password: "secret123",
			Defaults: ProfileDefaults{
				LocationName: "Test City",
				Latitude:     35.0,
				Longitude:    112.0,
				CreatedAt:    time.Now().UTC(),
			},
			RegisterIP:       "203.0.113.10",
			RegisterLocation: "US",
		})
		if err != nil {
			t.Fatalf("Register: %v", err)
		}
		if err := repo.UpdateLoginMeta(ctx, account.UserID, "198.51.100.1", "JP"); err != nil {
			t.Fatalf("UpdateLoginMeta: %v", err)
		}
		queryable, _ := txFactory.DB.(persistence.QueryableDB)
		var loginIP, loginLoc string
		if err := queryable.QueryRow(ctx, `SELECT COALESCE(last_login_ip,''), COALESCE(last_login_location,'') FROM users WHERE user_id = $1`, account.UserID).Scan(&loginIP, &loginLoc); err != nil {
			t.Fatalf("verify login meta: %v", err)
		}
		if loginIP != "198.51.100.1" {
			t.Fatalf("last_login_ip: got %q, want %q", loginIP, "198.51.100.1")
		}
		if loginLoc != "JP" {
			t.Fatalf("last_login_location: got %q, want %q", loginLoc, "JP")
		}
	})
}

func TestPostgresIT_PasswordResetFlow(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		_, err := repo.Register(ctx, RegisterParams{
			LoginID:  "eve@example.com",
			Nickname: "Eve",
			Password: "old-password",
			Defaults: ProfileDefaults{
				LocationName: "Test City",
				Latitude:     35.0,
				Longitude:    112.0,
				CreatedAt:    time.Now().UTC(),
			},
			RegisterIP:       "203.0.113.10",
			RegisterLocation: "US",
		})
		if err != nil {
			t.Fatalf("Register: %v", err)
		}

		code, err := repo.CreatePasswordReset(ctx, "eve@example.com", time.Now().UTC().Add(1*time.Hour))
		if err != nil {
			t.Fatalf("CreatePasswordReset: %v", err)
		}
		if code == "" {
			t.Fatal("expected non-empty verification code")
		}

		account, err := repo.ResetPassword(ctx, "eve@example.com", code, "new-password")
		if err != nil {
			t.Fatalf("ResetPassword: %v", err)
		}
		if account.LoginID != "eve@example.com" {
			t.Fatalf("login_id: got %q", account.LoginID)
		}

		_, err = repo.Authenticate(ctx, "eve@example.com", "new-password")
		if err != nil {
			t.Fatalf("Authenticate with new password: %v", err)
		}
	})
}
