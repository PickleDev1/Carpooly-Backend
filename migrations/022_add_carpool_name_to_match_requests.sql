-- Add carpool_name column to match_requests table
-- This allows the sender to specify a name for the carpool when sending a match request

ALTER TABLE match_requests ADD COLUMN carpool_name VARCHAR(255);

-- Add comment for documentation
COMMENT ON COLUMN match_requests.carpool_name IS 'Name chosen by the sender for the carpool when the request is accepted';

-- Set default names for existing requests (optional - can be removed if not needed)
UPDATE match_requests 
SET carpool_name = 'Carpool ' || SUBSTRING(id::text, 1, 8)
WHERE carpool_name IS NULL;
