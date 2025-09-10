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
pub use values::*;
pub use entities::*;
pub use events::*;
pub use services::*;
pub use constants::*;

/// 领域层结果类型
pub type DomainResult<T> = Result<T, DomainError>;

// === 分页相关类型 ===

/// 分页请求
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PaginationRequest {
    pub cursor: Option<i64>,
    pub limit: Option<i64>,
}

/// 分页响应
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PaginatedResponse<T> {
    pub items: Vec<T>,
    pub pagination: PaginationMeta,
}

/// 分页元数据
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PaginationMeta {
    pub has_more: bool,
    pub next_cursor: Option<i64>,
}