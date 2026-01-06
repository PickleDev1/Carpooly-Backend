# Frontend Implementation Plan - Review & Corrections

**Review Date:** 2025-01-XX  
**Status:** ✅ **MOSTLY CORRECT** with minor corrections needed

---

## Overall Assessment

The frontend plan is **well-structured and mostly correct**. It aligns well with the backend blueprint and implementation plan. However, there are a few **corrections and additions** needed to ensure 100% accuracy.

---

## ✅ What's Correct

1. **Phase Alignment** - Frontend phases correctly align with backend phases
2. **Type Definitions** - Match backend models correctly
3. **Scope Handling** - Correctly implements scope resolution
4. **Backward Compatibility** - Properly handles default to personal scope
5. **Error Handling** - Good coverage of error scenarios
6. **UI/UX Considerations** - Well thought out

---

## 🔧 Corrections & Additions Needed

### 1. API Endpoint Corrections

#### Issue: `POST /api/matching/find-matches` Endpoint Name

**Frontend Plan Shows:**
```typescript
async getPotentialMatches(filters: MatchFilters = {}, scope?: Scope)
```

**Backend Blueprint Shows:**
- The endpoint is `POST /api/matching/find-matches` (not `potential-matches`)

**Correction:**
```typescript
// In src/services/matching.ts
async getPotentialMatches(filters: MatchFilters = {}, scope?: Scope): Promise<PotentialMatchesResponse> {
  const endpoint = `${process.env.NEXT_PUBLIC_API_URL}/api/matching/find-matches`  // ✅ Correct
  // NOT: /api/matching/potential-matches  ❌
}
```

#### Issue: Missing `id` Field in MatchingPreferences

**Frontend Plan Shows:**
```typescript
export interface MatchingPreferences {
  // ... existing fields
  company_id?: string | null;
  site_id?: string | null;
}
```

**Backend Blueprint Shows:**
- After Phase 2 migration, `user_matching_preferences` has a new `id` field (surrogate PK)

**Correction:**
```typescript
export interface MatchingPreferences {
  id?: string;  // NEW - surrogate primary key
  user_id: string;
  company_id?: string | null;
  site_id?: string | null;
  // ... rest of fields
}
```

### 2. API Request/Response Format Corrections

#### Issue: `POST /api/matching/requests` Request Body

**Frontend Plan Shows:**
```typescript
const body = {
  to_user_id: toUserId,
  ...(potentialMatchId && { potential_match_id: potentialMatchId }),
  ...(message && { message }),
  ...(carpoolName && { carpool_name: carpoolName }),
  ...(preferredCarpoolSize && { preferred_carpool_size: preferredCarpoolSize }),
  ...(scope?.type === 'company' && {
    company_id: scope.companyId,
    site_id: scope.siteId,
  }),
}
```

**Backend Blueprint Shows:**
- `carpool_name` is **required** (not optional)
- `preferred_carpool_size` is **required** (not optional)

**Correction:**
```typescript
async sendRequest(
  toUserId: string,
  potentialMatchId: string,  // Required
  carpoolName: string,       // Required (not optional)
  preferredCarpoolSize: number,  // Required (not optional)
  message?: string,          // Optional
  scope?: Scope
): Promise<MatchRequestResponse> {
  const body = {
    to_user_id: toUserId,
    potential_match_id: potentialMatchId,  // Required
    carpool_name: carpoolName,             // Required
    preferred_carpool_size: preferredCarpoolSize,  // Required
    ...(message && { message }),          // Optional
    ...(scope?.type === 'company' && {
      company_id: scope.companyId,
      site_id: scope.siteId,
    }),
  }
}
```

### 3. Missing API Endpoint

#### Issue: Missing `GET /api/matching/requests` Response Fields

**Frontend Plan Shows:**
- Uses existing `MatchRequest` interface

**Backend Blueprint Shows:**
- Response includes `company_id` and `site_id` fields

**Verification Needed:**
```typescript
export interface MatchRequest {
  id: string;
  from_user_id: string;
  to_user_id: string;
  potential_match_id: string;
  message?: string;
  preferred_carpool_size: number;
  carpool_name: string;
  company_id?: string | null;  // NEW
  site_id?: string | null;     // NEW
  status: 'pending' | 'accepted' | 'rejected' | 'expired';
  expires_at: string;
  created_at: string;
  updated_at: string;
  from_user?: {
    id: string;
    name: string;
    display_name?: string;
    home_latitude?: number;
    home_longitude?: number;
  };
}
```

### 4. Site Selection API Correction

#### Issue: Site Selection Endpoint

**Frontend Plan Shows:**
```typescript
async updateCompanySite(companyId: string, siteId: string | null)
```

**Backend Blueprint Shows:**
- Endpoint: `PUT /api/me/company-site`
- Request body includes both `company_id` and `site_id`

**Verification:**
```typescript
// ✅ This is correct
async updateCompanySite(companyId: string, siteId: string | null): Promise<{ company_id: string; site_id: string | null; updated_at: string }> {
  const headers = await getHeaders()
  const response = await fetch(`${API_URL}/api/me/company-site`, {
    method: 'PUT',
    headers,
    body: JSON.stringify({ company_id: companyId, site_id: siteId }),
  })
  // ...
}
```

### 5. Missing Error Response Handling

#### Issue: "Not Configured" Response for Preferences

**Frontend Plan Shows:**
- Standard error handling

**Backend Blueprint Shows:**
- Special response when company preferences don't exist:
```json
{
  "configured": false,
  "message": "Company preferences not set up. Please configure in company hub.",
  "company_id": "uuid-company-1"
}
```

**Addition Needed:**
```typescript
// In src/services/matching.ts
async getPreferences(scope?: Scope): Promise<MatchingPreferences | { configured: false; message: string; company_id: string }> {
  // Handle both cases:
  // 1. Normal preferences object
  // 2. { configured: false, message: string, company_id: string }
  
  const response = await fetch(url, { headers })
  const data = await response.json()
  
  if (data.configured === false) {
    return data  // Return as-is, let component handle
  }
  
  return data as MatchingPreferences
}
```

### 6. URL Pattern Option Not Mentioned

#### Issue: Backend Supports URL Pattern Routing

**Frontend Plan Shows:**
- Only uses query parameters for scope

**Backend Blueprint Shows:**
- Also supports URL pattern: `/api/company/{slug}/...`

**Note:**
- This is **optional** - query params work fine
- URL pattern is more RESTful but requires routing changes
- Frontend plan is fine as-is, but could mention this option

**Optional Addition:**
```typescript
// Alternative approach (optional):
// Use URL pattern: /api/company/{slug}/matching/preferences
// Instead of: /api/matching/preferences?scope=company&company_id=...

// This would require Next.js dynamic routing:
// app/api/company/[slug]/matching/preferences/route.ts
// But current query param approach is simpler and works fine
```

### 7. Carpool Response Fields

#### Issue: Missing `company_id` and `site_id` in Carpool Interface

**Frontend Plan Shows:**
- Uses existing `Carpool` interface

**Backend Blueprint Shows:**
- Carpools now have `company_id` and `site_id` fields

**Addition Needed:**
```typescript
export interface Carpool {
  id: string;
  creator_id: string;
  carpool_name: string;
  status: boolean;
  recurring_option?: string;
  destination_address: string;
  seats: number;
  available_seats: number;
  company_id?: string | null;  // NEW
  site_id?: string | null;     // NEW
  created_at: string;
  updated_at: string;
}
```

### 8. Site Selection Flow Clarification

#### Issue: Site Selection Required Policy

**Frontend Plan Shows:**
- Site selection prompt blocks matching

**Backend Blueprint Shows:**
- **Option A (Chosen):** Site selection required before matching
- If `site_id IS NULL`, matching is blocked

**Verification:**
- Frontend plan correctly implements this
- Site selection prompt is shown when needed
- Matching is blocked until site selected

**✅ This is correct**

---

## 📋 Additional Recommendations

### 1. Add Loading States for Scope Switching

**Recommendation:**
```typescript
// In CompanyContext
const [isSwitching, setIsSwitching] = useState(false)

const setActiveCompany = async (companyId: string) => {
  setIsSwitching(true)
  try {
    // Switch company
    // Refresh data
    await refreshMemberships()
  } finally {
    setIsSwitching(false)
  }
}
```

### 2. Add Retry Logic for Failed API Calls

**Recommendation:**
```typescript
// In API service
async function fetchWithRetry(url: string, options: RequestInit, retries = 3): Promise<Response> {
  for (let i = 0; i < retries; i++) {
    try {
      const response = await fetch(url, options)
      if (response.ok) return response
      if (response.status >= 500 && i < retries - 1) {
        await new Promise(resolve => setTimeout(resolve, 1000 * (i + 1)))
        continue
      }
      return response
    } catch (error) {
      if (i === retries - 1) throw error
      await new Promise(resolve => setTimeout(resolve, 1000 * (i + 1)))
    }
  }
  throw new Error('Max retries exceeded')
}
```

### 3. Add Analytics Tracking

**Recommendation:**
```typescript
// Track company feature usage
const trackCompanyAction = (action: string, scope: Scope) => {
  analytics.track('company_action', {
    action,
    scope_type: scope.type,
    company_id: scope.type === 'company' ? scope.companyId : null,
  })
}
```

### 4. Add Feature Flag Support

**Recommendation:**
```typescript
// Feature flag for gradual rollout
const COMPANY_FEATURES_ENABLED = process.env.NEXT_PUBLIC_ENABLE_COMPANY_FEATURES === 'true'

// In components
{COMPANY_FEATURES_ENABLED && <CompanySelector />}
```

---

## ✅ Final Checklist

### API Contracts
- [x] All endpoint URLs correct
- [x] Request bodies match backend expectations
- [x] Response types include new fields (`company_id`, `site_id`)
- [x] Error responses handled
- [x] "Not configured" response handled

### Type Definitions
- [x] All interfaces include new fields
- [x] Scope type matches backend
- [x] Company types match backend models

### Implementation
- [x] Scope resolution correct
- [x] Backward compatibility maintained
- [x] Error handling comprehensive
- [x] Loading states considered
- [x] UI/UX considerations addressed

---

## 🎯 Summary

**Status:** ✅ **APPROVED WITH MINOR CORRECTIONS**

**Required Changes:**
1. Add `id` field to `MatchingPreferences` interface
2. Make `carpool_name` and `preferred_carpool_size` required in `sendRequest`
3. Add `company_id` and `site_id` to `Carpool` interface
4. Handle "not configured" response for preferences
5. Verify endpoint name: `/api/matching/find-matches` (not `potential-matches`)

**Optional Enhancements:**
1. Consider URL pattern routing (optional)
2. Add retry logic for API calls
3. Add analytics tracking
4. Add feature flags for gradual rollout

**Once these corrections are made, the frontend plan is ready for implementation!**

---

**Review Complete**  
**Next Step:** Frontend team makes corrections, then proceed with implementation

