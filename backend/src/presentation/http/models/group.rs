/// 群组相关HTTP模型
/// 
/// 迁移自models/api/group.rs，适配新架构

use super::{PaginatedResponse, LocationInfo, TimestampInfo, Pagination};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use utoipa::ToSchema;
use crate::utils::JsSafeI64;

/// 创建群组请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct CreateGroupRequest {
    /// 群组名称
    pub name: String,
    /// 群组描述
    pub description: Option<String>,
    /// 位置信息
    #[serde(flatten)]
    pub location: LocationInfo,
    /// 群组标签
    pub tags: Option<Vec<String>>,
}

/// 创建群组响应
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct CreateGroupResponse {
    pub group_id: String,
    pub name: String,
    pub created_at: DateTime<Utc>,
}

/// 群组信息
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct GroupInfo {
    pub id: JsSafeI64,
    pub group_id: String,
    pub name: String,
    pub description: Option<String>,
    pub creator_id: JsSafeI64,
    pub creator_nickname: String,
    pub member_count: JsSafeI64,
    pub distance: f64,
    #[serde(flatten)]
    pub location: LocationInfo,
    #[serde(flatten)]
    pub timestamps: TimestampInfo,
    pub tags: Vec<String>,
}

/// 群组详情
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct GroupDetail {
    #[serde(flatten)]
    pub info: GroupInfo,
    /// 用户在群组中的角色
    pub user_role: Option<String>,
    /// 是否已加入
    pub is_member: bool,
}

/// 更新群组请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct UpdateGroupRequest {
    /// 群组名称
    pub name: Option<String>,
    /// 群组描述
    pub description: Option<String>,
    /// 位置信息
    #[serde(flatten)]
    pub location: Option<LocationInfo>,
    /// 群组标签
    pub tags: Option<Vec<String>>,
}

/// 加入群组请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct JoinGroupRequest {
    pub group_id: String,
}

/// 加入群组响应
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct JoinGroupResponse {
    pub group_id: String,
    pub joined_at: DateTime<Utc>,
    pub role: String,
}

/// 离开群组请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct LeaveGroupRequest {
    pub group_id: String,
}

/// 附近群组查询请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct NearbyGroupsRequest {
    /// 纬度
    pub latitude: f64,
    /// 经度
    pub longitude: f64,
    /// 搜索半径（米）
    pub radius: f64,
    /// 分页参数
    #[serde(flatten)]
    pub pagination: Pagination,
}

/// 附近群组响应
pub type NearbyGroupsResponse = PaginatedResponse<GroupInfo>;

/// 群组搜索请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct SearchGroupsRequest {
    /// 搜索关键词
    pub keyword: String,
    /// 位置过滤
    #[serde(flatten)]
    pub location: Option<LocationInfo>,
    /// 搜索半径（米）
    pub radius: Option<f64>,
    /// 分页参数
    #[serde(flatten)]
    pub pagination: Pagination,
}

/// 群组搜索响应
pub type SearchGroupsResponse = PaginatedResponse<GroupInfo>;

/// 我的群组请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct MyGroupsRequest {
    /// 分页参数
    #[serde(flatten)]
    pub pagination: Pagination,
}

/// 我的群组响应
pub type MyGroupsResponse = PaginatedResponse<GroupInfo>;

/// 群组成员信息
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct GroupMember {
    pub user_id: i64,
    pub nickname: String,
    pub role: String,
    pub joined_at: DateTime<Utc>,
    pub last_active_at: DateTime<Utc>,
}

/// 群组成员列表请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct GroupMembersRequest {
    pub group_id: String,
    /// 分页参数
    #[serde(flatten)]
    pub pagination: Pagination,
}

/// 群组成员列表响应
pub type GroupMembersResponse = PaginatedResponse<GroupMember>;

/// 邀请用户加入群组请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct InviteUserRequest {
    pub group_id: String,
    pub user_id: i64,
}

/// 移除群组成员请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct RemoveMemberRequest {
    pub group_id: String,
    pub user_id: i64,
}

/// 更新成员角色请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct UpdateMemberRoleRequest {
    pub group_id: String,
    pub user_id: i64,
    pub role: String,
}

/// 删除群组请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct DeleteGroupRequest {
    pub group_id: String,
}
