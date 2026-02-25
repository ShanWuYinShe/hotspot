//! 生活瞬间 API

use serde::Deserialize;

use crate::dto::request::common::{Location, PaginationParams};
use crate::types::enums::PrivacyLevel;

#[derive(Debug, Deserialize)]
pub struct CreateMomentRequest {
    pub description: String,
    pub location: Location,
    pub tags: Option<Vec<String>>,
    pub image_file_ids: Option<Vec<String>>,
    pub privacy_level: Option<PrivacyLevel>,
    pub mentions: Option<Vec<String>>,
}

#[derive(Debug, Deserialize)]
pub struct EditMomentRequest {
    pub moment_id: String,
    pub description: Option<String>,
    pub image_file_ids: Option<Vec<String>>,
    pub tags: Option<Vec<String>>,
    pub location: Option<Location>,
    pub privacy_level: Option<PrivacyLevel>,
    pub mentions: Option<Vec<String>>,
}

#[derive(Debug, Deserialize)]
pub struct FavoriteMomentRequest {
    pub moment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct UnfavoriteMomentRequest {
    pub moment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct MyFavoritesRequest {
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct MomentOpenTheDoorRequest {
    pub moment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct UserMomentsRequest {
    pub user_id: String,
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct MyMomentsRequest {
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteMomentRequest {
    pub moment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct LikeMomentRequest {
    pub moment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct UnlikeMomentRequest {
    pub moment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct MomentLikesRequest {
    pub moment_id: String,
    pub pagination: Option<PaginationParams>,
}
