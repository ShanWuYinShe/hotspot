use super::PaginationQuery;
/// 群组相关查询定义
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FindNearbyGroupsQuery {
    pub user_id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub radius_km: Option<f64>,
    pub pagination: PaginationQuery,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchGroupsByNameQuery {
    pub keyword: String,
    pub pagination: PaginationQuery,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NearbyGroup {
    pub group_id: i64,
    pub name: String,
    pub latitude: f64,
    pub longitude: f64,
    pub distance_km: f64,
    pub created_at: chrono::DateTime<chrono::Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupSearchResult {
    pub group_id: i64,
    pub name: String,
    pub has_location: bool,
    pub created_at: chrono::DateTime<chrono::Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupMembersQuery {
    pub group_id: i64,
    pub pagination: PaginationQuery,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupMemberInfo {
    pub member_id: i64,
    pub user_id: i64,
    pub nickname: String,
    pub role: String,
    pub last_active_at: chrono::DateTime<chrono::Utc>,
    pub joined_at: chrono::DateTime<chrono::Utc>,
}

/// 按ID获取群组查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetGroupByIdQuery {
    pub group_id: i64,
    pub requester_id: i64,
}

/// 获取用户加入的群组查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetUserGroupsQuery {
    pub user_id: i64,
    pub pagination: PaginationQuery,
}

/// 获取用户拥有的群组查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetOwnedGroupsQuery {
    pub owner_id: i64,
    pub pagination: PaginationQuery,
}
