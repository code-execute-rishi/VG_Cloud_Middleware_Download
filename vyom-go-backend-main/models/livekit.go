package models

type LiveKitTokenRequest struct {
    DeviceID string `json:"device_id"`
}

type LiveKitTokenResponse struct {
    Token    string `json:"token"`
    URL      string `json:"url"`
    RoomName string `json:"room_name"`
}

type RefreshTokenRequest struct {
    DeviceID string `json:"device_id"`
}

type RefreshTokenResponse struct {
    Token    string `json:"token"`
    URL      string `json:"url"`
    RoomName string `json:"room_name"`
}