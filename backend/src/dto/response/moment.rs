//! 瞬间相关响应类型

use crate::types::enums::PrivacyLevel;
use serde::Serialize;

use super::common::{option_vec_id_as_string, RelativeLocation};
use super::feed::RecommendationContext;
use super::user::UserInfo;

/// 瞬间图片 - API 文档: MomentImage
#[derive(Debug, Serialize, Clone)]
pub struct MomentImage {
    pub url: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content_type: Option<String>,
}

/// 我的瞬间互动状态 - API 文档: MyMomentState
#[derive(Debug, Serialize, Clone)]
pub struct MyMomentState {
    pub is_liked: bool,
    pub is_favorited: bool,
}

/// 瞬间统一数据类型 - API 文档: MomentItem
/// 用于列表、搜索、详情、Feed 等所有场景
#[derive(Debug, Serialize, Clone)]
pub struct MomentItem {
    pub moment_id: String,
    pub description: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub images: Option<Vec<MomentImage>>,
    pub tags: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    #[serde(serialize_with = "option_vec_id_as_string::serialize")]
    pub mentions: Option<Vec<i64>>,
    pub created_at: String,
    pub likes_count: i32,
    pub comments_count: i32,
    pub shares_count: i32,
    pub user: UserInfo,
    pub location: RelativeLocation,
    // @owner-only（仅资源拥有者可见）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub privacy_level: Option<PrivacyLevel>,
    // @auth-only 字段（登录态返回）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub my_state: Option<MyMomentState>,
    // Feed 场景返回
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recommendation: Option<RecommendationContext>,
}

/// 瞬间点赞项 - API 文档: MomentLikeItem
#[derive(Debug, Serialize)]
pub struct MomentLikeItem {
    pub user: UserInfo,
    pub liked_at: String,
}
