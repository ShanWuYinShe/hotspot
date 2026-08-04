# API Contract Codegen — Feasibility Assessment

## Current state

Three codebases hand-maintain the same API contract against `spec/`:

| Layer | File(s) | Format |
|-------|---------|--------|
| Frontend | `frontend/src/api/endpoints.ts` + `types.ts` | TypeScript constants + interfaces |
| Backend | `backend-go/internal/modules/*/types.go` | Go structs |
| HarmonyOS | `harmonyapp/.../ApiEndpoints.ets` + `.../types/` | ArkTS constants + interfaces |

The contract authority is `spec/api_spec_*.md` (markdown with TypeScript-like interface blocks). Every endpoint lives in three hand-maintained places, and the 4 API-contract bugs recently fixed (MapViewport.zoom, location_name validation, admin/tags sort_by, MapHeatBucket fields) were all caused by drift between these three sources.

## Option 1: Spec → OpenAPI → typed clients

Convert `spec/` markdown into a single OpenAPI 3.1 YAML file, then generate types for all three languages.

- **Pros**: Industry standard; generators exist for Go (`oapi-codegen`), TypeScript (`openapi-typescript`), and ArkTS (`openapi-to-arkts` or similar); single source of truth.
- **Cons**: Initial conversion is manual (~2-4 days for ~50 endpoints); spec currently mixes TypeScript/descriptive prose — some restructuring needed; generators produce opinionated output that may not match current code style.
- **Effort**: L (1-2 weeks for conversion + plumbing into all three clients).
- **Recommended**: For a greenfield project or a major version bump. For this codebase, the manual cost is high.

## Option 2: Write a spec parser → emit types directly

Write a small Go/TypeScript parser that reads the markdown interface blocks from `spec/api_spec_*.md` and emits:
- `frontend/src/lib/types/api.ts` (TypeScript interfaces)
- `harmonyapp/.../types/{api,CommonTypes}.ets` (ArkTS interfaces)

- **Pros**: Custom; fits any spec format; no intermediate OpenAPI layer; generated output can exactly match current style.
- **Cons**: Parser maintenance burden; not battle-tested; doesn't solve endpoints.ts (path constants are not in the structured part of spec).
- **Effort**: M (2-4 days for parser + CI hook).
- **Recommended**: Feasible as a local dev tool. Adding it to a pre-commit hook would prevent drift.

## Option 3: CI diff check (minimal, quick win)

Add a CI step that compares the current frontend types against a reference snapshot generated from spec. On mismatch, the build fails with a report showing which fields differ.

- **Pros**: Immediate rollout (<1 day); catches drift before merge; no new tooling needed.
- **Cons**: Doesn't generate code — still manual update; snapshot is maintained by a human (but validated by CI).
- **Effort**: S (~1 day to write comparison script + CI hook).
- **Recommended**: **Start here.** This is the highest-ROI option: it prevents the class of bugs we just fixed without any new infrastructure. Implement as a CI-only check, then graduate to Option 2 when the spec format stabilizes.

## Recommendation

**Phase 1 (now)**: Option 3 — CI diff check. Write a script that parses `spec/api_spec_types.md` + module-specific `api_spec_*.md` and compares against `frontend/src/api/types.ts`. Run in CI alongside `bun check`.

**Phase 2 (when spec matures)**: Option 2 — custom parser that emits TS/ArkTS types. Reuse the parser from Phase 1, extend it to emit files.

**Phase 3 (major version)**: Option 1 — OpenAPI. Only if the product needs third-party API consumers or an official SDK.

## Implementation notes (Phase 1)

The comparison script would:
1. Parse `spec/api_spec_types.md` for base types (GeoPoint, Location, PaginationParams, etc.).
2. Parse each `api_spec_*.md` for module-specific request/response types.
3. Parse the generated `frontend/src/api/types.ts` via TypeScript compiler API or simple regex.
4. Report: extra fields in frontend (→ strict-decode 10000), missing fields (→ silent data loss), type mismatches (`number` vs `string`).

Rough estimate: 200-400 lines of TypeScript. Run as `bun run scripts/check-contract-drift.ts`.

## Status

Not implemented. This document serves as research and decision support. Adding it to the project backlog.
