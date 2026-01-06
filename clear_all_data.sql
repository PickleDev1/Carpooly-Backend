-- Clear all data from all tables (preserves table structure)
-- This script deletes all rows but keeps all columns and constraints

-- Option 1: Simple CASCADE approach (truncates main tables, cascades to dependent tables)
TRUNCATE TABLE users CASCADE;

-- Option 2: Explicit truncation of all tables (uncomment if Option 1 doesn't work)
-- TRUNCATE TABLE 
--     location_tracking,
--     carpool_stops,
--     carpool_rides,
--     carpool_schedules,
--     carpool_members,
--     match_requests,
--     potential_matches,
--     user_match_scores,
--     user_matching_preferences,
--     user_profiles,
--     user_analytics,
--     invite_links,
--     invites,
--     carpools,
--     matching_sessions,
--     user_activity,
--     users
-- CASCADE;

