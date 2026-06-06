package services

// Internal tests (same package) so we can call unexported helpers like
// smtpDeliver and build a fake smtp.Client via net.Pipe.

import (
	"bufio"
	"fmt"
	"net"
	"net/smtp"
	"testing"
)

// fakeSMTPServer starts a goroutine that acts as a minimal SMTP server on the
// server-side of a net.Pipe. It responds positively to the EHLO handshake,
// then to MAIL FROM, RCPT TO, DATA, and the dot-terminated message body.
// The returned client-side net.Conn can be wrapped with smtp.NewClient.
func fakeSMTPServerConn(t *testing.T) net.Conn {
	t.Helper()
	server, client := net.Pipe()
	t.Cleanup(func() {
		server.Close()
		client.Close()
	})

	go func() {
		defer server.Close()
		rdr := bufio.NewReader(server)

		// Greeting
		fmt.Fprintf(server, "220 localhost ESMTP\r\n")

		// EHLO / HELO
		rdr.ReadString('\n')
		fmt.Fprintf(server, "250-localhost\r\n250 AUTH PLAIN\r\n")

		// MAIL FROM
		rdr.ReadString('\n')
		fmt.Fprintf(server, "250 OK\r\n")

		// RCPT TO
		rdr.ReadString('\n')
		fmt.Fprintf(server, "250 OK\r\n")

		// DATA command
		rdr.ReadString('\n')
		fmt.Fprintf(server, "354 Start input, end with <CRLF>.<CRLF>\r\n")

		// Read until the terminating ".\r\n"
		for {
			line, err := rdr.ReadString('\n')
			if err != nil {
				return
			}
			if line == ".\r\n" {
				break
			}
		}
		fmt.Fprintf(server, "250 OK\r\n")

		// QUIT (optional — client may just close)
		rdr.ReadString('\n')
		fmt.Fprintf(server, "221 Bye\r\n")
	}()

	return client
}

// TestSmtpDeliver_Success exercises the happy-path of smtpDeliver end-to-end
// using a net.Pipe fake SMTP server.
func TestSmtpDeliver_Success(t *testing.T) {
	clientConn := fakeSMTPServerConn(t)

	c, err := smtp.NewClient(clientConn, "localhost")
	if err != nil {
		t.Fatalf("smtp.NewClient: %v", err)
	}

	msg := buildMessage("from@example.com", "to@example.com", "Test", "Hello")
	if err := smtpDeliver(c, "from@example.com", "to@example.com", msg); err != nil {
		t.Errorf("smtpDeliver returned unexpected error: %v", err)
	}
}

// TestSmtpDeliver_MailFromError exercises the MAIL FROM error branch.
func TestSmtpDeliver_MailFromError(t *testing.T) {
	server, client := net.Pipe()
	t.Cleanup(func() { server.Close(); client.Close() })

	go func() {
		defer server.Close()
		rdr := bufio.NewReader(server)
		fmt.Fprintf(server, "220 localhost ESMTP\r\n")
		rdr.ReadString('\n') // EHLO
		fmt.Fprintf(server, "250-localhost\r\n250 AUTH PLAIN\r\n")
		rdr.ReadString('\n') // MAIL FROM
		fmt.Fprintf(server, "550 Rejected\r\n")
	}()

	c, err := smtp.NewClient(client, "localhost")
	if err != nil {
		t.Fatalf("smtp.NewClient: %v", err)
	}

	err = smtpDeliver(c, "from@example.com", "to@example.com", []byte("test"))
	if err == nil {
		t.Fatal("expected error from MAIL FROM rejection, got nil")
	}
}

// TestSmtpDeliver_RcptToError exercises the RCPT TO error branch.
func TestSmtpDeliver_RcptToError(t *testing.T) {
	server, client := net.Pipe()
	t.Cleanup(func() { server.Close(); client.Close() })

	go func() {
		defer server.Close()
		rdr := bufio.NewReader(server)
		fmt.Fprintf(server, "220 localhost ESMTP\r\n")
		rdr.ReadString('\n') // EHLO
		fmt.Fprintf(server, "250-localhost\r\n250 AUTH PLAIN\r\n")
		rdr.ReadString('\n') // MAIL FROM
		fmt.Fprintf(server, "250 OK\r\n")
		rdr.ReadString('\n') // RCPT TO
		fmt.Fprintf(server, "550 No such user\r\n")
	}()

	c, err := smtp.NewClient(client, "localhost")
	if err != nil {
		t.Fatalf("smtp.NewClient: %v", err)
	}

	err = smtpDeliver(c, "from@example.com", "bad@example.com", []byte("test"))
	if err == nil {
		t.Fatal("expected error from RCPT TO rejection, got nil")
	}
}

// TestSendSTARTTLS_DialSucceedsNoTLSSupport tests the sendSTARTTLS path where
// smtp.Dial succeeds (real TCP connection) but StartTLS fails because the fake
// server responds with a 502 error to the STARTTLS command.
func TestSendSTARTTLS_DialSucceedsNoTLSSupport(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	port := ln.Addr().(*net.TCPAddr).Port

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		rdr := bufio.NewReader(conn)
		// Greeting
		fmt.Fprintf(conn, "220 localhost ESMTP\r\n")
		// EHLO — respond without STARTTLS capability
		rdr.ReadString('\n')
		fmt.Fprintf(conn, "250-localhost\r\n250 OK\r\n")
		// STARTTLS command — respond with 502 to cause failure
		rdr.ReadString('\n')
		fmt.Fprintf(conn, "502 Command not implemented\r\n")
	}()

	t.Setenv("SMTP_HOST", "127.0.0.1")
	t.Setenv("SMTP_PORT", fmt.Sprintf("%d", port))
	t.Setenv("SMTP_USER", "user@example.com")
	t.Setenv("SMTP_PASS", "password")
	t.Setenv("SMTP_FROM", "noreply@example.com")

	svc := &EmailService{}
	err = svc.SendVerificationEmail("to@example.com", "Test", "https://example.com/verify")
	// We expect an error (StartTLS not supported), but no panic.
	if err == nil {
		t.Fatal("expected an error from StartTLS failure, got nil")
	}
}
