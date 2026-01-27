# User Auth Flow Audit and Frontend Best Practices

## Task Description
Analyze `@frontend_react/` for authentication requirements and best practices.
1. Determine which pages require login (gated) vs public access.
2. Review file structure and implementation for best practices (Next.js, React, Shadcn/UI, Zustand).
3. Ensure user experience is seamless (redirects, loading states, error handling).

## Plan
- [ ] **Analyze Auth Requirements**:
    - [ ] List all routes in `src/app`.
    - [ ] Determine auth status for each (Public, Protected, Hybrid).
    - [ ] Verify `use-require-auth.ts` and `use-auth.ts` implementation matches requirements.
    - [ ] Check `proxy.ts` middleware configuration.
- [ ] **Codebase Audit**:
    - [ ] Review `src/components` for reusability and separation of concerns.
    - [ ] Check `src/stores` for state management best practices.
    - [ ] Verify `src/lib/api` for standardized API handling.
    - [ ] Inspect `src/hooks` for custom hook logic.
- [ ] **Implementation Verification**:
    - [ ] Ensure protected routes redirect to login or show modal.
    - [ ] Verify public routes are accessible without token.
    - [ ] Check loading states during auth checks.
- [ ] **Optimization & Refactoring**:
    - [ ] Suggest/Implement improvements for auth flow.
    - [ ] Refactor components/pages if they violate best practices.

## Context
- `frontend_react` is Next.js 15 + React + Zustand + Shadcn.
- Auth involves `proxy.ts` middleware and client-side hooks.
- Goal: "User even without login can use most features" -> Hybrid approach preferred.
