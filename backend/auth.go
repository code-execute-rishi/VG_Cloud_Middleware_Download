package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/livekit/protocol/auth"
)

type ClaimDeviceRequest struct {
	PairCode int64 `json:"pair_code"`
}

type ClaimDeviceResponse struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

func HandleClaimDevice(w http.ResponseWriter, r *http.Request) {
	// CORS Headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Helper for JSON Error
	sendError := func(w http.ResponseWriter, msg string, code int) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]string{"message": msg})
	}

	var req ClaimDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate Pair Code (Check Pending Store)
	storeMutex.RLock()
	pendingDevice, exists := pendingStore[req.PairCode]
	storeMutex.RUnlock()

	if !exists {
		// Fallback for Debug (optional)
		if req.PairCode == 9000 {
			// Allow Admin Override
		} else {
			sendError(w, "Device not found. Ensure middleware is running.", http.StatusUnauthorized)
			return
		}
	} else {
		// Promote to Claimed
		storeMutex.Lock()
		deviceStore[req.PairCode] = pendingDevice
		delete(pendingStore, req.PairCode)
		storeMutex.Unlock()
		log.Printf("Device %d promoted from Pending to Claimed via UI", req.PairCode)
	}

	// Generate LiveKit Token
	at := auth.NewAccessToken(LiveKitAPIKey, LiveKitAPISecret)

	// Helper for bool pointer
	boolPtr := func(b bool) *bool { return &b }

	grant := &auth.VideoGrant{
		RoomJoin:       true,
		Room:           "sim-room-01",
		CanPublish:     boolPtr(true),
		CanSubscribe:   boolPtr(true),
		CanPublishData: boolPtr(true),
	}

	// Identity can be dynamic
	at.AddGrant(grant).SetIdentity("gcs-commander").SetValidFor(24 * time.Hour)

	token, err := at.ToJWT()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Return Response
	json.NewEncoder(w).Encode(ClaimDeviceResponse{
		Token: token,
		URL:   LiveKitURL,
	})
}
