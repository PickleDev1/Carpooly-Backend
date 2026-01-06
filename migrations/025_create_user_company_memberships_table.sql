-- Migration 025: Create user_company_memberships table for Company Spaces feature
-- Phase 1: Database Schema - New Tables
--
-- This migration creates the user_company_memberships table which links users
-- to companies (and optionally sites) for the Company Spaces feature.
--
-- IMPORTANT: This is a NEW table - it does NOT affect any existing tables or data.
-- All existing functionality continues to work unchanged.

-- Create user_company_memberships table
CREATE TABLE user_company_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    site_id UUID REFERENCES sites(id),              -- Optional; user picks later via PUT /api/me/company-site
    role TEXT NOT NULL DEFAULT 'employee' CHECK (role IN ('employee', 'site_admin', 'company_admin')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'pending', 'invited', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- One membership per user per company
    UNIQUE(user_id, company_id)
);

-- Create indexes for performance
-- Index for looking up user's memberships (common query: "What companies is this user in?")
CREATE INDEX idx_memberships_user ON user_company_memberships(user_id, status);
-- Index for looking up company members (common query: "Who is in this company?")
CREATE INDEX idx_memberships_company ON user_company_memberships(company_id, status);
-- Partial index for site-based queries (only when site_id is set)
CREATE INDEX idx_memberships_site ON user_company_memberships(site_id) WHERE site_id IS NOT NULL;
-- Composite index for active memberships per company (for stats queries)
CREATE INDEX idx_memberships_company_active ON user_company_memberships(company_id, status) WHERE status = 'active';

-- Add table and column comments for documentation
COMMENT ON TABLE user_company_memberships IS 'Links users to companies and optionally sites. Enables multi-tenancy for Company Spaces feature';
COMMENT ON COLUMN user_company_memberships.site_id IS 'Optional site selection. User can pick site later via PUT /api/me/company-site. NULL means user has not selected a site yet';
COMMENT ON COLUMN user_company_memberships.role IS 'Role hierarchy: employee < site_admin < company_admin. Controls access to admin APIs';
COMMENT ON COLUMN user_company_memberships.status IS 'Membership status: active (can use features), pending (awaiting approval), invited (invited but not accepted), inactive (disabled)';

-- Role Hierarchy Explanation:
-- - employee: Can use company matching, create requests/carpools in company scope
-- - site_admin: Can view stats for their assigned site, manage users at their site
-- - company_admin: Can view all company stats, manage all sites, manage all users in company

-- Status Values Explanation:
-- - active: User is an active member and can use company features
-- - pending: User has requested to join, awaiting approval
-- - invited: User has been invited to join, awaiting acceptance
-- - inactive: Membership is disabled (soft delete)

