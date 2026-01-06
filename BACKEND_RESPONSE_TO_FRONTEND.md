# Backend Response to Frontend Questions

**Date:** 2025-01-XX  
**To:** Frontend Team  
**From:** Backend Team  
**Subject:** Company Spaces Feature - Backend Clarifications & Guarantees

---

## ✅ Backward Compatibility Guarantee

**CRITICAL:** We guarantee **100% backward compatibility** with existing frontend code. All existing endpoints, request formats, and response shapes will work exactly as they do today. Company features are **additive only** - they do not change existing behavior.

---

## 📋 Answers to Your Questions

### Question 1: API Endpoint Names ✅

**Current Backend Endpoints:**

1. **`GET /api/matching/potential-matches`** ✅ **EXISTS & WORKS**
   - Handler: `GetPotentialMatches`
   - Purpose: Get potential matches for the current user
   - **This is the endpoint you're currently using - it will continue to work**

2. **`POST /api/matching/find-matches`** ✅ **EXISTS (Different Purpose)**
   - Handler: `FindMatches`
   - Purpose: Force refresh/regenerate matches (different from getting existing matches)
   - This is a separate endpoint with different functionality

**Answer:**
- ✅ **Continue using `GET /api/matching/potential-matches`** - it works and will continue to work
- ✅ Both endpoints exist and serve different purposes
- ✅ No changes needed to your existing code
- ✅ When company features are added, `GET /api/matching/potential-matches` will support optional `?scope=company&company_id=...` query parameters (backward compatible)

**Current Behavior (Personal Scope - Default):**
```
GET /api/matching/potential-matches
→ Returns personal matches (company_id IS NULL)
```

**Future Behavior (Company Scope - Optional):**
```
GET /api/matching/potential-matches?scope=company&company_id=...
→ Returns company matches (when scope provided)
```

---

### Question 2: sendRequest Parameter Requirements ✅

**Current Backend Implementation:**

The backend **validates at runtime** (not compile-time via struct tags). The handler checks:
- `carpool_name` is required (validated in handler)
- `preferred_carpool_size` is required (validated in handler)
- `potential_match_id` is required (validated in handler)
- `to_user_id` is required (validated in handler)

**Current Handler Validation (lines 868-896):**
```go
// Validates carpool_name is present and not empty
if reqBody.CarpoolName == "" {
    // Returns 400 error
}

// Validates preferred_carpool_size (checked via validation function)
```

**Answer:**
- ✅ **Runtime validation is acceptable** - backend already does this
- ✅ Your current approach (optional parameters with runtime validation) is fine
- ✅ Backend will return clear error messages if required fields are missing:
  ```json
  {
    "error": "MISSING_CARPOOL_NAME",
    "message": "Carpool name is required when sending a match request",
    "field": "carpool_name"
  }
  ```
- ✅ **No breaking changes** - if your call sites already provide these values, everything works

**Error Response Format:**
```json
{
  "error": "ERROR_CODE",
  "message": "Human-readable error message",
  "field": "field_name"  // Optional, only for field-specific errors
}
```

---

### Question 3: Scope Parameter Format ✅

**Backend Supports (Priority Order):**

1. **Query Parameters (Recommended for GET):**
   ```
   GET /api/matching/preferences?scope=company&company_id=...
   GET /api/matching/requests?scope=company&company_id=...
   ```

2. **Request Body (For POST/PUT):**
   ```json
   {
     "scope": "company",
     "company_id": "...",
     "site_id": "..."  // Optional
   }
   ```

3. **URL Pattern (Future - Optional):**
   ```
   /api/company/{slug}/matching/preferences
   ```
   - This is optional and not required for MVP
   - Can be added later if needed

**Answer:**
- ✅ **Query parameters for GET requests** (recommended)
- ✅ **Request body for POST/PUT requests** (recommended)
- ✅ **Default to personal scope** when no scope provided (backward compatible)
- ✅ URL pattern support is optional - not required for initial implementation

**Example Usage:**
```typescript
// Personal scope (default - no params needed)
GET /api/matching/preferences

// Company scope (optional - add params)
GET /api/matching/preferences?scope=company&company_id=uuid-here

// POST with company scope
POST /api/matching/requests
Body: {
  ...existing fields...,
  "company_id": "uuid-here",  // Optional
  "site_id": "uuid-here"      // Optional
}
```

---

### Question 4: "Not Configured" Response Format ✅

**Response Format:**
```json
{
  "configured": false,
  "message": "Company preferences not set up. Please configure in company hub.",
  "company_id": "uuid-company-1"
}
```

**HTTP Status Code:** `200 OK` (not 404)

**Answer:**
- ✅ Status code: **200 OK** (not 404)
- ✅ Response includes `configured: false` flag
- ✅ Only applies to **company preferences** (not personal)
- ✅ Personal preferences will return 404 if not configured (existing behavior)

**When This Happens:**
- User requests company preferences: `GET /api/matching/preferences?scope=company&company_id=...`
- User has membership but hasn't set up company preferences yet
- Backend returns 200 with `configured: false`

---

### Question 5: Error Response Format ✅

**Standard Error Format:**
```json
{
  "error": "ERROR_CODE",
  "message": "Human-readable error message",
  "code": "MACHINE_READABLE_CODE",  // Same as "error" field
  "field": "field_name",            // Optional, for field-specific errors
  "company_id": "uuid-company-1"    // Optional, for company-related errors
}
```

**Common Error Codes:**

| HTTP Status | Error Code | When It Happens |
|------------|-----------|-----------------|
| 400 | `MISSING_CARPOOL_NAME` | `carpool_name` not provided |
| 400 | `EMPTY_CARPOOL_NAME` | `carpool_name` is empty/whitespace |
| 400 | `INVALID_CARPOOL_NAME` | `carpool_name` > 255 characters |
| 400 | `MISSING_COMPANY_ID` | `company_id` required when `scope=company` |
| 400 | `INVALID_COMPANY_ID` | Invalid `company_id` format |
| 400 | `INVALID_SITE_ID` | Invalid `site_id` format or doesn't belong to company |
| 403 | `MISSING_COMPANY_MEMBERSHIP` | User is not a member of the company |
| 403 | `INSUFFICIENT_PERMISSIONS` | User doesn't have required role |
| 404 | `COMPANY_NOT_FOUND` | Company doesn't exist |
| 404 | `SITE_NOT_FOUND` | Site doesn't exist |

**403 Forbidden Format (Same Structure):**
```json
{
  "error": "MISSING_COMPANY_MEMBERSHIP",
  "message": "User is not a member of this company",
  "code": "MISSING_COMPANY_MEMBERSHIP"
}
```

**Answer:**
- ✅ Standard format for all errors
- ✅ HTTP status code indicates error type (400, 403, 404, 500)
- ✅ `error` and `code` fields are the same (for consistency)
- ✅ Additional fields (`field`, `company_id`) only when relevant

---

### Question 6: Carpool Response Fields ✅

**Response Format:**

**Option A (Preferred):** Always include fields, use `null` when not applicable
```json
{
  "id": "carpool-uuid",
  "name": "Morning Commute",
  "company_id": null,        // Always present, null for personal
  "site_id": null,           // Always present, null for personal
  ...
}
```

**Option B (Also Supported):** Omit fields when `null`
```json
{
  "id": "carpool-uuid",
  "name": "Morning Commute",
  // company_id and site_id omitted when null
  ...
}
```

**Answer:**
- ✅ **Backend will use Option A** (always include, use `null`)
- ✅ Your TypeScript interface `company_id?: string | null` handles both cases
- ✅ Consistent format across all responses

**Implementation:**
- Personal carpools: `company_id: null, site_id: null`
- Company carpools: `company_id: "uuid", site_id: "uuid"` (or `null` if no site)

---

### Question 7: Site Selection Requirement ✅

**Current Policy (From Blueprint):**
- Site selection is **required** before matching in company scope
- If `site_id IS NULL` in membership, matching is blocked

**Error Response:**
```json
{
  "error": "SITE_NOT_SELECTED",
  "message": "Please select your site to enable company matching",
  "code": "SITE_NOT_SELECTED",
  "company_id": "uuid-company-1"
}
```

**HTTP Status Code:** `400 Bad Request`

**Answer:**
- ✅ Status code: **400 Bad Request**
- ✅ Error code: `SITE_NOT_SELECTED`
- ✅ Clear message: "Please select your site to enable company matching"
- ✅ Applies only to company scope matching (personal matching unaffected)

**When This Happens:**
- User tries to match in company scope: `GET /api/matching/potential-matches?scope=company&company_id=...`
- User has company membership but `site_id IS NULL`
- Backend returns 400 with `SITE_NOT_SELECTED` error

**User Flow:**
1. User joins company (auto-detected or manual)
2. User must select site: `PUT /api/me/company-site` with `site_id`
3. Then matching works in company scope

---

### Question 8: Backward Compatibility Guarantees ✅

**100% Backward Compatibility Guarantee:**

1. ✅ **All existing endpoints work unchanged**
   - No breaking changes to request/response formats
   - All existing functionality preserved

2. ✅ **Personal scope is default**
   - When no `scope` parameter provided → personal scope
   - Existing frontend code works without changes

3. ✅ **Response shapes unchanged**
   - Existing response fields remain the same
   - New fields (`company_id`, `site_id`) are additive only
   - Existing fields never removed or changed

4. ✅ **No breaking changes to existing users**
   - Personal users see no changes in behavior
   - All existing flows work exactly as before

**Examples:**

**Before (Current - Works):**
```typescript
GET /api/matching/preferences
→ Returns personal preferences (company_id IS NULL)

GET /api/matching/requests
→ Returns personal requests (company_id IS NULL)

POST /api/matching/requests
Body: { to_user_id, potential_match_id, carpool_name, preferred_carpool_size, message }
→ Creates personal request (company_id IS NULL)
```

**After (Future - Still Works):**
```typescript
// Same calls work exactly the same
GET /api/matching/preferences
→ Returns personal preferences (company_id IS NULL)

// New optional company features
GET /api/matching/preferences?scope=company&company_id=...
→ Returns company preferences (when scope provided)
```

**Answer:**
- ✅ **Confirmed: 100% backward compatible**
- ✅ All existing endpoints work unchanged
- ✅ Personal scope is default (no scope param = personal)
- ✅ No breaking changes to existing functionality
- ✅ Existing personal users see no changes

---

## 📋 Implementation Summary

### What Backend Provides

1. **Backward Compatible Endpoints:**
   - All existing endpoints work unchanged
   - Optional scope parameters for company features
   - Default to personal scope when no scope provided

2. **Error Handling:**
   - Standardized error format
   - Clear error codes
   - User-friendly messages

3. **Response Format:**
   - Consistent response shapes
   - Always include `company_id` and `site_id` (use `null` for personal)
   - No breaking changes to existing fields

### What Frontend Needs to Do

1. **Phase 0 (No Backend Changes):**
   - Type definitions
   - Company context
   - UI components (stubs)

2. **Phase 2 (After Backend Phase 3):**
   - Test backward compatibility
   - Verify existing endpoints work unchanged

3. **Phase 3+ (After Backend Phases 4-8):**
   - Add optional scope parameters
   - Integrate company features
   - Handle new error codes

---

## 🎯 Next Steps

1. **Backend Team:**
   - ✅ All questions answered
   - ✅ Backward compatibility guaranteed
   - ✅ Ready to proceed with implementation

2. **Frontend Team:**
   - ✅ Can proceed with Phase 0 (no backend needed)
   - ✅ Can plan Phase 2 (backward compatibility verification)
   - ✅ Can plan Phase 3+ (company features)

3. **Coordination:**
   - Align on Phase 2 timing (after Backend Phase 3)
   - Coordinate Phase 3+ (after Backend Phases 4-8)

---

## 📝 Additional Notes

### Endpoint Summary

| Endpoint | Method | Purpose | Scope Support |
|----------|--------|---------|---------------|
| `/api/matching/potential-matches` | GET | Get potential matches | ✅ Optional `?scope=company&company_id=...` |
| `/api/matching/find-matches` | POST | Force refresh matches | ✅ Optional in body |
| `/api/matching/preferences` | GET | Get preferences | ✅ Optional `?scope=company&company_id=...` |
| `/api/matching/preferences` | PUT | Update preferences | ✅ Optional in body |
| `/api/matching/requests` | GET | Get requests | ✅ Optional `?scope=company&company_id=...` |
| `/api/matching/requests` | POST | Create request | ✅ Optional in body |

### Error Handling Summary

- **400 Bad Request:** Missing/invalid parameters, site not selected
- **403 Forbidden:** Missing membership, insufficient permissions
- **404 Not Found:** Company/site not found, preferences not configured (personal)
- **200 OK with `configured: false`:** Company preferences not configured

---

**Status:** ✅ All Questions Answered  
**Backward Compatibility:** ✅ 100% Guaranteed  
**Ready for Implementation:** ✅ Yes

---

**Thank you for the thorough questions!**  
**Looking forward to smooth integration.**

