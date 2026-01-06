-- Migration 024: Create sites table for Company Spaces feature
-- Phase 1: Database Schema - New Tables
--
-- This migration creates the sites table which represents physical company locations
-- (offices, warehouses, campuses) where employees commute.
--
-- IMPORTANT: This is a NEW table - it does NOT affect any existing tables or data.
-- All existing functionality continues to work unchanged.

-- Create sites table
CREATE TABLE sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name TEXT NOT NULL,                              -- e.g., 'SJC14 Warehouse'
    code TEXT,                                       -- 'SJC14' (short identifier, unique per company)
    address TEXT,                                    -- Full address of the site
    latitude DOUBLE PRECISION,                       -- GPS latitude for destination
    longitude DOUBLE PRECISION,                      -- GPS longitude for destination
    timezone TEXT,                                   -- 'America/Los_Angeles' (IANA timezone)
    is_active BOOLEAN NOT NULL DEFAULT TRUE,        -- Soft delete flag
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Ensure code is unique per company (but code can be NULL)
    UNIQUE(company_id, code)
);

-- Create indexes for performance
-- Index for looking up sites by company
CREATE INDEX idx_sites_company ON sites(company_id);
-- Composite index for active sites per company (common query pattern)
CREATE INDEX idx_sites_company_active ON sites(company_id, is_active) WHERE is_active = TRUE;

-- Add table and column comments for documentation
COMMENT ON TABLE sites IS 'Represents physical company locations (offices, warehouses, campuses) where employees commute';
COMMENT ON COLUMN sites.code IS 'Short identifier unique per company (e.g., SJC14). NULL is allowed for sites without codes';
COMMENT ON COLUMN sites.latitude IS 'GPS latitude for the site location (used as destination for many carpools)';
COMMENT ON COLUMN sites.longitude IS 'GPS longitude for the site location (used as destination for many carpools)';
COMMENT ON COLUMN sites.timezone IS 'IANA timezone string (e.g., America/Los_Angeles) for the site location';
COMMENT ON COLUMN sites.is_active IS 'Soft delete flag - inactive sites are hidden but not deleted';

-- Note: The UNIQUE constraint on (company_id, code) allows:
-- - Multiple sites per company
-- - NULL codes (multiple sites can have NULL code)
-- - Unique codes per company (e.g., Company A can have "SJC14", Company B can also have "SJC14")

