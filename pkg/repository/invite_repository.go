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

func (r *InviteRepository) GetUserInvites(ctx context.Context, userID uuid.UUID) ([]models.Invite, error) {
	var invites []models.Invite
	var userEmail string

	// First get the user's email
	emailQuery := `SELECT email FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, emailQuery, userID).Scan(&userEmail)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user email\",\"error\":\"%v\"}", err)
		return nil, fmt.Errorf("failed to get user email: %v", err)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Found user email\",\"email\":\"%s\"}", userEmail)

	// Then get the invites using the email
	query := `
        SELECT 
            i.*,
            c.carpool_name,
            u.email as sender_email
        FROM invites i
        JOIN carpools c ON i.carpool_id = c.id
        JOIN users u ON i.from_user = u.id
        WHERE i.to_user_email = $1 AND i.status = 0
    `
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Executing invites query\",\"email\":\"%s\"}", userEmail)

	rows, err := r.db.QueryContext(ctx, query, userEmail)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Database query failed\",\"error\":\"%v\"}", err)
		return nil, fmt.Errorf("failed to get user invites: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var invite models.Invite
		var carpoolName string
		var senderEmail string

		err := rows.Scan(
			&invite.ID,
			&invite.FromUser,
			&invite.ToUser,
			&invite.CarpoolID,
			&invite.Message,
			&invite.Status,
			&invite.CreatedAt,
			&invite.UpdatedAt,
			&carpoolName,
			&senderEmail,
		)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to scan row\",\"error\":\"%v\"}", err)
			return nil, fmt.Errorf("failed to scan invite row: %v", err)
		}

		invite.CarpoolName = carpoolName
		invite.SenderEmail = senderEmail

		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Invite details\",\"sender_email\":\"%s\",\"carpool_name\":\"%s\"}",
			senderEmail, carpoolName)

		invites = append(invites, invite)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over invite rows: %v", err)
	}
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Retrieved invites\",\"count\":%d}", len(invites))
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
