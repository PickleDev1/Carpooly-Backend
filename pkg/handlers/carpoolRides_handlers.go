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

	"strconv"

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
	startTime := time.Now()
	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetCarpoolRide called\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())

	// Log all URL variables for debugging
	vars := mux.Vars(r)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"URL variables\",\"vars\":\"%+v\"}", vars)

	w.Header().Set("Content-Type", "application/json")
	rideIDStr := vars["rideID"]
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Parsed rideID from URL\",\"ride_id_str\":\"%s\"}", rideIDStr)

	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid ride ID format\",\"ride_id_str\":\"%s\",\"error\":\"%v\"}", rideIDStr, err)
		http.Error(w, "Invalid ride ID", http.StatusBadRequest)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Successfully parsed ride ID\",\"ride_id\":\"%s\"}", rideID)

	// Get Clerk ID from context for user context
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if ok {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Request from authenticated user\",\"clerk_id\":\"%s\",\"ride_id\":\"%s\"}", clerkID, rideID)
	} else {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"No Clerk ID in context\",\"ride_id\":\"%s\"}", rideID)
	}

	ride, err := h.carpoolRideRepo.GetCarpoolRide(r.Context(), rideID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Repository error getting carpool ride\",\"ride_id\":\"%s\",\"error\":\"%v\"}", rideID, err)
		http.Error(w, fmt.Sprintf("Failed to get carpool ride: %v", err), http.StatusInternalServerError)
		return
	}

	if ride == nil {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"Carpool ride not found\",\"ride_id\":\"%s\"}", rideID)
		http.Error(w, "Carpool ride not found", http.StatusNotFound)
		return
	}

	// Log ride details before sending response
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Retrieved ride details\",\"ride_id\":\"%s\",\"carpool_id\":\"%s\",\"participant_count\":%d,\"status\":%d,\"start_time\":\"%s\"}",
		rideID, ride.CarpoolID, len(ride.Participants), ride.Status, ride.StartTime.Format(time.RFC3339))

	// Ensure participants is never null in response
	if ride.Participants == nil {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"Participants is nil, initializing as empty slice\",\"ride_id\":\"%s\"}", rideID)
		ride.Participants = []models.User{}
	}

	// Log participant details for debugging
	for i, participant := range ride.Participants {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Participant details\",\"ride_id\":\"%s\",\"participant_index\":%d,\"participant_id\":\"%s\",\"display_name\":\"%s\"}",
			rideID, i, participant.ID, participant.DisplayName.String)
	}

	if err := json.NewEncoder(w).Encode(ride); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to encode response\",\"ride_id\":\"%s\",\"error\":\"%v\"}", rideID, err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetCarpoolRide completed successfully\",\"ride_id\":\"%s\",\"duration_ms\":%d,\"participant_count\":%d}",
		rideID, duration.Milliseconds(), len(ride.Participants))
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

	// If status is completed (2), log activity for all participants
	if req.Status == 2 {
		ride, err := h.carpoolRideRepo.GetCarpoolRide(r.Context(), rideID)
		if err == nil && ride != nil {
			for _, user := range ride.Participants {
				activity := &models.UserActivity{
					UserID:      user.ID,
					Type:        "ride_completed",
					RelatedID:   &ride.ID,
					RelatedType: ptrString("ride"),
					Description: ptrString("Completed a ride in carpool " + ride.CarpoolID.String()),
					Data:        ride,
					Timestamp:   time.Now(),
				}
				_ = h.userRepo.AddUserActivity(r.Context(), activity)
			}
			if ride.DriverID != nil {
				activity := &models.UserActivity{
					UserID:      *ride.DriverID,
					Type:        "ride_completed",
					RelatedID:   &ride.ID,
					RelatedType: ptrString("ride"),
					Description: ptrString("Completed a ride as driver in carpool " + ride.CarpoolID.String()),
					Data:        ride,
					Timestamp:   time.Now(),
				}
				_ = h.userRepo.AddUserActivity(r.Context(), activity)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
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
	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetCarpoolRidesByDate called\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())

	// Log all URL variables for debugging
	vars := mux.Vars(r)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetCarpoolRidesByDate URL variables\",\"vars\":\"%+v\"}", vars)

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

// GetRideByCarpoolAndDateParticipants returns the full ride object for a carpool and date
func (h *CarPoolRideHandler) GetRideByCarpoolAndDateParticipants(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] GetRideByCarpoolAndDateParticipants called. Method: %s, URL: %s, RemoteAddr: %s, UserAgent: %s", r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())
	vars := mux.Vars(r)
	carpoolIDStr := vars["carpoolID"]
	dateStr := vars["date"]
	log.Printf("[DEBUG] Input params: carpoolID=%s, date=%s", carpoolIDStr, dateStr)

	carpoolID, err := uuid.Parse(carpoolIDStr)
	if err != nil {
		log.Printf("[ERROR] Invalid carpool ID: %s, error: %v", carpoolIDStr, err)
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		log.Printf("[ERROR] Invalid date format: %s, error: %v", dateStr, err)
		http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	log.Printf("[INFO] Fetching rides for carpoolID=%s on date=%s", carpoolID, date.Format("2006-01-02"))
	rides, err := h.carpoolRideRepo.GetCarpoolRidesByDate(r.Context(), carpoolID, date)
	if err != nil {
		log.Printf("[ERROR] Failed to get rides for carpoolID=%s, date=%s, error: %v", carpoolID, date.Format("2006-01-02"), err)
		http.Error(w, "Failed to get rides", http.StatusInternalServerError)
		return
	}
	log.Printf("[INFO] Number of rides found: %d", len(rides))
	if len(rides) == 0 {
		log.Printf("[WARN] No ride found for carpoolID=%s on date=%s", carpoolID, date.Format("2006-01-02"))
		http.Error(w, "No ride found for this carpool and date", http.StatusNotFound)
		return
	}
	ride := rides[0]
	log.Printf("[INFO] Returning ride ID: %s for carpoolID=%s on date=%s", ride.ID, carpoolID, date.Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ride); err != nil {
		log.Printf("[ERROR] Failed to encode ride response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	log.Printf("[INFO] Successfully returned ride object for carpoolID=%s on date=%s", carpoolID, date.Format("2006-01-02"))
}

// AddParticipantToRide adds a user to the participants array of a ride
func (h *CarPoolRideHandler) AddParticipantToRide(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("{\"severity\":\"INFO\",\"message\":\"AddParticipantToRide called\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())

	vars := mux.Vars(r)
	rideIDStr := vars["rideID"]
	rawClerkID := vars["userID"]

	// Parse ride ID
	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid ride ID\",\"ride_id_str\":\"%s\",\"error\":\"%v\"}", rideIDStr, err)
		http.Error(w, "Invalid ride ID", http.StatusBadRequest)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Successfully parsed ride ID\",\"ride_id\":\"%s\"}", rideID)

	// Handle Clerk ID conversion
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

	// Get user object
	user, err := h.userRepo.GetUserByID(userID)
	if err != nil || user == nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"User not found\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Successfully retrieved user object\",\"user_id\":\"%s\",\"user_email\":\"%s\"}", userID, user.Email)

	// Add participant to ride
	err = h.carpoolRideRepo.AddParticipant(r.Context(), rideID, *user)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to add participant to ride\",\"ride_id\":\"%s\",\"user_id\":\"%s\",\"error\":\"%v\"}", rideID, userID, err)
		http.Error(w, "Failed to add participant", http.StatusInternalServerError)
		return
	}
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully added participant to ride\",\"ride_id\":\"%s\",\"user_id\":\"%s\"}", rideID, userID)

	// Fetch updated ride
	updatedRide, err := h.carpoolRideRepo.GetCarpoolRide(r.Context(), rideID)
	if err != nil || updatedRide == nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to fetch updated ride\",\"ride_id\":\"%s\",\"error\":\"%v\"}", rideID, err)
		http.Error(w, "Failed to fetch updated ride", http.StatusInternalServerError)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Successfully fetched updated ride\",\"ride_id\":\"%s\",\"participant_count\":%d}", rideID, len(updatedRide.Participants))

	// Return response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updatedRide); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to encode response\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"AddParticipantToRide completed\",\"duration_ms\":%d,\"ride_id\":\"%s\",\"user_id\":\"%s\",\"clerk_id\":\"%s\"}",
		duration.Milliseconds(), rideID, userID, clerkID)
}

// GetUserCompletedRides returns all completed rides for the authenticated user
func (h *CarPoolRideHandler) GetUserCompletedRides(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("🚀 [REQ-COMPLETED] ====== GetUserCompletedRides START ======")
	log.Printf("🚀 [REQ-COMPLETED] {\"severity\":\"INFO\",\"message\":\"GetUserCompletedRides called\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())

	ctx := r.Context()

	// Get Clerk ID from context for authentication
	clerkID, ok := middleware.GetClerkIDFromContext(ctx)
	if !ok {
		log.Printf("❌ [REQ-COMPLETED] {\"severity\":\"ERROR\",\"message\":\"Unauthorized: No Clerk ID in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("✅ [REQ-COMPLETED] {\"severity\":\"INFO\",\"message\":\"Authenticated user\",\"clerk_id\":\"%s\"}", clerkID)

	// Convert Clerk ID to user UUID
	userID, err := h.userRepo.GetUserIDByClerkID(ctx, clerkID)
	if err != nil {
		log.Printf("❌ [REQ-COMPLETED] {\"severity\":\"ERROR\",\"message\":\"Failed to get user UUID from Clerk ID\",\"clerk_id\":\"%s\",\"error\":%q}", clerkID, err.Error())
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}
	log.Printf("🔍 [REQ-COMPLETED] {\"severity\":\"INFO\",\"message\":\"Resolved user ID\",\"clerk_id\":\"%s\",\"user_id\":\"%s\"}", clerkID, userID.String())

	// Get limit from query parameters (default 50, max 200)
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	log.Printf("🔍 [REQ-COMPLETED] {\"severity\":\"DEBUG\",\"message\":\"Completed rides limit\",\"limit\":%d}", limit)

	// Get completed rides
	rides, err := h.carpoolRideRepo.GetUserCompletedRides(ctx, userID, limit)
	if err != nil {
		log.Printf("❌ [REQ-COMPLETED] {\"severity\":\"ERROR\",\"message\":\"Failed to get completed rides\",\"user_id\":\"%s\",\"error\":%q}", userID.String(), err.Error())
		http.Error(w, "Failed to get completed rides", http.StatusInternalServerError)
		return
	}

	// Ensure rides is never null in response
	if rides == nil {
		rides = []models.CarpoolRide{}
	}

	// Always fill miles_saved, never return CalculatedDistance
	for i := range rides {
		if rides[i].MilesSaved == nil && rides[i].CalculatedDistance != nil {
			rides[i].MilesSaved = rides[i].CalculatedDistance
		}
		rides[i].CalculatedDistance = nil // Remove CalculatedDistance from response
	}

	log.Printf("📊 [REQ-COMPLETED] {\"severity\":\"INFO\",\"message\":\"Retrieved completed rides\",\"user_id\":\"%s\",\"count\":%d}", userID.String(), len(rides))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rides); err != nil {
		log.Printf("❌ [REQ-COMPLETED] {\"severity\":\"ERROR\",\"message\":\"Failed to encode response\",\"error\":%v}", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	log.Printf("🎉 [REQ-COMPLETED] ====== GetUserCompletedRides SUCCESS ======")
	log.Printf("🎉 [REQ-COMPLETED] {\"severity\":\"INFO\",\"message\":\"GetUserCompletedRides completed\",\"duration_ms\":%d,\"user_id\":\"%s\",\"count\":%d}",
		duration.Milliseconds(), userID.String(), len(rides))
}
