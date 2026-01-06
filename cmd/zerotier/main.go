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
	"time"
)

// --- Configuration ---
const (
	API_URL = "http://localhost:8085"
)

type ZeroTierStatus struct {
	State     string         `json:"state"`
	NetworkID string         `json:"network_id"`
	IPAddress string         `json:"ip_address"`
	LastError string         `json:"last_error"`
	Peers     []ZeroTierPeer `json:"peers"`
}

type ZeroTierPeer struct {
	Address string `json:"address"`
	Version string `json:"version"`
	Latency int    `json:"latency"`
	Role    string `json:"role"`
}

// Reuse the robust helper from the monolithic main.go
func runZeroTier(args ...string) (string, error) {
	ztPath := "/usr/sbin/zerotier-cli"

	// Try executing directly
	cmd := exec.Command(ztPath, args...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), nil
	}

	// Check if already root
	if strings.Contains(err.Error(), "permission denied") || os.Geteuid() != 0 {
		sudoArgs := append([]string{"-n", ztPath}, args...)
		cmdSudo := exec.Command("sudo", sudoArgs...)
		outSudo, errSudo := cmdSudo.CombinedOutput()

		if errSudo != nil {
			log.Printf("[ZeroTier] Sudo execution execution failed: %v", errSudo)
			return string(out), err
		}
		return string(outSudo), nil
	}

	return string(out), err
}

func main() {
	log.Println("🚀 Starting Vyom ZeroTier Service...")

	client := &http.Client{Timeout: 5 * time.Second}
	var desiredNetworkID string

	for {
		// 1. Fetch Config First
		resp, err := client.Get(API_URL + "/api/config/zerotier")
		if err == nil && resp.StatusCode == 200 {
			var config struct {
				NetworkID string `json:"network_id"`
			}
			if json.NewDecoder(resp.Body).Decode(&config) == nil {
				desiredNetworkID = config.NetworkID
			}
			resp.Body.Close()
		}

		// 2. Check Status (Strict Mode)
		status := checkZeroTier(desiredNetworkID)

		// 3. Report Status to vyom-api
		payload, _ := json.Marshal(status)
		postResp, err := client.Post(API_URL+"/internal/zerotier", "application/json", bytes.NewBuffer(payload))
		if err != nil {
			log.Printf("⚠️ Failed to report status to API: %v", err)
		} else {
			postResp.Body.Close()
		}

		// 4. Auto-Join Logic
		if desiredNetworkID != "" && desiredNetworkID != status.NetworkID {
			log.Printf("🔄 ZeroTier Config Change/Mismatch! Joining %s...", desiredNetworkID)
			out, err := runZeroTier("join", desiredNetworkID)
			log.Printf("Join Result: %s (Err: %v)", out, err)
		}

		time.Sleep(10 * time.Second)
	}
}

func checkZeroTier(desiredID string) ZeroTierStatus {
	var status ZeroTierStatus
	status.State = "Unknown"
	status.NetworkID = desiredID // Default to desired
	status.Peers = []ZeroTierPeer{}

	if desiredID == "" {
		status.State = "Not Configured"
		return status
	}

	// 1. Get Network Status
	out, err := runZeroTier("-j", "listnetworks")
	if err != nil {
		status.LastError = fmt.Sprintf("CLI Not Found/Err: %v", err)
		return status
	}

	type ZTNetwork struct {
		ID            string   `json:"nwid"`
		Name          string   `json:"name"`
		Status        string   `json:"status"`
		Type          string   `json:"type"`
		AssignedAddrs []string `json:"assignedAddresses"`
	}

	var networks []ZTNetwork
	if err := json.Unmarshal([]byte(out), &networks); err == nil {
		var fallback *ZTNetwork
		found := false
		for i, nw := range networks {
			if nw.Status == "OK" && fallback == nil {
				fallback = &networks[i]
			}

			if nw.ID == desiredID {
				found = true
				status.NetworkID = nw.ID
				status.State = nw.Status
				if len(nw.AssignedAddrs) > 0 {
					status.IPAddress = strings.Split(nw.AssignedAddrs[0], "/")[0]
				}
				break
			}
		}

		if !found {
			if fallback != nil {
				// Fallback to the first healthy network found
				status.NetworkID = fallback.ID
				status.State = fallback.Status
				if len(fallback.AssignedAddrs) > 0 {
					status.IPAddress = strings.Split(fallback.AssignedAddrs[0], "/")[0]
				}
			} else {
				status.State = "Disconnected"
			}
		} else if status.State == "OK" {
			status.State = "Connected"
		}
	} else {
		status.LastError = "JSON Parse Error (Networks)"
	}

	// 2. Get Peers
	outPeers, err := runZeroTier("-j", "listpeers")
	if err == nil {
		type ZTPeer struct {
			Address string `json:"address"`
			Version string `json:"version"`
			Latency int    `json:"latency"`
			Role    string `json:"role"`
			Paths   []any  `json:"paths"`
		}
		var peers []ZTPeer
		if json.Unmarshal([]byte(outPeers), &peers) == nil {
			for _, p := range peers {
				// Filter out inactive/unreachable peers if desired, or keep all
				// Keeping all for visibility, maybe filter locally.
				status.Peers = append(status.Peers, ZeroTierPeer{
					Address: p.Address,
					Version: p.Version,
					Latency: p.Latency,
					Role:    p.Role,
				})
			}
		}
	}

	log.Printf("[ZeroTier] Report: ID=%s State=%s IP=%s Peers=%d", status.NetworkID, status.State, status.IPAddress, len(status.Peers))

	return status
}
