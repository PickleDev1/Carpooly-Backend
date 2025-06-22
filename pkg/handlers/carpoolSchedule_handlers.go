package handlers

import (
	"car-backend/middleware"
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"bytes"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CarpoolScheduleHandler struct {
	scheduleRepo *repository.CarpoolScheduleRepository
	carpoolRepo  *repository.CarPoolRepository
	rideRepo     *repository.CarPoolRideRepository
}

func NewCarpoolScheduleHandler(scheduleRepo *repository.CarpoolScheduleRepository, carpoolRepo *repository.CarPoolRepository, rideRepo *repository.CarPoolRideRepository) *CarpoolScheduleHandler {
	return &CarpoolScheduleHandler{
		scheduleRepo: scheduleRepo,
		carpoolRepo:  carpoolRepo,
		rideRepo:     rideRepo,
	}
}

func (h *CarpoolScheduleHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	// Log start of handler with full request details
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Starting CreateSchedule handler\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())

	// Get timezone from context for validation
	timezoneStr, ok := middleware.GetTimezoneFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"No timezone header provided for schedule creation\"}")
		// Don't fail the request, just log a warning
		timezoneStr = "UTC"
	} else {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Timezone header received for schedule creation\",\"timezone\":\"%s\"}", timezoneStr)
	}

	// Log all request headers
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = v[0]
	}
	headerJSON, _ := json.Marshal(headers)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Request headers\",\"headers\":%s}", string(headerJSON))

	// Read and log request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to read request body\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Raw request body\",\"body\":%s}", string(body))
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	// Parse request
	var req models.CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to decode request body\",\"error\":\"%v\",\"body\":%s}",
			err, string(body))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Log parsed request
	reqJSON, _ := json.Marshal(req)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Parsed request\",\"request\":%s}", string(reqJSON))

	// Normalize schedule type to lowercase
	req.ScheduleType = strings.ToLower(req.ScheduleType)

	// Map recurring options to schedule types
	switch req.ScheduleType {
	case "daily", "DAILY":
		req.ScheduleType = "daily"
	case "weekly", "WEEKLY":
		req.ScheduleType = "weekly"
	case "one_time", "ONE_TIME":
		req.ScheduleType = "one_time"
	default:
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid schedule type\",\"type\":\"%s\"}", req.ScheduleType)
		http.Error(w, "Invalid schedule type. Must be one of: ONE_TIME, DAILY, WEEKLY", http.StatusBadRequest)
		return
	}

	// Validate day_of_week for weekly schedules
	if req.ScheduleType == "weekly" {
		if req.DayOfWeek == nil {
			http.Error(w, "Day of week is required for weekly schedules", http.StatusBadRequest)
			return
		}
		if *req.DayOfWeek < 0 || *req.DayOfWeek > 6 {
			http.Error(w, "Day of week must be between 0 (Sunday) and 6 (Saturday)", http.StatusBadRequest)
			return
		}
	}

	// Parse carpool ID
	carpoolID, err := uuid.Parse(req.CarpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ID format\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			req.CarpoolID, err)
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}

	// Verify carpool exists
	carpool, err := h.carpoolRepo.GetCarPool(r.Context(), carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get carpool\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			carpoolID, err)
		http.Error(w, "Carpool not found", http.StatusNotFound)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Found carpool\",\"carpool_id\":\"%s\",\"carpool_name\":\"%s\"}",
		carpool.ID, carpool.CarpoolName)

	// Create schedule object
	schedule := &models.CarpoolSchedule{
		ID:           uuid.New(),
		CarpoolID:    carpoolID,
		ScheduleType: req.ScheduleType,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		DayOfWeek:    req.DayOfWeek,
		StartTime:    req.StartTime,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Log schedule object before creation
	scheduleJSON, _ := json.Marshal(schedule)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Attempting to create schedule\",\"schedule\":%s,\"timezone\":\"%s\"}", string(scheduleJSON), timezoneStr)

	// Create schedule
	if err := h.scheduleRepo.CreateCarpoolSchedule(r.Context(), schedule); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create schedule\",\"error\":\"%v\",\"schedule\":%s,\"timezone\":\"%s\"}",
			err, string(scheduleJSON), timezoneStr)
		http.Error(w, "Failed to create schedule", http.StatusInternalServerError)
		return
	}

	// Generate rides from schedule
	if err := h.generateRidesFromSchedule(r.Context(), schedule, carpool); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to generate rides from schedule\",\"error\":\"%v\",\"schedule_id\":\"%s\",\"timezone\":\"%s\"}",
			err, schedule.ID, timezoneStr)
		// Don't fail the request, just log the error
		// The schedule was created successfully, rides can be generated later
	}

	// Log success and response
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Schedule created successfully\",\"schedule_id\":\"%s\",\"carpool_id\":\"%s\",\"timezone\":\"%s\"}",
		schedule.ID, schedule.CarpoolID, timezoneStr)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Log response before sending
	responseJSON, _ := json.Marshal(schedule)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Sending response\",\"status\":201,\"body\":%s}", string(responseJSON))

	json.NewEncoder(w).Encode(schedule)
}

func (h *CarpoolScheduleHandler) GetCarpoolSchedules(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	carpoolIDStr := vars["carpoolID"]

	carpoolID, err := uuid.Parse(carpoolIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ID format\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}

	// Verify carpool exists
	_, err = h.carpoolRepo.GetCarPool(r.Context(), carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get carpool\",\"error\":\"%v\"}", err)
		http.Error(w, "Carpool not found", http.StatusNotFound)
		return
	}

	schedules, err := h.scheduleRepo.GetCarpoolSchedules(r.Context(), carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get schedules\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to get schedules", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedules)
}

func (h *CarpoolScheduleHandler) GetScheduleByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID, err := uuid.Parse(vars["scheduleID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid schedule ID format\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	schedule, err := h.scheduleRepo.GetScheduleByID(r.Context(), scheduleID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get schedule\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to get schedule", http.StatusInternalServerError)
		return
	}

	if schedule == nil {
		http.Error(w, "Schedule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedule)
}

func (h *CarpoolScheduleHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	// Log request details with headers
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = v[0]
	}
	headerJSON, _ := json.Marshal(headers)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Starting UpdateSchedule handler\",\"method\":\"%s\",\"url\":\"%s\",\"headers\":%s}",
		r.Method, r.URL.String(), string(headerJSON))

	vars := mux.Vars(r)
	carpoolID, err := uuid.Parse(vars["carpoolID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ID format\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			vars["carpoolID"], err)
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Parsed carpool ID\",\"carpool_id\":\"%s\"}", carpoolID)

	// Verify carpool exists
	carpool, err := h.carpoolRepo.GetCarPool(r.Context(), carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get carpool\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			carpoolID, err)
		http.Error(w, "Carpool not found", http.StatusNotFound)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Found carpool\",\"carpool_id\":\"%s\",\"carpool_name\":\"%s\"}",
		carpool.ID, carpool.CarpoolName)

	// Read and log request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to read request body\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Raw request body\",\"body\":%s}", string(body))
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	var req models.CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to decode request\",\"error\":\"%v\",\"raw_body\":%s}",
			err, string(body))
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Parsed request body\",\"request\":%+v}", req)

	// Update schedule object
	schedule := &models.CarpoolSchedule{
		CarpoolID:    carpoolID,
		ScheduleType: strings.ToLower(req.ScheduleType),
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		DayOfWeek:    req.DayOfWeek,
		StartTime:    req.StartTime,
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Created schedule object\",\"schedule\":%+v}", schedule)

	// Validate schedule type
	switch schedule.ScheduleType {
	case "daily", "weekly", "one_time":
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Valid schedule type\",\"type\":\"%s\"}", schedule.ScheduleType)
	default:
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid schedule type\",\"type\":\"%s\"}", schedule.ScheduleType)
		http.Error(w, "Invalid schedule type. Must be one of: one_time, daily, weekly", http.StatusBadRequest)
		return
	}

	// Validate day_of_week for weekly schedules
	if schedule.ScheduleType == "weekly" {
		if schedule.DayOfWeek == nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Missing day of week for weekly schedule\"}")
			http.Error(w, "Day of week is required for weekly schedules", http.StatusBadRequest)
			return
		}
		if *schedule.DayOfWeek < 0 || *schedule.DayOfWeek > 6 {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid day of week\",\"day\":%d}", *schedule.DayOfWeek)
			http.Error(w, "Day of week must be between 0 (Sunday) and 6 (Saturday)", http.StatusBadRequest)
			return
		}
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Valid day of week\",\"day\":%d}", *schedule.DayOfWeek)
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Attempting to update schedule\",\"carpool_id\":\"%s\"}", carpoolID)
	err = h.scheduleRepo.UpdateScheduleByCarpool(r.Context(), schedule)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update schedule\",\"carpool_id\":\"%s\",\"error\":\"%v\",\"schedule\":%+v}",
			carpoolID, err, schedule)
		http.Error(w, "Failed to update schedule", http.StatusInternalServerError)
		return
	}
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully updated schedule\",\"carpool_id\":\"%s\",\"schedule_id\":\"%s\"}",
		carpoolID, schedule.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	responseJSON, _ := json.Marshal(schedule)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Sending response\",\"body\":%s}", string(responseJSON))
	json.NewEncoder(w).Encode(schedule)
}

// generateRidesFromSchedule creates individual rides based on a carpool schedule
func (h *CarpoolScheduleHandler) generateRidesFromSchedule(ctx context.Context, schedule *models.CarpoolSchedule, carpool *models.Carpool) error {
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Generating rides from schedule\",\"schedule_id\":\"%s\",\"schedule_type\":\"%s\"}",
		schedule.ID, schedule.ScheduleType)

	var rides []*models.CarpoolRide
	currentDate := schedule.StartDate

	// Set end date - if schedule.EndDate is nil, use a reasonable default (e.g., 30 days from start)
	endDate := schedule.StartDate.AddDate(0, 0, 30) // Default to 30 days
	if schedule.EndDate != nil {
		endDate = *schedule.EndDate
	}

	// Generate rides until we reach the end date
	for currentDate.Before(endDate) || currentDate.Equal(endDate) {
		var shouldCreateRide bool

		switch schedule.ScheduleType {
		case "one_time":
			// For one-time schedules, only create one ride on the start date
			shouldCreateRide = currentDate.Equal(schedule.StartDate)
		case "daily":
			// For daily schedules, create a ride every day
			shouldCreateRide = true
		case "weekly":
			// For weekly schedules, create a ride on the specified day of week
			if schedule.DayOfWeek != nil {
				shouldCreateRide = int(currentDate.Weekday()) == *schedule.DayOfWeek
			}
		}

		if shouldCreateRide {
			// Create the ride start time by combining the date with the schedule time
			rideStartTime := time.Date(
				currentDate.Year(), currentDate.Month(), currentDate.Day(),
				schedule.StartTime.Hour(), schedule.StartTime.Minute(), 0, 0,
				schedule.StartTime.Location(),
			)

			// Only create rides that are in the future
			if rideStartTime.After(time.Now()) {
				ride := &models.CarpoolRide{
					ID:        uuid.New(),
					CarpoolID: schedule.CarpoolID,
					StartTime: rideStartTime,
					Status:    0, // Default status (pending)
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				rides = append(rides, ride)
				log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Generated ride\",\"ride_id\":\"%s\",\"start_time\":\"%s\"}",
					ride.ID, ride.StartTime.Format("2006-01-02 15:04:05"))
			}
		}

		// Move to next day
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Generated %d rides from schedule\",\"schedule_id\":\"%s\",\"ride_count\":%d}",
		len(rides), schedule.ID, len(rides))

	// Save rides to database
	for _, ride := range rides {
		if err := h.rideRepo.CreateCarpoolRide(ctx, ride); err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to save ride to database\",\"ride_id\":\"%s\",\"error\":\"%v\"}",
				ride.ID, err)
			// Continue with other rides even if one fails
			continue
		}
		log.Printf("{\"severity\":\"INFO\",\"message\":\"Saved ride to database\",\"ride_id\":\"%s\",\"start_time\":\"%s\"}",
			ride.ID, ride.StartTime.Format("2006-01-02 15:04:05"))
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully generated and saved rides from schedule\",\"schedule_id\":\"%s\",\"total_rides\":%d}",
		schedule.ID, len(rides))

	return nil
}
