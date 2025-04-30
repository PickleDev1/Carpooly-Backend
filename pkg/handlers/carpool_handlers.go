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
	// TODO: Implement
}

func (h *CarPoolHandler) DeleteCarPool(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)
	carpoolIDStr := params["id"]

	carpoolID, err := uuid.Parse(carpoolIDStr)
	if err != nil {
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}

	err = h.carpoolRepo.DeleteCarPool(context.Background(), carpoolID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Carpool not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to delete carpool: %v", err), http.StatusInternalServerError)
		return
	}

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
