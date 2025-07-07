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
	carpoolRepo *repository.CarPoolRepository
	userRepo    *repository.UserRepository
}

func NewCarPoolHandler(carpoolRepo *repository.CarPoolRepository, userRepo *repository.UserRepository) *CarPoolHandler {
	return &CarPoolHandler{
		carpoolRepo: carpoolRepo,
		userRepo:    userRepo,
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
	carpool := &models.Carpool{
		CreatorID:          userID, // Use actual userID from context(commented out in code above)
		CarpoolName:        req.CarpoolName,
		Status:             false, // Default status
		RecurringOption:    req.RecurringOption,
		AvailableSeats:     req.AvailableSeats,
		DestinationAddress: req.DestinationAddress,
		Seats:              req.Seats,
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

	json.NewEncoder(w).Encode(carpool)
}

func (h *CarPoolHandler) UpdateCarPool(w http.ResponseWriter, r *http.Request) {
	carpoolID := mux.Vars(r)["id"]
	var req models.UpdateCarPoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(carpoolID)
	if err != nil {
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}
	err = h.carpoolRepo.UpdateCarPool(r.Context(), id, &req)
	if err != nil {
		http.Error(w, "Failed to update carpool", http.StatusInternalServerError)
		return
	}
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
	w.Header().Set("Content-Type", "application/json")

	// Get Clerk ID from context (this stays as string)
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized - No clerk ID in context", http.StatusUnauthorized)
		return
	}

	// Get the corresponding UUID from users table using userRepo
	userUUID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user UUID\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	// Print the request
	log.Printf("Received GetUserCarpools request for userID: %s", userUUID)

	carpools, err := h.carpoolRepo.GetUserCarpools(r.Context(), userUUID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user carpools: %v", err), http.StatusInternalServerError)
		return
	}

	// Print the response
	log.Printf("Retrieved carpools for userID: %s: %+v", userUUID, carpools)

	json.NewEncoder(w).Encode(carpools)
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
	err = h.carpoolRepo.AddCarpoolMemberByAPI(r.Context(), carpoolID, userID)
	if err != nil {
		http.Error(w, "Failed to add member", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AddUserToFutureRidesAPI handles adding a user to all future rides' participants
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
