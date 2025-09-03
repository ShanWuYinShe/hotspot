use super::{
    Controller, created_response, map_domain_error_to_http_response, no_content_response,
    ok_response, paginated_response,
};
use crate::application::commands::{
    CreateGroupCommand, DeleteGroupCommand, JoinGroupCommand, KickMemberCommand, LeaveGroupCommand,
    TransferOwnershipCommand, UpdateGroupInfoCommand,
};
use crate::application::queries::{
    FindNearbyGroupsQuery, GroupMembersQuery, PaginationQuery, SearchGroupsByNameQuery,
    GetGroupByIdQuery, GetUserGroupsQuery, GetOwnedGroupsQuery,
};
use crate::application::{CommandHandler, QueryHandler};
use crate::presentation::http::auth::CurrentUser;
/// 群组HTTP控制器
///
/// 处理群组相关的HTTP请求，实现hotspot.md中定义的9个群组API端点
use axum::{
    Json,
    extract::{Path, State},
    response::Response,
};
use serde::{Deserialize, Serialize};

/// 群组控制器
#[derive(Clone)]
pub struct GroupController<GroupAppService> {
    group_service: GroupAppService,
}

impl<GroupAppService> GroupController<GroupAppService>
where
    GroupAppService: CommandHandler<CreateGroupCommand, Result = crate::application::GroupCommandResult>
        + CommandHandler<UpdateGroupInfoCommand, Result = crate::application::GroupCommandResult>
        + CommandHandler<DeleteGroupCommand, Result = crate::application::GroupCommandResult>
        // member management commands
        + CommandHandler<JoinGroupCommand, Result = crate::application::JoinGroupResult>
        + CommandHandler<LeaveGroupCommand, Result = crate::application::LeaveGroupResult>
        + CommandHandler<KickMemberCommand, Result = crate::application::KickMemberResult>
        + CommandHandler<
            TransferOwnershipCommand,
            Result = crate::application::TransferOwnershipResult,
        >
        // queries
        + QueryHandler<
            FindNearbyGroupsQuery,
            Result = crate::application::PaginatedResult<crate::application::NearbyGroup>,
        > + QueryHandler<
            SearchGroupsByNameQuery,
            Result = crate::application::PaginatedResult<crate::application::GroupSearchResult>,
        > + QueryHandler<
            GroupMembersQuery,
            Result = crate::application::PaginatedResult<crate::application::GroupMemberInfo>,
        > + QueryHandler<
            GetGroupByIdQuery,
            Result = Option<crate::domain::Group>,
        > + QueryHandler<
            GetUserGroupsQuery,
            Result = crate::application::PaginatedResult<crate::domain::Group>,
        > + QueryHandler<
            GetOwnedGroupsQuery,
            Result = crate::application::PaginatedResult<crate::domain::Group>,
        > + Clone
        + Send
        + Sync
        + 'static,
{
    pub fn new(group_service: GroupAppService) -> Self {
        Self { group_service }
    }

    pub async fn create_group(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(req): Json<CreateGroupRequest>,
    ) -> Response {
        let cmd = CreateGroupCommand {
            name: req.name,
            description: req.description,
            owner_id: current_user.user_id,
            latitude: req.latitude,
            longitude: req.longitude,
            location_name: req.location_name.unwrap_or_else(|| "未知位置".to_string()),
            password: if req.is_public { None } else { req.password },
        };
        match controller.group_service.handle_command(cmd).await {
            Ok(result) => created_response(result),
            Err(e) => map_domain_error_to_http_response(e),
        }
    }

    pub async fn update_group(
        axum::extract::State(controller): axum::extract::State<Self>,
        Path(group_id): Path<i64>,
        Json(req): Json<UpdateGroupRequest>,
    ) -> Response {
        let cmd = UpdateGroupInfoCommand {
            group_id,
            name: req.name,
            description: req.description,
        };
        match controller.group_service.handle_command(cmd).await {
            Ok(result) => ok_response(result),
            Err(e) => map_domain_error_to_http_response(e),
        }
    }

    pub async fn delete_group(
        axum::extract::State(controller): axum::extract::State<Self>,
        Path(group_id): Path<i64>,
    ) -> Response {
        let cmd = DeleteGroupCommand { group_id };
        match controller.group_service.handle_command(cmd).await {
            Ok(_) => no_content_response(),
            Err(e) => map_domain_error_to_http_response(e),
        }
    }

    pub async fn find_nearby_groups(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(request): Json<SearchByLocationRequest>,
    ) -> Response {
        let pagination = PaginationQuery {
            cursor: request.cursor,
            limit: request.limit.map(|v| v as i32),
        };
        let q = FindNearbyGroupsQuery {
            user_id: current_user.user_id,
            latitude: request.latitude,
            longitude: request.longitude,
            radius_km: Some(request.radius / 1000.0), // 转换米为公里
            pagination,
        };
        match controller.group_service.handle_query(q).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(e) => map_domain_error_to_http_response(e),
        }
    }

    pub async fn search_groups(
        axum::extract::State(controller): axum::extract::State<Self>,
        Json(request): Json<SearchByNameRequest>,
    ) -> Response {
        let pagination = PaginationQuery {
            cursor: request.cursor,
            limit: request.limit.map(|v| v as i32),
        };
        let q = SearchGroupsByNameQuery {
            keyword: request.query,
            pagination,
        };
        match controller.group_service.handle_query(q).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(e) => map_domain_error_to_http_response(e),
        }
    }

    /// 按ID搜索群组
    pub async fn search_by_id(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(request): Json<SearchByIdRequest>,
    ) -> Response {
        let query = GetGroupByIdQuery {
            group_id: request.group_id,
            requester_id: current_user.user_id,
        };
        
        match controller.group_service.handle_query(query).await {
            Ok(Some(group)) => ok_response(group),
            Ok(None) => map_domain_error_to_http_response(
                crate::domain::DomainError::Group(crate::domain::GroupDomainError::GroupNotFound { 
                    group_id: request.group_id 
                })
            ),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }

    // --- Member management stubs (POST-only) ---

    pub async fn join_group(
        axum::extract::State(controller): axum::extract::State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(req): Json<JoinGroupRequest>,
    ) -> Response {
        let cmd = JoinGroupCommand {
            group_id: req.group_id,
            user_id: current_user.user_id,
            password: req.password,
        };
        match controller.group_service.handle_command(cmd).await {
            Ok(result) => ok_response(result),
            Err(e) => map_domain_error_to_http_response(e),
        }
    }

    pub async fn leave_group(
        axum::extract::State(controller): axum::extract::State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(req): Json<LeaveGroupRequest>,
    ) -> Response {
        let cmd = LeaveGroupCommand {
            group_id: req.group_id,
            user_id: current_user.user_id,
        };
        match controller.group_service.handle_command(cmd).await {
            Ok(result) => ok_response(result),
            Err(e) => map_domain_error_to_http_response(e),
        }
    }

    pub async fn group_members(
        axum::extract::State(controller): axum::extract::State<Self>,
        Json(req): Json<GroupMembersRequest>,
    ) -> Response {
        let pagination = PaginationQuery {
            cursor: req.cursor,
            limit: req.limit.map(|v| v as i32),
        };
        let q = GroupMembersQuery {
            group_id: req.group_id,
            pagination,
        };
        match controller.group_service.handle_query(q).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(e) => map_domain_error_to_http_response(e),
        }
    }

    pub async fn transfer_ownership(
        axum::extract::State(controller): axum::extract::State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(req): Json<TransferOwnershipRequest>,
    ) -> Response {
        let cmd = TransferOwnershipCommand {
            group_id: req.group_id,
            new_owner_id: req.new_owner_id,
            current_owner_id: current_user.user_id,
        };
        match controller.group_service.handle_command(cmd).await {
            Ok(result) => ok_response(result),
            Err(e) => map_domain_error_to_http_response(e),
        }
    }

    pub async fn kick_member(
        axum::extract::State(controller): axum::extract::State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(req): Json<KickMemberRequest>,
    ) -> Response {
        let cmd = KickMemberCommand {
            group_id: req.group_id,
            target_user_id: req.target_user_id,
            admin_id: current_user.user_id,
        };
        match controller.group_service.handle_command(cmd).await {
            Ok(result) => ok_response(result),
            Err(e) => map_domain_error_to_http_response(e),
        }
    }

    /// 获取当前用户加入的群组列表
    pub async fn user_groups(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(request): Json<UserGroupsRequest>,
    ) -> Response {
        let query = GetUserGroupsQuery {
            user_id: current_user.user_id,
            pagination: crate::application::queries::PaginationQuery {
                cursor: request.cursor,
                limit: request.limit.map(|v| v as i32),
            },
        };
        
        match controller.group_service.handle_query(query).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }

    /// 获取当前用户拥有的群组列表
    pub async fn owned_groups(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(request): Json<OwnedGroupsRequest>,
    ) -> Response {
        let query = GetOwnedGroupsQuery {
            owner_id: current_user.user_id,
            pagination: crate::application::queries::PaginationQuery {
                cursor: request.cursor,
                limit: request.limit.map(|v| v as i32),
            },
        };
        
        match controller.group_service.handle_query(query).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
}

#[async_trait::async_trait]
impl<GroupAppService> Controller for GroupController<GroupAppService>
where
    GroupAppService: Send + Sync,
{
    fn name(&self) -> &'static str {
        "GroupController"
    }
}
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateGroupRequest {
    pub name: String,
    pub description: Option<String>,
    pub latitude: f64,
    pub longitude: f64,
    pub radius: f64, // 以米为单位
    pub is_public: bool,
    pub location_name: Option<String>,
    pub password: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateGroupRequest {
    pub name: Option<String>,
    pub description: Option<String>,
}

// ===== 群组搜索请求结构 =====

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchByLocationRequest {
    pub latitude: f64,
    pub longitude: f64,
    pub radius: f64, // 以米为单位
    pub cursor: Option<i64>,
    pub limit: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchByNameRequest {
    pub query: String,
    pub cursor: Option<i64>,
    pub limit: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchByIdRequest {
    pub group_id: i64, // 注意：hotspot.md要求使用UUID，但当前实现使用i64
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JoinGroupRequest {
    pub group_id: i64,
    pub password: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LeaveGroupRequest {
    pub group_id: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupMembersRequest {
    pub group_id: i64,
    pub cursor: Option<i64>,
    pub limit: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransferOwnershipRequest {
    pub group_id: i64,
    pub new_owner_id: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KickMemberRequest {
    pub group_id: i64,
    pub target_user_id: i64, // 修改字段名以匹配hotspot.md
}

// ===== 用户群组列表请求结构 =====

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserGroupsRequest {
    pub cursor: Option<i64>,
    pub limit: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OwnedGroupsRequest {
    pub cursor: Option<i64>,
    pub limit: Option<i64>,
}
