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
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Creating carpool ride\",\"carpool_id\":\"%s\",\"start_time\":\"%s\",\"start_time_zero\":%v}",
		ride.CarpoolID, ride.StartTime.Format(time.RFC3339), ride.StartTime.IsZero())

	// Convert participants to JSON
	participantsJSON, err := json.Marshal(ride.Participants)
	if err != nil {
		return fmt.Errorf("failed to marshal participants: %v", err)
	}

	query := `
			INSERT INTO carpool_rides (
				carpool_id, start_time, status, participants, created_at, updated_at,
				driver_id, location_lat, location_lng, miles_saved
			) VALUES ($1, $2, 0, $3, NOW(), NOW(), NULL, NULL, NULL, 0)
			RETURNING id, created_at, updated_at
	`

	err = tx.QueryRowContext(ctx, query,
		ride.CarpoolID, ride.StartTime, participantsJSON,
	).Scan(&ride.ID, &ride.CreatedAt, &ride.UpdatedAt)

	if err != nil {
		log.Printf("Failed to insert carpool ride: %v", err)
		return fmt.Errorf("failed to create carpool ride: %w", err)
	}

	// Set default values for the returned ride object
	ride.DriverID = nil // Set to nil since it's optional
	ride.LocationLat = nil
	ride.LocationLng = nil
	ride.MilesSaved = nil
	ride.Status = 0 // Default status

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

func (r *CarPoolRideRepository) UpdateCarpoolRideStatus(ctx context.Context, rideID uuid.UUID, status int) error {
	query := `
			UPDATE carpool_rides
			SET status = $2, updated_at = NOW() 
			WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, rideID, status)
	if err != nil {
		return fmt.Errorf("failed to update carpool ride status: %w", err)
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

func (r *CarPoolRideRepository) GetUserActiveRides(ctx context.Context, userID string) ([]models.CarpoolRide, error) {
	query := `
        SELECT DISTINCT cr.id, cr.carpool_id, cr.driver_id, cr.status, 
               cr.location_lat, cr.location_lng, cr.miles_saved, 
               cr.created_at, cr.updated_at
        FROM carpool_rides cr
        JOIN carpools c ON cr.carpool_id = c.id
        JOIN carpool_members cm ON c.id = cm.carpool_id
        WHERE cm.user_id = $1
        AND cr.status = 1  -- Active status
        ORDER BY cr.created_at DESC
    `

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Executing query for user active rides\",\"userID\":\"%s\"}", userID)

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active rides: %v", err)
	}
	defer rows.Close()

	var rides []models.CarpoolRide
	for rows.Next() {
		var ride models.CarpoolRide
		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan ride: %v", err)
		}
		rides = append(rides, ride)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Found active rides\",\"count\":%d}", len(rides))
	return rides, nil
}

func (r *CarPoolRideRepository) RemoveParticipant(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) error {
	query := `
        UPDATE carpool_rides
        SET 
            participants = (
                SELECT COALESCE(
                    jsonb_agg(participant)
                    FILTER (WHERE (participant->>'id')::uuid != $2),
                    '[]'::jsonb
                )
                FROM jsonb_array_elements(participants) participant
            ),
            updated_at = NOW()
        WHERE id = $1
    `

	result, err := r.db.ExecContext(ctx, query, rideID, userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to remove participant\",\"error\":\"%v\"}", err)
		return fmt.Errorf("failed to remove participant: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("ride not found")
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Participant removed from ride\",\"ride_id\":\"%s\",\"user_id\":\"%s\"}",
		rideID, userID)
	return nil
}

func (r *CarPoolRideRepository) GetCarpoolRidesByDate(ctx context.Context, carpoolID uuid.UUID, date time.Time) ([]models.CarpoolRide, error) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Fetching rides\",\"carpool_id\":\"%s\",\"date\":\"%s\"}",
		carpoolID, date.Format("2006-01-02"))

	query := `
        SELECT id, carpool_id, driver_id, start_time, status, 
               location_lat, location_lng, miles_saved, participants,
               created_at, updated_at
        FROM carpool_rides
        WHERE carpool_id = $1 
        AND DATE(start_time) = $2::date
    `

	rows, err := r.db.QueryContext(ctx, query, carpoolID, date.Format("2006-01-02"))
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Query failed\",\"error\":\"%v\"}", err)
		return nil, fmt.Errorf("failed to query carpool rides: %w", err)
	}
	defer rows.Close()

	var rides []models.CarpoolRide
	for rows.Next() {
		var ride models.CarpoolRide
		var participantsJSON []byte
		var locationLat, locationLng, milesSaved sql.NullFloat64
		var driverID sql.NullString

		err := rows.Scan(
			&ride.ID,
			&ride.CarpoolID,
			&driverID,
			&ride.StartTime,
			&ride.Status,
			&locationLat,
			&locationLng,
			&milesSaved,
			&participantsJSON,
			&ride.CreatedAt,
			&ride.UpdatedAt,
		)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"Scan failed\",\"error\":\"%v\"}", err)
			return nil, fmt.Errorf("failed to scan ride: %w", err)
		}

		// Convert NULL values to zero values
		if locationLat.Valid {
			ride.LocationLat = &locationLat.Float64
		}
		if locationLng.Valid {
			ride.LocationLng = &locationLng.Float64
		}
		if milesSaved.Valid {
			ride.MilesSaved = &milesSaved.Float64
		}
		if driverID.Valid {
			parsedDriverID, _ := uuid.Parse(driverID.String)
			ride.DriverID = &parsedDriverID
		}

		if err := json.Unmarshal(participantsJSON, &ride.Participants); err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"JSON unmarshal failed\",\"error\":\"%v\"}", err)
			return nil, fmt.Errorf("failed to unmarshal participants: %w", err)
		}

		rides = append(rides, ride)
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Found rides\",\"count\":%d}", len(rides))

	if rides == nil {
		rides = []models.CarpoolRide{}
	}

	return rides, nil
}

func (r *CarPoolRideRepository) UpdateCarpoolRideDriver(ctx context.Context, rideID uuid.UUID, driverID uuid.UUID) error {
	query := `
        UPDATE carpool_rides
        SET driver_id = $2, updated_at = NOW()
        WHERE id = $1
        RETURNING id
    `

	var returnedID uuid.UUID
	err := r.db.QueryRowContext(ctx, query, rideID, driverID).Scan(&returnedID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("carpool ride not found")
		}
		return fmt.Errorf("failed to update carpool ride driver: %w", err)
	}

	return nil
}

func (r *CarPoolRideRepository) GetUserTotalRides(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
        SELECT COUNT(DISTINCT cr.id)
        FROM carpool_rides cr
        WHERE cr.participants @> json_build_array(
            json_build_object(
                'id', $1::uuid
            )
        )::jsonb
        AND cr.start_time < NOW()  -- Only count rides that have passed
        AND cr.status = 2  -- Only count completed rides
    `

	var totalRides int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&totalRides)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get total rides\",\"error\":\"%v\"}", err)
		return 0, fmt.Errorf("failed to get total rides: %v", err)
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Retrieved total rides\",\"user_id\":\"%s\",\"total_rides\":%d,\"query_time\":\"%s\"}",
		userID, totalRides, time.Now().Format(time.RFC3339))
	return totalRides, nil
}

func (r *CarPoolRideRepository) GetActiveRides(ctx context.Context, userID uuid.UUID, timezoneStr string) ([]models.CarpoolRide, error) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetActiveRides called\",\"user_id\":\"%s\",\"timezone\":\"%s\"}", userID, timezoneStr)

	// Parse timezone
	loc, err := time.LoadLocation(timezoneStr)
	if err != nil {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"Invalid timezone, using UTC\",\"timezone\":\"%s\",\"error\":\"%v\"}", timezoneStr, err)
		loc = time.UTC
		timezoneStr = "UTC"
	}

	// Get current time in user's timezone
	now := time.Now().In(loc)
	windowStart := now.Add(-90 * time.Minute)
	windowEnd := now.Add(90 * time.Minute)

	// Convert to UTC for DB query
	windowStartUTC := windowStart.UTC()
	windowEndUTC := windowEnd.UTC()

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Active rides window\",\"window_start\":\"%s\",\"window_end\":\"%s\"}", windowStartUTC.Format(time.RFC3339), windowEndUTC.Format(time.RFC3339))

	query := `
		SELECT cr.id, cr.carpool_id, cr.driver_id, cr.start_time, cr.status, 
		       cr.location_lat, cr.location_lng, cr.miles_saved, cr.participants, 
		       cr.created_at, cr.updated_at
		FROM carpool_rides cr
		JOIN carpools c ON cr.carpool_id = c.id
		JOIN carpool_members cm ON c.id = cm.carpool_id
		WHERE cm.user_id = $1
		  AND cr.start_time >= $2
		  AND cr.start_time <= $3
		ORDER BY cr.start_time ASC NULLS LAST, cr.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, windowStartUTC, windowEndUTC)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetActiveRides query failed\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return nil, fmt.Errorf("failed to query active rides: %w", err)
	}
	defer rows.Close()

	var rides []models.CarpoolRide
	for rows.Next() {
		var ride models.CarpoolRide
		var participantsJSON []byte
		var locationLat, locationLng, milesSaved sql.NullFloat64
		var driverID sql.NullString
		var startTime sql.NullTime

		err := rows.Scan(
			&ride.ID,
			&ride.CarpoolID,
			&driverID,
			&startTime,
			&ride.Status,
			&locationLat,
			&locationLng,
			&milesSaved,
			&participantsJSON,
			&ride.CreatedAt,
			&ride.UpdatedAt,
		)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetActiveRides scan failed\",\"error\":\"%v\"}", err)
			return nil, fmt.Errorf("failed to scan ride: %v", err)
		}

		// Handle NULL start_time
		if startTime.Valid {
			ride.StartTime = startTime.Time
		} else {
			// If start_time is NULL, use created_at as a fallback
			ride.StartTime = ride.CreatedAt
		}

		// Convert NULL values to zero values
		if locationLat.Valid {
			ride.LocationLat = &locationLat.Float64
		}
		if locationLng.Valid {
			ride.LocationLng = &locationLng.Float64
		}
		if milesSaved.Valid {
			ride.MilesSaved = &milesSaved.Float64
		}
		if driverID.Valid {
			parsedDriverID, _ := uuid.Parse(driverID.String)
			ride.DriverID = &parsedDriverID
		}

		if err := json.Unmarshal(participantsJSON, &ride.Participants); err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"JSON unmarshal failed\",\"error\":\"%v\"}", err)
			return nil, fmt.Errorf("failed to unmarshal participants: %w", err)
		}

		// Log each ride found for debugging
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Found active ride\",\"ride_id\":\"%s\",\"start_time\":\"%s\",\"status\":%d}",
			ride.ID, ride.StartTime.Format(time.RFC3339), ride.Status)

		rides = append(rides, ride)
	}

	if err = rows.Err(); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetActiveRides rows error\",\"error\":\"%v\"}", err)
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetActiveRides found rides\",\"user_id\":\"%s\",\"timezone\":\"%s\",\"ride_count\":%d}", userID, timezoneStr, len(rides))
	return rides, nil
}
