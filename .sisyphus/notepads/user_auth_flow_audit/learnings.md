## [2026-01-27T03:24:00.000Z] Auth Requirements Analysis
- Found 26 main pages and 3 auth pages.
- `proxy.ts` middleware correctly protects only `/settings` and `/admin` routes.
- `useRequireAuth` implements the modal-first pattern.
- Goal: Classify remaining routes as Public vs Hybrid (some features protected).

### Route Classification
**Public (Fully accessible without login):**
- `/login`, `/register`, `/reset-password` (Auth)
- `/announcements/page.tsx`
- `/health/page.tsx`
- `/search/page.tsx` (Search functionality is public)
- `/map/page.tsx` (Map view is public)
- `/feed/page.tsx` (Viewing feed is public)
- `/moments/page.tsx` (Viewing moments list is public)
- `/moments/[momentId]/page.tsx` (Viewing detail is public)
- `/users/[userId]/page.tsx` (Viewing profiles is public)
- `/groups/page.tsx` (Listing groups is public)
- `/groups/[groupId]/page.tsx` (Viewing group detail is public)
- `/groups/[groupId]/members/page.tsx` (Viewing members is public)

**Protected (Must be logged in):**
- `/settings/*` (Handled by proxy.ts)
- `/admin/*` (Handled by proxy.ts)
- `/moments/create/page.tsx` (Logic exists to show login prompt, effectively protected)
- `/groups/create/page.tsx` (Should be protected)
- `/notifications/page.tsx` (Personal, must be protected)
- `/chat/*` (Personal, must be protected)

**Hybrid (Page loads, but actions require auth):**
- `/feed` (Like/Comment requires auth)
- `/moments/[momentId]` (Like/Comment requires auth)
- `/map` (Interactions require auth)
- `/groups` (Join requires auth)
- `/users/[userId]` (Follow requires auth)

### Action Items
- Verify that Protected routes (not covered by proxy) handle unauthenticated state gracefully (e.g. redirect or show login UI).
- Verify that Hybrid routes allow viewing but gate interactions using `useRequireAuth`.
