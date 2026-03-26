// cloud/relay/auth_test.go
package relay

import (
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RealityLink-Tech/MoonHub/pkg/agentidentity"
)

func TestAuthenticator_VerifyValid(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	agentID := agentidentity.DeriveAgentID(pub)
	pubB64 := base64.StdEncoding.EncodeToString(pub)

	dirServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/agents/"+agentID+"/pubkey" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"agentID":"` + agentID + `","publicKey":"` + pubB64 + `"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer dirServer.Close()

	auth := NewAuthenticator(dirServer.URL)

	sig := ed25519.Sign(priv, []byte("test-challenge"))
	sigB64 := base64.StdEncoding.EncodeToString(sig)
	token := agentID + ":" + sigB64

	result, err := auth.Verify("Bearer "+token, "test-challenge")
	if err != nil {
		t.Fatal(err)
	}
	if result != agentID {
		t.Errorf("agentID = %q, want %q", result, agentID)
	}
}

func TestAuthenticator_InvalidBearer(t *testing.T) {
	auth := NewAuthenticator("http://localhost:9999")

	_, err := auth.Verify("Basic abc123", "challenge")
	if err == nil {
		t.Error("expected error for non-Bearer auth")
	}
}

func TestAuthenticator_InvalidFormat(t *testing.T) {
	auth := NewAuthenticator("http://localhost:9999")

	_, err := auth.Verify("Bearer invalid-no-colon", "challenge")
	if err == nil {
		t.Error("expected error for invalid token format")
	}
}
