use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            CreateMomentRequest, EditMomentRequest, DeleteMomentRequest,
            SearchMomentsByKeywordRequest, SearchMomentsByTagRequest, SearchMomentsByLocationRequest,
            GetMomentCommentsRequest, GetUserMomentsListRequest, GetMyMomentsListRequest, MomentOpenTheDoorRequest,
            LikeMomentRequest, UnlikeMomentRequest, CheckMomentLikedRequest,
            FavoriteMomentRequest, UnfavoriteMomentRequest, CheckMomentFavoritedRequest,
            GetFavoritesListRequest, GetMomentLikesListRequest,
        },
        response::{
            ApiResponse, PaginatedData, MomentSearchItem, CommentListItem,
            CreateMomentData, MomentLikeItem, CheckLikedData, CheckFavoritedData,
        },
    },
    error::Result,
    services::moment_service::MomentService,
    state::AppState,
};


// ==================== Public APIs ====================

/// 按关键字搜索生活瞬间（全局）
pub async fn search_by_keyword(
    State(state): State<AppState>,
    Json(req): Json<SearchMomentsByKeywordRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.search_by_keyword(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 按标签搜索生活瞬间（全局）
pub async fn search_by_tag(
    State(state): State<AppState>,
    Json(req): Json<SearchMomentsByTagRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.search_by_tag(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 按位置搜索生活瞬间（范围过滤）
pub async fn search_by_location(
    State(state): State<AppState>,
    Json(req): Json<SearchMomentsByLocationRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.search_by_location(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取瞬间评论列表
pub async fn get_comments_list(
    State(state): State<AppState>,
    Json(req): Json<GetMomentCommentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<CommentListItem>>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_comments_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取指定用户生活瞬间历史（公开 API）
pub async fn get_user_moments_list(
    State(state): State<AppState>,
    Json(req): Json<GetUserMomentsListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_user_moments_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取当前用户生活瞬间历史（/me/ API）
pub async fn get_my_list(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetMyMomentsListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_my_moments_list(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取瞬间详情（通过 ID 获取指定瞬间的详情）
pub async fn open_the_door(
    State(state): State<AppState>,
    Json(req): Json<MomentOpenTheDoorRequest>,
) -> Result<Json<ApiResponse<MomentSearchItem>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.open_the_door(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取生活瞬间点赞列表
pub async fn get_likes_list(
    State(state): State<AppState>,
    Json(req): Json<GetMomentLikesListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentLikeItem>>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_likes_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}


// ==================== Protected APIs ====================

pub async fn create(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CreateMomentRequest>,
) -> Result<Json<ApiResponse<CreateMomentData>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    let moment_id = service.create_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success(CreateMomentData { id: moment_id })))
}


pub async fn update(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<EditMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    service.update_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn delete(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    service.delete_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


// ==================== 点赞 ====================

pub async fn like_moment(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<LikeMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    service.like_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn unlike_moment(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnlikeMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    service.unlike_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn check_liked(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CheckMomentLikedRequest>,
) -> Result<Json<ApiResponse<CheckLikedData>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    let is_liked = service.check_liked(user_id, &req).await?;
    Ok(Json(ApiResponse::success(CheckLikedData { is_liked })))
}


// ==================== 收藏 ====================

pub async fn favorite_moment(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<FavoriteMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    service.favorite_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn unfavorite_moment(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnfavoriteMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    service.unfavorite_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn check_favorited(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CheckMomentFavoritedRequest>,
) -> Result<Json<ApiResponse<CheckFavoritedData>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    let is_favorited = service.check_favorited(user_id, &req).await?;
    Ok(Json(ApiResponse::success(CheckFavoritedData { is_favorited })))
}


pub async fn get_favorites_list(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetFavoritesListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let service = MomentService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_favorites_list(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}
