use std::sync::Arc;
use axum::{Extension, Json, debug_handler, extract::State};
use tracing::{debug, info};

use crate::{
    middleware::auth::AppState,
    models::api::moment::{
        MomentInfo, CreateMomentRequest, CreateMomentResponse, DeleteMomentRequest,
        DeleteMomentResponse, SearchMomentByIdRequest, SearchMomentsByLocationResponse,
        SearchMomentsByLocationRequest, SearchMomentsByTagsRequest, SearchMomentsByTagsResponse,
        UserMomentsHistoryRequest, UserMomentsHistoryResponse,
    },
    models::api::user::AuthUser,
    utils::response::{ApiResponse, AppError, success_response},
    utils::ResponseMapper,
    validators::{moment_validator, common_validator},
};

// 创建动态
#[utoipa::path(
    post,
    path = "/api/v1/moment/create",
    tag = "动态",
    request_body = CreateMomentRequest,
    responses(
        (status = 200, description = "创建动态成功", body = ApiResponse<CreateMomentResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
    )
)]
#[debug_handler]
pub async fn create_moment(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<CreateMomentRequest>,
) -> Result<Json<ApiResponse<CreateMomentResponse>>, AppError> {
    info!(
        "【Controller】开始创建动态: 用户ID={}, 位置=({}, {}), 位置名称={}",
        current_user.user_id, payload.latitude, payload.longitude, payload.location_name
    );
    debug!(
        "【Controller】动态内容: {}, 标签: {:?}",
        &payload.description,
        payload.tags.as_deref().unwrap_or(&[])
    );

    // 验证动态参数
    moment_validator::validate_moment_creation_request(
        &payload.description,
        payload.latitude,
        payload.longitude,
        &payload.location_name,
        payload.tags.as_deref().unwrap_or(&[]),
    )?;

    let moment_id = state
        .moment_service
        .create_moment(
            current_user.user_id,
            &payload.description,
            payload.latitude,
            payload.longitude,
            &payload.location_name,
            payload.tags.as_deref().unwrap_or(&[]),
        )
        .await?;

    // 通过WebSocket发送动态通知
    let ws_notification_service = state.websocket_notification_service.clone();
    let notification_moment_id = moment_id.clone();
    let actor_id = current_user.user_id;
    let actor_nickname = current_user.nickname.clone();
    let location_name_for_notification = payload.location_name.clone();
    let description_for_notification = payload.description.clone();
    let latitude_for_notification = payload.latitude;
    let longitude_for_notification = payload.longitude;

    tokio::spawn(async move {
        use crate::ws::message_types::{SystemNotificationData, SystemNotificationType as WsSystemNotificationType};
        use chrono::Utc;
        use uuid::Uuid;

        let preview = description_for_notification.chars().take(50).collect::<String>();
        
        let system_notification_data = SystemNotificationData {
            notification_id: Uuid::new_v4(),
            notification_type: WsSystemNotificationType::NewMoment,
            title: "新动态".to_string(),
            content: format!("{} 在 {} 发布了新动态", actor_nickname, location_name_for_notification),
            data: Some(serde_json::json!({
                "moment_id": notification_moment_id.to_string(),
                "user_id": actor_id,
                "user_nickname": actor_nickname,
                "latitude": latitude_for_notification,
                "longitude": longitude_for_notification,
                "location_name": location_name_for_notification,
                "preview": preview
            })),
            created_at: Utc::now(),
        };

        if let Err(e) = ws_notification_service.notify_system_event(current_user.user_id, system_notification_data).await {
            tracing::error!("Failed to send new moment notification: {:?}", e);
        }
    });

    // 分派动态索引任务
    #[cfg(feature = "jobs")]
    if let Some(job_service) = &state.job_service {
        if let Err(e) = job_service.dispatch_moment_index(
            moment_id.clone(),
            current_user.user_id,
            payload.latitude,
            payload.longitude,
        ).await {
            error!("分派动态索引任务失败: {:?}", e);
        } else {
            info!("已分派动态索引任务: moment_id={}", moment_id);
        }
    }

    info!(
        "【Controller】动态创建成功: 用户ID={}, 动态ID={}",
        current_user.user_id, moment_id
    );
    Ok(success_response(CreateMomentResponse { moment_id }))
}

// 删除动态
#[utoipa::path(
    post,
    path = "/api/v1/moment/delete",
    tag = "动态",
    request_body = DeleteMomentRequest,
    responses(
        (status = 200, description = "删除动态成功", body = ApiResponse<DeleteMomentResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 403, description = "无权限", body = ApiResponse<()>),
        (status = 404, description = "动态未找到", body = ApiResponse<()>),
    )
)]
#[debug_handler]
pub async fn delete_moment(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<DeleteMomentRequest>,
) -> Result<Json<ApiResponse<DeleteMomentResponse>>, AppError> {
    info!(
        "【Controller】开始删除动态: 用户ID={}, 动态ID={}",
        current_user.user_id, payload.moment_id
    );

    let moment_id = &payload.moment_id;
    state
        .moment_service
        .delete_moment(moment_id)
        .await?;

    info!(
        "【Controller】动态删除成功: 用户ID={}, 动态ID={}",
        current_user.user_id, moment_id
    );
    Ok(success_response(DeleteMomentResponse { success: true }))
}

// 获取用户动态历史
#[utoipa::path(
    post,
    path = "/api/v1/moment/history/user",
    tag = "动态",
    request_body = UserMomentsHistoryRequest,
    responses(
        (status = 200, description = "获取用户动态历史成功", body = ApiResponse<UserMomentsHistoryResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
    )
)]
#[debug_handler]
pub async fn history(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<UserMomentsHistoryRequest>,
) -> Result<Json<ApiResponse<UserMomentsHistoryResponse>>, AppError> {
    let user_id = payload.user_id.unwrap_or_else(|| current_user.user_id);
    let cursor = payload.pagination.cursor.unwrap_or(0);
    let limit = payload.pagination.limit.unwrap_or(10);

    info!(
        "【Controller】开始获取用户动态历史: 当前用户ID={}, 查询用户ID={}, 游标={}, 限制={}",
        current_user.user_id, user_id, cursor, limit
    );

    // 验证分页参数
    if let (Some(c), Some(l)) = (payload.pagination.cursor, payload.pagination.limit) {
        common_validator::validate_pagination(c, l, 50)?;
    }

    let db_items = state
        .moment_service
        .search_moments_by_user(user_id, cursor, limit)
        .await?;

    debug!("【Controller】获取到{}条用户动态记录", db_items.len());

    let mut items = Vec::with_capacity(db_items.len());
    for db_info in db_items.iter() {
        items.push(MomentInfo {
            id: db_info.moment_id.parse::<i64>().unwrap_or(0),
            moment_id: db_info.moment_id.to_string(),
            user_id: db_info.user_id,
            nickname: db_info.user_nickname.clone(),
            description: Some(db_info.content.clone()),
            latitude: db_info.latitude,
            longitude: db_info.longitude,
            location_name: db_info.location_name.clone(),
            distance: db_info.distance.unwrap_or(0.0),
            created_at: db_info.created_at,
            tags: db_info.tags.clone(),
            likes_count: db_info.like_count,
            comments_count: db_info.comment_count,
            liked_by_me: db_info.is_liked,
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
        "【Controller】获取用户动态历史成功: 用户ID={}, 返回{}条记录, 是否有更多={}",
        user_id,
        items.len(),
        has_more
    );
    Ok(success_response(response))
}

// 获取附近动态
#[utoipa::path(
    post,
    path = "/api/v1/moment/search/location",
    tag = "动态",
    request_body = SearchMomentsByLocationRequest,
    responses(
        (status = 200, description = "获取附近动态成功", body = ApiResponse<SearchMomentByLocationResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
    )
)]
#[debug_handler]
pub async fn search_moment_by_location(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchMomentsByLocationRequest>,
) -> Result<Json<ApiResponse<SearchMomentsByLocationResponse>>, AppError> {
    info!(
        "【Controller】开始获取附近动态: 用户ID={}, 位置=({}, {}), 半径={}",
        current_user.user_id, payload.latitude, payload.longitude, payload.radius
    );

    // 验证位置参数
    common_validator::validate_location_params(payload.latitude, payload.longitude)?;

    // 验证搜索半径
    moment_validator::validate_search_radius(payload.radius)?;

    // 验证分页参数
    let cursor = payload.pagination.cursor.unwrap_or(0);
    let limit = payload.pagination.limit.unwrap_or(10);

    if let (Some(c), Some(l)) = (payload.pagination.cursor, payload.pagination.limit) {
        common_validator::validate_pagination(c, l, 50)?;
    }

    let db_items = state
        .moment_service
        .search_moments_by_location(
            current_user.user_id,
            payload.latitude,
            payload.longitude,
            payload.radius,
            cursor,
            limit,
        )
        .await?;

    debug!("【Controller】获取到{}条附近动态记录", db_items.len());

    let mut items = Vec::with_capacity(db_items.len());
    for db_info in db_items.iter() {
        items.push(MomentInfo {
            id: db_info.moment_id.parse::<i64>().unwrap_or(0),
            moment_id: db_info.moment_id.clone(),
            user_id: db_info.user_id,
            nickname: db_info.user_nickname.clone(),
            description: Some(db_info.content.clone()),
            latitude: db_info.latitude,
            longitude: db_info.longitude,
            location_name: db_info.location_name.clone(),
            distance: db_info.distance.unwrap_or(0.0),
            created_at: db_info.created_at,
            tags: db_info.tags.clone(),
            likes_count: db_info.like_count,
            comments_count: db_info.comment_count,
            liked_by_me: db_info.is_liked,
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
        "【Controller】获取附近动态成功: 用户ID={}, 位置=({}, {}), 返回{}条记录, 是否有更多={}",
        current_user.user_id,
        payload.latitude,
        payload.longitude,
        items.len(),
        has_more
    );
    Ok(success_response(response))
}

// 获取动态详情
#[utoipa::path(
    post,
    path = "/api/v1/moment/search/id",
    tag = "动态",
    request_body = SearchMomentByIdRequest,
    responses(
        (status = 200, description = "根据ID搜索动态成功", body = ApiResponse<MomentInfo>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 404, description = "动态未找到", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn search_moment_by_id(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchMomentByIdRequest>,
) -> Result<Json<ApiResponse<MomentInfo>>, AppError> {
    info!(
        "【Controller】开始获取动态详情: 用户ID={}, 动态ID={}",
        current_user.user_id, payload.moment_id
    );

    let db_info = state
        .moment_service
        .search_moment_by_id(current_user.user_id, &payload.moment_id)
        .await?;
    let api_info = MomentInfo {
        id: db_info.moment_id.parse::<i64>().unwrap_or(0),
        moment_id: db_info.moment_id.clone(),
        user_id: db_info.user_id,
        nickname: db_info.user_nickname,
        description: Some(db_info.content),
        latitude: db_info.latitude,
        longitude: db_info.longitude,
        location_name: db_info.location_name,
        distance: db_info.distance.unwrap_or(0.0),
        created_at: db_info.created_at,
        tags: db_info.tags,
        likes_count: db_info.like_count,
        comments_count: db_info.comment_count,
        liked_by_me: db_info.is_liked,
    };

    info!(
        "【Controller】获取动态详情成功: 用户ID={}, 动态ID={}, 发布用户ID={}",
        current_user.user_id, payload.moment_id, db_info.user_id
    );
    Ok(success_response(api_info))
}

// 根据标签搜索动态
#[utoipa::path(
    post,
    path = "/api/v1/moment/search/tags",
    tag = "动态",
    request_body = SearchMomentsByTagsRequest,
    responses(
        (status = 200, description = "根据标签搜索动态成功", body = ApiResponse<SearchMomentsByTagsResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
    )
)]
#[debug_handler]
pub async fn search_moment_by_tags(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchMomentsByTagsRequest>,
) -> Result<Json<ApiResponse<SearchMomentsByTagsResponse>>, AppError> {
    info!(
        "【Controller】开始根据标签搜索动态: 用户ID={}, 标签={:?}",
        current_user.user_id, payload.tags
    );

    let cursor = payload.pagination.cursor.unwrap_or(0);
    let limit = payload.pagination.limit.unwrap_or(10);

    let db_items = state
        .moment_service
        .search_moments_by_tags(current_user.user_id, &payload.tags, cursor, limit)
        .await?;

    debug!("【Controller】根据标签搜索获取到{}条记录", db_items.len());

    let mut items = Vec::with_capacity(db_items.len());
    for db_info in db_items.iter() {
        items.push(MomentInfo {
            id: db_info.moment_id.parse::<i64>().unwrap_or(0),
            moment_id: db_info.moment_id.to_string(),
            user_id: db_info.user_id,
            nickname: db_info.user_nickname.clone(),
            description: Some(db_info.content.clone()),
            latitude: db_info.latitude,
            longitude: db_info.longitude,
            location_name: db_info.location_name.clone(),
            distance: db_info.distance.unwrap_or(0.0),
            created_at: db_info.created_at,
            tags: db_info.tags.clone(),
            likes_count: db_info.like_count,
            comments_count: db_info.comment_count,
            liked_by_me: db_info.is_liked,
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
        "【Controller】根据标签搜索动态成功: 用户ID={}, 标签={:?}, 返回{}条记录, 是否有更多={}",
        current_user.user_id,
        payload.tags,
        items.len(),
        has_more
    );
    Ok(success_response(response))
}

// 点赞动态
#[utoipa::path(
    post,
    path = "/api/v1/moment/like",
    tag = "动态",
    request_body = SearchMomentByIdRequest,
    responses(
        (status = 200, description = "点赞动态成功", body = ApiResponse<bool>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 404, description = "动态未找到", body = ApiResponse<()>),
    )
)]
#[debug_handler]
pub async fn like_moment(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchMomentByIdRequest>,
) -> Result<Json<ApiResponse<bool>>, AppError> {
    info!(
        "【Controller】开始点赞动态: 用户ID={}, 动态ID={}",
        current_user.user_id, payload.moment_id
    );

    state
        .moment_service
        .like_moment(&payload.moment_id, current_user.user_id)
        .await?;

    info!(
        "【Controller】点赞动态成功: 用户ID={}, 动态ID={}",
        current_user.user_id, payload.moment_id
    );
    Ok(success_response(true))
}

// 取消点赞
#[utoipa::path(
    post,
    path = "/api/v1/moment/unlike",
    tag = "动态",
    request_body = SearchMomentByIdRequest,
    responses(
        (status = 200, description = "取消点赞动态成功", body = ApiResponse<bool>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 404, description = "动态未找到", body = ApiResponse<()>),
    )
)]
#[debug_handler]
pub async fn unlike_moment(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchMomentByIdRequest>,
) -> Result<Json<ApiResponse<bool>>, AppError> {
    info!(
        "【Controller】开始取消点赞动态: 用户ID={}, 动态ID={}",
        current_user.user_id, payload.moment_id
    );

    state
        .moment_service
        .unlike_moment(&payload.moment_id, current_user.user_id)
        .await?;

    info!(
        "【Controller】取消点赞动态成功: 用户ID={}, 动态ID={}",
        current_user.user_id, payload.moment_id
    );
    Ok(success_response(true))
}


