package handlers

import (
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"car-backend/middleware"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CarPoolHandler struct {
	carpoolRepo  *repository.CarPoolRepository
	userRepo     *repository.UserRepository
	scheduleRepo *repository.CarpoolScheduleRepository // Add this line
}

func NewCarPoolHandler(carpoolRepo *repository.CarPoolRepository, userRepo *repository.UserRepository, scheduleRepo *repository.CarpoolScheduleRepository) *CarPoolHandler {
	return &CarPoolHandler{
		carpoolRepo:  carpoolRepo,
		userRepo:     userRepo,
		scheduleRepo: scheduleRepo, // Add this line
	}
}

func ptrString(s string) *string { return &s }

func (h *CarPoolHandler) CreateCarPool(w http.ResponseWriter, r *http.Request) {
	var req models.CreateCarPoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to decode request: %v\"}", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get userID from context
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized - No user ID in context", http.StatusUnauthorized)
		return
	}
	log.Printf("User ID String: %s", clerkID)
	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	// For local testing, use this hardcoded UUID:
	//userIDStr := "b0337c8a-1eed-4a11-90c8-130016c47d0a"
	//userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to parse UUID: %v\"}", err)
		http.Error(w, "Invalid user ID format", http.StatusInternalServerError)
		return
	}

	// Create carpool object
	// The frontend sends the total number of seats they want
	// We automatically reserve one seat for the creator (driver)
	totalSeats := req.AvailableSeats // This is actually the total seats from frontend
	availableSeatsForOthers := totalSeats - 1
	if availableSeatsForOthers < 0 {
		availableSeatsForOthers = 0
	}
	log.Printf("[DEBUG] Creating carpool: totalSeats=%d, availableSeats=%d", totalSeats, availableSeatsForOthers)

	carpool := &models.Carpool{
		CreatorID:          userID, // Use actual userID from context
		CarpoolName:        req.CarpoolName,
		Status:             false, // Default status
		RecurringOption:    sql.NullString{String: req.RecurringOption, Valid: req.RecurringOption != ""},
		AvailableSeats:     availableSeatsForOthers, // Correct value
		Seats:              totalSeats,
		DestinationAddress: req.DestinationAddress,
	}

	if err := h.carpoolRepo.CreateCarPool(r.Context(), carpool); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create carpool: %v\"}", err)
		http.Error(w, "Failed to create carpool", http.StatusInternalServerError)
		return
	}

	// Add activity for carpool creation
	activity := &models.UserActivity{
		UserID:      carpool.CreatorID,
		Type:        "carpool_created",
		RelatedID:   &carpool.ID,
		RelatedType: ptrString("carpool"),
		Description: ptrString("Created carpool: " + carpool.CarpoolName),
		Data:        carpool,
		Timestamp:   time.Now(),
	}
	_ = h.userRepo.AddUserActivity(r.Context(), activity)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(carpool)
}

func (h *CarPoolHandler) GetCarPool(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)
	carpoolID, err := uuid.Parse(params["id"])
	if err != nil {
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}

	carpool, err := h.carpoolRepo.GetCarPool(context.Background(), carpoolID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Carpool not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get carpool: %v", err), http.StatusInternalServerError)
		return
	}

	// Fetch the first schedule (if any) for this carpool
	schedules, err := h.scheduleRepo.GetCarpoolSchedules(r.Context(), carpool.ID)
	var scheduleObj *struct {
		StartDate string `json:"start_date"`
		StartTime string `json:"start_time"`
	}
	if err == nil && len(schedules) > 0 {
		first := schedules[0]
		scheduleObj = &struct {
			StartDate string `json:"start_date"`
			StartTime string `json:"start_time"`
		}{
			StartDate: first.StartDate.Format("2006-01-02"),
			StartTime: first.StartTime.In(time.Local).Format("15:04"),
		}
	}

	// Build response with schedule
	type carpoolWithSchedule struct {
		models.Carpool
		Schedule *struct {
			StartDate string `json:"start_date"`
			StartTime string `json:"start_time"`
		} `json:"schedule,omitempty"`
	}
	response := carpoolWithSchedule{
		Carpool:  *carpool,
		Schedule: scheduleObj,
	}

	json.NewEncoder(w).Encode(response)
}

func (h *CarPoolHandler) UpdateCarPool(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("{\"severity\":\"INFO\",\"message\":\"UpdateCarPool called\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())

	w.Header().Set("Content-Type", "application/json")

	carpoolID := mux.Vars(r)["id"]
	var req models.UpdateCarPoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to decode request body\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ID format\",\"carpool_id\":\"%s\",\"error\":\"%v\"}", carpoolID, err)
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}

	// Verify carpool exists and user has permission to update it
	carpool, err := h.carpoolRepo.GetCarPool(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("{\"severity\":\"INFO\",\"message\":\"Carpool not found\",\"carpool_id\":\"%s\"}", id)
			http.Error(w, "Carpool not found", http.StatusNotFound)
			return
		}
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get carpool\",\"carpool_id\":\"%s\",\"error\":\"%v\"}", id, err)
		http.Error(w, "Failed to get carpool", http.StatusInternalServerError)
		return
	}

	// Get user ID from context to verify ownership
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"No clerk ID in context\",\"carpool_id\":\"%s\"}", id)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", clerkID, err)
		http.Error(w, "Failed to verify user", http.StatusInternalServerError)
		return
	}

	// Verify that the user is the creator of the carpool
	if carpool.CreatorID != userID {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"User not authorized to update carpool\",\"carpool_id\":\"%s\",\"user_id\":\"%s\",\"creator_id\":\"%s\"}",
			id, userID, carpool.CreatorID)
		http.Error(w, "Not authorized to update this carpool", http.StatusForbidden)
		return
	}

	// Log the update request
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Updating carpool\",\"carpool_id\":\"%s\",\"user_id\":\"%s\",\"requested_seats\":%d}",
		id, userID, req.AvailableSeats)

	err = h.carpoolRepo.UpdateCarPool(r.Context(), id, &req)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update carpool\",\"carpool_id\":\"%s\",\"error\":\"%v\"}", id, err)
		http.Error(w, "Failed to update carpool", http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully updated carpool\",\"carpool_id\":\"%s\",\"duration_ms\":%d}",
		id, duration.Milliseconds())
	w.WriteHeader(http.StatusOK)
}

func (h *CarPoolHandler) DeleteCarPool(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("{\"severity\":\"INFO\",\"message\":\"DeleteCarPool called\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())

	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)
	carpoolIDStr := params["id"]
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Attempting to delete carpool\",\"carpool_id\":\"%s\"}", carpoolIDStr)

	carpoolID, err := uuid.Parse(carpoolIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ID format\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			carpoolIDStr, err)
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}

	// First verify if the carpool exists and get its details
	carpool, err := h.carpoolRepo.GetCarPool(r.Context(), carpoolID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("{\"severity\":\"INFO\",\"message\":\"Carpool not found\",\"carpool_id\":\"%s\"}", carpoolID)
			http.Error(w, "Carpool not found", http.StatusNotFound)
			return
		}
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to verify carpool existence\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			carpoolID, err)
		http.Error(w, "Failed to verify carpool", http.StatusInternalServerError)
		return
	}

	// Get user ID from context to verify ownership
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"No clerk ID in context\",\"carpool_id\":\"%s\"}", carpoolID)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Convert clerk ID to user ID
	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}",
			clerkID, err)
		http.Error(w, "Failed to verify user", http.StatusInternalServerError)
		return
	}

	// Verify that the user is the creator of the carpool
	if carpool.CreatorID != userID {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"User not authorized to delete carpool\",\"carpool_id\":\"%s\",\"user_id\":\"%s\",\"creator_id\":\"%s\"}",
			carpoolID, userID, carpool.CreatorID)
		http.Error(w, "Not authorized to delete this carpool", http.StatusForbidden)
		return
	}

	// Delete the carpool
	err = h.carpoolRepo.DeleteCarPool(r.Context(), carpoolID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("{\"severity\":\"INFO\",\"message\":\"Carpool not found during deletion\",\"carpool_id\":\"%s\"}", carpoolID)
			http.Error(w, "Carpool not found", http.StatusNotFound)
			return
		}
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to delete carpool\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			carpoolID, err)
		http.Error(w, fmt.Sprintf("Failed to delete carpool: %v", err), http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully deleted carpool\",\"carpool_id\":\"%s\",\"duration_ms\":%d}",
		carpoolID, duration.Milliseconds())
	w.WriteHeader(http.StatusNoContent)
}

func (h *CarPoolHandler) SearchCarPools(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement
}

func (h *CarPoolHandler) GetCreatorCarpools(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	creatorID := vars["creatorID"]
	if creatorID == "" {
		http.Error(w, "Creator ID is required", http.StatusBadRequest)
		return
	}

	carpools, err := h.carpoolRepo.GetCarpoolsByCreatorID(r.Context(), creatorID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user carpools: %v\"}", err)
		http.Error(w, "Failed to get carpools", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(carpools)
}

func (h *CarPoolHandler) GetUserCarpools(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[GetUserCarpools] PANIC: %v", r)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()
	log.Printf("[GetUserCarpools] TOP OF HANDLER - Handler entered")
	vars := mux.Vars(r)
	userUUIDStr := vars["userID"]
	log.Printf("[GetUserCarpools] userID param from URL: %v", userUUIDStr)
	if userUUIDStr == "" {
		log.Printf("[GetUserCarpools] ERROR: userUUID is empty")
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	var userUUID uuid.UUID
	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		// Not a UUID, try as Clerk ID
		log.Printf("[GetUserCarpools] userID is not a UUID, trying as Clerk ID: %s", userUUIDStr)
		userUUID, err = h.userRepo.GetUserIDByClerkID(r.Context(), userUUIDStr)
		if err != nil {
			log.Printf("[GetUserCarpools] ERROR: Could not resolve Clerk ID %s to UUID: %v", userUUIDStr, err)
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		log.Printf("[GetUserCarpools] Clerk ID %s resolved to UUID %s", userUUIDStr, userUUID)
	}

	carpools, err := h.carpoolRepo.GetUserCarpools(r.Context(), userUUID)
	if err != nil {
		log.Printf("[GetUserCarpools] ERROR: Failed to get user carpools for userID %s: %v", userUUID, err)
		http.Error(w, "Failed to get user carpools", http.StatusInternalServerError)
		return
	}
	log.Printf("[GetUserCarpools] Retrieved %d carpools for userID: %s", len(carpools), userUUID)

	// For each carpool, fetch the first schedule (if any) and add it to the response
	type carpoolWithSchedule struct {
		models.Carpool
		Schedule *struct {
			StartDate string `json:"start_date"`
			StartTime string `json:"start_time"`
		} `json:"schedule,omitempty"`
	}

	var response []carpoolWithSchedule
	for i, carpool := range carpools {
		log.Printf("[GetUserCarpools] Carpool %d: ID=%s, CreatorID=%s, Name=%s", i, carpool.ID, carpool.CreatorID, carpool.CarpoolName)
		schedules, err := h.scheduleRepo.GetCarpoolSchedules(r.Context(), carpool.ID)
		var scheduleObj *struct {
			StartDate string `json:"start_date"`
			StartTime string `json:"start_time"`
		}
		if err == nil && len(schedules) > 0 {
			first := schedules[0]
			scheduleObj = &struct {
				StartDate string `json:"start_date"`
				StartTime string `json:"start_time"`
			}{
				StartDate: first.StartDate.Format("2006-01-02"),
				StartTime: first.StartTime.In(time.Local).Format("15:04"),
			}
		}
		response = append(response, carpoolWithSchedule{
			Carpool:  carpool,
			Schedule: scheduleObj,
		})
	}

	log.Printf("[GetUserCarpools] Sending response for userID: %s", userUUID)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[GetUserCarpools] ERROR: Failed to encode response for userID %s: %v", userUUID, err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	log.Printf("[GetUserCarpools] Handler completed for userID: %s", userUUIDStr)
}

func (h *CarPoolHandler) GetCarpoolMembers(w http.ResponseWriter, r *http.Request) {
	// Log request details
	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetCarpoolMembers called\",\"method\":\"%s\",\"url\":\"%s\"}",
		r.Method, r.URL.String())

	vars := mux.Vars(r)
	carpoolID, err := uuid.Parse(vars["carpoolID"])
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

	members, err := h.carpoolRepo.GetCarpoolMembers(r.Context(), carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get carpool members\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to get carpool members", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

// AddCarpoolMemberAPI handles adding a user to carpool_members
func (h *CarPoolHandler) AddCarpoolMemberAPI(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("{\"severity\":\"INFO\",\"message\":\"AddCarpoolMemberAPI called\",\"method\":\"%s\",\"url\":\"%s\"}", r.Method, r.URL.String())

	vars := mux.Vars(r)
	carpoolIDStr := vars["carpoolID"]
	carpoolID, err := uuid.Parse(carpoolIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ID format\",\"carpool_id_str\":\"%s\",\"error\":\"%v\"}", carpoolIDStr, err)
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Parsed carpool ID\",\"carpool_id\":\"%s\"}", carpoolID)

	var req models.AddCarpoolMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to decode request body\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid user ID format\",\"user_id_str\":\"%s\",\"error\":\"%v\"}", req.UserID, err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Adding member to carpool\",\"carpool_id\":\"%s\",\"user_id\":\"%s\"}", carpoolID, userID)

	// 1. Add user to carpool_members and decrement available seats
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Step 1: Adding user to carpool_members table\",\"carpool_id\":\"%s\",\"user_id\":\"%s\"}", carpoolID, userID)
	err = h.carpoolRepo.AddCarpoolMemberByAPI(r.Context(), carpoolID, userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to add user to carpool_members\",\"carpool_id\":\"%s\",\"user_id\":\"%s\",\"error\":\"%v\"}", carpoolID, userID, err)
		http.Error(w, "Failed to add member", http.StatusInternalServerError)
		return
	}
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully added user to carpool_members\",\"carpool_id\":\"%s\",\"user_id\":\"%s\"}", carpoolID, userID)

	// 2. Add user to all rides (both past and future) for this carpool
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Step 2: Adding user to all rides\",\"carpool_id\":\"%s\",\"user_id\":\"%s\"}", carpoolID, userID)
	err = h.carpoolRepo.AddUserToFutureRides(r.Context(), carpoolID, userID)
	if err != nil {
		// Log the error but don't fail the request since the member was already added
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"Failed to add user to rides\",\"carpool_id\":\"%s\",\"user_id\":\"%s\",\"error\":\"%v\"}", carpoolID, userID, err)
	} else {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully added user to all rides\",\"carpool_id\":\"%s\",\"user_id\":\"%s\"}", carpoolID, userID)
	}

	duration := time.Since(startTime)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"AddCarpoolMemberAPI completed\",\"carpool_id\":\"%s\",\"user_id\":\"%s\",\"duration_ms\":%d}", carpoolID, userID, duration.Milliseconds())

	w.WriteHeader(http.StatusNoContent)
}

// AddUserToAllRidesAPI handles adding a user to all rides' participants (past and future)
func (h *CarPoolHandler) AddUserToFutureRidesAPI(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	carpoolIDStr := vars["carpoolID"]
	carpoolID, err := uuid.Parse(carpoolIDStr)
	if err != nil {
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}
	var req models.AddCarpoolMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	err = h.carpoolRepo.AddUserToFutureRides(r.Context(), carpoolID, userID)
	if err != nil {
		http.Error(w, "Failed to add user to future rides", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetCarpoolCreator returns the creator ID of a carpool
func (h *CarPoolHandler) GetCarpoolCreator(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetCarpoolCreator called\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())

	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context for logging
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"No clerk ID in context for GetCarpoolCreator\"}")
	} else {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetCarpoolCreator request from user\",\"clerk_id\":\"%s\"}", clerkID)
	}

	params := mux.Vars(r)
	carpoolIDStr := params["id"]
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Parsing carpool ID\",\"carpool_id_str\":\"%s\"}", carpoolIDStr)

	carpoolID, err := uuid.Parse(carpoolIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ID format\",\"carpool_id_str\":\"%s\",\"error\":\"%v\",\"user_agent\":\"%s\"}",
			carpoolIDStr, err, r.UserAgent())
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Successfully parsed carpool ID\",\"carpool_id\":\"%s\"}", carpoolID)

	// Fetch carpool from database
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Fetching carpool from database\",\"carpool_id\":\"%s\"}", carpoolID)
	carpool, err := h.carpoolRepo.GetCarPool(r.Context(), carpoolID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("{\"severity\":\"INFO\",\"message\":\"Carpool not found in database\",\"carpool_id\":\"%s\",\"user_agent\":\"%s\"}",
				carpoolID, r.UserAgent())
			http.Error(w, "Carpool not found", http.StatusNotFound)
			return
		}
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Database error while fetching carpool\",\"carpool_id\":\"%s\",\"error\":\"%v\",\"user_agent\":\"%s\"}",
			carpoolID, err, r.UserAgent())
		http.Error(w, "Failed to get carpool", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Successfully retrieved carpool from database\",\"carpool_id\":\"%s\",\"carpool_name\":\"%s\",\"creator_id\":\"%s\"}",
		carpoolID, carpool.CarpoolName, carpool.CreatorID)

	// Prepare response
	response := map[string]interface{}{
		"creator_id": carpool.CreatorID,
		"carpool_id": carpool.ID,
	}

	// Log successful response
	duration := time.Since(startTime)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully retrieved carpool creator\",\"carpool_id\":\"%s\",\"creator_id\":\"%s\",\"carpool_name\":\"%s\",\"duration_ms\":%d,\"user_agent\":\"%s\"}",
		carpoolID, carpool.CreatorID, carpool.CarpoolName, duration.Milliseconds(), r.UserAgent())

	// Encode and send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to encode response\",\"carpool_id\":\"%s\",\"error\":\"%v\"}", carpoolID, err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Response sent successfully\",\"carpool_id\":\"%s\",\"creator_id\":\"%s\"}", carpoolID, carpool.CreatorID)
}
