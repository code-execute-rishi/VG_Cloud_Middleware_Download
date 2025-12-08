package handlers

import (
	"backend/db"
	"backend/middleware"
	"backend/models"
	"backend/queries"
	"backend/utils"
	"encoding/json"
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
    if r.Method != http.MethodPost {
        utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    claims := r.Context().Value(middleware.ClerkUserIDKey).(*clerk.SessionClaims)
    clerkUserID := claims.Subject

    usr, err := user.Get(r.Context(), claims.Subject)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch user: "+err.Error())
        return
    }

    var email string
    if usr.PrimaryEmailAddressID != nil && len(usr.EmailAddresses) > 0 {
        for _, emailAddr := range usr.EmailAddresses {
            if emailAddr.ID == *usr.PrimaryEmailAddressID {
                email = emailAddr.EmailAddress
                break
            }
        }
    }

    userID, err := queries.GetOrCreateUser(db.DB, clerkUserID, email)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get user: "+err.Error())
        return
    }

    var req models.LiveKitTokenRequest
    err = json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
        return
    }

    if req.DeviceID == "" {
        utils.RespondWithError(w, http.StatusBadRequest, "Device ID required")
        return
    }

    exists, err := queries.DeviceExistsByID(db.DB, req.DeviceID)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check device: "+err.Error())
        return
    }
    if !exists {
        utils.RespondWithError(w, http.StatusNotFound, "Device doesn't exist")
        return
    }

    hasAccess, err := queries.UserHasDeviceAccess(db.DB, req.DeviceID, userID)
    if err != nil || !hasAccess {
        utils.RespondWithError(w, http.StatusForbidden, "No access to this device")
        return
    }

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
        CanSubscribe: boolPtr(true),
        CanPublish:   boolPtr(false),
    }
    at.SetVideoGrant(grant).
        SetIdentity(userID).
        SetValidFor(24 * time.Hour)

    token, err := at.ToJWT()
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate token: "+err.Error())
        return
    }

    resp := models.LiveKitTokenResponse{
        Token:    token,
        URL:      os.Getenv("LIVEKIT_URL"),
        RoomName: roomName,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

func RefreshDeviceToken(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    var req models.RefreshTokenRequest
    err := json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
        return
    }

    if req.DeviceID == "" {
        utils.RespondWithError(w, http.StatusBadRequest, "Device ID required")
        return
    }

    exists, err := queries.DeviceExistsByID(db.DB, req.DeviceID)
    if err != nil || !exists {
        utils.RespondWithError(w, http.StatusNotFound, "Device not found")
        return
    }

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
//
    resp := models.RefreshTokenResponse{
        Token:    livekitToken,
        URL:      os.Getenv("LIVEKIT_URL"),
        RoomName: roomName,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}