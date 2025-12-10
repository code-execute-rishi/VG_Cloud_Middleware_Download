package handlers

import (
	"backend/db"
	"backend/middleware"
	"backend/models"
	"backend/queries"
	"backend/utils"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/livekit/protocol/auth"
)

func boolPtr(b bool) *bool {
    return &b
}

func GetLiveKitToken(w http.ResponseWriter, r *http.Request) {
    log.Println("🔵 GetLiveKitToken: Request received")
    
    if r.Method != http.MethodPost {
        log.Println("❌ GetLiveKitToken: Invalid method:", r.Method)
        utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    claims := r.Context().Value(middleware.ClerkUserIDKey).(*clerk.SessionClaims)
    clerkUserID := claims.Subject
    log.Println("🔵 GetLiveKitToken: Clerk User ID:", clerkUserID)

    usr, err := user.Get(r.Context(), claims.Subject)
    if err != nil {
        log.Println("❌ GetLiveKitToken: Failed to fetch user:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch user: "+err.Error())
        return
    }
    log.Println("✅ GetLiveKitToken: User fetched successfully")

    var email string
    if usr.PrimaryEmailAddressID != nil && len(usr.EmailAddresses) > 0 {
        for _, emailAddr := range usr.EmailAddresses {
            if emailAddr.ID == *usr.PrimaryEmailAddressID {
                email = emailAddr.EmailAddress
                break
            }
        }
    }
    log.Println("🔵 GetLiveKitToken: User email:", email)

    userID, err := queries.GetOrCreateUser(db.DB, clerkUserID, email)
    if err != nil {
        log.Println("❌ GetLiveKitToken: Failed to get user:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get user: "+err.Error())
        return
    }
    log.Println("🔵 GetLiveKitToken: Internal User ID:", userID)

    var req models.LiveKitTokenRequest
    err = json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        log.Println("❌ GetLiveKitToken: Invalid JSON:", err)
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
        return
    }
    log.Printf("📊 GetLiveKitToken: Request Body - DeviceID: %s", req.DeviceID)

    if req.DeviceID == "" {
        log.Println("❌ GetLiveKitToken: Device ID is empty")
        utils.RespondWithError(w, http.StatusBadRequest, "Device ID required")
        return
    }

    log.Println("🔵 GetLiveKitToken: Checking device existence...")
    exists, err := queries.DeviceExistsByID(db.DB, req.DeviceID)
    if err != nil {
        log.Println("❌ GetLiveKitToken: Error checking device:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check device: "+err.Error())
        return
    }
    if !exists {
        log.Println("❌ GetLiveKitToken: Device doesn't exist:", req.DeviceID)
        utils.RespondWithError(w, http.StatusNotFound, "Device doesn't exist")
        return
    }
    log.Println("✅ GetLiveKitToken: Device exists")

    log.Println("🔵 GetLiveKitToken: Checking user access...")
    hasAccess, err := queries.UserHasDeviceAccess(db.DB, req.DeviceID, userID)
    if err != nil || !hasAccess {
        log.Printf("❌ GetLiveKitToken: No access - UserID: %s, DeviceID: %s, Error: %v", userID, req.DeviceID, err)
        utils.RespondWithError(w, http.StatusForbidden, "No access to this device")
        return
    }
    log.Println("✅ GetLiveKitToken: User has access")

    log.Println("🔵 GetLiveKitToken: Getting/Creating LiveKit room...")
    roomName, err := queries.GetOrCreateLiveKitRoom(db.DB, req.DeviceID)
    if err != nil {
        log.Println("❌ GetLiveKitToken: Failed to get room:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get room: "+err.Error())
        return
    }
    log.Println("✅ GetLiveKitToken: Room name:", roomName)

    apiKey := os.Getenv("LIVEKIT_API_KEY")
    apiSecret := os.Getenv("LIVEKIT_API_SECRET")
    log.Printf("🔵 GetLiveKitToken: API Key present: %v, API Secret present: %v", apiKey != "", apiSecret != "")

    at := auth.NewAccessToken(apiKey, apiSecret)
    grant := &auth.VideoGrant{
        RoomJoin:     true,
        Room:         roomName,
        CanSubscribe: boolPtr(true),
        CanPublish:   boolPtr(false),
        CanPublishData: boolPtr(true),
    }
    at.SetVideoGrant(grant).
        SetIdentity(userID).
        SetValidFor(24 * time.Hour)

    log.Printf("📊 GetLiveKitToken: Token Config - Room: %s, Identity: %s, CanSubscribe: true, CanPublish: false, ValidFor: 24h", roomName, userID)

    token, err := at.ToJWT()
    if err != nil {
        log.Println("❌ GetLiveKitToken: Failed to generate token:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate token: "+err.Error())
        return
    }
    log.Println("✅ GetLiveKitToken: Token generated successfully")

    resp := models.LiveKitTokenResponse{
        Token:    token,
        URL:      os.Getenv("LIVEKIT_URL"),
        RoomName: roomName,
    }

    log.Printf("📊 GetLiveKitToken: Response - Token: %s..., URL: %s, RoomName: %s", token[:20], resp.URL, resp.RoomName)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
    log.Println("✅ GetLiveKitToken: Response sent successfully")
}

func RefreshDeviceToken(w http.ResponseWriter, r *http.Request) {
    log.Println("🔵 RefreshDeviceToken: Request received")
    
    if r.Method != http.MethodPost {
        log.Println("❌ RefreshDeviceToken: Invalid method:", r.Method)
        utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    var req models.RefreshTokenRequest
    err := json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        log.Println("❌ RefreshDeviceToken: Invalid JSON:", err)
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
        return
    }
    log.Printf("📊 RefreshDeviceToken: Request Body - DeviceID: %s", req.DeviceID)

    if req.DeviceID == "" {
        log.Println("❌ RefreshDeviceToken: Device ID is empty")
        utils.RespondWithError(w, http.StatusBadRequest, "Device ID required")
        return
    }

    log.Println("🔵 RefreshDeviceToken: Checking device existence...")
    exists, err := queries.DeviceExistsByID(db.DB, req.DeviceID)
    if err != nil || !exists {
        log.Printf("❌ RefreshDeviceToken: Device not found - DeviceID: %s, Error: %v", req.DeviceID, err)
        utils.RespondWithError(w, http.StatusNotFound, "Device not found")
        return
    }
    log.Println("✅ RefreshDeviceToken: Device exists")

    log.Println("🔵 RefreshDeviceToken: Getting/Creating LiveKit room...")
    roomName, err := queries.GetOrCreateLiveKitRoom(db.DB, req.DeviceID)
    if err != nil {
        log.Println("❌ RefreshDeviceToken: Failed to get room:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get room: "+err.Error())
        return
    }
    log.Println("✅ RefreshDeviceToken: Room name:", roomName)

    apiKey := os.Getenv("LIVEKIT_API_KEY")
    apiSecret := os.Getenv("LIVEKIT_API_SECRET")
    log.Printf("🔵 RefreshDeviceToken: API Key present: %v, API Secret present: %v", apiKey != "", apiSecret != "")

    at := auth.NewAccessToken(apiKey, apiSecret)
    grant := &auth.VideoGrant{
        RoomJoin:     true,
        Room:         roomName,
        CanPublish:   boolPtr(true),
        CanSubscribe: boolPtr(false),
        CanPublishData: boolPtr(true),
    }
    at.SetVideoGrant(grant).
        SetIdentity(req.DeviceID).
        SetValidFor(168 * time.Hour)

    log.Printf("📊 RefreshDeviceToken: Token Config - Room: %s, Identity: %s, CanPublish: true, CanSubscribe: false, ValidFor: 168h", roomName, req.DeviceID)

    livekitToken, err := at.ToJWT()
    if err != nil {
        log.Println("❌ RefreshDeviceToken: Failed to generate token:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate token: "+err.Error())
        return
    }
    log.Println("✅ RefreshDeviceToken: Token generated successfully")

    resp := models.RefreshTokenResponse{
        Token:    livekitToken,
        URL:      os.Getenv("LIVEKIT_URL"),
        RoomName: roomName,
    }

    log.Printf("📊 RefreshDeviceToken: Response - Token: %s..., URL: %s, RoomName: %s", livekitToken[:20], resp.URL, resp.RoomName)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
    log.Println("✅ RefreshDeviceToken: Response sent successfully")
}