use std::sync::Arc;

use axum::{Extension, Json, debug_handler, extract::State};

use crate::{
    middleware::auth::AppState,
    models::api::{group::*, user::AuthUser},
    utils::response::{ApiResponse, AppError, success_response},
    utils::ResponseMapper,
    models::constants::group_roles,
    validators::{group_validator, common_validator},
    ws::message_types::{GroupEventData, GroupEventType as WsGroupEventType},
};
use chrono::Utc;
use crate::utils::events::DomainEvent;



use crate::models::api::{
    group::{CreateGroupRequest, JoinGroupRequest, KickMemberRequest, TransferOwnershipRequest, GetUserGroupsRequest, GetOwnedGroupsRequest},
};

// 创建群组
#[utoipa::path(
    post,
    path = "/api/v1/group/create",
    tag = "群组",
    request_body = CreateGroupRequest,
    responses(
        (status = 200, description = "群组创建成功", body = ApiResponse<CreateGroupResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 409, description = "群组名称已存在", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn create_group(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<CreateGroupRequest>,
) -> Result<Json<ApiResponse<CreateGroupResponse>>, AppError> {
    // 验证请求参数
    group_validator::validate_group_creation_request(payload.description.as_ref())?;
    
    // 验证名称长度
    group_validator::validate_group_name(&payload.name)?;
    
    // 验证位置名称
    common_validator::validate_location_name(&payload.location_name)?;
    
    // 验证密码长度
    group_validator::validate_group_password(payload.password.as_deref())?;
    
    let group_id = state
        .group_service
        .create_group(
            payload.name.as_str(),
            payload.description.as_deref(),
            current_user.user_id,
            payload.latitude,
            payload.longitude,
            payload.location_name.as_str(),
            payload.password.as_deref(),
        )
        .await?;

    // 发布领域事件（WS 推送由事件处理器承担）
    state.event_bus.publish(DomainEvent::GroupJoined { group_id: group_id.into(), user_id: current_user.user_id.into() }).await;

    Ok(success_response(CreateGroupResponse { group_id }))
}

// 获取附近群组
#[utoipa::path(
    post,
    path = "/api/v1/group/search/location",
    tag = "群组",
    request_body = SearchGroupByLocationRequest,
    responses(
        (status = 200, description = "根据位置搜索群组成功", body = ApiResponse<SearchGroupByLocationResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn search_group_by_location(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchGroupByLocationRequest>,
) -> Result<Json<ApiResponse<SearchGroupByLocationResponse>>, AppError> {
    // 验证请求参数
    group_validator::validate_search_radius(payload.radius)?;
    common_validator::validate_location_params(payload.latitude, payload.longitude)?;
    
    // 验证并提取分页参数
    let (cursor, limit) = crate::utils::validation_pipeline::validators::extract_pagination(
        payload.cursor, 
        payload.limit, 
        10, 
        50
    )?;
    
    let (groups, has_more, next_cursor) = state
        .group_service
        .search_group_by_location(current_user.user_id, payload.radius, cursor, limit)
        .await?;

    let items = groups
        .into_iter()
        .map(|group| {
            // 验证角色
            if !group.user_role.is_empty() && !group_roles::is_valid_role(&group.user_role) {
                tracing::warn!("搜索群组返回了无效的用户角色: {}", group.user_role);
            }
            
            GroupInfo {
                group_id: group.group_id,
                name: group.name,
                description: group.description,
                owner_id: group.owner_id,
                creator_name: group.creator_name,
                created_at: group.created_at,
                last_active_at: group.last_active_at,
                latitude: group.latitude,
                longitude: group.longitude,
                location_name: group.location_name,
                member_count: group.member_count,
                distance: group.distance,
                is_password_required: group.is_password_required,
                is_member: group.is_member,
                user_role: group.user_role,
            }
        })
        .collect();

    let response = ResponseMapper::create_paginated_response(items, has_more, next_cursor);

    Ok(success_response(response))
}

// 按照名称搜索群组
#[utoipa::path(
    post,
    path = "/api/v1/group/search/name",
    tag = "群组",
    request_body = SearchGroupByNameRequest,
    responses(
        (status = 200, description = "根据名称搜索群组成功", body = ApiResponse<SearchGroupByNameResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn search_group_by_name(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchGroupByNameRequest>,
) -> Result<Json<ApiResponse<SearchGroupByNameResponse>>, AppError> {
    // 验证关键字和分页参数
    crate::utils::validation_pipeline::validators::validate_search_keyword(&payload.keyword, 50)?;
    let (cursor, limit) = crate::utils::validation_pipeline::validators::extract_pagination(
        payload.cursor, 
        payload.limit, 
        10, 
        50
    )?;
    
    let (groups, has_more, next_cursor) = state
        .group_service
        .search_group_by_name(current_user.user_id, payload.keyword.clone(), cursor, limit)
        .await?;

    let items = groups
        .into_iter()
        .map(|group| {
            // 验证角色
            if !group.user_role.is_empty() && !group_roles::is_valid_role(&group.user_role) {
                tracing::warn!("搜索群组返回了无效的用户角色: {}", group.user_role);
            }
            
            GroupInfo {
                group_id: group.group_id,
                name: group.name,
                description: group.description,
                owner_id: group.owner_id,
                creator_name: group.creator_name,
                created_at: group.created_at,
                last_active_at: group.last_active_at,
                latitude: group.latitude,
                longitude: group.longitude,
                location_name: group.location_name,
                member_count: group.member_count,
                distance: group.distance,
                is_password_required: group.is_password_required,
                is_member: group.is_member,
                user_role: group.user_role,
            }
        })
        .collect();

    let response = ResponseMapper::create_paginated_response(items, has_more, next_cursor);

    Ok(success_response(response))
}

// 按照group id搜索群组
#[utoipa::path(
    post,
    path = "/api/v1/group/search/id",
    tag = "群组",
    request_body = SearchGroupByIdRequest,
    responses(
        (status = 200, description = "根据群组ID搜索群组成功", body = ApiResponse<SearchGroupByIdResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 404, description = "群组未找到", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn search_group_by_id(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchGroupByIdRequest>,
) -> Result<Json<ApiResponse<SearchGroupByIdResponse>>, AppError> {
    let result = state
        .group_service
        .search_group_by_id(current_user.user_id, payload.group_id)
        .await?;
        
    // 验证用户角色是否合法
    if !result.user_role.is_empty() && !group_roles::is_valid_role(&result.user_role) {
        tracing::warn!("搜索群组返回了无效的用户角色: {}", result.user_role);
    }
    
    let info = GroupInfo {
        group_id: result.group_id,
        name: result.name,
        description: result.description,
        owner_id: result.owner_id,
        creator_name: result.creator_name,
        created_at: result.created_at,
        last_active_at: result.last_active_at,
        latitude: result.latitude,
        longitude: result.longitude,
        member_count: result.member_count,
        distance: result.distance,
        location_name: result.location_name,
        is_password_required: result.is_password_required,
        is_member: result.is_member,
        user_role: result.user_role,
    };
    Ok(success_response(info))
}

// 加入群组
#[utoipa::path(
    post,
    path = "/api/v1/group/join",
    tag = "群组",
    request_body = JoinGroupRequest,
    responses(
        (status = 200, description = "加入群组成功", body = ApiResponse<JoinGroupResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 404, description = "群组未找到", body = ApiResponse<()>),
        (status = 409, description = "已在群组中或密码错误", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn join_group(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<JoinGroupRequest>,
) -> Result<Json<ApiResponse<JoinGroupResponse>>, AppError> {
    // 检查群组是否存在
    if !state
        .group_service
        .check_group_exists(payload.group_id)
        .await?
    {
        return Err(AppError::NotFound("群组不存在".to_string()));
    }

    // 检查用户是否已经在群组中
    if state
        .group_service
        .check_user_in_group(payload.group_id, current_user.user_id)
        .await?
    {
        return Err(AppError::BadRequest("您已经是群组成员".to_string()));
    }

    // 检查群组是否需要密码
    let need_password = state
        .group_service
        .check_group_need_password(payload.group_id)
        .await?;
    if need_password {
        match payload.password {
            Some(ref password) => {
                if !state
                    .group_service
                    .verify_group_password(payload.group_id, password)
                    .await?
                {
                    return Err(AppError::BadRequest("密码错误".to_string()));
                }
            }
            None => return Err(AppError::BadRequest("该群组需要密码".to_string())),
        }
    }

    let result = state
        .group_service
        .join_group(payload.group_id, current_user.user_id)
        .await?;

    // 发送成员加入通知
    if result.success {
        let ws_notification_service = state.websocket_notification_service.clone();
        let actor_id_for_notification = current_user.user_id;
        let actor_nickname_for_notification = current_user.nickname.clone();
        let group_id_for_notification = payload.group_id;
        // let group_name_for_notification = ???; // Group name might be useful but not directly available here. Fetch if needed.

        tokio::spawn(async move {
            let event_data = GroupEventData {
                group_id: group_id_for_notification,
                event_type: WsGroupEventType::MemberJoined,
                actor_id: actor_id_for_notification,
                actor_nickname: actor_nickname_for_notification,
                target_id: None,
                target_nickname: None,
                created_at: Utc::now(),
                // group_name: Some(group_name_for_notification), // if available and needed
                // message: Some(format!("{} 加入了群组", actor_nickname_for_notification)),
            };

            // Notify all members of the group including the new member (actor)
            if let Err(e) = ws_notification_service.notify_group_event(event_data, None).await {
                tracing::warn!("Failed to send group member joined notification: {:?}", e);
            }
        });
    }

    // 发布领域事件
    if result.success {
        state.event_bus.publish(DomainEvent::GroupJoined { group_id: payload.group_id.into(), user_id: current_user.user_id.into() }).await;
    }

    let role = result.role.unwrap_or_default();
    // 验证返回的角色是否有效
    group_validator::validate_group_role(&role)?;

    let response = crate::models::api::group::JoinGroupResponse {
        success: result.success,
        role,
    };
    Ok(success_response(response))
}

// 退出群组
#[utoipa::path(
    post,
    path = "/api/v1/group/leave",
    tag = "群组",
    request_body = LeaveGroupRequest,
    responses(
        (status = 200, description = "退出群组成功", body = ApiResponse<()>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 404, description = "群组未找到或不在群组中", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn leave_group(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<LeaveGroupRequest>,
) -> Result<Json<ApiResponse<LeaveGroupResponse>>, AppError> {
    // 检查群组是否存在
    if !state
        .group_service
        .check_group_exists(payload.group_id)
        .await?
    {
        return Err(AppError::NotFound("群组不存在".to_string()));
    }

    // 检查用户是否在群组中
    if !state
        .group_service
        .check_user_in_group(payload.group_id, current_user.user_id)
        .await?
    {
        return Err(AppError::BadRequest("您不是群组成员".to_string()));
    }

    // 检查用户是否是群主
    if state
        .group_service
        .check_user_is_group_owner(payload.group_id, current_user.user_id)
        .await?
    {
        return Err(AppError::BadRequest(
            "群主不能直接离开群组，请先转让群主身份".to_string(),
        ));
    }

    let result = state
        .group_service
        .leave_group(payload.group_id, current_user.user_id)
        .await?;
        
    // 发送成员离开通知
    if result.success {
        let ws_notification_service = state.websocket_notification_service.clone();
        let actor_id_for_notification = current_user.user_id;
        let actor_nickname_for_notification = current_user.nickname.clone();
        let group_id_for_notification = payload.group_id;

        tokio::spawn(async move {
            let event_data = GroupEventData {
                group_id: group_id_for_notification,
                event_type: WsGroupEventType::MemberLeft,
                actor_id: actor_id_for_notification,
                actor_nickname: actor_nickname_for_notification,
                target_id: None, // Not applicable for leave event
                target_nickname: None,
                created_at: Utc::now(),
            };

            // Notify all members of the group. The leaving member won't get it if they are already disconnected from the room.
            // If they should explicitly NOT get it even if still connected to other rooms, use Some(actor_id) exclusion.
            if let Err(e) = ws_notification_service.notify_group_event(event_data, None).await {
                tracing::warn!("Failed to send group member left notification: {:?}", e);
            }
        });
    }

    if result.success {
        state.event_bus.publish(DomainEvent::GroupLeft { group_id: payload.group_id.into(), user_id: current_user.user_id.into() }).await;
    }

    // 群组成员活跃度更新任务 - 暂时跳过，功能未实现
    #[cfg(feature = "jobs")]
    if let Some(_job_service) = &state.job_service {
        tracing::debug!("群组成员活跃度更新任务功能暂未实现");
    }

    let response = LeaveGroupResponse {
        success: result.success,
    };
    Ok(success_response(response))
}

// 列出群组成员
#[utoipa::path(
    post,
    path = "/api/v1/group/members",
    tag = "群组",
    request_body = GroupMembersRequest,
    responses(
        (status = 200, description = "列出群组成员成功", body = ApiResponse<GroupMembersResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 404, description = "群组未找到", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn group_members(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<GroupMembersRequest>,
) -> Result<Json<ApiResponse<GroupMembersResponse>>, AppError> {
    // 验证分页参数
    let (cursor, limit) = crate::utils::validation_pipeline::validators::extract_pagination(
        payload.cursor,
        payload.limit,
        20,
        100
    )?;
    
    // 验证群组ID
    if !state.group_service.check_group_exists(payload.group_id).await? {
        return Err(AppError::NotFound("群组不存在".to_string()));
    }

    // 验证用户是否是群组成员
    let is_member = state
        .group_service
        .check_user_in_group(payload.group_id, current_user.user_id)
        .await?;
    if !is_member {
        return Err(AppError::NotFound("您不是该群组成员".to_string()));
    }

    let results = state
        .group_service
        .get_group_members(payload.group_id, cursor, limit, current_user.user_id)
        .await?;

    // 直接使用 service 层返回的结果，无需转换
    let members = results.items;

    let response = ResponseMapper::create_paginated_response(
        members, 
        results.pagination.has_more, 
        results.pagination.next_cursor
    );

    Ok(success_response(response))
}

// 踢出群组成员
#[utoipa::path(
    post,
    path = "/api/v1/group/kick",
    tag = "群组",
    request_body = KickMemberRequest,
    responses(
        (status = 200, description = "踢出成员成功", body = ApiResponse<()>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 403, description = "无权限", body = ApiResponse<()>),
        (status = 404, description = "群组或成员未找到", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn kick_member(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<KickMemberRequest>,
) -> Result<Json<ApiResponse<KickMemberResponse>>, AppError> {
    // 检查群组是否存在
    if !state
        .group_service
        .check_group_exists(payload.group_id)
        .await?
    {
        return Err(AppError::NotFound("群组不存在".to_string()));
    }

    // 检查被踢用户是否在群组中
    if !state
        .group_service
        .check_user_in_group(payload.group_id, payload.user_id)
        .await?
    {
        return Err(AppError::BadRequest("该用户不是群组成员".to_string()));
    }

    // 检查操作者权限
    let is_owner = state
        .group_service
        .check_user_is_group_owner(payload.group_id, current_user.user_id)
        .await?;
    let is_admin = state
        .group_service
        .check_user_is_group_admin(payload.group_id, current_user.user_id)
        .await?;
    if !is_owner && !is_admin {
        return Err(AppError::BadRequest("您没有权限执行此操作".to_string()));
    }

    // 检查被踢用户是否是群主
    if state
        .group_service
        .check_user_is_group_owner(payload.group_id, payload.user_id)
        .await?
    {
        return Err(AppError::BadRequest("不能踢出群主".to_string()));
    }

    // 如果操作者是管理员，检查被踢用户是否也是管理员
    if !is_owner && is_admin {
        if state
            .group_service
            .check_user_is_group_admin(payload.group_id, payload.user_id)
            .await?
        {
            return Err(AppError::BadRequest("管理员不能踢出其他管理员".to_string()));
        }
    }

    let result = state
        .group_service
        .kick_member(payload.group_id, payload.user_id, current_user.user_id)
        .await?;

    // 发送成员被踢通知
    if result.success {
        let ws_notification_service = state.websocket_notification_service.clone();
        let actor_id_for_notification = current_user.user_id;
        let actor_nickname_for_notification = current_user.nickname.clone();
        let group_id_for_notification = payload.group_id;
        let target_user_id_for_notification = payload.user_id;

        // 需要获取被踢用户的昵称 for notification
        let target_nickname_for_notification = match state.user_service.get_user_by_id(target_user_id_for_notification, current_user.user_id).await {
            Ok(user_row) => user_row.nickname,
            Err(e) => {
                tracing::error!("Failed to get target user info for kick notification: {:?}", e);
                "未知用户".to_string() // Fallback nickname
            }
        };

        tokio::spawn(async move {
            let event_data = GroupEventData {
                group_id: group_id_for_notification,
                event_type: WsGroupEventType::MemberKicked,
                actor_id: actor_id_for_notification,
                actor_nickname: actor_nickname_for_notification,
                target_id: Some(target_user_id_for_notification),
                target_nickname: Some(target_nickname_for_notification),
                created_at: Utc::now(),
            };

            // Notify all members, including the actor. The kicked member might also get it before being removed from room.
            if let Err(e) = ws_notification_service.notify_group_event(event_data, None).await {
                tracing::warn!("Failed to send group member kicked notification: {:?}", e);
            }
        });
    }

    if result.success {
        state.event_bus.publish(DomainEvent::GroupKicked { group_id: payload.group_id.into(), actor_id: current_user.user_id.into(), target_id: payload.user_id.into() }).await;
    }

    let response = KickMemberResponse {
        success: result.success,
    };
    Ok(success_response(response))
}

// 转移群组所有权
#[utoipa::path(
    post,
    path = "/api/v1/group/transfer_ownership",
    tag = "群组",
    request_body = TransferOwnershipRequest,
    responses(
        (status = 200, description = "转移所有权成功", body = ApiResponse<()>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 403, description = "无权限", body = ApiResponse<()>),
        (status = 404, description = "群组或新所有者未找到", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn transfer_ownership(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<TransferOwnershipRequest>,
) -> Result<Json<ApiResponse<TransferOwnershipResponse>>, AppError> {
    // 检查群组是否存在
    if !state
        .group_service
        .check_group_exists(payload.group_id)
        .await?
    {
        return Err(AppError::NotFound("群组不存在".to_string()));
    }

    // 检查当前用户是否是群主
    if !state
        .group_service
        .check_user_is_group_owner(payload.group_id, current_user.user_id)
        .await?
    {
        return Err(AppError::BadRequest("只有群主才能转让群主身份".to_string()));
    }

    // 检查新群主是否在群组中
    if !state
        .group_service
        .check_user_in_group(payload.group_id, payload.new_owner_id)
        .await?
    {
        return Err(AppError::BadRequest("新群主必须是群组成员".to_string()));
    }
    
    // 验证目标角色是否合法
    group_validator::validate_group_role(group_roles::OWNER)?;
    group_validator::validate_group_role(group_roles::MEMBER)?;

    let result = state
        .group_service
        .transfer_ownership(payload.group_id, payload.new_owner_id, current_user.user_id)
        .await?;

    // 发送群主转让通知
    if result.success {
        let ws_notification_service = state.websocket_notification_service.clone();
        let actor_id_for_notification = current_user.user_id; // Old owner
        let actor_nickname_for_notification = current_user.nickname.clone();
        let group_id_for_notification = payload.group_id;
        let target_user_id_for_notification = payload.new_owner_id; // New owner

        // 需要获取新群主的昵称
        let target_nickname_for_notification = match state.user_service.get_user_by_id(target_user_id_for_notification, current_user.user_id).await {
            Ok(user_row) => user_row.nickname,
            Err(e) => {
                tracing::error!("Failed to get target user info for ownership transfer notification: {:?}", e);
                "未知用户".to_string() // Fallback nickname
            }
        };

        tokio::spawn(async move {
            let event_data = GroupEventData {
                group_id: group_id_for_notification,
                event_type: WsGroupEventType::OwnerTransferred,
                actor_id: actor_id_for_notification,
                actor_nickname: actor_nickname_for_notification,
                target_id: Some(target_user_id_for_notification),
                target_nickname: Some(target_nickname_for_notification),
                created_at: Utc::now(),
            };

            // Notify all members of the group.
            if let Err(e) = ws_notification_service.notify_group_event(event_data, None).await {
                tracing::warn!("Failed to send group ownership transferred notification: {:?}", e);
            }
        });
    }

    if result.success {
        state.event_bus.publish(DomainEvent::GroupOwnershipTransferred { group_id: payload.group_id.into(), from_user_id: current_user.user_id.into(), to_user_user_id: payload.new_owner_id.into() }).await;
    }

    let response = TransferOwnershipResponse {
        success: result.success,
    };
    Ok(success_response(response))
}

// 获取用户群组列表
#[utoipa::path(
    post,
    path = "/api/v1/group/user_groups",
    tag = "群组",
    request_body = GetUserGroupsRequest,
    responses(
        (status = 200, description = "获取用户群组成功", body = ApiResponse<GetUserGroupsResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn get_user_groups(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<GetUserGroupsRequest>,
) -> Result<Json<ApiResponse<GetUserGroupsResponse>>, AppError> {
    let result = state
        .group_service
        .get_user_groups(current_user.user_id, payload.limit, payload.offset)
        .await?;

    Ok(success_response(result))
}

// 获取用户创建的群组列表
#[utoipa::path(
    post,
    path = "/api/v1/group/owned_groups",
    tag = "群组",
    request_body = GetOwnedGroupsRequest,
    responses(
        (status = 200, description = "获取用户创建的群组成功", body = ApiResponse<GetOwnedGroupsResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn get_owned_groups(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<GetOwnedGroupsRequest>,
) -> Result<Json<ApiResponse<GetOwnedGroupsResponse>>, AppError> {
    let result = state
        .group_service
        .get_owned_groups(current_user.user_id, payload.limit, payload.offset)
        .await?;

    Ok(success_response(result))
}
