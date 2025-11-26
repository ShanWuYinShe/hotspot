use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            CreateMomentRequest, EditMomentRequest, DeleteMomentRequest,
            SearchMomentsByLocationRequest, SearchMomentsByTagsRequest,
            ListMomentsRequest, LikeMomentRequest, UpdateFavoriteMomentRequest, 
            ListFavoritesRequest, OpenTheDoorRequest, GetMomentLikesRequest,
        },
        response::{
            ApiResponse, PaginatedData, MomentInfo, NearbyMomentItem, 
            CreateMomentData, MomentDetailData, MomentLikeItem,
        },
    },
    error::Result,
    middleware::auth::OptionalUserId,
    services::moment_service::MomentService,
    state::AppState,
};

/// POST /api/v1/protected/moments/create
pub async fn create(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CreateMomentRequest>,
) -> Result<Json<ApiResponse<CreateMomentData>>> {
    let service = MomentService::new(&state.db, &state.redis);
    let moment_id = service.create_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success(CreateMomentData { id: moment_id })))
}

/// POST /api/v1/protected/moments/update
pub async fn update(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<EditMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = MomentService::new(&state.db, &state.redis);
    service.update_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/moments/delete
pub async fn delete(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = MomentService::new(&state.db, &state.redis);
    service.delete_moment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/public/moments/get_list
pub async fn get_list(
    State(state): State<AppState>,
    Json(req): Json<ListMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentInfo>>>> {
    let service = MomentService::new(&state.db, &state.redis);
    let response = service.list_moments(None, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/moments/search_by_location
pub async fn search_by_location(
    State(state): State<AppState>,
    Json(req): Json<SearchMomentsByLocationRequest>,
) -> Result<Json<ApiResponse<PaginatedData<NearbyMomentItem>>>> {
    let service = MomentService::new(&state.db, &state.redis);
    let response = service.search_by_location(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/moments/search_by_tags
pub async fn search_by_tags(
    State(state): State<AppState>,
    Json(req): Json<SearchMomentsByTagsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentInfo>>>> {
    let service = MomentService::new(&state.db, &state.redis);
    let response = service.search_by_tags(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/moments/open_the_door
pub async fn open_the_door(
    State(state): State<AppState>,
    OptionalUserId(current_user_id): OptionalUserId,
    Json(req): Json<OpenTheDoorRequest>,
) -> Result<Json<ApiResponse<MomentDetailData>>> {
    let service = MomentService::new(&state.db, &state.redis);
    let response = service.get_detail(&req, current_user_id).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/moments/toggle_like
pub async fn toggle_like(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<LikeMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = MomentService::new(&state.db, &state.redis);
    service.toggle_like(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/moments/toggle_favorite
pub async fn toggle_favorite(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UpdateFavoriteMomentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = MomentService::new(&state.db, &state.redis);
    service.toggle_favorite(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/moments/get_favorites
pub async fn get_favorites(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<ListFavoritesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentInfo>>>> {
    let service = MomentService::new(&state.db, &state.redis);
    let response = service.list_favorites(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/moments/get_likes
pub async fn get_likes(
    State(state): State<AppState>,
    Json(req): Json<GetMomentLikesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<MomentLikeItem>>>> {
    let service = MomentService::new(&state.db, &state.redis);
    let response = service.get_moment_likes(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}
