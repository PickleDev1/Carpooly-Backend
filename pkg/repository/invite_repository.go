package repository

import (
	"car-backend/pkg/models"
	"context"
	"database/sql"
	"fmt"
	"log"

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
                    from_user, to_user_email, carpool_id, message, status
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
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Getting invite from database\",\"invite_id\":\"%s\"}", inviteID)

	invite := &models.Invite{}
	query := `
        SELECT id, from_user, to_user_email, carpool_id, message, status, created_at, updated_at
        FROM invites
        WHERE id = $1
    `

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Executing query\",\"invite_id\":\"%s\",\"query\":%q}",
		inviteID, query)

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
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"No invite found\",\"invite_id\":\"%s\"}", inviteID)
			return nil, nil
		}
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Database error getting invite\",\"invite_id\":\"%s\",\"error\":\"%v\"}",
			inviteID, err)
		return nil, fmt.Errorf("failed to get invite: %w", err)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Successfully retrieved invite\",\"invite_id\":\"%s\",\"status\":%d}",
		invite.ID, invite.Status)
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

func (r *InviteRepository) GetUserInvites(ctx context.Context, email string) ([]models.Invite, error) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Getting invites for email\",\"email\":\"%s\"}", email)

	// First, let's check if there are any invites at all with this email
	checkQuery := `SELECT COUNT(*) FROM invites WHERE to_user_email = $1`
	var count int
	err := r.db.QueryRowContext(ctx, checkQuery, email).Scan(&count)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to check invites count\",\"error\":\"%v\"}", err)
	} else {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Total invites found (including all statuses)\",\"count\":%d,\"email\":\"%s\"}", count, email)
	}

	query := `
        SELECT 
            i.id,
            i.carpool_id,
            i.from_user,
            i.to_user_email,
            i.message,
            i.status,
            i.created_at,
            i.updated_at,
            c.carpool_name,
            u.email as sender_email
        FROM invites i
        JOIN carpools c ON i.carpool_id = c.id
        JOIN users u ON i.from_user = u.id
        WHERE i.to_user_email = $1 AND i.status = 0
    `

	// Log the exact query and parameters being used
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Executing query\",\"query\":\"%s\",\"email\":\"%s\"}",
		query, email)

	rows, err := r.db.QueryContext(ctx, query, email)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Database query failed\",\"error\":\"%v\"}", err)
		return nil, fmt.Errorf("failed to get user invites: %v", err)
	}
	defer rows.Close()

	var invites []models.Invite
	for rows.Next() {
		var invite models.Invite
		err := rows.Scan(
			&invite.ID,
			&invite.CarpoolID,
			&invite.FromUser,
			&invite.ToUser,
			&invite.Message,
			&invite.Status,
			&invite.CreatedAt,
			&invite.UpdatedAt,
			&invite.CarpoolName,
			&invite.SenderEmail,
		)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to scan invite\",\"error\":\"%v\"}", err)
			return nil, fmt.Errorf("failed to scan invite: %v", err)
		}
		invites = append(invites, invite)
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Found invite\",\"invite_id\":\"%s\",\"status\":%d}",
			invite.ID, invite.Status)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Retrieved invites\",\"count\":%d,\"email\":\"%s\"}",
		len(invites), email)
	return invites, nil
}

func (r *InviteRepository) UpdateInviteStatus(ctx context.Context, inviteID uuid.UUID, status int) error {
	query := `
                UPDATE invites
                SET status = $2
                WHERE id = $1
        `
	//status=1 is accept
	//status=2 is reject
	result, err := r.db.ExecContext(ctx, query, inviteID, status)
	if err != nil {
		return fmt.Errorf("failed to update invite status: %w", err)
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

func (r *InviteRepository) AcceptInvite(ctx context.Context, inviteID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// First get the invite details
	var invite models.Invite
	query := `
        SELECT carpool_id, to_user_email
        FROM invites
        WHERE id = $1
    `
	err = tx.QueryRowContext(ctx, query, inviteID).Scan(&invite.CarpoolID, &invite.ToUser)
	if err != nil {
		return fmt.Errorf("failed to get invite: %v", err)
	}

	// Get user ID from email
	var userID uuid.UUID
	userQuery := `
        SELECT id
        FROM users
        WHERE email = $1
    `
	err = tx.QueryRowContext(ctx, userQuery, invite.ToUser).Scan(&userID)
	if err != nil {
		return fmt.Errorf("failed to get user ID: %v", err)
	}

	// Add user to carpool_members
	memberQuery := `
        INSERT INTO carpool_members (carpool_id, user_id)
        VALUES ($1, $2)
        ON CONFLICT (carpool_id, user_id) DO NOTHING
    `
	_, err = tx.ExecContext(ctx, memberQuery, invite.CarpoolID, userID)
	if err != nil {
		return fmt.Errorf("failed to add carpool member: %v", err)
	}

	// Update invite status to accepted
	updateQuery := `
        UPDATE invites
        SET status = $1, updated_at = NOW()
        WHERE id = $2
    `
	_, err = tx.ExecContext(ctx, updateQuery, models.InviteStatusAccepted, inviteID)
	if err != nil {
		return fmt.Errorf("failed to update invite status: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Invite accepted and member added\",\"invite_id\":\"%s\"}", inviteID)
	return nil
}
