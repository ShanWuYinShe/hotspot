//! 数据仓库接口定义

use uuid::Uuid;
use chrono::{DateTime, Utc};
use crate::services::group::GroupSearchResult;

// 导出子模块
pub mod group_repository;
pub mod user_repository;
pub mod message_repository;
pub mod moment_repository;
pub mod health_repository;

#[cfg(feature = "jobs")]
pub mod scheduler_repository;

// 从子模块中重新导出具体实现
pub use group_repository::PgGroupRepository;
pub use user_repository::PgUserRepository;
pub use message_repository::PgMessageRepository;
pub use moment_repository::PgMomentRepository;
pub use health_repository::PgHealthRepository;


/// 仓库错误类型
#[derive(Debug, thiserror::Error)]
pub enum RepositoryError {
    #[error("数据库错误: {0}")]
    Database(#[from] sqlx::Error),
    #[error("UUID解析错误: {0}")]
    UuidParse(#[from] uuid::Error),
    #[error("未找到记录")]
    NotFound,
    #[error("参数无效: {0}")]
    InvalidArgument(String),
    #[error("其他仓库错误: {0}")]
    Other(String),
}

/// 仓库结果类型
pub type RepositoryResult<T> = Result<T, RepositoryError>;

/// 通用仓库接口
pub trait Repository: Send + Sync {
    /// 获取仓库名称
    #[allow(dead_code)]
    fn name(&self) -> &str;
}

/// 群组仓库接口
pub trait GroupRepository: Repository {
    /// 查找不活跃的群组成员
    #[cfg(feature = "group_digest_scheduler")]
    fn find_inactive_members(
        &self,
        group_id: Uuid,
        min_inactivity_days: i32
    ) -> impl std::future::Future<Output = RepositoryResult<Vec<(i64, String)>>> + Send;
    /// 生成群组消息摘要
    #[cfg(feature = "group_digest_scheduler")]
    fn generate_message_digest(
        &self,
        group_id: Uuid,
        last_active_at: i64
    ) -> impl std::future::Future<Output = RepositoryResult<String>> + Send;
    /// 更新群组成员最后活跃时间
    #[allow(dead_code)]
    fn update_member_last_active(
        &self,
        group_id: Uuid,
        user_id: i64,
        timestamp: DateTime<Utc>
    ) -> impl std::future::Future<Output = RepositoryResult<()>> + Send;
    /// 查找需要发送摘要的群组
    #[cfg(feature = "group_digest_scheduler")]
    fn find_groups_needing_digest(
        &self,
        activity_days: i32,
        inactivity_days: i32,
        limit: i32
    ) -> impl std::future::Future<Output = RepositoryResult<Vec<(Uuid, String, DateTime<Utc>)>>> + Send;
    /// 检查群组是否存在
    fn check_group_exists(&self, group_id: Uuid) -> impl std::future::Future<Output = RepositoryResult<bool>> + Send;
    /// 检查用户是否为群组所有者
    fn check_user_is_group_owner(&self, group_id: Uuid, user_id: i64) -> impl std::future::Future<Output = RepositoryResult<bool>> + Send;
    /// 检查用户是否在群组中
    fn check_user_in_group(&self, group_id: Uuid, user_id: i64) -> impl std::future::Future<Output = RepositoryResult<bool>> + Send;
    /// 检查用户是否是群组管理员
    fn check_user_is_group_admin(&self, group_id: Uuid, user_id: i64) -> impl std::future::Future<Output = RepositoryResult<bool>> + Send;
    /// 获取群组密码哈希
    fn get_group_password_hash(&self, group_id: Uuid) -> impl std::future::Future<Output = RepositoryResult<Option<String>>> + Send;
    /// 验证群组密码
    fn verify_group_password(&self, group_id: Uuid, password: &str) -> impl std::future::Future<Output = RepositoryResult<bool>> + Send;
    /// 检查群组是否需要密码
    fn check_group_need_password(&self, group_id: Uuid) -> impl std::future::Future<Output = RepositoryResult<bool>> + Send;
    /// 获取群组管理员数量
    #[cfg(feature = "group_digest_scheduler")]
    fn count_group_admins(&self, group_id: Uuid) -> impl std::future::Future<Output = RepositoryResult<i64>> + Send;
    /// 获取群组成员数量
    #[cfg(feature = "group_digest_scheduler")]
    fn count_group_members(&self, group_id: Uuid) -> impl std::future::Future<Output = RepositoryResult<i64>> + Send;
    /// 哈希密码
    fn hash_password(&self, password: &str) -> impl std::future::Future<Output = RepositoryResult<String>> + Send;
    /// 创建群组
    fn create_group(
        &self,
        name: &str,
        description: Option<&str>,
        owner_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: &str,
        password: Option<&str>
    ) -> impl std::future::Future<Output = RepositoryResult<Uuid>> + Send;
    /// 加入群组
    fn join_group(
        &self,
        group_id: Uuid,
        user_id: i64
    ) -> impl std::future::Future<Output = RepositoryResult<(bool, String)>> + Send;
    /// 离开群组
    fn leave_group(
        &self,
        group_id: Uuid,
        user_id: i64
    ) -> impl std::future::Future<Output = RepositoryResult<bool>> + Send;
    /// 踢出成员
    fn kick_member(
        &self,
        group_id: Uuid,
        user_id: i64,
        admin_id: i64
    ) -> impl std::future::Future<Output = RepositoryResult<bool>> + Send;
    /// 转让群主
    fn transfer_ownership(
        &self,
        group_id: Uuid,
        new_owner_id: i64,
        current_owner_id: i64
    ) -> impl std::future::Future<Output = RepositoryResult<bool>> + Send;
    /// 获取群组成员列表
    fn get_group_members(
        &self,
        group_id: Uuid,
        cursor: i64,
        limit: i64
    ) -> impl std::future::Future<Output = RepositoryResult<(Vec<(i64, i64, String, DateTime<Utc>, String, DateTime<Utc>)>, bool, Option<i64>)>> + Send;
    /// 根据位置搜索群组
    fn search_group_by_location(
        &self,
        user_id: i64,
        radius: f64,
        cursor: i64,
        limit: i64
    ) -> impl std::future::Future<Output = RepositoryResult<(Vec<GroupSearchResult>, bool, Option<i64>)>> + Send;
    /// 根据名称搜索群组
    fn search_group_by_name(
        &self,
        user_id: i64,
        keyword: &str,
        cursor: i64,
        limit: i64
    ) -> impl std::future::Future<Output = RepositoryResult<(Vec<GroupSearchResult>, bool, Option<i64>)>> + Send;
    /// 根据ID获取群组
    fn search_group_by_id(
        &self,
        user_id: i64,
        group_id: Uuid
    ) -> impl std::future::Future<Output = RepositoryResult<GroupSearchResult>> + Send;
    
    /// 获取用户加入的群组列表
    fn get_user_groups(
        &self,
        user_id: i64,
        limit: Option<i64>,
        offset: Option<i64>
    ) -> impl std::future::Future<Output = RepositoryResult<Vec<GroupSearchResult>>> + Send;
    
    /// 获取用户群组总数
    fn count_user_groups(
        &self,
        user_id: i64
    ) -> impl std::future::Future<Output = RepositoryResult<i64>> + Send;
    
    /// 获取用户创建的群组列表
    fn get_owned_groups(
        &self,
        user_id: i64,
        limit: Option<i64>,
        offset: Option<i64>
    ) -> impl std::future::Future<Output = RepositoryResult<Vec<GroupSearchResult>>> + Send;
}
