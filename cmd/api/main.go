package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"device-middleware/internal/auth"
	"device-middleware/internal/backend"
)

// --- Configuration Constants ---
const (
	ConfigFileName = "demo_config.json"
	SetupPort      = ":8085"
)

var (
	BackendBaseURL  = backend.DefaultBaseURL
	FrontendBaseURL = "https://internetlinkpro.vyomgarud.com"

	apiClient *backend.BackendClient

	deviceStatus      GlobalDeviceStatus
	deviceStatusMutex sync.RWMutex
)

func init() {
	if envURL := os.Getenv("BACKEND_URL"); envURL != "" {
		BackendBaseURL = envURL
	} else {
		BackendBaseURL = "https://backend.internetlinkpro.vyomgarud.com"
	}
	log.Printf("🔗 [API] Backend URL Set to: %s", BackendBaseURL)
}

type ConfigFile struct {
	SSID       string `json:"ssid"`
	Password   string `json:"password"`
	Resolution string `json:"resolution"`
	FCPort     string `json:"fc_port"`
	FCBaud     int    `json:"fc_baud"`
}

type ConnectionState string

const (
	StateConnected    ConnectionState = "Connected"
	StateDisconnected ConnectionState = "Disconnected"
	StateError        ConnectionState = "Error"

	// Status File Paths
	StatusDir           = "/tmp/vyom-status"
	ZeroTierStatusFile  = StatusDir + "/zerotier.json"
	LiveKitStatusFile   = StatusDir + "/livekit.json"
	HardwareStatusFile  = StatusDir + "/hardware.json"
	TelemetryStatusFile = StatusDir + "/telemetry.json"
)

type AuthStatus struct {
	ConnectURL string `json:"connect_url"`
}

// --- Structs (Matching Dashboard Expectations) ---

type GlobalDeviceStatus struct {
	IsConfigured bool             `json:"is_configured"`
	IsConnected  bool             `json:"is_connected"`
	IsClaimed    bool             `json:"is_claimed"`
	Camera       CameraConfig     `json:"camera_config"`
	Hardware     HardwareStatus   `json:"hardware_status"`
	LiveKit      LiveKitStatus    `json:"livekit_status"`
	ZeroTier     ZeroTierStatus   `json:"zerotier_status"`
	Telemetry    *TelemetryStatus `json:"telemetry,omitempty"`
	Auth         AuthStatus       `json:"auth_status"`
	User         *auth.UserClaims `json:"user_info,omitempty"`
}

type CameraConfig struct {
	Resolution string `json:"resolution"`
}

type HardwareStatus struct {
	FCConnected  bool `json:"fc_connected"`
	CamConnected bool `json:"cam_connected"`
}

type TelemetryStatus struct {
	Battery *SysStatus      `json:"battery"`
	GPS     *GpsRaw         `json:"gps"`
	HUD     *VfrHud         `json:"hud"`
	System  *TelemetryState `json:"system"`
}
type TelemetryState struct {
	Armed bool   `json:"armed"`
	Mode  string `json:"mode"`
}
type SysStatus struct {
	Voltage          float32 `json:"voltage"`
	BatteryRemaining int     `json:"battery_remaining"`
}
type GpsRaw struct {
	FixType    uint8 `json:"fix_type"`
	Satellites uint8 `json:"satellites_visible"`
}
type VfrHud struct {
	Heading int16   `json:"heading"`
	Alt     float32 `json:"alt"`
}

type LiveKitStatus struct {
	State        string `json:"state"`
	RoomName     string `json:"room_name"`
	Participants int    `json:"participants"`
	LastError    string `json:"last_error"`
}

type ZeroTierStatus struct {
	State     string `json:"state"`
	NetworkID string `json:"network_id"`
	IPAddress string `json:"ip_address"`
	LastError string `json:"last_error"`
}

func CleanupPort(port int) {
	log.Printf("[Init] Checking if port %d is in use...", port)
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		ln.Close()
		return
	}
	log.Printf("[Init] Port %d is in use. Attempting to free it...", port)
	cmd := exec.Command("fuser", "-k", fmt.Sprintf("%d/tcp", port))
	if err := cmd.Run(); err == nil {
		log.Printf("[Init] Killed process on port %d using fuser.", port)
		time.Sleep(1 * time.Second)
		return
	}
	// ... (simplified cleanup for brevity) ...
	exec.Command("pkill", "-f", "vyom-api").Run()
	time.Sleep(1 * time.Second)
}

func main() {
	CleanupPort(8085)
	resetFlag := flag.Bool("reset", false, "Reset configuration and identity")
	flag.Parse()
	if *resetFlag {
		log.Println("WARNING: Resetting Device Configuration...")
		os.Remove(ConfigFileName)
		os.Remove(backend.IdentityFile)
		log.Println("Reset complete. Starting fresh...")
	}

	log.Println("🚀 Starting Vyom API Service...")
	logFile, err := os.OpenFile("api.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}

	apiClient = backend.NewBackendClient(BackendBaseURL)
	if err := apiClient.LoadOrCreateIdentity(); err != nil {
		log.Printf("⚠️ Failed to load identity (Initial): %v", err)
	}

	go startWebServer()
	runStateLoop()
}

func startWebServer() {
	ip := getOutboundIP()
	log.Printf("\n>>> WEB UI AVAILABLE: http://%s%s <<<\n", ip, SetupPort)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/system-info", handleSystemInfo)
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/update-config", handleUpdateConfig)
	mux.HandleFunc("/claim", handleClaim)
	mux.HandleFunc("/api/serial-ports", handleSerialPorts)
	mux.HandleFunc("/api/wifi-scan", handleWifiScan)
	mux.HandleFunc("/api/save-config", handleSaveConfig)
	mux.HandleFunc("/api/logs", handleLogs)
	mux.HandleFunc("/api/stream", handleLocalStream)
	mux.HandleFunc("/api/cameras", handleCameras)

	uiDir := "./ui/dist"
	if _, err := os.Stat(uiDir); os.IsNotExist(err) {
		if _, err := os.Stat("/opt/vyom/ui"); err == nil {
			uiDir = "/opt/vyom/ui"
		}
	}
	log.Printf("Serving UI from: %s", uiDir)
	fs := http.FileServer(http.Dir(uiDir))
	mux.Handle("/", fs)
	if err := http.ListenAndServe(SetupPort, corsMiddleware(mux)); err != nil {
		log.Fatalf("Web Server failed: %v", err)
	}
}

func runStateLoop() {
	// Start Status Aggregator
	go aggregateStatus()

	for {
		_, err := os.ReadFile(ConfigFileName)

		isConfigured := (err == nil)

		deviceStatusMutex.Lock()
		deviceStatus.IsConfigured = isConfigured
		deviceStatusMutex.Unlock()

		if !isConfigured {
			// Not Configured Logic
			ip := getOutboundIP()
			callbackURL := fmt.Sprintf("http://%s%s/claim", ip, SetupPort)
			connectURL := fmt.Sprintf("%s/connect?callback=%s", FrontendBaseURL, callbackURL)

			deviceStatusMutex.Lock()
			deviceStatus.Auth.ConnectURL = connectURL
			deviceStatusMutex.Unlock()

			time.Sleep(5 * time.Second)
			continue
		}

		// Identity Logic
		if apiClient.Identity == nil {
			if err := apiClient.LoadOrCreateIdentity(); err != nil {
				log.Printf("Failed to load identity: %v. Enabling Claim Mode.", err)

				// Fallback to Claim Mode
				ip := getOutboundIP()
				callbackURL := fmt.Sprintf("http://%s%s/claim", ip, SetupPort)
				connectURL := fmt.Sprintf("%s/connect?callback=%s", FrontendBaseURL, callbackURL)

				deviceStatusMutex.Lock()
				deviceStatus.Auth.ConnectURL = connectURL
				deviceStatusMutex.Unlock()
			}
		}

		if apiClient.Identity != nil {
			deviceStatusMutex.Lock()
			deviceStatus.IsClaimed = true
			deviceStatusMutex.Unlock()

			// Check JWT
			var tokenToParse string
			if apiClient.Identity.AuthToken != "" {
				tokenToParse = apiClient.Identity.AuthToken
			} else {
				tokenToParse = apiClient.Identity.Token
			}
			if claims, err := auth.ParseJWT(tokenToParse); err == nil {
				deviceStatusMutex.Lock()
				deviceStatus.User = claims
				deviceStatusMutex.Unlock()
			}
		}

		time.Sleep(5 * time.Second)
	}
}

func aggregateStatus() {
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		// Read ZeroTier
		if data, err := os.ReadFile(ZeroTierStatusFile); err == nil {
			var zt ZeroTierStatus
			if json.Unmarshal(data, &zt) == nil {
				deviceStatusMutex.Lock()
				deviceStatus.ZeroTier = zt
				deviceStatusMutex.Unlock()
			}
		}

		// Read LiveKit
		if data, err := os.ReadFile(LiveKitStatusFile); err == nil {
			var lk LiveKitStatus
			if json.Unmarshal(data, &lk) == nil {
				deviceStatusMutex.Lock()
				deviceStatus.LiveKit = lk
				if lk.State == "Connected" {
					deviceStatus.IsConnected = true
				} else {
					deviceStatus.IsConnected = false
				}
				deviceStatusMutex.Unlock()
			}
		}

		// Read Hardware
		if data, err := os.ReadFile(HardwareStatusFile); err == nil {
			var hw HardwareStatus
			if json.Unmarshal(data, &hw) == nil {
				deviceStatusMutex.Lock()
				deviceStatus.Hardware = hw
				deviceStatusMutex.Unlock()
			}
		}

		// Read Telemetry
		if data, err := os.ReadFile(TelemetryStatusFile); err == nil {
			var telem TelemetryStatus
			if json.Unmarshal(data, &telem) == nil {
				deviceStatusMutex.Lock()
				deviceStatus.Telemetry = &telem
				deviceStatusMutex.Unlock()
			}
		}
	}
}

// --- Handlers (Simplified) ---

func handleClaim(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	deviceID := r.URL.Query().Get("device_id")
	authToken := r.URL.Query().Get("auth_token")

	if token == "" || deviceID == "" {
		http.Error(w, "Missing token or device_id", http.StatusBadRequest)
		return
	}

	apiClient.Identity = &backend.Identity{
		DeviceID:  deviceID,
		Token:     token,
		AuthToken: authToken,
	}
	apiClient.TypifySaveIdentity()

	config := ConfigFile{
		Resolution: "640x480",
		FCPort:     "auto",
		FCBaud:     57600,
	}
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(ConfigFileName, data, 0644)

	log.Println("✅ [Claim] Token Received! Restarting API...")
	w.Write([]byte("Device Claimed! Restarting..."))

	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0) // Systemd will restart
	}()
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	deviceStatusMutex.RLock()
	statusCopy := deviceStatus
	deviceStatusMutex.RUnlock()

	// In a full implementation, we would fetch status from other services here
	// For now, we return what we know (Identity/Config status)

	if !statusCopy.IsConfigured {
		host := r.Host
		callbackURL := fmt.Sprintf("http://%s/claim", host)
		statusCopy.Auth.ConnectURL = fmt.Sprintf("%s/connect?callback=%s", FrontendBaseURL, callbackURL)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statusCopy)
}

func handleLocalStream(w http.ResponseWriter, r *http.Request) {
	// Proxy to vyom-camera's MJPEG stream on port 8081
	var conn net.Conn
	var err error
	for i := 0; i < 3; i++ {
		conn, err = net.Dial("tcp", "127.0.0.1:8081")
		if err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		http.Error(w, "Camera Stream Unavailable", http.StatusServiceUnavailable)
		return
	}
	defer conn.Close()
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=vyomboundary")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, conn)
}

func handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"device_id": "unknown",
		"version":   "0.2.0-microservices",
	}
	if apiClient != nil && apiClient.Identity != nil {
		resp["device_id"] = apiClient.Identity.DeviceID
	}
	json.NewEncoder(w).Encode(resp)
}

func handleUpdateConfig(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }
func handleSerialPorts(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode([]string{"/dev/ttyACM0"})
}
func handleWifiScan(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode([]string{"Vyom_Secure"})
}
func handleSaveConfig(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }
func handleLogs(w http.ResponseWriter, r *http.Request) {
	content, _ := os.ReadFile("api.log") // Only API logs for now
	json.NewEncoder(w).Encode(strings.Split(string(content), "\n"))
}

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleCameras(w http.ResponseWriter, r *http.Request) {
	// Mock response to satisfy UI
	cameras := []map[string]string{
		{
			"id":         "cam0",
			"name":       "Main Camera",
			"resolution": "640x480",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cameras)
}
