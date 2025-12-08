package handlers

import (
	"backend/db"
	"backend/models"
	"backend/queries"
	"backend/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func SaveZeroTierConfig(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    vars := mux.Vars(r)
    deviceID := vars["deviceId"]
    if deviceID == "" {
        utils.RespondWithError(w, http.StatusBadRequest, "Device ID required")
        return
    }

    userID, err := queries.GetUserIDFromDeviceID(db.DB, deviceID)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get user: "+err.Error())
        return
    }

    exists, err := queries.DeviceExistsByID(db.DB, deviceID)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check device: "+err.Error())
        return
    }
    if !exists {
        utils.RespondWithError(w, http.StatusNotFound, "Device doesn't exist")
        return
    }

    isOwner, err := queries.IsDeviceOwner(db.DB, deviceID, userID)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to verify ownership: "+err.Error())
        return
    }
    if !isOwner {
        utils.RespondWithError(w, http.StatusForbidden, "Only device owner can configure ZeroTier")
        return
    }

    config, err := queries.GetZeroTierConfig(db.DB, deviceID)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get config: "+err.Error())
        return
    }

    if config == nil {
        utils.RespondWithError(w, http.StatusBadRequest, "ZeroTier not configured for this device")
        return
    }

    err = queries.SaveZeroTierConfig(db.DB, deviceID, config.ZerotierIP)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to save config: "+err.Error())
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func GetZeroTierConfig(w http.ResponseWriter, r *http.Request) {
    log.Println("========================================")
    log.Println("🔵 GetZeroTierConfig: NEW REQUEST STARTED")
    log.Println("🔵 GetZeroTierConfig: Method:", r.Method)
    log.Println("🔵 GetZeroTierConfig: URL:", r.URL.String())
    log.Println("========================================")
    
    if r.Method != http.MethodGet {
        log.Println("❌ GetZeroTierConfig: Invalid method:", r.Method)
        utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    vars := mux.Vars(r)
    deviceID := vars["deviceId"]
    log.Println("🔵 GetZeroTierConfig: Device ID:", deviceID)
    
    if deviceID == "" {
        log.Println("❌ GetZeroTierConfig: Device ID is empty")
        utils.RespondWithError(w, http.StatusBadRequest, "Device ID required")
        return
    }

    log.Println("🔵 GetZeroTierConfig: Checking if device exists...")
    exists, err := queries.DeviceExistsByID(db.DB, deviceID)
    if err != nil {
        log.Println("❌ GetZeroTierConfig: Error checking device existence:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check device: "+err.Error())
        return
    }
    if !exists {
        log.Println("❌ GetZeroTierConfig: Device does not exist:", deviceID)
        utils.RespondWithError(w, http.StatusNotFound, "Device doesn't exist")
        return
    }
    log.Println("✅ GetZeroTierConfig: Device exists")

    log.Println("🔵 GetZeroTierConfig: Getting user ID from device...")
    userID, err := queries.GetUserIDFromDeviceID(db.DB, deviceID)
    if err != nil {
        log.Println("❌ GetZeroTierConfig: Error getting user ID:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get user: "+err.Error())
        return
    }
    log.Println("✅ GetZeroTierConfig: User ID:", userID)

    log.Println("🔵 GetZeroTierConfig: Verifying ownership...")
    isOwner, err := queries.IsDeviceOwner(db.DB, deviceID, userID)
    if err != nil {
        log.Println("❌ GetZeroTierConfig: Error verifying ownership:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to verify ownership: "+err.Error())
        return
    }
    if !isOwner {
        log.Println("❌ GetZeroTierConfig: User is not the owner. User ID:", userID, "Device ID:", deviceID)
        utils.RespondWithError(w, http.StatusForbidden, "Only device owner can view ZeroTier config")
        return
    }
    log.Println("✅ GetZeroTierConfig: Ownership verified")

    log.Println("🔵 GetZeroTierConfig: Checking for existing ZeroTier config...")
    config, err := queries.GetZeroTierConfig(db.DB, deviceID)
    if err != nil {
        log.Println("❌ GetZeroTierConfig: Error getting existing config:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get config: "+err.Error())
        return
    }

    if config != nil && config.ZerotierIP != "" {
        log.Println("✅ GetZeroTierConfig: Found existing config. IP:", config.ZerotierIP)
        resp := models.ZerotierConfigResponse{
            ZerotierIP: config.ZerotierIP,
            SSHCommand: fmt.Sprintf("ssh drone@%s", config.ZerotierIP),
        }
        log.Println("✅ GetZeroTierConfig: Returning cached config")
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
        return
    }
    log.Println("🔵 GetZeroTierConfig: No existing config found, proceeding with authorization...")

    log.Println("🔵 GetZeroTierConfig: Getting node ID from device...")
    nodeID, err := queries.GetDeviceNodeID(db.DB, deviceID)
    if err != nil {
        log.Println("❌ GetZeroTierConfig: Error getting node ID:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get node ID: "+err.Error())
        return
    }
    log.Println("✅ GetZeroTierConfig: Node ID:", nodeID)

    if nodeID == "" {
        log.Println("❌ GetZeroTierConfig: Node ID is empty for device:", deviceID)
        utils.RespondWithError(w, http.StatusBadRequest, "Device has no ZeroTier node ID")
        return
    }

    networkID := os.Getenv("ZEROTIER_NETWORK_ID")
    apiToken := os.Getenv("ZEROTIER_API_TOKEN")
    log.Println("🔵 GetZeroTierConfig: Network ID:", networkID)
    log.Println("🔵 GetZeroTierConfig: API Token loaded:", apiToken != "")

    if networkID == "" || apiToken == "" {
        log.Println("❌ GetZeroTierConfig: ZeroTier configuration missing from environment")
        utils.RespondWithError(w, http.StatusInternalServerError, "ZeroTier configuration missing")
        return
    }

    authorizeURL := fmt.Sprintf("https://api.zerotier.com/api/v1/network/%s/member/%s", networkID, nodeID)
    log.Println("🔵 GetZeroTierConfig: Authorize URL:", authorizeURL)
    
    authorizePayload := map[string]interface{}{
        "config": map[string]bool{
            "authorized": true,
        },
    }
    log.Println("🔵 GetZeroTierConfig: Authorization payload:", authorizePayload)

    authorizeData, _ := json.Marshal(authorizePayload)
    log.Println("🔵 GetZeroTierConfig: Marshalled authorization data:", string(authorizeData))
    
    authorizeReq, err := http.NewRequest("POST", authorizeURL, bytes.NewBuffer(authorizeData))
    if err != nil {
        log.Println("❌ GetZeroTierConfig: Error creating authorize request:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create authorize request: "+err.Error())
        return
    }
    log.Println("✅ GetZeroTierConfig: Authorization request created")

    authorizeReq.Header.Set("Authorization", "token "+apiToken)
    authorizeReq.Header.Set("Content-Type", "application/json")
    log.Println("🔵 GetZeroTierConfig: Request headers set")

    client := &http.Client{}
    log.Println("🔵 GetZeroTierConfig: Sending authorization request to ZeroTier API...")
    authorizeResp, err := client.Do(authorizeReq)
    if err != nil {
        log.Println("❌ GetZeroTierConfig: Error sending authorize request:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to authorize device: "+err.Error())
        return
    }
    defer authorizeResp.Body.Close()
    log.Println("✅ GetZeroTierConfig: Authorization response received. Status:", authorizeResp.StatusCode)

    if authorizeResp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(authorizeResp.Body)
        log.Println("❌ GetZeroTierConfig: Authorization failed. Status:", authorizeResp.StatusCode, "Body:", string(bodyBytes))
        utils.RespondWithError(w, http.StatusInternalServerError, "ZeroTier authorization failed")
        return
    }
    log.Println("✅ GetZeroTierConfig: Device authorized successfully")

    getMemberURL := fmt.Sprintf("https://api.zerotier.com/api/v1/network/%s/member/%s", networkID, nodeID)
    log.Println("🔵 GetZeroTierConfig: Get member URL:", getMemberURL)
    
    getMemberReq, err := http.NewRequest("GET", getMemberURL, nil)
    if err != nil {
        log.Println("❌ GetZeroTierConfig: Error creating get member request:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create member info request: "+err.Error())
        return
    }
    log.Println("✅ GetZeroTierConfig: Get member request created")

    getMemberReq.Header.Set("Authorization", "token "+apiToken)
    log.Println("🔵 GetZeroTierConfig: Fetching member info from ZeroTier API...")

    getMemberResp, err := client.Do(getMemberReq)
    if err != nil {
        log.Println("❌ GetZeroTierConfig: Error fetching member info:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get member info: "+err.Error())
        return
    }
    defer getMemberResp.Body.Close()
    log.Println("✅ GetZeroTierConfig: Member info response received. Status:", getMemberResp.StatusCode)

    var memberData models.ZerotierMemberResponse
    err = json.NewDecoder(getMemberResp.Body).Decode(&memberData)
    if err != nil {
        log.Println("❌ GetZeroTierConfig: Error parsing member data:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to parse member info: "+err.Error())
        return
    }
    log.Println("✅ GetZeroTierConfig: Member data parsed successfully")
    log.Println("🔵 GetZeroTierConfig: IP Assignments:", memberData.Config.IPAssignments)
    log.Println("🔵 GetZeroTierConfig: Authorized status:", memberData.Config.Authorized)

    if len(memberData.Config.IPAssignments) == 0 {
        log.Println("❌ GetZeroTierConfig: No IP assignments found. Device has not joined network yet.")
        utils.RespondWithError(w, http.StatusInternalServerError, "Device has not joined ZeroTier network yet")
        return
    }

    zerotierIP := memberData.Config.IPAssignments[0]
    log.Println("✅ GetZeroTierConfig: Assigned IP:", zerotierIP)

    log.Println("🔵 GetZeroTierConfig: Saving config to database...")
    err = queries.SaveZeroTierConfig(db.DB, deviceID, zerotierIP)
    if err != nil {
        log.Println("❌ GetZeroTierConfig: Error saving config to database:", err)
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to save config: "+err.Error())
        return
    }
    log.Println("✅ GetZeroTierConfig: Config saved to database successfully")

    resp := models.ZerotierConfigResponse{
        ZerotierIP: zerotierIP,
        SSHCommand: fmt.Sprintf("ssh drone@%s", zerotierIP),
    }
    log.Println("✅ GetZeroTierConfig: Returning response. SSH Command:", resp.SSHCommand)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
    log.Println("✅ GetZeroTierConfig: Response sent successfully")
}