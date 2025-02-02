package models

import (
	"time"

	"github.com/google/uuid"
)

// CarpoolSchedule represents a scheduled carpool ride
type CarpoolSchedule struct {
	ID           uuid.UUID  `json:"id"`
	CarpoolID    uuid.UUID  `json:"carpool_id"`
	ScheduleType string     `json:"schedule_type"` // one_time, daily, weekly
	StartDate    time.Time  `json:"start_date"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	DayOfWeek    *int       `json:"day_of_week,omitempty"` // 0 = Sunday, for weekly events
	StartTime    time.Time  `json:"start_time"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// CreateScheduleRequest represents the request body for creating a new schedule
type CreateScheduleRequest struct {
	CarpoolID    string     `json:"carpool_id"`
	ScheduleType string     `json:"schedule_type"`
	StartDate    time.Time  `json:"start_date"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	DayOfWeek    *int       `json:"day_of_week,omitempty"`
	StartTime    time.Time  `json:"start_time"`
}
