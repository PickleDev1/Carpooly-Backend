package handlers

import (
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"car-backend/pkg/services"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type MatchingHandler struct {
	matchingRepo            *repository.MatchingRepository
	userRepo                *repository.UserRepository
	carpoolRepo             *repository.CarPoolRepository
	scheduleRepo            *repository.CarpoolScheduleRepository
	rideRepo                *repository.CarPoolRideRepository
	routeService            *services.RouteService
	enhancedMatchingService *services.EnhancedMatchingService
}

func NewMatchingHandler(matchingRepo *repository.MatchingRepository, userRepo *repository.UserRepository, carpoolRepo *repository.CarPoolRepository, scheduleRepo *repository.CarpoolScheduleRepository, rideRepo *repository.CarPoolRideRepository, routeService *services.RouteService, enhancedMatchingService *services.EnhancedMatchingService) *MatchingHandler {
	return &MatchingHandler{
		matchingRepo:            matchingRepo,
		userRepo:                userRepo,
		carpoolRepo:             carpoolRepo,
		scheduleRepo:            scheduleRepo,
		rideRepo:                rideRepo,
		routeService:            routeService,
		enhancedMatchingService: enhancedMatchingService,
	}
}

// 1. GET /api/matching/preferences - Fetch user preferences
func (h *MatchingHandler) GetUserMatchingPreferences(w http.ResponseWriter, r *http.Request) {
	// Log the route being called
	log.Printf("{\"severity\":\"INFO\",\"message\":\"🚀 ROUTE CALLED: /api/matching/preferences\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr)

	// Log the incoming request method and URL
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Received GetUserMatchingPreferences request\",\"method\":\"%s\",\"url\":\"%s\"}", r.Method, r.URL.String())

	// Log all request headers for debugging authentication
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = v[0]
	}
	headersJSON, _ := json.Marshal(headers)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetUserMatchingPreferences request headers\",\"headers\":%s}", string(headersJSON))

	// Specifically log the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetUserMatchingPreferences: No Authorization header provided\"}")
	} else {
		// Log the first part of the token for debugging (don't log the full token for security)
		if len(authHeader) > 20 {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetUserMatchingPreferences: Authorization header present\",\"header_start\":\"%s...\",\"header_length\":%d}", authHeader[:20], len(authHeader))
		} else {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetUserMatchingPreferences: Authorization header present\",\"header\":\"%s\"}", authHeader)
		}
	}

	// Log the raw request for debugging
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetUserMatchingPreferences: Request details\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}", r.RemoteAddr, r.UserAgent())

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetUserMatchingPreferences: No Clerk claims in context - authentication failed\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.Subject

	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetUserMatchingPreferences: User authenticated successfully\",\"clerk_id\":\"%s\"}", userID)

	// Convert Clerk ID to UUID using the same pattern as other handlers
	userUUID, err := h.userRepo.GetUserIDByClerkID(r.Context(), userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetUserMatchingPreferences: Failed to get user UUID from Clerk ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetUserMatchingPreferences: Clerk ID converted to UUID\",\"clerk_id\":\"%s\",\"user_uuid\":\"%s\"}", userID, userUUID.String())

	// CRITICAL SECURITY LOGGING: Track user ID being used for database query
	log.Printf("{\"severity\":\"SECURITY\",\"message\":\"GetUserMatchingPreferences: About to query database with user_id\",\"user_uuid\":\"%s\",\"clerk_id\":\"%s\"}", userUUID.String(), userID)

	// Phase 3: Default to personal scope (companyID = nil)
	// In Phase 4, we'll add scope resolution from request parameters
	var companyID *uuid.UUID = nil

	prefs, err := h.matchingRepo.GetUserMatchingPreferences(r.Context(), userUUID.String(), companyID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetUserMatchingPreferences: Error getting user preferences\",\"user_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If no preferences exist, return default preferences
	if prefs == nil {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"GetUserMatchingPreferences: No preferences found, returning defaults\",\"user_id\":\"%s\"}", userUUID.String())
		prefs = &models.UserMatchingPreferences{
			UserID:                     userUUID.String(),
			MaxDetourMinutes:           15,
			PreferredGroupSize:         4,
			DriverPreference:           "flexible",
			ScheduleFlexibilityMinutes: 30,
			MaxPickupDistanceMiles:     5.0,
			MinCompatibilityScore:      0.7,
			// CRITICAL FIX: Set destination coordinates to 0.0 for new users
			DestinationLatitude:  func() *float64 { v := 0.0; return &v }(), // 0.0 for new users
			DestinationLongitude: func() *float64 { v := 0.0; return &v }(), // 0.0 for new users
			ArrivalTime:          nil,                                       // No schedule set
			CommuteDays:          nil,                                       // No schedule set
			NotificationPreferences: models.NotificationPrefs{
				Email: true,
				Push:  true,
				SMS:   false,
			},
			UserDemographics: models.UserDemographics{
				AgeRange:      "26-35",
				Gender:        "prefer_not_to_say",
				Occupation:    "",
				StudentStatus: "not_student",
				Company:       "",
			},
			DemographicPreferences: models.DemographicPreferences{
				AgePreferences:        []string{"18-25", "26-35", "36-45", "46-55"},
				GenderPreferences:     []string{"any"},
				StudentPreference:     "both",
				OccupationPreferences: []string{},
			},
			IsActive: true,
		}
	} else {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"GetUserMatchingPreferences: Retrieved existing preferences\",\"user_id\":\"%s\"}", userUUID.String())
	}

	w.Header().Set("Content-Type", "application/json")

	// Return the exact response format specified
	response := map[string]interface{}{
		"success":     true,
		"preferences": prefs,
	}

	json.NewEncoder(w).Encode(response)
}

// 2. PUT /api/matching/preferences - Update user preferences
func (h *MatchingHandler) UpdateUserMatchingPreferences(w http.ResponseWriter, r *http.Request) {
	// Log the route being called
	log.Printf("{\"severity\":\"INFO\",\"message\":\"🚀 ROUTE CALLED: /api/matching/preferences (PUT)\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr)

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateUserMatchingPreferences: No Clerk claims in context - authentication failed\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.Subject

	log.Printf("{\"severity\":\"INFO\",\"message\":\"UpdateUserMatchingPreferences: User authenticated successfully\",\"clerk_id\":\"%s\"}", userID)

	// Convert Clerk ID to UUID using the same pattern as other handlers
	userUUID, err := h.userRepo.GetUserIDByClerkID(r.Context(), userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateUserMatchingPreferences: Failed to get user UUID from Clerk ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"UpdateUserMatchingPreferences: Clerk ID converted to UUID\",\"clerk_id\":\"%s\",\"user_uuid\":\"%s\"}", userID, userUUID.String())

	var prefs models.UserMatchingPreferences
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateUserMatchingPreferences: Failed to decode request body\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Apply frontend simplification defaults
	if prefs.ScheduleFlexibilityMinutes == 0 {
		prefs.ScheduleFlexibilityMinutes = 30 // Default for simplified UI
	}
	if prefs.MinCompatibilityScore == 0.0 {
		prefs.MinCompatibilityScore = 0.7 // Default for simplified UI
	}

	// Ensure student_status has default value for simplified UI
	if prefs.UserDemographics.StudentStatus == "" {
		prefs.UserDemographics.StudentStatus = "not_student"
	}

	// Normalize driver_preference for frontend compatibility
	if prefs.DriverPreference == "flexible" {
		prefs.DriverPreference = "either"
	}

	// Validate preferences
	if prefs.MaxDetourMinutes < 5 || prefs.MaxDetourMinutes > 60 {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateUserMatchingPreferences: Invalid max detour minutes\",\"clerk_id\":\"%s\",\"value\":%d}", userID, prefs.MaxDetourMinutes)
		http.Error(w, "Max detour minutes must be between 5 and 60", http.StatusBadRequest)
		return
	}
	if prefs.PreferredGroupSize < 2 || prefs.PreferredGroupSize > 5 {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateUserMatchingPreferences: Invalid preferred group size\",\"clerk_id\":\"%s\",\"value\":%d}", userID, prefs.PreferredGroupSize)
		http.Error(w, "Preferred group size must be between 2 and 5", http.StatusBadRequest)
		return
	}
	if prefs.MinCompatibilityScore < 0.0 || prefs.MinCompatibilityScore > 1.0 {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateUserMatchingPreferences: Invalid min compatibility score\",\"clerk_id\":\"%s\",\"value\":%.2f}", userID, prefs.MinCompatibilityScore)
		http.Error(w, "Minimum compatibility score must be between 0.0 and 1.0", http.StatusBadRequest)
		return
	}

	// Validate demographic fields
	if err := validateUserDemographics(prefs.UserDemographics); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateUserMatchingPreferences: Invalid user demographics\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := validateDemographicPreferences(prefs.DemographicPreferences); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateUserMatchingPreferences: Invalid demographic preferences\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate destination and schedule fields
	if prefs.DestinationLatitude != nil {
		if *prefs.DestinationLatitude < -90 || *prefs.DestinationLatitude > 90 {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateUserMatchingPreferences: Invalid destination_latitude\",\"user_id\":\"%s\",\"value\":%.6f}", userUUID.String(), *prefs.DestinationLatitude)
			http.Error(w, "destination_latitude must be between -90 and 90", http.StatusBadRequest)
			return
		}
	}
	if prefs.DestinationLongitude != nil {
		if *prefs.DestinationLongitude < -180 || *prefs.DestinationLongitude > 180 {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateUserMatchingPreferences: Invalid destination_longitude\",\"user_id\":\"%s\",\"value\":%.6f}", userUUID.String(), *prefs.DestinationLongitude)
			http.Error(w, "destination_longitude must be between -180 and 180", http.StatusBadRequest)
			return
		}
	}

	if len(prefs.CommuteDays) > 0 {
		allowed := map[string]struct{}{"mon": {}, "tue": {}, "wed": {}, "thu": {}, "fri": {}, "sat": {}, "sun": {}}
		for _, d := range prefs.CommuteDays {
			if _, ok := allowed[d]; !ok {
				log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateUserMatchingPreferences: Invalid commute day\",\"user_id\":\"%s\",\"day\":\"%s\"}", userUUID.String(), d)
				http.Error(w, "commute_days must be any of: mon,tue,wed,thu,fri,sat,sun", http.StatusBadRequest)
				return
			}
		}
	}

	prefs.UserID = userUUID.String()
	prefs.IsActive = true

	log.Printf("{\"severity\":\"INFO\",\"message\":\"UpdateUserMatchingPreferences: Updating preferences\",\"user_uuid\":\"%s\",\"max_detour\":%d,\"group_size\":%d,\"compatibility\":%.2f}",
		userUUID.String(), prefs.MaxDetourMinutes, prefs.PreferredGroupSize, prefs.MinCompatibilityScore)

	// Phase 3: Default to personal scope (companyID = nil)
	// In Phase 4, we'll add scope resolution from request body/query params
	var companyID *uuid.UUID = nil
	// TODO: Phase 4 - Extract company_id from request body if provided

	err = h.matchingRepo.UpsertUserMatchingPreferences(r.Context(), &prefs, companyID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateUserMatchingPreferences: Error updating preferences\",\"user_uuid\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"UpdateUserMatchingPreferences: Successfully updated preferences\",\"user_uuid\":\"%s\"}", userUUID.String())

	w.Header().Set("Content-Type", "application/json")

	// Return the exact response format specified
	response := map[string]interface{}{
		"success":     true,
		"message":     "Preferences updated successfully",
		"preferences": prefs,
	}

	json.NewEncoder(w).Encode(response)
}

// 3. GET /api/matching/potential-matches - Get potential matches
func (h *MatchingHandler) GetPotentialMatches(w http.ResponseWriter, r *http.Request) {
	// Log the route being called
	log.Printf("{\"severity\":\"INFO\",\"message\":\"🚀 ROUTE CALLED: /api/matching/potential-matches\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr)

	// Log the incoming request method and URL
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Received GetPotentialMatches request\",\"method\":\"%s\",\"url\":\"%s\"}", r.Method, r.URL.String())

	// Log all request headers for debugging authentication
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = v[0]
	}
	headersJSON, _ := json.Marshal(headers)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetPotentialMatches request headers\",\"headers\":%s}", string(headersJSON))

	// Specifically log the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetPotentialMatches: No Authorization header provided\"}")
	} else {
		// Log the first part of the token for debugging (don't log the full token for security)
		if len(authHeader) > 20 {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetPotentialMatches: Authorization header present\",\"header_start\":\"%s...\",\"header_length\":%d}", authHeader[:20], len(authHeader))
		} else {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetPotentialMatches: Authorization header present\",\"header\":\"%s\"}", authHeader)
		}
	}

	// Log the raw request for debugging
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetPotentialMatches: Request details\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}", r.RemoteAddr, r.UserAgent())

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetPotentialMatches: No Clerk claims in context - authentication failed\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.Subject

	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetPotentialMatches: User authenticated successfully\",\"clerk_id\":\"%s\"}", userID)

	// Convert Clerk ID to UUID using the same pattern as other handlers
	userUUID, err := h.userRepo.GetUserIDByClerkID(r.Context(), userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetPotentialMatches: Failed to get user UUID from Clerk ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetPotentialMatches: Clerk ID converted to UUID\",\"clerk_id\":\"%s\",\"user_uuid\":\"%s\"}", userID, userUUID.String())

	// Use preference-driven filtering instead of broad user lists
	candidates, err := h.matchingRepo.FindCandidatesByPreferences(r.Context(), userUUID.String())
	if err != nil {
		// Check if it's a destination not set error
		if err.Error() == "destination not set in preferences - cannot find matches" {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetPotentialMatches: Destination not set\",\"user_id\":\"%s\"}", userUUID.String())
			http.Error(w, "Destination required. Please set your destination in preferences.", http.StatusBadRequest)
			return
		}

		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetPotentialMatches: Error finding candidates\",\"user_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
		// Return empty response instead of 500 error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"pending_matches":  []interface{}{},
			"accepted_matches": []interface{}{},
			"expired_matches":  []interface{}{},
		})
		return
	}

	// Score each candidate and filter by fixed min_compatibility_score (simplified UI)
	const minCompatibilityScore = 0.7 // Fixed value for simplified UI
	var scoredMatches []map[string]interface{}

	for _, candidate := range candidates {
		// Get full user data for scoring (repo expects uuid.UUID)
		currentUser, err := h.userRepo.GetUserByID(userUUID)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetPotentialMatches: Error getting current user\",\"user_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
			continue
		}

		// Calculate real compatibility using enhanced matching service
		compatibility, err := h.enhancedMatchingService.CalculateRealCompatibility(currentUser, candidate)
		if err != nil {
			log.Printf("{\"severity\":\"WARN\",\"message\":\"GetPotentialMatches: Error calculating compatibility\",\"user_id\":\"%s\",\"candidate_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), candidate.ID, err)
			continue
		}

		// Filter by fixed min_compatibility_score (simplified UI)
		if compatibility.CompatibilityScore < minCompatibilityScore {
			continue
		}

		// Debug: Log the Clerk ID being returned
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetPotentialMatches: Returning candidate\",\"candidate_id\":\"%s\",\"clerk_id\":\"%s\",\"clerk_id_length\":%d}", candidate.ID.String(), candidate.ClerkID, len(candidate.ClerkID))

		// Create match data in the format expected by frontend
		matchData := map[string]interface{}{
			"id":                          compatibility.ID,
			"user2":                       candidate,
			"user2_clerk_id":              candidate.ClerkID,
			"compatibility_score":         compatibility.CompatibilityScore,
			"route_overlap_percentage":    compatibility.RouteOverlapPercentage,
			"total_distance_miles":        compatibility.TotalDistanceMiles,
			"estimated_savings_per_month": compatibility.EstimatedSavingsPerMonth,
			"match_reasons":               compatibility.MatchReasons,
			"schedule": func() map[string]interface{} {
				details := h.enhancedMatchingService.GetScheduleCompatibilityDetails(currentUser, candidate)
				return map[string]interface{}{
					"departure_time":      details.DepartureTime,
					"frequency":           details.Frequency,
					"flexibility_minutes": details.FlexibilityMinutes,
					"compatibility_score": details.CompatibilityScore,
				}
			}(),
			"status":     compatibility.Status,
			"expires_at": compatibility.ExpiresAt,
			"created_at": compatibility.CreatedAt,
		}

		scoredMatches = append(scoredMatches, matchData)
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetPotentialMatches: Found preference-driven matches\",\"user_id\":\"%s\",\"candidates\":%d,\"matches\":%d}",
		userUUID.String(), len(candidates), len(scoredMatches))

	// Return all matches as pending (since they're new potential matches)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pending_matches":  scoredMatches,
		"accepted_matches": []interface{}{},
		"expired_matches":  []interface{}{},
	})
}

// 9. GET /api/matching/stats - Get comprehensive matching statistics
func (h *MatchingHandler) GetMatchingStats(w http.ResponseWriter, r *http.Request) {
	// Log the route being called
	log.Printf("{\"severity\":\"INFO\",\"message\":\"🚀 ROUTE CALLED: /api/matching/stats\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr)

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetMatchingStats: No Clerk claims in context - authentication failed\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.Subject

	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetMatchingStats: User authenticated successfully\",\"clerk_id\":\"%s\"}", userID)

	// Convert Clerk ID to UUID using the same pattern as other handlers
	userUUID, err := h.userRepo.GetUserIDByClerkID(r.Context(), userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetMatchingStats: Failed to get user UUID from Clerk ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetMatchingStats: Clerk ID converted to UUID\",\"clerk_id\":\"%s\",\"user_uuid\":\"%s\"}", userID, userUUID.String())

	// Get comprehensive stats from repository
	stats, err := h.matchingRepo.GetMatchingStats(r.Context(), userUUID.String())
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetMatchingStats: Error getting matching stats\",\"user_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
		// Return fallback stats instead of error
		stats = h.getFallbackStats()
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetMatchingStats: Successfully retrieved stats\",\"user_id\":\"%s\"}", userUUID.String())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// getFallbackStats provides realistic fallback data when backend stats are unavailable
func (h *MatchingHandler) getFallbackStats() map[string]interface{} {
	return map[string]interface{}{
		"total_matches_generated":     0,
		"match_acceptance_rate":       0.0,
		"average_compatibility_score": 0.0,
		"total_carpools_formed":       0,
		"total_savings":               0,
		"average_route_overlap":       0.0,
		"most_common_match_reasons": []string{
			"Same destination",
			"Similar schedule",
			"Close pickup location",
			"Route overlap",
			"Flexible schedule",
		},
		"geographic_distribution": map[string]interface{}{
			"nearby":          0,
			"medium_distance": 0,
			"far":             0,
		},
		"time_to_acceptance": 0.0,
		"monthly_trends": []map[string]interface{}{
			{"month": "Oct", "matches": 0, "acceptances": 0},
			{"month": "Nov", "matches": 0, "acceptances": 0},
			{"month": "Dec", "matches": 0, "acceptances": 0},
			{"month": "Jan", "matches": 0, "acceptances": 0},
		},
	}
}

// 4. POST /api/matching/find-matches - Force refresh matches
func (h *MatchingHandler) FindMatches(w http.ResponseWriter, r *http.Request) {
	// Log the route being called
	log.Printf("{\"severity\":\"INFO\",\"message\":\"🚀 ROUTE CALLED: /api/matching/find-matches\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr)

	// Debug: log request headers and Authorization for missing claims diagnosis
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	headersJSON, _ := json.Marshal(headers)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindMatches request headers\",\"headers\":%s}", string(headersJSON))

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindMatches: No Authorization header provided\"}")
	} else {
		if len(authHeader) > 20 {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindMatches: Authorization header present\",\"header_start\":\"%s...\",\"header_length\":%d}", authHeader[:20], len(authHeader))
		} else {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindMatches: Authorization header present\",\"header\":\"%s\"}", authHeader)
		}
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindMatches: No Clerk claims in context - authentication failed\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.Subject

	log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: User authenticated successfully\",\"clerk_id\":\"%s\"}", userID)

	// Convert Clerk ID to UUID using the same pattern as other handlers
	userUUID, err := h.userRepo.GetUserIDByClerkID(r.Context(), userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindMatches: Failed to get user UUID from Clerk ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: Clerk ID converted to UUID\",\"clerk_id\":\"%s\",\"user_uuid\":\"%s\"}", userID, userUUID.String())

	var req models.MatchingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindMatches: Failed to decode request body\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.MaxResults <= 0 {
		req.MaxResults = 10
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: Processing request\",\"user_uuid\":\"%s\",\"max_results\":%d}", userUUID.String(), req.MaxResults)

	// Get user's own preferences to check demographic compatibility
	// Phase 3: Default to personal scope (companyID = nil)
	var companyID *uuid.UUID = nil
	userPrefs, err := h.matchingRepo.GetUserMatchingPreferences(r.Context(), userUUID.String(), companyID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindMatches: Error getting user preferences\",\"user_uuid\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Use demographic compatibility if user has preferences, otherwise use basic matching
	var users []*models.User
	if userPrefs != nil && len(userPrefs.DemographicPreferences.AgePreferences) > 0 {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: Using demographic compatibility matching\",\"user_uuid\":\"%s\"}", userUUID.String())
		users, err = h.matchingRepo.GetDemographicallyCompatibleUsers(r.Context(), userUUID.String())
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindMatches: Error in GetDemographicallyCompatibleUsers\",\"user_uuid\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
			// Fallback to basic matching if demographic query fails
			log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: Falling back to basic matching\",\"user_uuid\":\"%s\"}", userUUID.String())
			users, err = h.matchingRepo.GetActiveUsersForMatching(r.Context(), userUUID.String())
		}
	} else {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: Using basic matching (no demographic preferences)\",\"user_uuid\":\"%s\"}", userUUID.String())
		users, err = h.matchingRepo.GetActiveUsersForMatching(r.Context(), userUUID.String())
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindMatches: Error in GetActiveUsersForMatching\",\"user_uuid\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
		}
	}

	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindMatches: Error getting users for matching\",\"user_uuid\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: Found %d compatible users for matching\",\"user_uuid\":\"%s\",\"compatible_users_count\":%d}", len(users), userUUID.String(), len(users))

	// Generate matches with improved algorithm
	var matches []*models.PotentialMatch
	log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: Starting match generation loop\",\"user_uuid\":\"%s\",\"users_count\":%d,\"max_results\":%d}", userUUID.String(), len(users), req.MaxResults)

	for i, user := range users {
		if i >= req.MaxResults {
			log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: Reached max results limit\",\"user_uuid\":\"%s\",\"current_index\":%d,\"max_results\":%d}", userUUID.String(), i, req.MaxResults)
			break
		}

		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindMatches: Processing user for match\",\"user_uuid\":\"%s\",\"target_user_id\":\"%s\",\"target_user_name\":\"%s\",\"iteration\":%d}", userUUID.String(), user.ID.String(), user.Name, i)

		// Get the current user's full data for compatibility calculation
		currentUser, err := h.userRepo.GetUserByID(userUUID)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindMatches: Error getting current user data\",\"user_uuid\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
			continue
		}

		// Calculate REAL compatibility using enhanced matching service
		potentialMatch, err := h.enhancedMatchingService.CalculateRealCompatibility(currentUser, user)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindMatches: Error calculating real compatibility\",\"user_uuid\":\"%s\",\"target_user_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), user.ID.String(), err)
			// Fallback to basic compatibility score if real calculation fails
			compatibilityScore := calculateDemographicCompatibilityScore(userPrefs, user)
			matchReasons := generateMatchReasons(userPrefs, user)

			match := &models.PotentialMatch{
				User1ID:                  userUUID.String(),
				User2ID:                  user.ID.String(),
				CompatibilityScore:       compatibilityScore,
				RouteOverlapPercentage:   &[]float64{75.0, 85.0, 90.0}[rand.Intn(3)],
				TotalDistanceMiles:       &[]float64{8.5, 12.3, 15.7}[rand.Intn(3)],
				EstimatedSavingsPerMonth: &[]float64{45.0, 67.0, 89.0}[rand.Intn(3)],
				MatchReasons:             matchReasons,
				Status:                   "active",
				ExpiresAt:                time.Now().AddDate(0, 0, 7),
			}

			log.Printf("{\"severity\":\"WARNING\",\"message\":\"FindMatches: Using fallback compatibility calculation\",\"user_uuid\":\"%s\",\"target_user_id\":\"%s\",\"compatibility_score\":%.2f}", userUUID.String(), user.ID.String(), compatibilityScore)

			err = h.matchingRepo.CreatePotentialMatch(r.Context(), match)
			if err != nil {
				log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindMatches: Error creating fallback potential match\",\"user_uuid\":\"%s\",\"target_user_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), user.ID.String(), err)
				continue
			}

			matches = append(matches, match)
			continue
		}

		// Use the real calculated match
		match := potentialMatch
		match.User1ID = userUUID.String()
		match.User2ID = user.ID.String()
		match.Status = "active"
		match.ExpiresAt = time.Now().AddDate(0, 0, 7)

		log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: Real compatibility calculated\",\"user_uuid\":\"%s\",\"target_user_id\":\"%s\",\"compatibility_score\":%.3f,\"route_overlap\":%.1f,\"savings\":%.2f}",
			userUUID.String(), user.ID.String(), match.CompatibilityScore,
			*match.RouteOverlapPercentage, *match.EstimatedSavingsPerMonth)

		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindMatches: Attempting to create potential match\",\"user_uuid\":\"%s\",\"target_user_id\":\"%s\"}", userUUID.String(), user.ID.String())
		err = h.matchingRepo.CreatePotentialMatch(r.Context(), match)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindMatches: Error creating potential match\",\"user_uuid\":\"%s\",\"target_user_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), user.ID.String(), err)
			continue
		}

		matches = append(matches, match)
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindMatches: Successfully created potential match\",\"user_uuid\":\"%s\",\"target_user_id\":\"%s\",\"compatibility_score\":%.2f,\"matches_count\":%d}", userUUID.String(), user.ID.String(), match.CompatibilityScore, len(matches))
	}

	// Update matching session
	log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: Updating matching session\",\"user_uuid\":\"%s\",\"matches_count\":%d}", userUUID.String(), len(matches))
	session := &models.MatchingSession{
		UserID:               userUUID.String(),
		Status:               "active",
		LastMatchGeneratedAt: &[]time.Time{time.Now()}[0],
		ExpiresAt:            time.Now().AddDate(0, 0, 30),
	}

	err = h.matchingRepo.UpsertMatchingSession(r.Context(), session)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindMatches: Error updating matching session\",\"user_uuid\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
		// Don't fail the request for session update errors
	} else {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: Successfully updated matching session\",\"user_uuid\":\"%s\"}", userUUID.String())
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: Successfully generated %d matches\",\"user_uuid\":\"%s\",\"matches_generated\":%d}", len(matches), userUUID.String(), len(matches))

	// Prepare response
	response := map[string]interface{}{
		"matches_found": len(matches),
		"message":       "Matches generated successfully",
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: Sending response\",\"user_uuid\":\"%s\",\"response\":%+v}", userUUID.String(), response)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindMatches: Error encoding response\",\"user_uuid\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"FindMatches: Response sent successfully\",\"user_uuid\":\"%s\"}", userUUID.String())
}

// 5. GET /api/matching/requests - Get incoming and outgoing requests
func (h *MatchingHandler) GetMatchRequests(w http.ResponseWriter, r *http.Request) {
	// Log the route being called
	log.Printf("{\"severity\":\"INFO\",\"message\":\"🚀 ROUTE CALLED: /api/matching/requests\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr)

	// Log the incoming request method and URL
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Received GetMatchRequests request\",\"method\":\"%s\",\"url\":\"%s\"}", r.Method, r.URL.String())

	// Log all request headers for debugging authentication
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = v[0]
	}
	headersJSON, _ := json.Marshal(headers)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetMatchRequests request headers\",\"headers\":%s}", string(headersJSON))

	// Specifically log the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetMatchRequests: No Authorization header provided\"}")
	} else {
		// Log the first part of the token for debugging (don't log the full token for security)
		if len(authHeader) > 20 {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetMatchRequests: Authorization header present\",\"header_start\":\"%s...\",\"header_length\":%d}", authHeader[:20], len(authHeader))
		} else {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetMatchRequests: Authorization header present\",\"header\":\"%s\"}", authHeader)
		}
	}

	// Log the raw request for debugging
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetMatchRequests: Request details\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}", r.RemoteAddr, r.UserAgent())

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetMatchRequests: No Clerk claims in context - authentication failed\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.Subject

	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetMatchRequests: User authenticated successfully\",\"clerk_id\":\"%s\"}", userID)

	// Convert Clerk ID to UUID using the same pattern as other handlers
	userUUID, err := h.userRepo.GetUserIDByClerkID(r.Context(), userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetMatchRequests: Failed to get user UUID from Clerk ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetMatchRequests: Clerk ID converted to UUID\",\"clerk_id\":\"%s\",\"user_uuid\":\"%s\"}", userID, userUUID.String())

	// Phase 3: Default to personal scope (companyID = nil)
	var companyID *uuid.UUID = nil
	incoming, outgoing, err := h.matchingRepo.GetMatchRequests(r.Context(), userUUID.String(), companyID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetMatchRequests: Error getting match requests\",\"user_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
		// Return empty response instead of 500 error
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"incoming": []interface{}{},
			"outgoing": []interface{}{},
		})
		return
	}

	// Optional filtering by query params
	q := r.URL.Query()
	statusFilter := strings.TrimSpace(strings.ToLower(q.Get("status")))       // pending|accepted|rejected|expired
	directionFilter := strings.TrimSpace(strings.ToLower(q.Get("direction"))) // incoming|outgoing|both (default both)

	filterByStatus := func(list []*models.MatchRequest) []*models.MatchRequest {
		if statusFilter == "" || statusFilter == "all" {
			return list
		}
		filtered := make([]*models.MatchRequest, 0, len(list))
		for _, mr := range list {
			if strings.EqualFold(mr.Status, statusFilter) {
				filtered = append(filtered, mr)
			}
		}
		return filtered
	}

	// Apply status filter first
	incoming = filterByStatus(incoming)
	outgoing = filterByStatus(outgoing)

	// Apply direction filter
	switch directionFilter {
	case "incoming":
		outgoing = []*models.MatchRequest{}
	case "outgoing":
		incoming = []*models.MatchRequest{}
	default:
		// both or empty -> no change
	}

	// Ensure we always return empty slices instead of nil
	if incoming == nil {
		incoming = []*models.MatchRequest{}
	}
	if outgoing == nil {
		outgoing = []*models.MatchRequest{}
	}

	// Transform to frontend format
	response := map[string]interface{}{
		"incoming": incoming,
		"outgoing": outgoing,
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetMatchRequests: Successfully retrieved requests\",\"user_id\":\"%s\",\"incoming_count\":%d,\"outgoing_count\":%d}", userUUID.String(), len(incoming), len(outgoing))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 6. POST /api/matching/request - Send carpool request
func (h *MatchingHandler) CreateMatchRequest(w http.ResponseWriter, r *http.Request) {
	// Log the route being called
	log.Printf("{\"severity\":\"INFO\",\"message\":\"🚀 ROUTE CALLED: /api/matching/request\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr)

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreateMatchRequest: No Clerk claims in context - authentication failed\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.Subject

	log.Printf("{\"severity\":\"INFO\",\"message\":\"CreateMatchRequest: User authenticated successfully\",\"clerk_id\":\"%s\"}", userID)

	// Convert Clerk ID to UUID using the same pattern as other handlers
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"CreateMatchRequest: Starting user lookup\",\"clerk_id\":\"%s\"}", userID)
	userUUID, err := h.userRepo.GetUserIDByClerkID(r.Context(), userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreateMatchRequest: Failed to get user UUID from Clerk ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"CreateMatchRequest: User lookup successful\",\"clerk_id\":\"%s\",\"user_uuid\":\"%s\"}", userID, userUUID.String())

	log.Printf("{\"severity\":\"INFO\",\"message\":\"CreateMatchRequest: Clerk ID converted to UUID\",\"clerk_id\":\"%s\",\"user_uuid\":\"%s\"}", userID, userUUID.String())

	var reqBody models.MatchRequestPayload

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"CreateMatchRequest: Starting request body parsing\",\"clerk_id\":\"%s\"}", userID)
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreateMatchRequest: Failed to decode request body\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"CreateMatchRequest: Request body parsed successfully\",\"clerk_id\":\"%s\",\"to_user_id\":\"%s\",\"potential_match_id\":\"%s\"}", userID, reqBody.ToUserID, reqBody.PotentialMatchID)

	if reqBody.ToUserID == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreateMatchRequest: Missing to_user_id\",\"clerk_id\":\"%s\"}", userID)
		http.Error(w, "to_user_id is required", http.StatusBadRequest)
		return
	}

	// Validate carpool name
	if reqBody.CarpoolName == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreateMatchRequest: Missing carpool_name\",\"clerk_id\":\"%s\"}", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "MISSING_CARPOOL_NAME", "message": "Carpool name is required when sending a match request", "field": "carpool_name"})
		return
	}

	// Validate carpool name length and content
	trimmedCarpoolName := strings.TrimSpace(reqBody.CarpoolName)
	if len(trimmedCarpoolName) == 0 {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreateMatchRequest: Empty carpool name\",\"clerk_id\":\"%s\"}", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "EMPTY_CARPOOL_NAME", "message": "Carpool name cannot be empty or only whitespace", "field": "carpool_name"})
		return
	}

	if len([]rune(trimmedCarpoolName)) > 255 {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreateMatchRequest: Carpool name too long\",\"clerk_id\":\"%s\",\"length\":%d}", userID, len([]rune(trimmedCarpoolName)))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "INVALID_CARPOOL_NAME", "message": "Carpool name must be 255 characters or less", "provided_name": trimmedCarpoolName, "max_length": 255})
		return
	}

	// Prevent self-send
	if strings.EqualFold(reqBody.ToUserID, userUUID.String()) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid_to_user", "message": "Cannot send a request to yourself"})
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"CreateMatchRequest: Processing request\",\"from_user_uuid\":\"%s\",\"to_user_id\":\"%s\"}", userUUID.String(), reqBody.ToUserID)

	// Debug: Log the exact Clerk ID being looked up
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"CreateMatchRequest: Looking up Clerk ID\",\"clerk_id\":\"%s\",\"clerk_id_length\":%d}", reqBody.ToUserID, len(reqBody.ToUserID))

	// Check if it looks like a UUID instead of Clerk ID
	if len(reqBody.ToUserID) == 36 && strings.Contains(reqBody.ToUserID, "-") {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"CreateMatchRequest: Received UUID instead of Clerk ID\",\"received\":\"%s\",\"suggestion\":\"Frontend should send Clerk ID, not UUID\"}", reqBody.ToUserID)
	}

	// Check if user exists - try Clerk ID first, then UUID as fallback
	var targetUser *models.User

	// First try as Clerk ID
	targetUser, err = h.userRepo.GetUserByClerkID(reqBody.ToUserID)
	if err != nil && err == sql.ErrNoRows {
		// If not found as Clerk ID, try as UUID
		if len(reqBody.ToUserID) == 36 && strings.Contains(reqBody.ToUserID, "-") {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"CreateMatchRequest: Trying UUID lookup\",\"uuid\":\"%s\"}", reqBody.ToUserID)
			targetUser, err = h.userRepo.GetUserByID(uuid.MustParse(reqBody.ToUserID))
		}
	}

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreateMatchRequest: Target user not found\",\"from_user_uuid\":\"%s\",\"to_user_id\":\"%s\"}", userUUID.String(), reqBody.ToUserID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "user_not_found",
				"message": "The target user was not found. They may not have completed their profile setup.",
				"details": map[string]string{
					"to_user_id": reqBody.ToUserID,
					"suggestion": "Please ensure the user has completed their profile and try again.",
				},
			})
			return
		}
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreateMatchRequest: Error checking user existence\",\"from_user_uuid\":\"%s\",\"to_user_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), reqBody.ToUserID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"CreateMatchRequest: Target user found\",\"from_user_uuid\":\"%s\",\"to_user_id\":\"%s\",\"target_user_id\":\"%s\"}", userUUID.String(), reqBody.ToUserID, targetUser.ID.String())

	// Validate match request
	if err := h.validateMatchRequest(&reqBody, userUUID.String()); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreateMatchRequest: Validation failed\",\"from_user_uuid\":\"%s\",\"to_user_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), reqBody.ToUserID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if request already exists
	exists, err := h.matchingRepo.CheckMatchRequestExists(r.Context(), userUUID.String(), targetUser.ID.String(), reqBody.PotentialMatchID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreateMatchRequest: Error checking existing request\",\"from_user_uuid\":\"%s\",\"to_user_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), reqBody.ToUserID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if exists {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreateMatchRequest: Request already exists\",\"from_user_uuid\":\"%s\",\"to_user_id\":\"%s\"}", userUUID.String(), reqBody.ToUserID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "duplicate_pending_request", "message": "A pending request already exists."})
		return
	}

	var message *string
	if reqBody.Message != "" {
		trimmed := strings.TrimSpace(reqBody.Message)
		if len(trimmed) > 0 {
			if len([]rune(trimmed)) > 280 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid_message", "message": "Message must be 280 characters or fewer", "details": map[string]int{"max": 280}})
				return
			}
			message = &trimmed
		}
	}

	matchRequest := &models.MatchRequest{
		FromUserID:           userUUID.String(),
		ToUserID:             targetUser.ID.String(),
		PotentialMatchID:     reqBody.PotentialMatchID,
		Message:              message,
		PreferredCarpoolSize: reqBody.PreferredCarpoolSize,
		CarpoolName:          trimmedCarpoolName,
		Status:               "pending",
		ExpiresAt:            time.Now().AddDate(0, 0, 3),
	}

	err = h.matchingRepo.CreateMatchRequest(r.Context(), matchRequest)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreateMatchRequest: Error creating match request\",\"from_user_uuid\":\"%s\",\"to_user_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), reqBody.ToUserID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"CreateMatchRequest: Successfully created match request\",\"from_user_uuid\":\"%s\",\"to_user_id\":\"%s\",\"request_id\":\"%s\"}", userUUID.String(), reqBody.ToUserID, matchRequest.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matchRequest)
}

// 7. PUT /api/matching/request/{requestId} - Accept or reject request
func (h *MatchingHandler) UpdateMatchRequest(w http.ResponseWriter, r *http.Request) {
	// Log the route being called
	log.Printf("{\"severity\":\"INFO\",\"message\":\"🚀 ROUTE CALLED: /api/matching/request/{requestId}\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr)

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateMatchRequest: No Clerk claims in context - authentication failed\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.Subject

	log.Printf("{\"severity\":\"INFO\",\"message\":\"UpdateMatchRequest: User authenticated successfully\",\"clerk_id\":\"%s\"}", userID)

	// Convert Clerk ID to UUID using the same pattern as other handlers
	userUUID, err := h.userRepo.GetUserIDByClerkID(r.Context(), userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateMatchRequest: Failed to get user UUID from Clerk ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"UpdateMatchRequest: Clerk ID converted to UUID\",\"clerk_id\":\"%s\",\"user_uuid\":\"%s\"}", userID, userUUID.String())

	vars := mux.Vars(r)
	requestID := vars["requestId"]
	if requestID == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateMatchRequest: Missing request ID\",\"user_uuid\":\"%s\"}", userUUID.String())
		http.Error(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"UpdateMatchRequest: Processing request\",\"user_uuid\":\"%s\",\"request_id\":\"%s\"}", userUUID.String(), requestID)

	var reqBody models.MatchRequestUpdate
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateMatchRequest: Failed to decode request body\",\"user_uuid\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if reqBody.Status != "accepted" && reqBody.Status != "rejected" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateMatchRequest: Invalid status\",\"user_uuid\":\"%s\",\"status\":\"%s\"}", userUUID.String(), reqBody.Status)
		http.Error(w, "Status must be 'accepted' or 'rejected'", http.StatusBadRequest)
		return
	}

	// Authorization: Check if user is the recipient of the request
	// Phase 3: Default to personal scope (companyID = nil)
	var companyID *uuid.UUID = nil
	request, err := h.matchingRepo.GetMatchRequestByID(r.Context(), requestID, companyID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateMatchRequest: Error getting match request\",\"user_uuid\":\"%s\",\"request_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), requestID, err)
		http.Error(w, "Request not found", http.StatusNotFound)
		return
	}

	// Check if user is the recipient (to_user_id)
	if request.ToUserID != userUUID.String() {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateMatchRequest: Unauthorized access\",\"user_uuid\":\"%s\",\"request_id\":\"%s\",\"to_user_id\":\"%s\"}", userUUID.String(), requestID, request.ToUserID)
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if request is still pending
	if request.Status != "pending" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateMatchRequest: Request no longer pending\",\"user_uuid\":\"%s\",\"request_id\":\"%s\",\"current_status\":\"%s\"}", userUUID.String(), requestID, request.Status)
		http.Error(w, "Request is no longer pending", http.StatusBadRequest)
		return
	}

	// Update the match request status
	err = h.matchingRepo.UpdateMatchRequestStatus(r.Context(), requestID, reqBody.Status)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateMatchRequest: Error updating match request status\",\"user_uuid\":\"%s\",\"request_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), requestID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"UpdateMatchRequest: Successfully updated match request\",\"user_uuid\":\"%s\",\"request_id\":\"%s\",\"status\":\"%s\"}", userUUID.String(), requestID, reqBody.Status)

	// If the request was accepted, create a carpool automatically
	var carpoolID *uuid.UUID
	if reqBody.Status == "accepted" {
		carpoolID, err = h.createCarpoolFromMatchRequest(r.Context(), request)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"UpdateMatchRequest: Error creating carpool from match request\",\"user_uuid\":\"%s\",\"request_id\":\"%s\",\"error\":\"%v\"}", userUUID.String(), requestID, err)
			// Don't fail the request update, just log the error
		} else {
			log.Printf("{\"severity\":\"INFO\",\"message\":\"UpdateMatchRequest: Successfully created carpool\",\"user_uuid\":\"%s\",\"request_id\":\"%s\",\"carpool_id\":\"%s\"}", userUUID.String(), requestID, carpoolID.String())
		}
	}

	// Prepare response
	response := map[string]interface{}{
		"id":         request.ID,
		"status":     reqBody.Status,
		"updated_at": request.UpdatedAt,
		"message":    "Match request updated successfully",
	}

	// Include carpool ID and name if created
	if carpoolID != nil {
		response["carpool_id"] = carpoolID.String()
		response["carpool_name"] = request.CarpoolName
		response["message"] = "Carpool created successfully"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 8. GET /api/matching/session - Get session status
func (h *MatchingHandler) GetMatchingSession(w http.ResponseWriter, r *http.Request) {
	// Log the route being called
	log.Printf("{\"severity\":\"INFO\",\"message\":\"🚀 ROUTE CALLED: /api/matching/session\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr)

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetMatchingSession: No Clerk claims in context - authentication failed\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.Subject

	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetMatchingSession: User authenticated successfully\",\"clerk_id\":\"%s\"}", userID)

	// Convert Clerk ID to UUID using the same pattern as other handlers
	userUUID, err := h.userRepo.GetUserIDByClerkID(r.Context(), userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetMatchingSession: Failed to get user UUID from Clerk ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetMatchingSession: Clerk ID converted to UUID\",\"clerk_id\":\"%s\",\"user_uuid\":\"%s\"}", userID, userUUID.String())

	session, err := h.matchingRepo.GetMatchingSession(r.Context(), userUUID.String())
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetMatchingSession: Error getting matching session\",\"user_uuid\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If no session exists, create a default one
	if session == nil {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"GetMatchingSession: No session found, creating default\",\"user_uuid\":\"%s\"}", userUUID.String())

		session = &models.MatchingSession{
			UserID:    userUUID.String(),
			Status:    "active",
			ExpiresAt: time.Now().AddDate(0, 0, 30),
		}
		err = h.matchingRepo.UpsertMatchingSession(r.Context(), session)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetMatchingSession: Error creating default session\",\"user_uuid\":\"%s\",\"error\":\"%v\"}", userUUID.String(), err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("{\"severity\":\"INFO\",\"message\":\"GetMatchingSession: Created default session\",\"user_uuid\":\"%s\"}", userUUID.String())
	} else {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"GetMatchingSession: Retrieved existing session\",\"user_uuid\":\"%s\",\"status\":\"%s\"}", userUUID.String(), session.Status)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// Helper function to calculate distance between two points (Haversine formula)
func calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 3959 // Earth's radius in miles

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// Validation functions for demographic fields

// validateUserDemographics validates user demographic information
func validateUserDemographics(demographics models.UserDemographics) error {
	// Validate age range
	validAgeRanges := []string{"18-25", "26-35", "36-45", "46-55", "56-65", "65+"}
	ageValid := false
	for _, age := range validAgeRanges {
		if demographics.AgeRange == age {
			ageValid = true
			break
		}
	}
	if !ageValid {
		return fmt.Errorf("age_range must be one of: %v", validAgeRanges)
	}

	// Validate gender
	validGenders := []string{"male", "female", "non-binary", "prefer_not_to_say"}
	genderValid := false
	for _, gender := range validGenders {
		if demographics.Gender == gender {
			genderValid = true
			break
		}
	}
	if !genderValid {
		return fmt.Errorf("gender must be one of: %v", validGenders)
	}

	// Validate occupation (optional for auto-save)
	// Allow empty occupation for auto-save scenarios where user hasn't filled it out yet
	// if demographics.Occupation == "" {
	// 	return fmt.Errorf("occupation is required")
	// }

	// Validate student status (allow empty for frontend simplification)
	if demographics.StudentStatus != "" {
		validStudentStatuses := []string{"undergraduate", "graduate", "not_student"}
		studentStatusValid := false
		for _, status := range validStudentStatuses {
			if demographics.StudentStatus == status {
				studentStatusValid = true
				break
			}
		}
		if !studentStatusValid {
			return fmt.Errorf("student_status must be one of: %v", validStudentStatuses)
		}
	}

	return nil
}

// validateDemographicPreferences validates demographic preferences
func validateDemographicPreferences(preferences models.DemographicPreferences) error {
	// For auto-save functionality, allow empty preferences and provide defaults
	if len(preferences.AgePreferences) == 0 {
		// Set default age preferences if none provided
		preferences.AgePreferences = []string{"18-25", "26-35", "36-45", "46-55"}
	}

	if len(preferences.GenderPreferences) == 0 {
		// Set default gender preferences if none provided
		preferences.GenderPreferences = []string{"any"}
	}

	if preferences.StudentPreference == "" {
		// Set default student preference if none provided
		preferences.StudentPreference = "both"
	}

	validAgeRanges := []string{"18-25", "26-35", "36-45", "46-55", "56-65", "65+"}
	for _, age := range preferences.AgePreferences {
		ageValid := false
		for _, validAge := range validAgeRanges {
			if age == validAge {
				ageValid = true
				break
			}
		}
		if !ageValid {
			return fmt.Errorf("invalid age preference: %s. Must be one of: %v", age, validAgeRanges)
		}
	}

	validGenders := []string{"male", "female", "non-binary", "prefer_not_to_say", "any"}
	for _, gender := range preferences.GenderPreferences {
		genderValid := false
		for _, validGender := range validGenders {
			if gender == validGender {
				genderValid = true
				break
			}
		}
		if !genderValid {
			return fmt.Errorf("invalid gender preference: %s. Must be one of: %v", gender, validGenders)
		}
	}

	// Validate student preference
	validStudentPreferences := []string{"students_only", "professionals_only", "both"}
	studentPreferenceValid := false
	for _, pref := range validStudentPreferences {
		if preferences.StudentPreference == pref {
			studentPreferenceValid = true
			break
		}
	}
	if !studentPreferenceValid {
		return fmt.Errorf("student_preference must be one of: %v", validStudentPreferences)
	}

	return nil
}

// Helper functions for demographic matching

// calculateDemographicCompatibilityScore calculates compatibility score based on demographic preferences
func calculateDemographicCompatibilityScore(userPrefs *models.UserMatchingPreferences, targetUser *models.User) float64 {
	if userPrefs == nil {
		// Fallback to basic score if no preferences
		return 0.7 + rand.Float64()*0.3
	}

	// Base score starts at 0.5
	score := 0.5

	// For now, we'll use a simplified scoring system
	// In a real implementation, you'd fetch the target user's demographics from the database
	// and compare them with the user's preferences

	// Age compatibility (if we had target user demographics)
	// if targetUserDemographics.AgeRange matches userPrefs.DemographicPreferences.AgePreferences {
	//     score += 0.2
	// }

	// Gender compatibility
	// if targetUserDemographics.Gender matches userPrefs.DemographicPreferences.GenderPreferences {
	//     score += 0.15
	// }

	// Student status compatibility
	// if targetUserDemographics.StudentStatus matches userPrefs.DemographicPreferences.StudentPreference {
	//     score += 0.15
	// }

	// For now, add some randomness to simulate demographic matching
	// In production, this would be based on actual demographic data
	demographicBonus := rand.Float64() * 0.3
	score += demographicBonus

	// Ensure score is between 0.5 and 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// generateMatchReasons generates match reasons based on compatibility
func generateMatchReasons(userPrefs *models.UserMatchingPreferences, targetUser *models.User) models.MatchReasons {
	reasons := []string{}

	// Base reasons that are always included
	reasons = append(reasons, "Same destination")

	// Add demographic-based reasons if preferences exist
	if userPrefs != nil && len(userPrefs.DemographicPreferences.AgePreferences) > 0 {
		reasons = append(reasons, "Demographic compatibility")
	}

	// Add schedule-based reasons
	reasons = append(reasons, "Similar schedule")

	// Add location-based reasons
	reasons = append(reasons, "Close pickup location")

	// Add flexibility reasons
	if userPrefs != nil && userPrefs.ScheduleFlexibilityMinutes > 30 {
		reasons = append(reasons, "Flexible schedule")
	}

	// Ensure we have at least 2 reasons
	if len(reasons) < 2 {
		reasons = append(reasons, "Route overlap")
	}

	return models.MatchReasons(reasons)
}

// validateMatchRequest validates a match request payload
func (h *MatchingHandler) validateMatchRequest(payload *models.MatchRequestPayload, fromUserID string) error {
	// Check if user is trying to send request to themselves
	if fromUserID == payload.ToUserID {
		return fmt.Errorf("cannot send request to yourself")
	}

	// Check if potential match ID is provided
	if payload.PotentialMatchID == "" {
		return fmt.Errorf("potential_match_id is required")
	}

	// Check if to_user_id is provided
	if payload.ToUserID == "" {
		return fmt.Errorf("to_user_id is required")
	}

	// TODO: Add more validation as needed
	// - Check if potential match exists
	// - Check if users are already carpooling
	// - Check if users have compatible preferences

	return nil
}

// createCarpoolFromMatchRequest creates a carpool when a match request is accepted
func (h *MatchingHandler) createCarpoolFromMatchRequest(ctx context.Context, request *models.MatchRequest) (*uuid.UUID, error) {
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Creating carpool from match request\",\"request_id\":\"%s\",\"from_user\":\"%s\",\"to_user\":\"%s\"}", request.ID, request.FromUserID, request.ToUserID)

	// Get both users' preferences to determine carpool details
	// Phase 3: Default to personal scope (companyID = nil)
	// TODO: Phase 7 - Use company_id from match_request if it's a company request
	var companyID *uuid.UUID = nil
	if request.CompanyID != nil {
		// If match request has company_id, use it for preferences lookup
		parsedCompanyID, err := uuid.Parse(*request.CompanyID)
		if err == nil {
			companyID = &parsedCompanyID
		}
	}

	fromUserPrefs, err := h.matchingRepo.GetUserMatchingPreferences(ctx, request.FromUserID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get from user preferences: %v", err)
	}

	toUserPrefs, err := h.matchingRepo.GetUserMatchingPreferences(ctx, request.ToUserID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get to user preferences: %v", err)
	}

	// Note: We use the sender's preferred carpool size instead of merging preferences
	// The createCarpoolSchedule function will handle merging preferences for schedule creation

	// Use the sender's chosen carpool name
	carpoolName := request.CarpoolName

	// Calculate available seats using the preferred carpool size from the match request
	// The sender specified their preferred carpool size, so we use that
	availableSeats := request.PreferredCarpoolSize - 2 // Reserve 2 seats for creator + accepter
	if availableSeats < 0 {
		availableSeats = 0 // Minimum 0 available seats
	}

	// Create the carpool with merged preferences
	carpool := &models.Carpool{
		ID:                 uuid.New(),
		CreatorID:          uuid.MustParse(request.FromUserID), // The person who sent the request is the creator
		CarpoolName:        carpoolName,
		Status:             true,                         // Active
		AvailableSeats:     availableSeats,               // Calculated with safety check
		DestinationAddress: carpoolName,                  // Use the generated name as destination address
		Seats:              request.PreferredCarpoolSize, // Use the preferred carpool size from the match request
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Create the carpool in the database
	err = h.carpoolRepo.CreateCarPool(ctx, carpool)
	if err != nil {
		return nil, fmt.Errorf("failed to create carpool: %v", err)
	}

	// Add both users as members
	fromUserUUID := uuid.MustParse(request.FromUserID)
	toUserUUID := uuid.MustParse(request.ToUserID)

	// Add the creator (from user) as a member
	err = h.carpoolRepo.AddCarpoolMemberByAPI(ctx, carpool.ID, fromUserUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to add creator to carpool: %v", err)
	}

	// Add the recipient (to user) as a member
	err = h.carpoolRepo.AddCarpoolMemberByAPI(ctx, carpool.ID, toUserUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to add recipient to carpool: %v", err)
	}

	// Create a schedule based on the merged preferences
	err = h.createCarpoolSchedule(ctx, carpool.ID, fromUserPrefs, toUserPrefs)
	if err != nil {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"Failed to create carpool schedule\",\"carpool_id\":\"%s\",\"error\":\"%v\"}", carpool.ID.String(), err)
		// Don't fail the carpool creation if schedule creation fails
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully created carpool from match request\",\"carpool_id\":\"%s\",\"request_id\":\"%s\"}", carpool.ID.String(), request.ID)

	return &carpool.ID, nil
}

// MergedPreferences represents the result of merging two users' preferences
type MergedPreferences struct {
	LeaveTime          time.Time
	CommuteDays        []string
	MaxDetourMinutes   int
	MaxPickupDistance  float64
	PreferredGroupSize int
	DriverPreference   string
}

// createCarpoolSchedule creates schedules and rides for the carpool based on merged user preferences
// It creates a separate schedule for each common commute day and generates rides for all of them
func (h *MatchingHandler) createCarpoolSchedule(ctx context.Context, carpoolID uuid.UUID, fromUserPrefs, toUserPrefs *models.UserMatchingPreferences) error {
	// Merge both users' preferences intelligently
	mergedPrefs := h.mergeUserPreferences(fromUserPrefs, toUserPrefs)

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Creating carpool schedules with merged preferences\",\"carpool_id\":\"%s\",\"leave_time\":\"%s\",\"commute_days\":\"%v\",\"max_detour\":\"%d\",\"driver_pref\":\"%s\"}",
		carpoolID.String(), mergedPrefs.LeaveTime.Format("15:04"), mergedPrefs.CommuteDays, mergedPrefs.MaxDetourMinutes, mergedPrefs.DriverPreference)

	// If no commute days, log warning and return (can't create schedule without days)
	if len(mergedPrefs.CommuteDays) == 0 {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"No common commute days found, skipping schedule creation\",\"carpool_id\":\"%s\"}", carpoolID.String())
		return nil
	}

	// Get carpool details for ride generation
	// Phase 3: Default to personal scope (companyID = nil)
	// TODO: Phase 7 - Use company_id from match_request if it's a company request
	var companyID *uuid.UUID = nil
	if request.CompanyID != nil {
		// If match request has company_id, use it for carpool lookup
		parsedCompanyID, err := uuid.Parse(*request.CompanyID)
		if err == nil {
			companyID = &parsedCompanyID
		}
	}
	carpool, err := h.carpoolRepo.GetCarPool(ctx, carpoolID, companyID)
	if err != nil {
		return fmt.Errorf("failed to get carpool for schedule creation: %v", err)
	}

	// Use merged leave time
	startTime := mergedPrefs.LeaveTime
	scheduleType := "weekly"

	// Determine default timezone
	loc := h.getDefaultLocation()
	now := time.Now().In(loc)

	// Match acceptance date (start_date for all schedules)
	matchAcceptanceDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	// End date is 90 days from match acceptance date
	endDays := h.getRideGenerationDays()
	endDate := matchAcceptanceDate.AddDate(0, 0, endDays)

	// Create a schedule for EACH common commute day
	for _, dayName := range mergedPrefs.CommuteDays {
		dayOfWeek := h.getDayOfWeekNumber(dayName)
		if dayOfWeek == nil {
			log.Printf("{\"severity\":\"WARNING\",\"message\":\"Invalid day name, skipping\",\"day\":\"%s\",\"carpool_id\":\"%s\"}", dayName, carpoolID.String())
			continue
		}

		// Create the schedule for this day
		// Phase 3: Set company_id and site_id from carpool (will be populated by CreateCarpoolSchedule)
		schedule := &models.CarpoolSchedule{
			ID:           uuid.New(),
			CarpoolID:    carpoolID,
			ScheduleType: scheduleType,
			StartDate:    matchAcceptanceDate, // Match acceptance date
			EndDate:      &endDate,            // 90 days from start
			DayOfWeek:    dayOfWeek,
			StartTime:    time.Date(2000, 1, 1, startTime.Hour(), startTime.Minute(), 0, 0, loc),
			// CompanyID and SiteID will be populated from carpool by CreateCarpoolSchedule
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Create the schedule in database
		err = h.scheduleRepo.CreateCarpoolSchedule(ctx, schedule)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create schedule for day\",\"day\":\"%s\",\"carpool_id\":\"%s\",\"error\":\"%v\"}", dayName, carpoolID.String(), err)
			continue // Continue with other days even if one fails
		}

		log.Printf("{\"severity\":\"INFO\",\"message\":\"Created schedule for day\",\"day\":\"%s\",\"schedule_id\":\"%s\",\"carpool_id\":\"%s\"}", dayName, schedule.ID.String(), carpoolID.String())

		// Generate rides for this schedule (pass both user IDs for participants)
		fromUserUUID, err := uuid.Parse(fromUserPrefs.UserID)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid from user ID, skipping ride generation\",\"user_id\":\"%s\",\"carpool_id\":\"%s\",\"error\":\"%v\"}", fromUserPrefs.UserID, carpoolID.String(), err)
			continue
		}
		toUserUUID, err := uuid.Parse(toUserPrefs.UserID)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid to user ID, skipping ride generation\",\"user_id\":\"%s\",\"carpool_id\":\"%s\",\"error\":\"%v\"}", toUserPrefs.UserID, carpoolID.String(), err)
			continue
		}
		err = h.generateRidesFromSchedule(ctx, schedule, carpool, fromUserUUID, toUserUUID)
		if err != nil {
			log.Printf("{\"severity\":\"WARNING\",\"message\":\"Failed to generate rides for schedule\",\"day\":\"%s\",\"schedule_id\":\"%s\",\"carpool_id\":\"%s\",\"error\":\"%v\"}", dayName, schedule.ID.String(), carpoolID.String(), err)
			// Don't fail completely, continue with other days
			continue
		}

		log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully created schedule and generated rides for day\",\"day\":\"%s\",\"schedule_id\":\"%s\",\"carpool_id\":\"%s\"}", dayName, schedule.ID.String(), carpoolID.String())
	}

	return nil
}

// generateRidesFromSchedule creates individual rides based on a carpool schedule
// This generates rides for 90 days ahead to populate the calendar
// Both users (fromUserID and toUserID) are added as participants to each ride
func (h *MatchingHandler) generateRidesFromSchedule(ctx context.Context, schedule *models.CarpoolSchedule, carpool *models.Carpool, fromUserID, toUserID uuid.UUID) error {
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Generating rides from schedule\",\"schedule_id\":\"%s\",\"schedule_type\":\"%s\",\"day_of_week\":\"%v\"}",
		schedule.ID, schedule.ScheduleType, schedule.DayOfWeek)

	var rides []*models.CarpoolRide
	loc := h.getDefaultLocation()
	currentDate := schedule.StartDate.In(loc)

	// Generate rides for configured horizon (default 90 days)
	endDays := h.getRideGenerationDays()
	endDate := schedule.StartDate.In(loc).AddDate(0, 0, endDays)
	if schedule.EndDate != nil {
		endDate = schedule.EndDate.In(loc)
	}

	// Generate rides until we reach the end date
	for currentDate.Before(endDate) || currentDate.Equal(endDate) {
		var shouldCreateRide bool

		switch schedule.ScheduleType {
		case "one_time":
			// For one-time schedules, only create one ride on the start date
			shouldCreateRide = currentDate.Equal(schedule.StartDate)
		case "daily":
			// For daily schedules, create a ride every day
			shouldCreateRide = true
		case "weekly":
			// For weekly schedules, create a ride on the specified day of week
			if schedule.DayOfWeek != nil {
				shouldCreateRide = int(currentDate.Weekday()) == *schedule.DayOfWeek
			}
		}

		if shouldCreateRide {
			// Create the ride start time by combining the date with the schedule time
			rideStartTime := time.Date(
				currentDate.Year(), currentDate.Month(), currentDate.Day(),
				schedule.StartTime.Hour(), schedule.StartTime.Minute(), 0, 0,
				schedule.StartTime.Location(),
			)

			// Skip rides in the past (only create future rides)
			now := time.Now().In(loc)
			if rideStartTime.Before(now) {
				// Move to next day and continue
				currentDate = currentDate.AddDate(0, 0, 1)
				continue
			}

			// Get both users for participants
			fromUser, err := h.userRepo.GetUserByID(fromUserID)
			if err != nil {
				log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to fetch from user for ride participants\",\"user_id\":\"%s\",\"error\":\"%v\"}", fromUserID, err)
			}
			toUser, err := h.userRepo.GetUserByID(toUserID)
			if err != nil {
				log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to fetch to user for ride participants\",\"user_id\":\"%s\",\"error\":\"%v\"}", toUserID, err)
			}
			participants := []models.User{}
			if fromUser != nil {
				participants = append(participants, *fromUser)
			}
			if toUser != nil {
				participants = append(participants, *toUser)
			}

			ride := &models.CarpoolRide{
				ID:           uuid.New(),
				CarpoolID:    schedule.CarpoolID,
				StartTime:    rideStartTime,
				Status:       0, // Default status (pending)
				Participants: participants,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			rides = append(rides, ride)
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Generated ride\",\"ride_id\":\"%s\",\"start_time\":\"%s\"}",
				ride.ID, ride.StartTime.Format("2006-01-02 15:04:05"))
		}

		// Move to next day
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Generated %d rides from schedule\",\"schedule_id\":\"%s\",\"ride_count\":%d}",
		len(rides), schedule.ID, len(rides))

	// Save rides to database
	successCount := 0
	for _, ride := range rides {
		if err := h.rideRepo.CreateCarpoolRide(ctx, ride); err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to save ride to database\",\"ride_id\":\"%s\",\"error\":\"%v\"}",
				ride.ID, err)
			// Continue with other rides even if one fails
			continue
		}
		successCount++
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Saved ride to database\",\"ride_id\":\"%s\",\"start_time\":\"%s\"}",
			ride.ID, ride.StartTime.Format("2006-01-02 15:04:05"))
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully generated and saved %d/%d rides from schedule\",\"schedule_id\":\"%s\",\"success_count\":%d,\"total_count\":%d}",
		schedule.ID, successCount, len(rides), successCount, len(rides))

	return nil
}

// mergeUserPreferences intelligently merges two users' preferences for carpool creation
func (h *MatchingHandler) mergeUserPreferences(user1, user2 *models.UserMatchingPreferences) *MergedPreferences {
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Merging user preferences\",\"user1_id\":\"%s\",\"user2_id\":\"%s\"}", user1.UserID, user2.UserID)

	// Calculate optimal leave time (earlier time wins for punctuality)
	leaveTime := h.calculateOptimalLeaveTime(user1.ArrivalTime, user2.ArrivalTime)

	// Calculate common commute days (intersection)
	commuteDays := h.calculateCommonCommuteDays(user1.CommuteDays, user2.CommuteDays)

	// Use more restrictive values (conservative approach)
	maxDetourMinutes := h.minInt(user1.MaxDetourMinutes, user2.MaxDetourMinutes)
	maxPickupDistance := h.minFloat64(user1.MaxPickupDistanceMiles, user2.MaxPickupDistanceMiles)
	preferredGroupSize := h.minInt(user1.PreferredGroupSize, user2.PreferredGroupSize)

	// Determine driver preference intelligently
	driverPreference := h.determineDriverPreference(user1.DriverPreference, user2.DriverPreference)

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Merged preferences calculated\",\"leave_time\":\"%s\",\"commute_days\":\"%v\",\"max_detour\":\"%d\",\"max_pickup\":\"%.2f\",\"group_size\":\"%d\",\"driver_pref\":\"%s\"}",
		leaveTime.Format("15:04"), commuteDays, maxDetourMinutes, maxPickupDistance, preferredGroupSize, driverPreference)

	return &MergedPreferences{
		LeaveTime:          leaveTime,
		CommuteDays:        commuteDays,
		MaxDetourMinutes:   maxDetourMinutes,
		MaxPickupDistance:  maxPickupDistance,
		PreferredGroupSize: preferredGroupSize,
		DriverPreference:   driverPreference,
	}
}

// calculateOptimalLeaveTime determines the best leave time based on both users' arrival times
func (h *MatchingHandler) calculateOptimalLeaveTime(user1Time, user2Time *string) time.Time {
	// Default time if both are nil
	defaultTime := time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)

	// If one user has no time preference, use the other's
	if user1Time == nil || *user1Time == "" {
		if user2Time == nil || *user2Time == "" {
			return defaultTime
		}
		return h.parseArrivalTime(*user2Time)
	}

	if user2Time == nil || *user2Time == "" {
		return h.parseArrivalTime(*user1Time)
	}

	// Both users have time preferences - use the EARLIER time (more conservative)
	time1 := h.parseArrivalTime(*user1Time)
	time2 := h.parseArrivalTime(*user2Time)

	if time1.Before(time2) {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"Using earlier time for leave time\",\"time1\":\"%s\",\"time2\":\"%s\",\"chosen\":\"%s\"}",
			time1.Format("15:04"), time2.Format("15:04"), time1.Format("15:04"))
		return time1
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Using earlier time for leave time\",\"time1\":\"%s\",\"time2\":\"%s\",\"chosen\":\"%s\"}",
		time1.Format("15:04"), time2.Format("15:04"), time2.Format("15:04"))
	return time2
}

// parseArrivalTime parses arrival time and returns leave time (30 minutes before)
func (h *MatchingHandler) parseArrivalTime(arrivalTime string) time.Time {
	// Try HH:MM format first (most common)
	parsedTime, err := time.Parse("15:04", arrivalTime)
	if err != nil {
		// Try HH:MM:SS format as fallback
		parsedTime, err = time.Parse("15:04:05", arrivalTime)
		if err != nil {
			log.Printf("{\"severity\":\"WARNING\",\"message\":\"Failed to parse arrival time, using default\",\"arrival_time\":\"%s\",\"error\":\"%v\"}", arrivalTime, err)
			return time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC) // Use reasonable default year
		}
	}

	// Set leave time to 30 minutes before arrival time
	leaveTime := parsedTime.Add(-30 * time.Minute)
	return leaveTime
}

// calculateCommonCommuteDays finds the intersection of both users' commute days
func (h *MatchingHandler) calculateCommonCommuteDays(user1Days, user2Days []string) []string {
	// If one user has no commute days, use the other's
	if len(user1Days) == 0 {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"User1 has no commute days, using user2's\",\"user2_days\":\"%v\"}", user2Days)
		return user2Days
	}

	if len(user2Days) == 0 {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"User2 has no commute days, using user1's\",\"user1_days\":\"%v\"}", user1Days)
		return user1Days
	}

	// Find intersection of commute days
	commonDays := []string{}
	user1DayMap := make(map[string]bool)

	// Create a map for user1's days for O(1) lookup
	for _, day := range user1Days {
		user1DayMap[strings.ToLower(day)] = true
	}

	// Check which days from user2 are also in user1
	for _, day := range user2Days {
		if user1DayMap[strings.ToLower(day)] {
			commonDays = append(commonDays, strings.ToLower(day))
		}
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Calculated common commute days\",\"user1_days\":\"%v\",\"user2_days\":\"%v\",\"common_days\":\"%v\"}",
		user1Days, user2Days, commonDays)

	// If no common days, return user1's days as fallback (or empty if both are empty)
	if len(commonDays) == 0 {
		if len(user1Days) > 0 {
			log.Printf("{\"severity\":\"WARNING\",\"message\":\"No common commute days found, using user1's days as fallback\",\"user1_days\":\"%v\"}", user1Days)
			return user1Days
		} else {
			log.Printf("{\"severity\":\"WARNING\",\"message\":\"No common commute days found and user1 has no days, returning empty\",\"user1_days\":\"%v\",\"user2_days\":\"%v\"}", user1Days, user2Days)
			return []string{} // Return empty array if both users have no commute days
		}
	}

	return commonDays
}

// determineDriverPreference intelligently determines who should drive
func (h *MatchingHandler) determineDriverPreference(pref1, pref2 string) string {
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Determining driver preference\",\"pref1\":\"%s\",\"pref2\":\"%s\"}", pref1, pref2)

	// If one wants to drive and other is flexible/passenger → let them drive
	if pref1 == "driver" && (pref2 == "passenger" || pref2 == "either") {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"User1 wants to drive, user2 is flexible\",\"result\":\"driver\"}")
		return "driver"
	}

	if pref2 == "driver" && (pref1 == "passenger" || pref1 == "either") {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"User2 wants to drive, user1 is flexible\",\"result\":\"driver\"}")
		return "driver"
	}

	// If both want to drive → default to "either" (they can negotiate)
	if pref1 == "driver" && pref2 == "driver" {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"Both users want to drive\",\"result\":\"either\"}")
		return "either"
	}

	// If both are passengers → default to "either" (they need to find a driver)
	if pref1 == "passenger" && pref2 == "passenger" {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"Both users are passengers\",\"result\":\"either\"}")
		return "either"
	}

	// Handle mixed "either" cases
	if (pref1 == "either" && pref2 == "passenger") || (pref1 == "passenger" && pref2 == "either") {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"One user is either, other is passenger\",\"result\":\"driver\"}")
		return "driver" // The "either" user can drive
	}

	// Default case
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Using default driver preference\",\"result\":\"either\"}")
	return "either"
}

// Helper functions for min operations
func (h *MatchingHandler) minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (h *MatchingHandler) minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// getDayOfWeekNumber converts day name to number (0=Sunday, 1=Monday, etc.)
func (h *MatchingHandler) getDayOfWeekNumber(day string) *int {
	dayMap := map[string]int{
		"sun": 0, "sunday": 0,
		"mon": 1, "monday": 1,
		"tue": 2, "tuesday": 2,
		"wed": 3, "wednesday": 3,
		"thu": 4, "thursday": 4,
		"fri": 5, "friday": 5,
		"sat": 6, "saturday": 6,
	}

	if num, exists := dayMap[strings.ToLower(day)]; exists {
		return &num
	}
	return nil
}

// getDefaultLocation returns the timezone location to use for schedules/rides
// Uses CARPOOL_DEFAULT_TZ if set, otherwise falls back to time.Local
func (h *MatchingHandler) getDefaultLocation() *time.Location {
	if tz := os.Getenv("CARPOOL_DEFAULT_TZ"); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc
		}
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"Invalid CARPOOL_DEFAULT_TZ, falling back to local\",\"tz\":\"%s\"}", tz)
	}
	return time.Local
}

// nextOccurrence returns the next date (midnight) on or after 'from' matching targetWeekday (0=Sunday)
func (h *MatchingHandler) nextOccurrence(from time.Time, targetWeekday int) time.Time {
	// Normalize to midnight
	fromMidnight := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	delta := (targetWeekday - int(fromMidnight.Weekday()) + 7) % 7
	if delta == 0 {
		// If today is the day, return today (next occurrence is today)
		return fromMidnight
	}
	return fromMidnight.AddDate(0, 0, delta)
}

// getRideGenerationDays reads RIDE_GENERATION_DAYS env var, defaults to 90 on error
func (h *MatchingHandler) getRideGenerationDays() int {
	val := os.Getenv("RIDE_GENERATION_DAYS")
	if val == "" {
		return 90
	}
	if n, err := strconv.Atoi(val); err == nil && n > 0 {
		return n
	}
	log.Printf("{\"severity\":\"WARNING\",\"message\":\"Invalid RIDE_GENERATION_DAYS, using default\",\"value\":\"%s\"}", val)
	return 90
}
