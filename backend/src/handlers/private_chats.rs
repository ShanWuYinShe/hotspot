use axum::{extract::State, Extension, Json};
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreatePrivateChatRequest, DeletePrivateChatRequest, GetPrivateChatsRequest,
            PrivateChatOpenTheDoorRequest, UpdatePrivateChatSettingsRequest,
        },
        response::{
            ApiResponse, CreatePrivateChatData, PaginatedData, PaginationMeta,
            PrivateChatDetailData, PrivateChatListItem, PrivateChatLastMessage, UserInfo,
        },
    },
    error::{AppError, Result},
    services::private_chat_service::PrivateChatService,
    state::AppState,
    repositories::{
        user_repo::UserRepository,
        private_message_repo::PrivateMessageRepository,
        chat_unread_repo::ChatUnreadRepository,
        user_follows_repo::UserFollowsRepository,
    },
    utils::validation::Validator,
};

/// 3.5.1 创建私聊
pub async fn create(
    State(state): State<AppState>,
    Extension(current_user_id): Extension<i64>,
    Json(req): Json<CreatePrivateChatRequest>,
) -> Result<Json<ApiResponse<CreatePrivateChatData>>> {
    let service = PrivateChatService::new(state.db.clone());
    
    let target_user_id = req.target_user_id;
    
    let chat = service.create_or_get(current_user_id, target_user_id).await?;
    
    Ok(Json(ApiResponse::success(CreatePrivateChatData {
        id: chat.private_chat_id.to_string(),
    })))
}

/// 3.5.2 获取私聊列表
pub async fn get_list(
    State(state): State<AppState>,
    Extension(current_user_id): Extension<i64>,
    Json(req): Json<GetPrivateChatsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<PrivateChatListItem>>>> {
    let service = PrivateChatService::new(state.db.clone());
    let user_repo = UserRepository::new(state.db.clone());
    let message_repo = PrivateMessageRepository::new(state.db.clone());
    let unread_repo = ChatUnreadRepository::new(state.db.clone());
    
    let limit = Validator::validate_limit(req.limit, 100)? as i64;
    let chats = service.list_by_user(current_user_id, limit + 1, req.cursor.clone()).await?;
    
    let has_more = chats.len() > limit as usize;
    let chats: Vec<_> = chats.into_iter().take(limit as usize).collect();
    let next_cursor = if has_more {
        chats.last().map(|c| c.private_chat_id.to_string())
    } else {
        None
    };
    
    let mut items = Vec::new();
    for chat in chats {
        // 获取对方用户信息
        let other_user_id = if chat.user_a_id == current_user_id {
            chat.user_b_id
        } else {
            chat.user_a_id
        };
        
        let other_user = user_repo.find_by_id(other_user_id).await?;
        let other_user_info = other_user.map(|u| UserInfo {
            user_id: u.user_id,
            nickname: u.nickname,
            avatar_url: u.avatar_url,
            user_type: u.user_type,
        }).unwrap_or_else(|| UserInfo {
            user_id: other_user_id,
            nickname: "未知用户".to_string(),
            avatar_url: None,
            user_type: crate::types::enums::UserType::Normal,
        });
        
        // 获取未读计数
        let unread_count = unread_repo.get_unread_count(
            "private_chat",
            &chat.private_chat_id.to_string(),
            current_user_id,
        ).await.unwrap_or(0);
        
        // 获取最后一条消息
        let last_message = message_repo.get_last_message(chat.private_chat_id).await?
            .map(|m| PrivateChatLastMessage {
                id: m.message_id.to_string(),
                content: Some(m.content),
                r#type: m.r#type.as_str().to_string(),
                sender_id: m.sender_id,
                sent_at: m.sent_at.to_rfc3339(),
            });
        
        let last_activity_at = last_message.as_ref()
            .map(|m| m.sent_at.clone())
            .unwrap_or_else(|| chat.created_at.to_rfc3339());
        
        items.push(PrivateChatListItem {
            id: chat.private_chat_id.to_string(),
            other_user: other_user_info,
            unread_count: unread_count as i32,
            last_message,
            last_activity_at,
        });
    }
    
    Ok(Json(ApiResponse::success(PaginatedData {
        items,
        pagination: PaginationMeta {
            has_more,
            next_cursor,
        },
    })))
}

/// 获取私聊详情 - POST /api/v1/protected/private_chats/open_the_door
pub async fn open_the_door(
    State(state): State<AppState>,
    Extension(current_user_id): Extension<i64>,
    Json(req): Json<PrivateChatOpenTheDoorRequest>,
) -> Result<Json<ApiResponse<PrivateChatDetailData>>> {
    let service = PrivateChatService::new(state.db.clone());
    let user_repo = UserRepository::new(state.db.clone());
    let message_repo = PrivateMessageRepository::new(state.db.clone());
    let unread_repo = ChatUnreadRepository::new(state.db.clone());
    
    let private_chat_id = Uuid::parse_str(&req.id)
        .map_err(|_| AppError::BadRequest("无效的私聊ID".to_string()))?;
    
    let chat = service.find_by_id(private_chat_id).await?
        .ok_or_else(|| AppError::NotFound("私聊不存在".to_string()))?;
    
    // 获取对方用户信息
    let other_user_id = if chat.user_a_id == current_user_id {
        chat.user_b_id
    } else {
        chat.user_a_id
    };
    
    let other_user = user_repo.find_by_id(other_user_id).await?;
    let other_user_info = other_user.map(|u| UserInfo {
        user_id: u.user_id,
        nickname: u.nickname,
        avatar_url: u.avatar_url,
        user_type: u.user_type,
    }).unwrap_or_else(|| UserInfo {
        user_id: other_user_id,
        nickname: "未知用户".to_string(),
        avatar_url: None,
        user_type: crate::types::enums::UserType::Normal,
    });
    
    // 获取设置
    let settings = service.get_settings(private_chat_id, current_user_id).await?;
    
    // 获取未读计数
    let unread_count = unread_repo.get_unread_count(
        "private_chat",
        &chat.private_chat_id.to_string(),
        current_user_id,
    ).await.unwrap_or(0);
    
    // 获取最后一条消息
    let last_message = message_repo.get_last_message(chat.private_chat_id).await?
        .map(|m| PrivateChatLastMessage {
            id: m.message_id.to_string(),
            content: Some(m.content),
            r#type: m.r#type.as_str().to_string(),
            sender_id: m.sender_id,
            sent_at: m.sent_at.to_rfc3339(),
        });
    
    let last_activity_at = last_message.as_ref()
        .map(|m| m.sent_at.clone())
        .unwrap_or_else(|| chat.created_at.to_rfc3339());
    
    // 获取消息限制信息
    // 通过关系判断消息限制
    let follows_repo = UserFollowsRepository::new(state.db.clone());
    let is_following = follows_repo.exists(current_user_id, other_user_id).await.unwrap_or(false);
    let is_followed_by = follows_repo.exists(other_user_id, current_user_id).await.unwrap_or(false);
    let is_mutual_follow = is_following && is_followed_by;
    
    let message_limit = if is_mutual_follow {
        None
    } else {
        let (can_message_freely, sent_count) = if chat.user_a_id == current_user_id {
            (chat.user_a_can_message_freely, chat.user_a_sent_count)
        } else {
            (chat.user_b_can_message_freely, chat.user_b_sent_count)
        };
        
        if can_message_freely {
            None
        } else {
            let total = if is_following { 3 } else { 1 };
            let remaining = (total - sent_count).max(0);
            
            Some(crate::dto::response::MessageLimit {
                remaining,
                total,
                is_unlocked: false,
            })
        }
    };

    Ok(Json(ApiResponse::success(PrivateChatDetailData {
        id: chat.private_chat_id.to_string(),
        other_user: other_user_info,
        message_limit,
        unread_count: unread_count as i32,
        is_muted: settings.as_ref().map(|s| s.is_muted).unwrap_or(false),
        is_pinned: settings.as_ref().map(|s| s.is_pinned).unwrap_or(false),
        is_archived: settings.as_ref().map(|s| s.is_archived).unwrap_or(false),
        last_message,
        last_activity_at,
        created_at: chat.created_at.to_rfc3339(),
    })))
}

/// 3.5.4 更新私聊设置
pub async fn update_settings(
    State(state): State<AppState>,
    Extension(current_user_id): Extension<i64>,
    Json(req): Json<UpdatePrivateChatSettingsRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = PrivateChatService::new(state.db.clone());
    
    let private_chat_id = Uuid::parse_str(&req.id)
        .map_err(|_| AppError::BadRequest("无效的私聊ID".to_string()))?;
    
    service.update_settings(
        private_chat_id,
        current_user_id,
        req.is_muted,
        req.is_pinned,
        req.is_archived,
    ).await?;
    
    Ok(Json(ApiResponse::success_empty()))
}

/// 3.5.5 删除私聊
pub async fn delete(
    State(state): State<AppState>,
    Extension(current_user_id): Extension<i64>,
    Json(req): Json<DeletePrivateChatRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = PrivateChatService::new(state.db.clone());
    
    let private_chat_id = Uuid::parse_str(&req.id)
        .map_err(|_| AppError::BadRequest("无效的私聊ID".to_string()))?;
    
    service.delete(private_chat_id, current_user_id).await?;
    
    Ok(Json(ApiResponse::success_empty()))
}
