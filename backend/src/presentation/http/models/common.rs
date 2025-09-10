/// 通用HTTP模型
/// 
/// 定义通用的请求/响应结构
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use utoipa::ToSchema;
use crate::utils::JsSafeI64;

/// 分页参数
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct Pagination {
    /// 游标，用于分页查询
    pub cursor: Option<JsSafeI64>,
    /// 每页数量，默认20，最大100
    pub limit: Option<JsSafeI64>,
}

impl Default for Pagination {
    fn default() -> Self {
        Self {
            cursor: None,
            limit: Some(JsSafeI64::new(20)),
        }
    }
}

/// 分页响应
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct PaginatedResponse<T> {
    /// 数据列表
    pub items: Vec<T>,
    /// 分页元信息
    pub pagination: PaginationMeta,
}

/// 分页元信息
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct PaginationMeta {
    /// 是否有更多数据
    pub has_more: bool,
    /// 下一页游标
    pub next_cursor: Option<JsSafeI64>,
}

/// 标准API响应
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct ApiResponse<T> {
    /// 响应码，0表示成功
    pub code: i32,
    /// 响应消息
    pub message: String,
    /// 响应数据
    pub data: Option<T>,
}

impl<T> ApiResponse<T> {
    /// 创建成功响应
    pub fn success(data: T) -> Self {
        Self {
            code: 0,
            message: "success".to_string(),
            data: Some(data),
        }
    }
    
    /// 创建成功响应（无数据）
    pub fn success_empty() -> ApiResponse<()> {
        ApiResponse {
            code: 0,
            message: "success".to_string(),
            data: Some(()),
        }
    }
    
    /// 创建错误响应
    pub fn error(code: i32, message: String) -> ApiResponse<()> {
        ApiResponse {
            code,
            message,
            data: None,
        }
    }
}

/// 位置信息
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct LocationInfo {
    /// 纬度
    pub latitude: f64,
    /// 经度
    pub longitude: f64,
    /// 位置名称
    pub location_name: String,
}

/// 时间戳信息
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct TimestampInfo {
    /// 创建时间
    pub created_at: DateTime<Utc>,
    /// 更新时间
    pub updated_at: Option<DateTime<Utc>>,
}
