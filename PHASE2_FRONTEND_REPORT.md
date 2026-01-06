# Phase 2 Implementation Report - Backend Schema Changes (Frontend View)

**Date:** 2025-01-XX  
**Status:** ✅ **COMPLETE**  
**Backward Compatibility:** ✅ **100% (Still No Behavior Changes)**

---

## Executive Summary

Backend Phase 2 is complete. This phase **only added new columns** to existing tables so that, in later phases, we can scope data by `company_id` and `site_id`.

- **No API behavior has changed yet.**
- **No responses have changed.**
- **No queries/handlers have been modified yet.** (That is Phase 3.)
- **All existing frontend behavior remains exactly the same.**

Think of Phase 2 as **laying extra wiring in the database** without connecting it to anything yet.

---

## What Changed in the Database

### 1. `user_matching_preferences` (Most Important)

**Goal:** Allow multiple preference records per user:
- One **personal** preference (no company)
- One **per-company** preference (for each company the user belongs to)

**Previously:**
- Primary key = `user_id`
- Exactly one row per user

**Now:**
- New surrogate primary key: `id: UUID`
- `user_id` is still present and indexed
- **New columns:**
  - `company_id: UUID | NULL`
  - `site_id: UUID | NULL`
- **New uniqueness rule:**
  - Unique on `(user_id, COALESCE(company_id, '0000...000'))`
  - This means:
    - One row per user where `company_id IS NULL` (personal)
    - One row per user per `company_id` (company-specific)

**Why this matters later (for you):**
- Backend will support:
  - `GET /api/matching/preferences` → personal (no company)
  - `GET /api/matching/preferences?scope=company&company_id=...` → company-specific
- Backend can safely store both personal + company preferences per user without breaking existing code.

**Safety for existing behavior:**
- All existing rows now have:
  - `id` set to a generated UUID
  - `company_id = NULL`
  - `site_id = NULL`
- All current queries still filter only on `user_id` (no change yet), so behavior is unchanged.

---

### 2. `match_requests`

**Goal:** Tag each match request with optional `company_id`/`site_id` for coworkers-only matching.

**New columns:**
- `company_id: UUID | NULL`
- `site_id: UUID | NULL`

**What this will mean later:**
- Personal request:
  - `company_id = NULL`
- Company request:
  - `company_id = 'uuid-company'`
  - Optional `site_id = 'uuid-site'`
- Backend can distinguish between personal vs company requests.

**Safety now:**
- All existing rows have `company_id = NULL` and `site_id = NULL`.
- Existing queries do **not** reference these columns yet, so behavior is unchanged.

---

### 3. `carpools`

**Goal:** Tag each carpool with optional `company_id`/`site_id` (personal vs company carpools).

**New columns:**
- `company_id: UUID | NULL`
- `site_id: UUID | NULL`

**What this will mean later:**
- Personal carpool:
  - `company_id = NULL`
- Company carpool:
  - `company_id = 'uuid-company'`
  - Optional `site_id = 'uuid-site'`

**Safety now:**
- All existing rows have `company_id = NULL` and `site_id = NULL`.
- Existing queries do **not** reference these columns yet.

---

### 4. `carpool_schedules`

**Goal:** Denormalize `company_id`/`site_id` onto schedules for faster queries and analytics.

**New columns:**
- `company_id: UUID | NULL`
- `site_id: UUID | NULL`

**What this will mean later:**
- Schedule inherits `company_id`/`site_id` from its carpool.
- Backend can quickly query schedules per company and site without heavy JOINs.

**Safety now:**
- All existing rows have `company_id = NULL` and `site_id = NULL`.
- No behavior change; existing schedule APIs are unchanged.

---

### 5. `carpool_rides`

**Goal:** Denormalize `company_id`/`site_id` onto rides for analytics and scoping.

**New columns:**
- `company_id: UUID | NULL`
- `site_id: UUID | NULL`

**What this will mean later:**
- Ride inherits `company_id`/`site_id` from its carpool.
- Enables company/site-based stats without complex JOINs.

**Safety now:**
- All existing rows have `company_id = NULL` and `site_id = NULL`.
- Existing ride endpoints behave exactly the same.

---

## Why This Still Doesn’t Affect You (Yet)

### 1. No Code Changes in Repositories/Handlers (That’s Phase 3)

- Phase 2 only added **migrations**.
- We did **not** change any Go code in:
  - `pkg/repository/*`
  - `pkg/handlers/*`
  - `main.go` routes
- All current queries ignore the new columns, so they behave exactly as before.

### 2. All New Columns Are Nullable

- No `NOT NULL` constraints were added to existing tables.
- Every existing row is valid with `company_id = NULL` and `site_id = NULL`.
- This means:
  - Personal data remains exactly as-is
  - No existing data was updated beyond adding defaults for new columns

### 3. Behavior Is Unchanged Until Phase 3

- Phase 3 is where we will **update queries** to filter by `company_id`:
  - `AND (company_id = $2 OR ($2 IS NULL AND company_id IS NULL))`
  - Default `companyID = nil` → `company_id IS NULL` (personal only)
- Frontend will see **zero change** until these filters are added.

---

## How This Supports Frontend Company Features Later

### Matching Preferences

You already plan to:
- Call:
  - `GET /api/matching/preferences` → personal
  - `GET /api/matching/preferences?scope=company&company_id=...` → company

Phase 2 enables this structurally by:
- Allowing multiple preference rows per user
- Tagging each row with optional `company_id`/`site_id`

### Match Requests, Carpools, Schedules, Rides

Later phases will:
- Use `company_id`/`site_id` to:
  - Restrict matches to coworkers
  - Scope requests, carpools, rides to a company
  - Power analytics endpoints (`/api/companies/{id}/stats`, etc.)

Phase 2 is the **schema foundation** that makes this possible.

---

## Frontend Impact Summary

### Right Now (After Phase 2)

- ✅ **No new API behavior**
- ✅ **No response shape changes**
- ✅ **No new required fields**
- ✅ **No scope logic active yet**
- ✅ **All existing requests/responses unchanged**

### Later (Phase 3+)

- Phase 3: Query filters
  - Backend starts respecting `scope` and `company_id` in queries
  - Still backward compatible (no scope → personal)

- Phase 4–8: Company APIs & analytics
  - New endpoints (`/api/me/company`, stats, etc.) become available
  - Your Phase 0 foundation code starts to be used

---

## Files Added in Phase 2

**New migrations:**
- `026_add_company_columns_to_preferences.sql`
- `027_add_company_columns_to_match_requests.sql`
- `028_add_company_columns_to_carpools.sql`
- `029_add_company_columns_to_schedules.sql`
- `030_add_company_columns_to_rides.sql`

**Docs:**
- `PHASE2_IMPLEMENTATION_SUMMARY.md` (backend-focused)
- `PHASE2_FRONTEND_REPORT.md` (this document)

---

## TL;DR for Frontend

- Backend has **added columns only**; nothing is wired up in code yet.
- All existing behavior is **100% identical** to pre-Phase-2.
- These columns are the foundation for:
  - Company-scoped preferences
  - Company-scoped requests/carpools/rides
  - Company + site analytics
- You don’t need to change anything yet; this just sets us up for Phase 3+.

---

**Status:** ✅ **Phase 2 Complete**  
**Frontend Impact:** ✅ **None (Behavior Unchanged)**  
**Ready For:** Phase 3 (Query Filters)

