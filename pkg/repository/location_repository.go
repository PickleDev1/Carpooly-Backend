package repository

import (
	"context"
	"database/sql"
	"time"

	"car-backend/pkg/models"

	"github.com/google/uuid"
)

type LocationRepository struct {
	db *sql.DB
}

func NewLocationRepository(db *sql.DB) *LocationRepository {
	return &LocationRepository{db: db}
}

// UpdateLocation updates or inserts a new location record
func (r *LocationRepository) UpdateLocation(ctx context.Context, userID, rideID uuid.UUID, location *models.UpdateLocationRequest) error {
	query := `
		INSERT INTO location_tracking (user_id, carpool_ride_id, latitude, longitude, timestamp)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		userID,
		rideID,
		location.Latitude,
		location.Longitude,
		time.Now(),
	)

	return err
}

// UpdateHomeLocation updates a user's home location
func (r *LocationRepository) UpdateHomeLocation(ctx context.Context, userID uuid.UUID, location *models.HomeLocationUpdate) error {
	query := `
		UPDATE users
		SET home_latitude = $1, home_longitude = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query,
		location.Latitude,
		location.Longitude,
		userID,
	)

	return err
}

// GetLatestLocation retrieves the most recent location for a user in a carpool ride
func (r *LocationRepository) GetLatestLocation(ctx context.Context, userID, rideID uuid.UUID) (*models.Location, error) {
	query := `
		SELECT id, user_id, carpool_ride_id, latitude, longitude, timestamp, created_at
		FROM location_tracking
		WHERE user_id = $1 AND carpool_ride_id = $2
		ORDER BY timestamp DESC
		LIMIT 1
	`

	location := &models.Location{}
	err := r.db.QueryRowContext(ctx, query, userID, rideID).Scan(
		&location.ID,
		&location.UserID,
		&location.CarpoolRideID,
		&location.Latitude,
		&location.Longitude,
		&location.Timestamp,
		&location.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return location, nil
}

// GetLocationHistory retrieves location history for a user in a carpool ride
func (r *LocationRepository) GetLocationHistory(ctx context.Context, userID, rideID uuid.UUID, limit int) ([]*models.Location, error) {
	if limit <= 0 {
		limit = 10 // default limit
	}

	query := `
		SELECT id, user_id, carpool_ride_id, latitude, longitude, timestamp, created_at
		FROM location_tracking
		WHERE user_id = $1 AND carpool_ride_id = $2
		ORDER BY timestamp DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, rideID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []*models.Location
	for rows.Next() {
		location := &models.Location{}
		err := rows.Scan(
			&location.ID,
			&location.UserID,
			&location.CarpoolRideID,
			&location.Latitude,
			&location.Longitude,
			&location.Timestamp,
			&location.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		locations = append(locations, location)
	}

	return locations, nil
}

// UpdateLocationSettings updates a user's location sharing settings
func (r *LocationRepository) UpdateLocationSettings(ctx context.Context, userID uuid.UUID, settings *models.LocationSettings) error {
	query := `
		UPDATE users
		SET location_sharing_enabled = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, settings.LocationSharingEnabled, userID)
	return err
}

// GetLocationSettings retrieves a user's location sharing settings
func (r *LocationRepository) GetLocationSettings(ctx context.Context, userID uuid.UUID) (*models.LocationSettings, error) {
	query := `
		SELECT location_sharing_enabled, home_latitude, home_longitude
		FROM users
		WHERE id = $1
	`

	settings := &models.LocationSettings{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&settings.LocationSharingEnabled,
		&settings.HomeLatitude,
		&settings.HomeLongitude,
	)
	if err != nil {
		return nil, err
	}

	return settings, nil
}

// GetAllLatestLocationsForRide returns the latest location for each user in a ride
func (r *LocationRepository) GetAllLatestLocationsForRide(ctx context.Context, rideID uuid.UUID) ([]*models.Location, error) {
	query := `
		SELECT DISTINCT ON (user_id) id, user_id, carpool_ride_id, latitude, longitude, timestamp, created_at
		FROM location_tracking
		WHERE carpool_ride_id = $1
		ORDER BY user_id, timestamp DESC
	`

	rows, err := r.db.QueryContext(ctx, query, rideID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []*models.Location
	for rows.Next() {
		location := &models.Location{}
		err := rows.Scan(
			&location.ID,
			&location.UserID,
			&location.CarpoolRideID,
			&location.Latitude,
			&location.Longitude,
			&location.Timestamp,
			&location.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		locations = append(locations, location)
	}

	return locations, nil
}
