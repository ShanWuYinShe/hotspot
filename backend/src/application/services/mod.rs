/// 应用服务模块 - 重新导出所有服务类型

// 基础服务
pub mod base_application_service;
pub use base_application_service::*;

// 用户服务
pub mod user_application_service;
pub use user_application_service::*;

// 群组服务
pub mod group_application_service;
pub use group_application_service::*;

// 消息服务
pub mod message_application_service;
pub use message_application_service::*;

// 瞬间服务
pub mod moment_application_service;
pub use moment_application_service::*;

// 搜索服务
pub mod search_application_service;
pub use search_application_service::*;

// 活动服务
pub mod activity_application_service;
pub use activity_application_service::*;

// 好友服务
pub mod friend_application_service;
pub use friend_application_service::*;

// 评论服务
pub mod comment_application_service;
pub use comment_application_service::*;

// 通知服务
pub mod notification_application_service;
pub use notification_application_service::*;

// 缓存服务
pub mod cache_application_service;
pub use cache_application_service::*;

// 作业服务
pub mod job_application_service;
pub use job_application_service::*;

// 屏蔽服务
pub mod block_application_service;
pub use block_application_service::*;

// 额外的服务类型已合并到各自的模块中
