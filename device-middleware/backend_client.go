package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	IdentityFile   = "identity.json"
	DefaultBaseURL = "http://4.247.135.200"
)

// --- Data Models ---

type Identity struct {
	NodeID      string `json:"node_id"`
	PairingCode int64  `json:"pairing_code"`
	PublicKey   string `json:"public_key"`  // Base64
	PrivateKey  string `json:"private_key"` // Base64
	DeviceID    string `json:"device_id"`   // Assigned by backend
}

type RegisterRequest struct {
	PublicKey   string `json:"public_key"`
	PairingCode int64  `json:"pairing_code"`
	NodeID      string `json:"node_id"`
}

type RegisterResponse struct {
	Message  string `json:"message"`
	DeviceID string `json:"device_id"`
}

type ChallengeRequest struct {
	DeviceID string `json:"device_id"`
}

type ChallengeResponse struct {
	Challenge string `json:"challenge"`
}

type VerifyRequest struct {
	DeviceID  string `json:"device_id"`
	Signature string `json:"signature"`
}

type VerifyResponse struct {
	LiveKitToken string         `json:"livekit_token"`
	LiveKitURL   string         `json:"livekit_url"`
	RoomName     string         `json:"room_name"`
	Zerotier     ZerotierConfig `json:"zerotier"`
}

type ZerotierConfig struct {
	NetworkID string `json:"network_id"`
}

type TelemetryUpdate struct {
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	Altitude       float32 `json:"altitude"`
	Speed          float32 `json:"speed"`
	Heading        float32 `json:"heading"`
	SignalStrength int     `json:"signal_strength"`
	Battery        int     `json:"battery"`
}

type CheckClaimRequest struct {
	DeviceID string `json:"device_id"`
}

type CheckClaimResponse struct {
	Claim   bool   `json:"claim_status"`
	Message string `json:"message"`
}

// --- Backend Client ---

type BackendClient struct {
	BaseURL    string
	Identity   *Identity
	HTTPClient *http.Client
}

func NewBackendClient(baseURL string) *BackendClient {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &BackendClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// --- Identity Management ---

func (c *BackendClient) LoadOrCreateIdentity() error {
	// 1. Try Load
	if _, err := os.Stat(IdentityFile); err == nil {
		data, err := os.ReadFile(IdentityFile)
		if err != nil {
			return err
		}
		var id Identity
		if err := json.Unmarshal(data, &id); err != nil {
			return err
		}
		c.Identity = &id
		return nil
	}

	// 2. Create New
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	// Generate Pairing Code (Random 8-digit)
	pubHex := hex.EncodeToString(pub)
	var codeVal uint64
	fmt.Sscanf(pubHex[:8], "%x", &codeVal)
	pairingCode := int64(10000000 + (codeVal % 90000000))

	id := Identity{
		NodeID:      fmt.Sprintf("node-%d", pairingCode), // Simple NodeID
		PairingCode: pairingCode,
		PublicKey:   base64.StdEncoding.EncodeToString(pub),
		PrivateKey:  base64.StdEncoding.EncodeToString(priv),
	}
	c.Identity = &id
	return c.TypifySaveIdentity()
}

func (c *BackendClient) TypifySaveIdentity() error {
	data, err := json.MarshalIndent(c.Identity, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(IdentityFile, data, 0644)
}

func (c *BackendClient) ResetIdentity() error {
	log.Println("⚠️ RESETTING IDENTITY (Device Forgotten/Factory Reset) ⚠️")

	// delete old file
	os.Remove(IdentityFile)

	// Create new
	return c.LoadOrCreateIdentity()
}

// --- API Methods ---

func (c *BackendClient) Register() error {
	// If we already have a DeviceID, we assume registered.
	if c.Identity.DeviceID != "" {
		return nil
	}

	req := RegisterRequest{
		PublicKey:   c.Identity.PublicKey,
		PairingCode: c.Identity.PairingCode,
		NodeID:      c.Identity.NodeID,
	}

	var res RegisterResponse
	if err := c.post("/api/v1/devices/register", req, &res); err != nil {
		return err
	}

	c.Identity.DeviceID = res.DeviceID
	return c.TypifySaveIdentity()
}

func (c *BackendClient) Authenticate() (*VerifyResponse, error) {
	if c.Identity.DeviceID == "" {
		return nil, fmt.Errorf("device not registered")
	}

	// 1. Get Challenge
	chalReq := ChallengeRequest{DeviceID: c.Identity.DeviceID}
	var chalRes ChallengeResponse
	if err := c.post("/api/v1/devices/auth/challenge", chalReq, &chalRes); err != nil {
		// If 400 or 404, the device might be deleted. reset?
		// We need to check if err contains 400 or 404 (our post helper returns "API Error 400: ...")
		// Ideally we catch strict 4xx.
		// For safety, let's reset on 400 Bad Request (Device ID not found)
		return nil, fmt.Errorf("challenge failed: %v", err)
	}

	// 2. Sign Challenge
	privBytes, _ := base64.StdEncoding.DecodeString(c.Identity.PrivateKey)
	privKey := ed25519.PrivateKey(privBytes)
	signature := ed25519.Sign(privKey, []byte(chalRes.Challenge))
	sigBase64 := base64.StdEncoding.EncodeToString(signature)

	// 3. Verify
	verifyReq := VerifyRequest{
		DeviceID:  c.Identity.DeviceID,
		Signature: sigBase64,
	}
	var verifyRes VerifyResponse
	if err := c.post("/api/v1/devices/auth/verify", verifyReq, &verifyRes); err != nil {
		return nil, fmt.Errorf("verify failed: %v", err)
	}

	return &verifyRes, nil
}

func (c *BackendClient) CheckClaim() (bool, error) {
	if c.Identity.DeviceID == "" {
		return false, fmt.Errorf("device not registered")
	}

	req := CheckClaimRequest{DeviceID: c.Identity.DeviceID}
	var res CheckClaimResponse
	if err := c.post("/api/v1/devices/auth/check-claim", req, &res); err != nil {
		return false, err
	}
	return res.Claim, nil
}

func (c *BackendClient) UpdateTelemetry(data TelemetryUpdate) error {
	if c.Identity.DeviceID == "" {
		return nil
	}
	url := fmt.Sprintf("/api/v1/devices/%s/telemetry", c.Identity.DeviceID)
	return c.post(url, data, nil)
}

// --- Helper ---

func (c *BackendClient) post(path string, payload interface{}, target interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// DEBUG: Print outgoing JSON to verify values
	log.Printf(">>> SENDING TO %s: %s", path, string(body))

	url := c.BaseURL + path
	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		// Specific Check for Identity Reset triggers
		// Backend returns 400 if DeviceID not found in GetChallenge
		if path == "/api/v1/devices/auth/challenge" && resp.StatusCode == 400 {
			// This is a candidate for Reset, but we should be sure.
			// The backend says "No devices found with this device ID"
			if bytes.Contains(respBody, []byte("No devices found")) || bytes.Contains(respBody, []byte("Device with this deviceID doesn't exist")) {
				return fmt.Errorf("DEVICE_FORGOTTEN")
			}
		}

		return fmt.Errorf("API Error %d: %s", resp.StatusCode, string(respBody))
	}

	if target != nil {
		return json.NewDecoder(resp.Body).Decode(target)
	}
	return nil
}
