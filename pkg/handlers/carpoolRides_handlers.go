package handlers

import (
	"car-backend/middleware"
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CarPoolRideHandler struct {
	carpoolRideRepo *repository.CarPoolRideRepository
	userRepo        *repository.UserRepository
	carpoolRepo     *repository.CarPoolRepository
}

func NewCarPoolRideHandler(carpoolRideRepo *repository.CarPoolRideRepository, userRepo *repository.UserRepository, carpoolRepo *repository.CarPoolRepository) *CarPoolRideHandler {
	return &CarPoolRideHandler{
		carpoolRideRepo: carpoolRideRepo,
		userRepo:        userRepo,
		carpoolRepo:     carpoolRepo,
	}
}

func (h *CarPoolRideHandler) CreateCarpoolRide(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get timezone from context for validation
	timezoneStr, ok := middleware.GetTimezoneFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"No timezone header provided for ride creation\"}")
		// Don't fail the request, just log a warning
		timezoneStr = "UTC"
	} else {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Timezone header received for ride creation\",\"timezone\":\"%s\"}", timezoneStr)
	}

	vars := mux.Vars(r)
	carpoolIDStr := vars["id"]

	carpoolID, err := uuid.Parse(carpoolIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ID\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Creating carpool ride\",\"carpool_id\":\"%s\",\"timezone\":\"%s\"}", carpoolID, timezoneStr)

	// Get all carpool members first
	members, err := h.carpoolRepo.GetCarpoolMembers(r.Context(), carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get carpool members\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to get carpool members", http.StatusInternalServerError)
		return
	}

	var ride models.CarpoolRide
	if err := json.NewDecoder(r.Body).Decode(&ride); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid request body\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set the carpoolID and participants
	ride.CarpoolID = carpoolID
	ride.Participants = members

	// Validate that StartTime is provided
	if ride.StartTime.IsZero() {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"StartTime is required but not provided\"}")
		http.Error(w, "StartTime is required", http.StatusBadRequest)
		return
	}

	// Log the start time in both UTC and user's timezone for debugging
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"StartTime provided in request\",\"start_time_utc\":\"%s\",\"timezone\":\"%s\"}",
		ride.StartTime.Format(time.RFC3339), timezoneStr)

	// These will be set by the repository, but we'll initialize them here for clarity
	ride.DriverID = nil
	ride.LocationLat = nil
	ride.LocationLng = nil
	ride.MilesSaved = nil

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Creating ride with participants\",\"carpool_id\":\"%s\",\"participant_count\":%d,\"start_time\":\"%s\",\"timezone\":\"%s\"}",
		ride.CarpoolID, len(members), ride.StartTime.Format(time.RFC3339), timezoneStr)

	if err := h.carpoolRideRepo.CreateCarpoolRide(r.Context(), &ride); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create carpool ride\",\"error\":\"%v\"}", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ride)
}

func (h *CarPoolRideHandler) GetCarpoolRide(w http.ResponseWriter, r *http.Request) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetCarpoolRide called\",\"method\":\"%s\",\"url\":\"%s\"}", r.Method, r.URL.String())

	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	rideIDStr := vars["rideID"]
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Parsed rideID from URL\",\"ride_id_str\":\"%s\"}", rideIDStr)

	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid ride ID format\",\"ride_id_str\":\"%s\",\"error\":\"%v\"}", rideIDStr, err)
		http.Error(w, "Invalid ride ID", http.StatusBadRequest)
		return
	}

	ride, err := h.carpoolRideRepo.GetCarpoolRide(r.Context(), rideID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Carpool ride not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get carpool ride: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(ride)
}

func (h *CarPoolRideHandler) DeleteCarpoolRide(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: Method: %s, URL: %s", r.Method, r.URL)

	vars := mux.Vars(r)
	carpoolIDStr := vars["carpoolID"]
	rideIDStr := vars["rideID"]

	carpoolID, err := uuid.Parse(carpoolIDStr)
	if err != nil {
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}

	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
		http.Error(w, "Invalid ride ID", http.StatusBadRequest)
		return
	}

	err = h.carpoolRideRepo.DeleteCarpoolRide(r.Context(), carpoolID, rideID)
	if err != nil {
		if err.Error() == "carpool ride not found" {
			http.Error(w, "Carpool ride not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to delete carpool ride: %v", err)
		http.Error(w, "Failed to delete carpool ride", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CarPoolRideHandler) UpdateCarpoolRideStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	rideIDStr := vars["rideID"]

	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
		http.Error(w, "Invalid ride ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Status int `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = h.carpoolRideRepo.UpdateCarpoolRideStatus(r.Context(), rideID, req.Status)
	if err != nil {
		if err.Error() == "carpool ride not found" {
			http.Error(w, "Carpool ride not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to update carpool ride status: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *CarPoolRideHandler) GetUserActiveRides(w http.ResponseWriter, r *http.Request) {
	// Log request details
	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetUserActiveRides called\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())

	vars := mux.Vars(r)
	rawClerkID := vars["userID"]

	// Add "user_" prefix if it's missing
	clerkID := rawClerkID
	if !strings.HasPrefix(rawClerkID, "user_") {
		clerkID = "user_" + rawClerkID
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Getting active rides for clerk ID\",\"clerk_id\":\"%s\"}", clerkID)

	// Convert clerk_id to user_id
	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}",
			clerkID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	rides, err := h.carpoolRideRepo.GetUserActiveRides(r.Context(), userID.String())
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get active rides\",\"user_id\":\"%s\",\"error\":\"%v\"}",
			userID, err)
		http.Error(w, "Failed to get active rides", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rides)
}

func (h *CarPoolRideHandler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	rideID, err := uuid.Parse(vars["rideID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid ride ID\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid ride ID", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(vars["userID"])
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid user ID\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	err = h.carpoolRideRepo.RemoveParticipant(r.Context(), rideID, userID)
	if err != nil {
		if err.Error() == "ride not found" {
			http.Error(w, "Ride not found", http.StatusNotFound)
			return
		}
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to remove participant\",\"error\":\"%v\"}", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *CarPoolRideHandler) GetCarpoolRidesByDate(w http.ResponseWriter, r *http.Request) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetCarpoolRidesByDate called\",\"method\":\"%s\",\"url\":\"%s\"}", r.Method, r.URL.String())

	// Get Clerk ID from the authenticated session using middleware helper
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get Clerk ID from context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Got Clerk ID from context\",\"clerk_id\":\"%s\"}", clerkID)

	// Convert Clerk ID to internal user UUID
	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID from Clerk ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", clerkID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Converted Clerk ID to user ID\",\"clerk_id\":\"%s\",\"user_id\":\"%s\"}", clerkID, userID)

	vars := mux.Vars(r)
	carpoolIDStr := vars["id"]
	dateStr := vars["date"]
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Parsed URL variables\",\"carpool_id_str\":\"%s\",\"date_str\":\"%s\"}", carpoolIDStr, dateStr)

	carpoolID, err := uuid.Parse(carpoolIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ID\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Parsed carpool ID\",\"carpool_id\":\"%s\"}", carpoolID)

	// Verify carpool exists and user has access
	carpool, err := h.carpoolRepo.GetCarPool(r.Context(), carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get carpool\",\"error\":\"%v\"}", err)
		http.Error(w, "Carpool not found", http.StatusNotFound)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Found carpool\",\"carpool_id\":\"%s\",\"creator_id\":\"%s\"}", carpoolID, carpool.CreatorID)

	// Check if user is creator or member of the carpool
	if carpool.CreatorID != userID {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"User is not creator, checking membership\",\"user_id\":\"%s\",\"creator_id\":\"%s\"}", userID, carpool.CreatorID)
		// Check if user is a member by getting carpool members
		members, err := h.carpoolRepo.GetCarpoolMembers(r.Context(), carpoolID)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get carpool members\",\"error\":\"%v\"}", err)
			http.Error(w, "Failed to verify access", http.StatusInternalServerError)
			return
		}

		isMember := false
		for _, member := range members {
			if member.ID == userID {
				isMember = true
				break
			}
		}

		if !isMember {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"User not authorized to access carpool\",\"user_id\":\"%s\",\"carpool_id\":\"%s\"}", userID, carpoolID)
			http.Error(w, "Not authorized to access this carpool", http.StatusForbidden)
			return
		}
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"User is member of carpool\",\"user_id\":\"%s\",\"carpool_id\":\"%s\"}", userID, carpoolID)
	} else {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"User is creator of carpool\",\"user_id\":\"%s\",\"carpool_id\":\"%s\"}", userID, carpoolID)
	}

	// Parse the date string (expected format: "2025-05-05")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid date format\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Parsed date\",\"date\":\"%s\"}", date.Format("2006-01-02"))

	rides, err := h.carpoolRideRepo.GetCarpoolRidesByDate(r.Context(), carpoolID, date)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get carpool rides\",\"error\":\"%v\"}", err)
		if err == sql.ErrNoRows {
			// Return empty array instead of null
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]models.CarpoolRide{})
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get carpool rides: %v", err), http.StatusInternalServerError)
		return
	}

	// If no rides found, return empty array instead of null
	if rides == nil {
		rides = []models.CarpoolRide{}
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Returning rides\",\"ride_count\":%d}", len(rides))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rides)
}

func (h *CarPoolRideHandler) UpdateCarpoolRideDriver(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rideIDStr := vars["rideID"]

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Received update driver request\",\"ride_id\":\"%s\"}", rideIDStr)

	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid ride ID\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid ride ID", http.StatusBadRequest)
		return
	}

	var req struct {
		DriverID string `json:"driver_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid request body\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Parsed request\",\"ride_id\":\"%s\",\"driver_id\":\"%s\"}",
		rideID, req.DriverID)

	driverID, err := uuid.Parse(req.DriverID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid driver ID\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid driver ID", http.StatusBadRequest)
		return
	}

	err = h.carpoolRideRepo.UpdateCarpoolRideDriver(r.Context(), rideID, driverID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update driver\",\"error\":\"%v\",\"ride_id\":\"%s\",\"driver_id\":\"%s\"}",
			err, rideID, driverID)
		if err.Error() == "carpool ride not found" {
			http.Error(w, "Carpool ride not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to update carpool ride driver: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully updated driver\",\"ride_id\":\"%s\",\"driver_id\":\"%s\"}",
		rideID, driverID)
	w.WriteHeader(http.StatusOK)
}

func (h *CarPoolRideHandler) GetUserTotalRides(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetUserTotalRides called\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())

	vars := mux.Vars(r)
	rawClerkID := vars["userID"]
	if rawClerkID == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Missing userID parameter\"}")
		http.Error(w, "Missing userID parameter", http.StatusBadRequest)
		return
	}

	// Add "user_" prefix if it's missing
	clerkID := rawClerkID
	if !strings.HasPrefix(rawClerkID, "user_") {
		clerkID = "user_" + rawClerkID
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Added user_ prefix to clerk ID\",\"raw_clerk_id\":\"%s\",\"clerk_id\":\"%s\"}",
			rawClerkID, clerkID)
	}

	// Convert clerk_id to user_id
	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}",
			clerkID, err)
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Successfully converted clerk ID to user ID\",\"clerk_id\":\"%s\",\"user_id\":\"%s\"}",
		clerkID, userID)

	totalRides, err := h.carpoolRideRepo.GetUserTotalRides(r.Context(), userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get total rides\",\"user_id\":\"%s\",\"error\":\"%v\"}",
			userID, err)
		http.Error(w, "Failed to get total rides", http.StatusInternalServerError)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Successfully retrieved total rides\",\"user_id\":\"%s\",\"total_rides\":%d}",
		userID, totalRides)

	response := struct {
		TotalRides int `json:"total_rides"`
	}{
		TotalRides: totalRides,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to encode response\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetUserTotalRides completed\",\"duration_ms\":%d,\"user_id\":\"%s\",\"total_rides\":%d}",
		duration.Milliseconds(), userID, totalRides)
}

func (h *CarPoolRideHandler) GetActiveRides(w http.ResponseWriter, r *http.Request) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetActiveRides called\",\"method\":\"%s\",\"url\":\"%s\"}", r.Method, r.URL.String())

	// Get timezone from context (set by middleware)
	timezoneStr, ok := middleware.GetTimezoneFromContext(r.Context())
	if !ok {
		timezoneStr = "UTC" // Fallback to UTC
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"No timezone in context, using UTC\"}")
	}

	// Parse timezone
	loc, err := time.LoadLocation(timezoneStr)
	if err != nil {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"Invalid timezone, using UTC\",\"timezone\":\"%s\",\"error\":\"%v\"}", timezoneStr, err)
		loc = time.UTC
		timezoneStr = "UTC"
	}

	// Get current time in user's timezone
	now := time.Now().In(loc)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Current time in user timezone\",\"timezone\":\"%s\",\"now\":\"%s\"}", timezoneStr, now.Format(time.RFC3339))

	// Get Clerk ID from the authenticated session using middleware helper
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get Clerk ID from context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Got Clerk ID from context\",\"clerk_id\":\"%s\"}", clerkID)

	// Convert Clerk ID to internal user UUID
	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID from Clerk ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", clerkID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Converted Clerk ID to user ID\",\"clerk_id\":\"%s\",\"user_id\":\"%s\"}", clerkID, userID)

	rides, err := h.carpoolRideRepo.GetActiveRides(r.Context(), userID, timezoneStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get active rides\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Failed to get active rides", http.StatusInternalServerError)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Retrieved active rides\",\"user_id\":\"%s\",\"ride_count\":%d}", userID, len(rides))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rides)
}
