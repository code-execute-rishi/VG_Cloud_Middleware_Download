package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/ivfreader"
)

const (
	API_URL    = "http://localhost:8085"
	StatusFile = "/tmp/vyom-status/livekit.json"
)

type LiveKitConfig struct {
	Token      string `json:"token"`
	LiveKitURL string `json:"livekit_url"`
}

type LiveKitStatus struct {
	State        string `json:"state"`
	RoomName     string `json:"room_name"`
	Participants int    `json:"participants"`
	LastError    string `json:"last_error"`
}

var (
	videoTrack      *lksdk.LocalSampleTrack
	videoTrackMutex sync.RWMutex
	activeRoom      *lksdk.Room
	activeRoomMutex sync.RWMutex
)

func reportStatus(state string, errStr string, room *lksdk.Room) {
	status := LiveKitStatus{
		State:     state,
		LastError: errStr,
	}
	if room != nil {
		status.RoomName = room.Name()
		status.Participants = 1 // Placeholder: Cannot access private participants field
	}

	data, _ := json.Marshal(status)
	os.MkdirAll("/tmp/vyom-status", 0777)
	os.WriteFile(StatusFile, data, 0666)
}

func startDataRelayServer() {
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:5000")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("[DataRelay] ❌ Failed to bind UDP 127.0.0.1:5000: %v", err)
	}
	defer conn.Close()
	log.Println("[DataRelay] 🟢 Listening on UDP 127.0.0.1:5000 for Telemetry...")

	buf := make([]byte, 65535)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("[DataRelay] Read Error: %v", err)
			continue
		}

		activeRoomMutex.RLock()
		room := activeRoom
		activeRoomMutex.RUnlock()

		if room != nil && room.LocalParticipant != nil {
			// Forward Data to Cloud
			if err := room.LocalParticipant.PublishData(buf[:n], lksdk.WithDataPublishReliable(true), lksdk.WithDataPublishTopic("telemetry")); err != nil {
				log.Printf("[DataRelay] Publish Failed: %v", err)
			}
		}
	}
}

func startVideoRelayServer() {
	addr := "127.0.0.1:5600"
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("[VideoRelay] ❌ Failed to bind %s: %v", addr, err)
	}
	defer ln.Close()
	log.Printf("[VideoRelay] 🟢 Listening on %s for Camera Stream...", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[VideoRelay] Accept Error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Println("[VideoRelay] 🎥 Camera Connected!")
		ivf, header, err := ivfreader.NewWith(conn)
		if err != nil {
			log.Printf("[VideoRelay] ❌ Failed to read IVF Header: %v. Closing connection.", err)
			conn.Close()
			continue
		}
		log.Printf("[VideoRelay] ✅ Stream Header Received! %dx%d", header.Width, header.Height)

		for {
			frame, _, err := ivf.ParseNextFrame()
			if err != nil {
				break
			}
			videoTrackMutex.RLock()
			track := videoTrack
			videoTrackMutex.RUnlock()

			if track != nil {
				sample := media.Sample{Data: frame, Duration: time.Second / 30}
				track.WriteSample(sample, nil)
			}
		}
		conn.Close()
		log.Println("[VideoRelay] Camera Disconnected. Waiting for reconnection...")
	}
}

func main() {
	log.Println("🛰️ Starting Vyom LiveKit Service (Real)...")

	// Start Video Relay Immediately
	go startVideoRelayServer()
	go startDataRelayServer()

	for {
		reportStatus("Disconnected", "", nil)
		activeRoomMutex.Lock()
		activeRoom = nil
		activeRoomMutex.Unlock()

		// 1. Fetch Config
		resp, err := http.Get(API_URL + "/api/config/livekit")
		if err != nil || resp.StatusCode != 200 {
			log.Printf("Failed to fetch LiveKit config: %v", err)
			reportStatus("Error", "Config Fetch Failed", nil)
			time.Sleep(5 * time.Second)
			continue
		}

		var config LiveKitConfig
		if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
			log.Printf("Failed to decode LiveKit config: %v", err)
			resp.Body.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		resp.Body.Close()

		if config.Token == "" || config.LiveKitURL == "" {
			log.Println("Empty Token or URL")
			time.Sleep(5 * time.Second)
			continue
		}

		// 2. Connect
		log.Printf("Connecting to LiveKit: %s", config.LiveKitURL)
		reportStatus("Connecting", "", nil)

		// Create Callback for Disconnect
		done := make(chan bool)
		cb := lksdk.NewRoomCallback()
		cb.OnDisconnected = func() {
			log.Println("⚠️ LiveKit Disconnected")
			close(done)
		}

		// Connect using Token
		room, err := lksdk.ConnectToRoomWithToken(config.LiveKitURL, config.Token, cb)

		if err != nil {
			log.Printf("Failed to connect to room: %v", err)
			reportStatus("Error", fmt.Sprintf("Connection Failed: %v", err), nil)
			time.Sleep(5 * time.Second)
			continue
		}

		// Start Video Relay (Already started at top of main)

		// Publish Video Track
		track, err := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8})
		if err == nil {
			if _, err := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{
				Name: "camera_low",
			}); err == nil {
				log.Println("✅ Published Video Track!")
				videoTrackMutex.Lock()
				videoTrack = track
				videoTrackMutex.Unlock()
			} else {
				log.Printf("❌ Failed to publish track: %v", err)
			}
		} else {
			log.Printf("❌ Failed to create track: %v", err)
		}

		log.Println("✅ Connected to LiveKit Room!")
		reportStatus("Connected", "", room)

		activeRoomMutex.Lock()
		activeRoom = room
		activeRoomMutex.Unlock()

		// Monitor Loop
		ticker := time.NewTicker(2 * time.Second)
	loop:
		for {
			select {
			case <-done:
				break loop
			case <-ticker.C:
				reportStatus("Connected", "", room)
			}
		}

		room.Disconnect()
		activeRoomMutex.Lock()
		activeRoom = nil
		activeRoomMutex.Unlock()
		log.Println("Disconnected from Room. Retrying...")
		time.Sleep(2 * time.Second)
	}
}
