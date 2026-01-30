# 会话白名单 Redis 性能优化指南

## 问题分析

用户担心：会话白名单在用户多的情况下是否会导致 Redis 负载过大？

## 性能评估

### 当前设计

```rust
// 每个会话一个 Redis key
key: session:{uuid}
value: user_id (i64)
TTL: access_token_expiration (通常 30-60 分钟)
```

### 负载计算

#### 场景 1: 10 万用户

```
活跃会话数：100,000 用户 × 2 设备 = 200,000 会话
内存占用：200,000 × 158 bytes = 31.6 MB
Redis QPS（有 auth_cache，90% 命中）：10,000 请求/s × 10% = 1,000/s
```

**结论**: ✅ 完全可控，无性能问题。

#### 场景 2: 100 万用户

```
活跃会话数：1,000,000 用户 × 2 设备 = 2,000,000 会话
内存占用：2,000,000 × 158 bytes = 316 MB
Redis QPS（有 auth_cache，90% 命中）：100,000 请求/s × 10% = 10,000/s
```

**结论**: ⚠️ 可控但需要优化，Redis QPS 开始接近瓶颈（单实例 ~50,000 QPS）。

#### 场景 3: 1000 万用户

```
活跃会话数：10,000,000 用户 × 2 设备 = 20,000,000 会话
内存占用：20,000,000 × 158 bytes = 3.16 GB
Redis QPS（有 auth_cache，90% 命中）：1,000,000 请求/s × 10% = 100,000/s
```

**结论**: ❌ 需要优化或引入 Redis Cluster。

## 优化方案

### 方案 1: 提高 auth_cache 命中率（推荐）

#### 当前配置

```rust
let auth_cache = AuthCache::new(
    10_000,  // 最大缓存 1 万个 token
    300,     // TTL 5 分钟
);
```

#### 优化配置

```rust
let auth_cache = AuthCache::new(
    100_000,  // 提升到 10 万
    600,      // 提升到 10 分钟
);
```

**效果**：
- 缓存命中率从 90% 提升到 95%
- Redis QPS 减少 50%
- 内存增加约 15MB（可接受）

**适用场景**: 100 万用户以下

### 方案 2: 使用 Redis Hash 合并会话（推荐 100 万+ 用户）

#### 当前设计（每个会话一个 key）

```redis
session:uuid-1 -> user_id_1
session:uuid-2 -> user_id_2
session:uuid-3 -> user_id_3
```

**Redis overhead**: 每个 key ~100 bytes

#### 优化设计（每个用户一个 Hash）

```redis
user_sessions:user_id -> {
    uuid-1: expiry_timestamp_1,
    uuid-2: expiry_timestamp_2,
    uuid-3: expiry_timestamp_3,
}
```

**Redis overhead**: 整个 Hash 只有 ~100 bytes

#### 实现示例

```rust
// 设置会话白名单
async fn set_session_whitelist(&self, session_id: Uuid, user_id: i64) -> Result<()> {
    let hash_key = format!("user_sessions:{}", user_id);
    let expiry = Utc::now().timestamp() + self.access_token_expiration;
    
    redis::cmd("HSET")
        .arg(&hash_key)
        .arg(session_id.to_string())
        .arg(expiry)
        .query_async(&mut self.redis.clone())
        .await?;
    
    // 设置整个 Hash 的 TTL（以最长的会话过期时间为准）
    redis::cmd("EXPIRE")
        .arg(&hash_key)
        .arg(self.access_token_expiration * 2)  // 留足够的缓冲
        .query_async(&mut self.redis.clone())
        .await?;
    
    Ok(())
}

// 验证会话白名单
async fn check_session_whitelist(&self, session_id: Uuid, user_id: i64) -> Result<bool> {
    let hash_key = format!("user_sessions:{}", user_id);
    
    // 获取过期时间
    let expiry: Option<i64> = redis::cmd("HGET")
        .arg(&hash_key)
        .arg(session_id.to_string())
        .query_async(&mut self.redis.clone())
        .await?;
    
    match expiry {
        Some(exp) => Ok(Utc::now().timestamp() < exp),
        None => Ok(false),
    }
}

// 移除会话白名单
async fn remove_session_whitelist(&self, session_id: Uuid, user_id: i64) -> Result<()> {
    let hash_key = format!("user_sessions:{}", user_id);
    
    redis::cmd("HDEL")
        .arg(&hash_key)
        .arg(session_id.to_string())
        .query_async(&mut self.redis.clone())
        .await?;
    
    Ok(())
}
```

#### 性能对比

| 指标 | 当前方案 | Hash 方案 | 提升 |
|-----|---------|----------|------|
| 内存占用（100万用户） | 316 MB | 190 MB | **40% ↓** |
| Redis Keys 数量 | 2,000,000 | 1,000,000 | **50% ↓** |
| Redis overhead | 高 | 低 | **40% ↓** |
| 撤销所有会话 | 需要多次 DEL | 一次 DEL | **更快** |

#### 权衡

**优点**:
- 内存占用减少 40%
- Key 数量减少 50%
- 批量操作更高效

**缺点**:
- 实现复杂度增加
- 需要手动管理过期时间（无法用 Redis TTL）
- 需要定期清理过期的 field（可用 Lua 脚本）

### 方案 3: 引入 Redis Cluster（推荐 1000 万+ 用户）

当用户规模达到千万级别时，单个 Redis 实例可能成为瓶颈，建议引入 Redis Cluster：

```rust
use redis::cluster::ClusterClient;

let redis_cluster = ClusterClient::new(vec![
    "redis://node1:6379",
    "redis://node2:6379",
    "redis://node3:6379",
])?;
```

**优点**:
- 水平扩展，QPS 和内存都可线性增长
- 高可用性

**缺点**:
- 运维复杂度增加
- 部分命令不支持（如 Lua 脚本跨 slot）

## 定期清理策略

### 问题

虽然 Redis key 有 TTL 自动过期，但数据库中的 `sessions` 表会持续增长。

### 解决方案：定期清理任务

```rust
/// 定期清理过期和已撤销的会话（数据库）
///
/// 建议：每天凌晨 3 点执行
pub async fn cleanup_expired_sessions(&self) -> Result<u64> {
    let result = sqlx::query!(
        r#"
        DELETE FROM sessions
        WHERE (expires_at < NOW() OR is_revoked = TRUE)
          AND created_at < NOW() - INTERVAL '30 days'
        "#
    )
    .execute(&self.db)
    .await?;

    let deleted = result.rows_affected();
    tracing::info!(deleted_sessions = deleted, "Cleaned up expired sessions");
    
    Ok(deleted)
}
```

**执行策略**:
- 保留最近 30 天的会话记录（用于审计）
- 使用 Tokio cron 或外部调度器（如 Kubernetes CronJob）

## 监控指标

建议添加以下监控指标：

### Redis 相关

```rust
// 1. 会话白名单 key 数量
redis::cmd("DBSIZE").query_async(&mut redis).await

// 2. 会话白名单内存占用
redis::cmd("INFO").arg("memory").query_async(&mut redis).await

// 3. Redis QPS
// 从 Redis INFO stats 中获取 instantaneous_ops_per_sec
```

### 应用层面

```rust
// 4. auth_cache 命中率
let stats = state.auth_cache.stats();
let hit_rate = calculate_hit_rate(stats);

// 5. 活跃会话数量
SELECT COUNT(*) FROM sessions WHERE is_revoked = FALSE AND expires_at > NOW();

// 6. revoke_token 使用率
// 监控 "令牌无效或已使用" 错误的频率
```

### 告警规则

| 指标 | 阈值 | 告警级别 | 处理建议 |
|-----|------|---------|---------|
| Redis 内存占用 | > 2GB | Warning | 考虑使用 Hash 方案 |
| Redis 内存占用 | > 4GB | Critical | 启用 Redis Cluster |
| Redis QPS | > 30,000 | Warning | 提高 auth_cache 容量和 TTL |
| Redis QPS | > 50,000 | Critical | 启用 Redis Cluster |
| auth_cache 命中率 | < 80% | Warning | 检查缓存配置 |
| 无效 revoke_token 错误 | 突然增长 | Critical | 可能遭受攻击 |

## 推荐部署策略

### 10 万用户以下

```
✅ 当前方案 + auth_cache（默认配置）
- Redis: 单实例
- auth_cache: 10,000 容量，300s TTL
```

### 10-100 万用户

```
✅ 当前方案 + auth_cache（优化配置）
- Redis: 单实例（建议 16GB 内存）
- auth_cache: 100,000 容量，600s TTL
```

### 100-1000 万用户

```
✅ Hash 方案 + auth_cache（优化配置）
- Redis: 单实例（建议 32GB 内存）
- auth_cache: 500,000 容量，600s TTL
```

### 1000 万用户以上

```
✅ Hash 方案 + Redis Cluster + auth_cache
- Redis: Cluster（3 主 3 从）
- auth_cache: 1,000,000 容量，600s TTL
```

## 性能基准测试

### 测试方法

```bash
# 使用 wrk 测试认证接口 QPS
wrk -t4 -c100 -d30s \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/user/me/profile

# 监控 Redis 性能
redis-cli --stat

# 监控 auth_cache 命中率
# 查看应用日志中的 cache_hit 字段统计
```

### 预期结果

| 用户规模 | QPS | P99 延迟 | Redis 内存 | auth_cache 命中率 |
|---------|-----|---------|-----------|-----------------|
| 10 万 | 5,000 | < 10ms | 32 MB | 90%+ |
| 100 万 | 50,000 | < 20ms | 316 MB | 92%+ |
| 1000 万 | 500,000 | < 50ms | 3.16 GB | 95%+ |

## 总结

**回答用户的担忧**：

✅ **10-100 万用户**: 当前设计完全没问题，Redis 负载可控  
✅ **100-1000 万用户**: 可考虑 Hash 方案优化  
⚠️ **1000 万用户以上**: 需要 Redis Cluster

**关键因素**：
1. **auth_cache 是关键**: 90% 缓存命中率使 Redis 负载减少 90%
2. **TTL 自动清理**: 无需手动维护，过期会话自动删除
3. **内存开销低**: 即使 100 万用户也只需 ~300MB

**建议**：
- 当前设计可以支撑到 100 万用户，无需优化
- 达到 100 万用户时考虑使用 Hash 方案
- 监控 Redis 内存和 QPS，提前预警

**性价比**: 当前方案实现简单、性能足够、自动清理，性价比最高。
