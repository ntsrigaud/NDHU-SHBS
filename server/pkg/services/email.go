package services

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
)

// EmailService sends transactional emails via SMTP.
// Config is read from env at send-time (not at struct creation) so it works
// across the full app lifetime and is easy to override in tests.
type EmailService struct{}

// SendVerificationEmail sends the account-verification link to a new user.
func (e *EmailService) SendVerificationEmail(toEmail, name, verificationURL string) error {
	subject := "Verify your NDHU SHBS account"
	body := fmt.Sprintf(`Hello %s,

Please verify your email address by clicking the link below:

%s

This link expires in 24 hours. If you did not create an account, ignore this email.

— NDHU Second-Hand Book Store`, name, verificationURL)

	return e.send(toEmail, subject, body)
}

// send dispatches to the correct TLS strategy based on SMTP_PORT.
func (e *EmailService) send(to, subject, body string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")

	msg := buildMessage(from, to, subject, body)

	if port == "465" {
		return e.sendImplicitTLS(host, port, user, pass, from, to, msg)
	}
	return e.sendSTARTTLS(host, port, user, pass, from, to, msg)
}

// sendImplicitTLS handles port 465 (SMTPS). Go's net/smtp only does STARTTLS
// natively, so for implicit TLS we dial with crypto/tls first, then wrap it.
func (e *EmailService) sendImplicitTLS(host, port, user, pass, from, to string, msg []byte) error {
	conn, err := tls.Dial("tcp", host+":"+port, &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return fmt.Errorf("smtp dial (implicit TLS): %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(smtp.PlainAuth("", user, pass, host)); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	return smtpDeliver(client, from, to, msg)
}

// sendSTARTTLS handles port 587 — plain dial then upgrade via STARTTLS.
func (e *EmailService) sendSTARTTLS(host, port, user, pass, from, to string, msg []byte) error {
	client, err := smtp.Dial(host + ":" + port)
	if err != nil {
		return fmt.Errorf("smtp dial (STARTTLS): %w", err)
	}
	defer client.Close()

	if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
		return fmt.Errorf("smtp starttls: %w", err)
	}
	if err := client.Auth(smtp.PlainAuth("", user, pass, host)); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	return smtpDeliver(client, from, to, msg)
}

func smtpDeliver(client *smtp.Client, from, to string, msg []byte) error {
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	return w.Close()
}

// buildMessage constructs a minimal RFC 2822 plain-text email.
func buildMessage(from, to, subject, body string) []byte {
	return []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body,
	))
}
