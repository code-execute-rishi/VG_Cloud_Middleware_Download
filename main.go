package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
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
	BackendBaseURL = DefaultBaseURL
)

func init() {
	if url := os.Getenv("BACKEND_URL"); url != "" {
		BackendBaseURL = url
	}
}

// --- Data Structures ---

type ConfigFile struct {
	SSID       string `json:"ssid"`
	Password   string `json:"password"`
	Resolution string `json:"resolution"`
}

type GlobalDeviceStatus struct {
	IsConfigured bool           `json:"is_configured"`
	IsConnected  bool           `json:"is_connected"` // Cloud/LiveKit Connected
	IsClaimed    bool           `json:"is_claimed"`
	Camera       CameraConfig   `json:"camera_config"`
	Hardware     HardwareStatus `json:"hardware_status"`
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
	cameraMutex      sync.Mutex
	cameraCancel     context.CancelFunc

	// Thread-safe Status
	deviceStatus      GlobalDeviceStatus
	deviceStatusMutex sync.RWMutex

	// Global Room Access for Restarts
	activeRoom      *lksdk.Room
	activeRoomMutex sync.Mutex

	apiClient *BackendClient
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
	mux.HandleFunc("/api/wifi-scan", handleWifiScan)
	mux.HandleFunc("/api/save-config", handleSaveConfig)
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

// --- STATE LOOP ---

func runStateLoop() {
	for {
		// Load Config
		if _, err := os.Stat(ConfigFileName); os.IsNotExist(err) {
			setStatus(false, false, false)
			log.Println("[Loop] No Config. Waiting in Setup Mode...")
			time.Sleep(2 * time.Second)
			continue
		}

		// Config Exists -> Mission Mode
		data, _ := os.ReadFile(ConfigFileName)
		var config ConfigFile
		json.Unmarshal(data, &config)
		if config.Resolution != "" {
			cameraResolution = config.Resolution
		}

		// Update Status (Configured, but not connected yet)
		setStatus(true, false, false)

		log.Println("[Loop] Config Found. Starting Mission Services...")

		// 1. Ensure Registered
		if err := apiClient.Register(); err != nil {
			log.Printf("[Loop] Registration Retry Failed: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// 2. Authenticate
		log.Println("[Loop] Authenticating...")
		authRes, err := apiClient.Authenticate()
		if err != nil {
			if strings.Contains(err.Error(), "DEVICE_FORGOTTEN") {
				log.Println("⚠️ DEVICE FORGOTTEN BY CLOUD. RESETTING IDENTITY... ⚠️")
				if resetErr := apiClient.ResetIdentity(); resetErr != nil {
					log.Printf("[Loop] Reset Identity Failed: %v", resetErr)
				}
				// Clear Status to force Setup UI
				setStatus(false, false, false)
				time.Sleep(2 * time.Second)
				continue
			}
			log.Printf("[Loop] Auth Failed: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// 2.5 Check Claim Status IMMEDIATELY
		isClaimed, _ := apiClient.CheckClaim()
		if isClaimed {
			log.Println("[Loop] Device is ALREADY Claimed.")
		}

		// 3. Connect LiveKit
		// Setup Context for Disconnect detection
		ctx, cancel := context.WithCancel(context.Background())

		cb := lksdk.NewRoomCallback()
		cb.OnDisconnected = func() {
			log.Println("[Callback] Room Disconnected. Cancelling loop context.")
			cancel()
		}

		log.Printf("[Loop] Connecting to Room: %s", authRes.RoomName)
		room, err := lksdk.ConnectToRoomWithToken(authRes.LiveKitURL, authRes.LiveKitToken, cb)
		if err != nil {
			log.Printf("[Loop] LiveKit Connect Failed: %v", err)
			cancel() // Clean up context
			time.Sleep(5 * time.Second)
			continue
		}

		// Update Global Active Room
		activeRoomMutex.Lock()
		activeRoom = room
		activeRoomMutex.Unlock()

		log.Println("[Loop] Connected! Joining Mission.")
		setStatus(true, true, isClaimed) // Updated with actual claim status

		// 4. Start Subsystems
		startCamera(room)

		// 5. Telemetry & Claim Check Loop
		// We block here until ctx is done (disconnect)
		telemetryAndClaimLoop(ctx, room)

		// Cleanup
		cancel() // Ensure context is closed

		activeRoomMutex.Lock()
		activeRoom = nil
		activeRoomMutex.Unlock()

		room.Disconnect()           // Ensure disconnected
		time.Sleep(4 * time.Second) // Increased delay to ensure clean disconnect

		log.Println("[Loop] Connection Lost/Restarting. Restarting loop...")
		time.Sleep(2 * time.Second)
	}
}

func telemetryAndClaimLoop(ctx context.Context, room *lksdk.Room) {
	// 1. Initialize MAVLink Node (Dual-Stack: UDP + Serial)
	endpoints := []gomavlib.EndpointConf{
		gomavlib.EndpointUDPServer{Address: ":14550"}, // Always support Simulator
	}

	// Scan for Real Hardware
	if serialPort := GetSerialPorts(); serialPort != "" {
		log.Printf("[MAVLink] 🔌 Auto-Detected Serial Port: %s. Binding...", serialPort)
		endpoints = append(endpoints, gomavlib.EndpointSerial{Device: serialPort, Baud: 57600})
	} else {
		log.Println("[MAVLink] ⚠️ No Serial Port Found. Running in Simulator Mode (UDP Only).")
	}

	node, err := gomavlib.NewNode(gomavlib.NodeConf{
		Endpoints:   endpoints,
		Dialect:     common.Dialect,
		OutVersion:  gomavlib.V2,
		OutSystemID: 10,
	})

	if err != nil {
		log.Printf("MAVLink Bind Failed: %v", err)
		return
	}
	defer node.Close()

	// 2. Shared State for MAVLink Data
	var (
		dataMutex sync.RWMutex
		// Default values to avoid nil pointers or zeros
		telemLat           int32   = -353632615
		telemLon           int32   = 1491652300
		telemAlt           int32   = 10000 // mm
		telemHdg           uint16  = 9000
		telemBatt          int     = 100
		telemVolt          uint16  = 12000
		telemRoll          float32 = 0
		telemPitch         float32 = 0
		telemYaw           float32 = 0
		telemAirSpeed      float32 = 0
		telemGndSpeed      float32 = 0
		telemThrottle      uint16  = 0
		telemWpDist        uint16  = 0
		telemWpSeq         uint16  = 0
		telemLastHeartbeat int64   = 0
		telemArmed         bool    = false
		telemFlightMode    string  = "Unknown"
	)

	// 3. Start MAVLink Listener Routine
	go func() {
		log.Println("MAVLink Listener Started on :14550")
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-node.Events():
				if frm, ok := evt.(*gomavlib.EventFrame); ok {
					dataMutex.Lock()
					switch msg := frm.Message().(type) {
					case *common.MessageGlobalPositionInt:
						telemLat = msg.Lat
						telemLon = msg.Lon
						telemAlt = msg.Alt // mm
						telemHdg = msg.Hdg
					case *common.MessageAttitude:
						telemRoll = msg.Roll
						telemPitch = msg.Pitch
						telemYaw = msg.Yaw
					case *common.MessageSysStatus:
						telemBatt = int(msg.BatteryRemaining)
						telemVolt = msg.VoltageBattery
					case *common.MessageVfrHud:
						telemAirSpeed = msg.Airspeed
						telemGndSpeed = msg.Groundspeed
						telemThrottle = msg.Throttle
					case *common.MessageNavControllerOutput:
						telemWpDist = msg.WpDist
					case *common.MessageMissionCurrent:
						telemWpSeq = msg.Seq
					case *common.MessageHeartbeat:
						telemLastHeartbeat = time.Now().Unix()
						// Check ARMED state (bit 7 of base_mode)
						// MAV_MODE_FLAG_SAFETY_ARMED = 128
						telemArmed = (msg.BaseMode & 128) != 0

						// Simple Flight Mode Mapping (ArduCopter/PX4 approximate)
						// This is a naive mapping. Real implementation requires dialect specific decoding.
						// For now, we return "Ready" if disarmed, "Flying" if armed, etc.
						if telemArmed {
							telemFlightMode = "Airborne"
						} else {
							telemFlightMode = "Standby"
						}
					}
					dataMutex.Unlock()
				}
			}
		}
	}()

	// 4. Tickers
	telemTicker := time.NewTicker(10 * time.Second)
	defer telemTicker.Stop()

	fastTicker := time.NewTicker(200 * time.Millisecond)
	defer fastTicker.Stop()

	claimTicker := time.NewTicker(10 * time.Second)
	defer claimTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Loop] Context Done (Disconnected). Exiting Telemetry Loop.")
			return
		case <-telemTicker.C:
			// --- CLOUD SYNC (Low Freq) ---
			dataMutex.RLock()
			cLat := float64(telemLat) / 1e7
			cLon := float64(telemLon) / 1e7
			cAlt := float32(telemAlt) / 1000.0
			cBatt := telemBatt
			cHdg := float32(telemHdg) / 100.0
			cArmed := telemArmed
			cMode := telemFlightMode
			lastHb := telemLastHeartbeat
			dataMutex.RUnlock()

			// Check for stale heartbeat (timeout 5s)
			if time.Now().Unix()-lastHb > 5 {
				cArmed = false
				cMode = "Offline"
				// Optional: Reset other telemetry? Keep last known location for now.
			}

			update := TelemetryUpdate{
				Latitude: cLat, Longitude: cLon, Altitude: cAlt,
				Speed: 0, Heading: cHdg, Battery: cBatt, SignalStrength: 100,
				Armed: cArmed, FlightMode: cMode,
			}
			apiClient.UpdateTelemetry(update)

		case <-fastTicker.C:
			// --- LIVEKIT HUD (High Freq) ---
			dataMutex.RLock()
			curRoll, curPitch, curYaw := telemRoll, telemPitch, telemYaw
			curLat, curLon, curAlt, curHdg := telemLat, telemLon, telemAlt, telemHdg
			curAirSpd, curGndSpd := telemAirSpeed, telemGndSpeed
			curBatt, curVolt := telemBatt, telemVolt
			curThrottle := telemThrottle
			curWpDist, curWpSeq := telemWpDist, telemWpSeq
			curHeartbeat := telemLastHeartbeat
			dataMutex.RUnlock()

			// Hardware Check
			fcConnected := (time.Now().Unix() - curHeartbeat) < 5
			_, camErr := os.Stat("/dev/video0")
			camConnected := (camErr == nil)

			// 1. ATTITUDE
			attPayload := map[string]interface{}{
				"type": "ATTITUDE",
				"data": map[string]interface{}{
					"roll":  curRoll,
					"pitch": curPitch,
					"yaw":   curYaw,
				},
			}
			dataAtt, _ := json.Marshal(attPayload)
			room.LocalParticipant.PublishData(dataAtt, lksdk.WithDataPublishReliable(false), lksdk.WithDataPublishTopic("telemetry"))

			// 2. GLOBAL_POSITION_INT
			posPayload := map[string]interface{}{
				"type": "GLOBAL_POSITION_INT",
				"data": map[string]interface{}{
					"lat":          curLat,
					"lon":          curLon,
					"alt":          curAlt,
					"relative_alt": curAlt,
					"hdg":          curHdg,
					"vx":           0, "vy": 0, "vz": 0,
				},
			}
			dataPos, _ := json.Marshal(posPayload)
			room.LocalParticipant.PublishData(dataPos, lksdk.WithDataPublishReliable(false), lksdk.WithDataPublishTopic("telemetry"))

			// 3. VFR_HUD
			hudPayload := map[string]interface{}{
				"type": "VFR_HUD",
				"data": map[string]interface{}{
					"heading":     float32(curHdg) / 100.0,
					"airspeed":    curAirSpd,
					"groundspeed": curGndSpd,
					"alt":         float32(curAlt) / 1000.0,
					"throttle":    curThrottle,
				},
			}
			dataHud, _ := json.Marshal(hudPayload)
			room.LocalParticipant.PublishData(dataHud, lksdk.WithDataPublishReliable(false), lksdk.WithDataPublishTopic("telemetry"))

			// 4. SYS_STATUS
			sysPayload := map[string]interface{}{
				"type": "SYS_STATUS",
				"data": map[string]interface{}{
					"voltage_battery":   curVolt,
					"battery_remaining": curBatt,
					"fc_connected":      fcConnected,
					"cam_connected":     camConnected,
				},
			}
			dataSys, _ := json.Marshal(sysPayload)
			room.LocalParticipant.PublishData(dataSys, lksdk.WithDataPublishReliable(false), lksdk.WithDataPublishTopic("telemetry"))

			// 5. NAV_CONTROLLER_OUTPUT
			navPayload := map[string]interface{}{
				"type": "NAV_CONTROLLER_OUTPUT",
				"data": map[string]interface{}{
					"wp_dist": curWpDist,
				},
			}
			dataNav, _ := json.Marshal(navPayload)
			room.LocalParticipant.PublishData(dataNav, lksdk.WithDataPublishReliable(false), lksdk.WithDataPublishTopic("telemetry"))

			// 6. MISSION_CURRENT
			misPayload := map[string]interface{}{
				"type": "MISSION_CURRENT",
				"data": map[string]interface{}{
					"seq": curWpSeq,
				},
			}
			dataMis, _ := json.Marshal(misPayload)
			if err := room.LocalParticipant.PublishData(dataMis, lksdk.WithDataPublishReliable(false), lksdk.WithDataPublishTopic("telemetry")); err != nil {
				log.Printf("Failed to publish MISSION_CURRENT: %v", err)
			}

		case <-claimTicker.C:
			// Check Claim Status Periodically
			isServerClaimed, err := apiClient.CheckClaim()
			if err != nil {
				if strings.Contains(err.Error(), "DEVICE_FORGOTTEN") {
					log.Println(">>> DEVICE FORGOTTEN (Hard Reset) <<<")
					log.Println("Wiping Identity and Restarting...")
					apiClient.ResetIdentity()
					os.Exit(0) // Supervisor will restart with new identity
				}

				log.Printf("[Loop] CheckClaim Error: %v", err)
				continue
			}

			deviceStatusMutex.Lock() // Write Lock
			wasClaimed := deviceStatus.IsClaimed

			// Update Status
			deviceStatus.IsClaimed = isServerClaimed

			// State Transitions
			if !wasClaimed && isServerClaimed {
				log.Println(">>> DEVICE CLAIMED BY USER! <<<")
				// Force status update (redundant but safe)
				deviceStatus.IsConfigured = true
				deviceStatus.IsConnected = true
			} else if wasClaimed && !isServerClaimed {
				log.Println(">>> DEVICE UNBOUND (Soft Release) <<<")
				// Device released by user. We stay connected but show Bind Screen.
			}

			// Update Hardware Status for Local UI
			dataMutex.RLock()
			lastHb := telemLastHeartbeat
			dataMutex.RUnlock()

			_, vidErr := os.Stat("/dev/video0")

			deviceStatus.Hardware = HardwareStatus{
				FCConnected:  (time.Now().Unix() - lastHb) < 5,
				CamConnected: (vidErr == nil),
			}

			// Disambiguate: "Unclaimed" vs "Deleted"
			// If CheckClaim returns false, it could be valid-unclaimed OR deleted.
			// We must check Liveness to confirm existence.
			if !isServerClaimed {
				if err := apiClient.CheckLiveness(); err != nil {
					if strings.Contains(err.Error(), "DEVICE_FORGOTTEN") {
						log.Println(">>> DEVICE FORGOTTEN (Hard Reset Confirmed) <<<")
						apiClient.ResetIdentity()
						os.Exit(0)
					}
				}
			}
			deviceStatusMutex.Unlock()
		}
	}
}

// --- HTTP HANDLERS ---

func handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"pairing_code": apiClient.Identity.PairingCode,
		"device_id":    apiClient.Identity.NodeID,
		"version":      "v2.3.0",
	}
	jsonResponse(w, resp)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	deviceStatusMutex.Lock() // Fixed: Using Lock instead of RLock because we modify the struct
	defer deviceStatusMutex.Unlock()

	// Update camera config in status live
	deviceStatus.Camera = CameraConfig{Resolution: cameraResolution}
	jsonResponse(w, deviceStatus)
}

func handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config ConfigFile
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}

	log.Println("Registering Device via Web UI...")
	if err := apiClient.Register(); err != nil {
		http.Error(w, fmt.Sprintf("Register Failed: %v", err), 500)
		return
	}

	file, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(ConfigFileName, file, 0644)

	jsonResponse(w, map[string]string{"status": "success", "message": "Saved. Connecting..."})
}

func handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Resolution string `json:"resolution"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	log.Printf(">>> UI REQUESTED CONFIG UPDATE: Resolution=%s", req.Resolution)

	// Update Runtime
	if req.Resolution != "" {
		cameraResolution = req.Resolution

		// Hot-Reload Camera (No Disconnect)
		activeRoomMutex.Lock()
		room := activeRoom
		activeRoomMutex.Unlock()

		if room != nil {
			log.Println("Config Changed. Restarting Camera Stream Only...")
			startCamera(room)
		}
	}

	// Update File
	data, _ := os.ReadFile(ConfigFileName)
	var config ConfigFile
	json.Unmarshal(data, &config)
	config.Resolution = req.Resolution
	file, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(ConfigFileName, file, 0644)

	jsonResponse(w, map[string]string{"status": "updated"})
}

func handleWifiScan(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, []interface{}{})
}

// --- HELPERS ---

func setStatus(conf, conn, claimed bool) {
	deviceStatusMutex.Lock()
	defer deviceStatusMutex.Unlock()
	deviceStatus.IsConfigured = conf
	deviceStatus.IsConnected = conn
	deviceStatus.IsClaimed = claimed
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
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func startCamera(room *lksdk.Room) {
	if room == nil {
		return
	}

	cameraMutex.Lock()
	defer cameraMutex.Unlock()
	if cameraCancel != nil {
		cameraCancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	cameraCancel = cancel

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ GStreamer/IVF Panic Recovered: %v", r)
			}
		}()

		// Delay to ensure previous ffmpeg is fully dead and device valid
		time.Sleep(3 * time.Second)

		pipePath := "camera_pipe.ivf"
		os.Remove(pipePath)
		syscall.Mkfifo(pipePath, 0666)

		// Parse Resolution
		var width, height uint32
		if n, _ := fmt.Sscanf(cameraResolution, "%dx%d", &width, &height); n != 2 {
			width, height = 640, 480
		}

		log.Printf("Starting GStreamer Pipeline: %dx%d", width, height)

		// GStreamer Pipeline:
		// 1. v4l2src -> raw video
		// 2. Tee -> Branch A (LiveKit/VP8), Branch B (Local/MJPEG)
		// Branch A: vp8enc -> ivfenc -> pipe
		// Branch B: jpegenc -> multipartmux -> tcpserversink (Port 8081)

		gstCmd := fmt.Sprintf(
			"v4l2src device=/dev/video0 ! image/jpeg,width=%d,height=%d,framerate=30/1 ! jpegdec ! videoconvert ! video/x-raw,format=I420 ! "+
				"tee name=t ! "+
				"queue max-size-buffers=4 leaky=downstream ! vp8enc error-resilient=1 deadline=1 keyframe-max-dist=30 cpu-used=4 ! video/x-vp8 ! queue ! avmux_ivf ! filesink location=%s sync=false async=false "+
				"t. ! queue ! videoscale ! video/x-raw,width=320,height=240 ! jpegenc ! multipartmux boundary=vyomboundary ! tcpserversink host=127.0.0.1 port=8081 sync=false",
			width, height, pipePath,
		)

		log.Println("Starting GStreamer Pipeline:", gstCmd)

		// Note: The constructed string was just for logging/reference. Exec requires args split.
		// Actually, gst-launch-1.0 parsing can be tricky with split args.
		// Let's use 'sh -c' to avoid splitting headaches for complex pipelines.
		cmd := exec.CommandContext(ctx, "sh", "-c", "gst-launch-1.0 -v "+gstCmd)

		// Capture Stderr for debugging
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Start(); err != nil {
			log.Printf("Camera Start Failed: %v | Stderr: %s", err, stderr.String())
			return
		}

		// Wait for process to exit asynchronously to log errors AND trigger fallback
		startTime := time.Now()
		go func() {
			if err := cmd.Wait(); err != nil {
				// IGNORE error if we killed it intentionally (Config Change)
				if ctx.Err() == context.Canceled {
					log.Println("Camera stopped intentionally (Context Cancelled).")
					return
				}

				log.Printf("FFmpeg Exited with Error: %v | Stderr: %s", err, stderr.String())

				// FALBACK PROTECTION: If it crashed quickly (< 10s), it's likely a config error. Revert to Safe Mode.
				if time.Since(startTime) < 10*time.Second {
					log.Println("⚠️ Camera unstable at this resolution. Reverting to 640x480 SAFE MODE...")

					go func() {
						time.Sleep(2 * time.Second) // Wait for cleanup
						log.Println("Applying Safe Resolution 640x480...")
						cameraResolution = "640x480"

						// Update config file
						config := ConfigFile{Resolution: "640x480"}
						file, _ := json.MarshalIndent(config, "", "  ")
						os.WriteFile(ConfigFileName, file, 0644)

						startCamera(room)
					}()
				}
			}
		}()

		time.Sleep(1 * time.Second)
		// Wait for pipe to have data (GStreamer actually started writing)
		waitForPipe := 0
		for waitForPipe < 10 {
			info, err := os.Stat(pipePath)
			if err == nil && info.Size() > 0 {
				break
			}
			time.Sleep(500 * time.Millisecond)
			waitForPipe++
		}

		file, err := os.Open(pipePath)
		if err != nil {
			log.Printf("Failed to open pipe: %v", err)
			return
		}

		// Create IVF Reader
		ivf, _, err := ivfreader.NewWith(file)
		if err != nil {
			log.Printf("Failed to create IVF reader: %v", err)
			file.Close()
			return
		}
		track, _ := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000})

		// Retry Loop for PublishTrack
		var pub *lksdk.LocalTrackPublication
		var pubErr error
		for i := 0; i < 3; i++ {
			pub, pubErr = room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{Name: "camera_feed", VideoWidth: int(width), VideoHeight: int(height)})
			if pubErr == nil {
				break
			}
			log.Printf("PublishTrack failed (attempt %d/3): %v", i+1, pubErr)
			time.Sleep(1 * time.Second)
		}

		if pubErr != nil {
			log.Printf("Camera Publish Failed Final: %v", pubErr)
			return
		}

		if pub != nil {
			defer room.LocalParticipant.UnpublishTrack(pub.SID())
		}

		// Watchdog: If no frames received in first 6 seconds, KILL IT (to trigger fallback)
		firstFrame := make(chan bool, 1)
		go func() {
			select {
			case <-firstFrame:
				return // All good
			case <-time.After(6 * time.Second):
				log.Println("🚨 Camera Watchdog: Stream STALLED (No frames). Killing to trigger fallback...")
				cancel() // This will cause cmd.Wait() to finish with error, triggering fallback
			}
		}()

		// ivf reader already created above.
		frameCnt := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
				payload, _, err := ivf.ParseNextFrame()
				if err != nil {
					return
				}
				if frameCnt == 0 {
					select {
					case firstFrame <- true:
					default:
					}
				}
				frameCnt++
				track.WriteSample(media.Sample{Data: payload, Duration: 33 * time.Millisecond}, nil)
			}
		}
	}()
}

// --- Hardware Helper Functions ---

func GetSerialPorts() string {
	// Priority List for RPi 3 + Pixhawk
	candidates := []string{
		"/dev/ttyACM0", // Pixhawk via USB
		"/dev/ttyUSB0", // Telemetry Radio / FTDI
		"/dev/ttyAMA0", // RPi UART (GPIO)
		"/dev/serial0", // RPi Serial Alias
	}

	for _, port := range candidates {
		if _, err := os.Stat(port); err == nil {
			return port
		}
	}
	return ""
}
