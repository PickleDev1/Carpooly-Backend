-- Migration 010: Add ride distance tracking for analytics
-- This adds GPS coordinates and calculated distance to carpool_rides table

-- Add GPS coordinate columns for start and end points
ALTER TABLE carpool_rides 
ADD COLUMN start_lat FLOAT,
ADD COLUMN start_lng FLOAT,
ADD COLUMN end_lat FLOAT,
ADD COLUMN end_lng FLOAT,
ADD COLUMN calculated_distance FLOAT; -- Distance in miles

-- Add index for better query performance on distance calculations
CREATE INDEX idx_carpool_rides_distance ON carpool_rides(calculated_distance) WHERE calculated_distance IS NOT NULL;

-- Add index for time-based queries (used in completed rides)
CREATE INDEX idx_carpool_rides_start_time ON carpool_rides(start_time);

-- Add comment to document the distance field
COMMENT ON COLUMN carpool_rides.calculated_distance IS 'Distance in miles calculated using Haversine formula between start and end coordinates'; 