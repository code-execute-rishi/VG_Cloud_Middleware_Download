package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	IdentityFile   = "identity.json"
	DefaultBaseURL = "http://4.247.135.200"
)

// --- Data Models ---

type Identity struct {
	DeviceID  string `json:"device_id"`
	Token     string `json:"token"`
	AuthToken string `json:"auth_token"`
}

type CheckClaimRequest struct {
	DeviceID string `json:"device_id"`
}

type CheckClaimResponse struct {
	Claim   bool   `json:"claim_status"`
	Message string `json:"message"`
}

type TelemetryUpdate struct {
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	Altitude       float32 `json:"altitude"`
	Speed          float32 `json:"speed"`
	Heading        float32 `json:"heading"`
	SignalStrength int     `json:"signal_strength"`
	Battery        int     `json:"battery"`
	FlightMode     string  `json:"flight_mode"`
}

type LKTokenResponse struct {
	Token      string `json:"token"`
	LiveKitURL string `json:"livekit_url"` // Check this name, might be "url" in backend
}

type VerifyResponse struct {
	LiveKitToken string `json:"livekit_token"`
	RoomName     string `json:"room_name"`
}

type ZerotierConfigResponse struct {
	ZerotierIP string `json:"zerotier_ip"`
	NetworkID  string `json:"network_id"`
	SSHCommand string `json:"ssh_command"`
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
	return nil
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
	os.Remove(IdentityFile)
	c.Identity = nil
	return nil
}

// --- API Methods ---

func (c *BackendClient) Authenticate() (*VerifyResponse, error) {
	if c.Identity == nil || c.Identity.Token == "" {
		return nil, fmt.Errorf("no token found")
	}

	lkToken, err := c.GetLiveKitToken(c.Identity.DeviceID)
	if err != nil {
		return nil, err
	}

	return &VerifyResponse{
		LiveKitToken: lkToken.Token,
		RoomName:     c.Identity.DeviceID,
	}, nil
}

func (c *BackendClient) GetLiveKitToken(deviceID string) (*LKTokenResponse, error) {
	req := map[string]string{"device_id": deviceID}
	var res LKTokenResponse
	if err := c.post("/api/v1/livekit/tokens", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *BackendClient) CheckClaim() (bool, bool) {
	// Returns (IsClaimed, IsServerClaimed)
	if c.Identity == nil || c.Identity.Token == "" {
		return false, false
	}

	// Remote check
	req := CheckClaimRequest{DeviceID: c.Identity.DeviceID}
	var res CheckClaimResponse
	if err := c.post("/api/v1/devices/auth/check-claim", req, &res); err != nil {
		return true, false // Assume claimed locally if error? Or fail?
	}
	return true, res.Claim
}

func (c *BackendClient) UpdateTelemetry(lat, lon, alt, speed, head float64, sig, batt int) error {
	if c.Identity == nil {
		return nil
	}

	data := TelemetryUpdate{
		Latitude: lat, Longitude: lon, Altitude: float32(alt),
		Speed: float32(speed), Heading: float32(head),
		SignalStrength: sig, Battery: batt,
		FlightMode: "Unknown",
	}

	url := fmt.Sprintf("/api/v1/devices/%s/telemetry", c.Identity.DeviceID)
	return c.post(url, data, nil)
}

func (c *BackendClient) CheckLiveness() error {
	// Check if device is still claimed on the server
	req := CheckClaimRequest{DeviceID: c.Identity.DeviceID}
	var res CheckClaimResponse
	err := c.post("/api/v1/devices/auth/check-claim", req, &res)

	if err != nil {
		if strings.Contains(err.Error(), "DEVICE_FORGOTTEN") {
			return fmt.Errorf("DEVICE_FORGOTTEN")
		}
	}
	return nil
}

func (c *BackendClient) GetZeroTierConfig(deviceID string) (*ZerotierConfigResponse, error) {
	url := fmt.Sprintf("/api/v1/devices/%s/zerotier", deviceID)
	var res ZerotierConfigResponse
	if err := c.get(url, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// --- Helper ---

func (c *BackendClient) post(path string, payload interface{}, target interface{}) error {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(b)
	}

	url := c.BaseURL + path
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	return c.doRequest(req, target)
}

func (c *BackendClient) get(path string, target interface{}) error {
	url := c.BaseURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	return c.doRequest(req, target)
}

func (c *BackendClient) doRequest(req *http.Request, target interface{}) error {
	req.Header.Set("Content-Type", "application/json")
	if c.Identity != nil {
		if c.Identity.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.Identity.AuthToken)
		} else if c.Identity.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Identity.Token)
		}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		respBodyStr := string(respBody)

		// Only treat as DEVICE_FORGOTTEN if it's a genuine "device doesn't exist" error
		// Don't confuse configuration errors (like missing LiveKit creds) with forgotten devices
		if resp.StatusCode == 401 {
			if strings.Contains(respBodyStr, "Unauthorized") && strings.Contains(respBodyStr, "Device") {
				return fmt.Errorf("DEVICE_FORGOTTEN")
			}
		}
		if resp.StatusCode == 404 {
			// Only if the error explicitly mentions device not existing
			if strings.Contains(respBodyStr, "Device doesn't exist") || strings.Contains(respBodyStr, "device not found") {
				return fmt.Errorf("DEVICE_FORGOTTEN")
			}
			// Don't treat "LiveKit credentials not configured" as DEVICE_FORGOTTEN
		}
		return fmt.Errorf("API Error %d: %s", resp.StatusCode, respBodyStr)
	}

	if target != nil {
		return json.NewDecoder(resp.Body).Decode(target)
	}
	return nil
}
