package models

import (
	"time"

	"github.com/google/uuid"
)

// Invite represents a carpool invitation
type Invite struct {
	ID        uuid.UUID `json:"id" db:"id"`
	FromUser  uuid.UUID `json:"from_user" db:"from_user"`
	ToUser    uuid.UUID `json:"to_user" db:"to_user"`
	CarpoolID uuid.UUID `json:"carpool_id" db:"carpool_id"`
	Message   string    `json:"message" db:"message"`
	Status    int       `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}


// CreateInviteRequest represents the request structure for creating a carpool invitation
type CreateInviteRequest struct {
	ToUser    uuid.UUID `json:"to_user"`
	CarpoolID uuid.UUID `json:"carpool_id"`
	Message   string    `json:"message"`
}

// UpdateInviteRequest represents the request structure for updating an invitation status
type UpdateInviteRequest struct {
	Status int `json:"status"` // e.g., 0: pending, 1: accepted, 2: rejected
}

// Add these constants for invite status
const (
	InviteStatusPending  = 0
	InviteStatusAccepted = 1
	InviteStatusRejected = 2
)

