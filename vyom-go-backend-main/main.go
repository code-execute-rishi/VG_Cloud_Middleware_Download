package main

import (
	"backend/db"
	"backend/handlers"
	"backend/middleware"
	"fmt"
	"log"
	"net/http"
	"os"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
    
    if os.Getenv("ENV") != "production" {
        err := godotenv.Load(".env.backend")
        if err != nil {
            log.Println("Local env file not found, using system env")
        }
    }

    err := db.Connect(
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
    )

    if err != nil {
        log.Fatal("DB failed:", err)
    }

    err = db.InitSchema()  
    if err != nil {
        log.Fatal("Schema failed:", err)
    }
    
    port := os.Getenv("PORT")
    clerk.SetKey(os.Getenv("CLERK_SECRET_KEY"))
    
    r := mux.NewRouter()
    r.HandleFunc("/api/v1/devices/register", handlers.RegisterDevice).Methods("POST") 
    r.HandleFunc("/api/v1/devices/auth/challenge", handlers.GetChallenge).Methods("POST")
    r.HandleFunc("/api/v1/devices/auth/verify", handlers.VerifyDevice).Methods("POST")
    r.HandleFunc("/api/v1/devices/auth/refresh", handlers.RefreshDeviceToken).Methods("POST")
    r.HandleFunc("/api/v1/devices/auth/check-claim", handlers.CheckClaim).Methods("POST")

    r.Handle("/api/v1/devices/claim", middleware.ClerkAuth(http.HandlerFunc(handlers.ClaimDevice))).Methods("POST")
    r.Handle("/api/v1/devices", middleware.ClerkAuth(http.HandlerFunc(handlers.GetDevices))).Methods("GET")
    r.Handle("/api/v1/devices/{deviceId}", middleware.ClerkAuth(http.HandlerFunc(handlers.GetDevice))).Methods("GET")
    r.Handle("/api/v1/devices/{deviceId}", middleware.ClerkAuth(http.HandlerFunc(handlers.UpdateDeviceStatus))).Methods("PATCH")
    r.Handle("/api/v1/devices/{deviceId}", middleware.ClerkAuth(http.HandlerFunc(handlers.DeleteDevice))).Methods("DELETE") 

    r.Handle("/api/v1/devices/{deviceId}/collaborators", middleware.ClerkAuth(http.HandlerFunc(handlers.GetCollaborators))).Methods("GET")
    r.Handle("/api/v1/devices/{deviceId}/collaborators", middleware.ClerkAuth(http.HandlerFunc(handlers.AddCollaborator))).Methods("POST")
    r.Handle("/api/v1/devices/{deviceId}/collaborators/{email}", middleware.ClerkAuth(http.HandlerFunc(handlers.RemoveCollaborator))).Methods("DELETE")
    
    r.Handle("/api/v1/livekit/tokens", middleware.ClerkAuth(http.HandlerFunc(handlers.GetLiveKitToken))).Methods("POST")

    r.HandleFunc("/api/v1/devices/{deviceId}/zerotier", handlers.SaveZeroTierConfig).Methods("POST")
    r.Handle("/api/v1/devices/{deviceId}/zerotier", middleware.ClerkAuth(http.HandlerFunc(handlers.GetZeroTierConfig))).Methods("GET")

    r.HandleFunc("/api/v1/devices/{deviceId}/telemetry", handlers.UpdateTelemetry).Methods("POST", "PATCH")

    fmt.Printf("Server starting on :%s\n", port)
    http.ListenAndServe(":"+port, r)
}