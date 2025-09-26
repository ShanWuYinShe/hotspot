/// 通用响应类型定义
/// 
/// 根据需求文档的API类型命名规范，定义通用的响应类型
/// 使用type alias来避免重复定义相同结构的响应类型

use serde::Serialize;

/// 通用的成功/失败响应类型
#[derive(Debug, Serialize)]
pub struct SuccessResponse {
    pub success: bool,
    pub message: String,
}

/// 通用的删除操作响应类型
pub type DeleteResponse = SuccessResponse;

/// 通用的更新操作响应类型  
pub type UpdateResponse = SuccessResponse;

/// 通用的标记操作响应类型
pub type MarkResponse = SuccessResponse;

/// 通用的创建操作响应类型（包含ID）
#[derive(Debug, Serialize)]
pub struct CreateResponse {
    pub id: String,
    pub message: String,
    pub created_at: String,
}

/// 通用的计数响应类型
#[derive(Debug, Serialize)]
pub struct CountResponse {
    pub count: i32,
}

/// 通用的状态检查响应类型
#[derive(Debug, Serialize)]
pub struct StatusResponse {
    pub status: String,
    pub message: String,
}

/// 通用的列表项响应类型（用于简单列表）
#[derive(Debug, Serialize)]
pub struct ListItemResponse {
    pub id: String,
    pub name: String,
    pub created_at: String,
}

/// 通用的用户简要信息响应类型
#[derive(Debug, Serialize)]
pub struct UserBriefResponse {
    pub user_id: String,
    pub nickname: String,
    pub avatar_url: Option<String>,
}

/// 通用的时间戳响应类型
#[derive(Debug, Serialize)]
pub struct TimestampResponse {
    pub timestamp: String,
}
