package handlers

import (
	"car-backend/middleware"
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

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

	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"No clerk ID in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", clerkID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	carpoolRideID, err := uuid.Parse(vars["rideID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ride ID\",\"ride_id\":\"%s\",\"error\":\"%v\"}", vars["rideID"], err)
		http.Error(w, "Invalid carpool ride ID", http.StatusBadRequest)
		return
	}

	var location models.UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&location); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid request body\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = h.locationRepo.UpdateLocation(r.Context(), userID, carpoolRideID, &location)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update location\",\"user_id\":\"%s\",\"ride_id\":\"%s\",\"error\":\"%v\"}", userID, carpoolRideID, err)
		http.Error(w, "Failed to update location", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Location updated successfully\",\"user_id\":\"%s\",\"ride_id\":\"%s\",\"latitude\":%f,\"longitude\":%f}", userID, carpoolRideID, location.Latitude, location.Longitude)
	w.WriteHeader(http.StatusOK)
}

// GetLatestLocation gets the most recent location for a user in a carpool ride
func (h *LocationHandler) GetLatestLocation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	userID, err := uuid.Parse(vars["userID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid user ID\",\"user_id\":\"%s\",\"error\":\"%v\"}", vars["userID"], err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	carpoolRideID, err := uuid.Parse(vars["rideID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ride ID\",\"ride_id\":\"%s\",\"error\":\"%v\"}", vars["rideID"], err)
		http.Error(w, "Invalid carpool ride ID", http.StatusBadRequest)
		return
	}

	location, err := h.locationRepo.GetLatestLocation(r.Context(), userID, carpoolRideID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get latest location\",\"user_id\":\"%s\",\"ride_id\":\"%s\",\"error\":\"%v\"}", userID, carpoolRideID, err)
		http.Error(w, "Failed to get latest location", http.StatusInternalServerError)
		return
	}

	if location == nil {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"Location not found\",\"user_id\":\"%s\",\"ride_id\":\"%s\"}", userID, carpoolRideID)
		http.Error(w, "Location not found", http.StatusNotFound)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Returning latest location\",\"user_id\":\"%s\",\"ride_id\":\"%s\",\"latitude\":%f,\"longitude\":%f,\"timestamp\":\"%s\"}", userID, carpoolRideID, location.Latitude, location.Longitude, location.Timestamp.Format(time.RFC3339))
	json.NewEncoder(w).Encode(location)
}

// GetLocationHistory gets location history for a user in a carpool ride
func (h *LocationHandler) GetLocationHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	userID, err := uuid.Parse(vars["userID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid user ID\",\"user_id\":\"%s\",\"error\":\"%v\"}", vars["userID"], err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	carpoolRideID, err := uuid.Parse(vars["rideID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ride ID\",\"ride_id\":\"%s\",\"error\":\"%v\"}", vars["rideID"], err)
		http.Error(w, "Invalid carpool ride ID", http.StatusBadRequest)
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	locations, err := h.locationRepo.GetLocationHistory(r.Context(), userID, carpoolRideID, limit)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get location history\",\"user_id\":\"%s\",\"ride_id\":\"%s\",\"error\":\"%v\"}", userID, carpoolRideID, err)
		http.Error(w, "Failed to get location history", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Returning location history\",\"user_id\":\"%s\",\"ride_id\":\"%s\",\"count\":%d}", userID, carpoolRideID, len(locations))
	json.NewEncoder(w).Encode(locations)
}

// UpdateLocationSettings updates a user's location sharing settings
func (h *LocationHandler) UpdateLocationSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"No clerk ID in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", clerkID, err)
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
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update location settings\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Failed to update location settings", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Location settings updated successfully\",\"user_id\":\"%s\"}", userID)
	w.WriteHeader(http.StatusOK)
}

// GetLocationSettings gets a user's location sharing settings
func (h *LocationHandler) GetLocationSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"No clerk ID in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", clerkID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	settings, err := h.locationRepo.GetLocationSettings(r.Context(), userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get location settings\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Failed to get location settings", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Returning location settings\",\"user_id\":\"%s\"}", userID)
	json.NewEncoder(w).Encode(settings)
}

// GetAllLatestLocations returns the latest location for all users in a ride
func (h *LocationHandler) GetAllLatestLocations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	rideID, err := uuid.Parse(vars["rideID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid ride ID\",\"ride_id\":\"%s\",\"error\":\"%v\"}", vars["rideID"], err)
		http.Error(w, "Invalid ride ID", http.StatusBadRequest)
		return
	}

	locations, err := h.locationRepo.GetAllLatestLocationsForRide(r.Context(), rideID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get all latest locations for ride\",\"ride_id\":\"%s\",\"error\":\"%v\"}", rideID, err)
		http.Error(w, "Failed to get locations", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Returning all latest locations for ride\",\"ride_id\":\"%s\",\"count\":%d}", rideID, len(locations))
	json.NewEncoder(w).Encode(locations)
}
