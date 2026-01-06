-- Migration 023: Create companies table for Company Spaces feature
-- Phase 1: Database Schema - New Tables
-- 
-- This migration creates the companies table which stores company/organization
-- information for the optional Company Spaces feature.
--
-- IMPORTANT: This is a NEW table - it does NOT affect any existing tables or data.
-- All existing functionality continues to work unchanged.

-- Create companies table
CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,                              -- e.g., 'Amazon'
    slug TEXT UNIQUE NOT NULL,                       -- 'amazon' (URL-safe, lowercase)
    primary_domain TEXT NOT NULL,                    -- 'amazon.com'
    additional_domains JSONB DEFAULT '[]'::jsonb,   -- ['amazon.co.uk', 'amzn.com']
    logo_url TEXT,                                   -- Optional company logo URL
    settings JSONB DEFAULT '{}'::jsonb,             -- Extensible config (see below)
    is_active BOOLEAN NOT NULL DEFAULT TRUE,         -- Soft delete flag
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_companies_slug ON companies(slug);
CREATE INDEX idx_companies_primary_domain ON companies(primary_domain);
-- GIN index for JSONB array searches (additional_domains)
CREATE INDEX idx_companies_domains ON companies USING GIN(additional_domains);
-- Partial index for active companies only
CREATE INDEX idx_companies_active ON companies(is_active) WHERE is_active = TRUE;

-- Add table and column comments for documentation
COMMENT ON TABLE companies IS 'Stores company/organization information for company spaces feature';
COMMENT ON COLUMN companies.slug IS 'URL-safe identifier (lowercase, alphanumeric + hyphens), unique across all companies';
COMMENT ON COLUMN companies.primary_domain IS 'Primary email domain for auto-detection (e.g., amazon.com)';
COMMENT ON COLUMN companies.additional_domains IS 'JSONB array of additional email domains (e.g., ["amazon.co.uk", "amzn.com"])';
COMMENT ON COLUMN companies.settings IS 'JSONB with extensible config: require_invite_code (bool), allow_cross_site_matching (bool), default_timezone (string), estimated_avg_commute_distance_km (number)';
COMMENT ON COLUMN companies.is_active IS 'Soft delete flag - inactive companies are hidden but not deleted';

-- Example settings JSONB structure:
-- {
--   "require_invite_code": false,
--   "allow_cross_site_matching": true,
--   "default_timezone": "America/Los_Angeles",
--   "estimated_avg_commute_distance_km": 15.5
-- }

