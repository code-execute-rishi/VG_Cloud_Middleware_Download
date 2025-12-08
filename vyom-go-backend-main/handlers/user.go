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

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/gorilla/mux"
)

func GetClerkIDandEmail(w http.ResponseWriter, r *http.Request) {
    claims := r.Context().Value(middleware.ClerkUserIDKey).(*clerk.SessionClaims)

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

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "clerk_user_id": claims.Subject,
        "email": email,
    })
}

func ClaimDevice(w http.ResponseWriter, r *http.Request) {
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

    var req models.ClaimRequest
    err = json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
        return
    }

    if req.PairingCode == 0 {
        utils.RespondWithError(w, http.StatusBadRequest, "Pairing code[Int] is required")
        return
    }

    if req.Name == "" {
        utils.RespondWithError(w, http.StatusBadRequest, "Name[String] is required")
        return
    }

    userID, err := queries.GetOrCreateUser(db.DB, clerkUserID, email)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get/create user: "+err.Error())
        return
    }

    deviceID, err := queries.ClaimDevice(db.DB, req.PairingCode, userID, req.Name)
    if err != nil {
        utils.RespondWithError(w, http.StatusBadRequest, "Failed to claim device: " + err.Error())
        return
    }

    resp := models.ClaimResponse{
        Message:  "Device Claimed Successfully!!",
        DeviceID: deviceID,
        Name:     req.Name,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

func GetDevices(w http.ResponseWriter, r *http.Request) {
	log.Println("========================================")
	log.Println("🔵 GetDevices: NEW REQUEST STARTED")
	log.Println("🔵 GetDevices: Method:", r.Method)
	log.Println("🔵 GetDevices: URL:", r.URL.String())
	log.Println("========================================")

	if r.Method != http.MethodGet {
		log.Println("❌ GetDevices: Invalid method:", r.Method)
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	claims := r.Context().Value(middleware.ClerkUserIDKey).(*clerk.SessionClaims)
	clerkUserID := claims.Subject
	log.Println("🔵 GetDevices: Clerk User ID:", clerkUserID)

	log.Println("🔵 GetDevices: Fetching user from Clerk...")
	usr, err := user.Get(r.Context(), claims.Subject)
	if err != nil {
		log.Println("❌ GetDevices: Failed to fetch user from Clerk:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch user: "+err.Error())
		return
	}
	log.Println("✅ GetDevices: User fetched from Clerk")

	var email string
	if usr.PrimaryEmailAddressID != nil && len(usr.EmailAddresses) > 0 {
		for _, emailAddr := range usr.EmailAddresses {
			if emailAddr.ID == *usr.PrimaryEmailAddressID {
				email = emailAddr.EmailAddress
				break
			}
		}
	}
	log.Println("🔵 GetDevices: User email:", email)

	log.Println("🔵 GetDevices: Getting or creating user in database...")
	userID, err := queries.GetOrCreateUser(db.DB, clerkUserID, email)
	if err != nil {
		log.Println("❌ GetDevices: Failed to get/create user:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get/create user: "+err.Error())
		return
	}
	log.Println("✅ GetDevices: User ID:", userID)

	log.Println("🔵 GetDevices: Fetching user devices from database...")
	devices, err := queries.GetUserDevices(db.DB, userID)
	if err != nil {
		log.Println("❌ GetDevices: Failed to fetch devices:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch devices: "+err.Error())
		return
	}
	log.Println("✅ GetDevices: Found", len(devices), "devices")

	log.Println("🔵 GetDevices: Fetching collaborators for each device...")
	for i := range devices {
		log.Println("🔵 GetDevices: Fetching collaborators for device:", devices[i].ID)
		collaborators, err := queries.GetCollaborators(db.DB, devices[i].ID)
		if err != nil {
			log.Println("❌ GetDevices: Failed to fetch collaborators for device", devices[i].ID, ":", err)
			// Continue with empty collaborators rather than failing entire request
			devices[i].Collaborators = []models.Collaborator{}
			continue
		}
		log.Println("✅ GetDevices: Found", len(collaborators), "collaborators for device:", devices[i].ID)
		devices[i].Collaborators = collaborators
	}
	log.Println("✅ GetDevices: All collaborators fetched")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
	log.Println("✅ GetDevices: Response sent successfully")
}

func GetDevice(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    vars := mux.Vars(r)
    deviceID := vars["deviceId"]
    if deviceID == "" {
        utils.RespondWithError(w, http.StatusBadRequest, "Device ID required")
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

    exists, err := queries.DeviceExistsByID(db.DB, deviceID)
    if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get the device: "+err.Error())
        return
    }
    if !exists {
        utils.RespondWithError(w, http.StatusBadRequest, "Device with this deviceID doesn't exist")
        return
    }

    device, err := queries.GetDeviceByID(db.DB, deviceID, userID)
    if err != nil {
        utils.RespondWithError(w, http.StatusNotFound, "Device not found: "+err.Error())
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(device)
}


func DeleteDevice(w http.ResponseWriter, r *http.Request) {
	log.Println("========================================")
	log.Println("🔵 DeleteDevice: NEW REQUEST STARTED")
	log.Println("🔵 DeleteDevice: Method:", r.Method)
	log.Println("🔵 DeleteDevice: URL:", r.URL.String())
	log.Println("========================================")

	if r.Method != http.MethodDelete {
		log.Println("❌ DeleteDevice: Invalid method:", r.Method)
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	vars := mux.Vars(r)
	deviceID := vars["deviceId"]
	log.Println("🔵 DeleteDevice: Device ID:", deviceID)

	if deviceID == "" {
		log.Println("❌ DeleteDevice: Device ID is empty")
		utils.RespondWithError(w, http.StatusBadRequest, "Device ID required")
		return
	}

	claims := r.Context().Value(middleware.ClerkUserIDKey).(*clerk.SessionClaims)
	clerkUserID := claims.Subject
	log.Println("🔵 DeleteDevice: Clerk User ID:", clerkUserID)

	log.Println("🔵 DeleteDevice: Fetching user from Clerk...")
	usr, err := user.Get(r.Context(), claims.Subject)
	if err != nil {
		log.Println("❌ DeleteDevice: Failed to fetch user from Clerk:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch user: "+err.Error())
		return
	}
	log.Println("✅ DeleteDevice: User fetched from Clerk")

	var email string
	if usr.PrimaryEmailAddressID != nil && len(usr.EmailAddresses) > 0 {
		for _, emailAddr := range usr.EmailAddresses {
			if emailAddr.ID == *usr.PrimaryEmailAddressID {
				email = emailAddr.EmailAddress
				break
			}
		}
	}
	log.Println("🔵 DeleteDevice: User email:", email)

	log.Println("🔵 DeleteDevice: Getting or creating user in database...")
	userID, err := queries.GetOrCreateUser(db.DB, clerkUserID, email)
	if err != nil {
		log.Println("❌ DeleteDevice: Failed to get user:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get user: "+err.Error())
		return
	}
	log.Println("✅ DeleteDevice: User ID:", userID)

	log.Println("🔵 DeleteDevice: Checking if device exists...")
	exists, err := queries.DeviceExistsByID(db.DB, deviceID)
	if err != nil {
		log.Println("❌ DeleteDevice: Error checking device:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get the device: "+err.Error())
		return
	}
	if !exists {
		log.Println("❌ DeleteDevice: Device doesn't exist:", deviceID)
		utils.RespondWithError(w, http.StatusBadRequest, "Device with this deviceID doesn't exist")
		return
	}
	log.Println("✅ DeleteDevice: Device exists")

	log.Println("🔵 DeleteDevice: Deleting device from database...")
	err = queries.DeleteDevice(db.DB, deviceID, userID)
	if err != nil {
		log.Println("❌ DeleteDevice: Failed to delete device:", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	log.Println("✅ DeleteDevice: Device deleted successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
	log.Println("✅ DeleteDevice: Response sent successfully")
}