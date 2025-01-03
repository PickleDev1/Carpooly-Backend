package handlers

import (
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/google/uuid"
)

type UserHandler struct {
	userRepo    *repository.UserRepository
	clerkClient clerk.Client
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
	ctx := r.Context()
	claims, ok := clerk.SessionClaimsFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user from our database
	user, err := h.userRepo.GetByID(ctx, claims.Subject)
	if err != nil {
		http.Error(w, "Failed to get user profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// UpdateProfile updates the user's profile
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := clerk.SessionClaimsFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var update models.UpdateUserProfile
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.userRepo.UpdateProfile(ctx, claims.Subject, &update); err != nil {
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

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
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
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

	if err := h.userRepo.CreateUser(ctx, user); err != nil {
		http.Error(w, "Failed to create profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Print the entire request
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to dump request: %v\"}", err)
	} else {
		fmt.Println("**Full Request:**", string(dump))
	}

	ctx := r.Context()

	var user models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

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
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create user: %v\"}", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
}
