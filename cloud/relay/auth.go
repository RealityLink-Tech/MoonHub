// cloud/relay/auth.go
package relay

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Authenticator verifies agent WebSocket connections against the directory.
type Authenticator struct {
	directoryURL string
	httpClient   *http.Client
}

// NewAuthenticator creates a new relay authenticator.
func NewAuthenticator(directoryURL string) *Authenticator {
	return &Authenticator{
		directoryURL: strings.TrimRight(directoryURL, "/"),
		httpClient:   &http.Client{Timeout: 5 * time.Second},
	}
}

// Verify checks a Bearer token against the directory.
// The challenge should include a timestamp to prevent replay attacks.
// Returns the verified agentID on success.
func (a *Authenticator) Verify(authHeader, challenge string) (string, error) {
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("unsupported auth scheme")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid token format")
	}

	agentID := parts[0]
	sigB64 := parts[1]

	sigBytes, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return "", fmt.Errorf("invalid signature encoding: %w", err)
	}

	pubKey, err := a.fetchPublicKey(agentID)
	if err != nil {
		return "", fmt.Errorf("fetch public key: %w", err)
	}

	if !ed25519.Verify(pubKey, []byte(challenge), sigBytes) {
		return "", fmt.Errorf("invalid signature")
	}

	return agentID, nil
}

// GenerateChallenge creates a unique challenge string for authentication.
// The challenge includes a timestamp to prevent replay attacks.
func GenerateChallenge() string {
	return fmt.Sprintf("relay-auth-%d", time.Now().UnixNano())
}

func (a *Authenticator) fetchPublicKey(agentID string) (ed25519.PublicKey, error) {
	resp, err := a.httpClient.Get(a.directoryURL + "/agents/" + agentID + "/pubkey")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("directory returned %d", resp.StatusCode)
	}

	var result struct {
		PublicKey string `json:"publicKey"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	keyBytes, err := base64.StdEncoding.DecodeString(result.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public key encoding: %w", err)
	}

	return ed25519.PublicKey(keyBytes), nil
}
