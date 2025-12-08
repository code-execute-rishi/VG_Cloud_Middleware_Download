package models

import "time"

type Collaborator struct {
    ID      string    `json:"id"`
    Email   string    `json:"email"`
    AddedAt time.Time `json:"added_at"`
}

type AddCollaboratorRequest struct {
    Email string `json:"email"`
}