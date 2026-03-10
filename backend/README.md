# HotSpot Backend

[![Rust](https://img.shields.io/badge/Rust-1.75+-orange.svg)](https://www.rust-lang.org/)
[![Axum](https://img.shields.io/badge/Axum-0.8-blue.svg)](https://github.com/tokio-rs/axum)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-blue.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7.0+-red.svg)](https://redis.io/)

基于 Rust 的高性能、分布式社交应用后端服务。

## 目录

- [技术栈](#技术栈)
- [项目架构](#项目架构)
- [项目结构](#项目结构)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [开发指南](#开发指南)
- [数据库管理](#数据库管理)
- [API 文档](#api-文档)
- [测试](#测试)
- [部署](#部署)
- [监控与运维](#监控与运维)
- [故障排查](#故障排查)
- [性能优化](#性能优化)
- [贡献指南](#贡献指南)

---

## 技术栈

### 核心框架

- **[Axum 0.8](https://github.com/tokio-rs/axum)** - 高性能异步 Web 框架
- **[Tokio](https://tokio.rs/)** - 异步运行时
- **[Tower](https://github.com/tower-rs/tower)** - 中间件生态系统

### 数据存储

- **[PostgreSQL 16+](https://www.postgresql.org/)** - 主数据库（带 PostGIS 地理扩展）
- **[Redis 7.0+](https://redis.io/)** - 缓存、会话存储、分布式锁、Pub/Sub
- **[S3/MinIO](https://min.io/)** - 对象存储（图片、视频、文件）

### 认证与安全

- **JWT** - Access Token + Refresh Token 机制
- **bcrypt** - 密码哈希（work factor 10-12）
- **Redis Session** - 会话管理与 Token 黑名单

### 缓存策略

- **L1 缓存** - [Moka](https://github.com/moka-rs/moka) 进程内缓存
- **L2 缓存** - Redis 分布式缓存
- **Redis Pub/Sub** - 跨实例缓存失效通知

### 其他特性

- **WebSocket** - 实时消息推送
- **Rate Limiting** - 多层级限流（IP、用户、端点）
- **Bloom Filter** - 用户 Token 失效高效检测
- **PostGIS** - 地理位置查询与附近功能

---

## 项目架构

### 分层架构

```
┌─────────────────────────────────────────┐
│         HTTP/WebSocket Handlers         │  路由层：参数提取、响应封装
├─────────────────────────────────────────┤
│      Middleware (Auth, RateLimit)       │  中间件层：认证、限流、日志
├─────────────────────────────────────────┤
│           Service Layer                 │  业务逻辑层：事务、权限、缓存
├─────────────────────────────────────────┤
│         Repository Layer                │  数据访问层：SQL 封装
├─────────────────────────────────────────┤
│   Database (PostgreSQL) + Cache (Redis) │  存储层
└─────────────────────────────────────────┘
```

### 核心设计原则

1. **严格单向依赖**：Handler → Service → Repository
2. **业务逻辑集中**：所有业务逻辑在 Service 层
3. **数据验证前置**：Service 层验证所有引用完整性
4. **缓存透明化**：Service 层管理缓存，Repository 层无感知
5. **统一错误处理**：`ApiResponse<T>` 统一响应格式

### 分布式支持

- ✅ **多实例部署**：无状态设计，支持水平扩展
- ✅ **分布式缓存**：Redis L2 缓存 + Pub/Sub 失效通知
- ✅ **分布式锁**：带所有者验证的 Lua 脚本原子锁
- ✅ **连接池管理**：可配置的 PostgreSQL 连接池
- ✅ **限流降级**：Redis 故障时的可配置降级策略
- ⚠️ **WebSocket 限制**：当前为单实例推送（待优化）

---

## 项目结构

```
backend/
├── migrations/              # 数据库迁移文件（按模块分类）
│   ├── 001_prerequisites.sql
│   ├── 002_user_module.sql
│   ├── 003_private_chat_module.sql
│   └── ...
├── src/
│   ├── main.rs             # 应用入口
│   ├── config.rs           # 配置管理
│   ├── state.rs            # 应用状态
│   ├── error.rs            # 错误类型定义
│   ├── constants.rs        # 常量定义
│   │
│   ├── handlers/           # HTTP 处理器
│   │   ├── auth.rs        # 认证接口
│   │   ├── user.rs        # 用户管理
│   │   ├── moments.rs     # 瞬间/动态
│   │   └── ...
│   │
│   ├── services/           # 业务逻辑层
│   │   ├── auth_service.rs
│   │   ├── user_service.rs
│   │   ├── moment_service.rs
│   │   └── ...
│   │
│   ├── repositories/       # 数据访问层
│   │   ├── user_repo.rs
│   │   ├── moment_repo.rs
│   │   └── ...
│   │
│   ├── middleware/         # 中间件
│   │   ├── auth.rs        # JWT 认证
│   │   ├── rate_limit.rs  # 限流
│   │   └── admin_auth.rs  # 管理员认证
│   │
│   ├── routes/             # 路由定义
│   │   └── v1/
│   │       ├── mod.rs     # 路由聚合
│   │       ├── auth.rs
│   │       └── ...
│   │
│   ├── dto/                # 数据传输对象
│   │   ├── request.rs     # 请求 DTO
│   │   └── response.rs    # 响应 DTO
│   │
│   ├── types/              # 类型定义
│   │   └── enums.rs       # 枚举类型
│   │
│   ├── utils/              # 工具模块
│   │   ├── cache.rs       # 缓存管理
│   │   ├── cache_invalidation.rs  # 缓存失效
│   │   ├── session_token.rs       # Token 管理
│   │   ├── web_cookie.rs          # Cookie 工具
│   │   └── ...
│   │
│   ├── websocket/          # WebSocket 模块
│   │   ├── handler.rs     # WebSocket 处理器
│   │   └── types.rs       # WebSocket 类型
│   │
│   └── events/             # 事件系统
│       ├── event_bus.rs   # 事件总线
│       ├── event_handler.rs
│       └── event_types.rs
│
├── .env.example            # 配置模板（完整注释）
├── .env                    # 实际配置（不提交）
├── Cargo.toml              # 项目依赖
└── README.md               # 本文档
```

---

## 快速开始

### 环境要求

- **Rust**: 1.75+ (推荐使用 [rustup](https://rustup.rs/))
- **PostgreSQL**: 16+ (需启用 PostGIS 扩展)
- **Redis**: 7.0+
- **MinIO**: Latest (可选，开发环境)

### 安装依赖

#### macOS

```bash
# 安装 PostgreSQL + PostGIS
brew install postgresql@16 postgis

# 安装 Redis
brew install redis

# 安装 MinIO（可选）
brew install minio/stable/minio
```

#### Ubuntu/Debian

```bash
# 安装 PostgreSQL + PostGIS
sudo apt update
sudo apt install postgresql-16 postgresql-16-postgis-3

# 安装 Redis
sudo apt install redis-server

# 安装 MinIO（可选）
wget https://dl.min.io/server/minio/release/linux-amd64/minio
chmod +x minio
```

### 初始化项目

```bash
# 1. 克隆项目
git clone <repository-url>
cd hotspot/backend

# 2. 复制配置文件
cp .env.example .env

# 3. 编辑配置（必须修改的项）
vim .env
# 修改:
# - DATABASE_URL (数据库连接)
# - JWT_SECRET (生成: openssl rand -base64 32)
# - SUPER_ADMIN_PASSWORD (强密码)
# - S3_ACCESS_KEY / S3_SECRET_KEY (如果使用 MinIO)

# 4. 创建数据库
createdb hotspot_db

# 5. 启用 PostGIS 扩展
psql hotspot_db -c "CREATE EXTENSION IF NOT EXISTS postgis;"
psql hotspot_db -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"

# 6. 运行数据库迁移
psql hotspot_db < migrations/001_prerequisites.sql
psql hotspot_db < migrations/002_user_module.sql
psql hotspot_db < migrations/003_private_chat_module.sql
# ... 按顺序运行所有迁移文件

# 7. 启动 Redis
redis-server

# 8. 启动 MinIO（可选）
minio server /data --console-address ":9001"

# 9. 编译并运行
cargo run
```

### 验证服务

```bash
# 健康检查
curl http://localhost:5496/health

# 预期输出: {"status":"ok"}
```

---

## 配置说明

### 配置文件

- **`.env.example`** - 配置模板，包含所有可用配置项和详细注释
- **`.env`** - 实际使用的配置文件，**不要提交到代码仓库**

### 配置分类

#### 1. 数据库配置

```env
# PostgreSQL 连接字符串
DATABASE_URL=postgresql://user:password@localhost:5432/hotspot_db

# 连接池配置（分布式部署关键参数）
DB_POOL_MAX_CONNECTIONS=20          # 最大连接数
DB_POOL_MIN_CONNECTIONS=5           # 最小连接数
DB_POOL_ACQUIRE_TIMEOUT_SECS=10     # 获取连接超时（秒）
DB_POOL_IDLE_TIMEOUT_SECS=600       # 空闲连接超时（秒）
DB_POOL_MAX_LIFETIME_SECS=1800      # 连接最大生命周期（秒）
DB_POOL_TEST_BEFORE_ACQUIRE=true    # 获取前健康检查
```

**推荐配置（3 实例场景）**：
- PostgreSQL `max_connections=100`
- 每个实例 `DB_POOL_MAX_CONNECTIONS=20`
- 总计 60 连接，预留 40 给其他服务

#### 2. Redis 配置

```env
# Redis 连接字符串
REDIS_URL=redis://localhost:6379
```

**用途**：
- L2 缓存（用户信息、Token 验证等）
- Session 存储
- 分布式锁
- Redis Pub/Sub 缓存失效通知
- 限流计数器

#### Redis node_id 租约（可选，分布式推荐）

```env
# 启用后，实例启动时从 Redis 自动抢占 node_id（0..=1023）
ID_NODE_ID_ALLOCATE_WITH_REDIS=true

# 租约键前缀（最终 key 形如 hotspot:id_node:lease:17）
ID_NODE_ID_REDIS_KEY_PREFIX=hotspot:id_node:lease

# 租约 TTL 与续租间隔（秒）
ID_NODE_ID_REDIS_LEASE_TTL_SECS=90
ID_NODE_ID_REDIS_RENEW_INTERVAL_SECS=30

# 租约丢失后是否强制退出（推荐 true，避免 ID 冲突）
ID_NODE_ID_REDIS_FATAL_ON_LOSS=true
```

#### 3. JWT 配置

```env
# JWT 签名密钥（必须修改）
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production

# Access Token 过期时间（秒）
JWT_ACCESS_TOKEN_EXPIRATION=1800      # 30 分钟

# Refresh Token 过期时间（秒）
JWT_REFRESH_TOKEN_EXPIRATION=1209600  # 14 天
```

**生成强密钥**：
```bash
openssl rand -base64 32
```

#### 4. 服务器配置

```env
# 监听地址（0.0.0.0 = 所有接口）
SERVER_HOST=0.0.0.0

# 监听端口
SERVER_PORT=5496
```

#### 5. Rate Limit 配置

```env
# 全局限流
RATE_LIMIT_GLOBAL_PER_HOUR=20000          # 全局每小时（约 5.5 次/秒）
RATE_LIMIT_USER_PER_5MINUTES=500          # 单用户每 5 分钟（约 1.67 次/秒）
RATE_LIMIT_USER_PER_HOUR=5000             # 单用户每小时（约 1.39 次/秒）

# Redis 故障降级策略（新增）
RATE_LIMIT_FAIL_OPEN=true                 # true=允许通过, false=拒绝请求

# 端点级限流（认证相关）
RATE_LIMIT_REGISTER_PER_10MINUTES=10      # 用户注册
RATE_LIMIT_LOGIN_PER_10MINUTES=30         # 用户登录
RATE_LIMIT_AUTH_RESET_PER_10MINUTES=10    # 请求重置密码
RATE_LIMIT_GUEST_PER_10MINUTES=60         # 游客登录
RATE_LIMIT_REFRESH_PER_10MINUTES=120      # Token 刷新

# 端点级限流（业务相关）
RATE_LIMIT_MOMENTS_CREATE_PER_MINUTE=20   # 创建瞬间
RATE_LIMIT_COMMENTS_CREATE_PER_MINUTE=60  # 创建评论
RATE_LIMIT_REPORTS_CREATE_PER_HOUR=10     # 创建举报
```

**限流策略**：
- **IP 限制 > 用户限制**：考虑共享 IP 场景（公司/学校/家庭网络）
- **fail-open（推荐）**：Redis 故障时允许通过，优先保证可用性
- **fail-closed**：Redis 故障时拒绝请求，优先保证安全性

#### 6. S3/MinIO 配置

```env
# S3 服务端点
S3_ENDPOINT=http://localhost:9000

# S3 区域
S3_REGION=us-east-1

# S3 存储桶
S3_BUCKET=hotspot

# S3 访问密钥（必须修改）
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin

# 强制路径风格（MinIO=true，AWS=false）
S3_FORCE_PATH_STYLE=true

# 公开访问 URL 前缀
S3_PUBLIC_BASE=/s3  # 本地: /s3，生产: https://cdn.example.com
```

#### 密码重置验证码投递（可选）

```env
# 配置后会将验证码通过 webhook 投递到短信/邮件网关
AUTH_RESET_DELIVERY_WEBHOOK_URL=

# 可选：Bearer Token
AUTH_RESET_DELIVERY_WEBHOOK_TOKEN=

# 超时（毫秒）
AUTH_RESET_DELIVERY_TIMEOUT_MS=3000
```

#### 超级管理员配置

```env
# 超级管理员登录 ID
SUPER_ADMIN_LOGIN_ID=superadmin

# 超级管理员初始密码（必须修改）
SUPER_ADMIN_PASSWORD=change-this-strong-password
```

**安全建议**：
- 首次部署后立即修改默认密码
- 使用强密码（16+ 字符，包含大小写、数字、特殊字符）
- 定期更换管理员密码

#### 8. CORS 配置

```env
# 允许的前端域名（逗号分隔）
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

### 完整配置清单

完整配置项以 `.env.example` 为准，所有配置项都有中文注释与默认建议值。

---

## 开发指南

### 编译与运行

```bash
# 开发模式（带调试信息）
cargo run

# 开发模式（带详细日志）
RUST_LOG=debug cargo run

# Release 模式（生产优化）
cargo build --release
./target/release/hotspot-backend
```

### 代码风格

```bash
# 格式化代码
cargo fmt

# 检查代码风格
cargo fmt -- --check

# 运行 Clippy 检查
cargo clippy

# 自动修复 Clippy 警告
cargo clippy --fix
```

### 类型检查

```bash
# 快速类型检查（不生成可执行文件）
cargo check
```

### 开发最佳实践

#### 1. 分层架构规则

```rust
// ✅ 正确：Handler 只提取参数，调用 Service
pub async fn create_moment(
    State(state): State<AppState>,
    AuthenticatedUser { user_id, .. }: AuthenticatedUser,
    Json(req): Json<CreateMomentRequest>,
) -> Result<Json<ApiResponse<Moment>>, ApiError> {
    let moment = MomentService::create(&state.db, user_id, req).await?;
    Ok(Json(ApiResponse::success(moment)))
}

// ❌ 错误：Handler 中包含业务逻辑
pub async fn create_moment(...) -> Result<...> {
    // ❌ 不要在 Handler 中直接操作数据库
    let moment = sqlx::query!(...).fetch_one(&state.db).await?;
    // ❌ 不要在 Handler 中进行复杂验证
    if moment.content.len() > 1000 { ... }
}
```

#### 2. Service 层规则

```rust
// ✅ 正确：Service 验证引用完整性
pub async fn create_moment(
    db: &PgPool,
    user_id: i64,
    req: CreateMomentRequest,
) -> Result<Moment> {
    // 验证用户是否存在
    if !UserRepository::exists(db, user_id).await? {
        return Err(ApiError::UserNotFound);
    }
    
    // 验证提及的用户是否存在
    if let Some(mentions) = &req.mentions {
        if !UserRepository::exists_all(db, mentions).await? {
            return Err(ApiError::InvalidMentions);
        }
    }
    
    // 通过 Repository 操作数据库
    MomentRepository::create(db, user_id, &req).await
}

// ❌ 错误：直接写 SQL
pub async fn create_moment(...) -> Result<Moment> {
    // ❌ 不要在 Service 中直接写 SQL
    let moment = sqlx::query!("INSERT INTO ...").execute(db).await?;
}
```

#### 3. Repository 层规则

```rust
// ✅ 正确：Repository 只封装 SQL
pub async fn create(
    db: &PgPool,
    user_id: i64,
    req: &CreateMomentRequest,
) -> Result<Moment> {
    let moment_id = Uuid::new_v4();
    
    sqlx::query!(
        r#"
        INSERT INTO moments (moment_id, user_id, content, images)
        VALUES ($1, $2, $3, $4)
        RETURNING *
        "#,
        moment_id,
        user_id,
        req.content,
        &req.images,
    )
    .fetch_one(db)
    .await
    .map_err(Into::into)
}

// ❌ 错误：Repository 包含业务逻辑
pub async fn create(...) -> Result<Moment> {
    // ❌ 不要在 Repository 中进行业务验证
    if req.content.len() > 1000 {
        return Err(ApiError::ContentTooLong);
    }
    // ...
}
```

#### 4. 错误处理

```rust
// ✅ 正确：使用 ? 操作符传播错误
pub async fn get_user(db: &PgPool, user_id: i64) -> Result<User> {
    UserRepository::find_by_id(db, user_id)
        .await?
        .ok_or(ApiError::UserNotFound)
}

// ❌ 错误：unwrap() 可能导致 panic
pub async fn get_user(db: &PgPool, user_id: i64) -> Result<User> {
    let user = UserRepository::find_by_id(db, user_id).await.unwrap();  // ❌
    Ok(user)
}
```

#### 5. API 响应格式

```rust
// ✅ 正确：使用统一的 ApiResponse
Ok(Json(ApiResponse::success(data)))
Ok(Json(ApiResponse::success_empty()))

// ❌ 错误：直接返回数据
Ok(Json(data))  // ❌
```

---

## 数据库管理

### 迁移文件

迁移文件位于 `migrations/` 目录，按模块分类：

```
migrations/
├── 001_prerequisites.sql      # 前置条件（扩展、函数）
├── 002_user_module.sql        # 用户模块
├── 003_private_chat_module.sql # 私聊模块
├── 004_group_chat_module.sql  # 群聊模块
├── 005_moment_module.sql      # 瞬间模块
├── 006_comment_module.sql     # 评论模块
├── 007_notification_module.sql # 通知模块
├── 008_feed_module.sql        # 信息流模块
├── 009_admin_rbac_module.sql  # 管理员 RBAC
├── 010_report_module.sql      # 举报模块
├── 011_system_config_module.sql # 系统配置
├── 012_recommendation_module.sql # 推荐模块
└── 013_share_module.sql       # 分享模块
```

### 运行迁移

```bash
# 按顺序运行所有迁移
for file in migrations/*.sql; do
    echo "Running $file..."
    psql $DATABASE_URL -f "$file"
done
```

### 回滚迁移

**警告**：暂不支持自动回滚，需要手动编写回滚 SQL。

### 数据库备份

```bash
# 备份数据库
pg_dump -h localhost -U hotspot hotspot_db > backup_$(date +%Y%m%d_%H%M%S).sql

# 恢复数据库
psql -h localhost -U hotspot hotspot_db < backup_20260201_183000.sql
```

---

## API 文档

### 文档位置

API 规范文档位于项目根目录的 `spec/` 文件夹：

```
spec/
├── api_spec.md                  # 主规范（总览）
├── api_spec_auth.md            # 认证接口
├── api_spec_user.md            # 用户管理
├── api_spec_moment.md          # 瞬间/动态
├── api_spec_chat.md            # 私聊
├── api_spec_group.md           # 群聊
├── api_spec_feed.md            # 信息流
├── api_spec_notification.md   # 通知
├── api_spec_search.md          # 搜索
├── api_spec_share.md           # 分享
├── api_spec_report.md          # 举报
├── api_spec_system.md          # 系统配置
├── api_spec_admin_*.md         # 管理员接口
├── api_spec_websocket.md       # WebSocket 事件
├── api_spec_errors.md          # 错误码定义
└── api_spec_common_types.md    # 通用类型
```

### API 规范

- **协议**：HTTP/1.1 + WebSocket
- **方法**：所有业务接口使用 POST（健康检查除外）
- **前缀**：`/api/v1/`
- **响应格式**：统一 JSON-RPC 风格

```typescript
interface ApiResponse<T> {
  code: number;        // 0 = 成功，非 0 = 错误码
  message: string;     // 错误描述
  data?: T;            // 响应数据
  request_id: string;  // 请求追踪 ID
}
```

### 接口示例

```bash
# 用户登录
curl -X POST http://localhost:5496/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "login_id": "user@example.com",
    "password": "password123",
    "login_type": "email"
  }'

# 创建瞬间（需要 JWT）
curl -X POST http://localhost:5496/api/v1/moments/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  -d '{
    "content": "Hello World!",
    "visibility": "public"
  }'
```

---

## 测试

### 单元测试

```bash
# 运行所有测试
cargo test

# 使用独立测试数据库和独立测试 Redis
DATABASE_URL=postgresql://hotspot:hotspot@127.0.0.1:5432/hotspot_test \
REDIS_URL=redis://127.0.0.1:6380/0 \
TEST_REDIS_URL=redis://127.0.0.1:6380/0 \
cargo test

# 仅连接指定测试 Redis 运行 Redis 相关测试
TEST_REDIS_URL=redis://127.0.0.1:6380/0 cargo test id_gen::tests -- --nocapture

# 运行特定模块测试
cargo test --test user_tests

# 运行特定测试函数
cargo test test_create_user

# 带输出的测试
cargo test -- --nocapture
```

### 集成测试

```bash
# 运行集成测试
cargo test --test '*' -- --test-threads=1
```

### 压力测试

```bash
# 使用 wrk 进行压力测试
wrk -t10 -c100 -d30s http://localhost:5496/health

# 使用 ab 进行压力测试
ab -n 10000 -c 100 http://localhost:5496/health
```

---

## 部署

### Docker 镜像

> 推荐在项目根目录下分别为后端和前端创建 Dockerfile，并使用多阶段构建减少镜像体积。

#### backend/Dockerfile

```dockerfile
FROM rust:1.75 AS builder
WORKDIR /app
COPY backend/ ./
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/app/target \
    cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends libpq5 ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/target/release/hotspot-backend /usr/local/bin/
ENV RUST_LOG=info,hotspot_backend=info
WORKDIR /app
EXPOSE 5496
CMD ["hotspot-backend"]
```

#### frontend_svelte/Dockerfile

```dockerfile
FROM node:20-alpine AS deps
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@latest --activate
COPY frontend_svelte/pnpm-lock.yaml frontend_svelte/package.json ./
RUN pnpm install --frozen-lockfile

FROM node:20-alpine AS builder
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@latest --activate
COPY --from=deps /app/node_modules ./node_modules
COPY frontend_svelte/ ./
ENV NODE_ENV=production
RUN pnpm build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/build ./build
EXPOSE 4173
CMD ["node", "build/index.js"]
```

> 如需在前端容器中设置 API 地址，请在运行前导出 `PUBLIC_API_BASE`（或通过反向代理与后端同域）。AMap 密钥使用 `VITE_AMAP_KEY`、`VITE_AMAP_SERVICE_HOST` 注入。

### docker-compose 部署栈

在项目根目录新建 `docker/docker-compose.prod.yml`：

```yaml
version: '3.9'

services:
  postgres:
    image: postgis/postgis:18-3.4
    container_name: hotspot-postgres
    environment:
      POSTGRES_DB: hotspot_db
      POSTGRES_USER: hotspot
      POSTGRES_PASSWORD: change-me
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U $$POSTGRES_USER -d $$POSTGRES_DB']
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  redis:
    image: redis:7.2-alpine
    container_name: hotspot-redis
    command: ["redis-server", "--save", "", "--appendonly", "yes"]
    volumes:
      - redis_data:/data
    restart: unless-stopped

  backend:
    build:
      context: ..
      dockerfile: backend/Dockerfile
    container_name: hotspot-backend
    env_file:
      - ../backend/.env.production
    environment:
      DATABASE_URL: postgresql://hotspot:change-me@postgres:5432/hotspot_db
      REDIS_URL: redis://redis:6379
      S3_ENDPOINT: http://minio:9000
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_started
    ports:
      - "5496:5496"
    restart: unless-stopped

  frontend:
    build:
      context: ..
      dockerfile: frontend_svelte/Dockerfile
    container_name: hotspot-frontend
    environment:
      PUBLIC_API_BASE: http://backend:5496
      VITE_AMAP_KEY: ""
      VITE_AMAP_SERVICE_HOST: ""
    ports:
      - "4173:4173"
    depends_on:
      - backend
    restart: unless-stopped

  minio:
    image: minio/minio:RELEASE.2024-06-13T22-48-17Z
    container_name: hotspot-minio
    command: server /data --console-address :9001
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
  minio_data:
```

### Nginx 反向代理（可选）

为了在同一个域名下暴露前端与后端，并满足 HTTPS/缓存需求，可以再新增一个 Nginx 容器：

1. 新建 `docker/nginx/default.conf`：

    ```nginx
    server {
        listen 80;
        server_name _;
        client_max_body_size 25m;

        location /api/ {
            proxy_pass http://backend:5496/api/;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        location / {
            proxy_pass http://frontend:4173/;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
    ```

2. 在 `docker/docker-compose.prod.yml` 中追加服务：

    ```yaml
      nginx:
        image: nginx:1.25-alpine
        container_name: hotspot-nginx
        ports:
          - "80:80"
        volumes:
          - ./nginx/default.conf:/etc/nginx/conf.d/default.conf:ro
        depends_on:
          - frontend
          - backend
        restart: unless-stopped
    ```

3. 若需 HTTPS，可在该容器内挂载证书并监听 443，或改用 Caddy/Traefik 获取自动证书。

> 使用反向代理后，请确保 DNS 指向 Nginx 所在实例，并透传 `X-Forwarded-Proto` 等头部，后端才能生成正确的绝对 URL。

### 部署步骤

1. **准备环境变量**
   - 后端：复制 `backend/.env.example` 为 `.env.production`，将 `DATABASE_URL`/`REDIS_URL` 等指向容器主机名 (`postgres`、`redis`)。
   - 前端：新建 `frontend_svelte/.env.production`，写入 `PUBLIC_API_BASE=https://api.example.com`、`VITE_AMAP_KEY=xxx` 等公开变量。
2. **构建镜像**
   ```bash
   docker compose -f docker/docker-compose.prod.yml build
   ```
3. **启动基础设施**
   ```bash
   docker compose -f docker/docker-compose.prod.yml up -d postgres redis minio
   ```
4. **运行数据库迁移**：进入 backend 容器 `docker compose exec backend bash`，再运行 `sqlx migrate run`（遵循“由人工触发”原则）。
5. **启动应用**
   ```bash
   docker compose -f docker/docker-compose.prod.yml up -d backend frontend nginx
   ```
6. **验证**：通过 `http://localhost`（Nginx 代理）访问前端，通过 `http://localhost/api/v1/system/health` 或直接 `http://localhost:5496/health` 校验后端，并查看日志 `docker compose logs -f backend`。

> 生产环境通常会在前端容器前增加 Nginx/Caddy，将 `/api` 转发到 backend，其他路径交给前端容器或 CDN；记得透传 `X-Forwarded-Proto` 等头部。

### 生产环境部署清单

#### 1. 环境配置

- [ ] 修改 `JWT_SECRET` 为强随机字符串
- [ ] 修改 `SUPER_ADMIN_PASSWORD` 为强密码
- [ ] 修改 S3 密钥为强密码
- [ ] 配置正确的 `CORS_ALLOWED_ORIGINS`
- [ ] 配置正确的 `S3_PUBLIC_BASE`（CDN 地址）
- [ ] 设置 `RUST_LOG=info`

#### 2. 数据库

- [ ] 运行所有数据库迁移
- [ ] 配置数据库连接池（参考 3 实例场景）
- [ ] 启用数据库连接健康检查
- [ ] 配置自动备份

#### 3. Redis

- [ ] 配置 Redis 持久化
- [ ] 启用 Redis 认证
- [ ] 配置 Redis 最大内存策略

#### 4. 监控

- [ ] 配置日志聚合（ELK/Loki）
- [ ] 配置指标收集（Prometheus）
- [ ] 配置报警（Alertmanager）
- [ ] 配置链路追踪（Jaeger/Zipkin）

#### 5. 安全

- [ ] 启用 HTTPS（Nginx/Caddy 反向代理）
- [ ] 配置防火墙规则
- [ ] 启用 DDoS 防护
- [ ] 定期更新依赖

---

## 监控与运维

### 内置监控指标

服务启动后，自动每 60 秒输出关键指标：

```log
[INFO] cache_hit_rate=87.32% cache_hits=12453 cache_misses=1801 cache_errors=3 缓存性能指标
[INFO] pool_size=18 pool_idle=7 数据库连接池状态
```

### 关键指标

| 指标 | 说明 | 健康阈值 |
|------|------|----------|
| `cache_hit_rate` | 缓存命中率 | > 70% |
| `cache_hits` | 缓存命中次数 | - |
| `cache_misses` | 缓存未命中次数 | - |
| `cache_errors` | 缓存错误次数 | < 10/分钟 |
| `pool_size` | 连接池当前连接数 | < max_connections |
| `pool_idle` | 连接池空闲连接数 | > 0 |

### 报警建议

1. **连接池耗尽**：`pool_idle == 0` 且 `pool_size == max_connections`
2. **缓存命中率低**：`cache_hit_rate < 70%`
3. **缓存错误频繁**：`cache_errors > 10/分钟`
4. **Redis 连接断开**：日志中出现 `Redis Pub/Sub 连接失败`

### 日志级别

```env
# 开发环境
RUST_LOG=debug,hotspot_backend=trace

# 生产环境（推荐）
RUST_LOG=info,hotspot_backend=info

# 排查问题
RUST_LOG=debug,sqlx=debug,tower_http=debug
```

### 响应日志（Response Logger）

- 已在 `main.rs` 全局启用 `response_logger` 中间件，所有 JSON 响应都会被 tracing 记录，方便 App 端排查问题。
- 日志中自动携带 `request_id`，可与前端/移动端上报的 `X-Request-Id` 对应。
- 仅在 `Content-Type` 含 `json` 且状态码不为 204/304 时记录，避免泄露二进制或空响应。
- 默认最多输出前 8KB (`LOG_PREVIEW_BYTES=8 * 1024`)，其余部分以 `truncated=true` 标记，可按需调整该常量。
- 示例：
  ```log
  [INFO] method=POST uri=/api/v1/auth/login status=200 truncated=false response_preview={"code":0,...} response body
  ```
- 如需关闭，可在自定义构建中移除 `response_logger` 中间件或通过日志等级控制 (`RUST_LOG`) 过滤。

---

## 故障排查

### 1. 服务启动失败

#### 症状：`connection refused` 或 `authentication failed`

**原因**：数据库连接失败

**解决**：
1. 检查 `DATABASE_URL` 是否正确
2. 确认 PostgreSQL 服务已启动：`pg_isready`
3. 验证用户名、密码、数据库名
4. 检查防火墙规则

#### 症状：`missing field: XXX`

**原因**：配置文件缺失字段

**解决**：
1. 对比 `.env.example` 补充缺失配置
2. 确保没有语法错误（引号、等号等）

### 2. Redis 连接问题

#### 症状：`connection refused` 或 `timeout`

**原因**：Redis 连接失败

**解决**：
1. 检查 `REDIS_URL` 是否正确
2. 确认 Redis 服务已启动：`redis-cli ping`
3. 临时降级：设置 `RATE_LIMIT_FAIL_OPEN=true`

### 3. 限流误触发

#### 症状：正常用户被限流

**原因**：限流阈值过低或多用户共享 IP

**解决**：
1. 检查 `RATE_LIMIT_*` 配置是否过低
2. 调整 IP 限流阈值（考虑共享 IP 场景）
3. 通过管理后台临时调整限流值

### 4. 缓存命中率低

#### 症状：`cache_hit_rate < 70%`

**原因**：缓存策略不合理或 Redis 频繁重启

**解决**：
1. 检查 Redis 是否频繁重启
2. 调整缓存 TTL
3. 优化缓存预热逻辑
4. 检查缓存失效事件是否过于频繁

### 5. 数据库连接池耗尽

#### 症状：`timed out acquiring connection` 或 `pool_idle == 0`

**原因**：连接池配置过小或存在连接泄漏

**解决**：
1. 增加 `DB_POOL_MAX_CONNECTIONS`
2. 检查是否有长时间占用连接的查询
3. 检查是否正确释放连接

---

## 性能优化

### 数据库优化

#### 1. 连接池配置

根据实例数量和数据库最大连接数合理配置：

```env
# 假设 PostgreSQL max_connections=100，部署 3 个实例
# 每个实例最大连接数 = (100 - 预留20) / 3 ≈ 26，保守设置为 20
DB_POOL_MAX_CONNECTIONS=20
DB_POOL_MIN_CONNECTIONS=5
```

#### 2. 查询优化

- 使用 `EXPLAIN ANALYZE` 分析慢查询
- 为常用查询字段添加索引
- 避免 `SELECT *`，只查询需要的字段
- 使用游标分页（Keyset Pagination），避免 OFFSET

#### 3. PostGIS 优化

- 所有地理查询使用 PostGIS 函数（`ST_Distance`、`ST_DWithin`）
- 为地理列添加 GIST 索引
- 禁止在应用层进行距离计算

### 缓存优化

#### 1. L1 + L2 缓存

- **L1（Moka）**：高频访问数据（Token 验证、用户基本信息）
- **L2（Redis）**：中频访问数据（用户详情、动态列表）

#### 2. 缓存 TTL

```rust
// 高频数据 - 短 TTL（5 分钟）
cache.set("token:verify", data, 300).await?;

// 中频数据 - 中 TTL（30 分钟）
cache.set("user:profile", data, 1800).await?;

// 低频数据 - 长 TTL（1 小时）
cache.set("group:info", data, 3600).await?;
```

#### 3. 缓存失效

使用 Redis Pub/Sub 实现跨实例缓存失效：

```rust
// 更新数据后发布失效事件
cache_invalidator.publish(InvalidationEvent::UserProfileUpdated {
    user_id: user_id.to_string(),
}).await?;
```

### 限流优化

#### 1. 使用 Lua 脚本保证原子性

```rust
// Lua 脚本合并 INCR 和 EXPIRE 操作
let script = r#"
    local count = redis.call('INCR', KEYS[1])
    if count == 1 then
        redis.call('EXPIRE', KEYS[1], ARGV[1])
    end
    return count
"#;
```

#### 2. 限流降级策略

```env
# Redis 故障时允许请求通过（推荐）
RATE_LIMIT_FAIL_OPEN=true

# Redis 故障时拒绝请求（高安全场景）
RATE_LIMIT_FAIL_OPEN=false
```

### 分布式优化

#### 1. 无状态设计

- 所有状态存储在 Redis/PostgreSQL
- 支持水平扩展
- 负载均衡器随机分配请求

#### 2. 分布式锁

使用带所有者验证的 Lua 脚本锁：

```rust
let owner_id = Uuid::new_v4().to_string();
if cache_manager.try_lock_with_owner("critical_section", &owner_id).await? {
    // 执行临界区代码
    cache_manager.unlock_with_owner("critical_section", &owner_id).await?;
}
```

#### 3. Redis Pub/Sub 重连

自动重连机制，指数退避：

```rust
// 重连间隔: 2^n 秒，最大 60 秒
let backoff_secs = std::cmp::min(2_u64.pow(retry_count), 60);
```

---

## 贡献指南

### 提交代码

1. Fork 本项目
2. 创建特性分支：`git checkout -b feature/xxx`
3. 提交代码：`git commit -m "feat: add xxx"`
4. 推送分支：`git push origin feature/xxx`
5. 提交 Pull Request

### 代码规范

- 遵循 Rust 官方代码风格
- 运行 `cargo fmt` 格式化代码
- 运行 `cargo clippy` 检查警告
- 为新功能编写测试
- 更新相关文档

### Commit 规范

```
feat: 新功能
fix: 修复 Bug
docs: 文档更新
style: 代码格式（不影响功能）
refactor: 重构
perf: 性能优化
test: 测试相关
chore: 构建/工具相关
```

---

## 许可证

本项目采用 MIT 许可证，详见 [LICENSE](../LICENSE) 文件。

---

## 联系方式

- **项目主页**: [GitHub](https://github.com/your-org/hotspot)
- **问题反馈**: [Issues](https://github.com/your-org/hotspot/issues)
- **邮箱**: support@example.com

---

## 更新日志

### v0.0.1 (2026-02-01)

#### ✨ 新特性

- ✅ 完整的用户认证系统（JWT + Refresh Token）
- ✅ 多层级限流（IP、用户、端点）
- ✅ L1+L2 缓存架构
- ✅ 地理位置查询（PostGIS）
- ✅ 实时消息推送（WebSocket）
- ✅ 对象存储（S3/MinIO）
- ✅ 管理员 RBAC 权限系统

#### 🚀 性能优化

- ✅ 数据库连接池配置（支持分布式部署）
- ✅ 分布式锁安全性（Lua 脚本 + 所有者验证）
- ✅ Redis Pub/Sub 重连机制（指数退避）
- ✅ 限流原子性（Lua 脚本合并 INCR+EXPIRE）
- ✅ 缓存失效事件扩展（7 种事件类型）
- ✅ 监控指标系统（缓存命中率、连接池状态）
- ✅ 限流降级策略（fail-open/fail-closed）

#### 📝 文档

- ✅ 完整的配置说明（42 个配置项，全部带注释）
- ✅ API 规范文档（17 个模块）
- ✅ 开发指南与最佳实践
- ✅ 部署指南与故障排查

---

**感谢使用 HotSpot Backend！** 🎉
