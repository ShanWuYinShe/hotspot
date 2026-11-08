package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
)

func TestAPIErrorMappingAdditionalSentinelsCov(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code httpx.Code
	}{
		{"unauthenticated", ErrUnauthenticated, httpx.CodeUnauthorized},
		{"account banned", ErrAccountBanned, httpx.CodeAccountBanned},
		{"register code throttled", ErrRegisterCodeThrottled, httpx.CodeRateLimited},
		{"invalid register code", ErrInvalidRegisterCode, httpx.CodeInvalidOldCredential},
		{"missing register code", ErrMissingRegisterCode, httpx.CodeInvalidRequest},
		{"registration closed", ErrRegistrationClosed, httpx.CodeConflict},
		{"guest disabled", ErrGuestDisabled, httpx.CodeConflict},
	}
	for _, tc := range tests {
		apiErr := apiErrorFrom(tc.err)
		if apiErr == nil || apiErr.Code != tc.code {
			t.Fatalf("%s: expected code %d, got %#v", tc.name, tc.code, apiErr)
		}
	}
	if apiErr := apiErrorFrom(ErrRegisterCodeThrottled); apiErr.RetryAfterSeconds != int(registerCodeThrottleWindow.Seconds()) {
		t.Fatalf("register code throttle Retry-After = %d, want %d", apiErr.RetryAfterSeconds, int(registerCodeThrottleWindow.Seconds()))
	}
	if apiErr := apiErrorFrom(ErrAccountLocked); apiErr.RetryAfterSeconds != int(defaultLockDuration.Seconds()) {
		t.Fatalf("lockout Retry-After = %d, want %d", apiErr.RetryAfterSeconds, int(defaultLockDuration.Seconds()))
	}
}

func TestLoginIDEmailAndContactEdgeCasesCov(t *testing.T) {
	invalid := []string{
		"\x01@b.com",
		"a@.b.com",
		"a@b.com.",
		"a@b..com",
		"a@b",
		"a@@b.com",
		"@b.com",
		"a@",
	}
	for _, loginID := range invalid {
		if isValidEmailLoginID(loginID) {
			t.Fatalf("expected %q invalid", loginID)
		}
	}
	if !isValidEmailLoginID("a@b.com") {
		t.Fatal("expected valid email")
	}
	if email, phone := loginIDContact("a@b.com"); email != "a@b.com" || phone != "" {
		t.Fatalf("email contact: %q/%q", email, phone)
	}
}

func TestHashPasswordRejectsOverlongPasswordCov(t *testing.T) {
	if _, err := hashPassword(strings.Repeat("a", 73)); err == nil {
		t.Fatal("expected bcrypt overlong password error")
	}
}

func TestIsAllDigitsEmptyCov(t *testing.T) {
	if isAllDigits("") {
		t.Fatal("empty string is not all digits")
	}
	if !isAllDigits("13800138000") || isAllDigits("138a") {
		t.Fatal("unexpected isAllDigits result")
	}
}

func TestNullIfEmptyCov(t *testing.T) {
	if got := nullIfEmpty(""); got != nil {
		t.Fatalf("empty string must map to nil, got %#v", got)
	}
	if got := nullIfEmpty("x"); got != "x" {
		t.Fatalf("non-empty value must pass through, got %#v", got)
	}
}

func TestWrapBase64LinesLongInputCov(t *testing.T) {
	wrapped := wrapBase64Lines(strings.Repeat("A", 160))
	lines := strings.Split(wrapped, "\r\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), wrapped)
	}
	if len(lines[0]) != 76 || len(lines[1]) != 76 || len(lines[2]) != 8 {
		t.Fatalf("unexpected line lengths: %d/%d/%d", len(lines[0]), len(lines[1]), len(lines[2]))
	}
}

func TestModuleAccessorsCov(t *testing.T) {
	m, cancel := newTestModule(t)
	defer cancel()
	writer := &recordingProfileWriter{}
	m.SetProfileWriter(writer)
	if m.repo.profileWriter != writer {
		t.Fatal("SetProfileWriter did not wire repo writer")
	}
	if m.PlatformAuth() == nil || m.Service() == nil {
		t.Fatal("expected module accessors to return wired dependencies")
	}
}
func TestPostgresRepositoryNilFactoryFailsClosedCov(t *testing.T) {
	repo := NewPostgresRepository(nil)
	ctx := context.Background()
	now := time.Now().UTC()
	cases := []struct {
		name string
		fn   func() error
	}{
		{"Register", func() error {
			_, err := repo.Register(ctx, RegisterParams{LoginID: "alice", Password: "password1"})
			return err
		}},
		{"AttachCredential", func() error { return repo.AttachCredential(ctx, "u1", "alice", "password1") }},
		{"Authenticate", func() error { _, err := repo.Authenticate(ctx, "alice", "password1"); return err }},
		{"CreatePasswordReset", func() error { _, err := repo.CreatePasswordReset(ctx, "alice", now); return err }},
		{"ResetPassword", func() error { _, err := repo.ResetPassword(ctx, "alice", "123456", "password1"); return err }},
		{"ChangePassword", func() error { return repo.ChangePassword(ctx, "u1", "old", "password1") }},
		{"CreateRegisterCode", func() error { _, err := repo.CreateRegisterCode(ctx, "alice", now); return err }},
		{"VerifyRegisterCode", func() error { return repo.VerifyRegisterCode(ctx, "alice", "123456") }},
		{"ConsumeRegisterCode", func() error { return repo.ConsumeRegisterCode(ctx, "alice", "123456") }},
		{"PurgeUserData", func() error { return repo.PurgeUserData(ctx, "u1") }},
		{"SweepRetention", func() error { _, err := repo.SweepRetention(ctx, now, time.Hour, time.Hour); return err }},
	}
	for _, tc := range cases {
		if err := tc.fn(); !errors.Is(err, persistence.ErrNotImplemented) {
			t.Fatalf("%s: expected ErrNotImplemented, got %v", tc.name, err)
		}
	}
}
