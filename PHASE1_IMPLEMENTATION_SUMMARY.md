# Phase 1 Implementation Summary - Database Schema (New Tables)

**Date:** 2025-01-XX  
**Status:** ✅ **COMPLETE**

---

## 📋 What Was Implemented

### Migration Files Created

1. ✅ **`migrations/023_create_companies_table.sql`**
   - Creates `companies` table
   - Stores company/organization information
   - Includes indexes for performance

2. ✅ **`migrations/024_create_sites_table.sql`**
   - Creates `sites` table
   - Represents physical company locations
   - Links to companies via foreign key

3. ✅ **`migrations/025_create_user_company_memberships_table.sql`**
   - Creates `user_company_memberships` table
   - Links users to companies (and optionally sites)
   - Includes role and status management

---

## 🔍 Detailed Explanation

### 1. Companies Table (`023_create_companies_table.sql`)

**Purpose:** Store company/organization information for the Company Spaces feature.

**Key Fields:**
- `id`: UUID primary key
- `name`: Company name (e.g., "Amazon")
- `slug`: URL-safe identifier (e.g., "amazon") - unique, lowercase
- `primary_domain`: Primary email domain for auto-detection (e.g., "amazon.com")
- `additional_domains`: JSONB array of additional domains (e.g., ["amazon.co.uk", "amzn.com"])
- `logo_url`: Optional company logo for branding
- `settings`: JSONB for extensible configuration
- `is_active`: Soft delete flag

**Indexes Created:**
- `idx_companies_slug`: Fast lookup by slug (for URL routing)
- `idx_companies_primary_domain`: Fast lookup by domain (for auto-detection)
- `idx_companies_domains`: GIN index for JSONB array searches
- `idx_companies_active`: Partial index for active companies only

**Why This Is Safe:**
- ✅ **New table** - doesn't touch any existing tables
- ✅ **No foreign key dependencies** - can be created independently
- ✅ **No data migration** - empty table initially
- ✅ **Zero impact on existing code** - existing queries don't reference this table

---

### 2. Sites Table (`024_create_sites_table.sql`)

**Purpose:** Represent physical company locations (offices, warehouses, campuses) where employees commute.

**Key Fields:**
- `id`: UUID primary key
- `company_id`: Foreign key to `companies` table (CASCADE delete)
- `name`: Site name (e.g., "SJC14 Warehouse")
- `code`: Short identifier (e.g., "SJC14") - unique per company
- `address`: Full address
- `latitude`/`longitude`: GPS coordinates (used as destination for carpools)
- `timezone`: IANA timezone string (e.g., "America/Los_Angeles")
- `is_active`: Soft delete flag

**Constraints:**
- `UNIQUE(company_id, code)`: Ensures unique codes per company (but allows NULL codes)

**Indexes Created:**
- `idx_sites_company`: Fast lookup by company
- `idx_sites_company_active`: Composite index for active sites per company

**Why This Is Safe:**
- ✅ **New table** - doesn't touch any existing tables
- ✅ **Foreign key to companies** - safe because companies table is created first
- ✅ **No data migration** - empty table initially
- ✅ **Zero impact on existing code** - existing queries don't reference this table

---

### 3. User Company Memberships Table (`025_create_user_company_memberships_table.sql`)

**Purpose:** Link users to companies (and optionally sites) for multi-tenancy.

**Key Fields:**
- `id`: UUID primary key
- `user_id`: Foreign key to `users` table (CASCADE delete)
- `company_id`: Foreign key to `companies` table (CASCADE delete)
- `site_id`: Optional foreign key to `sites` table (user picks later)
- `role`: Role hierarchy (`employee`, `site_admin`, `company_admin`)
- `status`: Membership status (`active`, `pending`, `invited`, `inactive`)

**Constraints:**
- `UNIQUE(user_id, company_id)`: One membership per user per company
- `CHECK` constraints on `role` and `status` for data integrity

**Indexes Created:**
- `idx_memberships_user`: Fast lookup of user's memberships
- `idx_memberships_company`: Fast lookup of company members
- `idx_memberships_site`: Partial index for site-based queries
- `idx_memberships_company_active`: Composite index for active memberships

**Why This Is Safe:**
- ✅ **New table** - doesn't touch any existing tables
- ✅ **Foreign keys to users, companies, sites** - all created in correct order
- ✅ **No data migration** - empty table initially
- ✅ **Zero impact on existing code** - existing queries don't reference this table

---

## ✅ Safety Guarantees

### 1. No Existing Tables Modified
- ✅ All three migrations create **NEW tables only**
- ✅ No `ALTER TABLE` statements on existing tables
- ✅ No existing data affected

### 2. No Existing Code Affected
- ✅ Existing queries don't reference these tables
- ✅ Existing handlers don't use these tables
- ✅ Existing API endpoints work unchanged

### 3. Proper Foreign Key Ordering
- ✅ Migration 023: Creates `companies` (no dependencies)
- ✅ Migration 024: Creates `sites` (depends on `companies`)
- ✅ Migration 025: Creates `user_company_memberships` (depends on `users`, `companies`, `sites`)

### 4. Rollback Safety
- ✅ Can be rolled back by dropping tables in reverse order
- ✅ No data loss (tables are empty initially)
- ✅ No impact on existing functionality

---

## 🧪 Verification Steps

### Step 1: Run Migrations

```bash
# Run migrations in order
psql -d your_database -f migrations/023_create_companies_table.sql
psql -d your_database -f migrations/024_create_sites_table.sql
psql -d your_database -f migrations/025_create_user_company_memberships_table.sql
```

### Step 2: Verify Tables Exist

```sql
-- Check that all three tables were created
SELECT table_name 
FROM information_schema.tables 
WHERE table_name IN ('companies', 'sites', 'user_company_memberships')
ORDER BY table_name;

-- Expected output:
-- companies
-- sites
-- user_company_memberships
```

### Step 3: Verify Indexes Created

```sql
-- Check indexes on companies table
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename = 'companies';

-- Check indexes on sites table
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename = 'sites';

-- Check indexes on user_company_memberships table
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename = 'user_company_memberships';
```

### Step 4: Verify Constraints Work

```sql
-- Test UNIQUE constraint on companies.slug
INSERT INTO companies (name, slug, primary_domain) VALUES ('Test Company', 'test', 'test.com');
-- Should succeed

INSERT INTO companies (name, slug, primary_domain) VALUES ('Test Company 2', 'test', 'test2.com');
-- Should fail with unique constraint violation

-- Test UNIQUE constraint on user_company_memberships (user_id, company_id)
-- (Requires existing user and company)
-- INSERT INTO user_company_memberships (user_id, company_id) VALUES (...);
-- INSERT INTO user_company_memberships (user_id, company_id) VALUES (...);
-- Second insert should fail

-- Test CHECK constraint on role
INSERT INTO user_company_memberships (user_id, company_id, role) 
VALUES ('...', '...', 'invalid_role');
-- Should fail with check constraint violation
```

### Step 5: Verify Foreign Keys Work

```sql
-- Test CASCADE delete on sites (when company deleted)
-- 1. Create company and site
-- 2. Delete company
-- 3. Verify site is also deleted

-- Test CASCADE delete on user_company_memberships (when user or company deleted)
-- 1. Create membership
-- 2. Delete user or company
-- 3. Verify membership is also deleted
```

### Step 6: Verify Existing Functionality Still Works

```bash
# Run existing tests
go test ./...

# Or manually test existing endpoints
curl -X GET http://localhost:8080/api/matching/preferences
curl -X GET http://localhost:8080/api/matching/requests
# Should work exactly as before
```

---

## 📊 Database Schema After Phase 1

```
companies (NEW)
├── id (UUID, PK)
├── name (TEXT)
├── slug (TEXT, UNIQUE)
├── primary_domain (TEXT)
├── additional_domains (JSONB)
├── logo_url (TEXT)
├── settings (JSONB)
├── is_active (BOOLEAN)
├── created_at (TIMESTAMPTZ)
└── updated_at (TIMESTAMPTZ)

sites (NEW)
├── id (UUID, PK)
├── company_id (UUID, FK → companies.id, CASCADE)
├── name (TEXT)
├── code (TEXT)
├── address (TEXT)
├── latitude (DOUBLE PRECISION)
├── longitude (DOUBLE PRECISION)
├── timezone (TEXT)
├── is_active (BOOLEAN)
├── created_at (TIMESTAMPTZ)
└── updated_at (TIMESTAMPTZ)

user_company_memberships (NEW)
├── id (UUID, PK)
├── user_id (UUID, FK → users.id, CASCADE)
├── company_id (UUID, FK → companies.id, CASCADE)
├── site_id (UUID, FK → sites.id, NULLABLE)
├── role (TEXT, CHECK)
├── status (TEXT, CHECK)
├── created_at (TIMESTAMPTZ)
└── updated_at (TIMESTAMPTZ)
```

---

## 🎯 Next Steps

### Phase 1 Complete ✅

- [x] Companies table created
- [x] Sites table created
- [x] User company memberships table created
- [x] All indexes created
- [x] All constraints created
- [x] Documentation added

### Ready for Phase 2

**Phase 2:** Add nullable `company_id` and `site_id` columns to existing tables
- Will add columns to: `user_matching_preferences`, `match_requests`, `carpools`, `carpool_schedules`, `carpool_rides`
- All columns will be nullable (existing data remains valid)
- All existing data will have `company_id IS NULL` (personal scope)

---

## 📝 Notes

1. **All tables are empty initially** - no data migration needed
2. **All foreign keys use CASCADE delete** - ensures data integrity
3. **All indexes are optimized** - partial indexes where appropriate
4. **All constraints are enforced** - CHECK constraints for data validation
5. **Zero impact on existing code** - new tables don't affect existing queries

---

**Status:** ✅ **PHASE 1 COMPLETE**  
**Ready for:** Phase 2 (Add Columns to Existing Tables)

