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
	"path/filepath"
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

type GlobalDeviceStatus struct {
	IsConfigured bool           `json:"is_configured"`
	IsConnected  bool           `json:"is_connected"` // Cloud/LiveKit Connected
	IsClaimed    bool           `json:"is_claimed"`
	Camera       CameraConfig   `json:"camera_config"`
	Hardware     HardwareStatus `json:"hardware_status"`
	LiveKit      LiveKitStatus  `json:"livekit_status"`
	ZeroTier     ZeroTierStatus `json:"zerotier_status"`
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
			deviceStatusMutex.Lock()
			deviceStatus.LiveKit.State = StateError
			deviceStatus.LiveKit.LastError = "Registration Failed"
			deviceStatusMutex.Unlock()
			time.Sleep(5 * time.Second)
			continue
		}

		// 2. Authenticate
		log.Println("[Loop] Authenticating...")
		authRes, err := apiClient.Authenticate()
		if err != nil {
			deviceStatusMutex.Lock()
			deviceStatus.LiveKit.State = StateError
			deviceStatus.LiveKit.LastError = "Auth Failed"
			deviceStatusMutex.Unlock()

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
			deviceStatusMutex.Lock()
			deviceStatus.LiveKit.State = StateDisconnected
			deviceStatusMutex.Unlock()
			cancel()
		}

		// Track Participants
		cb.OnParticipantConnected = func(p *lksdk.RemoteParticipant) {
			deviceStatusMutex.Lock()
			deviceStatus.LiveKit.Participants++
			deviceStatusMutex.Unlock()
		}
		cb.OnParticipantDisconnected = func(p *lksdk.RemoteParticipant) {
			deviceStatusMutex.Lock()
			if deviceStatus.LiveKit.Participants > 0 {
				deviceStatus.LiveKit.Participants--
			}
			deviceStatusMutex.Unlock()
		}

		log.Printf("[Loop] Connecting to Room: %s", authRes.RoomName)
		room, err := lksdk.ConnectToRoomWithToken(authRes.LiveKitURL, authRes.LiveKitToken, cb)
		if err != nil {
			log.Printf("[Loop] LiveKit Connect Failed: %v", err)
			deviceStatusMutex.Lock()
			deviceStatus.LiveKit.State = StateError
			deviceStatus.LiveKit.LastError = "Connection Failed"
			deviceStatusMutex.Unlock()
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

		// Update Detailed LiveKit Status
		deviceStatusMutex.Lock()
		deviceStatus.LiveKit.State = StateConnected
		deviceStatus.LiveKit.RoomName = room.Name()
		deviceStatus.LiveKit.Participants = len(room.GetRemoteParticipants())
		deviceStatus.LiveKit.LastError = ""
		deviceStatusMutex.Unlock()

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
	// 1. Shared State for MAVLink Data (Accessible by Tickers)
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

	// 2. Start MAVLink Connection Manager (Auto-Reconnect)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Dynamic Port Scan
				endpoints := []gomavlib.EndpointConf{
					gomavlib.EndpointUDPServer{Address: ":14550"},
				}

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
					log.Printf("MAVLink Bind Failed: %v. Retrying in 5s...", err)
					time.Sleep(5 * time.Second)
					continue
				}

				// Active Session Loop
				log.Println("MAVLink Listener Started on :14550")

				// We use a heartbeat watchdog to kill dead connections
				// If no message for 10s, we assume disconnected and restart to re-scan ports
				lastPacketTime := time.Now()

				// Monitor Routine
				monitorCtx, monitorCancel := context.WithCancel(context.Background())
				go func() {
					ticker := time.NewTicker(2 * time.Second)
					defer ticker.Stop()
					for {
						select {
						case <-monitorCtx.Done():
							return
						case <-ticker.C:
							if time.Since(lastPacketTime) > 10*time.Second {
								log.Println("❌ MAVLink Timeout (No Data). Restarting Connection...")
								node.Close()
								return
							}
						}
					}
				}()

				// Event Loop
				for evt := range node.Events() {
					lastPacketTime = time.Now() // Keep alive

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
							telemArmed = (msg.BaseMode & 128) != 0
							if telemArmed {
								telemFlightMode = "Airborne"
							} else {
								telemFlightMode = "Standby"
							}
						}
						dataMutex.Unlock()
					}
				}

				monitorCancel() // Stop monitor
				// node.Close() // REMOVED: Monitor or Loop Exit implies closed/done. Double-closing causes panic.
				log.Println("MAVLink Session Ended. Restarting...")
				time.Sleep(2 * time.Second)
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

			// --- ZeroTier Status Check ---
			var ztState = StateDisconnected
			var ztIP = ""

			if ifaces, err := net.Interfaces(); err == nil {
				for _, i := range ifaces {
					if strings.HasPrefix(i.Name, "zt") {
						addrs, _ := i.Addrs()
						for _, addr := range addrs {
							if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
								ztIP = ipnet.IP.String()
								ztState = StateConnected
								break
							}
						}
					}
				}
			}

			deviceStatusMutex.Lock()
			deviceStatus.ZeroTier.State = ztState
			deviceStatus.ZeroTier.IPAddress = ztIP
			if ztState == StateDisconnected {
				// Only set error if we expect it to be there but it's not (simplified for now)
				deviceStatus.ZeroTier.LastError = "No Interface"
			} else {
				deviceStatus.ZeroTier.LastError = ""
			}
			deviceStatusMutex.Unlock()

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
		log.Printf("[API ERROR] Register Failed: %v", err)
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
		FCPort     string `json:"fc_port"`
		FCBaud     int    `json:"fc_baud"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	log.Printf(">>> UI REQUESTED CONFIG UPDATE: Resolution=%s, FCPort=%s, FCBaud=%d", req.Resolution, req.FCPort, req.FCBaud)

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
	if req.Resolution != "" {
		config.Resolution = req.Resolution
	}
	if req.FCPort != "" {
		config.FCPort = req.FCPort
	}
	if req.FCBaud != 0 {
		config.FCBaud = req.FCBaud
	}

	file, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(ConfigFileName, file, 0644)

	jsonResponse(w, map[string]string{"status": "updated"})
}

func handleSerialPorts(w http.ResponseWriter, r *http.Request) {
	matches, _ := filepath.Glob("/dev/tty*")
	// Filter for common serial devices
	var ports []string
	for _, m := range matches {
		if strings.HasPrefix(m, "/dev/ttyUSB") || strings.HasPrefix(m, "/dev/ttyACM") || strings.HasPrefix(m, "/dev/ttyAMA") || strings.HasPrefix(m, "/dev/serial") {
			ports = append(ports, m)
		}
	}
	jsonResponse(w, ports)
}

func handleWifiScan(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, []interface{}{})
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("device.log")
	if err != nil {
		http.Error(w, "Failed to read log file", 500)
		return
	}
	if r.URL.Query().Get("download") == "true" {
		w.Header().Set("Content-Disposition", "attachment; filename=device.log")
		w.Header().Set("Content-Type", "application/octet-stream")
	} else {
		w.Header().Set("Content-Type", "text/plain")
	}
	w.Write(data)
}

// --- HELPERS ---

func setStatus(conf, conn, claimed bool) {
	deviceStatusMutex.Lock()
	defer deviceStatusMutex.Unlock()
	deviceStatus.IsConfigured = conf
	deviceStatus.IsConnected = conn
	deviceStatus.IsClaimed = claimed

	// Update Legacy/Fallback State if disconnected
	if !conn {
		deviceStatus.LiveKit.State = StateDisconnected
	}
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
		// 1. v4l2src -> raw video (REQUESTED RES)
		// 2. Tee -> Branch A (LiveKit/VP8), Branch B (Local/MJPEG)

		// Fix: v4l2src fails with memory errors on RPi Bookworm/Bullseye.
		// Fix: libcamerasrc is missing.
		// Fix: verified rpicam-vid exists. Implementing pipe: rpicam-vid (YUV420) -> stdout -> fdsrc (GST).

		// Fix: Raw YUV pipe caused "Green Screen" due to stride/padding (alignment) mismatches between rpicam-vid and fdsrc.
		// Fix: Switching to MJPEG pipe. JPEG is self-describing so stride doesn't matter.
		// Pipeline: rpicam-vid (MJPEG) -> stdout -> fdsrc -> jpegdec -> videoconvert -> I420

		// Dynamic Pipeline Construction
		var sourcePipeline string
		if _, err := exec.LookPath("rpicam-vid"); err == nil {
			log.Println("[Camera] Using Raspberry Pi Camera (rpicam-vid)")
			sourcePipeline = fmt.Sprintf(
				"rpicam-vid --timeout 0 --nopreview --width %d --height %d --framerate 30 --codec mjpeg -o - | "+
					"gst-launch-1.0 fdsrc ! image/jpeg,width=%d,height=%d,framerate=30/1 ! jpegdec ! videoconvert ! video/x-raw,format=I420",
				width, height, width, height,
			)
		} else {
			log.Println("[Camera] rpicam-vid not found. Falling back to V4L2 (USB/Virtual Camera)...")
			sourcePipeline = fmt.Sprintf(
				"gst-launch-1.0 v4l2src device=/dev/video0 ! videoconvert ! video/x-raw,format=I420,width=%d,height=%d,framerate=30/1",
				width, height,
			)
		}

		fullCmd := fmt.Sprintf(
			"%s ! tee name=t ! "+
				"queue max-size-buffers=4 leaky=downstream ! vp8enc error-resilient=1 deadline=1 keyframe-max-dist=30 cpu-used=5 ! \"video/x-vp8\" ! queue ! avmux_ivf ! filesink location=%s sync=false async=false "+
				"t. ! queue max-size-buffers=4 leaky=downstream ! videoscale ! \"video/x-raw,width=320,height=240\" ! jpegenc ! multipartmux boundary=vyomboundary ! tcpserversink host=127.0.0.1 port=8081 sync=false",
			sourcePipeline, pipePath,
		)

		log.Printf("[Camera] Starting Pipeline Step: %s", fullCmd)

		// Use 'sh -c' to execute the pipe
		cmd := exec.Command("sh", "-c", fullCmd)
		cmd.Stdout = os.Stdout // Redirect stdout/err to console for debugging
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			log.Printf("[Camera] ❌ Failed to start pipeline: %v", err)
			return
		}

		log.Printf("[Camera] ✅ Pipeline started with PID %d", cmd.Process.Pid)

		// Wait for command in a goroutine so we don't block
		startTime := time.Now()
		go func() {
			if err := cmd.Wait(); err != nil {
				// IGNORE error if we killed it intentionally (Config Change)
				if ctx.Err() == context.Canceled {
					log.Println("Camera stopped intentionally (Context Cancelled).")
					return
				}

				log.Printf("FFmpeg Exited with Error: %v | Stderr: %s", err, cmd.Stderr.(*os.File).Name()) // Note: Stderr is now os.Stderr, not a buffer

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
		// Wait for pipe to be ready (just a small sleep to let GStreamer process start)
		time.Sleep(200 * time.Millisecond)

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
			log.Printf("[Hardware] Found Serial Port: %s", port)
			return port
		}
	}
	log.Println("[Hardware] No designated serial ports found.")
	return ""
}
