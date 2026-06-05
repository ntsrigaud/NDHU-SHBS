package util_test

import (
	"testing"

	"shbs-server/pkg/util"
)

// ── HashToken ─────────────────────────────────────────────────────────────────

func TestHashToken_Deterministic(t *testing.T) {
	raw := "my-super-secret-jwt-token"
	h1 := util.HashToken(raw)
	h2 := util.HashToken(raw)
	if h1 != h2 {
		t.Error("HashToken is not deterministic: two calls with same input returned different results")
	}
}

func TestHashToken_DifferentInputs(t *testing.T) {
	h1 := util.HashToken("token-a")
	h2 := util.HashToken("token-b")
	if h1 == h2 {
		t.Error("HashToken: different inputs should produce different hashes")
	}
}

func TestHashToken_IsHex(t *testing.T) {
	h := util.HashToken("some-token")
	// SHA-256 hex digest is always 64 lowercase hex characters.
	if len(h) != 64 {
		t.Errorf("HashToken returned %d chars, want 64", len(h))
	}
	for _, c := range h {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("HashToken returned non-hex character: %c", c)
		}
	}
}

func TestHashToken_NotPlaintext(t *testing.T) {
	raw := "my-super-secret-jwt-token"
	h := util.HashToken(raw)
	if h == raw {
		t.Error("HashToken returned the raw token — hashing did not occur")
	}
}
