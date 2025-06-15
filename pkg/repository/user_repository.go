package repository

import (
	"car-backend/pkg/models"
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
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
	query := `
        UPDATE users 
        SET 
            display_name = COALESCE($1, display_name),
            city = COALESCE($2, city),
            state = COALESCE($3, state),
            location_sharing_enabled = COALESCE($4, location_sharing_enabled),
            home_latitude = COALESCE($5, home_latitude),
            home_longitude = COALESCE($6, home_longitude),
            updated_at = CURRENT_TIMESTAMP
        WHERE id = $7
    `
	_, err := r.db.ExecContext(ctx, query,
		update.DisplayName,
		update.City,
		update.State,
		update.LocationSharingEnabled,
		update.HomeLatitude,
		update.HomeLongitude,
		userID,
	)
	return err
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
	var userID uuid.UUID

	// Query to get user ID from users table using clerk_id
	query := `
        SELECT id 
        FROM users 
        WHERE clerk_id = $1
    `

	err := r.db.QueryRowContext(ctx, query, clerkID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"No user found for clerk_id\",\"clerk_id\":\"%s\"}", clerkID)
			return uuid.Nil, fmt.Errorf("no user found for clerk_id: %s", clerkID)
		}
		return uuid.Nil, fmt.Errorf("error querying user: %w", err)
	}

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
