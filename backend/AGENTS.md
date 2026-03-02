# Backend - Agent Guide

> Rust / Axum 后端服务的 AI Agent 操作指南。

## 技术栈

- **语言**：Rust (Edition 2021)
- **Web 框架**：Axum
- **运行时**：Tokio
- **数据库**：PostgreSQL + PostGIS，通过 SQLx（编译期宏）访问
- **缓存**：Redis + Moka（本地缓存）
- **认证**：JWT (jsonwebtoken) + bcrypt
- **存储**：AWS S3 SDK (MinIO 兼容)
- **日志**：tracing + tracing-subscriber

> 具体版本以 `Cargo.toml` 为准。

## 架构分层（强制）

```
Handler → Service → Repository
```

**严格禁止跨层调用。** 每一层的职责不可混淆：

| 层 | 职责 | 禁止 |
|----|------|------|
| **Handler** | 提取参数 → 调 Service → 返回 `ApiResponse` | 写 SQL、写业务逻辑、直接访问数据库 |
| **Service** | 业务规则、权限校验、事务、缓存、事件发布 | 写 SQL、直接构造 HTTP 响应 |
| **Repository** | SQL 执行与结果映射 | 业务判断、权限校验、发事件 |

### 正确示例

```rust
// Handler —— 只做提取和调用
pub async fn create_moment(
    State(state): State<AppState>,
    Extension(user_id): Extension<String>,
    Json(req): Json<CreateMomentRequest>,
) -> impl IntoResponse {
    match state.moment_service.create(&user_id, req).await {
        Ok(data) => ApiResponse::success(data),
        Err(e) => ApiResponse::from(e),
    }
}
```

### 错误示例

```rust
// 错误：Handler 中写业务逻辑
pub async fn create_moment(/* ... */) -> impl IntoResponse {
    // ❌ 不要在 Handler 里做业务校验
    if req.content.len() > 500 { return ApiResponse::error(10000, "too long"); }
    // ❌ 不要在 Handler 里直接查数据库
    let user = sqlx::query_as!(User, "SELECT ...", user_id).fetch_one(&pool).await?;
}
```

## SQL 规则（严格执行）

### 必须使用 SQLx 宏

```rust
// ✅ 正确：使用宏，编译期检查
let user = sqlx::query_as!(User, "SELECT * FROM users WHERE id = $1", user_id)
    .fetch_one(&pool)
    .await?;

// ❌ 禁止：非宏调用（无编译期检查）
let user = sqlx::query("SELECT * FROM users WHERE id = $1")
    .bind(user_id)
    .fetch_one(&pool)
    .await?;
```

**禁止的写法**（会被 rg 扫描拦截）：
- `sqlx::query(` — 使用 `sqlx::query!` 替代
- `sqlx::query_as::<` — 使用 `sqlx::query_as!` 替代
- `sqlx::query_scalar::<` — 使用 `sqlx::query_scalar!` 替代
- `sqlx::query_as_unchecked!` — 不应使用

### 分页

- 只使用游标/keyset 分页，**禁止 OFFSET**。
- cursor 编解码仅在 Repository 层完成。
- cursor 对上层代码不透明。

### 地理查询

- 距离计算、范围查询、排序**必须在 PostGIS 中完成**。
- **禁止在 Rust 代码中计算距离或角度**。

## 错误处理

- 使用 `thiserror` 定义 `AppError` 枚举。
- 业务错误码必须来自 `spec/api_spec_errors.md`，禁止自行编号。
- 多个 Rust variant 可映射同一错误码（便于可读性）。
- 服务端内部错误统一映射到 `10600`，不泄露技术细节。
- `sqlx::Error`、`redis::RedisError` 等统一映射到 `10600`。
- 日志必须带 `X-Request-Id`，禁止记录密码、Token、隐私内容。

## 认证与会话

- JWT 仅包含 `user_id` 和 `session_id`，不写入可变字段。
- Web 端：HttpOnly Cookie（`access_token` + `refresh_token`）+ CSRF。
- App 端：`Authorization: Bearer`。
- 登出/撤销：写入黑名单，校验时必须查黑名单。
- Token 缓存：L1 Moka（60s）+ L2 Redis session 白名单。

## 中间件链路

路由级中间件按以下顺序串联：

| 认证级别 | 中间件链 |
|----------|----------|
| Token 可选 | `optional_auth` → `rate_limit` → `resource_guard` → `idempotency` |
| Token 必须 | `auth` → `rate_limit` → `resource_guard` → `idempotency` |
| 管理员 | `auth` → `admin_auth` → `rate_limit` → `resource_guard` → `idempotency` |

## 事件系统

- 新增事件时须同步更新事件类型定义和对应领域的 handler 分发逻辑。

## 常见错误清单

以下是 Agent 最容易犯的错误，每次编写代码前检查：

- [ ] 是否在 Handler 里写了业务逻辑或 SQL？
- [ ] 是否使用了 `sqlx::query(` 而非 `sqlx::query!`？
- [ ] 是否使用了 OFFSET 分页？
- [ ] 是否在 Rust 中计算了距离/角度（应在 PostGIS 中）？
- [ ] 是否使用了 spec 中不存在的错误码？
- [ ] 是否在 JWT 中写入了可变字段？
- [ ] 是否遗漏了 MUST 幂等接口的幂等键支持？
- [ ] 是否用 `#[allow(...)]` 压制了编译警告？

## 安全规则

文件删除、批量变更、Git 操作规范、数据库操作安全等全局规则见根目录 `AGENTS.md` 的「权限边界」和「Git 操作规范」章节，此处不重复。

**后端额外约束**：
- 禁止自行执行 `deploy/` 目录下的脚本，须告知用户脚本内容和影响后由用户执行。
- 禁止自行执行数据库迁移/重置命令（`sqlx migrate run`、`sqlx database reset`、`psql` 等），须告知用户后由用户手动执行。

## 验证命令

```bash
# 编译检查
cargo check

# Lint
cargo clippy

# 禁止模式扫描
rg 'sqlx::query\(' src/
rg 'sqlx::query_as::<' src/
rg 'sqlx::query_scalar::<' src/
rg 'sqlx::query_as_unchecked!' src/
```
