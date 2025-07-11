package repository

import (
	"car-backend/pkg/models"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) AddUserActivity(ctx context.Context, activity *models.UserActivity) error {
	dataJSON, err := json.Marshal(activity.Data)
	if err != nil {
		return err
	}
	query := `
        INSERT INTO user_activity (user_id, activity_type, related_id, related_type, description, data, timestamp)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `
	_, err = r.db.ExecContext(ctx, query,
		activity.UserID,
		activity.Type,
		activity.RelatedID,
		activity.RelatedType,
		activity.Description,
		dataJSON,
		activity.Timestamp,
	)
	return err
}

func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) error {
	// Generate a new UUID if ID is not provided
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	query := `
        INSERT INTO users (id, email, name, display_name, city, state, clerk_id, location_sharing_enabled, home_latitude, home_longitude)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT (email) DO UPDATE SET
            name = EXCLUDED.name,
            display_name = EXCLUDED.display_name,
            city = EXCLUDED.city,
            state = EXCLUDED.state,
            clerk_id = EXCLUDED.clerk_id,
            location_sharing_enabled = EXCLUDED.location_sharing_enabled,
            home_latitude = EXCLUDED.home_latitude,
            home_longitude = EXCLUDED.home_longitude,
            updated_at = NOW()
    `

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.Name, user.DisplayName, user.City, user.State, user.ClerkID,
		user.LocationSharingEnabled, user.HomeLatitude, user.HomeLongitude,
	)
	return err
}

func (r *UserRepository) UpdateProfile(ctx context.Context, userID string, update *models.UpdateUserProfile) error {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"UpdateProfile called\",\"user_id\":\"%s\"}", userID)

	// Build dynamic query based on which fields are provided
	var setClauses []string
	var args []interface{}
	argIndex := 1

	// Helper function to add field if not nil
	addField := func(field interface{}, columnName string) {
		if field != nil {
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", columnName, argIndex))
			args = append(args, field)
			argIndex++
		}
	}

	// Add fields that are provided
	addField(update.DisplayName, "display_name")
	addField(update.City, "city")
	addField(update.State, "state")
	addField(update.LocationSharingEnabled, "location_sharing_enabled")
	addField(update.HomeLatitude, "home_latitude")
	addField(update.HomeLongitude, "home_longitude")

	// If no fields to update, return early
	if len(setClauses) == 0 {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"No fields to update\",\"user_id\":\"%s\"}", userID)
		return nil
	}

	// Add updated_at
	setClauses = append(setClauses, fmt.Sprintf("updated_at = CURRENT_TIMESTAMP"))

	// Build the query
	query := fmt.Sprintf(`
		UPDATE users 
		SET %s
		WHERE id = $%d
	`, strings.Join(setClauses, ", "), argIndex)
	args = append(args, userID)

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Executing UpdateProfile query\",\"query\":\"%s\",\"args\":%v}", query, args)

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to update profile\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return fmt.Errorf("failed to update profile: %v", err)
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"Profile updated successfully\",\"user_id\":\"%s\",\"fields_updated\":%d}", userID, len(setClauses)-1) // -1 for updated_at
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.DisplayName,
		&user.City,
		&user.State,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.ClerkID,
		&user.LocationSharingEnabled,
		&user.HomeLatitude,
		&user.HomeLongitude,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) CreateUserIfNotExists(ctx context.Context, user *models.User) error {
	// Check if user exists
	var exists bool
	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)",
		user.Email,
	).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check user existence: %v", err)
	}

	if exists {
		// User already exists, no need to create
		return nil
	}

	query := `
        INSERT INTO users (
            id, email, name, display_name, city, state, location_sharing_enabled, home_latitude, home_longitude
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING created_at, updated_at`

	err = r.db.QueryRowContext(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.DisplayName,
		user.City,
		user.State,
		user.LocationSharingEnabled,
		user.HomeLatitude,
		user.HomeLongitude,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	return nil
}

func (r *UserRepository) GetUserIDByClerkID(ctx context.Context, clerkID string) (uuid.UUID, error) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetUserIDByClerkID called\",\"clerk_id\":\"%s\",\"clerk_id_length\":%d}", clerkID, len(clerkID))

	// Trim whitespace and check for empty
	clerkID = strings.TrimSpace(clerkID)
	if clerkID == "" {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Empty clerk_id after trimming\"}")
		return uuid.Nil, fmt.Errorf("empty clerk_id")
	}

	var userID uuid.UUID

	// Query to get user ID from users table using clerk_id
	query := `
        SELECT id 
        FROM users 
        WHERE clerk_id = $1
    `

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Executing query\",\"query\":\"%s\",\"clerk_id\":\"%s\"}", query, clerkID)

	err := r.db.QueryRowContext(ctx, query, clerkID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"No user found for clerk_id\",\"clerk_id\":\"%s\"}", clerkID)

			// Let's also check if there are any similar clerk_ids for debugging
			var similarClerkIDs []string
			rows, debugErr := r.db.QueryContext(ctx, "SELECT clerk_id FROM users WHERE clerk_id ILIKE $1 LIMIT 5", "%"+clerkID+"%")
			if debugErr == nil {
				defer rows.Close()
				for rows.Next() {
					var similarID string
					if rows.Scan(&similarID) == nil {
						similarClerkIDs = append(similarClerkIDs, similarID)
					}
				}
			}
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Similar clerk_ids found\",\"similar_ids\":%v}", similarClerkIDs)

			return uuid.Nil, fmt.Errorf("no user found for clerk_id: %s", clerkID)
		}
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Database error in GetUserIDByClerkID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", clerkID, err)
		return uuid.Nil, fmt.Errorf("error querying user: %w", err)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Successfully found user ID\",\"clerk_id\":\"%s\",\"user_id\":\"%s\"}", clerkID, userID)
	return userID, nil
}

func (r *UserRepository) GetUserByClerkID(clerkID string) (*models.User, error) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Getting user by Clerk ID\",\"clerk_id\":\"%s\"}", clerkID)

	var user models.User
	query := `
        SELECT id, clerk_id, email, name, display_name, city, state, location_sharing_enabled, home_latitude, home_longitude, created_at, updated_at
        FROM users
        WHERE clerk_id = $1
    `

	err := r.db.QueryRow(query, clerkID).Scan(
		&user.ID,
		&user.ClerkID,
		&user.Email,
		&user.Name,
		&user.DisplayName,
		&user.City,
		&user.State,
		&user.LocationSharingEnabled,
		&user.HomeLatitude,
		&user.HomeLongitude,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"No user found\",\"clerk_id\":\"%s\"}", clerkID)
			return nil, fmt.Errorf("user not found")
		}
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Database error getting user\",\"clerk_id\":\"%s\",\"error\":\"%v\"}",
			clerkID, err)
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Found user\",\"clerk_id\":\"%s\",\"user_id\":\"%s\"}",
		user.ClerkID, user.ID)
	return &user, nil
}

func (r *UserRepository) GetUserByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `
        SELECT id, clerk_id, email, name, display_name, city, state, location_sharing_enabled, home_latitude, home_longitude, created_at, updated_at
        FROM users
        WHERE id = $1
    `
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.ClerkID,
		&user.Email,
		&user.Name,
		&user.DisplayName,
		&user.City,
		&user.State,
		&user.LocationSharingEnabled,
		&user.HomeLatitude,
		&user.HomeLongitude,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetUserActivities(ctx context.Context, userID uuid.UUID, limit int) ([]models.UserActivity, error) {
	query := `
		SELECT id, user_id, activity_type, related_id, related_type, description, data, timestamp
		FROM user_activity
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []models.UserActivity
	for rows.Next() {
		var a models.UserActivity
		var dataRaw []byte
		if err := rows.Scan(&a.ID, &a.UserID, &a.Type, &a.RelatedID, &a.RelatedType, &a.Description, &dataRaw, &a.Timestamp); err != nil {
			return nil, err
		}
		if len(dataRaw) > 0 {
			_ = json.Unmarshal(dataRaw, &a.Data)
		}
		activities = append(activities, a)
	}
	return activities, nil
}

// DeleteUser deletes a user and handles all related data cleanup
func (r *UserRepository) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	log.Printf("{\"severity\":\"INFO\",\"message\":\"Starting user deletion\",\"user_id\":\"%s\"}", userID)

	// Start a transaction to ensure data consistency
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to begin transaction\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// 1. Remove user from carpool_members
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Removing user from carpool_members\",\"user_id\":\"%s\"}", userID)
	_, err = tx.ExecContext(ctx, "DELETE FROM carpool_members WHERE user_id = $1", userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to remove from carpool_members\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return fmt.Errorf("failed to remove from carpool_members: %v", err)
	}

	// 2. Remove user from carpool_rides participants (JSON array)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Removing user from carpool_rides participants\",\"user_id\":\"%s\"}", userID)
	_, err = tx.ExecContext(ctx, `
		UPDATE carpool_rides 
		SET participants = (
			SELECT jsonb_agg(participant)
			FROM jsonb_array_elements(participants) AS participant
			WHERE participant->>'id' != $1::text
		)
		WHERE participants @> jsonb_build_array(jsonb_build_object('id', $1::text))
	`, userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to remove from carpool_rides participants\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return fmt.Errorf("failed to remove from carpool_rides participants: %v", err)
	}

	// 3. Remove user's invites (both sent and received)
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Removing user's invites\",\"user_id\":\"%s\"}", userID)
	_, err = tx.ExecContext(ctx, "DELETE FROM invites WHERE from_user = $1", userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to remove invites\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return fmt.Errorf("failed to remove invites: %v", err)
	}

	// 4. Remove user's location data
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Removing user's location data\",\"user_id\":\"%s\"}", userID)
	_, err = tx.ExecContext(ctx, "DELETE FROM location_tracking WHERE user_id = $1", userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to remove location data\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return fmt.Errorf("failed to remove location data: %v", err)
	}

	// 5. Remove user's activity records
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Removing user's activity records\",\"user_id\":\"%s\"}", userID)
	_, err = tx.ExecContext(ctx, "DELETE FROM user_activity WHERE user_id = $1", userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to remove activity records\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return fmt.Errorf("failed to remove activity records: %v", err)
	}

	// 6. Finally, delete the user record
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"Deleting user record\",\"user_id\":\"%s\"}", userID)
	_, err = tx.ExecContext(ctx, "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to delete user record\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return fmt.Errorf("failed to delete user record: %v", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to commit transaction\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"User deleted successfully\",\"user_id\":\"%s\"}", userID)
	return nil
}
