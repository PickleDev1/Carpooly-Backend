package handlers

import (
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"encoding/json"
	"io"
	"log"
	"net/http"
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
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Starting CreateSchedule handler\",\"method\":\"%s\",\"url\":\"%s\",\"headers\":%v}",
		r.Method, r.URL.String(), r.Header)

	// Log auth header specifically
	authHeader := r.Header.Get("Authorization")
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Auth header\",\"auth\":\"%s\"}", authHeader)

	// Log request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to read request body\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Received request body\",\"body\":%s}", string(body))
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	var req models.CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to decode request body\",\"error\":\"%v\",\"body\":%s}", err, string(body))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Decoded request\",\"carpool_id\":\"%s\",\"schedule_type\":\"%s\"}",
		req.CarpoolID, req.ScheduleType)

	// Validate schedule type
	if req.ScheduleType != "one_time" && req.ScheduleType != "daily" && req.ScheduleType != "weekly" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid schedule type\",\"type\":\"%s\"}", req.ScheduleType)
		http.Error(w, "Invalid schedule type. Must be one of: one_time, daily, weekly", http.StatusBadRequest)
		return
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

	if err := h.scheduleRepo.CreateCarpoolSchedule(r.Context(), schedule); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create schedule\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to create schedule", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
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
