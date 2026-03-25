package devices

import (
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	token, expiresAt, err := GenerateToken()
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("expected non-empty token")
	}
	if len(token) != 64 {
		t.Errorf("expected 64-char hex token, got len=%d", len(token))
	}

	if expiresAt.Before(time.Now()) {
		t.Error("expected expiry time to be in the future")
	}

	// Token should be valid for approximately 30 days
	expectedExpiry := time.Now().Add(TokenValidity)
	diff := expectedExpiry.Sub(expiresAt)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("expected expiry around %v, got %v", expectedExpiry, expiresAt)
	}
}

func TestGenerateTokenUniqueness(t *testing.T) {
	tokens := make(map[string]bool)

	for i := 0; i < 100; i++ {
		token, _, err := GenerateToken()
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		if tokens[token] {
			t.Errorf("duplicate token generated: %s", token)
		}
		tokens[token] = true
	}
}

func TestValidateTokenFormat(t *testing.T) {
	// Valid token
	token, _, err := GenerateToken()
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if !ValidateToken(token) {
		t.Error("expected token to be valid")
	}

	// Invalid tokens
	invalidTokens := []string{
		"",
		"short",
		"not-base64!@#$",
		"====",
	}

	for _, invalid := range invalidTokens {
		if ValidateToken(invalid) {
			t.Errorf("expected token %q to be invalid", invalid)
		}
	}
}
