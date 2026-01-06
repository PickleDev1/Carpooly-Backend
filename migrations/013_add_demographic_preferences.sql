-- Migration 013: Add demographic preferences to user_matching_preferences table
-- This migration adds JSONB columns for user demographics and demographic preferences
-- to the existing user_matching_preferences table created in migration 012

-- Add new columns with empty JSONB defaults as specified in requirements
ALTER TABLE user_matching_preferences 
ADD COLUMN IF NOT EXISTS user_demographics JSONB DEFAULT '{}'::jsonb;

ALTER TABLE user_matching_preferences 
ADD COLUMN IF NOT EXISTS demographic_preferences JSONB DEFAULT '{}'::jsonb;

-- Update existing records to have the new structure (backward compatibility)
-- Set default values for existing records that have empty JSONB objects
UPDATE user_matching_preferences 
SET user_demographics = '{
  "age_range": "26-35",
  "gender": "prefer_not_to_say",
  "occupation": "",
  "student_status": "not_student", 
  "company": ""
}'::jsonb
WHERE user_demographics = '{}'::jsonb OR user_demographics IS NULL;

UPDATE user_matching_preferences
SET demographic_preferences = '{
  "age_preferences": ["18-25", "26-35", "36-45", "46-55"],
  "gender_preferences": ["any"],
  "student_preference": "both",
  "occupation_preferences": []
}'::jsonb
WHERE demographic_preferences = '{}'::jsonb OR demographic_preferences IS NULL;

-- Add comments for documentation
COMMENT ON COLUMN user_matching_preferences.user_demographics IS 'JSONB field containing user demographic information (age_range, gender, occupation, student_status, company)';
COMMENT ON COLUMN user_matching_preferences.demographic_preferences IS 'JSONB field containing user preferences for matching demographics (age_preferences, gender_preferences, student_preference, occupation_preferences)';

-- Create indexes for the new JSONB columns for better query performance
-- Using proper PostgreSQL JSONB index syntax
CREATE INDEX IF NOT EXISTS idx_user_matching_preferences_user_demographics ON user_matching_preferences USING GIN (user_demographics);
CREATE INDEX IF NOT EXISTS idx_user_matching_preferences_demographic_preferences ON user_matching_preferences USING GIN (demographic_preferences); 