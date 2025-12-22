package main

import (
	"context"
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
	"syscall"
	"time"

	"github.com/bluenviron/gomavlib/v3"
	"github.com/bluenviron/gomavlib/v3/pkg/dialects/common"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	webrtc "github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	ivfreader "github.com/pion/webrtc/v4/pkg/media/ivfreader"
)

// --- Configuration Constants ---
const (
	ConfigFileName = "demo_config.json"
	SetupPort      = ":8085" // Changed to 8085 to bust cache
	MavlinkAddress = "udp://:14550"
)

var (
	BackendBaseURL  = DefaultBaseURL
	FrontendBaseURL = "https://middleware-gcs-assigment.vercel.app" // Default Frontend
)

func init() {
	// Allow Overriding via Environment Variable
	if envURL := os.Getenv("BACKEND_URL"); envURL != "" {
		BackendBaseURL = envURL
	} else {
		BackendBaseURL = "https://vg-cloud-backend.onrender.com"
	}
	log.Printf("🔗 Backend URL Set to: %s", BackendBaseURL)
}

// --- Data Structures ---

type ConfigFile struct {
	SSID       string `json:"ssid"`
	Password   string `json:"password"`
	Resolution string `json:"resolution"`
	FCPort     string `json:"fc_port"`
	FCBaud     int    `json:"fc_baud"`
}

// --- Data Functions ---
// Identity struct moved to line 153

// ConnectionState Enum
type ConnectionState string

const (
	StateConnected    ConnectionState = "Connected"
	StateDisconnected ConnectionState = "Disconnected"
	StateError        ConnectionState = "Error"
)

type LiveKitStatus struct {
	State        ConnectionState `json:"state"`
	RoomName     string          `json:"room_name"`
	Participants int             `json:"participants"`
	LastError    string          `json:"last_error"`
}

type ZeroTierStatus struct {
	State     ConnectionState `json:"state"`
	NetworkID string          `json:"network_id"`
	IPAddress string          `json:"ip_address"`
	LastError string          `json:"last_error"`
}

type AuthStatus struct {
	ConnectURL string `json:"connect_url"`
}

type GlobalDeviceStatus struct {
	IsConfigured bool           `json:"is_configured"`
	IsConnected  bool           `json:"is_connected"` // Cloud/LiveKit Connected
	IsClaimed    bool           `json:"is_claimed"`
	Camera       CameraConfig   `json:"camera_config"`
	Hardware     HardwareStatus `json:"hardware_status"`
	LiveKit      LiveKitStatus  `json:"livekit_status"`
	ZeroTier     ZeroTierStatus `json:"zerotier_status"`
	Auth         AuthStatus     `json:"auth_status"`
}

type HardwareStatus struct {
	FCConnected  bool `json:"fc_connected"`
	CamConnected bool `json:"cam_connected"`
}

type CameraConfig struct {
	Resolution string `json:"resolution"`
}

// Telemetry & LiveKit Structs (Same as before)
type LiveKitTelemetry struct {
	Timestamp           int64                `json:"timestamp"`
	Attitude            *Attitude            `json:"attitude,omitempty"`
	SysStatus           *SysStatus           `json:"sys_status,omitempty"`
	GlobalPositionInt   *GlobalPosition      `json:"global_position_int,omitempty"`
	Mode                string               `json:"mode,omitempty"`
	Armed               bool                 `json:"armed"`
	GpsRawInt           *GpsRaw              `json:"gps_raw_int,omitempty"`
	VfrHud              *VfrHud              `json:"vfr_hud,omitempty"`
	NavControllerOutput *NavControllerOutput `json:"nav_controller_output,omitempty"`
	MissionCurrent      *MissionCurrent      `json:"mission_current,omitempty"`
	HomePosition        *HomePosition        `json:"home_position,omitempty"`
}
type NavControllerOutput struct {
	NavRoll       float32 `json:"nav_roll"`
	NavPitch      float32 `json:"nav_pitch"`
	NavBearing    int16   `json:"nav_bearing"`
	TargetBearing int16   `json:"target_bearing"`
	WpDist        uint16  `json:"wp_dist"`
	AltError      float32 `json:"alt_error"`
	AspdError     float32 `json:"aspd_error"`
	XtrackError   float32 `json:"xtrack_error"`
}
type MissionCurrent struct {
	Seq uint16 `json:"seq"`
}
type HomePosition struct {
	Lat int32 `json:"lat"`
	Lon int32 `json:"lon"`
	Alt int32 `json:"alt"`
}
type VfrHud struct {
	Airspeed    float32 `json:"airspeed"`
	Groundspeed float32 `json:"groundspeed"`
	Heading     int16   `json:"heading"`
	Throttle    uint16  `json:"throttle"`
	Alt         float32 `json:"alt"`
	Climb       float32 `json:"climb"`
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
	Lat int32  `json:"lat"`
	Lon int32  `json:"lon"`
	Alt int32  `json:"alt"`
	Hdg uint16 `json:"hdg"`
	Vx  int16  `json:"vx"`
	Vy  int16  `json:"vy"`
	Vz  int16  `json:"vz"`
}
type GpsRaw struct {
	FixType    uint8 `json:"fix_type"`
	Satellites uint8 `json:"satellites_visible"`
}
type LiveKitDataMessage struct {
	Type string           `json:"type"`
	Data LiveKitTelemetry `json:"data"`
}

// --- MISSING TYPES (Moved from backend_client.go) ---
type Identity struct {
	DeviceID  string `json:"device_id"`
	Token     string `json:"token"`
	AuthToken string `json:"auth_token"`
}

type CheckClaimRequest struct {
	DeviceID string `json:"device_id"`
}

type CheckClaimResponse struct {
	Claim   bool   `json:"claim_status"`
	Message string `json:"message"`
}

type TelemetryUpdate struct {
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	Altitude       float32 `json:"altitude"`
	Speed          float32 `json:"speed"`
	Heading        float32 `json:"heading"`
	SignalStrength int     `json:"signal_strength"`
	Battery        int     `json:"battery"`
	Armed          bool    `json:"armed"`
	FlightMode     string  `json:"flight_mode"`
}

type ZerotierConfig struct {
	NetworkID string `json:"network_id"`
}

type VerifyResponse struct {
	LiveKitToken string         `json:"livekit_token"`
	LiveKitURL   string         `json:"livekit_url"`
	AuthToken    string         `json:"auth_token"`
	RoomName     string         `json:"room_name"`
	Zerotier     ZerotierConfig `json:"zerotier"`
}

type LKTokenResponse struct {
	Token      string `json:"token"`
	LiveKitURL string `json:"livekit_url"`
}

// --- Global State ---
var (
	cameraMutex  sync.Mutex
	cameraCancel context.CancelFunc

	// Thread-safe Status
	deviceStatus      GlobalDeviceStatus
	deviceStatusMutex sync.RWMutex

	// Global Room Access for Restarts
	activeRoom      *lksdk.Room
	activeRoomMutex sync.Mutex

	apiClient *BackendClient

	// V3 Setup Logic State
	setupConnectURL string
)

// --- MAIN ENTRY POINT ---

func main() {
	resetFlag := flag.Bool("reset", false, "Reset configuration and identity")
	flag.Parse()

	if *resetFlag {
		log.Println("WARNING: Resetting Device Configuration...")
		os.Remove(ConfigFileName)
		os.Remove(IdentityFile)
		log.Println("Reset complete. Starting fresh...")
	}

	log.Println("Booting Drone Device Middleware...")

	// Setup Multi-Writer Logging (File + Console)
	logFile, err := os.OpenFile("device.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	} else {
		log.Println("⚠️ Failed to open device.log for writing")
	}

	// 1. Initialize API Client & Identity
	apiClient = NewBackendClient(BackendBaseURL)
	if err := apiClient.LoadOrCreateIdentity(); err != nil {
		log.Fatalf("Failed to load identity: %v", err)
	}

	// 2. Start Persistent Web Server (For UI and Setup)
	go startWebServer()

	// 3. Main Loop
	runStateLoop()
}

func startWebServer() {
	ip := getOutboundIP()
	log.Printf("\n>>> WEB UI AVAILABLE: http://%s%s <<<\n", ip, SetupPort)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/system-info", handleSystemInfo)
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/update-config", handleUpdateConfig)
	mux.HandleFunc("/claim", handleClaim) // V3: Direct Claim Handler
	mux.HandleFunc("/api/serial-ports", handleSerialPorts)
	mux.HandleFunc("/api/wifi-scan", handleWifiScan)
	mux.HandleFunc("/api/save-config", handleSaveConfig)
	mux.HandleFunc("/api/logs", handleLogs)
	mux.HandleFunc("/api/stream", handleLocalStream)

	// Determine UI Directory
	uiDir := "./ui/dist"
	if _, err := os.Stat(uiDir); os.IsNotExist(err) {
		// Fallback to installed location (Debian Package)
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

func handleClaim(w http.ResponseWriter, r *http.Request) {
	// Expected Query Pars: ?token=JWT&device_id=UUID&name=MyDrone
	token := r.URL.Query().Get("token")
	deviceID := r.URL.Query().Get("device_id")
	authToken := r.URL.Query().Get("auth_token")

	if token == "" || deviceID == "" {
		http.Error(w, "Missing token or device_id", http.StatusBadRequest)
		return
	}

	// Save Identity
	apiClient.Identity = &Identity{
		DeviceID:  deviceID,
		Token:     token,
		AuthToken: authToken, // New: Save API Auth Token
	}
	apiClient.TypifySaveIdentity()

	// Create Config File to mark "Configured"
	config := ConfigFile{
		Resolution: "640x480",
		FCPort:     "auto",
		FCBaud:     57600,
	}
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(ConfigFileName, data, 0644)

	log.Println("✅ [Claim] Token Received! Restarting Process...")
	w.Write([]byte("Device Claimed! Restarting..."))

	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
}

func setStatus(configured, connected, claimed bool) {
	deviceStatusMutex.Lock()
	defer deviceStatusMutex.Unlock()
	deviceStatus.IsConfigured = configured
	deviceStatus.IsConnected = connected
	deviceStatus.IsClaimed = claimed
}

func handleLocalStream(w http.ResponseWriter, r *http.Request) {
	// 1. Connect to GStreamer's raw TCP output with Retry
	var conn net.Conn
	var err error

	// Try for 5 seconds
	for i := 0; i < 10; i++ {
		conn, err = net.Dial("tcp", "127.0.0.1:8081")
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err != nil {
		http.Error(w, "Camera Stream Unavailable", http.StatusServiceUnavailable)
		log.Printf("[Proxy] Failed to connect to GStreamer (127.0.0.1:8081): %v", err)
		return
	}
	defer conn.Close()

	// 2. Set headers for MJPEG Stream
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=vyomboundary")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// 3. Proxy the bytes directly
	buf := make([]byte, 32*1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		if _, err := w.Write(buf[:n]); err != nil {
			break
		}
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

// --- STATE LOOP ---

func runStateLoop() {
	for {
		// Load Config
		var config ConfigFile
		configData, err := os.ReadFile(ConfigFileName)
		if err == nil {
			json.Unmarshal(configData, &config)
		}

		if os.IsNotExist(err) {
			setStatus(false, false, false)

			// --- V3 DEVICE FLOW LOGIC ---
			ip := getOutboundIP()
			// Construct Magic Link: Frontend URL + Callback to Local Device
			callbackURL := fmt.Sprintf("http://%s%s/claim", ip, SetupPort)
			// Frontend expects: /connect?callback=...
			setupConnectURL = fmt.Sprintf("%s/connect?callback=%s", FrontendBaseURL, callbackURL)

			log.Printf("\n>>> 🚀 SETUP REQUIRED 🚀 <<<\n")
			log.Printf("1. Open: %s\n", setupConnectURL)
			log.Printf("2. Login & Click 'Connect Device'\n")
			log.Printf("Waiting for token...\n")

			// Update UI Status
			deviceStatusMutex.Lock()
			deviceStatus.Auth = AuthStatus{
				ConnectURL: setupConnectURL,
			}
			deviceStatus.IsConfigured = false
			deviceStatusMutex.Unlock()

			time.Sleep(5 * time.Second)
			continue
		}

		// 2. Load Identity & Config
		if apiClient.Identity == nil {
			if err := apiClient.LoadOrCreateIdentity(); err != nil {
				log.Printf("Failed to load identity: %v - Retrying in 5s...", err)
				time.Sleep(5 * time.Second)
				continue
			}
		}

		log.Println("[Loop] Config & Identity Found. Starting Mission Services...")

		// 2. Authenticate
		// V3: Token is in Identity. We just use it.
		// Optional: VerifyToken endpoint call
		log.Println("[Loop] Authenticating...")
		// Assuming Authenticate() in backend_client.go is updated to check/refresh token
		_, err = apiClient.Authenticate()
		if err != nil {
			deviceStatusMutex.Lock()
			deviceStatus.LiveKit.State = StateError
			deviceStatus.LiveKit.LastError = "Auth Failed"
			deviceStatusMutex.Unlock()

			if strings.Contains(err.Error(), "DEVICE_FORGOTTEN") {
				log.Println("⚠️ DEVICE FORGOTTEN BY CLOUD. RESETTING IDENTITY... ⚠️")

				// Graceful Camera Shutdown
				cameraMutex.Lock()
				if cameraCancel != nil {
					cameraCancel()
				}
				cameraMutex.Unlock()

				time.Sleep(2 * time.Second) // Allow GStreamer to handle signal

				if resetErr := apiClient.ResetIdentity(); resetErr != nil {
					log.Printf("[Loop] Reset Identity Failed: %v", resetErr)
				}
				os.Remove(ConfigFileName) // Configure wipe
				os.Exit(0)                // Restart process
			}
			log.Printf("[Loop] Auth Failed: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// 2.5 Check Claim Status IMMEDIATELY
		isClaimed, _ := apiClient.CheckClaim()
		if isClaimed {
			log.Println("[Loop] Device is ALREADY Claimed.")
			deviceStatusMutex.Lock()
			deviceStatus.IsClaimed = true
			deviceStatusMutex.Unlock()
		} else {
			// In V3, we get Token => We are claimed.
			// But if user unclaims in backend, this check catches it.
		}

		// 3. Connect to LiveKit
		log.Printf("[Loop] Connecting to Room: %s", apiClient.Identity.DeviceID)
		connectToLiveKit(apiClient.Identity.DeviceID, apiClient)

		// 4. Start Telemetry Loop & MAVLink
		// Pass the parsed Config to the telemetry loop
		telemetryAndClaimLoop(apiClient, config)

		// If loop returns, it means we lost connection or Reset triggered.
		// Wait before restart
		time.Sleep(2 * time.Second)
	}
}

// --- TELEMETRY & CLAIM LOOP ---

func telemetryAndClaimLoop(client *BackendClient, config ConfigFile) {
	// Start MAVLink
	// Using GStreamer Pipeline for Video

	// Determine Resolution
	res := config.Resolution
	if res == "" {
		res = "640x480"
	}
	// START CAMERA
	ctx, cancel := context.WithCancel(context.Background())
	cameraMutex.Lock()
	cameraCancel = cancel
	// cameraResolution = res // Updated Global (Removed unused)
	cameraMutex.Unlock()

	go startCamera(ctx, res)
	defer cancel()

	// MAVLINK SETUP
	fcPort := config.FCPort
	fcBaud := config.FCBaud
	if fcPort == "" {
		fcPort = "auto"
	}
	if fcBaud == 0 {
		fcBaud = 57600
	}

	var node *gomavlib.Node
	var err error

	var endpoints []gomavlib.EndpointConf

	if fcPort == "auto" {
		log.Println("[Hardware] Auto-Detecting Flight Controller...")
		// 1. Search for Serial Ports
		files, _ := os.ReadDir("/dev")
		var foundPort string
		for _, f := range files {
			if strings.HasPrefix(f.Name(), "ttyACM") || strings.HasPrefix(f.Name(), "ttyUSB") {
				foundPort = "/dev/" + f.Name()
				break
			}
		}

		if foundPort != "" {
			log.Printf("[Hardware] ✅ Found Physical Device: %s", foundPort)
			fcPort = foundPort
			endpoints = []gomavlib.EndpointConf{
				gomavlib.EndpointSerial{Device: foundPort, Baud: fcBaud},
			}
		} else {
			log.Println("[Hardware] ⚠️ No Physical Device Found. Switching to SITL Mode (UDP :14550)...")
			endpoints = []gomavlib.EndpointConf{
				gomavlib.EndpointUDPServer{Address: ":14550"},
			}
		}
	} else {
		// Manual Configuration
		log.Printf("[Hardware] Connecting to configured port: %s", fcPort)
		if strings.HasPrefix(fcPort, "tcp:") {
			address := strings.TrimPrefix(fcPort, "tcp:")
			endpoints = []gomavlib.EndpointConf{gomavlib.EndpointTCPClient{Address: address}}
		} else if strings.HasPrefix(fcPort, "udp:") {
			address := strings.TrimPrefix(fcPort, "udp:")
			endpoints = []gomavlib.EndpointConf{gomavlib.EndpointUDPClient{Address: address}}
		} else {
			endpoints = []gomavlib.EndpointConf{
				gomavlib.EndpointSerial{Device: fcPort, Baud: fcBaud},
			}
		}
	}

	node, err = gomavlib.NewNode(gomavlib.NodeConf{
		Endpoints:   endpoints,
		Dialect:     common.Dialect,
		OutVersion:  gomavlib.V2,
		OutSystemID: 10,
	})

	if err != nil {
		log.Printf("[MAVLink] Connection Failed: %v", err)
	} else {
		defer node.Close()
		log.Printf("[MAVLink] Link Active")

		// Request Data Streams
		msg := &common.MessageRequestDataStream{
			TargetSystem:    1,
			TargetComponent: 1,
			ReqStreamId:     uint8(common.MAV_DATA_STREAM_ALL),
			ReqMessageRate:  4, // 4 Hz
			StartStop:       1,
		}
		node.WriteMessageAll(msg)
		log.Println("[MAVLink] Requested Data Streams (4Hz)")
	}

	ticker := time.NewTicker(300 * time.Millisecond) // 3Hz Telemetry
	defer ticker.Stop()

	claimTicker := time.NewTicker(5 * time.Second)
	defer claimTicker.Stop()

	// Local Telemetry State (Raw MAVLink Units)
	var telemMutex sync.RWMutex
	var telemLat, telemLon int32
	var telemAlt int32                           // mm
	var telemHdg uint16                          // cdeg
	var telemSpeed, telemBatt, telemVolt float32 // Speed m/s, Battery %, Voltage V
	var telemMode string = "Standby"
	var telemVfrHud VfrHud
	var telemGpsRaw GpsRaw
	var telemNavOutput NavControllerOutput
	var telemMissionCurrent MissionCurrent
	var telemHome HomePosition

	var telemLastHeartbeat int64

	// MAVLink Monitor Routine
	if node != nil {
		go func() {
			for evt := range node.Events() {
				select {
				case <-ctx.Done():
					return
				default:
					if frm, ok := evt.(*gomavlib.EventFrame); ok {
						// Heartbeat
						if _, ok := frm.Message().(*common.MessageHeartbeat); ok {
							telemLastHeartbeat = time.Now().Unix()
							deviceStatusMutex.Lock()
							deviceStatus.Hardware.FCConnected = true
							deviceStatusMutex.Unlock()
						}
						// Global Position
						if msg, ok := frm.Message().(*common.MessageGlobalPositionInt); ok {
							telemMutex.Lock()
							telemLat = msg.Lat
							telemLon = msg.Lon
							telemAlt = msg.Alt
							telemHdg = msg.Hdg
							telemMutex.Unlock()
						}
						// Sys Status (Battery)
						if msg, ok := frm.Message().(*common.MessageSysStatus); ok {
							telemMutex.Lock()
							telemBatt = float32(msg.BatteryRemaining)
							telemVolt = float32(msg.VoltageBattery) / 1000.0 // mV to V
							telemMutex.Unlock()
							// Debug Log
							if telemVolt > 0 {
								// log.Printf("🔋 Battery: %d%%, Voltage: %.2fV", msg.BatteryRemaining, telemVolt)
							}
						}
						// VFR HUD (Speed)
						if msg, ok := frm.Message().(*common.MessageVfrHud); ok {
							telemMutex.Lock()
							telemSpeed = float32(msg.Groundspeed)
							telemVfrHud = VfrHud{
								Airspeed:    msg.Airspeed,
								Groundspeed: msg.Groundspeed,
								Heading:     msg.Heading,
								Throttle:    msg.Throttle,
								Alt:         msg.Alt,
								Climb:       msg.Climb,
							}
							telemMutex.Unlock()
							// log.Printf("🚀 Speed: %.2f m/s", telemSpeed)
						}
						// GPS RAW INT (Satellites)
						if msg, ok := frm.Message().(*common.MessageGpsRawInt); ok {
							telemMutex.Lock()
							telemGpsRaw = GpsRaw{
								FixType:    uint8(msg.FixType),
								Satellites: msg.SatellitesVisible,
							}
							telemMutex.Unlock()
						}
						// NAV CONTROLLER OUTPUT (Wp Dist, etc)
						if msg, ok := frm.Message().(*common.MessageNavControllerOutput); ok {
							telemMutex.Lock()
							telemNavOutput = NavControllerOutput{
								NavRoll:       msg.NavRoll,
								NavPitch:      msg.NavPitch,
								NavBearing:    msg.NavBearing,
								TargetBearing: msg.TargetBearing,
								WpDist:        msg.WpDist,
								AltError:      msg.AltError,
								AspdError:     msg.AspdError,
								XtrackError:   msg.XtrackError,
							}
							telemMutex.Unlock()
						}
						// MISSION CURRENT (Seq)
						if msg, ok := frm.Message().(*common.MessageMissionCurrent); ok {
							telemMutex.Lock()
							telemMissionCurrent = MissionCurrent{
								Seq: msg.Seq,
							}
							telemMutex.Unlock()
							log.Printf("📍 Mission Current: %d", msg.Seq)
						}
						// HOME POSITION
						if msg, ok := frm.Message().(*common.MessageHomePosition); ok {
							telemMutex.Lock()
							telemHome = HomePosition{
								Lat: msg.Latitude,
								Lon: msg.Longitude,
								Alt: msg.Altitude,
							}
							telemMutex.Unlock()
							log.Printf("🏠 Home Position Recv: Lat=%d Lon=%d", msg.Latitude, msg.Longitude)
						}
					}
				}
			}
		}()
	}

	for {
		select {
		case <-ticker.C:
			// Send Telemetry to LiveKit Data Channel
			if activeRoom != nil && activeRoom.LocalParticipant != nil {

				// Prepare Payload
				telemMutex.RLock()
				payload := LiveKitTelemetry{
					Timestamp: time.Now().UnixMilli(),
					GlobalPositionInt: &GlobalPosition{
						Lat: telemLat,
						Lon: telemLon,
						Alt: telemAlt,
						Hdg: telemHdg,
					},
					SysStatus: &SysStatus{
						BatteryRemaining: int(telemBatt),
						Voltage:          telemVolt,
					},
					VfrHud:              &telemVfrHud,
					GpsRawInt:           &telemGpsRaw,
					Mode:                telemMode,
					Armed:               false, // Todo read heartbeat custom mode
					NavControllerOutput: &telemNavOutput,
					MissionCurrent:      &telemMissionCurrent,
					HomePosition:        &telemHome,
				}
				telemMutex.RUnlock()

				data, _ := json.Marshal(LiveKitDataMessage{
					Type: "telemetry",
					Data: payload,
				})

				if err := activeRoom.LocalParticipant.PublishData(data, lksdk.WithDataPublishReliable(true), lksdk.WithDataPublishTopic("telemetry")); err != nil {
					// log.Printf("Failed to publish data: %v", err)
					// Keep quiet on high freq errors
				}
			}

		case <-claimTicker.C:
			// Check Connection Health
			// Ping /telemetry endpoint to update "Last Seen"
			// Check Connection Health
			// Ping /telemetry endpoint to update "Last Seen"
			telemMutex.RLock()
			client.UpdateTelemetry(float64(telemLat)/1e7, float64(telemLon)/1e7, float64(telemAlt)/1000.0, float64(telemSpeed), float64(telemHdg), 100, int(telemBatt))
			telemMutex.RUnlock()

			isClaimed, isServerClaimed := client.CheckClaim()

			deviceStatusMutex.Lock()
			deviceStatus.IsClaimed = isClaimed || isServerClaimed

			// Check LiveKit
			deviceStatus.LiveKit.State = StateDisconnected
			if activeRoom != nil && activeRoom.LocalParticipant != nil {
				deviceStatus.LiveKit.State = StateConnected
				// deviceStatus.LiveKit.Participants = len(activeRoom.GetParticipants()) // Fix me: GetParticipants not available?
				deviceStatus.IsConnected = true
			} else {
				deviceStatus.IsConnected = false
			}

			// Hardware Status Check
			lastHb := telemLastHeartbeat
			_, vidErr := os.Stat("/dev/video0")

			deviceStatus.Hardware = HardwareStatus{
				FCConnected:  (time.Now().Unix() - lastHb) < 5,
				CamConnected: (vidErr == nil),
			}

			// Disambiguate: "Unclaimed" vs "Deleted"
			// If CheckClaim returns false, it could be valid-unclaimed OR deleted.
			// We must check Liveness to confirm existence.
			if !isServerClaimed {
				// We must check Liveness
				if err := client.CheckLiveness(); err != nil {
					if strings.Contains(err.Error(), "DEVICE_FORGOTTEN") {
						log.Println(">>> DEVICE FORGOTTEN (Hard Reset Confirmed) <<<")

						// Graceful Camera Shutdown
						cameraMutex.Lock()
						if cameraCancel != nil {
							cameraCancel()
						}
						cameraMutex.Unlock()

						time.Sleep(2 * time.Second) // Allow GStreamer to handle signal

						client.ResetIdentity()
						os.Remove(ConfigFileName) // Configure wipe
						os.Exit(0)
					}
				}
			}
			// deviceStatus.Hardware updated above
			deviceStatusMutex.Unlock()
		}
	}
}

// --- HTTP HANDLERS ---

func handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"device_id":  "unknown",
		"version":    "0.1.0",
		"build_time": time.Now().String(),
	}
	if apiClient != nil && apiClient.Identity != nil {
		resp["device_id"] = apiClient.Identity.DeviceID
	}
	json.NewEncoder(w).Encode(resp)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	deviceStatusMutex.RLock()
	defer deviceStatusMutex.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deviceStatus)
}

func handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	// Not implemented for demo
	w.WriteHeader(http.StatusOK)
}

func handleSerialPorts(w http.ResponseWriter, r *http.Request) {
	// Mock
	ports := []string{"/dev/ttyACM0", "/dev/ttyUSB0"}
	json.NewEncoder(w).Encode(ports)
}

func handleWifiScan(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode([]string{"Vyom_Secure", "Guest_Wifi"})
}

func handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	// Read last 50 lines of device.log
	// Simple implementation
	content, _ := os.ReadFile("device.log")
	lines := strings.Split(string(content), "\n")
	start := 0
	if len(lines) > 50 {
		start = len(lines) - 50
	}
	json.NewEncoder(w).Encode(lines[start:])
}

// --- UTILS ---

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
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
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func startCamera(ctx context.Context, res string) {
	// 1. Check if rpicam-vid exists
	rpiPath, err := exec.LookPath("rpicam-vid")
	_ = rpiPath // Silence "declared and not used"
	useLibCamera := (err == nil)
	// useLibCamera := false // Force V4L2 for testing on desktop

	// Crash Counter
	crashCount := 0
	lastCrashTime := time.Now()
	forceTestPattern := false

	for {
		// Check for Context Cancel
		select {
		case <-ctx.Done():
			log.Println("[Camera] Supervisor stopping...")
			return
		default:
		}

		var cmd *exec.Cmd
		// Resolution Parsing
		width := "640"
		height := "480"
		if res == "1280x720" {
			width = "1280"
			height = "720"
		}

		// determine mode
		useRealCamera := !forceTestPattern

		// If using libcamera
		if useRealCamera && useLibCamera {
			log.Println("[Camera] Using rpicam-vid (Libcamera)...")
			cmd = exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf(
				"rpicam-vid -t 0 --inline --width %s --height %s --framerate 30 --codec libav --libav-format yuv420p -o - | "+
					"gst-launch-1.0 fdsrc ! videoparse width=%s height=%s framerate=30/1 format=i420 ! "+
					"tee name=t ! queue max-size-buffers=1 leaky=downstream ! vp8enc deadline=1 ! avmux_ivf ! tcpclientsink host=127.0.0.1 port=5600 "+
					"t. ! queue ! videoscale ! video/x-raw,width=320,height=240 ! jpegenc ! multipartmux boundary=vyomboundary ! tcpserversink host=0.0.0.0 port=8081",
				width, height, width, height,
			))
		} else if useRealCamera {
			// Auto-Detect V4L2
			videoDevice := "/dev/video0"
			if _, err := os.Stat(videoDevice); err == nil {
				log.Printf("[Camera] Found physical camera at %s", videoDevice)
				fullCmd := fmt.Sprintf(
					"gst-launch-1.0 v4l2src device=%s ! videoconvert ! video/x-raw,format=I420,width=%s,height=%s,framerate=30/1 ! "+
						"tee name=t ! queue max-size-buffers=4 leaky=downstream ! vp8enc error-resilient=1 deadline=1 keyframe-max-dist=30 cpu-used=5 ! \"video/x-vp8\" ! queue ! avmux_ivf ! tcpclientsink host=127.0.0.1 port=5600 "+
						"t. ! queue max-size-buffers=4 leaky=downstream ! videoscale ! \"video/x-raw,width=320,height=240\" ! jpegenc ! multipartmux boundary=vyomboundary ! tcpserversink host=127.0.0.1 port=8081 sync=false",
					videoDevice, width, height,
				)
				log.Println("[Camera] Starting Pipeline Step: " + fullCmd)
				cmd = exec.Command("sh", "-c", fullCmd)
				cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			} else {
				log.Println("[Camera] No physical camera found. Using Test Source (Snow)...")
				// Fallback to snow immediately if no device
				forceTestPattern = true
				continue // Restart loop to hit the else block below (or handle here, but cleaner to restart)
			}
		}

		// If we decided to use test pattern (or fallback triggered)
		if forceTestPattern {
			log.Println("[Camera] Using Test Source (Snow)...")
			fullCmd := fmt.Sprintf(
				"gst-launch-1.0 videotestsrc is-live=true pattern=snow ! videoconvert ! video/x-raw,format=I420,width=%s,height=%s,framerate=30/1 ! "+
					"tee name=t ! queue max-size-buffers=4 leaky=downstream ! vp8enc error-resilient=1 deadline=1 keyframe-max-dist=30 cpu-used=5 ! \"video/x-vp8\" ! queue ! avmux_ivf ! tcpclientsink host=127.0.0.1 port=5600 "+
					"t. ! queue max-size-buffers=4 leaky=downstream ! videoscale ! \"video/x-raw,width=320,height=240\" ! jpegenc ! multipartmux boundary=vyomboundary ! tcpserversink host=127.0.0.1 port=8081 sync=false",
				width, height,
			)
			log.Println("[Camera] Starting Pipeline Step: " + fullCmd)
			cmd = exec.Command("sh", "-c", fullCmd)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		}

		// Start Process
		if err := cmd.Start(); err != nil {
			log.Printf("[Camera] Failed to start pipeline: %v. Retrying in 2s...", err)
			time.Sleep(2 * time.Second)
			continue
		}
		log.Printf("[Camera] ✅ Pipeline started with PID %d", cmd.Process.Pid)

		// Wait for Process to Exit (Blocking)
		err := cmd.Wait()

		// If we are here, process exited.
		log.Printf("[Camera] ⚠️ Pipeline Exited: %v", err)

		// Check for Context Cancel (Clean Shutdown)
		select {
		case <-ctx.Done():
			return // Normal exit
		default:
		}

		// Crash Logic
		if time.Since(lastCrashTime) < 30*time.Second {
			crashCount++
		} else {
			crashCount = 1 // Reset
		}
		lastCrashTime = time.Now()

		if crashCount >= 3 && !forceTestPattern {
			log.Println("[Camera] 🚨 Too many crashes! Falling back to Test Pattern (Snow) for stability.")
			forceTestPattern = true
		}

		log.Println("[Camera] Restarting pipeline in 2 seconds...")
		time.Sleep(2 * time.Second)
	}
}

func connectToLiveKit(roomName string, client *BackendClient) {
	activeRoomMutex.Lock()
	defer activeRoomMutex.Unlock()

	// Check connection state
	if activeRoom != nil && activeRoom.LocalParticipant != nil {
		return
	}

	log.Println("[LiveKit] Requesting Token...")
	token, err := client.GetLiveKitToken(roomName)
	if err != nil {
		log.Printf("[LiveKit] Token Error: %v", err)
		return
	}
	if token.LiveKitURL == "" {
		log.Println("[LiveKit] ❌ Critical: LiveKit URL is missing in backend response. Config required in Dashboard.")
		// STRICT MODE: Do not fallback. Let the loop retry later.
		return
	}

	// --- ZeroTier Integration ---
	// Attempt to pull ZeroTier config and join network
	ztConfig, err := client.GetZeroTierConfig(client.Identity.DeviceID)
	if err != nil {
		log.Printf("[ZeroTier] Failed to get config: %v", err)
	} else if ztConfig != nil && ztConfig.NetworkID != "" {
		log.Printf("[ZeroTier] Found Network ID: %s. Ensuring membership...", ztConfig.NetworkID)

		// Check current networks
		out, _ := exec.Command("zerotier-cli", "listnetworks").Output()
		if !strings.Contains(string(out), ztConfig.NetworkID) {
			log.Printf("[ZeroTier] Joining network %s...", ztConfig.NetworkID)
			if err := exec.Command("zerotier-cli", "join", ztConfig.NetworkID).Run(); err != nil {
				log.Printf("[ZeroTier] ❌ Join Failed: %v", err)
			} else {
				log.Println("[ZeroTier] ✅ Join command executed.")
			}
		} else {
			log.Println("[ZeroTier] Already joined network.")
		}
	} else {
		log.Println("[ZeroTier] No Network ID configured in Backend.")
	}

	// Connect
	cb := lksdk.NewRoomCallback()
	cb.OnParticipantConnected = func(p *lksdk.RemoteParticipant) {
		log.Printf("participant connected: %s", p.Identity())
	}
	// Handle Video Publishing
	// We read from pipe `camera_pipe.ivf`

	// Handle Video Publishing
	// We read from pipe `camera_pipe.ivf`

	// We used to do NewRoom + JoinWithToken but ConnectToRoom is easier if URL is dynamic.
	// However, ConnectToRoom might require APIKey/Secret if Token is not enough... wait.
	// We use JoinWithToken below.

	// We used to do NewRoom + JoinWithToken but ConnectToRoom is easier if URL is dynamic.
	// However, ConnectToRoom might require APIKey/Secret if Token is not enough... wait.
	// lksdk.ConnectToRoomWithToken is better?
	// Let's check sdk...
	// If ConnectToRoom uses callback and options...
	// Options can take Token? No.
	// Let's use JoinWithToken on a NewRoom to be safe and explicit.

	room2 := lksdk.NewRoom(cb)
	if err := room2.JoinWithToken(token.LiveKitURL, token.Token); err != nil {
		log.Printf("[LiveKit] Join Failed: %v", err)
		return
	}
	log.Println("[Loop] Connected! Joining Mission.")
	activeRoom = room2

	// Start Video Ingoroutine
	log.Printf("[Loop] Connected Identity: %s", room2.LocalParticipant.Identity())
	go publishVideoToRoom(room2)
}

func publishVideoToRoom(room *lksdk.Room) {
	// Create Track
	var track *lksdk.LocalSampleTrack
	var err error

	// Retry loop for Publishing Track
	for {
		track, err = lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeVP8,
		})
		if err != nil {
			log.Printf("[LiveKit] Failed to create track: %v. Retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if _, err := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{
			Name:        "camera_feed",
			Source:      livekit.TrackSource_CAMERA,
			VideoWidth:  640,
			VideoHeight: 480,
		}); err != nil {
			log.Printf("[LiveKit] Failed to publish track: %v. Retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// Success
		break
	}

	log.Println("[Video] Uplink Ready. Listening for Camera Stream on TCP :5600...")

	// Listen for GStreamer Connection (Server Mode)
	ln, err := net.Listen("tcp", ":5600")
	if err != nil {
		log.Printf("[Video] ❌ Failed to bind port 5600: %v", err)
		return
	}
	defer ln.Close()

	// Accept Loop
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[Video] Accept Error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Println("[Video] 🎥 Camera Connected!")

		// Reader Loop
		// Note: GStreamer tcpclientsink should send the IVF Header immediately upon connection.
		ivf, header, err := ivfreader.NewWith(conn)
		_ = header // Silence unused
		if err != nil {
			log.Printf("[Video] ❌ Failed to read IVF Header: %v. Closing connection.", err)
			conn.Close()
			continue
		}

		log.Printf("[Video] ✅ Stream Header Received! %dx%d",
			header.Width, header.Height)

		for {
			frame, _, err := ivf.ParseNextFrame()
			if err != nil {
				// Connection close or stream restart
				break
			}
			// Send
			sample := media.Sample{Data: frame, Duration: time.Second / 30} // Approx
			if err := track.WriteSample(sample, nil); err != nil {
				log.Printf("[Video] WriteSample Error: %v", err)
				break
			}
		}

		conn.Close()
		log.Println("[Video] Camera Disconnected. Waiting for reconnection...")
	}
}
