package repository

import (
	"car-backend/pkg/models"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

type InviteRepository struct {
	db          *sql.DB
	carpoolRepo *CarPoolRepository
}

func NewInviteRepository(db *sql.DB) *InviteRepository {
	return &InviteRepository{
		db:          db,
		carpoolRepo: NewCarPoolRepository(db),
	}
}

func (r *InviteRepository) CreateInvite(ctx context.Context, invite *models.Invite) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	query := `
            INSERT INTO invites (
                    from_user, to_user_email, carpool_id, status
            ) VALUES ($1, $2, $3, $4)
            RETURNING id, created_at, updated_at
    `

	err = tx.QueryRowContext(
		ctx, query,
		invite.FromUser, invite.ToUser, invite.CarpoolID, invite.Status,
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
        SELECT id, from_user, to_user_email, carpool_id, status, created_at, updated_at
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

// GetAcceptedInvitesSentByUser fetches all accepted invites sent by a specific user
func (r *InviteRepository) GetAcceptedInvitesSentByUser(ctx context.Context, fromUserID uuid.UUID) ([]models.Invite, error) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Getting accepted invites sent by user\",\"from_user_id\":\"%s\"}", fromUserID)

	// First, let's check if there are any invites at all for this user (any status)
	checkQuery := `SELECT COUNT(*) FROM invites WHERE from_user = $1`
	var totalInvites int
	err := r.db.QueryRowContext(ctx, checkQuery, fromUserID).Scan(&totalInvites)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to check total invites count\",\"error\":\"%v\"}", err)
	} else {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Total invites sent by user (all statuses)\",\"count\":%d,\"from_user_id\":\"%s\"}", totalInvites, fromUserID)
	}

	// Check how many accepted invites there are
	acceptedQuery := `SELECT COUNT(*) FROM invites WHERE from_user = $1 AND status = 1`
	var acceptedInvites int
	err = r.db.QueryRowContext(ctx, acceptedQuery, fromUserID).Scan(&acceptedInvites)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to check accepted invites count\",\"error\":\"%v\"}", err)
	} else {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Accepted invites sent by user\",\"count\":%d,\"from_user_id\":\"%s\"}", acceptedInvites, fromUserID)
	}

	// Check all statuses for this user
	statusQuery := `SELECT status, COUNT(*) FROM invites WHERE from_user = $1 GROUP BY status`
	statusRows, err := r.db.QueryContext(ctx, statusQuery, fromUserID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to check invite statuses\",\"error\":\"%v\"}", err)
	} else {
		defer statusRows.Close()
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Invite status breakdown for user\",\"from_user_id\":\"%s\"}", fromUserID)
		for statusRows.Next() {
			var status int
			var count int
			if statusRows.Scan(&status, &count) == nil {
				log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Status count\",\"status\":%d,\"count\":%d}", status, count)
			}
		}
	}

	query := `
        SELECT 
            i.id,
            i.carpool_id,
            i.from_user,
            i.to_user_email,
            i.status,
            i.created_at,
            i.updated_at,
            c.carpool_name,
            u.email as sender_email
        FROM invites i
        JOIN carpools c ON i.carpool_id = c.id
        JOIN users u ON i.from_user = u.id
        WHERE i.from_user = $1 AND i.status = 1
        ORDER BY i.updated_at DESC
    `

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Executing main query\",\"query\":\"%s\",\"from_user_id\":\"%s\"}",
		query, fromUserID)

	rows, err := r.db.QueryContext(ctx, query, fromUserID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Database query failed\",\"error\":\"%v\"}", err)
		return nil, fmt.Errorf("failed to get accepted invites: %v", err)
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
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Found accepted invite\",\"invite_id\":\"%s\",\"to_user\":\"%s\",\"carpool_name\":\"%s\"}",
			invite.ID, invite.ToUser, invite.CarpoolName)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Retrieved accepted invites\",\"count\":%d,\"from_user_id\":\"%s\"}",
		len(invites), fromUserID)
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
		return err
	}
	defer tx.Rollback()

	// 1. Update invite status
	_, err = tx.ExecContext(ctx, `UPDATE invites SET status = 1 WHERE id = $1`, inviteID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update invite status\",\"error\":\"%v\"}", err)
		tx.Rollback()
		return err
	}

	// 2. Get invite details
	var carpoolID uuid.UUID
	var toUserEmail string
	err = tx.QueryRowContext(ctx, `SELECT carpool_id, to_user_email FROM invites WHERE id = $1`, inviteID).Scan(&carpoolID, &toUserEmail)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get invite details\",\"error\":\"%v\"}", err)
		tx.Rollback()
		return err
	}

	// 3. Get user by email
	var user models.User
	err = tx.QueryRowContext(ctx, `SELECT id, clerk_id, email, name, display_name, city, state, created_at, updated_at FROM users WHERE email = $1`, toUserEmail).Scan(
		&user.ID, &user.ClerkID, &user.Email, &user.Name, &user.DisplayName, &user.City, &user.State, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user by email\",\"error\":\"%v\"}", err)
		tx.Rollback()
		return err
	}

	// 4. Add to carpool_members
	_, err = tx.ExecContext(ctx, `INSERT INTO carpool_members (id, carpool_id, user_id, created_at, updated_at) VALUES (gen_random_uuid(), $1, $2, NOW(), NOW()) ON CONFLICT (carpool_id, user_id) DO NOTHING`, carpoolID, user.ID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to insert into carpool_members\",\"error\":\"%v\"}", err)
		tx.Rollback()
		return err
	}

	// 4.5. Decrement available seats (but not below 0)
	_, err = tx.ExecContext(ctx, `UPDATE carpools SET available_seats = GREATEST(available_seats - 1, 0), updated_at = NOW() WHERE id = $1`, carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update available seats\",\"error\":\"%v\"}", err)
		tx.Rollback()
		return err
	}

	// 5. For each ride (past and future), update participants in Go
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Processing rides for user addition\",\"carpool_id\":\"%s\",\"user_id\":\"%s\",\"user_email\":\"%s\"}", carpoolID, user.ID, user.Email)
	rows, err := tx.QueryContext(ctx, `SELECT id, participants, start_time FROM carpool_rides WHERE carpool_id = $1 ORDER BY start_time ASC`, carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to select carpool rides\",\"carpool_id\":\"%s\",\"error\":\"%v\"}", carpoolID, err)
		tx.Rollback()
		return err
	}
	defer rows.Close()

	rideCount := 0
	addedToRides := 0
	alreadyInRides := 0
	errorCount := 0

	for rows.Next() {
		rideCount++
		var rideID uuid.UUID
		var participantsJSON []byte
		var startTime time.Time
		if err := rows.Scan(&rideID, &participantsJSON, &startTime); err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to scan ride row\",\"ride_count\":%d,\"ride_id\":\"%v\",\"error\":\"%v\"}", rideCount, rideID, err)
			errorCount++
			continue
		}
		// Remove the time check - add to ALL rides, not just future ones
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Processing ride for participant update\",\"ride_count\":%d,\"ride_id\":\"%v\",\"user_id\":\"%v\",\"start_time\":\"%v\"}", rideCount, rideID, user.ID, startTime)
		var participants []models.User
		if len(participantsJSON) > 0 {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Unmarshalling participants JSON\",\"ride_id\":\"%v\",\"json_length\":%d}", rideID, len(participantsJSON))
			if err := json.Unmarshal(participantsJSON, &participants); err != nil {
				log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to unmarshal participants JSON\",\"ride_id\":\"%v\",\"error\":\"%v\"}", rideID, err)
				errorCount++
				continue
			}
		}

		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Current participants in ride\",\"ride_id\":\"%v\",\"participant_count\":%d}", rideID, len(participants))

		// Check if user already in participants
		alreadyIn := false
		for _, p := range participants {
			if p.ID == user.ID {
				alreadyIn = true
				break
			}
		}
		if !alreadyIn {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Adding user to participants\",\"ride_id\":\"%v\",\"user_id\":\"%v\",\"before_count\":%d}", rideID, user.ID, len(participants))
			participants = append(participants, user)
			updatedJSON, err := json.Marshal(participants)
			if err != nil {
				log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to marshal updated participants\",\"ride_id\":\"%v\",\"error\":\"%v\"}", rideID, err)
				errorCount++
				continue
			}
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Updating ride participants in DB\",\"ride_id\":\"%v\",\"user_id\":\"%v\",\"after_count\":%d}", rideID, user.ID, len(participants))
			_, err = tx.ExecContext(ctx,
				"UPDATE carpool_rides SET participants = $1, updated_at = NOW() WHERE id = $2",
				updatedJSON, rideID,
			)
			if err != nil {
				log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update ride participants in DB\",\"ride_id\":\"%v\",\"error\":\"%+v\"}", rideID, err)
				errorCount++
				continue
			}
			addedToRides++
			log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully added user to ride participants\",\"ride_id\":\"%v\",\"user_id\":\"%v\"}", rideID, user.ID)
		} else {
			alreadyInRides++
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"User already in participants\",\"ride_id\":\"%v\",\"user_id\":\"%v\"}", rideID, user.ID)
		}
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Ride processing completed\",\"carpool_id\":\"%s\",\"user_id\":\"%s\",\"total_rides\":%d,\"added_to_rides\":%d,\"already_in_rides\":%d,\"errors\":%d}",
		carpoolID, user.ID, rideCount, addedToRides, alreadyInRides, errorCount)

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
