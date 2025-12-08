package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/bluenviron/gomavlib/v3"
	"github.com/bluenviron/gomavlib/v3/pkg/dialects/common"
	lksdk "github.com/livekit/server-sdk-go/v2"
	webrtc "github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	ivfreader "github.com/pion/webrtc/v4/pkg/media/ivfreader"
)

// --- Configuration Constants ---
const (
	ConfigFileName = "demo_config.json"
	SetupPort      = ":8080"
	MavlinkAddress = "udp://:14550"
	PollInterval   = 3 * time.Second
)

var AuthAPIURL = "http://localhost:8080/api/v1/devices/auth"

func init() {
	if url := os.Getenv("BACKEND_URL"); url != "" {
		AuthAPIURL = url + "/api/v1/devices/auth"
	}
}

// --- Data Structures ---

// ConfigFile represents the local configuration structure
type ConfigFile struct {
	SSID       string `json:"ssid"`
	Password   string `json:"password"`
	Resolution string `json:"resolution"`
}

// TelemetryPayload matches the JSON structure required by the frontend
type TelemetryPayload struct {
	Timestamp         int64           `json:"timestamp"`
	Attitude          *Attitude       `json:"attitude,omitempty"`
	SysStatus         *SysStatus      `json:"sys_status,omitempty"`
	GlobalPositionInt *GlobalPosition `json:"global_position_int,omitempty"`
	Mode              string          `json:"mode,omitempty"`
	Armed             bool            `json:"armed"`
	GpsRawInt         *GpsRaw         `json:"gps_raw_int,omitempty"`
}

type Attitude struct {
	Roll  float32 `json:"roll"`
	Pitch float32 `json:"pitch"`
	Yaw   float32 `json:"yaw"`
}

type SysStatus struct {
	Voltage          float32 `json:"voltage"`
	BatteryRemaining int     `json:"battery_remaining"`
}

type GlobalPosition struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
	Alt float32 `json:"alt"` // Relative altitude in meters
	Hdg uint16  `json:"hdg"` // Heading in degrees
	Vx  int16   `json:"vx"`  // Ground X Speed (cm/s)
	Vy  int16   `json:"vy"`  // Ground Y Speed (cm/s)
	Vz  int16   `json:"vz"`  // Ground Z Speed (cm/s)
}

type GpsRaw struct {
	FixType    uint8 `json:"fix_type"`
	Satellites uint8 `json:"satellites_visible"`
}

type TelemetryMessage struct {
	Type    string           `json:"type"`
	Payload TelemetryPayload `json:"payload"`
}

// --- Auth Structures ---

type ChallengeResponse struct {
	Challenge string `json:"challenge"`
}

type VerifyRequest struct {
	DeviceID  string `json:"device_id"`
	Signature string `json:"signature"`
}

type VerifyResponse struct {
	Token string `json:"token"`
	URL   string `json:"url"` // LiveKit URL
}

// --- Global State ---
var (
	cameraResolution = "640x480"
	cameraBitrate    = "500k"
	cameraMutex      sync.Mutex
	cameraCancel     context.CancelFunc
	pairingCode      int64
	isConnected      bool
	statusMutex      sync.RWMutex
	privKey          ed25519.PrivateKey // Store private key for auth
	pubKeyHex        string             // Store public key for auth
)

// --- MAIN ENTRY POINT ---

func main() {
	log.Println("Booting Drone Device Middleware...")

	// 1. Generate/Load Identity (Mocking persistent identity for demo)
	generateIdentity()

	// 2. Check for Configuration
	if _, err := os.Stat(ConfigFileName); os.IsNotExist(err) {
		startSetupMode()
	} else {
		startMissionMode()
	}
}

// --- SETUP MODE ---

func startSetupMode() {
	log.Println("\n=== SETUP MODE DETECTED ===")
	log.Println("No configuration found. Starting Local Config Portal.")

	ip := getOutboundIP()
	log.Printf("\n>>> CONNECT YOUR PHONE HERE: http://%s%s <<<\n\n", ip, SetupPort)

	// API Handlers
	http.HandleFunc("/api/system-info", handleSystemInfo)
	http.HandleFunc("/api/wifi-scan", handleWifiScan)
	http.HandleFunc("/api/save-config", handleSaveConfig)

	// Serve React Frontend
	fs := http.FileServer(http.Dir("./ui/dist"))
	http.Handle("/", fs)

	log.Printf("Listening on %s...", SetupPort)
	if err := http.ListenAndServe(SetupPort, corsMiddleware(http.DefaultServeMux)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"pairing_code": pairingCode,
		"device_id":    fmt.Sprintf("DRONE-%d", pairingCode%1000), // Simple ID
		"version":      "v1.0.0-beta",
	}
	jsonResponse(w, resp)
}

func handleWifiScan(w http.ResponseWriter, r *http.Request) {
	// Mocked WiFi List
	networks := []map[string]interface{}{
		{"ssid": "Vyom-Office", "signal": 95, "secure": true},
		{"ssid": "Guest-Network", "signal": 80, "secure": false},
		{"ssid": "Pixel-Hotspot", "signal": 60, "secure": true},
		{"ssid": "Area-51", "signal": 20, "secure": true},
	}
	time.Sleep(1 * time.Second) // Simulate scan delay
	jsonResponse(w, networks)
}

func handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config ConfigFile
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate (Simple check)
	if config.SSID == "" {
		http.Error(w, "SSID is required", http.StatusBadRequest)
		return
	}

	// Save to file
	file, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(ConfigFileName, file, 0644); err != nil {
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	log.Printf("Configuration saved for SSID: %s. Rebooting...", config.SSID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success", "message":"Configuration saved. Rebooting..."}`))

	// Trigger "Reboot" (Exit after short delay, expecting user/supervisor to restart)
	// In a real device, we might exec.Command("reboot").
	go func() {
		time.Sleep(2 * time.Second)
		log.Println("REBOOTING NOW...")
		os.Exit(0)
	}()
}

// --- MISSION MODE ---

func startMissionMode() {
	log.Println("\n=== MISSION MODE STARTED ===")

	// Load Config
	data, err := os.ReadFile(ConfigFileName)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}
	var config ConfigFile
	json.Unmarshal(data, &config)

	if config.Resolution != "" {
		cameraResolution = config.Resolution
	}
	log.Printf("Loaded Config: SSID=%s, Resolution=%s", config.SSID, cameraResolution)

	// Since we are "connected" (mocked via Ethernet/Phone Hotspot), we proceed.

	// Start Local Dashboard for debugging (on different port if needed, or same)
	go func() {
		// Just a simple status endpoint for mission mode
		mux := http.NewServeMux()
		mux.HandleFunc("/api/status", handleStatus)
		// We could serve the same UI but maybe in a "Connected" state,
		// but for now let's just expose the status API for external checks.
		log.Println("Mission Monitor listening on :8081")
		http.ListenAndServe(":8081", corsMiddleware(mux))
	}()

	// 1. Announce Device
	announceDevice()

	// 2. Auth Loop -> LiveKit
	connectToLiveKit()
}

// --- Helpers & Logic ---

func generateIdentity() {
	var err error
	var pubKey ed25519.PublicKey
	pubKey, privKey, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate keys: %v", err)
	}
	pubKeyHex = hex.EncodeToString(pubKey)

	// Deterministic Pairing Code for Demo?
	// Ideally we want this to be persistent but for this script it regenerates.
	// That means if you reboot, the pairing code changes.
	// For a better demo, let's cache the key or code, but for now we follow previous logic.

	var codeVal uint64
	fmt.Sscanf(pubKeyHex[:8], "%x", &codeVal)
	pairingCode = int64(10000000 + (codeVal % 90000000))
	fmt.Printf("DEVICE PAIRING CODE: %d\n", pairingCode)
}

func announceDevice() {
	client := &http.Client{Timeout: 5 * time.Second}
	announceBody, _ := json.Marshal(map[string]interface{}{
		"pairing_code": pairingCode,
		"public_key":   pubKeyHex,
	})

	baseURL := "http://localhost:8080"
	if url := os.Getenv("BACKEND_URL"); url != "" {
		baseURL = url
	}
	// Fix: If running in Mission Mode, the 'localhost' refers to where current process is running.
	// If the backend is elsewhere, env var must be set.

	announceURL := baseURL + "/api/v1/devices/announce"
	// Update Main Auth URL
	AuthAPIURL = baseURL + "/api/v1/devices/auth"

	go func() {
		for {
			resp, err := client.Post(announceURL, "application/json", bytes.NewBuffer(announceBody))
			if err == nil && resp.StatusCode == http.StatusOK {
				log.Println("Device announced to Cloud Backend.")
				resp.Body.Close()
				break
			}
			if err != nil {
				log.Printf("Announce failed (will retry): %v", err)
			}
			time.Sleep(3 * time.Second)
		}
	}()
}

func connectToLiveKit() {
	client := &http.Client{Timeout: 5 * time.Second}
	var token, lkURL string

	for {
		// Poll for challenge
		resp, err := client.Post(AuthAPIURL+"/challenge", "application/json", bytes.NewBuffer([]byte(fmt.Sprintf(`{"device_id": "%d"}`, pairingCode))))
		if err != nil {
			log.Printf("Waiting for pairing... (%v)", err)
			time.Sleep(PollInterval)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			time.Sleep(PollInterval)
			continue
		}

		var challengeResp ChallengeResponse
		json.NewDecoder(resp.Body).Decode(&challengeResp)
		resp.Body.Close()

		signature := ed25519.Sign(privKey, []byte(challengeResp.Challenge))

		verifyBody, _ := json.Marshal(VerifyRequest{
			DeviceID:  fmt.Sprintf("%d", pairingCode),
			Signature: hex.EncodeToString(signature),
		})
		vResp, err := client.Post(AuthAPIURL+"/verify", "application/json", bytes.NewBuffer(verifyBody))
		if err != nil {
			time.Sleep(PollInterval)
			continue
		}

		if vResp.StatusCode == http.StatusOK {
			var verifyResp VerifyResponse
			json.NewDecoder(vResp.Body).Decode(&verifyResp)
			token = verifyResp.Token
			lkURL = verifyResp.URL
			vResp.Body.Close()
			break
		}
		vResp.Body.Close()
		time.Sleep(PollInterval)
	}

	// Connect
	log.Printf("Connecting to LiveKit: %s", lkURL)
	room, err := lksdk.ConnectToRoomWithToken(lkURL, token, lksdk.NewRoomCallback())
	if err != nil {
		log.Fatalf("Failed to connect to LiveKit: %v", err)
	}

	log.Printf("Connected to Room: %s", room.Name())

	statusMutex.Lock()
	isConnected = true
	statusMutex.Unlock()

	startCamera(room)
	runMavlink(room)
}

// --- Status & Utils ---

func handleStatus(w http.ResponseWriter, r *http.Request) {
	statusMutex.RLock()
	status := "WAITING"
	if isConnected {
		status = "CONNECTED"
	}
	resp := map[string]interface{}{
		"pairing_code": pairingCode,
		"status":       status,
		"resolution":   cameraResolution,
	}
	statusMutex.RUnlock()
	jsonResponse(w, resp)
}

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80") // Connect to Google DNS (doesn't send data)
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// --- EXISTING CAMERA & MAVLINK CODE ---

// startCamera starts the ffmpeg process and publishes to LiveKit.
func startCamera(room *lksdk.Room) {
	cameraMutex.Lock()
	defer cameraMutex.Unlock()

	// Stop existing camera if running
	if cameraCancel != nil {
		log.Println("Stopping previous camera instance...")
		cameraCancel()
		time.Sleep(500 * time.Millisecond) // Give it time to exit gracefully
	}

	ctx, cancel := context.WithCancel(context.Background())
	cameraCancel = cancel

	go func() {
		log.Printf("Starting camera: %s @ %s", cameraResolution, cameraBitrate)

		pipePath := "camera_pipe.ivf"

		// Ensure pipe cleanliness
		os.Remove(pipePath)
		if err := syscall.Mkfifo(pipePath, 0666); err != nil {
			log.Printf("Failed to create named pipe: %v", err)
			return
		}

		// Start ffmpeg
		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-f", "v4l2",
			"-video_size", cameraResolution,
			"-i", "/dev/video0",
			"-c:v", "libvpx",
			"-b:v", cameraBitrate,
			"-minrate", cameraBitrate,
			"-maxrate", cameraBitrate,
			"-r", "30",
			"-bufsize", "100k",
			"-cpu-used", "8",
			"-g", "10",
			"-qmin", "4",
			"-qmax", "40",
			"-static-thresh", "0",
			"-lag-in-frames", "0",
			"-error-resilient", "1",
			"-auto-alt-ref", "0",
			"-deadline", "realtime",
			"-f", "ivf",
			"-y", pipePath,
		)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Start(); err != nil {
			log.Printf("Failed to start ffmpeg: %v", err)
			return
		}

		// Process monitor
		go func() {
			if err := cmd.Wait(); err != nil {
				if ctx.Err() == nil {
					log.Printf("ffmpeg exited unexpectedly: %v", err)
				}
			}
		}()

		// Give ffmpeg a moment into the pipe
		time.Sleep(1 * time.Second)

		file, err := os.OpenFile(pipePath, os.O_RDONLY, os.ModeNamedPipe)
		if err != nil {
			log.Printf("Failed to open pipe: %v", err)
			return
		}
		defer file.Close()

		// Determine Dimensions for Track
		var width, height uint32
		fmt.Sscanf(cameraResolution, "%dx%d", &width, &height)

		track, err := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeVP8,
			ClockRate: 90000,
		})
		if err != nil {
			log.Printf("Failed to create sample track: %v", err)
			return
		}

		pub, err := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{
			Name:        "camera_feed",
			VideoWidth:  int(width),
			VideoHeight: int(height),
		})
		if err != nil {
			log.Printf("Failed to publish track: %v", err)
			return
		}
		defer room.LocalParticipant.UnpublishTrack(pub.SID())

		ivf, _, err := ivfreader.NewWith(file)
		if err != nil {
			log.Printf("Failed to create IVF reader: %v", err)
			return
		}

		log.Println("Camera stream active.")

		// Frame Loop
		for {
			select {
			case <-ctx.Done():
				return
			default:
				payload, _, err := ivf.ParseNextFrame()
				if err != nil {
					if err == io.EOF {
						return
					}
					return
				}
				track.WriteSample(media.Sample{
					Data:     payload,
					Duration: 33 * time.Millisecond,
				}, nil)
			}
		}
	}()
}

func runMavlink(room *lksdk.Room) {
	node, err := gomavlib.NewNode(gomavlib.NodeConf{
		Endpoints: []gomavlib.EndpointConf{
			gomavlib.EndpointUDPServer{Address: ":14550"},
		},
		Dialect:     common.Dialect,
		OutVersion:  gomavlib.V2,
		OutSystemID: 10,
	})
	if err != nil {
		// Just log error for demo if no mavlink source
		log.Printf("Warning: Failed to create MAVLink node (is port 14550 busy?): %v", err)
		return
	}
	defer node.Close()
	log.Println("Listening for MAVLink packets on :14550")

	// Telemetry State
	var currentAttitude *Attitude
	var currentSysStatus *SysStatus
	var currentGlobalPos *GlobalPosition
	var currentMode string = "UNKNOWN"
	var currentArmed bool = false
	var currentGpsRaw *GpsRaw
	lastMavlinkMsg := time.Now()

	// Publish Loop
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		simLat, simLon, simAlt, simHeading := -35.363262, 149.165237, 10.0, 0.0

		for range ticker.C {
			if time.Since(lastMavlinkMsg) > 5*time.Second {
				// Simulate
				simHeading += 1.0
				if simHeading >= 360 {
					simHeading = 0
				}
				currentMode = "SIMULATION"
				currentArmed = true
				currentAttitude = &Attitude{Yaw: float32(simHeading * 3.14 / 180)}
				currentGlobalPos = &GlobalPosition{Lat: simLat, Lon: simLon, Alt: float32(simAlt)}
			}

			if currentMode == "UNKNOWN" {
				continue
			}

			payload := TelemetryPayload{
				Timestamp:         time.Now().UnixMilli(),
				Attitude:          currentAttitude,
				SysStatus:         currentSysStatus,
				GlobalPositionInt: currentGlobalPos,
				Mode:              currentMode,
				Armed:             currentArmed,
				GpsRawInt:         currentGpsRaw,
			}

			data, _ := json.Marshal(TelemetryMessage{Type: "telemetry", Payload: payload})
			room.LocalParticipant.PublishData(data, lksdk.WithDataPublishReliable(false))
		}
	}()

	for evt := range node.Events() {
		if frm, ok := evt.(*gomavlib.EventFrame); ok {
			lastMavlinkMsg = time.Now()
			switch msg := frm.Message().(type) {
			case *common.MessageHeartbeat:
				currentArmed = (msg.BaseMode & common.MAV_MODE_FLAG_SAFETY_ARMED) != 0
				currentMode = fmt.Sprintf("MODE(%d)", msg.CustomMode)
			case *common.MessageAttitude:
				currentAttitude = &Attitude{Roll: msg.Roll, Pitch: msg.Pitch, Yaw: msg.Yaw}
			case *common.MessageSysStatus:
				currentSysStatus = &SysStatus{Voltage: float32(msg.VoltageBattery) / 1000, BatteryRemaining: int(msg.BatteryRemaining)}
			case *common.MessageGlobalPositionInt:
				currentGlobalPos = &GlobalPosition{Lat: float64(msg.Lat) / 1e7, Lon: float64(msg.Lon) / 1e7, Alt: float32(msg.RelativeAlt) / 1000, Hdg: msg.Hdg / 100}
			case *common.MessageGpsRawInt:
				currentGpsRaw = &GpsRaw{FixType: uint8(msg.FixType), Satellites: msg.SatellitesVisible}
			}
		}
	}
}
