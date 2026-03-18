# Backend - Agent Guide

> Rust / Axum 后端服务的 AI Agent 操作指南。

## 使用范围

本文件只补充 Rust 后端特有的硬约束（分层、SQLx、地理查询等），其他实现细节和功能结构请 Agent 通过代码与 `spec/` 自行探索。

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

> Handler / Service / Repository 的具体文件命名和目录结构，以现有代码为准，本文件不做约束。

## SQL 规则（严格执行）

### 必须使用 SQLx 宏

```rust
// ✅ sqlx::query_as!(...) — 编译期检查
// ❌ sqlx::query("...").bind(x) — 无编译期检查，禁止使用
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

- 使用 `thiserror` 定义 `ApiError` 枚举。
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

## 额外禁止

除根目录 `AGENTS.md` 中的全局禁止项外，后端额外禁止：
- 绕过 Handler → Service → Repository 分层直接跨层访问。
- 在 Handler 或 Service 中直接写 SQL，或使用非宏 `sqlx::query("...")` 系列。
- 使用 OFFSET 进行分页，或在 Rust 代码中自行计算距离/角度。

## 安全规则

文件删除、批量变更、Git 操作规范、数据库操作安全等全局规则见根目录 `AGENTS.md` 的「权限边界」和「Git 操作规范」章节，此处不重复。

> **⚠️ `tmp/` 路径规则**：所有临时文件必须存放在项目根目录的 `tmp/` 下。详见根目录 `AGENTS.md` 的「Agent 行为准则」章节。

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
