use super::{
    Controller, created_response, map_domain_error_to_http_response, map_application_error_to_http_response, no_content_response,
    ok_response, paginated_response,
};
use crate::application::commands::{
    CreateGroupCommand, DeleteGroupCommand, JoinGroupCommand, KickMemberCommand, LeaveGroupCommand,
    TransferOwnershipCommand, UpdateGroupInfoCommand,
    GroupCommandResult, JoinGroupResult, LeaveGroupResult, KickMemberResult, TransferOwnershipResult,
};
use crate::application::queries::{
    SearchGroupsByLocationQuery, GroupMembersQuery, PaginationQuery, SearchGroupsByNameQuery,
    SearchGroupsByIdQuery, SearchGroupsByUserIdQuery, SearchGroupsByOwnerIdQuery,
    NearbyGroup, GroupSearchResult, GroupMemberInfo,
};
use crate::application::{CommandHandler, QueryHandler, PaginatedResult};
use crate::presentation::http::auth::CurrentUser;
/// 群组HTTP控制器
/// 
/// 功能职责：
/// - 处理群组相关的 HTTP API 请求
/// - 实现群组创建、管理、成员操作等接口
/// - 管理群组权限和角色分配功能
/// - 提供基于地理位置的群组搜索和推荐
/// - 协调群组应用服务和领域逻辑
/// 
/// API 端点：
/// - POST /group/create - 创建群组
/// - POST /group/search_by_name - 按名称搜索群组
/// - POST /group/search_by_location - 按位置搜索群组
/// - POST /group/join - 加入群组
/// - POST /group/leave - 退出群组
/// - POST /group/members - 获取群组成员
/// - POST /group/transfer_ownership - 转让群主
/// - POST /group/kick - 踢出成员
/// 
/// 权限管理：
/// - 群主权限：群组设置、成员管理、权限分配
/// - 管理员权限：成员管理、内容审核
/// - 普通成员权限：基础参与和互动功能
/// - 访客权限：查看公开群组信息
use axum::{
    Json,
    extract::{Path, State},
    response::Response,
};
use serde::{Deserialize, Serialize};

/// 辅助函数：将字符串转换为i64，处理错误
fn parse_id_string(id_str: &str, field_name: &str) -> Result<i64, crate::domain::DomainError> {
    id_str.parse::<i64>().map_err(|_| {
        crate::domain::DomainError::BusinessRuleViolation {
            rule: format!("Invalid {} format", field_name)
        }
    })
}

/// 群组控制器
#[derive(Clone)]
pub struct GroupController<GroupAppService> {
    group_service: GroupAppService,
}

impl<GroupAppService> GroupController<GroupAppService>
where
    GroupAppService: CommandHandler<CreateGroupCommand, Result = GroupCommandResult>
        + CommandHandler<UpdateGroupInfoCommand, Result = GroupCommandResult>
        + CommandHandler<DeleteGroupCommand, Result = GroupCommandResult>
        // member management commands
        + CommandHandler<JoinGroupCommand, Result = JoinGroupResult>
        + CommandHandler<LeaveGroupCommand, Result = LeaveGroupResult>
        + CommandHandler<KickMemberCommand, Result = KickMemberResult>
        + CommandHandler<
            TransferOwnershipCommand,
            Result = TransferOwnershipResult,
        >
        // queries
        + QueryHandler<
            SearchGroupsByLocationQuery,
            Result = PaginatedResult<NearbyGroup>,
        > + QueryHandler<
            SearchGroupsByNameQuery,
            Result = PaginatedResult<GroupSearchResult>,
        > + QueryHandler<
            GroupMembersQuery,
            Result = PaginatedResult<GroupMemberInfo>,
        > + QueryHandler<
            SearchGroupsByIdQuery,
            Result = Option<crate::domain::GroupProfile>,
        > + QueryHandler<
            SearchGroupsByUserIdQuery,
            Result = crate::application::PaginatedResult<crate::domain::GroupProfile>,
        > + QueryHandler<
            SearchGroupsByOwnerIdQuery,
            Result = crate::application::PaginatedResult<crate::domain::GroupProfile>,
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
        Json(req): Json<CreateRequest>,
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
        match CommandHandler::<CreateGroupCommand>::handle(&controller.group_service, cmd).await {
            Ok(result) => created_response(result),
            Err(e) => map_application_error_to_http_response(e),
        }
    }

    pub async fn update_group(
        axum::extract::State(controller): axum::extract::State<Self>,
        Path(group_id): Path<i64>,
        Json(req): Json<UpdateGroupRequest>,
    ) -> Response {
        let cmd = UpdateGroupInfoCommand {
            group_id: group_id.to_string(),
            name: req.name,
            description: req.description,
        };
        match CommandHandler::<UpdateGroupInfoCommand>::handle(&controller.group_service, cmd).await {
            Ok(result) => ok_response(result),
            Err(e) => map_application_error_to_http_response(e),
        }
    }

    pub async fn delete_group(
        axum::extract::State(controller): axum::extract::State<Self>,
        Path(group_id): Path<i64>,
    ) -> Response {
        let cmd = DeleteGroupCommand { group_id: group_id.to_string() };
        match CommandHandler::<DeleteGroupCommand>::handle(&controller.group_service, cmd).await {
            Ok(_) => no_content_response(),
            Err(e) => map_application_error_to_http_response(e),
        }
    }

    pub async fn search_by_location(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(request): Json<SearchByLocationRequest>,
    ) -> Response {
        let pagination = PaginationQuery {
            cursor: request.cursor.and_then(|c| c.parse().ok()),
            limit: request.limit.map(|l| l as usize),
        };
        let q = SearchGroupsByLocationQuery {
            user_id: current_user.user_id,
            latitude: request.latitude,
            longitude: request.longitude,
            radius_km: Some(request.radius / 1000.0), // 转换米为公里
            pagination,
        };
        match QueryHandler::<SearchGroupsByLocationQuery>::handle(&controller.group_service, q).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(e) => map_application_error_to_http_response(e),
        }
    }

    pub async fn search_by_name(
        axum::extract::State(controller): axum::extract::State<Self>,
        Json(request): Json<SearchByNameRequest>,
    ) -> Response {
        let pagination = PaginationQuery {
            cursor: request.cursor.and_then(|c| c.parse().ok()),
            limit: request.limit.map(|l| l as usize),
        };
        let q = SearchGroupsByNameQuery {
            keyword: request.query,
            pagination,
        };
        match QueryHandler::<SearchGroupsByNameQuery>::handle(&controller.group_service, q).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(e) => map_application_error_to_http_response(e),
        }
    }

    /// 按ID搜索群组
    pub async fn search_by_id(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(request): Json<SearchByIdRequest>,
    ) -> Response {
        let group_id = match parse_id_string(&request.group_id, "group_id") {
            Ok(id) => id,
            Err(error) => return map_application_error_to_http_response(crate::application::ApplicationError::Domain(error)),
        };
        
        let query = SearchGroupsByIdQuery {
            group_id,
            requester_id: current_user.user_id,
        };
        
        match QueryHandler::<SearchGroupsByIdQuery>::handle(&controller.group_service, query).await {
            Ok(Some(group)) => ok_response(group),
            Ok(None) => map_domain_error_to_http_response(
                crate::domain::DomainError::GroupNotFound
            ),
            Err(error) => map_application_error_to_http_response(error),
        }
    }

    // --- Member management stubs (POST-only) ---

    pub async fn join_group(
        axum::extract::State(controller): axum::extract::State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(req): Json<JoinRequest>,
    ) -> Response {
        let cmd = JoinGroupCommand {
            group_id: req.group_id.to_string(),
            user_id: current_user.user_id,
            password: req.password,
        };
        match CommandHandler::<JoinGroupCommand>::handle(&controller.group_service, cmd).await {
            Ok(result) => ok_response(result),
            Err(e) => map_application_error_to_http_response(e),
        }
    }

    pub async fn leave_group(
        axum::extract::State(controller): axum::extract::State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(req): Json<LeaveRequest>,
    ) -> Response {
        let cmd = LeaveGroupCommand {
            group_id: req.group_id.to_string(),
            user_id: current_user.user_id,
        };
        match CommandHandler::<LeaveGroupCommand>::handle(&controller.group_service, cmd).await {
            Ok(result) => ok_response(result),
            Err(e) => map_application_error_to_http_response(e),
        }
    }

    pub async fn group_members(
        axum::extract::State(controller): axum::extract::State<Self>,
        Json(req): Json<MembersRequest>,
    ) -> Response {
        let pagination = PaginationQuery {
            cursor: req.cursor.and_then(|c| c.parse().ok()),
            limit: req.limit.map(|v| v as usize),
        };
        let group_id = match parse_id_string(&req.group_id, "group_id") {
            Ok(id) => id,
            Err(error) => return map_application_error_to_http_response(crate::application::ApplicationError::Domain(error)),
        };
        
        let q = GroupMembersQuery {
            group_id,
            pagination,
        };
        match QueryHandler::<GroupMembersQuery>::handle(&controller.group_service, q).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(e) => map_application_error_to_http_response(e),
        }
    }

    pub async fn transfer_ownership(
        axum::extract::State(controller): axum::extract::State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(req): Json<TransferOwnershipRequest>,
    ) -> Response {
        let new_owner_id = match parse_id_string(&req.new_owner_id, "new_owner_id") {
            Ok(id) => id,
            Err(error) => return map_application_error_to_http_response(crate::application::ApplicationError::Domain(error)),
        };
        
        let cmd = TransferOwnershipCommand {
            group_id: req.group_id.to_string(),
            new_owner_id,
            current_owner_id: current_user.user_id,
        };
        match CommandHandler::<TransferOwnershipCommand>::handle(&controller.group_service, cmd).await {
            Ok(result) => ok_response(result),
            Err(e) => map_application_error_to_http_response(e),
        }
    }

    pub async fn kick_member(
        axum::extract::State(controller): axum::extract::State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(req): Json<KickRequest>,
    ) -> Response {
        let target_user_id = match parse_id_string(&req.target_user_id, "target_user_id") {
            Ok(id) => id,
            Err(error) => return map_application_error_to_http_response(crate::application::ApplicationError::Domain(error)),
        };
        
        let cmd = KickMemberCommand {
            group_id: req.group_id.to_string(),
            target_user_id,
            admin_id: current_user.user_id,
        };
        match CommandHandler::<KickMemberCommand>::handle(&controller.group_service, cmd).await {
            Ok(result) => ok_response(result),
            Err(e) => map_application_error_to_http_response(e),
        }
    }

    /// 获取当前用户加入的群组列表
    pub async fn user_groups(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(request): Json<UserGroupsRequest>,
    ) -> Response {
        let query = SearchGroupsByUserIdQuery {
            user_id: current_user.user_id,
            pagination: crate::application::queries::PaginationQuery {
                cursor: request.cursor.and_then(|c| c.parse().ok()),
                limit: request.limit.map(|v| v as usize),
            },
        };
        
        match QueryHandler::<SearchGroupsByUserIdQuery>::handle(&controller.group_service, query).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(error) => map_application_error_to_http_response(error),
        }
    }

    /// 获取当前用户拥有的群组列表
    pub async fn owned_groups(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(request): Json<OwnedGroupsRequest>,
    ) -> Response {
        let query = SearchGroupsByOwnerIdQuery {
            owner_id: current_user.user_id,
            pagination: crate::application::queries::PaginationQuery {
                cursor: request.cursor.and_then(|c| c.parse().ok()),
                limit: request.limit.map(|v| v as usize),
            },
        };
        
        match QueryHandler::<SearchGroupsByOwnerIdQuery>::handle(&controller.group_service, query).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(error) => map_application_error_to_http_response(error),
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
pub struct CreateRequest {
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
    pub cursor: Option<String>,
    pub limit: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchByNameRequest {
    pub query: String,
    pub cursor: Option<String>,
    pub limit: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchByIdRequest {
    pub group_id: String, // 前端发送字符串格式的64位整数
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JoinRequest {
    pub group_id: String,
    pub password: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LeaveRequest {
    pub group_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MembersRequest {
    pub group_id: String,
    pub cursor: Option<String>,
    pub limit: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransferOwnershipRequest {
    pub group_id: String,
    pub new_owner_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KickRequest {
    pub group_id: String,
    pub target_user_id: String, // 前端发送字符串格式的64位整数
}

// ===== 用户群组列表请求结构 =====

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserGroupsRequest {
    pub cursor: Option<String>,
    pub limit: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OwnedGroupsRequest {
    pub cursor: Option<String>,
    pub limit: Option<i32>,
}
