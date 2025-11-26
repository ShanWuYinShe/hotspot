# HotSpot 数据库迁移文件

## 📁 迁移文件结构（7个文件）

| 文件 | 内容 | 说明 |
|------|------|------|
| `001_prerequisites.sql` | 扩展、枚举、函数 | PostgreSQL扩展、所有枚举类型、通用触发器函数 |
| `002_user_module.sql` | 用户模块 | 用户表、设置、关系（关注/拉黑/好友）、位置历史、在线状态、会话、密码重置 |
| `003_chat_module.sql` | 聊天模块 | 私聊、群聊、成员管理、未读计数 |
| `004_message_module.sql` | 消息模块 | 私聊消息、群聊消息、已读状态、消息反应 |
| `005_moment_comment_module.sql` | 内容模块 | 生活瞬间、评论、点赞、收藏 |
| `006_notification_module.sql` | 通知模块 | 收件箱、订阅设置、设备令牌 |
| `007_system_module.sql` | 系统模块 | 举报、信息流、排行榜、系统配置 |

## 🎯 设计原则

### 1. 模块化组织
- 按功能用途划分文件，每个文件包含相关的所有表
- **索引与表定义在同一文件**：每个表的索引都紧跟在该表的 CREATE TABLE 语句后面
- **触发器紧跟表定义**：触发器定义在索引之后
- 减少文件数量，便于管理和理解
- 保持依赖关系清晰

### 2. ID设计模式
- **自增主键**: 所有表都有 `id BIGSERIAL PRIMARY KEY`（仅数据库内部使用）
- **业务ID**: 
  - `user_id BIGINT`：用户业务ID（纯数字，对外展示）
  - 其他ID：UUID字符串（`moment_id`, `message_id`, `group_chat_id` 等，既是业务ID也用作cursor）
- **分页游标**: 
  - `cursor_id VARCHAR(36)`：users表的UUID字段，仅用于分页查询，不用于任何业务逻辑

### 3. 数据完整性
- **不使用数据库级外键**: 仅在应用层维护关系
- **软删除**: 使用 `is_deleted` + `deleted_at`
- **时区**: 统一使用 `TIMESTAMPTZ` (UTC)

### 4. 地理位置
- 使用 PostGIS 的 `geography(POINT, 4326)` 类型
- 保留 `latitude` 和 `longitude` 字段便于查询
- 通过触发器自动同步 `location` 字段
- **所有地理位置字段必须使用 GIST 索引**

### 5. 高并发优化
- **表分离**: `chat_unread_counts` 独立表避免锁竞争
- **在线状态分离**: `user_online_status` 独立表避免频繁更新 users 表
- **索引优化**: 针对高频查询创建复合索引
- **JSONB**: 灵活存储动态数据（如通知data字段）

## 🚀 迁移执行

### 使用 sqlx-cli

```bash
# 安装 sqlx-cli
cargo install sqlx-cli --no-default-features --features postgres

# 创建数据库
sqlx database create

# 执行所有迁移
sqlx migrate run

# 查看迁移状态
sqlx migrate info

# 回滚最后一次迁移
sqlx migrate revert
```

### 环境变量

```bash
# .env 文件
DATABASE_URL=postgresql://username:password@localhost:5432/hotspot_db
```

## 📊 表关系概览

```
users (核心)
├── user_settings (1:1)
├── user_online_status (1:1) ✨ 高并发优化
├── user_follows (N:N 自关联)
├── user_blocks (N:N 自关联)
├── user_friend_requests (N:N)
├── user_friend_remarks (N:N)
├── user_location_history (1:N)
├── sessions (1:N)
├── password_reset_tokens (1:N)
├── private_chats (N:N)
│   ├── private_chat_settings (1:N)
│   ├── private_messages (1:N)
│   └── chat_unread_counts (1:N) ✨ 高并发优化
├── group_chats (1:N)
│   ├── group_settings (1:1)
│   ├── group_members (N:N)
│   ├── group_member_settings (N:N)
│   ├── group_invites (N:N)
│   ├── group_applications (N:N)
│   ├── group_messages (1:N)
│   └── chat_unread_counts (1:N) ✨ 高并发优化
├── moments (1:N)
│   ├── moment_likes (N:N)
│   ├── moment_favorites (N:N)
│   └── comments (1:N)
│       └── comment_likes (N:N)
├── inbox_items (1:N)
├── notification_subscriptions (1:1)
├── device_tokens (1:N)
└── reports (1:N)
```

## 🔑 关键表说明

### 高并发优化表

#### user_online_status - 用户在线状态表
- **目的**: 从 users 表分离在线状态，避免高频更新锁住用户基础信息
- **字段**: is_online, last_active_at, last_seen_at, connection_count
- **使用场景**: WebSocket 连接/断开、用户活跃状态更新

#### chat_unread_counts - 聊天未读计数表
- **目的**: 独立管理未读计数，避免锁竞争（群聊中可能涉及数百人）
- **字段**: chat_type, chat_id, user_id, unread_count, last_read_at, last_read_message_id
- **使用场景**: 消息发送、标记已读、查询未读数

### 核心业务表

#### users - 用户表
- 包含用户基础信息和当前位置（快照）
- **不包含**: last_seen（迁移到 user_online_status）
- **位置字段**: current_latitude, current_longitude, location, location_name

#### user_settings - 用户设置表
- 从 users 表分离的配置项
- **新增**: allow_private_chat（是否允许别人发起私聊）

#### private_chats / group_chats - 聊天表
- 管理聊天室元数据
- **不包含**: 消息内容（在 private_messages / group_messages）
- **不包含**: 未读计数（在 chat_unread_counts）

## 📝 索引策略

### 必须遵守的规则
1. **地理位置字段**: 必须使用 **GIST 索引**（`USING GIST`）
2. **数组字段**（tags, mentions）: 使用 **GIN 索引**（`USING GIN`）
3. **全文搜索字段**（nickname, content）: 使用 **GIN 索引** + `pg_trgm` 扩展
4. **软删除字段**: 使用 **部分索引**（`WHERE is_deleted = FALSE`）
5. **时间排序字段**: 使用 **降序索引**（`DESC`）

### 索引位置
- 所有索引都紧跟在对应表的 CREATE TABLE 语句后面
- 不再有单独的"优化索引"章节

## ⚠️ 注意事项

1. **迁移顺序**: 必须严格按照文件编号顺序执行
2. **依赖关系**: 后续迁移依赖前面的表和类型定义
3. **备份策略**: 执行迁移前务必备份数据库
4. **性能监控**: 定期检查索引使用情况，优化慢查询
5. **扩展依赖**: 第一个迁移文件必须创建所有必需的 PostgreSQL 扩展

## 🆕 版本历史

### v1.2.0 (2025-01-15): 模块化重构
- 按用途重新组织迁移文件（用户、聊天、消息、内容、通知、系统）
- 新增高并发优化表：user_online_status, chat_unread_counts
- 索引紧跟表定义，移除独立的优化索引章节
- 消息模块独立（从聊天模块分离）
- 修正系统配置默认坐标为南天门（35.154922, 112.682524）
- user_settings 表新增 allow_private_chat 字段

### v1.1.0 (2025-01-XX): 优化迁移文件组织
- 将索引定义合并到各模块文件中
- 在建表时直接定义软删除字段
- 精简为6个模块化迁移文件

### v1.0.0 (2025-01-XX): 初始版本
- 基础数据库结构
- 核心业务表定义
