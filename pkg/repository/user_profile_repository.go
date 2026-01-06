package repository

import (
	"car-backend/pkg/models"
	"context"
	"database/sql"
	"fmt"
	"time"
)

type UserProfileRepository struct {
	db *sql.DB
}

func NewUserProfileRepository(db *sql.DB) *UserProfileRepository {
	return &UserProfileRepository{db: db}
}

// GetUserProfile retrieves a user's enhanced profile
func (r *UserProfileRepository) GetUserProfile(ctx context.Context, userID string) (*models.UserProfile, error) {
	query := `
		SELECT up.id, up.user_id, up.schedule, up.current_group_size, 
		       up.is_available_for_matching, up.last_active, up.created_at, up.updated_at
		FROM user_profiles up
		WHERE up.user_id = $1
	`

	var profile models.UserProfile
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&profile.ID, &profile.UserID, &profile.Schedule, &profile.CurrentGroupSize,
		&profile.IsAvailableForMatching, &profile.LastActive, &profile.CreatedAt, &profile.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Create default profile if none exists
		return r.createDefaultProfile(ctx, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("error getting user profile: %w", err)
	}

	return &profile, nil
}

// UpsertUserProfile creates or updates a user profile
func (r *UserProfileRepository) UpsertUserProfile(ctx context.Context, profile *models.UserProfile) error {
	query := `
		INSERT INTO user_profiles (
			user_id, schedule, current_group_size, is_available_for_matching, 
			last_active, updated_at
		) VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET
			schedule = EXCLUDED.schedule,
			current_group_size = EXCLUDED.current_group_size,
			is_available_for_matching = EXCLUDED.is_available_for_matching,
			last_active = EXCLUDED.last_active,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.ExecContext(ctx, query,
		profile.UserID, profile.Schedule, profile.CurrentGroupSize,
		profile.IsAvailableForMatching, profile.LastActive,
	)

	if err != nil {
		return fmt.Errorf("error upserting user profile: %w", err)
	}

	return nil
}

// GetAvailableUsers retrieves all users available for matching
func (r *UserProfileRepository) GetAvailableUsers(ctx context.Context) ([]*models.UserProfile, error) {
	query := `
		SELECT up.id, up.user_id, up.schedule, up.current_group_size,
		       up.is_available_for_matching, up.last_active, up.created_at, up.updated_at
		FROM user_profiles up
		WHERE up.is_available_for_matching = true 
		AND up.last_active > NOW() - INTERVAL '7 days'
		ORDER BY up.last_active DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error getting available users: %w", err)
	}
	defer rows.Close()

	var profiles []*models.UserProfile
	for rows.Next() {
		var profile models.UserProfile
		err := rows.Scan(
			&profile.ID, &profile.UserID, &profile.Schedule, &profile.CurrentGroupSize,
			&profile.IsAvailableForMatching, &profile.LastActive, &profile.CreatedAt, &profile.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning user profile: %w", err)
		}
		profiles = append(profiles, &profile)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user profiles: %w", err)
	}

	return profiles, nil
}

// UpdateLastActive updates the last active timestamp for a user
func (r *UserProfileRepository) UpdateLastActive(ctx context.Context, userID string) error {
	query := `
		UPDATE user_profiles 
		SET last_active = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("error updating last active: %w", err)
	}

	return nil
}

// createDefaultProfile creates a default profile for a user
func (r *UserProfileRepository) createDefaultProfile(ctx context.Context, userID string) (*models.UserProfile, error) {
	defaultProfile := &models.UserProfile{
		UserID: userID,
		Schedule: models.Schedule{
			DepartureTime:      "08:30",
			Frequency:          "daily",
			FlexibilityMinutes: 30,
			DaysOfWeek:         []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
		},
		CurrentGroupSize:       1,
		IsAvailableForMatching: true,
		LastActive:             time.Now(),
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	err := r.UpsertUserProfile(ctx, defaultProfile)
	if err != nil {
		return nil, fmt.Errorf("error creating default profile: %w", err)
	}

	return defaultProfile, nil
}
