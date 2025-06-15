package handlers

import (
	"car-backend/middleware"
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type LocationHandler struct {
	locationRepo *repository.LocationRepository
	userRepo     *repository.UserRepository
}

func NewLocationHandler(locationRepo *repository.LocationRepository, userRepo *repository.UserRepository) *LocationHandler {
	return &LocationHandler{
		locationRepo: locationRepo,
		userRepo:     userRepo,
	}
}

// UpdateLocation handles location updates from users
func (h *LocationHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get clerk ID from context
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"No clerk ID in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user ID from clerk ID
	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	// Get carpool ride ID from URL
	vars := mux.Vars(r)
	carpoolRideID, err := uuid.Parse(vars["rideID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ride ID\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid carpool ride ID", http.StatusBadRequest)
		return
	}

	// Parse location update request
	var location models.LocationUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&location); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid request body\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update location
	err = h.locationRepo.UpdateLocation(r.Context(), userID, carpoolRideID, &location)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update location\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to update location", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetLatestLocation gets the most recent location for a user in a carpool ride
func (h *LocationHandler) GetLatestLocation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	userID, err := uuid.Parse(vars["userID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid user ID\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	carpoolRideID, err := uuid.Parse(vars["rideID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ride ID\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid carpool ride ID", http.StatusBadRequest)
		return
	}

	location, err := h.locationRepo.GetLatestLocation(r.Context(), userID, carpoolRideID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get latest location\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to get latest location", http.StatusInternalServerError)
		return
	}

	if location == nil {
		http.Error(w, "Location not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(location)
}

// GetLocationHistory gets location history for a user in a carpool ride
func (h *LocationHandler) GetLocationHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	userID, err := uuid.Parse(vars["userID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid user ID\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	carpoolRideID, err := uuid.Parse(vars["rideID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ride ID\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid carpool ride ID", http.StatusBadRequest)
		return
	}

	// Get limit from query parameter, default to 10
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	locations, err := h.locationRepo.GetLocationHistory(r.Context(), userID, carpoolRideID, limit)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get location history\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to get location history", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(locations)
}

// UpdateLocationSettings updates a user's location sharing settings
func (h *LocationHandler) UpdateLocationSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get clerk ID from context
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"No clerk ID in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user ID from clerk ID
	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	var settings models.LocationSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid request body\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = h.locationRepo.UpdateLocationSettings(r.Context(), userID, &settings)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update location settings\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to update location settings", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetLocationSettings gets a user's location sharing settings
func (h *LocationHandler) GetLocationSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get clerk ID from context
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"No clerk ID in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user ID from clerk ID
	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	settings, err := h.locationRepo.GetLocationSettings(r.Context(), userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get location settings\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to get location settings", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(settings)
}
