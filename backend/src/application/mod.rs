/// 应用层 - 用例编排和业务流程
/// 
/// 架构原则：
/// - 依赖领域层和端口层
/// - 实现CQRS模式（命令查询分离）
/// - 编排业务用例
/// - 处理事务边界
/// - 发布领域事件
pub mod commands;
pub mod queries;
pub mod handlers;
pub mod services;

// 重新导出核心类型
pub use commands::{
    Command, CreateTempUserCommand, RegisterUserCommand, UpdateUserProfileCommand,
    UpgradeUserCommand, UpdateUserLocationCommand, SetUserPasswordCommand, DeleteUserAccountCommand,
    UserRegistrationResult, UserCommandResult, LocationUpdateResult
};
pub use queries::{
    Query, PaginationQuery, LocationQuery, GetUserByIdQuery, SearchUsersByNameQuery,
    SearchUsersByIdQuery, SearchUsersByLocationQuery, CheckPasswordStatusQuery,
    SearchByGlobalQuery, SearchType, NearbyUser, UserSearchResult, PasswordStatus,
    GetRecentActivitiesQuery, ActivityType
};
pub use handlers::{CommandHandler, QueryHandler, EventHandler, UserView, GroupView, MessageView, PaginatedResult, PaginationMeta};
pub use services::{
    UserApplicationService, GroupApplicationService, /* MessageApplicationService, */ MomentApplicationService,
    SearchApplicationServiceImpl, SearchApplicationService, ActivityApplicationService, 
    /* FriendApplicationServiceImpl, */ CommentApplicationServiceImpl, UserApplicationServicePort,
    NotificationApplicationService, CacheApplicationService, JobApplicationService
};

/// 应用层结果类型
pub type ApplicationResult<T> = Result<T, ApplicationError>;

/// 应用层错误
#[derive(thiserror::Error, Debug)]
pub enum ApplicationError {
    #[error("领域错误: {0}")]
    Domain(#[from] crate::domain::DomainError),
    
    #[error("基础设施错误: {message}")]
    Infrastructure { message: String },
    
    #[error("并发冲突")]
    ConcurrencyConflict,
    
    #[error("事务失败")]
    TransactionFailed,
    
    #[error("事件发布失败")]
    EventPublishFailed,
}