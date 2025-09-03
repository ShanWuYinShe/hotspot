/// CQRS 查询模块
/// 
/// 定义系统中所有的读操作查询和查询处理器

pub mod user_queries;
pub mod group_queries;
pub mod message_queries;
pub mod moment_queries;
pub mod search_queries;
pub mod activity_queries;

// 重新导出查询类型
pub use user_queries::*;
pub use group_queries::*;
pub use message_queries::*;
pub use moment_queries::*;
pub use search_queries::*;
pub use activity_queries::*;

use crate::domain::DomainResult;
use async_trait::async_trait;

/// 查询处理器基础trait
#[async_trait]
pub trait QueryHandler<TQuery> {
    type Result;
    
    /// 处理查询
    async fn handle_query(&self, query: TQuery) -> DomainResult<Self::Result>;
}

/// 分页查询参数
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PaginationQuery {
    pub cursor: Option<i64>,
    pub limit: Option<i32>,
}

impl Default for PaginationQuery {
    fn default() -> Self {
        Self {
            cursor: None,
            limit: Some(20),
        }
    }
}

impl PaginationQuery {
    /// 创建默认分页查询
    pub fn new() -> Self {
        Self::default()
    }
    
    /// 获取实际的limit值
    pub fn get_limit(&self) -> i32 {
        self.limit.unwrap_or(20).clamp(1, 100) // 限制在1-100之间
    }
    
    /// 获取实际的cursor值
    pub fn get_cursor(&self) -> i64 {
        self.cursor.unwrap_or(0)
    }
}

/// 分页查询结果
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PaginatedResult<T> {
    pub items: Vec<T>,
    pub has_more: bool,
    pub next_cursor: Option<i64>,
}

impl<T> PaginatedResult<T> {
    /// 创建分页结果
    pub fn new(items: Vec<T>, has_more: bool, next_cursor: Option<i64>) -> Self {
        Self {
            items,
            has_more,
            next_cursor,
        }
    }
    
    /// 创建空结果
    pub fn empty() -> Self {
        Self {
            items: Vec::new(),
            has_more: false,
            next_cursor: None,
        }
    }
}
