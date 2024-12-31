package repository

import (
	"car-backend/pkg/models"
    "context"
    "database/sql"
    "fmt"
	"log"
	"github.com/google/uuid"
)

type CarPoolRideRepository struct {
	db *sql.DB
}

func NewCarPoolRideRepository(db *sql.DB) *CarPoolRideRepository {
	return &CarPoolRideRepository{db: db}
}

func (r *CarPoolRideRepository) CreateCarpoolRide(ctx context.Context, ride *models.CarpoolRide) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
			return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	log.Printf("Creating carpool ride for carpoolID: %s", ride.CarpoolID) 

	query := `
			INSERT INTO carpool_rides (
				carpool_id, driver_id, status, location_lat, location_lng, miles_saved
			) VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, created_at, updated_at
	`

	err = tx.QueryRowContext(ctx, query,
			ride.CarpoolID, ride.DriverID, ride.Status, ride.LocationLat, ride.LocationLng, ride.MilesSaved,
	).Scan(&ride.ID, &ride.CreatedAt, &ride.UpdatedAt)

	if err != nil {
			log.Printf("Failed to insert carpool ride: %v", err)
			return fmt.Errorf("failed to create carpool ride: %w", err)
	}

	log.Printf("Carpool ride created successfully: %v", ride.ID)

	if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

func (r *CarPoolRideRepository) GetCarpoolRide(ctx context.Context, rideID uuid.UUID) (*models.CarpoolRide, error) {
	ride := &models.CarpoolRide{}

	query := `
			SELECT id, carpool_id, driver_id, status, location_lat, location_lng, miles_saved, created_at, updated_at
			FROM carpool_rides
			WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, rideID).Scan(
			&ride.ID,
			&ride.CarpoolID,
			&ride.DriverID,
			&ride.Status,
			&ride.LocationLat,
			&ride.LocationLng,
			&ride.MilesSaved,
			&ride.CreatedAt,
			&ride.UpdatedAt,
	)
	if err != nil {
			if err == sql.ErrNoRows {
					return nil, nil
			}
			return nil, fmt.Errorf("failed to get carpool ride: %w", err)
	}

	return ride, nil
}

func (r *CarPoolRideRepository) DeleteCarpoolRide(ctx context.Context, carpoolID uuid.UUID, rideID uuid.UUID) error {
	query := `
                DELETE FROM carpool_rides
                WHERE id = $1 AND carpool_id = $2
        `

	result, err := r.db.ExecContext(ctx, query, rideID, carpoolID)
	if err != nil {
			return fmt.Errorf("failed to delete carpool ride: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
			return fmt.Errorf("carpool ride not found")
	}

	return nil
}