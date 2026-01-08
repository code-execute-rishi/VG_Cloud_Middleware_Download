package main

import (
	"bytes"
	"encoding/json"
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

// --- API Status Structs ---
type APIState struct {
	Armed bool   `json:"armed"`
	Mode  string `json:"mode"`
}
type APIStatus struct {
	Battery     *SysStatus `json:"battery"`
	GPS         *GpsRaw    `json:"gps"`
	HUD         *VfrHud    `json:"hud"`
	System      *APIState  `json:"system"`
	FCConnected bool       `json:"fc_connected"`
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

	lastTelem     LiveKitTelemetry
	lastHeartbeat time.Time
	telemMutex    sync.Mutex
)

func main() {
	log.Println("Starting Vyom Telemetry Service (Hot-Plug Enabled)...")

	// 1. Fetch Config from API
	var config TelemetryConfig
	fetchConfig := func() TelemetryConfig {
		var cfg TelemetryConfig
		for {
			resp, err := http.Get(API_URL + "/api/config/telemetry")
			if err == nil && resp.StatusCode == 200 {
				json.NewDecoder(resp.Body).Decode(&cfg)
				resp.Body.Close()
				return cfg
			}
			log.Println("Waiting for API to be ready...")
			time.Sleep(2 * time.Second)
		}
	}
	config = fetchConfig()

	// Apply Defaults
	if config.FCPort == "" {
		config.FCPort = "auto"
	}
	if config.FCBaud == 0 {
		config.FCBaud = 57600
	}

	// 2. Start Config Watcher (Exits on Change)
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		for range ticker.C {
			newCfg := fetchConfig()
			// Normalize defaults for comparison
			if newCfg.FCPort == "" {
				newCfg.FCPort = "auto"
			}
			if newCfg.FCBaud == 0 {
				newCfg.FCBaud = 57600
			}

			if newCfg.FCPort != config.FCPort || newCfg.FCBaud != config.FCBaud {
				log.Printf("[Config] Config Changed (Port: %s -> %s). Restarting...", config.FCPort, newCfg.FCPort)
				os.Exit(0) // Systemd will restart us
			}
		}
	}()

	// 3. Start Status Writer Loop (Always Run)
	go func() {
		// UDP Connection to LiveKit Relay
		addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:5000")
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			log.Printf("[Error] Failed to dial LiveKit Relay: %v", err)
		} else {
			defer conn.Close()
			log.Println("[Info] UDP Telemetry Link Established (:5000)")
		}

		ticker := time.NewTicker(200 * time.Millisecond) // 5Hz
		for range ticker.C {
			telemMutex.Lock()

			// Check connectivity
			connected := time.Since(lastHeartbeat) < 3*time.Second
			if !connected {
				lastTelem.Mode = "DISCONNECTED"
			}

			status := APIStatus{
				Battery:     lastTelem.SysStatus,
				GPS:         lastTelem.GpsRawInt,
				HUD:         lastTelem.VfrHud,
				FCConnected: connected,
				System: &APIState{
					Armed: lastTelem.Armed,
					Mode:  lastTelem.Mode,
				},
			}

			// Construct LiveKit Message
			lkMsg := LiveKitDataMessage{
				Type: "telemetry",
				Data: lastTelem,
			}
			lkMsg.Data.Timestamp = time.Now().UnixMilli() // Current time
			telemMutex.Unlock()

			// Write to Disk (Local UI)
			data, _ := json.Marshal(status)
			os.MkdirAll("/tmp/vyom-status", 0777)
			os.WriteFile("/tmp/vyom-status/telemetry.json", data, 0666)

			// Send to LiveKit (Cloud)
			if conn != nil {
				if lkData, err := json.Marshal(lkMsg); err == nil {
					conn.Write(lkData)
				}
			}
		}
	}()

	// 4. Handle Disabled State
	if config.FCPort == "disabled" {
		log.Println("[Info] Telemetry Disabled (Idle Mode).")
		select {} // Block forever (waiting for config change restart)
	}

	// Start API Reporter (Send status to API Service)
	go func() {
		client := &http.Client{Timeout: 500 * time.Millisecond}
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
			telemMutex.Lock()

			// Check Staleness
			if time.Since(lastHeartbeat) > 3*time.Second {
				telemMutex.Unlock()
				continue // Do not report stale data (Let API timeout)
			}

			// Prepare minimal status updates for API
			update := map[string]interface{}{
				"latitude":        0.0,
				"longitude":       0.0,
				"altitude":        0.0,
				"speed":           0.0,
				"heading":         0.0,
				"signal_strength": 100,
				"battery":         0,
				"flight_mode":     lastTelem.Mode,
			}
			if lastTelem.GlobalPositionInt != nil {
				update["latitude"] = float64(lastTelem.GlobalPositionInt.Lat) / 1e7
				update["longitude"] = float64(lastTelem.GlobalPositionInt.Lon) / 1e7
				update["altitude"] = float32(lastTelem.GlobalPositionInt.Alt) / 1000.0
			}
			if lastTelem.VfrHud != nil {
				update["heading"] = float32(lastTelem.VfrHud.Heading)
				update["speed"] = lastTelem.VfrHud.Groundspeed
			}
			if lastTelem.SysStatus != nil {
				update["battery"] = lastTelem.SysStatus.BatteryRemaining
			}
			telemMutex.Unlock()

			data, _ := json.Marshal(update)
			client.Post(API_URL+"/internal/telemetry", "application/json", bytes.NewBuffer(data))
		}
	}()

	// 5. Main Reconnection Loop
	for {
		runMavlinkSession(config)
		log.Println("[Warn] MAVLink Session Ended. Restarting in 1s...")
		time.Sleep(1 * time.Second)
	}
}

func runMavlinkSession(config TelemetryConfig) {
	var endpoints []gomavlib.EndpointConf

	// --- Device Detection Logic ---
	if config.FCPort == "auto" {
		log.Println("[Auto] Scanning for Flight Controller...")

		var found string
		// Scanning Loop
		for {
			files, _ := os.ReadDir("/dev")
			for _, f := range files {
				if strings.HasPrefix(f.Name(), "ttyACM") || strings.HasPrefix(f.Name(), "ttyUSB") {
					found = "/dev/" + f.Name()
					break
				}
			}

			if found != "" {
				log.Printf("[Info] Found Device: %s", found)
				break
			}

			// Log occasionally/verbose?
			// log.Println("... scanning ...")
			time.Sleep(2 * time.Second)
		}

		endpoints = []gomavlib.EndpointConf{
			gomavlib.EndpointSerial{Device: found, Baud: config.FCBaud},
		}

	} else {
		// Manual Config with Basic Validation to prevent arbitrary SSRF/Command Injection
		if strings.HasPrefix(config.FCPort, "tcp:") {
			addr := strings.TrimPrefix(config.FCPort, "tcp:")
			if !isValidNetworkAddress(addr) {
				log.Printf("[Security] Invalid TCP Address rejected: %s", addr)
				return // Early exit, auto-restart loop will retry
			}
			endpoints = []gomavlib.EndpointConf{gomavlib.EndpointTCPClient{Address: addr}}
		} else if strings.HasPrefix(config.FCPort, "udpin:") {
			addr := strings.TrimPrefix(config.FCPort, "udpin:")
			// UDP Server usually binds local, less risk, but validate format
			if !isValidNetworkAddress(addr) {
				log.Printf("[Security] Invalid UDP Bind Address rejected: %s", addr)
				return
			}
			endpoints = []gomavlib.EndpointConf{gomavlib.EndpointUDPServer{Address: addr}}
		} else if strings.HasPrefix(config.FCPort, "udp:") {
			addr := strings.TrimPrefix(config.FCPort, "udp:")
			if !isValidNetworkAddress(addr) {
				log.Printf("[Security] Invalid UDP Address rejected: %s", addr)
				return
			}
			endpoints = []gomavlib.EndpointConf{gomavlib.EndpointUDPClient{Address: addr}}
		} else {
			// Serial Device
			// Simple check to ensure it looks like a device path or COM port
			if !strings.HasPrefix(config.FCPort, "/dev/") && !strings.HasPrefix(config.FCPort, "COM") {
				log.Printf("[Security] Invalid Serial Port Rejected: %s", config.FCPort)
				return
			}
			endpoints = []gomavlib.EndpointConf{gomavlib.EndpointSerial{Device: config.FCPort, Baud: config.FCBaud}}
		}
	}

	// --- Create Node ---
	node, err := gomavlib.NewNode(gomavlib.NodeConf{
		Endpoints:   endpoints,
		Dialect:     ardupilotmega.Dialect,
		OutVersion:  gomavlib.V2,
		OutSystemID: 200,
	})
	if err != nil {
		log.Printf("[Error] Failed to create MAVLink Node: %v", err)
		return
	}
	defer node.Close()

	log.Println("[Info] MAVLink Node Started. Loop OK.")

	// --- Heartbeat Ticker ---
	hbTicker := time.NewTicker(1 * time.Second)
	defer hbTicker.Stop()

	// --- Event Loop ---
	streamRequested := false
	events := node.Events()

	for {
		select {
		case <-hbTicker.C:
			node.WriteMessageAll(&ardupilotmega.MessageHeartbeat{
				Type:           ardupilotmega.MAV_TYPE_GCS,
				Autopilot:      ardupilotmega.MAV_AUTOPILOT_INVALID,
				BaseMode:       0,
				CustomMode:     0,
				SystemStatus:   ardupilotmega.MAV_STATE_ACTIVE,
				MavlinkVersion: 3,
			})

		case evt, ok := <-events:
			if !ok {
				log.Println("[Error] Event Channel Closed (Device Disconnected?)")
				return
			}

			switch e := evt.(type) {
			case *gomavlib.EventFrame:
				frm := e
				// Forwarding
				fwMutex.Lock()
				for _, n := range fwNodes {
					n.WriteFrameExcept(frm.Channel, frm.Frame)
				}
				fwMutex.Unlock()

				msg := frm.Message()
				telemMutex.Lock()
				lastHeartbeat = time.Now()
				telemMutex.Unlock()

				telemMutex.Lock()
				switch m := msg.(type) {
				case *ardupilotmega.MessageHeartbeat:
					if !streamRequested {
						log.Printf("[Info] [Telemetry] Heartbeat from SysID:%d! Requesting Stream...", frm.SystemID())
						node.WriteMessageAll(&ardupilotmega.MessageRequestDataStream{
							TargetSystem:    frm.SystemID(),
							TargetComponent: frm.ComponentID(),
							ReqStreamId:     uint8(ardupilotmega.MAV_DATA_STREAM_ALL),
							ReqMessageRate:  4, // 4Hz
							StartStop:       1,
						})
						streamRequested = true
					}
					lastTelem.Armed = (m.BaseMode & ardupilotmega.MAV_MODE_FLAG_SAFETY_ARMED) != 0
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

			case *gomavlib.EventParseError:
				// log.Printf("[Warn] MAVLink Parse Error: %v", e.Error)
			case *gomavlib.EventStreamRequested:
				log.Printf("[Info] Stream Requested")
			case *gomavlib.EventChannelOpen:
				log.Printf("[Info] Channel Opened")
			case *gomavlib.EventChannelClose:
				log.Printf("[Error] Channel Closed")
				return // Trigger restart logic
			}
		}
	}
}

// Simple validator for host:port
func isValidNetworkAddress(addr string) bool {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "" || port == "" {
		return false
	}
	// Allow IP addresses (v4/v6) and "localhost"
	// Block other hostnames to prevent DNS rebinding or internal scanning via name
	if host != "localhost" && net.ParseIP(host) == nil {
		return false
	}
	return true
}
