-- Test script to verify demographic preferences migration
-- Run this after applying migration 013 to verify everything works

-- Test 1: Check if columns were added
SELECT column_name, data_type, is_nullable, column_default 
FROM information_schema.columns 
WHERE table_name = 'user_matching_preferences' 
AND column_name IN ('user_demographics', 'demographic_preferences')
ORDER BY column_name;

-- Test 2: Check if default values are set correctly
SELECT 
    user_id,
    user_demographics,
    demographic_preferences
FROM user_matching_preferences 
LIMIT 5;

-- Test 3: Test inserting a new record with demographic data
INSERT INTO user_matching_preferences (
    user_id,
    max_detour_minutes,
    preferred_group_size,
    driver_preference,
    schedule_flexibility_minutes,
    max_pickup_distance_miles,
    min_compatibility_score,
    notification_preferences,
    user_demographics,
    demographic_preferences,
    is_active
) VALUES (
    gen_random_uuid(),
    20,
    3,
    'flexible',
    45,
    3.5,
    0.8,
    '{"email": true, "push": true, "sms": false}'::jsonb,
    '{
        "age_range": "26-35",
        "gender": "female",
        "occupation": "Software Engineer",
        "student_status": "not_student",
        "company": "Google"
    }'::jsonb,
    '{
        "age_preferences": ["18-25", "26-35", "36-45"],
        "gender_preferences": ["female", "any"],
        "student_preference": "both",
        "occupation_preferences": ["Software Engineer", "Product Manager"]
    }'::jsonb,
    true
);

-- Test 4: Verify the insert worked
SELECT 
    user_id,
    user_demographics->>'age_range' as age_range,
    user_demographics->>'gender' as gender,
    user_demographics->>'occupation' as occupation,
    demographic_preferences->'age_preferences' as age_preferences,
    demographic_preferences->'gender_preferences' as gender_preferences
FROM user_matching_preferences 
WHERE user_demographics->>'occupation' = 'Software Engineer'
LIMIT 1;

-- Test 5: Test JSONB querying capabilities
SELECT 
    user_id,
    user_demographics->>'age_range' as age_range
FROM user_matching_preferences 
WHERE user_demographics->>'age_range' = '26-35'
LIMIT 5;

-- Test 6: Test array querying in demographic preferences
SELECT 
    user_id,
    demographic_preferences->'age_preferences' as age_preferences
FROM user_matching_preferences 
WHERE demographic_preferences->'age_preferences' ? '26-35'
LIMIT 5;

-- Clean up test data (optional)
-- DELETE FROM user_matching_preferences WHERE user_demographics->>'occupation' = 'Software Engineer'; 