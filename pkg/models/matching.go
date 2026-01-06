package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// UserMatchingPreferences represents user preferences for carpool matching
type UserMatchingPreferences struct {
	ID                         string                 `json:"id,omitempty" db:"id"` // Surrogate primary key (added in Phase 2)
	UserID                     string                 `json:"user_id" db:"user_id"`
	MaxDetourMinutes           int                    `json:"max_detour_minutes" db:"max_detour_minutes"`
	PreferredGroupSize         int                    `json:"preferred_group_size" db:"preferred_group_size"`
	DriverPreference           string                 `json:"driver_preference" db:"driver_preference"`
	ScheduleFlexibilityMinutes int                    `json:"schedule_flexibility_minutes" db:"schedule_flexibility_minutes"`
	MaxPickupDistanceMiles     float64                `json:"max_pickup_distance_miles" db:"max_pickup_distance_miles"`
	MinCompatibilityScore      float64                `json:"min_compatibility_score" db:"min_compatibility_score"`
	NotificationPreferences    NotificationPrefs      `json:"notification_preferences" db:"notification_preferences"`
	UserDemographics           UserDemographics       `json:"user_demographics" db:"user_demographics"`
	DemographicPreferences     DemographicPreferences `json:"demographic_preferences" db:"demographic_preferences"`
	IsActive                   bool                   `json:"is_active" db:"is_active"`
	// New destination and schedule fields
	DestinationLatitude  *float64 `json:"destination_latitude" db:"destination_latitude"`
	DestinationLongitude *float64 `json:"destination_longitude" db:"destination_longitude"`
	ArrivalTime          *string  `json:"arrival_time" db:"arrival_time"` // Format: "HH:MM:SS"
	CommuteDays          []string `json:"commute_days" db:"commute_days"` // Values: mon,tue,wed,thu,fri,sat,sun
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// UserDemographics represents user demographic information
type UserDemographics struct {
	AgeRange      string `json:"age_range"`      // '18-25' | '26-35' | '36-45' | '46-55' | '56-65' | '65+'
	Gender        string `json:"gender"`         // 'male' | 'female' | 'non-binary' | 'prefer_not_to_say'
	Occupation    string `json:"occupation"`     // required string
	StudentStatus string `json:"student_status"` // 'undergraduate' | 'graduate' | 'not_student'
	Company       string `json:"company"`        // optional string
}

// DemographicPreferences represents user preferences for matching demographics
type DemographicPreferences struct {
	AgePreferences        []string `json:"age_preferences"`        // array of age ranges (at least one required)
	GenderPreferences     []string `json:"gender_preferences"`     // array of genders (at least one required)
	StudentPreference     string   `json:"student_preference"`     // 'students_only' | 'professionals_only' | 'both'
	OccupationPreferences []string `json:"occupation_preferences"` // optional array of occupation preferences
}

// PotentialMatch represents a potential match between two users
type PotentialMatch struct {
	ID                       string       `json:"id" db:"id"`
	User1ID                  string       `json:"user1_id" db:"user1_id"`
	User2ID                  string       `json:"user2_id" db:"user2_id"`
	CompatibilityScore       float64      `json:"compatibility_score" db:"compatibility_score"`
	RouteOverlapPercentage   *float64     `json:"route_overlap_percentage" db:"route_overlap_percentage"`
	TotalDistanceMiles       *float64     `json:"total_distance_miles" db:"total_distance_miles"`
	EstimatedSavingsPerMonth *float64     `json:"estimated_savings_per_month" db:"estimated_savings_per_month"`
	MatchReasons             MatchReasons `json:"match_reasons" db:"match_reasons"`
	Status                   string       `json:"status" db:"status"`
	ExpiresAt                time.Time    `json:"expires_at" db:"expires_at"`
	CreatedAt                time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time    `json:"updated_at" db:"updated_at"`

	// Joined data
	User1 *User `json:"user1,omitempty"`
	User2 *User `json:"user2,omitempty"`
}

// MatchRequest represents a carpool request between matched users
type MatchRequest struct {
	ID                   string    `json:"id" db:"id"`
	FromUserID           string    `json:"from_user_id" db:"from_user_id"`
	ToUserID             string    `json:"to_user_id" db:"to_user_id"`
	PotentialMatchID     string    `json:"potential_match_id" db:"potential_match_id"`
	Message              *string   `json:"message" db:"message"`
	PreferredCarpoolSize int       `json:"preferred_carpool_size" db:"preferred_carpool_size"`
	CarpoolName          string    `json:"carpool_name" db:"carpool_name"`
	Status               string    `json:"status" db:"status"`
	ExpiresAt            time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`

	// Joined data
	FromUser       *User           `json:"from_user,omitempty"`
	ToUser         *User           `json:"to_user,omitempty"`
	PotentialMatch *PotentialMatch `json:"potential_match,omitempty"`
}

// MatchRequestPayload represents the request body for creating a match request
type MatchRequestPayload struct {
	PotentialMatchID     string `json:"potential_match_id" binding:"required"`
	ToUserID             string `json:"to_user_id" binding:"required"`
	Message              string `json:"message,omitempty"`
	PreferredCarpoolSize int    `json:"preferred_carpool_size" binding:"required"`
	CarpoolName          string `json:"carpool_name" binding:"required"`
}

// MatchRequestsResponse represents the response for getting user requests
type MatchRequestsResponse struct {
	Incoming []MatchRequest `json:"incoming"`
	Outgoing []MatchRequest `json:"outgoing"`
}

// UpdateMatchRequestPayload represents the request body for updating a match request
type UpdateMatchRequestPayload struct {
	Status string `json:"status" binding:"required"`
}

// MatchingSession represents an active matching session for a user
type MatchingSession struct {
	ID                   string     `json:"id" db:"id"`
	UserID               string     `json:"user_id" db:"user_id"`
	Status               string     `json:"status" db:"status"`
	LastMatchGeneratedAt *time.Time `json:"last_match_generated_at" db:"last_match_generated_at"`
	ExpiresAt            time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// NotificationPrefs represents notification preferences
type NotificationPrefs struct {
	Email bool `json:"email"`
	Push  bool `json:"push"`
	SMS   bool `json:"sms"`
}

// MatchReasons represents reasons why users were matched
type MatchReasons []string

// Value implements driver.Valuer for JSON serialization
func (nr NotificationPrefs) Value() (driver.Value, error) {
	return json.Marshal(nr)
}

// Scan implements sql.Scanner for JSON deserialization
func (nr *NotificationPrefs) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}

	return json.Unmarshal(bytes, nr)
}

// Value implements driver.Valuer for JSON serialization
func (mr MatchReasons) Value() (driver.Value, error) {
	return json.Marshal(mr)
}

// Scan implements sql.Scanner for JSON deserialization
func (mr *MatchReasons) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}

	return json.Unmarshal(bytes, mr)
}

// Value implements driver.Valuer for JSON serialization
func (ud UserDemographics) Value() (driver.Value, error) {
	return json.Marshal(ud)
}

// Scan implements sql.Scanner for JSON deserialization
func (ud *UserDemographics) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}

	return json.Unmarshal(bytes, ud)
}

// Value implements driver.Valuer for JSON serialization
func (dp DemographicPreferences) Value() (driver.Value, error) {
	return json.Marshal(dp)
}

// Scan implements sql.Scanner for JSON deserialization
func (dp *DemographicPreferences) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}

	return json.Unmarshal(bytes, dp)
}

// MatchingRequest represents the request body for finding matches
type MatchingRequest struct {
	MaxResults int `json:"max_results"`
}

// MatchRequestUpdate represents the request body for updating match requests
type MatchRequestUpdate struct {
	Status string `json:"status"`
}

// CompatibilityScore represents the breakdown of compatibility scoring
type CompatibilityScore struct {
	GeographicScore float64 `json:"geographic_score"`
	ScheduleScore   float64 `json:"schedule_score"`
	PreferenceScore float64 `json:"preference_score"`
	TotalScore      float64 `json:"total_score"`
}

// RouteAnalysis represents route compatibility analysis
type RouteAnalysis struct {
	TotalDistanceMiles       float64 `json:"total_distance_miles"`
	RouteOverlapPercentage   float64 `json:"route_overlap_percentage"`
	DetourMinutes            float64 `json:"detour_minutes"`
	EstimatedSavingsPerMonth float64 `json:"estimated_savings_per_month"`
}
