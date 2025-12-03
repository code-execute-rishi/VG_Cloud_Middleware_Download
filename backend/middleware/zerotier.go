package middleware

import (
	"log"
	"os/exec"
	"strings"
	"time"
)

// JoinZeroTier joins the specified ZeroTier network and waits for approval.
func JoinZeroTier(networkID string) error {
	if networkID == "" {
		log.Println("ZeroTier Network ID not provided. Skipping auto-join.")
		return nil
	}

	log.Printf("Joining ZeroTier Network: %s...", networkID)
	cmd := exec.Command("zerotier-cli", "join", networkID)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Failed to run zerotier-cli join: %v\nOutput: %s", err, string(output))
		return err
	}

	log.Println("Join command sent. Waiting for network approval...")

	// Poll for status
	for {
		cmd := exec.Command("zerotier-cli", "listnetworks")
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Failed to list networks: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		outStr := string(output)
		if strings.Contains(outStr, networkID) {
			if strings.Contains(outStr, "OK") {
				log.Println("ZeroTier Network Status: OK (Connected)")
				return nil
			} else if strings.Contains(outStr, "ACCESS_DENIED") {
				log.Println("Waiting for Admin Approval...")
			} else {
				log.Println("ZeroTier Status: Waiting for connection...")
			}
		} else {
			log.Println("Network ID not found in list. Retrying join...")
			exec.Command("zerotier-cli", "join", networkID).Run()
		}

		time.Sleep(1 * time.Second)
	}
}
