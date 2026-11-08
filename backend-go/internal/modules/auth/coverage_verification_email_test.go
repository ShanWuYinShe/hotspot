package auth

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// covFakeSMTPOptions 控制假 SMTP 服务器的行为，用于覆盖 dial/sendMail 各分支。
type covFakeSMTPOptions struct {
	greeting          string
	advertiseSTARTTLS bool
	rejectMail        bool
	rejectRcpt        bool
	rejectData        bool
	closeAfterData    bool
}

type covFakeSMTP struct {
	opts covFakeSMTPOptions
	ln   net.Listener
	wg   sync.WaitGroup
}

func newCovFakeSMTP(t *testing.T, opts covFakeSMTPOptions) (host, port string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &covFakeSMTP{opts: opts, ln: ln}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				s.handle(conn)
			}()
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		s.wg.Wait()
	})
	addr := ln.Addr().(*net.TCPAddr)
	return addr.IP.String(), strconv.Itoa(addr.Port)
}

func (s *covFakeSMTP) handle(conn net.Conn) {
	defer conn.Close()
	greeting := s.opts.greeting
	if greeting == "" {
		greeting = "220 fake ESMTP"
	}
	if _, err := fmt.Fprintf(conn, "%s\r\n", greeting); err != nil {
		return
	}
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	respond := func(line string) bool {
		if _, err := writer.WriteString(line); err != nil {
			return false
		}
		return writer.Flush() == nil
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			if s.opts.advertiseSTARTTLS {
				if !respond("250-fake\r\n250-STARTTLS\r\n250 OK\r\n") {
					return
				}
			} else if !respond("250-fake\r\n250 OK\r\n") {
				return
			}
		case strings.HasPrefix(cmd, "STARTTLS"):
			_ = respond("220 Ready to start TLS\r\n")
			// 直接关闭连接：客户端 TLS 握手失败，覆盖 starttls 错误分支。
			return
		case strings.HasPrefix(cmd, "MAIL FROM"):
			if s.opts.rejectMail {
				if !respond("550 rejected\r\n") {
					return
				}
			} else if !respond("250 OK\r\n") {
				return
			}
		case strings.HasPrefix(cmd, "RCPT TO"):
			if s.opts.rejectRcpt {
				if !respond("550 rejected\r\n") {
					return
				}
			} else if !respond("250 OK\r\n") {
				return
			}
		case strings.HasPrefix(cmd, "DATA"):
			if s.opts.rejectData {
				if !respond("550 rejected\r\n") {
					return
				}
				continue
			}
			if !respond("354 End data with <CR><LF>.<CR><LF>\r\n") {
				return
			}
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dataLine, "\r\n") == "." {
					break
				}
			}
			if !respond("250 OK: queued\r\n") {
				return
			}
			if s.opts.closeAfterData {
				return
			}
		case strings.HasPrefix(cmd, "QUIT"):
			_ = respond("221 Bye\r\n")
			return
		default:
			if !respond("250 OK\r\n") {
				return
			}
		}
	}
}

func TestSMTPSendVerificationSuccessCov(t *testing.T) {
	host, port := newCovFakeSMTP(t, covFakeSMTPOptions{})
	sender := &smtpVerificationSender{
		host: host, port: port, from: "HotSpot <noreply@example.com>",
	}
	if err := sender.SendVerification(context.Background(), "user@example.com", ChannelEmail, "123456"); err != nil {
		t.Fatalf("send verification: %v", err)
	}
}

func TestSMTPSendVerificationFailureBranchesCov(t *testing.T) {
	cases := []struct {
		name    string
		opts    covFakeSMTPOptions
		contain string
	}{
		{"mail rejected", covFakeSMTPOptions{rejectMail: true}, "mail from"},
		{"rcpt rejected", covFakeSMTPOptions{rejectRcpt: true}, "rcpt to"},
		{"data rejected", covFakeSMTPOptions{rejectData: true}, "smtp data"},
		{"quit failed", covFakeSMTPOptions{closeAfterData: true}, ""},
		{"invalid greeting", covFakeSMTPOptions{greeting: "not-smtp"}, "smtp client"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host, port := newCovFakeSMTP(t, tc.opts)
			sender := &smtpVerificationSender{host: host, port: port, from: "noreply@example.com"}
			err := sender.SendVerification(context.Background(), "user@example.com", ChannelEmail, "123456")
			if err == nil {
				t.Fatal("expected SMTP failure")
			}
			if tc.contain != "" && !strings.Contains(err.Error(), tc.contain) {
				t.Fatalf("expected error containing %q, got %v", tc.contain, err)
			}
		})
	}
}

func TestSMTPDialRequiresSTARTTLSForCredentialsCov(t *testing.T) {
	host, port := newCovFakeSMTP(t, covFakeSMTPOptions{})
	sender := &smtpVerificationSender{host: host, port: port, username: "user", password: "pass", from: "noreply@example.com"}
	err := sender.SendVerification(context.Background(), "user@example.com", ChannelEmail, "123456")
	if err == nil || !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("expected STARTTLS requirement error, got %v", err)
	}
}

func TestSMTPDialSTARTTLSFailureCov(t *testing.T) {
	host, port := newCovFakeSMTP(t, covFakeSMTPOptions{advertiseSTARTTLS: true})
	sender := &smtpVerificationSender{host: host, port: port, username: "user", password: "pass", from: "noreply@example.com"}
	err := sender.SendVerification(context.Background(), "user@example.com", ChannelEmail, "123456")
	if err == nil || !strings.Contains(err.Error(), "starttls") {
		t.Fatalf("expected starttls error, got %v", err)
	}
}
