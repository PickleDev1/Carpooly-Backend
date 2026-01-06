# Company Spaces Feature - Backend Blueprint

**Version:** 1.0 (with Safety Requirements)  
**Date:** 2025-01-XX  
**Status:** Ready for Frontend Review

---

## ⚠️ CRITICAL SAFETY NOTICE

**This feature is designed to be 100% backward compatible. However, implementation MUST follow these safety requirements:**

1. **PRIMARY KEY RESOLUTION:** Changed to support both personal AND company preferences (see Data Model section)
2. **MUST add `company_id IS NULL` filters to ALL existing queries** - Prevents data leaks
3. **MUST default to personal scope** - When no scope provided, use `company_id IS NULL`
4. **MUST complete Phase 3 (Query Filters) before enabling company features** - See Migration Plan

**See "⚠️ CRITICAL SAFETY REQUIREMENTS" section below for detailed implementation requirements.**

**Related Document:** `COMPANY_SPACES_SAFETY_ANALYSIS.md` contains detailed safety analysis.

---

## 📋 FRONTEND REVIEW RESPONSES

**This section addresses all concerns raised in the frontend team review:**

### ✅ CRITICAL ISSUE #1: Primary Key Contradiction - RESOLVED

**Resolution:** Changed to **Option A - Composite Key Approach**

- Added surrogate `id` column as new PRIMARY KEY
- Kept `user_id` with unique index for backward compatibility
- Users can now have BOTH personal AND company preferences simultaneously
- All existing queries continue to work (just need `AND company_id IS NULL` filter)

**See Data Model section for complete migration strategy.**

### ✅ MEDIUM PRIORITY #2: Query Filter Safeguards - ADDRESSED

**Safeguards Added:**
- Code review checklist
- Automated test requirements
- Database views (optional)
- Phase 3 broken into sub-phases with verification after each

**See "Query Filter Safeguards" section in Safety Requirements.**

### ✅ MEDIUM PRIORITY #3: Phase 3 Breakdown - COMPLETED

**Phase 3 now broken into 5 sub-phases:**
- Phase 3a: `user_matching_preferences` queries
- Phase 3b: `match_requests` queries
- Phase 3c: `carpools` queries
- Phase 3d: `carpool_rides` queries
- Phase 3e: `carpool_schedules` queries

**Each sub-phase includes:**
- Verification checklist
- Rollback plan
- Testing requirements

**See Migration Plan section for details.**

### ✅ MEDIUM PRIORITY #4: API Contract Documentation - COMPLETED

**Added complete documentation:**
- Request/response examples for all modified endpoints
- Error response formats (standardized)
- Backward compatibility guarantees
- Error codes documented

**See "API Contract Documentation" section and individual endpoint sections.**

### ✅ LOW PRIORITY #5: Site Selection Policy - CLARIFIED

**Chosen: Option A - Site Selection Required**
- Users cannot use company matching until site is selected
- Users can change site at any time
- Existing carpools/rides not affected by site change
- Frontend should show confirmation on site change

**See "Site Semantics" section in Enhancements.**

### ✅ LOW PRIORITY #6: Email Domain Policy - CLARIFIED

**Policies Defined:**
- Email change handling (create new membership, don't revoke existing)
- Domain change handling (update company, don't affect existing memberships)
- Multiple email handling (only primary email used)
- Blacklist for generic domains

**See "Email Domain Auto-Detection" section in Enhancements.**

---

## Table of Contents

1. [Overview](#overview)
2. [Core Concepts](#core-concepts)
3. [Data Model](#data-model)
4. [Scope Resolution](#scope-resolution)
5. [API Design](#api-design)
6. [Security & Authorization](#security--authorization)
7. [Enhancements & Edge Cases](#enhancements--edge-cases)
8. [Migration Plan](#migration-plan)
9. [Frontend Integration Points](#frontend-integration-points)

---

## Overview

### Goal

Add an optional **"Company Spaces"** layer to Carpooly that allows users to carpool within their company (coworkers-only) in addition to personal use. This is an **additive feature** that does not disrupt existing personal carpooling functionality.

### Key Principles

- ✅ **Personal mode remains unchanged** - All existing APIs work exactly as before
- ✅ **Company mode is opt-in** - Users must explicitly join a company
- ✅ **Clean data separation** - Personal data (`company_id IS NULL`) vs Company data (`company_id = <uuid>`)
- ✅ **Backward compatible** - No breaking changes to existing endpoints
- ✅ **Multi-tenant isolation** - Company A users cannot see Company B data

---

## Core Concepts

### Scope Model

Every request operates in one of two scopes:

```typescript
type Scope = 
  | { type: 'personal' }
  | { type: 'company'; companyId: string; siteId?: string }
```

**Personal Scope:**
- All data with `company_id IS NULL`
- Default behavior when no company context is provided
- Works exactly like current Carpooly

**Company Scope:**
- All data with `company_id = <company_uuid>`
- Optionally filtered by `site_id` for site-specific operations
- Only accessible to users with active company membership

### Membership Model

Users can have:
- **Personal-only**: No company memberships
- **Company-only**: One or more company memberships, no personal usage
- **Both**: Personal carpools + company carpools (separate contexts)

---

## Data Model

### New Tables

#### 1. `companies`

Stores company/organization information.

```sql
CREATE TABLE companies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,                    -- e.g., 'Amazon'
  slug TEXT UNIQUE NOT NULL,             -- 'amazon' (URL-safe, lowercase)
  primary_domain TEXT NOT NULL,          -- 'amazon.com'
  additional_domains JSONB DEFAULT '[]', -- ['amazon.co.uk', 'amzn.com']
  logo_url TEXT,                         -- Optional branding
  settings JSONB NOT NULL DEFAULT '{}',  -- See settings schema below
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_companies_slug ON companies(slug);
CREATE INDEX idx_companies_domains ON companies USING GIN(additional_domains);
```

**Settings JSONB Schema:**
```json
{
  "require_invite_code": boolean,           // Default: false
  "allow_cross_site_matching": boolean,      // Default: true
  "default_timezone": string,                // Default: "UTC"
  "estimated_avg_commute_distance_km": number, // Default: 10
  "emission_factor_kg_co2_per_km": number   // Default: 0.2
}
```

#### 2. `sites`

Represents physical company locations (offices, warehouses, campuses).

```sql
CREATE TABLE sites (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  name TEXT NOT NULL,                      -- e.g., 'SJC14 Warehouse'
  code TEXT,                                -- 'SJC14' (short identifier)
  address TEXT,
  latitude DOUBLE PRECISION,
  longitude DOUBLE PRECISION,
  timezone TEXT,                           -- 'America/Los_Angeles'
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(company_id, code)                  -- Unique code per company
);

CREATE INDEX idx_sites_company ON sites(company_id);
CREATE INDEX idx_sites_company_active ON sites(company_id, is_active);
```

#### 3. `user_company_memberships`

Links users to companies (and optionally sites).

```sql
CREATE TABLE user_company_memberships (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  site_id UUID REFERENCES sites(id),        -- Optional; user picks later
  role TEXT NOT NULL DEFAULT 'employee',    -- 'employee' | 'site_admin' | 'company_admin'
  status TEXT NOT NULL DEFAULT 'active',    -- 'active' | 'pending' | 'invited' | 'inactive'
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(user_id, company_id)               -- One membership per user/company
);

CREATE INDEX idx_memberships_user ON user_company_memberships(user_id, status);
CREATE INDEX idx_memberships_company ON user_company_memberships(company_id, status);
CREATE INDEX idx_memberships_site ON user_company_memberships(site_id) WHERE site_id IS NOT NULL;
```

**Role Hierarchy:**
- `employee`: Can use company matching, create requests/carpools
- `site_admin`: Can view stats for their site, manage users at their site
- `company_admin`: Can view all company stats, manage all sites, manage all users

### Modified Tables (Add Company/Site Tagging)

#### 4. `user_matching_preferences`

**CRITICAL CHANGE:** Must support multiple preference records per user (personal + per-company).

**⚠️ PRIMARY KEY RESOLUTION:** To support both personal AND company preferences, we must change the primary key structure. However, this is done in a backward-compatible way.

**Migration Strategy:**
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
-- This handles NULL company_id for personal preferences
CREATE UNIQUE INDEX idx_preferences_user_company 
  ON user_matching_preferences(user_id, COALESCE(company_id, '00000000-0000-0000-0000-000000000000'::uuid));

CREATE INDEX idx_preferences_company ON user_matching_preferences(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_preferences_user ON user_matching_preferences(user_id); -- For backward compatibility queries
```

**Backward Compatibility:**
- ✅ All existing queries that filter by `user_id` continue to work (just need `AND company_id IS NULL` filter)
- ✅ Existing API contracts remain unchanged
- ✅ No breaking changes to response shapes
- ✅ Personal preferences are still uniquely identified by `user_id` (via unique index)

**Semantics:**
- `company_id IS NULL` → Personal preferences (one per user, identified by unique index on `user_id`)
- `company_id IS NOT NULL` → Company-specific preferences (one per user per company)
- **Users can have BOTH personal and company preferences simultaneously**

#### 5. `match_requests`

```sql
ALTER TABLE match_requests 
  ADD COLUMN company_id UUID REFERENCES companies(id),
  ADD COLUMN site_id UUID REFERENCES sites(id);

CREATE INDEX idx_match_requests_company ON match_requests(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_match_requests_company_site ON match_requests(company_id, site_id) WHERE company_id IS NOT NULL;
```

#### 6. `carpools`

```sql
ALTER TABLE carpools 
  ADD COLUMN company_id UUID REFERENCES companies(id),
  ADD COLUMN site_id UUID REFERENCES sites(id);

CREATE INDEX idx_carpools_company ON carpools(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_carpools_company_site ON carpools(company_id, site_id) WHERE company_id IS NOT NULL;
```

#### 7. `carpool_schedules`

```sql
ALTER TABLE carpool_schedules 
  ADD COLUMN company_id UUID REFERENCES companies(id),
  ADD COLUMN site_id UUID REFERENCES sites(id);

CREATE INDEX idx_schedules_company ON carpool_schedules(company_id) WHERE company_id IS NOT NULL;
```

#### 8. `carpool_rides`

```sql
ALTER TABLE carpool_rides 
  ADD COLUMN company_id UUID REFERENCES companies(id),
  ADD COLUMN site_id UUID REFERENCES sites(id);

CREATE INDEX idx_rides_company ON carpool_rides(company_id, start_time) WHERE company_id IS NOT NULL;
CREATE INDEX idx_rides_company_site ON carpool_rides(company_id, site_id, start_time) WHERE company_id IS NOT NULL;
```

**Tagging Propagation Rules:**
- `match_requests` → `carpools`: Copy `company_id`/`site_id` when creating carpool
- `carpools` → `schedules`/`rides`: Copy `company_id`/`site_id` when creating schedules/rides
- **Invariant:** If `carpool.company_id IS NOT NULL`, all related schedules/rides must have the same `company_id`

---

## Scope Resolution

### Centralized Scope Resolution

Backend will have a single helper function that determines scope from request context:

```go
type Scope struct {
    Type      string  // "personal" | "company"
    CompanyID *uuid.UUID
    SiteID    *uuid.UUID
}

func ResolveScope(r *http.Request, userID uuid.UUID) (Scope, error)
```

### Resolution Rules (Priority Order)

1. **URL Pattern** (Highest Priority)
   - `/api/company/{slug}/...` → Force company scope
   - Lookup company by `slug`, validate user membership

2. **Query Parameters**
   - `?scope=company&company_id=...` → Company scope
   - Validate user has active membership

3. **Request Body** (for POST/PUT)
   - `{ "scope": "company", "company_id": "..." }` → Company scope
   - Validate membership

4. **Default** (Lowest Priority)
   - No indicators → Personal scope

### Validation Rules

- If `company_id` provided but user lacks active membership → **403 Forbidden**
- If `site_id` provided but doesn't belong to `company_id` → **400 Bad Request**
- If URL has `/company/{slug}` but company not found → **404 Not Found**

---

## API Design

### New Endpoints

#### 1. `GET /api/me/company`

**Purpose:** Get user's company memberships and context.

**Response:**
```json
{
  "memberships": [
    {
      "company_id": "uuid-company-1",
      "company_name": "Amazon",
      "company_slug": "amazon",
      "role": "employee",
      "status": "active",
      "site": {
        "id": "uuid-site-1",
        "name": "SJC14 Warehouse",
        "code": "SJC14",
        "timezone": "America/Los_Angeles"
      }
    }
  ]
}
```

**Auth:** Requires authenticated user.

---

#### 2. `PUT /api/me/company-site`

**Purpose:** Update user's site selection within a company.

**Request:**
```json
{
  "company_id": "uuid-company-1",
  "site_id": "uuid-site-1"
}
```

**Validation:**
- User must have active membership in `company_id`
- `site_id` must belong to `company_id`
- If `site_id` is `null`, clears site selection

**Response:**
```json
{
  "company_id": "uuid-company-1",
  "site_id": "uuid-site-1",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

---

#### 3. `GET /api/companies/{companyId}/stats`

**Purpose:** Company-wide analytics (admin only).

**Auth:** Requires `company_admin` or `site_admin` role for that company.

**Response:**
```json
{
  "company_id": "uuid-company-1",
  "name": "Amazon",
  "total_users": 253,
  "active_users_last_30d": 137,
  "total_carpools": 42,
  "active_carpools": 29,
  "rides_last_30d": 412,
  "miles_saved_last_30d": 8200,
  "co2_saved_last_30d_kg": 2100
}
```

**Filters:**
- All counts filtered by `company_id`
- `active_users_last_30d`: Users who participated in rides in last 30 days
- `rides_last_30d`: Rides with `start_time >= now() - 30 days`

---

#### 4. `GET /api/companies/{companyId}/sites/{siteId}/stats`

**Purpose:** Site-specific analytics (admin only).

**Auth:** Requires `company_admin` or `site_admin` (for that site) role.

**Response:** Same shape as company stats, but filtered by `site_id`.

---

#### 5. `GET /api/companies/{companyId}/adoption`

**Purpose:** Adoption breakdown by site (admin only).

**Auth:** Requires `company_admin` role.

**Response:**
```json
{
  "sites": [
    {
      "site_id": "uuid-site-1",
      "site_name": "SJC14 Warehouse",
      "site_code": "SJC14",
      "users_onboarded": 120,
      "active_users_last_30d": 75,
      "carpools": 18,
      "rides_last_30d": 210
    }
  ]
}
```

---

### Modified Endpoints (Scope-Aware)

#### 6. `GET /api/matching/preferences`

**Current Behavior (Backward Compatible):**
- No query params → Returns personal preferences (`company_id IS NULL`)
- **Implementation:** Repository method filters by `company_id IS NULL`

**New Behavior:**
- `?scope=personal` → Personal preferences (explicit)
- `?scope=company&company_id=...` → Company-specific preferences

**Backend Implementation:**
```go
// Handler resolves scope and passes to repository
scope := ResolveScope(r, userID)
var companyID *uuid.UUID
if scope.Type == "company" {
    companyID = scope.CompanyID
} else {
    companyID = nil // Personal scope
}
prefs, err := repo.GetUserMatchingPreferences(ctx, userID, companyID)
```

**Repository Query (Safe):**
```sql
SELECT ... 
FROM user_matching_preferences 
WHERE user_id = $1 
  AND (company_id = $2 OR ($2 IS NULL AND company_id IS NULL))
-- If $2 is NULL, only returns personal preferences (company_id IS NULL)
-- If $2 is provided, only returns company preferences for that company
```

**Response (Personal):**
```json
{
  "user_id": "uuid-user-1",
  "company_id": null,
  "site_id": null,
  // ... existing preference fields
}
```

**Response (Company):**
```json
{
  "user_id": "uuid-user-1",
  "company_id": "uuid-company-1",
  "site_id": "uuid-site-1",
  // ... existing preference fields
}
```

**Response (Not Configured):**
```json
{
  "configured": false,
  "message": "Company preferences not set up. Please configure in company hub."
}
```

---

#### 7. `PUT /api/matching/preferences`

**Current Behavior:**
- Body without `company_id` → Upserts personal preferences

**New Behavior:**
- Body with `company_id` → Upserts company-specific preferences
- Validates user has active membership in `company_id`

**Request (Personal):**
```json
{
  // No company_id field
  "arrival_time": "09:00",
  "commute_days": ["Monday", "Wednesday"],
  // ... other fields
}
```

**Request (Company):**
```json
{
  "company_id": "uuid-company-1",
  "site_id": "uuid-site-1",  // Optional
  "arrival_time": "09:00",
  "commute_days": ["Monday", "Wednesday"],
  // ... other fields
}
```

---

#### 8. `POST /api/matching/find-matches`

**Current Behavior:**
- Returns potential matches across all users (personal scope)

**New Behavior:**
- Default (no scope) → Personal matches (backward compatible)
- `scope=company&company_id=...` → Company-only matches

**Request (Personal):**
```json
{
  // No scope/company_id fields
  // ... existing filter fields
}
```

**Request (Company):**
```json
{
  "scope": "company",
  "company_id": "uuid-company-1",
  "site_id": "uuid-site-1",  // Optional, for site-specific matching
  // ... existing filter fields
}
```

**Backend Filtering Logic (Company Scope):**
1. Only users with active `user_company_memberships` for `company_id`
2. Only users with `matching_preferences` where `company_id = :company_id`
3. If `site_id` provided and `allow_cross_site_matching = false`: Only same `site_id`
4. If `site_id IS NULL` in membership: Behavior depends on company settings (see Site Semantics)

**Response:** Same shape as current, but filtered to company users only.

---

#### 9. `GET /api/matching/requests`

**Current Behavior:**
- Returns personal requests (`company_id IS NULL`)
- **Implementation:** Repository method filters by `company_id IS NULL`

**New Behavior:**
- Default → Personal requests (backward compatible)
- `?scope=company&company_id=...` → Company requests

**Query Parameters:**
- `scope=personal` (default)
- `scope=company&company_id=...`

**Backend Implementation:**
```go
// Handler resolves scope
scope := ResolveScope(r, userID)
var companyID *uuid.UUID
if scope.Type == "company" {
    companyID = scope.CompanyID
} else {
    companyID = nil // Personal scope
}
incoming, outgoing, err := repo.GetMatchRequests(ctx, userID, companyID)
```

**Repository Query (Safe):**
```sql
-- Incoming requests
SELECT ... 
FROM match_requests mr
WHERE mr.to_user_id = $1 
  AND mr.status != 'expired'
  AND (mr.company_id = $2 OR ($2 IS NULL AND mr.company_id IS NULL))
-- If $2 is NULL, only returns personal requests
-- If $2 is provided, only returns requests for that company

-- Outgoing requests - same filter
```

**Response:** Same shape, but filtered by scope.

---

#### 10. `POST /api/matching/requests`

**Current Behavior:**
- Creates personal request (`company_id IS NULL`)

**New Behavior:**
- Body without `company_id` → Personal request
- Body with `company_id` → Company request

**Request (Personal):**
```json
{
  "to_user_id": "uuid-user-2",
  "potential_match_id": "uuid-match-1",
  "message": "Hi!",
  "carpool_name": "Morning Commute",
  "preferred_carpool_size": 4
  // No company_id
}
```

**Request (Company):**
```json
{
  "to_user_id": "uuid-user-2",
  "potential_match_id": "uuid-match-1",
  "message": "Hi!",
  "carpool_name": "Morning Commute",
  "preferred_carpool_size": 4,
  "company_id": "uuid-company-1",
  "site_id": "uuid-site-1"  // Optional
}
```

**Validation:**
- Both sender and recipient must have active membership in `company_id`
- If `site_id` provided, must belong to `company_id`

---

#### 11. `PUT /api/matching/requests/{id}` (Accept/Reject)

**Current Behavior:** Unchanged signature

**New Validation:**
- If request has `company_id`, ensure current user has membership in that company
- When accepted, carpool created with same `company_id`/`site_id` as request

---

#### 12. `GET /api/carpools/{id}`

**New Validation:**
- If carpool has `company_id IS NOT NULL`:
  - User must have active membership in that company
  - Otherwise → **403 Forbidden**

**Backend Implementation:**
```go
// Get carpool
carpool, err := repo.GetCarPool(ctx, carpoolID)
if err != nil {
    return 404
}

// If company carpool, verify membership
if carpool.CompanyID != nil {
    membership, err := getUserCompanyMembership(userID, *carpool.CompanyID)
    if err != nil || membership.Status != "active" {
        return 403 // Forbidden
    }
}

// Return carpool
```

**Note:** All carpool queries already filter by `company_id` through `GetUserCarpools()`, but individual carpool access must also be validated.

---

#### 13. `GET /api/carpools/{id}/schedules`

**New Validation:**
- Same as carpool access check
- Only returns schedules for that carpool (already scoped)

---

#### 14. `GET /api/carpools/{id}/rides/{date}`

**New Validation:**
- Same as carpool access check
- Only returns rides for that carpool (already scoped)

---

## Security & Authorization

### Company Membership Validation

**Every company-scoped endpoint must:**
1. Resolve scope from request
2. If `scope.type == "company"`:
   - Lookup `user_company_memberships` for `(user_id, company_id)`
   - Verify `status = 'active'`
   - Verify `role` is sufficient for operation
3. If validation fails → **403 Forbidden**

### Role-Based Access Control

| Endpoint | Required Role |
|----------|---------------|
| `GET /api/companies/{id}/stats` | `company_admin` or `site_admin` |
| `GET /api/companies/{id}/sites/{id}/stats` | `company_admin` or `site_admin` (for that site) |
| `GET /api/companies/{id}/adoption` | `company_admin` |
| All other company endpoints | `employee` or higher |

### Data Isolation Rules

**Critical:** Every query in company context must include `company_id` filter.

**Examples:**
- `SELECT * FROM carpools WHERE company_id = :company_id AND ...`
- `SELECT * FROM match_requests WHERE company_id = :company_id AND ...`
- Never mix personal and company data in same query result

**Test Assertions:**
- User from Company A cannot see Company B's carpools/requests/rides
- User from Company A cannot access Company B's admin stats

---

## Enhancements & Edge Cases

### 1. Site Semantics

**Problem:** What happens when `user_company_memberships.site_id IS NULL`?

**Solution (CHOSEN): Option A - Site Selection Required**

- **Default Behavior:** User **CANNOT** use company matching until they pick a site
- **Frontend UX:** Show prompt: "Please select your site to enable company matching"
- **Site Selection:** User can change their site selection at any time via `PUT /api/me/company-site`
- **Impact on Existing Data:** If user changes site:
  - Existing carpools/rides continue (no disruption)
  - New matches will use new site
  - Existing matches remain valid

**Cross-Site Matching:**
- If `allow_cross_site_matching = false`: Only match users with same `site_id`
- If `allow_cross_site_matching = true`: Match across all sites in company

**Site Selection Change Policy:**
- Users can change site at any time
- Change takes effect immediately for new matches
- Existing carpools/rides are not affected
- Frontend should show confirmation: "Changing your site will affect future matches. Continue?"

**Implementation:**
```go
func getAllowedMatchingSites(membership UserCompanyMembership, company Company) ([]uuid.UUID, error) {
    // Option A: Require site selection
    if membership.SiteID == nil {
        return nil, fmt.Errorf("site selection required for company matching")
    }
    
    if !company.Settings.AllowCrossSiteMatching {
        return []uuid.UUID{*membership.SiteID}, nil // Only same site
    }
    
    // Return all active sites for company
    return getActiveSiteIDs(company.ID), nil
}
```

---

### 2. Company Lifecycle Management

**Company Deactivated:**
- Block new `match_requests`/`carpools` with that `company_id`
- Existing rides continue (or disable based on policy)
- Admin APIs return **403 Forbidden**

**Site Deactivated (`is_active = false`):**
- Block new matches/carpools for that site
- Existing carpools/rides continue
- Users with that `site_id` should be prompted to select new site

**Membership Revoked:**
- User cannot see company carpools
- User cannot create new company requests
- User cannot access admin stats
- Existing carpools: **Decision needed** - Remove member or keep but block new actions?

---

### 3. Email Domain Auto-Detection

**Rules:**
1. Extract domain from user's **verified primary email** (not secondary emails)
2. Lookup companies where `primary_domain = domain` OR `domain IN additional_domains`
3. If match found and no membership exists:
   - Create `user_company_memberships` with `role='employee'`, `status='active'`
   - `site_id = NULL` (user picks later)
4. If multiple matches: Pick first company found, log warning
5. **Blacklist:** Never auto-match generic domains (`gmail.com`, `yahoo.com`, `outlook.com`, `hotmail.com`, etc.)

**Email Change Policy:**
- When user changes primary email:
  - Check if new domain matches a company
  - If yes and no membership exists: Create new membership
  - **Do NOT revoke existing memberships** (user may have multiple valid emails)
  - Log email change and membership creation

**Domain Change Policy:**
- When company domain changes:
  - Update `companies.primary_domain` or `additional_domains`
  - **Do NOT affect existing memberships** (users already members)
  - New users with new domain will be auto-matched

**Multiple Email Handling:**
- Only use **primary verified email** for auto-detection
- Secondary emails are ignored for auto-membership
- Users can manually join companies regardless of email domain

**Logging:**
- Log all auto-membership creations for audit
- Log email changes that trigger new memberships
- Log when auto-detection is skipped (blacklisted domain, existing membership, etc.)

**Error Handling:**
- If auto-detection fails (database error, etc.): Log error, continue without membership
- User can still manually join company via invite code

---

### 4. Company Slug Resolution

**Requirements:**
- `slug` must be UNIQUE, NOT NULL, URL-safe (lowercase, alphanumeric + hyphens)
- Indexed for fast lookups
- Helper function: `resolveCompanyFromSlug(slug) → (companyId, error)`

**URL Pattern:**
- `/api/company/{slug}/...` → Force company scope
- Lookup company by slug, validate membership

---

### 5. Settings Defaults & Validation

**Application-Level Defaults:**
```go
type CompanySettings struct {
    RequireInviteCode              bool    `json:"require_invite_code" default:"false"`
    AllowCrossSiteMatching         bool    `json:"allow_cross_site_matching" default:"true"`
    DefaultTimezone                string  `json:"default_timezone" default:"UTC"`
    EstimatedAvgCommuteDistanceKm  float64 `json:"estimated_avg_commute_distance_km" default:"10.0"`
    EmissionFactorKgCo2PerKm       float64 `json:"emission_factor_kg_co2_per_km" default:"0.2"`
}
```

**CO₂ Calculation:**
```
co2_saved_kg = ride_count * avg_distance_km * emission_factor_kg_co2_per_km
```

**Note:** This is approximate until route-level distance data is available.

---

## Migration Plan

### Phase 1: Add New Tables (No Behavior Change)
- Create `companies`, `sites`, `user_company_memberships` tables
- No existing code touched
- **Risk:** Low ✅

### Phase 2: Add Nullable Columns (Backward Compatible)
- Add `company_id`/`site_id` to existing tables (all nullable)
- Add unique index to `user_matching_preferences` (keep existing PK)
- Existing data remains valid (`company_id IS NULL`)
- **Risk:** Low ✅

### Phase 3: Add Query Filters (CRITICAL - Must Do Before Phase 4)

**This phase is broken into sub-phases for safety and verification:**

#### Phase 3a: Update `user_matching_preferences` Queries
- Update `GetUserMatchingPreferences()` to accept `companyID *uuid.UUID` parameter
- Add `WHERE company_id IS NULL` filter for personal scope
- Update `UpsertUserMatchingPreferences()` to handle `company_id` in conflict clause
- **Verification:**
  - [ ] All existing tests pass
  - [ ] Personal preferences still accessible
  - [ ] No data leaks (only returns `company_id IS NULL` for personal scope)
- **Rollback:** Revert method signatures and queries if issues found

#### Phase 3b: Update `match_requests` Queries
- Update `GetMatchRequests()` to accept `companyID *uuid.UUID` parameter
- Add `WHERE company_id IS NULL` filter to incoming/outgoing queries
- Update `GetMatchRequestsByUserID()` similarly
- **Verification:**
  - [ ] All existing tests pass
  - [ ] Personal requests still accessible
  - [ ] No data leaks (only returns `company_id IS NULL` requests)
- **Rollback:** Revert method signatures and queries if issues found

#### Phase 3c: Update `carpools` Queries
- Update `GetUserCarpools()` to accept `companyID *uuid.UUID` parameter
- Add `WHERE company_id IS NULL` filter
- Update all carpool retrieval methods
- **Verification:**
  - [ ] All existing tests pass
  - [ ] Personal carpools still accessible
  - [ ] No data leaks
- **Rollback:** Revert method signatures and queries if issues found

#### Phase 3d: Update `carpool_rides` Queries
- Update all ride queries to filter by `company_id` (through carpools join or denormalized column)
- Add `companyID` parameter where needed
- **Verification:**
  - [ ] All existing tests pass
  - [ ] Personal rides still accessible
  - [ ] No data leaks
- **Rollback:** Revert queries if issues found

#### Phase 3e: Update `carpool_schedules` Queries
- Update all schedule queries to filter by `company_id`
- Add `companyID` parameter where needed
- **Verification:**
  - [ ] All existing tests pass
  - [ ] Personal schedules still accessible
  - [ ] No data leaks
- **Rollback:** Revert queries if issues found

**Overall Phase 3 Verification:**
- [ ] All existing functionality works (personal scope)
- [ ] No data leaks between scopes
- [ ] All tests pass
- [ ] Performance is acceptable

**Risk:** Medium ⚠️ (requires thorough testing after each sub-phase)  
**MUST COMPLETE ALL SUB-PHASES BEFORE:** Enabling any company features (Phase 4+)

### Phase 4: Scope Resolution & Handler Updates
- Implement scope resolution helper
- Update all handlers to resolve scope and pass to repositories
- Add `/api/me/company` and `/api/me/company-site` endpoints
- **Risk:** Medium ⚠️ (requires careful testing)

### Phase 5: Membership Detection
- Implement email domain → company auto-detection
- Create memberships on user creation/login
- **Risk:** Low ✅ (additive only)

### Phase 6: Scope-Aware APIs (Company Features)
- Update matching/preferences/requests endpoints to accept scope
- Default to personal when scope not provided (backward compatible)
- Enable company matching, requests, carpools
- **Risk:** Medium ⚠️ (requires careful testing)
- **MUST COMPLETE AFTER:** Phase 3 (query filters)

### Phase 7: Tagging Propagation
- Ensure `company_id`/`site_id` propagate: requests → carpools → schedules → rides
- Add validation to enforce consistency
- **Risk:** Medium ⚠️ (data integrity critical)

### Phase 8: Admin Analytics
- Implement company/site stats endpoints
- Add role-based access control
- **Risk:** Low ✅ (new endpoints only)

### Critical Path

**DO NOT skip Phase 3!** It is required for safety:

```
Phase 1 (New Tables) → Phase 2 (Columns) → Phase 3 (Query Filters) → 
Phase 4 (Scope Resolution) → Phase 5 (Membership) → Phase 6 (Company APIs) → 
Phase 7 (Tagging) → Phase 8 (Analytics)
```

**Testing Requirements:**
- After Phase 3: Verify personal scope works (all existing functionality)
- After Phase 6: Verify company scope works (new functionality)
- After Phase 7: Verify data isolation (no cross-company leaks)

---

## Frontend Integration Points

### 1. Scope Indication

**Frontend must explicitly indicate scope in requests:**

**Option A: Query Parameters**
```
GET /api/matching/preferences?scope=company&company_id=...
```

**Option B: Request Body**
```json
POST /api/matching/find-matches
{
  "scope": "company",
  "company_id": "...",
  ...
}
```

**Option C: URL Pattern**
```
GET /api/company/amazon/matching/preferences
```

**Recommendation:** Use URL pattern for company routes, query params for mixed routes.

---

### 2. Company Context Selection

**Frontend flow:**
1. User logs in
2. Call `GET /api/me/company` to get memberships
3. If multiple memberships, show selector: "Which company space?"
4. User selects → Frontend stores `activeCompanyId` in state
5. All subsequent requests include `company_id` when in company context

---

### 3. Site Selection

**Flow:**
1. User enters company space
2. If `membership.site_id IS NULL`:
   - Show site picker: "Select your site to enable matching"
   - Call `PUT /api/me/company-site` with selected site
3. If site selected, enable company matching

---

### 4. UI Separation

**Frontend should maintain clear separation:**
- **Personal Hub:** `/matching`, `/carpools` (no company context)
- **Company Hub:** `/company/{slug}/matching`, `/company/{slug}/carpools`

**Data Display:**
- Never mix personal and company carpools in same list
- Calendar views should be scope-aware (personal vs company)

---

### 5. Error Handling

**Frontend should handle:**
- **403 Forbidden:** User lacks membership or insufficient role
- **404 Not Found:** Company/site not found
- **400 Bad Request:** Invalid `company_id`/`site_id` combination

**User Feedback:**
- "You need to join this company to access this feature"
- "Please select your site to enable company matching"

---

## ⚠️ CRITICAL SAFETY REQUIREMENTS

### Backward Compatibility Guarantees

**ALL existing functionality MUST continue to work unchanged:**

1. **Default Behavior:** All endpoints default to **personal scope** (`company_id IS NULL`) when no scope is provided
2. **Query Filters:** All existing queries MUST add `WHERE company_id IS NULL` filter to prevent data leaks
3. **Primary Key:** `user_matching_preferences.user_id` remains PRIMARY KEY (do not change)
4. **Method Signatures:** Add optional `companyID *uuid.UUID` parameters, default to `nil` (personal)

### Required Code Changes (MUST DO BEFORE ENABLING COMPANY FEATURES)

#### 1. Repository Method Updates

**All repository methods that query scoped tables MUST be updated:**

**A. `GetUserMatchingPreferences()` - Add scope filtering**
```go
// BEFORE (Current - will break with company data)
func (r *MatchingRepository) GetUserMatchingPreferences(ctx context.Context, userID string) (*models.UserMatchingPreferences, error) {
    query := `SELECT ... FROM user_matching_preferences WHERE user_id = $1`
    // Returns first row found - could be company preferences!
}

// AFTER (Safe - filters by scope)
func (r *MatchingRepository) GetUserMatchingPreferences(ctx context.Context, userID string, companyID *uuid.UUID) (*models.UserMatchingPreferences, error) {
    query := `
        SELECT ... 
        FROM user_matching_preferences 
        WHERE user_id = $1 
          AND (company_id = $2 OR ($2 IS NULL AND company_id IS NULL))
    `
    // If companyID is nil, get personal preferences (company_id IS NULL)
    // If companyID provided, get company-specific preferences
}
```

**B. `UpsertUserMatchingPreferences()` - Handle company_id in conflict**
```go
// BEFORE (Current - will break with company data)
func (r *MatchingRepository) UpsertUserMatchingPreferences(ctx context.Context, prefs *models.UserMatchingPreferences) error {
    query := `... ON CONFLICT (user_id) DO UPDATE ...`
    // Only works for personal preferences
}

// AFTER (Safe - handles both personal and company)
func (r *MatchingRepository) UpsertUserMatchingPreferences(ctx context.Context, prefs *models.UserMatchingPreferences, companyID *uuid.UUID) error {
    if companyID == nil {
        // Personal preferences - use existing ON CONFLICT (user_id)
        query := `... ON CONFLICT (user_id) DO UPDATE ...`
    } else {
        // Company preferences - use unique index conflict
        query := `... ON CONFLICT (user_id, company_id) DO UPDATE ...`
    }
}
```

**C. `GetMatchRequests()` - Add company_id filter**
```go
// BEFORE (Current - returns both personal and company requests)
func (r *MatchingRepository) GetMatchRequests(ctx context.Context, userID string) ([]*models.MatchRequest, []*models.MatchRequest, error) {
    incomingQuery := `SELECT ... FROM match_requests WHERE to_user_id = $1 AND status != 'expired'`
    // Returns ALL requests - data leak!
}

// AFTER (Safe - filters by scope)
func (r *MatchingRepository) GetMatchRequests(ctx context.Context, userID string, companyID *uuid.UUID) ([]*models.MatchRequest, []*models.MatchRequest, error) {
    incomingQuery := `
        SELECT ... 
        FROM match_requests mr
        WHERE mr.to_user_id = $1 
          AND mr.status != 'expired'
          AND (mr.company_id = $2 OR ($2 IS NULL AND mr.company_id IS NULL))
    `
    // If companyID is nil, only return personal requests
    // If companyID provided, only return requests for that company
}
```

**D. `GetUserCarpools()` - Add company_id filter**
```go
// BEFORE (Current - returns both personal and company carpools)
func (r *CarPoolRepository) GetUserCarpools(ctx context.Context, userID uuid.UUID) ([]models.Carpool, error) {
    query := `SELECT ... FROM carpools c JOIN carpool_members cm ON c.id = cm.carpool_id WHERE cm.user_id = $1`
    // Returns ALL carpools - data leak!
}

// AFTER (Safe - filters by scope)
func (r *CarPoolRepository) GetUserCarpools(ctx context.Context, userID uuid.UUID, companyID *uuid.UUID) ([]models.Carpool, error) {
    query := `
        SELECT ... 
        FROM carpools c
        JOIN carpool_members cm ON c.id = cm.carpool_id
        WHERE cm.user_id = $1
          AND (c.company_id = $2 OR ($2 IS NULL AND c.company_id IS NULL))
    `
}
```

**E. All `carpool_rides` queries - Add company_id filter**
```go
// All queries that fetch rides must filter by company_id
// Either join through carpools or use denormalized company_id column

// Example: Get rides for carpool
query := `
    SELECT cr.* 
    FROM carpool_rides cr
    JOIN carpools c ON cr.carpool_id = c.id
    WHERE cr.carpool_id = $1
      AND (c.company_id = $2 OR ($2 IS NULL AND c.company_id IS NULL))
`
```

**F. All `carpool_schedules` queries - Add company_id filter**
```go
// Similar to rides - filter through carpools or use denormalized column
query := `
    SELECT cs.* 
    FROM carpool_schedules cs
    JOIN carpools c ON cs.carpool_id = c.id
    WHERE cs.carpool_id = $1
      AND (c.company_id = $2 OR ($2 IS NULL AND c.company_id IS NULL))
`
```

#### 2. Handler Updates

**All handlers must resolve scope and pass to repositories:**

```go
// Example: Matching preferences handler
func (h *MatchingHandler) GetMatchingPreferences(w http.ResponseWriter, r *http.Request) {
    // 1. Get authenticated user
    userID := getAuthenticatedUserID(r)
    
    // 2. Resolve scope from request
    scope := ResolveScope(r, userID)
    var companyID *uuid.UUID
    if scope.Type == "company" {
        companyID = scope.CompanyID
    } else {
        companyID = nil // Personal scope
    }
    
    // 3. Call repository with scope
    prefs, err := h.matchingRepo.GetUserMatchingPreferences(r.Context(), userID, companyID)
    // ...
}
```

#### 3. Scope Resolution Helper

**Create centralized scope resolution:**

```go
type Scope struct {
    Type      string      // "personal" | "company"
    CompanyID *uuid.UUID
    SiteID    *uuid.UUID
}

func ResolveScope(r *http.Request, userID uuid.UUID) (Scope, error) {
    // Priority 1: URL pattern /api/company/{slug}/...
    if strings.HasPrefix(r.URL.Path, "/api/company/") {
        slug := extractSlugFromPath(r.URL.Path)
        company, err := getCompanyBySlug(slug)
        if err != nil {
            return Scope{}, err
        }
        // Validate user has membership
        membership, err := getUserCompanyMembership(userID, company.ID)
        if err != nil {
            return Scope{}, fmt.Errorf("user not member of company")
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
        companyID, err := uuid.Parse(companyIDStr)
        if err != nil {
            return Scope{}, fmt.Errorf("invalid company_id")
        }
        // Validate membership
        membership, err := getUserCompanyMembership(userID, companyID)
        if err != nil {
            return Scope{}, fmt.Errorf("user not member of company")
        }
        return Scope{
            Type:      "company",
            CompanyID: &companyID,
            SiteID:    membership.SiteID,
        }, nil
    }
    
    // Priority 3: Request body (for POST/PUT)
    // ... similar logic
    
    // Default: Personal scope
    return Scope{Type: "personal"}, nil
}
```

### Safety Checklist

**Before enabling company features, verify:**

- [ ] All repository methods have `companyID *uuid.UUID` parameter
- [ ] All queries filter by `company_id IS NULL` when `companyID == nil`
- [ ] All handlers resolve scope and pass to repositories
- [ ] Scope resolution helper is implemented and tested
- [ ] Personal scope is default (when no scope provided)
- [ ] All existing tests pass with new filters
- [ ] New tests verify data isolation (personal vs company)

**See `COMPANY_SPACES_SAFETY_ANALYSIS.md` for detailed safety requirements.**

---

## API Contract Documentation

### Backward Compatibility Guarantees

**ALL existing endpoints work unchanged when no scope is provided:**

1. ✅ `GET /api/matching/preferences` (no params) → Returns personal preferences
2. ✅ `GET /api/matching/requests` (no params) → Returns personal requests  
3. ✅ `GET /api/carpools/users/{userId}` → Returns personal carpools only
4. ✅ `POST /api/matching/requests` (no company_id) → Creates personal request
5. ✅ All existing response shapes remain unchanged (just filtered by scope)

**Error Response Format (Standardized):**

```json
{
  "error": "ERROR_CODE",
  "message": "Human-readable error message",
  "code": "MACHINE_READABLE_CODE",
  "field": "field_name",  // Optional, for validation errors
  "company_id": "uuid-company-1"  // Optional, for company-related errors
}
```

**Common Error Codes:**
- `FORBIDDEN` - User lacks required permissions
- `BAD_REQUEST` - Invalid request parameters
- `NOT_FOUND` - Resource not found
- `MISSING_COMPANY_MEMBERSHIP` - User not member of company
- `INVALID_COMPANY_ID` - Invalid company_id format
- `SITE_SELECTION_REQUIRED` - User must select site before matching

### Complete API Examples

See individual endpoint sections above for complete request/response examples with error handling.

---

## Questions for Frontend Team

1. **URL Structure:** Do you prefer `/api/company/{slug}/...` or query params `?scope=company&company_id=...`?

2. **Scope Indication:** How do you want to pass scope? URL pattern, header, query param, or body?

3. **Mixed Usage:** How should users switch between personal and company contexts? Tab switcher? Separate navigation?

4. **Analytics UI:** What admin dashboard features do you need beyond the stats endpoints?

**Note:** Site selection policy and email domain policy have been clarified in the blueprint (see Enhancements section).

---

## Implementation Safety Checklist

### Before Starting Implementation

- [ ] Review `COMPANY_SPACES_SAFETY_ANALYSIS.md` for detailed safety requirements
- [ ] Understand that Phase 3 (Query Filters) is **MANDATORY** before enabling company features
- [ ] Plan testing strategy for backward compatibility

### During Implementation

- [ ] Phase 1: Create new tables (test: tables exist)
- [ ] Phase 2: Add nullable columns (test: existing data still accessible)
- [ ] **Phase 3: Add query filters (test: ALL existing functionality still works)**
- [ ] Phase 4: Implement scope resolution (test: personal scope works)
- [ ] Phase 5: Add membership detection (test: memberships created correctly)
- [ ] Phase 6: Enable company APIs (test: company scope works, personal still works)
- [ ] Phase 7: Verify tagging propagation (test: data consistency)
- [ ] Phase 8: Add admin analytics (test: role-based access)

### Testing Requirements

**Critical Tests:**
- [ ] Personal users only see personal data (`company_id IS NULL`)
- [ ] Company users only see their company's data
- [ ] Users cannot access other companies' data
- [ ] All existing endpoints work unchanged (backward compatibility)
- [ ] No data leaks between personal and company scopes
- [ ] No data leaks between different companies

---

## Next Steps

1. **Frontend Review:** Review this blueprint and provide feedback
2. **API Contract Finalization:** Lock down exact request/response shapes
3. **Safety Review:** Review `COMPANY_SPACES_SAFETY_ANALYSIS.md` with team
4. **Implementation:** Backend implements in phases (see Migration Plan)
5. **Integration Testing:** Test end-to-end flows with frontend
6. **Security Audit:** Verify data isolation and access control

---

**Document Status:** Ready for Frontend Review (with Safety Requirements)  
**Last Updated:** 2025-01-XX  
**Contact:** [Backend Team]  
**Related Documents:** `COMPANY_SPACES_SAFETY_ANALYSIS.md`

