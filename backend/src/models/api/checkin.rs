use crate::utils::response::{PaginatedResponse, Pagination};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use utoipa::ToSchema;

// 签到API请求/响应模型
#[derive(Debug, Serialize, Deserialize)]
pub struct CreateCheckinRequest {
    pub description: String,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub tags: Option<Vec<String>>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct CreateCheckinResponse {
    pub checkin_id: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct DeleteCheckinRequest {
    pub checkin_id: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct DeleteCheckinResponse {
    pub success: bool,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SearchCheckInByIdRequest {
    pub checkin_id: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SearchCheckinsByLocationRequest {
    pub latitude: f64,
    pub longitude: f64,
    pub radius: f64,
    #[serde(flatten)]
    pub pagination: Pagination,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct UserCheckinsHistoryRequest {
    pub user_id: Option<i64>,
    #[serde(flatten)]
    pub pagination: Pagination,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SearchCheckinsByTagsRequest {
    pub tags: Vec<String>,
    pub latitude: f64,
    pub longitude: f64,
    #[serde(flatten)]
    pub pagination: Pagination,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct LikeCheckinRequest {
    pub checkin_id: String,
}

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct LikeCheckinResponse {
    pub success: bool,
    pub likes_count: i64,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct UnlikeCheckinRequest {
    pub checkin_id: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct UnlikeCheckinResponse {
    pub success: bool,
    pub likes_count: i64,
}

#[derive(Debug, Serialize, Deserialize, Clone, ToSchema)]
pub struct CheckInInfo {
    pub id: i64,
    pub checkin_id: String,
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

pub type UserCheckinsHistoryResponse = PaginatedResponse<CheckInInfo>;
pub type SearchCheckinByLocationResponse = PaginatedResponse<CheckInInfo>;
pub type SearchCheckinsByTagsResponse = PaginatedResponse<CheckInInfo>;
