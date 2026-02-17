//! 群组相关响应类型

use crate::types::enums::GroupJoinMethod;
use serde::{Deserialize, Serialize};

use super::common::RelativeLocation;
use super::user::UserInfo;

/// 群组基础信息 - API 文档: GroupInfo
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct GroupInfo {
    pub group_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub members_count: i32,
}

/// 我在群内的状态 - API 文档: Membership
#[derive(Debug, Serialize)]
pub struct GroupMyStateData {
    pub role: crate::types::enums::GroupRole,
    pub is_joined: bool,
    pub unread_count: i32,
    pub is_muted: bool,
    pub is_pinned: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub joined_at: Option<String>,
}

/// 群组设置 - API 文档: GroupSettings
#[derive(Debug, Serialize, Deserialize)]
pub struct GroupSettings {
    pub join_method: GroupJoinMethod,
    pub is_searchable: bool,
    pub max_members: i32,
    pub location: crate::dto::request::Location,
}

/// 群组统一数据类型 - API 文档: GroupItem
/// 用于列表、搜索、详情、Feed 等所有场景
#[derive(Debug, Serialize)]
pub struct GroupItem {
    pub group: GroupInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    pub has_password: bool,
    pub location: RelativeLocation,
    // 列表/详情页返回
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_activity_at: Option<String>,
    // 详情页返回，搜索/列表不返回
    #[serde(skip_serializing_if = "Option::is_none")]
    pub created_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub owner_id: Option<String>,
    // === @auth-only ===
    #[serde(skip_serializing_if = "Option::is_none")]
    pub my_state: Option<GroupMyStateData>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub friends_in_group: Option<i32>,
    // === @group-admin-only ===
    #[serde(skip_serializing_if = "Option::is_none")]
    pub join_method: Option<GroupJoinMethod>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_searchable: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max_members: Option<i32>,
}

/// 群公告项 - API 文档: GroupAnnouncementItem
#[derive(Debug, Serialize)]
pub struct GroupAnnouncementItem {
    pub announcement_id: String,
    pub group_id: String,
    pub title: String,
    pub content: String,
    pub created_by: UserInfo,
    pub created_at: String,
    pub is_pinned: bool,
}
