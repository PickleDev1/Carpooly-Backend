-- Migration 030: Add company_id and site_id columns to carpool_rides
-- Phase 2: Database Schema - Add Columns
--
-- This migration adds nullable company_id and site_id columns to carpool_rides table.
-- These columns are denormalized from carpools for query performance (avoids joins).
--
-- IMPORTANT: This migration is backward compatible:
-- - All existing rows get company_id = NULL (personal rides)
-- - All existing rows get site_id = NULL
-- - Existing queries continue to work (just need to add company_id IS NULL filter in Phase 3)
--
-- NOTE: In Phase 7, we'll ensure company_id/site_id are copied from carpools when creating rides.

-- Add company_id and site_id columns
-- These are nullable - NULL means personal ride
-- These are denormalized from carpools for performance (can be derived via JOIN, but denormalized for speed)
ALTER TABLE carpool_rides 
  ADD COLUMN company_id UUID REFERENCES companies(id),
  ADD COLUMN site_id UUID REFERENCES sites(id);

-- Create indexes for performance
-- Composite index for company-scoped queries with time ordering (common for stats/analytics)
-- Partial index (only when company_id IS NOT NULL) for better performance
CREATE INDEX idx_rides_company ON carpool_rides(company_id, start_time) WHERE company_id IS NOT NULL;

-- Composite index for company + site queries with time ordering (common for site-specific stats)
CREATE INDEX idx_rides_company_site ON carpool_rides(company_id, site_id, start_time) WHERE company_id IS NOT NULL;

-- Add column comments for documentation
COMMENT ON COLUMN carpool_rides.company_id IS 'Denormalized from carpools for query performance. NULL for personal rides, UUID for company rides';
COMMENT ON COLUMN carpool_rides.site_id IS 'Denormalized from carpools for query performance. Optional site within company';

-- Verification: All existing rows should have company_id IS NULL
-- Run this after migration to verify:
-- SELECT COUNT(*) FROM carpool_rides WHERE company_id IS NOT NULL;
-- Expected: 0 (all existing data is personal)
--
-- NOTE: After Phase 7, we'll populate these from carpools for existing rides:
-- UPDATE carpool_rides cr
-- SET company_id = c.company_id, site_id = c.site_id
-- FROM carpools c
-- WHERE cr.carpool_id = c.id AND cr.company_id IS NULL;

