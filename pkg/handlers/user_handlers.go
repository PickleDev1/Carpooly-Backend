package handlers

import (
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"time"

	"car-backend/middleware"

	"database/sql"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type UserHandler struct {
	userRepo *repository.UserRepository
	//clerkClient clerk.Client
}

func NewUserHandler(userRepo *repository.UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
	}
}

// HandleWebhook processes Clerk webhooks for user events
func (h *UserHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Implement webhook handling for Clerk events
	// This is where you'll handle user creation/updates from Clerk
	// Example: user.created, user.updated, etc.
}

// GetProfile returns the user's profile
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	log.Printf("Starting get profile handler")
	ctx := r.Context()

	// Log request headers
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = v[0]
	}
	headersJSON, _ := json.Marshal(headers)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetProfile request headers\",\"headers\":%s}", string(headersJSON))

	claims, ok := clerk.SessionClaimsFromContext(ctx)
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Unauthorized: No Clerk claims in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Authenticated user\",\"clerk_id\":\"%s\"}", claims.Subject)

	// Convert Clerk ID to user UUID
	userID, err := h.userRepo.GetUserIDByClerkID(ctx, claims.Subject)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user UUID from Clerk ID\",\"clerk_id\":\"%s\",\"error\":%q}", claims.Subject, err.Error())
		http.Error(w, "Failed to get user profile", http.StatusInternalServerError)
		return
	}

	// Get user from our database using UUID
	user, err := h.userRepo.GetByID(ctx, userID.String())
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user profile\",\"user_id\":\"%s\",\"error\":%q}", userID.String(), err.Error())
		http.Error(w, "Failed to get user profile", http.StatusInternalServerError)
		return
	}

	userJSON, _ := json.Marshal(user)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Fetched user profile\",\"user\":%s}", string(userJSON))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// UpdateProfile updates the user's profile
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := clerk.SessionClaimsFromContext(ctx)
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Unauthorized: No Clerk claims in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Log request headers
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = v[0]
	}
	headersJSON, _ := json.Marshal(headers)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"UpdateProfile request headers\",\"headers\":%s}", string(headersJSON))

	// Log raw request body
	bodyBytes, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to dump request: %v\"}", err)
	} else {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Raw UpdateProfile request\",\"body\":%q}", string(bodyBytes))
	}

	var update models.UpdateUserProfile
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to decode UpdateUserProfile\",\"error\":%q}", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updateJSON, _ := json.Marshal(update)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Decoded UpdateUserProfile\",\"update\":%s}", string(updateJSON))

	// Convert Clerk ID to user UUID
	userID, err := h.userRepo.GetUserIDByClerkID(ctx, claims.Subject)
	if err != nil {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"User not found, creating new user\",\"clerk_id\":\"%s\",\"error\":%q}", claims.Subject, err.Error())

		// Create new user if they don't exist
		newUser := &models.User{
			ID:          uuid.New(),
			ClerkID:     claims.Subject,
			DisplayName: sql.NullString{String: "", Valid: false},
			City:        sql.NullString{String: "", Valid: false},
			State:       sql.NullString{String: "", Valid: false},
		}

		// Set provided fields from the update request
		if update.DisplayName != nil {
			newUser.DisplayName = sql.NullString{String: *update.DisplayName, Valid: *update.DisplayName != ""}
		}
		if update.City != nil {
			newUser.City = sql.NullString{String: *update.City, Valid: *update.City != ""}
		}
		if update.State != nil {
			newUser.State = sql.NullString{String: *update.State, Valid: *update.State != ""}
		}
		if update.LocationSharingEnabled != nil {
			newUser.LocationSharingEnabled = *update.LocationSharingEnabled
		}
		if update.HomeLatitude != nil {
			newUser.HomeLatitude = *update.HomeLatitude
		}
		if update.HomeLongitude != nil {
			newUser.HomeLongitude = *update.HomeLongitude
		}

		if err := h.userRepo.CreateUser(ctx, newUser); err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create new user\",\"clerk_id\":\"%s\",\"error\":%q}", claims.Subject, err.Error())
			http.Error(w, "Failed to create user profile", http.StatusInternalServerError)
			return
		}

		userID = newUser.ID
		log.Printf("{\"severity\":\"INFO\",\"message\":\"Created new user for profile update\",\"user_id\":\"%s\",\"clerk_id\":\"%s\"}", userID.String(), claims.Subject)
	} else {
		// User exists, update their profile
		if err := h.userRepo.UpdateProfile(ctx, userID.String(), &update); err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update user profile\",\"error\":%q}", err.Error())
			http.Error(w, "Failed to update profile", http.StatusInternalServerError)
			return
		}
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"User profile updated successfully\",\"user_id\":\"%s\"}", userID.String())
	w.WriteHeader(http.StatusOK)
}

func (h *UserHandler) AuthenticateUser(w http.ResponseWriter, r *http.Request) {
	// Get user info from Clerk authentication
	userID := r.Header.Get("X-User-ID") // Or however you're getting the user ID
	userEmail := r.Header.Get("X-User-Email")
	userName := r.Header.Get("X-User-Name")

	// Create user object
	user := &models.User{
		ID:          uuid.MustParse(userID),
		Email:       userEmail,
		Name:        userName,
		DisplayName: sql.NullString{String: userName, Valid: userName != ""}, // Default display name to actual name
		// City and State can be updated later by the user
	}

	// Try to create user if they don't exist
	if err := h.userRepo.CreateUserIfNotExists(r.Context(), user); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create/verify user: %v\"}", err)
		http.Error(w, "Failed to process user authentication", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// CreateProfile creates a new user profile
func (h *UserHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := clerk.SessionClaimsFromContext(ctx)
	log.Printf("Received CreateProfile request with headers: %v", r.Header)
	if !ok {
		log.Printf("No clerk claims found in context")
		//http.Error(w, "Unauthorized - No claims found", http.StatusUnauthorized)
		//return
	}

	var profile models.UpdateUserProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
		return
	}

	user := &models.User{
		ID:          uuid.New(),
		ClerkID:     claims.Subject,
		DisplayName: sql.NullString{String: *profile.DisplayName, Valid: profile.DisplayName != nil && *profile.DisplayName != ""},
		City:        sql.NullString{String: *profile.City, Valid: profile.City != nil && *profile.City != ""},
		State:       sql.NullString{String: *profile.State, Valid: profile.State != nil && *profile.State != ""},
	}

	if err := h.userRepo.CreateUser(r.Context(), user); err != nil {
		http.Error(w, "Failed to create profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Log the incoming request method and URL
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Received CreateUser request\",\"method\":\"%s\",\"url\":\"%s\"}", r.Method, r.URL.String())

	// Log all request headers
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = v[0]
	}
	headersJSON, _ := json.Marshal(headers)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Request headers\",\"headers\":%s}", string(headersJSON))

	// Log the raw request body
	bodyBytes, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to dump request: %v\"}", err)
	} else {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Raw request\",\"body\":%q}", string(bodyBytes))
	}

	ctx := r.Context()

	var user models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid request body\",\"error\":%q}", err.Error())
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userJSON, _ := json.Marshal(user)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Decoded CreateUserRequest\",\"user\":%s}", string(userJSON))

	// Create a new user model with additional fields (optional)
	newUser := &models.User{
		Email:       user.Email,
		Name:        user.Name,
		DisplayName: sql.NullString{String: user.DisplayName, Valid: user.DisplayName != ""},
		City:        sql.NullString{String: user.City, Valid: user.City != ""},
		State:       sql.NullString{String: user.State, Valid: user.State != ""},
		ClerkID:     user.ClerkID,
	}

	if err := h.userRepo.CreateUser(ctx, newUser); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create user\",\"error\":%q}", err.Error())
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"User created successfully\",\"email\":\"%s\",\"clerk_id\":\"%s\"}", newUser.Email, newUser.ClerkID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
}

func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {

	// Get clerk_id from context
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized - No clerk ID in context", http.StatusUnauthorized)
		return
	}

	// Get user from database using clerk_id
	user, err := h.userRepo.GetUserByClerkID(clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// GetUserByID returns all user data for a given user ID or Clerk ID
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetUserByID called\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())

	ctx := r.Context()

	// Get user ID from URL path using mux.Vars
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Missing user ID in path params\"}")
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Processing user ID\",\"id\":\"%s\",\"id_length\":%d,\"starts_with_user\":%t}",
		id, len(id), len(id) > 5 && id[:5] == "user_")

	var userID string
	if len(id) > 5 && id[:5] == "user_" {
		// It's a Clerk ID, convert to user UUID
		uuid, err := h.userRepo.GetUserIDByClerkID(ctx, id)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to convert Clerk ID to user UUID\",\"clerk_id\":\"%s\",\"error\":%q}", id, err.Error())
			if err.Error() == "no user found for clerk_id: "+id {
				http.Error(w, "User not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to get user by Clerk ID", http.StatusInternalServerError)
			}
			return
		}
		userID = uuid.String()
		log.Printf("{\"severity\":\"INFO\",\"message\":\"Converted Clerk ID to user UUID\",\"clerk_id\":\"%s\",\"user_id\":\"%s\"}", id, userID)
	} else {
		userID = id
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Fetching user by ID\",\"user_id\":\"%s\"}", userID)

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user by ID\",\"user_id\":\"%s\",\"error\":%q}", userID, err.Error())
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	userJSON, _ := json.Marshal(user)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Fetched user by ID\",\"user\":%s}", string(userJSON))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to encode response\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetUserByID completed\",\"duration_ms\":%d,\"user_id\":\"%s\"}",
		duration.Milliseconds(), userID)
}

func (h *UserHandler) GetUserActivities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Log request headers
	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = v[0]
	}
	headersJSON, _ := json.Marshal(headers)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetUserActivities request headers\",\"headers\":%s}", string(headersJSON))

	claims, ok := clerk.SessionClaimsFromContext(ctx)
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Unauthorized: No Clerk claims in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Authenticated user for activity fetch\",\"clerk_id\":\"%s\"}", claims.Subject)

	userID, err := h.userRepo.GetUserIDByClerkID(ctx, claims.Subject)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"User not found for activity fetch\",\"clerk_id\":\"%s\",\"error\":%q}", claims.Subject, err.Error())
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Resolved user ID for activity fetch\",\"user_id\":\"%s\"}", userID.String())

	limit := 50 // default limit
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Activity fetch limit\",\"limit\":%d}", limit)

	activities, err := h.userRepo.GetUserActivities(ctx, userID, limit)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to fetch activities\",\"user_id\":\"%s\",\"error\":%q}", userID.String(), err.Error())
		http.Error(w, "Failed to fetch activities", http.StatusInternalServerError)
		return
	}
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Fetched activities\",\"user_id\":\"%s\",\"count\":%d}", userID.String(), len(activities))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activities)
}

// DeleteUser deletes a user by ID (UUID or Clerk ID)
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("{\"severity\":\"INFO\",\"message\":\"DeleteUser called\",\"method\":\"%s\",\"url\":\"%s\",\"remote_addr\":\"%s\",\"user_agent\":\"%s\"}",
		r.Method, r.URL.String(), r.RemoteAddr, r.UserAgent())

	ctx := r.Context()

	// Get user ID from URL path using mux.Vars
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Missing user ID in path params\"}")
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	var userID uuid.UUID
	if len(id) > 5 && id[:5] == "user_" {
		// It's a Clerk ID, convert to user UUID
		uuid, err := h.userRepo.GetUserIDByClerkID(ctx, id)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to convert Clerk ID to user UUID\",\"clerk_id\":\"%s\",\"error\":%q}", id, err.Error())
			if err.Error() == "no user found for clerk_id: "+id {
				http.Error(w, "User not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to get user by Clerk ID", http.StatusInternalServerError)
			}
			return
		}
		userID = uuid
		log.Printf("{\"severity\":\"INFO\",\"message\":\"Converted Clerk ID to user UUID\",\"clerk_id\":\"%s\",\"user_id\":\"%s\"}", id, userID)
	} else {
		// It's a UUID
		parsedUUID, err := uuid.Parse(id)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid UUID format\",\"id\":\"%s\",\"error\":%q}", id, err.Error())
			http.Error(w, "Invalid user ID format", http.StatusBadRequest)
			return
		}
		userID = parsedUUID
		log.Printf("{\"severity\":\"INFO\",\"message\":\"Parsed UUID\",\"user_id\":\"%s\"}", userID)
	}

	// Verify user exists before deletion
	user, err := h.userRepo.GetByID(ctx, userID.String())
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"User not found\",\"user_id\":\"%s\",\"error\":%q}", userID, err.Error())
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Found user to delete\",\"user_id\":\"%s\",\"email\":\"%s\",\"display_name\":\"%s\"}",
		userID, user.Email, user.DisplayName.String)

	// Delete the user
	err = h.userRepo.DeleteUser(ctx, userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to delete user\",\"user_id\":\"%s\",\"error\":%q}", userID, err.Error())
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]string{
		"message": "User deleted successfully",
		"user_id": userID.String(),
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to encode response\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"DeleteUser completed\",\"duration_ms\":%d,\"user_id\":\"%s\",\"email\":\"%s\"}",
		duration.Milliseconds(), userID, user.Email)
}
