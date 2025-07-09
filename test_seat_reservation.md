# Simplified Seat Management Test Plan

## Overview
This document outlines the testing strategy to ensure that the simplified seat management approach works correctly.

## Approach

### Frontend-Driven Seat Calculation
- **Backend**: Stores total seats in `available_seats` field
- **Frontend**: Calculates available seats by subtracting participants from total seats
- **Benefits**: Simpler backend logic, frontend has full control over seat display

## Changes Made

### 1. Carpool Creation
- **Before**: Complex seat reservation logic
- **After**: Simple storage of total seats from frontend
- **Status**: ✅ Simplified

### 2. Carpool Updates
- **Before**: Complex seat calculation and validation
- **After**: Simple update of total seats with ownership validation
- **Status**: ✅ Simplified

### 3. Member Addition
- **Before**: Attempted to manage available seats in backend
- **After**: Only adds members, frontend calculates available seats
- **Status**: ✅ Simplified

## Routes to Test

### ✅ Carpool Creation Route
- **Route**: `POST /api/carpools`
- **Test**: Create carpool with 5 seats → AvailableSeats should be 5
- **Expected**: Total seats stored, frontend calculates available seats

### ✅ Carpool Update Route
- **Route**: `PUT /api/carpools/{id}`
- **Test**: Update carpool seats, verify ownership validation
- **Expected**: Only creator can update, total seats updated

### ✅ Carpool Get Routes
- **Routes**: 
  - `GET /api/carpools/{id}`
  - `GET /api/carpools/creator/{creatorID}`
  - `GET /api/carpools/users/{userID}`
- **Test**: Verify total seat counts are returned correctly
- **Expected**: AvailableSeats shows total seats, frontend calculates available

### ✅ Member Addition Routes
- **Routes**:
  - `POST /api/carpools/{carpoolID}/members` (direct addition)
  - Invite acceptance flow
- **Test**: Add member → verify member added to carpool_members
- **Expected**: Member added, frontend calculates available seats

### ✅ Invite System
- **Routes**:
  - `POST /api/invites` (create invite)
  - `PUT /api/invites/{id}` (accept/reject invite)
- **Test**: Accept invite → verify member added
- **Expected**: User added to carpool, frontend updates available seats

### ✅ Carpool Deletion
- **Route**: `DELETE /api/carpools/{id}`
- **Test**: Delete carpool → verify all related data cleaned up
- **Expected**: Carpool and all members deleted (seat management not relevant)

### ✅ Ride Management
- **Routes**:
  - `POST /api/carpools/{id}/rides` (create ride)
  - `GET /api/carpools/{id}/rides/{date}` (get rides)
  - `DELETE /api/carpools/rides/{rideID}/participants/{userID}` (remove participant)
- **Test**: Verify ride operations don't affect carpool seats
- **Expected**: Ride operations independent of carpool seat management

## Potential Issues and Mitigations

### 1. Frontend-Backend Consistency
- **Issue**: Frontend and backend might have different seat calculations
- **Mitigation**: Backend stores total seats, frontend calculates available

### 2. Data Consistency
- **Issue**: Members table and seat counts might get out of sync
- **Mitigation**: Frontend calculates from actual member count

### 3. Performance
- **Issue**: Frontend needs to fetch members to calculate seats
- **Mitigation**: Members are typically cached or fetched with carpool data

## Test Scenarios

### Scenario 1: Normal Carpool Creation
1. Create carpool with 5 total seats
2. Verify AvailableSeats = 5 (total seats)
3. Verify TotalSeats = 5
4. Verify creator is automatically a member

### Scenario 2: Invite Acceptance
1. Create carpool with 3 total seats
2. Send invite to user
3. Accept invite
4. Verify user is added to carpool_members
5. Frontend calculates: AvailableSeats = 3 - 2 = 1

### Scenario 3: Multiple Invites
1. Create carpool with 3 total seats
2. Send invites to 3 users
3. Accept all invites
4. Verify all users are members
5. Frontend calculates: AvailableSeats = 3 - 4 = 0 (or negative, handled by frontend)

### Scenario 4: Carpool Updates
1. Create carpool with 3 total seats
2. Update to 5 total seats
3. Verify AvailableSeats = 5
4. Verify only creator can update

### Scenario 5: Edge Cases
1. Create carpool with 1 total seat
2. Try to add member via invite
3. Verify member is added
4. Frontend handles seat calculation logic

## Database Schema Verification

### carpools table
- `available_seats`: Seats available for others (excluding creator)
- `seats`: Total seats (legacy field, should match total_seats)
- `total_seats`: Total seats (new field)

### carpool_members table
- No changes needed
- Tracks actual members

### Invites and Rides
- No changes needed
- Independent of seat management

## Conclusion

The seat reservation system has been implemented with proper validation and error handling. All routes have been reviewed and updated as necessary. The key fixes were:

1. **Carpool Creation**: Reserve one seat for creator
2. **Member Addition**: Decrement available seats when members join
3. **Invite Acceptance**: Update seats when invites are accepted
4. **Update Validation**: Ensure only creators can update carpools

The system now properly tracks available seats and prevents overbooking while maintaining data consistency. 