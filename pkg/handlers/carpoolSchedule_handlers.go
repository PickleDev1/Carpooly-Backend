package handlers

import (
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
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
}

func NewCarpoolScheduleHandler(scheduleRepo *repository.CarpoolScheduleRepository, carpoolRepo *repository.CarPoolRepository) *CarpoolScheduleHandler {
	return &CarpoolScheduleHandler{
		scheduleRepo: scheduleRepo,
		carpoolRepo:  carpoolRepo,
	}
}

func (h *CarpoolScheduleHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	// Log start of handler with full request details
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Starting CreateSchedule handler\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())

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
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Attempting to create schedule\",\"schedule\":%s}", string(scheduleJSON))

	// Create schedule
	if err := h.scheduleRepo.CreateCarpoolSchedule(r.Context(), schedule); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create schedule\",\"error\":\"%v\",\"schedule\":%s}",
			err, string(scheduleJSON))
		http.Error(w, "Failed to create schedule", http.StatusInternalServerError)
		return
	}

	// Log success and response
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Schedule created successfully\",\"schedule_id\":\"%s\",\"carpool_id\":\"%s\"}",
		schedule.ID, schedule.CarpoolID)

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
