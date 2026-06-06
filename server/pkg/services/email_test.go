package services_test

import (
	"strings"
	"testing"

	"shbs-server/pkg/services"
)

// TestSendVerificationEmail_FailsWhenSMTPUnreachable_ImplicitTLS verifies that
// SendVerificationEmail returns an error (not a panic) when the SMTP server is
// unreachable and port 465 (implicit TLS) is configured. This exercises the
// send → sendImplicitTLS → tls.Dial error path.
func TestSendVerificationEmail_FailsWhenSMTPUnreachable_ImplicitTLS(t *testing.T) {
	t.Setenv("SMTP_HOST", "127.0.0.1")
	t.Setenv("SMTP_PORT", "465")
	t.Setenv("SMTP_USER", "test@example.com")
	t.Setenv("SMTP_PASS", "password")
	t.Setenv("SMTP_FROM", "noreply@example.com")

	svc := &services.EmailService{}
	err := svc.SendVerificationEmail("user@example.com", "Test User", "https://example.com/verify?token=abc")
	if err == nil {
		t.Fatal("expected an error when SMTP is unreachable on port 465, got nil")
	}
}

// TestSendVerificationEmail_FailsWhenSMTPUnreachable_STARTTLS verifies that
// SendVerificationEmail returns an error when the SMTP server is unreachable
// and port 587 (STARTTLS) is configured. This exercises the
// send → sendSTARTTLS → smtp.Dial error path.
func TestSendVerificationEmail_FailsWhenSMTPUnreachable_STARTTLS(t *testing.T) {
	t.Setenv("SMTP_HOST", "127.0.0.1")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USER", "test@example.com")
	t.Setenv("SMTP_PASS", "password")
	t.Setenv("SMTP_FROM", "noreply@example.com")

	svc := &services.EmailService{}
	err := svc.SendVerificationEmail("user@example.com", "Test User", "https://example.com/verify?token=abc")
	if err == nil {
		t.Fatal("expected an error when SMTP is unreachable on port 587, got nil")
	}
}

// TestSendVerificationEmail_ErrorContainsContext verifies that the returned
// error message is descriptive (contains "smtp" context) so operators can
// diagnose SMTP failures quickly.
func TestSendVerificationEmail_ErrorContainsContext(t *testing.T) {
	t.Setenv("SMTP_HOST", "127.0.0.1")
	t.Setenv("SMTP_PORT", "465")
	t.Setenv("SMTP_USER", "test@example.com")
	t.Setenv("SMTP_PASS", "password")
	t.Setenv("SMTP_FROM", "noreply@example.com")

	svc := &services.EmailService{}
	err := svc.SendVerificationEmail("user@example.com", "Test User", "https://example.com/verify")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "smtp") {
		t.Errorf("expected error to mention 'smtp', got: %s", err.Error())
	}
}
