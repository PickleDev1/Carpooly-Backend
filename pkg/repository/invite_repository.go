package repository

import (
    "car-backend/pkg/models"
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	
)

type InviteRepository struct {
    db *sql.DB
}

func NewInviteRepository(db *sql.DB) *InviteRepository {
    return &InviteRepository{db: db}
}

func (r *InviteRepository) CreateInvite(ctx context.Context, invite *models.Invite) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
            return fmt.Errorf("failed to begin transaction: %v", err)
    }
    defer tx.Rollback()

    query := `
            INSERT INTO invites (
                    from_user, to_user, carpool_id, message, status
            ) VALUES ($1, $2, $3, $4, $5)
            RETURNING id, created_at, updated_at
    `

    err = tx.QueryRowContext(
            ctx, query,
            invite.FromUser, invite.ToUser, invite.CarpoolID, invite.Message, invite.Status,
    ).Scan(&invite.ID, &invite.CreatedAt, &invite.UpdatedAt)

    if err != nil {
            return fmt.Errorf("failed to create invite: %v", err)
    }

    if err := tx.Commit(); err != nil {
            return fmt.Errorf("failed to commit transaction: %v", err)
    }

    return nil
}

func (r *InviteRepository) GetInvite(ctx context.Context, inviteID uuid.UUID) (*models.Invite, error) {
    invite := &models.Invite{}

    query := `
            SELECT id, from_user, to_user, carpool_id, message, status, created_at, updated_at
            FROM invites
            WHERE id = $1
    `
	//Just a little note

    err := r.db.QueryRowContext(ctx, query, inviteID).Scan(
            &invite.ID,
            &invite.FromUser,
            &invite.ToUser,
            &invite.CarpoolID,
            &invite.Message,
            &invite.Status,
            &invite.CreatedAt,
            &invite.UpdatedAt,
    )
    if err != nil {
            if err == sql.ErrNoRows {
                    return nil, nil
            }
            return nil, fmt.Errorf("failed to get invite: %w", err)
    }

    return invite, nil
}

func (r *InviteRepository) DeleteInvite(ctx context.Context, inviteID uuid.UUID) error {
	query := `
			DELETE FROM invites
			WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, inviteID)
	if err != nil {
			return fmt.Errorf("failed to delete invite: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
			return fmt.Errorf("invite not found")
	}

	return nil
}
