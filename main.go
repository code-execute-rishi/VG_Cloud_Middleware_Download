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

// --- Main ---

func main() {
	log.Println("Booting Drone Device Middleware...")

	// 1. Generate Ed25519 Keypair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate keys: %v", err)
	}

	// 2. Derive Pairing Code
	pubKeyHex := hex.EncodeToString(pubKey)
	pairingCode := pubKeyHex[:8]
	fmt.Printf("\n=== PAIRING CODE: %s ===\n\n", pairingCode)
	log.Printf("Public Key: %s", pubKeyHex)

	// 2.5 Announce Device to Backend (New Flow)
	client := &http.Client{Timeout: 5 * time.Second}
	announceBody, _ := json.Marshal(map[string]string{
		"pairing_code": pairingCode,
		"public_key":   pubKeyHex,
	})

	// Retry loop for announcement
	// Announce Device (Synchronous First Attempt)
	baseURL := "http://localhost:8080"
	if url := os.Getenv("BACKEND_URL"); url != "" {
		baseURL = url
	}
	announceURL := baseURL + "/api/v1/devices/announce"
	log.Println("Announcing device to backend...")
	resp, err := client.Post(announceURL, "application/json", bytes.NewBuffer(announceBody))
	if err == nil && resp.StatusCode == http.StatusOK {
		log.Println("Device successfully announced to Backend.")
		resp.Body.Close()
	} else {
		if err != nil {
			log.Printf("Failed to announce device: %v", err)
		} else {
			log.Printf("Failed to announce device: Status %d", resp.StatusCode)
		}
		// Start background retry if failed
		go func() {
			for {
				time.Sleep(2 * time.Second)
				resp, err := client.Post(announceURL, "application/json", bytes.NewBuffer(announceBody))
				if err == nil && resp.StatusCode == http.StatusOK {
					log.Println("Device successfully announced to Backend (Retry).")
					resp.Body.Close()
					break
				}
			}
		}()
	}

	// 3. Authentication Loop
	var token string
	var lkURL string

	for {
		// Poll for challenge
		resp, err := client.Post(AuthAPIURL+"/challenge", "application/json", bytes.NewBuffer([]byte(fmt.Sprintf(`{"device_id": "%s"}`, pairingCode))))
		if err != nil {
			log.Printf("Failed to contact auth server: %v. Retrying...", err)
			time.Sleep(PollInterval)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			log.Println("Device not claimed yet. Waiting...")
			time.Sleep(PollInterval)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("Unexpected status from auth server: %d", resp.StatusCode)
			time.Sleep(PollInterval)
			continue
		}

		var challengeResp ChallengeResponse
		if err := json.NewDecoder(resp.Body).Decode(&challengeResp); err != nil {
			log.Printf("Failed to decode challenge: %v", err)
			time.Sleep(PollInterval)
			continue
		}

		// Sign the challenge
		signature := ed25519.Sign(privKey, []byte(challengeResp.Challenge))
		sigHex := hex.EncodeToString(signature)

		// Verify signature
		verifyBody, _ := json.Marshal(VerifyRequest{
			DeviceID:  pairingCode,
			Signature: sigHex,
		})
		vResp, err := client.Post(AuthAPIURL+"/verify", "application/json", bytes.NewBuffer(verifyBody))
		if err != nil {
			log.Printf("Failed to verify signature: %v", err)
			time.Sleep(PollInterval)
			continue
		}
		defer vResp.Body.Close()

		if vResp.StatusCode == http.StatusOK {
			var verifyResp VerifyResponse
			if err := json.NewDecoder(vResp.Body).Decode(&verifyResp); err != nil {
				log.Printf("Failed to decode verify response: %v", err)
				time.Sleep(PollInterval)
				continue
			}
			token = verifyResp.Token
			lkURL = verifyResp.URL
			log.Println("Authentication Successful! Connecting to LiveKit...")
			break
		} else {
			log.Printf("Verification failed: %d", vResp.StatusCode)
			time.Sleep(PollInterval)
		}
	}

	// 4. Connect to LiveKit
	room, err := lksdk.ConnectToRoomWithToken(lkURL, token, lksdk.NewRoomCallback())
	if err != nil {
		log.Fatalf("Failed to connect to LiveKit: %v", err)
	}
	defer room.Disconnect()

	log.Println("Connected to LiveKit Room:", room.Name())

	// 4.5 Publish Live Webcam
	go func() {
		log.Println("Setting up live webcam feed...")

		// Aggressive cleanup: Kill any existing ffmpeg instances
		exec.Command("pkill", "-x", "ffmpeg").Run()
		time.Sleep(500 * time.Millisecond)

		pipePath := "camera_pipe.ivf"
		os.Remove(pipePath) // Clean up old pipe
		if err := syscall.Mkfifo(pipePath, 0666); err != nil {
			log.Printf("Failed to create named pipe: %v", err)
			return
		}

		// Context for cleanup
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle OS signals (This needs to be handled carefully in a goroutine,
		// but since main handles signals and exits, this context will be cancelled when main exits?
		// No, main's context is not passed here.
		// Actually, we should use a global context or just rely on process exit.
		// For simplicity, let's just run ffmpeg.

		// Start ffmpeg in background with context
		go func() {
			log.Println("Starting ffmpeg...")
			cmd := exec.CommandContext(ctx, "ffmpeg",
				"-f", "v4l2",
				"-video_size", "640x480",
				"-i", "/dev/video0",
				"-c:v", "libvpx",
				"-b:v", "500k",
				"-minrate", "500k",
				"-maxrate", "500k",
				"-r", "30",
				"-bufsize", "50k", // Extremely small buffer
				"-cpu-used", "8", // Max speed for realtime
				"-g", "10", // Keyframe every ~0.3s
				"-qmin", "4",
				"-qmax", "40",
				"-static-thresh", "0", // CRITICAL: Disable static frame skipping
				"-lag-in-frames", "0",
				"-error-resilient", "1",
				"-auto-alt-ref", "0",
				"-deadline", "realtime",
				"-f", "ivf",
				"-y", pipePath,
			)
			// Capture stderr for debugging
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				// Don't log error if killed by context
				if ctx.Err() == nil {
					log.Printf("ffmpeg exited with error: %v\nStderr: %s", err, stderr.String())
				}
			}
		}()

		// Give ffmpeg a moment to start writing
		time.Sleep(2 * time.Second)

		// Open the named pipe
		file, err := os.OpenFile(pipePath, os.O_RDONLY, os.ModeNamedPipe)
		if err != nil {
			log.Printf("Failed to open pipe: %v", err)
			return
		}
		defer file.Close()

		// Create Sample Track
		track, err := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeVP8,
			ClockRate: 90000,
		})
		if err != nil {
			log.Printf("Failed to create sample track: %v", err)
			return
		}

		if _, err := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{
			Name:        "camera_feed",
			VideoWidth:  640,
			VideoHeight: 480,
		}); err != nil {
			log.Printf("Failed to publish video track: %v", err)
			return
		}

		log.Println("Live webcam track published successfully! Starting stream loop...")

		// Create IVF Reader
		ivf, header, err := ivfreader.NewWith(file)
		if err != nil {
			log.Printf("Failed to create IVF reader: %v", err)
			return
		}
		log.Printf("IVF Header: %dx%d, Timebase: %d/%d, Frames: %d",
			header.Width, header.Height, header.TimebaseNumerator, header.TimebaseDenominator, header.NumFrames)

		// Frame Loop
		for {
			payload, _, err := ivf.ParseNextFrame()
			if err != nil {
				if err == io.EOF {
					log.Println("End of video stream")
					break
				}
				log.Printf("Failed to parse frame: %v", err)
				break
			}

			// Write to LiveKit
			if err := track.WriteSample(media.Sample{
				Data:     payload,
				Duration: 33 * time.Millisecond, // ~30fps
			}, nil); err != nil {
				log.Printf("Failed to write sample: %v", err)
			}
		}
	}()

	// 5. MAVLink Bridge
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
		ticker := time.NewTicker(100 * time.Millisecond) // 10Hz update rate
		defer ticker.Stop()

		// Simulation state
		simLat := -35.363262
		simLon := 149.165237
		simAlt := 10.0
		simHeading := 0.0

		for range ticker.C {
			// Check for simulation mode
			if time.Since(lastMavlinkMsg) > 5*time.Second {
				// Simulate Data
				if currentMode != "SIMULATION" {
					log.Println("Switching to SIMULATION mode...")
				}
				simHeading += 1.0
				if simHeading >= 360 {
					simHeading = 0
				}
				simLat += 0.00001 * 1 // Move slightly

				currentMode = "SIMULATION"
				currentArmed = true
				currentAttitude = &Attitude{Roll: 0, Pitch: 0, Yaw: float32(simHeading * 3.14 / 180)}
				currentSysStatus = &SysStatus{Voltage: 12.5, BatteryRemaining: 85}
				currentGlobalPos = &GlobalPosition{
					Lat: simLat,
					Lon: simLon,
					Alt: float32(simAlt),
					Hdg: uint16(simHeading * 100),
					Vx:  100, Vy: 0, Vz: 0,
				}
				currentGpsRaw = &GpsRaw{FixType: 3, Satellites: 12}
			}

			// Send if we have at least some data
			if currentAttitude == nil && currentSysStatus == nil && currentGlobalPos == nil && currentMode == "UNKNOWN" {
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

			msg := TelemetryMessage{
				Type:    "telemetry",
				Payload: payload,
			}

			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to marshal telemetry: %v", err)
				continue
			}

			if err := room.LocalParticipant.PublishData(data, lksdk.WithDataPublishReliable(false)); err != nil {
				log.Printf("Failed to publish data: %v", err)
			}
		}
	}()

	// MAVLink Event Loop
	for evt := range node.Events() {
		if frm, ok := evt.(*gomavlib.EventFrame); ok {
			lastMavlinkMsg = time.Now() // Reset simulation timer
			switch msg := frm.Message().(type) {
			case *common.MessageHeartbeat:
				// Simple mode mapping for ArduPilot/PX4
				// This is a simplification. Real mapping depends on Autopilot type.
				// Assuming ArduCopter for now.
				switch msg.CustomMode {
				case 0:
					currentMode = "STABILIZE"
				case 3:
					currentMode = "AUTO"
				case 4:
					currentMode = "GUIDED"
				case 5:
					currentMode = "LOITER"
				case 6:
					currentMode = "RTL"
				case 9:
					currentMode = "LAND"
				default:
					currentMode = fmt.Sprintf("MODE(%d)", msg.CustomMode)
				}

				// Armed Status (MAV_MODE_FLAG_SAFETY_ARMED = 128)
				currentArmed = (msg.BaseMode & common.MAV_MODE_FLAG_SAFETY_ARMED) != 0

			case *common.MessageAttitude:
				currentAttitude = &Attitude{
					Roll:  msg.Roll,
					Pitch: msg.Pitch,
					Yaw:   msg.Yaw,
				}
			case *common.MessageSysStatus:
				currentSysStatus = &SysStatus{
					Voltage:          float32(msg.VoltageBattery) / 1000.0, // mV to V
					BatteryRemaining: int(msg.BatteryRemaining),
				}
			case *common.MessageGlobalPositionInt:
				currentGlobalPos = &GlobalPosition{
					Lat: float64(msg.Lat) / 1e7,
					Lon: float64(msg.Lon) / 1e7,
					Alt: float32(msg.RelativeAlt) / 1000.0, // mm to m
					Hdg: msg.Hdg / 100,                     // cdeg to deg
					Vx:  msg.Vx,
					Vy:  msg.Vy,
					Vz:  msg.Vz,
				}
			case *common.MessageGpsRawInt:
				currentGpsRaw = &GpsRaw{
					FixType:    uint8(msg.FixType),
					Satellites: msg.SatellitesVisible,
				}
			}
		}
	}
}
