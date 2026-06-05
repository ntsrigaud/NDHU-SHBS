package util_test

import (
	"shbs-server/pkg/util"
	"testing"
)

// ── HashPassword / VerifyPassword ─────────────────────────────────────────────

func TestHashPassword_Roundtrip(t *testing.T) {
	plain := "SecurePass1!"
	hash, err := util.HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty string")
	}
	if err := util.VerifyPassword(hash, plain); err != nil {
		t.Errorf("VerifyPassword failed for correct password: %v", err)
	}
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	hash, _ := util.HashPassword("CorrectPass1!")
	if err := util.VerifyPassword(hash, "WrongPass999!"); err == nil {
		t.Fatal("Expected error for wrong password, got nil")
	}
}

// ── IsStrongPassword ──────────────────────────────────────────────────────────

func TestIsStrongPassword(t *testing.T) {
	cases := []struct {
		password string
		want     bool
	}{
		{"SecurePass1!", true},
		{"short1!", false},             // too short
		{"alllowercase1!", false},      // no uppercase
		{"ALLUPPERCASE1!", false},      // no lowercase
		{"NoDigitPass!", false},        // no digit
		{"NoSpecial123", false},        // no special char
		{"Valid@Passw0rd", true},
		{"", false},
	}
	for _, c := range cases {
		if got := util.IsStrongPassword(c.password); got != c.want {
			t.Errorf("IsStrongPassword(%q) = %v, want %v", c.password, got, c.want)
		}
	}
}
