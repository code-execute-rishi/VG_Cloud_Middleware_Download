package main

import (
	"context"
	"encoding/json"
	"fmt"
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
)

var (
	// Default Backend URL found in backend_client.go
	BackendBaseURL = DefaultBaseURL
)

func init() {
	if url := os.Getenv("BACKEND_URL"); url != "" {
		BackendBaseURL = url
	}
}

// --- Data Structures (Local Config) ---

type ConfigFile struct {
	SSID       string `json:"ssid"`
	Password   string `json:"password"`
	Resolution string `json:"resolution"`
}

// TelemetryPayload for Local MAVLink Publishing (LiveKit Data Channel)
type LiveKitTelemetry struct {
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
	Alt float32 `json:"alt"`
	Hdg uint16  `json:"hdg"`
	Vx  int16   `json:"vx"`
	Vy  int16   `json:"vy"`
	Vz  int16   `json:"vz"`
}

type GpsRaw struct {
	FixType    uint8 `json:"fix_type"`
	Satellites uint8 `json:"satellites_visible"`
}

type LiveKitDataMessage struct {
	Type    string           `json:"type"`
	Payload LiveKitTelemetry `json:"payload"`
}

// --- Global State ---
var (
	cameraResolution = "640x480"
	cameraBitrate    = "500k"
	cameraMutex      sync.Mutex
	cameraCancel     context.CancelFunc
	isConnected      bool
	statusMutex      sync.RWMutex

	// API Client
	apiClient *BackendClient
)

// --- MAIN ENTRY POINT ---

func main() {
	log.Println("Booting Drone Device Middleware...")

	// 1. Initialize API Client & Identity
	apiClient = NewBackendClient(BackendBaseURL)
	if err := apiClient.LoadOrCreateIdentity(); err != nil {
		log.Fatalf("Failed to load identity: %v", err)
	}

	// 2. Check for Configuration (WiFi/Setup)
	// Note: We still use 'demo_config.json' for local settings like Resolution/SSID
	if _, err := os.Stat(ConfigFileName); os.IsNotExist(err) {
		startSetupMode()
	} else {
		startMissionMode()
	}
}

// --- SETUP MODE ---

func startSetupMode() {
	log.Println("\n=== SETUP MODE DETECTED ===")
	log.Printf("Pairing Code: %d", apiClient.Identity.PairingCode)

	ip := getOutboundIP()
	log.Printf("\n>>> CONNECT TO: http://%s%s <<<\n\n", ip, SetupPort)

	http.HandleFunc("/api/system-info", handleSystemInfo)
	http.HandleFunc("/api/wifi-scan", handleWifiScan)
	http.HandleFunc("/api/save-config", handleSaveConfig)

	fs := http.FileServer(http.Dir("./ui/dist"))
	http.Handle("/", fs)

	if err := http.ListenAndServe(SetupPort, corsMiddleware(http.DefaultServeMux)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	// Return the pairing code so the frontend can display it
	resp := map[string]interface{}{
		"pairing_code": apiClient.Identity.PairingCode,
		"device_id":    apiClient.Identity.NodeID,
		"version":      "v1.0.0-beta",
	}
	jsonResponse(w, resp)
}

func handleWifiScan(w http.ResponseWriter, r *http.Request) {
	networks := []map[string]interface{}{
		{"ssid": "Vyom-Office", "signal": 95, "secure": true},
		{"ssid": "Guest-Network", "signal": 80, "secure": false},
	}
	time.Sleep(500 * time.Millisecond)
	jsonResponse(w, networks)
}

func handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Decode Request
	var config ConfigFile
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 2. REGISTER WITH CLOUD (CRITICAL STEP)
	// We must register successfully before we can save config and assume Mission Mode.
	log.Println("Validating Registration with Cloud...")

	if err := apiClient.Register(); err != nil {
		log.Printf("❌ Registration Failed: %v", err)

		// Return 500 so Frontend shows error
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Cloud Registration Failed: %v", err),
		})
		return
	}

	log.Println("✅ Device Registered Successfully!")

	// 3. Save Config (Only if Register succeeded)
	file, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(ConfigFileName, file, 0644); err != nil {
		http.Error(w, "Failed to write config file", http.StatusInternalServerError)
		return
	}

	// 4. Response & Reboot
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success", "message":"Registered and Configured. Rebooting..."}`))

	go func() {
		time.Sleep(2 * time.Second)
		log.Println("REBOOTING...")
		os.Exit(0)
	}()
}

// --- MISSION MODE ---

func startMissionMode() {
	log.Println("\n=== MISSION MODE STARTED ===")

	// Load Local Config
	data, err := os.ReadFile(ConfigFileName)
	if err == nil {
		var config ConfigFile
		json.Unmarshal(data, &config)
		if config.Resolution != "" {
			cameraResolution = config.Resolution
		}
	}

	// 1. Ensure Registered (Idempotent check)
	if err := apiClient.Register(); err != nil {
		log.Printf("Registration check failed (running offline?): %v", err)
	}

	// 2. Authenticate with Cloud (LiveKit)
	log.Println("Authenticating with Cloud...")
	authRes, err := apiClient.Authenticate()
	if err != nil {
		log.Fatalf("Authentication Failed: %v", err)
	}

	// 3. Connect to LiveKit
	log.Printf("Connecting to LiveKit Room: %s", authRes.RoomName)
	room, err := lksdk.ConnectToRoomWithToken(authRes.LiveKitURL, authRes.LiveKitToken, lksdk.NewRoomCallback())
	if err != nil {
		log.Fatalf("LiveKit Connection Failed: %v", err)
	}
	log.Println("LiveKit Connected!")

	statusMutex.Lock()
	isConnected = true
	statusMutex.Unlock()

	// 4. Start Subsystems
	startCamera(room)
	go startTelemetryLoop(room)

	// Keep alive
	select {}
}

func startTelemetryLoop(room *lksdk.Room) {
	node, err := gomavlib.NewNode(gomavlib.NodeConf{
		Endpoints: []gomavlib.EndpointConf{
			gomavlib.EndpointUDPServer{Address: ":14550"},
		},
		Dialect:     common.Dialect,
		OutVersion:  gomavlib.V2,
		OutSystemID: 10,
	})
	if err != nil {
		log.Printf("MAVLink Bind Warning: %v", err)
	}

	// State
	var currentBattery int = 100

	ticker := time.NewTicker(2 * time.Second)

	// Simulation Variables
	simLat, simLon := -35.363262, 149.165237

	go func() {
		for range ticker.C {
			simLon += 0.0001
			currentBattery -= 1
			if currentBattery < 0 {
				currentBattery = 100
			}

			// 1. Send to Cloud (REST API)
			update := TelemetryUpdate{
				Latitude:       simLat,
				Longitude:      simLon,
				Altitude:       10,
				Speed:          5.0,
				Heading:        90,
				Battery:        currentBattery,
				SignalStrength: 85,
			}
			if err := apiClient.UpdateTelemetry(update); err != nil {
				log.Printf("Cloud Telemetry Error: %v", err)
			} else {
				log.Printf("Telemetry pushed to Cloud.")
			}

			// 2. Publish to LiveKit
			lkPayload := LiveKitTelemetry{
				Timestamp: time.Now().UnixMilli(),
				GlobalPositionInt: &GlobalPosition{
					Lat: simLat,
					Lon: simLon,
					Alt: 10,
				},
				SysStatus: &SysStatus{
					BatteryRemaining: currentBattery,
				},
				Mode:  "GUIDED",
				Armed: true,
			}
			data, _ := json.Marshal(LiveKitDataMessage{Type: "telemetry", Payload: lkPayload})
			room.LocalParticipant.PublishData(data, lksdk.WithDataPublishReliable(false))
		}
	}()

	if node != nil {
		defer node.Close()
		log.Println("MAVLink Bridge Active")
		for evt := range node.Events() {
			if frm, ok := evt.(*gomavlib.EventFrame); ok {
				_ = frm
			}
		}
	}
}

// --- Utils ---

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

// --- Camera (Unchanged) ---

func startCamera(room *lksdk.Room) {
	cameraMutex.Lock()
	defer cameraMutex.Unlock()
	if cameraCancel != nil {
		cameraCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	cameraCancel = cancel

	go func() {
		pipePath := "camera_pipe.ivf"
		os.Remove(pipePath)
		syscall.Mkfifo(pipePath, 0666)

		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-f", "v4l2", "-video_size", cameraResolution, "-i", "/dev/video0",
			"-c:v", "libvpx", "-b:v", cameraBitrate, "-deadline", "realtime",
			"-f", "ivf", "-y", pipePath,
		)
		if err := cmd.Start(); err != nil {
			log.Printf("Camera Start Failed: %v", err)
			return
		}

		time.Sleep(1 * time.Second)
		file, err := os.OpenFile(pipePath, os.O_RDONLY, os.ModeNamedPipe)
		if err != nil {
			return
		}
		defer file.Close()

		var width, height uint32
		fmt.Sscanf(cameraResolution, "%dx%d", &width, &height)
		track, _ := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000})
		pub, _ := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{Name: "camera_feed", VideoWidth: int(width), VideoHeight: int(height)})
		if pub != nil {
			defer room.LocalParticipant.UnpublishTrack(pub.SID())
		}

		ivf, _, _ := ivfreader.NewWith(file)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				payload, _, err := ivf.ParseNextFrame()
				if err != nil {
					return
				}
				track.WriteSample(media.Sample{Data: payload, Duration: 33 * time.Millisecond}, nil)
			}
		}
	}()
}
