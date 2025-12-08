package utils

import (
	cryptoRand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
)


func GenerateChallenge() string {
	bytes := make([]byte, 32)
	cryptoRand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func RespondWithError(w http.ResponseWriter, code int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}