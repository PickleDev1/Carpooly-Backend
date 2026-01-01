package repository

import (
	"car-backend/pkg/models"
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/google/uuid"
)

type MatchingRepository struct {
	db *sql.DB
}

func NewMatchingRepository(db *sql.DB) *MatchingRepository {
	return &MatchingRepository{db: db}
}

// User Matching Preferences Methods

func (r *MatchingRepository) GetUserMatchingPreferences(ctx context.Context, userID string, companyID *uuid.UUID) (*models.UserMatchingPreferences, error) {
	// CRITICAL SECURITY LOGGING: Track the user ID being queried
	log.Printf("{\"severity\":\"SECURITY\",\"message\":\"GetUserMatchingPreferences: Starting database query\",\"user_id\":\"%s\",\"company_id\":\"%v\"}", userID, companyID)

	// Phase 3: Add company_id filter to prevent data leaks
	// When companyID is nil (default), filter by company_id IS NULL (personal scope)
	// When companyID is provided, filter by company_id = companyID (company scope)
	query := `
		SELECT id, user_id, max_detour_minutes, preferred_group_size, driver_preference,
			schedule_flexibility_minutes, max_pickup_distance_miles, min_compatibility_score,
			notification_preferences, user_demographics, demographic_preferences, 
			destination_latitude, destination_longitude, arrival_time, commute_days,
			company_id, site_id, is_active, created_at, updated_at
		FROM user_matching_preferences 
		WHERE user_id = $1 
		  AND (company_id = $2 OR ($2 IS NULL AND company_id IS NULL))
	`

	// CRITICAL SECURITY LOGGING: Log the exact query and parameters
	log.Printf("{\"severity\":\"SECURITY\",\"message\":\"GetUserMatchingPreferences: Executing query with user_id and company_id filter\",\"query\":\"%s\",\"user_id\":\"%s\",\"company_id\":\"%v\"}", query, userID, companyID)

	var prefs models.UserMatchingPreferences
	var destinationLatitude, destinationLongitude sql.NullFloat64
	var arrivalTime sql.NullString
	var commuteDays sql.NullString
	var companyIDVal, siteIDVal sql.NullString

	err := r.db.QueryRowContext(ctx, query, userID, companyID).Scan(
		&prefs.ID, &prefs.UserID, &prefs.MaxDetourMinutes, &prefs.PreferredGroupSize, &prefs.DriverPreference,
		&prefs.ScheduleFlexibilityMinutes, &prefs.MaxPickupDistanceMiles, &prefs.MinCompatibilityScore,
		&prefs.NotificationPreferences, &prefs.UserDemographics, &prefs.DemographicPreferences,
		&destinationLatitude, &destinationLongitude, &arrivalTime, &commuteDays,
		&companyIDVal, &siteIDVal, &prefs.IsActive, &prefs.CreatedAt, &prefs.UpdatedAt,
	)

	// Handle NULL values properly
	if destinationLatitude.Valid {
		prefs.DestinationLatitude = &destinationLatitude.Float64
	}
	if destinationLongitude.Valid {
		prefs.DestinationLongitude = &destinationLongitude.Float64
	}
	if arrivalTime.Valid {
		prefs.ArrivalTime = &arrivalTime.String
	}
	if commuteDays.Valid {
		// Parse the array string into []string
		// PostgreSQL returns arrays as strings like "{mon,tue,wed}"
		commuteDaysStr := commuteDays.String
		if commuteDaysStr != "" && commuteDaysStr != "{}" {
			// Remove curly braces and split by comma
			commuteDaysStr = strings.Trim(commuteDaysStr, "{}")
			if commuteDaysStr != "" {
				prefs.CommuteDays = strings.Split(commuteDaysStr, ",")
			}
		}
	} else {
		// If NULL, set to empty slice
		prefs.CommuteDays = []string{}
	}

	// Handle company_id and site_id (Phase 2 columns)
	if companyIDVal.Valid {
		prefs.CompanyID = &companyIDVal.String
	}
	if siteIDVal.Valid {
		prefs.SiteID = &siteIDVal.String
	}

	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetUserMatchingPreferences: Database scan error\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)

		// If the new columns don't exist, try the old query
		// Check if error is due to missing columns (backward compatibility for old migrations)
		if strings.Contains(err.Error(), "destination_latitude") || strings.Contains(err.Error(), "destination_longitude") ||
			strings.Contains(err.Error(), "arrival_time") || strings.Contains(err.Error(), "commute_days") ||
			strings.Contains(err.Error(), "company_id") || strings.Contains(err.Error(), "site_id") ||
			strings.Contains(err.Error(), "id") {
			log.Printf("{\"severity\":\"INFO\",\"message\":\"GetUserMatchingPreferences: New columns not found, using fallback query\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)

			// Fallback query without new columns (for databases that haven't run Phase 2 migrations yet)
			fallbackQuery := `
				SELECT user_id, max_detour_minutes, preferred_group_size, driver_preference,
					schedule_flexibility_minutes, max_pickup_distance_miles, min_compatibility_score,
					notification_preferences, user_demographics, demographic_preferences, 
					is_active, created_at, updated_at
				FROM user_matching_preferences 
				WHERE user_id = $1
			`

			err = r.db.QueryRowContext(ctx, fallbackQuery, userID).Scan(
				&prefs.UserID, &prefs.MaxDetourMinutes, &prefs.PreferredGroupSize, &prefs.DriverPreference,
				&prefs.ScheduleFlexibilityMinutes, &prefs.MaxPickupDistanceMiles, &prefs.MinCompatibilityScore,
				&prefs.NotificationPreferences, &prefs.UserDemographics, &prefs.DemographicPreferences,
				&prefs.IsActive, &prefs.CreatedAt, &prefs.UpdatedAt,
			)

			// Set default values for new fields (0.0 for coordinates, null for schedule, null for company fields)
			prefs.DestinationLatitude = func() *float64 { v := 0.0; return &v }()
			prefs.DestinationLongitude = func() *float64 { v := 0.0; return &v }()
			prefs.ArrivalTime = nil
			prefs.CommuteDays = nil
			prefs.CompanyID = nil
			prefs.SiteID = nil
		}
	}

	if err == sql.ErrNoRows {
		log.Printf("{\"severity\":\"SECURITY\",\"message\":\"GetUserMatchingPreferences: No preferences found for user\",\"user_id\":\"%s\"}", userID)
		return nil, nil // No preferences found
	}
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetUserMatchingPreferences: Database query failed\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return nil, fmt.Errorf("error getting user matching preferences: %w", err)
	}

	// CRITICAL SECURITY LOGGING: Log the data being returned to verify it belongs to the correct user
	log.Printf("{\"severity\":\"SECURITY\",\"message\":\"GetUserMatchingPreferences: Query successful, returning data\",\"user_id\":\"%s\",\"returned_user_id\":\"%s\",\"destination_lat\":\"%v\",\"destination_lng\":\"%v\"}",
		userID, prefs.UserID, prefs.DestinationLatitude, prefs.DestinationLongitude)

	// CRITICAL FIX: Ensure destination coordinates are 0.0 if not set (not null)
	if prefs.DestinationLatitude == nil {
		prefs.DestinationLatitude = func() *float64 { v := 0.0; return &v }()
	}
	if prefs.DestinationLongitude == nil {
		prefs.DestinationLongitude = func() *float64 { v := 0.0; return &v }()
	}

	// Ensure backward compatibility by setting default values if fields are empty
	if prefs.UserDemographics.AgeRange == "" {
		prefs.UserDemographics = models.UserDemographics{
			AgeRange:      "26-35",
			Gender:        "prefer_not_to_say",
			Occupation:    "",
			StudentStatus: "not_student",
			Company:       "",
		}
	}

	if len(prefs.DemographicPreferences.AgePreferences) == 0 {
		prefs.DemographicPreferences = models.DemographicPreferences{
			AgePreferences:        []string{"18-25", "26-35", "36-45", "46-55"},
			GenderPreferences:     []string{"any"},
			StudentPreference:     "both",
			OccupationPreferences: []string{},
		}
	}

	return &prefs, nil
}

func (r *MatchingRepository) UpsertUserMatchingPreferences(ctx context.Context, prefs *models.UserMatchingPreferences, companyID *uuid.UUID) error {
	// Migration 026 was run - id is PRIMARY KEY, user_id is not unique
	// Use SELECT-then-UPDATE/INSERT approach (works for both schemas)
	
	// Check if record exists (for personal preferences: company_id IS NULL)
	var existingID uuid.UUID
	checkQuery := `
		SELECT id FROM user_matching_preferences 
		WHERE user_id = $1 AND company_id IS NULL
	`
	err := r.db.QueryRowContext(ctx, checkQuery, prefs.UserID).Scan(&existingID)

	if err == nil {
		// Record exists, update it
		updateQuery := `
			UPDATE user_matching_preferences SET
				max_detour_minutes = $2,
				preferred_group_size = $3,
				driver_preference = $4,
				schedule_flexibility_minutes = $5,
				max_pickup_distance_miles = $6,
				min_compatibility_score = $7,
				notification_preferences = $8,
				user_demographics = $9,
				demographic_preferences = $10,
				destination_latitude = $11,
				destination_longitude = $12,
				arrival_time = $13,
				commute_days = $14,
				is_active = $15,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
		`
		_, err = r.db.ExecContext(ctx, updateQuery,
			existingID,
			prefs.MaxDetourMinutes, prefs.PreferredGroupSize, prefs.DriverPreference,
			prefs.ScheduleFlexibilityMinutes, prefs.MaxPickupDistanceMiles, prefs.MinCompatibilityScore,
			prefs.NotificationPreferences, prefs.UserDemographics, prefs.DemographicPreferences,
			prefs.DestinationLatitude, prefs.DestinationLongitude, prefs.ArrivalTime, prefs.CommuteDays,
			prefs.IsActive,
		)
	} else if err == sql.ErrNoRows {
		// Record doesn't exist, insert it (with company_id IS NULL for personal)
		// Try with company_id column first (migration 026 run), fall back if column doesn't exist
		insertQuery := `
			INSERT INTO user_matching_preferences (
				user_id, company_id, site_id, max_detour_minutes, preferred_group_size, 
				driver_preference, schedule_flexibility_minutes, max_pickup_distance_miles, 
				min_compatibility_score, notification_preferences, user_demographics, 
				demographic_preferences, destination_latitude, destination_longitude, 
				arrival_time, commute_days, is_active, created_at, updated_at
			) VALUES ($1, NULL, NULL, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`
		_, err = r.db.ExecContext(ctx, insertQuery,
			prefs.UserID,
			prefs.MaxDetourMinutes, prefs.PreferredGroupSize, prefs.DriverPreference,
			prefs.ScheduleFlexibilityMinutes, prefs.MaxPickupDistanceMiles, prefs.MinCompatibilityScore,
			prefs.NotificationPreferences, prefs.UserDemographics, prefs.DemographicPreferences,
			prefs.DestinationLatitude, prefs.DestinationLongitude, prefs.ArrivalTime, prefs.CommuteDays,
			prefs.IsActive,
		)
		
		// If company_id column doesn't exist (migration 026 not run), try without it
		if err != nil && strings.Contains(err.Error(), "column") && strings.Contains(err.Error(), "does not exist") {
			insertQueryNoCompany := `
				INSERT INTO user_matching_preferences (
					user_id, max_detour_minutes, preferred_group_size, 
					driver_preference, schedule_flexibility_minutes, max_pickup_distance_miles, 
					min_compatibility_score, notification_preferences, user_demographics, 
					demographic_preferences, destination_latitude, destination_longitude, 
					arrival_time, commute_days, is_active, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			`
			_, err = r.db.ExecContext(ctx, insertQueryNoCompany,
				prefs.UserID,
				prefs.MaxDetourMinutes, prefs.PreferredGroupSize, prefs.DriverPreference,
				prefs.ScheduleFlexibilityMinutes, prefs.MaxPickupDistanceMiles, prefs.MinCompatibilityScore,
				prefs.NotificationPreferences, prefs.UserDemographics, prefs.DemographicPreferences,
				prefs.DestinationLatitude, prefs.DestinationLongitude, prefs.ArrivalTime, prefs.CommuteDays,
				prefs.IsActive,
			)
		}
	}

	if err != nil {
		return fmt.Errorf("error upserting user matching preferences: %w", err)
	}

	return nil
}

// Potential Matches Methods

func (r *MatchingRepository) GetPotentialMatches(ctx context.Context, userID string, status string) ([]*models.PotentialMatch, error) {
	query := `
		SELECT pm.id, pm.user1_id, pm.user2_id, pm.compatibility_score, pm.route_overlap_percentage,
		       pm.total_distance_miles, pm.estimated_savings_per_month, pm.match_reasons, pm.status,
		       pm.expires_at, pm.created_at, pm.updated_at,
		       u1.id, u1.name, u1.display_name, u1.home_latitude, u1.home_longitude,
		       u2.id, u2.name, u2.display_name, u2.home_latitude, u2.home_longitude
		FROM potential_matches pm
		JOIN users u1 ON (pm.user1_id = u1.id OR pm.user2_id = u1.id) AND u1.id::text != $1
		JOIN users u2 ON (pm.user1_id = u2.id OR pm.user2_id = u2.id) AND u2.id::text != $1
		WHERE (pm.user1_id::text = $1 OR pm.user2_id::text = $1)
	`

	if status != "" {
		query += " AND pm.status = $2"
	}

	query += " ORDER BY pm.compatibility_score DESC"

	var rows *sql.Rows
	var err error

	if status != "" {
		rows, err = r.db.QueryContext(ctx, query, userID, status)
	} else {
		rows, err = r.db.QueryContext(ctx, query, userID)
	}

	if err != nil {
		return nil, fmt.Errorf("error getting potential matches: %w", err)
	}
	defer rows.Close()

	var matches []*models.PotentialMatch
	for rows.Next() {
		var match models.PotentialMatch
		var user1, user2 models.User

		err := rows.Scan(
			&match.ID, &match.User1ID, &match.User2ID, &match.CompatibilityScore, &match.RouteOverlapPercentage,
			&match.TotalDistanceMiles, &match.EstimatedSavingsPerMonth, &match.MatchReasons, &match.Status,
			&match.ExpiresAt, &match.CreatedAt, &match.UpdatedAt,
			&user1.ID, &user1.Name, &user1.DisplayName, &user1.HomeLatitude, &user1.HomeLongitude,
			&user2.ID, &user2.Name, &user2.DisplayName, &user2.HomeLatitude, &user2.HomeLongitude,
		)

		if err != nil {
			return nil, fmt.Errorf("error scanning potential match: %w", err)
		}

		// Determine which user is the other user (not the requesting user)
		if user1.ID.String() == userID {
			match.User2 = &user2
		} else {
			match.User2 = &user1
		}

		matches = append(matches, &match)
	}

	// Ensure we always return empty slices instead of nil
	if matches == nil {
		matches = []*models.PotentialMatch{}
	}

	return matches, nil
}

func (r *MatchingRepository) CreatePotentialMatch(ctx context.Context, match *models.PotentialMatch) error {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"CreatePotentialMatch: Starting database insert\",\"user1_id\":\"%s\",\"user2_id\":\"%s\",\"compatibility_score\":%.2f}", match.User1ID, match.User2ID, match.CompatibilityScore)

	query := `
		INSERT INTO potential_matches (
			user1_id, user2_id, compatibility_score, route_overlap_percentage,
			total_distance_miles, estimated_savings_per_month, match_reasons, status, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user1_id, user2_id) DO UPDATE SET
			compatibility_score = EXCLUDED.compatibility_score,
			route_overlap_percentage = EXCLUDED.route_overlap_percentage,
			total_distance_miles = EXCLUDED.total_distance_miles,
			estimated_savings_per_month = EXCLUDED.estimated_savings_per_month,
			match_reasons = EXCLUDED.match_reasons,
			status = EXCLUDED.status,
			expires_at = EXCLUDED.expires_at,
			updated_at = CURRENT_TIMESTAMP
	`

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"CreatePotentialMatch: Executing query\",\"user1_id\":\"%s\",\"user2_id\":\"%s\"}", match.User1ID, match.User2ID)
	_, err := r.db.ExecContext(ctx, query,
		match.User1ID, match.User2ID, match.CompatibilityScore, match.RouteOverlapPercentage,
		match.TotalDistanceMiles, match.EstimatedSavingsPerMonth, match.MatchReasons, match.Status, match.ExpiresAt,
	)

	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreatePotentialMatch: Database error\",\"user1_id\":\"%s\",\"user2_id\":\"%s\",\"error\":\"%v\"}", match.User1ID, match.User2ID, err)
		return fmt.Errorf("error creating potential match: %w", err)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"CreatePotentialMatch: Successfully created/updated potential match\",\"user1_id\":\"%s\",\"user2_id\":\"%s\"}", match.User1ID, match.User2ID)
	return nil
}

// Match Requests Methods

func (r *MatchingRepository) GetMatchRequests(ctx context.Context, userID string, companyID *uuid.UUID) ([]*models.MatchRequest, []*models.MatchRequest, error) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetMatchRequests: Starting query\",\"user_id\":\"%s\",\"company_id\":\"%v\"}", userID, companyID)

	// Phase 3: Add company_id filter to prevent data leaks
	// Get incoming requests (requests sent TO this user)
	incomingQuery := `
		SELECT mr.id, mr.from_user_id, mr.to_user_id, mr.potential_match_id, 
		       mr.message, mr.preferred_carpool_size, mr.carpool_name, mr.status, 
		       mr.company_id, mr.site_id, mr.created_at, mr.updated_at, mr.expires_at,
		       u.name, u.display_name, u.home_latitude, u.home_longitude
		FROM match_requests mr
		JOIN users u ON mr.from_user_id = u.id
		WHERE mr.to_user_id = $1 
		  AND mr.status != 'expired'
		  AND (mr.company_id = $2 OR ($2 IS NULL AND mr.company_id IS NULL))
		ORDER BY mr.created_at DESC
	`

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetMatchRequests: Executing incoming query\",\"query\":\"%s\",\"user_id\":\"%s\",\"company_id\":\"%v\"}", incomingQuery, userID, companyID)

	incomingRows, err := r.db.QueryContext(ctx, incomingQuery, userID, companyID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting incoming requests: %w", err)
	}
	defer incomingRows.Close()

	var incoming []*models.MatchRequest
	for incomingRows.Next() {
		var request models.MatchRequest
		var userName string
		var userDisplayName sql.NullString
		var userHomeLat, userHomeLng sql.NullFloat64
		var companyIDVal, siteIDVal sql.NullString

		err := incomingRows.Scan(
			&request.ID, &request.FromUserID, &request.ToUserID, &request.PotentialMatchID,
			&request.Message, &request.PreferredCarpoolSize, &request.CarpoolName, &request.Status,
			&companyIDVal, &siteIDVal, &request.CreatedAt, &request.UpdatedAt, &request.ExpiresAt,
			&userName, &userDisplayName, &userHomeLat, &userHomeLng,
		)

		// Handle company_id and site_id
		if companyIDVal.Valid {
			request.CompanyID = &companyIDVal.String
		}
		if siteIDVal.Valid {
			request.SiteID = &siteIDVal.String
		}
		if err != nil {
			return nil, nil, fmt.Errorf("error scanning incoming request: %w", err)
		}

		// Add user details to the request
		request.FromUser = &models.User{
			ID:            uuid.MustParse(request.FromUserID),
			Name:          userName,
			DisplayName:   userDisplayName,
			HomeLatitude:  userHomeLat.Float64,
			HomeLongitude: userHomeLng.Float64,
		}

		incoming = append(incoming, &request)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetMatchRequests: Incoming query completed\",\"user_id\":\"%s\",\"rows_found\":%d}", userID, len(incoming))

	// Get outgoing requests (requests sent BY this user)
	outgoingQuery := `
		SELECT mr.id, mr.from_user_id, mr.to_user_id, mr.potential_match_id, 
		       mr.message, mr.preferred_carpool_size, mr.carpool_name, mr.status,
		       mr.company_id, mr.site_id, mr.created_at, mr.updated_at, mr.expires_at,
		       u.name, u.display_name, u.home_latitude, u.home_longitude
		FROM match_requests mr
		JOIN users u ON mr.to_user_id = u.id
		WHERE mr.from_user_id = $1 
		  AND mr.status != 'expired'
		  AND (mr.company_id = $2 OR ($2 IS NULL AND mr.company_id IS NULL))
		ORDER BY mr.created_at DESC
	`

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetMatchRequests: Executing outgoing query\",\"query\":\"%s\",\"user_id\":\"%s\",\"company_id\":\"%v\"}", outgoingQuery, userID, companyID)

	outgoingRows, err := r.db.QueryContext(ctx, outgoingQuery, userID, companyID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting outgoing requests: %w", err)
	}
	defer outgoingRows.Close()

	var outgoing []*models.MatchRequest
	for outgoingRows.Next() {
		var request models.MatchRequest
		var userName string
		var userDisplayName sql.NullString
		var userHomeLat, userHomeLng sql.NullFloat64
		var companyIDVal, siteIDVal sql.NullString

		err := outgoingRows.Scan(
			&request.ID, &request.FromUserID, &request.ToUserID, &request.PotentialMatchID,
			&request.Message, &request.PreferredCarpoolSize, &request.CarpoolName, &request.Status,
			&companyIDVal, &siteIDVal, &request.CreatedAt, &request.UpdatedAt, &request.ExpiresAt,
			&userName, &userDisplayName, &userHomeLat, &userHomeLng,
		)

		// Handle company_id and site_id
		if companyIDVal.Valid {
			request.CompanyID = &companyIDVal.String
		}
		if siteIDVal.Valid {
			request.SiteID = &siteIDVal.String
		}
		if err != nil {
			return nil, nil, fmt.Errorf("error scanning outgoing request: %w", err)
		}

		// Add user details to the request
		request.ToUser = &models.User{
			ID:            uuid.MustParse(request.ToUserID),
			Name:          userName,
			DisplayName:   userDisplayName,
			HomeLatitude:  userHomeLat.Float64,
			HomeLongitude: userHomeLng.Float64,
		}

		outgoing = append(outgoing, &request)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetMatchRequests: Outgoing query completed\",\"user_id\":\"%s\",\"rows_found\":%d}", userID, len(outgoing))
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetMatchRequests: Total results\",\"user_id\":\"%s\",\"incoming_count\":%d,\"outgoing_count\":%d}", userID, len(incoming), len(outgoing))

	return incoming, outgoing, nil
}

func (r *MatchingRepository) CreateMatchRequest(ctx context.Context, request *models.MatchRequest) error {
	// Phase 3: Include company_id and site_id in INSERT
	query := `
		INSERT INTO match_requests (
			from_user_id, to_user_id, potential_match_id, message, preferred_carpool_size, 
			carpool_name, status, expires_at, company_id, site_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`

	// Handle potential_match_id - it might be empty or null
	var potentialMatchID interface{}
	if request.PotentialMatchID != "" {
		potentialMatchID = request.PotentialMatchID
	} else {
		potentialMatchID = nil
	}

	// Handle company_id and site_id (can be nil for personal requests)
	var companyIDVal, siteIDVal interface{}
	if request.CompanyID != nil && *request.CompanyID != "" {
		companyIDVal = *request.CompanyID
	} else {
		companyIDVal = nil
	}
	if request.SiteID != nil && *request.SiteID != "" {
		siteIDVal = *request.SiteID
	} else {
		siteIDVal = nil
	}

	err := r.db.QueryRowContext(ctx, query,
		request.FromUserID, request.ToUserID, potentialMatchID,
		request.Message, request.PreferredCarpoolSize, request.CarpoolName, request.Status, request.ExpiresAt,
		companyIDVal, siteIDVal,
	).Scan(&request.ID, &request.CreatedAt, &request.UpdatedAt)

	if err != nil {
		return fmt.Errorf("error creating match request: %w", err)
	}

	return nil
}

func (r *MatchingRepository) UpdateMatchRequestStatus(ctx context.Context, requestID string, status string) error {
	query := `
        UPDATE match_requests 
        SET status = $1, updated_at = CURRENT_TIMESTAMP, acted_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, status, requestID)
	if err != nil {
		return fmt.Errorf("error updating match request status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no match request found with ID: %s", requestID)
	}

	return nil
}

// Matching Session Methods

func (r *MatchingRepository) GetMatchingSession(ctx context.Context, userID string) (*models.MatchingSession, error) {
	query := `
		SELECT id, user_id, status, last_match_generated_at, expires_at, created_at, updated_at
		FROM matching_sessions
		WHERE user_id = $1
	`

	var session models.MatchingSession
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&session.ID, &session.UserID, &session.Status, &session.LastMatchGeneratedAt,
		&session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No session found
	}
	if err != nil {
		return nil, fmt.Errorf("error getting matching session: %w", err)
	}

	return &session, nil
}

func (r *MatchingRepository) UpsertMatchingSession(ctx context.Context, session *models.MatchingSession) error {
	query := `
		INSERT INTO matching_sessions (
			user_id, status, last_match_generated_at, expires_at
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			status = EXCLUDED.status,
			last_match_generated_at = EXCLUDED.last_match_generated_at,
			expires_at = EXCLUDED.expires_at,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		session.UserID, session.Status, session.LastMatchGeneratedAt, session.ExpiresAt,
	).Scan(&session.ID, &session.CreatedAt, &session.UpdatedAt)

	if err != nil {
		return fmt.Errorf("error upserting matching session: %w", err)
	}

	return nil
}

// Utility Methods

// GetActiveUsersForMatching returns users for matching without demographic filtering
func (r *MatchingRepository) GetActiveUsersForMatching(ctx context.Context, excludeUserID string) ([]*models.User, error) {
	// First try with user_matching_preferences table
	query := `
		SELECT u.id, u.name, u.display_name, u.home_latitude, u.home_longitude
		FROM users u
		JOIN user_matching_preferences ump ON u.id = ump.user_id
		WHERE u.id != $1 AND ump.is_active = true
	`

	rows, err := r.db.QueryContext(ctx, query, excludeUserID)
	if err != nil {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"GetActiveUsersForMatching: Query with preferences failed, trying fallback\",\"user_id\":\"%s\",\"error\":\"%v\"}", excludeUserID, err)

		// Fallback: get all users without preferences table
		fallbackQuery := `
			SELECT u.id, u.name, u.display_name, u.home_latitude, u.home_longitude
			FROM users u
			WHERE u.id != $1
		`

		rows, err = r.db.QueryContext(ctx, fallbackQuery, excludeUserID)
		if err != nil {
			return nil, fmt.Errorf("error getting active users for matching: %w", err)
		}
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Name, &user.DisplayName,
			&user.HomeLatitude, &user.HomeLongitude,
		)

		if err != nil {
			return nil, fmt.Errorf("error scanning user: %w", err)
		}

		users = append(users, &user)
	}

	// Ensure we always return empty slices instead of nil
	if users == nil {
		users = []*models.User{}
	}

	return users, nil
}

// GetDemographicallyCompatibleUsers returns users that match demographic preferences
func (r *MatchingRepository) GetDemographicallyCompatibleUsers(ctx context.Context, userID string) ([]*models.User, error) {
	query := `
		SELECT u.id, u.name, u.display_name, u.home_latitude, u.home_longitude,
		       ump.user_demographics, ump.demographic_preferences
		FROM users u
		JOIN user_matching_preferences ump ON u.id = ump.user_id
		WHERE u.id != $1 AND ump.is_active = true
	`

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetDemographicallyCompatibleUsers: Executing query\",\"query\":\"%s\",\"user_id\":\"%s\"}", query, userID)

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetDemographicallyCompatibleUsers: Database query failed\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		// Check if the table exists
		var tableExists bool
		checkTableQuery := `SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'user_matching_preferences')`
		if checkErr := r.db.QueryRowContext(ctx, checkTableQuery).Scan(&tableExists); checkErr == nil {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetDemographicallyCompatibleUsers: user_matching_preferences table exists: %v\",\"user_id\":\"%s\"}", tableExists, userID)
		}
		return nil, fmt.Errorf("error getting demographically compatible users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		var userDemographics models.UserDemographics
		var demographicPreferences models.DemographicPreferences

		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetDemographicallyCompatibleUsers: Scanning row\",\"user_id\":\"%s\"}", userID)

		err := rows.Scan(
			&user.ID, &user.Name, &user.DisplayName,
			&user.HomeLatitude, &user.HomeLongitude,
			&userDemographics, &demographicPreferences,
		)

		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetDemographicallyCompatibleUsers: Error scanning user row\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
			return nil, fmt.Errorf("error scanning user: %w", err)
		}

		// Check if this user is demographically compatible with the requesting user
		if isDemographicallyCompatible(userDemographics, demographicPreferences) {
			users = append(users, &user)
		}
	}

	// Ensure we always return empty slices instead of nil
	if users == nil {
		users = []*models.User{}
	}

	return users, nil
}

// isDemographicallyCompatible checks if two users are demographically compatible
func isDemographicallyCompatible(userDemographics models.UserDemographics, preferences models.DemographicPreferences) bool {
	// Check age compatibility
	ageCompatible := false
	for _, preferredAge := range preferences.AgePreferences {
		if userDemographics.AgeRange == preferredAge {
			ageCompatible = true
			break
		}
	}

	// Check gender compatibility
	genderCompatible := false
	for _, preferredGender := range preferences.GenderPreferences {
		if userDemographics.Gender == preferredGender || preferredGender == "any" {
			genderCompatible = true
			break
		}
	}

	// Check student status compatibility
	studentCompatible := checkStudentCompatibility(preferences.StudentPreference, userDemographics.StudentStatus)

	return ageCompatible && genderCompatible && studentCompatible
}

// checkStudentCompatibility checks if student status is compatible
func checkStudentCompatibility(preference, status string) bool {
	switch preference {
	case "students_only":
		return status == "undergraduate" || status == "graduate"
	case "professionals_only":
		return status == "not_student"
	case "both":
		return true
	default:
		return true
	}
}

func (r *MatchingRepository) ExpireOldMatches(ctx context.Context) error {
	query := `
		UPDATE potential_matches 
		SET status = 'expired', updated_at = CURRENT_TIMESTAMP
		WHERE expires_at < CURRENT_TIMESTAMP AND status = 'active'
	`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("error expiring old matches: %w", err)
	}

	return nil
}

func (r *MatchingRepository) ExpireOldRequests(ctx context.Context) error {
	query := `
		UPDATE match_requests 
		SET status = 'expired', updated_at = CURRENT_TIMESTAMP
		WHERE expires_at < CURRENT_TIMESTAMP AND status = 'pending'
	`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("error expiring old requests: %w", err)
	}

	return nil
}

// GetMatchingStats returns comprehensive statistics for a user's matching activity
func (r *MatchingRepository) GetMatchingStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 1. Total matches generated
	var totalMatches int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM potential_matches WHERE user1_id = $1 OR user2_id = $1
	`, userID).Scan(&totalMatches)
	if err != nil {
		log.Printf("Error getting total matches: %v", err)
		totalMatches = 0
	}
	stats["total_matches_generated"] = totalMatches

	// 2. Match acceptance rate
	var acceptedRequests, totalRequests int
	err = r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(CASE WHEN status = 'accepted' THEN 1 END) as accepted,
			COUNT(*) as total
		FROM match_requests 
		WHERE to_user_id = $1
	`, userID).Scan(&acceptedRequests, &totalRequests)
	if err != nil {
		log.Printf("Error getting acceptance rate: %v", err)
		acceptedRequests, totalRequests = 0, 0
	}

	var acceptanceRate float64
	if totalRequests > 0 {
		acceptanceRate = float64(acceptedRequests) / float64(totalRequests)
	}
	stats["match_acceptance_rate"] = math.Round(acceptanceRate*100) / 100

	// 3. Average compatibility score
	var avgCompatibility sql.NullFloat64
	err = r.db.QueryRowContext(ctx, `
		SELECT AVG(compatibility_score) 
		FROM potential_matches 
		WHERE (user1_id = $1 OR user2_id = $1) AND compatibility_score IS NOT NULL
	`, userID).Scan(&avgCompatibility)
	if err != nil {
		log.Printf("Error getting average compatibility: %v", err)
		avgCompatibility.Float64 = 0.0
		avgCompatibility.Valid = true
	}
	stats["average_compatibility_score"] = math.Round(avgCompatibility.Float64*100) / 100

	// 4. Total carpools formed (accepted matches)
	stats["total_carpools_formed"] = acceptedRequests

	// 5. Total savings (estimated)
	var totalSavings float64
	err = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(estimated_savings_per_month), 0)
		FROM potential_matches 
		WHERE (user1_id = $1 OR user2_id = $1) AND status = 'accepted'
	`, userID).Scan(&totalSavings)
	if err != nil {
		log.Printf("Error getting total savings: %v", err)
		totalSavings = 0
	}
	stats["total_savings"] = math.Round(totalSavings)

	// 6. Average route overlap
	var avgRouteOverlap sql.NullFloat64
	err = r.db.QueryRowContext(ctx, `
		SELECT AVG(route_overlap_percentage) 
		FROM potential_matches 
		WHERE (user1_id = $1 OR user2_id = $1) AND route_overlap_percentage IS NOT NULL
	`, userID).Scan(&avgRouteOverlap)
	if err != nil {
		log.Printf("Error getting average route overlap: %v", err)
		avgRouteOverlap.Float64 = 0.0
		avgRouteOverlap.Valid = true
	}
	stats["average_route_overlap"] = math.Round(avgRouteOverlap.Float64*100) / 100

	// 7. Most common match reasons (fallback for now)
	stats["most_common_match_reasons"] = []string{
		"Same destination",
		"Similar schedule",
		"Close pickup location",
		"Route overlap",
		"Flexible schedule",
	}

	// 8. Geographic distribution (fallback for now)
	stats["geographic_distribution"] = map[string]interface{}{
		"nearby":          0,
		"medium_distance": 0,
		"far":             0,
	}

	// 9. Time to acceptance (fallback for now)
	stats["time_to_acceptance"] = 0.0

	// 10. Monthly trends (last 4 months)
	monthlyTrends := []map[string]interface{}{}
	months := []string{"Oct", "Nov", "Dec", "Jan"}

	for _, month := range months {
		monthlyTrends = append(monthlyTrends, map[string]interface{}{
			"month":       month,
			"matches":     0,
			"acceptances": 0,
		})
	}
	stats["monthly_trends"] = monthlyTrends

	return stats, nil
}

// UpsertMatchScore creates or updates a match score
func (r *MatchingRepository) UpsertMatchScore(ctx context.Context, score *models.MatchScore) error {
	query := `
		INSERT INTO user_match_scores (
			user_id, potential_match_id, total_score, location_score, schedule_score,
			demographic_score, route_score, group_size_score, role_compatibility_score,
			match_reasons, dealbreakers, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, potential_match_id) DO UPDATE SET
			total_score = EXCLUDED.total_score,
			location_score = EXCLUDED.location_score,
			schedule_score = EXCLUDED.schedule_score,
			demographic_score = EXCLUDED.demographic_score,
			route_score = EXCLUDED.route_score,
			group_size_score = EXCLUDED.group_size_score,
			role_compatibility_score = EXCLUDED.role_compatibility_score,
			match_reasons = EXCLUDED.match_reasons,
			dealbreakers = EXCLUDED.dealbreakers,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.ExecContext(ctx, query,
		score.UserID, score.PotentialMatchID, score.TotalScore, score.LocationScore,
		score.ScheduleScore, score.DemographicScore, score.RouteScore, score.GroupSizeScore,
		score.RoleCompatibilityScore, score.MatchReasons, score.Dealbreakers,
	)

	if err != nil {
		return fmt.Errorf("error upserting match score: %w", err)
	}

	return nil
}

// CheckMatchRequestExists checks if a match request already exists between two users
func (r *MatchingRepository) CheckMatchRequestExists(ctx context.Context, fromUserID, toUserID, potentialMatchID string) (bool, error) {
	// Check for any pending request between these two users (regardless of potential_match_id)
	// since potential_match_id is now just a generated string, not a foreign key
	query := `
		SELECT COUNT(*) 
		FROM match_requests 
		WHERE from_user_id = $1 AND to_user_id = $2 AND status = 'pending'
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, fromUserID, toUserID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking match request existence: %w", err)
	}

	return count > 0, nil
}

// GetMatchRequestsByUserID retrieves all match requests for a user
func (r *MatchingRepository) GetMatchRequestsByUserID(ctx context.Context, userID string, companyID *uuid.UUID) (*models.MatchRequestsResponse, error) {
	// Phase 3: Add company_id filter to prevent data leaks
	// Get incoming requests (requests sent TO this user)
	incomingQuery := `
		SELECT 
			mr.id, mr.from_user_id, mr.to_user_id, mr.potential_match_id, 
			mr.message, mr.preferred_carpool_size, mr.carpool_name, mr.status, 
			mr.company_id, mr.site_id, mr.expires_at, mr.created_at, mr.updated_at,
			u.name, u.display_name, u.home_latitude, u.home_longitude
		FROM match_requests mr
		JOIN users u ON mr.from_user_id = u.id
		WHERE mr.to_user_id = $1
		  AND (mr.company_id = $2 OR ($2 IS NULL AND mr.company_id IS NULL))
		ORDER BY mr.created_at DESC
	`

	// Get outgoing requests (requests sent BY this user)
	outgoingQuery := `
		SELECT 
			mr.id, mr.from_user_id, mr.to_user_id, mr.potential_match_id, 
			mr.message, mr.preferred_carpool_size, mr.carpool_name, mr.status,
			mr.company_id, mr.site_id, mr.expires_at, mr.created_at, mr.updated_at,
			u.name, u.display_name, u.home_latitude, u.home_longitude
		FROM match_requests mr
		JOIN users u ON mr.to_user_id = u.id
		WHERE mr.from_user_id = $1
		  AND (mr.company_id = $2 OR ($2 IS NULL AND mr.company_id IS NULL))
		ORDER BY mr.created_at DESC
	`

	// Execute incoming requests query
	incomingRows, err := r.db.QueryContext(ctx, incomingQuery, userID, companyID)
	if err != nil {
		return nil, fmt.Errorf("error querying incoming match requests: %w", err)
	}
	defer incomingRows.Close()

	var incoming []models.MatchRequest
	for incomingRows.Next() {
		var request models.MatchRequest
		var fromUser models.User
		var message sql.NullString
		var companyIDVal, siteIDVal sql.NullString

		err := incomingRows.Scan(
			&request.ID, &request.FromUserID, &request.ToUserID, &request.PotentialMatchID,
			&message, &request.PreferredCarpoolSize, &request.CarpoolName, &request.Status,
			&companyIDVal, &siteIDVal, &request.ExpiresAt, &request.CreatedAt, &request.UpdatedAt,
			&fromUser.Name, &fromUser.DisplayName, &fromUser.HomeLatitude, &fromUser.HomeLongitude,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning incoming match request: %w", err)
		}

		// Handle company_id and site_id
		if companyIDVal.Valid {
			request.CompanyID = &companyIDVal.String
		}
		if siteIDVal.Valid {
			request.SiteID = &siteIDVal.String
		}

		if message.Valid {
			request.Message = &message.String
		}
		request.FromUser = &fromUser
		incoming = append(incoming, request)
	}

	// Execute outgoing requests query
	outgoingRows, err := r.db.QueryContext(ctx, outgoingQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying outgoing match requests: %w", err)
	}
	defer outgoingRows.Close()

	var outgoing []models.MatchRequest
	for outgoingRows.Next() {
		var request models.MatchRequest
		var toUser models.User
		var message sql.NullString
		var companyIDVal, siteIDVal sql.NullString

		err := outgoingRows.Scan(
			&request.ID, &request.FromUserID, &request.ToUserID, &request.PotentialMatchID,
			&message, &request.PreferredCarpoolSize, &request.CarpoolName, &request.Status,
			&companyIDVal, &siteIDVal, &request.ExpiresAt, &request.CreatedAt, &request.UpdatedAt,
			&toUser.Name, &toUser.DisplayName, &toUser.HomeLatitude, &toUser.HomeLongitude,
		)

		// Handle company_id and site_id
		if companyIDVal.Valid {
			request.CompanyID = &companyIDVal.String
		}
		if siteIDVal.Valid {
			request.SiteID = &siteIDVal.String
		}
		if err != nil {
			return nil, fmt.Errorf("error scanning outgoing match request: %w", err)
		}

		if message.Valid {
			request.Message = &message.String
		}
		request.ToUser = &toUser
		outgoing = append(outgoing, request)
	}

	return &models.MatchRequestsResponse{
		Incoming: incoming,
		Outgoing: outgoing,
	}, nil
}

// GetMatchRequestByID retrieves a single match request by ID
func (r *MatchingRepository) GetMatchRequestByID(ctx context.Context, requestID string, companyID *uuid.UUID) (*models.MatchRequest, error) {
	// Phase 3: Add company_id filter to prevent data leaks
	query := `
		SELECT 
			id, from_user_id, to_user_id, potential_match_id, 
			message, preferred_carpool_size, carpool_name, status, 
			company_id, site_id, expires_at, created_at, updated_at
		FROM match_requests
		WHERE id = $1
		  AND (company_id = $2 OR ($2 IS NULL AND company_id IS NULL))
	`

	var request models.MatchRequest
	var message sql.NullString
	var companyIDVal, siteIDVal sql.NullString

	err := r.db.QueryRowContext(ctx, query, requestID, companyID).Scan(
		&request.ID, &request.FromUserID, &request.ToUserID, &request.PotentialMatchID,
		&message, &request.PreferredCarpoolSize, &request.CarpoolName, &request.Status,
		&companyIDVal, &siteIDVal, &request.ExpiresAt, &request.CreatedAt, &request.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("match request not found: %s", requestID)
		}
		return nil, fmt.Errorf("error getting match request: %w", err)
	}

	if message.Valid {
		request.Message = &message.String
	}

	// Handle company_id and site_id
	if companyIDVal.Valid {
		request.CompanyID = &companyIDVal.String
	}
	if siteIDVal.Valid {
		request.SiteID = &siteIDVal.String
	}

	return &request, nil
}

// FindCandidatesByPreferences finds potential matches based on user's saved preferences
// This method implements strict preference-driven filtering instead of broad user lists
func (r *MatchingRepository) FindCandidatesByPreferences(ctx context.Context, userID string) ([]*models.User, error) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindCandidatesByPreferences: Starting for user\",\"user_id\":\"%s\"}", userID)

	// First, get the current user's preferences to use as filter criteria
	// Phase 3: Default to personal scope (companyID = nil)
	var companyID *uuid.UUID = nil
	userPrefs, err := r.GetUserMatchingPreferences(ctx, userID, companyID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindCandidatesByPreferences: Error getting user preferences\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return nil, fmt.Errorf("error getting user preferences: %w", err)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindCandidatesByPreferences: Got user preferences\",\"user_id\":\"%s\",\"destination_lat\":\"%v\",\"destination_lng\":\"%v\"}",
		userID, userPrefs.DestinationLatitude, userPrefs.DestinationLongitude)

	// Check if destination is set (required for matching)
	if userPrefs.DestinationLatitude == nil || userPrefs.DestinationLongitude == nil ||
		*userPrefs.DestinationLatitude == 0.0 || *userPrefs.DestinationLongitude == 0.0 {
		return nil, fmt.Errorf("destination not set in preferences - cannot find matches")
	}

	// Get current user's home location for proximity filtering
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindCandidatesByPreferences: Getting current user home location\",\"user_id\":\"%s\"}", userID)
	var currentUserHomeLat, currentUserHomeLng float64
	err = r.db.QueryRowContext(ctx, "SELECT home_latitude, home_longitude FROM users WHERE id = $1", userID).Scan(&currentUserHomeLat, &currentUserHomeLng)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindCandidatesByPreferences: Error getting current user home location\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return nil, fmt.Errorf("error getting current user home location: %w", err)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindCandidatesByPreferences: Got current user home location\",\"user_id\":\"%s\",\"home_lat\":\"%.6f\",\"home_lng\":\"%.6f\"}",
		userID, currentUserHomeLat, currentUserHomeLng)

	// Build a simpler query first - let's start with basic filtering
	query := `
		SELECT DISTINCT
            u.id, u.name, u.display_name, u.home_latitude, u.home_longitude, u.clerk_id,
			ump.user_demographics, ump.demographic_preferences,
			ump.arrival_time, ump.commute_days
		FROM users u
		JOIN user_matching_preferences ump ON u.id = ump.user_id
		WHERE u.id != $1 
		AND ump.is_active = true
		AND u.home_latitude IS NOT NULL 
		AND u.home_longitude IS NOT NULL
		AND ump.destination_latitude IS NOT NULL 
		AND ump.destination_longitude IS NOT NULL
		-- Simple distance check: use a basic bounding box first
		AND u.home_latitude BETWEEN $2 - 0.1 AND $2 + 0.1
		AND u.home_longitude BETWEEN $3 - 0.1 AND $3 + 0.1
		AND ump.destination_latitude BETWEEN $4 - 0.1 AND $4 + 0.1
		AND ump.destination_longitude BETWEEN $5 - 0.1 AND $5 + 0.1
	`

	args := []interface{}{
		userID,
		currentUserHomeLat,              // $2: current user's home latitude
		currentUserHomeLng,              // $3: current user's home longitude
		*userPrefs.DestinationLatitude,  // $4: current user's destination latitude
		*userPrefs.DestinationLongitude, // $5: current user's destination longitude
	}

	// Add optional schedule filters if user has them set
	argIndex := 6
	if userPrefs.ArrivalTime != nil && *userPrefs.ArrivalTime != "" {
		// Use fixed 30-minute flexibility for simplified UI
		query += fmt.Sprintf(`
		AND (
			ump.arrival_time IS NULL 
			OR ABS(EXTRACT(EPOCH FROM (ump.arrival_time - $%d::time))/60) <= 30
		)`, argIndex)
		args = append(args, *userPrefs.ArrivalTime)
		argIndex++
	}

	if len(userPrefs.CommuteDays) > 0 {
		// Build a safe ARRAY[...] literal from validated day values to avoid driver array binding issues
		validDays := map[string]bool{"mon": true, "tue": true, "wed": true, "thu": true, "fri": true, "sat": true, "sun": true}
		var dayLiterals []string
		for _, d := range userPrefs.CommuteDays {
			dl := strings.ToLower(strings.TrimSpace(d))
			if validDays[dl] {
				dayLiterals = append(dayLiterals, fmt.Sprintf("'%s'", dl))
			}
		}
		if len(dayLiterals) > 0 {
			query += "\n\t\tAND (\n\t\t\tump.commute_days IS NULL \n\t\t\tOR ump.commute_days && ARRAY[" + strings.Join(dayLiterals, ",") + "]::text[]\n\t\t)\n"
		}
	}

	// Add demographic compatibility filter
	query += `
		AND (
			-- Check if demographics are compatible
			-- This is a simplified check - we'll do detailed compatibility in Go
			ump.user_demographics IS NOT NULL 
			AND ump.demographic_preferences IS NOT NULL
		)
	`

	// Add ordering and limit - simplified
	query += `
        ORDER BY u.home_latitude, u.home_longitude
        LIMIT 50
    `

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindCandidatesByPreferences: Executing preference-driven query\",\"user_id\":\"%s\",\"max_distance\":\"%.2f\",\"query\":\"%s\",\"args_count\":%d}",
		userID, userPrefs.MaxPickupDistanceMiles, query, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindCandidatesByPreferences: Database query failed\",\"user_id\":\"%s\",\"error\":\"%v\",\"query\":\"%s\",\"args\":\"%v\"}", userID, err, query, args)
		return nil, fmt.Errorf("error finding candidates by preferences: %w", err)
	}
	defer rows.Close()

	var candidates []*models.User
	for rows.Next() {
		var user models.User
		var userDemographics models.UserDemographics
		var demographicPreferences models.DemographicPreferences
		var arrivalTime sql.NullString
		var commuteDaysRaw sql.NullString

		err := rows.Scan(
			&user.ID, &user.Name, &user.DisplayName,
			&user.HomeLatitude, &user.HomeLongitude, &user.ClerkID,
			&userDemographics, &demographicPreferences,
			&arrivalTime, &commuteDaysRaw,
		)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindCandidatesByPreferences: Error scanning candidate\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
			continue
		}

		// Parse commute_days if present (Postgres returns arrays like "{mon,wed,fri}")
		if commuteDaysRaw.Valid {
			s := strings.Trim(commuteDaysRaw.String, "{}")
			if s != "" {
				_ = strings.Split(s, ",") // parsed but not used yet; keep for future extensions
			}
		}

		// Detailed demographic compatibility check
		if isDemographicallyCompatible(userDemographics, demographicPreferences) {
			candidates = append(candidates, &user)
		}
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"FindCandidatesByPreferences: Found %d candidates\",\"user_id\":\"%s\"}",
		len(candidates), userID)

	return candidates, nil
}
