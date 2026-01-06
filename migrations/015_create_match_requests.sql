-- Create match_requests table for carpool request system
CREATE TABLE match_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    potential_match_id UUID NOT NULL REFERENCES potential_matches(id) ON DELETE CASCADE,
    message TEXT,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected', 'expired')),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Ensure no duplicate requests between same users for same potential match
    UNIQUE(from_user_id, to_user_id, potential_match_id)
);

-- Create indexes for performance
CREATE INDEX idx_match_requests_from_user ON match_requests(from_user_id);
CREATE INDEX idx_match_requests_to_user ON match_requests(to_user_id);
CREATE INDEX idx_match_requests_status ON match_requests(status);
CREATE INDEX idx_match_requests_expires_at ON match_requests(expires_at);
CREATE INDEX idx_match_requests_potential_match ON match_requests(potential_match_id);

-- Composite indexes for common queries
CREATE INDEX idx_match_requests_to_user_status ON match_requests(to_user_id, status);
CREATE INDEX idx_match_requests_from_user_status ON match_requests(from_user_id, status);
CREATE INDEX idx_match_requests_expires_status ON match_requests(expires_at, status);

-- Add comments for documentation
COMMENT ON TABLE match_requests IS 'Stores carpool requests between users based on potential matches';
COMMENT ON COLUMN match_requests.from_user_id IS 'User who sent the request';
COMMENT ON COLUMN match_requests.to_user_id IS 'User who received the request';
COMMENT ON COLUMN match_requests.potential_match_id IS 'The potential match that triggered this request';
COMMENT ON COLUMN match_requests.message IS 'Optional message from sender (currently not used in frontend)';
COMMENT ON COLUMN match_requests.status IS 'Request status: pending, accepted, rejected, expired';
COMMENT ON COLUMN match_requests.expires_at IS 'When the request expires (7 days from creation)';
