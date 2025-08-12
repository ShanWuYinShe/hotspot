//! 消息相关控制器
use std::sync::Arc;

use axum::{Extension, Json, debug_handler, extract::State};

use crate::ws::message_types::RoomType;
use crate::utils::events::DomainEvent;
use crate::{
    middleware::auth::AppState,
    models::api::message::{
        ConversationsRequest, ConversationsResponse, DeleteDirectMessageRequest,
        DeleteGroupMessageRequest, DeleteMessageResponse, DirectMessageHistoryRequest,
        DirectMessageHistoryResponse, GetGroupUnreadCountRequest, GroupMessageHistoryRequest,
        GroupMessageHistoryResponse, MarkReadRequest, MarkReadResponse, SendDirectMessageRequest,
        SendDirectMessageResponse, SendGroupMessageRequest, SendGroupMessageResponse,
        UnreadCountResponse,
    },
    models::api::user::AuthUser,
    utils::{
        ResponseMapper,
        response::{ApiResponse, AppError, success_response},
        validation_pipeline::validators,
    },
    validators::{common_validator, message_validator},
}; // 修正RoomType的导入路径

// 发送群组文本消息
#[utoipa::path(
    post,
    path = "/api/v1/chat/group/send",
    tag = "消息",
    request_body = SendGroupMessageRequest,
    responses(
        (status = 200, description = "发送群组消息成功", body = ApiResponse<SendGroupMessageResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 403, description = "无权限或不在群组中", body = ApiResponse<()>),
        (status = 404, description = "群组未找到", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn send_group_message(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SendGroupMessageRequest>,
) -> Result<Json<ApiResponse<SendGroupMessageResponse>>, AppError> {
    // 验证消息内容
    message_validator::validate_message_content(&payload.content, &payload.message_type)?;

    // 验证位置参数（如果提供了位置信息）
    if payload.latitude.abs() > 0.0 || payload.longitude.abs() > 0.0 {
        common_validator::validate_location_params(payload.latitude, payload.longitude)?;
    }

    // 验证群组ID
    let group_id = common_validator::validate_uuid(&payload.group_id, "群组ID")?;

    // 发送消息（权限检查在service层进行）
    let send_group_message_result = state
        .message_service
        .send_group_message(
            group_id,
            current_user.user_id,
            &payload.content,
            &payload.message_type,
            payload.latitude,
            payload.longitude,
        )
        .await?;

    // 发布领域事件（事件处理器负责WS与作业等副作用）
    state.event_bus.publish(DomainEvent::MessageSent {
        room: crate::utils::types::RoomKey::new(RoomType::Group, group_id),
        message_id: send_group_message_result.message_id.into(),
        sender_id: current_user.user_id.into(),
        sender_nickname: current_user.nickname.clone(),
        content: payload.content.clone(),
        message_type: payload.message_type.clone(),
        created_at: send_group_message_result.created_at,
        latitude: if payload.latitude.abs() > 0.0 { Some(payload.latitude) } else { None },
        longitude: if payload.longitude.abs() > 0.0 { Some(payload.longitude) } else { None },
    }).await;

    // 触发群组成员活跃度更新任务
    #[cfg(feature = "jobs")]
    if let (Ok(group_uuid), Some(job_service)) =
        (uuid::Uuid::parse_str(&payload.group_id), &state.job_service)
    {
        if let Err(e) = job_service
            .dispatch_group_member_activity(group_uuid, current_user.user_id, "message_sent")
            .await
        {
            tracing::warn!("触发群组成员活跃度更新任务失败: {:?}", e);
        }
    }

    let response = SendGroupMessageResponse {
        group_id: payload.group_id.clone(),
        message_id: send_group_message_result.message_id.to_string(),
        message_type: payload.message_type.clone(),
        sent_at: chrono::Utc::now(),
    };
    Ok(success_response(response))
}

#[utoipa::path(
    post,
    path = "/api/v1/chat/group/history",
    tag = "消息",
    request_body = GroupMessageHistoryRequest,
    responses(
        (status = 200, description = "获取群组消息历史成功", body = ApiResponse<GroupMessageHistoryResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 403, description = "无权限或不在群组中", body = ApiResponse<()>),
        (status = 404, description = "群组未找到", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn group_history(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<GroupMessageHistoryRequest>,
) -> Result<Json<ApiResponse<GroupMessageHistoryResponse>>, AppError> {
    // 验证群组ID和分页参数
    let group_id =
        crate::utils::validation_pipeline::validators::validate_uuid(&payload.group_id, "群组ID")?;
    let (cursor, limit) = crate::utils::validation_pipeline::validators::extract_pagination(
        payload.pagination.cursor,
        payload.pagination.limit,
        20,
        50,
    )?;

    let (messages, has_more, next_cursor) = state
        .message_service
        .get_group_messages(group_id, current_user.user_id, cursor, limit)
        .await?;

    // 当用户查看群组消息时，更新活跃度
    #[cfg(feature = "jobs")]
    if let Some(job_service) = &state.job_service {
        if let Err(e) = job_service
            .dispatch_group_member_activity(group_id, current_user.user_id, "message_read")
            .await
        {
            tracing::warn!("触发群组成员活跃度更新任务失败: {:?}", e);
        }
    }

    let items = ResponseMapper::map_group_messages_to_history_items(messages);
    let response = ResponseMapper::create_paginated_response(items, has_more, next_cursor);

    Ok(success_response(response))
}

// 发送私聊消息
#[utoipa::path(
    post,
    path = "/api/v1/chat/direct/send",
    tag = "消息",
    request_body = SendDirectMessageRequest,
    responses(
        (status = 200, description = "发送私聊消息成功", body = ApiResponse<SendDirectMessageResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 403, description = "无权限或不在对话中", body = ApiResponse<()>),
        (status = 404, description = "对话未找到", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn send_direct_message(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SendDirectMessageRequest>,
) -> Result<Json<ApiResponse<SendDirectMessageResponse>>, AppError> {
    // 验证消息内容
    message_validator::validate_message_content(&payload.content, &payload.message_type)?;

    // 验证位置参数（如果提供了位置信息）
    if payload.latitude.abs() > 0.0 || payload.longitude.abs() > 0.0 {
        common_validator::validate_location_params(payload.latitude, payload.longitude)?;
    }

    // 验证接收者ID
    let recipient_id = message_validator::validate_recipient_id(&payload.recipient_id)?;

    // 获取或创建私信会话ID
    let conversation_id = state
        .message_service
        .get_or_create_conversation(current_user.user_id, recipient_id)
        .await?;

    let message = state
        .message_service
        .send_direct_message(
            conversation_id,
            current_user.user_id,
            recipient_id,
            &payload.content,
            &payload.message_type,
            payload.latitude,
            payload.longitude,
        )
        .await?;

    // 发布领域事件（事件处理器负责WS与作业等副作用）
    state.event_bus.publish(DomainEvent::MessageSent {
        room: crate::utils::types::RoomKey::new(RoomType::Conversation, conversation_id),
        message_id: message.message_id.into(),
        sender_id: current_user.user_id.into(),
        sender_nickname: current_user.nickname.clone(),
        content: payload.content.clone(),
        message_type: payload.message_type.clone(),
        created_at: message.created_at,
        latitude: if payload.latitude.abs() > 0.0 { Some(payload.latitude) } else { None },
        longitude: if payload.longitude.abs() > 0.0 { Some(payload.longitude) } else { None },
    }).await;

    let response = SendDirectMessageResponse {
        message_id: message.message_id.to_string(),
        conversation_id: message.conversation_id.to_string(),
        recipient_id: payload.recipient_id.clone(),
        message_type: message.message_type.clone(),
        sent_at: message.created_at,
    };
    Ok(success_response(response))
}

// 获取私聊消息历史
#[utoipa::path(
    post,
    path = "/api/v1/chat/direct/history",
    tag = "消息",
    request_body = DirectMessageHistoryRequest,
    responses(
        (status = 200, description = "获取私聊消息历史成功", body = ApiResponse<DirectMessageHistoryResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 403, description = "无权限或不在对话中", body = ApiResponse<()>),
        (status = 404, description = "对话未找到", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn user_history(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<DirectMessageHistoryRequest>,
) -> Result<Json<ApiResponse<DirectMessageHistoryResponse>>, AppError> {
    // 验证会话ID
    message_validator::validate_conversation_id(payload.conversation_id.as_ref())?;

    let conversation_id = crate::utils::validation_pipeline::validators::validate_uuid(
        payload.conversation_id.as_ref().unwrap(),
        "会话ID",
    )?;

    // 验证并提取分页参数
    let (cursor, limit) = crate::utils::validation_pipeline::validators::extract_pagination(
        payload.pagination.cursor,
        payload.pagination.limit,
        20,
        50,
    )?;

    let (messages, has_more, next_cursor) = state
        .message_service
        .get_direct_messages(conversation_id, current_user.user_id, cursor, limit)
        .await?;

    let items = ResponseMapper::map_messages_to_history_items(messages);
    let response = ResponseMapper::create_paginated_response(items, has_more, next_cursor);

    Ok(success_response(response))
}

// 获取用户会话列表
#[utoipa::path(
    post,
    path = "/api/v1/chat/conversations",
    tag = "消息",
    request_body = ConversationsRequest,
    responses(
        (status = 200, description = "获取用户会话列表成功", body = ApiResponse<ConversationsResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn user_conversations(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<ConversationsRequest>,
) -> Result<Json<ApiResponse<ConversationsResponse>>, AppError> {
    // 验证分页参数
    crate::utils::validation_pipeline::validators::validate_pagination_if_provided(
        payload.pagination.cursor,
        payload.pagination.limit,
        50,
    )?;

    let conversations = state
        .message_service
        .get_conversations(current_user.user_id)
        .await?;

    let items = conversations
        .into_iter()
        .map(|conv| crate::models::api::message::ConversationItem {
            conversation_id: conv.conversation_id.to_string(),
            peer_id: if conv.user1_id == current_user.user_id {
                conv.user2_id
            } else {
                conv.user1_id
            },
            peer_name: String::new(), // 可补充
            last_message_preview: conv.last_message_preview.unwrap_or_default(),
            last_message_time: conv.last_message_at,
            unread_count: 0, // 可补充
        })
        .collect();

    let response = ResponseMapper::create_paginated_response(items, false, None);

    Ok(success_response(response))
}

// 删除群组消息
#[utoipa::path(
    post,
    path = "/api/v1/chat/group/delete",
    tag = "消息",
    request_body = DeleteGroupMessageRequest,
    responses(
        (status = 200, description = "删除群组消息成功", body = ApiResponse<DeleteMessageResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 403, description = "无权限", body = ApiResponse<()>),
        (status = 404, description = "消息未找到", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn delete_group_message(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<DeleteGroupMessageRequest>,
) -> Result<Json<ApiResponse<DeleteMessageResponse>>, AppError> {
    let message_id_to_delete = common_validator::validate_uuid(&payload.message_id, "消息ID")?;

    // delete_group_message 现在返回 Result<Uuid, AppError> (group_id)
    let group_id_of_deleted_message = state
        .message_service
        .delete_group_message(message_id_to_delete, current_user.user_id)
        .await?;

    // 发布事件（WS 推送由事件处理器负责）
    // no-op: actor_id handled in event payload; keep if needed later
    state.event_bus.publish(DomainEvent::MessageDeleted { room: crate::utils::types::RoomKey::new(RoomType::Group, group_id_of_deleted_message), message_id: message_id_to_delete.into(), actor_id: Some(current_user.user_id.into()) }).await;

    Ok(success_response(DeleteMessageResponse { success: true }))
}

// 删除私聊消息
#[utoipa::path(
    post,
    path = "/api/v1/chat/direct/delete",
    tag = "消息",
    request_body = DeleteDirectMessageRequest,
    responses(
        (status = 200, description = "删除私聊消息成功", body = ApiResponse<DeleteMessageResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 403, description = "无权限", body = ApiResponse<()>),
        (status = 404, description = "消息未找到", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn delete_direct_message(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<DeleteDirectMessageRequest>,
) -> Result<Json<ApiResponse<DeleteMessageResponse>>, AppError> {
    let message_id_to_delete = common_validator::validate_uuid(&payload.message_id, "消息ID")?;

    // delete_direct_message 现在返回 Result<Uuid, AppError> (conversation_id)
    let conversation_id_of_deleted_message = state
        .message_service
        .delete_direct_message(message_id_to_delete, current_user.user_id)
        .await?;

    // 发布事件（WS 推送由事件处理器负责）
    // no-op: actor_id handled in event payload; keep if needed later
    state.event_bus.publish(DomainEvent::MessageDeleted { room: crate::utils::types::RoomKey::new(RoomType::Conversation, conversation_id_of_deleted_message), message_id: message_id_to_delete.into(), actor_id: Some(current_user.user_id.into()) }).await;

    Ok(success_response(DeleteMessageResponse { success: true }))
}

// 标记消息已读
#[utoipa::path(
    post,
    path = "/api/v1/chat/mark_read",
    tag = "消息",
    request_body = MarkReadRequest,
    responses(
        (status = 200, description = "标记消息已读成功", body = ApiResponse<MarkReadResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn mark_direct_messages_read(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<MarkReadRequest>,
) -> Result<Json<ApiResponse<MarkReadResponse>>, AppError> {
    // 会话ID是必须的
    if payload.conversation_id.is_empty() {
        return Err(AppError::BadRequest("必须提供会话ID".to_string()));
    }

    // 解析会话ID
    let conversation_id = common_validator::validate_uuid(&payload.conversation_id, "会话ID")?;

    // 解析消息ID（如果存在）
    let message_id = if let Some(msg_id) = payload.message_id {
        if msg_id.is_empty() {
            None
        } else {
            Some(common_validator::validate_uuid(&msg_id, "消息ID")?)
        }
    } else {
        None
    };

    // 调用service标记已读
    let _ = state
        .message_service
        .mark_direct_messages_read(conversation_id, current_user.user_id, message_id)
        .await?;

    // 发布事件（WS 推送由事件处理器负责）
    state.event_bus.publish(DomainEvent::DirectMessagesRead { conversation_id: conversation_id.into(), reader_id: current_user.user_id.into(), message_id: message_id.map(Into::into) }).await;

    // 构建响应
    let response = MarkReadResponse {
        success: true,
        marked_count: None, // 不再返回标记数量
    };

    Ok(success_response(response))
}

// 标记群组消息已读
#[utoipa::path(
    post,
    path = "/api/v1/chat/group/mark-read",
    tag = "消息",
    request_body = MarkReadRequest,
    responses(
        (status = 200, description = "标记群组消息已读成功", body = ApiResponse<MarkReadResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 403, description = "无权限或不在群组中", body = ApiResponse<()>),
        (status = 404, description = "群组未找到", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn mark_group_messages_read(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<MarkReadRequest>,
) -> Result<Json<ApiResponse<MarkReadResponse>>, AppError> {
    // 群组ID是必须的
    if payload.conversation_id.is_empty() {
        return Err(AppError::BadRequest("必须提供群组ID".to_string()));
    }

    // 解析群组ID
    let group_id = common_validator::validate_uuid(&payload.conversation_id, "群组ID")?;

    // 解析消息ID（如果存在）
    let message_id = if let Some(msg_id) = payload.message_id {
        if msg_id.is_empty() {
            None
        } else {
            Some(common_validator::validate_uuid(&msg_id, "消息ID")?)
        }
    } else {
        None
    };

    // 调用service标记已读
    let _ = state
        .message_service
        .mark_group_messages_read(group_id, current_user.user_id, message_id)
        .await?;

    // 发布事件（预留：如需群组已读通知，可在事件处理器实现）
    state.event_bus.publish(DomainEvent::GroupMessagesRead { group_id: group_id.into(), reader_id: current_user.user_id.into(), message_id: message_id.map(Into::into) }).await;
    // let _reader_id = current_user.user_id;

    // tokio::spawn(async move {
    //     // 注意：这里需要实际的会话ID，暂时注释掉WebSocket通知
    //     // 待实现获取群组会话ID的逻辑
    //     /*
    //     if let Err(e) = ws_notification_service.notify_message_read_to_recipient(
    //         reader_id,
    //         conversation_id, // 需要获取实际的会话ID
    //         message_id,
    //         chrono::Utc::now(),
    //     ).await {
    //         tracing::warn!(
    //             "Failed to send message read notification for group {}: {:?}",
    //             group_id,
    //             e
    //         );
    //     }
    //     */
    // });

    // 构建响应
    let response = MarkReadResponse {
        success: true,
        marked_count: None, // 不再返回标记数量
    };

    Ok(success_response(response))
}

// 获取未读消息计数
#[utoipa::path(
    get,
    path = "/api/v1/chat/unread_count",
    tag = "消息",
    responses(
        (status = 200, description = "获取未读消息计数成功", body = ApiResponse<crate::models::api::message::UnreadCountResponse>)
    )
)]
#[debug_handler]
pub async fn get_direct_message_unread_count(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
) -> Result<Json<ApiResponse<crate::models::api::message::UnreadCountResponse>>, AppError> {
    let unread_count = state
        .message_service
        .get_direct_message_unread_count(current_user.user_id)
        .await?;

    let response = crate::models::api::message::UnreadCountResponse { unread_count };

    Ok(success_response(response))
}

// 获取群组未读消息数量
#[utoipa::path(
    post,
    path = "/api/v1/chat/group/unread_count",
    tag = "消息",
    request_body = GetGroupUnreadCountRequest,
    responses(
        (status = 200, description = "获取群组未读消息数量成功", body = ApiResponse<UnreadCountResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 403, description = "无权限或不在群组中", body = ApiResponse<()>),
        (status = 404, description = "群组未找到", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn get_group_unread_count(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<GetGroupUnreadCountRequest>,
) -> Result<Json<ApiResponse<UnreadCountResponse>>, AppError> {
    // 验证群组ID格式
    let group_id = validators::validate_uuid(&payload.group_id, "group_id")?;

    // 验证用户是否在群组中
    let is_member = state
        .group_service
        .check_user_in_group(group_id, current_user.user_id)
        .await?;

    if !is_member {
        return Err(AppError::Forbidden("您不在该群组中".to_string()));
    }

    // 获取用户未读消息数量
    let unread_count = state
        .message_service
        .get_group_messages_unread_count(current_user.user_id)
        .await?;

    let response = UnreadCountResponse { unread_count };

    Ok(success_response(response))
}
