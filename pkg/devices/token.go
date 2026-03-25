package devices

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"time"
)

const (
	// TokenValidity is how long a token is valid (30 days).
	TokenValidity = 30 * 24 * time.Hour

	// TokenLength is the number of random bytes in a token.
	TokenLength = 32
)

// GenerateToken creates a new random access token (64-char lowercase hex).
func GenerateToken() (string, time.Time, error) {
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", time.Time{}, err
	}

	token := hex.EncodeToString(bytes)
	expiresAt := time.Now().Add(TokenValidity)

	return token, expiresAt, nil
}

// ValidateToken checks if a token string is plausibly formatted (hex or legacy base64).
// Actual authorization uses DeviceStore.ValidateToken against stored records.
func ValidateToken(token string) bool {
	if len(token) == hex.EncodedLen(TokenLength) {
		_, err := hex.DecodeString(token)
		return err == nil
	}

	if len(token) < 10 {
		return false
	}

	_, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		_, err = base64.RawURLEncoding.DecodeString(token)
		return err == nil
	}

	return true
}
