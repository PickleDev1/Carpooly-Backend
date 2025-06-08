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
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/resend/resend-go/v2"
)

type InviteHandler struct {
	inviteRepo  *repository.InviteRepository
	userRepo    *repository.UserRepository
	carpoolRepo *repository.CarPoolRepository
}

func NewInviteHandler(inviteRepo *repository.InviteRepository, userRepo *repository.UserRepository, carpoolRepo *repository.CarPoolRepository) *InviteHandler {
	return &InviteHandler{
		inviteRepo:  inviteRepo,
		userRepo:    userRepo,
		carpoolRepo: carpoolRepo,
	}
}

func (h *InviteHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	var invite models.CreateInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&invite); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to decode request body\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get the logged-in user's ID from context
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"No clerk ID in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Convert clerk ID to user ID
	fromUserID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}",
			clerkID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	// Parse carpool ID
	carpoolID, err := uuid.Parse(invite.CarpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ID format\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			invite.CarpoolID, err)
		http.Error(w, "Invalid carpool ID format", http.StatusBadRequest)
		return
	}

	// Create new invite
	newInvite := &models.Invite{
		ID:        uuid.New(),
		CarpoolID: carpoolID,
		FromUser:  fromUserID,
		ToUser:    invite.Email,
		Status:    models.InviteStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create the invite in the database
	if err := h.inviteRepo.CreateInvite(r.Context(), newInvite); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create invite\",\"invite_id\":\"%s\",\"error\":\"%v\"}",
			newInvite.ID, err)
		http.Error(w, "Failed to create invite", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully created invite\",\"invite_id\":\"%s\",\"from_user\":\"%s\",\"to_user\":\"%s\"}",
		newInvite.ID, fromUserID, invite.Email)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newInvite)
}

func (h *InviteHandler) sendInviteEmail(invite *models.Invite, fromUser *models.User, carpool *models.Carpool) error {
	resendAPIKey := os.Getenv("RESEND_API_KEY")
	if resendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	client := resend.NewClient(resendAPIKey)

	htmlContent := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<div style="background-color: #4F46E5; padding: 20px; text-align: center;">
				<h1 style="color: white; margin: 0;">Carpooly Invitation</h1>
			</div>
			<div style="padding: 20px; border: 1px solid #ddd; border-top: none;">
				<p>Hello,</p>
				<p>You've been invited by %s to join a carpool group: <strong>%s</strong></p>
				<div style="text-align: center; margin-top: 30px;">
					<a href="https://carpooly-web.vercel.app/invites/%s" 
					   style="background-color: #4F46E5; color: white; padding: 12px 24px; 
							  text-decoration: none; border-radius: 4px;">
						View Invitation
					</a>
				</div>
			</div>
		</div>
	`, fromUser.Name, carpool.CarpoolName, invite.ID)

	params := &resend.SendEmailRequest{
		From:    "Carpooly <invites@carpooly.app>",
		To:      []string{invite.ToUser},
		Subject: fmt.Sprintf("Join %s's Carpool Group on Carpooly", fromUser.Name),
		Html:    htmlContent,
	}

	_, err := client.Emails.Send(params)
	return err
}

func (h *InviteHandler) GetInvite(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	inviteIDStr := vars["id"]

	inviteID, err := uuid.Parse(inviteIDStr)
	if err != nil {
		http.Error(w, "Invalid invite ID", http.StatusBadRequest)
		return
	}

	invite, err := h.inviteRepo.GetInvite(r.Context(), inviteID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invite not found\",\"invite_id\":\"%s\"}", inviteID)
			http.Error(w, "Invite not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get invite: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(invite)
}

func (h *InviteHandler) DeleteInvite(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	inviteIDStr := vars["id"]

	inviteID, err := uuid.Parse(inviteIDStr)
	if err != nil {
		http.Error(w, "Invalid invite ID", http.StatusBadRequest)
		return
	}

	err = h.inviteRepo.DeleteInvite(r.Context(), inviteID)
	if err != nil {
		if err.Error() == "invite not found" {
			http.Error(w, "Invite not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to delete invite: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *InviteHandler) GetUserInvites(w http.ResponseWriter, r *http.Request) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Starting GetUserInvites handler\"}")
	w.Header().Set("Content-Type", "application/json")

	// Get clerk ID from URL
	vars := mux.Vars(r)
	clerkID := vars["userID"]

	// Get user's email from clerk ID
	user, err := h.userRepo.GetUserByClerkID(clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user\",\"clerk_id\":\"%s\",\"error\":\"%v\"}",
			clerkID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get invites using email
	invites, err := h.inviteRepo.GetUserInvites(r.Context(), user.Email)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get invites\",\"email\":\"%s\",\"error\":\"%v\"}",
			user.Email, err)
		http.Error(w, "Failed to get invites", http.StatusInternalServerError)
		return
	}

	// Log the response
	invitesJSON, _ := json.MarshalIndent(invites, "", "  ")
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Found invites\",\"email\":\"%s\",\"count\":%d,\"invites\":%s}",
		user.Email, len(invites), string(invitesJSON))

	json.NewEncoder(w).Encode(invites)
}

func (h *InviteHandler) UpdateInviteStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get clerk ID from context
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"No clerk ID in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	inviteIDStr := vars["id"]

	inviteID, err := uuid.Parse(inviteIDStr)
	if err != nil {
		http.Error(w, "Invalid invite ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get the invite to verify the user is the intended recipient
	invite, err := h.inviteRepo.GetInvite(r.Context(), inviteID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get invite\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to get invite", http.StatusInternalServerError)
		return
	}

	// Get user details from clerk ID
	user, err := h.userRepo.GetUserByClerkID(clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user details\",\"error\":\"%v\"}", err)
		http.Error(w, "Failed to get user details", http.StatusInternalServerError)
		return
	}

	// Verify this user is the intended recipient
	if user.Email != invite.ToUser {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"User not authorized to update this invite\",\"user_email\":\"%s\",\"invite_email\":\"%s\"}",
			user.Email, invite.ToUser)
		http.Error(w, "Not authorized to update this invite", http.StatusForbidden)
		return
	}

	if req.Status == models.InviteStatusAccepted {
		// Update invite status only
		err = h.inviteRepo.UpdateInviteStatus(r.Context(), inviteID, req.Status)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update invite status\",\"error\":\"%v\"}", err)
			http.Error(w, "Failed to update invite status", http.StatusInternalServerError)
			return
		}

		carpoolID := invite.CarpoolID.String()
		userID := user.ID.String()

		// 1. Call add-to-carpool-members API
		addToMembersURL := os.Getenv("CARPOOLY_API_URL") + "/api/carpools/" + carpoolID + "/members"
		if addToMembersURL == "/api/carpools/"+carpoolID+"/members" {
			addToMembersURL = "http://localhost:8080/api/carpools/" + carpoolID + "/members"
		}
		membersPayload := map[string]string{"user_id": userID}
		membersJSON, _ := json.Marshal(membersPayload)
		reqMembers, err := http.NewRequest("POST", addToMembersURL, strings.NewReader(string(membersJSON)))
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create HTTP request to add-to-carpool-members\",\"error\":\"%v\"}", err)
			http.Error(w, "Failed to add user to carpool members", http.StatusInternalServerError)
			return
		}
		reqMembers.Header.Set("Content-Type", "application/json")
		if auth := r.Header.Get("Authorization"); auth != "" {
			reqMembers.Header.Set("Authorization", auth)
		}
		client := &http.Client{Timeout: 10 * time.Second}
		respMembers, err := client.Do(reqMembers)
		if err != nil || respMembers.StatusCode >= 300 {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to call add-to-carpool-members API\",\"status\":%d,\"error\":\"%v\"}", respMembers.StatusCode, err)
			http.Error(w, "Failed to add user to carpool members", http.StatusInternalServerError)
			return
		}
		defer respMembers.Body.Close()

		// 2. Call add-to-future-rides API
		addToRidesURL := os.Getenv("CARPOOLY_API_URL") + "/api/carpools/" + carpoolID + "/add-to-future-rides"
		if addToRidesURL == "/api/carpools/"+carpoolID+"/add-to-future-rides" {
			addToRidesURL = "http://localhost:8080/api/carpools/" + carpoolID + "/add-to-future-rides"
		}
		ridesPayload := map[string]string{"user_id": userID}
		ridesJSON, _ := json.Marshal(ridesPayload)
		reqRides, err := http.NewRequest("POST", addToRidesURL, strings.NewReader(string(ridesJSON)))
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create HTTP request to add-to-future-rides\",\"error\":\"%v\"}", err)
			http.Error(w, "Failed to add user to rides", http.StatusInternalServerError)
			return
		}
		reqRides.Header.Set("Content-Type", "application/json")
		if auth := r.Header.Get("Authorization"); auth != "" {
			reqRides.Header.Set("Authorization", auth)
		}
		respRides, err := client.Do(reqRides)
		if err != nil || respRides.StatusCode >= 300 {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to call add-to-future-rides API\",\"status\":%d,\"error\":\"%v\"}", respRides.StatusCode, err)
			http.Error(w, "Failed to add user to rides", http.StatusInternalServerError)
			return
		}
		defer respRides.Body.Close()
	} else {
		err = h.inviteRepo.UpdateInviteStatus(r.Context(), inviteID, req.Status)
	}

	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update invite\",\"error\":\"%v\"}", err)
		http.Error(w, fmt.Sprintf("Failed to update invite: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
