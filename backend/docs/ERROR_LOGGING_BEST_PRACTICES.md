# 后端错误处理与日志记录最佳实践

## 设计原则

### 1. 双层信息策略

```rust
// ✅ 正确：详细错误记录到日志，规范错误码返回给客户端
pub async fn follow_user(&self, from_user_id: i64, req: &FollowRequest) -> Result<()> {
    let to_user_id = req.user_id;
    
    // 检查黑名单
    if self.blocks_repo.is_blocked(to_user_id, from_user_id).await? {
        // 记录详细上下文到日志
        tracing::warn!(
            from_user_id,
            to_user_id,
            action = "follow_user",
            reason = "blocked_by_target",
            "关注失败：已被对方拉黑"
        );
        // 返回规范错误码
        return Err(AppError::BlockedByUser("已被对方拉黑".to_string()));
    }
    
    // ... 其他逻辑
}

// ❌ 错误：没有日志记录
pub async fn follow_user(&self, from_user_id: i64, req: &FollowRequest) -> Result<()> {
    if self.blocks_repo.is_blocked(to_user_id, from_user_id).await? {
        return Err(AppError::BlockedByUser("已被对方拉黑".to_string()));
    }
}

// ❌ 错误：泄露敏感信息到客户端
pub async fn follow_user(&self, from_user_id: i64, req: &FollowRequest) -> Result<()> {
    if self.blocks_repo.is_blocked(to_user_id, from_user_id).await? {
        return Err(AppError::BlockedByUser(format!(
            "User {} is blocked by user {}", 
            from_user_id, 
            to_user_id
        ))); // ❌ 泄露 user_id
    }
}
```

### 2. 日志级别选择

| 级别 | 使用场景 | 示例 |
|-----|---------|------|
| `error!` | 系统错误、数据库错误、不应该发生的错误 | 数据库连接失败、Redis 不可用 |
| `warn!` | 业务规则冲突、权限拒绝、预期的失败 | 黑名单拦截、权限不足、资源不存在 |
| `info!` | 重要的业务事件 | 用户注册、关注成功、群组创建 |
| `debug!` | 调试信息、幂等操作 | 已关注（幂等）、缓存命中 |
| `trace!` | 详细的执行流程 | 函数入口、循环迭代 |

### 3. 结构化字段

```rust
// ✅ 正确：使用结构化字段
tracing::warn!(
    from_user_id,              // 关键业务 ID
    to_user_id,
    action = "follow_user",    // 操作类型
    reason = "blocked",        // 失败原因
    error_code = %err.error_code(),  // 错误码
    "关注失败：已被对方拉黑"      // 人类可读的描述
);

// ❌ 错误：全部拼接到字符串
tracing::warn!("关注失败：用户 {} 被用户 {} 拉黑", from_user_id, to_user_id);
```

## 典型场景模式

### 场景 1：资源不存在

```rust
// ✅ 最佳实践
pub async fn get_moment(&self, moment_id: &str, viewer_id: Option<i64>) -> Result<MomentDetail> {
    let moment = self.moment_repo.find_by_id(moment_id).await?
        .ok_or_else(|| {
            tracing::warn!(
                moment_id,
                viewer_id,
                action = "get_moment",
                reason = "not_found",
                "动态不存在"
            );
            AppError::MomentNotFound("动态不存在".to_string())
        })?;
    
    Ok(moment)
}
```

### 场景 2：权限检查失败

```rust
// ✅ 最佳实践
pub async fn send_private_message(
    &self,
    from_user_id: i64,
    req: &SendPrivateMessageRequest,
) -> Result<String> {
    let to_user_id = req.target_user_id;
    
    // 检查对方是否允许接收私聊
    let settings = self.user_settings_repo.find_by_user_id(to_user_id).await?;
    if let Some(s) = settings {
        if s.private_chat == PrivateChatPermission::Nobody {
            tracing::warn!(
                from_user_id,
                to_user_id,
                action = "send_private_message",
                reason = "permission_denied",
                permission = "private_chat_nobody",
                "发送私聊失败：对方不允许接收私聊"
            );
            return Err(AppError::PrivateChatNotAllowed(
                "对方不允许接收私聊".to_string()
            ));
        }
    }
    
    // ... 继续执行
}
```

### 场景 3：业务规则冲突

```rust
// ✅ 最佳实践
pub async fn create_group(
    &self,
    creator_id: i64,
    req: &CreateGroupRequest,
) -> Result<String> {
    // 检查用户已加入的群组数量
    let current_count = self.group_repo.count_user_groups(creator_id).await?;
    let max_groups = 100; // 从配置获取
    
    if current_count >= max_groups {
        tracing::warn!(
            creator_id,
            current_count,
            max_groups,
            action = "create_group",
            reason = "quota_exceeded",
            "创建群组失败：已达到最大群组数量限制"
        );
        return Err(AppError::QuotaExceeded(
            format!("已达到最大群组数量限制（{}）", max_groups)
        ));
    }
    
    // ... 创建群组
}
```

### 场景 4：数据库/外部错误

```rust
// ✅ 最佳实践
pub async fn update_profile(
    &self,
    user_id: i64,
    req: &UpdateProfileRequest,
) -> Result<MyCard> {
    // 更新数据库
    let updated = self.user_repo.update_profile(user_id, req).await
        .map_err(|e| {
            // 数据库错误应该用 error! 级别
            tracing::error!(
                user_id,
                action = "update_profile",
                error = %e,
                error_type = "database",
                "更新用户资料失败：数据库错误"
            );
            e
        })?;
    
    // 清除缓存（失败不影响主流程）
    if let Err(e) = self.clear_user_cache(user_id).await {
        tracing::warn!(
            user_id,
            error = %e,
            "清除用户缓存失败（非致命）"
        );
    }
    
    Ok(updated)
}
```

### 场景 5：幂等操作

```rust
// ✅ 最佳实践
pub async fn follow_user(&self, from_user_id: i64, req: &FollowRequest) -> Result<()> {
    let to_user_id = req.user_id;
    
    // 已关注则视为幂等成功
    if self.follows_repo.exists(from_user_id, to_user_id).await? {
        tracing::debug!(
            from_user_id,
            to_user_id,
            action = "follow_user",
            result = "idempotent",
            "已关注，幂等返回成功"
        );
        return Ok(());
    }
    
    // ... 创建关注关系
}
```

## 常见反模式

### ❌ 反模式 1：完全没有日志

```rust
// ❌ 错误
pub async fn delete_moment(&self, moment_id: &str, user_id: i64) -> Result<()> {
    let moment = self.moment_repo.find_by_id(moment_id).await?
        .ok_or_else(|| AppError::MomentNotFound("动态不存在".to_string()))?;
    
    if moment.user_id != user_id {
        return Err(AppError::PermissionDenied("无权删除该动态".to_string()));
    }
    
    self.moment_repo.delete(moment_id).await?;
    Ok(())
}

// ✅ 正确：添加关键日志
pub async fn delete_moment(&self, moment_id: &str, user_id: i64) -> Result<()> {
    let moment = self.moment_repo.find_by_id(moment_id).await?
        .ok_or_else(|| {
            tracing::warn!(moment_id, user_id, "动态不存在");
            AppError::MomentNotFound("动态不存在".to_string())
        })?;
    
    if moment.user_id != user_id {
        tracing::warn!(
            moment_id,
            moment_owner = moment.user_id,
            requester = user_id,
            "删除动态失败：无权限"
        );
        return Err(AppError::PermissionDenied("无权删除该动态".to_string()));
    }
    
    self.moment_repo.delete(moment_id).await
        .map_err(|e| {
            tracing::error!(moment_id, user_id, error = %e, "删除动态失败");
            e
        })?;
    
    tracing::info!(moment_id, user_id, "动态删除成功");
    Ok(())
}
```

### ❌ 反模式 2：日志泄露敏感信息

```rust
// ❌ 错误：日志中包含密码
tracing::info!(user_id, password = %req.password, "用户登录");

// ✅ 正确：不记录敏感信息
tracing::info!(user_id, login_id = %req.login_id, "用户登录尝试");
```

### ❌ 反模式 3：错误消息包含内部实现细节

```rust
// ❌ 错误
return Err(AppError::BadRequest(format!(
    "User {} blocked by user {} at table user_blocks row 12345",
    from_user_id,
    to_user_id
)));

// ✅ 正确：用户友好的错误消息
tracing::warn!(
    from_user_id,
    to_user_id,
    table = "user_blocks",
    row_id = 12345,  // 详细信息记录到日志
    "关注失败：已被对方拉黑"
);
return Err(AppError::BlockedByUser("已被对方拉黑".to_string()));
```

## 实施步骤

### 第一优先级：关键业务流程

1. **认证相关**（`auth_service.rs`）
   - 登录失败
   - Token 验证失败
   - 刷新 Token 失败

2. **关注系统**（`user_service.rs`）
   - 关注/取消关注
   - 拉黑/解除拉黑
   - 权限检查

3. **消息发送**（`private_message_service.rs`, `group_message_service.rs`）
   - 发送失败
   - 权限检查
   - 黑名单拦截

### 第二优先级：资源操作

4. **内容创建**（`moment_service.rs`, `comment_service.rs`）
   - 创建失败
   - 引用校验失败

5. **群组管理**（`group_service.rs`）
   - 加入/退出
   - 权限操作
   - 成员管理

### 第三优先级：查询操作

6. **搜索和列表**
   - 资源不存在（用 debug 级别）
   - 权限不足（用 warn 级别）

## 代码模板

### 模板 1：资源不存在

```rust
let resource = self.repo.find_by_id(id).await?
    .ok_or_else(|| {
        tracing::warn!(
            id,
            user_id,
            action = "get_resource",
            reason = "not_found",
            "资源不存在"
        );
        AppError::ResourceNotFound("资源不存在".to_string())
    })?;
```

### 模板 2：权限检查

```rust
if !self.has_permission(user_id, resource_id).await? {
    tracing::warn!(
        user_id,
        resource_id,
        resource_type = "moment",
        action = "delete",
        reason = "permission_denied",
        "权限不足"
    );
    return Err(AppError::PermissionDenied("无权操作该资源".to_string()));
}
```

### 模板 3：业务规则冲突

```rust
if current_count >= limit {
    tracing::warn!(
        user_id,
        current_count,
        limit,
        action = "create_resource",
        reason = "quota_exceeded",
        "配额已满"
    );
    return Err(AppError::QuotaExceeded(format!("已达到上限（{}）", limit)));
}
```

### 模板 4：外部错误传播

```rust
let result = self.repo.update(data).await
    .map_err(|e| {
        tracing::error!(
            user_id,
            resource_id,
            action = "update_resource",
            error = %e,
            error_type = "database",
            "数据库更新失败"
        );
        e
    })?;
```

## 实施检查清单

### ✅ 应该记录的信息

- [ ] 操作类型（action）
- [ ] 关键业务 ID（user_id, moment_id 等）
- [ ] 失败原因（reason）
- [ ] 错误码（error_code）
- [ ] 上下文数据（current_count, limit 等）

### ❌ 不应该记录的信息

- [ ] 密码、Token
- [ ] 完整的请求 body（可能包含敏感信息）
- [ ] 数据库 SQL（可能泄露表结构）
- [ ] 内部实现细节

### ✅ 应该返回给客户端的信息

- [ ] 规范定义的错误码
- [ ] 用户友好的错误消息
- [ ] 必要的字段名（如"nickname"参数无效）

### ❌ 不应该返回给客户端的信息

- [ ] 数据库错误详情
- [ ] 其他用户的 ID
- [ ] 内部状态信息
- [ ] 堆栈跟踪

## 示例修复：用户关注功能

### 修复前

```rust
pub async fn follow_user(&self, from_user_id: i64, req: &FollowRequest) -> Result<()> {
    let to_user_id = req.user_id;

    if from_user_id == to_user_id {
        return Err(AppError::BadRequest("不能关注自己".to_string()));
    }

    let target_user = self.user_repo.find_by_id(to_user_id).await?
        .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

    if self.blocks_repo.is_blocked(to_user_id, from_user_id).await? {
        return Err(AppError::BlockedByUser("已被对方拉黑".to_string()));
    }

    // ... 更多逻辑
}
```

### 修复后

```rust
pub async fn follow_user(&self, from_user_id: i64, req: &FollowRequest) -> Result<()> {
    let to_user_id = req.user_id;

    // 1. 自我操作检查
    if from_user_id == to_user_id {
        tracing::warn!(
            user_id = from_user_id,
            action = "follow_user",
            reason = "self_operation",
            error_code = 10411,
            "关注失败：不能关注自己"
        );
        return Err(AppError::BadRequest("不能关注自己".to_string()));
    }

    // 2. 目标用户存在性检查
    let target_user = self.user_repo.find_by_id(to_user_id).await?
        .ok_or_else(|| {
            tracing::warn!(
                from_user_id,
                to_user_id,
                action = "follow_user",
                reason = "target_not_found",
                error_code = 10310,
                "关注失败：目标用户不存在"
            );
            AppError::UserNotFound("用户不存在".to_string())
        })?;

    // 3. 黑名单检查
    if self.blocks_repo.is_blocked(to_user_id, from_user_id).await? {
        tracing::warn!(
            from_user_id,
            to_user_id,
            action = "follow_user",
            reason = "blocked_by_target",
            error_code = 10211,
            "关注失败：已被对方拉黑"
        );
        return Err(AppError::BlockedByUser("已被对方拉黑".to_string()));
    }

    // 4. 系统用户检查
    if target_user.user_type == crate::types::enums::UserType::System {
        tracing::warn!(
            from_user_id,
            to_user_id,
            user_type = ?target_user.user_type,
            action = "follow_user",
            reason = "system_user",
            error_code = 10213,
            "关注失败：系统用户不允许被关注"
        );
        return Err(AppError::FollowNotAllowed("对方不允许被关注".to_string()));
    }

    // 5. 幂等检查
    if self.follows_repo.exists(from_user_id, to_user_id).await? {
        tracing::debug!(
            from_user_id,
            to_user_id,
            action = "follow_user",
            result = "idempotent",
            "已关注，幂等返回成功"
        );
        return Ok(());
    }

    // 6. 隐私设置检查
    let target_settings = self.user_settings_repo.find_by_user_id(to_user_id).await?;
    if let Some(settings) = target_settings {
        if !settings.allow_follow {
            // 需要审批：创建关注请求
            let request_id = self.follow_request_repo
                .create(from_user_id, to_user_id, None)
                .await?;
            
            tracing::info!(
                from_user_id,
                to_user_id,
                request_id = %request_id,
                action = "follow_user",
                result = "request_created",
                "创建关注请求（需要审批）"
            );
            
            self.event_bus.publish(AppEvent::FollowRequestReceived {
                request_id: request_id.to_string(),
                requester_id: from_user_id,
                target_id: to_user_id,
            });
            return Ok(());
        }
    }

    // 7. 创建关注关系
    self.follows_repo.create(from_user_id, to_user_id).await
        .map_err(|e| {
            tracing::error!(
                from_user_id,
                to_user_id,
                action = "follow_user",
                error = %e,
                error_type = "database",
                "创建关注关系失败"
            );
            e
        })?;

    // 8. 发布事件
    let is_mutual = self.follows_repo.exists(to_user_id, from_user_id).await?;
    
    tracing::info!(
        from_user_id,
        to_user_id,
        is_mutual,
        action = "follow_user",
        result = "success",
        "关注成功"
    );
    
    self.event_bus.publish(AppEvent::NewFollower {
        follower_id: from_user_id,
        user_id: to_user_id,
        is_mutual,
    });
    
    if is_mutual {
        self.event_bus.publish(AppEvent::MutualFollow {
            user_id: from_user_id,
            mutual_user_id: to_user_id,
        });
    }

    Ok(())
}
```

## 性能考虑

### 结构化字段的性能

- ✅ 结构化字段（`tracing::warn!(user_id, action = "...")`）性能开销很小
- ✅ 在生产环境可以通过日志级别控制
- ✅ 支持高效的日志过滤和聚合

### 建议配置

```toml
# 开发环境
RUST_LOG=debug,hyper=info,sqlx=info

# 生产环境
RUST_LOG=info,hotspot_backend=warn
```

## 总结

**核心原则**：

1. **详细日志 + 规范响应**：日志记录详细上下文，响应返回规范错误码
2. **结构化字段**：便于查询、过滤、聚合
3. **合理分级**：error/warn/info/debug/trace
4. **保护隐私**：不记录密码、Token 等敏感信息
5. **用户友好**：返回清晰的、可操作的错误消息

**实施建议**：

- 优先修复关键业务流程（认证、关注、消息）
- 使用模板快速复制粘贴
- 在 Code Review 中强制检查
- 通过日志监控识别未覆盖的错误路径
