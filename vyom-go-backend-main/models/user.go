package models

import "time"

type ClaimRequest struct {
	PairingCode int    `json:"pairing_code"`
	Name        string `json:"name"`
}

type ClaimResponse struct {
	Message  string `json:"message"`
	DeviceID string `json:"device_id"`
	Name     string `json:"name"`
}

type Device struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Status         string         `json:"status"`
	LastSeen       *time.Time     `json:"last_seen"`
	Latitude       *float64       `json:"latitude"`
	Longitude      *float64       `json:"longitude"`
	Altitude       *float64       `json:"altitude"`
	Speed          *float64       `json:"speed"`
	Heading        *float64       `json:"heading"`
	SignalStrength *int           `json:"signal_strength"`
	Battery        *int           `json:"battery"`
	Armed          *bool          `json:"armed"`
	FlightMode     *string        `json:"flight_mode"`
	Collaborators  []Collaborator `json:"collaborators"`
}

type DeviceDetail struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Status         string         `json:"status"`
	LastSeen       *time.Time     `json:"last_seen"`
	Latitude       *float64       `json:"latitude"`
	Longitude      *float64       `json:"longitude"`
	Altitude       *float64       `json:"altitude"`
	Speed          *float64       `json:"speed"`
	Heading        *float64       `json:"heading"`
	SignalStrength *int           `json:"signal_strength"`
	Battery        *int           `json:"battery"`
	Armed          *bool          `json:"armed"`
	FlightMode     *string        `json:"flight_mode"`
	Collaborators  []Collaborator `json:"collaborators"`
}
