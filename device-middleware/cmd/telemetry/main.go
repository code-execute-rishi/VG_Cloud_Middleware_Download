package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bluenviron/gomavlib/v3"
	"github.com/bluenviron/gomavlib/v3/pkg/dialects/ardupilotmega"
)

// --- Config ---
const (
	API_URL = "http://localhost:8085"
)

// Local Telemetry State
type TelemetryConfig struct {
	FCPort string `json:"fc_port"`
	FCBaud int    `json:"fc_baud"`
}

// --- LiveKit Telemetry Structs ---
type LiveKitDataMessage struct {
	Type string           `json:"type"`
	Data LiveKitTelemetry `json:"data"`
}

type LiveKitTelemetry struct {
	Timestamp         int64           `json:"timestamp"`
	Attitude          *Attitude       `json:"attitude,omitempty"`
	SysStatus         *SysStatus      `json:"sys_status,omitempty"`
	GlobalPositionInt *GlobalPosition `json:"global_position_int,omitempty"`
	Mode              string          `json:"mode,omitempty"`
	Armed             bool            `json:"armed"`
	GpsRawInt         *GpsRaw         `json:"gps_raw_int,omitempty"`
	VfrHud            *VfrHud         `json:"vfr_hud,omitempty"`
	HomePosition      *HomePosition   `json:"home_position,omitempty"`
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
}
type GpsRaw struct {
	FixType    uint8 `json:"fix_type"`
	Satellites uint8 `json:"satellites_visible"`
}
type VfrHud struct {
	Heading     int16   `json:"heading"`
	Alt         float32 `json:"alt"`
	Airspeed    float32 `json:"airspeed"`
	Groundspeed float32 `json:"groundspeed"`
	Throttle    uint16  `json:"throttle"`
	Climb       float32 `json:"climb"`
}
type HomePosition struct {
	Lat int32 `json:"lat"`
	Lon int32 `json:"lon"`
	Alt int32 `json:"alt"`
}

// --- Global State ---
var (
	fwNodes = make(map[string]*gomavlib.Node)
	fwMutex sync.Mutex

	lastTelem  LiveKitTelemetry
	telemMutex sync.Mutex
)

func main() {
	log.Println("🚁 Starting Vyom Telemetry Service (Real MAVLink)...")

	// 1. Fetch Config from API
	var config TelemetryConfig
	for {
		resp, err := http.Get(API_URL + "/api/config/telemetry")
		if err == nil && resp.StatusCode == 200 {
			json.NewDecoder(resp.Body).Decode(&config)
			resp.Body.Close()
			break
		}
		log.Println("Waiting for API to be ready...")
		time.Sleep(2 * time.Second)
	}

	// Apply Defaults
	if config.FCPort == "" {
		config.FCPort = "auto"
	}
	if config.FCBaud == 0 {
		config.FCBaud = 57600
	}

	// 2. Setup MAVLink Endpoints
	var endpoints []gomavlib.EndpointConf
	if config.FCPort == "auto" {
		log.Println("[Hardware] Auto-Detecting Flight Controller...")
		// Simple detection logic
		found := ""
		files, _ := os.ReadDir("/dev")
		for _, f := range files {
			if strings.HasPrefix(f.Name(), "ttyACM") || strings.HasPrefix(f.Name(), "ttyUSB") {
				found = "/dev/" + f.Name()
				break
			}
		}
		if found != "" {
			log.Printf("✅ Found Device: %s", found)
			endpoints = []gomavlib.EndpointConf{
				gomavlib.EndpointSerial{Device: found, Baud: config.FCBaud},
			}
		} else {
			log.Println("⚠️ No physical device found. Checking for SITL (UDP :14550)...")
			// In Legacy, SITL was EndpointUDPServer because SITL connects TO us.
			endpoints = []gomavlib.EndpointConf{
				gomavlib.EndpointUDPServer{Address: ":14550"},
			}
		}
	} else {
		// Manual Config
		if strings.HasPrefix(config.FCPort, "tcp:") {
			endpoints = []gomavlib.EndpointConf{gomavlib.EndpointTCPClient{Address: strings.TrimPrefix(config.FCPort, "tcp:")}}
		} else if strings.HasPrefix(config.FCPort, "udp:") {
			endpoints = []gomavlib.EndpointConf{gomavlib.EndpointUDPClient{Address: strings.TrimPrefix(config.FCPort, "udp:")}}
		} else {
			endpoints = []gomavlib.EndpointConf{gomavlib.EndpointSerial{Device: config.FCPort, Baud: config.FCBaud}}
		}
	}

	// 3. Create Main Node
	node, err := gomavlib.NewNode(gomavlib.NodeConf{
		Endpoints:   endpoints,
		Dialect:     ardupilotmega.Dialect,
		OutVersion:  gomavlib.V2,
		OutSystemID: 255, // GCS ID
	})
	if err != nil {
		log.Fatalf("Failed to create MAVLink Node: %v", err)
	}
	defer node.Close()

	log.Println("✅ MAVLink Node Started. Listening for events...")

	// Write Hardware Status (FC Connected)
	// API expects: {"fc_connected": true, "cam_connected": ...}
	// Note: We shouldn't overwrite checkHardwareStatus from API if possible.
	// Actually, API reads "HardwareStatusFile" which is monolithic.
	// If API writes cam_connected and attempts to read, we have a race/overwrite issue if we write ONLY fc_connected.
	// Strategy: vyom-telemetry creates a separate status file? No, API expects one.
	// Better Strategy: API manages the aggregate file.
	// BUT API reads from file.
	// Let's create a dedicated file for telemetry service connection status, e.g., "telemetry_meta.json"?
	// Or just update the loop in API to look for specific file?
	// For now, let's just make sure Telemetry Data is written.
	// The Dashboard "Flight Controller" status checks "fc_connected".
	// API sets "fc_connected" if it reads HardwareStatusFile.
	// We should write to `hardware.json` as well.

	go func() {
		// Periodically assert FC connection in hardware.json
		// CAUTION: This might race with API's camera check if they write to same file.
		// Ideally API should merge, but it just reads/writes entire structs.
		// Workaround: We will NOT write hardware.json here to avoid conflict.
		// Instead, we update current API logic to infer FC Connected if Telemetry Data is recent.
		// OR we rely on `vyom-telemetry` to be the source of truth for FC.

		// For now, let's proceed with just updating `telemetry.json`.
		// I will modify API to set IsConnected based on Telemetry != nil.
	}()

	// 4. UDP Client for LiveKit Relay
	udpAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:5000")
	udpConn, _ := net.DialUDP("udp", nil, udpAddr)

	// 5. GCS Endpoint Sync Loop
	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			resp, err := client.Get(API_URL + "/api/config/gcs-endpoints")
			if err != nil {
				continue
			}

			type Endpoint struct {
				ID        string `json:"id"`
				Host      string `json:"host"`
				Port      int    `json:"port"`
				Enabled   bool   `json:"enabled"`
				Telemetry bool   `json:"enable_telemetry"`
			}
			var eps []Endpoint
			json.NewDecoder(resp.Body).Decode(&eps)
			resp.Body.Close()

			fwMutex.Lock()
			currentIDs := make(map[string]bool)
			for _, ep := range eps {
				if !ep.Enabled || !ep.Telemetry {
					continue
				}
				id := ep.ID
				currentIDs[id] = true

				if _, exists := fwNodes[id]; !exists {
					log.Printf("Adding GCS Forward: %s (%s:%d)", id, ep.Host, ep.Port)
					addr := fmt.Sprintf("%s:%d", ep.Host, ep.Port)
					// GCS is listening, so we are a Client sending to them?
					// Usually forwarding to GCS means sending UDP packets to their listening port.
					// EndpointUDPClient sends to a destination.
					n, err := gomavlib.NewNode(gomavlib.NodeConf{
						Endpoints:   []gomavlib.EndpointConf{gomavlib.EndpointUDPClient{Address: addr}},
						Dialect:     ardupilotmega.Dialect,
						OutVersion:  gomavlib.V2,
						OutSystemID: 254, // Proxy ID
					})
					if err == nil {
						fwNodes[id] = n
					}
				}
			}
			// Cleanup
			for id, n := range fwNodes {
				if !currentIDs[id] {
					log.Printf("Removing GCS Forward: %s", id)
					n.Close()
					delete(fwNodes, id)
				}
			}
			fwMutex.Unlock()
		}
	}()

	// 6. LiveKit Sender Loop (5Hz Rate Limit)
	// 6. LiveKit Sender Loop (5Hz Rate Limit) & Local Status File Write (1Hz)
	go func() {
		lkTicker := time.NewTicker(200 * time.Millisecond)
		fileTicker := time.NewTicker(1 * time.Second)
		statusDir := "/tmp/vyom-status"
		statusFile := statusDir + "/telemetry.json"
		os.MkdirAll(statusDir, 0777)

		for {
			select {
			case <-lkTicker.C:
				telemMutex.Lock()
				t := lastTelem
				t.Timestamp = time.Now().UnixMilli()
				telemMutex.Unlock()

				if udpConn != nil {
					msg := LiveKitDataMessage{Type: "telemetry", Data: t}
					b, _ := json.Marshal(msg)
					udpConn.Write(b)
				}
			case <-fileTicker.C:
				telemMutex.Lock()
				t := lastTelem
				telemMutex.Unlock()

				// Map to API Struct (TelemetryStatus)
				// API expects:
				// type TelemetryStatus struct {
				// 	Battery *SysStatus      `json:"battery"`
				//  GPS     *GpsRaw         `json:"gps"`
				// 	HUD     *VfrHud         `json:"hud"`
				// 	System  *TelemetryState `json:"system"`
				// }
				type TelemetryState struct {
					Armed bool   `json:"armed"`
					Mode  string `json:"mode"`
				}
				type APIStatus struct {
					Battery *SysStatus      `json:"battery"`
					GPS     *GpsRaw         `json:"gps"`
					HUD     *VfrHud         `json:"hud"`
					System  *TelemetryState `json:"system"`
				}

				apiStatus := APIStatus{
					Battery: t.SysStatus,
					GPS:     t.GpsRawInt,
					HUD:     t.VfrHud,
					System: &TelemetryState{
						Armed: t.Armed,
						Mode:  t.Mode,
					},
				}

				if data, err := json.Marshal(apiStatus); err == nil {
					os.WriteFile(statusFile, data, 0666)
				}
			}
		}
	}()

	// 7. Event Loop
	for evt := range node.Events() {
		if frm, ok := evt.(*gomavlib.EventFrame); ok {
			// Forwarding
			fwMutex.Lock()
			for _, n := range fwNodes {
				n.WriteFrameExcept(frm.Channel, frm.Frame)
			}
			fwMutex.Unlock()

			// Parsing
			msg := frm.Message()
			telemMutex.Lock()
			switch m := msg.(type) {
			case *ardupilotmega.MessageHeartbeat:
				lastTelem.Armed = (m.BaseMode & ardupilotmega.MAV_MODE_FLAG_SAFETY_ARMED) != 0
				// custom mode mapping is complex, simple string for now
				if lastTelem.Armed {
					lastTelem.Mode = "ARMED"
				} else {
					lastTelem.Mode = "DISARMED"
				}
			case *ardupilotmega.MessageAttitude:
				if lastTelem.Attitude == nil {
					lastTelem.Attitude = &Attitude{}
				}
				lastTelem.Attitude.Roll = m.Roll
				lastTelem.Attitude.Pitch = m.Pitch
				lastTelem.Attitude.Yaw = m.Yaw
			case *ardupilotmega.MessageSysStatus:
				if lastTelem.SysStatus == nil {
					lastTelem.SysStatus = &SysStatus{}
				}
				lastTelem.SysStatus.Voltage = float32(m.VoltageBattery) / 1000.0
				lastTelem.SysStatus.BatteryRemaining = int(m.BatteryRemaining)
			case *ardupilotmega.MessageGlobalPositionInt:
				if lastTelem.GlobalPositionInt == nil {
					lastTelem.GlobalPositionInt = &GlobalPosition{}
				}
				lastTelem.GlobalPositionInt.Lat = m.Lat
				lastTelem.GlobalPositionInt.Lon = m.Lon
				lastTelem.GlobalPositionInt.Alt = m.Alt
				lastTelem.GlobalPositionInt.Hdg = m.Hdg
			case *ardupilotmega.MessageGpsRawInt:
				if lastTelem.GpsRawInt == nil {
					lastTelem.GpsRawInt = &GpsRaw{}
				}
				lastTelem.GpsRawInt.FixType = uint8(m.FixType)
				lastTelem.GpsRawInt.Satellites = m.SatellitesVisible
			case *ardupilotmega.MessageVfrHud:
				if lastTelem.VfrHud == nil {
					lastTelem.VfrHud = &VfrHud{}
				}
				lastTelem.VfrHud.Heading = m.Heading
				lastTelem.VfrHud.Alt = m.Alt
				lastTelem.VfrHud.Airspeed = m.Airspeed
				lastTelem.VfrHud.Groundspeed = m.Groundspeed
				lastTelem.VfrHud.Throttle = m.Throttle
				lastTelem.VfrHud.Climb = m.Climb
			}
			telemMutex.Unlock()
		}
	}
}
