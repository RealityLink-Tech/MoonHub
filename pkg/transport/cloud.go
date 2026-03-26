// pkg/transport/cloud.go
package transport

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/agentidentity"
)

// RegisterRequest is the body sent to POST /agents/register.
type RegisterRequest struct {
	AgentID   string `json:"agentID"`
	AgentName string `json:"agentName"`
	PublicKey string `json:"publicKey"`
	Endpoint  string `json:"endpoint"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

// AgentLookupResult is the response from GET /agents/:id.
type AgentLookupResult struct {
	AgentID              string `json:"agentID"`
	AgentName            string `json:"agentName"`
	Online               bool   `json:"online"`
	PublicKeyFingerprint string `json:"publicKeyFingerprint"`
	RelayEndpoint        string `json:"relayEndpoint"`
}

// CloudClient handles device-side interaction with the cloud directory.
type CloudClient struct {
	directoryURL string
	identity     *agentidentity.AgentIdentity
	httpClient   *http.Client
}

// NewCloudClient creates a new cloud directory client.
func NewCloudClient(directoryURL string, identity *agentidentity.AgentIdentity) *CloudClient {
	return &CloudClient{
		directoryURL: strings.TrimRight(directoryURL, "/"),
		identity:     identity,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Register registers this agent with the cloud directory.
func (c *CloudClient) Register(endpoint string) error {
	pubKeyB64 := base64.StdEncoding.EncodeToString(c.identity.PublicKeyBytes())
	timestamp := time.Now().Unix()

	msg := fmt.Sprintf("%s%s%s%d", c.identity.AgentID, c.identity.AgentName, pubKeyB64, timestamp)
	sig := c.identity.Sign([]byte(msg))

	req := RegisterRequest{
		AgentID:   c.identity.AgentID,
		AgentName: c.identity.AgentName,
		PublicKey: pubKeyB64,
		Endpoint:  endpoint,
		Timestamp: timestamp,
		Signature: base64.StdEncoding.EncodeToString(sig),
	}

	return c.postJSON("/agents/register", req)
}

// Heartbeat sends a heartbeat to the cloud directory.
func (c *CloudClient) Heartbeat() error {
	timestamp := time.Now().Unix()
	msg := fmt.Sprintf("%s%d", c.identity.AgentID, timestamp)
	sig := c.identity.Sign([]byte(msg))

	body := map[string]interface{}{
		"timestamp": timestamp,
		"signature": base64.StdEncoding.EncodeToString(sig),
	}

	return c.putJSON("/agents/"+c.identity.AgentID+"/heartbeat", body)
}

// Unregister removes this agent from the cloud directory.
func (c *CloudClient) Unregister() error {
	timestamp := time.Now().Unix()
	msg := fmt.Sprintf("%s%d", c.identity.AgentID, timestamp)
	sig := c.identity.Sign([]byte(msg))

	body := map[string]interface{}{
		"timestamp": timestamp,
		"signature": base64.StdEncoding.EncodeToString(sig),
	}

	reqBody, _ := json.Marshal(body)
	httpReq, _ := http.NewRequest("DELETE", c.directoryURL+"/agents/"+c.identity.AgentID, bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unregister: status %d", resp.StatusCode)
	}
	return nil
}

// LookupAgent queries the cloud directory for a remote agent.
func (c *CloudClient) LookupAgent(agentID string) (*AgentLookupResult, error) {
	resp, err := c.httpClient.Get(c.directoryURL + "/agents/" + agentID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lookup: status %d", resp.StatusCode)
	}

	var result AgentLookupResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPublicKey fetches a remote agent's public key from the directory.
func (c *CloudClient) GetPublicKey(agentID string) (ed25519.PublicKey, error) {
	resp, err := c.httpClient.Get(c.directoryURL + "/agents/" + agentID + "/pubkey")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get pubkey: status %d", resp.StatusCode)
	}

	var result struct {
		PublicKey string `json:"publicKey"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	keyBytes, err := base64.StdEncoding.DecodeString(result.PublicKey)
	if err != nil {
		return nil, err
	}
	return ed25519.PublicKey(keyBytes), nil
}

func (c *CloudClient) postJSON(path string, body interface{}) error {
	reqBody, _ := json.Marshal(body)
	resp, err := c.httpClient.Post(c.directoryURL+path, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("post %s: status %d, body: %s", path, resp.StatusCode, string(respBody))
	}
	return nil
}

func (c *CloudClient) putJSON(path string, body interface{}) error {
	reqBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("PUT", c.directoryURL+path, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("put %s: status %d", path, resp.StatusCode)
	}
	return nil
}

// StartHeartbeat starts a background goroutine that sends heartbeats at the configured interval.
// Call StopHeartbeat to stop it.
func (c *CloudClient) StartHeartbeat(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				c.Heartbeat()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// StopHeartbeat sends an unregister request and should be called on shutdown.
// Note: if using StartHeartbeat, cancel the context instead to stop the goroutine.
func (c *CloudClient) Stop() error {
	return c.Unregister()
}
