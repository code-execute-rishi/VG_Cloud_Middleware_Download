package camera

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Supervisor manages the camera polling loop and restart logic
func StartCameraSupervisor(ctx context.Context, res string) {

	log.Println("[Camera] Supervisor started.")

	// 1. Check if rpicam-vid exists
	_, err := exec.LookPath("rpicam-vid")
	useLibCamera := (err == nil)

	// Crash Counter
	crashCount := 0
	lastCrashTime := time.Now()
	forceTestPattern := false

	for {
		// Check for Context Cancel
		select {
		case <-ctx.Done():
			log.Println("[Camera] Supervisor stopping...")
			return
		default:
		}

		var cmd *exec.Cmd
		// Resolution Parsing
		width := "640"
		height := "480"
		if res == "1280x720" {
			width = "1280"
			height = "720"
		}

		// determine mode
		useRealCamera := !forceTestPattern

		// If using libcamera
		if useRealCamera && useLibCamera {
			log.Println("[Camera] Using rpicam-vid (Libcamera)...")
			cmd = exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf(
				"rpicam-vid -t 0 --inline --width %s --height %s --framerate 30 --codec yuv420 -o - | "+
					"gst-launch-1.0 fdsrc ! videoparse width=%s height=%s framerate=30/1 format=i420 ! "+
					"tee name=t ! queue max-size-buffers=1 leaky=downstream ! vp8enc deadline=1 ! avmux_ivf ! tcpclientsink host=127.0.0.1 port=5600 "+
					"t. ! queue ! videoscale ! video/x-raw,width=320,height=240 ! jpegenc ! multipartmux boundary=vyomboundary ! tcpserversink host=0.0.0.0 port=8081",
				width, height, width, height,
			))
		} else if useRealCamera {
			// Auto-Detect V4L2
			videoDevice := "/dev/video0"
			if _, err := os.Stat(videoDevice); err == nil {
				log.Printf("[Camera] Found physical camera at %s", videoDevice)
				fullCmd := fmt.Sprintf(
					"gst-launch-1.0 v4l2src device=%s ! videoconvert ! video/x-raw,format=I420,width=%s,height=%s,framerate=30/1 ! "+
						"tee name=t ! queue max-size-buffers=4 leaky=downstream ! vp8enc error-resilient=1 deadline=1 keyframe-max-dist=30 cpu-used=5 ! \"video/x-vp8\" ! queue ! avmux_ivf ! tcpclientsink host=127.0.0.1 port=5600 "+
						"t. ! queue max-size-buffers=4 leaky=downstream ! videoscale ! \"video/x-raw,width=320,height=240\" ! jpegenc ! multipartmux boundary=vyomboundary ! tcpserversink host=127.0.0.1 port=8081 sync=false",
					videoDevice, width, height,
				)
				log.Println("[Camera] Starting Pipeline Step: " + fullCmd)
				cmd = exec.Command("sh", "-c", fullCmd)
				cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			} else {
				log.Println("[Camera] No physical camera found. Using Test Source (Snow)...")
				// Fallback to snow immediately if no device
				forceTestPattern = true
				continue // Restart loop to hit the else block below (or handle here, but cleaner to restart)
			}
		}

		// If we decided to use test pattern (or fallback triggered)
		if forceTestPattern {
			log.Println("[Camera] Using Test Source (Snow)...")
			fullCmd := fmt.Sprintf(
				"gst-launch-1.0 videotestsrc is-live=true pattern=snow ! videoconvert ! video/x-raw,format=I420,width=%s,height=%s,framerate=30/1 ! "+
					"tee name=t ! queue max-size-buffers=4 leaky=downstream ! vp8enc error-resilient=1 deadline=1 keyframe-max-dist=30 cpu-used=5 ! \"video/x-vp8\" ! queue ! avmux_ivf ! tcpclientsink host=127.0.0.1 port=5600 "+
					"t. ! queue max-size-buffers=4 leaky=downstream ! videoscale ! \"video/x-raw,width=320,height=240\" ! jpegenc ! multipartmux boundary=vyomboundary ! tcpserversink host=127.0.0.1 port=8081 sync=false",
				width, height,
			)
			log.Println("[Camera] Starting Pipeline Step: " + fullCmd)
			cmd = exec.Command("sh", "-c", fullCmd)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		}

		// Start Process
		if err := cmd.Start(); err != nil {
			log.Printf("[Camera] Failed to start pipeline: %v. Retrying in 2s...", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Wait
		err := cmd.Wait()
		log.Printf("[Camera] Pipeline exited: %v. Restarting...", err)

		// Crash Logic
		if time.Since(lastCrashTime) < 30*time.Second {
			crashCount++
		} else {
			crashCount = 1 // Reset
		}
		lastCrashTime = time.Now()

		if crashCount >= 3 && !forceTestPattern {
			log.Println("[Camera] 🚨 Too many crashes! Falling back to Test Pattern (Snow) for stability.")
			forceTestPattern = true
		}

		log.Println("[Camera] Restarting pipeline in 2 seconds...")
		time.Sleep(2 * time.Second)
	}
}
