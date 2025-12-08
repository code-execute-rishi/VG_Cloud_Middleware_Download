package handlers

import (
	"backend/db"
	"backend/middleware"
	"backend/models"
	"backend/queries"
	"backend/utils"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gorilla/mux"
	"github.com/livekit/protocol/auth"
)


func RegisterDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not Allowed")
		return
	}

	var req models.RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.PublicKey == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Public Key[String] is required to register the device & cant be null")
		return
	}

	if req.PairingCode < 10000000 || req.PairingCode > 99999999 {
		utils.RespondWithError(w, http.StatusBadRequest, "Pairing Code[Int] is required & must be exactly 8 digits")
		return
	}

	if req.NodeID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "NodeID[String] is required to register the device & cant be null")
		return
	}

	exists, err := queries.DeviceExistsByPublicKey(db.DB, req.PublicKey)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Cant check if the Public Key exists or not: "+err.Error())
		return
	}
	if exists {
		utils.RespondWithError(w, http.StatusConflict, "A device with this Public Key already exists, Public Key must be unique")
		return
	}

	exists, err = queries.PairingCodeExists(db.DB, req.PairingCode)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Cant check the pairing code: "+err.Error())
		return
	}
	if exists {
		utils.RespondWithError(w, http.StatusConflict, "A device with this Pairing Code already exists, Pairing Code must be unique")
		return
	}

	deviceID, err := queries.CreateDevice(db.DB, req.PublicKey, req.PairingCode, req.NodeID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to register device: "+err.Error())
		return
	}

	resp := models.RegisterResponse{
		Message:  "Device successfully registered!!",
		DeviceID: deviceID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func GetChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.ChallengeRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.DeviceID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Device ID[String] is required & cannot be null")
		return
	}

	exists, err := queries.DeviceExistsByID(db.DB, req.DeviceID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Cant check device by ID: "+err.Error())
		return
	}
	if !exists {
		utils.RespondWithError(w, http.StatusBadRequest, "No devices found with this device ID")
		return
	}

	challenge, err := queries.GenerateChallenge(db.DB, req.DeviceID)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Challenge cant be generated: "+err.Error())
		return
	}

	resp := models.ChallengeResponse{
		Message:   "Challenge successfully generated",
		Challenge: challenge,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func VerifyDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.VerifyRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.DeviceID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Device ID[String] is required & cannot be null")
		return
	}

	if req.Signature == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Signature[String] is required & cannot be null")
		return
	}

	challenge, err := queries.GetChallenge(db.DB, req.DeviceID)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Challenge not found or expired: "+err.Error())
		return
	}

	publicKeyB64, err := queries.GetPublicKey(db.DB, req.DeviceID)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Device not found: "+err.Error())
		return
	}

	signatureBytes, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid Signature Encoding")
		return
	}

	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Invalid Public Key Encoding")
		return
	}

	isValid := ed25519.Verify(publicKeyBytes, []byte(challenge), signatureBytes)
	if !isValid {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid Signature")
		return
	}

	_ = queries.DeleteChallenge(db.DB, req.DeviceID)

	roomName, err := queries.GetOrCreateLiveKitRoom(db.DB, req.DeviceID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get room: "+err.Error())
		return
	}

	apiKey := os.Getenv("LIVEKIT_API_KEY")
	apiSecret := os.Getenv("LIVEKIT_API_SECRET")

	at := auth.NewAccessToken(apiKey, apiSecret)
	grant := &auth.VideoGrant{
		RoomJoin:     true,
		Room:         roomName,
		CanPublish:   boolPtr(true),
		CanSubscribe: boolPtr(false),
	}
	at.SetVideoGrant(grant).
		SetIdentity(req.DeviceID).
		SetValidFor(168 * time.Hour)

	livekitToken, err := at.ToJWT()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate token: "+err.Error())
		return
	}

	ztConfig, err := queries.GetZeroTierConfig(db.DB, req.DeviceID)
	var zerotierResp models.ZerotierConfig
	if err == nil && ztConfig != nil {
		zerotierResp = *ztConfig
	}

	resp := models.VerifyResponse{
		LivekitToken: livekitToken,
		LivekitURL:   os.Getenv("LIVEKIT_URL"),
		RoomName:     roomName,
		Zerotier:     zerotierResp,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func CheckClaim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.CheckClaimRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.DeviceID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Device ID[String] is required & cannot be null")
		return
	}

	claim, err := queries.CheckClaim(db.DB, req.DeviceID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error checking claim status: " + err.Error())
		return
	}

	message := "Claim pending"
	if claim {
		message = "Claimed successfully"
	}

	resp := models.CheckClaimResponse{
		Claim:   claim,
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func UpdateDeviceStatus(w http.ResponseWriter, r *http.Request) {
    log.Println("🔵 UpdateDeviceStatus: Request received")
    
    if r.Method != http.MethodPatch {
        log.Println("❌ UpdateDeviceStatus: Invalid method:", r.Method)
        utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    vars := mux.Vars(r)
    deviceID := vars["deviceId"]
    log.Println("🔵 UpdateDeviceStatus: Device ID:", deviceID)
    
    if deviceID == "" {
        log.Println("❌ UpdateDeviceStatus: Device ID is empty")
        utils.RespondWithError(w, http.StatusBadRequest, "Device ID required")
        return
    }

    claims := r.Context().Value(middleware.ClerkUserIDKey).(*clerk.SessionClaims)
    clerkUserID := claims.Subject
    log.Println("🔵 UpdateDeviceStatus: Clerk User ID:", clerkUserID)

    userID, err := queries.GetOrCreateUser(db.DB, clerkUserID, "")
    if err != nil {
        log.Println("❌ UpdateDeviceStatus: Failed to get user:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get user: "+err.Error())
        return
    }
    log.Println("✅ UpdateDeviceStatus: User ID:", userID)

    log.Println("🔵 UpdateDeviceStatus: Checking device existence...")
    exists, err := queries.DeviceExistsByID(db.DB, deviceID)
    if err != nil {
        log.Println("❌ UpdateDeviceStatus: Error checking device:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check device: "+err.Error())
        return
    }
    if !exists {
        log.Println("❌ UpdateDeviceStatus: Device doesn't exist:", deviceID)
        utils.RespondWithError(w, http.StatusNotFound, "Device doesn't exist")
        return
    }
    log.Println("✅ UpdateDeviceStatus: Device exists")

    log.Println("🔵 UpdateDeviceStatus: Checking device access...")
    hasAccess, err := queries.UserHasDeviceAccess(db.DB, deviceID, userID)
    if err != nil || !hasAccess {
        log.Println("❌ UpdateDeviceStatus: No access. User:", userID, "Device:", deviceID)
        utils.RespondWithError(w, http.StatusForbidden, "No access to this device")
        return
    }
    log.Println("✅ UpdateDeviceStatus: Access verified")

    var req models.UpdateDeviceStatusRequest
    err = json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        log.Println("❌ UpdateDeviceStatus: Invalid JSON:", err)
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
        return
    }
    log.Println("🔵 UpdateDeviceStatus: New status:", req.Status)

    validStatuses := map[string]bool{
        "Airborne":    true,
        "StandBy":     true,
        "Maintenance": true,
    }

    if !validStatuses[req.Status] {
        log.Println("❌ UpdateDeviceStatus: Invalid status:", req.Status)
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid status. Must be Airborne, StandBy, or Maintenance")
        return
    }

    log.Println("🔵 UpdateDeviceStatus: Updating device status in database...")
    err = queries.UpdateDeviceStatus(db.DB, deviceID, req.Status)
    if err != nil {
        log.Println("❌ UpdateDeviceStatus: Failed to update status:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update status: "+err.Error())
        return
    }
    log.Println("✅ UpdateDeviceStatus: Status updated successfully")

    response := map[string]interface{}{
        "success": true,
        "message": "Device status updated successfully",
        "status":  req.Status,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
    log.Println("✅ UpdateDeviceStatus: Response sent")
}