//! Repository 端口定义（抽象接口，供 application 层依赖）

use std::future::Future;
use std::pin::Pin;
use uuid::Uuid;

#[derive(Debug, thiserror::Error)]
pub enum RepoPortError {
    #[error("database error: {0}")]
    Database(String),
    #[error("not found: {0}")]
    NotFound(String),
    #[error("invalid argument: {0}")]
    InvalidArgument(String),
    #[error("connection error: {0}")]
    Connection(String),
    #[error("timeout error: {0}")]
    Timeout(String),
    #[error("other: {0}")]
    Other(String),
}

impl RepoPortError {
    /// 从数据库错误创建
    pub fn from_db_error(err: impl std::fmt::Display) -> Self {
        Self::Database(err.to_string())
    }

    /// 创建未找到错误
    pub fn not_found(resource: &str) -> Self {
        Self::NotFound(resource.to_string())
    }

    /// 创建无效参数错误
    pub fn invalid_argument(msg: &str) -> Self {
        Self::InvalidArgument(msg.to_string())
    }

    // 从仓储错误转换方法已删除，因为传统repositories模块已迁移
}

pub type RepoPortResult<T> = Result<T, RepoPortError>;

pub trait GroupRepoPort: Send + Sync {
    fn name(&self) -> &str;
    fn check_group_exists(&self, group_id: Uuid) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>>;
    fn check_user_in_group(&self, group_id: Uuid, user_id: i64) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>>;
}

/// 用户仓储端口 - 定义用户相关的最小必要操作
pub trait UserRepoPort: Send + Sync {
    fn name(&self) -> &str;
    /// 检查登录ID是否已被使用
    fn is_login_id_taken(&self, login_id: &str) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>>;
    /// 检查用户是否设置了密码
    fn is_password_set(&self, user_id: i64) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>>;
    /// 检查用户是否为临时用户
    fn is_temp_user(&self, user_id: i64) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>>;
}

/// 动态仓储端口 - 定义动态相关的最小必要操作
pub trait MomentRepoPort: Send + Sync {
    fn name(&self) -> &str;
    /// 点赞动态
    fn like_moment(&self, moment_id: &str, user_id: i64) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>>;
    /// 取消点赞动态
    fn unlike_moment(&self, moment_id: &str, user_id: i64) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>>;
}


