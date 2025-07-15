use std::sync::Arc;

use axum::{Extension, Json, debug_handler, extract::State};
use tracing::{debug, info, error};


use crate::{
    middleware::auth::AppState,
    models::api::checkin::{
        CheckInInfo, CreateCheckinRequest, CreateCheckinResponse, DeleteCheckinRequest,
        DeleteCheckinResponse, SearchCheckInByIdRequest, SearchCheckinByLocationResponse,
        SearchCheckinsByLocationRequest, SearchCheckinsByTagsRequest, SearchCheckinsByTagsResponse,
        UserCheckinsHistoryRequest, UserCheckinsHistoryResponse,
    },
    models::api::user::AuthUser,
    utils::response::{ApiResponse, AppError, success_response},
    utils::ResponseMapper,
    validators::{checkin_validator, common_validator},
};

// 创建签到
#[utoipa::path(
    post,
    path = "/api/v1/checkin/create",
    tag = "签到",
    request_body = CreateCheckinRequest,
    responses(
        (status = 200, description = "创建签到成功", body = ApiResponse<CreateCheckinResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
    )
)]
#[debug_handler]
pub async fn create_checkin(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<CreateCheckinRequest>,
) -> Result<Json<ApiResponse<CreateCheckinResponse>>, AppError> {
    info!(
        "【Controller】开始创建签到: 用户ID={}, 位置=({}, {}), 位置名称={}",
        current_user.user_id, payload.latitude, payload.longitude, payload.location_name
    );
    debug!(
        "【Controller】签到内容: {}, 标签: {:?}",
        &payload.description,
        payload.tags.as_deref().unwrap_or(&[])
    );

    // 验证签到参数
    checkin_validator::validate_checkin_creation_request(
        &payload.description,
        payload.latitude,
        payload.longitude,
        &payload.location_name,
        payload.tags.as_deref().unwrap_or(&[]),
    )?;

    let checkin_id = state
        .checkin_service
        .create_checkin(
            current_user.user_id,
            &payload.description,
            payload.latitude,
            payload.longitude,
            &payload.location_name,
            payload.tags.as_deref().unwrap_or(&[]),
        )
        .await?;

    // 通过WebSocket发送签到通知 (使用 WebSocketNotificationService)
    let ws_notification_service = state.websocket_notification_service.clone();
    let notification_checkin_id = checkin_id.clone(); // Clone for notification use
    let actor_id = current_user.user_id;
    let actor_nickname = current_user.nickname.clone();
    let location_name_for_notification = payload.location_name.clone();
    let description_for_notification = payload.description.clone(); // For preview
    let latitude_for_notification = payload.latitude;
    let longitude_for_notification = payload.longitude;

    tokio::spawn(async move {
        use crate::ws::message_types::{SystemNotificationData, SystemNotificationType as WsSystemNotificationType};
        use chrono::Utc;
        use uuid::Uuid;

        let preview = description_for_notification.chars().take(50).collect::<String>();
        
        let system_notification_data = SystemNotificationData {
            notification_id: Uuid::new_v4(), // This should ideally be the checkin_id itself or derived, if it's a Uuid
            notification_type: WsSystemNotificationType::NewCheckIn, // Assuming this variant exists or will be added
            title: "新签到".to_string(),
            content: format!("{} 在 {} 签到了", actor_nickname, location_name_for_notification),
            data: Some(serde_json::json!({
                "checkin_id": notification_checkin_id.to_string(), // Ensure it's string if checkin_id is Uuid
                "user_id": actor_id,
                "user_nickname": actor_nickname, // already captured
                "latitude": latitude_for_notification,
                "longitude": longitude_for_notification,
                "location_name": location_name_for_notification, // already captured
                "preview": preview
            })),
            created_at: Utc::now(),
        };

        // Notify the user who created the checkin about the successful creation
        if let Err(e) = ws_notification_service.notify_system_event(current_user.user_id, system_notification_data).await {
            tracing::error!("Failed to send new check-in notification: {:?}", e);
        }
    });

    // 分派签到索引任务
    if let Some(job_service) = &state.job_service {
        if let Err(e) = job_service.dispatch_checkin_index(
            checkin_id.clone(),
            current_user.user_id,
            payload.latitude,
            payload.longitude,
        ).await {
            error!("分派签到索引任务失败: {:?}", e);
        } else {
            info!("已分派签到索引任务: checkin_id={}", checkin_id);
        }

        // 分派附近签到通知任务 (假设半径1000米内的用户)
        // 注意：当前版本暂不实现附近用户通知功能，预留接口供后续扩展
        let nearby_user_ids: Vec<i64> = vec![];
        if !nearby_user_ids.is_empty() {
            if let Err(e) = job_service.dispatch_nearby_checkin_notification(
                checkin_id.clone(),
                current_user.user_id,
                current_user.nickname.clone(),
                nearby_user_ids,
                payload.latitude,
                payload.longitude,
                payload.description.clone(),
            ).await {
                error!("分派附近签到通知任务失败: {:?}", e);
            } else {
                info!("已分派附近签到通知任务: checkin_id={}", checkin_id);
            }
        }
    }

    info!(
        "【Controller】签到创建成功: 用户ID={}, 签到ID={}",
        current_user.user_id, checkin_id
    );
    Ok(success_response(CreateCheckinResponse { checkin_id }))
}

// 删除签到
#[utoipa::path(
    post,
    path = "/api/v1/checkin/delete",
    tag = "签到",
    request_body = DeleteCheckinRequest,
    responses(
        (status = 200, description = "删除签到成功", body = ApiResponse<DeleteCheckinResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 403, description = "无权限", body = ApiResponse<()>),
        (status = 404, description = "签到未找到", body = ApiResponse<()>),
    )
)]
#[debug_handler]
pub async fn delete_checkin(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<DeleteCheckinRequest>,
) -> Result<Json<ApiResponse<DeleteCheckinResponse>>, AppError> {
    info!(
        "【Controller】开始删除签到: 用户ID={}, 签到ID={}",
        current_user.user_id, payload.checkin_id
    );

    let checkin_id = payload.checkin_id.parse().map_err(|_| AppError::BadRequest("无效的签到ID".to_string()))?;
    state
        .checkin_service
        .delete_checkin_record(checkin_id)
        .await?;

    info!(
        "【Controller】签到删除成功: 用户ID={}, 签到ID={}",
        current_user.user_id, checkin_id
    );
    Ok(success_response(DeleteCheckinResponse { success: true }))
}

// 获取用户签到历史
#[utoipa::path(
    post,
    path = "/api/v1/checkin/history/user",
    tag = "签到",
    request_body = UserCheckinsHistoryRequest,
    responses(
        (status = 200, description = "获取用户签到历史成功", body = ApiResponse<PaginatedResponse<UserCheckinsHistoryResponse>>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
    )
)]
#[debug_handler]
pub async fn history(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<UserCheckinsHistoryRequest>,
) -> Result<Json<ApiResponse<UserCheckinsHistoryResponse>>, AppError> {
    let user_id = payload.user_id.unwrap_or_else(|| current_user.user_id);
    let cursor = payload.pagination.cursor.unwrap_or(0);
    let limit = payload.pagination.limit.unwrap_or(10);

    info!(
        "【Controller】开始获取用户签到历史: 当前用户ID={}, 查询用户ID={}, 游标={}, 限制={}",
        current_user.user_id, user_id, cursor, limit
    );

    // 验证分页参数
    if let (Some(c), Some(l)) = (payload.pagination.cursor, payload.pagination.limit) {
        common_validator::validate_pagination(c, l, 50)?;
    }

    let db_items = state
        .checkin_service
        .search_checkins_by_user(user_id, cursor, limit)
        .await?;

    debug!("【Controller】获取到{}条用户签到记录", db_items.len());

    let mut items = Vec::with_capacity(db_items.len());
    for db_info in db_items.iter() {
        items.push(CheckInInfo {
            id: db_info.id,
            checkin_id: db_info.checkin_id.to_string(),
            user_id: db_info.user_id,
            nickname: db_info.nickname.clone(),
            description: db_info.description.clone(),
            latitude: db_info.latitude,
            longitude: db_info.longitude,
            location_name: db_info.location_name.clone(),
            distance: db_info.distance,
            created_at: db_info.created_at,
            tags: db_info.tags.clone(),
            likes_count: db_info.likes_count,
            comments_count: 0,
            liked_by_me: db_info.liked_by_me,
        });
    }
    let has_more = items.len() as i64 > payload.pagination.limit.unwrap_or(10);
    let next_cursor = items.last().map(|item| item.id);
    let items = if has_more {
        items[..items.len() - 1].to_vec()
    } else {
        items
    };
    let response = ResponseMapper::create_paginated_response(
        items.clone(),
        has_more,
        next_cursor
    );

    info!(
        "【Controller】获取用户签到历史成功: 用户ID={}, 返回{}条记录, 是否有更多={}",
        user_id,
        items.len(),
        has_more
    );
    Ok(success_response(response))
}

// 获取附近签到
#[utoipa::path(
    post,
    path = "/api/v1/checkin/search/location",
    tag = "签到",
    request_body = SearchCheckinsByLocationRequest,
    responses(
        (status = 200, description = "获取附近签到成功", body = ApiResponse<SearchCheckinByLocationResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
    )
)]
#[debug_handler]
pub async fn search_checkin_by_location(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchCheckinsByLocationRequest>,
) -> Result<Json<ApiResponse<SearchCheckinByLocationResponse>>, AppError> {
    info!(
        "【Controller】开始获取附近签到: 用户ID={}, 位置=({}, {}), 半径={}",
        current_user.user_id, payload.latitude, payload.longitude, payload.radius
    );

    // 验证位置参数
    common_validator::validate_location_params(payload.latitude, payload.longitude)?;

    // 验证搜索半径
    checkin_validator::validate_search_radius(payload.radius)?;

    // 验证分页参数
    let cursor = payload.pagination.cursor.unwrap_or(0);
    let limit = payload.pagination.limit.unwrap_or(10);

    if let (Some(c), Some(l)) = (payload.pagination.cursor, payload.pagination.limit) {
        common_validator::validate_pagination(c, l, 50)?;
    }

    let db_items = state
        .checkin_service
        .search_checkins_by_location(
            current_user.user_id,
            payload.latitude,
            payload.longitude,
            payload.radius,
            cursor,
            limit,
        )
        .await?;

    debug!("【Controller】获取到{}条附近签到记录", db_items.len());

    let mut items = Vec::with_capacity(db_items.len());
    for db_info in db_items.iter() {
        items.push(CheckInInfo {
            id: db_info.id,
            checkin_id: db_info.checkin_id.clone(),
            user_id: db_info.user_id,
            nickname: db_info.nickname.clone(),
            description: db_info.description.clone(),
            latitude: db_info.latitude,
            longitude: db_info.longitude,
            location_name: db_info.location_name.clone(),
            distance: db_info.distance,
            created_at: db_info.created_at,
            tags: db_info.tags.clone(),
            likes_count: db_info.likes_count,
            comments_count: db_info.comments_count,
            liked_by_me: db_info.liked_by_me,
        });
    }
    let has_more = items.len() as i64 > limit;
    let next_cursor = items.last().map(|item| item.id);
    let items = if has_more {
        items[..items.len() - 1].to_vec()
    } else {
        items
    };
    let response = ResponseMapper::create_paginated_response(
        items.clone(),
        has_more,
        next_cursor
    );

    info!(
        "【Controller】获取附近签到成功: 用户ID={}, 位置=({}, {}), 返回{}条记录, 是否有更多={}",
        current_user.user_id,
        payload.latitude,
        payload.longitude,
        items.len(),
        has_more
    );
    Ok(success_response(response))
}

// 获取签到详情
#[utoipa::path(
    post,
    path = "/api/v1/checkin/search/id",
    tag = "签到",
    request_body = SearchCheckInByIdRequest,
    responses(
        (status = 200, description = "根据ID搜索签到成功", body = ApiResponse<CheckInInfo>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 404, description = "签到未找到", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn search_checkin_by_id(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchCheckInByIdRequest>,
) -> Result<Json<ApiResponse<CheckInInfo>>, AppError> {
    info!(
        "【Controller】开始获取签到详情: 用户ID={}, 签到ID={}",
        current_user.user_id, payload.checkin_id
    );

    let db_info = state
        .checkin_service
        .search_checkin_by_id(current_user.user_id, &payload.checkin_id)
        .await?;
    let api_info = CheckInInfo {
        id: db_info.id,
        checkin_id: db_info.checkin_id.clone(),
        user_id: db_info.user_id,
        nickname: db_info.nickname,
        description: db_info.description,
        latitude: db_info.latitude,
        longitude: db_info.longitude,
        location_name: db_info.location_name,
        distance: db_info.distance,
        created_at: db_info.created_at,
        tags: db_info.tags,
        likes_count: db_info.likes_count,
        comments_count: db_info.comments_count,
        liked_by_me: db_info.liked_by_me,
    };

    info!(
        "【Controller】获取签到详情成功: 用户ID={}, 签到ID={}, 发布用户ID={}",
        current_user.user_id, payload.checkin_id, db_info.user_id
    );
    Ok(success_response(api_info))
}

// 根据标签搜索签到
#[utoipa::path(
    post,
    path = "/api/v1/checkin/search/tags",
    tag = "签到",
    request_body = SearchCheckinsByTagsRequest,
    responses(
        (status = 200, description = "根据标签搜索签到成功", body = ApiResponse<SearchCheckinsByTagsResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
    )
)]
#[debug_handler]
pub async fn search_checkin_by_tags(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchCheckinsByTagsRequest>,
) -> Result<Json<ApiResponse<SearchCheckinsByTagsResponse>>, AppError> {
    info!(
        "【Controller】开始根据标签搜索签到: 用户ID={}, 标签={:?}",
        current_user.user_id, payload.tags
    );

    let cursor = payload.pagination.cursor.unwrap_or(0);
    let limit = payload.pagination.limit.unwrap_or(10);

    let db_items = state
        .checkin_service
        .search_checkins_by_tags(current_user.user_id, &payload.tags, cursor, limit)
        .await?;

    debug!("【Controller】根据标签搜索获取到{}条记录", db_items.len());

    let mut items = Vec::with_capacity(db_items.len());
    for db_info in db_items.iter() {
        items.push(CheckInInfo {
            id: db_info.id,
            checkin_id: db_info.checkin_id.to_string(),
            user_id: db_info.user_id,
            nickname: db_info.nickname.clone(),
            description: db_info.description.clone(),
            latitude: db_info.latitude,
            longitude: db_info.longitude,
            location_name: db_info.location_name.clone(),
            distance: db_info.distance,
            created_at: db_info.created_at,
            tags: db_info.tags.clone(),
            likes_count: db_info.likes_count,
            comments_count: db_info.comments_count,
            liked_by_me: db_info.liked_by_me,
        });
    }
    let has_more = items.len() as i64 > limit;
    let next_cursor = items.last().map(|item| item.id);
    let items = if has_more {
        items[..items.len() - 1].to_vec()
    } else {
        items
    };
    let response = ResponseMapper::create_paginated_response(
        items.clone(),
        has_more,
        next_cursor
    );

    info!(
        "【Controller】根据标签搜索签到成功: 用户ID={}, 标签={:?}, 返回{}条记录, 是否有更多={}",
        current_user.user_id,
        payload.tags,
        items.len(),
        has_more
    );
    Ok(success_response(response))
}

// 点赞签到
#[utoipa::path(
    post,
    path = "/api/v1/checkin/like",
    tag = "签到",
    request_body = SearchCheckInByIdRequest,
    responses(
        (status = 200, description = "点赞签到成功", body = ApiResponse<bool>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 404, description = "签到未找到", body = ApiResponse<()>),
    )
)]
#[debug_handler]
pub async fn like_checkin(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchCheckInByIdRequest>,
) -> Result<Json<ApiResponse<bool>>, AppError> {
    info!(
        "【Controller】开始点赞签到: 用户ID={}, 签到ID={}",
        current_user.user_id, payload.checkin_id
    );

    state
        .checkin_service
        .like_checkin(current_user.user_id, &payload.checkin_id)
        .await?;

    info!(
        "【Controller】点赞签到成功: 用户ID={}, 签到ID={}",
        current_user.user_id, payload.checkin_id
    );
    Ok(success_response(true))
}

// 取消点赞
#[utoipa::path(
    post,
    path = "/api/v1/checkin/unlike",
    tag = "签到",
    request_body = SearchCheckInByIdRequest,
    responses(
        (status = 200, description = "取消点赞签到成功", body = ApiResponse<bool>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 404, description = "签到未找到", body = ApiResponse<()>),
    )
)]
#[debug_handler]
pub async fn unlike_checkin(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchCheckInByIdRequest>,
) -> Result<Json<ApiResponse<bool>>, AppError> {
    info!(
        "【Controller】开始取消点赞: 用户ID={}, 签到ID={}",
        current_user.user_id, payload.checkin_id
    );

    state
        .checkin_service
        .unlike_checkin(&payload.checkin_id, current_user.user_id)
        .await?;

    info!(
        "【Controller】取消点赞成功: 用户ID={}, 签到ID={}",
        current_user.user_id, payload.checkin_id
    );
    Ok(success_response(true))
}
