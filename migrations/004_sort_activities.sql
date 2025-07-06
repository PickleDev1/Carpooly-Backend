CREATE TABLE user_activity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_type VARCHAR(32) NOT NULL, -- e.g. 'carpool_created', 'invite_sent', 'invite_received', 'ride_completed'
    related_id UUID,                    -- ID of the related entity (carpool, invite, ride, etc.)
    related_type VARCHAR(32),           -- e.g. 'carpool', 'invite', 'ride'
    description TEXT,                   -- Human-readable summary
    data JSONB,                         -- Optional: raw data for the activity
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for fast lookup by user and time
CREATE INDEX idx_user_activity_user_time ON user_activity(user_id, timestamp DESC);

-- Index for filtering by activity type
CREATE INDEX idx_user_activity_type ON user_activity(activity_type);

-- Migration: Add 'activity_type' column to user_activity table if not present
ALTER TABLE user_activity ADD COLUMN IF NOT EXISTS activity_type VARCHAR(64);
-- Optionally, you can update existing rows to set a default value if needed