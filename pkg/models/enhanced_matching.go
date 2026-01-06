package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// UserProfile represents enhanced user profile for matching
type UserProfile struct {
	ID                     string    `json:"id" db:"id"`
	UserID                 string    `json:"user_id" db:"user_id"`
	Schedule               Schedule  `json:"schedule" db:"schedule"`
	CurrentGroupSize       int       `json:"current_group_size" db:"current_group_size"`
	IsAvailableForMatching bool      `json:"is_available_for_matching" db:"is_available_for_matching"`
	LastActive             time.Time `json:"last_active" db:"last_active"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`

	// Joined data
	User        *User                    `json:"user,omitempty"`
	Preferences *UserMatchingPreferences `json:"preferences,omitempty"`
}

// Schedule represents user schedule information
type Schedule struct {
	DepartureTime      string   `json:"departure_time"`      // "08:30"
	Frequency          string   `json:"frequency"`           // "daily" | "weekly" | "custom"
	FlexibilityMinutes int      `json:"flexibility_minutes"` // 30
	DaysOfWeek         []string `json:"days_of_week"`        // ["monday", "tuesday", ...]
}

// MatchScore represents detailed compatibility scoring
type MatchScore struct {
	ID                     string    `json:"id" db:"id"`
	UserID                 string    `json:"user_id" db:"user_id"`
	PotentialMatchID       string    `json:"potential_match_id" db:"potential_match_id"`
	TotalScore             float64   `json:"total_score" db:"total_score"`
	LocationScore          float64   `json:"location_score" db:"location_score"`
	ScheduleScore          float64   `json:"schedule_score" db:"schedule_score"`
	DemographicScore       float64   `json:"demographic_score" db:"demographic_score"`
	RouteScore             float64   `json:"route_score" db:"route_score"`
	GroupSizeScore         float64   `json:"group_size_score" db:"group_size_score"`
	RoleCompatibilityScore float64   `json:"role_compatibility_score" db:"role_compatibility_score"`
	MatchReasons           []string  `json:"match_reasons" db:"match_reasons"`
	Dealbreakers           []string  `json:"dealbreakers" db:"dealbreakers"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

// MatchFilters represents filters for matching
type MatchFilters struct {
	MinScore          *float64 `json:"min_score,omitempty"`
	MaxDistance       *float64 `json:"max_distance,omitempty"`
	AgeRanges         []string `json:"age_ranges,omitempty"`
	Genders           []string `json:"genders,omitempty"`
	StudentPreference *string  `json:"student_preference,omitempty"`
	DriverPreference  *string  `json:"driver_preference,omitempty"`
}

// Enhanced PotentialMatch with detailed scoring
type EnhancedPotentialMatch struct {
	*PotentialMatch
	MatchScore   *MatchScore  `json:"match_score,omitempty"`
	User1Profile *UserProfile `json:"user1_profile,omitempty"`
	User2Profile *UserProfile `json:"user2_profile,omitempty"`
}

// Value implements driver.Valuer for JSON serialization
func (s Schedule) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implements sql.Scanner for JSON deserialization
func (s *Schedule) Scan(value interface{}) error {
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

	return json.Unmarshal(bytes, s)
}

// Value implements driver.Valuer for JSON serialization
func (ms MatchScore) Value() (driver.Value, error) {
	return json.Marshal(ms)
}

// Scan implements sql.Scanner for JSON deserialization
func (ms *MatchScore) Scan(value interface{}) error {
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

	return json.Unmarshal(bytes, ms)
}
