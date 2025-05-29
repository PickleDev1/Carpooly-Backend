package repository

import (
	"car-backend/pkg/models"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Delete the carpool (assuming carpool_stops table has ON DELETE CASCADE)
	result, err := tx.ExecContext(ctx, `DELETE FROM carpools WHERE id = $1`, carpoolID)
	if err != nil {
		return fmt.Errorf("failed to delete carpool: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows // No carpool found
	}

	// Log the operation
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Deleted carpool %s\"}", carpoolID)

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

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

func (r *CarPoolRepository) AddCarpoolMember(ctx context.Context, carpoolID, userID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// First add the member to carpool_members
	memberQuery := `
        INSERT INTO carpool_members (id, carpool_id, user_id, created_at, updated_at)
        VALUES (gen_random_uuid(), $1, $2, NOW(), NOW())
    `
	_, err = tx.ExecContext(ctx, memberQuery, carpoolID, userID)
	if err != nil {
		return fmt.Errorf("failed to add carpool member: %v", err)
	}

	// Get all rides (both past and future) for this carpool
	ridesQuery := `
        SELECT id, participants 
        FROM carpool_rides 
        WHERE carpool_id = $1
    `
	rows, err := tx.QueryContext(ctx, ridesQuery, carpoolID)
	if err != nil {
		return fmt.Errorf("failed to get carpool rides: %v", err)
	}
	defer rows.Close()

	// Get the user details to add to participants
	var user models.User
	userQuery := `
        SELECT id, clerk_id, email, name, display_name, city, state, created_at, updated_at
        FROM users
        WHERE id = $1
    `
	err = tx.QueryRowContext(ctx, userQuery, userID).Scan(
		&user.ID,
		&user.ClerkID,
		&user.Email,
		&user.Name,
		&user.DisplayName,
		&user.City,
		&user.State,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to get user details: %v", err)
	}

	// Update each ride's participants
	for rows.Next() {
		var rideID uuid.UUID
		var participantsJSON []byte
		err := rows.Scan(&rideID, &participantsJSON)
		if err != nil {
			return fmt.Errorf("failed to scan ride: %v", err)
		}

		var participants []models.User
		err = json.Unmarshal(participantsJSON, &participants)
		if err != nil {
			return fmt.Errorf("failed to unmarshal participants: %v", err)
		}

		// Add the new user to participants
		participants = append(participants, user)

		// Marshal updated participants back to JSON
		updatedParticipantsJSON, err := json.Marshal(participants)
		if err != nil {
			return fmt.Errorf("failed to marshal updated participants: %v", err)
		}

		// Update the ride
		updateQuery := `
            UPDATE carpool_rides 
            SET participants = $1, updated_at = NOW()
            WHERE id = $2
        `
		_, err = tx.ExecContext(ctx, updateQuery, updatedParticipantsJSON, rideID)
		if err != nil {
			return fmt.Errorf("failed to update ride participants: %v", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// Add methods like:
// UpdateCarPool
// SearchCarPools
