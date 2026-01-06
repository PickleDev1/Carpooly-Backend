-- Migration 014: Enhanced Matching Algorithm
-- This migration adds enhanced user profiles and match scoring capabilities

-- Enhanced User Profiles Table
-- Add new columns to user_profiles table (create if doesn't exist)
CREATE TABLE IF NOT EXISTS user_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    schedule JSONB DEFAULT '{}',
    current_group_size INTEGER DEFAULT 1,
    is_available_for_matching BOOLEAN DEFAULT true,
    last_active TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id)
);

-- If user_profiles table already exists, add new columns
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_profiles' AND column_name = 'schedule') THEN
        ALTER TABLE user_profiles ADD COLUMN schedule JSONB DEFAULT '{}';
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_profiles' AND column_name = 'current_group_size') THEN
        ALTER TABLE user_profiles ADD COLUMN current_group_size INTEGER DEFAULT 1;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_profiles' AND column_name = 'is_available_for_matching') THEN
        ALTER TABLE user_profiles ADD COLUMN is_available_for_matching BOOLEAN DEFAULT true;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'user_profiles' AND column_name = 'last_active') THEN
        ALTER TABLE user_profiles ADD COLUMN last_active TIMESTAMP WITH TIME ZONE DEFAULT NOW();
    END IF;
END $$;

-- Matching Results Table
-- Store calculated match scores
CREATE TABLE IF NOT EXISTS user_match_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    potential_match_id UUID REFERENCES users(id) ON DELETE CASCADE,
    total_score DECIMAL(5,2),
    location_score DECIMAL(5,2),
    schedule_score DECIMAL(5,2),
    demographic_score DECIMAL(5,2),
    route_score DECIMAL(5,2),
    group_size_score DECIMAL(5,2),
    role_compatibility_score DECIMAL(5,2),
    match_reasons JSONB,
    dealbreakers JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, potential_match_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_user_match_scores_user_id ON user_match_scores(user_id);
CREATE INDEX IF NOT EXISTS idx_user_match_scores_total_score ON user_match_scores(total_score DESC);
CREATE INDEX IF NOT EXISTS idx_user_match_scores_created_at ON user_match_scores(created_at);
CREATE INDEX IF NOT EXISTS idx_user_profiles_user_id ON user_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_profiles_available ON user_profiles(is_available_for_matching, last_active);

-- Add comments for documentation
COMMENT ON TABLE user_profiles IS 'Enhanced user profiles for matching algorithm';
COMMENT ON TABLE user_match_scores IS 'Stores calculated match scores between users';
COMMENT ON COLUMN user_profiles.schedule IS 'JSONB field containing user schedule information';
COMMENT ON COLUMN user_profiles.current_group_size IS 'Current number of people in user group';
COMMENT ON COLUMN user_profiles.is_available_for_matching IS 'Whether user is available for matching';
COMMENT ON COLUMN user_profiles.last_active IS 'Last time user was active in the system'; 