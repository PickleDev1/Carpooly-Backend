# Company Spaces Feature - Implementation Plan

**Status:** ✅ Approved by Frontend Team (with fixes applied)  
**Date:** 2025-01-XX  
**Based on:** `COMPANY_SPACES_BLUEPRINT.md`  
**Review:** All frontend review issues addressed

---

## 📋 Frontend Review Fixes Applied

All issues identified in the frontend review have been addressed:

- ✅ **Issue #1:** Error response format specification added (HTTPError struct, proper status codes)
- ✅ **Issue #2:** Complete CompanyRepository interface with all methods (GetCompanyByID, GetCompanyBySlug, GetCompanyByDomain, GetUserMemberships, CreateMembership, UpdateMembershipSite, GetCompanySites, GetSite)
- ✅ **Issue #3:** Site repository methods added (GetCompanySites, GetSite)
- ✅ **Issue #4:** Upsert conflict clause clarified (use `ON CONFLICT ON CONSTRAINT idx_preferences_user_company`)
- ✅ **Issue #5:** Validation logic for company requests added (sender/recipient membership validation, site validation)
- ✅ **Issue #6:** Carpool access validation helper added (validateCarpoolAccess function)
- ✅ **Issue #7:** Site validation in site selection added (validates site belongs to company)
- ✅ **Issue #8:** Role validation helper created (ValidateAdminRole, ValidateSiteAdminRole)
- ✅ **Issue #9:** Test data setup section added (test_data/company_test_data.sql)
- ✅ **Issue #10:** Performance considerations added (index usage verification, EXPLAIN ANALYZE)

**See individual phases for detailed implementations.**

---

## Table of Contents

1. [Overview](#overview)
2. [Phase 1: Database Schema - New Tables](#phase-1-database-schema---new-tables)
3. [Phase 2: Database Schema - Add Columns](#phase-2-database-schema---add-columns)
4. [Phase 3: Query Filters (CRITICAL)](#phase-3-query-filters-critical)
5. [Phase 4: Scope Resolution & Handlers](#phase-4-scope-resolution--handlers)
6. [Phase 5: Membership Detection](#phase-5-membership-detection)
6. [Phase 6: Company APIs](#phase-6-company-apis)
7. [Phase 7: Tagging Propagation](#phase-7-tagging-propagation)
8. [Phase 8: Admin Analytics](#phase-8-admin-analytics)
9. [Testing Strategy](#testing-strategy)
10. [Rollback Procedures](#rollback-procedures)

---

## Overview

### Implementation Principles

1. **Backward Compatibility First** - Existing functionality must never break
2. **Incremental Deployment** - Each phase is independently testable
3. **Safety Gates** - Cannot proceed to next phase until current phase is verified
4. **Data Isolation** - Personal and company data must never mix

### Critical Path

```
Phase 1 (New Tables) 
  → Phase 2 (Add Columns) 
  → Phase 3 (Query Filters) ⚠️ MANDATORY
  → Phase 4 (Scope Resolution)
  → Phase 5 (Membership)
  → Phase 6 (Company APIs)
  → Phase 7 (Tagging)
  → Phase 8 (Analytics)
```

**⚠️ DO NOT skip Phase 3 or proceed to Phase 4+ until Phase 3 is complete and verified.**

---

## Phase 1: Database Schema - New Tables

**Goal:** Create new tables for companies, sites, and memberships  
**Risk:** Low ✅  
**Duration:** 1-2 days  
**Blocks:** Nothing (can be done independently)

### Tasks

#### 1.1 Create Migration File

**File:** `migrations/023_create_companies_table.sql`

```sql
-- Create companies table
CREATE TABLE companies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  slug TEXT UNIQUE NOT NULL,
  primary_domain TEXT NOT NULL,
  additional_domains JSONB DEFAULT '[]'::jsonb,
  logo_url TEXT,
  settings JSONB NOT NULL DEFAULT '{}'::jsonb,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_companies_slug ON companies(slug);
CREATE INDEX idx_companies_domains ON companies USING GIN(additional_domains);
CREATE INDEX idx_companies_active ON companies(is_active) WHERE is_active = TRUE;

COMMENT ON TABLE companies IS 'Stores company/organization information for company spaces feature';
COMMENT ON COLUMN companies.slug IS 'URL-safe identifier (lowercase, alphanumeric + hyphens)';
COMMENT ON COLUMN companies.settings IS 'JSONB with require_invite_code, allow_cross_site_matching, default_timezone, etc.';
```

#### 1.2 Create Sites Table

**File:** `migrations/024_create_sites_table.sql`

```sql
-- Create sites table
CREATE TABLE sites (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  code TEXT,
  address TEXT,
  latitude DOUBLE PRECISION,
  longitude DOUBLE PRECISION,
  timezone TEXT,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(company_id, code)
);

CREATE INDEX idx_sites_company ON sites(company_id);
CREATE INDEX idx_sites_company_active ON sites(company_id, is_active) WHERE is_active = TRUE;

COMMENT ON TABLE sites IS 'Represents physical company locations (offices, warehouses, campuses)';
COMMENT ON COLUMN sites.code IS 'Short identifier unique per company (e.g., SJC14)';
```

#### 1.3 Create User Company Memberships Table

**File:** `migrations/025_create_user_company_memberships.sql`

```sql
-- Create user_company_memberships table
CREATE TABLE user_company_memberships (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  site_id UUID REFERENCES sites(id),
  role TEXT NOT NULL DEFAULT 'employee' CHECK (role IN ('employee', 'site_admin', 'company_admin')),
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'pending', 'invited', 'inactive')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(user_id, company_id)
);

CREATE INDEX idx_memberships_user ON user_company_memberships(user_id, status);
CREATE INDEX idx_memberships_company ON user_company_memberships(company_id, status);
CREATE INDEX idx_memberships_site ON user_company_memberships(site_id) WHERE site_id IS NOT NULL;

COMMENT ON TABLE user_company_memberships IS 'Links users to companies and optionally sites';
COMMENT ON COLUMN user_company_memberships.role IS 'employee | site_admin | company_admin';
COMMENT ON COLUMN user_company_memberships.status IS 'active | pending | invited | inactive';
```

### Verification

- [ ] Run migrations successfully
- [ ] Verify tables exist: `SELECT table_name FROM information_schema.tables WHERE table_name IN ('companies', 'sites', 'user_company_memberships');`
- [ ] Verify indexes created
- [ ] Verify constraints work (try inserting invalid data)
- [ ] No existing functionality affected (run existing tests)

### Rollback

```sql
-- If needed, rollback Phase 1
DROP TABLE IF EXISTS user_company_memberships CASCADE;
DROP TABLE IF EXISTS sites CASCADE;
DROP TABLE IF EXISTS companies CASCADE;
```

---

## Phase 2: Database Schema - Add Columns

**Goal:** Add nullable `company_id` and `site_id` columns to existing tables  
**Risk:** Low ✅  
**Duration:** 1 day  
**Blocks:** Phase 3 (must complete before query filters)

### Tasks

#### 2.1 Update `user_matching_preferences` Table

**File:** `migrations/026_add_company_columns_to_preferences.sql`

```sql
-- Step 1: Add surrogate ID column (new primary key)
ALTER TABLE user_matching_preferences 
  ADD COLUMN id UUID DEFAULT gen_random_uuid();

-- Step 2: Populate ID for existing rows
UPDATE user_matching_preferences SET id = gen_random_uuid() WHERE id IS NULL;

-- Step 3: Make ID NOT NULL and set as PRIMARY KEY
ALTER TABLE user_matching_preferences 
  ALTER COLUMN id SET NOT NULL;

-- Step 4: Drop old primary key constraint
ALTER TABLE user_matching_preferences 
  DROP CONSTRAINT user_matching_preferences_pkey;

-- Step 5: Create new primary key on id
ALTER TABLE user_matching_preferences 
  ADD PRIMARY KEY (id);

-- Step 6: Add company_id and site_id columns
ALTER TABLE user_matching_preferences 
  ADD COLUMN company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
  ADD COLUMN site_id UUID REFERENCES sites(id);

-- Step 7: Create unique constraint to prevent duplicate (user_id, company_id) combinations
CREATE UNIQUE INDEX idx_preferences_user_company 
  ON user_matching_preferences(user_id, COALESCE(company_id, '00000000-0000-0000-0000-000000000000'::uuid));

CREATE INDEX idx_preferences_company ON user_matching_preferences(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_preferences_user ON user_matching_preferences(user_id); -- For backward compatibility queries

COMMENT ON COLUMN user_matching_preferences.company_id IS 'NULL for personal preferences, UUID for company-specific preferences';
COMMENT ON COLUMN user_matching_preferences.site_id IS 'Optional site within company';
```

#### 2.2 Update `match_requests` Table

**File:** `migrations/027_add_company_columns_to_match_requests.sql`

```sql
ALTER TABLE match_requests 
  ADD COLUMN company_id UUID REFERENCES companies(id),
  ADD COLUMN site_id UUID REFERENCES sites(id);

CREATE INDEX idx_match_requests_company ON match_requests(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_match_requests_company_site ON match_requests(company_id, site_id) WHERE company_id IS NOT NULL;

COMMENT ON COLUMN match_requests.company_id IS 'NULL for personal requests, UUID for company requests';
```

#### 2.3 Update `carpools` Table

**File:** `migrations/028_add_company_columns_to_carpools.sql`

```sql
ALTER TABLE carpools 
  ADD COLUMN company_id UUID REFERENCES companies(id),
  ADD COLUMN site_id UUID REFERENCES sites(id);

CREATE INDEX idx_carpools_company ON carpools(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_carpools_company_site ON carpools(company_id, site_id) WHERE company_id IS NOT NULL;

COMMENT ON COLUMN carpools.company_id IS 'NULL for personal carpools, UUID for company carpools';
```

#### 2.4 Update `carpool_schedules` Table

**File:** `migrations/029_add_company_columns_to_schedules.sql`

```sql
ALTER TABLE carpool_schedules 
  ADD COLUMN company_id UUID REFERENCES companies(id),
  ADD COLUMN site_id UUID REFERENCES sites(id);

CREATE INDEX idx_schedules_company ON carpool_schedules(company_id) WHERE company_id IS NOT NULL;

COMMENT ON COLUMN carpool_schedules.company_id IS 'Denormalized from carpools for query performance';
```

#### 2.5 Update `carpool_rides` Table

**File:** `migrations/030_add_company_columns_to_rides.sql`

```sql
ALTER TABLE carpool_rides 
  ADD COLUMN company_id UUID REFERENCES companies(id),
  ADD COLUMN site_id UUID REFERENCES sites(id);

CREATE INDEX idx_rides_company ON carpool_rides(company_id, start_time) WHERE company_id IS NOT NULL;
CREATE INDEX idx_rides_company_site ON carpool_rides(company_id, site_id, start_time) WHERE company_id IS NOT NULL;

COMMENT ON COLUMN carpool_rides.company_id IS 'Denormalized from carpools for query performance';
```

### Verification

- [ ] Run all migrations successfully
- [ ] Verify columns exist: Check each table has `company_id` and `site_id` columns
- [ ] Verify all existing rows have `company_id IS NULL` (personal data)
- [ ] Verify indexes created
- [ ] Run existing tests - all should pass (no behavior change yet)
- [ ] Verify `user_matching_preferences` can have multiple rows per user (test insert)

**Data Migration Verification Script:**

```sql
-- Verify all existing data has company_id IS NULL
SELECT 
    'user_matching_preferences' as table_name,
    COUNT(*) as total_rows,
    COUNT(*) FILTER (WHERE company_id IS NOT NULL) as company_rows
FROM user_matching_preferences
UNION ALL
SELECT 'match_requests', COUNT(*), COUNT(*) FILTER (WHERE company_id IS NOT NULL) FROM match_requests
UNION ALL
SELECT 'carpools', COUNT(*), COUNT(*) FILTER (WHERE company_id IS NOT NULL) FROM carpools
UNION ALL
SELECT 'carpool_schedules', COUNT(*), COUNT(*) FILTER (WHERE company_id IS NOT NULL) FROM carpool_schedules
UNION ALL
SELECT 'carpool_rides', COUNT(*), COUNT(*) FILTER (WHERE company_id IS NOT NULL) FROM carpool_rides;

-- All company_rows should be 0 (no company data yet)
```

### Rollback

```sql
-- Rollback Phase 2 (in reverse order)
ALTER TABLE carpool_rides DROP COLUMN IF EXISTS company_id, DROP COLUMN IF EXISTS site_id;
ALTER TABLE carpool_schedules DROP COLUMN IF EXISTS company_id, DROP COLUMN IF EXISTS site_id;
ALTER TABLE carpools DROP COLUMN IF EXISTS company_id, DROP COLUMN IF EXISTS site_id;
ALTER TABLE match_requests DROP COLUMN IF EXISTS company_id, DROP COLUMN IF EXISTS site_id;

-- Rollback user_matching_preferences (restore original PK)
ALTER TABLE user_matching_preferences DROP CONSTRAINT IF EXISTS user_matching_preferences_pkey;
ALTER TABLE user_matching_preferences DROP COLUMN IF EXISTS id;
ALTER TABLE user_matching_preferences ADD PRIMARY KEY (user_id);
ALTER TABLE user_matching_preferences DROP COLUMN IF EXISTS company_id, DROP COLUMN IF EXISTS site_id;
DROP INDEX IF EXISTS idx_preferences_user_company;
DROP INDEX IF EXISTS idx_preferences_company;
DROP INDEX IF EXISTS idx_preferences_user;
```

---

## Phase 3: Query Filters (CRITICAL)

**Goal:** Add `company_id` filters to ALL existing queries to prevent data leaks  
**Risk:** Medium ⚠️  
**Duration:** 3-5 days  
**Blocks:** Phase 4+ (MUST complete before enabling company features)

**⚠️ THIS PHASE IS MANDATORY - DO NOT SKIP OR PROCEED TO PHASE 4+ UNTIL COMPLETE**

### Phase 3a: Update `user_matching_preferences` Queries

#### Tasks

**File:** `pkg/repository/matching_repository.go`

**3a.1 Update `GetUserMatchingPreferences()` Method**

```go
// BEFORE
func (r *MatchingRepository) GetUserMatchingPreferences(ctx context.Context, userID string) (*models.UserMatchingPreferences, error) {
    query := `
        SELECT user_id, max_detour_minutes, ...
        FROM user_matching_preferences 
        WHERE user_id = $1
    `
    // ...
}

// AFTER
func (r *MatchingRepository) GetUserMatchingPreferences(ctx context.Context, userID string, companyID *uuid.UUID) (*models.UserMatchingPreferences, error) {
    query := `
        SELECT user_id, max_detour_minutes, preferred_group_size, driver_preference,
            schedule_flexibility_minutes, max_pickup_distance_miles, min_compatibility_score,
            notification_preferences, user_demographics, demographic_preferences, 
            destination_latitude, destination_longitude, arrival_time, commute_days,
            company_id, site_id, is_active, created_at, updated_at
        FROM user_matching_preferences 
        WHERE user_id = $1 
          AND (company_id = $2 OR ($2 IS NULL AND company_id IS NULL))
    `
    
    var prefs models.UserMatchingPreferences
    var companyIDVal sql.NullString
    var siteIDVal sql.NullString
    // ... existing scan logic ...
    err := r.db.QueryRowContext(ctx, query, userID, companyID).Scan(
        &prefs.UserID, &prefs.MaxDetourMinutes, ..., &companyIDVal, &siteIDVal, ...
    )
    // Handle company_id and site_id in response
    // ...
}
```

**3a.2 Update `UpsertUserMatchingPreferences()` Method**

```go
// BEFORE
func (r *MatchingRepository) UpsertUserMatchingPreferences(ctx context.Context, prefs *models.UserMatchingPreferences) error {
    query := `
        INSERT INTO user_matching_preferences (...)
        VALUES (...)
        ON CONFLICT (user_id) DO UPDATE SET ...
    `
}

// AFTER
func (r *MatchingRepository) UpsertUserMatchingPreferences(ctx context.Context, prefs *models.UserMatchingPreferences, companyID *uuid.UUID) error {
    // Use the unique index name for conflict resolution
    // This is more reliable than using COALESCE in conflict target
    if companyID == nil {
        // Personal preferences - company_id IS NULL
        query := `
            INSERT INTO user_matching_preferences (
                user_id, company_id, site_id, max_detour_minutes, preferred_group_size, 
                driver_preference, schedule_flexibility_minutes, max_pickup_distance_miles, 
                min_compatibility_score, notification_preferences, user_demographics, 
                demographic_preferences, destination_latitude, destination_longitude, 
                arrival_time, commute_days, is_active, updated_at
            ) VALUES ($1, NULL, NULL, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, CURRENT_TIMESTAMP)
            ON CONFLICT ON CONSTRAINT idx_preferences_user_company
            DO UPDATE SET
                max_detour_minutes = EXCLUDED.max_detour_minutes,
                preferred_group_size = EXCLUDED.preferred_group_size,
                driver_preference = EXCLUDED.driver_preference,
                schedule_flexibility_minutes = EXCLUDED.schedule_flexibility_minutes,
                max_pickup_distance_miles = EXCLUDED.max_pickup_distance_miles,
                min_compatibility_score = EXCLUDED.min_compatibility_score,
                notification_preferences = EXCLUDED.notification_preferences,
                user_demographics = EXCLUDED.user_demographics,
                demographic_preferences = EXCLUDED.demographic_preferences,
                destination_latitude = EXCLUDED.destination_latitude,
                destination_longitude = EXCLUDED.destination_longitude,
                arrival_time = EXCLUDED.arrival_time,
                commute_days = EXCLUDED.commute_days,
                is_active = EXCLUDED.is_active,
                updated_at = CURRENT_TIMESTAMP
        `
    } else {
        // Company preferences - company_id IS NOT NULL
        query := `
            INSERT INTO user_matching_preferences (
                user_id, company_id, site_id, max_detour_minutes, preferred_group_size, 
                driver_preference, schedule_flexibility_minutes, max_pickup_distance_miles, 
                min_compatibility_score, notification_preferences, user_demographics, 
                demographic_preferences, destination_latitude, destination_longitude, 
                arrival_time, commute_days, is_active, updated_at
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, CURRENT_TIMESTAMP)
            ON CONFLICT ON CONSTRAINT idx_preferences_user_company
            DO UPDATE SET
                max_detour_minutes = EXCLUDED.max_detour_minutes,
                preferred_group_size = EXCLUDED.preferred_group_size,
                driver_preference = EXCLUDED.driver_preference,
                schedule_flexibility_minutes = EXCLUDED.schedule_flexibility_minutes,
                max_pickup_distance_miles = EXCLUDED.max_pickup_distance_miles,
                min_compatibility_score = EXCLUDED.min_compatibility_score,
                notification_preferences = EXCLUDED.notification_preferences,
                user_demographics = EXCLUDED.user_demographics,
                demographic_preferences = EXCLUDED.demographic_preferences,
                destination_latitude = EXCLUDED.destination_latitude,
                destination_longitude = EXCLUDED.destination_longitude,
                arrival_time = EXCLUDED.arrival_time,
                commute_days = EXCLUDED.commute_days,
                is_active = EXCLUDED.is_active,
                updated_at = CURRENT_TIMESTAMP
        `
    }
    // Execute with appropriate parameters
    // ...
}
```

**3a.3 Update Handler to Pass `companyID`**

**File:** `pkg/handlers/matching_handlers.go`

```go
// Update GetMatchingPreferences handler
func (h *MatchingHandler) GetMatchingPreferences(w http.ResponseWriter, r *http.Request) {
    // ... existing auth code ...
    
    // Default to personal scope (nil companyID)
    var companyID *uuid.UUID = nil
    
    // For now, always use personal scope (company scope comes in Phase 4)
    prefs, err := h.matchingRepo.GetUserMatchingPreferences(r.Context(), userID, companyID)
    // ... rest of handler ...
}
```

#### Verification

- [ ] All existing tests pass
- [ ] Personal preferences still accessible (test with `companyID = nil`)
- [ ] No data leaks (verify query only returns `company_id IS NULL` when `companyID = nil`)
- [ ] Can insert personal preferences
- [ ] Can insert company preferences (test with test company)
- [ ] Unique constraint prevents duplicates

#### Rollback

- Revert method signatures to original
- Revert queries to original (remove `company_id` filter)
- Revert handler changes

---

### Phase 3b: Update `match_requests` Queries

#### Tasks

**File:** `pkg/repository/matching_repository.go`

**3b.1 Update `GetMatchRequests()` Method**

```go
// BEFORE
func (r *MatchingRepository) GetMatchRequests(ctx context.Context, userID string) ([]*models.MatchRequest, []*models.MatchRequest, error) {
    incomingQuery := `
        SELECT ... 
        FROM match_requests mr
        WHERE mr.to_user_id = $1 AND mr.status != 'expired'
    `
    // ...
}

// AFTER
func (r *MatchingRepository) GetMatchRequests(ctx context.Context, userID string, companyID *uuid.UUID) ([]*models.MatchRequest, []*models.MatchRequest, error) {
    incomingQuery := `
        SELECT mr.id, mr.from_user_id, mr.to_user_id, mr.potential_match_id, 
               mr.message, mr.preferred_carpool_size, mr.carpool_name, mr.status, 
               mr.company_id, mr.site_id, mr.created_at, mr.updated_at, mr.expires_at,
               u.name, u.display_name, u.home_latitude, u.home_longitude
        FROM match_requests mr
        JOIN users u ON mr.from_user_id = u.id
        WHERE mr.to_user_id = $1 
          AND mr.status != 'expired'
          AND (mr.company_id = $2 OR ($2 IS NULL AND mr.company_id IS NULL))
        ORDER BY mr.created_at DESC
    `
    
    outgoingQuery := `
        SELECT mr.id, mr.from_user_id, mr.to_user_id, mr.potential_match_id, 
               mr.message, mr.preferred_carpool_size, mr.carpool_name, mr.status,
               mr.company_id, mr.site_id, mr.created_at, mr.updated_at, mr.expires_at,
               u.name, u.display_name, u.home_latitude, u.home_longitude
        FROM match_requests mr
        JOIN users u ON mr.to_user_id = u.id
        WHERE mr.from_user_id = $1 
          AND mr.status != 'expired'
          AND (mr.company_id = $2 OR ($2 IS NULL AND mr.company_id IS NULL))
        ORDER BY mr.created_at DESC
    `
    // Update Scan to include company_id and site_id
    // ...
}
```

**3b.2 Update `GetMatchRequestsByUserID()` Method**

```go
// Similar updates - add companyID parameter and filter
func (r *MatchingRepository) GetMatchRequestsByUserID(ctx context.Context, userID string, companyID *uuid.UUID) (*models.MatchRequestsResponse, error) {
    // Add company_id filter to both incoming and outgoing queries
    // ...
}
```

**3b.3 Update `CreateMatchRequest()` Method**

```go
// Add company_id and site_id to INSERT
func (r *MatchingRepository) CreateMatchRequest(ctx context.Context, request *models.MatchRequest, companyID *uuid.UUID, siteID *uuid.UUID) error {
    query := `
        INSERT INTO match_requests (
            from_user_id, to_user_id, potential_match_id, message, 
            preferred_carpool_size, carpool_name, status, expires_at,
            company_id, site_id
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        RETURNING id, created_at, updated_at
    `
    // ...
}
```

**3b.4 Update Handlers**

**File:** `pkg/handlers/matching_handlers.go`

```go
// Update GetMatchRequests handler
func (h *MatchingHandler) GetMatchRequests(w http.ResponseWriter, r *http.Request) {
    // ... existing auth code ...
    
    // Default to personal scope
    var companyID *uuid.UUID = nil
    
    incoming, outgoing, err := h.matchingRepo.GetMatchRequests(r.Context(), userID, companyID)
    // ... rest of handler ...
}

// Update CreateMatchRequest handler
func (h *MatchingHandler) CreateMatchRequest(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...
    
    // For now, always use personal scope (nil companyID)
    var companyID *uuid.UUID = nil
    var siteID *uuid.UUID = nil
    
    err = h.matchingRepo.CreateMatchRequest(r.Context(), matchRequest, companyID, siteID)
    // ... rest of handler ...
}
```

#### Verification

- [ ] All existing tests pass
- [ ] Personal requests still accessible
- [ ] No data leaks (only returns `company_id IS NULL` requests)
- [ ] Can create personal requests
- [ ] Response includes `company_id` and `site_id` fields (both null for personal)

#### Rollback

- Revert method signatures
- Revert queries
- Revert handler changes

---

### Phase 3c: Update `carpools` Queries

#### Tasks

**File:** `pkg/repository/carpool_repository.go`

**3c.1 Update `GetUserCarpools()` Method**

```go
// BEFORE
func (r *CarPoolRepository) GetUserCarpools(ctx context.Context, userID uuid.UUID) ([]models.Carpool, error) {
    query := `
        SELECT DISTINCT ... 
        FROM carpools c
        JOIN carpool_members cm ON c.id = cm.carpool_id
        WHERE cm.user_id = $1
    `
}

// AFTER
func (r *CarPoolRepository) GetUserCarpools(ctx context.Context, userID uuid.UUID, companyID *uuid.UUID) ([]models.Carpool, error) {
    query := `
        SELECT DISTINCT 
            c.id, c.creator_id, c.carpool_name, c.status, c.recurring_option,
            c.destination_address, c.seats, c.company_id, c.site_id,
            c.created_at, c.updated_at,
            (c.seats - COALESCE(member_count.count, 0)) as available_seats,
            COALESCE(member_count.count, 0) as current_members
        FROM carpools c
        JOIN carpool_members cm ON c.id = cm.carpool_id
        LEFT JOIN (
            SELECT carpool_id, COUNT(*) as count 
            FROM carpool_members 
            GROUP BY carpool_id
        ) member_count ON c.id = member_count.carpool_id
        WHERE cm.user_id = $1
          AND (c.company_id = $2 OR ($2 IS NULL AND c.company_id IS NULL))
    `
    // Update Scan to include company_id and site_id
    // ...
}
```

**3c.2 Update All Other Carpool Queries**

- `GetCarPool()` - Add company_id filter if needed
- `GetCarpoolsByCreator()` - Add company_id filter
- Any other carpool retrieval methods

**3c.3 Update `CreateCarPool()` Method**

```go
// Add company_id and site_id to INSERT
func (r *CarPoolRepository) CreateCarPool(ctx context.Context, carpool *models.Carpool) error {
    query := `
        INSERT INTO carpools (
            id, creator_id, carpool_name, status, recurring_option,
            available_seats, destination_address, seats, company_id, site_id,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `
    // ...
}
```

**3c.4 Update Handlers**

**File:** `pkg/handlers/carpool_handlers.go`

```go
// Update GetUserCarpools handler
func (h *CarpoolHandler) GetUserCarpools(w http.ResponseWriter, r *http.Request) {
    // ... existing auth code ...
    
    // Default to personal scope
    var companyID *uuid.UUID = nil
    
    carpools, err := h.carpoolRepo.GetUserCarpools(r.Context(), userID, companyID)
    // ... rest of handler ...
}
```

#### Verification

- [ ] All existing tests pass
- [ ] Personal carpools still accessible
- [ ] No data leaks
- [ ] Can create personal carpools
- [ ] Response includes `company_id` and `site_id` fields

#### Rollback

- Revert method signatures
- Revert queries
- Revert handler changes

---

### Phase 3d: Update `carpool_rides` Queries

#### Tasks

**File:** `pkg/repository/carpoolRide_repository.go`

**3d.1 Update All Ride Queries**

Add `company_id` filter through carpools join or use denormalized column:

```go
// Example: GetRidesForCarpool
func (r *CarPoolRideRepository) GetRidesForCarpool(ctx context.Context, carpoolID uuid.UUID, companyID *uuid.UUID) ([]*models.CarpoolRide, error) {
    query := `
        SELECT cr.* 
        FROM carpool_rides cr
        JOIN carpools c ON cr.carpool_id = c.id
        WHERE cr.carpool_id = $1
          AND (c.company_id = $2 OR ($2 IS NULL AND c.company_id IS NULL))
        ORDER BY cr.start_time ASC
    `
    // ...
}
```

**3d.2 Update `CreateCarpoolRide()` Method**

```go
// Add company_id and site_id to INSERT (copy from carpool)
func (r *CarPoolRideRepository) CreateCarpoolRide(ctx context.Context, ride *models.CarpoolRide) error {
    // Get carpool to copy company_id/site_id
    carpool, err := r.getCarpoolForRide(ctx, ride.CarpoolID)
    if err != nil {
        return err
    }
    
    query := `
        INSERT INTO carpool_rides (
            id, carpool_id, driver_id, status, location_lat, location_lng,
            miles_saved, participants, start_time, company_id, site_id,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
    `
    // Use carpool.CompanyID and carpool.SiteID
    // ...
}
```

**3d.3 Update Handlers**

Update all ride handlers to pass `companyID` parameter.

#### Verification

- [ ] All existing tests pass
- [ ] Personal rides still accessible
- [ ] No data leaks
- [ ] Can create rides for personal carpools
- [ ] Company_id/site_id copied from carpool when creating rides

#### Rollback

- Revert queries
- Revert handler changes

---

### Phase 3e: Update `carpool_schedules` Queries

#### Tasks

**File:** `pkg/repository/carpoolSchedule_repository.go`

**3e.1 Update All Schedule Queries**

Similar to rides - add `company_id` filter:

```go
// Example: GetSchedulesForCarpool
func (r *CarpoolScheduleRepository) GetSchedulesForCarpool(ctx context.Context, carpoolID uuid.UUID, companyID *uuid.UUID) ([]*models.CarpoolSchedule, error) {
    query := `
        SELECT cs.* 
        FROM carpool_schedules cs
        JOIN carpools c ON cs.carpool_id = c.id
        WHERE cs.carpool_id = $1
          AND (c.company_id = $2 OR ($2 IS NULL AND c.company_id IS NULL))
    `
    // ...
}
```

**3e.2 Update `CreateCarpoolSchedule()` Method**

```go
// Add company_id and site_id to INSERT (copy from carpool)
func (r *CarpoolScheduleRepository) CreateCarpoolSchedule(ctx context.Context, schedule *models.CarpoolSchedule) error {
    // Get carpool to copy company_id/site_id
    // Insert with company_id/site_id
    // ...
}
```

**3e.3 Update Handlers**

Update all schedule handlers.

#### Verification

- [ ] All existing tests pass
- [ ] Personal schedules still accessible
- [ ] No data leaks
- [ ] Can create schedules for personal carpools
- [ ] Company_id/site_id copied from carpool

#### Rollback

- Revert queries
- Revert handler changes

---

### Phase 3: Overall Verification

**After completing all sub-phases:**

- [ ] All existing tests pass
- [ ] All existing functionality works (personal scope)
- [ ] No data leaks (verify with test data)
- [ ] Performance is acceptable (no significant slowdown)
- [ ] Code review completed
- [ ] Documentation updated

**⚠️ DO NOT PROCEED TO PHASE 4+ UNTIL ALL VERIFICATION CHECKS PASS**

---

## Phase 4: Scope Resolution & Handlers

**Goal:** Implement scope resolution and update handlers to use it  
**Risk:** Medium ⚠️  
**Duration:** 2-3 days  
**Blocks:** Phase 5+ (membership detection)

### Tasks

#### 4.1 Create Scope Resolution Helper

**File:** `pkg/utils/scope_resolver.go` (new file)

```go
package utils

import (
    "fmt"
    "net/http"
    "strings"
    "github.com/google/uuid"
)

type Scope struct {
    Type      string      // "personal" | "company"
    CompanyID *uuid.UUID
    SiteID    *uuid.UUID
}

// HTTPError represents an HTTP error with status code
type HTTPError struct {
    Status  int
    Message string
    Code    string
}

func (e *HTTPError) Error() string {
    return e.Message
}

func ResolveScope(r *http.Request, userID uuid.UUID, companyRepo *CompanyRepository) (Scope, *HTTPError) {
    // Priority 1: URL pattern /api/company/{slug}/...
    if strings.HasPrefix(r.URL.Path, "/api/company/") {
        slug := extractSlugFromPath(r.URL.Path)
        company, err := companyRepo.GetCompanyBySlug(r.Context(), slug)
        if err != nil {
            return Scope{}, &HTTPError{
                Status:  http.StatusNotFound,
                Message: "Company not found",
                Code:    "COMPANY_NOT_FOUND",
            }
        }
        
        // Validate user has membership
        membership, err := companyRepo.GetUserCompanyMembership(r.Context(), userID, company.ID)
        if err != nil {
            return Scope{}, &HTTPError{
                Status:  http.StatusForbidden,
                Message: "User is not a member of this company",
                Code:    "MISSING_COMPANY_MEMBERSHIP",
            }
        }
        
        return Scope{
            Type:      "company",
            CompanyID: &company.ID,
            SiteID:    membership.SiteID,
        }, nil
    }
    
    // Priority 2: Query parameters
    if scopeParam := r.URL.Query().Get("scope"); scopeParam == "company" {
        companyIDStr := r.URL.Query().Get("company_id")
        if companyIDStr == "" {
            return Scope{}, &HTTPError{
                Status:  http.StatusBadRequest,
                Message: "company_id required when scope=company",
                Code:    "MISSING_COMPANY_ID",
            }
        }
        
        companyID, err := uuid.Parse(companyIDStr)
        if err != nil {
            return Scope{}, &HTTPError{
                Status:  http.StatusBadRequest,
                Message: "Invalid company_id format",
                Code:    "INVALID_COMPANY_ID",
            }
        }
        
        // Validate membership
        membership, err := companyRepo.GetUserCompanyMembership(r.Context(), userID, companyID)
        if err != nil {
            return Scope{}, &HTTPError{
                Status:  http.StatusForbidden,
                Message: "User is not a member of this company",
                Code:    "MISSING_COMPANY_MEMBERSHIP",
            }
        }
        
        siteIDStr := r.URL.Query().Get("site_id")
        var siteID *uuid.UUID
        if siteIDStr != "" {
            parsedSiteID, err := uuid.Parse(siteIDStr)
            if err != nil {
                return Scope{}, &HTTPError{
                    Status:  http.StatusBadRequest,
                    Message: "Invalid site_id format",
                    Code:    "INVALID_SITE_ID",
                }
            }
            siteID = &parsedSiteID
        }
        
        return Scope{
            Type:      "company",
            CompanyID: &companyID,
            SiteID:    siteID,
        }, nil
    }
    
    // Priority 3: Request body (for POST/PUT)
    // Check if body has scope/company_id (parse JSON)
    // This would be handled in individual handlers that parse the body
    
    // Default: Personal scope
    return Scope{Type: "personal"}, nil
}

// Helper functions to implement
func extractSlugFromPath(path string) string {
    // Extract slug from /api/company/{slug}/...
    parts := strings.Split(path, "/")
    if len(parts) >= 4 && parts[2] == "company" {
        return parts[3]
    }
    return ""
}
```

#### 4.2 Create Company Repository Methods

**File:** `pkg/repository/company_repository.go` (new file)

```go
package repository

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "github.com/google/uuid"
)

type CompanyRepository struct {
    db *sql.DB
}

func NewCompanyRepository(db *sql.DB) *CompanyRepository {
    return &CompanyRepository{db: db}
}

// Company model
type Company struct {
    ID                uuid.UUID
    Name              string
    Slug              string
    PrimaryDomain     string
    AdditionalDomains []string
    Settings          map[string]interface{}
    IsActive          bool
    CreatedAt         time.Time
    UpdatedAt         time.Time
}

// Site model
type Site struct {
    ID        uuid.UUID
    CompanyID uuid.UUID
    Name      string
    Code      *string
    Address   *string
    Latitude  *float64
    Longitude *float64
    Timezone  *string
    IsActive  bool
    CreatedAt time.Time
    UpdatedAt time.Time
}

// UserCompanyMembership model
type UserCompanyMembership struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    CompanyID uuid.UUID
    SiteID    *uuid.UUID
    Role      string
    Status    string
    CreatedAt time.Time
    UpdatedAt time.Time
}

// GetCompanyByID retrieves a company by its ID
func (r *CompanyRepository) GetCompanyByID(ctx context.Context, companyID uuid.UUID) (*Company, error) {
    query := `
        SELECT id, name, slug, primary_domain, additional_domains, settings, is_active, created_at, updated_at
        FROM companies 
        WHERE id = $1 AND is_active = TRUE
    `
    
    var company Company
    var additionalDomainsJSON []byte
    var settingsJSON []byte
    
    err := r.db.QueryRowContext(ctx, query, companyID).Scan(
        &company.ID, &company.Name, &company.Slug, &company.PrimaryDomain,
        &additionalDomainsJSON, &settingsJSON, &company.IsActive,
        &company.CreatedAt, &company.UpdatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("company not found")
        }
        return nil, fmt.Errorf("failed to get company: %w", err)
    }
    
    // Parse JSON fields
    if err := json.Unmarshal(additionalDomainsJSON, &company.AdditionalDomains); err != nil {
        company.AdditionalDomains = []string{}
    }
    if err := json.Unmarshal(settingsJSON, &company.Settings); err != nil {
        company.Settings = make(map[string]interface{})
    }
    
    return &company, nil
}

// GetCompanyBySlug retrieves a company by its slug
func (r *CompanyRepository) GetCompanyBySlug(ctx context.Context, slug string) (*Company, error) {
    query := `
        SELECT id, name, slug, primary_domain, additional_domains, settings, is_active, created_at, updated_at
        FROM companies 
        WHERE slug = $1 AND is_active = TRUE
    `
    
    var company Company
    var additionalDomainsJSON []byte
    var settingsJSON []byte
    
    err := r.db.QueryRowContext(ctx, query, slug).Scan(
        &company.ID, &company.Name, &company.Slug, &company.PrimaryDomain,
        &additionalDomainsJSON, &settingsJSON, &company.IsActive,
        &company.CreatedAt, &company.UpdatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("company not found")
        }
        return nil, fmt.Errorf("failed to get company: %w", err)
    }
    
    // Parse JSON fields
    if err := json.Unmarshal(additionalDomainsJSON, &company.AdditionalDomains); err != nil {
        company.AdditionalDomains = []string{}
    }
    if err := json.Unmarshal(settingsJSON, &company.Settings); err != nil {
        company.Settings = make(map[string]interface{})
    }
    
    return &company, nil
}

// GetCompanyByDomain retrieves a company by email domain
func (r *CompanyRepository) GetCompanyByDomain(ctx context.Context, domain string) (*Company, error) {
    query := `
        SELECT id, name, slug, primary_domain, additional_domains, settings, is_active, created_at, updated_at
        FROM companies 
        WHERE (primary_domain = $1 OR $1 = ANY(SELECT jsonb_array_elements_text(additional_domains)))
          AND is_active = TRUE
        LIMIT 1
    `
    
    var company Company
    var additionalDomainsJSON []byte
    var settingsJSON []byte
    
    err := r.db.QueryRowContext(ctx, query, domain).Scan(
        &company.ID, &company.Name, &company.Slug, &company.PrimaryDomain,
        &additionalDomainsJSON, &settingsJSON, &company.IsActive,
        &company.CreatedAt, &company.UpdatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("company not found for domain: %s", domain)
        }
        return nil, fmt.Errorf("failed to get company by domain: %w", err)
    }
    
    // Parse JSON fields
    if err := json.Unmarshal(additionalDomainsJSON, &company.AdditionalDomains); err != nil {
        company.AdditionalDomains = []string{}
    }
    if err := json.Unmarshal(settingsJSON, &company.Settings); err != nil {
        company.Settings = make(map[string]interface{})
    }
    
    return &company, nil
}

// GetUserCompanyMembership retrieves a user's membership in a company
func (r *CompanyRepository) GetUserCompanyMembership(ctx context.Context, userID, companyID uuid.UUID) (*UserCompanyMembership, error) {
    query := `
        SELECT id, user_id, company_id, site_id, role, status, created_at, updated_at
        FROM user_company_memberships 
        WHERE user_id = $1 AND company_id = $2 AND status = 'active'
    `
    
    var membership UserCompanyMembership
    var siteID sql.NullString
    
    err := r.db.QueryRowContext(ctx, query, userID, companyID).Scan(
        &membership.ID, &membership.UserID, &membership.CompanyID, &siteID,
        &membership.Role, &membership.Status, &membership.CreatedAt, &membership.UpdatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("user is not a member of this company")
        }
        return nil, fmt.Errorf("failed to get membership: %w", err)
    }
    
    if siteID.Valid {
        parsedSiteID, err := uuid.Parse(siteID.String)
        if err == nil {
            membership.SiteID = &parsedSiteID
        }
    }
    
    return &membership, nil
}

// GetUserMemberships retrieves all active memberships for a user
func (r *CompanyRepository) GetUserMemberships(ctx context.Context, userID uuid.UUID) ([]*UserCompanyMembership, error) {
    query := `
        SELECT id, user_id, company_id, site_id, role, status, created_at, updated_at
        FROM user_company_memberships 
        WHERE user_id = $1 AND status = 'active'
        ORDER BY created_at DESC
    `
    
    rows, err := r.db.QueryContext(ctx, query, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get memberships: %w", err)
    }
    defer rows.Close()
    
    var memberships []*UserCompanyMembership
    for rows.Next() {
        var membership UserCompanyMembership
        var siteID sql.NullString
        
        err := rows.Scan(
            &membership.ID, &membership.UserID, &membership.CompanyID, &siteID,
            &membership.Role, &membership.Status, &membership.CreatedAt, &membership.UpdatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan membership: %w", err)
        }
        
        if siteID.Valid {
            parsedSiteID, err := uuid.Parse(siteID.String)
            if err == nil {
                membership.SiteID = &parsedSiteID
            }
        }
        
        memberships = append(memberships, &membership)
    }
    
    return memberships, nil
}

// CreateMembership creates a new company membership
func (r *CompanyRepository) CreateMembership(ctx context.Context, membership *UserCompanyMembership) error {
    query := `
        INSERT INTO user_company_memberships (id, user_id, company_id, site_id, role, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
    `
    
    _, err := r.db.ExecContext(ctx, query,
        membership.ID, membership.UserID, membership.CompanyID, membership.SiteID,
        membership.Role, membership.Status,
    )
    if err != nil {
        return fmt.Errorf("failed to create membership: %w", err)
    }
    
    return nil
}

// UpdateMembershipSite updates a user's site selection within a company
func (r *CompanyRepository) UpdateMembershipSite(ctx context.Context, userID, companyID uuid.UUID, siteID *uuid.UUID) error {
    query := `
        UPDATE user_company_memberships
        SET site_id = $1, updated_at = CURRENT_TIMESTAMP
        WHERE user_id = $2 AND company_id = $3 AND status = 'active'
    `
    
    result, err := r.db.ExecContext(ctx, query, siteID, userID, companyID)
    if err != nil {
        return fmt.Errorf("failed to update membership site: %w", err)
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    if rowsAffected == 0 {
        return fmt.Errorf("membership not found or not active")
    }
    
    return nil
}

// GetCompanySites retrieves all active sites for a company
func (r *CompanyRepository) GetCompanySites(ctx context.Context, companyID uuid.UUID) ([]*Site, error) {
    query := `
        SELECT id, company_id, name, code, address, latitude, longitude, timezone, is_active, created_at, updated_at
        FROM sites 
        WHERE company_id = $1 AND is_active = TRUE
        ORDER BY name
    `
    
    rows, err := r.db.QueryContext(ctx, query, companyID)
    if err != nil {
        return nil, fmt.Errorf("failed to get sites: %w", err)
    }
    defer rows.Close()
    
    var sites []*Site
    for rows.Next() {
        var site Site
        var code, address, timezone sql.NullString
        var lat, lng sql.NullFloat64
        
        err := rows.Scan(
            &site.ID, &site.CompanyID, &site.Name, &code, &address,
            &lat, &lng, &timezone, &site.IsActive, &site.CreatedAt, &site.UpdatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan site: %w", err)
        }
        
        if code.Valid {
            site.Code = &code.String
        }
        if address.Valid {
            site.Address = &address.String
        }
        if lat.Valid {
            site.Latitude = &lat.Float64
        }
        if lng.Valid {
            site.Longitude = &lng.Float64
        }
        if timezone.Valid {
            site.Timezone = &timezone.String
        }
        
        sites = append(sites, &site)
    }
    
    return sites, nil
}

// GetSite retrieves a site by ID
func (r *CompanyRepository) GetSite(ctx context.Context, siteID uuid.UUID) (*Site, error) {
    query := `
        SELECT id, company_id, name, code, address, latitude, longitude, timezone, is_active, created_at, updated_at
        FROM sites 
        WHERE id = $1
    `
    
    var site Site
    var code, address, timezone sql.NullString
    var lat, lng sql.NullFloat64
    
    err := r.db.QueryRowContext(ctx, query, siteID).Scan(
        &site.ID, &site.CompanyID, &site.Name, &code, &address,
        &lat, &lng, &timezone, &site.IsActive, &site.CreatedAt, &site.UpdatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("site not found")
        }
        return nil, fmt.Errorf("failed to get site: %w", err)
    }
    
    if code.Valid {
        site.Code = &code.String
    }
    if address.Valid {
        site.Address = &address.String
    }
    if lat.Valid {
        site.Latitude = &lat.Float64
    }
    if lng.Valid {
        site.Longitude = &lng.Float64
    }
    if timezone.Valid {
        site.Timezone = &timezone.String
    }
    
    return &site, nil
}
```

#### 4.3 Update All Handlers to Use Scope Resolution

**Files:** All handler files

```go
// Example: Update GetMatchingPreferences handler
func (h *MatchingHandler) GetMatchingPreferences(w http.ResponseWriter, r *http.Request) {
    // ... existing auth code ...
    
    // Resolve scope
    scope, httpErr := utils.ResolveScope(r, userID, h.companyRepo)
    if httpErr != nil {
        w.WriteHeader(httpErr.Status)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error":   httpErr.Code,
            "message": httpErr.Message,
        })
        return
    }
    
    var companyID *uuid.UUID
    if scope.Type == "company" {
        companyID = scope.CompanyID
    } else {
        companyID = nil // Personal scope
    }
    
    prefs, err := h.matchingRepo.GetUserMatchingPreferences(r.Context(), userID, companyID)
    // ... rest of handler ...
}
```

### Verification

- [ ] Scope resolution works for personal (no params)
- [ ] Scope resolution works for company (query params)
- [ ] Scope resolution works for company (URL pattern)
- [ ] Invalid company_id returns error
- [ ] Non-member access returns 403
- [ ] All handlers updated
- [ ] All existing tests pass

### Rollback

- Revert scope resolver
- Revert handler changes
- Remove company repository

---

## Phase 5: Membership Detection

**Goal:** Auto-detect company membership from email domain  
**Risk:** Low ✅  
**Duration:** 1-2 days

### Tasks

#### 5.1 Create Email Domain Detection Service

**File:** `pkg/services/company_membership_service.go` (new file)

```go
package services

import (
    "context"
    "strings"
    "github.com/google/uuid"
)

type CompanyMembershipService struct {
    companyRepo *repository.CompanyRepository
    membershipRepo *repository.CompanyMembershipRepository
}

func (s *CompanyMembershipService) AutoDetectMembership(ctx context.Context, userID uuid.UUID, email string) error {
    // Extract domain
    parts := strings.Split(email, "@")
    if len(parts) != 2 {
        return nil // Invalid email, skip
    }
    domain := strings.ToLower(parts[1])
    
    // Check blacklist
    if isBlacklistedDomain(domain) {
        return nil // Skip generic domains
    }
    
    // Lookup company by domain
    company, err := s.companyRepo.GetCompanyByDomain(ctx, domain)
    if err != nil {
        return nil // No company found, skip
    }
    
    // Check if membership already exists
    existing, _ := s.membershipRepo.GetUserCompanyMembership(ctx, userID, company.ID)
    if existing != nil {
        return nil // Already a member
    }
    
    // Create membership
    membership := &models.UserCompanyMembership{
        UserID:    userID,
        CompanyID: company.ID,
        SiteID:    nil, // User picks later
        Role:      "employee",
        Status:    "active",
    }
    
    return s.membershipRepo.CreateMembership(ctx, membership)
}

func isBlacklistedDomain(domain string) bool {
    blacklist := []string{"gmail.com", "yahoo.com", "outlook.com", "hotmail.com", "icloud.com"}
    for _, bl := range blacklist {
        if domain == bl {
            return true
        }
    }
    return false
}
```

#### 5.2 Integrate into User Creation/Login

**File:** `pkg/handlers/auth_handlers.go` (or wherever user creation happens)

```go
// After user is created/verified
func (h *AuthHandler) OnUserCreated(ctx context.Context, userID uuid.UUID, email string) {
    // Auto-detect company membership
    err := h.companyMembershipService.AutoDetectMembership(ctx, userID, email)
    if err != nil {
        log.Printf("Failed to auto-detect membership: %v", err)
        // Don't fail user creation
    }
}
```

### Verification

- [ ] Email domain detection works
- [ ] Generic domains are blacklisted
- [ ] Memberships created correctly
- [ ] Existing memberships not duplicated
- [ ] User creation still works if detection fails

### Rollback

- Remove auto-detection service
- Remove integration points

---

## Phase 6: Company APIs

**Goal:** Implement new company endpoints and enable company scope  
**Risk:** Medium ⚠️  
**Duration:** 3-4 days

### Tasks

#### 6.1 Create Company Handlers

**File:** `pkg/handlers/company_handlers.go` (new file)

```go
package handlers

import (
    "encoding/json"
    "net/http"
    "github.com/google/uuid"
)

type CompanyHandler struct {
    companyRepo *repository.CompanyRepository
    userRepo    *repository.UserRepository
}

func NewCompanyHandler(companyRepo *repository.CompanyRepository, userRepo *repository.UserRepository) *CompanyHandler {
    return &CompanyHandler{
        companyRepo: companyRepo,
        userRepo:    userRepo,
    }
}

// GET /api/me/company
func (h *CompanyHandler) GetUserCompanies(w http.ResponseWriter, r *http.Request) {
    // Get authenticated user
    userID, err := getAuthenticatedUserID(r)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Get user memberships
    memberships, err := h.companyRepo.GetUserMemberships(r.Context(), userID)
    if err != nil {
        log.Printf("Failed to get user memberships: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    // Format response
    response := map[string]interface{}{
        "memberships": []map[string]interface{}{},
    }
    
    for _, m := range memberships {
        // Get company details
        company, err := h.companyRepo.GetCompanyByID(r.Context(), m.CompanyID)
        if err != nil {
            log.Printf("Failed to get company: %v", err)
            continue
        }
        
        membershipData := map[string]interface{}{
            "company_id":   m.CompanyID.String(),
            "company_name": company.Name,
            "company_slug": company.Slug,
            "role":         m.Role,
            "status":       m.Status,
        }
        
        // Get site details if site_id exists
        if m.SiteID != nil {
            site, err := h.companyRepo.GetSite(r.Context(), *m.SiteID)
            if err == nil {
                membershipData["site"] = map[string]interface{}{
                    "id":       site.ID.String(),
                    "name":     site.Name,
                    "code":     site.Code,
                    "timezone": site.Timezone,
                }
            }
        }
        
        response["memberships"] = append(response["memberships"].([]map[string]interface{}), membershipData)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// PUT /api/me/company-site
func (h *CompanyHandler) UpdateUserSite(w http.ResponseWriter, r *http.Request) {
    // Get authenticated user
    userID, err := getAuthenticatedUserID(r)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Parse request body
    var req struct {
        CompanyID uuid.UUID  `json:"company_id"`
        SiteID    *uuid.UUID `json:"site_id"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Validate user has membership
    membership, err := h.companyRepo.GetUserCompanyMembership(r.Context(), userID, req.CompanyID)
    if err != nil {
        http.Error(w, "User is not a member of this company", http.StatusForbidden)
        return
    }
    
    if membership.Status != "active" {
        http.Error(w, "Membership is not active", http.StatusForbidden)
        return
    }
    
    // If site_id provided, validate it belongs to company
    if req.SiteID != nil {
        site, err := h.companyRepo.GetSite(r.Context(), *req.SiteID)
        if err != nil {
            http.Error(w, "Site not found", http.StatusNotFound)
            return
        }
        
        if site.CompanyID != req.CompanyID {
            http.Error(w, "Invalid site_id for this company", http.StatusBadRequest)
            return
        }
        
        if !site.IsActive {
            http.Error(w, "Site is not active", http.StatusBadRequest)
            return
        }
    }
    
    // Update membership
    err = h.companyRepo.UpdateMembershipSite(r.Context(), userID, req.CompanyID, req.SiteID)
    if err != nil {
        log.Printf("Failed to update membership site: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    // Return response
    response := map[string]interface{}{
        "company_id": req.CompanyID.String(),
        "site_id":    nil,
        "updated_at": time.Now().Format(time.RFC3339),
    }
    
    if req.SiteID != nil {
        response["site_id"] = req.SiteID.String()
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

#### 6.2 Update Matching Handlers for Company Scope

**File:** `pkg/handlers/matching_handlers.go`

**6.2.1 Update `CreateMatchRequest` Handler with Validation**

```go
func (h *MatchingHandler) CreateMatchRequest(w http.ResponseWriter, r *http.Request) {
    // ... existing auth code ...
    
    // Parse request body
    var reqBody models.MatchRequestPayload
    if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Resolve scope from request body
    var companyID *uuid.UUID
    var siteID *uuid.UUID
    
    if reqBody.CompanyID != "" {
        parsedCompanyID, err := uuid.Parse(reqBody.CompanyID)
        if err != nil {
            http.Error(w, "Invalid company_id format", http.StatusBadRequest)
            return
        }
        companyID = &parsedCompanyID
        
        // Validate sender has membership
        senderMembership, err := h.companyRepo.GetUserCompanyMembership(r.Context(), userID, *companyID)
        if err != nil || senderMembership.Status != "active" {
            http.Error(w, "Sender is not a member of this company", http.StatusForbidden)
            return
        }
        
        // Validate recipient has membership
        recipientID, err := uuid.Parse(reqBody.ToUserID)
        if err != nil {
            http.Error(w, "Invalid to_user_id", http.StatusBadRequest)
            return
        }
        
        recipientMembership, err := h.companyRepo.GetUserCompanyMembership(r.Context(), recipientID, *companyID)
        if err != nil || recipientMembership.Status != "active" {
            http.Error(w, "Recipient is not a member of this company", http.StatusForbidden)
            return
        }
        
        // Validate site_id if provided
        if reqBody.SiteID != "" {
            parsedSiteID, err := uuid.Parse(reqBody.SiteID)
            if err != nil {
                http.Error(w, "Invalid site_id format", http.StatusBadRequest)
                return
            }
            
            site, err := h.companyRepo.GetSite(r.Context(), parsedSiteID)
            if err != nil {
                http.Error(w, "Site not found", http.StatusNotFound)
                return
            }
            
            if site.CompanyID != *companyID {
                http.Error(w, "Invalid site_id for this company", http.StatusBadRequest)
                return
            }
            
            siteID = &parsedSiteID
        }
    }
    
    // Create match request with company_id/site_id
    matchRequest := &models.MatchRequest{
        // ... existing fields ...
        CompanyID: companyID,
        SiteID:    siteID,
    }
    
    err = h.matchingRepo.CreateMatchRequest(r.Context(), matchRequest, companyID, siteID)
    // ... rest of handler ...
}
```

**6.2.2 Update All Other Matching Handlers**

```go
// Update all handlers to use scope resolution (from Phase 4)
// Enable company scope in:
// - GetMatchingPreferences (already done in Phase 3a)
// - UpsertMatchingPreferences (already done in Phase 3a)
// - FindMatches - Add scope parameter and validation
// - GetMatchRequests (already done in Phase 3b)
// - UpdateMatchRequest - Add validation for company requests
```

#### 6.3 Update Carpool Handlers for Company Scope

**File:** `pkg/handlers/carpool_handlers.go`

**6.3.1 Add Carpool Access Validation**

```go
// Helper function to validate carpool access
func (h *CarpoolHandler) validateCarpoolAccess(ctx context.Context, carpoolID uuid.UUID, userID uuid.UUID) error {
    carpool, err := h.carpoolRepo.GetCarPool(ctx, carpoolID)
    if err != nil {
        return fmt.Errorf("carpool not found: %w", err)
    }
    
    // If company carpool, validate membership
    if carpool.CompanyID != nil {
        membership, err := h.companyRepo.GetUserCompanyMembership(ctx, userID, *carpool.CompanyID)
        if err != nil {
            return fmt.Errorf("user is not a member of this company: %w", err)
        }
        
        if membership.Status != "active" {
            return fmt.Errorf("membership is not active")
        }
    }
    
    return nil
}

// Update GetCarPool handler
func (h *CarpoolHandler) GetCarPool(w http.ResponseWriter, r *http.Request) {
    carpoolID := extractCarpoolID(r)
    userID := getAuthenticatedUserID(r)
    
    // Validate access
    if err := h.validateCarpoolAccess(r.Context(), carpoolID, userID); err != nil {
        if strings.Contains(err.Error(), "not found") {
            http.Error(w, "Carpool not found", http.StatusNotFound)
        } else {
            http.Error(w, "Access denied: "+err.Error(), http.StatusForbidden)
        }
        return
    }
    
    // Get carpool
    carpool, err := h.carpoolRepo.GetCarPool(r.Context(), carpoolID)
    if err != nil {
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    // Return carpool
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(carpool)
}
```

**6.3.2 Update All Carpool Handlers**

```go
// Update handlers to:
// - Filter by company_id (already done in Phase 3c)
// - Validate membership for company carpools (use validateCarpoolAccess helper)
// - Copy company_id/site_id when creating carpools from match requests
```

### Verification

- [ ] All new endpoints work
- [ ] Company scope works for all matching APIs
- [ ] Company scope works for all carpool APIs
- [ ] Personal scope still works (backward compatibility)
- [ ] Data isolation verified (no cross-company leaks)
- [ ] Error handling works correctly

### Rollback

- Revert handler changes
- Remove new endpoints

---

## Phase 7: Tagging Propagation

**Goal:** Ensure company_id/site_id propagate correctly through data chain  
**Risk:** Medium ⚠️  
**Duration:** 2-3 days

### Tasks

#### 7.1 Update Carpool Creation from Match Request

**File:** `pkg/handlers/matching_handlers.go`

```go
// In createCarpoolFromMatchRequest
func (h *MatchingHandler) createCarpoolFromMatchRequest(ctx context.Context, request *models.MatchRequest) (*uuid.UUID, error) {
    // ... existing code ...
    
    carpool := &models.Carpool{
        // ... existing fields ...
        CompanyID: request.CompanyID,  // Copy from request
        SiteID:    request.SiteID,     // Copy from request
    }
    
    // ... rest of creation ...
}
```

#### 7.2 Update Schedule Creation

**File:** `pkg/handlers/matching_handlers.go`

```go
// In createCarpoolSchedule
func (h *MatchingHandler) createCarpoolSchedule(ctx context.Context, carpoolID uuid.UUID, ...) error {
    // Get carpool to copy company_id/site_id
    carpool, err := h.carpoolRepo.GetCarPool(ctx, carpoolID)
    
    schedule := &models.CarpoolSchedule{
        // ... existing fields ...
        CompanyID: carpool.CompanyID,  // Copy from carpool
        SiteID:    carpool.SiteID,      // Copy from carpool
    }
    
    // ... rest of creation ...
}
```

#### 7.3 Update Ride Creation

**File:** `pkg/handlers/matching_handlers.go`

```go
// In generateRidesFromSchedule
func (h *MatchingHandler) generateRidesFromSchedule(...) error {
    // Get carpool to copy company_id/site_id
    carpool, err := h.carpoolRepo.GetCarPool(ctx, schedule.CarpoolID)
    
    ride := &models.CarpoolRide{
        // ... existing fields ...
        CompanyID: carpool.CompanyID,  // Copy from carpool
        SiteID:    carpool.SiteID,     // Copy from carpool
    }
    
    // ... rest of creation ...
}
```

### Verification

- [ ] Company_id propagates: request → carpool → schedule → ride
- [ ] Site_id propagates correctly
- [ ] Personal data remains NULL (no accidental tagging)
- [ ] Data consistency verified

### Rollback

- Revert propagation logic
- Data cleanup if needed

---

## Phase 8: Admin Analytics

**Goal:** Implement company admin endpoints  
**Risk:** Low ✅  
**Duration:** 2-3 days

### Tasks

#### 8.1 Create Role Validation Helper

**File:** `pkg/utils/role_validator.go` (new file)

```go
package utils

import (
    "context"
    "fmt"
    "github.com/google/uuid"
)

// Role hierarchy levels
const (
    RoleLevelEmployee     = 1
    RoleLevelSiteAdmin    = 2
    RoleLevelCompanyAdmin = 3
)

var roleLevels = map[string]int{
    "employee":      RoleLevelEmployee,
    "site_admin":    RoleLevelSiteAdmin,
    "company_admin": RoleLevelCompanyAdmin,
}

// ValidateAdminRole validates that a user has the required role or higher
func ValidateAdminRole(
    ctx context.Context,
    companyRepo *repository.CompanyRepository,
    userID, companyID uuid.UUID,
    requiredRole string,
) error {
    membership, err := companyRepo.GetUserCompanyMembership(ctx, userID, companyID)
    if err != nil {
        return fmt.Errorf("user is not a member of this company: %w", err)
    }
    
    if membership.Status != "active" {
        return fmt.Errorf("membership is not active")
    }
    
    requiredLevel, exists := roleLevels[requiredRole]
    if !exists {
        return fmt.Errorf("invalid required role: %s", requiredRole)
    }
    
    userLevel, exists := roleLevels[membership.Role]
    if !exists {
        return fmt.Errorf("invalid user role: %s", membership.Role)
    }
    
    if userLevel < requiredLevel {
        return fmt.Errorf("insufficient permissions: requires %s, user has %s", requiredRole, membership.Role)
    }
    
    return nil
}

// ValidateSiteAdminRole validates that a user is site_admin or company_admin for a specific site
func ValidateSiteAdminRole(
    ctx context.Context,
    companyRepo *repository.CompanyRepository,
    userID, companyID, siteID uuid.UUID,
) error {
    membership, err := companyRepo.GetUserCompanyMembership(ctx, userID, companyID)
    if err != nil {
        return fmt.Errorf("user is not a member of this company: %w", err)
    }
    
    if membership.Status != "active" {
        return fmt.Errorf("membership is not active")
    }
    
    // Company admin can access any site
    if membership.Role == "company_admin" {
        return nil
    }
    
    // Site admin can only access their assigned site
    if membership.Role == "site_admin" {
        if membership.SiteID == nil || *membership.SiteID != siteID {
            return fmt.Errorf("site admin can only access their assigned site")
        }
        return nil
    }
    
    return fmt.Errorf("insufficient permissions: requires site_admin or company_admin")
}
```

#### 8.2 Create Admin Handlers

**File:** `pkg/handlers/company_admin_handlers.go` (new file)

```go
package handlers

import (
    "encoding/json"
    "net/http"
    "github.com/google/uuid"
    "car-backend/pkg/utils"
)

type CompanyAdminHandler struct {
    companyRepo *repository.CompanyRepository
    adminRepo   *repository.CompanyAdminRepository
}

func NewCompanyAdminHandler(companyRepo *repository.CompanyRepository, adminRepo *repository.CompanyAdminRepository) *CompanyAdminHandler {
    return &CompanyAdminHandler{
        companyRepo: companyRepo,
        adminRepo:   adminRepo,
    }
}

// GET /api/companies/{companyId}/stats
func (h *CompanyAdminHandler) GetCompanyStats(w http.ResponseWriter, r *http.Request) {
    // Get authenticated user
    userID, err := getAuthenticatedUserID(r)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Extract company ID from URL
    companyID, err := extractCompanyIDFromPath(r.URL.Path)
    if err != nil {
        http.Error(w, "Invalid company ID", http.StatusBadRequest)
        return
    }
    
    // Validate admin role (site_admin or company_admin)
    if err := utils.ValidateAdminRole(r.Context(), h.companyRepo, userID, companyID, "site_admin"); err != nil {
        http.Error(w, err.Error(), http.StatusForbidden)
        return
    }
    
    // Calculate stats
    stats, err := h.adminRepo.GetCompanyStats(r.Context(), companyID)
    if err != nil {
        log.Printf("Failed to get company stats: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}

// GET /api/companies/{companyId}/sites/{siteId}/stats
func (h *CompanyAdminHandler) GetSiteStats(w http.ResponseWriter, r *http.Request) {
    // Get authenticated user
    userID, err := getAuthenticatedUserID(r)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Extract company ID and site ID from URL
    companyID, siteID, err := extractCompanyAndSiteIDFromPath(r.URL.Path)
    if err != nil {
        http.Error(w, "Invalid company or site ID", http.StatusBadRequest)
        return
    }
    
    // Validate site admin role
    if err := utils.ValidateSiteAdminRole(r.Context(), h.companyRepo, userID, companyID, siteID); err != nil {
        http.Error(w, err.Error(), http.StatusForbidden)
        return
    }
    
    // Calculate stats
    stats, err := h.adminRepo.GetSiteStats(r.Context(), companyID, siteID)
    if err != nil {
        log.Printf("Failed to get site stats: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}

// GET /api/companies/{companyId}/adoption
func (h *CompanyAdminHandler) GetAdoption(w http.ResponseWriter, r *http.Request) {
    // Get authenticated user
    userID, err := getAuthenticatedUserID(r)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Extract company ID from URL
    companyID, err := extractCompanyIDFromPath(r.URL.Path)
    if err != nil {
        http.Error(w, "Invalid company ID", http.StatusBadRequest)
        return
    }
    
    // Validate company_admin role
    if err := utils.ValidateAdminRole(r.Context(), h.companyRepo, userID, companyID, "company_admin"); err != nil {
        http.Error(w, err.Error(), http.StatusForbidden)
        return
    }
    
    // Calculate adoption
    adoption, err := h.adminRepo.GetCompanyAdoption(r.Context(), companyID)
    if err != nil {
        log.Printf("Failed to get adoption: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(adoption)
}
```

#### 8.2 Create Admin Repository Methods

**File:** `pkg/repository/company_admin_repository.go` (new file)

```go
// Stats calculation queries
func (r *CompanyAdminRepository) GetCompanyStats(ctx context.Context, companyID uuid.UUID) (*CompanyStats, error) {
    // Complex aggregation queries
    // Return stats
}
```

### Verification

- [ ] Admin endpoints work
- [ ] Role-based access control works
- [ ] Stats are accurate
- [ ] Performance is acceptable

### Rollback

- Remove admin handlers
- Remove admin repository

---

## Testing Strategy

### Test Data Setup

**File:** `test_data/company_test_data.sql` (new file)

```sql
-- Test companies
INSERT INTO companies (id, name, slug, primary_domain, is_active, settings) 
VALUES 
    ('00000000-0000-0000-0000-000000000001', 'Test Company A', 'test-company-a', 'testa.com', TRUE, '{}'::jsonb),
    ('00000000-0000-0000-0000-000000000002', 'Test Company B', 'test-company-b', 'testb.com', TRUE, '{}'::jsonb)
ON CONFLICT (id) DO NOTHING;

-- Test sites
INSERT INTO sites (id, company_id, name, code, is_active)
VALUES 
    ('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'Site A1', 'A1', TRUE),
    ('00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', 'Site A2', 'A2', TRUE),
    ('00000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000002', 'Site B1', 'B1', TRUE)
ON CONFLICT (id) DO NOTHING;

-- Test users (if needed)
-- INSERT INTO users (id, email, name, clerk_id) VALUES ...

-- Test memberships
-- INSERT INTO user_company_memberships (id, user_id, company_id, site_id, role, status) VALUES ...
```

### Unit Tests

**Files to Test:**
- `pkg/repository/company_repository.go` - All methods
- `pkg/repository/matching_repository.go` - Updated methods with company_id filters
- `pkg/repository/carpool_repository.go` - Updated methods
- `pkg/utils/scope_resolver.go` - Scope resolution logic
- `pkg/utils/role_validator.go` - Role validation
- `pkg/services/company_membership_service.go` - Auto-detection

**Test Cases:**
- [ ] Repository methods with company_id filters return correct data
- [ ] Scope resolution works for all priority levels
- [ ] Membership detection works correctly
- [ ] Tagging propagation works (request → carpool → schedule → ride)
- [ ] Role validation works for all role combinations
- [ ] Error handling works correctly

### Integration Tests

**Test Scenarios:**
- [ ] Personal scope works end-to-end (all existing functionality)
- [ ] Company scope works end-to-end (new functionality)
- [ ] Data isolation (no leaks between personal and company)
- [ ] Data isolation (no leaks between companies)
- [ ] Backward compatibility (all existing endpoints work without scope)
- [ ] Access validation (403 errors for unauthorized access)
- [ ] Site selection flow works
- [ ] Auto-membership detection works

**Test Data Requirements:**
- Create test users in different companies
- Create test carpools (personal and company)
- Create test match requests (personal and company)
- Verify isolation between companies

### Manual Testing Checklist

**Personal Scope:**
- [ ] Create personal carpool (should work as before)
- [ ] View personal requests (only personal)
- [ ] View personal carpools (only personal)
- [ ] All existing frontend flows work

**Company Scope:**
- [ ] Create company carpool (new functionality)
- [ ] View company requests (only that company)
- [ ] View company carpools (only that company)
- [ ] Switch between personal and company contexts

**Data Isolation:**
- [ ] User from Company A cannot see Company B data
- [ ] Personal data not visible in company scope
- [ ] Company data not visible in personal scope
- [ ] Access denied (403) for unauthorized company access

**Error Handling:**
- [ ] Invalid company_id returns 400
- [ ] Missing membership returns 403
- [ ] Invalid site_id returns 400
- [ ] Insufficient role returns 403

### Performance Testing

**Considerations:**
- [ ] Query performance with company_id filters (use EXPLAIN ANALYZE)
- [ ] Index usage verification
- [ ] Large dataset testing (if applicable)
- [ ] Concurrent request handling

**Performance Monitoring:**
- Monitor query execution times before/after Phase 3
- Check index usage with `EXPLAIN ANALYZE`
- Monitor API response times
- Check for N+1 query problems

---

## Rollback Procedures

### Full Rollback (Emergency)

1. **Stop all company features:**
   - Disable company scope in scope resolver (always return personal)
   - Disable company endpoints
   - Set feature flag: `ENABLE_COMPANY_FEATURES=false`

2. **Data cleanup (if needed):**
   - Mark company data as inactive: `UPDATE companies SET is_active = FALSE`
   - Or delete test company data (if in development)

3. **Code rollback:**
   - Revert to commit before Phase 3
   - Keep database schema (columns can stay, just not used)
   - All queries will still filter by `company_id IS NULL` (personal only)

### Partial Rollback

**Rollback Specific Phase:**
- Phase 3: Revert query changes, keep schema
- Phase 4: Revert scope resolver, keep Phase 3
- Phase 6: Disable company endpoints, keep Phase 3-4

**Rollback Steps:**
1. Identify problematic phase
2. Revert code changes for that phase
3. Keep previous phases intact
4. Fix issues and re-deploy

### Migration Rollback

**If migration fails:**
```sql
-- Phase 2 rollback (if needed)
-- See Phase 2 rollback section above

-- Phase 1 rollback (if needed)
DROP TABLE IF EXISTS user_company_memberships CASCADE;
DROP TABLE IF EXISTS sites CASCADE;
DROP TABLE IF EXISTS companies CASCADE;
```

**Note:** Always test rollback procedures in staging before production deployment.

---

## Logging Requirements

### Critical Logging Points

**1. Scope Resolution:**
```go
log.Printf("{\"severity\":\"INFO\",\"message\":\"Scope resolved\",\"user_id\":\"%s\",\"scope_type\":\"%s\",\"company_id\":\"%v\"}",
    userID, scope.Type, scope.CompanyID)
```

**2. Membership Validation Failures (Security Audit):**
```go
log.Printf("{\"severity\":\"WARNING\",\"message\":\"Membership validation failed\",\"user_id\":\"%s\",\"company_id\":\"%s\",\"reason\":\"%s\"}",
    userID, companyID, err.Error())
```

**3. Company Data Access:**
```go
log.Printf("{\"severity\":\"INFO\",\"message\":\"Company data accessed\",\"user_id\":\"%s\",\"company_id\":\"%s\",\"resource\":\"%s\"}",
    userID, companyID, resourceType)
```

**4. Auto-Membership Creation:**
```go
log.Printf("{\"severity\":\"INFO\",\"message\":\"Auto-membership created\",\"user_id\":\"%s\",\"company_id\":\"%s\",\"domain\":\"%s\"}",
    userID, companyID, emailDomain)
```

**5. Access Denied (403) Events:**
```go
log.Printf("{\"severity\":\"WARNING\",\"message\":\"Access denied\",\"user_id\":\"%s\",\"company_id\":\"%s\",\"resource\":\"%s\",\"reason\":\"%s\"}",
    userID, companyID, resourceType, reason)
```

### Logging Best Practices

- Use structured logging (JSON format)
- Include user_id, company_id, scope_type in all relevant logs
- Log security events (access denied, validation failures)
- Log data isolation events (company data access)
- Don't log sensitive data (passwords, tokens)

---

## Migration Testing

### Pre-Production Testing

**Before running migrations in production:**

1. **Test on Staging Database:**
   - Use production-like data volume
   - Test all migrations in order
   - Verify rollback procedures work

2. **Verify Data Integrity:**
   - Run data migration verification scripts
   - Check that all existing data has `company_id IS NULL`
   - Verify no data corruption

3. **Test Query Performance:**
   - Run `EXPLAIN ANALYZE` on key queries
   - Verify indexes are used
   - Check for performance regressions

4. **Test Rollback:**
   - Practice rollback procedures
   - Verify data remains intact after rollback
   - Test partial rollbacks

5. **Load Testing:**
   - Test with production-like load
   - Monitor query performance
   - Check for deadlocks or contention

### Migration Checklist

**Before Migration:**
- [ ] Backup database
- [ ] Test migrations on staging
- [ ] Verify rollback procedures
- [ ] Review migration SQL for syntax errors
- [ ] Check for potential locks/contention

**During Migration:**
- [ ] Run migrations in order
- [ ] Monitor for errors
- [ ] Check query performance
- [ ] Verify data integrity

**After Migration:**
- [ ] Run verification scripts
- [ ] Test existing functionality
- [ ] Monitor performance
- [ ] Check logs for errors

---

## Success Criteria

### Phase 3 Complete
- ✅ All existing functionality works (personal scope)
- ✅ No data leaks
- ✅ All tests pass
- ✅ Query performance acceptable
- ✅ Indexes being used

### Phase 6 Complete
- ✅ Company features work
- ✅ Personal features still work
- ✅ Data isolation verified
- ✅ Access validation works
- ✅ Error handling correct

### Full Implementation Complete
- ✅ All phases complete
- ✅ All tests pass
- ✅ Performance acceptable
- ✅ Logging in place
- ✅ Frontend integration successful
- ✅ Production deployment successful
- ✅ Monitoring and alerts configured

---

**Implementation Plan Complete**  
**Ready to begin Phase 1**

**Last Updated:** 2025-01-XX  
**Status:** Approved with Frontend Review Fixes Applied

