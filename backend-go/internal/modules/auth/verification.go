package auth

import (
	"context"
	"strings"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/observability"
)

// VerificationChannel identifies the delivery channel for a verification code.
type VerificationChannel string

const (
	ChannelEmail VerificationChannel = "email"
	ChannelSMS   VerificationChannel = "sms"
)

// verificationSender delivers verification codes to users. The interface is
// channel-agnostic so it can serve password reset, account upgrade, and any
// future flow that needs code-based verification. When a real SMS/SMTP gateway
// is wired, add a new implementation and select it in defaultVerificationSender.
type verificationSender interface {
	SendVerification(ctx context.Context, to string, channel VerificationChannel, code string) error
}

// detectChannel infers the delivery channel from the login_id format: an "@"
// means email, an all-digit string means phone/SMS, anything else (username)
// defaults to email so the dev log still records the attempt.
func detectChannel(loginID string) VerificationChannel {
	if strings.Contains(loginID, "@") {
		return ChannelEmail
	}
	if isAllDigits(loginID) {
		return ChannelSMS
	}
	return ChannelEmail
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// defaultVerificationSender picks the delivery backend based on the environment.
// In dev/test a logVerificationSender writes the code to the application log so
// the password-reset (and future upgrade) flow is end-to-end usable without an
// external gateway. In production-like environments a noopVerificationSender
// preserves the previous "not configured" behavior until a real gateway is wired.
func defaultVerificationSender(cfg config.Config) verificationSender {
	if !cfg.IsProductionLike() {
		return &logVerificationSender{}
	}
	return &noopVerificationSender{}
}

// logVerificationSender "delivers" verification codes by writing them to the
// log. This makes verification flows exercisable in dev/test (the developer
// reads the code from the log) without an external SMS/SMTP gateway.
type logVerificationSender struct{}

func (*logVerificationSender) SendVerification(_ context.Context, to string, channel VerificationChannel, code string) error {
	observability.L().Info("verification code generated (dev/test mode — no external gateway, read code from log)",
		"channel", string(channel),
		"login_id", observability.Redact(to),
		"code", code,
	)
	return nil
}

// noopVerificationSender is the production fallback when no SMS/SMTP gateway is
// configured; SendVerification reports the service as unavailable.
type noopVerificationSender struct{}

func (*noopVerificationSender) SendVerification(_ context.Context, _ string, _ VerificationChannel, _ string) error {
	return ErrVerificationServiceUnavailable
}
