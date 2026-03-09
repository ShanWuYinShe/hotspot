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

### 示例

```rust
// ✅ Handler —— 只做提取和调用，禁止写业务逻辑或 SQL
match state.moment_service.create(&user_id, req).await {
    Ok(data) => ApiResponse::success(data),
    Err(e) => ApiResponse::from(e),
}
```

### Service 文件命名

- 大部分遵循 `<module>_service.rs` 模式（如 `auth_service.rs`、`moment_service.rs`）。
- 逻辑较大时拆为目录（如 `group_service/`），内部按子域拆分（`crud.rs`、`members.rs`、`applications.rs` 等）。
- 辅助/策略模块不带 `_service` 后缀（如 `private_chat_permission.rs`、`stranger_message_limiter.rs`）。

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

## 实现检查清单

- [ ] 架构一致性：有无类似功能？遵循分层模式（Handler → Service → Repository）？
- [ ] 类型复用：复用现有 DTO？可共享字段？
- [ ] 错误处理：复用 `ApiError` 或已有错误类型？
- [ ] 服务复用：扩展现有 Service 而非新建？
- [ ] 中间件复用：复用认证/限流/幂等中间件？
- [ ] SQL 复用：扩展现有 Repository 而非新查询？

**禁止**：绕过统一层、重复实现相同逻辑、引入不一致模式

## 常见错误

- [ ] Handler 里写业务逻辑或 SQL？
- [ ] 使用 `sqlx::query(` 而非 `sqlx::query!`？
- [ ] 使用 OFFSET 分页？
- [ ] Rust 中计算距离/角度（应在 PostGIS）？
- [ ] 使用 spec 中不存在的错误码？
- [ ] JWT 中写入可变字段？
- [ ] 遗漏 MUST 幂等接口的幂等键？
- [ ] 用 `#[allow(...)]` 压制编译警告？
- [ ] 绕过统一层或重复实现逻辑？

## 安全规则

文件删除、批量变更、Git 操作规范、数据库操作安全等全局规则见根目录 `AGENTS.md` 的「权限边界」和「Git 操作规范」章节，此处不重复。

> **⚠️ `tmp/` 路径规则（强制）**：所有临时文件、会话文件（`session-context-*.md`、`session-progress-*.md`）、设计文档等必须存放在**项目根目录（Monorepo 根目录）的 `tmp/`** 下，即 `backend/` 的上级目录中的 `tmp/`。**绝对禁止在 `backend/` 目录下创建 `tmp/` 目录**（如 `backend/tmp/`）。详见根目录 `AGENTS.md` 的「Agent 行为准则」章节。

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
