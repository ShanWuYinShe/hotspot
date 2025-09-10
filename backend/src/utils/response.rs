/// HTTP 响应工具模块
/// 
/// 功能职责：
/// - 提供统一的 HTTP 响应格式和工具函数
/// - 实现标准化的 API 响应结构和错误处理
/// - 支持多种响应类型和数据格式
/// - 提供分页查询的响应封装和元数据管理
/// - 实现响应的性能优化和压缩支持
/// 
/// 响应类型：
/// - 成功响应：标准的成功操作响应格式
/// - 错误响应：统一的错误信息和状态码
/// - 分页响应：包含分页元数据的列表响应
/// - 文件响应：文件上传下载的响应处理
/// 
/// 响应格式：
/// - 统一的 JSON 响应结构：code/msg/data
/// - HTTP 状态码统一为 200，业务状态通过 code 字段区分
/// - 支持国际化的错误消息和提示信息
/// - 包含请求追踪 ID 和调试信息
/// 
/// 工具函数：
/// - ok_response：创建成功响应
/// - error_response：创建错误响应
/// - paginated_response：创建分页响应
/// - created_response：创建资源创建响应
use axum::Json;
use serde::{Deserialize, Serialize};
use utoipa::ToSchema;

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct BasicSuccess {
    pub success: bool,
}

// 从utils/error.rs中导入AppError
pub use crate::utils::error::AppError;

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct ApiResponse<T> {
    pub code: i32,       // 状态码，0表示成功，非0表示失败
    pub msg: String,     // 状态消息
    pub data: Option<T>, // 成功时为数据对象，失败时为null
}

impl<T> ApiResponse<T> {
    /// 创建错误响应
    pub fn error(code: i32, msg: &str) -> Self {
        ApiResponse {
            code,
            msg: msg.to_string(),
            data: None,
        }
    }

    /// 创建成功响应
    pub fn success(data: T) -> Self {
        ApiResponse {
            code: 0,
            msg: "success".to_string(),
            data: Some(data),
        }
    }
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
    pub limit: Option<i32>,  // 32位整数在JavaScript中是安全的
    /// 分页游标，用于获取下一页数据
    pub cursor: Option<crate::utils::JsSafeI64>,
}

/// 通用分页响应元数据
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct PaginationMeta {
    /// 是否有更多数据
    pub has_more: bool,
    /// 下一页游标
    pub next_cursor: Option<crate::utils::JsSafeI64>,
}

/// 通用分页响应结构体
/// 适用于所有需要分页的API响应
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct PaginatedResponse<T> {
    pub items: Vec<T>,
    #[serde(flatten)]
    pub pagination: PaginationMeta,
}
