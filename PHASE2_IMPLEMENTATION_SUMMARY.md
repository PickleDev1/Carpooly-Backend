# Phase 2 Implementation Summary - Database Schema (Add Columns)

**Date:** 2025-01-XX  
**Status:** ✅ **COMPLETE**

---

## 📋 What Was Implemented

### Migration Files Created

1. ✅ **`migrations/026_add_company_columns_to_preferences.sql`**
   - Adds surrogate `id` as new PRIMARY KEY (allows multiple preference records per user)
   - Adds `company_id` and `site_id` columns
   - Creates unique constraint for (user_id, company_id) combinations

2. ✅ **`migrations/027_add_company_columns_to_match_requests.sql`**
   - Adds `company_id` and `site_id` columns to `match_requests` table

3. ✅ **`migrations/028_add_company_columns_to_carpools.sql`**
   - Adds `company_id` and `site_id` columns to `carpools` table

4. ✅ **`migrations/029_add_company_columns_to_schedules.sql`**
   - Adds `company_id` and `site_id` columns to `carpool_schedules` table (denormalized)

5. ✅ **`migrations/030_add_company_columns_to_rides.sql`**
   - Adds `company_id` and `site_id` columns to `carpool_rides` table (denormalized)

---

## 🔍 Detailed Explanation

### 1. User Matching Preferences (026) - Most Complex

**Why This Is Special:**
- Currently has `user_id` as PRIMARY KEY (one row per user)
- Need to support multiple rows per user (personal + per-company)
- Must maintain backward compatibility

**What We Did:**
1. **Added surrogate `id` column** - New UUID primary key
2. **Populated IDs** - All existing rows get a UUID
3. **Dropped old PK** - Removed `user_id` as primary key
4. **Created new PK** - `id` is now the primary key
5. **Added company columns** - `company_id` and `site_id` (nullable)
6. **Created unique constraint** - Prevents duplicate (user_id, company_id) combinations
   - Uses `COALESCE(company_id, '00000000-0000-0000-0000-000000000000'::uuid)` to handle NULL
   - Personal preferences: `company_id IS NULL` → one per user
   - Company preferences: `company_id IS NOT NULL` → one per user per company

**Result:**
- ✅ Users can have personal preferences (`company_id IS NULL`)
- ✅ Users can have company-specific preferences (`company_id = <uuid>`)
- ✅ Backward compatible - existing queries still work (just need filter in Phase 3)

---

### 2. Match Requests (027)

**What We Did:**
- Added nullable `company_id` and `site_id` columns
- Created partial indexes for performance
- All existing rows get `company_id = NULL` (personal requests)

**Result:**
- ✅ Personal requests: `company_id IS NULL`
- ✅ Company requests: `company_id = <uuid>`
- ✅ Backward compatible - existing queries still work

---

### 3. Carpools (028)

**What We Did:**
- Added nullable `company_id` and `site_id` columns
- Created partial indexes for performance
- All existing rows get `company_id = NULL` (personal carpools)

**Result:**
- ✅ Personal carpools: `company_id IS NULL`
- ✅ Company carpools: `company_id = <uuid>`
- ✅ Backward compatible - existing queries still work

---

### 4. Carpool Schedules (029)

**What We Did:**
- Added nullable `company_id` and `site_id` columns (denormalized from carpools)
- Created partial index for performance
- All existing rows get `company_id = NULL` (personal schedules)

**Why Denormalized:**
- Avoids JOINs in analytics queries
- Faster queries for company stats
- Will be populated from carpools in Phase 7

**Result:**
- ✅ Personal schedules: `company_id IS NULL`
- ✅ Company schedules: `company_id = <uuid>`
- ✅ Backward compatible - existing queries still work

---

### 5. Carpool Rides (030)

**What We Did:**
- Added nullable `company_id` and `site_id` columns (denormalized from carpools)
- Created composite indexes with `start_time` for analytics queries
- All existing rows get `company_id = NULL` (personal rides)

**Why Denormalized:**
- Avoids JOINs in analytics queries
- Faster queries for company stats (especially time-based)
- Will be populated from carpools in Phase 7

**Result:**
- ✅ Personal rides: `company_id IS NULL`
- ✅ Company rides: `company_id = <uuid>`
- ✅ Backward compatible - existing queries still work

---

## ✅ Safety Guarantees

### 1. All Columns Are Nullable
- ✅ `company_id` and `site_id` are nullable
- ✅ All existing rows get `NULL` (personal scope)
- ✅ No data loss or corruption

### 2. All Existing Data Remains Valid
- ✅ All existing rows have `company_id IS NULL`
- ✅ All existing rows have `site_id IS NULL`
- ✅ Existing queries continue to work (just need filter in Phase 3)

### 3. Foreign Keys Are Safe
- ✅ Foreign keys reference `companies` and `sites` tables (created in Phase 1)
- ✅ `ON DELETE CASCADE` only on `user_matching_preferences.company_id` (if company deleted, preferences deleted)
- ✅ Other foreign keys don't cascade (safer - prevents accidental data loss)

### 4. Indexes Are Optimized
- ✅ Partial indexes (`WHERE company_id IS NOT NULL`) - only index company data
- ✅ Composite indexes for common query patterns
- ✅ No performance impact on existing queries (indexes only used when filtering by company_id)

---

## 🧪 Verification Steps

### Step 1: Run Migrations

```bash
# Run migrations in order
psql -d your_database -f migrations/026_add_company_columns_to_preferences.sql
psql -d your_database -f migrations/027_add_company_columns_to_match_requests.sql
psql -d your_database -f migrations/028_add_company_columns_to_carpools.sql
psql -d your_database -f migrations/029_add_company_columns_to_schedules.sql
psql -d your_database -f migrations/030_add_company_columns_to_rides.sql
```

### Step 2: Verify Columns Exist

```sql
-- Check user_matching_preferences
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'user_matching_preferences'
  AND column_name IN ('id', 'company_id', 'site_id');

-- Check match_requests
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'match_requests'
  AND column_name IN ('company_id', 'site_id');

-- Check carpools
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'carpools'
  AND column_name IN ('company_id', 'site_id');

-- Check carpool_schedules
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'carpool_schedules'
  AND column_name IN ('company_id', 'site_id');

-- Check carpool_rides
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'carpool_rides'
  AND column_name IN ('company_id', 'site_id');
```

### Step 3: Verify All Existing Data Has company_id IS NULL

```sql
-- Verify all existing data has company_id IS NULL (personal scope)
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

-- Expected: All company_rows should be 0 (no company data yet)
```

### Step 4: Verify Primary Key Change (user_matching_preferences)

```sql
-- Verify new primary key exists
SELECT constraint_name, constraint_type
FROM information_schema.table_constraints
WHERE table_name = 'user_matching_preferences'
  AND constraint_type = 'PRIMARY KEY';

-- Expected: Should show PRIMARY KEY on 'id' column

-- Verify unique constraint exists
SELECT indexname, indexdef
FROM pg_indexes
WHERE tablename = 'user_matching_preferences'
  AND indexname = 'idx_preferences_user_company';

-- Expected: Should show unique index on (user_id, COALESCE(company_id, ...))
```

### Step 5: Verify Indexes Created

```sql
-- Check all indexes were created
SELECT tablename, indexname
FROM pg_indexes
WHERE tablename IN ('user_matching_preferences', 'match_requests', 'carpools', 'carpool_schedules', 'carpool_rides')
  AND indexname LIKE '%company%'
ORDER BY tablename, indexname;

-- Expected: Should show all company-related indexes
```

### Step 6: Test Multiple Preferences Per User (user_matching_preferences)

```sql
-- Test that a user can have both personal and company preferences
-- (Requires existing user and company)
-- 
-- 1. Verify user has personal preference (company_id IS NULL)
-- 2. Insert company preference (company_id = <uuid>)
-- 3. Verify both exist
-- 4. Try to insert duplicate (should fail unique constraint)
```

### Step 7: Verify Existing Functionality Still Works

```bash
# Run existing tests
go test ./...

# Or manually test existing endpoints
curl -X GET http://localhost:8080/api/matching/preferences
curl -X GET http://localhost:8080/api/matching/requests
curl -X GET http://localhost:8080/api/carpools/users/{userID}
# Should work exactly as before
```

---

## 📊 Database Schema After Phase 2

### Modified Tables

```
user_matching_preferences
├── id (UUID, PK) ← NEW PRIMARY KEY
├── user_id (UUID) ← No longer PK, but still indexed
├── company_id (UUID, NULLABLE, FK → companies.id) ← NEW
├── site_id (UUID, NULLABLE, FK → sites.id) ← NEW
└── ... (existing columns unchanged)

match_requests
├── ... (existing columns unchanged)
├── company_id (UUID, NULLABLE, FK → companies.id) ← NEW
└── site_id (UUID, NULLABLE, FK → sites.id) ← NEW

carpools
├── ... (existing columns unchanged)
├── company_id (UUID, NULLABLE, FK → companies.id) ← NEW
└── site_id (UUID, NULLABLE, FK → sites.id) ← NEW

carpool_schedules
├── ... (existing columns unchanged)
├── company_id (UUID, NULLABLE, FK → companies.id) ← NEW (denormalized)
└── site_id (UUID, NULLABLE, FK → sites.id) ← NEW (denormalized)

carpool_rides
├── ... (existing columns unchanged)
├── company_id (UUID, NULLABLE, FK → companies.id) ← NEW (denormalized)
└── site_id (UUID, NULLABLE, FK → sites.id) ← NEW (denormalized)
```

---

## 🎯 Next Steps

### Phase 2 Complete ✅

- [x] Surrogate id added to user_matching_preferences
- [x] company_id and site_id added to all 5 tables
- [x] All indexes created
- [x] All constraints created
- [x] Documentation added

### Ready for Phase 3

**Phase 3:** Add query filters to prevent data leaks
- **CRITICAL** - Must complete before enabling company features
- Will add `AND (company_id = $2 OR ($2 IS NULL AND company_id IS NULL))` filters
- Defaults to `companyID = nil` (personal scope)
- All existing queries will continue to work

---

## 📝 Important Notes

1. **All columns are nullable** - Existing data remains valid
2. **All existing data has company_id IS NULL** - Personal scope by default
3. **Primary key change is backward compatible** - Existing queries still work (just need filter)
4. **Denormalized columns** - Will be populated from carpools in Phase 7
5. **Zero impact on existing code** - No queries modified yet (that's Phase 3)

---

## ⚠️ Critical Reminder

**Phase 3 is MANDATORY** before enabling company features. Phase 3 adds query filters to prevent data leaks. Without Phase 3, company data could be visible to personal users (security issue).

**Do NOT skip Phase 3!**

---

**Status:** ✅ **PHASE 2 COMPLETE**  
**Ready for:** Phase 3 (Query Filters - CRITICAL)

