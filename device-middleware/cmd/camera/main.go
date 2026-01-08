package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	Port = ":8081"
)

// StreamBroadcaster handles fan-out of the video stream
type StreamBroadcaster struct {
	mu      sync.Mutex
	clients map[chan []byte]bool
}

func NewBroadcaster() *StreamBroadcaster {
	return &StreamBroadcaster{
		clients: make(map[chan []byte]bool),
	}
}

func (b *StreamBroadcaster) Subscribe() chan []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Bump buffer to 5 frames to absorb network jitter.
	// 5 * ~100KB is fine for RAM.
	ch := make(chan []byte, 5)
	b.clients[ch] = true
	log.Printf("[Camera] Client Subscribed. Total: %d", len(b.clients))
	return ch
}

func (b *StreamBroadcaster) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.clients[ch]; ok {
		delete(b.clients, ch)
		close(ch)
	}
	log.Printf("[Camera] Client Unsubscribed. Total: %d", len(b.clients))
}

func (b *StreamBroadcaster) Broadcast(frame []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- frame:
		default:
			// Dropping a COMPLETE frame is fine (stutter).
			// Dropping a chunk was causing tearing.
		}
	}
}

var broadcaster = NewBroadcaster()

func main() {
	log.Println("🎥 Starting Vyom Camera Service (Robust Go-Framing)...")

	go runGStreamerPipeline()

	http.HandleFunc("/", handleStream)

	log.Printf("📹 Serving Real Camera Stream on %s", Port)
	if err := http.ListenAndServe(Port, nil); err != nil {
		log.Fatalf("Failed to start camera service: %v", err)
	}
}

func runGStreamerPipeline() {
	formats := []struct {
		Name string
	}{
		{"MJPEG"}, // Try MJPEG first
		{"YUY2"},  // Then Raw
		{"ANY"},
	}

	var lastConfigRes string
	safeMode := false
	currentFormatIdx := 0
	consecutiveFailures := 0

	for {
		log.Println("[Camera] Starting GStreamer Pipeline...")

		hasLibCamera := false
		if _, err := exec.LookPath("gst-inspect-1.0"); err == nil {
			cmd := exec.Command("gst-inspect-1.0", "libcamerasrc")
			if err := cmd.Run(); err == nil {
				hasLibCamera = true
			}
		}

		// 1. Read Config
		type Config struct {
			Resolution   string `json:"resolution"`
			CameraDevice string `json:"camera_device"` // "Name (/dev/videoX)" or "auto"
		}
		var width, height = 640, 480
		var currentConfigRes = "640x480"
		var selectedDevice = "/dev/video0" // Default

		if data, err := os.ReadFile("demo_config.json"); err == nil {
			var cfg Config
			if json.Unmarshal(data, &cfg) == nil {
				// Resolution
				if cfg.Resolution != "" {
					currentConfigRes = cfg.Resolution
				}
				// Device
				if cfg.CameraDevice != "" && cfg.CameraDevice != "auto" {
					// 1. Try Parse "Name (/dev/videoX)" -> "/dev/videoX"
					if idx := strings.LastIndex(cfg.CameraDevice, "("); idx != -1 {
						path := strings.TrimSuffix(cfg.CameraDevice[idx+1:], ")")
						if strings.HasPrefix(path, "/dev/video") {
							selectedDevice = path
						}
					} else if strings.HasPrefix(cfg.CameraDevice, "/dev/video") {
						// 2. Fallback: Raw path
						selectedDevice = cfg.CameraDevice
					}
				}

				if safeMode && currentConfigRes != lastConfigRes {
					log.Printf("[Camera] Config changed from %s to %s. Resetting Safe Mode.", lastConfigRes, currentConfigRes)
					safeMode = false
					consecutiveFailures = 0
				}
				lastConfigRes = currentConfigRes
				fmt.Sscanf(currentConfigRes, "%dx%d", &width, &height)
				log.Printf("[Camera] Loaded Config: Res=%dx%d, Device=%s", width, height, selectedDevice)
			}
		}

		if safeMode {
			width, height = 640, 480
			log.Println("[Camera] SAFE MODE ACTIVE: Forcing 640x480")
		}

		// 3. Construct Source Pipeline
		// NOTE: We do NOT use multipartmux here anymore. We act as the muxer in Go.
		// We output RAW JPEG stream.
		var cmdStr string

		if hasLibCamera {
			// LibCamera (Raspberry Pi)
			// Src -> Raw -> Tee -> (Scaling->Jpeg->FdSink) + (VP8->Cloud)
			cmdStr = fmt.Sprintf(
				"exec gst-launch-1.0 -q libcamerasrc camera-name=%s ! video/x-raw,width=%d,height=%d ! videoconvert ! "+
					"tee name=t ! queue max-size-buffers=5 leaky=downstream ! videorate ! video/x-raw,framerate=15/1 ! videoscale ! video/x-raw,width=%d,height=%d ! jpegenc ! fdsink sync=false "+
					"t. ! queue max-size-buffers=5 leaky=downstream ! videorate ! video/x-raw,framerate=15/1 ! vp8enc error-resilient=1 deadline=1 keyframe-max-dist=30 cpu-used=5 ! queue ! avmux_ivf ! tcpclientsink host=127.0.0.1 port=5600",
				selectedDevice, width, height, width, height,
			)
		} else if _, err := os.Stat(selectedDevice); err == nil {
			format := formats[currentFormatIdx]
			log.Printf("[Camera] Using Device: %s. Attempting Format: %s", selectedDevice, format.Name)

			if format.Name == "MJPEG" {
				// MJPEG PASSTHROUGH (Parse -> Split)
				// Local: JpegParse -> FdSink (Clean JPEGs)
				// Cloud: JpegDec -> VP8
				cmdStr = fmt.Sprintf(
					"exec gst-launch-1.0 -q v4l2src device=%s ! image/jpeg,width=%d,height=%d,framerate=30/1 ! jpegparse ! "+
						"tee name=t ! queue max-size-buffers=5 leaky=downstream ! fdsink sync=false "+
						"t. ! queue max-size-buffers=5 leaky=downstream ! jpegdec ! videoconvert ! videorate ! video/x-raw,framerate=15/1 ! vp8enc error-resilient=1 deadline=1 keyframe-max-dist=30 cpu-used=5 ! queue ! avmux_ivf ! tcpclientsink host=127.0.0.1 port=5600",
					selectedDevice, width, height,
				)
			} else {
				// YUY2/Raw -> Re-Encode
				var baseSrc string
				if format.Name == "YUY2" {
					baseSrc = fmt.Sprintf("v4l2src device=%s ! video/x-raw,format=YUY2,width=%d,height=%d", selectedDevice, width, height)
				} else {
					baseSrc = fmt.Sprintf("v4l2src device=%s ! decodebin ! videoconvert ! videoscale ! video/x-raw,width=%d,height=%d", selectedDevice, width, height)
				}
				cmdStr = fmt.Sprintf(
					"exec gst-launch-1.0 -q %s ! videoconvert ! videorate ! video/x-raw,framerate=15/1 ! "+
						"tee name=t ! queue max-size-buffers=5 leaky=downstream ! jpegenc ! fdsink sync=false "+
						"t. ! queue max-size-buffers=5 leaky=downstream ! vp8enc error-resilient=1 deadline=1 keyframe-max-dist=30 cpu-used=5 ! queue ! avmux_ivf ! tcpclientsink host=127.0.0.1 port=5600",
					baseSrc,
				)
			}
		} else {
			// Test Source
			cmdStr = fmt.Sprintf(
				"exec gst-launch-1.0 -q videotestsrc is-live=true pattern=snow ! videoconvert ! videorate ! video/x-raw,framerate=15/1,width=%d,height=%d ! "+
					"tee name=t ! queue max-size-buffers=5 leaky=downstream ! jpegenc ! fdsink sync=false "+
					"t. ! queue max-size-buffers=5 leaky=downstream ! vp8enc error-resilient=1 deadline=1 keyframe-max-dist=30 cpu-used=5 ! queue ! avmux_ivf ! tcpclientsink host=127.0.0.1 port=5600",
				width, height,
			)
		}

		if safeMode {
			log.Println("⚠️ Engaging SAFE MODE PIPELINE (Robust Fallback).")
			if _, err := os.Stat("/dev/video0"); err == nil {
				cmdStr = "exec gst-launch-1.0 -q v4l2src device=/dev/video0 ! image/jpeg,width=640,height=480 ! jpegparse ! fdsink sync=false"
			} else {
				cmdStr = "exec gst-launch-1.0 -q videotestsrc is-live=true ! videoscale ! video/x-raw,width=640,height=480 ! jpegenc ! fdsink sync=false"
			}
		}

		cmd := exec.Command("sh", "-c", cmdStr)
		cmd.Stderr = os.Stderr

		// Stop Watcher Setup
		stopWatcher := make(chan bool)
		go func() {
			initialStat, err := os.Stat("demo_config.json")
			if err != nil {
				return
			}
			initialTime := initialStat.ModTime()
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-stopWatcher:
					return
				case <-ticker.C:
					stat, err := os.Stat("demo_config.json")
					if err == nil && !stat.ModTime().Equal(initialTime) {
						log.Println("[Camera] 🔄 Config File Changed! Triggering Hot Reload...")
						if cmd.Process != nil {
							cmd.Process.Kill()
						}
						return
					}
				}
			}
		}()

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("Failed to get stdout: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if err := cmd.Start(); err != nil {
			log.Printf("Failed to start GStreamer: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// --- ROBUST FRAME PARSING LOOP ---
		// We read the stream and search for JPEG EOI (FF D9).
		// This guarantees we only broadcast COMPLETE frames.
		readBuf := make([]byte, 8192) // Read in larger chunks
		var frameBuffer []byte

		started := time.Now()
		bytesReadTotal := 0

		// SOI (Start of Image): FF D8
		// EOI (End of Image):   FF D9
		eoiMarker := []byte{0xFF, 0xD9}

		for {
			n, err := stdout.Read(readBuf)
			if n > 0 {
				bytesReadTotal += n
				// Append new data to buffer
				frameBuffer = append(frameBuffer, readBuf[:n]...)

				// Scan for EOI marker
				// We loop in case multiple frames are in the buffer
				for {
					idx := bytes.Index(frameBuffer, eoiMarker)
					if idx == -1 {
						break // No full frame yet, keep reading
					}

					// Full frame found! (Up to idx + 2 bytes for FF D9)
					frameLen := idx + 2
					fullFrame := make([]byte, frameLen)
					copy(fullFrame, frameBuffer[:frameLen])

					// Broadcast ONLY the full frame
					broadcaster.Broadcast(fullFrame)

					// Remove the processed frame from the buffer
					frameBuffer = frameBuffer[frameLen:]
				}
			}
			if err != nil {
				log.Printf("Read error (Pipeline Died): %v", err)
				break
			}
		}

		cmd.Wait()
		close(stopWatcher)
		duration := time.Since(started)
		log.Printf("[Camera] Pipeline exited after %v. Bytes read: %d", duration, bytesReadTotal)

		if duration < 5*time.Second || bytesReadTotal == 0 {
			consecutiveFailures++
			log.Printf("⚠️ Pipeline failed quickly (%v). Failures: %d", duration, consecutiveFailures)

			if consecutiveFailures >= 2 {
				// Fallback Logic
				if !hasLibCamera && !strings.Contains(cmdStr, "videotestsrc") {
					log.Println("⚠️ High-Res failed. Enabling Persistent SAFE MODE.")
					safeMode = true
					width, height = 640, 480
					currentFormatIdx = 0
					consecutiveFailures = 0
					continue
				}
				if !hasLibCamera && !strings.Contains(cmdStr, "videotestsrc") {
					currentFormatIdx = (currentFormatIdx + 1) % len(formats)
					log.Printf("⚠️ Format failed. Switching to next format: %s", formats[currentFormatIdx].Name)
					consecutiveFailures = 0
				}
			}
		} else {
			consecutiveFailures = 0
		}

		log.Println("[Camera] Restarting in 2s...")
		time.Sleep(2 * time.Second)
	}
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	log.Println("[Camera] HTTP Client Connected")

	// Standard Multipart Header
	boundary := "vyomboundary"
	w.Header().Set("Content-Type", fmt.Sprintf("multipart/x-mixed-replace; boundary=%s", boundary))
	w.WriteHeader(http.StatusOK)

	ch := broadcaster.Subscribe()
	defer broadcaster.Unsubscribe(ch)

	for {
		select {
		case <-r.Context().Done():
			log.Println("[Camera] HTTP Client Disconnected")
			return
		case frame := <-ch:
			// Manually construct the Frame Header
			// --boundary
			// Content-Type: image/jpeg
			// Content-Length: <len>
			// <Global Header End>
			// [Data]

			header := fmt.Sprintf("\r\n--%s\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", boundary, len(frame))

			if _, err := w.Write([]byte(header)); err != nil {
				return
			}
			if _, err := w.Write(frame); err != nil {
				return
			}
			// Flush to ensure browser sees the frame immediately
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}
