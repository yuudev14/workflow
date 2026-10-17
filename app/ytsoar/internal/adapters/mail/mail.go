// Package mail sends the invite and password reset emails. Without an SMTP host it writes
// them to the log, so dev, tests and installs with no mail server still work.
package mail

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/yuudev14/ytsoar/internal/application/auth"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// New returns an SMTP mailer, or a log mailer when cfg.Host is empty.
func New(log logger.Logger, cfg Config) auth.Mailer {
	if cfg.Host == "" {
		log.Warn("SMTP_HOST is empty: emails go to the log")
		return logMailer{logger: log}
	}
	return &smtpMailer{cfg: cfg}
}

type logMailer struct{ logger logger.Logger }

func (m logMailer) Send(_ context.Context, to, subject, body string) error {
	m.logger.Infow("email not sent: SMTP_HOST is empty", "to", to, "subject", subject, "body", body)
	return nil
}

type smtpMailer struct{ cfg Config }

// Send uses STARTTLS when the server offers it. PlainAuth refuses to send the password
// without TLS, except to localhost.
func (m *smtpMailer) Send(_ context.Context, to, subject, body string) error {
	// a header value with a line break could add headers, so refuse it
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("mail header contains a line break")
	}
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	msg := "From: " + m.cfg.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Date: " + time.Now().Format(time.RFC1123Z) + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		strings.ReplaceAll(body, "\n", "\r\n")
	addr := net.JoinHostPort(m.cfg.Host, fmt.Sprint(m.cfg.Port))
	return smtp.SendMail(addr, auth, m.cfg.From, []string{to}, []byte(msg))
}
