use crate::utils::response::{PaginatedResponse, Pagination};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use utoipa::ToSchema;

// 动态API请求/响应模型
#[derive(Debug, Serialize, Deserialize)]
pub struct CreateMomentRequest {
    pub description: String,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub tags: Option<Vec<String>>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct CreateMomentResponse {
    pub moment_id: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct DeleteMomentRequest {
    pub moment_id: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct DeleteMomentResponse {
    pub success: bool,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SearchMomentByIdRequest {
    pub moment_id: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SearchMomentsByLocationRequest {
    pub latitude: f64,
    pub longitude: f64,
    pub radius: f64,
    #[serde(flatten)]
    pub pagination: Pagination,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct UserMomentsHistoryRequest {
    pub user_id: Option<i64>,
    #[serde(flatten)]
    pub pagination: Pagination,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SearchMomentsByTagsRequest {
    pub tags: Vec<String>,
    pub latitude: f64,
    pub longitude: f64,
    #[serde(flatten)]
    pub pagination: Pagination,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct LikeMomentRequest {
    pub moment_id: String,
}

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct LikeMomentResponse {
    pub success: bool,
    pub likes_count: i64,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct UnlikeMomentRequest {
    pub moment_id: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct UnlikeMomentResponse {
    pub success: bool,
    pub likes_count: i64,
}

#[derive(Debug, Serialize, Deserialize, Clone, ToSchema)]
pub struct MomentInfo {
    pub id: i64,
    pub moment_id: String,
    pub user_id: i64,
    pub nickname: String,
    pub description: Option<String>,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub distance: f64,
    pub created_at: DateTime<Utc>,
    pub tags: Vec<String>,
    pub likes_count: i64,
    pub comments_count: i64,
    pub liked_by_me: bool,
}

pub type UserMomentsHistoryResponse = PaginatedResponse<MomentInfo>;
pub type SearchMomentsByLocationResponse = PaginatedResponse<MomentInfo>;
pub type SearchMomentsByTagsResponse = PaginatedResponse<MomentInfo>;
