/// 查询模块 - 重新导出所有查询类型
use serde::{Deserialize, Serialize};
use crate::domain::Coordinates;

// === 基础查询类型 ===

/// 查询特征
pub trait Query {
    /// 查询结果类型
    type Result;
}

/// 分页请求参数
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaginationQuery {
    pub cursor: Option<i64>,
    pub limit: Option<usize>,
}

impl PaginationQuery {
    /// 创建默认分页查询
    pub fn default() -> Self {
        Self {
            cursor: None,
            limit: Some(20),
        }
    }
    
    /// 获取有效的限制数量
    pub fn effective_limit(&self) -> usize {
        self.limit.unwrap_or(20).min(100)
    }
    
    /// 获取限制数量
    pub fn get_limit(&self) -> i64 {
        self.effective_limit() as i64
    }
    
    /// 获取游标值
    pub fn get_cursor(&self) -> i64 {
        self.cursor.unwrap_or(0)
    }
}

/// 位置查询参数
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LocationQuery {
    /// 中心坐标
    pub center: Coordinates,
    /// 搜索半径（米）
    pub radius: f64,
}

impl LocationQuery {
    /// 创建位置查询
    pub fn new(center: Coordinates, radius: f64) -> Self {
        Self { center, radius }
    }
}

// 用户查询
pub mod user_queries;
pub use user_queries::*;

// 群组查询
pub mod group_queries;
pub use group_queries::*;

// 消息查询
pub mod message_queries;
pub use message_queries::*;

// 瞬间查询
pub mod moment_queries;
pub use moment_queries::*;

// 搜索查询
pub mod search_queries;
pub use search_queries::*;

// 活动查询
pub mod activity_queries;
pub use activity_queries::*;

// 额外的查询类型已合并到各自的模块中
