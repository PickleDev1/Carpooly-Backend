package handlers

import (
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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

	// Log the received request
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Received invite request\",\"from_user\":\"%s\",\"to_email\":\"%s\",\"carpool_id\":\"%s\"}",
		invite.FromUser, invite.Email, invite.CarpoolID)

	// Get sender's details
	userID, err := uuid.Parse(invite.FromUser)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid user ID format\",\"id\":\"%s\",\"error\":\"%v\"}",
			invite.FromUser, err)
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	fromUser, err := h.userRepo.GetUserByID(userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get sender details\",\"user_id\":\"%s\",\"error\":\"%v\"}",
			userID, err)
		http.Error(w, "Failed to get sender details", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Found sender details\",\"user_id\":\"%s\",\"name\":\"%s\"}",
		fromUser.ID, fromUser.Name)

	// Get carpool details
	carpoolID := uuid.MustParse(invite.CarpoolID)
	carpool, err := h.carpoolRepo.GetCarPool(r.Context(), carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get carpool details\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			carpoolID, err)
		http.Error(w, "Failed to get carpool details", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Found carpool details\",\"carpool_id\":\"%s\",\"name\":\"%s\"}",
		carpool.ID, carpool.CarpoolName)

	// Create new invite
	newInvite := &models.Invite{
		ID:        uuid.New(),
		CarpoolID: carpoolID,
		FromUser:  fromUser.ID,
		ToUser:    invite.Email,
		Message:   invite.Message,
		Status:    models.InviteStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.inviteRepo.CreateInvite(r.Context(), newInvite); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create invite\",\"invite_id\":\"%s\",\"error\":\"%v\"}",
			newInvite.ID, err)
		http.Error(w, "Failed to create invite", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully created invite\",\"invite_id\":\"%s\"}",
		newInvite.ID)

	// Send email invitation
	if err := h.sendInviteEmail(newInvite, fromUser, carpool); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to send invite email: %v\"}", err)
		// Continue execution as the invite was created successfully
	}

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
				<p>Message from %s:</p>
				<blockquote style="border-left: 4px solid #4F46E5; margin: 0; padding-left: 20px;">
					%s
				</blockquote>
				<div style="text-align: center; margin-top: 30px;">
					<a href="https://carpooly-web.vercel.app/invites/%s" 
					   style="background-color: #4F46E5; color: white; padding: 12px 24px; 
							  text-decoration: none; border-radius: 4px;">
						View Invitation
					</a>
				</div>
			</div>
		</div>
	`, fromUser.Name, carpool.CarpoolName, fromUser.Name, invite.Message, invite.ID)

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
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Starting GetUserInvites handler\",\"method\":\"%s\",\"path\":\"%s\"}",
		r.Method, r.URL.Path)

	// Log request headers
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Request headers\",\"auth\":\"%s\",\"content-type\":\"%s\"}",
		r.Header.Get("Authorization"), r.Header.Get("Content-Type"))

	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	clerkID := vars["userID"]
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Looking up user by Clerk ID\",\"clerk_id\":\"%s\"}", clerkID)

	// First get the user by their Clerk ID
	user, err := h.userRepo.GetUserByClerkID(clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user by Clerk ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}",
			clerkID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Found user\",\"clerk_id\":\"%s\",\"user_id\":\"%s\",\"email\":\"%s\"}",
		clerkID, user.ID, user.Email)

	invites, err := h.inviteRepo.GetUserInvites(r.Context(), user.ID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user invites\",\"user_id\":\"%s\",\"error\":\"%v\"}",
			user.ID, err)
		http.Error(w, fmt.Sprintf("Failed to get user invites: %v", err), http.StatusInternalServerError)
		return
	}

	// Log the response data
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Retrieved invites\",\"clerk_id\":\"%s\",\"user_id\":\"%s\",\"count\":%d}",
		clerkID, user.ID, len(invites))

	if err := json.NewEncoder(w).Encode(invites); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to encode response\",\"user_id\":\"%s\",\"error\":\"%v\"}",
			user.ID, err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Successfully sent invites response\",\"clerk_id\":\"%s\",\"user_id\":\"%s\",\"count\":%d}",
		clerkID, user.ID, len(invites))
}

func (h *InviteHandler) UpdateInviteStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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

	// If status is accepted (1), add user to carpool_members
	if req.Status == models.InviteStatusAccepted {
		err = h.inviteRepo.AcceptInvite(r.Context(), inviteID)
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
