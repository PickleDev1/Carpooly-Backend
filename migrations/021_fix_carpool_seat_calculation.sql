-- Migration 021: Fix carpool seat calculation
-- This migration recalculates available_seats based on actual member count
-- to fix data inconsistency issues

-- Fix existing data by recalculating available seats based on actual member count
UPDATE carpools 
SET available_seats = (
    SELECT seats - COUNT(*) 
    FROM carpool_members 
    WHERE carpool_id = carpools.id
);

-- Add comment for documentation
COMMENT ON COLUMN carpools.available_seats IS 'Available seats calculated as total_seats - actual_member_count';

-- Log the fix
DO $$
DECLARE
    fixed_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO fixed_count FROM carpools;
    RAISE NOTICE 'Fixed available_seats calculation for % carpools', fixed_count;
END $$;
