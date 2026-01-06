-- Migration 028: Add company_id and site_id columns to carpools
-- Phase 2: Database Schema - Add Columns
--
-- This migration adds nullable company_id and site_id columns to carpools table.
-- These columns allow carpools to be scoped to a company (coworkers-only carpools).
--
-- IMPORTANT: This migration is backward compatible:
-- - All existing rows get company_id = NULL (personal carpools)
-- - All existing rows get site_id = NULL
-- - Existing queries continue to work (just need to add company_id IS NULL filter in Phase 3)

-- Add company_id and site_id columns
-- These are nullable - NULL means personal carpool
ALTER TABLE carpools 
  ADD COLUMN company_id UUID REFERENCES companies(id),
  ADD COLUMN site_id UUID REFERENCES sites(id);

-- Create indexes for performance
-- Partial index for company-scoped queries (only when company_id IS NOT NULL)
CREATE INDEX idx_carpools_company ON carpools(company_id) WHERE company_id IS NOT NULL;

-- Composite index for company + site queries (common pattern for company carpools)
CREATE INDEX idx_carpools_company_site ON carpools(company_id, site_id) WHERE company_id IS NOT NULL;

-- Add column comments for documentation
COMMENT ON COLUMN carpools.company_id IS 'NULL for personal carpools, UUID for company carpools';
COMMENT ON COLUMN carpools.site_id IS 'Optional site within company for site-specific carpools';

-- Verification: All existing rows should have company_id IS NULL
-- Run this after migration to verify:
-- SELECT COUNT(*) FROM carpools WHERE company_id IS NOT NULL;
-- Expected: 0 (all existing data is personal)

