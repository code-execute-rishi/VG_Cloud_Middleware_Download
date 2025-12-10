package handlers

import (
	"backend/db"
	"backend/models"
	"backend/queries"
	"backend/utils"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func f64(v *float64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func i(v *int) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func UpdateTelemetry(w http.ResponseWriter, r *http.Request) {
	log.Println("🔵 UpdateTelemetry: Request received")

	if r.Method != http.MethodPost && r.Method != http.MethodPatch {
		log.Println("❌ UpdateTelemetry: Invalid method:", r.Method)
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	vars := mux.Vars(r)
	deviceID := vars["deviceId"]
	log.Println("🔵 UpdateTelemetry: Device ID:", deviceID)

	if deviceID == "" {
		log.Println("❌ UpdateTelemetry: Device ID is empty")
		utils.RespondWithError(w, http.StatusBadRequest, "Device ID required")
		return
	}

	log.Println("🔵 UpdateTelemetry: Checking device existence...")
	exists, err := queries.DeviceExistsByID(db.DB, deviceID)
	if err != nil {
		log.Println("❌ UpdateTelemetry: Error checking device:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to check device: "+err.Error())
		return
	}
	if !exists {
		log.Println("❌ UpdateTelemetry: Device doesn't exist:", deviceID)
		utils.RespondWithError(w, http.StatusNotFound, "Device doesn't exist")
		return
	}
	log.Println("✅ UpdateTelemetry: Device exists")

	var req models.UpdateTelemetryRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Println("❌ UpdateTelemetry: Invalid JSON:", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	log.Printf("📊 INCOMING TELEMETRY DATA:\n"+
		"  Device ID: %s\n"+
		"  Latitude: %v\n"+
		"  Longitude: %v\n"+
		"  Altitude: %v\n"+
		"  Speed: %v\n"+
		"  Heading: %v\n"+
		"  Signal Strength: %v\n"+
		"  Battery: %v\n",
		deviceID,
		f64(req.Latitude),
		f64(req.Longitude),
		f64(req.Altitude),
		f64(req.Speed),
		f64(req.Heading),
		i(req.SignalStrength),
		i(req.Battery),
	)

	update := models.TelemetryUpdate{
		DeviceID:       deviceID,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		Altitude:       req.Altitude,
		Speed:          req.Speed,
		Heading:        req.Heading,
		SignalStrength: req.SignalStrength,
		Battery:        req.Battery,
	}

	log.Println("🔵 UpdateTelemetry: Updating telemetry in database...")
	err = queries.UpdateDeviceTelemetry(db.DB, update)
	if err != nil {
		log.Println("❌ UpdateTelemetry: Failed to update telemetry:", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update telemetry: "+err.Error())
		return
	}
	log.Println("✅ UpdateTelemetry: Telemetry updated successfully")

	response := map[string]interface{}{
		"success": true,
		"message": "Telemetry updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Println("✅ UpdateTelemetry: Response sent")
}