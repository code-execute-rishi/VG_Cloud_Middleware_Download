package models

type UpdateTelemetryRequest struct {
    Latitude       *float64 `json:"latitude,omitempty"`
    Longitude      *float64 `json:"longitude,omitempty"`
    Altitude       *float64 `json:"altitude,omitempty"`
    Speed          *float64 `json:"speed,omitempty"`
    Heading        *float64 `json:"heading,omitempty"`
    SignalStrength *int     `json:"signal_strength,omitempty"`
    Battery        *int     `json:"battery,omitempty"`
}

type TelemetryUpdate struct {
    DeviceID       string
    Latitude       *float64
    Longitude      *float64
    Altitude       *float64
    Speed          *float64
    Heading        *float64
    SignalStrength *int
    Battery        *int
}