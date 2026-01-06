-- Migration 017: Fix driver_preference constraint to allow 'flexible'
-- The frontend sends 'flexible' but the constraint was too restrictive

-- Drop any existing driver_preference constraints
ALTER TABLE user_matching_preferences 
DROP CONSTRAINT IF EXISTS user_matching_preferences_driver_preference_check;

ALTER TABLE user_matching_preferences 
DROP CONSTRAINT IF EXISTS chk_driver_preference;

-- Add the correct constraint that allows 'flexible'
ALTER TABLE user_matching_preferences 
ADD CONSTRAINT user_matching_preferences_driver_preference_check 
CHECK (driver_preference IN ('driver', 'passenger', 'either', 'flexible'));

-- Update any existing 'flexible' values to 'either' for consistency with backend logic
UPDATE user_matching_preferences 
SET driver_preference = 'either' 
WHERE driver_preference = 'flexible';
