package auth

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

// SMTP 验证码发信（verificationSender 的生产实现）。注册/忘记密码等邮箱
// 验证码经 SMTP_HOST 指定的网关投递；未配置 SMTP_HOST 时不启用，sender
// 选择仍走 noop（生产）/log（dev/test），见 defaultVerificationSender。
// 端口 465 走隐式 TLS，其余端口走 STARTTLS——拒绝明文会话，凭证只在
// 加密通道上传输。SMS 通道无网关，返回与 noop 一致的不可用错误。

const smtpDialTimeout = 15 * time.Second

// smtpSessionTimeout 是单次 SMTP 会话（含 STARTTLS/AUTH/MAIL/RCPT/DATA/QUIT）
// 的整体 deadline（审查 P2-4）：此前只有 dial 有超时，认证后的协议交互无
// deadline——慢/坏网关可把请求 goroutine 无限挂住（HTTP WriteTimeout 不杀
// handler），持续占用连接与内存。
const smtpSessionTimeout = 20 * time.Second

// ErrSMSChannelUnavailable：短信通道未接网关。对外复用与配置缺失一致的
// 业务错误码（10600 验证码服务暂不可用），不向客户端泄露细节。
var ErrSMSChannelUnavailable = errors.New("auth: sms gateway not configured")

// smtpVerificationSender 通过 SMTP 投递邮箱验证码。
type smtpVerificationSender struct {
	host     string
	port     string
	username string
	password string
	from     string
}

// newSMTPVerificationSender 仅在 SMTPHost 已配置时返回实现，否则返回 nil。
// 端口空值在此兜底为 587（STARTTLS），不依赖 Load() 的默认值注入顺序。
func newSMTPVerificationSender(cfg config.Config) *smtpVerificationSender {
	if cfg.SMTPHost == "" {
		return nil
	}
	port := cfg.SMTPPort
	if port == "" {
		port = "587"
	}
	return &smtpVerificationSender{
		host:     cfg.SMTPHost,
		port:     port,
		username: cfg.SMTPUsername,
		password: cfg.SMTPPassword,
		from:     cfg.SMTPFrom,
	}
}

func (s *smtpVerificationSender) SendVerification(ctx context.Context, to string, channel VerificationChannel, code string) error {
	if channel != ChannelEmail {
		// detectChannel 对全数字 login_id 判为 SMS；当前无短信网关。
		return ErrSMSChannelUnavailable
	}
	return s.sendMail(ctx, to, verificationEmailBody(code))
}

// dial 建立与 SMTP 服务器的会话：465 隐式 TLS 直连，其余端口明文拨号后
// 升级 STARTTLS。安全边界在凭证：配置了 SMTP_USERNAME 时必须走加密通道
// （无 STARTTLS 即失败），凭证不落明文；无凭证的本地捕获服务器（mailpit
// 等）明文放行。
func (s *smtpVerificationSender) dial(ctx context.Context) (*smtp.Client, error) {
	addr := net.JoinHostPort(s.host, s.port)
	host := s.host
	d := net.Dialer{Timeout: smtpDialTimeout}
	if s.port == "465" {
		conn, err := tls.DialWithDialer(&d, "tcp", addr, &tls.Config{ServerName: host})
		if err != nil {
			return nil, fmt.Errorf("auth: smtp tls dial: %w", err)
		}
		// 整体会话 deadline（见 smtpSessionTimeout 注释）：未设为单次协议交互
		// 兜底，避免坏网关挂住 goroutine。
		if err := conn.SetDeadline(time.Now().Add(smtpSessionTimeout)); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("auth: smtp set deadline: %w", err)
		}
		return smtp.NewClient(conn, host)
	}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("auth: smtp dial: %w", err)
	}
	// 同 465 分支：deadline 设在底层 TCP conn 上，后续 STARTTLS 包出的
	// tls.Conn 仍继承该 deadline。
	if err := conn.SetDeadline(time.Now().Add(smtpSessionTimeout)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("auth: smtp set deadline: %w", err)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("auth: smtp client: %w", err)
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			_ = c.Close()
			return nil, fmt.Errorf("auth: smtp starttls: %w", err)
		}
	} else if s.username != "" {
		// 凭证只在加密通道上传输：配置了账号却无 STARTTLS 时拒绝，
		// 不回退明文认证。无凭证的本地捕获服务器（mailpit 等）明文放行。
		_ = c.Quit()
		return nil, errors.New("auth: smtp server does not support STARTTLS (required when credentials are set)")
	}
	return c, nil
}

func (s *smtpVerificationSender) sendMail(ctx context.Context, to, htmlBody string) error {
	c, err := s.dial(ctx)
	if err != nil {
		return err
	}
	// Quit 成功后 Close 幂等兜底；任何中间步骤出错时 Close 释放连接。
	defer c.Close()

	if s.username != "" {
		auth := smtp.PlainAuth("", s.username, s.password, s.host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("auth: smtp auth: %w", err)
		}
	}
	// to 来自 login_id，注册入口已用 isValidEmailLoginID 拒绝空白/控制字符；
	// net/smtp 的 Mail/Rcpt 内部 validateLine 亦二次拒绝 CR/LF（无头注入可能）。
	if err := c.Mail(smtpEnvelopeFrom(s.from)); err != nil { // #nosec G707 -- to 经 isValidEmailLoginID 拒绝空白/控制字符，net/smtp validateLine 二次拒绝 CR/LF
		return fmt.Errorf("auth: smtp mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil { // #nosec G707 -- 同上
		return fmt.Errorf("auth: smtp rcpt to: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("auth: smtp data: %w", err)
	}
	if _, err := w.Write([]byte(verificationEmailMessage(s.from, to, htmlBody))); err != nil {
		_ = w.Close()
		return fmt.Errorf("auth: smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("auth: smtp close data: %w", err)
	}
	return c.Quit()
}

// verificationEmailMessage 构造 RFC 5322 邮件：中文主题按 RFC 2047 Q 编码，
// 正文单部分 text/html（验证码大号展示）经 base64 传输保证传输层透明。
func verificationEmailMessage(from, to, htmlBody string) string {
	subject := mime.QEncoding.Encode("utf-8", "HotSpot 验证码")
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n")
	b.WriteString("\r\n")
	b.WriteString(wrapBase64Lines(base64.StdEncoding.EncodeToString([]byte(htmlBody))))
	return b.String()
}

// wrapBase64Lines 按 RFC 2045 将 base64 正文折行为每行 ≤76 字符。
func wrapBase64Lines(s string) string {
	const lineLen = 76
	var b strings.Builder
	for len(s) > lineLen {
		b.WriteString(s[:lineLen])
		b.WriteString("\r\n")
		s = s[lineLen:]
	}
	b.WriteString(s)
	return b.String()
}

// verificationEmailBody 渲染验证码邮件正文。行内样式兼容各家 Webmail 的
// 样式裁剪；有效期文案须与 CreateRegisterCode 的 10 分钟 TTL 一致。
func verificationEmailBody(code string) string {
	return `<!doctype html>
<html lang="zh-CN">
<body style="margin:0;padding:0;background:#f4f6fa;">
  <div style="max-width:520px;margin:0 auto;padding:32px 24px;font-family:-apple-system,'PingFang SC','Microsoft YaHei',sans-serif;color:#1c2333;">
    <div style="font-size:20px;font-weight:700;letter-spacing:1px;">HotSpot</div>
    <div style="margin-top:24px;background:#ffffff;border-radius:12px;padding:32px;">
      <p style="margin:0 0 8px;font-size:16px;font-weight:600;">你的验证码</p>
      <p style="margin:0 0 20px;font-size:14px;color:#5a6478;">请在下方输入验证码完成验证：</p>
      <div style="font-size:36px;font-weight:700;letter-spacing:10px;text-align:center;padding:18px 0;background:#f4f6fa;border-radius:8px;">` + code + `</div>
      <p style="margin:20px 0 0;font-size:13px;color:#5a6478;">验证码 10 分钟内有效。如非本人操作，请忽略本邮件，切勿将验证码告知他人。</p>
    </div>
    <p style="margin:16px 0 0;font-size:12px;color:#9aa3b5;">此邮件由 HotSpot 系统发送，请勿直接回复。</p>
  </div>
</body>
</html>`
}

// smtpEnvelopeFrom 从「展示名 <addr>」形态提取裸地址（SMTP 信封要求）；
// 解析失败原样返回，交由 SMTP 服务器报错暴露配置问题。
func smtpEnvelopeFrom(from string) string {
	open := strings.LastIndex(from, "<")
	closeIdx := strings.LastIndex(from, ">")
	if open >= 0 && closeIdx > open {
		return strings.TrimSpace(from[open+1 : closeIdx])
	}
	return strings.TrimSpace(from)
}
