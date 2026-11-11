package server

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestAttachTokenRoundTripAndBinding(t *testing.T) {
	tokens, err := newAttachTokens(nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	tok := tokens.issue("sid-1")
	if err := tokens.verify("sid-1", tok); err != nil {
		t.Fatalf("fresh token rejected: %v", err)
	}
	if err := tokens.verify("sid-2", tok); err != errTokenInvalid {
		t.Fatalf("token for another sid must be invalid, got %v", err)
	}
}

func TestAttachTokenExpiry(t *testing.T) {
	tokens, _ := newAttachTokens(nil, time.Minute)
	now := time.Now()
	tokens.now = func() time.Time { return now }
	tok := tokens.issue("sid")

	tokens.now = func() time.Time { return now.Add(59 * time.Second) }
	if err := tokens.verify("sid", tok); err != nil {
		t.Fatalf("token within its ttl rejected: %v", err)
	}
	tokens.now = func() time.Time { return now.Add(61 * time.Second) }
	if err := tokens.verify("sid", tok); err != errTokenExpired {
		t.Fatalf("expected expiry, got %v", err)
	}
}

func TestAttachTokenRejectsTamperingAndGarbage(t *testing.T) {
	tokens, _ := newAttachTokens(nil, 0)
	tok := tokens.issue("sid")

	flipped := []byte(tok)
	at := len(flipped) - 10
	if flipped[at] == 'A' {
		flipped[at] = 'B'
	} else {
		flipped[at] = 'A'
	}
	for name, candidate := range map[string]string{
		"flipped mac":   string(flipped),
		"empty":         "",
		"not base64":    "!!!",
		"truncated":     tok[:len(tok)-4],
		"extended":      tok + "AAAA",
		"oversized":     strings.Repeat("A", 4096),
		"expiry bumped": "_" + tok[1:],
	} {
		if err := tokens.verify("sid", candidate); err == nil {
			t.Errorf("%s: token accepted", name)
		}
	}
}

func TestAttachTokenSecretHandling(t *testing.T) {
	secret := bytes.Repeat([]byte{7}, 32)
	a, err := newAttachTokens(secret, 0)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newAttachTokens(secret, 0)
	if err := b.verify("sid", a.issue("sid")); err != nil {
		t.Fatalf("instances sharing a secret must accept each other's tokens: %v", err)
	}

	other, _ := newAttachTokens(nil, 0)
	if err := other.verify("sid", a.issue("sid")); err == nil {
		t.Fatal("a different secret must reject the token")
	}

	if _, err := newAttachTokens([]byte("too short"), 0); err == nil {
		t.Fatal("short secrets must be rejected")
	}

	secret[0] = 9
	if err := a.verify("sid", a.issue("sid")); err != nil {
		t.Fatal("mutating the caller's secret must not affect issued tokens")
	}
}
