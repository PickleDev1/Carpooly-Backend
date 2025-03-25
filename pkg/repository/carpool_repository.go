package repository

import (
	"car-backend/pkg/models"
	"context"
	"database/sql"
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

// Add methods like:
// UpdateCarPool
// SearchCarPools
