package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                     uuid.UUID `json:"id" db:"id"`
	ClerkID                string    `json:"clerk_id" db:"clerk_id"`
	Email                  string    `json:"email" db:"email"`
	Name                   string    `json:"name" db:"name"`
	DisplayName            string    `json:"display_name" db:"display_name"`
	City                   string    `json:"city" db:"city"`
	State                  string    `json:"state" db:"state"`
	LocationSharingEnabled bool      `json:"location_sharing_enabled" db:"location_sharing_enabled"`
	HomeLatitude           float64   `json:"home_latitude" db:"home_latitude"`
	HomeLongitude          float64   `json:"home_longitude" db:"home_longitude"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

type CreateUserRequest struct {
	Email                  string  `json:"email"`
	Name                   string  `json:"name"`
	DisplayName            string  `json:"display_name"`
	City                   string  `json:"city"`
	State                  string  `json:"state"`
	ClerkID                string  `json:"clerk_id"`
	LocationSharingEnabled bool    `json:"location_sharing_enabled"`
	HomeLatitude           float64 `json:"home_latitude"`
	HomeLongitude          float64 `json:"home_longitude"`
}

type UpdateUserProfile struct {
	DisplayName            *string  `json:"display_name,omitempty"`
	City                   *string  `json:"city,omitempty"`
	State                  *string  `json:"state,omitempty"`
	LocationSharingEnabled *bool    `json:"location_sharing_enabled,omitempty"`
	HomeLatitude           *float64 `json:"home_latitude,omitempty"`
	HomeLongitude          *float64 `json:"home_longitude,omitempty"`
}

// UserActivity represents a row in the user_activity table
type UserActivity struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	UserID      uuid.UUID   `json:"user_id" db:"user_id"`
	Type        string      `json:"type" db:"activity_type"`
	RelatedID   *uuid.UUID  `json:"related_id,omitempty" db:"related_id"`
	RelatedType *string     `json:"related_type,omitempty" db:"related_type"`
	Description *string     `json:"description,omitempty" db:"description"`
	Data        interface{} `json:"data,omitempty" db:"data"`
	Timestamp   time.Time   `json:"timestamp" db:"timestamp"`
}
