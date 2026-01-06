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

func (r *MatchingRepository) GetUserMatchingPreferences(ctx context.Context, userID string) (*models.UserMatchingPreferences, error) {
	// CRITICAL SECURITY LOGGING: Track the user ID being queried
	log.Printf("{\"severity\":\"SECURITY\",\"message\":\"GetUserMatchingPreferences: Starting database query\",\"user_id\":\"%s\"}", userID)

	query := `
		SELECT id, user_id, max_detour_minutes, preferred_group_size, driver_preference,
			schedule_flexibility_minutes, max_pickup_distance_miles, min_compatibility_score,
			notification_preferences, user_demographics, demographic_preferences, 
			destination_latitude, destination_longitude, arrival_time, commute_days,
			is_active, created_at, updated_at
		FROM user_matching_preferences 
		WHERE user_id = $1
	`

	// CRITICAL SECURITY LOGGING: Log the exact query and parameters
	log.Printf("{\"severity\":\"SECURITY\",\"message\":\"GetUserMatchingPreferences: Executing query\",\"query\":\"%s\",\"user_id\":\"%s\"}", query, userID)

	var prefs models.UserMatchingPreferences
	var destinationLatitude, destinationLongitude sql.NullFloat64
	var arrivalTime sql.NullString
	var commuteDays sql.NullString

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&prefs.ID, &prefs.UserID, &prefs.MaxDetourMinutes, &prefs.PreferredGroupSize, &prefs.DriverPreference,
		&prefs.ScheduleFlexibilityMinutes, &prefs.MaxPickupDistanceMiles, &prefs.MinCompatibilityScore,
		&prefs.NotificationPreferences, &prefs.UserDemographics, &prefs.DemographicPreferences,
		&destinationLatitude, &destinationLongitude, &arrivalTime, &commuteDays,
		&prefs.IsActive, &prefs.CreatedAt, &prefs.UpdatedAt,
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

	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetUserMatchingPreferences: Database scan error\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)

		// If the new columns don't exist, try the old query
		// Check if error is due to missing columns (backward compatibility for old migrations)
		if strings.Contains(err.Error(), "destination_latitude") || strings.Contains(err.Error(), "destination_longitude") ||
			strings.Contains(err.Error(), "arrival_time") || strings.Contains(err.Error(), "commute_days") ||
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

			// Set default values for new fields (0.0 for coordinates, null for schedule)
			prefs.DestinationLatitude = func() *float64 { v := 0.0; return &v }()
			prefs.DestinationLongitude = func() *float64 { v := 0.0; return &v }()
			prefs.ArrivalTime = nil
			prefs.CommuteDays = nil
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

func (r *MatchingRepository) UpsertUserMatchingPreferences(ctx context.Context, prefs *models.UserMatchingPreferences) error {
	// 🚨 DEPLOYMENT CHECK: This log confirms the latest code is deployed
	log.Printf("🚀🚀🚀 UpsertUserMatchingPreferences: NEW CODE VERSION - Using SELECT-then-UPDATE/INSERT approach 🚀🚀🚀")

	// Migration was run - id is PRIMARY KEY, user_id is not unique
	// Use SELECT-then-UPDATE/INSERT approach to handle the unique index
	// NO ON CONFLICT CLAUSES - This function does NOT use ON CONFLICT

	// Check if record exists
	// Handle case where id column might not exist yet
	var existingID uuid.UUID
	var checkErr error

	// First try with id column
	checkQuery := `
		SELECT id FROM user_matching_preferences 
		WHERE user_id = $1
	`
	log.Printf("🔍 Checking if preferences exist for user_id: %s", prefs.UserID)
	checkErr = r.db.QueryRowContext(ctx, checkQuery, prefs.UserID).Scan(&existingID)

	// If that fails because id doesn't exist, try without it
	if checkErr != nil && (strings.Contains(checkErr.Error(), "column") && strings.Contains(checkErr.Error(), "does not exist")) {
		checkQueryNoCompany := `
			SELECT user_id FROM user_matching_preferences 
			WHERE user_id = $1
		`
		var dummyUUID uuid.UUID
		checkErr = r.db.QueryRowContext(ctx, checkQueryNoCompany, prefs.UserID).Scan(&dummyUUID)
		if checkErr == nil {
			// Record exists but we don't have id column - use user_id for update
			updateQueryNoID := `
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
				WHERE user_id = $1
			`
			_, err := r.db.ExecContext(ctx, updateQueryNoID,
				prefs.UserID,
				prefs.MaxDetourMinutes, prefs.PreferredGroupSize, prefs.DriverPreference,
				prefs.ScheduleFlexibilityMinutes, prefs.MaxPickupDistanceMiles, prefs.MinCompatibilityScore,
				prefs.NotificationPreferences, prefs.UserDemographics, prefs.DemographicPreferences,
				prefs.DestinationLatitude, prefs.DestinationLongitude, prefs.ArrivalTime, prefs.CommuteDays,
				prefs.IsActive,
			)
			if err != nil {
				return fmt.Errorf("error updating user matching preferences: %w", err)
			}
			return nil
		}
	}

	if checkErr == nil {
		// Record exists, update it using id
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
		_, err := r.db.ExecContext(ctx, updateQuery,
			existingID,
			prefs.MaxDetourMinutes, prefs.PreferredGroupSize, prefs.DriverPreference,
			prefs.ScheduleFlexibilityMinutes, prefs.MaxPickupDistanceMiles, prefs.MinCompatibilityScore,
			prefs.NotificationPreferences, prefs.UserDemographics, prefs.DemographicPreferences,
			prefs.DestinationLatitude, prefs.DestinationLongitude, prefs.ArrivalTime, prefs.CommuteDays,
			prefs.IsActive,
		)
		if err != nil {
			return fmt.Errorf("error updating user matching preferences: %w", err)
		}
		return nil
	} else if checkErr == sql.ErrNoRows {
		// Record doesn't exist, insert it
		log.Printf("➕ Record doesn't exist, inserting new preferences")
		// Try with id column first
		insertQuery := `
			INSERT INTO user_matching_preferences (
				user_id, max_detour_minutes, preferred_group_size, 
				driver_preference, schedule_flexibility_minutes, max_pickup_distance_miles, 
				min_compatibility_score, notification_preferences, user_demographics, 
				demographic_preferences, destination_latitude, destination_longitude, 
				arrival_time, commute_days, is_active, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`
		log.Printf("🔧 Executing INSERT query (NO ON CONFLICT - this is a plain INSERT)")
		_, err := r.db.ExecContext(ctx, insertQuery,
			prefs.UserID,
			prefs.MaxDetourMinutes, prefs.PreferredGroupSize, prefs.DriverPreference,
			prefs.ScheduleFlexibilityMinutes, prefs.MaxPickupDistanceMiles, prefs.MinCompatibilityScore,
			prefs.NotificationPreferences, prefs.UserDemographics, prefs.DemographicPreferences,
			prefs.DestinationLatitude, prefs.DestinationLongitude, prefs.ArrivalTime, prefs.CommuteDays,
			prefs.IsActive,
		)

		// If INSERT fails due to unique constraint violation, record was created between SELECT and INSERT
		// Try to update it instead
		if err != nil {
			log.Printf("⚠️ INSERT failed with error: %v", err)
			if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
				log.Printf("🔄 Unique constraint violation detected - record was created between SELECT and INSERT, trying UPDATE instead")
				// Race condition - record was inserted between our SELECT and INSERT
				// Try to get the id and update
				var raceID uuid.UUID
				raceCheckQuery := `
					SELECT id FROM user_matching_preferences 
					WHERE user_id = $1
				`
				if raceErr := r.db.QueryRowContext(ctx, raceCheckQuery, prefs.UserID).Scan(&raceID); raceErr == nil {
					log.Printf("🔄 Found record with id: %s, updating instead", raceID.String())
					// Now update it
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
						raceID,
						prefs.MaxDetourMinutes, prefs.PreferredGroupSize, prefs.DriverPreference,
						prefs.ScheduleFlexibilityMinutes, prefs.MaxPickupDistanceMiles, prefs.MinCompatibilityScore,
						prefs.NotificationPreferences, prefs.UserDemographics, prefs.DemographicPreferences,
						prefs.DestinationLatitude, prefs.DestinationLongitude, prefs.ArrivalTime, prefs.CommuteDays,
						prefs.IsActive,
					)
					if err == nil {
						log.Printf("✅✅✅ Successfully updated preferences after race condition ✅✅✅")
						return nil
					}
				}
			}
			// Check if error mentions ON CONFLICT - this should NEVER happen with our code
			if strings.Contains(err.Error(), "ON CONFLICT") || strings.Contains(err.Error(), "conflict") {
				log.Printf("🚨🚨🚨 CRITICAL: Error mentions ON CONFLICT but our code doesn't use it! Error: %v 🚨🚨🚨", err)
			}
			log.Printf("❌❌❌ FINAL ERROR in INSERT/UPDATE: %v ❌❌❌", err)
			return fmt.Errorf("error inserting user matching preferences: %w", err)
		}
		log.Printf("✅✅✅ Successfully inserted new preferences ✅✅✅")
		return nil
	}

	// If we get here, there was an error checking
	log.Printf("❌❌❌ ERROR checking if preferences exist: %v ❌❌❌", checkErr)
	return fmt.Errorf("error checking if preferences exist: %w", checkErr)
}

// Potential Matches Methods

func (r *MatchingRepository) GetPotentialMatches(ctx context.Context, userID string, status string) ([]*models.PotentialMatch, error) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetPotentialMatches: Starting query\",\"user_id\":\"%s\",\"status\":\"%s\"}", userID, status)

	// Simplified query: get the other user directly using CASE
	// Use DISTINCT to prevent duplicate matches if there are any edge cases
	// Note: We use DISTINCT without ON since we want to order by compatibility_score
	query := `
		SELECT DISTINCT
		       pm.id, pm.user1_id, pm.user2_id, pm.compatibility_score, pm.route_overlap_percentage,
		       pm.total_distance_miles, pm.estimated_savings_per_month, pm.match_reasons, pm.status,
		       pm.expires_at, pm.created_at, pm.updated_at,
		       CASE 
		         WHEN pm.user1_id::text = $1 THEN pm.user2_id
		         ELSE pm.user1_id
		       END as other_user_id,
		       u.id, u.name, u.display_name, u.home_latitude, u.home_longitude, u.clerk_id
		FROM potential_matches pm
		JOIN users u ON (
			CASE 
				WHEN pm.user1_id::text = $1 THEN pm.user2_id = u.id
				ELSE pm.user1_id = u.id
			END
		)
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
		var otherUserID string
		var otherUser models.User

		err := rows.Scan(
			&match.ID, &match.User1ID, &match.User2ID, &match.CompatibilityScore, &match.RouteOverlapPercentage,
			&match.TotalDistanceMiles, &match.EstimatedSavingsPerMonth, &match.MatchReasons, &match.Status,
			&match.ExpiresAt, &match.CreatedAt, &match.UpdatedAt,
			&otherUserID, // This is the other user's ID (not the requesting user)
			&otherUser.ID, &otherUser.Name, &otherUser.DisplayName, &otherUser.HomeLatitude, &otherUser.HomeLongitude, &otherUser.ClerkID,
		)

		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetPotentialMatches: Error scanning match\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
			return nil, fmt.Errorf("error scanning potential match: %w", err)
		}

		// Set the other user as User2
		match.User2 = &otherUser

		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetPotentialMatches: Found match\",\"match_id\":\"%s\",\"user1_id\":\"%s\",\"user2_id\":\"%s\",\"other_user_id\":\"%s\",\"compatibility_score\":%.2f}",
			match.ID, match.User1ID, match.User2ID, otherUserID, match.CompatibilityScore)

		matches = append(matches, &match)
	}

	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetPotentialMatches: Found %d matches\",\"user_id\":\"%s\",\"status\":\"%s\"}", len(matches), userID, status)

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

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"CreatePotentialMatch: Executing query\",\"user1_id\":\"%s\",\"user2_id\":\"%s\",\"status\":\"%s\",\"compatibility_score\":%.2f}",
		match.User1ID, match.User2ID, match.Status, match.CompatibilityScore)
	_, err := r.db.ExecContext(ctx, query,
		match.User1ID, match.User2ID, match.CompatibilityScore, match.RouteOverlapPercentage,
		match.TotalDistanceMiles, match.EstimatedSavingsPerMonth, match.MatchReasons, match.Status, match.ExpiresAt,
	)

	if err != nil {
		// Enhanced error logging to help diagnose the issue
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreatePotentialMatch: Database error\",\"user1_id\":\"%s\",\"user2_id\":\"%s\",\"status\":\"%s\",\"error\":\"%v\",\"error_type\":\"%T\"}",
			match.User1ID, match.User2ID, match.Status, err, err)

		// Check if it's a constraint violation (status not allowed)
		if strings.Contains(err.Error(), "check constraint") || strings.Contains(err.Error(), "status") {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"CreatePotentialMatch: Status constraint violation - status '%s' may not be allowed. Run migration 031 to add 'pending' status.\",\"user1_id\":\"%s\",\"user2_id\":\"%s\"}",
				match.Status, match.User1ID, match.User2ID)
		}

		return fmt.Errorf("error creating potential match: %w", err)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"CreatePotentialMatch: Successfully created/updated potential match\",\"user1_id\":\"%s\",\"user2_id\":\"%s\"}", match.User1ID, match.User2ID)
	return nil
}

// Match Requests Methods

func (r *MatchingRepository) GetMatchRequests(ctx context.Context, userID string) ([]*models.MatchRequest, []*models.MatchRequest, error) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetMatchRequests: Starting query\",\"user_id\":\"%s\"}", userID)

	// Get incoming requests (requests sent TO this user)
	incomingQuery := `
		SELECT mr.id, mr.from_user_id, mr.to_user_id, mr.potential_match_id, 
		       mr.message, mr.preferred_carpool_size, mr.carpool_name, mr.status, 
		       mr.created_at, mr.updated_at, mr.expires_at,
		       u.name, u.display_name, u.home_latitude, u.home_longitude
		FROM match_requests mr
		JOIN users u ON mr.from_user_id = u.id
		WHERE mr.to_user_id = $1 
		  AND mr.status != 'expired'
		ORDER BY mr.created_at DESC
	`

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetMatchRequests: Executing incoming query\",\"query\":\"%s\",\"user_id\":\"%s\"}", incomingQuery, userID)

	incomingRows, err := r.db.QueryContext(ctx, incomingQuery, userID)
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

		err := incomingRows.Scan(
			&request.ID, &request.FromUserID, &request.ToUserID, &request.PotentialMatchID,
			&request.Message, &request.PreferredCarpoolSize, &request.CarpoolName, &request.Status,
			&request.CreatedAt, &request.UpdatedAt, &request.ExpiresAt,
			&userName, &userDisplayName, &userHomeLat, &userHomeLng,
		)
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
		       mr.created_at, mr.updated_at, mr.expires_at,
		       u.name, u.display_name, u.home_latitude, u.home_longitude
		FROM match_requests mr
		JOIN users u ON mr.to_user_id = u.id
		WHERE mr.from_user_id = $1 
		  AND mr.status != 'expired'
		ORDER BY mr.created_at DESC
	`

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetMatchRequests: Executing outgoing query\",\"query\":\"%s\",\"user_id\":\"%s\"}", outgoingQuery, userID)

	outgoingRows, err := r.db.QueryContext(ctx, outgoingQuery, userID)
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

		err := outgoingRows.Scan(
			&request.ID, &request.FromUserID, &request.ToUserID, &request.PotentialMatchID,
			&request.Message, &request.PreferredCarpoolSize, &request.CarpoolName, &request.Status,
			&request.CreatedAt, &request.UpdatedAt, &request.ExpiresAt,
			&userName, &userDisplayName, &userHomeLat, &userHomeLng,
		)
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
	query := `
		INSERT INTO match_requests (
			from_user_id, to_user_id, potential_match_id, message, preferred_carpool_size, 
			carpool_name, status, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	// Handle potential_match_id - it might be empty or null
	var potentialMatchID interface{}
	if request.PotentialMatchID != "" {
		potentialMatchID = request.PotentialMatchID
	} else {
		potentialMatchID = nil
	}

	err := r.db.QueryRowContext(ctx, query,
		request.FromUserID, request.ToUserID, potentialMatchID,
		request.Message, request.PreferredCarpoolSize, request.CarpoolName, request.Status, request.ExpiresAt,
	).Scan(&request.ID, &request.CreatedAt, &request.UpdatedAt)

	if err != nil {
		// Check if it's a duplicate key violation (PostgreSQL error code 23505)
		if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("duplicate_request: a match request already exists between these users with this potential match")
		}
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
	// Check for any request with the same (from_user_id, to_user_id, potential_match_id)
	// This matches the unique constraint: match_requests_from_user_id_to_user_id_potential_match_id_key
	var query string
	var args []interface{}

	if potentialMatchID != "" {
		// Check for exact match including potential_match_id
		query = `
			SELECT COUNT(*) 
			FROM match_requests 
			WHERE from_user_id = $1 AND to_user_id = $2 AND potential_match_id = $3
		`
		args = []interface{}{fromUserID, toUserID, potentialMatchID}
	} else {
		// If potential_match_id is empty, check for any pending request between these users
		query = `
			SELECT COUNT(*) 
			FROM match_requests 
			WHERE from_user_id = $1 AND to_user_id = $2 AND status = 'pending'
		`
		args = []interface{}{fromUserID, toUserID}
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking match request existence: %w", err)
	}

	return count > 0, nil
}

// CheckAnyMatchRequestExists checks if ANY match request exists between two users in either direction
// This is used to filter out potential matches where a request has already been sent
func (r *MatchingRepository) CheckAnyMatchRequestExists(ctx context.Context, user1ID, user2ID string) (bool, error) {
	// Check for any request between these two users in either direction (excluding expired)
	query := `
		SELECT COUNT(*) 
		FROM match_requests 
		WHERE (
			(from_user_id = $1 AND to_user_id = $2) OR 
			(from_user_id = $2 AND to_user_id = $1)
		) AND status != 'expired'
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, user1ID, user2ID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking match request existence: %w", err)
	}

	return count > 0, nil
}

// CheckActiveCarpoolWithSameDestination checks if two users have an active carpool together
// and if that carpool's destination matches the given destination coordinates (within 1 mile)
func (r *MatchingRepository) CheckActiveCarpoolWithSameDestination(ctx context.Context, user1ID, user2ID string, destLat, destLng *float64) (bool, error) {
	// If no destination provided, can't check - return false (no carpool blocking)
	if destLat == nil || destLng == nil || *destLat == 0.0 || *destLng == 0.0 {
		return false, nil
	}

	// Check if they have an active carpool together
	// Get carpools where both users are members and the carpool is active
	query := `
		SELECT DISTINCT c.id, c.destination_address
		FROM carpools c
		INNER JOIN carpool_members cm1 ON c.id = cm1.carpool_id
		INNER JOIN carpool_members cm2 ON c.id = cm2.carpool_id
		WHERE cm1.user_id = $1 
		  AND cm2.user_id = $2
		  AND c.status = true
		LIMIT 1
	`

	var carpoolID string
	var destinationAddress sql.NullString
	err := r.db.QueryRowContext(ctx, query, user1ID, user2ID).Scan(&carpoolID, &destinationAddress)

	if err == sql.ErrNoRows {
		// No active carpool together
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error checking active carpool: %w", err)
	}

	// If carpool exists, check if destination matches
	// Get both users' current destination preferences to compare with the potential match destination
	// The potential match destination (destLat, destLng) is what the requesting user is currently looking for
	user1Prefs, err := r.GetUserMatchingPreferences(ctx, user1ID)
	if err != nil {
		log.Printf("{\"severity\":\"WARN\",\"message\":\"CheckActiveCarpoolWithSameDestination: Error getting user1 preferences\",\"user_id\":\"%s\",\"error\":\"%v\"}", user1ID, err)
		// If we can't get preferences, assume destination doesn't match (don't block)
		return false, nil
	}

	user2Prefs, err := r.GetUserMatchingPreferences(ctx, user2ID)
	if err != nil {
		log.Printf("{\"severity\":\"WARN\",\"message\":\"CheckActiveCarpoolWithSameDestination: Error getting user2 preferences\",\"user_id\":\"%s\",\"error\":\"%v\"}", user2ID, err)
		// If we can't get preferences, assume destination doesn't match (don't block)
		return false, nil
	}

	// Check if the potential match destination (what requesting user is looking for) matches
	// BOTH users' current destinations. If they have a carpool together and BOTH are looking
	// for the same destination, then filter them out (they already have a carpool for that destination).
	// If only one matches, they might be looking for different destinations, so allow the match.
	const destinationThreshold = 0.015 // ~1 mile in degrees

	checkDestinationMatch := func(userDestLat, userDestLng *float64) bool {
		if userDestLat == nil || userDestLng == nil || *userDestLat == 0.0 || *userDestLng == 0.0 {
			return false
		}
		// Calculate distance using simple lat/lng difference (approximation)
		latDiff := math.Abs(*destLat - *userDestLat)
		lngDiff := math.Abs(*destLng - *userDestLng)
		// Combined distance (rough approximation)
		distance := math.Sqrt(latDiff*latDiff + lngDiff*lngDiff)
		return distance <= destinationThreshold
	}

	// Check if potential match destination matches BOTH users' current destinations
	// This ensures we only filter if they're both looking for the same destination they already have a carpool for
	user1Match := checkDestinationMatch(user1Prefs.DestinationLatitude, user1Prefs.DestinationLongitude)
	user2Match := checkDestinationMatch(user2Prefs.DestinationLatitude, user2Prefs.DestinationLongitude)

	// Only filter if BOTH users' destinations match the potential match destination
	// This means they both want to go to the same place they already have a carpool for
	if user1Match && user2Match {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"CheckActiveCarpoolWithSameDestination: Found active carpool with same destination (both users match)\",\"user1_id\":\"%s\",\"user2_id\":\"%s\",\"carpool_id\":\"%s\"}",
			user1ID, user2ID, carpoolID)
		return true, nil
	}

	// They have a carpool but it's for a different destination, or only one user matches
	// (meaning they might be looking for different destinations) - allow the match
	return false, nil
}

// GetMatchRequestsByUserID retrieves all match requests for a user
func (r *MatchingRepository) GetMatchRequestsByUserID(ctx context.Context, userID string) (*models.MatchRequestsResponse, error) {
	// Get incoming requests (requests sent TO this user)
	incomingQuery := `
		SELECT 
			mr.id, mr.from_user_id, mr.to_user_id, mr.potential_match_id, 
			mr.message, mr.preferred_carpool_size, mr.carpool_name, mr.status, 
			mr.expires_at, mr.created_at, mr.updated_at,
			u.name, u.display_name, u.home_latitude, u.home_longitude
		FROM match_requests mr
		JOIN users u ON mr.from_user_id = u.id
		WHERE mr.to_user_id = $1
		ORDER BY mr.created_at DESC
	`

	// Get outgoing requests (requests sent BY this user)
	outgoingQuery := `
		SELECT 
			mr.id, mr.from_user_id, mr.to_user_id, mr.potential_match_id, 
			mr.message, mr.preferred_carpool_size, mr.carpool_name, mr.status,
			mr.expires_at, mr.created_at, mr.updated_at,
			u.name, u.display_name, u.home_latitude, u.home_longitude
		FROM match_requests mr
		JOIN users u ON mr.to_user_id = u.id
		WHERE mr.from_user_id = $1
		ORDER BY mr.created_at DESC
	`

	// Execute incoming requests query
	incomingRows, err := r.db.QueryContext(ctx, incomingQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying incoming match requests: %w", err)
	}
	defer incomingRows.Close()

	var incoming []models.MatchRequest
	for incomingRows.Next() {
		var request models.MatchRequest
		var fromUser models.User
		var message sql.NullString

		err := incomingRows.Scan(
			&request.ID, &request.FromUserID, &request.ToUserID, &request.PotentialMatchID,
			&message, &request.PreferredCarpoolSize, &request.CarpoolName, &request.Status,
			&request.ExpiresAt, &request.CreatedAt, &request.UpdatedAt,
			&fromUser.Name, &fromUser.DisplayName, &fromUser.HomeLatitude, &fromUser.HomeLongitude,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning incoming match request: %w", err)
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

		err := outgoingRows.Scan(
			&request.ID, &request.FromUserID, &request.ToUserID, &request.PotentialMatchID,
			&message, &request.PreferredCarpoolSize, &request.CarpoolName, &request.Status,
			&request.ExpiresAt, &request.CreatedAt, &request.UpdatedAt,
			&toUser.Name, &toUser.DisplayName, &toUser.HomeLatitude, &toUser.HomeLongitude,
		)
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
func (r *MatchingRepository) GetMatchRequestByID(ctx context.Context, requestID string) (*models.MatchRequest, error) {
	query := `
		SELECT 
			id, from_user_id, to_user_id, potential_match_id, 
			message, preferred_carpool_size, carpool_name, status, 
			expires_at, created_at, updated_at
		FROM match_requests
		WHERE id = $1
	`

	var request models.MatchRequest
	var message sql.NullString

	err := r.db.QueryRowContext(ctx, query, requestID).Scan(
		&request.ID, &request.FromUserID, &request.ToUserID, &request.PotentialMatchID,
		&message, &request.PreferredCarpoolSize, &request.CarpoolName, &request.Status,
		&request.ExpiresAt, &request.CreatedAt, &request.UpdatedAt,
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

	return &request, nil
}

// FindCandidatesByPreferences finds potential matches based on user's saved preferences.
//
// MATCHING LOGIC (Purposeful, not random):
// When users match and create a carpool, their preferences are merged:
//   - Commute days: Intersection (User A: Mon/Wed/Fri + User B: Mon/Wed = Carpool: Mon/Wed only)
//   - Arrival time: Earlier time wins (both users can arrive on time)
//   - MaxDetourMinutes: More restrictive value (conservative approach)
//   - MaxPickupDistance: More restrictive value (conservative approach)
//
// Therefore, matching requires:
//  1. Home proximity: Within MaxPickupDistanceMiles (for pickup feasibility)
//  2. Destination compatibility:
//     - Same street/address (within 1.0 miles): Always match (e.g., "1889 Mowry" vs "1900 Mowry Avenue")
//     - Different destinations: Within MaxDetourMinutes tolerance (1 min ≈ 0.5 miles at 30mph, min 2.0 miles)
//  3. Schedule compatibility:
//     - Commute days: At least 1 day overlap (required for carpool to have rides)
//     - Arrival time: Within ScheduleFlexibilityMinutes + 15min buffer (for time merging)
//  4. Driver preference: Compatible (driver + passenger, or flexible)
//  5. Demographics: Match preferences (if set)
//
// This ensures matches can actually form functional carpools with merged preferences.
func (r *MatchingRepository) FindCandidatesByPreferences(ctx context.Context, userID string) ([]*models.User, error) {
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"🔍 MATCHING: FindCandidatesByPreferences: Starting for user\",\"user_id\":\"%s\"}", userID)

	// First, get the current user's preferences to use as filter criteria
	userPrefs, err := r.GetUserMatchingPreferences(ctx, userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindCandidatesByPreferences: Error getting user preferences\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return nil, fmt.Errorf("error getting user preferences: %w", err)
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"🔍 MATCHING: FindCandidatesByPreferences: Got user preferences\",\"user_id\":\"%s\",\"destination_lat\":\"%v\",\"destination_lng\":\"%v\"}",
		userID, userPrefs.DestinationLatitude, userPrefs.DestinationLongitude)

	// Check if destination is set (required for matching)
	// Note: We check for nil OR exactly 0.0 (which is the default value for new users)
	if userPrefs.DestinationLatitude == nil || userPrefs.DestinationLongitude == nil {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"FindCandidatesByPreferences: Destination coordinates are nil\",\"user_id\":\"%s\"}", userID)
		return nil, fmt.Errorf("destination not set in preferences - cannot find matches")
	}

	// Check if destination is set to 0.0 (default value means not set)
	// Note: We check if BOTH are 0.0, not just one (since 0.0,0.0 is the default)
	if *userPrefs.DestinationLatitude == 0.0 && *userPrefs.DestinationLongitude == 0.0 {
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"FindCandidatesByPreferences: Destination is set to default (0.0, 0.0) - user needs to set a destination\",\"user_id\":\"%s\",\"dest_lat\":%.6f,\"dest_lng\":%.6f}",
			userID, *userPrefs.DestinationLatitude, *userPrefs.DestinationLongitude)
		return nil, fmt.Errorf("destination not set in preferences (still at default 0.0) - cannot find matches")
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindCandidatesByPreferences: Destination is set\",\"user_id\":\"%s\",\"dest_lat\":%.6f,\"dest_lng\":%.6f}",
		userID, *userPrefs.DestinationLatitude, *userPrefs.DestinationLongitude)

	// Get current user's home location for proximity filtering
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindCandidatesByPreferences: Getting current user home location\",\"user_id\":\"%s\"}", userID)
	var currentUserHomeLat, currentUserHomeLng sql.NullFloat64
	err = r.db.QueryRowContext(ctx, "SELECT home_latitude, home_longitude FROM users WHERE id = $1", userID).Scan(&currentUserHomeLat, &currentUserHomeLng)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindCandidatesByPreferences: Error getting current user home location\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return nil, fmt.Errorf("error getting current user home location: %w", err)
	}

	// Check if home location is set
	if !currentUserHomeLat.Valid || !currentUserHomeLng.Valid ||
		currentUserHomeLat.Float64 == 0.0 || currentUserHomeLng.Float64 == 0.0 {
		var homeLatVal, homeLngVal interface{}
		if currentUserHomeLat.Valid {
			homeLatVal = currentUserHomeLat.Float64
		}
		if currentUserHomeLng.Valid {
			homeLngVal = currentUserHomeLng.Float64
		}
		log.Printf("{\"severity\":\"WARNING\",\"message\":\"FindCandidatesByPreferences: User home location not set\",\"user_id\":\"%s\",\"home_lat_valid\":%v,\"home_lng_valid\":%v,\"home_lat\":%v,\"home_lng\":%v}",
			userID, currentUserHomeLat.Valid, currentUserHomeLng.Valid, homeLatVal, homeLngVal)
		return nil, fmt.Errorf("user home location not set - cannot find matches")
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindCandidatesByPreferences: Got current user home location\",\"user_id\":\"%s\",\"home_lat\":\"%.6f\",\"home_lng\":\"%.6f\"}",
		userID, currentUserHomeLat.Float64, currentUserHomeLng.Float64)

	// Convert MaxPickupDistanceMiles to degrees for bounding box
	// 1 degree latitude ≈ 69 miles, 1 degree longitude ≈ 69 * cos(latitude) miles
	// Use a conservative conversion: 1 mile ≈ 0.0145 degrees (at mid-latitudes)
	milesToDegrees := 0.0145 // Approximate conversion factor
	maxDistanceDegrees := userPrefs.MaxPickupDistanceMiles * milesToDegrees
	// Add 50% buffer for bounding box approximation to catch more candidates
	// The actual haversine distance check will filter precisely later
	// Ensure minimum bounding box of 0.01 degrees (≈0.7 miles) to catch very close neighbors
	// This prevents edge case where MaxPickupDistanceMiles is 0 or very small
	maxDistanceDegrees = math.Max(0.01, maxDistanceDegrees*1.5)

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindCandidatesByPreferences: Using distance filter\",\"user_id\":\"%s\",\"max_pickup_miles\":\"%.2f\",\"max_distance_degrees\":\"%.6f\"}",
		userID, userPrefs.MaxPickupDistanceMiles, maxDistanceDegrees)

	// Build a query that filters by both home proximity and destination proximity
	// Using bounding box for initial filtering (will be refined by actual distance calculation)
	query := `
		SELECT DISTINCT
            u.id, u.name, u.display_name, u.home_latitude, u.home_longitude, u.clerk_id,
			ump.user_demographics, ump.demographic_preferences,
			ump.arrival_time, ump.commute_days,
			ump.destination_latitude, ump.destination_longitude,
			-- Include ORDER BY expressions in SELECT for DISTINCT compatibility
			CASE WHEN ump.destination_latitude IS NOT NULL AND ump.destination_longitude IS NOT NULL
				THEN ABS(ump.destination_latitude - $4::DECIMAL) + ABS(ump.destination_longitude - $5::DECIMAL)
				ELSE 999 END as dest_distance,
			ABS(u.home_latitude - $2::DECIMAL) + ABS(u.home_longitude - $3::DECIMAL) as home_distance
		FROM users u
		JOIN user_matching_preferences ump ON u.id = ump.user_id
		WHERE u.id != $1 
		AND ump.is_active = true
		AND u.home_latitude IS NOT NULL 
		AND u.home_longitude IS NOT NULL
		AND ump.destination_latitude IS NOT NULL 
		AND ump.destination_longitude IS NOT NULL
		-- Filter by home location proximity using MaxPickupDistanceMiles
		-- Cast parameters to DECIMAL to avoid type ambiguity
		AND u.home_latitude BETWEEN ($2::DECIMAL - $6::DECIMAL) AND ($2::DECIMAL + $6::DECIMAL)
		AND u.home_longitude BETWEEN ($3::DECIMAL - $6::DECIMAL) AND ($3::DECIMAL + $6::DECIMAL)
		-- Filter by destination proximity (same destination = better match)
		AND ump.destination_latitude BETWEEN ($4::DECIMAL - $7::DECIMAL) AND ($4::DECIMAL + $7::DECIMAL)
		AND ump.destination_longitude BETWEEN ($5::DECIMAL - $7::DECIMAL) AND ($5::DECIMAL + $7::DECIMAL)
	`

	// Use reasonable bounding box for destination (0.15 degrees ≈ 10 miles) to catch same area/neighborhood
	// This is large enough to catch addresses on the same street (like "1889 Mowry" vs "1900 Mowry Avenue")
	// but not so large that it includes unrelated destinations
	// The actual haversine distance check will filter precisely based on MaxDetourMinutes
	destinationBoundingBox := 0.15 // degrees (≈ 10 miles) - reasonable for same area/neighborhood

	args := []interface{}{
		userID,
		currentUserHomeLat.Float64,      // $2: current user's home latitude
		currentUserHomeLng.Float64,      // $3: current user's home longitude
		*userPrefs.DestinationLatitude,  // $4: current user's destination latitude
		*userPrefs.DestinationLongitude, // $5: current user's destination longitude
		maxDistanceDegrees,              // $6: max distance in degrees for home proximity
		destinationBoundingBox,          // $7: bounding box for destination proximity
	}

	// Add optional schedule filters if user has them set
	// PURPOSE: When users match, their arrival times are merged (earlier time wins)
	// So we need compatible arrival times - within the user's flexibility tolerance
	// Add a small buffer (15 minutes) to account for rounding/geocoding differences
	argIndex := 8
	if userPrefs.ArrivalTime != nil && *userPrefs.ArrivalTime != "" {
		// Use user's ScheduleFlexibilityMinutes preference
		// Default to 30 minutes if not set (reasonable default)
		flexibilityMinutes := userPrefs.ScheduleFlexibilityMinutes
		if flexibilityMinutes <= 0 {
			flexibilityMinutes = 30 // Reasonable default: 30 minutes
		}
		// Add small buffer (15 minutes) to account for rounding and slight variations
		// This ensures we don't filter out compatible matches due to minor time differences
		flexibilityMinutes = flexibilityMinutes + 15

		// Allow NULL arrival_time (user hasn't set it) OR times within flexibility window
		// This is purposeful: users with compatible schedules can form carpools
		query += fmt.Sprintf(`
		AND (
			ump.arrival_time IS NULL 
			OR ABS(EXTRACT(EPOCH FROM (ump.arrival_time - $%d::time))/60) <= $%d
		)`, argIndex, argIndex+1)
		args = append(args, *userPrefs.ArrivalTime, flexibilityMinutes)
		argIndex += 2
	}

	// Filter by commute days - PURPOSE: When users match, carpool uses intersection of their days
	// Example: User A (Mon/Wed/Fri) + User B (Mon/Wed) = Carpool on Mon/Wed only
	// So we require at least 1 day overlap (using && operator for array intersection)
	if len(userPrefs.CommuteDays) > 0 {
		// Build a safe ARRAY[...] literal from validated day values
		validDays := map[string]bool{"mon": true, "tue": true, "wed": true, "thu": true, "fri": true, "sat": true, "sun": true}
		var dayLiterals []string
		for _, d := range userPrefs.CommuteDays {
			dl := strings.ToLower(strings.TrimSpace(d))
			if validDays[dl] {
				dayLiterals = append(dayLiterals, fmt.Sprintf("'%s'", dl))
			}
		}
		if len(dayLiterals) > 0 {
			// Use overlap (&&) which requires at least 1 day in common
			// This is purposeful: users need at least one common day to form a carpool
			// Allow NULL commute_days (user hasn't set it) OR at least 1 day overlap
			query += "\n\t\tAND (\n\t\t\tump.commute_days IS NULL \n\t\t\tOR ump.commute_days && ARRAY[" + strings.Join(dayLiterals, ",") + "]::text[]\n\t\t)\n"
		}
	}

	// Don't require demographics in SQL query - we'll check in Go code
	// This allows matches even if demographics aren't set (for testing and flexibility)

	// Add driver preference compatibility if specified
	// Make this more lenient - only filter if both users have strict preferences
	// Allow "flexible" users and NULL to match with anyone
	if userPrefs.DriverPreference != "" && userPrefs.DriverPreference != "flexible" {
		// If user wants to be driver, find passengers, flexible users, or NULL
		// If passenger, find drivers, flexible users, or NULL
		// This allows more matches while still respecting preferences
		if userPrefs.DriverPreference == "driver" {
			query += "\n\t\tAND (ump.driver_preference = 'passenger' OR ump.driver_preference = 'flexible' OR ump.driver_preference IS NULL)"
		} else if userPrefs.DriverPreference == "passenger" {
			query += "\n\t\tAND (ump.driver_preference = 'driver' OR ump.driver_preference = 'flexible' OR ump.driver_preference IS NULL)"
		}
	}
	// If user is flexible, don't filter by driver preference at all

	// Smart ordering: prioritize closer homes and destinations
	// This helps find best matches first (will be re-sorted by compatibility score in handler)
	// Note: Use the aliases from SELECT list since we're using DISTINCT
	query += `
        ORDER BY 
			-- Prioritize closer destinations (if both have destinations)
			dest_distance,
			-- Then prioritize closer homes
			home_distance
        LIMIT 50
    `

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"🔍 MATCHING: FindCandidatesByPreferences: Executing preference-driven query\",\"user_id\":\"%s\",\"max_distance\":\"%.2f\",\"destination_bbox\":\"%.3f\",\"home_bbox\":\"%.6f\",\"args_count\":%d}",
		userID, userPrefs.MaxPickupDistanceMiles, destinationBoundingBox, maxDistanceDegrees, len(args))

	// Log query parameters for debugging
	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"🔍 MATCHING: FindCandidatesByPreferences: Query parameters\",\"user_id\":\"%s\",\"home_lat\":\"%.6f\",\"home_lng\":\"%.6f\",\"dest_lat\":\"%.6f\",\"dest_lng\":\"%.6f\",\"max_pickup_miles\":\"%.2f\"}",
		userID, currentUserHomeLat.Float64, currentUserHomeLng.Float64, *userPrefs.DestinationLatitude, *userPrefs.DestinationLongitude, userPrefs.MaxPickupDistanceMiles)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindCandidatesByPreferences: Database query failed\",\"user_id\":\"%s\",\"error\":\"%v\",\"query\":\"%s\",\"args\":\"%v\"}", userID, err, query, args)
		return nil, fmt.Errorf("error finding candidates by preferences: %w", err)
	}
	defer rows.Close()

	var candidates []*models.User
	var candidatesFromDB int
	var filteredByHomeDistance int
	var filteredByDestDistance int
	var filteredByDemographics int
	for rows.Next() {
		var user models.User
		var userDemographics models.UserDemographics
		var demographicPreferences models.DemographicPreferences
		var arrivalTime sql.NullString
		var commuteDaysRaw sql.NullString
		var candidateDestLat, candidateDestLng sql.NullFloat64
		var destDistance, homeDistance float64 // ORDER BY expressions (not used, but must be scanned)

		err := rows.Scan(
			&user.ID, &user.Name, &user.DisplayName,
			&user.HomeLatitude, &user.HomeLongitude, &user.ClerkID,
			&userDemographics, &demographicPreferences,
			&arrivalTime, &commuteDaysRaw,
			&candidateDestLat, &candidateDestLng,
			&destDistance, &homeDistance, // Scan the ORDER BY expressions
		)
		if err != nil {
			log.Printf("{\"severity\":\"ERROR\",\"message\":\"FindCandidatesByPreferences: Error scanning candidate\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
			continue
		}

		candidatesFromDB++
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"🔍 MATCHING: FindCandidatesByPreferences: Candidate %d from DB - Processing\",\"user_id\":\"%s\",\"candidate_id\":\"%s\",\"candidate_name\":\"%s\",\"candidate_clerk_id\":\"%s\",\"candidate_home_lat\":\"%.6f\",\"candidate_home_lng\":\"%.6f\",\"candidate_dest_lat\":\"%.6f\",\"candidate_dest_lng\":\"%.6f\"}",
			candidatesFromDB, userID, user.ID.String(), user.Name, user.ClerkID, user.HomeLatitude, user.HomeLongitude, candidateDestLat.Float64, candidateDestLng.Float64)

		// Parse commute_days if present (Postgres returns arrays like "{mon,wed,fri}")
		if commuteDaysRaw.Valid {
			s := strings.Trim(commuteDaysRaw.String, "{}")
			if s != "" {
				_ = strings.Split(s, ",") // parsed but not used yet; keep for future extensions
			}
		}

		// Calculate actual distance between home locations using haversine formula
		homeDistanceMiles := haversineDistance(
			currentUserHomeLat.Float64, currentUserHomeLng.Float64,
			user.HomeLatitude, user.HomeLongitude,
		)

		// Filter by MaxPickupDistanceMiles (actual distance, not just bounding box)
		if homeDistanceMiles > userPrefs.MaxPickupDistanceMiles {
			filteredByHomeDistance++
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"🔍 MATCHING: FindCandidatesByPreferences: Candidate filtered by home distance\",\"user_id\":\"%s\",\"candidate_id\":\"%s\",\"candidate_name\":\"%s\",\"distance_miles\":\"%.2f\",\"max_miles\":\"%.2f\"}",
				userID, user.ID.String(), user.Name, homeDistanceMiles, userPrefs.MaxPickupDistanceMiles)
			continue
		}

		// Check destination proximity - PURPOSE: Users going to same/similar destinations can share rides
		// When carpools are created, they use a shared destination, so we need compatible destinations
		if candidateDestLat.Valid && candidateDestLng.Valid {
			destDistanceMiles := haversineDistance(
				*userPrefs.DestinationLatitude, *userPrefs.DestinationLongitude,
				candidateDestLat.Float64, candidateDestLng.Float64,
			)

			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"🔍 MATCHING: FindCandidatesByPreferences: Checking destination distance\",\"user_id\":\"%s\",\"candidate_id\":\"%s\",\"candidate_name\":\"%s\",\"dest_distance_miles\":\"%.4f\",\"user_dest_lat\":\"%.6f\",\"user_dest_lng\":\"%.6f\",\"candidate_dest_lat\":\"%.6f\",\"candidate_dest_lng\":\"%.6f\"}",
				userID, user.ID.String(), user.Name, destDistanceMiles, *userPrefs.DestinationLatitude, *userPrefs.DestinationLongitude, candidateDestLat.Float64, candidateDestLng.Float64)

			// Same street/address check: If destinations are very close (within 1.0 miles),
			// treat them as the same destination (e.g., "1889 Mowry" vs "1900 Mowry Avenue")
			// This accounts for geocoding differences for addresses on the same street
			// Using 1.0 miles instead of 0.5 to be more lenient for same-street addresses
			// even if geocoding is slightly inaccurate
			if destDistanceMiles <= 1.0 {
				log.Printf("{\"severity\":\"DEBUG\",\"message\":\"🔍 MATCHING: FindCandidatesByPreferences: Same/similar destination (within 1.0 miles) - ALLOWING\",\"user_id\":\"%s\",\"candidate_id\":\"%s\",\"candidate_name\":\"%s\",\"dest_distance_miles\":\"%.4f\"}",
					userID, user.ID.String(), user.Name, destDistanceMiles)
				// Allow match - same destination area
			} else {
				// For different destinations, use MaxDetourMinutes to determine if acceptable
				// PURPOSE: User's MaxDetourMinutes represents their willingness to detour
				// At 30mph average, 1 minute ≈ 0.5 miles
				// Add 20% buffer to account for geocoding differences and route variations
				// Minimum 2 miles to ensure destinations are reasonably close
				maxDestDistance := math.Max(2.0, float64(userPrefs.MaxDetourMinutes)*0.5*1.2)

				if destDistanceMiles > maxDestDistance {
					filteredByDestDistance++
					log.Printf("{\"severity\":\"DEBUG\",\"message\":\"🔍 MATCHING: FindCandidatesByPreferences: Candidate filtered by destination distance (exceeds detour tolerance)\",\"user_id\":\"%s\",\"candidate_id\":\"%s\",\"candidate_name\":\"%s\",\"dest_distance_miles\":\"%.2f\",\"max_allowed\":\"%.2f\",\"max_detour_minutes\":%d}",
						userID, user.ID.String(), user.Name, destDistanceMiles, maxDestDistance, userPrefs.MaxDetourMinutes)
					continue
				} else {
					log.Printf("{\"severity\":\"DEBUG\",\"message\":\"🔍 MATCHING: FindCandidatesByPreferences: Destination within detour tolerance - ALLOWING\",\"user_id\":\"%s\",\"candidate_id\":\"%s\",\"candidate_name\":\"%s\",\"dest_distance_miles\":\"%.2f\",\"max_allowed\":\"%.2f\",\"max_detour_minutes\":%d}",
						userID, user.ID.String(), user.Name, destDistanceMiles, maxDestDistance, userPrefs.MaxDetourMinutes)
				}
			}
		} else {
			// If candidate doesn't have destination set, this shouldn't happen (SQL filter prevents it)
			// But log it for debugging
			log.Printf("{\"severity\":\"WARNING\",\"message\":\"FindCandidatesByPreferences: Candidate missing destination coordinates\",\"user_id\":\"%s\",\"candidate_id\":\"%s\",\"candidate_name\":\"%s\"}",
				userID, user.ID.String(), user.Name)
		}

		// Detailed demographic compatibility check
		// If demographics aren't set, allow the match (for testing and flexibility)
		demographicsSet := userDemographics.AgeRange != "" || userDemographics.Gender != "" || userDemographics.StudentStatus != ""
		preferencesSet := len(demographicPreferences.AgePreferences) > 0 || len(demographicPreferences.GenderPreferences) > 0 || demographicPreferences.StudentPreference != ""

		if !demographicsSet || !preferencesSet {
			// If demographics aren't fully set, allow the match
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindCandidatesByPreferences: Allowing match without full demographics\",\"user_id\":\"%s\",\"candidate_id\":\"%s\",\"demographics_set\":%v,\"preferences_set\":%v}",
				userID, user.ID.String(), demographicsSet, preferencesSet)
			candidates = append(candidates, &user)
		} else if isDemographicallyCompatible(userDemographics, demographicPreferences) {
			candidates = append(candidates, &user)
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindCandidatesByPreferences: Candidate passed all filters\",\"user_id\":\"%s\",\"candidate_id\":\"%s\",\"candidate_name\":\"%s\"}",
				userID, user.ID.String(), user.Name)
		} else {
			filteredByDemographics++
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"FindCandidatesByPreferences: Candidate filtered by demographics\",\"user_id\":\"%s\",\"candidate_id\":\"%s\",\"candidate_name\":\"%s\"}",
				userID, user.ID.String(), user.Name)
		}
	}

	// CRITICAL SUMMARY LOG - Shows exactly why candidates were filtered
	log.Printf("{\"severity\":\"INFO\",\"message\":\"🔍 MATCHING: FindCandidatesByPreferences: Summary - CRITICAL DEBUG INFO\",\"user_id\":\"%s\",\"candidates_from_db\":%d,\"filtered_by_home_distance\":%d,\"filtered_by_dest_distance\":%d,\"filtered_by_demographics\":%d,\"final_candidates\":%d,\"user_dest_lat\":%.6f,\"user_dest_lng\":%.6f,\"destination_bbox_degrees\":%.3f,\"max_pickup_miles\":%.2f,\"max_detour_minutes\":%d}",
		userID, candidatesFromDB, filteredByHomeDistance, filteredByDestDistance, filteredByDemographics, len(candidates), *userPrefs.DestinationLatitude, *userPrefs.DestinationLongitude, destinationBoundingBox, userPrefs.MaxPickupDistanceMiles, userPrefs.MaxDetourMinutes)

	// Log each final candidate for debugging
	for i, candidate := range candidates {
		log.Printf("{\"severity\":\"DEBUG\",\"message\":\"🔍 MATCHING: FindCandidatesByPreferences: Final candidate\",\"user_id\":\"%s\",\"candidate_index\":%d,\"candidate_id\":\"%s\",\"candidate_name\":\"%s\",\"candidate_clerk_id\":\"%s\"}",
			userID, i+1, candidate.ID.String(), candidate.Name, candidate.ClerkID)
	}

	return candidates, nil
}

// haversineDistance calculates the distance between two points using the Haversine formula (in miles)
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 3959 // Earth's radius in miles

	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180
	lng1Rad := lng1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lng2Rad := lng2 * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlng := lng2Rad - lng1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
