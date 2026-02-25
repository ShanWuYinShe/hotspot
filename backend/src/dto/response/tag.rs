//! 标签相关响应类型

use serde::Serialize;

use super::common::{id_as_string, RelativeLocation};
use super::user::UserInfo;

/// 标签建议项 - API 文档: TagSuggestItem
#[derive(Debug, Serialize)]
pub struct TagSuggestItem {
    pub name: String,
    pub usage_count: i32,
}

/// 综合搜索建议数据 - API 文档: SearchSuggestions
#[derive(Debug, Serialize)]
pub struct SearchSuggestions {
    pub items: Vec<SearchSuggestItem>,
}

/// 综合搜索建议项 - API 规范: @api_spec_search.md SearchSuggestItem
#[derive(Debug, Serialize)]
#[serde(tag = "type", content = "payload", rename_all = "snake_case")]
pub enum SearchSuggestItem {
    Tag(TagSuggestItem),
    User(UserSuggestItem),
    Group(GroupSuggestItem),
    Moment(MomentSuggestItem),
}

/// 用户建议项 - API 规范: @api_spec_search.md UserSuggestItem
/// 用途：显示用户基本信息，点击后跳转到用户主页
#[derive(Debug, Serialize)]
pub struct UserSuggestItem {
    #[serde(serialize_with = "id_as_string::serialize")]
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    /// 个性签名，用于副标题展示
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    /// 相对位置，仅当请求包含 reference_point 时返回
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<RelativeLocation>,
}

/// 群组建议项 - API 规范: @api_spec_search.md GroupSuggestItem
/// 用途：显示群组基本信息，点击后跳转到群组页
#[derive(Debug, Serialize)]
pub struct GroupSuggestItem {
    pub group_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    /// 群组简介，用于副标题展示（优先于成员数）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    /// 成员数量，当无 description 时显示"X 位成员"
    pub members_count: i32,
    /// 相对位置，仅当请求包含 reference_point 时返回
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<RelativeLocation>,
}

/// 瞬间建议项 - API 规范: @api_spec_search.md MomentSuggestItem
/// 用途：显示瞬间预览信息，点击后跳转到瞬间详情页
#[derive(Debug, Serialize)]
pub struct MomentSuggestItem {
    pub moment_id: String,
    /// 瞬间文字预览（后端建议限制长度，如前100字符）
    pub description_preview: String,
    /// 作者信息
    pub author: UserInfo,
    /// 相对位置，仅当请求包含 reference_point 时返回
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<RelativeLocation>,
}

/// 管理员标签项 - API 文档: AdminTagItem
#[derive(Debug, Serialize)]
pub struct AdminTagItem {
    pub name: String,
    pub usage_count: i32,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_used_at: Option<String>,
}
