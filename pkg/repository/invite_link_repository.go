package repository

import (
	"car-backend/pkg/models"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

type InviteLinkRepository struct {
	db *sql.DB
}

func NewInviteLinkRepository(db *sql.DB) *InviteLinkRepository {
	return &InviteLinkRepository{db: db}
}

// generateInviteCode creates a unique 8-character invite code
func (r *InviteLinkRepository) generateInviteCode() (string, error) {
	for {
		bytes := make([]byte, 4)
		if _, err := rand.Read(bytes); err != nil {
			return "", err
		}
		code := hex.EncodeToString(bytes)[:8] // Take first 8 characters

		// Check if code already exists
		var exists bool
		err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM invite_links WHERE invite_code = $1)", code).Scan(&exists)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
		// If code exists, try again
	}
}

// CreateInviteLink creates a new invite link for a carpool
func (r *InviteLinkRepository) CreateInviteLink(ctx context.Context, carpoolID, createdBy uuid.UUID, expiresInDays, maxUses int) (*models.InviteLink, error) {
	// Generate unique invite code
	inviteCode, err := r.generateInviteCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite code: %v", err)
	}

	// Set default expiration (30 days) if not specified
	if expiresInDays <= 0 {
		expiresInDays = 30
	}

	// Set default max uses (unlimited) if not specified
	if maxUses <= 0 {
		maxUses = -1
	}

	expiresAt := time.Now().AddDate(0, 0, expiresInDays)

	query := `
		INSERT INTO invite_links (carpool_id, invite_code, created_by, expires_at, max_uses)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	inviteLink := &models.InviteLink{
		CarpoolID:   carpoolID,
		InviteCode:  inviteCode,
		CreatedBy:   createdBy,
		IsActive:    true,
		ExpiresAt:   expiresAt,
		MaxUses:     maxUses,
		CurrentUses: 0,
	}

	err = r.db.QueryRowContext(ctx, query,
		carpoolID, inviteCode, createdBy, expiresAt, maxUses,
	).Scan(&inviteLink.ID, &inviteLink.CreatedAt, &inviteLink.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create invite link: %v", err)
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Created invite link\",\"invite_code\":\"%s\",\"carpool_id\":\"%s\",\"created_by\":\"%s\"}",
		inviteCode, carpoolID, createdBy)

	return inviteLink, nil
}

// GetInviteLinkByCode retrieves an invite link by its code (public endpoint)
func (r *InviteLinkRepository) GetInviteLinkByCode(ctx context.Context, inviteCode string) (*models.InviteLink, error) {
	query := `
		SELECT 
			il.id, il.carpool_id, il.invite_code, il.created_by, il.is_active,
			il.expires_at, il.max_uses, il.current_uses, il.created_at, il.updated_at,
			c.carpool_name, u.display_name as creator_name
		FROM invite_links il
		JOIN carpools c ON il.carpool_id = c.id
		JOIN users u ON il.created_by = u.id
		WHERE il.invite_code = $1
	`

	inviteLink := &models.InviteLink{}
	err := r.db.QueryRowContext(ctx, query, inviteCode).Scan(
		&inviteLink.ID, &inviteLink.CarpoolID, &inviteLink.InviteCode, &inviteLink.CreatedBy,
		&inviteLink.IsActive, &inviteLink.ExpiresAt, &inviteLink.MaxUses, &inviteLink.CurrentUses,
		&inviteLink.CreatedAt, &inviteLink.UpdatedAt, &inviteLink.CarpoolName, &inviteLink.CreatorName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invite link not found")
		}
		return nil, fmt.Errorf("failed to get invite link: %v", err)
	}

	// Check if invite is expired
	if time.Now().After(inviteLink.ExpiresAt) {
		inviteLink.IsActive = false
	}

	// Check if invite has reached max uses
	if inviteLink.MaxUses > 0 && inviteLink.CurrentUses >= inviteLink.MaxUses {
		inviteLink.IsActive = false
	}

	return inviteLink, nil
}

// JoinViaInviteLink adds a user to a carpool via invite link
func (r *InviteLinkRepository) JoinViaInviteLink(ctx context.Context, inviteCode string, userID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Get invite link
	inviteLink, err := r.GetInviteLinkByCode(ctx, inviteCode)
	if err != nil {
		return err
	}

	// Check if invite is still active
	if !inviteLink.IsActive {
		return fmt.Errorf("invite link is no longer active")
	}

	// Check if user is already a member
	var isMember bool
	err = tx.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM carpool_members WHERE carpool_id = $1 AND user_id = $2)",
		inviteLink.CarpoolID, userID,
	).Scan(&isMember)
	if err != nil {
		return fmt.Errorf("failed to check membership: %v", err)
	}

	if isMember {
		return fmt.Errorf("user is already a member of this carpool")
	}

	// Add user to carpool members
	_, err = tx.ExecContext(ctx,
		"INSERT INTO carpool_members (id, carpool_id, user_id, created_at, updated_at) VALUES (gen_random_uuid(), $1, $2, NOW(), NOW())",
		inviteLink.CarpoolID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to add user to carpool: %v", err)
	}

	// Add user to all future rides as a participant
	carpoolRepo := NewCarPoolRepository(r.db)
	err = carpoolRepo.AddUserToFutureRides(ctx, inviteLink.CarpoolID, userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to add user to future rides via invite link\",\"carpool_id\":\"%s\",\"user_id\":\"%s\",\"error\":\"%v\"}", inviteLink.CarpoolID, userID, err)
		// Do not return error, just log it
	}

	// Increment current uses
	_, err = tx.ExecContext(ctx,
		"UPDATE invite_links SET current_uses = current_uses + 1 WHERE id = $1",
		inviteLink.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update invite link usage: %v", err)
	}

	// Decrement available seats (but not below 0)
	_, err = tx.ExecContext(ctx,
		"UPDATE carpools SET available_seats = GREATEST(available_seats - 1, 0), updated_at = NOW() WHERE id = $1",
		inviteLink.CarpoolID,
	)
	if err != nil {
		return fmt.Errorf("failed to update available seats: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"User joined carpool via invite link\",\"user_id\":\"%s\",\"carpool_id\":\"%s\",\"invite_code\":\"%s\"}",
		userID, inviteLink.CarpoolID, inviteCode)

	return nil
}

// GetInviteLinksByCarpool gets all invite links for a carpool
func (r *InviteLinkRepository) GetInviteLinksByCarpool(ctx context.Context, carpoolID uuid.UUID) ([]models.InviteLink, error) {
	query := `
		SELECT 
			il.id, il.carpool_id, il.invite_code, il.created_by, il.is_active,
			il.expires_at, il.max_uses, il.current_uses, il.created_at, il.updated_at,
			c.carpool_name, u.display_name as creator_name
		FROM invite_links il
		JOIN carpools c ON il.carpool_id = c.id
		JOIN users u ON il.created_by = u.id
		WHERE il.carpool_id = $1
		ORDER BY il.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, carpoolID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invite links: %v", err)
	}
	defer rows.Close()

	var inviteLinks []models.InviteLink
	for rows.Next() {
		var inviteLink models.InviteLink
		err := rows.Scan(
			&inviteLink.ID, &inviteLink.CarpoolID, &inviteLink.InviteCode, &inviteLink.CreatedBy,
			&inviteLink.IsActive, &inviteLink.ExpiresAt, &inviteLink.MaxUses, &inviteLink.CurrentUses,
			&inviteLink.CreatedAt, &inviteLink.UpdatedAt, &inviteLink.CarpoolName, &inviteLink.CreatorName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invite link: %v", err)
		}

		// Check if invite is expired
		if time.Now().After(inviteLink.ExpiresAt) {
			inviteLink.IsActive = false
		}

		// Check if invite has reached max uses
		if inviteLink.MaxUses > 0 && inviteLink.CurrentUses >= inviteLink.MaxUses {
			inviteLink.IsActive = false
		}

		inviteLinks = append(inviteLinks, inviteLink)
	}

	return inviteLinks, nil
}

// DeactivateInviteLink deactivates an invite link
func (r *InviteLinkRepository) DeactivateInviteLink(ctx context.Context, inviteLinkID uuid.UUID) error {
	query := `UPDATE invite_links SET is_active = false WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, inviteLinkID)
	if err != nil {
		return fmt.Errorf("failed to deactivate invite link: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invite link not found")
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Deactivated invite link\",\"invite_link_id\":\"%s\"}", inviteLinkID)
	return nil
}
