-- Migration 016: Add destination and schedule fields to user_matching_preferences
-- This migration adds destination coordinates and schedule preferences to enable
-- preference-driven matching instead of random/broad user lists

-- Add destination fields (REQUIRED for matching)
ALTER TABLE user_matching_preferences 
ADD COLUMN IF NOT EXISTS destination_latitude DOUBLE PRECISION;

ALTER TABLE user_matching_preferences 
ADD COLUMN IF NOT EXISTS destination_longitude DOUBLE PRECISION;

-- Add schedule fields (OPTIONAL)
ALTER TABLE user_matching_preferences 
ADD COLUMN IF NOT EXISTS arrival_time TIME;

ALTER TABLE user_matching_preferences 
ADD COLUMN IF NOT EXISTS commute_days TEXT[];

-- Add constraints for data validation (only if they don't exist)
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_destination_latitude') THEN
        ALTER TABLE user_matching_preferences 
        ADD CONSTRAINT chk_destination_latitude 
        CHECK (destination_latitude IS NULL OR (destination_latitude >= -90 AND destination_latitude <= 90));
    END IF;
END $$;

DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_destination_longitude') THEN
        ALTER TABLE user_matching_preferences 
        ADD CONSTRAINT chk_destination_longitude 
        CHECK (destination_longitude IS NULL OR (destination_longitude >= -180 AND destination_longitude <= 180));
    END IF;
END $$;

DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_commute_days') THEN
        ALTER TABLE user_matching_preferences 
        ADD CONSTRAINT chk_commute_days 
        CHECK (commute_days IS NULL OR (
            array_length(commute_days, 1) IS NULL OR 
            commute_days <@ ARRAY['mon','tue','wed','thu','fri','sat','sun']::TEXT[]
        ));
    END IF;
END $$;

-- Add comments for documentation
COMMENT ON COLUMN user_matching_preferences.destination_latitude IS 'Latitude of user destination (required for matching)';
COMMENT ON COLUMN user_matching_preferences.destination_longitude IS 'Longitude of user destination (required for matching)';
COMMENT ON COLUMN user_matching_preferences.arrival_time IS 'Preferred arrival time at destination (optional)';
COMMENT ON COLUMN user_matching_preferences.commute_days IS 'Days of the week user commutes (optional, values: mon,tue,wed,thu,fri,sat,sun)';

-- Create indexes for performance on new fields
CREATE INDEX IF NOT EXISTS idx_user_matching_preferences_destination 
ON user_matching_preferences(destination_latitude, destination_longitude) 
WHERE destination_latitude IS NOT NULL AND destination_longitude IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_matching_preferences_commute_days 
ON user_matching_preferences USING GIN (commute_days) 
WHERE commute_days IS NOT NULL;
