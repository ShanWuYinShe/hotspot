use chrono::{Duration, Utc};
use redis::aio::ConnectionManager;
use sqlx::PgPool;

use crate::{
    dto::{
        request::{CreateTempRequest, LoginRequest, RegisterRequest, RefreshTokenRequest},
        response::AuthResponse,
    },
    error::{AppError, Result},
    models::user::User,
    repositories::user_repo::UserRepository,
    utils::{id_gen::IdGenerator, jwt::JwtUtil, password::PasswordUtil},
};

pub struct AuthService {
    user_repo: UserRepository,
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
            redis: redis.clone(),
            jwt_secret,
            access_token_expiration,
            refresh_token_expiration,
        }
    }

    /// 用户注册
    pub async fn register(&self, req: &RegisterRequest) -> Result<AuthResponse> {
        // 1. 检查登录ID是否已存在
        if self.user_repo.exists_by_login_id(&req.login_id).await? {
            return Err(AppError::Conflict("该登录ID已被注册".to_string()));
        }

        // 2. 生成用户ID（类型：普通用户 0x02）
        let user_id = IdGenerator::generate_user_id(0x02);

        // 3. 对客户端哈希进行 bcrypt 二次哈希
        let password_hash = PasswordUtil::hash_password(&req.password_hash)?;

        // 4. 创建用户
        let user = User {
            user_id,
            login_id: Some(req.login_id.clone()),
            nickname: req.nickname.clone(),
            password_hash: Some(password_hash),
            ..Default::default()
        };

        self.user_repo.create(&user).await?;

        // 5. 生成 JWT Token
        let access_token =
            JwtUtil::generate_token(user_id, self.access_token_expiration, &self.jwt_secret)?;
        let refresh_token =
            JwtUtil::generate_token(user_id, self.refresh_token_expiration, &self.jwt_secret)?;

        // 6. 存储 Refresh Token 到 Redis
        let key = format!("refresh_token:{}", user_id);
        redis::cmd("SETEX")
            .arg(&key)
            .arg(self.refresh_token_expiration)
            .arg(&refresh_token)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        // 7. 构造响应
        Ok(AuthResponse {
            user_id: user_id.to_string(),
            nickname: user.nickname,
            access_token,
            refresh_token,
            expires_at: Utc::now() + Duration::seconds(self.access_token_expiration),
        })
    }

    /// 用户登录
    pub async fn login(&self, req: &LoginRequest) -> Result<AuthResponse> {
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
        redis::cmd("SETEX")
            .arg(&key)
            .arg(self.refresh_token_expiration)
            .arg(&refresh_token)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        // 5. 更新最后活跃时间
        self.user_repo.update_last_active(user.user_id).await?;

        Ok(AuthResponse {
            user_id: user.user_id.to_string(),
            nickname: user.nickname,
            access_token,
            refresh_token,
            expires_at: Utc::now() + Duration::seconds(self.access_token_expiration),
        })
    }

    /// 创建临时用户
    pub async fn create_temp(&self, req: &CreateTempRequest) -> Result<AuthResponse> {
        // 1. 生成临时用户ID（类型：临时用户 0x01）
        let user_id = IdGenerator::generate_user_id(0x01);

        // 2. 创建用户（无密码）
        let user = User {
            user_id,
            login_id: None,
            nickname: req.nickname.clone(),
            password_hash: None,
            ..Default::default()
        };

        self.user_repo.create(&user).await?;

        // 3. 生成 Token
        let access_token =
            JwtUtil::generate_token(user_id, self.access_token_expiration, &self.jwt_secret)?;
        let refresh_token =
            JwtUtil::generate_token(user_id, self.refresh_token_expiration, &self.jwt_secret)?;

        // 4. 存储 Refresh Token
        let key = format!("refresh_token:{}", user_id);
        redis::cmd("SETEX")
            .arg(&key)
            .arg(self.refresh_token_expiration)
            .arg(&refresh_token)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        Ok(AuthResponse {
            user_id: user_id.to_string(),
            nickname: user.nickname,
            access_token,
            refresh_token,
            expires_at: Utc::now() + Duration::seconds(self.access_token_expiration),
        })
    }

    /// 刷新Token
    pub async fn refresh_token(&self, req: &RefreshTokenRequest) -> Result<AuthResponse> {
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
            .ok_or_else(|| AppError::NotFound)?;

        // 4. 生成新的 Token
        let access_token =
            JwtUtil::generate_token(user_id, self.access_token_expiration, &self.jwt_secret)?;
        let refresh_token =
            JwtUtil::generate_token(user_id, self.refresh_token_expiration, &self.jwt_secret)?;

        // 5. 更新 Redis 中的 Refresh Token
        redis::cmd("SETEX")
            .arg(&key)
            .arg(self.refresh_token_expiration)
            .arg(&refresh_token)
            .query_async(&mut self.redis.clone())
            .await
            .map_err(|_| AppError::Internal)?;

        Ok(AuthResponse {
            user_id: user_id.to_string(),
            nickname: user.nickname,
            access_token,
            refresh_token,
            expires_at: Utc::now() + Duration::seconds(self.access_token_expiration),
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
}

