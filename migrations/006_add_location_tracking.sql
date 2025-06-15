-- Create location_tracking table
CREATE TABLE IF NOT EXISTS location_tracking (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    carpool_ride_id UUID NOT NULL REFERENCES carpool_rides(id) ON DELETE CASCADE,
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_location_tracking_user_id ON location_tracking(user_id);
CREATE INDEX IF NOT EXISTS idx_location_tracking_ride_id ON location_tracking(carpool_ride_id);
CREATE INDEX IF NOT EXISTS idx_location_tracking_timestamp ON location_tracking(timestamp);

-- Add location sharing settings and home location to users table
ALTER TABLE users
ADD COLUMN IF NOT EXISTS location_sharing_enabled BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS home_latitude DECIMAL(10, 8),
ADD COLUMN IF NOT EXISTS home_longitude DECIMAL(11, 8);

-- Add comment to explain the table
COMMENT ON TABLE location_tracking IS 'Stores basic location data for users during carpool rides';
COMMENT ON COLUMN location_tracking.latitude IS 'Latitude coordinate (-90 to 90)';
COMMENT ON COLUMN location_tracking.longitude IS 'Longitude coordinate (-180 to 180)';
COMMENT ON COLUMN location_tracking.timestamp IS 'When the location was recorded';
COMMENT ON COLUMN users.location_sharing_enabled IS 'Whether the user has enabled location sharing';
COMMENT ON COLUMN users.home_latitude IS 'User''s home location latitude';
COMMENT ON COLUMN users.home_longitude IS 'User''s home location longitude'; 