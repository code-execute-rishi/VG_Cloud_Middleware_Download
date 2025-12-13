package main

import (
	"context"
	"encoding/json"
	"flag"
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
	IsConfigured bool         `json:"is_configured"`
	IsConnected  bool         `json:"is_connected"` // Cloud/LiveKit Connected
	IsClaimed    bool         `json:"is_claimed"`
	Camera       CameraConfig `json:"camera_config"`
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
	cameraBitrate    = "500k"
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

	fs := http.FileServer(http.Dir("./ui/dist"))
	mux.Handle("/", fs)

	if err := http.ListenAndServe(SetupPort, corsMiddleware(mux)); err != nil {
		log.Fatalf("Web Server failed: %v", err)
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
	// 1. Initialize MAVLink Node
	node, err := gomavlib.NewNode(gomavlib.NodeConf{
		Endpoints:   []gomavlib.EndpointConf{gomavlib.EndpointUDPServer{Address: ":14550"}},
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
		telemLat      int32   = -353632615
		telemLon      int32   = 1491652300
		telemAlt      int32   = 10000 // mm
		telemHdg      uint16  = 9000
		telemBatt     int     = 100
		telemVolt     uint16  = 12000
		telemRoll     float32 = 0
		telemPitch    float32 = 0
		telemYaw      float32 = 0
		telemAirSpeed float32 = 0
		telemGndSpeed float32 = 0
		telemThrottle uint16  = 0
		telemWpDist   uint16  = 0
		telemWpSeq    uint16  = 0
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
			dataMutex.RUnlock()

			update := TelemetryUpdate{
				Latitude: cLat, Longitude: cLon, Altitude: cAlt,
				Speed: 0, Heading: cHdg, Battery: cBatt, SignalStrength: 100,
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
			dataMutex.RUnlock()

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
			// Check Claim Status
			deviceStatusMutex.RLock()
			claimed := deviceStatus.IsClaimed
			deviceStatusMutex.RUnlock()

			if !claimed {
				isClaimed, err := apiClient.CheckClaim()
				if err == nil && isClaimed {
					log.Println(">>> DEVICE CLAIMED BY USER! <<<")
					setStatus(true, true, true)
				}
			}
		}
	}
}

// --- HTTP HANDLERS ---

func handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"pairing_code": apiClient.Identity.PairingCode,
		"device_id":    apiClient.Identity.NodeID,
		"version":      "v2.0.1-patched",
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
		http.Error(w, "Method not allowed", 405)
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
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		Resolution string `json:"resolution"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}

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
		// Small delay to ensure previous ffmpeg is fully dead and device valid
		time.Sleep(1 * time.Second)

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
			log.Printf("Failed to open pipe: %v", err)
			return
		}
		defer file.Close()

		var width, height uint32
		if n, _ := fmt.Sscanf(cameraResolution, "%dx%d", &width, &height); n != 2 {
			log.Printf("Invalid Resolution format '%s', defaulting to 640x480", cameraResolution)
			width, height = 640, 480
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
