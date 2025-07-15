use serde::Serialize;
use sqlx::FromRow;
use std::sync::Arc;
use tracing::{debug, error, info, warn};

use crate::{
    repositories::{RepositoryError, user_repository::UserRepository},
    services::ConfigService,
    utils::{
        constants::{DEFAULT_LATITUDE, DEFAULT_LOCATION_NAME, DEFAULT_LONGITUDE},
        jwt::{generate_jwt_token, verify_jwt_token},
        response::AppError,
        result_ext::ResultExt,
        UserIdPartition, UserType,
    },
};

// =============== 返回结构体 ===============

#[derive(Debug, Clone, Serialize, FromRow)]
pub struct UserRow {
    pub user_id: i64,
    pub nickname: String,
    pub last_active: chrono::DateTime<chrono::Utc>,
    pub latitude: f64,
    pub longitude: f64,
    pub distance: f64,
    pub location_name: String,
    pub last_active_at: chrono::DateTime<chrono::Utc>,
}

#[derive(Debug, Serialize)]
pub struct UserLoginResult {
    pub user_id: i64,
    pub nickname: String,
    pub access_token: String,
    pub refresh_token: String,
    pub expires_at: chrono::DateTime<chrono::Utc>,
}

#[derive(Debug)]
pub struct AuthUserRow {
    pub user_id: i64,
    pub nickname: String,
}

#[derive(Debug, Clone)]
pub struct UserService {
    config_service: Arc<ConfigService>,
    user_repository: Arc<crate::repositories::PgUserRepository>,
    job_dispatcher: Option<Arc<crate::jobs::dispatcher::JobDispatcher>>,
}

impl UserService {
    pub fn new(
        config_service: Arc<ConfigService>,
        user_repository: Arc<crate::repositories::PgUserRepository>,
    ) -> Self {
        Self {
            config_service,
            user_repository,
            job_dispatcher: None,
        }
    }

    /// 设置任务调度器
    pub fn with_job_dispatcher(mut self, job_dispatcher: Arc<crate::jobs::dispatcher::JobDispatcher>) -> Self {
        self.job_dispatcher = Some(job_dispatcher);
        self
    }

    // =============== 核心业务方法 ===============

    /// 检查登录ID是否已存在
    pub async fn is_login_id_taken(&self, login_id: &str) -> Result<bool, AppError> {
        self.user_repository
            .is_login_id_taken(login_id)
            .await
            .map_repository_error("检查登录ID")
    }

    /// 检查用户是否已设置密码
    pub async fn is_password_set(&self, user_id: i64) -> Result<bool, AppError> {
        // 临时用户永远没有设置密码
        if self
            .user_repository
            .is_temp_user(user_id)
            .await
            .unwrap_or(false)
        {
            return Ok(false);
        }

        self.user_repository
            .is_password_set(user_id)
            .await
            .map_repository_error("检查密码状态")
    }

    // =============== 操作函数 ===============

    // 哈希密码
    pub async fn hash_password(&self, password: &str) -> Result<String, AppError> {
        self.user_repository
            .hash_password(password)
            .await
            .map_repository_error("密码加密")
    }

    /// 更新用户位置
    pub async fn update_location(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: &str,
    ) -> Result<(), AppError> {
        self.user_repository
            .update_user_location(user_id, latitude, longitude, location_name)
            .await
            .map_repository_error("更新用户位置")
    }

    /// 用户登录
    pub async fn login(&self, login_id: &str, password: &str) -> Result<UserLoginResult, AppError> {
        info!("用户登录: {}", login_id);

        // 获取用户信息和密码哈希
        let (user_id, nickname) = self
            .user_repository
            .get_user_login_info(login_id)
            .await
            .map_repository_error_with_msg("获取用户登录信息", "用户不存在或密码错误")?;

        let password_hash = self
            .user_repository
            .get_password_hash(login_id)
            .await
            .map_repository_error_with_msg("获取用户密码", "用户不存在或密码错误")?
            .ok_or_else(|| AppError::NotFound("用户不存在或密码错误".to_string()))?;

        // 验证密码
        let is_valid = self
            .user_repository
            .verify_password(&password_hash, password)
            .await
            .map_repository_error("密码验证")?;

        if !is_valid {
            return Err(AppError::NotFound("用户不存在或密码错误".to_string()));
        }

        // 更新最后登录时间
        self.user_repository.update_last_active(login_id).await.ok();

        // 生成令牌
        let (access_token, refresh_token, expires_at) = self.generate_tokens(user_id, false)?;

        Ok(UserLoginResult {
            user_id,
            nickname,
            access_token,
            refresh_token,
            expires_at,
        })
    }

    // 根据ID搜索用户
    pub async fn get_user_by_id(
        &self,
        user_id: i64,
        current_user_id: i64,
    ) -> Result<UserRow, AppError> {
        self.user_repository
            .get_user_by_id(user_id, current_user_id)
            .await
            .map_repository_error_with_msg("查询用户", "用户不存在")
    }

    // 查找附近的用户
    pub async fn find_nearby_users(
        &self,
        longitude: f64,
        latitude: f64,
        radius: f64,
        cursor: i64,
        limit: i64,
        current_user_id: i64,
    ) -> Result<(Vec<UserRow>, bool, Option<i64>), AppError> {
        info!(
            "【Service】开始查找附近用户: 位置=({}, {}), 半径={}, 游标={}, 限制={}",
            longitude, latitude, radius, cursor, limit
        );

        // 使用仓库模式查询附近用户
        let (user_rows, has_more, next_cursor_id) = self
            .user_repository
            .find_nearby_users(longitude, latitude, radius, cursor, limit, current_user_id)
            .await
            .map_repository_error("查询附近用户")?;

        info!(
            "【Service】查找附近用户成功: 返回{}个用户，是否有更多={}",
            user_rows.len(),
            has_more
        );

        Ok((user_rows, has_more, next_cursor_id))
    }

    // 根据名称搜索用户
    pub async fn find_users_by_name(
        &self,
        keyword: &str,
        cursor: i64,
        limit: i64,
        current_user_id: i64,
    ) -> Result<(Vec<UserRow>, bool, Option<i64>), AppError> {
        info!(
            "【Service】开始搜索用户: 关键词={}, 游标={}, 限制={}",
            keyword, cursor, limit
        );

        // 使用仓库模式查询用户
        let (user_rows, has_more, next_cursor_id) = self
            .user_repository
            .find_users_by_name(keyword, cursor, limit, current_user_id)
            .await
            .map_repository_error("搜索用户")?;

        info!(
            "【Service】搜索用户成功: 返回{}个用户，是否有更多={}",
            user_rows.len(),
            has_more
        );

        Ok((user_rows, has_more, next_cursor_id))
    }

    // 获取认证用户信息（仅返回基本信息，用于认证流程）
    pub async fn get_auth_info(&self, user_id: i64) -> Result<AuthUserRow, AppError> {
        self.user_repository
            .get_auth_info(user_id)
            .await
            .map_repository_error_with_msg("获取用户认证信息", "用户不存在")
    }

    // 注册并生成令牌
    pub async fn signup_with_token(
        &self,
        login_id: &str,
        password_hash: &str, // 从前端收到的无盐哈希值
        nickname: &str,
    ) -> Result<UserLoginResult, AppError> {
        info!(
            "【Service】开始用户注册: login_id={}, nickname={}",
            login_id, nickname
        );

        // 检查用户名是否已存在
        if self.is_login_id_taken(login_id).await? {
            warn!("【Service】注册失败: 登录ID已存在 - {}", login_id);
            return Err(AppError::Conflict("该账号已被注册".to_string()));
        }

        debug!("【Service】登录ID检查通过，开始创建用户");

        // 使用Argon2对前端传来的无盐哈希值再次加盐哈希
        // 把前端的哈希值当作"密码"再次哈希
        let argon2_hash = self.hash_password(password_hash).await?;

        // 创建用户（正式用户必须有密码）
        let user_id = self
            .user_repository
            .create_user(login_id, nickname, &argon2_hash)
            .await
            .map_err(|e| {
                error!("创建用户失败: {:?}", e);
                AppError::InternalServerError("创建用户失败".to_string())
            })?;

        debug!("【Service】用户创建成功，用户ID: {}", user_id);

        // 异步设置初始位置（不阻断注册流程）
        // 设计思路：用户注册是核心业务逻辑，不应该因为位置设置失败而导致注册失败
        // 位置设置作为辅助功能，通过任务队列异步执行，失败时只记录日志而不影响注册结果
        if let Some(job_dispatcher) = &self.job_dispatcher {
            if let Err(e) = job_dispatcher
                .dispatch_location_index(
                    user_id,
                    DEFAULT_LATITUDE,
                    DEFAULT_LONGITUDE,
                    Some(DEFAULT_LOCATION_NAME.to_string()),
                )
                .await
            {
                error!("位置初始化任务调度失败: user_id={}, error={:?}", user_id, e);
            } else {
                debug!("位置初始化任务调度成功: user_id={}", user_id);
            }
        } else {
            warn!("任务调度器未配置，跳过位置初始化: user_id={}", user_id);
        }

        // 生成JWT令牌
        let (access_token, refresh_token, expires_at) = self.generate_tokens(user_id, false)?;

        info!(
            "【Service】用户注册成功: user_id={}, nickname={}",
            user_id, nickname
        );

        Ok(UserLoginResult {
            user_id,
            nickname: nickname.to_string(),
            access_token,
            refresh_token,
            expires_at,
        })
    }

    // 创建临时用户
    pub async fn create_temp_user(&self, nickname: &str) -> Result<UserLoginResult, AppError> {
        info!("【Service】开始创建临时用户: nickname={}", nickname);

        // 生成临时登录ID
        let temp_login_id = format!("temp_{}", uuid::Uuid::new_v4().to_string().replace("-", ""));
        debug!("【Service】生成临时登录ID: {}", temp_login_id);

        // 创建临时用户（使用递增的用户ID）
        let user_id = self
            .user_repository
            .create_temp_user(&temp_login_id, nickname)
            .await
            .map_err(|e| {
                error!("创建临时用户失败: {:?}", e);
                AppError::InternalServerError("创建临时用户失败".to_string())
            })?;

        debug!("【Service】临时用户创建成功，用户ID: {}", user_id);

        // 异步设置初始位置（不阻断创建流程）
        // 设计思路：临时用户创建也是核心业务逻辑，不应该因为位置设置失败而导致创建失败
        // 客户端可以稍后通过专门的位置更新接口来设置真实位置
        if let Some(job_dispatcher) = &self.job_dispatcher {
            if let Err(e) = job_dispatcher
                .dispatch_location_index(
                    user_id,
                    DEFAULT_LATITUDE,
                    DEFAULT_LONGITUDE,
                    Some(DEFAULT_LOCATION_NAME.to_string()),
                )
                .await
            {
                error!("临时用户位置初始化任务调度失败: user_id={}, error={:?}", user_id, e);
            } else {
                debug!("临时用户位置初始化任务调度成功: user_id={}", user_id);
            }
        } else {
            warn!("任务调度器未配置，跳过临时用户位置初始化: user_id={}", user_id);
        }

        // 生成JWT令牌
        let (access_token, refresh_token, expires_at) = self.generate_tokens(user_id, true)?;

        info!(
            "【Service】临时用户创建成功: user_id={}, nickname={}",
            user_id, nickname
        );

        Ok(UserLoginResult {
            user_id,
            nickname: nickname.to_string(),
            access_token,
            refresh_token,
            expires_at,
        })
    }

    // 判断用户ID是否为临时用户
    pub fn is_temp_user(&self, user_id: i64) -> bool {
        UserIdPartition::is_user_type(user_id, UserType::Temporary)
    }

    // 刷新令牌
    pub async fn refresh_token(&self, refresh_token: &str) -> Result<UserLoginResult, AppError> {
        // 验证刷新令牌
        let config = self.config_service.get_config();
        let jwt_secret = &config.auth.jwt_secret;
        let claims = verify_jwt_token(refresh_token, jwt_secret)
            .map_err(|_| AppError::Unauthorized("刷新令牌无效或已过期".to_string()))?;

        // 获取用户信息
        let user_id = claims.sub;
        // 使用claims中的临时用户标志
        let is_temp = claims.temp;

        // 从仓库获取用户昵称
        let nickname = self
            .user_repository
            .get_user_nickname(user_id)
            .await
            .map_err(|e| match e {
                RepositoryError::NotFound => AppError::NotFound("用户不存在".to_string()),
                _ => {
                    error!("获取用户信息失败: {:?}", e);
                    AppError::InternalServerError("获取用户信息失败".to_string())
                }
            })?;

        // 生成新的访问令牌和刷新令牌
        let (new_access_token, new_refresh_token, expires_at) =
            self.generate_tokens(user_id, is_temp)?;

        Ok(UserLoginResult {
            user_id,
            nickname,
            access_token: new_access_token,
            refresh_token: new_refresh_token,
            expires_at,
        })
    }

    /// 设置/修改密码（仅正式用户可用）
    pub async fn set_password(
        &self,
        user_id: i64,
        new_password: &str,
        old_password: Option<&str>,
    ) -> Result<(), AppError> {
        // 检查是否为临时用户
        if self.is_temp_user(user_id) {
            return Err(AppError::Forbidden(
                "临时用户不能设置或修改密码，请注册正式账号".into(),
            ));
        }

        // 检查当前密码是否已设置
        let is_first_time_setting = !self
            .user_repository
            .is_password_set(user_id)
            .await
            .map_err(|e| {
                error!("检查用户密码状态失败: {:?}", e);
                AppError::InternalServerError("获取用户信息失败".to_string())
            })?;

        // 如果不是首次设置密码，需要验证旧密码
        if !is_first_time_setting {
            match old_password {
                Some(old_pwd) => {
                    // 获取当前密码哈希（正式用户必须有密码）
                    let stored_hash = self
                        .user_repository
                        .get_user_password_hash(user_id)
                        .await
                        .map_err(|e| {
                            error!("获取密码哈希失败: {:?}", e);
                            AppError::InternalServerError("获取用户信息失败".to_string())
                        })?;

                    // 验证旧密码
                    let is_valid = self
                        .user_repository
                        .verify_password(&stored_hash, old_pwd)
                        .await
                        .map_err(|e| {
                            error!("密码验证失败: {:?}", e);
                            AppError::InternalServerError("密码验证失败".to_string())
                        })?;

                    if !is_valid {
                        return Err(AppError::BadRequest("原密码不正确".to_string()));
                    }
                }
                None => {
                    // 已有密码但未提供原密码
                    return Err(AppError::BadRequest("修改密码时必须提供原密码".to_string()));
                }
            }
        }

        // 生成新密码哈希
        let password_hash = self
            .user_repository
            .hash_password(new_password)
            .await
            .map_err(|e| {
                error!("密码加密失败: {:?}", e);
                AppError::InternalServerError("密码处理失败".to_string())
            })?;

        // 更新数据库
        self.user_repository
            .update_user_password(user_id, &password_hash)
            .await
            .map_err(|e| {
                error!("设置密码失败: {:?}", e);
                AppError::InternalServerError("设置密码失败".to_string())
            })?;

        info!(
            "【Service】用户密码{}成功: user_id={}",
            if is_first_time_setting {
                "设置"
            } else {
                "修改"
            },
            user_id
        );
        Ok(())
    }

    // 生成访问和刷新令牌的辅助函数
    fn generate_tokens(
        &self,
        user_id: i64,
        is_temp: bool,
    ) -> Result<(String, String, chrono::DateTime<chrono::Utc>), AppError> {
        let config = self.config_service.get_config();
        let jwt_secret = &config.auth.jwt_secret;
        let access_expires_in = config.auth.jwt_expires_in.as_secs() as i64;
        let refresh_expires_in = config.auth.refresh_token_expires_in.as_secs() as i64;

        // 生成访问令牌
        let (access_token, expires_at) =
            generate_jwt_token(user_id, jwt_secret, access_expires_in, is_temp).map_err(|e| {
                error!("生成访问令牌失败: {:?}", e);
                AppError::InternalServerError("生成访问令牌失败".to_string())
            })?;

        // 生成刷新令牌
        let (refresh_token, _) =
            generate_jwt_token(user_id, jwt_secret, refresh_expires_in, is_temp).map_err(|e| {
                error!("生成刷新令牌失败: {:?}", e);
                AppError::InternalServerError("生成刷新令牌失败".to_string())
            })?;

        Ok((access_token, refresh_token, expires_at))
    }

    /// 获取用户ID分区信息
    pub fn get_user_id_partition_info(&self, user_id: i64) -> Result<(UserType, i64), AppError> {
        UserIdPartition::parse_user_id(user_id).map_err(|e| {
            AppError::BadRequest(format!("无效的用户ID: {}", e))
        })
    }

    /// 获取用户类型信息（用于内部验证）
    pub fn get_user_type_info(&self, user_id: i64) -> Result<UserTypeInfo, AppError> {
        let (user_type, sequence) = UserIdPartition::parse_user_id(user_id)
            .map_err(|e| AppError::BadRequest(format!("无效的用户ID: {}", e)))?;

        Ok(UserTypeInfo {
            user_id,
            user_type,
            sequence,
            type_name: user_type.name().to_string(),
            is_privileged: user_type.is_privileged(),
            is_temporary: user_type.is_temporary(),
        })
    }

    /// 获取所有用户类型的范围信息
    pub fn get_user_type_ranges() -> Vec<UserTypeRangeInfo> {
        use UserType::*;
        let user_types = [Regular, Temporary, Admin, System];
        
        user_types.iter().map(|&user_type| {
            let (min_id, max_id) = UserIdPartition::get_type_range(user_type);
            UserTypeRangeInfo {
                user_type,
                type_name: user_type.name().to_string(),
                min_id,
                max_id,
                is_privileged: user_type.is_privileged(),
                is_temporary: user_type.is_temporary(),
                capacity: max_id - min_id + 1,
            }
        }).collect()
    }

    /// 验证用户ID是否有效
    pub fn is_valid_user_id(&self, user_id: i64) -> bool {
        UserIdPartition::is_valid_user_id(user_id)
    }
}

// =============== 用户ID分区相关结构体 ===============

/// 用户类型信息
#[derive(Debug, Serialize)]
pub struct UserTypeInfo {
    pub user_id: i64,
    pub user_type: UserType,
    pub sequence: i64,
    pub type_name: String,
    pub is_privileged: bool,
    pub is_temporary: bool,
}

/// 用户类型范围信息
#[derive(Debug, Serialize)]
pub struct UserTypeRangeInfo {
    pub user_type: UserType,
    pub type_name: String,
    pub min_id: i64,
    pub max_id: i64,
    pub is_privileged: bool,
    pub is_temporary: bool,
    pub capacity: i64,
}
