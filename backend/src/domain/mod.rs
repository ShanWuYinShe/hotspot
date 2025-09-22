/// 领域层 - 纯业务逻辑，零外部依赖
/// 
/// 架构原则：
/// - 不依赖任何外部框架或技术实现
/// - 包含核心业务规则和领域知识
/// - 定义领域事件和业务不变量
/// - 提供领域服务和聚合根

pub mod errors;
pub mod values;
pub mod entities;
pub mod events;
pub mod services;
pub mod constants;

// 重新导出核心类型
pub use errors::*;
// 具体导出values模块以避免冲突
pub use values::{
    UserId, LoginId, Password, UserNickname, Location, Coordinates,
    Tag, TagSet, Nickname, Description, // content模块的类型
    GroupId, GroupRole, GroupType, GroupVisibility,
    MomentId, MessageId,
    PaginationRequest, PaginationMeta, PaginatedResponse,
};
// 具体导出entities模块以避免冲突
pub use entities::{
    User, UserProfile, UserDetail, UserAuth, UserSearchResult,
    Group, GroupProfile, GroupAuth, GroupDetail, GroupSearchResult, GroupMember,
    Moment,
    Comment,
    GroupRole as EntityGroupRole, // 使用entities中的GroupRole
    Tag as EntityTag, // 使用entities中的Tag
};
pub use events::*;
pub use services::*;
pub use constants::*;

/// 领域层结果类型
pub type DomainResult<T> = Result<T, DomainError>;

// === 分页相关类型 ===
// 分页类型已从values模块导入，无需重复定义