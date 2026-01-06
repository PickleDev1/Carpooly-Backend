-- Migration 027: Add company_id and site_id columns to match_requests
-- Phase 2: Database Schema - Add Columns
--
-- This migration adds nullable company_id and site_id columns to match_requests table.
-- These columns allow match requests to be scoped to a company (coworkers-only).
--
-- IMPORTANT: This migration is backward compatible:
-- - All existing rows get company_id = NULL (personal requests)
-- - All existing rows get site_id = NULL
-- - Existing queries continue to work (just need to add company_id IS NULL filter in Phase 3)

-- Add company_id and site_id columns
-- These are nullable - NULL means personal match request
ALTER TABLE match_requests 
  ADD COLUMN company_id UUID REFERENCES companies(id),
  ADD COLUMN site_id UUID REFERENCES sites(id);

-- Create indexes for performance
-- Partial index for company-scoped queries (only when company_id IS NOT NULL)
CREATE INDEX idx_match_requests_company ON match_requests(company_id) WHERE company_id IS NOT NULL;

-- Composite index for company + site queries (common pattern for company matching)
CREATE INDEX idx_match_requests_company_site ON match_requests(company_id, site_id) WHERE company_id IS NOT NULL;

-- Add column comments for documentation
COMMENT ON COLUMN match_requests.company_id IS 'NULL for personal requests, UUID for company requests';
COMMENT ON COLUMN match_requests.site_id IS 'Optional site within company for site-specific matching';

-- Verification: All existing rows should have company_id IS NULL
-- Run this after migration to verify:
-- SELECT COUNT(*) FROM match_requests WHERE company_id IS NOT NULL;
-- Expected: 0 (all existing data is personal)

