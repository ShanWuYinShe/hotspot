use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            CheckFavoritedRequest, CheckLikedRequest, CreateMomentRequest, DeleteMomentRequest,
            FavoriteMomentRequest, GetMomentInteractionStateRequest, LikeMomentRequest,
            MomentLikesRequest, MomentOpenTheDoorRequest, MyFavoritesRequest, MyMomentsRequest,
            SearchMomentsByKeywordRequest, SearchMomentsByLocationRequest,
            SearchMomentsByTagRequest, UnfavoriteMomentRequest, UnlikeMomentRequest,
            UpdateMomentRequest,
        },
        response::{
            ApiResponse, CheckFavoritedData, CheckLikedData, MomentInteractionStateData,
            MomentCard, MomentItem, MomentLikeItem, PaginatedData, UserMomentListItem,
        },
    },
    error::Result,
    middleware::auth::{AuthenticatedUser, OptionalAuthenticatedUser},
    state::AppState,
};

// ==================== Token 可选 ====================

/// 按关键字搜索生活瞬间（全局）
pub async fn search_by_keyword(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<SearchMomentsByKeywordRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentItem>>>> {
    let response = state
        .moment_service
        .search_by_keyword(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 按标签搜索生活瞬间（全局）
pub async fn search_by_tag(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<SearchMomentsByTagRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentItem>>>> {
    let response = state
        .moment_service
        .search_by_tag(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 按位置搜索生活瞬间（范围过滤）
pub async fn search_by_location(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<SearchMomentsByLocationRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentItem>>>> {
    let response = state
        .moment_service
        .search_by_location(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 获取当前用户生活瞬间历史（/me/ API）
pub async fn get_my_moments(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<UserMomentListItem>>>> {
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
) -> Result<Json<ApiResponse<MomentCard>>> {
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
    Json(req): Json<UpdateMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .moment_service
        .update_moment(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
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

pub async fn check_liked(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<CheckLikedRequest>,
) -> Result<Json<ApiResponse<CheckLikedData>>> {
    let is_liked = state.moment_service.check_liked(user.user_id, &req).await?;
    Ok(Json(ApiResponse::success(CheckLikedData { is_liked })))
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

pub async fn check_favorited(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<CheckFavoritedRequest>,
) -> Result<Json<ApiResponse<CheckFavoritedData>>> {
    let is_favorited = state
        .moment_service
        .check_favorited(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(CheckFavoritedData {
        is_favorited,
    })))
}

/// 获取瞬间互动状态（点赞+收藏）
pub async fn get_interaction_state(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<GetMomentInteractionStateRequest>,
) -> Result<Json<ApiResponse<MomentInteractionStateData>>> {
    let data = state
        .moment_service
        .get_interaction_state(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(data)))
}

pub async fn get_my_favorites(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<MyFavoritesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentCard>>>> {
    let response = state
        .moment_service
        .get_favorites_list(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}
