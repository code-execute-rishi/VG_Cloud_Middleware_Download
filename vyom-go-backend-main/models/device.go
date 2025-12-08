package models

type RegisterRequest struct {
	PublicKey          string            `json:"public_key"`
	PairingCode        int               `json:"pairing_code"`
	NodeID             string            `json:"node_id"`
}

type RegisterResponse struct {
	Message            string            `json:"message"`
	DeviceID           string            `json:"device_id"`
}

type ChallengeRequest struct {
	DeviceID           string            `json:"device_id"`
}

type ChallengeResponse struct {
	Message            string            `json:"message"`
	Challenge          string            `json:"challenge"`
}

type VerifyRequest struct {
	DeviceID           string            `json:"device_id"`
	Signature          string            `json:"signature"`
}

type VerifyResponse struct {
	LivekitToken       string            `json:"livekit_token"`
	LivekitURL         string            `json:"livekit_url"`
	RoomName           string            `json:"room_name"`
	Zerotier           ZerotierConfig    `json:"zerotier"`
}

type CheckClaimRequest struct {
	DeviceID           string            `json:"device_id"`
}

type CheckClaimResponse struct {
	Claim              bool              `json:"claim_status"`
    Message            string            `json:"message"`
}

type UpdateDeviceStatusRequest struct {
    Status string `json:"status"`
}