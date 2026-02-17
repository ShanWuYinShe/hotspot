//! 统一搜索 + Feed API

use serde::Deserialize;

use crate::dto::request::common::GeoPoint;
use crate::types::enums::{DismissTargetType, FeedSource, GroupRole};

#[derive(Debug, Deserialize)]
pub struct FeedMomentsRequest {
    pub source: FeedSource,
    pub reference_point: GeoPoint,
    pub radius: Option<i32>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct FeedGroupsRequest {
    pub reference_point: GeoPoint,
    pub radius: Option<i32>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct FeedUsersRequest {
    pub reference_point: GeoPoint,
    pub radius: Option<i32>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct FeedTrendingRequest {}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize, Hash)]
#[serde(rename_all = "snake_case")]
pub enum SearchSuggestType {
    Tag,
    User,
    Group,
    Moment,
}

#[derive(Debug, Deserialize)]
pub struct SearchSuggestRequest {
    pub prefix: String,
    pub reference_point: GeoPoint,
    pub types: Option<Vec<SearchSuggestType>>,
    pub radius: Option<i32>,
    pub limit: Option<i32>,
}

/// 统一用户搜索请求 - POST /api/v1/users/search
#[derive(Debug, Deserialize)]
pub struct SearchUsersRequest {
    pub keyword: Option<String>,
    pub tag: Option<String>,
    pub radius: Option<f64>,
    pub sort_by: Option<String>,
    pub reference_point: GeoPoint,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

/// 统一群组搜索请求 - POST /api/v1/groups/search
#[derive(Debug, Deserialize)]
pub struct SearchGroupsRequest {
    pub keyword: Option<String>,
    pub tag: Option<String>,
    pub radius: Option<f64>,
    pub sort_by: Option<String>,
    pub reference_point: GeoPoint,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

/// 统一瞬间搜索请求 - POST /api/v1/moments/search
#[derive(Debug, Deserialize)]
pub struct SearchMomentsRequest {
    pub keyword: Option<String>,
    pub tag: Option<String>,
    pub radius: Option<f64>,
    pub sort_by: Option<String>,
    pub reference_point: GeoPoint,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

/// 群组成员搜索请求 - POST /api/v1/groups/members/search
#[derive(Debug, Deserialize)]
pub struct SearchGroupMembersRequest {
    pub group_id: String,
    pub keyword: String,
    pub role: Option<GroupRole>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct DismissFeedItemRequest {
    pub target_type: DismissTargetType,
    pub target_id: String,
}
