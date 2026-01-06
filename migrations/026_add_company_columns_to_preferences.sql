-- Migration 026: Add company_id and site_id columns to user_matching_preferences
-- Phase 2: Database Schema - Add Columns
--
-- This migration:
-- 1. Adds a surrogate ID column as the new primary key (allows multiple preference records per user)
-- 2. Adds company_id and site_id columns for company-specific preferences
-- 3. Creates unique constraint to prevent duplicate (user_id, company_id) combinations
--
-- IMPORTANT: This migration is backward compatible:
-- - All existing rows get id = gen_random_uuid() (new primary key)
-- - All existing rows get company_id = NULL (personal preferences)
-- - All existing rows get site_id = NULL
-- - Existing queries continue to work (just need to add company_id IS NULL filter in Phase 3)

-- Step 1: Add surrogate ID column (new primary key)
-- This allows multiple preference records per user (personal + per-company)
ALTER TABLE user_matching_preferences 
  ADD COLUMN id UUID DEFAULT gen_random_uuid();

-- Step 2: Populate ID for existing rows
-- Ensure all existing rows have a UUID
UPDATE user_matching_preferences SET id = gen_random_uuid() WHERE id IS NULL;

-- Step 3: Make ID NOT NULL and set as PRIMARY KEY
-- First ensure no NULLs exist
ALTER TABLE user_matching_preferences 
  ALTER COLUMN id SET NOT NULL;

-- Step 4: Drop old primary key constraint (user_id was the old PK)
-- This allows multiple rows per user (one personal + multiple company-specific)
ALTER TABLE user_matching_preferences 
  DROP CONSTRAINT IF EXISTS user_matching_preferences_pkey;

-- Step 5: Create new primary key on id
ALTER TABLE user_matching_preferences 
  ADD PRIMARY KEY (id);

-- Step 6: Add company_id and site_id columns
-- These are nullable - NULL means personal preferences
ALTER TABLE user_matching_preferences 
  ADD COLUMN company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
  ADD COLUMN site_id UUID REFERENCES sites(id);

-- Step 7: Create unique constraint to prevent duplicate (user_id, company_id) combinations
-- This handles NULL company_id for personal preferences using COALESCE
-- Personal preferences: company_id IS NULL → COALESCE returns sentinel UUID
-- Company preferences: company_id IS NOT NULL → COALESCE returns actual UUID
-- Result: One personal preference per user, one company preference per user per company
CREATE UNIQUE INDEX idx_preferences_user_company 
  ON user_matching_preferences(user_id, COALESCE(company_id, '00000000-0000-0000-0000-000000000000'::uuid));

-- Step 8: Create indexes for performance
-- Index for company-scoped queries (only when company_id IS NOT NULL)
CREATE INDEX idx_preferences_company ON user_matching_preferences(company_id) WHERE company_id IS NOT NULL;

-- Index for user-scoped queries (backward compatibility - existing queries use user_id)
CREATE INDEX idx_preferences_user ON user_matching_preferences(user_id);

-- Add column comments for documentation
COMMENT ON COLUMN user_matching_preferences.id IS 'Surrogate primary key - allows multiple preference records per user (personal + per-company)';
COMMENT ON COLUMN user_matching_preferences.company_id IS 'NULL for personal preferences, UUID for company-specific preferences';
COMMENT ON COLUMN user_matching_preferences.site_id IS 'Optional site within company';

-- Verification: All existing rows should have company_id IS NULL
-- Run this after migration to verify:
-- SELECT COUNT(*) FROM user_matching_preferences WHERE company_id IS NOT NULL;
-- Expected: 0 (all existing data is personal)

