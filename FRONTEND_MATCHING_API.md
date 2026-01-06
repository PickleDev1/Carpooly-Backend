# Frontend Matching API Implementation Guide

## Overview
The matching API has been enhanced to provide smart, preference-based matching with detailed compatibility scores. All routes are now working and return comprehensive match data.

---

## API Endpoints

### 1. Get Potential Matches
**GET** `/api/matching/potential-matches`

Returns all potential matches for the authenticated user, filtered by their preferences and sorted by compatibility score (best matches first).

**Headers:**
```
Authorization: Bearer <clerk_token>
X-User-Timezone: <timezone> (optional)
```

**Response:**
```json
{
  "pending_matches": [
    {
      "id": "match_<uuid1>_<uuid2>_<timestamp>",
      "user2": {
        "id": "<uuid>",
        "clerk_id": "user_...",
        "name": "John Doe",
        "display_name": "John",
        "home_latitude": 37.7749,
        "home_longitude": -122.4194,
        "email": "john@example.com"
      },
      "user2_clerk_id": "user_...",
      "compatibility_score": 0.85,           // 0.0 to 1.0 (for calculations)
      "compatibility_percentage": 85.0,      // 0 to 100 (for display)
      "route_overlap_percentage": 75.5,      // 0 to 100
      "total_distance_miles": 12.3,
      "estimated_savings_per_month": 67.50,
      "match_reasons": [
        "Live in the same neighborhood",
        "High route overlap",
        "Similar work schedule"
      ],
      "schedule": {
        "departure_time": "8:00 AM",
        "frequency": "Daily",
        "flexibility_minutes": 15,
        "compatibility_score": 0.80,          // 0.0 to 1.0
        "compatibility_percentage": 80.0      // 0 to 100
      },
      "status": "pending",
      "expires_at": "2026-01-09T00:00:00Z",
      "created_at": "2026-01-02T00:00:00Z"
    }
  ],
  "accepted_matches": [],
  "expired_matches": []
}
```

**Key Points:**
- Matches are **already sorted by compatibility score** (highest first)
- Only matches above user's `min_compatibility_score` preference are returned
- All matches respect user's distance, schedule, and demographic preferences

---

### 2. Find/Generate New Matches
**POST** `/api/matching/find-matches`

Forces the system to generate new matches. Use this when user wants to refresh their match list.

**Headers:**
```
Authorization: Bearer <clerk_token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "max_results": 10  // Optional, defaults to 10
}
```

**Response:**
```json
{
  "matches_found": 5,
  "message": "Matches generated successfully"
}
```

**Note:** After calling this, call `GET /api/matching/potential-matches` to get the newly generated matches.

---

### 3. Get User Matching Preferences
**GET** `/api/matching/preferences`

Get the current user's matching preferences.

**Response:**
```json
{
  "success": true,
  "preferences": {
    "user_id": "<uuid>",
    "max_detour_minutes": 15,
    "preferred_group_size": 4,
    "driver_preference": "flexible",
    "schedule_flexibility_minutes": 30,
    "max_pickup_distance_miles": 5.0,
    "min_compatibility_score": 0.7,
    "destination_latitude": 37.7849,
    "destination_longitude": -122.4094,
    "arrival_time": "09:00:00",
    "commute_days": ["mon", "tue", "wed", "thu", "fri"],
    "user_demographics": {
      "age_range": "26-35",
      "gender": "male",
      "occupation": "Software Engineer",
      "student_status": "not_student"
    },
    "demographic_preferences": {
      "age_preferences": ["26-35", "36-45"],
      "gender_preferences": ["any"],
      "student_preference": "both",
      "occupation_preferences": []
    },
    "is_active": true
  }
}
```

---

### 4. Update User Matching Preferences
**PUT** `/api/matching/preferences`

Update the user's matching preferences. This will affect future match results.

**Request Body:** (all fields optional)
```json
{
  "max_detour_minutes": 15,
  "preferred_group_size": 4,
  "driver_preference": "flexible",  // "driver" | "passenger" | "flexible"
  "schedule_flexibility_minutes": 30,
  "max_pickup_distance_miles": 5.0,
  "min_compatibility_score": 0.7,   // 0.0 to 1.0
  "destination_latitude": 37.7849,
  "destination_longitude": -122.4094,
  "arrival_time": "09:00:00",       // HH:MM:SS format
  "commute_days": ["mon", "tue", "wed", "thu", "fri"],
  "user_demographics": {...},
  "demographic_preferences": {...}
}
```

---

### 5. Get Match Requests
**GET** `/api/matching/requests`

Get incoming and outgoing match requests.

**Response:**
```json
{
  "incoming": [
    {
      "id": "<uuid>",
      "from_user_id": "<uuid>",
      "to_user_id": "<uuid>",
      "potential_match_id": "<uuid>",
      "message": "Would you like to carpool?",
      "status": "pending",
      "carpool_name": "Morning Commute",
      "from_user": {...},
      "to_user": {...}
    }
  ],
  "outgoing": [...]
}
```

---

## Key Features for Frontend Implementation

### 1. Compatibility Scores
- **`compatibility_percentage`**: Display as "85% Match" or progress bar
- **`compatibility_score`**: Use for calculations (0.0 to 1.0)
- Scores are **already filtered** - only matches above user's `min_compatibility_score` are returned

### 2. Match Sorting
- Matches are **pre-sorted by compatibility score** (best first)
- No need to sort on frontend - display in order received

### 3. Filtering
All filtering is done server-side based on user preferences:
- ✅ Home distance (respects `max_pickup_distance_miles`)
- ✅ Destination proximity (uses `max_detour_minutes`)
- ✅ Schedule compatibility (uses `schedule_flexibility_minutes`)
- ✅ Commute days overlap
- ✅ Demographic preferences
- ✅ Driver preference matching
- ✅ Minimum compatibility score

**Frontend doesn't need to filter** - just display what's returned.

### 4. Display Recommendations

**Match Card Display:**
```javascript
// Example match card data
{
  compatibility: match.compatibility_percentage,  // 85.0
  routeOverlap: match.route_overlap_percentage,   // 75.5
  savings: match.estimated_savings_per_month,     // 67.50
  reasons: match.match_reasons,                   // ["Live nearby", ...]
  schedule: match.schedule.compatibility_percentage, // 80.0
  user: match.user2                               // User object
}
```

**Visual Indicators:**
- **Compatibility %**: Progress bar or badge (green: 80%+, yellow: 70-79%, red: <70%)
- **Route Overlap**: Percentage badge
- **Savings**: Dollar amount with currency symbol
- **Match Reasons**: List of badges or tags
- **Schedule**: Show departure time and compatibility

### 5. Error Handling

**400 Bad Request:**
```json
{
  "error": "Destination required. Please set your destination in preferences."
}
```
→ Redirect user to preferences page to set destination

**401 Unauthorized:**
→ User needs to authenticate/re-login

**500 Internal Server Error:**
→ Show generic error message, log for debugging

### 6. User Flow Recommendations

1. **Initial Load:**
   - Call `GET /api/matching/potential-matches` on page load
   - Display matches in order (already sorted)

2. **Refresh Matches:**
   - Call `POST /api/matching/find-matches`
   - Then call `GET /api/matching/potential-matches` to get new results

3. **Update Preferences:**
   - User updates preferences via `PUT /api/matching/preferences`
   - Automatically refresh matches after update

4. **Match Request:**
   - User clicks "Request Match" on a potential match
   - Use `potential_match_id` from the match object
   - Create match request via appropriate endpoint

---

## Response Field Reference

| Field | Type | Range | Description |
|-------|------|-------|-------------|
| `compatibility_score` | float | 0.0 - 1.0 | Raw score for calculations |
| `compatibility_percentage` | float | 0 - 100 | Display-friendly percentage |
| `route_overlap_percentage` | float | 0 - 100 | How much route overlaps |
| `total_distance_miles` | float | > 0 | Total commute distance |
| `estimated_savings_per_month` | float | >= 0 | Monthly savings in dollars |
| `match_reasons` | array | - | Array of reason strings |
| `schedule.compatibility_percentage` | float | 0 - 100 | Schedule match quality |

---

## Example Frontend Code

```javascript
// Fetch potential matches
async function getPotentialMatches() {
  const response = await fetch('/api/matching/potential-matches', {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    }
  });
  
  if (!response.ok) {
    if (response.status === 400) {
      // Destination not set - redirect to preferences
      router.push('/preferences');
      return;
    }
    throw new Error('Failed to fetch matches');
  }
  
  const data = await response.json();
  return data.pending_matches; // Already sorted by compatibility
}

// Display match card
function MatchCard({ match }) {
  return (
    <div className="match-card">
      <div className="compatibility-badge">
        {match.compatibility_percentage}% Match
      </div>
      
      <div className="user-info">
        <h3>{match.user2.name}</h3>
        <p>{match.user2.display_name}</p>
      </div>
      
      <div className="match-details">
        <div>Route Overlap: {match.route_overlap_percentage}%</div>
        <div>Schedule Match: {match.schedule.compatibility_percentage}%</div>
        <div>Est. Savings: ${match.estimated_savings_per_month}/month</div>
      </div>
      
      <div className="match-reasons">
        {match.match_reasons.map((reason, i) => (
          <span key={i} className="reason-badge">{reason}</span>
        ))}
      </div>
      
      <button onClick={() => requestMatch(match.id)}>
        Request Match
      </button>
    </div>
  );
}
```

---

## Notes

1. **No Frontend Filtering Needed**: All filtering is done server-side based on user preferences
2. **Pre-sorted Results**: Matches are sorted by compatibility score (best first)
3. **Percentage Fields**: Use `compatibility_percentage` for display, `compatibility_score` for calculations
4. **Real-time Updates**: After updating preferences, call `find-matches` then `potential-matches` to refresh
5. **Error Handling**: Always check for 400 status (destination required) and redirect to preferences if needed

---

## Testing Checklist

- [ ] Display matches sorted by compatibility (should already be sorted)
- [ ] Show compatibility percentage (0-100)
- [ ] Display route overlap percentage
- [ ] Show estimated savings
- [ ] Display match reasons as badges/tags
- [ ] Handle "destination required" error (400)
- [ ] Handle authentication errors (401)
- [ ] Refresh matches after preference update
- [ ] Generate new matches button works
- [ ] Match request creation works

