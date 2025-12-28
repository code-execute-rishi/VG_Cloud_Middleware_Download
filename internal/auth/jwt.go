package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

type UserClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func ParseJWT(tokenString string) (*UserClaims, error) {
	// Simple JWT parser (Header.Payload.Signature)
	parts := strings.Split(tokenString, ".")
	if len(parts) < 2 {
		return nil, nil // Not a valid JWT structure
	}

	payloadPart := parts[1]
	// Fix Padding
	if l := len(payloadPart) % 4; l > 0 {
		payloadPart += strings.Repeat("=", 4-l)
	}

	decoded, err := base64.URLEncoding.DecodeString(payloadPart)
	if err != nil {
		return nil, err
	}

	var claims UserClaims
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, err
	}

	return &claims, nil
}
