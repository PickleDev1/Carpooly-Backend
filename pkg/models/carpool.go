package models

import (
	"time"

	"github.com/google/uuid"
)

// Carpool represents a carpool group
type Carpool struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	CreatorID          uuid.UUID `json:"creator_id" db:"creator_id"`
	CarpoolName        string    `json:"carpool_name" db:"carpool_name"`
	Status             bool      `json:"status" db:"status"`
	RecurringOption    string    `json:"recurring_option" db:"recurring_option"`
	AvailableSeats     int       `json:"available_seats" db:"available_seats"`
	DestinationAddress string    `json:"destination_address" db:"destination_address"`
	Seats              int       `json:"seats" db:"seats"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// CarpoolMember represents a member of a carpool
type CarpoolMember struct {
	ID        uuid.UUID `json:"id" db:"id"`
	CarpoolID uuid.UUID `json:"carpool_id" db:"carpool_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CarpoolRide represents a specific ride instance
type CarpoolRide struct {
	ID           uuid.UUID `json:"id"`
	CarpoolID    uuid.UUID `json:"carpool_id"`
	DriverID     uuid.UUID `json:"driver_id"`
	StartTime    time.Time `json:"start_time"`
	Status       int       `json:"status"`
	LocationLat  float64   `json:"location_lat"`
	LocationLng  float64   `json:"location_lng"`
	MilesSaved   float64   `json:"miles_saved"`
	Participants []User    `json:"participants"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Stop represents a stop in a carpool ride
type Stop struct {
	ID            uuid.UUID `json:"id" db:"id"`
	CarpoolRideID uuid.UUID `json:"carpool_ride_id" db:"carpool_ride_id"`
	Address       string    `json:"address" db:"address"`
	StopOrder     int       `json:"stop_order" db:"stop_order"`
	StopType      string    `json:"stop_type" db:"stop_type"`
	UserID        uuid.UUID `json:"user_id" db:"user_id"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// CreateCarPoolRequest represents the request structure for creating a new carpool
type CreateCarPoolRequest struct {
	CarpoolName        string `json:"carpool_name"`
	RecurringOption    string `json:"recurring_option"`
	AvailableSeats     int    `json:"available_seats"`
	DestinationAddress string `json:"destination_address"`
	Seats              int    `json:"seats"`
}

// UpdateCarPoolRequest represents the request structure for updating a carpool
type UpdateCarPoolRequest struct {
	CarpoolName        string `json:"carpool_name"`
	Status             bool   `json:"status"`
	RecurringOption    string `json:"recurring_option"`
	AvailableSeats     int    `json:"available_seats"`
	DestinationAddress string `json:"destination_address"`
	Seats              int    `json:"seats"`
}

// CreateRideRequest represents the request structure for creating a new ride
type CreateRideRequest struct {
	CarpoolID uuid.UUID     `json:"carpool_id"`
	Stops     []StopRequest `json:"stops"`
}

// StopRequest represents the request structure for a stop
type StopRequest struct {
	Address   string    `json:"address"`
	StopOrder int       `json:"stop_order"`
	StopType  string    `json:"stop_type"`
	UserID    uuid.UUID `json:"user_id"`
}

type SearchFilters struct {
	StartDate       *time.Time `json:"start_date,omitempty"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	MinSeats        *int       `json:"min_seats,omitempty"`
	MaxDistance     *float64   `json:"max_distance,omitempty"`
	MusicPreference *string    `json:"music_preference,omitempty"`
	SmokingAllowed  *bool      `json:"smoking_allowed,omitempty"`
	PetsAllowed     *bool      `json:"pets_allowed,omitempty"`
}

// CreateCarPoolMemberRequest represents the request structure for adding a member to a carpool
type CreateCarPoolMemberRequest struct {
	CarpoolID uuid.UUID `json:"carpool_id"`
	UserID    uuid.UUID `json:"user_id"`
}

// AddCarpoolMemberRequest represents the request structure for adding a member to a carpool via API
// Used for both add-to-members and add-to-future-rides endpoints
type AddCarpoolMemberRequest struct {
	UserID string `json:"user_id"`
}
