package handlers

import (
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type InviteHandler struct {
	inviteRepo *repository.InviteRepository
}

func NewInviteHandler(repo *repository.InviteRepository) *InviteHandler {
	return &InviteHandler{
		inviteRepo: repo,
	}
}

func (h *InviteHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {

	var invite models.CreateInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&invite); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create new invite using the from_user as the user UUID
	newInvite := &models.Invite{
		ID:        uuid.New(),
		CarpoolID: uuid.MustParse(invite.CarpoolID),
		FromUser:  uuid.MustParse(invite.FromUser),
		ToUser:    invite.Email,
		Message:   invite.Message,
		Status:    models.InviteStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.inviteRepo.CreateInvite(r.Context(), newInvite); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create invite: %v\"}", err)
		http.Error(w, "Failed to create invite", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newInvite)
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
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	userIDStr := vars["userID"]
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Getting invites for userID\",\"userID\":\"%s\"}", userIDStr)

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid user ID format\",\"error\":\"%v\"}", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	invites, err := h.inviteRepo.GetUserInvites(r.Context(), userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user invites\",\"error\":\"%v\"}", err)
		http.Error(w, fmt.Sprintf("Failed to get user invites: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Retrieved invites\",\"count\":%d}", len(invites))
	json.NewEncoder(w).Encode(invites)
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
