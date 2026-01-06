-- Quick Fix: Add preferred_carpool_size column to match_requests table
-- This will resolve the "column does not exist" error

-- Step 1: Add the column with default value
ALTER TABLE match_requests 
ADD COLUMN preferred_carpool_size INTEGER DEFAULT 4;

-- Step 2: Add constraint to ensure valid carpool sizes (2-8 people)
ALTER TABLE match_requests 
ADD CONSTRAINT check_preferred_carpool_size 
CHECK (preferred_carpool_size >= 2 AND preferred_carpool_size <= 8);

-- Step 3: Add comment for documentation
COMMENT ON COLUMN match_requests.preferred_carpool_size IS 'Preferred carpool size (2-8 people) specified by the user who sent the request';

-- Step 4: Create index for performance
CREATE INDEX IF NOT EXISTS idx_match_requests_preferred_carpool_size 
ON match_requests(preferred_carpool_size);

-- Step 5: Verify the column was added successfully
SELECT column_name, data_type, is_nullable, column_default 
FROM information_schema.columns 
WHERE table_name = 'match_requests' 
AND column_name = 'preferred_carpool_size';
