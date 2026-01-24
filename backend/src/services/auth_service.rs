use chrono::{Duration, Utc};
use redis::aio::ConnectionManager;
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            ChangePasswordRequest, GuestRequest, LoginRequest, RegisterRequest,
            RequestPasswordResetRequest, ResetPasswordRequest, RevokeSessionRequest,
            SendVerificationCodeRequest, SessionsRequest, VerifyCodeRequest,
            VerifyResetCodeRequest,
        },
        response::{
            AuthData, CheckStatusData, PaginatedData, PaginationMeta, RequestPasswordResetData,
            SendVerificationCodeData, SessionItem, VerifyCodeData, VerifyResetCodeData,
        },
    },
    error::{AppError, Result},
    repositories::{
        password_reset_repo::PasswordResetRepository, session_repo::SessionRepository,
        user_repo::UserRepository, verification_repo::VerificationRepository,
    },
    utils::{
        bloom_filter::ResourceExistenceChecker, id_gen::IdGenerator, jwt::JwtUtil,
        password::PasswordUtil, token::TokenUtil, validation::Validator,
    },
};

// ClientType 已移至 types::enums 模块
pub use crate::types::enums::ClientType;

#[derive(Debug, Clone)]
pub struct AuthIssueResult {
    pub data: AuthData,
    /// handler 用于写入 HttpOnly cookie（web 场景）
    pub refresh_token_raw: String,
}

#[derive(Clone)]
pub struct AuthService {
    db: PgPool,
    user_repo: UserRepository,
    session_repo: SessionRepository,
    password_reset_repo: PasswordResetRepository,
    verification_repo: VerificationRepository,
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
        user_repo: UserRepository,
        session_repo: SessionRepository,
        password_reset_repo: PasswordResetRepository,
        verification_repo: VerificationRepository,
    ) -> Self {
        Self {
            db: db.clone(),
            user_repo,
            session_repo,
            password_reset_repo,
            verification_repo,
            redis: redis.clone(),
            jwt_secret,
            access_token_expiration,
            refresh_token_expiration,
        }
    }

    // 辅助方法：设置 Session 白名单
    async fn set_session_whitelist(&self, session_id: Uuid, user_id: i64) -> Result<()> {
        let key = format!("{}:{}", crate::constants::REDIS_KEY_SESSION, session_id);
        // TTL 与 Access Token 一致
        let _: () = redis::cmd("SETEX")
            .arg(&key)
            .arg(self.access_token_expiration)
            .arg(user_id)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|e| AppError::Redis(e))?;
        Ok(())
    }

    // 辅助方法：移除 Session 白名单
    async fn remove_session_whitelist(&self, session_id: Uuid) -> Result<()> {
        let key = format!("{}:{}", crate::constants::REDIS_KEY_SESSION, session_id);
        let _: () = redis::cmd("DEL")
            .arg(&key)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|e| AppError::Redis(e))?;
        Ok(())
    }

    /// 用户注册
    pub async fn register(
        &self,
        req: &RegisterRequest,
        client_type: ClientType,
        _ip_address: Option<std::net::IpAddr>,
        user_agent: Option<&str>,
        device_fingerprint: Option<&str>,
    ) -> Result<AuthIssueResult> {
        // 0. 如果提供了 verify_token，进行验证（手机号注册场景）
        if let Some(verify_token) = &req.verify_token {
            let _phone = self.validate_verify_token(verify_token, "register").await?;
            // 验证通过，可以继续注册
            // 注意：这里的 phone 应该与 req.login_id 一致（如果是手机号注册）
        }

        // 1. 检查登录ID是否已存在
        if self.user_repo.exists_by_login_id(&req.login_id).await? {
            return Err(AppError::AlreadyExists("该登录ID已被注册".to_string()));
        }

        // 2. 从数据库序列生成用户ID
        let user_id = IdGenerator::generate_user_id(&self.db).await?;

        // 3. 对密码进行服务端慢哈希（bcrypt）
        let password_hash = PasswordUtil::hash_password(&req.password)?;

        // 4. 创建用户
        use crate::types::enums::UserType;
        self.user_repo
            .create(
                user_id,
                UserType::Normal,
                Some(req.login_id.clone()),
                req.nickname.clone(),
                Some(password_hash.clone()),
            )
            .await?;
        if let Err(e) = ResourceExistenceChecker::new(self.redis.clone())
            .record_user(user_id)
            .await
        {
            tracing::warn!(user_id, error = %e, "record_user bloom failed");
        }

        // 5. 生成 access_token（JWT）+ refresh_token（opaque）
        let session_id = Uuid::new_v4();
        let access_token = JwtUtil::generate_token(
            user_id,
            session_id,
            self.access_token_expiration,
            &self.jwt_secret,
        )?;
        let refresh_token = TokenUtil::generate_refresh_token();

        // DB 仅存 hash
        let access_token_hash = TokenUtil::hash_token(&access_token, &self.jwt_secret);
        let refresh_token_hash = TokenUtil::hash_token(&refresh_token, &self.jwt_secret);

        // 6. 创建会话记录
        let session_expires_at = Utc::now() + Duration::seconds(self.refresh_token_expiration);
        self.session_repo
            .create(
                user_id,
                session_id,
                None, // device_type
                None, // device_name
                device_fingerprint.map(|s| s.to_string()),
                None, // ip_address
                user_agent.map(|s| s.to_string()),
                access_token_hash,
                refresh_token_hash,
                session_expires_at,
            )
            .await?;

        // 7. 设置 Session 白名单 (Active Session Whitelist)
        self.set_session_whitelist(session_id, user_id).await?;

        // 8. 构造响应（web 不在 body 返回 refresh_token）
        let refresh_token_in_body = if client_type == ClientType::Web {
            None
        } else {
            Some(refresh_token.clone())
        };

        Ok(AuthIssueResult {
            data: AuthData {
                user_id,
                nickname: req.nickname.clone(),
                access_token,
                refresh_token: refresh_token_in_body,
                access_token_expires_at: (Utc::now()
                    + Duration::seconds(self.access_token_expiration))
                .to_rfc3339(),
                refresh_token_expires_at: (Utc::now()
                    + Duration::seconds(self.refresh_token_expiration))
                .to_rfc3339(),
            },
            refresh_token_raw: refresh_token,
        })
    }

    /// 用户登录
    pub async fn login(
        &self,
        req: &LoginRequest,
        client_type: ClientType,
        _ip_address: Option<std::net::IpAddr>,
        user_agent: Option<&str>,
        device_fingerprint: Option<&str>,
    ) -> Result<AuthIssueResult> {
        // 1. 查找用户
        let user = self
            .user_repo
            .find_by_login_id(&req.login_id)
            .await?
            .ok_or_else(|| AppError::Unauthorized("用户名或密码错误".to_string()))?;

        if self.user_repo.is_banned(user.user_id).await? {
            return Err(AppError::AccountBanned("账号已被封禁".to_string()));
        }

        // 2. 验证密码
        let password_hash = user
            .password_hash
            .ok_or_else(|| AppError::Unauthorized("该账号未设置密码".to_string()))?;

        if !PasswordUtil::verify_password(&req.password, &password_hash)? {
            return Err(AppError::PasswordError("用户名或密码错误".to_string()));
        }

        // 3. 生成 access_token（JWT）+ refresh_token（opaque）
        let session_id = Uuid::new_v4();
        let access_token = JwtUtil::generate_token(
            user.user_id,
            session_id,
            self.access_token_expiration,
            &self.jwt_secret,
        )?;
        let refresh_token = TokenUtil::generate_refresh_token();

        let access_token_hash = TokenUtil::hash_token(&access_token, &self.jwt_secret);
        let refresh_token_hash = TokenUtil::hash_token(&refresh_token, &self.jwt_secret);

        // 4. 创建会话记录
        let session_expires_at = Utc::now() + Duration::seconds(self.refresh_token_expiration);
        self.session_repo
            .create(
                user.user_id,
                session_id,
                None, // device_type
                None, // device_name
                device_fingerprint.map(|s| s.to_string()),
                None, // ip_address
                user_agent.map(|s| s.to_string()),
                access_token_hash,
                refresh_token_hash,
                session_expires_at,
            )
            .await?;

        // 5. 设置 Session 白名单
        self.set_session_whitelist(session_id, user.user_id).await?;

        // 6. 更新最后活跃时间
        self.user_repo.update_last_active(user.user_id).await?;

        let refresh_token_in_body = if client_type == ClientType::Web {
            None
        } else {
            Some(refresh_token.clone())
        };

        Ok(AuthIssueResult {
            data: AuthData {
                user_id: user.user_id,
                nickname: user.nickname,
                access_token,
                refresh_token: refresh_token_in_body,
                access_token_expires_at: (Utc::now()
                    + Duration::seconds(self.access_token_expiration))
                .to_rfc3339(),
                refresh_token_expires_at: (Utc::now()
                    + Duration::seconds(self.refresh_token_expiration))
                .to_rfc3339(),
            },
            refresh_token_raw: refresh_token,
        })
    }

    /// 创建访客用户
    pub async fn create_guest(
        &self,
        req: &GuestRequest,
        client_type: ClientType,
        _ip_address: Option<std::net::IpAddr>,
        user_agent: Option<&str>,
        device_fingerprint: Option<&str>,
    ) -> Result<AuthIssueResult> {
        // 1. 从数据库序列生成用户ID（临时用户和注册用户共用序列）
        let user_id = IdGenerator::generate_user_id(&self.db).await?;

        // 2. 创建用户（无密码）
        use crate::types::enums::UserType;
        self.user_repo
            .create(user_id, UserType::Temp, None, req.nickname.clone(), None)
            .await?;
        if let Err(e) = ResourceExistenceChecker::new(self.redis.clone())
            .record_user(user_id)
            .await
        {
            tracing::warn!(user_id, error = %e, "record_user bloom failed");
        }

        // 3. 生成 access_token（JWT）+ refresh_token（opaque）
        let session_id = Uuid::new_v4();
        let access_token = JwtUtil::generate_token(
            user_id,
            session_id,
            self.access_token_expiration,
            &self.jwt_secret,
        )?;
        let refresh_token = TokenUtil::generate_refresh_token();

        let access_token_hash = TokenUtil::hash_token(&access_token, &self.jwt_secret);
        let refresh_token_hash = TokenUtil::hash_token(&refresh_token, &self.jwt_secret);

        // 4. 创建会话记录
        let session_expires_at = Utc::now() + Duration::seconds(self.refresh_token_expiration);
        self.session_repo
            .create(
                user_id,
                session_id,
                None, // device_type
                None, // device_name
                device_fingerprint.map(|s| s.to_string()),
                None, // ip_address
                user_agent.map(|s| s.to_string()),
                access_token_hash,
                refresh_token_hash,
                session_expires_at,
            )
            .await?;

        // 5. 设置 Session 白名单
        self.set_session_whitelist(session_id, user_id).await?;

        let refresh_token_in_body = if client_type == ClientType::Web {
            None
        } else {
            Some(refresh_token.clone())
        };

        Ok(AuthIssueResult {
            data: AuthData {
                user_id,
                nickname: req.nickname.clone(),
                access_token,
                refresh_token: refresh_token_in_body,
                access_token_expires_at: (Utc::now()
                    + Duration::seconds(self.access_token_expiration))
                .to_rfc3339(),
                refresh_token_expires_at: (Utc::now()
                    + Duration::seconds(self.refresh_token_expiration))
                .to_rfc3339(),
            },
            refresh_token_raw: refresh_token,
        })
    }

    /// 刷新Token（支持 Refresh Token Rotation 和 重用检测，并绑定客户端指纹）
    pub async fn refresh_token(
        &self,
        refresh_token_raw: &str,
        client_type: ClientType,
        _ip_address: Option<std::net::IpAddr>,
        user_agent: Option<&str>,
        device_fingerprint: Option<&str>,
    ) -> Result<AuthIssueResult> {
        // 1. refresh_token -> hash（DB 查找用）
        let refresh_token_hash = TokenUtil::hash_token(refresh_token_raw, &self.jwt_secret);

        // 2. 尝试通过 refresh_token_hash 查找会话（作为当前 Token）
        let session = self
            .session_repo
            .find_by_refresh_token(&refresh_token_hash)
            .await?;

        // 3. 处理会话查找结果
        let session = match session {
            Some(s) => s,
            None => {
                // 如果作为当前 Token 未找到，检查是否作为"前一个 Token"被使用（重用检测）
                if let Some(compromised_session) = self
                    .session_repo
                    .find_by_previous_refresh_token(&refresh_token_hash)
                    .await?
                {
                    // !!! 安全警告：检测到 Token 重用 !!!
                    // 这意味着有人（可能是攻击者）试图使用一个已经被轮换过的旧 Token。
                    // 应对措施：立即撤销该会话，切断攻击者和合法用户的访问。
                    tracing::warn!(
                        "Refresh Token Reuse Detected! Revoking session. SessionId: {}, UserId: {}",
                        compromised_session.session_id,
                        compromised_session.user_id
                    );
                    self.session_repo
                        .revoke(compromised_session.session_id)
                        .await?;

                    // 移除白名单（立即失效）
                    self.remove_session_whitelist(compromised_session.session_id)
                        .await?;

                    return Err(AppError::ReLoginRequired(
                        "安全警告：检测到Token重用，为了您的账户安全，已强制登出".to_string(),
                    ));
                }

                // 既不是当前 Token 也不是前一个 Token -> 无效
                return Err(AppError::RefreshTokenInvalid(
                    "Refresh Token 无效或已过期".to_string(),
                ));
            }
        };

        let user_id = session.user_id;

        // 5. 验证客户端指纹（可选，增强安全性）
        // 如果当前请求的 User-Agent 与会话创建时的不一致，拒绝刷新
        if let (Some(req_ua), Some(session_ua)) = (user_agent, &session.user_agent) {
            if req_ua != session_ua {
                tracing::warn!(
                    "Token Refresh blocked due to User-Agent mismatch. SessionId: {}, UserId: {}, ReqUA: {}, SessionUA: {}",
                    session.session_id,
                    user_id,
                    req_ua,
                    session_ua
                );
                // 不撤销会话，只拒绝这次刷新，防止因为浏览器升级等原因导致误杀
                return Err(AppError::ReLoginRequired(
                    "环境异常，请重新登录".to_string(),
                ));
            }
        }

        // 验证设备指纹
        if let (Some(req_fp), Some(session_fp)) = (device_fingerprint, &session.device_fingerprint)
        {
            if req_fp != session_fp {
                tracing::warn!(
                    "Token Refresh blocked due to Device Fingerprint mismatch. SessionId: {}, UserId: {}, ReqFP: {}, SessionFP: {}",
                    session.session_id,
                    user_id,
                    req_fp,
                    session_fp
                );
                return Err(AppError::ReLoginRequired(
                    "设备环境异常，请重新登录".to_string(),
                ));
            }
        }

        // 6. 查找用户
        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or_else(|| AppError::NotFound("用户不存在".to_string()))?;

        if self.user_repo.is_banned(user_id).await? {
            return Err(AppError::AccountBanned("账号已被封禁".to_string()));
        }

        // 7. 生成新的 Token (Token Rotation)
        // 保持 Session ID 不变
        let access_token = JwtUtil::generate_token(
            user_id,
            session.session_id,
            self.access_token_expiration,
            &self.jwt_secret,
        )?;
        let new_refresh_token = TokenUtil::generate_refresh_token();

        let access_token_hash = TokenUtil::hash_token(&access_token, &self.jwt_secret);
        let new_refresh_token_hash = TokenUtil::hash_token(&new_refresh_token, &self.jwt_secret);

        // 8. 更新会话中的 token（轮换：current -> previous, new -> current）
        //    这里必须防并发：同一个 refresh_token 同时被两个请求刷新，会导致 token 覆盖/错乱，进而出现“下次刷新直接无效”
        //    使用乐观锁：只有 refresh_token_hash 仍等于本次请求提供的 hash 才允许轮换
        let session_expires_at = Utc::now() + Duration::seconds(self.refresh_token_expiration);
        let rotated = self
            .session_repo
            .rotate_tokens_if_refresh_matches(
                session.session_id,
                &refresh_token_hash,
                access_token_hash,
                new_refresh_token_hash,
                session_expires_at,
            )
            .await?;

        if !rotated {
            // 典型原因：并发刷新导致当前 refresh_token 已被轮换（但会话本身仍有效）。
            // 不做撤销（避免误杀），提示客户端稍后重试/以最新 token 为准。
            return Err(AppError::AccessTokenInvalid(
                "Refresh Token 已被轮换，请重试".to_string(),
            ));
        }

        // 9. 刷新 Session 白名单 TTL (因为 Access Token 续期了)
        self.set_session_whitelist(session.session_id, user_id)
            .await?;

        let refresh_token_in_body = if client_type == ClientType::Web {
            None
        } else {
            Some(new_refresh_token.clone())
        };

        Ok(AuthIssueResult {
            data: AuthData {
                user_id,
                nickname: user.nickname,
                access_token,
                refresh_token: refresh_token_in_body,
                access_token_expires_at: (Utc::now()
                    + Duration::seconds(self.access_token_expiration))
                .to_rfc3339(),
                refresh_token_expires_at: (Utc::now()
                    + Duration::seconds(self.refresh_token_expiration))
                .to_rfc3339(),
            },
            refresh_token_raw: new_refresh_token,
        })
    }

    /// 登出
    pub async fn logout(&self, user_id: i64, token: &str) -> Result<()> {
        // 1. 解析 access_token，拿到 sid
        let claims = JwtUtil::verify_token(token, &self.jwt_secret)?;
        let sid = Uuid::parse_str(&claims.sid)
            .map_err(|_| AppError::Unauthorized("无效的Session".to_string()))?;

        // 2. 撤销会话（DB + Redis 白名单）
        if let Some(session) = self.session_repo.find_by_id(sid).await? {
            if session.user_id == user_id {
                self.session_repo.revoke(session.session_id).await?;
                self.remove_session_whitelist(session.session_id).await?;
            }
        }

        // 登出主要动作是撤销 Refresh Token（会话），并移除 Redis 中的 Session 白名单。
        // 这确保了 Access Token 即使在有效期内也无法通过 AuthMiddleware 验证。

        Ok(())
    }

    /// 检查认证状态
    pub async fn check_status(&self, user_id: i64, token: &str) -> Result<CheckStatusData> {
        // 1. 验证 Token
        let claims = JwtUtil::verify_token(token, &self.jwt_secret)?;

        // 2. 查找用户
        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or_else(|| AppError::NotFound("用户不存在".to_string()))?;

        Ok(CheckStatusData {
            user_id,
            nickname: user.nickname,
            is_valid: true,
            expires_at: chrono::DateTime::<Utc>::from_timestamp(claims.exp, 0)
                .unwrap_or_else(|| Utc::now())
                .to_rfc3339(),
        })
    }

    /// 请求密码重置
    pub async fn request_password_reset(
        &self,
        req: &RequestPasswordResetRequest,
    ) -> Result<RequestPasswordResetData> {
        // 0. 如果提供了 verify_token，进行验证（手机号重置密码场景）
        if let Some(verify_token) = &req.verify_token {
            let _phone = self
                .validate_verify_token(verify_token, "reset_password")
                .await?;
            // 验证通过，可以继续重置密码
        }

        // 安全考虑：避免通过接口响应判断账号是否存在（防用户枚举）
        // 不论 login_id 是否存在，都返回 reset_id（但只有存在账号时才会真实创建重置记录）。

        // 1) 生成 reset_id（始终返回）
        let public_reset_id = uuid::Uuid::new_v4();

        // 2) 查找用户（不存在则直接返回成功）
        let Some(user) = self.user_repo.find_by_login_id(&req.login_id).await? else {
            return Ok(RequestPasswordResetData {
                reset_id: public_reset_id.to_string(),
            });
        };

        // 3) 生成重置码（6位，安全随机）
        // rand 0.9: 通过 rand::rand_core 访问 OsRng
        use rand::rand_core::{OsRng, TryRngCore};
        let mut rng = OsRng;
        let n = rng.try_next_u32().map_err(|_| AppError::internal("RNG error"))? % 1_000_000;
        let reset_code = format!("{:06}", n);

        // 4) 存储到 DB（15分钟有效，仅存 code_hash）
        let code_hash = PasswordUtil::hash_password(&reset_code)?;
        let expires_at = Utc::now() + Duration::minutes(15);
        let reset_id = self
            .password_reset_repo
            .create(user.user_id, reset_code.clone(), code_hash, expires_at)
            .await?;

        // 5) 发送验证码（初始版本：暂以日志方式投递，后续接入短信/邮件通道）
        //
        // 安全：生产环境禁止把验证码明文写入日志（避免日志泄露导致账号接管）。
        // 本地开发（debug build）允许输出，便于联调。
        if cfg!(debug_assertions) {
            tracing::debug!(
                "password_reset send code={} reset_id={} user_id={}",
                reset_code,
                reset_id,
                user.user_id
            );
        } else {
            tracing::info!(
                "password_reset requested reset_id={} user_id={}",
                reset_id,
                user.user_id
            );
        }

        Ok(RequestPasswordResetData {
            reset_id: reset_id.to_string(),
        })
    }

    /// 验证重置码
    pub async fn verify_reset_code(
        &self,
        req: &VerifyResetCodeRequest,
    ) -> Result<VerifyResetCodeData> {
        let reset_id = uuid::Uuid::parse_str(&req.reset_id)
            .map_err(|_| AppError::BadRequest("reset_id 格式错误".to_string()))?;

        let Some(row) = self
            .password_reset_repo
            .find_valid_by_reset_id(reset_id)
            .await?
        else {
            return Err(AppError::BadRequest("重置请求不存在或已过期".to_string()));
        };

        // 1) 校验验证码（bcrypt）
        if !PasswordUtil::verify_password(&req.code, &row.code_hash)? {
            return Err(AppError::BadRequest("验证码错误".to_string()));
        }

        // 2) 生成 reset_token（不透明随机字符串，5分钟有效）
        let reset_token = TokenUtil::generate_refresh_token();
        let reset_token_hash = TokenUtil::hash_token(&reset_token, &self.jwt_secret);
        let now = Utc::now();
        let token_expires_at = now + Duration::minutes(5);

        self.password_reset_repo
            .set_verified(row.id, reset_token_hash, now, token_expires_at)
            .await?;

        Ok(VerifyResetCodeData { reset_token })
    }

    /// 重置密码
    pub async fn reset_password(&self, req: &ResetPasswordRequest) -> Result<()> {
        // 1) 验证重置 Token（opaque -> hash -> DB lookup）
        let reset_token_hash = TokenUtil::hash_token(&req.reset_token, &self.jwt_secret);
        let Some(row) = self
            .password_reset_repo
            .find_valid_by_reset_token_hash(&reset_token_hash)
            .await?
        else {
            return Err(AppError::Unauthorized("重置Token无效或已过期".to_string()));
        };
        let user_id = row.user_id;

        // 2. 对密码进行服务端慢哈希（bcrypt）
        let password_hash = PasswordUtil::hash_password(&req.password)?;

        // 3. 更新密码
        self.user_repo
            .update_password(user_id, password_hash.clone())
            .await?;

        // 4) 标记 reset_token 已使用
        self.password_reset_repo
            .mark_used(row.id, Utc::now())
            .await?;

        // 5. 撤销所有会话（强制重新登录）
        let revoked_session_ids = self.session_repo.revoke_all_by_user(user_id).await?;
        if !revoked_session_ids.is_empty() {
            let keys: Vec<String> = revoked_session_ids
                .into_iter()
                .map(|sid| format!("{}:{}", crate::constants::REDIS_KEY_SESSION, sid))
                .collect();
            let _: () = redis::cmd("DEL")
                .arg(keys)
                .query_async(&mut self.redis.clone())
                .await
                .map_err(|e| AppError::Redis(e))?;
        }

        Ok(())
    }

    /// 修改密码（需要验证旧密码）
    pub async fn change_password(&self, user_id: i64, req: &ChangePasswordRequest) -> Result<()> {
        // 1. 查找用户
        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or_else(|| AppError::NotFound("用户不存在".to_string()))?;

        // 2. 验证用户是否有密码（临时用户没有密码）
        let stored_hash = user.password_hash.as_ref().ok_or_else(|| {
            AppError::BadRequest("临时用户请使用账户升级功能设置密码".to_string())
        })?;

        // 3. 验证旧密码
        if !PasswordUtil::verify_password(&req.old_password, stored_hash)? {
            return Err(AppError::PasswordError("当前密码错误".to_string()));
        }

        // 4. 对新密码进行服务端慢哈希（bcrypt）
        let new_password_hash = PasswordUtil::hash_password(&req.new_password)?;

        // 5. 更新密码
        self.user_repo
            .update_password(user_id, new_password_hash)
            .await?;

        // 6. 撤销所有会话（强制重新登录）
        let revoked_session_ids = self.session_repo.revoke_all_by_user(user_id).await?;
        if !revoked_session_ids.is_empty() {
            let keys: Vec<String> = revoked_session_ids
                .into_iter()
                .map(|sid| format!("{}:{}", crate::constants::REDIS_KEY_SESSION, sid))
                .collect();
            let _: () = redis::cmd("DEL")
                .arg(keys)
                .query_async(&mut self.redis.clone())
                .await
                .map_err(|e| AppError::Redis(e))?;
        }

        Ok(())
    }

    /// 列出活跃会话
    /// current_session_id: 当前请求的会话ID（可选，用于标记当前会话）
    pub async fn list_sessions(
        &self,
        user_id: i64,
        req: &SessionsRequest,
        current_session_id: Option<Uuid>,
    ) -> Result<PaginatedData<SessionItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = req.cursor.clone();

        let (sessions, next_cursor) = self
            .session_repo
            .list_by_user(user_id, limit, cursor)
            .await?;

        let items = sessions
            .into_iter()
            .map(|s| {
                let is_current = current_session_id
                    .map(|id| id == s.session_id)
                    .unwrap_or(false);
                SessionItem {
                    session_id: s.session_id.to_string(),
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
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    /// 撤销会话
    pub async fn revoke_session(&self, user_id: i64, req: &RevokeSessionRequest) -> Result<()> {
        // 1. 解析会话ID
        let session_id = uuid::Uuid::parse_str(&req.session_id)
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

        // 4. 删除会话（数据库）
        self.session_repo.delete(session_id).await?;

        // 5. 移除 Session 白名单（Active Invalidation）
        // 确保 Access Token 立即失效
        self.remove_session_whitelist(session_id).await?;

        Ok(())
    }

    /// 发送验证码
    pub async fn send_verification_code(
        &self,
        req: &SendVerificationCodeRequest,
    ) -> Result<SendVerificationCodeData> {
        // 1. 验证手机号格式
        Validator::validate_not_empty(&req.phone, "手机号")?;

        // 2. 验证码类型已是强类型枚举，直接使用字符串形式
        let code_type = req.r#type.as_str();

        // 3. 检查图形验证码（可选）
        self.verify_captcha_token(req.captcha_token.as_deref())
            .await?;

        // 4. 防刷：检查是否在限流期内（60秒内只能发送一次）
        let within_60s = self
            .verification_repo
            .check_recent_code(&req.phone, code_type, 60)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        if within_60s {
            return Err(AppError::RateLimited(
                "验证码发送过于频繁，请60秒后再试".to_string(),
            ));
        }

        // 4. 检查今日发送次数（每天最多10次）
        let today_count = self
            .verification_repo
            .count_today_codes(&req.phone, code_type)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        if today_count >= 10 {
            return Err(AppError::DailyRateLimitExceeded);
        }

        // 5. 生成验证码（6位数字）
        let code = TokenUtil::generate_numeric_code(6);

        // 6. 计算过期时间（5分钟）
        let expires_at = Utc::now() + Duration::minutes(5);

        // 7. 保存验证码记录
        let verification_code = self
            .verification_repo
            .create_code(&req.phone, &code, code_type, expires_at)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        // 8. 发送短信（写入发送队列，供外部短信服务消费）
        let content = format!("您的验证码是: {}, 5分钟内有效", code);
        self.enqueue_sms(&req.phone, &content, code_type, &verification_code.code_id)
            .await?;

        // 9. 返回响应
        Ok(SendVerificationCodeData {
            code_id: verification_code.code_id,
            expires_at: expires_at.to_rfc3339(),
            retry_after: Some(60),
        })
    }

    async fn verify_captcha_token(&self, token: Option<&str>) -> Result<()> {
        let Some(token) = token else {
            return Ok(());
        };
        let key = format!("captcha:token:{}", token);
        let mut conn = self.redis.clone();
        let exists: bool = redis::cmd("EXISTS")
            .arg(&key)
            .query_async(&mut conn)
            .await
            .unwrap_or(false);
        if !exists {
            return Err(AppError::InvalidFormat(
                "图形验证码无效或已过期".to_string(),
            ));
        }
        let _: () = redis::cmd("DEL")
            .arg(&key)
            .query_async(&mut conn)
            .await
            .unwrap_or(());
        Ok(())
    }

    async fn enqueue_sms(
        &self,
        phone: &str,
        content: &str,
        code_type: &str,
        code_id: &str,
    ) -> Result<()> {
        let payload = serde_json::json!({
            "phone": phone,
            "content": content,
            "type": code_type,
            "code_id": code_id,
            "created_at": Utc::now().to_rfc3339(),
        });
        let mut conn = self.redis.clone();
        redis::cmd("LPUSH")
            .arg("sms:outbox")
            .arg(payload.to_string())
            .query_async::<()>(&mut conn)
            .await
            .map_err(|e| AppError::MessageQueueError(format!("短信队列写入失败: {}", e)))?;
        Ok(())
    }

    /// 验证验证码
    pub async fn verify_code(&self, req: &VerifyCodeRequest) -> Result<VerifyCodeData> {
        // 1. 查找验证码记录
        let verification_code = self
            .verification_repo
            .find_by_code_id(&req.code_id)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?
            .ok_or_else(|| AppError::NotFound("验证码不存在或已失效".to_string()))?;

        // 2. 检查是否已过期
        if verification_code.expires_at < Utc::now() {
            return Err(AppError::Conflict("验证码已过期".to_string()));
        }

        // 3. 检查尝试次数
        if verification_code.attempts >= verification_code.max_attempts {
            return Err(AppError::Conflict(
                "验证次数已超限，请重新获取验证码".to_string(),
            ));
        }

        // 4. 验证验证码
        if verification_code.code != req.code {
            // 增加尝试次数
            self.verification_repo
                .increment_attempts(&req.code_id)
                .await
                .map_err(|e| AppError::DatabaseError(e.to_string()))?;

            return Err(AppError::Conflict("验证码错误".to_string()));
        }

        // 5. 标记为已验证
        self.verification_repo
            .mark_verified(&req.code_id)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        // 6. 生成 verify_token（用于后续注册/重置密码等操作）
        let verify_token = TokenUtil::generate_token();

        // 7. 将 verify_token 存储到 Redis（有效期 10 分钟）
        let verify_token_key = format!("verify_token:{}", verify_token);
        let verify_token_data = format!("{}:{}", verification_code.phone, verification_code.r#type);
        let _: () = redis::cmd("SETEX")
            .arg(&verify_token_key)
            .arg(600) // 10分钟
            .arg(&verify_token_data)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|e| AppError::Redis(e))?;

        let verify_expires_at = Utc::now() + Duration::minutes(10);

        Ok(VerifyCodeData {
            verified: true,
            verify_token,
            expires_at: verify_expires_at.to_rfc3339(),
        })
    }

    /// 验证 verify_token（内部方法，用于注册/重置密码时验证）
    async fn validate_verify_token(
        &self,
        verify_token: &str,
        expected_type: &str,
    ) -> Result<String> {
        let verify_token_key = format!("verify_token:{}", verify_token);

        let verify_token_data: Option<String> = redis::cmd("GET")
            .arg(&verify_token_key)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|e| AppError::Redis(e))?;

        let verify_token_data = verify_token_data
            .ok_or_else(|| AppError::VerifyTokenInvalid("验证令牌无效或已过期".to_string()))?;

        // 解析 phone:type
        let parts: Vec<&str> = verify_token_data.split(':').collect();
        if parts.len() != 2 {
            return Err(AppError::VerifyTokenInvalid(
                "验证令牌格式无效".to_string(),
            ));
        }

        let phone = parts[0];
        let code_type = parts[1];

        if code_type != expected_type {
            return Err(AppError::VerifyTokenInvalid(
                "验证令牌类型不匹配".to_string(),
            ));
        }

        // 删除已使用的 verify_token
        let _: () = redis::cmd("DEL")
            .arg(&verify_token_key)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|e| AppError::Redis(e))?;

        Ok(phone.to_string())
    }
}
