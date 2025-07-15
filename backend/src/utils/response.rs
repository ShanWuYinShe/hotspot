use axum::Json;
use serde::{Deserialize, Serialize};
use utoipa::ToSchema;

// 从utils/error.rs中导入AppError
pub use crate::utils::error::AppError;

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct ApiResponse<T> {
    pub code: i32,       // 状态码，0表示成功，非0表示失败
    pub msg: String,     // 状态消息
    pub data: Option<T>, // 成功时为数据对象，失败时为null
}

/// 创建标准成功响应
/// 设置code为0，msg为"success"，并将数据放入data字段
pub fn success_response<T: Serialize>(data: T) -> Json<ApiResponse<T>> {
    Json(ApiResponse {
        code: 0,
        msg: "success".to_string(),
        data: Some(data),
    })
}

/// 通用分页参数
/// 用于所有需要分页功能的API请求
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct Pagination {
    /// 每页限制数量，默认为20，最大50
    pub limit: Option<i64>,
    /// 分页游标，用于获取下一页数据
    pub cursor: Option<i64>,
}

/// 通用分页响应元数据
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct PaginationMeta {
    /// 是否有更多数据
    pub has_more: bool,
    /// 下一页游标
    pub next_cursor: Option<i64>,
}

/// 通用分页响应结构体
/// 适用于所有需要分页的API响应
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct PaginatedResponse<T> {
    pub items: Vec<T>,
    #[serde(flatten)]
    pub pagination: PaginationMeta,
}
