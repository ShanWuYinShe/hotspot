//! 用户模块请求

use serde::Deserialize;

use crate::dto::request::common::{Location, PaginationParams};
use crate::types::enums::{FollowSource, Gender, PrivacyLevel, PrivateChatPermission};

#[derive(Debug, Deserialize)]
pub struct MyProfileRequest {}

#[derive(Debug, Deserialize)]
pub struct GetRelationshipRequest {
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct ListFriendsRequest {
    pub user_id: Option<String>,
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct ListFollowingRequest {
    pub user_id: Option<String>,
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct ListFollowersRequest {
    pub user_id: Option<String>,
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct UserGroupsRequest {
    pub user_id: String,
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct UserLikedMomentsRequest {
    pub user_id: String,
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct UserOpenTheDoorRequest {
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct FollowRequest {
    pub user_id: String,
    pub source: Option<FollowSource>,
    pub source_moment_id: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UnfollowRequest {
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct BlockRequest {
    pub user_id: String,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UnblockRequest {
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct MyBlockedRequest {
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateProfileRequest {
    pub nickname: Option<String>,
    pub bio: Option<String>,
    pub gender: Option<Gender>,
    pub avatar_file_id: Option<String>,
    pub tags: Option<Vec<String>>,
    pub privacy_level: Option<PrivacyLevel>,
    pub clear_bio: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateLocationRequest {
    pub location: Location,
}

#[derive(Debug, Deserialize)]
pub struct UpgradeAccountRequest {
    pub login_id: String,
    pub password: String,
}

#[derive(Debug, Deserialize)]
pub struct DeleteAccountRequest {
    pub confirmation: String,
}

#[derive(Debug, Deserialize)]
pub struct MySettingsRequest {}

#[derive(Debug, Deserialize)]
pub struct UserSettingsPayload {
    pub privacy_level: Option<PrivacyLevel>,
    pub location_visible: Option<bool>,
    pub presence_visible: Option<bool>,
    pub social_lists_visible: Option<bool>,
    pub groups_visible: Option<bool>,
    pub liked_moments_visible: Option<bool>,
    pub stats_visible: Option<bool>,
    pub allow_follow: Option<bool>,
    pub allow_private_chat: Option<PrivateChatPermission>,
    pub searchable: Option<bool>,
    pub phone_searchable: Option<bool>,
    pub nickname_searchable: Option<bool>,
    pub location_searchable: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateSettingsRequest {
    pub settings: UserSettingsPayload,
}

#[derive(Debug, Deserialize)]
pub struct SetRemarkRequest {
    pub user_id: String,
    pub remark: String,
}
