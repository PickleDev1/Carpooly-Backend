package handlers

import (
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
)

type WebhookHandler struct {
	userRepo *repository.UserRepository
}

func NewWebhookHandler(repo *repository.UserRepository) *WebhookHandler {
	return &WebhookHandler{
		userRepo: repo,
	}
}

type ClerkWebhookEvent struct {
	Data struct {
		ID             string `json:"id"`
		EmailAddresses []struct {
			EmailAddress string `json:"email_address"`
		} `json:"email_addresses"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	} `json:"data"`
	Type string `json:"type"`
}

func (h *WebhookHandler) HandleClerkWebhook(w http.ResponseWriter, r *http.Request) {
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Received Clerk webhook\"}")

	var event ClerkWebhookEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to decode webhook\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Webhook event\",\"type\":\"%s\",\"user_id\":\"%s\"}",
		event.Type, event.Data.ID)

	// Handle user.created event
	if event.Type == "user.created" {
		// Create new user in our database
		user := &models.User{
			ID:          uuid.New(),
			ClerkID:     event.Data.ID,
			Email:       event.Data.EmailAddresses[0].EmailAddress,
			Name:        event.Data.FirstName + " " + event.Data.LastName,
			DisplayName: event.Data.FirstName,
		}

		if err := h.userRepo.CreateUser(r.Context(), user); err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create user\",\"error\":\"%v\"}", err)
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		log.Printf("{\"severity\":\"INFO\",\"message\":\"Created new user\",\"user_id\":\"%s\"}", user.ID)
	}

	w.WriteHeader(http.StatusOK)
}
