//! 认证服务层实现
//!
//! 核心业务逻辑：注册、登录、会话管理

use chrono::{Duration, TimeZone, Utc};
use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use sqlx::types::ipnetwork::IpNetwork;
use sqlx::PgPool;
use std::net::IpAddr;
use uuid::Uuid;

use crate::constants::REDIS_KEY_SESSION;
use crate::dto::request::{
    ChangePasswordRequest, GuestRequest, LoginRequest, RegisterRequest,
    RequestPasswordResetRequest, ResetPasswordRequest, RevokeSessionRequest, SessionsRequest,
};
use crate::dto::response::{
    AuthData, CheckStatusData, PaginatedData, PaginationMeta, SessionItem,
};
use crate::error::{AppError, Result};
use crate::repositories::{
    session_repo::SessionRepository,
    user_repo::UserRepository, user_settings_repo::UserSettingsRepository,
    verification_repo::VerificationRepository,
};
use crate::types::enums::{ClientType, UserStatus, VerificationCodeType};
use crate::utils::cache_invalidation::{CacheInvalidator, InvalidationEvent};
use crate::utils::{
    jwt::JwtUtil, password::PasswordUtil, session_token::SessionTokenGenerator,
    token::TokenUtil, validation::Validator,
};

const REVOKE_TOKEN_TTL_SECS: usize = 300;

#[derive(Debug, Clone)]
pub struct AuthIssue {
    pub data: AuthData,
    pub refresh_token_raw: String,
}

#[derive(Clone)]
pub struct AuthService {
    db: PgPool,
    redis: ConnectionManager,
    jwt_secret: String,
    access_token_expiration: i64,
    refresh_token_expiration: i64,
    user_repo: UserRepository,
    session_repo: SessionRepository,
    settings_repo: UserSettingsRepository,
    verification_repo: VerificationRepository,
}

impl AuthService {
    pub fn new(
        db: &PgPool,
        redis: &ConnectionManager,
        jwt_secret: String,
        access_token_expiration: i64,
        refresh_token_expiration: i64,
        user_repo: UserRepository,
        session_repo: SessionRepository,
        settings_repo: UserSettingsRepository,
        verification_repo: VerificationRepository,
    ) -> Self {
        Self {
            db: db.clone(),
            redis: redis.clone(),
            jwt_secret,
            access_token_expiration,
            refresh_token_expiration,
            user_repo,
            session_repo,
            settings_repo,
            verification_repo,
        }
    }

    /// 用户注册
    pub async fn register(
        &self,
        req: &RegisterRequest,
        _client_type: ClientType,
        ip_address: Option<String>,
        user_agent: Option<&str>,
        device_fingerprint: Option<&str>,
    ) -> Result<AuthIssue> {
        if self
            .user_repo
            .find_by_identifier(&req.login_id)
            .await?
            .is_some()
        {
            return Err(AppError::AlreadyExists("该用户已存在".to_string()));
        }

        let password_hash = PasswordUtil::hash_password(&req.password)?;
        let user = self
            .user_repo
            .create(
                req.login_id.clone(),
                password_hash,
                Some(req.nickname.clone()),
                None,
                None,
            )
            .await?;

        self.settings_repo.create(user.user_id).await?;

        self.issue_tokens(
            user.user_id,
            user.nickname,
            ip_address,
            user_agent,
            device_fingerprint,
        )
        .await
    }

    /// 用户登录
    pub async fn login(
        &self,
        req: &LoginRequest,
        _client_type: ClientType,
        ip_address: Option<String>,
        user_agent: Option<&str>,
        device_fingerprint: Option<&str>,
    ) -> Result<AuthIssue> {
        let user = self
            .user_repo
            .find_by_identifier(&req.login_id)
            .await?
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

        if user.status == UserStatus::Banned {
            return Err(AppError::AccountBanned("账号已被封禁".to_string()));
        }

        if !PasswordUtil::verify_password(&req.password, &user.password_hash)? {
            return Err(AppError::PasswordError("密码错误".to_string()));
        }

        self.issue_tokens(
            user.user_id,
            user.nickname,
            ip_address,
            user_agent,
            device_fingerprint,
        )
        .await
    }

    /// 创建访客用户
    pub async fn create_guest(
        &self,
        req: &GuestRequest,
        _client_type: ClientType,
        ip_address: Option<String>,
        user_agent: Option<&str>,
        device_fingerprint: Option<&str>,
    ) -> Result<AuthIssue> {
        let guest_login_id = format!("guest_{}", Uuid::now_v7());
        let guest_password = TokenUtil::generate_token();
        let password_hash = PasswordUtil::hash_password(&guest_password)?;

        let user = self
            .user_repo
            .create(
                guest_login_id,
                password_hash,
                Some(req.nickname.clone()),
                None,
                None,
            )
            .await?;

        self.settings_repo.create(user.user_id).await?;

        self.issue_tokens(
            user.user_id,
            user.nickname,
            ip_address,
            user_agent,
            device_fingerprint,
        )
        .await
    }

    /// 刷新 token
    pub async fn refresh_token(
        &self,
        refresh_token_raw: &str,
        _client_type: ClientType,
        _ip_address: Option<String>,
        _user_agent: Option<&str>,
        _device_fingerprint: Option<&str>,
    ) -> Result<AuthIssue> {
        let refresh_token_hash = TokenUtil::hash_token(refresh_token_raw, &self.jwt_secret);

        if let Some(session) = self
            .session_repo
            .find_by_refresh_token(&refresh_token_hash)
            .await?
        {
            let user = self
                .user_repo
                .find_by_id(session.user_id)
                .await?
                .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

            let access_token = JwtUtil::generate_token(
                session.user_id,
                session.session_id,
                self.access_token_expiration,
                &self.jwt_secret,
            )?;
            let new_refresh_token_raw = TokenUtil::generate_refresh_token();
            let access_token_hash = TokenUtil::hash_token(&access_token, &self.jwt_secret);
            let new_refresh_token_hash =
                TokenUtil::hash_token(&new_refresh_token_raw, &self.jwt_secret);
            let refresh_expires_at =
                Utc::now() + Duration::seconds(self.refresh_token_expiration);

            let rotated = self
                .session_repo
                .rotate_tokens_if_refresh_matches(
                    session.session_id,
                    &refresh_token_hash,
                    access_token_hash,
                    new_refresh_token_hash,
                    refresh_expires_at,
                )
                .await?;

            if !rotated {
                return Err(AppError::RefreshTokenInvalid(
                    "refresh token 已过期".to_string(),
                ));
            }

            self.set_session_whitelist(session.session_id, session.user_id)
                .await?;

            return Ok(AuthIssue {
                data: AuthData {
                    user_id: user.user_id,
                    nickname: user.nickname.unwrap_or_default(),
                    access_token,
                    refresh_token: Some(new_refresh_token_raw.clone()),
                    access_token_expires_at: (Utc::now()
                        + Duration::seconds(self.access_token_expiration))
                    .to_rfc3339(),
                    refresh_token_expires_at: Some(refresh_expires_at.to_rfc3339()),
                },
                refresh_token_raw: new_refresh_token_raw,
            });
        }

        if let Some(session) = self
            .session_repo
            .find_by_previous_refresh_token(&refresh_token_hash)
            .await?
        {
            self.revoke_session_by_id(session.user_id, session.session_id)
                .await?;
            return Err(AppError::ReLoginRequired(
                "refresh token 已失效，请重新登录".to_string(),
            ));
        }

        Err(AppError::RefreshTokenInvalid(
            "refresh token 无效或已过期".to_string(),
        ))
    }

    /// 登出
    pub async fn logout(&self, user_id: i64, access_token: &str) -> Result<()> {
        let claims = JwtUtil::verify_token(access_token, &self.jwt_secret)?;
        let token_user_id = claims
            .sub
            .parse::<i64>()
            .map_err(|_| AppError::Unauthorized("无效的用户ID".to_string()))?;
        if token_user_id != user_id {
            return Err(AppError::Forbidden("无权限登出该会话".to_string()));
        }

        let session_id = Uuid::parse_str(&claims.sid)
            .map_err(|_| AppError::Unauthorized("无效的会话".to_string()))?;
        self.revoke_session_by_id(user_id, session_id).await
    }

    /// 检查认证状态
    pub async fn check_status(
        &self,
        user_id: Option<i64>,
        token: Option<&str>,
    ) -> Result<CheckStatusData> {
        if let Some(user_id) = user_id {
            let nickname = self
                .user_repo
                .find_by_id(user_id)
                .await?
                .and_then(|u| u.nickname);

            let expires_at = token
                .and_then(|raw| JwtUtil::verify_token(raw, &self.jwt_secret).ok())
                .and_then(|claims| Utc.timestamp_opt(claims.exp, 0).single())
                .map(|dt| dt.to_rfc3339());

            return Ok(CheckStatusData {
                is_valid: true,
                user_id: Some(user_id),
                nickname,
                expires_at,
            });
        }

        Ok(CheckStatusData {
            is_valid: false,
            user_id: None,
            nickname: None,
            expires_at: None,
        })
    }

    /// 请求密码重置（发送验证码）
    /// 安全说明：无论 login_id 是否存在，均返回成功响应，不暴露账号是否注册
    pub async fn request_password_reset(&self, req: &RequestPasswordResetRequest) -> Result<()> {
        // 无论用户是否存在，都生成验证码流程但仅对已存在用户实际存储
        let user = self.user_repo.find_by_identifier(&req.login_id).await?;
        if user.is_none() {
            // 不暴露用户是否存在，静默返回成功
            tracing::info!(login_id = %req.login_id, "Password reset requested for non-existent user, returning success silently");
            return Ok(());
        }

        // 生成 6 位数字验证码
        let code = TokenUtil::generate_numeric_code(6);
        let expires_at = Utc::now() + Duration::minutes(10);

        self.verification_repo
            .create_code(
                &req.login_id,
                &code,
                VerificationCodeType::ResetPassword.as_str(),
                expires_at,
            )
            .await
            .map_err(|e| AppError::internal(e.to_string()))?;

        // TODO: 实际发送验证码（邮件/短信），当前仅日志输出
        tracing::info!(login_id = %req.login_id, code = %code, "Password reset verification code generated (NOT sent - need email/SMS integration)");

        Ok(())
    }

    /// 验证验证码并重置密码
    /// 重置成功后自动登录并返回 Token，同时吊销该账号所有现有会话
    pub async fn reset_password(
        &self,
        req: &ResetPasswordRequest,
        _client_type: ClientType,
        ip_address: Option<String>,
        user_agent: Option<&str>,
        device_fingerprint: Option<&str>,
    ) -> Result<AuthIssue> {
        // 1. 验证码校验
        let verification = self
            .verification_repo
            .find_valid(
                &req.login_id,
                &req.verification_code,
                VerificationCodeType::ResetPassword.as_str(),
            )
            .await
            .map_err(|e| AppError::internal(e.to_string()))?
            .ok_or_else(|| {
                AppError::VerifyTokenInvalid("验证码无效或已过期".to_string())
            })?;

        // 2. 标记验证码已使用
        self.verification_repo
            .mark_used(verification.code_id)
            .await
            .map_err(|e| AppError::internal(e.to_string()))?;

        // 3. 查找用户
        let user = self
            .user_repo
            .find_by_identifier(&req.login_id)
            .await?
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

        // 4. 更新密码
        let password_hash = PasswordUtil::hash_password(&req.new_password)?;
        self.user_repo
            .update_password(user.user_id, &password_hash)
            .await?;

        // 5. 吊销该用户所有现有会话（与 change_password 行为一致）
        let session_ids = self.session_repo.revoke_all_by_user(user.user_id).await?;
        for session_id in session_ids {
            self.clear_session_whitelist(session_id).await?;
            self.publish_session_revoked(user.user_id, session_id)
                .await?;
        }

        // 6. 自动登录：颁发新 token
        self.issue_tokens(
            user.user_id,
            user.nickname,
            ip_address,
            user_agent,
            device_fingerprint,
        )
        .await
    }

    /// 修改密码
    pub async fn change_password(
        &self,
        user_id: i64,
        req: &ChangePasswordRequest,
    ) -> Result<()> {
        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

        if !PasswordUtil::verify_password(&req.old_password, &user.password_hash)? {
            return Err(AppError::PasswordError("原密码错误".to_string()));
        }

        let password_hash = PasswordUtil::hash_password(&req.new_password)?;
        self.user_repo.update_password(user_id, &password_hash).await?;
        Ok(())
    }

    /// 获取会话列表
    pub async fn list_sessions(
        &self,
        user_id: i64,
        req: &SessionsRequest,
        current_session_id: Option<Uuid>,
    ) -> Result<PaginatedData<SessionItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;

        let (rows, next_cursor) =
            self.session_repo.list_by_user(user_id, limit, cursor).await?;

        let token_generator = SessionTokenGenerator::new(&self.jwt_secret);
        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            let revoke_token = token_generator.generate_revoke_token(row.session_id, user_id);
            self.store_revoke_token(&revoke_token, user_id, row.session_id)
                .await?;

            let ip_address = row.ip_address.as_ref().and_then(mask_ip_address);
            let is_current = current_session_id
                .map(|current| current == row.session_id)
                .unwrap_or(false);

            items.push(SessionItem {
                display_id: token_generator.generate_display_id(row.session_id),
                revoke_token,
                device_type: row.device_type,
                device_name: row.device_name,
                ip_address,
                ip_location: None,
                created_at: row.created_at.to_rfc3339(),
                last_active_at: row.last_active_at.to_rfc3339(),
                is_current,
            });
        }

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
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
        let token_generator = SessionTokenGenerator::new(&self.jwt_secret);
        if !token_generator.verify_revoke_token_format(&req.revoke_token) {
            return Err(AppError::NotFound("撤销令牌无效".to_string()));
        }

        let key = revoke_token_key(&req.revoke_token);
        let mut conn = self.redis.clone();
        let value: Option<String> = conn
            .get(&key)
            .await
            .map_err(|e| AppError::internal(e.to_string()))?;
        let value = value.ok_or_else(|| AppError::NotFound("撤销令牌无效".to_string()))?;
        let _ = conn
            .del::<_, ()>(&key)
            .await
            .map_err(|e| AppError::internal(e.to_string()))?;

        let mut parts = value.split(':');
        let stored_user_id = parts
            .next()
            .and_then(|v| v.parse::<i64>().ok())
            .ok_or_else(|| AppError::NotFound("撤销令牌无效".to_string()))?;
        let stored_session_id = parts
            .next()
            .ok_or_else(|| AppError::NotFound("撤销令牌无效".to_string()))?;
        let session_id = Uuid::parse_str(stored_session_id)
            .map_err(|_| AppError::NotFound("撤销令牌无效".to_string()))?;

        if stored_user_id != user_id {
            return Err(AppError::Forbidden("无权限撤销该会话".to_string()));
        }

        self.revoke_session_by_id(user_id, session_id).await
    }

    async fn issue_tokens(
        &self,
        user_id: i64,
        nickname: Option<String>,
        ip_address: Option<String>,
        user_agent: Option<&str>,
        device_fingerprint: Option<&str>,
    ) -> Result<AuthIssue> {
        let session_id = Uuid::now_v7();
        let access_token = JwtUtil::generate_token(
            user_id,
            session_id,
            self.access_token_expiration,
            &self.jwt_secret,
        )?;
        let refresh_token_raw = TokenUtil::generate_refresh_token();
        let access_token_hash = TokenUtil::hash_token(&access_token, &self.jwt_secret);
        let refresh_token_hash = TokenUtil::hash_token(&refresh_token_raw, &self.jwt_secret);
        let refresh_expires_at = Utc::now() + Duration::seconds(self.refresh_token_expiration);

        let ip_network = ip_address.and_then(|ip| ip.parse::<IpNetwork>().ok());
        self.session_repo
            .create(
                user_id,
                session_id,
                None,
                None,
                device_fingerprint.map(|s| s.to_string()),
                ip_network,
                user_agent.map(|s| s.to_string()),
                access_token_hash,
                refresh_token_hash,
                refresh_expires_at,
            )
            .await?;

        self.set_session_whitelist(session_id, user_id).await?;

        Ok(AuthIssue {
            data: AuthData {
                user_id,
                nickname: nickname.unwrap_or_default(),
                access_token,
                refresh_token: Some(refresh_token_raw.clone()),
                access_token_expires_at:
                    (Utc::now() + Duration::seconds(self.access_token_expiration)).to_rfc3339(),
                refresh_token_expires_at: Some(refresh_expires_at.to_rfc3339()),
            },
            refresh_token_raw,
        })
    }

    async fn revoke_session_by_id(&self, user_id: i64, session_id: Uuid) -> Result<()> {
        self.session_repo.revoke(session_id).await?;
        self.clear_session_whitelist(session_id).await?;
        self.publish_session_revoked(user_id, session_id).await?;
        Ok(())
    }

    pub async fn revoke_all_sessions(&self, user_id: i64) -> Result<()> {
        let session_ids = self.session_repo.revoke_all_by_user(user_id).await?;
        for session_id in session_ids {
            self.clear_session_whitelist(session_id).await?;
            self.publish_session_revoked(user_id, session_id).await?;
        }
        Ok(())
    }

    async fn set_session_whitelist(&self, session_id: Uuid, user_id: i64) -> Result<()> {
        let key = session_key(session_id);
        let mut conn = self.redis.clone();
        conn.set_ex::<_, _, ()>(
            key,
            user_id.to_string(),
            self.refresh_token_expiration as u64,
        )
            .await
            .map_err(|e| AppError::internal(e.to_string()))?;
        Ok(())
    }

    async fn clear_session_whitelist(&self, session_id: Uuid) -> Result<()> {
        let key = session_key(session_id);
        let mut conn = self.redis.clone();
        conn.del::<_, ()>(&key)
            .await
            .map_err(|e| AppError::internal(e.to_string()))?;
        Ok(())
    }

    async fn publish_session_revoked(&self, user_id: i64, session_id: Uuid) -> Result<()> {
        let invalidator = CacheInvalidator::new(self.redis.clone());
        invalidator
            .publish(InvalidationEvent::SessionRevoked {
                user_id,
                session_id: session_id.to_string(),
            })
            .await
            .map_err(|e| AppError::internal(e.to_string()))?;
        Ok(())
    }

    async fn store_revoke_token(
        &self,
        token: &str,
        user_id: i64,
        session_id: Uuid,
    ) -> Result<()> {
        let key = revoke_token_key(token);
        let value = format!("{}:{}", user_id, session_id);
        let mut conn = self.redis.clone();
        conn.set_ex::<_, _, ()>(key, value, REVOKE_TOKEN_TTL_SECS as u64)
            .await
            .map_err(|e| AppError::internal(e.to_string()))?;
        Ok(())
    }
}

fn session_key(session_id: Uuid) -> String {
    format!("{}:{}", REDIS_KEY_SESSION, session_id)
}

fn revoke_token_key(token: &str) -> String {
    format!("revoke_token:{}", token)
}

fn mask_ip_address(ip: &IpNetwork) -> Option<String> {
    match ip.ip() {
        IpAddr::V4(addr) => {
            let octets = addr.octets();
            Some(format!("{}.{}.xxx.xxx", octets[0], octets[1]))
        }
        IpAddr::V6(_) => None,
    }
}
