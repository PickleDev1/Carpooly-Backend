-- Migration 029: Add company_id and site_id columns to carpool_schedules
-- Phase 2: Database Schema - Add Columns
--
-- This migration adds nullable company_id and site_id columns to carpool_schedules table.
-- These columns are denormalized from carpools for query performance (avoids joins).
--
-- IMPORTANT: This migration is backward compatible:
-- - All existing rows get company_id = NULL (personal schedules)
-- - All existing rows get site_id = NULL
-- - Existing queries continue to work (just need to add company_id IS NULL filter in Phase 3)
--
-- NOTE: In Phase 7, we'll ensure company_id/site_id are copied from carpools when creating schedules.

-- Add company_id and site_id columns
-- These are nullable - NULL means personal schedule
-- These are denormalized from carpools for performance (can be derived via JOIN, but denormalized for speed)
ALTER TABLE carpool_schedules 
  ADD COLUMN company_id UUID REFERENCES companies(id),
  ADD COLUMN site_id UUID REFERENCES sites(id);

-- Create indexes for performance
-- Partial index for company-scoped queries (only when company_id IS NOT NULL)
CREATE INDEX idx_schedules_company ON carpool_schedules(company_id) WHERE company_id IS NOT NULL;

-- Add column comments for documentation
COMMENT ON COLUMN carpool_schedules.company_id IS 'Denormalized from carpools for query performance. NULL for personal schedules, UUID for company schedules';
COMMENT ON COLUMN carpool_schedules.site_id IS 'Denormalized from carpools for query performance. Optional site within company';

-- Verification: All existing rows should have company_id IS NULL
-- Run this after migration to verify:
-- SELECT COUNT(*) FROM carpool_schedules WHERE company_id IS NOT NULL;
-- Expected: 0 (all existing data is personal)
--
-- NOTE: After Phase 7, we'll populate these from carpools for existing schedules:
-- UPDATE carpool_schedules cs
-- SET company_id = c.company_id, site_id = c.site_id
-- FROM carpools c
-- WHERE cs.carpool_id = c.id AND cs.company_id IS NULL;

