//! 认证服务层实现
//!
//! 核心业务逻辑：注册、登录、会话管理

use chrono::{Duration, TimeZone, Utc};
use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use serde::Serialize;
use sqlx::types::ipnetwork::IpNetwork;
use std::net::IpAddr;
use std::time::Duration as StdDuration;
use uuid::Uuid;

use crate::constants::REDIS_KEY_SESSION;
use crate::dto::request::{
    ChangePasswordRequest, GuestRequest, LoginRequest, RegisterRequest,
    RequestPasswordResetRequest, ResetPasswordRequest, RevokeSessionRequest, SessionsRequest,
};
use crate::dto::response::{AuthData, CheckStatusData, SessionItem, SessionsListData};
use crate::error::{AppError, Result};
use crate::repositories::{
    session_repo::SessionRepository, user::settings_repo::UserSettingsRepository,
    user_repo::UserRepository, verification_repo::VerificationRepository,
};
use crate::types::enums::{ClientType, UserStatus, VerificationCodeType};
use crate::utils::cache_invalidation::{CacheInvalidator, InvalidationEvent};
use crate::utils::{
    bloom_filter::ResourceExistenceChecker, jwt::JwtUtil, password::PasswordUtil, token::TokenUtil,
    validation::Validator,
};

fn detect_delivery_channel(login_id: &str) -> &'static str {
    if login_id.contains('@') {
        "email"
    } else {
        "phone"
    }
}

#[derive(Serialize)]
struct ResetCodeDeliveryPayload<'a> {
    scene: &'static str,
    login_id: &'a str,
    client_type: ClientType,
    channel: &'static str,
    verification_code: &'a str,
    expires_at: &'a str,
}

#[derive(Debug, Clone)]
pub struct AuthIssue {
    pub data: AuthData,
    pub refresh_token_raw: String,
}

#[derive(Clone)]
pub struct AuthService {
    redis: ConnectionManager,
    jwt_secret: String,
    access_token_expiration: i64,
    refresh_token_expiration: i64,
    user_repo: UserRepository,
    session_repo: SessionRepository,
    settings_repo: UserSettingsRepository,
    verification_repo: VerificationRepository,
    reset_delivery_webhook_url: Option<String>,
    reset_delivery_webhook_token: Option<String>,
    reset_delivery_timeout_ms: u64,
    http_client: reqwest::Client,
}

impl AuthService {
    pub fn new(
        redis: &ConnectionManager,
        jwt_secret: String,
        access_token_expiration: i64,
        refresh_token_expiration: i64,
        user_repo: UserRepository,
        session_repo: SessionRepository,
        settings_repo: UserSettingsRepository,
        verification_repo: VerificationRepository,
        reset_delivery_webhook_url: Option<String>,
        reset_delivery_webhook_token: Option<String>,
        reset_delivery_timeout_ms: u64,
    ) -> Self {
        Self {
            redis: redis.clone(),
            jwt_secret,
            access_token_expiration,
            refresh_token_expiration,
            user_repo,
            session_repo,
            settings_repo,
            verification_repo,
            reset_delivery_webhook_url,
            reset_delivery_webhook_token,
            reset_delivery_timeout_ms,
            http_client: reqwest::Client::new(),
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
        let started_at = std::time::Instant::now();
        tracing::info!(
            login_id_len = req.login_id.chars().count(),
            nickname_len = req.nickname.chars().count(),
            has_ip = ip_address.is_some(),
            has_user_agent = user_agent.is_some(),
            has_device_fingerprint = device_fingerprint.is_some(),
            "auth.register started"
        );
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

        if let Err(e) = ResourceExistenceChecker::new(self.redis.clone())
            .record_user(user.user_id)
            .await
        {
            tracing::warn!(user_id = user.user_id, error = %e, "record_user bloom failed");
        }

        self.settings_repo.create(user.user_id).await?;

        let issue = self
            .issue_tokens(user.user_id, ip_address, user_agent, device_fingerprint)
            .await?;
        tracing::info!(
            user_id = user.user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "auth.register completed"
        );
        Ok(issue)
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
        let started_at = std::time::Instant::now();
        tracing::info!(
            login_id_len = req.login_id.chars().count(),
            has_ip = ip_address.is_some(),
            has_user_agent = user_agent.is_some(),
            has_device_fingerprint = device_fingerprint.is_some(),
            "auth.login started"
        );
        // 安全：用户不存在和密码错误统一返回 10100，防止账号枚举攻击
        // 真实原因仅记录在服务端日志中，不暴露给客户端
        let user = match self.user_repo.find_by_identifier(&req.login_id).await? {
            Some(user) => user,
            None => {
                tracing::warn!(
                    login_id = %req.login_id,
                    reason = "user_not_found",
                    "登录失败：用户不存在（客户端收到通用认证错误）"
                );
                return Err(AppError::Unauthorized("用户名或密码错误".to_string()));
            }
        };

        if user.status == UserStatus::Banned {
            return Err(AppError::AccountBanned("账号已被封禁".to_string()));
        }

        let stored_hash = user.password_hash.as_deref().ok_or_else(|| {
            tracing::warn!(login_id = %req.login_id, reason = "password_missing");
            AppError::Unauthorized("用户名或密码错误".to_string())
        })?;

        if !PasswordUtil::verify_password(&req.password, stored_hash)? {
            tracing::warn!(
                login_id = %req.login_id,
                user_id = user.user_id,
                reason = "wrong_password",
                "登录失败：密码错误（客户端收到通用认证错误）"
            );
            return Err(AppError::Unauthorized("用户名或密码错误".to_string()));
        }

        let issue = self
            .issue_tokens(user.user_id, ip_address, user_agent, device_fingerprint)
            .await?;
        tracing::info!(
            user_id = user.user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "auth.login completed"
        );
        Ok(issue)
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
        let started_at = std::time::Instant::now();
        tracing::info!(
            nickname_len = req.nickname.chars().count(),
            has_ip = ip_address.is_some(),
            has_user_agent = user_agent.is_some(),
            has_device_fingerprint = device_fingerprint.is_some(),
            "auth.create_guest started"
        );
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

        if let Err(e) = ResourceExistenceChecker::new(self.redis.clone())
            .record_user(user.user_id)
            .await
        {
            tracing::warn!(user_id = user.user_id, error = %e, "record_user bloom failed");
        }

        self.settings_repo.create(user.user_id).await?;

        let issue = self
            .issue_tokens(user.user_id, ip_address, user_agent, device_fingerprint)
            .await?;
        tracing::info!(
            user_id = user.user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "auth.create_guest completed"
        );
        Ok(issue)
    }

    /// 刷新 token
    ///
    /// ## 宽限期机制（Spec: api_spec_auth.md）
    ///
    /// 旧 refresh_token 在被轮换后，仍有 30 秒宽限期可用。
    /// 宽限期内使用旧 refresh_token 刷新，服务端返回与首次刷新相同的新 token 对（不生成新的）。
    /// 宽限期过后，旧 refresh_token 彻底失效。
    /// 设计目的：解决多标签页/多并发请求同时触发刷新的竞态问题。
    pub async fn refresh_token(
        &self,
        refresh_token_raw: &str,
        _client_type: ClientType,
        _ip_address: Option<String>,
        _user_agent: Option<&str>,
        _device_fingerprint: Option<&str>,
    ) -> Result<AuthIssue> {
        let started_at = std::time::Instant::now();
        tracing::info!(
            refresh_token_len = refresh_token_raw.len(),
            "auth.refresh_token started"
        );
        let refresh_token_hash = TokenUtil::hash_token(refresh_token_raw, &self.jwt_secret);

        // 1. 当前 refresh_token 有效 → 轮换并返回新 token
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
            let refresh_expires_at = Utc::now() + Duration::seconds(self.refresh_token_expiration);

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

            let auth_issue = AuthIssue {
                data: AuthData {
                    user_id: user.user_id,
                    access_token,
                    refresh_token: Some(new_refresh_token_raw.clone()),
                    access_token_expires_at: (Utc::now()
                        + Duration::seconds(self.access_token_expiration))
                    .to_rfc3339(),
                    refresh_token_expires_at: Some(refresh_expires_at.to_rfc3339()),
                },
                refresh_token_raw: new_refresh_token_raw,
            };

            // 缓存新 token 对到 Redis，供宽限期内的并发请求使用（30 秒 TTL）
            self.cache_grace_period_tokens(&refresh_token_hash, &auth_issue)
                .await;
            tracing::info!(
                user_id = session.user_id,
                session_id = %session.session_id,
                mode = "rotate",
                elapsed_ms = started_at.elapsed().as_millis(),
                "auth.refresh_token completed"
            );

            return Ok(auth_issue);
        }

        // 2. 匹配 previous_refresh_token → 检查宽限期
        if let Some(_session) = self
            .session_repo
            .find_by_previous_refresh_token(&refresh_token_hash)
            .await?
        {
            // 尝试从宽限期缓存获取新 token 对
            if let Some(cached_issue) = self.get_grace_period_tokens(&refresh_token_hash).await {
                tracing::debug!(
                    "Grace period hit: returning cached tokens for rotated refresh_token"
                );
                tracing::info!(
                    mode = "grace_cache_hit",
                    elapsed_ms = started_at.elapsed().as_millis(),
                    "auth.refresh_token completed"
                );
                return Ok(cached_issue);
            }

            // 宽限期已过，旧 refresh_token 彻底失效（不撤销会话，只拒绝刷新）
            return Err(AppError::RefreshTokenInvalid(
                "refresh token 已被轮换且宽限期已过".to_string(),
            ));
        }

        // 3. 均不匹配 → 无效或已过期
        tracing::warn!(
            mode = "invalid",
            elapsed_ms = started_at.elapsed().as_millis(),
            "auth.refresh_token rejected"
        );
        Err(AppError::RefreshTokenInvalid(
            "refresh token 无效或已过期".to_string(),
        ))
    }

    /// 将新 token 对缓存到 Redis（宽限期 30 秒）
    async fn cache_grace_period_tokens(
        &self,
        old_refresh_token_hash: &str,
        auth_issue: &AuthIssue,
    ) {
        let cache_key = format!("grace:refresh:{}", old_refresh_token_hash);
        let cache_value = serde_json::json!({
            "user_id": auth_issue.data.user_id,
            "access_token": auth_issue.data.access_token,
            "refresh_token_raw": auth_issue.refresh_token_raw,
            "access_token_expires_at": auth_issue.data.access_token_expires_at,
            "refresh_token_expires_at": auth_issue.data.refresh_token_expires_at,
        });

        let result: redis::RedisResult<()> = redis::cmd("SET")
            .arg(&cache_key)
            .arg(cache_value.to_string())
            .arg("EX")
            .arg(30_u64)
            .query_async(&mut self.redis.clone())
            .await;

        if let Err(e) = result {
            tracing::warn!(error = %e, "Failed to cache grace period tokens");
        }
    }

    /// 从 Redis 获取宽限期缓存的 token 对
    async fn get_grace_period_tokens(&self, old_refresh_token_hash: &str) -> Option<AuthIssue> {
        let cache_key = format!("grace:refresh:{}", old_refresh_token_hash);
        let cached: redis::RedisResult<Option<String>> = self.redis.clone().get(&cache_key).await;

        match cached {
            Ok(Some(json_str)) => {
                let value: serde_json::Value = serde_json::from_str(&json_str).ok()?;
                Some(AuthIssue {
                    data: AuthData {
                        user_id: value["user_id"].as_i64()?,
                        access_token: value["access_token"].as_str()?.to_string(),
                        refresh_token: Some(value["refresh_token_raw"].as_str()?.to_string()),
                        access_token_expires_at: value["access_token_expires_at"]
                            .as_str()?
                            .to_string(),
                        refresh_token_expires_at: value["refresh_token_expires_at"]
                            .as_str()
                            .map(|s| s.to_string()),
                    },
                    refresh_token_raw: value["refresh_token_raw"].as_str()?.to_string(),
                })
            }
            _ => None,
        }
    }

    /// 登出
    pub async fn logout(&self, user_id: i64, access_token: &str) -> Result<()> {
        let started_at = std::time::Instant::now();
        tracing::info!(user_id, "auth.logout started");
        let claims = JwtUtil::verify_token(access_token, &self.jwt_secret)?;
        let token_user_id = Validator::parse_id(&claims.sub, "用户ID")?;
        if token_user_id != user_id {
            return Err(AppError::Forbidden("无权限登出该会话".to_string()));
        }

        let session_id = Validator::parse_uuid(&claims.sid, "会话")?;
        self.revoke_session_by_id(user_id, session_id).await?;
        tracing::info!(
            user_id,
            session_id = %session_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "auth.logout completed"
        );
        Ok(())
    }

    /// 检查认证状态
    pub async fn check_status(
        &self,
        user_id: Option<i64>,
        token: Option<&str>,
    ) -> Result<CheckStatusData> {
        if let Some(_user_id) = user_id {
            let expires_at = token
                .and_then(|raw| JwtUtil::verify_token(raw, &self.jwt_secret).ok())
                .and_then(|claims| Utc.timestamp_opt(claims.exp, 0).single())
                .map(|dt| dt.to_rfc3339());

            return Ok(CheckStatusData {
                is_valid: true,
                expires_at,
            });
        }

        Ok(CheckStatusData {
            is_valid: false,
            expires_at: None,
        })
    }

    /// 请求密码重置（发送验证码）
    /// 安全说明：无论 login_id 是否存在，均返回成功响应，不暴露账号是否注册
    pub async fn request_password_reset(&self, req: &RequestPasswordResetRequest) -> Result<()> {
        tracing::info!(
            client_type = ?req.client_type,
            login_id = %req.login_id,
            "Password reset requested"
        );

        // 无论用户是否存在，都返回成功，不暴露账号状态。
        if self
            .user_repo
            .find_by_identifier(&req.login_id)
            .await?
            .is_none()
        {
            // 不暴露用户是否存在，静默返回成功
            return Ok(());
        }

        // 生成 6 位数字验证码
        let code = TokenUtil::generate_numeric_code(6);
        let expires_at = Utc::now() + Duration::minutes(10);

        self.verification_repo
            .create_code(
                &req.login_id,
                &code,
                VerificationCodeType::ResetPassword,
                expires_at,
            )
            .await
            .map_err(|e| AppError::internal(e.to_string()))?;

        // 发送失败不影响统一成功响应，避免通过错误差异泄露账号存在性。
        if let Err(err) = self
            .deliver_reset_verification_code(req, &code, expires_at)
            .await
        {
            tracing::error!(
                login_id = %req.login_id,
                error = %err,
                "Password reset verification code delivery failed"
            );
        }

        Ok(())
    }

    async fn deliver_reset_verification_code(
        &self,
        req: &RequestPasswordResetRequest,
        code: &str,
        expires_at: chrono::DateTime<Utc>,
    ) -> Result<()> {
        let Some(webhook_url) = self.reset_delivery_webhook_url.as_deref() else {
            tracing::warn!(
                "Password reset delivery webhook is not configured; skip external delivery"
            );
            return Ok(());
        };

        let expires_at_rfc3339 = expires_at.to_rfc3339();
        let payload = ResetCodeDeliveryPayload {
            scene: "reset_password",
            login_id: &req.login_id,
            client_type: req.client_type,
            channel: detect_delivery_channel(&req.login_id),
            verification_code: code,
            expires_at: &expires_at_rfc3339,
        };

        let mut request_builder = self
            .http_client
            .post(webhook_url)
            .timeout(StdDuration::from_millis(self.reset_delivery_timeout_ms))
            .json(&payload);
        if let Some(token) = self.reset_delivery_webhook_token.as_deref() {
            request_builder = request_builder.bearer_auth(token);
        }

        let response = request_builder
            .send()
            .await
            .map_err(|e| AppError::internal(format!("验证码发送请求失败: {}", e)))?;
        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_default();
            return Err(AppError::internal(format!(
                "验证码发送返回异常状态: status={}, body={}",
                status, body
            )));
        }

        tracing::info!(
            login_id = %req.login_id,
            channel = detect_delivery_channel(&req.login_id),
            "Password reset verification code delivered"
        );
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
        let started_at = std::time::Instant::now();
        tracing::info!(
            login_id_len = req.login_id.chars().count(),
            has_ip = ip_address.is_some(),
            has_user_agent = user_agent.is_some(),
            has_device_fingerprint = device_fingerprint.is_some(),
            "auth.reset_password started"
        );
        // 1. 验证码校验
        let verification = self
            .verification_repo
            .find_valid(
                &req.login_id,
                &req.verification_code,
                VerificationCodeType::ResetPassword,
            )
            .await
            .map_err(|e| AppError::internal(e.to_string()))?
            .ok_or_else(|| AppError::VerifyTokenInvalid("验证码无效或已过期".to_string()))?;

        // 2. 标记验证码已使用
        self.verification_repo
            .mark_used(verification.code_id)
            .await
            .map_err(|e| AppError::internal(e.to_string()))?;

        // 3. 查找用户（安全：不暴露账号存在性，统一返回 10100）
        // 真实原因仅记录在服务端日志中，不暴露给客户端
        let user = match self.user_repo.find_by_identifier(&req.login_id).await? {
            Some(user) => user,
            None => {
                tracing::warn!(
                    login_id = %req.login_id,
                    reason = "user_not_found_after_valid_code",
                    "重置密码失败：验证码有效但用户不存在（客户端收到通用验证码错误）"
                );
                return Err(AppError::VerifyTokenInvalid(
                    "验证码无效或已过期".to_string(),
                ));
            }
        };

        // 4. 更新密码
        let password_hash = PasswordUtil::hash_password(&req.new_password)?;
        self.user_repo
            .update_password(user.user_id, &password_hash)
            .await?;

        // 5. 吊销该用户所有现有会话（与 change_password 行为一致）
        let session_ids = self.session_repo.revoke_all_by_user(user.user_id).await?;
        let revoked_count = session_ids.len();
        for session_id in session_ids {
            self.clear_session_whitelist(session_id).await?;
            self.publish_session_revoked(user.user_id, session_id)
                .await?;
        }

        // 6. 自动登录：颁发新 token
        let issue = self
            .issue_tokens(user.user_id, ip_address, user_agent, device_fingerprint)
            .await?;
        tracing::info!(
            user_id = user.user_id,
            revoked_sessions = revoked_count,
            elapsed_ms = started_at.elapsed().as_millis(),
            "auth.reset_password completed"
        );
        Ok(issue)
    }

    /// 修改密码
    /// 成功后吊销该用户除当前会话外的所有会话，当前会话保持有效
    pub async fn change_password(
        &self,
        user_id: i64,
        current_session_id: Uuid,
        req: &ChangePasswordRequest,
    ) -> Result<()> {
        let started_at = std::time::Instant::now();
        tracing::info!(user_id, "auth.change_password started");
        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

        let stored_hash = user
            .password_hash
            .as_deref()
            .ok_or_else(|| AppError::PasswordError("原密码错误".to_string()))?;

        if !PasswordUtil::verify_password(&req.old_password, stored_hash)? {
            return Err(AppError::PasswordError("原密码错误".to_string()));
        }

        let password_hash = PasswordUtil::hash_password(&req.new_password)?;
        self.user_repo
            .update_password(user_id, &password_hash)
            .await?;

        // 吊销其他会话，当前会话保持有效
        let session_ids = self
            .session_repo
            .revoke_all_except(user_id, current_session_id)
            .await?;
        let revoked_count = session_ids.len();
        for session_id in session_ids {
            self.clear_session_whitelist(session_id).await?;
            self.publish_session_revoked(user_id, session_id).await?;
        }
        tracing::info!(
            user_id,
            revoked_sessions = revoked_count,
            elapsed_ms = started_at.elapsed().as_millis(),
            "auth.change_password completed"
        );

        Ok(())
    }

    /// 获取会话列表（直接返回全部活跃会话，无需分页）
    pub async fn list_sessions(
        &self,
        user_id: i64,
        _req: &SessionsRequest,
        current_session_id: Option<Uuid>,
    ) -> Result<SessionsListData> {
        let rows = self.session_repo.list_all_by_user(user_id).await?;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            let ip_address = row.ip_address.as_ref().and_then(mask_ip_address);
            let is_current = current_session_id
                .map(|current| current == row.session_id)
                .unwrap_or(false);

            items.push(SessionItem {
                session_id: row.session_id.to_string(),
                device_type: row.device_type,
                device_name: row.device_name,
                ip_address,
                ip_location: None,
                created_at: row.created_at.to_rfc3339(),
                last_active_at: row.last_active_at.to_rfc3339(),
                is_current,
            });
        }

        Ok(SessionsListData { items })
    }

    /// 撤销会话
    /// 禁止撤销当前会话
    pub async fn revoke_session(
        &self,
        user_id: i64,
        current_session_id: Uuid,
        req: &RevokeSessionRequest,
    ) -> Result<()> {
        let started_at = std::time::Instant::now();
        let session_id = Validator::parse_uuid(&req.session_id, "会话标识")?;
        tracing::info!(
            user_id,
            current_session_id = %current_session_id,
            target_session_id = %session_id,
            "auth.revoke_session started"
        );

        if session_id == current_session_id {
            tracing::warn!(
                user_id,
                session_id = %session_id,
                "auth.revoke_session rejected: cannot revoke current session"
            );
            return Err(AppError::BusinessRuleViolation(
                "不能撤销当前会话，如需登出请使用登出接口".to_string(),
            ));
        }

        let revoked = self
            .session_repo
            .revoke_if_owned_by(user_id, session_id)
            .await?;
        if !revoked {
            tracing::warn!(
                user_id,
                session_id = %session_id,
                "auth.revoke_session rejected: not found or not owned"
            );
            return Err(AppError::NotFound("会话不存在或无权限撤销".to_string()));
        }

        self.clear_session_whitelist(session_id).await?;
        self.publish_session_revoked(user_id, session_id).await?;
        tracing::info!(
            user_id,
            session_id = %session_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "auth.revoke_session completed"
        );
        Ok(())
    }

    async fn issue_tokens(
        &self,
        user_id: i64,
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
                access_token,
                refresh_token: Some(refresh_token_raw.clone()),
                access_token_expires_at: (Utc::now()
                    + Duration::seconds(self.access_token_expiration))
                .to_rfc3339(),
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

    pub async fn revoke_all_sessions(&self, user_id: i64, current_session_id: Uuid) -> Result<i64> {
        let started_at = std::time::Instant::now();
        tracing::info!(
            user_id,
            current_session_id = %current_session_id,
            "auth.revoke_all_sessions started"
        );
        let session_ids = self
            .session_repo
            .revoke_all_except(user_id, current_session_id)
            .await?;
        let revoked_count = session_ids.len() as i64;
        for session_id in session_ids {
            self.clear_session_whitelist(session_id).await?;
            self.publish_session_revoked(user_id, session_id).await?;
        }
        tracing::info!(
            user_id,
            revoked_count,
            elapsed_ms = started_at.elapsed().as_millis(),
            "auth.revoke_all_sessions completed"
        );
        Ok(revoked_count)
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
}

#[cfg(test)]
mod tests {
    #[test]
    fn detect_delivery_channel_should_distinguish_email_and_phone() {
        assert_eq!(super::detect_delivery_channel("user@example.com"), "email");
        assert_eq!(super::detect_delivery_channel("13800138000"), "phone");
    }
}

fn session_key(session_id: Uuid) -> String {
    format!("{}:{}", REDIS_KEY_SESSION, session_id)
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
