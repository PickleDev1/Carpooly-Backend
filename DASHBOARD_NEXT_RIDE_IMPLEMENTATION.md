# Dashboard Next Ride Implementation

## Overview
This document describes how to implement a dashboard feature that shows the user's next upcoming ride with driver information.

## Implementation Steps

### 1. Add Repository Method (`pkg/repository/carpoolRide_repository.go`)

Add this code at the end of the file (after line 901):

```go
// NextRideInfo represents the next ride with carpool and driver information for dashboard
type NextRideInfo struct {
	Ride         *models.CarpoolRide `json:"ride"`
	CarpoolName  string              `json:"carpool_name"`
	Driver       *models.User        `json:"driver,omitempty"` // nil if no driver assigned
	IsUserDriver bool                `json:"is_user_driver"`  // true if the requesting user is the driver
}

// GetUserNextRide returns the next upcoming ride for a user with carpool and driver information
// Returns nil if no upcoming rides are found
func (r *CarPoolRideRepository) GetUserNextRide(ctx context.Context, userID uuid.UUID) (*NextRideInfo, error) {
	query := `
		SELECT 
			cr.id, cr.carpool_id, cr.driver_id, cr.start_time, cr.status,
			cr.location_lat, cr.location_lng, cr.miles_saved, cr.participants,
			cr.created_at, cr.updated_at,
			c.carpool_name,
			-- Driver information (if driver_id is set)
			driver.id as driver_id, driver.name as driver_name, 
			driver.display_name as driver_display_name, driver.email as driver_email,
			driver.clerk_id as driver_clerk_id
		FROM carpool_rides cr
		JOIN carpools c ON cr.carpool_id = c.id
		JOIN carpool_members cm ON c.id = cm.carpool_id
		LEFT JOIN users driver ON cr.driver_id = driver.id
		WHERE cm.user_id = $1
		  AND cr.start_time > NOW()  -- Only future rides
		  AND cr.status IN (0, 1)     -- Pending or Active status
		ORDER BY cr.start_time ASC
		LIMIT 1
	`

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetUserNextRide: Executing query\",\"user_id\":\"%s\"}", userID)

	var ride models.CarpoolRide
	var participantsJSON []byte
	var locationLat, locationLng, milesSaved sql.NullFloat64
	var driverID sql.NullString
	var carpoolName string
	
	// Driver fields (nullable)
	var driverUserID sql.NullString
	var driverName sql.NullString
	var driverDisplayName sql.NullString
	var driverEmail sql.NullString
	var driverClerkID sql.NullString

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
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
		&carpoolName,
		&driverUserID,
		&driverName,
		&driverDisplayName,
		&driverEmail,
		&driverClerkID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetUserNextRide: No upcoming rides found\",\"user_id\":\"%s\"}", userID)
			return nil, nil // No upcoming rides - not an error
		}
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"GetUserNextRide: Query failed\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		return nil, fmt.Errorf("failed to query next ride: %w", err)
	}

	// Handle NULL values for ride
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
		driverUUID, err := uuid.Parse(driverID.String)
		if err == nil {
			ride.DriverID = &driverUUID
		}
	}

	// Parse participants JSON
	if len(participantsJSON) > 0 {
		if err := json.Unmarshal(participantsJSON, &ride.Participants); err != nil {
			log.Printf("{\"severity\":\"WARNING\",\"message\":\"GetUserNextRide: Failed to unmarshal participants\",\"error\":\"%v\"}", err)
			ride.Participants = []models.User{}
		}
	}

	// Build driver info if driver is assigned
	var driver *models.User
	isUserDriver := false
	if driverUserID.Valid {
		driverUUID, err := uuid.Parse(driverUserID.String)
		if err == nil {
			driver = &models.User{
				ID:          driverUUID,
				Name:        driverName.String,
				DisplayName: driverDisplayName.String,
				Email:       driverEmail.String,
				ClerkID:     driverClerkID.String,
			}
			// Check if the requesting user is the driver
			isUserDriver = driverUUID == userID
		}
	}

	log.Printf("{\"severity\":\"DEBUG\",\"message\":\"GetUserNextRide: Found next ride\",\"user_id\":\"%s\",\"ride_id\":\"%s\",\"carpool_name\":\"%s\",\"has_driver\":%v,\"is_user_driver\":%v}",
		userID, ride.ID, carpoolName, driver != nil, isUserDriver)

	return &NextRideInfo{
		Ride:         &ride,
		CarpoolName:  carpoolName,
		Driver:       driver,
		IsUserDriver: isUserDriver,
	}, nil
}
```

### 2. Add Handler Method (`pkg/handlers/carpoolRides_handlers.go`)

Add this handler method:

```go
// GetUserNextRide returns the next upcoming ride for the authenticated user
// GET /api/rides/next
func (h *CarPoolRideHandler) GetUserNextRide(w http.ResponseWriter, r *http.Request) {
	log.Printf("{\"severity\":\"INFO\",\"message\":\"GetUserNextRide called\",\"method\":\"%s\",\"url\":\"%s\"}", r.Method, r.URL.String())

	// Get Clerk ID from the authenticated session
	clerkID, ok := middleware.GetClerkIDFromContext(r.Context())
	if !ok {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get Clerk ID from context\"}")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Convert clerk_id to user_id
	userID, err := h.userRepo.GetUserIDByClerkID(r.Context(), clerkID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get user ID\",\"clerk_id\":\"%s\",\"error\":\"%v\"}", clerkID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get next ride
	nextRide, err := h.carpoolRideRepo.GetUserNextRide(r.Context(), userID)
	if err != nil {
		log.Printf("{\"severity\":\"ERROR\",\"message\":\"Failed to get next ride\",\"user_id\":\"%s\",\"error\":\"%v\"}", userID, err)
		http.Error(w, "Failed to get next ride", http.StatusInternalServerError)
		return
	}

	// If no next ride found, return null
	if nextRide == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nextRide)
}
```

### 3. Register Route (`main.go`)

Add this route registration in the `protected` router section:

```go
protected.HandleFunc("/rides/next", carpoolRideHandler.GetUserNextRide).Methods("GET")
```

## API Response Format

### Success Response (Next Ride Found)
```json
{
  "ride": {
    "id": "uuid",
    "carpool_id": "uuid",
    "driver_id": "uuid",
    "start_time": "2026-01-13T08:00:00Z",
    "status": 0,
    "participants": [...],
    ...
  },
  "carpool_name": "Work Commute",
  "driver": {
    "id": "uuid",
    "name": "John Doe",
    "display_name": "John",
    "email": "john@example.com",
    ...
  },
  "is_user_driver": false
}
```

### Success Response (No Next Ride)
```json
null
```

## Frontend Usage

The frontend can call `GET /api/rides/next` to get the user's next ride. The response includes:
- **ride**: Full ride details including start_time, participants, etc.
- **carpool_name**: Name of the carpool
- **driver**: Driver information (null if no driver assigned)
- **is_user_driver**: Boolean indicating if the requesting user is the driver

This makes it easy to display:
- "Next Ride: [Carpool Name] at [Time]"
- "Driver: [Driver Name]" or "You are driving" or "No driver assigned"

## Benefits

1. **Single API Call**: Gets all needed info in one request
2. **Clear Driver Status**: `is_user_driver` flag makes it easy to show "You" vs driver name
3. **Null Handling**: Returns null if no upcoming rides (easy to handle in frontend)
4. **Efficient**: Only fetches the next ride (LIMIT 1), ordered by start_time
