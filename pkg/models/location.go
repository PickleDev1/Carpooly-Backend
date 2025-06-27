package models

import (
	"time"

	"github.com/google/uuid"
)

// Location represents a user's location during a carpool ride
type Location struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	CarpoolRideID uuid.UUID `json:"carpool_ride_id"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	Timestamp     time.Time `json:"timestamp"`
	CreatedAt     time.Time `json:"created_at"`
}

// LocationUpdateRequest represents the data needed to update a user's location
type UpdateLocationRequest struct {
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp,omitempty"` // optional
}

// LocationSettings represents a user's location sharing preferences
type LocationSettings struct {
	LocationSharingEnabled bool    `json:"location_sharing_enabled"`
	HomeLatitude           float64 `json:"home_latitude,omitempty"`
	HomeLongitude          float64 `json:"home_longitude,omitempty"`
}

// HomeLocationUpdate represents the request to update a user's home location
type HomeLocationUpdate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
