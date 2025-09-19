# HotSpot 数据库迁移

## 🚀 快速开始

### 全新部署（推荐）

```bash
# 1. 创建数据库
psql -h localhost -U postgres -c "CREATE DATABASE hotspot;"

# 2. 执行完整架构迁移
psql -h localhost -U postgres -d hotspot -f migrations/01_complete_schema.sql

# 3. 设置应用权限
psql -h localhost -U postgres -d hotspot -c "
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO hotspot_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO hotspot_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO hotspot_app;
"
```

### 开发环境重置

```bash
# 快速重置开发数据库
cd backend/migrations
./reset_dev_db.sh
```

## 📋 迁移文件

### 当前架构

| 文件 | 说明 | 状态 |
|------|------|------|
| **`01_complete_schema.sql`** | **HotSpot 完整数据库架构** | ✅ **推荐使用** |

这是唯一需要的迁移文件，包含：

- ✅ **完全按照需求文档设计**：严格遵循 `backend_doc.mdc` 规范
- ✅ **统一消息架构**：群聊和私聊使用相同表结构
- ✅ **用户ID编码规范**：64位整数，高8位编码用户类型
- ✅ **软删除机制**：完整的触发器和恢复功能
- ✅ **高性能设计**：分区表、覆盖索引、PostGIS支持
- ✅ **16个核心表**：涵盖所有业务功能模块

### 归档文件

| 目录 | 说明 |
|------|------|
| `archived/legacy_migrations/` | 旧版迁移文件（已废弃，仅作参考） |

## 📊 数据库架构概览

### 🎯 核心设计原则

1. **统一消息架构**
   - 群聊和私聊使用相同的 `groups` 表
   - 通过 `group_type` 区分：`public_chat` vs `private_chat`
   - 所有消息统一存储在 `messages` 表

2. **用户ID编码规范**
   - 64位整数：`TTTTTTTT SSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSSS`
   - 高8位（T）：用户类型（1=临时，2=普通，3=系统）
   - 低56位（S）：用户序号

3. **软删除机制**
   - 所有表支持 `is_deleted` + `deleted_at`
   - 自动触发器管理删除时间戳
   - 支持数据恢复和审计

### 📋 表结构总览

| 功能模块 | 表数量 | 核心表 |
|----------|--------|--------|
| **用户系统** | 4 | `users`, `user_settings`, `user_activity`, `user_relationships` |
| **群组系统** | 2 | `groups`, `group_members` |
| **消息系统** | 3 | `messages`, `message_media`, `message_reactions` |
| **瞬间系统** | 4 | `moments`, `tags`, `moment_tags`, `moment_likes` |
| **评论系统** | 1 | `moment_comments` |
| **安全系统** | 1 | `reports` |
| **通知系统** | 1 | `notifications` |
| **总计** | **16** | **+ 分区表 + 视图** |

### 🔧 核心业务函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `generate_user_id(type)` | 生成编码用户ID | `SELECT generate_user_id(2);` |
| `get_user_type(user_id)` | 提取用户类型 | `SELECT get_user_type(144115188075855872);` |
| `create_private_chat_group(u1, u2)` | 创建私聊群组 | `SELECT create_private_chat_group(1001, 1002);` |
| `send_unified_message(...)` | 发送统一消息 | 支持群聊和私聊 |

## 🔍 验证安装

### 基础验证

```sql
-- 检查表数量（应该是24个，包含分区表）
SELECT COUNT(*) as table_count 
FROM information_schema.tables 
WHERE table_schema = 'public' AND table_type = 'BASE TABLE';

-- 检查扩展
SELECT extname FROM pg_extension WHERE extname IN ('postgis', 'pgcrypto', 'uuid-ossp');
```

### 功能验证

```sql
-- 测试用户ID编码
SELECT 
    generate_user_id(1) as temp_user,
    generate_user_id(2) as regular_user,
    generate_user_id(3) as system_user;

-- 测试用户类型提取
SELECT 
    get_user_type(generate_user_id(1)) as temp_type,
    get_user_type(generate_user_id(2)) as regular_type,
    get_user_type(generate_user_id(3)) as system_type;

-- 测试私聊群组创建
SELECT create_private_chat_group(
    generate_user_id(2), 
    generate_user_id(2)
) as private_chat_id;
```

### 性能验证

```sql
-- 检查分区表
SELECT 
    schemaname,
    tablename,
    partitionbounds
FROM pg_tables 
WHERE tablename LIKE 'messages_%' OR tablename LIKE 'user_activity_%'
ORDER BY tablename;

-- 检查索引
SELECT 
    schemaname,
    tablename,
    indexname,
    indexdef
FROM pg_indexes 
WHERE schemaname = 'public'
  AND indexname LIKE 'idx_%'
ORDER BY tablename, indexname;
```

## 🛠️ 维护任务

### 定期维护

```sql
-- 1. 更新表统计信息（每日）
ANALYZE users, groups, messages, moments;

-- 2. 创建新分区（每月）
-- 注意：需要实现自动分区管理函数

-- 3. 清理过期数据（每周）
DELETE FROM users 
WHERE get_user_type(id) = 1 
  AND updated_at < NOW() - INTERVAL '7 days';
```

### 监控查询

```sql
-- 检查分区修剪效果
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM messages 
WHERE created_at >= '2024-01-01' 
  AND created_at < '2024-02-01';

-- 检查用户ID分布
SELECT 
    get_user_type(id) as user_type,
    COUNT(*) as count
FROM users 
GROUP BY get_user_type(id);
```

## 🚨 故障排除

### 常见问题

1. **扩展缺失**
   ```sql
   -- 确保所有必需扩展已安装
   CREATE EXTENSION IF NOT EXISTS postgis;
   CREATE EXTENSION IF NOT EXISTS pgcrypto;
   CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
   ```

2. **权限问题**
   ```sql
   -- 重新设置应用权限
   GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO hotspot_app;
   GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO hotspot_app;
   GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO hotspot_app;
   ```

3. **分区表问题**
   ```sql
   -- 检查分区表状态
   SELECT * FROM pg_partitioned_table;
   ```

### 性能问题

1. **查询慢**：检查是否正确使用了分区键
2. **索引失效**：重新分析表统计信息
3. **连接数过多**：检查连接池配置

## 📚 相关文档

- **需求文档**：`../.cursor/rules/backend_doc.mdc`
- **API设计**：`../src/presentation/http/`
- **业务逻辑**：`../src/application/services/`

## 🎯 最佳实践

1. **始终使用 `01_complete_schema.sql`** 进行新部署
2. **定期备份数据库** 在执行任何迁移前
3. **在测试环境验证** 所有数据库变更
4. **监控性能指标** 特别是分区表的查询性能
5. **保持统计信息更新** 确保查询优化器做出正确决策

---

**🎉 现在可以开始使用全新的 HotSpot 数据库架构进行开发了！**