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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CarPoolHandler struct {
	carpoolRepo *repository.CarPoolRepository
}

func NewCarPoolHandler(repo *repository.CarPoolRepository) *CarPoolHandler {
	return &CarPoolHandler{
		carpoolRepo: repo,
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
	//userID, ok := middleware.GetUserIDFromContext(r.Context())
	//if !ok {
	// For local testing, use this hardcoded UUID:
	userIDStr := "b0337c8a-1eed-4a11-90c8-130016c47d0a"
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to parse UUID: %v\"}", err)
		http.Error(w, "Invalid user ID format", http.StatusInternalServerError)
		return
	}

	//	http.Error(w, "Unauthorized - No user ID in context", http.StatusUnauthorized)
	//	return
	//}

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

	vars := mux.Vars(r)
	userIDStr := vars["userID"]

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	carpools, err := h.carpoolRepo.GetUserCarpools(r.Context(), userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user carpools: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(carpools)
}
