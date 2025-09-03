/// 应用服务模块
/// 
/// 实现用例服务，协调领域对象和基础设施组件
/// 应用服务不包含业务逻辑，只负责协调和编排

pub mod user_application_service;
pub mod group_application_service;
pub mod message_application_service;
pub mod moment_application_service;
pub mod notification_application_service;
pub mod cache_application_service;
pub mod job_application_service;
pub mod search_application_service;
pub mod nearby_notification_service;
pub mod activity_application_service;
pub mod friend_application_service;

#[cfg(feature = "email_channel")]
pub mod email_job_handler;

// 重新导出应用服务类型
pub use user_application_service::*;
pub use group_application_service::*;
pub use message_application_service::*;
pub use moment_application_service::*;
pub use notification_application_service::*;
pub use cache_application_service::*;
pub use job_application_service::*;
pub use search_application_service::*;
pub use nearby_notification_service::*;
pub use activity_application_service::*;
pub use friend_application_service::*;

#[cfg(feature = "email_channel")]
pub use email_job_handler::*;

use crate::domain::DomainResult;
use async_trait::async_trait;

/// 应用服务基础trait
#[async_trait]
pub trait ApplicationService: Send + Sync {
    /// 服务名称
    fn service_name(&self) -> &'static str;
    
    /// 健康检查
    async fn health_check(&self) -> DomainResult<bool>;
}

