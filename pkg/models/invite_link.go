package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// InviteLink represents a public invite link for a carpool
type InviteLink struct {
	ID          uuid.UUID `json:"id" db:"id"`
	CarpoolID   uuid.UUID `json:"carpool_id" db:"carpool_id"`
	InviteCode  string    `json:"invite_code" db:"invite_code"`
	CreatedBy   uuid.UUID `json:"created_by" db:"created_by"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	ExpiresAt   time.Time `json:"expires_at" db:"expires_at"`
	MaxUses     int       `json:"max_uses" db:"max_uses"`
	CurrentUses int       `json:"current_uses" db:"current_uses"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	// Joined fields for responses
	CarpoolName string         `json:"carpool_name,omitempty"`
	CreatorName sql.NullString `json:"creator_name,omitempty"`
}

// CreateInviteLinkRequest represents the request to create an invite link
type CreateInviteLinkRequest struct {
	ExpiresInDays int `json:"expires_in_days,omitempty"` // Optional, defaults to 30 days
	MaxUses       int `json:"max_uses,omitempty"`        // Optional, defaults to unlimited
}

// JoinInviteLinkRequest represents the request to join via invite link
type JoinInviteLinkRequest struct {
	// No additional fields needed - user is determined from auth token
}

// InviteLinkResponse represents the public invite link response
type InviteLinkResponse struct {
	InviteCode  string    `json:"invite_code"`
	CarpoolName string    `json:"carpool_name"`
	CreatorName string    `json:"creator_name"`
	ExpiresAt   time.Time `json:"expires_at"`
	IsActive    bool      `json:"is_active"`
}
