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

// 瞬间服务
pub mod moment_application_service;
pub use moment_application_service::*;

// 搜索服务
pub mod search_application_service;
pub use search_application_service::*;

// 增强搜索服务
pub mod enhanced_search_service;
pub use enhanced_search_service::*;

// 搜索优化服务
pub mod search_optimization_service;
pub use search_optimization_service::*;

// 内容安全服务
pub mod content_security_service;
pub use content_security_service::*;

// 搜索分析服务
pub mod search_analytics_service;
#[cfg(feature = "search_analytics")]
pub use search_analytics_service::*;

// 实时通知推送服务
pub mod realtime_notification_service;
pub use realtime_notification_service::*;

// 活动服务
pub mod activity_application_service;
pub use activity_application_service::*;

// 评论服务
pub mod comment_application_service;
pub use comment_application_service::*;

// 好友服务
pub mod friend_application_service;
pub use friend_application_service::*;

// 安全服务
pub mod security_application_service;
pub use security_application_service::*;

// 通知服务
pub mod notification_application_service;
pub use notification_application_service::*;

// WebSocket通知集成服务
pub mod websocket_notification_service;
pub use websocket_notification_service::*;

// 统一通知协调器
pub mod unified_notification_coordinator;
pub use unified_notification_coordinator::*;

// 缓存服务
pub mod cache_application_service;
pub use cache_application_service::*;

// 作业服务
pub mod job_application_service;
pub use job_application_service::*;

// 额外的服务类型已合并到各自的模块中
