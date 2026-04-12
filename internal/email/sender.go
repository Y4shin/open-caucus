// Package email provides an interface for sending emails with pluggable backends.
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/Y4shin/open-caucus/internal/config"
)

// SendOptions holds optional email headers and attachments.
type SendOptions struct {
	ICSData    string   // optional iCalendar attachment content
	MessageID  string   // RFC 5322 Message-ID (e.g. "<uuid@domain>")
	References []string // RFC 5322 References header for threading
}

// Sender sends emails.
type Sender interface {
	Send(ctx context.Context, to, subject, htmlBody, textBody string, opts *SendOptions) error
	Enabled() bool
	FromDomain() string // domain part of the configured from address
}

// NewSender returns an appropriate Sender based on the configuration.
func NewSender(cfg *config.EmailConfig) Sender {
	if cfg == nil || !cfg.Enabled || cfg.SMTPHost == "" {
		return &NoopSender{}
	}
	return &SMTPSender{cfg: cfg}
}

// SMTPSender sends emails via SMTP with STARTTLS.
type SMTPSender struct {
	cfg *config.EmailConfig
}

func (s *SMTPSender) Enabled() bool { return true }

func (s *SMTPSender) FromDomain() string {
	if idx := strings.LastIndex(s.cfg.FromAddress, "@"); idx >= 0 {
		return s.cfg.FromAddress[idx+1:]
	}
	return "localhost"
}

func (s *SMTPSender) Send(_ context.Context, to, subject, htmlBody, textBody string, opts *SendOptions) error {
	addr := net.JoinHostPort(s.cfg.SMTPHost, fmt.Sprintf("%d", s.cfg.SMTPPort))

	from := s.cfg.FromAddress
	fromHeader := from
	if s.cfg.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", s.cfg.FromName, from)
	}

	hasICS := opts != nil && opts.ICSData != ""
	altBoundary := "----=_Alt_OpenCaucus"
	mixedBoundary := "----=_Mixed_OpenCaucus"

	var msg strings.Builder
	fmt.Fprintf(&msg, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	fmt.Fprintf(&msg, "From: %s\r\n", fromHeader)
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", subject)
	if opts != nil && opts.MessageID != "" {
		fmt.Fprintf(&msg, "Message-ID: %s\r\n", opts.MessageID)
	}
	if opts != nil && len(opts.References) > 0 {
		fmt.Fprintf(&msg, "References: %s\r\n", strings.Join(opts.References, " "))
	}
	fmt.Fprintf(&msg, "MIME-Version: 1.0\r\n")

	if hasICS {
		// multipart/mixed: body + ICS attachment
		fmt.Fprintf(&msg, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n", mixedBoundary)
		fmt.Fprintf(&msg, "\r\n")
		fmt.Fprintf(&msg, "--%s\r\n", mixedBoundary)
		fmt.Fprintf(&msg, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n", altBoundary)
		fmt.Fprintf(&msg, "\r\n")
	} else {
		fmt.Fprintf(&msg, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n", altBoundary)
		fmt.Fprintf(&msg, "\r\n")
	}

	// Plain text part
	fmt.Fprintf(&msg, "--%s\r\n", altBoundary)
	fmt.Fprintf(&msg, "Content-Type: text/plain; charset=utf-8\r\n\r\n")
	fmt.Fprintf(&msg, "%s\r\n", textBody)
	// HTML part
	fmt.Fprintf(&msg, "--%s\r\n", altBoundary)
	fmt.Fprintf(&msg, "Content-Type: text/html; charset=utf-8\r\n\r\n")
	fmt.Fprintf(&msg, "%s\r\n", htmlBody)
	fmt.Fprintf(&msg, "--%s--\r\n", altBoundary)

	if hasICS {
		// ICS calendar attachment
		fmt.Fprintf(&msg, "--%s\r\n", mixedBoundary)
		fmt.Fprintf(&msg, "Content-Type: text/calendar; charset=utf-8; method=REQUEST\r\n")
		fmt.Fprintf(&msg, "Content-Disposition: attachment; filename=\"invite.ics\"\r\n\r\n")
		fmt.Fprintf(&msg, "%s\r\n", opts.ICSData)
		fmt.Fprintf(&msg, "--%s--\r\n", mixedBoundary)
	}

	// Connect and send.
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("email dial: %w", err)
	}
	client, err := smtp.NewClient(conn, s.cfg.SMTPHost)
	if err != nil {
		conn.Close()
		return fmt.Errorf("email client: %w", err)
	}
	defer client.Close()

	// STARTTLS if supported.
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.SMTPHost}); err != nil {
			return fmt.Errorf("email starttls: %w", err)
		}
	}

	// Authenticate if credentials provided.
	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("email auth: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("email mail: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("email rcpt: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("email data: %w", err)
	}
	if _, err := w.Write([]byte(msg.String())); err != nil {
		return fmt.Errorf("email write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("email close: %w", err)
	}
	return client.Quit()
}

// NoopSender discards all emails. Used when email is not configured.
type NoopSender struct{}

func (s *NoopSender) Enabled() bool    { return false }
func (s *NoopSender) FromDomain() string { return "localhost" }
func (s *NoopSender) Send(_ context.Context, _, _, _, _ string, _ *SendOptions) error {
	return fmt.Errorf("email sending is not configured")
}

// MockSender records sent emails for testing.
type MockSender struct {
	Sent []MockEmail
}

// MockEmail represents a single sent email.
type MockEmail struct {
	To         string
	Subject    string
	HTMLBody   string
	TextBody   string
	ICSData    string
	MessageID  string
	References []string
}

func (s *MockSender) Enabled() bool    { return true }
func (s *MockSender) FromDomain() string { return "test.local" }

func (s *MockSender) Send(_ context.Context, to, subject, htmlBody, textBody string, opts *SendOptions) error {
	e := MockEmail{To: to, Subject: subject, HTMLBody: htmlBody, TextBody: textBody}
	if opts != nil {
		e.ICSData = opts.ICSData
		e.MessageID = opts.MessageID
		e.References = opts.References
	}
	s.Sent = append(s.Sent, e)
	return nil
}
