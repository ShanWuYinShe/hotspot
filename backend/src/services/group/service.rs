use std::sync::Arc;
use serde::Serialize;
use crate::utils::response::PaginatedResponse;
use uuid::Uuid;
use chrono::{DateTime, Utc};
use utoipa::ToSchema;

use crate::{
    config::db::Database,
    repositories::{GroupRepository, RepositoryError},
    utils::{response::AppError, result_ext::ResultExt},
    models::api::group::{GroupMemberInfo, GroupMembersResponse},
};

// =============== 返回结构体 ===============

/// 群组搜索结果，用于替换复杂的16字段元组
#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct GroupSearchResult {
    pub id: i64,
    pub group_id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub owner_id: i64,
    pub creator_name: Option<String>,
    pub created_at: DateTime<Utc>,
    pub last_active_at: DateTime<Utc>,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub member_count: Option<i64>,
    pub distance: Option<f64>,
    pub is_password_required: Option<bool>,
    pub is_member: Option<bool>,
    pub user_role: Option<String>,
}

/// 群组行数据，用于 API 响应
#[derive(Debug, Clone, Serialize)]
pub struct GroupRow {
    pub id: i64,
    pub group_id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub owner_id: i64,
    pub creator_name: String,
    pub created_at: DateTime<Utc>,
    pub last_active_at: DateTime<Utc>,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub member_count: i64,
    pub distance: f64,
    pub is_password_required: bool,
    pub is_member: bool,
    pub user_role: String,
}

impl From<GroupSearchResult> for GroupRow {
    fn from(result: GroupSearchResult) -> Self {
        GroupRow {
            id: result.id,
            group_id: result.group_id,
            name: result.name,
            description: result.description,
            owner_id: result.owner_id,
            creator_name: result.creator_name.unwrap_or_default(),
            created_at: result.created_at,
            last_active_at: result.last_active_at,
            latitude: result.latitude,
            longitude: result.longitude,
            location_name: result.location_name,
            member_count: result.member_count.unwrap_or(0),
            distance: result.distance.unwrap_or(0.0),
            is_password_required: result.is_password_required.unwrap_or(false),
            is_member: result.is_member.unwrap_or(false),
            user_role: result.user_role.unwrap_or_default(),
        }
    }
}

#[derive(Debug, Serialize)]
pub struct GroupActionResult {
    pub success: bool,
    pub role: Option<String>,
}

#[derive(Debug, Clone)]
pub struct GroupService {
    group_repository: Arc<crate::repositories::PgGroupRepository>,
}

impl GroupService {
    pub fn new(db: Database) -> Self {
        let group_repository = Arc::new(crate::repositories::PgGroupRepository::new(db.pg_pool.clone()));
        Self { group_repository }
    }

    // =============== 验证函数 ===============

    // 检查群组是否存在
    pub async fn check_group_exists(&self, group_id: Uuid) -> Result<bool, AppError> {
        self.group_repository
            .check_group_exists(group_id)
            .await
            .map_repository_error("查询群组")
    }

    // 检查用户是否是群主
    pub async fn check_user_is_group_owner(
        &self,
        group_id: Uuid,
        user_id: i64,
    ) -> Result<bool, AppError> {
        self.group_repository
            .check_user_is_group_owner(group_id, user_id)
            .await
            .map_repository_error("检查群组所有权")
    }

    // 检查用户是否在群组中
    pub async fn check_user_in_group(
        &self,
        group_id: Uuid,
        user_id: i64,
    ) -> Result<bool, AppError> {
        self.group_repository
            .check_user_in_group(group_id, user_id)
            .await
            .map_repository_error("检查群组成员")
    }

    // 检查用户是否是群组管理员
    pub async fn check_user_is_group_admin(
        &self,
        group_id: Uuid,
        user_id: i64,
    ) -> Result<bool, AppError> {
        self.group_repository
            .check_user_is_group_admin(group_id, user_id)
            .await
            .map_repository_error("检查群组管理员")
    }

    // 检查群组是否需要密码
    pub async fn check_group_need_password(&self, group_id: Uuid) -> Result<bool, AppError> {
        self.group_repository
            .check_group_need_password(group_id)
            .await
            .map_repository_error("检查群组密码需求")
    }

    // 验证群组密码
    pub async fn verify_group_password(
        &self,
        group_id: Uuid,
        password: &str,
    ) -> Result<bool, AppError> {
        self.group_repository
            .verify_group_password(group_id, password)
            .await
            .map_repository_error("验证群组密码")
    }

    // =============== 操作函数 ===============

    // 创建群组（事务操作）
    pub async fn create_group(
        &self,
        name: &str,
        description: Option<&str>,
        owner_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: &str,
        password: Option<&str>,
    ) -> Result<Uuid, AppError> {
        self.group_repository.create_group(name, description, owner_id, latitude, longitude, location_name, password)
            .await
            .map_err(|e| {
                tracing::error!("创建群组失败: {:?}", e);
                AppError::InternalServerError("创建群组失败".to_string())
            })
    }

    // =============== 业务函数 ===============

    // 加入群组
    pub async fn join_group(
        &self,
        group_id: Uuid,
        user_id: i64,
    ) -> Result<GroupActionResult, AppError> {
        let (success, role) = self.group_repository.join_group(group_id, user_id)
            .await
            .map_err(|e| {
                tracing::error!("加入群组失败: {:?}", e);
                AppError::InternalServerError("加入群组失败".to_string())
            })?;
            
        Ok(GroupActionResult {
            success,
            role: Some(role),
        })
    }

    pub async fn leave_group(
        &self,
        group_id: Uuid,
        user_id: i64,
    ) -> Result<GroupActionResult, AppError> {
        let success = self.group_repository.leave_group(group_id, user_id)
            .await
            .map_err(|e| {
                tracing::error!("离开群组失败: {:?}", e);
                AppError::InternalServerError("离开群组失败".to_string())
            })?;

        Ok(GroupActionResult {
            success,
            role: None,
        })
    }

    pub async fn kick_member(
        &self,
        group_id: Uuid,
        user_id: i64,
        admin_id: i64,
    ) -> Result<GroupActionResult, AppError> {
        let success = self.group_repository.kick_member(group_id, user_id, admin_id)
            .await
            .map_err(|e| {
                match e {
                    RepositoryError::InvalidArgument(msg) => AppError::BadRequest(msg),
                    _ => {
                        tracing::error!("踢出成员失败: {:?}", e);
                        AppError::InternalServerError("踢出成员失败".to_string())
                    }
                }
            })?;

        Ok(GroupActionResult {
            success,
            role: None,
        })
    }

    pub async fn transfer_ownership(
        &self,
        group_id: Uuid,
        new_owner_id: i64,
        user_id: i64,
    ) -> Result<GroupActionResult, AppError> {
        let success = self.group_repository.transfer_ownership(group_id, new_owner_id, user_id)
            .await
            .map_err(|e| {
                tracing::error!("转让群主失败: {:?}", e);
                AppError::InternalServerError("转让群主失败".to_string())
            })?;

        Ok(GroupActionResult {
            success,
            role: None,
        })
    }

    // 获取群组成员列表
    pub async fn get_group_members(
        &self,
        group_id: Uuid,
        cursor: i64,
        limit: i64,
        user_id: i64,
    ) -> Result<GroupMembersResponse, AppError> {
        // 先检查用户是否属于该群组
        let is_member = self.check_user_in_group(group_id, user_id).await?;
        if !is_member {
            return Err(AppError::Forbidden("您不是该群组的成员".to_string()));
        }

        // 获取成员列表
        let (rows, has_more, next_cursor) = self.group_repository.get_group_members(group_id, cursor, limit)
            .await
            .map_err(|e| {
                tracing::error!("获取群组成员失败: {:?}", e);
                AppError::InternalServerError("获取群组成员失败".to_string())
            })?;

        let members = rows.into_iter()
            .map(|(id, user_id, nickname, last_active_at, role, join_time)| {
                GroupMemberInfo {
                    id,
                    user_id,
                    nickname,
                    last_active_at,
                    role,
                    join_time,
                }
            })
            .collect();

        Ok(GroupMembersResponse {
            items: members,
            pagination: crate::utils::response::PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    // 根据位置搜索群组
    pub async fn search_group_by_location(
        &self,
        user_id: i64,
        radius: f64,
        cursor: i64,
        limit: i64,
    ) -> Result<(Vec<GroupRow>, bool, Option<i64>), AppError> {
        let (search_results, has_more, next_cursor) = self.group_repository.search_group_by_location(user_id, radius, cursor, limit)
            .await
            .map_err(|e| {
                tracing::error!("查询附近群组失败: {:?}", e);
                AppError::InternalServerError("查询附近群组失败".to_string())
            })?;

        let groups = search_results.into_iter()
            .map(|result| result.into())
            .collect();

        Ok((groups, has_more, next_cursor))
    }

    // 根据名称搜索群组
    pub async fn search_group_by_name(
        &self,
        user_id: i64,
        keyword: String,
        cursor: i64,
        limit: i64,
    ) -> Result<(Vec<GroupRow>, bool, Option<i64>), AppError> {
        let (search_results, has_more, next_cursor) = self.group_repository.search_group_by_name(user_id, &keyword, cursor, limit)
            .await
            .map_err(|e| {
                tracing::error!("搜索群组失败: {:?}", e);
                AppError::InternalServerError("搜索群组失败".to_string())
            })?;

        let groups = search_results.into_iter()
            .map(|result| result.into())
            .collect();

        Ok((groups, has_more, next_cursor))
    }

    // 根据ID获取群组
    pub async fn search_group_by_id(
        &self,
        user_id: i64,
        group_id: Uuid,
    ) -> Result<GroupRow, AppError> {
        let search_result = self.group_repository.search_group_by_id(user_id, group_id)
            .await
            .map_err(|e| {
                tracing::error!("查询群组失败: {:?}", e);
                match e {
                    RepositoryError::NotFound => AppError::NotFound("群组不存在".to_string()),
                    _ => AppError::InternalServerError("查询群组失败".to_string())
                }
            })?;

        Ok(search_result.into())
    }

    /// 获取用户群组列表
    pub async fn get_user_groups(
        &self,
        user_id: i64,
        limit: Option<i64>,
        offset: Option<i64>
    ) -> Result<PaginatedResponse<crate::models::api::group::GroupInfo>, AppError> {
        // 获取群组列表
        let groups = self.group_repository
            .get_user_groups(user_id, limit, offset)
            .await
            .map_repository_error("获取用户群组")?;

        // 获取总数 (虽然暂时未使用，但保留以便将来扩展)
        let _total = self.group_repository
            .count_user_groups(user_id)
            .await
            .map_repository_error("统计用户群组数量")?;

        // 转换数据格式
        let group_rows: Vec<GroupRow> = groups.into_iter().map(|g| g.into()).collect();
        let items: Vec<crate::models::api::group::GroupInfo> = group_rows.into_iter().map(|g| g.into()).collect();
        
        // 计算分页信息
        let limit_val = limit.unwrap_or(20);
        let has_more = items.len() as i64 == limit_val;
        let next_cursor = if has_more {
            Some(offset.unwrap_or(0) + limit_val)
        } else {
            None
        };

        Ok(PaginatedResponse {
            items,
            pagination: crate::utils::response::PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 获取用户创建的群组列表
    pub async fn get_owned_groups(
        &self,
        user_id: i64,
        limit: Option<i64>,
        offset: Option<i64>
    ) -> Result<PaginatedResponse<crate::models::api::group::GroupInfo>, AppError> {
        // 获取群组列表
        let groups = self.group_repository
            .get_owned_groups(user_id, limit, offset)
            .await
            .map_repository_error("获取用户创建的群组")?;

        // 转换数据格式
        let group_rows: Vec<GroupRow> = groups.into_iter().map(|g| g.into()).collect();
        let items: Vec<crate::models::api::group::GroupInfo> = group_rows.into_iter().map(|g| g.into()).collect();
        
        // 计算分页信息
        let limit_val = limit.unwrap_or(20);
        let has_more = items.len() as i64 == limit_val;
        let next_cursor = if has_more {
            Some(offset.unwrap_or(0) + limit_val)
        } else {
            None
        };

        Ok(PaginatedResponse {
            items,
            pagination: crate::utils::response::PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }
}

impl From<GroupRow> for crate::models::api::group::GroupInfo {
    fn from(row: GroupRow) -> Self {
        crate::models::api::group::GroupInfo {
            group_id: row.group_id,
            name: row.name,
            description: row.description,
            owner_id: row.owner_id,
            creator_name: row.creator_name,
            created_at: row.created_at,
            last_active_at: row.last_active_at,
            latitude: row.latitude,
            longitude: row.longitude,
            member_count: row.member_count,
            distance: row.distance,
            location_name: row.location_name,
            is_password_required: row.is_password_required,
            is_member: row.is_member,
            user_role: row.user_role,
        }
    }
}
