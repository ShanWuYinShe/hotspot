use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            CheckFavoritedRequest, CheckLikedRequest, CreateMomentRequest, DeleteMomentRequest,
            FavoriteMomentRequest, LikeMomentRequest, MomentCommentsRequest, MomentLikesRequest,
            MomentOpenTheDoorRequest, MyFavoritesRequest, MyMomentsRequest,
            SearchMomentsByKeywordRequest, SearchMomentsByLocationRequest,
            SearchMomentsByTagRequest, UnfavoriteMomentRequest, UnlikeMomentRequest,
            UpdateMomentRequest,
        },
        response::{
            ApiResponse, CheckFavoritedData, CheckLikedData, CommentListItem, CreateMomentData,
            MomentLikeItem, MomentSearchItem, PaginatedData,
        },
    },
    error::Result,
    state::AppState,
};

// ==================== Public APIs ====================

/// 按关键字搜索生活瞬间（全局）
pub async fn search_by_keyword(
    State(state): State<AppState>,
    Json(req): Json<SearchMomentsByKeywordRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let response = state.moment_service.search_by_keyword(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 按标签搜索生活瞬间（全局）
pub async fn search_by_tag(
    State(state): State<AppState>,
    Json(req): Json<SearchMomentsByTagRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let response = state.moment_service.search_by_tag(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 按位置搜索生活瞬间（范围过滤）
pub async fn search_by_location(
    State(state): State<AppState>,
    Json(req): Json<SearchMomentsByLocationRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let response = state.moment_service.search_by_location(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取瞬间评论列表
pub async fn get_comments(
    State(state): State<AppState>,
    Json(req): Json<MomentCommentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<CommentListItem>>>> {
    let response = state.moment_service.get_comments_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取当前用户生活瞬间历史（/me/ API）
pub async fn get_my_moments(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MyMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let response = state
        .moment_service
        .get_my_moments_list(user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取瞬间详情（通过 ID 获取指定瞬间的详情）
pub async fn open_the_door(
    State(state): State<AppState>,
    Json(req): Json<MomentOpenTheDoorRequest>,
) -> Result<Json<ApiResponse<MomentSearchItem>>> {
    let response = state.moment_service.open_the_door(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取生活瞬间点赞列表
pub async fn get_likes(
    State(state): State<AppState>,
    Json(req): Json<MomentLikesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentLikeItem>>>> {
    let response = state.moment_service.get_likes_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

// ==================== Protected APIs ====================

pub async fn create(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CreateMomentRequest>,
) -> Result<Json<ApiResponse<CreateMomentData>>> {
    let moment_id = state.moment_service.create_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success(CreateMomentData {
        id: moment_id,
    })))
}

pub async fn update(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.moment_service.update_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn delete(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.moment_service.delete_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

// ==================== 点赞 ====================

pub async fn like(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<LikeMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.moment_service.like_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn unlike(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnlikeMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.moment_service.unlike_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn check_liked(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CheckLikedRequest>,
) -> Result<Json<ApiResponse<CheckLikedData>>> {
    let is_liked = state.moment_service.check_liked(user_id, &req).await?;
    Ok(Json(ApiResponse::success(CheckLikedData { is_liked })))
}

// ==================== 收藏 ====================

pub async fn favorite(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<FavoriteMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.moment_service.favorite_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn unfavorite(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnfavoriteMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .moment_service
        .unfavorite_moment(user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn check_favorited(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CheckFavoritedRequest>,
) -> Result<Json<ApiResponse<CheckFavoritedData>>> {
    let is_favorited = state.moment_service.check_favorited(user_id, &req).await?;
    Ok(Json(ApiResponse::success(CheckFavoritedData {
        is_favorited,
    })))
}

pub async fn get_my_favorites(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<MyFavoritesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let response = state
        .moment_service
        .get_favorites_list(user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}
