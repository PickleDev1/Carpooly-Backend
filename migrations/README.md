# Database Migrations

This directory contains all database migration files for the Carpooly backend. These migrations should be run in order when setting up a new database or updating an existing one.

## Migration Order

Run these migrations in the following order:

1. `001_create_users.sql` - Creates the users table
2. `002_create_carpools.sql` - Creates the carpools table
3. `003_create_carpoolSchedule.sql` - Creates the carpool schedule table
4. `004_sort_activities.sql` - Adds sorting to activities
5. `006_add_location_tracking.sql` - Adds location tracking to users
6. `009_create_invite_links.sql` - Creates invite links table
7. `010_add_ride_distance_tracking.sql` - Adds distance tracking to rides
8. `011_drop_unused_carpool_ride_columns.sql` - Cleans up unused columns
9. `012_create_matching_system.sql` - Creates the matching system tables
10. `013_add_demographic_preferences.sql` - Adds demographic preferences
11. `014_enhanced_matching_algorithm.sql` - Enhances matching algorithm
12. `015_create_match_requests.sql` - Creates match requests table
13. `016_add_destination_preferences.sql` - Adds destination preferences
14. `017_fix_driver_preference_constraint.sql` - Fixes driver preference constraint
15. `018_match_requests_acted_at_indexes.sql` - Adds acted_at column and indexes
16. `019_fix_match_requests_foreign_key.sql` - Fixes potential_match_id foreign key

## Key Schema Changes

### Match Requests System (015, 018, 019)
- **015**: Creates `match_requests` table with foreign key to `potential_matches`
- **018**: Adds `acted_at` column and performance indexes
- **019**: Removes foreign key constraint on `potential_match_id` (now nullable)

### Matching System (012, 013, 014, 016)
- **012**: Creates core matching tables (`potential_matches`, `user_matching_preferences`)
- **013**: Adds demographic preferences
- **014**: Enhances matching algorithm
- **016**: Adds destination preferences and schedule fields

## Production Deployment

When deploying to production:

1. **Run all migrations in order** from 001 to 019
2. **Verify schema** matches development database
3. **Test all endpoints** to ensure functionality works

## Important Notes

- **Migration 019** is critical for match request functionality
- **Migration 016** adds destination preferences (required for matching)
- **Migration 017** fixes driver preference constraint
- All migrations are **idempotent** and can be run multiple times safely

## Schema Verification

After running all migrations, verify these key tables exist:
- `users` (with location tracking)
- `carpools` and `carpool_schedules`
- `user_matching_preferences` (with destination fields)
- `match_requests` (with nullable potential_match_id)
- `potential_matches`
- `invite_links`

## Troubleshooting

If you encounter issues:
1. Check that all previous migrations have been run
2. Verify database permissions
3. Check for constraint violations
4. Review migration logs for specific errors
