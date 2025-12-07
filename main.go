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

// --- Configuration ---
const (
	MAVLinkAddress = "udp://:14550"
	PollInterval   = 3 * time.Second
)

var AuthAPIURL = "http://localhost:8080/api/v1/devices/auth"

func init() {
	if url := os.Getenv("BACKEND_URL"); url != "" {
		AuthAPIURL = url + "/api/v1/devices/auth"
	}
}

// --- Data Structures ---

// TelemetryPayload matches the JSON structure required by the frontend
type TelemetryPayload struct {
	Timestamp         int64           `json:"timestamp"`
	Attitude          *Attitude       `json:"attitude,omitempty"`
	SysStatus         *SysStatus      `json:"sys_status,omitempty"`
	GlobalPositionInt *GlobalPosition `json:"global_position_int,omitempty"`
	Mode              string          `json:"mode,omitempty"` // Added Mode
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
)

// --- Camera Manager ---

// startCamera starts the ffmpeg process and publishes to LiveKit.
// It returns a context that is cancelled when the camera is stopped.
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
		// Note: We avoid 'pkill' now and rely on Context cancellation to kill the specific process.
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
				// Only log if not cancelled
				if ctx.Err() == nil {
					log.Printf("ffmpeg exited unexpectedly: %v\nStderr: %s", err, stderr.String())
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
						return // End of stream
					}
					// If pipe is broken
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

// --- HTTP Handlers ---

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

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(resp)
}

func handleConfig(w http.ResponseWriter, r *http.Request, room *lksdk.Room) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Resolution string `json:"resolution"`
		Bitrate    string `json:"bitrate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	// Update Global Config
	cameraMutex.Lock()
	if req.Resolution != "" {
		cameraResolution = req.Resolution
	}
	if req.Bitrate != "" {
		cameraBitrate = req.Bitrate
	}
	cameraMutex.Unlock()

	log.Printf("Config updated: %s %s", cameraResolution, cameraBitrate)

	// Restart Camera
	if room != nil {
		startCamera(room)
	}

	w.WriteHeader(http.StatusOK)
}

// --- Main ---

// Global Room Reference for HTTP Handlers
var GlobalRoom *lksdk.Room

func main() {
	log.Println("Booting Drone Device Middleware...")

	// 1. Generate Ed25519 Keypair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate keys: %v", err)
	}

	// 2. Derive Pairing Code (INT64 now)
	pubKeyHex := hex.EncodeToString(pubKey)
	var codeVal uint64
	fmt.Sscanf(pubKeyHex[:8], "%x", &codeVal)
	// Ensure strict 8 digits (modulo 100,000,000)
	// Actually, to guarantee 8 digits (10000000-99999999) we might want a range.
	// But "8 digits" usually means max 8 digits.
	// If the user wants EXACTLY 8 digits, we should ensure it's > 10000000.
	// But let's just do modulo 100000000 for now (0-99999999).
	// To be safe and look cool: 10000000 + (val % 90000000) -> 10000000 to 99999999
	pairingCode = int64(10000000 + (codeVal % 90000000))

	fmt.Printf("\n=== PAIRING CODE: %d ===\n\n", pairingCode)

	// Start Local Dashboard Server (Task 2)
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/status", handleStatus)
		mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
			handleConfig(w, r, GlobalRoom)
		})
		fs := http.FileServer(http.Dir("./ui/dist"))
		mux.Handle("/", fs)
		log.Println("Local Dashboard listening on :8081")
		http.ListenAndServe(":8081", mux)
	}()

	// 2.5 Announce Device to Backend
	client := &http.Client{Timeout: 5 * time.Second}
	announceBody, _ := json.Marshal(map[string]interface{}{
		"pairing_code": pairingCode,
		"public_key":   pubKeyHex,
	})

	baseURL := "http://localhost:8080"
	if url := os.Getenv("BACKEND_URL"); url != "" {
		baseURL = url
	}
	AuthAPIURL = baseURL + "/api/v1/devices/auth"
	announceURL := baseURL + "/api/v1/devices/announce"

	go func() {
		for {
			resp, err := client.Post(announceURL, "application/json", bytes.NewBuffer(announceBody))
			if err == nil && resp.StatusCode == http.StatusOK {
				log.Println("Device announced.")
				resp.Body.Close()
				break
			}
			if err != nil {
				log.Printf("Announce failed: %v", err)
			} else {
				log.Printf("Announce failed: Status %d", resp.StatusCode)
			}
			time.Sleep(3 * time.Second)
		}
	}()

	// 3. Authentication Loop
	var token string
	var lkURL string

	for {
		// Poll for challenge
		// Note: Backend expects String ID. We send integer as string.
		resp, err := client.Post(AuthAPIURL+"/challenge", "application/json", bytes.NewBuffer([]byte(fmt.Sprintf(`{"device_id": "%d"}`, pairingCode))))
		if err != nil {
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

	// 4. Connect to LiveKit
	log.Printf("Connecting to LiveKit: %s", lkURL)
	room, err := lksdk.ConnectToRoomWithToken(lkURL, token, lksdk.NewRoomCallback())
	if err != nil {
		log.Fatalf("Failed to connect to LiveKit: %v", err)
	}

	GlobalRoom = room
	log.Printf("Connected to LiveKit Room: %s", room.Name())

	// Start Camera with defaults
	statusMutex.Lock()
	isConnected = true
	statusMutex.Unlock()

	startCamera(room)

	// 5. MAVLink Bridge
	runMavlink(room)
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
		log.Fatalf("Failed to create MAVLink node: %v", err)
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
				// Simplified Mode Logic
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
