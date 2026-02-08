# HotSpot 项目 - AI 编程助手指南

> 本文档专为 AI 编码助手（如 Claude、GitHub Copilot）设计，包含项目结构、开发规范、构建命令等核心信息。

## 📁 项目结构

```
hotspot/
├── spec/                    # API 规范文档（契约唯一权威来源）
│   ├── api_spec.mdc         # 主规范索引
│   ├── api_spec_*.mdc       # 各模块 API 定义
│   ├── backend_rule.mdc     # 后端实现规范
│   ├── backend_doc.mdc      # 后端功能文档
│   ├── frontend_rule.mdc    # 前端实现规范
│   └── frontend_doc.mdc     # 前端功能文档
├── backend/                 # Rust 后端（Axum 0.8 + PostgreSQL + Redis）
│   ├── src/                 # 源代码
│   ├── migrations/          # 数据库迁移
│   └── Cargo.toml           # Rust 依赖
└── frontend_svelte/         # Svelte 5 前端（SvelteKit）
    ├── src/                 # 源代码
    ├── package.json         # Node 依赖
    └── vite.config.ts       # Vite 配置
```

## 🔑 核心原则

### 1. **契约唯一权威**
- 所有 API 端点、字段、类型、错误码、WebSocket 事件以 `spec/api_spec*.mdc` 为准
- **禁止**在代码中硬编码 API 路径（如 `/api/v1/...`），必须使用常量引用
- **禁止**在任何文档中复制粘贴 API 规范内容

### 2. **严格分层架构**
```
Handler（参数提取）→ Service（业务逻辑）→ Repository（SQL 封装）
```
- Handler：只做参数提取和响应封装
- Service：所有业务逻辑、权限验证、事务管理
- Repository：只封装 SQL，不含业务判断

### 3. **统一响应格式与错误分层**
```typescript
interface ApiResponse<T> {
  code: number;        // 0 = 成功，非 0 = 业务错误码
  message?: string;    // 错误描述
  data?: T;            // 响应数据
  request_id: string;  // 请求追踪 ID
}
```

**错误处理两个层次（重要）**：

1. **HTTP 层错误**（网络/连接错误）：
   - HTTP status !== 200（如 500、502、503、超时等）
   - 请求根本没到业务层
   - 前端应作为"请求失败/网络错误"处理
   - 不涉及业务逻辑，不使用 `code` 字段

2. **业务层错误**（业务逻辑错误）：
   - HTTP status === 200（请求成功到达业务层）
   - 业务成功：`code === 0`
   - 业务失败：`code !== 0`（如 10001 用户不存在、10200 权限不足等）
   - **只能**通过 `code === 0` 判断业务成功，**禁止**使用 HTTP 状态码

## 🛠️ 后端开发（Rust/Axum）

### 构建与运行
```bash
cd backend

# 开发模式
cargo run

# 详细日志
RUST_LOG=debug cargo run

# 生产构建
cargo build --release
./target/release/hotspot-backend

# 类型检查（快速）
cargo check

# 格式化代码
cargo fmt

# Lint 检查
cargo clippy
```

### 测试
```bash
# 运行所有测试
cargo test

# 运行单个测试模块
cargo test --test user_tests

# 运行单个测试函数
cargo test test_create_user

# 带输出运行
cargo test -- --nocapture

# 集成测试
cargo test --test '*' -- --test-threads=1
```

### 代码规范

#### 命名约定
- 文件/模块：`snake_case`（如 `user_service.rs`）
- 类型/结构体：`PascalCase`（如 `UserInfo`）
- 函数/变量：`snake_case`（如 `create_user`）
- 常量：`SCREAMING_SNAKE_CASE`（如 `JWT_SECRET`）

#### 导入顺序
```rust
// 1. 标准库
use std::sync::Arc;

// 2. 外部 crate
use axum::{Json, extract::State};
use sqlx::PgPool;

// 3. 本地模块
use crate::error::AppError;
use crate::dto::UserResponse;
```

#### 类型约束
- **必须**：明确的类型注解（函数签名、公开字段）
- **禁止**：`unwrap()` 在生产代码中使用，使用 `?` 或 `ok_or()`
- **推荐**：使用 `Result<T, AppError>` 处理错误

#### 错误处理
```rust
// ✅ 正确
pub async fn get_user(db: &PgPool, user_id: i64) -> Result<User, AppError> {
    UserRepository::find_by_id(db, user_id)
        .await?
        .ok_or(AppError::UserNotFound)
}

// ❌ 错误
pub async fn get_user(db: &PgPool, user_id: i64) -> User {
    UserRepository::find_by_id(db, user_id).await.unwrap()  // 可能 panic
}
```

#### 地理位置计算
- **禁止**在 Rust 中计算距离/角度
- **必须**使用 PostgreSQL/PostGIS 计算（`ST_Distance`、`ST_DWithin`）
- **必须**为地理位置字段创建 GIST 索引

## 🎨 前端开发（Svelte 5/SvelteKit）

### 构建与运行
```bash
cd frontend_svelte

# 安装依赖
pnpm install

# 开发模式
pnpm dev

# 开发模式（自动打开浏览器）
pnpm dev -- --open

# 生产构建
pnpm build

# 预览生产构建
pnpm preview

# 类型检查
pnpm check

# 类型检查（监听模式）
pnpm check:watch
```

### 代码规范

#### 文件命名
- 组件：`kebab-case.svelte`（如 `user-card.svelte`）
- 路由页面：`+page.svelte`、`+layout.svelte`
- TypeScript：`kebab-case.ts`（如 `api-client.ts`）

#### 导入顺序
```typescript
// 1. Svelte/SvelteKit
import { dev } from '$app/environment';
import { goto } from '$app/navigation';

// 2. 本地库
import type { ApiResponse } from '$lib/types/api';
import { post } from '$lib/api/client';
import { ENDPOINTS } from '$lib/api/endpoints';

// 3. 组件
import UserCard from '$lib/components/user-card.svelte';
```

#### 类型约束
- **禁止**使用 `any`
- **必须**为所有 API 响应定义类型
- **必须**使用可选链（`?.`）和空值合并（`??`）处理可选字段

#### API 调用规范
```typescript
// ✅ 正确：使用端点常量
import { ENDPOINTS } from '$lib/api/endpoints';
const response = await post(ENDPOINTS.AUTH.LOGIN, { login_id, password });

// ❌ 错误：硬编码路径
const response = await post('/api/v1/auth/login', { login_id, password });
```

#### 错误处理分层
```typescript
// ✅ 正确：区分 HTTP 层和业务层错误
try {
    const response = await post(ENDPOINTS.AUTH.LOGIN, { login_id, password });
    
    // 业务层判断：只有 HTTP 200 时才有 response.code
    if (response.code === 0 && response.data) {
        // 业务成功
    } else {
        // 业务失败（用户不存在、密码错误等）
        showError(response.message);
    }
} catch (error) {
    // HTTP 层错误：网络错误、超时、服务器 5xx 等
    showError('网络连接失败，请检查网络');
}

// ❌ 错误：混淆两个层次
if (response.ok) {  // HTTP 200 不代表业务成功
    // 错误：即使 HTTP 200，业务可能失败（code !== 0）
}
```

## 📋 常见开发任务

### 添加新 API 端点

#### 1. 后端（Rust）
```rust
// 1. 在 handlers/ 中定义处理器
pub async fn create_user(
    State(state): State<AppState>,
    Json(req): Json<CreateUserRequest>,
) -> Result<Json<ApiResponse<User>>, AppError> {
    let user = UserService::create(&state.db, req).await?;
    Ok(Json(ApiResponse::success(user)))
}

// 2. 在 routes/v1/ 中注册路由
router.route("/user/create", post(create_user))

// 3. 在 services/ 中实现业务逻辑
pub async fn create(db: &PgPool, req: CreateUserRequest) -> Result<User> {
    // 业务验证
    UserRepository::create(db, &req).await
}

// 4. 在 repositories/ 中实现数据访问
pub async fn create(db: &PgPool, req: &CreateUserRequest) -> Result<User> {
    sqlx::query_as!(User, "INSERT INTO users ...")
        .fetch_one(db)
        .await
        .map_err(Into::into)
}
```

#### 2. 前端（Svelte）
```typescript
// 1. 在 lib/api/endpoints.ts 中添加端点常量
export const USER = {
    CREATE: '/api/v1/user/create',
};

// 2. 在 lib/api/services/users.ts 中添加服务方法
export async function createUser(data: CreateUserRequest) {
    return post<User>(ENDPOINTS.USER.CREATE, data);
}

// 3. 在组件中调用
import { createUser } from '$lib/api/services/users';

const response = await createUser({ nickname, login_id, password });
if (response.code === 0 && response.data) {
    // 处理成功
}
```

### 数据库迁移

```bash
cd backend

# 创建新迁移文件
touch migrations/014_new_feature.sql

# 编写 SQL
vim migrations/014_new_feature.sql

# 运行迁移
psql $DATABASE_URL -f migrations/014_new_feature.sql
```

### WebSocket 事件处理

```typescript
// 前端：订阅 WebSocket 事件
import { wsClient } from '$lib/websocket';

// 订阅群组消息
wsClient.subscribe('group', groupId);

// 监听新消息事件
wsClient.on('ChatMessageNew', (event) => {
    // 更新 UI
});
```

## ⚠️ 常见陷阱

### 1. **API 路径硬编码**
```rust
// ❌ 错误
const url = "/api/v1/auth/login";

// ✅ 正确
use crate::routes::ENDPOINTS;
const url = ENDPOINTS.auth.login;
```

### 2. **混淆 HTTP 层和业务层错误**
```typescript
// ❌ 错误：混淆两个层次
if (response.status === 200) {  // HTTP 200 只说明请求到达业务层
    // 业务可能失败（code !== 0）
}

// ✅ 正确：区分两个层次
try {
    const response = await post(ENDPOINTS.AUTH.LOGIN, data);
    // HTTP 200 时才能判断业务成功
    if (response.code === 0) {
        // 业务成功
    } else {
        // 业务失败（code !== 0）
    }
} catch (error) {
    // HTTP 层错误（网络错误、5xx 等）
}
```

### 3. **Handler 中包含业务逻辑**
```rust
// ❌ 错误：在 Handler 中直接操作数据库
pub async fn create_user(State(state): State<AppState>, ...) -> Result<...> {
    sqlx::query!("INSERT INTO users ...").execute(&state.db).await?;
}

// ✅ 正确：调用 Service
pub async fn create_user(State(state): State<AppState>, ...) -> Result<...> {
    UserService::create(&state.db, req).await?;
}
```

### 4. **cursor 解析**
```typescript
// ❌ 错误：前端解析 cursor
const decoded = atob(cursor);

// ✅ 正确：cursor 是不透明字符串，只透传
const nextPage = await fetchData({ cursor: response.pagination.next_cursor });
```

### 5. **地理距离计算**
```rust
// ❌ 错误：在 Rust 中计算距离
fn haversine_distance(lat1: f64, lon1: f64, lat2: f64, lon2: f64) -> f64 { ... }

// ✅ 正确：使用 PostGIS
SELECT *, ST_Distance(location, $1) as distance FROM users WHERE ...
```

## 📚 文档参考

- **API 规范**：`spec/api_spec*.mdc`
- **后端规范**：`spec/backend_rule.mdc`
- **前端规范**：`spec/frontend_rule.mdc`
- **Cursor 规则**：`.cursor/rules/harmony_app_rule.mdc`

## 🔍 调试技巧

### 后端日志
```bash
# 查看详细日志
RUST_LOG=debug,hotspot_backend=trace cargo run

# 只看业务日志
RUST_LOG=info,hotspot_backend=debug cargo run
```

### 前端开发工具
- Svelte DevTools 浏览器扩展
- `console.log()` 调试（开发模式）
- TypeScript 类型检查：`pnpm check`

---

**重要提醒**：在修改代码前，务必先查阅 `spec/` 目录下的 API 规范文档，确保与契约保持一致。
