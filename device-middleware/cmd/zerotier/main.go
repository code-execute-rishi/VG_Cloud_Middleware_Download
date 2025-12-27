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
	State     string `json:"state"`
	NetworkID string `json:"network_id"`
	IPAddress string `json:"ip_address"`
	LastError string `json:"last_error"`
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
		log.Printf("[ZeroTier] Direct execution failed (%v). Trying sudo...", err)
		sudoArgs := append([]string{"-n", ztPath}, args...)
		cmdSudo := exec.Command("sudo", sudoArgs...)
		outSudo, errSudo := cmdSudo.CombinedOutput()

		if errSudo != nil {
			log.Printf("[ZeroTier] Sudo execution also failed: %v", errSudo)
			return string(out), err
		}
		return string(outSudo), nil
	}

	return string(out), err
}

func main() {
	log.Println("🚀 Starting Vyom ZeroTier Service...")

	client := &http.Client{Timeout: 5 * time.Second}

	for {
		status := checkZeroTier()

		// 2. Report Status to vyom-api
		payload, _ := json.Marshal(status)
		resp, err := client.Post(API_URL+"/internal/zerotier", "application/json", bytes.NewBuffer(payload))
		if err != nil {
			log.Printf("⚠️ Failed to report status to API: %v", err)
		} else {
			resp.Body.Close()
		}

		time.Sleep(10 * time.Second)
	}
}

func checkZeroTier() ZeroTierStatus {
	var status ZeroTierStatus
	status.State = "Unknown"

	// Use JSON output for robustness
	out, err := runZeroTier("-j", "listnetworks")
	if err != nil {
		status.LastError = fmt.Sprintf("CLI Error: %v", err)
		return status
	}

	// Output is a JSON array of objects
	type ZTNetwork struct {
		ID            string   `json:"nwid"`
		Name          string   `json:"name"`
		Status        string   `json:"status"`
		Type          string   `json:"type"`
		AssignedAddrs []string `json:"assignedAddresses"`
	}

	var networks []ZTNetwork
	if err := json.Unmarshal([]byte(out), &networks); err != nil {
		log.Printf("[ZeroTier] JSON Parse Error: %v. Raw: %s", err, out)
		// Fallback to text parsing if JSON is invalid
		status.LastError = "JSON Parse Error"
		return status
	}

	if len(networks) > 0 {
		nw := networks[0]
		status.NetworkID = nw.ID
		status.State = nw.Status
		if len(nw.AssignedAddrs) > 0 {
			// e.g. "172.25.x.x/16"
			status.IPAddress = strings.Split(nw.AssignedAddrs[0], "/")[0]
		}
		log.Printf("[ZeroTier] Parsed (JSON): ID=%s State=%s IP=%s", status.NetworkID, status.State, status.IPAddress)
	} else {
		status.State = "Not Configured"
	}

	if status.State == "OK" {
		status.State = "Connected"
	}
	log.Printf("[ZeroTier] Final Report: ID=%s State=%s IP=%s", status.NetworkID, status.State, status.IPAddress)

	return status
}
