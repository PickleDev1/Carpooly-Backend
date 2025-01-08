package handlers

import (
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"car-backend/middleware"

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
	log.Printf("Received CreateInvite request: %s", r.URL)
	// Get userID from context
	userID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized - No user ID in context", http.StatusUnauthorized)
		return
	}

	// Parse the userID to UUID
	fromUserID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}
	var req models.CreateInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to decode request: %v\"}", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Received CreateInvite request with data: %+v", req)
	// Placeholder: Replace with actual user ID retrieval logic
	//userID, _ := uuid.Parse("6893bb9a-44d3-458e-b1a0-cb7b74c5cce1")

	// Create Invite object
	invite := &models.Invite{
		FromUser:  fromUserID,
		ToUser:    req.ToUser,
		CarpoolID: req.CarpoolID,
		Message:   req.Message,
		Status:    models.InviteStatusPending,
	}

	if err := h.inviteRepo.CreateInvite(r.Context(), invite); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create invite: %v\"}", err)
		http.Error(w, "Failed to create invite", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invite)
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

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	invites, err := h.inviteRepo.GetUserInvites(r.Context(), userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user invites: %v", err), http.StatusInternalServerError)
		return
	}

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

	err = h.inviteRepo.UpdateInviteStatus(r.Context(), inviteID, req.Status)
	if err != nil {
		if err.Error() == "invite not found" {
			http.Error(w, "Invite not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to update invite status: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
