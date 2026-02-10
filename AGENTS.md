# HotSpot - Agent Guide

> For agentic coding tools. Keep changes contract-safe and minimal.

## Contract Priority / 契约优先
- `spec/` is the single source of truth for API contracts.
- `spec/api_spec_*.mdc` defines endpoints, fields, error codes, and WebSocket events.
- `spec/api_spec_base.mdc` defines global rules (pagination, idempotency, etc.).
- OpenAPI is not used for now; prefer the `.mdc` specs for AI work.
- backend/frontend_svelte/harmonyapp 必须严格遵循 spec，冲突以 spec 为准。
- Do not copy/paste spec contents into docs; reference by file/section title only.
- Never hardcode API paths; use endpoint constants.

## Key Spec Files (quick lookup)
- `spec/api_spec_index.mdc` index
- `spec/api_spec_types.mdc` shared types
- `spec/api_spec_errors.mdc` error codes
- `spec/api_spec_websocket.mdc` WS events
- `spec/backend_rule.mdc` backend rules
- `spec/frontend_rule.mdc` frontend rules

## Project Layout
- `spec/` API specs + rules; `backend/` Rust; `frontend_svelte/` SvelteKit; `harmonyapp/` HarmonyOS.

## Build / Lint / Test Commands
### Backend (Rust)
```bash
cd backend
cargo run
RUST_LOG=debug cargo run
cargo build --release
cargo check
cargo fmt
cargo clippy
cargo test
cargo test --test user_tests
cargo test test_create_user
cargo test -- --nocapture
```

### Frontend (SvelteKit)
```bash
cd frontend_svelte
pnpm install
pnpm dev
pnpm dev -- --open
pnpm build
pnpm preview
pnpm check
pnpm check:watch
```
- No unit-test runner configured in `frontend_svelte/package.json`.

### HarmonyOS (ETS / Hvigor)
When modifying `harmonyapp/` or collecting build output, ask the user for DevEco Studio install dir or Command Line Tools dir. Derive `DEVECO_SDK_HOME` from that location.
```bash
# DevEco Studio (macOS)
DEVECO_SDK_HOME="$DEVECO_STUDIO_DIR/Contents/sdk" node "$DEVECO_STUDIO_DIR/Contents/tools/hvigor/bin/hvigorw.js" --tasks   # or type-check
# DevEco Studio (non-macOS)
DEVECO_SDK_HOME="$DEVECO_STUDIO_DIR/sdk" node "$DEVECO_STUDIO_DIR/tools/hvigor/bin/hvigorw.js" --tasks   # or type-check
# Command Line Tools (all platforms)
DEVECO_SDK_HOME="$COMMANDLINE_TOOL_DIR/sdk" node "$COMMANDLINE_TOOL_DIR/hvigor/bin/hvigorw.js" --tasks   # or type-check
```
- Hvigor tasks are provided by `appTasks`/`hapTasks`; confirm with `--tasks`.
- Linter configuration: `harmonyapp/code-linter.json5`.
- Use DevEco Studio when Hvigor tasks are unclear.

## Backend (Rust) Style
- Architecture: Handler -> Service -> Repository. Keep SQL in repositories only.
- Handlers: param extraction + response wrapping only.
- Services: business logic, permission checks, transactions.
- Repositories: SQL only, no business logic.
- Imports order:
```rust
// 1) std
use std::sync::Arc;

// 2) external crates
use axum::{Json, extract::State};
use sqlx::PgPool;

// 3) local modules
use crate::error::AppError;
use crate::dto::UserResponse;
```
- Naming: files snake_case, structs PascalCase, functions snake_case, consts SCREAMING_SNAKE_CASE.
- Formatting: `cargo fmt` (rustfmt).
- Error handling: prefer `Result<T, AppError>` + `?`; never `unwrap()` in production code.
- Types: explicit in public APIs; avoid dynamic map types.
- SQL: prefer `sqlx::query!` / `query_as!`; avoid string-built SQL.
- Logging: use `tracing`, avoid `println!`.
- Geo: do not compute distance/angles in Rust; use PostGIS (`ST_Distance`, `ST_DWithin`) + GIST index.
- Response envelope: `ApiResponse<T>`; business success is `code === 0`.

## Frontend (Svelte/TS) Style
- Naming: `kebab-case.svelte`, `+page.svelte`, `+layout.svelte`, `kebab-case.ts`.
- Indentation: follow existing tabs in TS/Svelte files.
- Imports order:
```ts
// 1) Svelte/SvelteKit
import { dev } from '$app/environment';
import { goto } from '$app/navigation';

// 2) local libs/types
import type { ApiResponse } from '$lib/types/api';
import { post } from '$lib/api/client';
import { ENDPOINTS } from '$lib/api/endpoints';

// 3) components
import UserCard from '$lib/components/user-card.svelte';
```
- Types: no `any`; define all API types in `frontend_svelte/src/lib/types/api.ts`.
- API calls: use `$lib/api/client.ts` + `frontend_svelte/src/lib/api/endpoints.ts`.
- Error handling: HTTP errors (network/5xx) vs business errors (`code !== 0`).
- Only `code === 0` indicates business success.
- Use optional chaining (`?.`) and nullish coalescing (`??`) for optional fields.
- Cursor values are opaque; do not decode or parse.

## HarmonyOS (ETS) Style
- Files: `.ets` in PascalCase for components/pages; constants in `*Constants.ets`.
- Indentation: follow existing 2-space ETS style.
- Imports order: system/ohos -> core utilities -> features/components.
- Types: explicit parameter/return types and `Promise<T>`; avoid `any`.
- API types live in `harmonyapp/entry/src/main/ets/core/types/api.ets`.
- Endpoints/client: use `ApiEndpoints` + `ApiClient`
  (`harmonyapp/entry/src/main/ets/core/config/ApiEndpoints.ets`).
- Network: go through `HttpUtil`; WebSocket through `WebSocketService`.
- Error handling: same HTTP vs business split; `code === 0` only success.
- Lint/security: follow `code-linter.json5` (no unsafe crypto).

## Error Handling Rules (All Clients)
- HTTP layer error: request failed (non-200, network, timeout) -> treat as network failure.
- Business layer error: HTTP 200 but `code !== 0` -> show domain error.
- Always log or surface `request_id` when available.

## Pagination & Cursor
- Cursor is opaque; pass through only.
- Use `PaginationMeta` / `PaginatedData` types.
- Never store decoded cursor or assume format.

## Secrets & Config
- Do not commit secrets or `.env` files.
- Backend config uses env vars; keep defaults in `.env.example` only.
- Frontend config should avoid embedding secrets; use public envs only.

## Common Pitfalls
- Hardcoded API paths or ad-hoc endpoints; mixing HTTP status with business success.
- SQL or business logic inside handlers.
- Rust `unwrap()` or manual geo distance math.
