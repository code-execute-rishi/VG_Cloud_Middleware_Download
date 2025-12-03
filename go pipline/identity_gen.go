func generateDeviceIdentity() (ed25519.PrivateKey, string, string) {
    pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        log.Fatalf("Critical: Key generation failed - %v", err)
    }

    deviceID := hex.EncodeToString(pubKey)


    pairingCode := deviceID[:8]

    fmt.Printf("[IDENTITY] Private Key Secured in Memory.\n")
    fmt.Printf("[IDENTITY] Device ID: %s...\n", pairingCode)
    
    return privKey, deviceID, pairingCode