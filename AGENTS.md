# HotSpot AI Agent Guidelines

## Project Overview
- Backend: Rust (Axum 0.7+) + PostgreSQL + Redis + PostGIS
- Frontend: Next.js 15 (App Router) + TypeScript + TanStack Query + Zustand
- API Style: JSON-RPC (all endpoints use POST, unified `/api/v1/` prefix)

## Build & Test Commands

### Frontend (`frontend_react/`)
```bash
cd frontend_react
pnpm install              # Install dependencies
pnpm dev                  # Start dev server (localhost:3000)
pnpm build                # Production build
pnpm lint                 # Run ESLint
pnpm type-check           # TypeScript type check
```

### Backend (`backend/`)
```bash
cd backend
cargo build               # Build project
cargo run                 # Run server (localhost:8080)
cargo test                # Run tests
cargo clippy              # Lint
cargo fmt                 # Format code
```

### Run Single Test
```bash
# Backend
cargo test --test <test_name>

# Frontend (if configured)
pnpm test -- <test_file>
```

## Critical Architectural Rules

### Backend (Rust/Axum)
- **Layered**: Handler → Service → Repository (strictly one-way)
- **Handler**: Only extract params/call Service/return ApiResponse. NO business logic, NO direct DB access
- **Service**: All business logic, permissions, transactions, cache. NO SQL writing
- **Repository**: SQL encapsulation & data mapping only. NO business logic
- **Auth**: JWT with `sid`, bcrypt password hashing (work factor 10-12)
- **Pagination**: Cursor/Keyset based only. NO OFFSET
- **Geo**: All distance/radius/distance sorting in PostGIS (GIST index). NO client-side calculations
- **Response**: Unified `ApiResponse<T>` with `code/message/data/request_id`. Business success = `code === 0`, NOT HTTP status
- **IDs**: `user_id` = i64, other IDs = UUID strings

### Frontend (Next.js/React)
- **API Calls**: All through `src/lib/api/services/*`. NO direct fetch in components/hooks/pages
- **POST Only**: All API endpoints use POST method. NO GET/PUT/DELETE (per @api_spec.mdc)
- **Paths**: Use `src/lib/api/endpoints.ts` constants. NO hardcoded `/api/v1/` strings
- **Success**: Check `ApiResponse.code === 0`. NOT HTTP status code
- **SSR**: Server Components for initial data fetch, Client Components for interactivity
- **Server Actions**: Use `'use server'` for mutations, pair with `revalidateTag`
- **State**: Zustand (client state) + TanStack Query (server state)
- **Cursor**: Opaque string. NO parsing/decoding/construction by frontend
- **Messages**: API only submits intent + gets `message_id`. UI only shows WebSocket-pushed messages. NO API→UI direct insert
- **WebSocket**: Event types from `@api_spec_websocket.mdc`. NO local event definitions

## Code Style Guidelines

### Imports
- **Backend**: Use `use` statements, group std/external/third-party
- **Frontend**: Absolute imports from `@/`. NO relative paths like `../components`

### Naming
- **Backend**: `snake_case` for all identifiers
- **Frontend**: `camelCase` for variables/functions, `PascalCase` for components/types

### TypeScript
- **STRICT mode**. NO `any` or `as any`
- Optional fields: Use `?.` or `??` for safe access
- All API responses must have type definitions in `src/types/api.ts`

### Error Handling
- **Backend**: Use `thiserror::Error` enum, propagate with `?`. Error codes from `@api_spec_errors.mdc`
- **Frontend**: Check `ApiResponse.code === 0`, use toast notifications for errors

### Data Integrity
- **Backend**: Service layer validates all references (use `repo.exists()` before inserts)
- **Frontend**: Use optimistic UI updates, handle rollback on failure

## API Contract Compliance
- **Single source of truth**: `@spec/api_spec*.mdc` defines all endpoints/fields/errors/WS events
- **NO duplication**: Don't copy endpoint/field lists into code or docs
- **References**: Use section titles from specs (NO "Chapter X/Y" numbering)

## WebSocket
- **Heartbeat**: 30s ping, 10s pong timeout, exponential backoff reconnect
- **Auth**: Complete within 5s of connection
- **Subscriptions**: Use `useWebSocket` hook, manage through event handlers

## Security
- **Password**: TLS plaintext submission + bcrypt storage
- **Tokens**: Access token in memory (Zustand), refresh in httpOnly cookie
- **SQL**: Parameterized queries (SQLx) to prevent injection

## Location Services
- **Forbidden**: NO Haversine/distance calculations in Rust or JS
- **Required**: Use PostGIS `ST_Distance()`, `ST_DWithin()` with GIST indexes

## Before Committing
1. Run lint/typecheck commands
2. Verify no `any` types (frontend) or unwrap() abuse
3. Check all API calls use POST method
4. Ensure cursor is never parsed/constructed
5. Verify success checks use `code === 0`
