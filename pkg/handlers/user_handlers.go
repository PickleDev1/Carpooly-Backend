package handlers

import (
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"

	"car-backend/middleware"

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
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user UUID from Clerk ID\",\"clerk_id\":\"%s\",\"error\":%q}", claims.Subject, err.Error())
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	if err := h.userRepo.UpdateProfile(ctx, userID.String(), &update); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update user profile\",\"error\":%q}", err.Error())
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
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
		DisplayName: userName, // Default display name to actual name
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
		DisplayName: *profile.DisplayName,
		City:        *profile.City,
		State:       *profile.State,
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
		DisplayName: user.DisplayName,
		City:        user.City,
		State:       user.State,
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
	log.Printf("Starting GetUserByID handler")
	ctx := r.Context()

	// Get user ID from URL path using mux.Vars
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Missing user ID in path params\"}")
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	var userID string
	if len(id) > 5 && id[:5] == "user_" {
		// It's a Clerk ID, convert to user UUID
		uuid, err := h.userRepo.GetUserIDByClerkID(ctx, id)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to convert Clerk ID to user UUID\",\"clerk_id\":\"%s\",\"error\":%q}", id, err.Error())
			http.Error(w, "Failed to get user by Clerk ID", http.StatusInternalServerError)
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
	json.NewEncoder(w).Encode(user)
}
