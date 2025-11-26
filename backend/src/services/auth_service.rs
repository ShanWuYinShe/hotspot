use chrono::{Duration, Utc};
use redis::aio::ConnectionManager;
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreateTempRequest, LoginRequest, RegisterRequest, RefreshTokenRequest,
            RequestPasswordResetRequest, ResetPasswordRequest, SetPasswordRequest,
            VerifyResetCodeRequest, ListSessionsRequest, RevokeSessionRequest,
        },
        response::{
            AuthData, CheckStatusData, SessionItem, PaginatedData, PaginationMeta,
            SetPasswordData, RequestPasswordResetData, VerifyResetCodeData,
        },
    },
    error::{AppError, Result},
    repositories::{
        user_repo::UserRepository,
        session_repo::SessionRepository
    },
    utils::{id_gen::IdGenerator, jwt::JwtUtil, password::PasswordUtil, validation::Validator},
};

pub struct AuthService {
    user_repo: UserRepository,
    session_repo: SessionRepository,
    redis: ConnectionManager,
    jwt_secret: String,
    access_token_expiration: i64,
    refresh_token_expiration: i64,
}

impl AuthService {
    pub fn new(
        db: &PgPool,
        redis: &ConnectionManager,
        jwt_secret: String,
        access_token_expiration: i64,
        refresh_token_expiration: i64,
    ) -> Self {
        Self {
            user_repo: UserRepository::new(db.clone()),
            session_repo: SessionRepository::new(db.clone()),
            redis: redis.clone(),
            jwt_secret,
            access_token_expiration,
            refresh_token_expiration,
        }
    }

    /// 用户注册
    pub async fn register(&self, req: &RegisterRequest) -> Result<AuthData> {
        // 1. 检查登录ID是否已存在
        if self.user_repo.exists_by_login_id(&req.login_id).await? {
            return Err(AppError::Conflict("该登录ID已被注册".to_string()));
        }

        // 2. 生成用户ID（类型：普通用户 0x02）
        let user_id = IdGenerator::generate_user_id(0x02);

        // 3. 对客户端哈希进行 bcrypt 二次哈希
        let password_hash = PasswordUtil::hash_password(&req.password_hash)?;

        // 4. 创建用户
        self.user_repo.create(
            user_id,
            Some(req.login_id.clone()),
            req.nickname.clone(),
            Some(password_hash.clone()),
        ).await?;

        // 5. 生成 JWT Token
        let access_token =
            JwtUtil::generate_token(user_id, self.access_token_expiration, &self.jwt_secret)?;
        let refresh_token =
            JwtUtil::generate_token(user_id, self.refresh_token_expiration, &self.jwt_secret)?;

        // 6. 存储 Refresh Token 到 Redis
        let key = format!("refresh_token:{}", user_id);
        let _: () = redis::cmd("SETEX")
            .arg(&key)
            .arg(self.refresh_token_expiration)
            .arg(&refresh_token)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        // 7. 构造响应
        Ok(AuthData {
            user_id: user_id.to_string(),
            nickname: req.nickname.clone(),
            access_token,
            refresh_token,
            expires_at: (Utc::now() + Duration::seconds(self.access_token_expiration)).to_rfc3339(),
        })
    }

    /// 用户登录
    pub async fn login(&self, req: &LoginRequest) -> Result<AuthData> {
        // 1. 查找用户
        let user = self
            .user_repo
            .find_by_login_id(&req.login_id)
            .await?
            .ok_or_else(|| AppError::Unauthorized("用户名或密码错误".to_string()))?;

        // 2. 验证密码
        let password_hash = user
            .password_hash
            .ok_or_else(|| AppError::Unauthorized("该账号未设置密码".to_string()))?;

        if !PasswordUtil::verify_password(&req.password_hash, &password_hash)? {
            return Err(AppError::Unauthorized("用户名或密码错误".to_string()));
        }

        // 3. 生成 Token
        let access_token =
            JwtUtil::generate_token(user.user_id, self.access_token_expiration, &self.jwt_secret)?;
        let refresh_token = JwtUtil::generate_token(
            user.user_id,
            self.refresh_token_expiration,
            &self.jwt_secret,
        )?;

        // 4. 存储 Refresh Token
        let key = format!("refresh_token:{}", user.user_id);
        let _: () = redis::cmd("SETEX")
            .arg(&key)
            .arg(self.refresh_token_expiration)
            .arg(&refresh_token)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        // 5. 更新最后活跃时间
        self.user_repo.update_last_active(user.user_id).await?;

        Ok(AuthData {
            user_id: user.user_id.to_string(),
            nickname: user.nickname,
            access_token,
            refresh_token,
            expires_at: (Utc::now() + Duration::seconds(self.access_token_expiration)).to_rfc3339(),
        })
    }

    /// 创建临时用户
    pub async fn create_temp(&self, req: &CreateTempRequest) -> Result<AuthData> {
        // 1. 生成临时用户ID（类型：临时用户 0x01）
        let user_id = IdGenerator::generate_user_id(0x01);

        // 2. 创建用户（无密码）
        self.user_repo.create(
            user_id,
            None,
            req.nickname.clone(),
            None,
        ).await?;

        // 3. 生成 Token
        let access_token =
            JwtUtil::generate_token(user_id, self.access_token_expiration, &self.jwt_secret)?;
        let refresh_token =
            JwtUtil::generate_token(user_id, self.refresh_token_expiration, &self.jwt_secret)?;

        // 4. 存储 Refresh Token
        let key = format!("refresh_token:{}", user_id);
        let _: () = redis::cmd("SETEX")
            .arg(&key)
            .arg(self.refresh_token_expiration)
            .arg(&refresh_token)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        Ok(AuthData {
            user_id: user_id.to_string(),
            nickname: req.nickname.clone(),
            access_token,
            refresh_token,
            expires_at: (Utc::now() + Duration::seconds(self.access_token_expiration)).to_rfc3339(),
        })
    }

    /// 刷新Token
    pub async fn refresh_token(&self, req: &RefreshTokenRequest) -> Result<AuthData> {
        // 1. 验证 Refresh Token
        let claims = JwtUtil::verify_token(&req.refresh_token, &self.jwt_secret)?;
        let user_id = claims
            .sub
            .parse::<i64>()
            .map_err(|_| AppError::Unauthorized("无效的Token".to_string()))?;

        // 2. 检查 Redis 中的 Refresh Token
        let key = format!("refresh_token:{}", user_id);
        let stored_token: Option<String> = redis::cmd("GET")
            .arg(&key)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        if stored_token.as_deref() != Some(&req.refresh_token) {
            return Err(AppError::Unauthorized("Refresh Token 无效".to_string()));
        }

        // 3. 查找用户
        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or_else(|| AppError::NotFound("用户不存在".to_string()))?;

        // 4. 生成新的 Token
        let access_token =
            JwtUtil::generate_token(user_id, self.access_token_expiration, &self.jwt_secret)?;
        let refresh_token =
            JwtUtil::generate_token(user_id, self.refresh_token_expiration, &self.jwt_secret)?;

        // 5. 更新 Redis 中的 Refresh Token
        let _: () = redis::cmd("SETEX")
            .arg(&key)
            .arg(self.refresh_token_expiration)
            .arg(&refresh_token)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        Ok(AuthData {
            user_id: user_id.to_string(),
            nickname: user.nickname,
            access_token,
            refresh_token,
            expires_at: (Utc::now() + Duration::seconds(self.access_token_expiration)).to_rfc3339(),
        })
    }

    /// 登出
    pub async fn logout(&self, user_id: i64, token: &str) -> Result<()> {
        // 1. 删除 Refresh Token
        let key = format!("refresh_token:{}", user_id);
        let _: () = redis::cmd("DEL")
            .arg(&key)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        // 2. 将 Access Token 加入黑名单
        let blacklist_key = format!("token_blacklist:{}", token);
        let _: () = redis::cmd("SETEX")
            .arg(&blacklist_key)
            .arg(self.access_token_expiration)
            .arg(1)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        Ok(())
    }

    /// 检查认证状态
    pub async fn check_status(&self, user_id: i64, token: &str) -> Result<CheckStatusData> {
        // 1. 验证 Token
        let claims = JwtUtil::verify_token(token, &self.jwt_secret)?;
        
        // 2. 检查 Token 是否在黑名单
        let blacklist_key = format!("token_blacklist:{}", token);
        let is_blacklisted: Option<i32> = redis::cmd("GET")
            .arg(&blacklist_key)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        if is_blacklisted.is_some() {
            return Err(AppError::Unauthorized("Token 已失效".to_string()));
        }

        // 3. 查找用户
        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or_else(|| AppError::NotFound("用户不存在".to_string()))?;

        Ok(CheckStatusData {
            user_id: user_id.to_string(),
            nickname: user.nickname,
            is_valid: true,
            expires_at: (Utc::now() + Duration::seconds(claims.exp - Utc::now().timestamp())).to_rfc3339(),
        })
    }

    /// 设置密码（临时用户升级或修改密码）
    pub async fn set_password(&self, user_id: i64, req: &SetPasswordRequest) -> Result<SetPasswordData> {
        // 1. 查找用户
        let _user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or_else(|| AppError::NotFound("用户不存在".to_string()))?;

        // 2. 对客户端哈希进行 bcrypt 二次哈希
        let password_hash = PasswordUtil::hash_password(&req.password_hash)?;

        // 3. 更新用户密码
        self.user_repo
            .update_password(user_id, password_hash.clone())
            .await?;

        Ok(SetPasswordData {
            can_login: true,
        })
    }

    /// 请求密码重置
    pub async fn request_password_reset(&self, req: &RequestPasswordResetRequest) -> Result<RequestPasswordResetData> {
        use uuid::Uuid;
        
        // 1. 查找用户
        let user = self
            .user_repo
            .find_by_login_id(&req.login_id)
            .await?
            .ok_or_else(|| AppError::NotFound("用户不存在".to_string()))?;

        // 2. 生成重置码（6位数字）
        use std::time::{SystemTime, UNIX_EPOCH};
        let nanos = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .subsec_nanos();
        let reset_code = format!("{:06}", nanos % 1000000);
        
        // 3. 生成重置ID
        let reset_id = Uuid::new_v4().to_string();

        // 4. 存储到 Redis（15分钟有效）
        let key = format!("password_reset:{}:{}", reset_id, user.user_id);
        let _: () = redis::cmd("SETEX")
            .arg(&key)
            .arg(900) // 15分钟
            .arg(&reset_code)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        // TODO: 发送验证码到用户邮箱/手机（根据 login_id 类型判断）
        tracing::info!("密码重置验证码: {} (reset_id: {})", reset_code, reset_id);

        Ok(RequestPasswordResetData {
            reset_id,
        })
    }

    /// 验证重置码
    pub async fn verify_reset_code(&self, req: &VerifyResetCodeRequest) -> Result<VerifyResetCodeData> {
        // 1. 从 Redis 获取验证码（需要遍历所有匹配的key）
        let pattern = format!("password_reset:{}:*", req.reset_id);
        let keys: Vec<String> = redis::cmd("KEYS")
            .arg(&pattern)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        if keys.is_empty() {
            return Err(AppError::BadRequest("重置请求不存在或已过期".to_string()));
        }

        let key = &keys[0];
        let stored_code: Option<String> = redis::cmd("GET")
            .arg(key)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        // 2. 验证验证码
        if stored_code.as_deref() != Some(&req.code) {
            return Err(AppError::BadRequest("验证码错误".to_string()));
        }

        // 3. 生成临时重置 Token（5分钟有效）
        let user_id = key.split(':').last()
            .and_then(|s| s.parse::<i64>().ok())
            .ok_or_else(|| AppError::Internal)?;

        let reset_token = JwtUtil::generate_token(user_id, 300, &self.jwt_secret)?;

        // 4. 存储重置 Token 到 Redis
        let token_key = format!("reset_token:{}", reset_token);
        let _: () = redis::cmd("SETEX")
            .arg(&token_key)
            .arg(300) // 5分钟
            .arg(user_id)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        // 5. 删除验证码
        let _: () = redis::cmd("DEL")
            .arg(key)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        Ok(VerifyResetCodeData {
            reset_token,
        })
    }

    /// 重置密码
    pub async fn reset_password(&self, req: &ResetPasswordRequest) -> Result<()> {
        // 1. 验证重置 Token
        let token_key = format!("reset_token:{}", req.reset_token);
        let user_id: Option<i64> = redis::cmd("GET")
            .arg(&token_key)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        let user_id = user_id.ok_or_else(|| {
            AppError::Unauthorized("重置Token无效或已过期".to_string())
        })?;

        // 2. 对客户端哈希进行 bcrypt 二次哈希
        let password_hash = PasswordUtil::hash_password(&req.password_hash)?;

        // 3. 更新密码
        self.user_repo
            .update_password(user_id, password_hash.clone())
            .await?;

        // 4. 删除重置 Token
        let _: () = redis::cmd("DEL")
            .arg(&token_key)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        // 5. 删除所有 Refresh Token（强制重新登录）
        let refresh_key = format!("refresh_token:{}", user_id);
        let _: () = redis::cmd("DEL")
            .arg(&refresh_key)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        Ok(())
    }

    /// 列出活跃会话
    /// current_session_id: 当前请求的会话ID（可选，用于标记当前会话）
    pub async fn list_sessions(
        &self,
        user_id: i64,
        req: &ListSessionsRequest,
        current_session_id: Option<Uuid>,
    ) -> Result<PaginatedData<SessionItem>> {
        let limit = Validator::validate_pagination(req.limit, 100)?;
        let cursor = req.cursor.as_ref().and_then(|c| Uuid::parse_str(c).ok());
        let sessions = self
            .session_repo
            .list_by_user(user_id, limit + 1, cursor)
            .await?;

        let has_more = sessions.len() > limit as usize;
        let sessions_to_return: Vec<_> = sessions.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            sessions_to_return.last().map(|s| s.session_id.to_string())
        } else {
            None
        };

        let items = sessions_to_return
            .into_iter()
            .map(|s| {
                let is_current = current_session_id.map(|id| id == s.session_id).unwrap_or(false);
                SessionItem {
                    id: s.session_id.to_string(),
                    device_type: s.device_type,
                    device_name: s.device_name,
                    ip_address: s.ip_address.map(|ip| ip.to_string()),
                    created_at: s.created_at.to_rfc3339(),
                    last_active_at: s.last_active_at.to_rfc3339(),
                    is_current,
                }
            })
            .collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 撤销会话
    pub async fn revoke_session(
        &self,
        user_id: i64,
        req: &RevokeSessionRequest,
    ) -> Result<()> {
        // 1. 解析会话ID
        let session_id = uuid::Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("会话ID格式错误".to_string()))?;

        // 2. 查找会话
        let session = self
            .session_repo
            .find_by_id(session_id)
            .await?
            .ok_or_else(|| AppError::NotFound("会话不存在".to_string()))?;

        // 3. 验证会话所有权
        if session.user_id != user_id {
            return Err(AppError::Forbidden("无权操作该会话".to_string()));
        }

        // 4. 删除会话
        self.session_repo.delete(session_id).await?;

        // 5. 将该会话的 token 加入黑名单（如果存在）
        // TODO: 需要在会话表中存储 token，或者使用其他方式标记

        Ok(())
    }
}
