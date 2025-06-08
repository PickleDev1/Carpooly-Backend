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

type CarPoolRepository struct {
	db *sql.DB
}

func NewCarPoolRepository(db *sql.DB) *CarPoolRepository {
	return &CarPoolRepository{db: db}
}

func (r *CarPoolRepository) CreateCarPool(ctx context.Context, carpool *models.Carpool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Insert main carpool record
	query := `
            INSERT INTO carpools (
                creator_id, carpool_name, status, recurring_option,
                available_seats, destination_address, seats
            ) VALUES ($1, $2, $3, $4, $5, $6, $7)
            RETURNING id, created_at, updated_at`

	err = tx.QueryRowContext(
		ctx, query,
		carpool.CreatorID, carpool.CarpoolName, carpool.Status,
		carpool.RecurringOption, carpool.AvailableSeats, carpool.DestinationAddress, carpool.Seats,
	).Scan(&carpool.ID, &carpool.CreatedAt, &carpool.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert carpool: %v", err)
	}

	// Insert carpool member (creator)
	memberQuery := `
        INSERT INTO carpool_members (carpool_id, user_id)
        VALUES ($1, $2)
    `

	_, err = tx.ExecContext(ctx, memberQuery, carpool.ID, carpool.CreatorID)
	if err != nil {
		return fmt.Errorf("failed to insert carpool member: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

func (r *CarPoolRepository) GetCarPool(ctx context.Context, carpoolID uuid.UUID) (*models.Carpool, error) {
	// Create a new Carpool struct to store the retrieved data
	carpool := &models.Carpool{}

	// Begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Get carpool details
	carpoolQuery := `
        SELECT id, creator_id, carpool_name, status, recurring_option,
               available_seats, destination_address, seats, created_at, updated_at
        FROM carpools
        WHERE id = $1`

	err = tx.QueryRowContext(ctx, carpoolQuery, carpoolID).Scan(
		&carpool.ID,
		&carpool.CreatorID,
		&carpool.CarpoolName,
		&carpool.Status,
		&carpool.RecurringOption,
		&carpool.AvailableSeats,
		&carpool.DestinationAddress,
		&carpool.Seats,
		&carpool.CreatedAt,
		&carpool.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get carpool: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return carpool, nil
}

func (r *CarPoolRepository) DeleteCarPool(ctx context.Context, carpoolID uuid.UUID) error {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Starting carpool deletion transaction\",\"carpool_id\":\"%s\"}", carpoolID)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to begin transaction\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			carpoolID, err)
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// First delete all associated rides
	deleteRidesQuery := `DELETE FROM carpool_rides WHERE carpool_id = $1`
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Deleting associated rides\",\"carpool_id\":\"%s\"}", carpoolID)

	_, err = tx.ExecContext(ctx, deleteRidesQuery, carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to delete associated rides\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			carpoolID, err)
		return fmt.Errorf("failed to delete associated rides: %v", err)
	}

	// Delete all carpool members
	deleteMembersQuery := `DELETE FROM carpool_members WHERE carpool_id = $1`
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Deleting carpool members\",\"carpool_id\":\"%s\"}", carpoolID)

	_, err = tx.ExecContext(ctx, deleteMembersQuery, carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to delete carpool members\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			carpoolID, err)
		return fmt.Errorf("failed to delete carpool members: %v", err)
	}

	// Delete all invites
	deleteInvitesQuery := `DELETE FROM invites WHERE carpool_id = $1`
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Deleting carpool invites\",\"carpool_id\":\"%s\"}", carpoolID)

	_, err = tx.ExecContext(ctx, deleteInvitesQuery, carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to delete carpool invites\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			carpoolID, err)
		return fmt.Errorf("failed to delete carpool invites: %v", err)
	}

	// Finally delete the carpool
	deleteCarpoolQuery := `DELETE FROM carpools WHERE id = $1`
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Deleting carpool\",\"carpool_id\":\"%s\"}", carpoolID)

	result, err := tx.ExecContext(ctx, deleteCarpoolQuery, carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to delete carpool\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			carpoolID, err)
		return fmt.Errorf("failed to delete carpool: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get rows affected\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			carpoolID, err)
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		log.Printf("{\"severity\":\"INFO\",\"message\":\"No carpool found to delete\",\"carpool_id\":\"%s\"}", carpoolID)
		return sql.ErrNoRows
	}

	if err = tx.Commit(); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to commit transaction\",\"carpool_id\":\"%s\",\"error\":\"%v\"}",
			carpoolID, err)
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Successfully deleted carpool and all associated data\",\"carpool_id\":\"%s\"}", carpoolID)
	return nil
}

func (r *CarPoolRepository) GetCarpoolsByCreatorID(ctx context.Context, creatorID string) ([]models.Carpool, error) {
	var carpools []models.Carpool

	query := `
        SELECT id, creator_id, carpool_name, status, recurring_option,
               available_seats, destination_address, seats, created_at, updated_at
        FROM carpools
        WHERE creator_id = $1
    `

	rows, err := r.db.QueryContext(ctx, query, creatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to query carpools: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var carpool models.Carpool
		err := rows.Scan(
			&carpool.ID,
			&carpool.CreatorID,
			&carpool.CarpoolName,
			&carpool.Status,
			&carpool.RecurringOption,
			&carpool.AvailableSeats,
			&carpool.DestinationAddress,
			&carpool.Seats,
			&carpool.CreatedAt,
			&carpool.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan carpool: %v", err)
		}
		carpools = append(carpools, carpool)
	}

	return carpools, nil
}

func (r *CarPoolRepository) GetUserCarpools(ctx context.Context, userID uuid.UUID) ([]models.Carpool, error) {
	var carpools []models.Carpool

	// Join carpools and carpool_members tables to get carpools where the user is a member
	query := `
           SELECT DISTINCT c.* 
                FROM carpools c
                JOIN carpool_members cm ON c.id = cm.carpool_id
                WHERE cm.user_id = $1

    `

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user carpools: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var carpool models.Carpool
		err := rows.Scan(
			&carpool.ID,
			&carpool.CreatorID,
			&carpool.CarpoolName,
			&carpool.Status,
			&carpool.RecurringOption,
			&carpool.AvailableSeats,
			&carpool.DestinationAddress,
			&carpool.Seats,
			&carpool.CreatedAt,
			&carpool.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan carpool: %w", err)
		}
		carpools = append(carpools, carpool)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over carpool rows: %w", err)
	}

	return carpools, nil
}

func (r *CarPoolRepository) GetCarpoolMembers(ctx context.Context, carpoolID uuid.UUID) ([]models.User, error) {
	query := `
        SELECT u.id, u.clerk_id, u.email, u.name, u.display_name, 
               u.city, u.state, u.created_at, u.updated_at
        FROM users u
        JOIN carpool_members cm ON u.id = cm.user_id
        WHERE cm.carpool_id = $1
        ORDER BY u.created_at DESC
    `

	rows, err := r.db.QueryContext(ctx, query, carpoolID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to query carpool members\",\"error\":\"%v\"}", err)
		return nil, fmt.Errorf("failed to query carpool members: %v", err)
	}
	defer rows.Close()

	var members []models.User
	for rows.Next() {
		var member models.User
		err := rows.Scan(
			&member.ID,
			&member.ClerkID,
			&member.Email,
			&member.Name,
			&member.DisplayName,
			&member.City,
			&member.State,
			&member.CreatedAt,
			&member.UpdatedAt,
		)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to scan member\",\"error\":\"%v\"}", err)
			return nil, fmt.Errorf("failed to scan member: %v", err)
		}
		members = append(members, member)
	}

	return members, nil
}

// AddCarpoolMemberByAPI adds a user to carpool_members
func (r *CarPoolRepository) AddCarpoolMemberByAPI(ctx context.Context, carpoolID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO carpool_members (carpool_id, user_id, created_at, updated_at)
        VALUES ($1, $2, NOW(), NOW())
        ON CONFLICT (carpool_id, user_id) DO NOTHING
    `, carpoolID, userID)
	return err
}

// AddUserToFutureRides adds a user to all future rides' participants
func (r *CarPoolRepository) AddUserToFutureRides(ctx context.Context, carpoolID, userID uuid.UUID) error {
	// Get user details
	var user models.User
	err := r.db.QueryRowContext(ctx, `
        SELECT id, clerk_id, email, name, display_name, city, state, created_at, updated_at
        FROM users WHERE id = $1
    `, userID).Scan(
		&user.ID, &user.ClerkID, &user.Email, &user.Name, &user.DisplayName, &user.City, &user.State, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return err
	}
	// Get all future rides
	rows, err := r.db.QueryContext(ctx, `SELECT id, participants, start_time FROM carpool_rides WHERE carpool_id = $1`, carpoolID)
	if err != nil {
		return err
	}
	defer rows.Close()
	now := time.Now()
	for rows.Next() {
		var rideID uuid.UUID
		var participantsJSON []byte
		var startTime time.Time
		if err := rows.Scan(&rideID, &participantsJSON, &startTime); err != nil {
			return err
		}
		if startTime.Before(now) {
			continue
		}
		var participants []models.User
		if len(participantsJSON) > 0 {
			if err := json.Unmarshal(participantsJSON, &participants); err != nil {
				return err
			}
		}
		alreadyIn := false
		for _, p := range participants {
			if p.ID == user.ID {
				alreadyIn = true
				break
			}
		}
		if !alreadyIn {
			participants = append(participants, user)
			updatedJSON, err := json.Marshal(participants)
			if err != nil {
				return err
			}
			_, err = r.db.ExecContext(ctx, `UPDATE carpool_rides SET participants = $1, updated_at = NOW() WHERE id = $2`, updatedJSON, rideID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Add methods like:
// UpdateCarPool
// SearchCarPools
