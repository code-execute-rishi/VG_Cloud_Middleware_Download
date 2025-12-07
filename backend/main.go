package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
	"vyom-backend/middleware"

	"github.com/joho/godotenv"
	"github.com/livekit/protocol/auth"
	"github.com/rs/cors"
)

// --- Configuration ---
const (
	Port = ":8080"
)

var (
	LiveKitAPIKey    string
	LiveKitAPISecret string
	LiveKitURL       string
)

// --- Data Store ---
type Device struct {
	PairingCode int64
	PublicKey   ed25519.PublicKey
	ClaimedAt   time.Time
}

var (
	deviceStore  = make(map[int64]Device) // Key: PairingCode (which acts as DeviceID for now)
	pendingStore = make(map[int64]Device) // Key: PairingCode (Pending devices)
	storeMutex   sync.RWMutex
	challenges   = make(map[string]string) // Key: DeviceID (string), Value: Challenge
	chalMutex    sync.Mutex
)

// --- API Structures ---

type ClaimRequest struct {
	PairingCode int64  `json:"pairing_code"`
	PublicKey   string `json:"public_key"` // Hex encoded
}

type AnnounceRequest struct {
	PairingCode int64  `json:"pairing_code"`
	PublicKey   string `json:"public_key"` // Hex encoded
}

type ChallengeRequest struct {
	DeviceID string `json:"device_id"`
}

type ChallengeResponse struct {
	Challenge string `json:"challenge"`
}

type VerifyRequest struct {
	DeviceID  string `json:"device_id"`
	Signature string `json:"signature"` // Hex encoded
}

type VerifyResponse struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

type TokenRequest struct {
	RoomName    string `json:"room_name"`
	Identity    string `json:"identity"`
	IsPublisher bool   `json:"is_publisher"`
}

type TokenResponse struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

// --- Handlers ---

func handleAnnounce(w http.ResponseWriter, r *http.Request) {
	var req AnnounceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	pubKeyBytes, err := hex.DecodeString(req.PublicKey)
	if err != nil || len(pubKeyBytes) != ed25519.PublicKeySize {
		http.Error(w, "Invalid public key", http.StatusBadRequest)
		return
	}

	storeMutex.Lock()
	pendingStore[req.PairingCode] = Device{
		PairingCode: req.PairingCode,
		PublicKey:   ed25519.PublicKey(pubKeyBytes),
		ClaimedAt:   time.Now(),
	}
	storeMutex.Unlock()

	log.Printf("Device announced: %d", req.PairingCode)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "announced"})
}

func handleClaim(w http.ResponseWriter, r *http.Request) {
	var req ClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	pubKeyBytes, err := hex.DecodeString(req.PublicKey)
	if err != nil || len(pubKeyBytes) != ed25519.PublicKeySize {
		http.Error(w, "Invalid public key", http.StatusBadRequest)
		return
	}

	storeMutex.Lock()
	deviceStore[req.PairingCode] = Device{
		PairingCode: req.PairingCode,
		PublicKey:   ed25519.PublicKey(pubKeyBytes),
		ClaimedAt:   time.Now(),
	}
	storeMutex.Unlock()

	log.Printf("Device claimed: %d", req.PairingCode)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "claimed"})
}

func handleChallenge(w http.ResponseWriter, r *http.Request) {
	var req ChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Helper: Convert string ID to int for lookup if numeric
	var deviceIDInt int64
	var exists bool

	// Parse DeviceID as int (assuming pairing code is used as ID)
	_, err := fmt.Sscanf(req.DeviceID, "%d", &deviceIDInt)
	if err != nil {
		// If fails to parse, maybe it's not a numeric ID.
		// In this specific flow, ID SHOULD be numeric.
		http.Error(w, "Invalid Device ID format", http.StatusBadRequest)
		return
	}

	storeMutex.RLock()
	_, exists = deviceStore[deviceIDInt]
	storeMutex.RUnlock()

	if !exists {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	// Generate random challenge
	b := make([]byte, 32)
	rand.Read(b)
	challenge := hex.EncodeToString(b)

	chalMutex.Lock()
	challenges[req.DeviceID] = challenge
	chalMutex.Unlock()

	json.NewEncoder(w).Encode(ChallengeResponse{Challenge: challenge})
}

func handleVerify(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var deviceIDInt int64
	fmt.Sscanf(req.DeviceID, "%d", &deviceIDInt)

	storeMutex.RLock()
	device, exists := deviceStore[deviceIDInt]
	storeMutex.RUnlock()

	if !exists {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	chalMutex.Lock()
	challenge, ok := challenges[req.DeviceID]
	delete(challenges, req.DeviceID) // One-time use
	chalMutex.Unlock()

	if !ok {
		http.Error(w, "Challenge expired or invalid", http.StatusBadRequest)
		return
	}

	sigBytes, err := hex.DecodeString(req.Signature)
	if err != nil {
		http.Error(w, "Invalid signature format", http.StatusBadRequest)
		return
	}

	if !ed25519.Verify(device.PublicKey, []byte(challenge), sigBytes) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Signature valid, generate LiveKit token
	at := auth.NewAccessToken(LiveKitAPIKey, LiveKitAPISecret)
	// Helper for bool pointer
	boolPtr := func(b bool) *bool { return &b }

	grant := &auth.VideoGrant{
		RoomJoin:   true,
		Room:       "sim-room-01", // Hardcoded for now or dynamic
		CanPublish: boolPtr(true),
	}
	at.AddGrant(grant).SetIdentity(req.DeviceID).SetValidFor(24 * time.Hour)

	token, err := at.ToJWT()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(VerifyResponse{
		Token: token,
		URL:   LiveKitURL,
	})
}

func handleLiveKitTokens(w http.ResponseWriter, r *http.Request) {
	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Helper for bool pointer
	boolPtr := func(b bool) *bool { return &b }

	at := auth.NewAccessToken(LiveKitAPIKey, LiveKitAPISecret)
	grant := &auth.VideoGrant{
		RoomJoin:     true,
		Room:         req.RoomName,
		CanPublish:   boolPtr(req.IsPublisher),
		CanSubscribe: boolPtr(!req.IsPublisher),
	}
	at.AddGrant(grant).SetIdentity(req.Identity).SetValidFor(24 * time.Hour)

	token, err := at.ToJWT()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(TokenResponse{
		Token: token,
		URL:   LiveKitURL,
	})
}

// handleClaimDevice processes the user's request to claim a pending device.
// Input: { "pair_code": 12345 }
// Logic: Checks pendingStore. If found, moves to deviceStore. Returns LiveKit token.
func handleClaimDevice(w http.ResponseWriter, r *http.Request) {
	// 1. Parse Input
	var req struct {
		PairCode int64 `json:"pair_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	storeMutex.Lock()
	defer storeMutex.Unlock()

	// 2. Check Pending Store
	device, exists := pendingStore[req.PairCode]
	if !exists {
		// Check if it's already claimed just in case
		if _, claimed := deviceStore[req.PairCode]; claimed {
			http.Error(w, "Device already claimed", http.StatusConflict)
		} else {
			http.Error(w, "Device not found or not announcing", http.StatusNotFound)
		}
		return
	}

	// 3. Move to Device Store
	deviceStore[req.PairCode] = device
	delete(pendingStore, req.PairCode)
	log.Printf("Device %d claimed by user!", req.PairCode)

	// 4. Generate LiveKit Token for the *User* (Viewer)
	at := auth.NewAccessToken(LiveKitAPIKey, LiveKitAPISecret)

	// Helper for bool pointer
	boolPtr := func(b bool) *bool { return &b }

	// Grant permissions to join the room associated with this device.
	// NOTE: For this simple implementation, we assume all devices go to "sim-room-01"
	// or we could make the room name dynamic based on the pair code (e.g. "room-<pair_code>")
	// matching what the device middleware will eventually join.
	// The prompt implies the device joins a room. The *device middleware* currently hardcodes "sim-room-01" in `handleVerify` (wait, no, `handleVerify` in backend sets it).
	// Let's check `handleVerify` in file view...
	// Line 221: Room: "sim-room-01"
	// So both device and user should join "sim-room-01".

	roomName := "sim-room-01"
	identity := fmt.Sprintf("user-%d-%s", req.PairCode, time.Now().Format("150405"))

	grant := &auth.VideoGrant{
		RoomJoin:     true,
		Room:         roomName,
		CanPublish:   boolPtr(false), // User is viewer/controller, usually doesn't publish video? Or maybe yes for voice?
		CanSubscribe: boolPtr(true),
	}
	at.AddGrant(grant).SetIdentity(identity).SetValidFor(24 * time.Hour)

	token, err := at.ToJWT()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// 5. Response
	resp := struct {
		Token string `json:"token"`
		URL   string `json:"url"`
	}{
		Token: token,
		URL:   LiveKitURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// --- Main ---

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	LiveKitAPIKey = os.Getenv("LIVEKIT_API_KEY")
	LiveKitAPISecret = os.Getenv("LIVEKIT_API_SECRET")
	LiveKitURL = os.Getenv("LIVEKIT_URL")

	if LiveKitAPIKey == "" || LiveKitAPISecret == "" || LiveKitURL == "" {
		log.Fatal("LIVEKIT_API_KEY, LIVEKIT_API_SECRET, and LIVEKIT_URL must be set")
	}

	// ZeroTier Auto-Join
	ztNetworkID := os.Getenv("ZEROTIER_NETWORK_ID")
	if ztNetworkID != "" {
		if err := middleware.JoinZeroTier(ztNetworkID); err != nil {
			log.Printf("ZeroTier Join failed: %v", err)
		}
	}

	mux := http.NewServeMux()

	// Device Routes
	mux.HandleFunc("/api/v1/devices/announce", handleAnnounce)
	mux.HandleFunc("/api/v1/devices/claim", handleClaim)
	mux.HandleFunc("/api/v1/devices/auth/challenge", handleChallenge)
	mux.HandleFunc("/api/v1/devices/auth/verify", handleVerify)

	// New Claim Endpoint (Task 1)
	mux.HandleFunc("/api/claim-device", handleClaimDevice)

	// Frontend Routes
	mux.HandleFunc("/api/v1/livekit/tokens", handleLiveKitTokens)

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"}, // Allow all origins for ZeroTier access
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	handler := c.Handler(mux)

	// Listen on 0.0.0.0 to accept external traffic (ZeroTier)
	addr := "0.0.0.0" + Port
	log.Printf("Backend Server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
