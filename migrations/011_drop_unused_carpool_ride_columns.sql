-- Migration: Drop unused columns from carpool_rides
ALTER TABLE carpool_rides
    DROP COLUMN IF EXISTS start_lat,
    DROP COLUMN IF EXISTS start_lng,
    DROP COLUMN IF EXISTS end_lat,
    DROP COLUMN IF EXISTS end_lng,
    DROP COLUMN IF EXISTS calculated_distance; 