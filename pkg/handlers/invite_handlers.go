package handlers

import (
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"encoding/json"
	"log"
	"net/http"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"fmt"
	"database/sql"
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
    var req models.CreateInviteRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to decode request: %v\"}", err)
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
    }

	log.Printf("Received CreateInvite request with data: %+v", req)
    // Placeholder: Replace with actual user ID retrieval logic
    userID, _ := uuid.Parse("c9407e8b-8cfc-4e63-b01a-b32c44b69261") 

    // Create Invite object
    invite := &models.Invite{
            FromUser:  userID, 
            ToUser:    req.ToUser,
            CarpoolID: req.CarpoolID,
            Message:   req.Message,
            Status:    0, // Initial status: pending
    }

    if err := h.inviteRepo.CreateInvite(r.Context(), invite); err != nil {
            log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create invite: %v\"}", err)
            http.Error(w, "Failed to create invite", http.StatusInternalServerError)
            return
    }

    w.WriteHeader(http.StatusCreated) 
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



