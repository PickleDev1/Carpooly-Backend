# Dashboard Next Ride Feature - Frontend Implementation Guide

## Overview

A new API endpoint has been added to display the user's next upcoming carpool ride on the dashboard. This endpoint returns all the information needed to show:
- The next ride's details (time, date, etc.)
- The carpool name
- Who is driving (or if no driver is assigned)
- Whether the current user is the driver

## API Endpoint

**Endpoint:** `GET /api/rides/next`

**Authentication:** Required (uses Clerk authentication via Authorization header)

**Method:** GET

**Headers:**
```
Authorization: Bearer <clerk_token>
Content-Type: application/json
```

## Response Format

### Success Response (Next Ride Found)

```json
{
  "ride": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "carpool_id": "660e8400-e29b-41d4-a716-446655440001",
    "driver_id": "770e8400-e29b-41d4-a716-446655440002",
    "start_time": "2026-01-13T08:00:00Z",
    "status": 0,
    "participants": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "John Doe",
        "display_name": "John",
        "email": "john@example.com",
        "clerk_id": "user_abc123"
      },
      {
        "id": "880e8400-e29b-41d4-a716-446655440003",
        "name": "Jane Smith",
        "display_name": "Jane",
        "email": "jane@example.com",
        "clerk_id": "user_xyz789"
      }
    ],
    "location_lat": 37.7749,
    "location_lng": -122.4194,
    "miles_saved": 15.5,
    "created_at": "2026-01-10T10:00:00Z",
    "updated_at": "2026-01-10T10:00:00Z"
  },
  "carpool_name": "Work Commute",
  "driver": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "name": "John Doe",
    "display_name": "John",
    "email": "john@example.com",
    "clerk_id": "user_abc123",
    "city": null,
    "state": null,
    "location_sharing_enabled": false,
    "home_latitude": 0,
    "home_longitude": 0,
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z"
  },
  "is_user_driver": false
}
```

### Success Response (No Next Ride)

```json
null
```

**Status Code:** 200 OK

### Error Responses

**401 Unauthorized** - Missing or invalid authentication token
```json
{
  "error": "Unauthorized"
}
```

**404 Not Found** - User not found
```json
{
  "error": "User not found"
}
```

**500 Internal Server Error** - Server error
```json
{
  "error": "Failed to get next ride"
}
```

## Response Fields Explained

### `ride` object
The full carpool ride object containing:
- `id`: Unique ride identifier
- `carpool_id`: ID of the carpool this ride belongs to
- `driver_id`: UUID of the driver (can be `null` if no driver assigned)
- `start_time`: ISO 8601 timestamp of when the ride starts (e.g., "2026-01-13T08:00:00Z")
- `status`: Ride status (0 = Pending, 1 = Active, 2 = Completed)
- `participants`: Array of user objects participating in the ride
- `location_lat` / `location_lng`: Optional pickup location coordinates
- `miles_saved`: Optional miles saved by carpooling
- `created_at` / `updated_at`: Timestamps

### `carpool_name` string
The name of the carpool (e.g., "Work Commute", "Morning Route")

### `driver` object (nullable)
- **If driver is assigned:** Contains full user object with driver's information
- **If no driver assigned:** This field will be `null`

**Note:** Driver object includes minimal fields needed for display. Some fields like `created_at`, `home_latitude` are set to zero/default values and can be ignored.

### `is_user_driver` boolean
- `true`: The authenticated user is the driver for this ride
- `false`: Someone else is driving, or no driver is assigned

## Frontend Implementation

### 1. API Call Example (TypeScript/React)

```typescript
interface NextRideInfo {
  ride: {
    id: string;
    carpool_id: string;
    driver_id: string | null;
    start_time: string; // ISO 8601 format
    status: number;
    participants: Array<{
      id: string;
      name: string;
      display_name: string;
      email: string;
      clerk_id: string;
    }>;
    location_lat?: number;
    location_lng?: number;
    miles_saved?: number;
    created_at: string;
    updated_at: string;
  };
  carpool_name: string;
  driver: {
    id: string;
    name: string;
    display_name: string;
    email: string;
    clerk_id: string;
  } | null;
  is_user_driver: boolean;
}

async function getNextRide(): Promise<NextRideInfo | null> {
  const response = await fetch('/api/rides/next', {
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${getClerkToken()}`,
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    if (response.status === 401) {
      throw new Error('Unauthorized');
    }
    if (response.status === 404) {
      throw new Error('User not found');
    }
    throw new Error('Failed to get next ride');
  }

  const data = await response.json();
  return data; // Will be null if no upcoming rides
}
```

### 2. React Component Example

```tsx
import { useState, useEffect } from 'react';

function DashboardNextRide() {
  const [nextRide, setNextRide] = useState<NextRideInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchNextRide() {
      try {
        setLoading(true);
        const data = await getNextRide();
        setNextRide(data);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load next ride');
        setNextRide(null);
      } finally {
        setLoading(false);
      }
    }

    fetchNextRide();
  }, []);

  if (loading) {
    return <div>Loading next ride...</div>;
  }

  if (error) {
    return <div>Error: {error}</div>;
  }

  if (!nextRide) {
    return (
      <div className="next-ride-card">
        <h3>Next Ride</h3>
        <p>No upcoming rides scheduled</p>
      </div>
    );
  }

  const { ride, carpool_name, driver, is_user_driver } = nextRide;
  const startTime = new Date(ride.start_time);
  const formattedTime = startTime.toLocaleTimeString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  });
  const formattedDate = startTime.toLocaleDateString('en-US', {
    weekday: 'long',
    month: 'long',
    day: 'numeric',
  });

  return (
    <div className="next-ride-card">
      <h3>Next Ride</h3>
      <div className="ride-info">
        <h4>{carpool_name}</h4>
        <p className="ride-time">
          {formattedDate} at {formattedTime}
        </p>
        
        <div className="driver-info">
          {is_user_driver ? (
            <span className="driver-badge">You are driving</span>
          ) : driver ? (
            <span>Driver: {driver.display_name || driver.name}</span>
          ) : (
            <span className="no-driver">No driver assigned</span>
          )}
        </div>

        <div className="participants-count">
          {ride.participants.length} participant{ride.participants.length !== 1 ? 's' : ''}
        </div>
      </div>
    </div>
  );
}
```

### 3. Display Logic

#### Driver Display Logic

```typescript
function getDriverDisplayText(nextRide: NextRideInfo | null): string {
  if (!nextRide) return 'No upcoming rides';
  
  if (nextRide.is_user_driver) {
    return 'You are driving';
  }
  
  if (nextRide.driver) {
    return `Driver: ${nextRide.driver.display_name || nextRide.driver.name}`;
  }
  
  return 'No driver assigned';
}
```

#### Time Formatting

```typescript
function formatRideTime(startTime: string): string {
  const date = new Date(startTime);
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const rideDate = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  
  const daysDiff = Math.floor((rideDate.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));
  
  let dateStr: string;
  if (daysDiff === 0) {
    dateStr = 'Today';
  } else if (daysDiff === 1) {
    dateStr = 'Tomorrow';
  } else if (daysDiff < 7) {
    dateStr = date.toLocaleDateString('en-US', { weekday: 'long' });
  } else {
    dateStr = date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
    });
  }
  
  const timeStr = date.toLocaleTimeString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  });
  
  return `${dateStr} at ${timeStr}`;
}
```

## UI/UX Recommendations

### Visual Design Suggestions

1. **Card Layout**
   - Use a prominent card/box on the dashboard
   - Include the carpool name as a header
   - Show date/time prominently
   - Display driver information clearly

2. **Driver Badge**
   - If `is_user_driver === true`: Show a badge like "You are driving" with a different color (e.g., green)
   - If driver exists: Show "Driver: [Name]" in normal text
   - If no driver: Show "No driver assigned" in muted/gray text, possibly with a "Sign up as driver" button

3. **Time Display**
   - For today: "Today at 8:00 AM"
   - For tomorrow: "Tomorrow at 8:00 AM"
   - For this week: "Wednesday at 8:00 AM"
   - For next week+: "Jan 20 at 8:00 AM"

4. **Empty State**
   - When `nextRide === null`, show a friendly message like:
     - "No upcoming rides scheduled"
     - Optionally include a link to "Find matches" or "Create carpool"

5. **Click Action**
   - Make the card clickable to navigate to the full ride details
   - Could link to: `/carpools/${ride.carpool_id}/rides/${ride.id}` or similar

### Example UI Component Structure

```
┌─────────────────────────────────────┐
│  Next Ride                          │
├─────────────────────────────────────┤
│  Work Commute                       │
│                                     │
│  Tomorrow at 8:00 AM                │
│                                     │
│  🚗 You are driving                 │
│                                     │
│  2 participants                     │
│                                     │
│  [View Details →]                   │
└─────────────────────────────────────┘
```

Or when someone else is driving:

```
┌─────────────────────────────────────┐
│  Next Ride                          │
├─────────────────────────────────────┤
│  Work Commute                       │
│                                     │
│  Wednesday at 8:00 AM               │
│                                     │
│  Driver: John                       │
│                                     │
│  3 participants                     │
│                                     │
│  [View Details →]                   │
└─────────────────────────────────────┘
```

## Error Handling

### Recommended Error Handling

```typescript
try {
  const nextRide = await getNextRide();
  // Handle success
} catch (error) {
  if (error.message === 'Unauthorized') {
    // Redirect to login or refresh token
    handleAuthError();
  } else if (error.message === 'User not found') {
    // This shouldn't happen for authenticated users, but handle gracefully
    console.error('User not found');
  } else {
    // Network or server error
    showErrorMessage('Unable to load next ride. Please try again.');
  }
}
```

## Refresh Strategy

### When to Refresh

1. **On Dashboard Load**: Fetch when the dashboard component mounts
2. **After Ride Actions**: Refresh after:
   - Creating a new carpool
   - Accepting a match request
   - Signing up as driver
   - Leaving a ride
3. **Periodic Refresh**: Optionally refresh every 5-10 minutes to catch new rides
4. **After Navigation**: Refresh when user returns to dashboard

### Example with Auto-Refresh

```typescript
useEffect(() => {
  fetchNextRide();
  
  // Refresh every 5 minutes
  const interval = setInterval(() => {
    fetchNextRide();
  }, 5 * 60 * 1000);
  
  return () => clearInterval(interval);
}, []);
```

## Testing Scenarios

### Test Cases to Cover

1. **User has next ride, is driver**
   - Verify `is_user_driver === true`
   - Verify driver info shows "You are driving"

2. **User has next ride, is passenger**
   - Verify `is_user_driver === false`
   - Verify driver name is displayed

3. **User has next ride, no driver assigned**
   - Verify `driver === null`
   - Verify "No driver assigned" is shown

4. **User has no upcoming rides**
   - Verify response is `null`
   - Verify empty state is shown

5. **Multiple upcoming rides**
   - Verify only the earliest one is returned (backend handles this)

6. **Error cases**
   - Test with invalid token (401)
   - Test with network error
   - Test with server error (500)

## Notes

- The `start_time` is in ISO 8601 format (UTC). You may need to convert to user's local timezone.
- The `driver` object only contains essential fields for display. Don't rely on `created_at`, `home_latitude`, etc. as they're set to default values.
- The `participants` array includes all users in the ride, including the driver.
- The endpoint only returns rides with `status` 0 (Pending) or 1 (Active) that are in the future.
- If a user has multiple upcoming rides, only the earliest one (by `start_time`) is returned.

## Questions?

If you need any clarification or run into issues, please reach out to the backend team!
