
func performSecureHandshake(deviceID string, privKey ed25519.PrivateKey) string {
    client := &http.Client{Timeout: 5 * time.Second}

    for {
        
        resp, _ := client.Post(AuthURL+"/challenge", "application/json", 
            body{"device_id": deviceID})

        
        if resp.StatusCode == 404 {
           
            sleepDuration := 3*time.Second + time.Duration(rand.Intn(500))*time.Millisecond
            log.Printf("[STATUS] Waiting for user to enter code... (Retrying in %v)", sleepDuration)
            time.Sleep(sleepDuration)
            continue
        }

        var challenge struct{ Nonce string }
        json.NewDecoder(resp.Body).Decode(&challenge)
        
        signature := ed25519.Sign(privKey, []byte(challenge.Nonce))
        

        
        log.Println("[SUCCESS] Identity Verified. Token Received.")
        return token
    }
}