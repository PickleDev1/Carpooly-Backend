-- ============================================================================
-- SAFE DEV DATABASE CLEAR SCRIPT
-- ============================================================================
-- This script deletes ALL data from ALL tables while preserving table structure
-- Use ONLY in development environment!
-- ============================================================================

-- Start a transaction for safety (can rollback if needed)
BEGIN;

-- Truncate all tables with CASCADE to handle foreign key dependencies
-- CASCADE automatically truncates dependent tables
TRUNCATE TABLE 
    users,
    user_analytics,
    carpools,
    carpool_members,
    carpool_rides,
    carpool_stops,
    carpool_schedules,
    invites,
    invite_links,
    user_activity,
    location_tracking,
    user_matching_preferences,
    potential_matches,
    match_requests,
    matching_sessions,
    user_profiles,
    user_match_scores,
    companies,
    sites,
    user_company_memberships
CASCADE;

-- Commit the transaction
COMMIT;

-- ============================================================================
-- ALTERNATIVE: If the above doesn't work, use this simpler version:
-- ============================================================================
-- TRUNCATE TABLE users CASCADE;
-- ============================================================================
-- This will cascade to all dependent tables automatically
-- ============================================================================

