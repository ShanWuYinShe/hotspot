# HotSpot AI Agent Guidelines

## Project Overview

- **Backend**: Rust (Axum 0.7+) + PostgreSQL + Redis + PostGIS
- **Frontend**: Next.js 16 (App Router) + React 19 + TypeScript + TanStack Query + Zustand
- **Mobile**: HarmonyOS (ArkTS)
- **API Style**: JSON-RPC (all endpoints use POST, unified `/api/v1/` prefix)

---

## Build, Lint, and Test Commands

### Frontend (`frontend_react/`)

```bash
cd frontend_react

# Development
pnpm dev                  # Start dev server (localhost:3000)

# Build
pnpm build                # Production build
pnpm start                # Start production server

# Linting
pnpm lint                 # Run ESLint
pnpm lint:fix             # Auto-fix linting issues

# Type Checking
pnpm type-check           # TypeScript type check

# Testing (if configured)
pnpm test                 # Run all tests
pnpm test -- <test_file>  # Run single test file
```

### Backend (`backend/`)

```bash
cd backend

# Build
cargo build               # Build project
cargo build --release      # Release build

# Run
cargo run                 # Run server (localhost:8080)
cargo run --release       # Run release server

# Testing
cargo test                # Run all tests
cargo test --test <test_name>  # Run single test
cargo test -- --nocapture  # Run with output

# Linting
cargo clippy              # Lint with Clippy
cargo clippy --fix        # Auto-fix Clippy warnings

# Formatting
cargo fmt                 # Format code
cargo fmt -- --check       # Check formatting without changes
```

### Important Rule for Dev Server Management

**CRITICAL**: When running the frontend dev server for testing, you MUST shut it down immediately after use:

```bash
# After completing testing or before finishing work
pkill -f "next dev"
pkill -f "node.*next"
```

**DO NOT leave the dev server running** - The user will run it themselves. Only start it when specifically needed for testing, and stop it immediately after.

---

## Critical Architectural Rules

### Backend (Rust/Axum)

- **Layered**: Handler → Service → Repository (strictly one-way data flow)
- **Handler**: Only extract params, call Service, return ApiResponse. NO business logic, NO direct DB access
- **Service**: All business logic, permissions, transactions, cache. NO SQL writing
- **Repository**: SQL encapsulation & data mapping only. NO business logic
- **Auth**: JWT with `sid`, bcrypt password hashing (work factor 10-12)
- **Pagination**: Cursor/Keyset based only. NO OFFSET
- **Geo**: All distance/radius/sorting in PostGIS (GIST index). NO client-side calculations
- **Response**: Unified `ApiResponse<T>` with `code/message/data/request_id`. Business success = `code === 0`, NOT HTTP status
- **IDs**: `user_id` = i64, other IDs = UUID strings

### Frontend (Next.js/React)

- **API Calls**: All through `src/lib/api/services/*`. NO direct fetch in components/hooks/pages
- **POST Only**: All API endpoints use POST method. NO GET/PUT/DELETE (per @spec/api_spec.mdc)
- **Paths**: Use `src/lib/api/endpoints.ts` constants. NO hardcoded `/api/v1/` strings
- **Success**: Check `ApiResponse.code === 0`. NOT HTTP status code
- **SSR**: Server Components for initial data fetch, Client Components for interactivity
- **Server Actions**: Use `'use server'` for mutations, pair with `revalidateTag`
- **State**: Zustand (client state) + TanStack Query (server state)
- **Cursor**: Opaque string. NO parsing/decoding/construction by frontend
- **Messages**: API only submits intent + gets `message_id`. UI only shows WebSocket-pushed messages. NO API→UI direct insert
- **WebSocket**: Event types from `@api_spec_websocket.mdc`. NO local event definitions
- **Dev Server Management**: DO NOT automatically run `pnpm dev`. The user will run it themselves. If you must run it for testing, shut it down immediately after use with `pkill -f "next dev"` or `pkill -f "node.*next"`

---

## Code Style Guidelines

### Imports

#### Backend (Rust)

```rust
use std::collections::HashMap;
use axum::{extract::State, Json};
use serde::{Deserialize, Serialize};
use sqlx::PgPool;

use crate::models::User;
use crate::services::UserService;
```

- Group std/external/third-party imports with empty lines between groups
- Use `use crate::` for internal imports
- Absolute imports are recommended over relative paths

#### Frontend (Next.js/React)

```typescript
// ✅ CORRECT: Absolute imports from @/
import { Button } from "@/components/ui/button";
import { useState } from "react";
import { FeedService } from "@/lib/api/services";

// ❌ INCORRECT: Relative paths
import { Button } from "../components/ui/button";
import { FeedService } from "../../lib/api/services";
```

- **Rule**: Always use absolute imports from `@/` (configured in tsconfig.json)
- **NO relative paths** like `../components/Button` or `../../lib/utils`

### Naming Conventions

#### Backend (Rust)

```rust
// Types/Structs
struct User {
    user_id: i64,
    nickname: String,
}

// Variables
let user_id: i64 = 123;
let current_time: u64 = get_timestamp();

// Functions
fn get_user_profile(user_id: i64) -> Result<User> {
    // ...
}
```

- **Variables/Functions**: `snake_case`
- **Types/Structs**: `PascalCase`
- **Constants**: `UPPER_SNAKE_CASE`

#### Frontend (Next.js/React)

```typescript
// Components
export function UserAvatar() {}
export function FeedList() {}

// Hooks
export function useAuth() {}
export function useWebSocket() {}

// Types/Interfaces
export interface ApiResponse<T> {}
export type FeedSource = "all" | "following" | "nearby";

// Variables/Functions
const userId = 123;
const isLoading = false;
function getUserProfile() {}
```

- **Variables/Functions**: `camelCase` (e.g., `getUserInfo`, `isLoading`)
- **Components**: `PascalCase` (e.g., `Button`, `AuthTabs`)
- **Types/Interfaces**: `PascalCase` (e.g., `ApiResponse<T>`, `AuthData`)
- **Constants**: `UPPER_SNAKE_CASE` (e.g., `API_BASE`, `ENDPOINTS`)

### TypeScript

#### Configuration

```json
{
  "compilerOptions": {
    "strict": true, // STRICT mode enabled
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "noPropertyAccessFromIndexSignature": true
  }
}
```

#### Rules

- **STRICT mode**: Enabled. NO `any` or `as any` types
- **Optional fields**: Use `?.` or `??` for safe access
- **All API responses** must have type definitions in `src/types/api.ts`

```typescript
// ✅ CORRECT
const user = response.data?.user ?? null;
const name = user?.nickname ?? "Anonymous";

// ❌ INCORRECT
const user = response.data!.user as any;
```

### Error Handling

#### Backend (Rust)

```rust
use thiserror::Error;

#[derive(Error, Debug)]
pub enum UserServiceError {
    #[error("User not found")]
    NotFound,
    #[error("Invalid credentials")]
    InvalidCredentials,
}

pub async fn get_user(
    pool: &PgPool,
    user_id: i64
) -> Result<User, UserServiceError> {
    // Service layer validates
    if !repo.exists(pool, user_id).await? {
        return Err(UserServiceError::NotFound);
    }
    repo.get_user(pool, user_id).await.map_err(Into::into)
}
```

- **Use**: `thiserror::Error` enum for domain errors
- **Propagate**: Use `?` operator for error propagation
- **Error codes**: From `@api_spec_errors.mdc`
- **Validation**: Service layer validates all references

#### Frontend (Next.js/React)

```typescript
// Check API response
import { toast } from "@/hooks/use-toast";
import { isSuccess } from "@/lib/api/client";

const response = await feedService.getMoments();

if (isSuccess(response)) {
  // Success handling
  setMoments(response.data?.items ?? []);
} else {
  // Error handling
  toast.error("加载失败", response.message);
}

// ✅ CORRECT: Check code === 0
if (response.code === 0) {
}

// ❌ INCORRECT: Check HTTP status
if (response.ok) {
}
```

- **Check**: `ApiResponse.code === 0` for business success
- **NOT**: HTTP status code
- **Use**: Toast notifications for errors
- **Server Components**: Return user-friendly error UI on data load failure

### Data Integrity

#### Backend (Rust)

```rust
// Service layer validation
pub async fn create_moment(
    pool: &PgPool,
    data: CreateMomentRequest
) -> Result<Moment, UserServiceError> {
    // Validate user exists
    if !repo.exists(pool, data.user_id).await? {
        return Err(UserServiceError::NotFound);
    }

    // Validate references
    if data.mentions.is_some() && !users_exist(pool, &data.mentions.unwrap()).await? {
        return Err(UserServiceError::InvalidMentions);
    }

    repo.insert_moment(pool, &data).await.map_err(Into::into)
}
```

- **Service layer**: Validates all references using `repo.exists()` before inserts
- **Use**: Repository for all SQL operations

#### Frontend (Next.js/React)

```typescript
// Optimistic UI updates
const [moments, setMoments] = useState(initialMoments);

const handleLike = async (momentId: string) => {
  // Optimistic update
  setMoments((prev) =>
    prev.map((m) => (m.moment_id === momentId ? { ...m, is_liked: true } : m)),
  );

  const result = await feedService.likeMoment(momentId);

  if (result.code !== 0) {
    // Rollback on failure
    setMoments((prev) =>
      prev.map((m) =>
        m.moment_id === momentId ? { ...m, is_liked: false } : m,
      ),
    );
    toast.error("操作失败", result.message);
  }
};
```

- **Use**: Optimistic UI updates
- **Handle**: Rollback on failure

---

## API Contract Compliance

- **Single source of truth**: `@spec/api_spec*.mdc` defines all endpoints/fields/errors/WS events
- **NO duplication**: Don't copy endpoint/field lists into code or docs
- **References**: Use section titles from specs (NO "Chapter X/Y" numbering)

**API Specification Files:**

- `@spec/api_spec.mdc` - Main API specification
- `@spec/api_spec_auth.mdc` - Authentication endpoints
- `@spec/api_spec_user.mdc` - User management endpoints
- `@spec/api_spec_moment.mdc` - Moments/posts endpoints
- `@spec/api_spec_chat.mdc` - Chat (private & group) endpoints
- `@spec/api_spec_group.mdc` - Group chat endpoints
- `@spec/api_spec_feed.mdc` - Feed/trending endpoints
- `@spec/api_spec_notification.mdc` - Notification endpoints
- `@spec/api_spec_system.mdc` - System config endpoints
- `@spec/api_spec_errors.mdc` - Error code definitions
- `@spec/api_spec_websocket.mdc` - WebSocket event definitions

---

## WebSocket

### Connection Management

- **Heartbeat**: 30s ping, 10s pong timeout
- **Reconnect**: Exponential backoff
- **Auth**: Complete within 5s of connection

### Subscriptions

- **Use**: `useWebSocket` hook
- **Manage**: Through event handlers
- **Event Types**: From `@api_spec_websocket.mdc` (NO local definitions)

### Message Flow

1. Client connects → WebSocket handshake
2. Auth request/response → Bearer token
3. Subscribe to topics → Subscription confirmation
4. Receive messages → Update UI via event handlers
5. Heartbeat → Keep connection alive

---

## Security

### Password Handling

- **TLS**: Plaintext submission over HTTPS
- **Storage**: bcrypt hashing (work factor 10-12)

### Token Management

- **Access token**: In memory (Zustand store)
- **Refresh token**: In httpOnly cookie
- **Rotation**: Automatic refresh on 401

### SQL Injection Prevention

- **Use**: Parameterized queries (SQLx)
- **NO**: String concatenation or interpolation

---

## Location Services

### Forbidden Operations

- **NO**: Haversine formula or distance calculations in Rust or JavaScript
- **NO**: Client-side radius filtering or sorting

### Required Implementation

- **Use**: PostGIS `ST_Distance()`, `ST_DWithin()` with GIST indexes
- **All**: Distance/radius/sorting must be database-side

---

## Before Committing

1. **Run lint/typecheck commands**

   ```bash
   # Frontend
   cd frontend_react
   pnpm lint
   pnpm type-check

   # Backend
   cd backend
   cargo clippy
   cargo fmt -- --check
   ```

2. **Verify no `any` types** (frontend) or `unwrap()` abuse (backend)

3. **Check all API calls use POST method** (except system health check)

4. **Ensure cursor is never parsed/constructed** (opaque string only)

5. **Verify success checks use `code === 0`**

---

## HarmonyOS Specific Rules

**Location**: `.cursor/rules/harmony_app_rule.mdc` (943 lines)

### Evidence Output Template (Mandatory)

- **Purpose**: AI must output "verifiable proof" in actionable format, not just "complied/done"
- **Format**: Single sentence, objective, verifiable. No subjective words like "excellent/elegant/smooth"

### API Reference Rule (Enforced)

- All API endpoints/fields/types/events must reference `@api_spec.mdc` as single source of truth
- **NO duplication** of endpoint/field lists into this file
- **References allowed**: Section titles only, NO "Chapter X/Y" numbering

### Example Code Boundary

```typescript
// ✅ CORRECT: Use endpoint constants
import { ENDPOINTS } from "@/lib/api/endpoints";
await feedService.getMoments(ENDPOINTS.FEED.MOMENTS);

// ❌ INCORRECT: Hardcoded endpoints
await feedService.getMoments("/api/v1/feed/moments");
```

---

## File Structure Reference

### Frontend (`frontend_react/`)

```
src/
├── app/              # Next.js App Router
│   ├── (auth)/     # Auth pages
│   ├── (main)/     # Main app pages
│   ├── (admin)/    # Admin panel
│   ├── layout.tsx   # Root layout
│   └── globals.css  # Global styles
├── components/      # React components
│   ├── ui/         # shadcn/ui base components
│   ├── common/      # Shared business components
│   ├── icons/       # Lucide React icons
│   └── [domain]/   # Feature-specific components
├── hooks/          # Custom React hooks
├── lib/            # Utilities
│   ├── api/        # API client & services (17 files)
│   ├── auth/        # Auth utilities
│   └── utils/       # Helper functions
├── stores/          # Zustand state
└── types/          # TypeScript type definitions
```

### Key Files

- **API Client**: `src/lib/api/client.ts` - Unified HTTP client with token injection
- **Endpoints**: `src/lib/api/endpoints.ts` - Centralized API path constants
- **Services**: `src/lib/api/services/` - Business logic layer (17 service files)
- **Auth Store**: `src/stores/auth.ts` - Multi-account state management
- **Types**: `src/types/api.ts` - Complete API type definitions

---

## Testing Guidelines

### Backend (Rust)

- **Unit tests**: Test functions and modules independently
- **Integration tests**: Test API handlers with database
- **Use**: `tokio::test` for async tests
- **Mock**: External dependencies (database, API clients)

### Frontend (Next.js/React)

- **Component tests**: Test UI components with React Testing Library
- **Hook tests**: Test custom hooks independently
- **Integration tests**: Test API calls and state management
- **Mock**: API services and external dependencies

---

## Quick Reference

### POST-Only API Endpoints

All API endpoints use POST method with request body:

- Authentication: `POST /api/v1/auth/login`
- User profile: `POST /api/v1/user/profile`
- Moments: `POST /api/v1/moments/create`
- Comments: `POST /api/v1/comments/create`
- Chat: `POST /api/v1/private_chats/send`
- Etc.

### Success Check Pattern

```typescript
// ✅ CORRECT: Check business code
if (response.code === 0) {
  // Success
}

// ❌ INCORRECT: Check HTTP status
if (response.ok) {
  // Wrong
}
```

### Import Pattern

```typescript
// ✅ CORRECT: Absolute from @/
import { Button } from "@/components/ui/button";

// ❌ INCORRECT: Relative path
import { Button } from "../components/ui/button";
```

---

## Notes for AI Agents

1. **Always check existing files** before creating new ones
2. **Follow the established patterns** in the codebase
3. **Run lint/typecheck** before committing changes
4. **Stop dev server** immediately after testing if you started it
5. **Reference specs** - use `@spec/api_spec*.mdc` for API contracts, don't duplicate
