# 会话撤销安全漏洞修复文档

## 问题概述

在之前的实现中，`SessionItem` 直接暴露了真实的 `session_id`，并且撤销会话时直接使用 `session_id`，存在严重的安全漏洞。

### 漏洞影响

| 风险类型 | 原实现（不安全） | 新实现（安全） |
|---------|----------------|--------------|
| **会话劫持** | 真实 session_id 暴露 | 只暴露 display_id |
| **批量撤销攻击** | session_id 可反复使用 | revoke_token 一次性使用 |
| **token 泄露** | session_id 永久有效 | revoke_token 5分钟过期 |
| **重放攻击** | session_id 可重放撤销 | revoke_token 不可重放 |

## 修复方案

### 1. 实现一次性 revoke_token 机制

#### 核心概念

- **display_id**: 用户可见的会话标识（如 `ses_abc123`），便于识别但无法用于操作
- **revoke_token**: 一次性撤销令牌，用于安全地撤销会话

#### 实现细节

**新增模块**: `backend/src/utils/session_token.rs`

```rust
/// 会话令牌生成器
#[derive(Clone)]
pub struct SessionTokenGenerator {
    secret: Vec<u8>,
}

impl SessionTokenGenerator {
    /// 生成展示用的会话ID：ses_{前8位UUID}
    pub fn generate_display_id(&self, session_id: Uuid) -> String;
    
    /// 生成一次性撤销令牌（HMAC-SHA256 签名）
    pub fn generate_revoke_token(&self, session_id: Uuid, user_id: i64) -> String;
    
    /// 验证 revoke_token 格式
    pub fn verify_revoke_token_format(&self, token: &str) -> bool;
}
```

#### 令牌结构

```
revoke_token = base64(HMAC-SHA256(secret, payload))

payload = session_id:user_id:salt:timestamp

salt = UUID v4（每次生成都不同）
```

### 2. 修改 DTO 定义

#### Response DTO

```rust
pub struct SessionItem {
    pub display_id: String,        // 展示用ID（如 "ses_abc123"）
    pub revoke_token: String,       // 一次性撤销令牌
    // ... 其他字段
    // ❌ 移除：pub session_id: String
}
```

#### Request DTO

```rust
pub struct RevokeSessionRequest {
    pub revoke_token: String,  // 一次性令牌
    // ❌ 移除：pub session_id: String
}
```

### 3. 修改业务逻辑

#### 列出会话 (`list_sessions`)

```rust
pub async fn list_sessions(...) -> Result<PaginatedData<SessionItem>> {
    // 1. 从数据库查询会话
    let (sessions, next_cursor) = self.session_repo.list_by_user(...).await?;
    
    // 2. 为每个会话生成 display_id 和 revoke_token
    for s in sessions {
        let display_id = self.session_token_gen.generate_display_id(s.session_id);
        let revoke_token = self.session_token_gen.generate_revoke_token(s.session_id, user_id);
        
        // 3. 将 revoke_token 存储到 Redis（TTL: 5分钟）
        redis::cmd("SETEX")
            .arg(format!("revoke_token:{}", revoke_token))
            .arg(300)
            .arg(s.session_id.to_string())
            .query_async(&mut self.redis).await?;
        
        items.push(SessionItem {
            display_id,
            revoke_token,
            // ... 其他字段
        });
    }
}
```

#### 撤销会话 (`revoke_session`)

```rust
pub async fn revoke_session(user_id: i64, req: &RevokeSessionRequest) -> Result<()> {
    // 1. 验证 revoke_token 格式
    if !self.session_token_gen.verify_revoke_token_format(&req.revoke_token) {
        return Err(AppError::BadRequest("令牌格式错误"));
    }
    
    // 2. 从 Redis 查询 revoke_token 对应的 session_id
    let revoke_key = format!("revoke_token:{}", req.revoke_token);
    let session_id_str: Option<String> = redis::cmd("GET")
        .arg(&revoke_key)
        .query_async(&mut self.redis).await?;
    
    let session_id = session_id_str.ok_or(AppError::NotFound("令牌无效或已使用"))?;
    
    // 3. 立即删除 revoke_token（一次性使用）
    redis::cmd("DEL").arg(&revoke_key).query_async(&mut self.redis).await?;
    
    // 4. 验证会话所有权
    let session = self.session_repo.find_by_id(session_id).await?
        .ok_or(AppError::NotFound("会话不存在"))?;
    
    if session.user_id != user_id {
        return Err(AppError::Forbidden("无权操作该会话"));
    }
    
    // 5. 撤销会话（DB + Redis 白名单）
    self.session_repo.delete(session_id).await?;
    self.remove_session_whitelist(session_id).await?;
    
    Ok(())
}
```

### 4. 集成 auth_cache 失效

在 handler 层清除认证缓存：

```rust
pub async fn revoke_session(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(revoke_req): Json<RevokeSessionRequest>,
) -> Result<Json<ApiResponse<()>>> {
    // 撤销会话
    state.auth_service.revoke_session(user.user_id, &revoke_req).await?;
    
    // 清除认证缓存（异步执行）
    tokio::spawn({
        let auth_cache = state.auth_cache.clone();
        let user_id = user.user_id;
        async move {
            auth_cache.invalidate_by_user(user_id).await;
        }
    });
    
    Ok(Json(ApiResponse::success_empty()))
}
```

## 安全特性

### 1. 一次性使用

- revoke_token 使用后立即从 Redis 中删除
- 即使 token 泄露，攻击者也只能撤销一次

### 2. 时间限制

- revoke_token 在 Redis 中的 TTL 为 5 分钟
- 过期后自动失效，无需手动清理

### 3. HMAC 签名

- 使用 HMAC-SHA256 签名，防止伪造和篡改
- 签名密钥为 JWT secret

### 4. 不暴露真实 ID

- 真实 session_id 从不对外暴露
- 只返回 display_id 和 revoke_token

## Redis 性能影响

### revoke_token 存储开销

```
每个 revoke_token 的 Redis 占用：
key: "revoke_token:{token}" ≈ 60 bytes
value: session_id (UUID string) ≈ 36 bytes
Redis overhead ≈ 100 bytes
总计 ≈ 196 bytes

假设场景：
- 10 万活跃用户
- 每个用户平均每天查看 2 次会话列表
- 每次列表包含 2 个会话
- revoke_token TTL = 5 分钟

峰值 revoke_token 数量：
= (10 万用户 × 2 次查看 × 2 会话) / (24小时 × 12个5分钟)
≈ 1,389 个 token

内存占用：
= 1,389 × 196 bytes
≈ 272 KB（可忽略不计）
```

**结论**: revoke_token 的存储开销极低，完全可以接受。

### 会话白名单性能分析

```
会话白名单 Keys 数量：
= 活跃用户数 × 平均设备数
= 100,000 × 2
= 200,000 keys

每个 key 的内存占用：
key: "session:{uuid}" ≈ 50 bytes
value: user_id (i64) ≈ 8 bytes
Redis overhead ≈ 100 bytes
总计 ≈ 158 bytes/key

总内存占用：
= 200,000 × 158 bytes
≈ 31.6 MB

QPS（已有 auth_cache，缓存命中率 90%）：
- 假设每秒 10,000 个认证请求
- 实际 Redis QPS = 1,000/s（完全可承受）
```

**结论**: 即使 100 万用户，Redis 内存占用也只需 ~300MB，QPS 在 auth_cache 加持下完全可控。

## 进一步优化建议

### 优化 1: 使用 Redis Hash 减少内存开销

如果需要进一步优化，可以考虑将同一用户的所有会话白名单合并为一个 Hash：

```rust
// 当前方案（每个会话一个 key）
key: session:{uuid}
value: user_id

// 优化方案（每个用户一个 Hash）
key: user_sessions:{user_id}
fields: {
    session_id_1: expiry_timestamp_1,
    session_id_2: expiry_timestamp_2,
    ...
}
```

**优点**:
- Redis overhead 减少（每个用户只有 1 个 key）
- 内存占用减少约 40%
- 可以批量操作同一用户的所有会话

**缺点**:
- 实现复杂度增加
- 需要手动管理过期时间
- 无法使用 Redis 的自动 TTL

**建议**: 暂时不优化，等用户规模达到 100 万+ 再考虑。

### 优化 2: 使用 Bloom Filter 预检查

在查询 Redis 前，先用 Bloom Filter 快速判断 session 是否可能存在：

```rust
// 1. Bloom Filter 预检查（内存操作，极快）
if !bloom_filter.might_contain(session_id) {
    return Err(AppError::Unauthorized("Session无效"));
}

// 2. Redis 精确查询
let is_valid: bool = redis::cmd("EXISTS")
    .arg(format!("session:{}", session_id))
    .query_async(&mut self.redis).await?;
```

**优点**:
- 减少无效请求对 Redis 的查询
- 内存占用极低（~10MB 可支持 100 万会话）

**缺点**:
- 增加实现复杂度
- 需要在会话创建/撤销时同步更新 Bloom Filter

**建议**: 等 Redis QPS 成为瓶颈时再考虑。

## 测试建议

### 单元测试

```rust
#[tokio::test]
async fn test_revoke_token_one_time_use() {
    // 1. 生成 revoke_token
    let token = gen.generate_revoke_token(session_id, user_id);
    
    // 2. 第一次撤销应该成功
    assert!(revoke_session(token.clone()).await.is_ok());
    
    // 3. 第二次撤销应该失败（token 已失效）
    assert!(revoke_session(token).await.is_err());
}

#[tokio::test]
async fn test_revoke_token_expiration() {
    // 1. 生成 revoke_token
    let token = gen.generate_revoke_token(session_id, user_id);
    
    // 2. 等待 5 分钟 + 1 秒
    tokio::time::sleep(Duration::from_secs(301)).await;
    
    // 3. 撤销应该失败（token 已过期）
    assert!(revoke_session(token).await.is_err());
}
```

### 集成测试

```bash
# 1. 获取会话列表
curl -X POST http://localhost:8080/api/v1/auth/me/sessions \
  -H "Authorization: Bearer $TOKEN"

# 2. 使用 revoke_token 撤销会话
curl -X POST http://localhost:8080/api/v1/auth/me/sessions/revoke \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"revoke_token": "xxx"}'

# 3. 再次使用同一 revoke_token 应该失败
curl -X POST http://localhost:8080/api/v1/auth/me/sessions/revoke \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"revoke_token": "xxx"}'
# Expected: {"code": 10300, "message": "令牌无效或已使用"}
```

### 安全测试

```bash
# 1. 尝试伪造 revoke_token（应该失败）
curl -X POST http://localhost:8080/api/v1/auth/me/sessions/revoke \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"revoke_token": "fake_token_123"}'
# Expected: {"code": 10001, "message": "令牌格式错误"}

# 2. 尝试使用其他用户的 revoke_token（应该失败）
# （需要先获取另一个用户的 revoke_token）
curl -X POST http://localhost:8080/api/v1/auth/me/sessions/revoke \
  -H "Authorization: Bearer $USER_A_TOKEN" \
  -d '{"revoke_token": "$USER_B_REVOKE_TOKEN"}'
# Expected: {"code": 10200, "message": "无权操作该会话"}
```

## 迁移指南

### 后端迁移

1. ✅ 已完成：添加 `session_token.rs` 模块
2. ✅ 已完成：修改 `SessionItem` DTO
3. ✅ 已完成：修改 `RevokeSessionRequest` DTO
4. ✅ 已完成：更新 `list_sessions()` 方法
5. ✅ 已完成：更新 `revoke_session()` 方法
6. ✅ 已完成：集成 auth_cache 失效

### 前端迁移

1. **更新 API 类型定义**:

```typescript
interface SessionItem {
  display_id: string;      // 新增
  revoke_token: string;    // 新增
  // session_id: string;   // 移除
  device_type?: string;
  // ... 其他字段
}

interface RevokeSessionRequest {
  revoke_token: string;  // 修改
  // session_id: string; // 移除
}
```

2. **更新 UI 显示**:

```tsx
// 显示会话时使用 display_id
<div>会话 ID: {session.display_id}</div>

// 撤销时使用 revoke_token
<button onClick={() => revokeSession(session.revoke_token)}>
  撤销此会话
</button>
```

3. **错误处理**:

```typescript
try {
  await api.revokeSession({ revoke_token });
} catch (error) {
  if (error.code === 10300) {
    toast.error("令牌已失效，请刷新会话列表");
  }
}
```

## 总结

本次修复完全解决了会话撤销的安全漏洞，实现了：

✅ **一次性 revoke_token**: 防止批量攻击  
✅ **时间限制**: 5 分钟自动过期  
✅ **HMAC 签名**: 防止伪造  
✅ **不暴露真实 ID**: 降低会话劫持风险  
✅ **集成 auth_cache**: 撤销后立即失效  
✅ **性能开销可控**: Redis 内存占用和 QPS 都在可接受范围

**推荐立即上线**，安全性显著提升，性能影响可忽略不计。
