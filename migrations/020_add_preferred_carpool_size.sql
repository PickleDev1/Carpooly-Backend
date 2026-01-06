-- Migration 020: Add preferred_carpool_size to match_requests table
-- This migration adds the preferred carpool size field to match requests
-- to support dynamic seat management when creating carpools from matches

-- Add preferred_carpool_size column to match_requests table
ALTER TABLE match_requests 
ADD COLUMN preferred_carpool_size INTEGER DEFAULT 4;

-- Add constraint to ensure valid carpool sizes (2-8 people)
ALTER TABLE match_requests 
ADD CONSTRAINT check_preferred_carpool_size 
CHECK (preferred_carpool_size >= 2 AND preferred_carpool_size <= 8);

-- Add comment for documentation
COMMENT ON COLUMN match_requests.preferred_carpool_size IS 'Preferred carpool size (2-8 people) specified by the user who sent the request';

-- Create index for performance on the new column
CREATE INDEX IF NOT EXISTS idx_match_requests_preferred_carpool_size 
ON match_requests(preferred_carpool_size);
