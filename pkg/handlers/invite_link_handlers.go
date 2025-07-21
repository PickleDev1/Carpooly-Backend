package handlers

import (
	"car-backend/middleware"
	"car-backend/pkg/models"
	"car-backend/pkg/repository"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type InviteLinkHandler struct {
	inviteLinkRepo *repository.InviteLinkRepository
	userRepo       *repository.UserRepository
	carpoolRepo    *repository.CarPoolRepository
}

func NewInviteLinkHandler(inviteLinkRepo *repository.InviteLinkRepository, userRepo *repository.UserRepository, carpoolRepo *repository.CarPoolRepository) *InviteLinkHandler {
	return &InviteLinkHandler{
		inviteLinkRepo: inviteLinkRepo,
		userRepo:       userRepo,
		carpoolRepo:    carpoolRepo,
	}
}

// CreateInviteLink creates a new invite link for a carpool
func (h *InviteLinkHandler) CreateInviteLink(w http.ResponseWriter, r *http.Request) {
	log.Printf("{\"severity\":\"INFO\",\"message\":\"CreateInviteLink called\"}")

	// Get the logged-in user's ID from context
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"No clerk ID in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Convert clerk ID to user ID
	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"clerk_id\":\"%s\",\"error\":%v}", clerkID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	// Get carpool ID from URL
	vars := mux.Vars(r)
	carpoolIDStr := vars["carpoolID"]
	carpoolID, err := uuid.Parse(carpoolIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ID\",\"carpool_id\":\"%s\"}", carpoolIDStr)
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}

	// Verify carpool exists
	carpool, err := h.carpoolRepo.GetCarPool(r.Context(), carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get carpool\",\"carpool_id\":\"%s\",\"error\":%v}", carpoolID, err)
		http.Error(w, "Carpool not found", http.StatusNotFound)
		return
	}

	// Check if user is a member of the carpool (either creator or member)
	isCreator := carpool.CreatorID == userID
	isMember := false

	if !isCreator {
		// Check if user is a member
		members, err := h.carpoolRepo.GetCarpoolMembers(r.Context(), carpoolID)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get carpool members\",\"carpool_id\":\"%s\",\"error\":%v}", carpoolID, err)
			http.Error(w, "Failed to verify membership", http.StatusInternalServerError)
			return
		}

		for _, member := range members {
			if member.ID == userID {
				isMember = true
				break
			}
		}
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Authorization check\",\"user_id\":\"%s\",\"carpool_creator\":\"%s\",\"is_creator\":%v,\"is_member\":%v,\"carpool_id\":\"%s\"}", userID, carpool.CreatorID, isCreator, isMember, carpoolID)

	if !isCreator && !isMember {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"User not authorized to create invite link\",\"user_id\":\"%s\",\"carpool_creator\":\"%s\"}", userID, carpool.CreatorID)
		http.Error(w, "You must be a member of this carpool to create invite links", http.StatusForbidden)
		return
	}

	// Parse request body
	var req models.CreateInviteLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to decode request\",\"error\":%v}", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Always create a new invite link
	inviteLink, err := h.inviteLinkRepo.CreateInviteLink(r.Context(), carpoolID, userID, req.ExpiresInDays, req.MaxUses)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to create invite link\",\"error\":%v}", err)
		http.Error(w, "Failed to create invite link", http.StatusInternalServerError)
		return
	}

	// Get creator name for response
	creator, err := h.userRepo.GetUserByID(carpool.CreatorID)
	creatorName := "Unknown"
	if err != nil {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"Failed to get creator name\",\"creator_id\":\"%s\",\"error\":%v}", carpool.CreatorID, err)
		// Continue without creator name
	} else if creator != nil {
		if creator.DisplayName.Valid {
			creatorName = creator.DisplayName.String
		} else {
			creatorName = ""
		}
	}

	// Build response
	response := models.InviteLinkResponse{
		InviteCode:  inviteLink.InviteCode,
		CarpoolName: carpool.CarpoolName,
		CreatorName: creatorName,
		ExpiresAt:   inviteLink.ExpiresAt,
		IsActive:    inviteLink.IsActive,
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Created invite link\",\"invite_code\":\"%s\",\"carpool_id\":\"%s\"}", inviteLink.InviteCode, carpoolID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetInviteLink retrieves invite link details (public endpoint)
func (h *InviteLinkHandler) GetInviteLink(w http.ResponseWriter, r *http.Request) {
	inviteCode := mux.Vars(r)["code"]
	log.Printf("[DEBUG] GetInviteLink handler called for invite_code: %s", inviteCode)
	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetInviteLink called\"}")

	if inviteCode == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Missing invite code\"}")
		http.Error(w, "Missing invite code", http.StatusBadRequest)
		return
	}

	// Get invite link details
	inviteLink, err := h.inviteLinkRepo.GetInviteLinkByCode(r.Context(), inviteCode)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get invite link\",\"invite_code\":\"%s\",\"error\":%v}", inviteCode, err)
		http.Error(w, "Invite link not found", http.StatusNotFound)
		return
	}

	// Build response
	response := models.InviteLinkResponse{
		InviteCode:  inviteLink.InviteCode,
		CarpoolName: inviteLink.CarpoolName,
		CreatorName: inviteLink.CreatorName.String, // FIXED: use .String
		ExpiresAt:   inviteLink.ExpiresAt,
		IsActive:    inviteLink.IsActive,
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Retrieved invite link\",\"invite_code\":\"%s\",\"is_active\":%v}", inviteCode, inviteLink.IsActive)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// JoinViaInviteLink allows a user to join a carpool via invite link
func (h *InviteLinkHandler) JoinViaInviteLink(w http.ResponseWriter, r *http.Request) {
	log.Printf("{\"severity\":\"INFO\",\"message\":\"JoinViaInviteLink called\"}")

	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"No clerk ID in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"clerk_id\":\"%s\",\"error\":%v}", clerkID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	inviteCode := vars["code"]

	if inviteCode == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Missing invite code\"}")
		http.Error(w, "Missing invite code", http.StatusBadRequest)
		return
	}

	// Get invite link details
	inviteLink, err := h.inviteLinkRepo.GetInviteLinkByCode(r.Context(), inviteCode)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get invite link\",\"invite_code\":\"%s\",\"error\":%v}", inviteCode, err)
		http.Error(w, "Invite link not found", http.StatusNotFound)
		return
	}

	// Detailed logging for invite link status
	log.Printf("[JoinViaInviteLink] invite_code=%s is_active=%v expires_at=%v now=%v max_uses=%d current_uses=%d", inviteCode, inviteLink.IsActive, inviteLink.ExpiresAt, time.Now(), inviteLink.MaxUses, inviteLink.CurrentUses)

	if !inviteLink.IsActive {
		log.Printf("[JoinViaInviteLink] Invite link is not active")
		http.Error(w, "Invite link is no longer active", http.StatusBadRequest)
		return
	}
	if time.Now().After(inviteLink.ExpiresAt) {
		log.Printf("[JoinViaInviteLink] Invite link is expired: expires_at=%v now=%v", inviteLink.ExpiresAt, time.Now())
		http.Error(w, "Invite link has expired", http.StatusBadRequest)
		return
	}
	if inviteLink.MaxUses > 0 && inviteLink.CurrentUses >= inviteLink.MaxUses {
		log.Printf("[JoinViaInviteLink] Invite link has reached max uses: max_uses=%d current_uses=%d", inviteLink.MaxUses, inviteLink.CurrentUses)
		http.Error(w, "Invite link has reached its maximum number of uses", http.StatusBadRequest)
		return
	}

	// Check if user is already a member
	isMember, err := h.carpoolRepo.IsUserMemberOfCarpool(r.Context(), inviteLink.CarpoolID, userID)
	if err != nil {
		log.Printf("[JoinViaInviteLink] ERROR: Failed to check membership: %v", err)
		http.Error(w, "Failed to check membership", http.StatusInternalServerError)
		return
	}
	if isMember {
		log.Printf("[JoinViaInviteLink] User is already a member of the carpool")
		http.Error(w, "You are already a member of this carpool", http.StatusBadRequest)
		return
	}

	// Proceed to join via invite link
	err = h.inviteLinkRepo.JoinViaInviteLink(r.Context(), inviteCode, userID)
	if err != nil {
		log.Printf("[JoinViaInviteLink] ERROR: Failed to join via invite link: %v", err)
		http.Error(w, fmt.Sprintf("Failed to join carpool: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("[JoinViaInviteLink] User %s joined carpool %s via invite link %s", userID, inviteLink.CarpoolID, inviteCode)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Successfully joined carpool",
	})
}

// GetInviteLinksByCarpool gets all invite links for a carpool
func (h *InviteLinkHandler) GetInviteLinksByCarpool(w http.ResponseWriter, r *http.Request) {
	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetInviteLinksByCarpool called\"}")

	// Get the logged-in user's ID from context
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"No clerk ID in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Convert clerk ID to user ID
	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"clerk_id\":\"%s\",\"error\":%v}", clerkID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	// Get carpool ID from URL
	vars := mux.Vars(r)
	carpoolIDStr := vars["id"]
	carpoolID, err := uuid.Parse(carpoolIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid carpool ID\",\"carpool_id\":\"%s\"}", carpoolIDStr)
		http.Error(w, "Invalid carpool ID", http.StatusBadRequest)
		return
	}

	// Verify carpool exists
	carpool, err := h.carpoolRepo.GetCarPool(r.Context(), carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get carpool\",\"carpool_id\":\"%s\",\"error\":%v}", carpoolID, err)
		http.Error(w, "Carpool not found", http.StatusNotFound)
		return
	}

	// Check if user is a member of the carpool (either creator or member)
	isCreator := carpool.CreatorID == userID
	isMember := false

	if !isCreator {
		// Check if user is a member
		members, err := h.carpoolRepo.GetCarpoolMembers(r.Context(), carpoolID)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get carpool members\",\"carpool_id\":\"%s\",\"error\":%v}", carpoolID, err)
			http.Error(w, "Failed to verify membership", http.StatusInternalServerError)
			return
		}

		for _, member := range members {
			if member.ID == userID {
				isMember = true
				break
			}
		}
	}

	if !isCreator && !isMember {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"User not authorized to view invite links\",\"user_id\":\"%s\",\"carpool_creator\":\"%s\"}", userID, carpool.CreatorID)
		http.Error(w, "You must be a member of this carpool to view invite links", http.StatusForbidden)
		return
	}

	// Get invite links
	inviteLinks, err := h.inviteLinkRepo.GetInviteLinksByCarpool(r.Context(), carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get invite links\",\"carpool_id\":\"%s\",\"error\":%v}", carpoolID, err)
		http.Error(w, "Failed to get invite links", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Retrieved invite links\",\"carpool_id\":\"%s\",\"count\":%d}", carpoolID, len(inviteLinks))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"invite_links": inviteLinks,
		"count":        len(inviteLinks),
	})
}

// DeactivateInviteLink deactivates an invite link
func (h *InviteLinkHandler) DeactivateInviteLink(w http.ResponseWriter, r *http.Request) {
	log.Printf("{\"severity\":\"INFO\",\"message\":\"DeactivateInviteLink called\"}")

	// Get the logged-in user's ID from context
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"No clerk ID in context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Convert clerk ID to user ID
	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"clerk_id\":\"%s\",\"error\":%v}", clerkID, err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	inviteLinkIDStr := vars["id"]
	inviteLinkID, err := uuid.Parse(inviteLinkIDStr)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Invalid invite link ID\",\"invite_link_id\":\"%s\"}", inviteLinkIDStr)
		http.Error(w, "Invalid invite link ID", http.StatusBadRequest)
		return
	}

	// TODO: Verify user is authorized to deactivate this invite link
	// For now, we'll allow any authenticated user to deactivate any invite link

	// Deactivate invite link
	err = h.inviteLinkRepo.DeactivateInviteLink(r.Context(), inviteLinkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to deactivate invite link\",\"invite_link_id\":\"%s\",\"error\":%v}", inviteLinkID, err)
		http.Error(w, "Failed to deactivate invite link", http.StatusInternalServerError)
		return
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Deactivated invite link\",\"invite_link_id\":\"%s\",\"user_id\":\"%s\"}", inviteLinkID, userID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Invite link deactivated successfully",
	})
}
