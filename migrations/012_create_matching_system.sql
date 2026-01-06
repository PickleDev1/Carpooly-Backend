-- Create carpool matching system tables

-- Table: user_matching_preferences
CREATE TABLE user_matching_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    max_detour_minutes INTEGER DEFAULT 15,
    preferred_group_size INTEGER DEFAULT 4,
    driver_preference VARCHAR(20) DEFAULT 'flexible' CHECK (driver_preference IN ('driver', 'passenger', 'flexible')),
    schedule_flexibility_minutes INTEGER DEFAULT 30,
    max_pickup_distance_miles DECIMAL(5,2) DEFAULT 5.0,
    min_compatibility_score DECIMAL(3,2) DEFAULT 0.7,
    notification_preferences JSONB DEFAULT '{"email": true, "push": true, "sms": false}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Table: potential_matches
CREATE TABLE potential_matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user1_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    compatibility_score DECIMAL(3,2) NOT NULL,
    route_overlap_percentage DECIMAL(5,2),
    total_distance_miles DECIMAL(6,2),
    estimated_savings_per_month DECIMAL(8,2),
    match_reasons JSONB,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'expired', 'accepted', 'rejected')),
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP + INTERVAL '7 days'),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user1_id, user2_id)
);

-- Table: match_requests
CREATE TABLE match_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    potential_match_id UUID REFERENCES potential_matches(id) ON DELETE CASCADE,
    message TEXT,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected', 'expired')),
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP + INTERVAL '3 days'),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Table: matching_sessions
CREATE TABLE matching_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'paused', 'expired')),
    last_match_generated_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP + INTERVAL '30 days'),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id)
);

-- Create indexes for performance
CREATE INDEX idx_potential_matches_user1 ON potential_matches(user1_id, status, expires_at);
CREATE INDEX idx_potential_matches_user2 ON potential_matches(user2_id, status, expires_at);
CREATE INDEX idx_potential_matches_compatibility ON potential_matches(compatibility_score DESC);
CREATE INDEX idx_match_requests_from_user ON match_requests(from_user_id, status);
CREATE INDEX idx_match_requests_to_user ON match_requests(to_user_id, status);
CREATE INDEX idx_matching_sessions_user ON matching_sessions(user_id, status);
CREATE INDEX idx_user_matching_preferences_active ON user_matching_preferences(is_active);

-- Add comments for documentation
COMMENT ON TABLE user_matching_preferences IS 'Stores user preferences for carpool matching';
COMMENT ON TABLE potential_matches IS 'Stores potential matches between users with compatibility scores';
COMMENT ON TABLE match_requests IS 'Stores carpool requests between matched users';
COMMENT ON TABLE matching_sessions IS 'Tracks active matching sessions for users'; 