package main

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
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
	ch := make(chan []byte, 10) // Buffer slightly
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

func (b *StreamBroadcaster) Broadcast(data []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- data:
		default:
			// Slow client, drop packet to avoid blocking everyone
		}
	}
}

var broadcaster = NewBroadcaster()

func main() {
	log.Println("🎥 Starting Vyom Camera Service (Persistent GStreamer)...")

	// Start GStreamer in background immediately
	go runGStreamerPipeline()

	http.HandleFunc("/", handleStream)

	log.Printf("📹 Serving Real Camera Stream on %s", Port)
	if err := http.ListenAndServe(Port, nil); err != nil {
		log.Fatalf("Failed to start camera service: %v", err)
	}
}

func runGStreamerPipeline() {
	for {
		log.Println("[Camera] Starting GStreamer Pipeline...")

		width := "640"
		height := "480"

		// Pipeline Logic:
		// 1. Check if 'libcamerasrc' exists (Preferred for RPi)
		// 2. Check /dev/video0 (Fallback)
		// 3. TestSrc (Fallback)

		src := ""
		hasLibCamera := false
		if _, err := exec.LookPath("gst-inspect-1.0"); err == nil {
			cmd := exec.Command("gst-inspect-1.0", "libcamerasrc")
			if err := cmd.Run(); err == nil {
				hasLibCamera = true
			}
		}

		if hasLibCamera {
			log.Println("[Camera] 'libcamerasrc' found. Using libcamera pipeline.")
			// libcamerasrc automatically handles ISP and format negotiation.
			src = fmt.Sprintf("libcamerasrc ! video/x-raw,width=%s,height=%s,framerate=30/1", width, height)
		} else if _, err := os.Stat("/dev/video0"); err == nil {
			log.Println("[Camera] Found /dev/video0 (v4l2src).")
			src = fmt.Sprintf("v4l2src device=/dev/video0 ! videoconvert ! video/x-raw,format=I420,width=%s,height=%s,framerate=30/1", width, height)
		} else {
			log.Println("[Camera] No camera found. Using testsrc (Snow).")
			src = fmt.Sprintf("videotestsrc is-live=true pattern=snow ! videoconvert ! video/x-raw,format=I420,width=%s,height=%s,framerate=30/1", width, height)
		}

		// Use queues to prevent blocking.
		cmdStr := fmt.Sprintf(
			"gst-launch-1.0 -q %s ! "+
				"tee name=t ! queue max-size-buffers=4 leaky=downstream ! vp8enc error-resilient=1 deadline=1 keyframe-max-dist=30 cpu-used=5 ! queue ! avmux_ivf ! tcpclientsink host=127.0.0.1 port=5600 "+
				"t. ! queue max-size-buffers=4 leaky=downstream ! videoscale ! video/x-raw,width=320,height=240 ! jpegenc ! multipartmux boundary=vyomboundary ! fdsink",
			src,
		)

		cmd := exec.Command("sh", "-c", cmdStr)
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

		// Read Loop
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				// Copy data to broadcast
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				broadcaster.Broadcast(chunk)
			}
			if err != nil {
				log.Printf("Read error (Pipeline Died): %v", err)
				break
			}
		}

		cmd.Wait()
		log.Println("[Camera] Pipeline exited. Restarting in 2s...")
		time.Sleep(2 * time.Second)
	}
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	log.Println("[Camera] HTTP Client Connected")

	// Send Headers
	mimeWriter := multipart.NewWriter(w)
	mimeWriter.SetBoundary("vyomboundary")
	w.Header().Set("Content-Type", fmt.Sprintf("multipart/x-mixed-replace; boundary=%s", mimeWriter.Boundary()))
	w.WriteHeader(http.StatusOK)

	ch := broadcaster.Subscribe()
	defer broadcaster.Unsubscribe(ch)

	for {
		select {
		case <-r.Context().Done():
			log.Println("[Camera] HTTP Client Disconnected")
			return
		case chunk := <-ch:
			if _, err := w.Write(chunk); err != nil {
				return
			}
			// Flush if possible
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}
