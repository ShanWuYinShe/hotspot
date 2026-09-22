package auth

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/config"
)

// --- SMTP sender selection ---

func TestDefaultVerificationSenderSMTPConfigured(t *testing.T) {
	// SMTP_HOST 已配置：任何环境都用 SMTP sender（邮箱通道端到端可用）
	cfg := config.Config{AppEnv: "production", SMTPHost: "smtp.example.com"}
	if _, ok := defaultVerificationSender(cfg).(*smtpVerificationSender); !ok {
		t.Fatalf("expected *smtpVerificationSender when SMTP_HOST set")
	}
	cfg.AppEnv = "dev"
	if _, ok := defaultVerificationSender(cfg).(*smtpVerificationSender); !ok {
		t.Fatalf("expected *smtpVerificationSender in dev when SMTP_HOST set")
	}
}

func TestNewSMTPVerificationSenderRequiresHost(t *testing.T) {
	if s := newSMTPVerificationSender(config.Config{}); s != nil {
		t.Fatal("expected nil sender without SMTP_HOST")
	}
	if s := newSMTPVerificationSender(config.Config{SMTPHost: "smtp.example.com", SMTPPort: ""}); s == nil {
		t.Fatal("expected sender with SMTP_HOST")
	} else if s.port != "587" {
		t.Fatalf("Load() defaults port to 587; got %q", s.port)
	}
}

// --- SMTP sender behaviour ---

func TestSMTPSenderRejectsSMSChannel(t *testing.T) {
	s := &smtpVerificationSender{host: "smtp.example.com", port: "587"}
	if err := s.SendVerification(context.Background(), "13800138000", ChannelSMS, "123456"); err != ErrSMSChannelUnavailable {
		t.Fatalf("expected ErrSMSChannelUnavailable, got %v", err)
	}
}

func TestSMTPSenderDialFailsFastOnUnreachableHost(t *testing.T) {
	// 先占用再释放一个本机端口：拨回该端口必然立即 ECONNREFUSED。
	// 此前用 RFC 5737 TEST-NET-1（192.0.2.1），在 macOS/部分网络上会等满
	// 系统 TCP 超时（实测 ~15s），拖慢整个包的测试。
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	s := &smtpVerificationSender{host: "127.0.0.1", port: strconv.Itoa(port)}
	if err := s.SendVerification(context.Background(), "a@b.com", ChannelEmail, "123456"); err == nil {
		t.Fatal("expected dial error for closed SMTP port")
	}
}

// --- message construction ---

func TestVerificationEmailMessage(t *testing.T) {
	from := "HotSpot <noreply@chaoset.com>"
	msg := verificationEmailMessage(from, "user@example.com", "<p>code</p>")
	for _, want := range []string{
		"From: " + from + "\r\n",
		"To: user@example.com\r\n",
		"Subject: =?utf-8?q?",
		"MIME-Version: 1.0\r\n",
		"Content-Type: text/html; charset=utf-8\r\n",
		"Content-Transfer-Encoding: base64\r\n",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message missing %q", want)
		}
	}
	// 正文 base64 行 ≤76 字符（RFC 2045）
	body := msg[strings.Index(msg, "\r\n\r\n")+4:]
	for _, line := range strings.Split(body, "\r\n") {
		if len(line) > 76 {
			t.Fatalf("base64 body line exceeds 76 chars: %d", len(line))
		}
	}
}

func TestVerificationEmailBodyContainsCode(t *testing.T) {
	body := verificationEmailBody("008797")
	if !strings.Contains(body, "008797") || !strings.Contains(body, "10 分钟") {
		t.Fatal("body must render the code and TTL notice")
	}
}

func TestSMTPEnvelopeFrom(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"HotSpot <noreply@chaoset.com>", "noreply@chaoset.com"},
		{"noreply@chaoset.com", "noreply@chaoset.com"},
		{"  bare@x.com  ", "bare@x.com"},
	} {
		if got := smtpEnvelopeFrom(tc.in); got != tc.want {
			t.Fatalf("smtpEnvelopeFrom(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
