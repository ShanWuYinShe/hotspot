/// 瞬间相关HTTP模型
/// 
/// 迁移自models/api/moment.rs，适配新架构
use super::{PaginatedResponse, LocationInfo, TimestampInfo, Pagination};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use utoipa::ToSchema;
use crate::utils::JsSafeI64;

/// 创建瞬间请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct CreateMomentRequest {
    pub description: String,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub tags: Option<Vec<String>>,
}

/// 创建瞬间响应
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct CreateMomentResponse {
    pub moment_id: String,
}

/// 删除瞬间请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct DeleteMomentRequest {
    pub moment_id: String,
}

/// 删除瞬间响应
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct DeleteMomentResponse {
    pub success: bool,
}

/// 根据ID搜索瞬间请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct SearchMomentByIdRequest {
    pub moment_id: String,
}

/// 根据位置搜索瞬间请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct SearchMomentsByLocationRequest {
    pub latitude: f64,
    pub longitude: f64,
    pub radius: f64,
    #[serde(flatten)]
    pub pagination: Pagination,
}

/// 用户瞬间历史请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct UserMomentsHistoryRequest {
    pub user_id: Option<JsSafeI64>,
    #[serde(flatten)]
    pub pagination: Pagination,
}

/// 根据标签搜索瞬间请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct SearchMomentsByTagsRequest {
    pub tags: Vec<String>,
    pub latitude: f64,
    pub longitude: f64,
    #[serde(flatten)]
    pub pagination: Pagination,
}

/// 点赞瞬间请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct LikeMomentRequest {
    pub moment_id: String,
}

/// 点赞瞬间响应
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct LikeMomentResponse {
    pub success: bool,
    pub likes_count: JsSafeI64,
}

/// 取消点赞瞬间请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct UnlikeMomentRequest {
    pub moment_id: String,
}

/// 取消点赞瞬间响应
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct UnlikeMomentResponse {
    pub success: bool,
    pub likes_count: JsSafeI64,
}

/// 瞬间信息
#[derive(Debug, Serialize, Deserialize, Clone, ToSchema)]
pub struct MomentInfo {
    pub id: JsSafeI64,
    pub moment_id: String,
    pub user_id: JsSafeI64,
    pub user_nickname: String,
    pub description: String,
    pub likes_count: JsSafeI64,
    pub comments_count: i32,  // 32位整数在JavaScript中是安全的
    pub is_liked: bool,
    #[serde(flatten)]
    pub location: LocationInfo,
    #[serde(flatten)]
    pub timestamps: TimestampInfo,
    pub tags: Vec<String>,
}

/// 瞬间详情
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct MomentDetail {
    #[serde(flatten)]
    pub info: MomentInfo,
    /// 最近的评论
    pub recent_comments: Vec<MomentComment>,
}

/// 瞬间评论
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct MomentComment {
    pub id: JsSafeI64,
    pub comment_id: String,
    pub moment_id: String,
    pub user_id: JsSafeI64,
    pub user_nickname: String,
    pub content: String,
    pub created_at: DateTime<Utc>,
}

/// 添加瞬间评论请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct AddMomentCommentRequest {
    pub moment_id: String,
    pub content: String,
}

/// 添加瞬间评论响应
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct AddMomentCommentResponse {
    pub comment_id: String,
    pub created_at: DateTime<Utc>,
}

/// 瞬间评论列表请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct MomentCommentsRequest {
    pub moment_id: String,
    #[serde(flatten)]
    pub pagination: Pagination,
}

/// 瞬间评论列表响应
pub type MomentCommentsResponse = PaginatedResponse<MomentComment>;

/// 删除瞬间评论请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct DeleteMomentCommentRequest {
    pub comment_id: String,
}

/// 附近瞬间响应
pub type NearbyMomentsResponse = PaginatedResponse<MomentInfo>;

/// 用户瞬间历史响应
pub type UserMomentsResponse = PaginatedResponse<MomentInfo>;

/// 标签瞬间搜索响应
pub type TagMomentsResponse = PaginatedResponse<MomentInfo>;
