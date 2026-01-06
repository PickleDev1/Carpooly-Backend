# Company Spaces Feature - Safety Analysis & Breaking Changes

**CRITICAL:** This document identifies ALL potential breaking changes and provides fixes to ensure 100% backward compatibility.

---

## 🚨 CRITICAL BREAKING CHANGES IDENTIFIED

### 1. `user_matching_preferences` - PRIMARY KEY CHANGE

**Current State:**
- `user_id` is PRIMARY KEY (one row per user)
- Code uses `QueryRow` expecting exactly one row
- `ON CONFLICT (user_id)` in upsert

**Blueprint Change:**
- Wants composite key `(user_id, company_id)` to support multiple preference records per user

**BREAKING IMPACT:**
- ❌ `GetUserMatchingPreferences()` uses `QueryRow` - will fail if multiple rows exist
- ❌ `UpsertUserMatchingPreferences()` uses `ON CONFLICT (user_id)` - will break with new PK
- ❌ All existing code assumes one preference record per user

**SAFE FIX:**
- **DO NOT change the primary key structure**
- Instead: Add `company_id` as nullable column, keep `user_id` as PK
- Add unique constraint: `(user_id, COALESCE(company_id, '00000000-0000-0000-0000-000000000000'::uuid))`
- **Modify queries to filter by company_id:**
  - `GetUserMatchingPreferences()`: Add `AND company_id IS NULL` for personal scope
  - `UpsertUserMatchingPreferences()`: Add `company_id` to conflict clause when provided

**Migration:**
```sql
-- Add nullable columns (SAFE)
ALTER TABLE user_matching_preferences 
  ADD COLUMN company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
  ADD COLUMN site_id UUID REFERENCES sites(id);

-- Add unique constraint (handles NULL company_id for personal)
CREATE UNIQUE INDEX idx_preferences_user_company 
  ON user_matching_preferences(user_id, COALESCE(company_id, '00000000-0000-0000-0000-000000000000'::uuid));

-- DO NOT drop existing PK - keep user_id as PK for backward compatibility
```

**Code Changes Required:**
```go
// GetUserMatchingPreferences - Add scope parameter
func (r *MatchingRepository) GetUserMatchingPreferences(ctx context.Context, userID string, companyID *uuid.UUID) (*models.UserMatchingPreferences, error) {
    query := `
        SELECT ... 
        FROM user_matching_preferences 
        WHERE user_id = $1 AND (company_id = $2 OR ($2 IS NULL AND company_id IS NULL))
    `
    // If companyID is nil, get personal preferences (company_id IS NULL)
    // If companyID provided, get company-specific preferences
}

// UpsertUserMatchingPreferences - Add company_id handling
func (r *MatchingRepository) UpsertUserMatchingPreferences(ctx context.Context, prefs *models.UserMatchingPreferences, companyID *uuid.UUID) error {
    if companyID == nil {
        // Personal preferences - use existing ON CONFLICT (user_id)
        query := `... ON CONFLICT (user_id) DO UPDATE ...`
    } else {
        // Company preferences - use composite conflict
        query := `... ON CONFLICT (user_id, company_id) DO UPDATE ...`
    }
}
```

---

### 2. `match_requests` Queries - Missing `company_id` Filter

**Current State:**
- Queries in `GetMatchRequests()` don't filter by `company_id`
- Will return BOTH personal and company requests (data leak)

**BREAKING IMPACT:**
- ❌ Personal users will see company requests
- ❌ Company users will see personal requests
- ❌ Security violation - cross-company data exposure

**SAFE FIX:**
- **Add `WHERE company_id IS NULL` to all existing queries** (for personal scope)
- Add new overloaded methods for company scope
- Ensure default behavior (no scope param) = personal only

**Code Changes Required:**
```go
// GetMatchRequests - Add company_id filter for personal scope
func (r *MatchingRepository) GetMatchRequests(ctx context.Context, userID string, companyID *uuid.UUID) ([]*models.MatchRequest, []*models.MatchRequest, error) {
    // Incoming requests
    incomingQuery := `
        SELECT ... 
        FROM match_requests mr
        JOIN users u ON mr.from_user_id = u.id
        WHERE mr.to_user_id = $1 
          AND mr.status != 'expired'
          AND (mr.company_id = $2 OR ($2 IS NULL AND mr.company_id IS NULL))
        ORDER BY mr.created_at DESC
    `
    
    // Outgoing requests - same filter
    outgoingQuery := `
        SELECT ... 
        FROM match_requests mr
        JOIN users u ON mr.to_user_id = u.id
        WHERE mr.from_user_id = $1 
          AND mr.status != 'expired'
          AND (mr.company_id = $2 OR ($2 IS NULL AND mr.company_id IS NULL))
        ORDER BY mr.created_at DESC
    `
    
    // If companyID is nil, only return personal requests (company_id IS NULL)
    // If companyID provided, only return requests for that company
}
```

---

### 3. `carpools` Queries - Missing `company_id` Filter

**Current State:**
- `GetUserCarpools()` doesn't filter by `company_id`
- Will return BOTH personal and company carpools

**BREAKING IMPACT:**
- ❌ Personal users will see company carpools
- ❌ Company users will see personal carpools
- ❌ Security violation

**SAFE FIX:**
- Add `WHERE company_id IS NULL` to existing queries (for personal scope)
- Add scope parameter to methods

**Code Changes Required:**
```go
// GetUserCarpools - Add company_id filter
func (r *CarPoolRepository) GetUserCarpools(ctx context.Context, userID uuid.UUID, companyID *uuid.UUID) ([]models.Carpool, error) {
    query := `
        SELECT DISTINCT ... 
        FROM carpools c
        JOIN carpool_members cm ON c.id = cm.carpool_id
        WHERE cm.user_id = $1
          AND (c.company_id = $2 OR ($2 IS NULL AND c.company_id IS NULL))
    `
}
```

---

### 4. `carpool_rides` Queries - Missing `company_id` Filter

**Current State:**
- Various queries don't filter by `company_id`
- Will return rides from both personal and company carpools

**BREAKING IMPACT:**
- ❌ Data leak across scopes

**SAFE FIX:**
- Add filters to all ride queries
- Ensure carpool-level filtering propagates to rides

---

## ✅ SAFE CHANGES (No Breaking Impact)

### 1. New Tables
- `companies`, `sites`, `user_company_memberships`
- ✅ Completely new - no impact on existing code

### 2. Nullable Columns Added
- Adding `company_id`, `site_id` as nullable columns to existing tables
- ✅ Existing data remains valid (all NULL = personal)
- ✅ Existing queries work (just need to add filters)

### 3. New Endpoints
- `/api/me/company`, `/api/companies/{id}/stats`, etc.
- ✅ Completely new - no impact on existing endpoints

---

## 🔧 REQUIRED CODE CHANGES FOR SAFETY

### Phase 1: Add Filters to Existing Queries (CRITICAL)

**All queries that read from scoped tables MUST filter by `company_id`:**

1. **`GetUserMatchingPreferences()`**
   - Add `AND company_id IS NULL` for personal scope
   - Add overload for company scope

2. **`GetMatchRequests()`**
   - Add `AND (company_id IS NULL OR company_id = :company_id)` filter
   - Default to `company_id IS NULL` for backward compatibility

3. **`GetUserCarpools()`**
   - Add `AND (company_id IS NULL OR company_id = :company_id)` filter
   - Default to `company_id IS NULL`

4. **All `carpool_rides` queries**
   - Add `company_id` filter (can join through carpools or denormalize)

5. **All `carpool_schedules` queries**
   - Add `company_id` filter

### Phase 2: Update Method Signatures

**Add optional `companyID *uuid.UUID` parameter to:**
- `GetUserMatchingPreferences(ctx, userID, companyID)`
- `UpsertUserMatchingPreferences(ctx, prefs, companyID)`
- `GetMatchRequests(ctx, userID, companyID)`
- `GetUserCarpools(ctx, userID, companyID)`
- All other scoped queries

**Default behavior:** If `companyID == nil`, filter by `company_id IS NULL` (personal scope)

### Phase 3: Update Handlers

**All handlers must:**
1. Resolve scope from request (personal vs company)
2. Pass `companyID` to repository methods
3. Default to `nil` (personal) if no scope provided

---

## 📋 MIGRATION CHECKLIST

### Database Migrations (Safe)
- [x] Create `companies` table
- [x] Create `sites` table
- [x] Create `user_company_memberships` table
- [x] Add nullable `company_id`, `site_id` to `match_requests`
- [x] Add nullable `company_id`, `site_id` to `carpools`
- [x] Add nullable `company_id`, `site_id` to `carpool_schedules`
- [x] Add nullable `company_id`, `site_id` to `carpool_rides`
- [x] Add nullable `company_id`, `site_id` to `user_matching_preferences`
- [x] Add unique index on `(user_id, COALESCE(company_id, ...))` for preferences
- [ ] **DO NOT drop existing PK on `user_matching_preferences`**

### Code Changes (Required for Safety)
- [ ] Update `GetUserMatchingPreferences()` to filter by `company_id`
- [ ] Update `UpsertUserMatchingPreferences()` to handle `company_id` in conflict clause
- [ ] Update `GetMatchRequests()` to filter by `company_id`
- [ ] Update `GetUserCarpools()` to filter by `company_id`
- [ ] Update all `carpool_rides` queries to filter by `company_id`
- [ ] Update all `carpool_schedules` queries to filter by `company_id`
- [ ] Add scope resolution middleware/helper
- [ ] Update all handlers to pass scope to repositories

### Testing (Critical)
- [ ] Test that personal users only see personal data
- [ ] Test that company users only see their company's data
- [ ] Test that users cannot access other companies' data
- [ ] Test backward compatibility (no scope = personal)
- [ ] Test that existing personal data remains accessible

---

## 🎯 RECOMMENDED APPROACH

### Option A: Backward-Compatible Implementation (SAFEST)

1. **Keep existing PK structure** - Don't change `user_id` as PK
2. **Add filters to ALL queries** - Default to `company_id IS NULL`
3. **Add scope parameters** - Optional `companyID` parameter, defaults to `nil`
4. **Gradual rollout** - Test personal scope first, then add company scope

### Option B: Blueprint Approach (RISKY)

- Changes PK structure
- Requires extensive code refactoring
- Higher risk of breaking existing functionality

**RECOMMENDATION: Use Option A**

---

## ⚠️ FINAL WARNING

**DO NOT implement the blueprint as-is without these fixes:**

1. ❌ Changing `user_matching_preferences` PK will break existing code
2. ❌ Not filtering queries by `company_id` will cause data leaks
3. ❌ Not defaulting to personal scope will break existing functionality

**MUST DO:**
1. ✅ Keep existing PK structure
2. ✅ Add `company_id IS NULL` filters to all existing queries
3. ✅ Default all methods to personal scope
4. ✅ Test thoroughly before enabling company features

---

**Status:** ⚠️ **BLUEPRINT REQUIRES MODIFICATIONS FOR SAFETY**

