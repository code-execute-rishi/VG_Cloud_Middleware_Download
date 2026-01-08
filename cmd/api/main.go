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
// --- Configuration Constants ---
const (
	ConfigFilePath = "/etc/vyom/config.json"
	SetupPort      = ":8085"
)

var (
	// Allow internal calls (localhost) and specific frontend domain
	AllowedOrigins = map[string]bool{
		"http://localhost:5173":                      true, // Vite Dev
		"http://localhost:8085":                      true, // Local Served
		"https://internetlinkpro.vyomgarud.com":      true, // Production Dashboard
		"https://install.internetlink.vyomgarud.com": true, // Installer
	}

	BackendBaseURL  = backend.DefaultBaseURL
	FrontendBaseURL = "https://internetlinkpro.vyomgarud.com"

	apiClient *backend.BackendClient

	deviceStatus        GlobalDeviceStatus
	deviceStatusMutex   sync.RWMutex
	lastTelemetryUpdate time.Time
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
	SSID         string `json:"ssid"`
	Password     string `json:"password"`
	Resolution   string `json:"resolution"`
	CameraDevice string `json:"camera_device"` // e.g. "USB Camera (/dev/video4)"
	CameraType   string `json:"camera_type"`   // e.g. "usb", "csi"
	FCPort       string `json:"fc_port"`
	FCBaud       int    `json:"fc_baud"`
	LiveKitURL   string `json:"livekit_url"`
	LiveKitToken string `json:"livekit_token"`
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

var configFileMutex sync.Mutex

type AuthStatus struct {
	ConnectURL string `json:"connect_url"`
}

// --- Structs (Matching Dashboard Expectations) ---

type GlobalDeviceStatus struct {
	IsConfigured  bool             `json:"is_configured"`
	IsConnected   bool             `json:"is_connected"`
	IsClaimed     bool             `json:"is_claimed"`
	Camera        CameraConfig     `json:"camera_config"`
	Hardware      HardwareStatus   `json:"hardware_status"`
	LiveKitConfig LiveKitConfig    `json:"livekit_config"`
	LiveKit       LiveKitStatus    `json:"livekit_status"`
	ZeroTier      ZeroTierStatus   `json:"zerotier_status"`
	Telemetry     *TelemetryStatus `json:"telemetry,omitempty"`
	Auth          AuthStatus       `json:"auth_status"`
	User          *auth.UserClaims `json:"user_info,omitempty"`
}

type CameraConfig struct {
	Resolution   string `json:"resolution"`
	CameraType   string `json:"camera_type,omitempty"`
	CameraDevice string `json:"camera_device,omitempty"`
	FCPort       string `json:"fc_port"`
	FCBaud       int    `json:"fc_baud"`
}

type LiveKitConfig struct {
	LiveKitURL string `json:"livekit_url"`
	Token      string `json:"token"`
}

type HardwareStatus struct {
	FCConnected  bool `json:"fc_connected"`
	CamConnected bool `json:"cam_connected"`
}

type TelemetryStatus struct {
	Battery     *SysStatus      `json:"battery"`
	GPS         *GpsRaw         `json:"gps"`
	HUD         *VfrHud         `json:"hud"`
	System      *TelemetryState `json:"system"`
	FCConnected bool            `json:"fc_connected"`
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
	State     string         `json:"state"`
	NetworkID string         `json:"network_id"`
	IPAddress string         `json:"ip_address"`
	LastError string         `json:"last_error"`
	Peers     []ZeroTierPeer `json:"peers"`
}

type ZeroTierPeer struct {
	Address string `json:"address"`
	Version string `json:"version"`
	Latency int    `json:"latency"`
	Role    string `json:"role"`
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
	log.Println(">>> VYOM MIDDLEWARE API - VERSION: DEBUG-TRACE-2 <<<")
	CleanupPort(8085)
	resetFlag := flag.Bool("reset", false, "Reset configuration and identity")
	flag.Parse()
	if *resetFlag {
		log.Println("WARNING: Resetting Device Configuration...")
		os.Remove(ConfigFilePath)
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
	mux.HandleFunc("/api/system-stats", handleSystemStats)
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/update-config", handleUpdateConfig)

	mux.HandleFunc("/api/config/gcs-endpoints", handleGCSEndpoints) // Used by Telemetry Service
	mux.HandleFunc("/api/gcs/endpoints", handleGCSEndpoints)        // Used by UI
	mux.HandleFunc("/api/config/livekit", handleLiveKitConfig)
	mux.HandleFunc("/api/config/telemetry", handleTelemetryConfig)
	// Internal Handlers
	mux.HandleFunc("/internal/zerotier", handleInternalZeroTier)
	mux.HandleFunc("/internal/telemetry", handleInternalTelemetry)

	mux.HandleFunc("/claim", handleClaim)
	mux.HandleFunc("/api/serial-ports", handleSerialPorts)
	mux.HandleFunc("/api/wifi-scan", handleWifiScan)
	mux.HandleFunc("/api/save-config", authMiddleware(handleSaveConfig))
	mux.HandleFunc("/api/logs", handleLogs) // Logs are read-only, but arguably sensitive. Open for now for debugging.
	mux.HandleFunc("/api/stream", handleLocalStream)
	mux.HandleFunc("/api/cameras", handleCameras)
	mux.HandleFunc("/api/config/zerotier", handleZeroTierConfig)
	mux.HandleFunc("/api/restart", handleRestart)

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

func handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Println("⚠️ RESTART REQUESTED: Deleting config and restarting middleware...")

	// 1. Delete Config Files
	os.Remove(ConfigFilePath)
	os.Remove(backend.IdentityFile)

	// 2. Clear Status Files (Prevent stale state)
	os.RemoveAll(StatusDir)

	// 3. Restart Service
	// Attempt systemctl restart first
	go func() {
		// Wait a second to let the response go through
		time.Sleep(1 * time.Second)

		// Restart all microservices
		services := []string{"vyom-api", "vyom-camera", "vyom-telemetry", "vyom-livekit", "vyom-zerotier"}
		args := append([]string{"restart"}, services...)
		cmd := exec.Command("systemctl", args...)

		if err := cmd.Run(); err != nil {
			log.Printf("❌ systemctl restart failed: %v. Attempting manual exit.", err)
			os.Exit(0)
		}
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Middleware Restarting..."))
}

func runStateLoop() {
	// Start Status Aggregator
	go aggregateStatus()
	go checkHardwareStatus()
	go checkHardwareStatus()

	for {

		configFileMutex.Lock()
		data, err := os.ReadFile(ConfigFilePath)
		configFileMutex.Unlock()

		isConfigured := (err == nil)

		if isConfigured {
			var cfg ConfigFile
			if json.Unmarshal(data, &cfg) == nil {
				deviceStatusMutex.Lock()
				deviceStatus.Camera.Resolution = cfg.Resolution
				deviceStatus.Camera.CameraDevice = cfg.CameraDevice
				deviceStatus.Camera.CameraType = cfg.CameraType
				deviceStatus.Camera.FCPort = cfg.FCPort
				deviceStatus.Camera.FCBaud = cfg.FCBaud

				deviceStatus.LiveKitConfig.LiveKitURL = cfg.LiveKitURL
				deviceStatus.LiveKitConfig.Token = cfg.LiveKitToken
				deviceStatusMutex.Unlock()
			}
		}

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
		// Read ZeroTier - REMOVED: vyom-zerotier pushes to /internal/zerotier directly.
		// Polling file causes race conditions.
		/*
			if data, err := os.ReadFile(ZeroTierStatusFile); err == nil {
				var zt ZeroTierStatus
				if json.Unmarshal(data, &zt) == nil {
					deviceStatusMutex.Lock()
					deviceStatus.ZeroTier = zt
					deviceStatusMutex.Unlock()
				}
			}
		*/

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
				deviceStatus.Hardware.FCConnected = telem.FCConnected
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
	os.WriteFile(ConfigFilePath, data, 0644)

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
	// REDACT SENSITIVE DATA
	if statusCopy.LiveKitConfig.Token != "" {
		// Just show existence, not value
		statusCopy.LiveKitConfig.Token = "***REDACTED***"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statusCopy)
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Check if Identity is loaded/claimed
		if apiClient.Identity == nil || apiClient.Identity.Token == "" {
			// Device not claimed yet. Allow config?
			// Policy: If NOT claimed, allow local config (Setup Mode).
			// If claimed, require Auth.
			// However, usually we want to secure it always.
			// For simplicity: If not claimed, we allow. If claimed, we check.
		} else {
			// 2. Verify Authorization Header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized (Missing Header)", http.StatusUnauthorized)
				return
			}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Unauthorized (Invalid Format)", http.StatusUnauthorized)
				return
			}
			token := parts[1]

			// Compare with stored Identity Token (or AuthToken)
			// The frontend sends the Token it received from /api/status -> user_info or auth_status
			// Actually, frontend sends 'status.auth_status.token'.
			// But wait, 'auth_status' in 'handleStatus' usually just has ConnectURL.
			// Let's check 'handleClaim'. It sets Identity.Token.

			// We validate against the Device Identity Token
			if token != apiClient.Identity.Token && token != apiClient.Identity.AuthToken {
				log.Printf("[Auth] Token Mismatch. Got: %s... Expected: %s...", token[:5], apiClient.Identity.Token[:5])
				http.Error(w, "Unauthorized (Invalid Token)", http.StatusUnauthorized)
				return
			}
		}

		next(w, r)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && AllowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			// Allow valid, explicit origins, else block (or empty).
			// For dev convenience, if no Origin header (curl), we pass.
			// But if Origin is present and NOT allowed, we don't set the header, browser blocks.
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleLocalStream(w http.ResponseWriter, r *http.Request) {
	// Proxy to vyom-camera's MJPEG stream on port 8081
	resp, err := http.Get("http://127.0.0.1:8081")
	if err != nil {
		log.Printf("ERROR Proxying Camera Stream: %v", err)
		http.Error(w, "Camera Stream Unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Copy headers from camera service
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	// Stream the body
	io.Copy(w, resp.Body)
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

func handleSystemStats(w http.ResponseWriter, r *http.Request) {
	type SystemStats struct {
		CPUUsage   float64  `json:"cpu_usage"`
		CPUTemp    float64  `json:"cpu_temp"` // NEW
		RAMUsed    uint64   `json:"ram_used"`
		RAMTotal   uint64   `json:"ram_total"`
		DiskUsed   uint64   `json:"disk_used"`
		DiskTotal  uint64   `json:"disk_total"`
		Interfaces []string `json:"interfaces"`
		NetRxSpeed float64  `json:"net_rx_speed"` // KB/s
		NetTxSpeed float64  `json:"net_tx_speed"` // KB/s
	}

	var stats SystemStats

	// 1. Get Memory Info (Linux)
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		var total, available uint64
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			val := parseMemValue(fields[1]) // KB
			if strings.HasPrefix(fields[0], "MemTotal") {
				total = val * 1024
			} else if strings.HasPrefix(fields[0], "MemAvailable") {
				available = val * 1024
			}
		}
		stats.RAMTotal = total
		stats.RAMUsed = total - available
	}

	// 2. CPU Load & Temp
	stats.CPUUsage = getCPULoad()
	stats.CPUTemp = getCPUTemp()

	// 3. Network Interfaces
	if ifaces, err := net.Interfaces(); err == nil {
		for _, i := range ifaces {
			// Filter loopback and down
			if i.Flags&net.FlagUp != 0 && i.Flags&net.FlagLoopback == 0 {
				stats.Interfaces = append(stats.Interfaces, i.Name)
			}
		}
	}

	// 4. Network Speed
	stats.NetRxSpeed, stats.NetTxSpeed = getNetSpeed()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func parseMemValue(s string) uint64 {
	var v uint64
	fmt.Sscanf(s, "%d", &v)
	return v
}

var lastIdle, lastTotal uint64

func getCPULoad() float64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == "cpu" {
			var user, nice, system, idle, iowait, irq, softirq, steal uint64
			fmt.Sscanf(fields[1], "%d", &user)
			fmt.Sscanf(fields[2], "%d", &nice)
			fmt.Sscanf(fields[3], "%d", &system)
			fmt.Sscanf(fields[4], "%d", &idle)
			fmt.Sscanf(fields[5], "%d", &iowait)
			fmt.Sscanf(fields[6], "%d", &irq)
			fmt.Sscanf(fields[7], "%d", &softirq)
			fmt.Sscanf(fields[8], "%d", &steal)

			total := user + nice + system + idle + iowait + irq + softirq + steal
			idleTotal := idle + iowait

			diffTotal := total - lastTotal
			diffIdle := idleTotal - lastIdle

			lastTotal = total
			lastIdle = idleTotal

			if diffTotal == 0 {
				return 0
			}
			return float64(diffTotal-diffIdle) / float64(diffTotal) * 100
		}
	}
	return 0
}

func getCPUTemp() float64 {
	// Try standard thermal zone
	data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		return 0
	}
	var temp int
	fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &temp)
	return float64(temp) / 1000.0
}

var (
	lastNetRx, lastNetTx uint64
	lastNetTime          time.Time
)

func getNetSpeed() (rxKBps, txKBps float64) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}

	var currentRx, currentTx uint64
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		// Skip headers
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) < 2 {
				continue
			}
			fields := strings.Fields(parts[1])
			if len(fields) < 9 {
				continue
			}
			// Sum up all interfaces (except lo ideally, but sum is fine for load estimation)
			if strings.TrimSpace(parts[0]) == "lo" {
				continue
			}

			var rx, tx uint64
			fmt.Sscanf(fields[0], "%d", &rx)
			fmt.Sscanf(fields[8], "%d", &tx)

			currentRx += rx
			currentTx += tx
		}
	}

	now := time.Now()
	if lastNetTime.IsZero() {
		lastNetRx = currentRx
		lastNetTx = currentTx
		lastNetTime = now
		return 0, 0
	}

	duration := now.Sub(lastNetTime).Seconds()
	if duration <= 0 {
		return 0, 0
	}

	rxKBps = float64(currentRx-lastNetRx) / 1024.0 / duration
	txKBps = float64(currentTx-lastNetTx) / 1024.0 / duration

	lastNetRx = currentRx
	lastNetTx = currentTx
	lastNetTime = now

	return rxKBps, txKBps
}

func handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	configFileMutex.Lock()
	defer configFileMutex.Unlock()

	// 1. Read Existing Config
	var configMap map[string]interface{}
	data, err := os.ReadFile(ConfigFilePath)
	if err == nil {
		if err := json.Unmarshal(data, &configMap); err != nil {
			log.Printf("[API] [Error] Corrupt config file read: %v. Aborting update to prevent data loss.", err)
			http.Error(w, "Internal Configuration Error (Read)", http.StatusInternalServerError)
			return
		}
	}
	if configMap == nil {
		// Only valid if file doesn't exist. If it exists but failed to unmarshal, we returned above.
		configMap = make(map[string]interface{})
	}

	// 2. Parse Incoming Partial Update
	var updateMap map[string]interface{}
	if err := json.Unmarshal(body, &updateMap); err != nil {
		log.Printf("[API] Failed to parse update JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 3. Merge Updates
	log.Printf("[API] DEBUG: Config Update from %s: %+v", r.RemoteAddr, updateMap)
	for k, v := range updateMap {
		configMap[k] = v
	}

	// 4. Write Back Full Config (Atomic Update)
	finalBytes, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal config", http.StatusInternalServerError)
		return
	}

	// Write to temp file first
	tmpFile := ConfigFilePath + ".tmp"
	if err := os.WriteFile(tmpFile, finalBytes, 0644); err != nil {
		log.Printf("[API] Failed to write temp config: %v", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}
	// Rename to atomic replace
	if err := os.Rename(tmpFile, ConfigFilePath); err != nil {
		log.Printf("[API] Failed to rename config: %v", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	log.Printf("[API] Configuration Updated (Merged/Atomic): %s", string(finalBytes))

	// 5. Update Memory State Immediately (Fix Race Condition)
	var cfg ConfigFile
	if json.Unmarshal(finalBytes, &cfg) == nil {
		deviceStatusMutex.Lock()
		deviceStatus.Camera.Resolution = cfg.Resolution
		deviceStatus.Camera.CameraDevice = cfg.CameraDevice
		deviceStatus.Camera.CameraType = cfg.CameraType
		deviceStatus.Camera.FCPort = cfg.FCPort
		deviceStatus.Camera.FCBaud = cfg.FCBaud

		deviceStatus.LiveKitConfig.LiveKitURL = cfg.LiveKitURL
		deviceStatus.LiveKitConfig.Token = cfg.LiveKitToken
		deviceStatusMutex.Unlock()
	}

	w.WriteHeader(http.StatusOK)
}
func handleSerialPorts(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode([]string{"/dev/ttyACM0"})
}
func handleWifiScan(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode([]string{"Vyom_Secure"})
}
func handleSaveConfig(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }
func handleLogs(w http.ResponseWriter, r *http.Request) {
	service := strings.ToLower(r.URL.Query().Get("service"))
	log.Printf("[DEBUG-TRACE] handleLogs: raw='%s' normalized='%s'", r.URL.Query().Get("service"), service)

	var filename string
	switch service {
	case "camera":
		filename = "camera.log"
	case "telemetry":
		filename = "telemetry.log"
	case "zerotier":
		filename = "zerotier.log"
	case "livekit":
		filename = "livekit.log"
	case "", "api":
		filename = "api.log"
	default:
		// Do NOT fallback to api.log for unknown services
		http.Error(w, fmt.Sprintf("Unknown service: %s", service), http.StatusBadRequest)
		return
	}

	// Prioritize Production Logs
	content, err := os.ReadFile("/var/log/vyom/" + filename)
	if err != nil {
		// Fallback to local (Dev)
		content, err = os.ReadFile(filename)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Log file not found: %s", filename), http.StatusNotFound)
		return
	}

	if r.URL.Query().Get("download") == "true" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		w.Header().Set("Content-Type", "text/plain")
		w.Write(content)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Return last 100 lines
	lines := strings.Split(string(content), "\n")
	if len(lines) > 200 {
		lines = lines[len(lines)-200:]
	}
	json.NewEncoder(w).Encode(lines)
}

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func handleCameras(w http.ResponseWriter, r *http.Request) {
	var cameras []string
	// Run v4l2-ctl --list-devices
	out, err := exec.Command("v4l2-ctl", "--list-devices").Output()
	if err == nil {
		log.Printf("[API] raw v4l2 output: %s", string(out)) // Log for debugging
		lines := strings.Split(string(out), "\n")
		var currentName string
		for _, line := range lines {
			if line == "" {
				continue
			}
			// Device group header (ends with colon)
			if strings.HasSuffix(line, ":") {
				currentName = strings.TrimSuffix(strings.TrimSpace(line), ":")
				// Clean up name (remove bus info if present)
				if idx := strings.LastIndex(currentName, "("); idx > 0 {
					currentName = strings.TrimSpace(currentName[:idx])
				}
				// Clean extra colons
				currentName = strings.TrimSuffix(currentName, ":")
			} else if strings.HasPrefix(strings.TrimSpace(line), "/dev/video") {
				// Device path
				devPath := strings.TrimSpace(line)

				// Duplicate Check by NAME only (we only want one entry per physical camera group)
				alreadyAdded := false
				for _, cam := range cameras {
					if strings.HasPrefix(cam, currentName) {
						alreadyAdded = true
						break
					}
				}

				if !alreadyAdded {
					cameras = append(cameras, fmt.Sprintf("%s (%s)", currentName, devPath))
				}
			}
		}
	} else {
		log.Printf("[API] v4l2-ctl failed: %v", err)
		// Fallback to simple file scan if v4l2-ctl missing
		files, _ := os.ReadDir("/dev")
		for _, f := range files {
			if strings.HasPrefix(f.Name(), "video") {
				cameras = append(cameras, "/dev/"+f.Name())
			}
		}
	}
	if len(cameras) == 0 {
		cameras = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cameras)
}

func handleInternalZeroTier(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var ztStatus ZeroTierStatus
	if err := json.NewDecoder(r.Body).Decode(&ztStatus); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	deviceStatusMutex.Lock()
	deviceStatus.ZeroTier = ztStatus
	deviceStatusMutex.Unlock()

	// Also write to file for persistence/other services if needed
	data, _ := json.Marshal(ztStatus)
	os.MkdirAll(StatusDir, 0777)
	os.WriteFile(ZeroTierStatusFile, data, 0666)

	// Report to Cloud if Connected
	if ztStatus.State == "Connected" && ztStatus.IPAddress != "" && apiClient != nil {
		go func() {
			if err := apiClient.SaveZeroTierConfig(ztStatus.IPAddress); err != nil {
				log.Printf("[API] Failed to report ZeroTier IP to Cloud: %v", err)
			}
			if ztStatus.NetworkID != "" {
				// We don't have NodeID in this struct yet, might need to update struct or assume it's sent
			}
		}()
	}

	w.WriteHeader(http.StatusOK)
}

func handleGCSEndpoints(w http.ResponseWriter, r *http.Request) {
	if apiClient == nil {
		http.Error(w, "Backend not initialized", http.StatusServiceUnavailable)
		return
	}
	endpoints, err := apiClient.GetTelemetryEndpoints()
	if err != nil {
		log.Printf("ERROR fetching GCS Endpoints: %v", err)
		http.Error(w, fmt.Sprintf("Failed to fetch endpoints: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("[API] Debug GCS Endpoints from Backend: %+v", endpoints)

	// Map to UI-friendly format (Host->ip, TelemetryPort->port)
	type UIEndpoint struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		IP      string `json:"ip"`
		Port    int    `json:"port"`
		Enabled bool   `json:"enabled"`
	}
	var uiEndpoints = make([]UIEndpoint, 0)
	for _, ep := range endpoints {
		uiEndpoints = append(uiEndpoints, UIEndpoint{
			ID:      ep.ID,
			Name:    ep.Name,
			IP:      ep.Host,
			Port:    ep.TelemetryPort,
			Enabled: ep.Enabled,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(uiEndpoints)
}

func handleLiveKitConfig(w http.ResponseWriter, r *http.Request) {
	// 1. Check Manual Config First
	deviceStatusMutex.RLock()
	manualURL := deviceStatus.LiveKitConfig.LiveKitURL
	manualToken := deviceStatus.LiveKitConfig.Token
	deviceStatusMutex.RUnlock()

	if manualURL != "" && manualToken != "" {
		log.Println("[API] Using Manual LiveKit Config from demo_config.json")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Token      string `json:"token"`
			LiveKitURL string `json:"livekit_url"`
		}{
			Token:      manualToken,
			LiveKitURL: manualURL,
		})
		return
	}

	// 2. Fallback to Auto-Provisioning (Also retrieves Manual Config from Cloud)
	if apiClient == nil || apiClient.Identity == nil {
		http.Error(w, "Identity not loaded", http.StatusServiceUnavailable)
		return
	}

	// ENABLED: Required to fetch manual config entered in Cloud Frontend.
	// NOTE: This also enables auto-provisioning if backend supports it.
	lkResp, err := apiClient.GetLiveKitToken(apiClient.Identity.DeviceID)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(lkResp)
		return
	}

	// Return 404 to indicate "Not Configured"
	http.Error(w, "LiveKit Not Configured", http.StatusNotFound)
}

func handleTelemetryConfig(w http.ResponseWriter, r *http.Request) {
	deviceStatusMutex.RLock()
	defer deviceStatusMutex.RUnlock()

	fcPort := deviceStatus.Camera.FCPort
	if fcPort == "" {
		fcPort = "auto"
	}
	fcBaud := deviceStatus.Camera.FCBaud
	if fcBaud == 0 {
		fcBaud = 57600
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		FCPort string `json:"fc_port"`
		FCBaud int    `json:"fc_baud"`
	}{
		FCPort: fcPort,
		FCBaud: fcBaud,
	})
}

func handleInternalTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Only allow local
	if !strings.HasPrefix(r.RemoteAddr, "127.0.0.1") && !strings.HasPrefix(r.RemoteAddr, "[::1]") {
		if !strings.Contains(r.RemoteAddr, "[::1]") { // IPv6 localhost check might differ
			// Allow for now, or ensure 127.0.0.1
		}
	}

	var update map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("[API] Failed to decode internal telemetry: %v", err)
		return
	}

	deviceStatusMutex.Lock()
	now := time.Now()
	lastTelemetryUpdate = now
	// We could also populate deviceStatus.Telemetry here if we wanted real-time mirroring
	// But simply updating the timestamp prevents "Disconnected" staleness.
	deviceStatusMutex.Unlock()

	w.WriteHeader(http.StatusOK)
}

func handleZeroTierConfig(w http.ResponseWriter, r *http.Request) {
	if apiClient == nil || apiClient.Identity == nil {
		http.Error(w, "Identity not loaded", http.StatusServiceUnavailable)
		return
	}
	config, err := apiClient.GetZeroTierConfig(apiClient.Identity.DeviceID)
	if err != nil {
		if strings.Contains(err.Error(), "DEVICE_FORGOTTEN") {
			log.Println("🚨 Device Forgotten by Cloud! Factory Resetting...")
			apiClient.ResetIdentity()
			os.Remove(ConfigFilePath)
			os.Exit(0) // Restart to Claim Mode
		}
		log.Printf("ERROR fetching ZeroTier Config: %v", err)
		http.Error(w, fmt.Sprintf("Failed to fetch ZT config: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func checkHardwareStatus() {
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		// 1. Check Camera Status
		// Try to connect to camera port 8081
		camConnected := false
		conn, err := net.DialTimeout("tcp", "127.0.0.1:8081", 500*time.Millisecond)
		if err == nil {
			camConnected = true
			conn.Close()
		}

		// 2. Check Telemetry Staleness
		// If no update for > 4 seconds, mark FC as disconnected
		fcConnected := false
		deviceStatusMutex.RLock()
		if time.Since(lastTelemetryUpdate) < 5*time.Second {
			fcConnected = true
		}
		deviceStatusMutex.RUnlock()

		// Update Status
		deviceStatusMutex.Lock()
		deviceStatus.Hardware.CamConnected = camConnected
		deviceStatus.Hardware.FCConnected = fcConnected

		if !fcConnected {
			// Clear Telemetry Data to reflect disconnection in UI
			deviceStatus.Telemetry = nil
		}
		deviceStatusMutex.Unlock()
	}
}
