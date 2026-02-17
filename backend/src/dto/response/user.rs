//! 用户相关响应类型

use crate::types::enums::{
    Gender, PrivacyLevel, PrivateChatPermission, UserPresenceStatus, UserType,
};
use serde::{Deserialize, Serialize};

use super::common::{id_as_string, RelativeLocation};

/// 用户基础信息 - API 文档: UserInfo
/// 注意：不含 user_type（普通 API 不暴露管理员身份）
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserInfo {
    #[serde(serialize_with = "id_as_string::serialize")]
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
}

impl UserInfo {
    /// 从数据库字段创建 UserInfo
    pub fn new(user_id: i64, nickname: impl Into<String>, avatar_url: Option<String>) -> Self {
        Self {
            user_id,
            nickname: nickname.into(),
            avatar_url,
        }
    }
}

/// 用户关系信息 - API 文档: Relationship
/// 设计说明：不暴露 is_blocked_by（对方是否拉黑了我），被拉黑的用户不应感知到自己被拉黑
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Relationship {
    pub is_following: bool,   // 我是否关注了对方
    pub is_followed_by: bool, // 对方是否关注了我
    pub is_blocked: bool,     // 我是否拉黑了对方
}

/// 社交上下文（展示与当前用户的共同点）- API 文档: UserItem.social_context
#[derive(Debug, Serialize, Clone)]
pub struct SocialContext {
    pub mutual_friends_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mutual_friends_preview: Option<Vec<UserInfo>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub common_groups_count: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub common_tags: Option<Vec<String>>,
}

/// 用户统计数据 - API 文档: UserStats
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct UserStats {
    pub moments_count: i32,
    pub friends_count: i32,
    pub followers_count: i32,
    pub following_count: i32,
    pub groups_count: i32,
}

/// 用户统一数据类型 - API 文档: UserItem
/// 用于列表、搜索、详情、Feed 等所有场景
/// 根据是否有认证 token，返回不同字段：
/// - 基础字段（始终返回）: user, bio, tags, presence_status, last_online, location
/// - @auth-only 字段（仅登录态返回）: stats, relation, social_context
#[derive(Debug, Serialize, Clone)]
pub struct UserItem {
    pub user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    // 隐私受控字段（隐私限制时不返回，不暴露原因）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub presence_status: Option<UserPresenceStatus>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_online: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<RelativeLocation>,
    // @auth-only 字段（登录态返回）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub stats: Option<UserStats>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub relation: Option<Relationship>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub social_context: Option<SocialContext>,
}

/// 当前用户资料 - API 文档: MyProfile
#[derive(Debug, Serialize)]
pub struct MyProfile {
    #[serde(serialize_with = "id_as_string::serialize")]
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub user_type: UserType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub gender: Option<Gender>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub email: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub phone: Option<String>,
    pub location: RelativeLocation,
    pub presence_status: UserPresenceStatus,
    pub last_online: String,
    pub created_at: String,
    pub stats: UserStats,
}

/// 用户设置数据 - API 文档: UserSettings
#[derive(Debug, Serialize, Deserialize)]
pub struct UserSettingsData {
    // 隐私设置
    pub privacy_level: PrivacyLevel,
    pub location_visible: bool,
    pub presence_visible: bool,
    pub social_lists_visible: bool,
    pub groups_visible: bool,
    pub liked_moments_visible: bool,
    pub stats_visible: bool,
    // 社交设置
    pub allow_follow: bool,
    pub allow_private_chat: PrivateChatPermission,
    // 搜索设置
    pub searchable: bool,
    pub phone_searchable: bool,
    pub nickname_searchable: bool,
    pub location_searchable: bool,
}

/// 好友列表项 - API 文档: FriendItem
#[derive(Debug, Serialize)]
pub struct FriendItem {
    pub user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub presence_status: Option<UserPresenceStatus>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_online: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub remark: Option<String>,
    pub followed_at: String,
    pub followed_back_at: String,
}

/// 关注列表项 - API 文档: FollowingItem
#[derive(Debug, Serialize)]
pub struct FollowingItem {
    pub user: UserInfo,
    pub followed_at: String,
    pub is_mutual: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub presence_status: Option<UserPresenceStatus>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_online: Option<String>,
}

/// 粉丝列表项 - API 文档: FollowerItem
#[derive(Debug, Serialize)]
pub struct FollowerItem {
    pub user: UserInfo,
    pub followed_at: String,
    pub is_following_back: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub presence_status: Option<UserPresenceStatus>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_online: Option<String>,
}

/// 拉黑用户列表项 - API 文档: BlockedUserSearchItem
#[derive(Debug, Serialize)]
pub struct BlockedUserSearchItem {
    pub user: UserInfo,
    pub blocked_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
}
