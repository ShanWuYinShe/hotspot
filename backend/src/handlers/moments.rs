use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            CreateMomentRequest, DeleteMomentRequest, EditMomentRequest, FavoriteMomentRequest,
            LikeMomentRequest, MomentLikesRequest, MomentOpenTheDoorRequest, MyFavoritesRequest,
            MyMomentsRequest, SearchMomentsRequest, UnfavoriteMomentRequest, UnlikeMomentRequest,
        },
        response::{ApiResponse, MomentItem, MomentLikeItem, PaginatedData},
    },
    error::Result,
    middleware::auth::{AuthenticatedUser, OptionalAuthenticatedUser},
    state::AppState,
};

// ==================== Token 可选 ====================

/// 统一搜索瞬间（支持关键字/标签/位置组合）
pub async fn search(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<SearchMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentItem>>>> {
    let sort_by = req.sort_by.as_deref().unwrap_or("relevance");
    tracing::debug!(sort_by = %sort_by, "Search sort parameter");

    let response = state.moment_service.search(user.user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取当前用户生活瞬间历史（/me/ API）
pub async fn get_my_moments(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentItem>>>> {
    let response = state
        .moment_service
        .get_my_moments_list(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取瞬间详情（通过 ID 获取指定瞬间的详情）
pub async fn open_the_door(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<MomentOpenTheDoorRequest>,
) -> Result<Json<ApiResponse<MomentItem>>> {
    let response = state
        .moment_service
        .open_the_door(user.user_id, &req)
        .await?;
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

// ==================== Token 必须 ====================

pub async fn create(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<CreateMomentRequest>,
) -> Result<Json<ApiResponse<MomentItem>>> {
    let moment = state
        .moment_service
        .create_moment(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(moment)))
}

pub async fn update(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<EditMomentRequest>,
) -> Result<Json<ApiResponse<MomentItem>>> {
    let moment = state
        .moment_service
        .update_moment(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(moment)))
}

pub async fn delete(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<DeleteMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .moment_service
        .delete_moment(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

// ==================== 点赞 ====================

pub async fn like(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<LikeMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.moment_service.like_moment(user.user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn unlike(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UnlikeMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .moment_service
        .unlike_moment(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

// ==================== 收藏 ====================

pub async fn favorite(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<FavoriteMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .moment_service
        .favorite_moment(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn unfavorite(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UnfavoriteMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .moment_service
        .unfavorite_moment(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn get_my_favorites(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyFavoritesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentItem>>>> {
    let response = state
        .moment_service
        .get_favorites_list(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}
