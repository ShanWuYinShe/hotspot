use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            CreateMomentRequest, DeleteMomentRequest,
            EditMomentRequest, FavoriteMomentRequest,
            GeoPoint, LikeMomentRequest, MomentLikesRequest, MomentOpenTheDoorRequest,
            MyFavoritesRequest, MyMomentsRequest, SearchMomentsByKeywordRequest,
            SearchMomentsByLocationRequest, SearchMomentsByTagRequest, SearchMomentsRequest,
            UnfavoriteMomentRequest, UnlikeMomentRequest,
        },
        response::{
            ApiResponse, MomentLikeItem, MomentSearchItem, PaginatedData, UserMomentListItem,
        },
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
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let reference_point = req.reference_point.unwrap_or(GeoPoint {
        latitude: 0.0,
        longitude: 0.0,
    });

    if let Some(radius) = req.radius {
        let location_req = SearchMomentsByLocationRequest {
            kw: req.keyword,
            reference_point,
            radius,
            limit: req.limit,
            cursor: req.cursor,
        };
        let response = state
            .moment_service
            .search_by_location(user.user_id, &location_req)
            .await?;
        Ok(Json(ApiResponse::success(response)))
    } else if let Some(tag) = req.tag {
        let tag_req = SearchMomentsByTagRequest {
            tag,
            reference_point,
            limit: req.limit,
            cursor: req.cursor,
        };
        let response = state
            .moment_service
            .search_by_tag(user.user_id, &tag_req)
            .await?;
        Ok(Json(ApiResponse::success(response)))
    } else if let Some(keyword) = req.keyword {
        let kw_req = SearchMomentsByKeywordRequest {
            kw: keyword,
            reference_point,
            limit: req.limit,
            cursor: req.cursor,
        };
        let response = state
            .moment_service
            .search_by_keyword(user.user_id, &kw_req)
            .await?;
        Ok(Json(ApiResponse::success(response)))
    } else {
        Err(crate::error::AppError::BadRequest(
            "至少需要提供 keyword、tag、radius 中的一个搜索条件".to_string(),
        ))
    }
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
) -> Result<Json<ApiResponse<MomentSearchItem>>> {
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
) -> Result<Json<ApiResponse<UserMomentListItem>>> {
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
) -> Result<Json<ApiResponse<PaginatedData<MomentSearchItem>>>> {
    let response = state
        .moment_service
        .get_favorites_list(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}
