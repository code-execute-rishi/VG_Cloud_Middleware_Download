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
	"net/url"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/gorilla/mux"
)

func GetCollaborators(w http.ResponseWriter, r *http.Request) {
	log.Println("========================================")
	log.Println("🔵 GetCollaborators: NEW REQUEST STARTED")
	log.Println("🔵 GetCollaborators: Method:", r.Method)
	log.Println("🔵 GetCollaborators: URL:", r.URL.String())
	log.Println("========================================")

	if r.Method != http.MethodGet {
		log.Println("❌ GetCollaborators: Invalid method:", r.Method)
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	vars := mux.Vars(r)
	deviceID := vars["deviceId"]
	log.Println("🔵 GetCollaborators: Device ID:", deviceID)

	if deviceID == "" {
		log.Println("❌ GetCollaborators: Device ID is empty")
		utils.RespondWithError(w, http.StatusBadRequest, "Device ID required")
		return
	}

	claims := r.Context().Value(middleware.ClerkUserIDKey).(*clerk.SessionClaims)
	clerkUserID := claims.Subject
	log.Println("🔵 GetCollaborators: Clerk User ID:", clerkUserID)

	log.Println("🔵 GetCollaborators: Fetching user from Clerk...")
	usr, err := user.Get(r.Context(), claims.Subject)
	if err != nil {
		log.Println("❌ GetCollaborators: Failed to fetch user from Clerk:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch user: "+err.Error())
		return
	}
	log.Println("✅ GetCollaborators: User fetched from Clerk")

	var email string
	if usr.PrimaryEmailAddressID != nil && len(usr.EmailAddresses) > 0 {
		for _, emailAddr := range usr.EmailAddresses {
			if emailAddr.ID == *usr.PrimaryEmailAddressID {
				email = emailAddr.EmailAddress
				break
			}
		}
	}
	log.Println("🔵 GetCollaborators: User email:", email)

	log.Println("🔵 GetCollaborators: Getting or creating user in database...")
	userID, err := queries.GetOrCreateUser(db.DB, clerkUserID, email)
	if err != nil {
		log.Println("❌ GetCollaborators: Failed to get user:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get user: "+err.Error())
		return
	}
	log.Println("✅ GetCollaborators: User ID:", userID)

	log.Println("🔵 GetCollaborators: Checking if device exists...")
	exists, err := queries.DeviceExistsByID(db.DB, deviceID)
	if err != nil {
		log.Println("❌ GetCollaborators: Error checking device:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check device: "+err.Error())
		return
	}
	if !exists {
		log.Println("❌ GetCollaborators: Device doesn't exist:", deviceID)
		utils.RespondWithError(w, http.StatusNotFound, "Device doesn't exist")
		return
	}
	log.Println("✅ GetCollaborators: Device exists")

	log.Println("🔵 GetCollaborators: Verifying device ownership...")
	isOwner, err := queries.IsDeviceOwner(db.DB, deviceID, userID)
	if err != nil {
		log.Println("❌ GetCollaborators: Error verifying ownership:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to verify ownership: "+err.Error())
		return
	}
	if !isOwner {
		log.Println("❌ GetCollaborators: User is not the owner. User:", userID, "Device:", deviceID)
		utils.RespondWithError(w, http.StatusForbidden, "Only device owner can view collaborators")
		return
	}
	log.Println("✅ GetCollaborators: Ownership verified")

	log.Println("🔵 GetCollaborators: Fetching collaborators from database...")
	collaborators, err := queries.GetCollaborators(db.DB, deviceID)
	if err != nil {
		log.Println("❌ GetCollaborators: Error fetching collaborators:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch collaborators: "+err.Error())
		return
	}
	log.Println("✅ GetCollaborators: Found", len(collaborators), "collaborators")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collaborators)
	log.Println("✅ GetCollaborators: Response sent successfully")
}

func AddCollaborator(w http.ResponseWriter, r *http.Request) {
	log.Println("========================================")
	log.Println("🔵 AddCollaborator: NEW REQUEST STARTED")
	log.Println("🔵 AddCollaborator: Method:", r.Method)
	log.Println("🔵 AddCollaborator: URL:", r.URL.String())
	log.Println("========================================")

	if r.Method != http.MethodPost {
		log.Println("❌ AddCollaborator: Invalid method:", r.Method)
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	vars := mux.Vars(r)
	deviceID := vars["deviceId"]
	log.Println("🔵 AddCollaborator: Device ID:", deviceID)

	if deviceID == "" {
		log.Println("❌ AddCollaborator: Device ID is empty")
		utils.RespondWithError(w, http.StatusBadRequest, "Device ID required")
		return
	}

	claims := r.Context().Value(middleware.ClerkUserIDKey).(*clerk.SessionClaims)
	clerkUserID := claims.Subject
	log.Println("🔵 AddCollaborator: Clerk User ID:", clerkUserID)

	log.Println("🔵 AddCollaborator: Fetching user from Clerk...")
	usr, err := user.Get(r.Context(), claims.Subject)
	if err != nil {
		log.Println("❌ AddCollaborator: Failed to fetch user from Clerk:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch user: "+err.Error())
		return
	}
	log.Println("✅ AddCollaborator: User fetched from Clerk")

	var ownerEmail string
	if usr.PrimaryEmailAddressID != nil && len(usr.EmailAddresses) > 0 {
		for _, emailAddr := range usr.EmailAddresses {
			if emailAddr.ID == *usr.PrimaryEmailAddressID {
				ownerEmail = emailAddr.EmailAddress
				break
			}
		}
	}
	log.Println("🔵 AddCollaborator: Owner email:", ownerEmail)

	log.Println("🔵 AddCollaborator: Getting or creating owner user in database...")
	ownerID, err := queries.GetOrCreateUser(db.DB, clerkUserID, ownerEmail)
	if err != nil {
		log.Println("❌ AddCollaborator: Failed to get owner user:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get owner user: "+err.Error())
		return
	}
	log.Println("✅ AddCollaborator: Owner User ID:", ownerID)

	log.Println("🔵 AddCollaborator: Checking if device exists...")
	exists, err := queries.DeviceExistsByID(db.DB, deviceID)
	if err != nil {
		log.Println("❌ AddCollaborator: Error checking device:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check device: "+err.Error())
		return
	}
	if !exists {
		log.Println("❌ AddCollaborator: Device doesn't exist:", deviceID)
		utils.RespondWithError(w, http.StatusNotFound, "Device doesn't exist")
		return
	}
	log.Println("✅ AddCollaborator: Device exists")

	log.Println("🔵 AddCollaborator: Checking if device is owned...")
	OwnerNotNull, err := queries.IsDeviceOwned(db.DB, deviceID)
	if err != nil {
		log.Println("❌ AddCollaborator: Error checking device ownership:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check device: "+err.Error())
		return
	}
	if !OwnerNotNull {
		log.Println("❌ AddCollaborator: Device not owned:", deviceID)
		utils.RespondWithError(w, http.StatusNotFound, "Device not owned!! Cant add collaborators")
		return
	}
	log.Println("✅ AddCollaborator: Device is owned")

	log.Println("🔵 AddCollaborator: Verifying device ownership...")
	isOwner, err := queries.IsDeviceOwner(db.DB, deviceID, ownerID)
	if err != nil {
		log.Println("❌ AddCollaborator: Error verifying ownership:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to verify ownership: "+err.Error())
		return
	}
	if !isOwner {
		log.Println("❌ AddCollaborator: User is not the owner. User:", ownerID, "Device:", deviceID)
		utils.RespondWithError(w, http.StatusForbidden, "Only device owner can add collaborators")
		return
	}
	log.Println("✅ AddCollaborator: Ownership verified")

	log.Println("🔵 AddCollaborator: Parsing request body...")
	var req models.AddCollaboratorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("❌ AddCollaborator: Invalid JSON:", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	log.Println("🔵 AddCollaborator: Collaborator email:", req.Email)

	if req.Email == "" {
		log.Println("❌ AddCollaborator: Email is empty")
		utils.RespondWithError(w, http.StatusBadRequest, "Email is required")
		return
	}

	log.Println("🔵 AddCollaborator: Getting or creating collaborator user by email...")
	_, err = queries.GetOrCreateUserByEmail(db.DB, req.Email)
	if err != nil {
		log.Println("❌ AddCollaborator: Failed to get or create collaborator user:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get or create collaborator user: "+err.Error())
		return
	}
	log.Println("✅ AddCollaborator: Collaborator user created/found")

	log.Println("🔵 AddCollaborator: Adding collaborator to device...")
	collaboratorID, err := queries.AddCollaborator(db.DB, deviceID, req.Email)
	if err != nil {
		log.Println("❌ AddCollaborator: Failed to add collaborator:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to add collaborator: "+err.Error())
		return
	}
	log.Println("✅ AddCollaborator: Collaborator added successfully. Collaborator ID:", collaboratorID)

	response := map[string]string{
		"message":          "Collaborator added successfully",
		"email":            req.Email,
		"collaborator_id":  collaboratorID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Println("✅ AddCollaborator: Response sent successfully")
}

func RemoveCollaborator(w http.ResponseWriter, r *http.Request) {
	log.Println("========================================")
	log.Println("🔵 RemoveCollaborator: NEW REQUEST STARTED")
	log.Println("🔵 RemoveCollaborator: Method:", r.Method)
	log.Println("🔵 RemoveCollaborator: URL:", r.URL.String())
	log.Println("========================================")

	if r.Method != http.MethodDelete {
		log.Println("❌ RemoveCollaborator: Invalid method:", r.Method)
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	vars := mux.Vars(r)
	deviceID := vars["deviceId"]
	collaboratorEmail := vars["email"]

	// URL decode the email
	decodedEmail, err := url.QueryUnescape(collaboratorEmail)
	if err != nil {
		log.Println("❌ RemoveCollaborator: Failed to decode email:", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid email format")
		return
	}

	log.Println("🔵 RemoveCollaborator: Device ID:", deviceID)
	log.Println("🔵 RemoveCollaborator: Collaborator email (raw):", collaboratorEmail)
	log.Println("🔵 RemoveCollaborator: Collaborator email (decoded):", decodedEmail)

	if deviceID == "" || decodedEmail == "" {
		log.Println("❌ RemoveCollaborator: Device ID or email is empty")
		utils.RespondWithError(w, http.StatusBadRequest, "Device ID and email required")
		return
	}

	claims := r.Context().Value(middleware.ClerkUserIDKey).(*clerk.SessionClaims)
	clerkUserID := claims.Subject
	log.Println("🔵 RemoveCollaborator: Clerk User ID:", clerkUserID)

	log.Println("🔵 RemoveCollaborator: Fetching user from Clerk...")
	usr, err := user.Get(r.Context(), claims.Subject)
	if err != nil {
		log.Println("❌ RemoveCollaborator: Failed to fetch user from Clerk:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch user: "+err.Error())
		return
	}
	log.Println("✅ RemoveCollaborator: User fetched from Clerk")

	var email string
	if usr.PrimaryEmailAddressID != nil && len(usr.EmailAddresses) > 0 {
		for _, emailAddr := range usr.EmailAddresses {
			if emailAddr.ID == *usr.PrimaryEmailAddressID {
				email = emailAddr.EmailAddress
				break
			}
		}
	}
	log.Println("🔵 RemoveCollaborator: User email:", email)

	log.Println("🔵 RemoveCollaborator: Getting or creating user in database...")
	userID, err := queries.GetOrCreateUser(db.DB, clerkUserID, email)
	if err != nil {
		log.Println("❌ RemoveCollaborator: Failed to get user:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get user: "+err.Error())
		return
	}
	log.Println("✅ RemoveCollaborator: User ID:", userID)

	log.Println("🔵 RemoveCollaborator: Checking if device exists...")
	exists, err := queries.DeviceExistsByID(db.DB, deviceID)
	if err != nil {
		log.Println("❌ RemoveCollaborator: Error checking device:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check device: "+err.Error())
		return
	}
	if !exists {
		log.Println("❌ RemoveCollaborator: Device doesn't exist:", deviceID)
		utils.RespondWithError(w, http.StatusNotFound, "Device doesn't exist")
		return
	}
	log.Println("✅ RemoveCollaborator: Device exists")

	log.Println("🔵 RemoveCollaborator: Verifying device ownership...")
	isOwner, err := queries.IsDeviceOwner(db.DB, deviceID, userID)
	if err != nil {
		log.Println("❌ RemoveCollaborator: Error verifying ownership:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to verify ownership: "+err.Error())
		return
	}
	if !isOwner {
		log.Println("❌ RemoveCollaborator: User is not the owner. User:", userID, "Device:", deviceID)
		utils.RespondWithError(w, http.StatusForbidden, "Only device owner can remove collaborators")
		return
	}
	log.Println("✅ RemoveCollaborator: Ownership verified")

	log.Println("🔵 RemoveCollaborator: Removing collaborator from device...")
	err = queries.RemoveCollaborator(db.DB, deviceID, decodedEmail)
	if err != nil {
		log.Println("❌ RemoveCollaborator: Failed to remove collaborator:", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Failed to remove collaborator: "+err.Error())
		return
	}
	log.Println("✅ RemoveCollaborator: Collaborator removed successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
	log.Println("✅ RemoveCollaborator: Response sent successfully")
}