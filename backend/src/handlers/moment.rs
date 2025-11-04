use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            CreateMomentRequest, UpdateMomentRequest, DeleteMomentRequest,
            SearchMomentByLocationRequest, SearchMomentByTagsRequest,
            ListMyMomentsRequest, LikeMomentRequest, UnlikeMomentRequest,
            CollectMomentRequest, UncollectMomentRequest, ListCollectionsRequest,
        },
        response::{ApiResponse, PaginatedResponse, MomentResponse, MomentSummaryResponse},
    },
    error::Result,
    middleware::auth::OptionalUserId,
    services::moment_service::MomentService,
    state::AppState,
};

/// POST /api/v1/protected/moment/create
pub async fn create(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<CreateMomentRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = MomentService::new(&state.db, &state.redis);
    let moment_id = service.create_moment(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "瞬间创建成功",
        "id": moment_id
    }))))
}

/// POST /api/v1/protected/moment/update
pub async fn update(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UpdateMomentRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = MomentService::new(&state.db, &state.redis);
    service.update_moment(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "瞬间更新成功"
    }))))
}

/// POST /api/v1/protected/moment/delete
pub async fn delete(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<DeleteMomentRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = MomentService::new(&state.db, &state.redis);
    service.delete_moment(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "瞬间删除成功"
    }))))
}

/// POST /api/v1/protected/moment/my_moments
pub async fn my_moments(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<ListMyMomentsRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<MomentResponse>>>> {
    
    let service = MomentService::new(&state.db, &state.redis);
    let response = service.list_my_moments(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/moment/search_by_location
pub async fn search_by_location(
    State(state): State<AppState>,
    OptionalUserId(current_user_id): OptionalUserId,
    Json(req): Json<SearchMomentByLocationRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<MomentResponse>>>> {
    let service = MomentService::new(&state.db, &state.redis);
    let response = service.search_by_location(&req, current_user_id).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/moment/search_by_tags
pub async fn search_by_tags(
    State(state): State<AppState>,
    OptionalUserId(current_user_id): OptionalUserId,
    Json(req): Json<SearchMomentByTagsRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<MomentResponse>>>> {
    let service = MomentService::new(&state.db, &state.redis);
    let response = service.search_by_tags(&req, current_user_id).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/moment/like
pub async fn like(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<LikeMomentRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = MomentService::new(&state.db, &state.redis);
    service.like_moment(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "点赞成功"
    }))))
}

/// POST /api/v1/protected/moment/unlike
pub async fn unlike(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UnlikeMomentRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = MomentService::new(&state.db, &state.redis);
    service.unlike_moment(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "已取消点赞"
    }))))
}

/// POST /api/v1/protected/moment/collect
pub async fn collect(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<CollectMomentRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = MomentService::new(&state.db, &state.redis);
    service.collect_moment(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "收藏成功"
    }))))
}

/// POST /api/v1/protected/moment/uncollect
pub async fn uncollect(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<UncollectMomentRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    
    let service = MomentService::new(&state.db, &state.redis);
    service.uncollect_moment(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "已取消收藏"
    }))))
}

/// POST /api/v1/protected/moment/collections
/// 获取收藏列表（使用摘要响应，性能优化）
pub async fn collections(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req_body): Json<ListCollectionsRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<MomentSummaryResponse>>>> {
    
    let service = MomentService::new(&state.db, &state.redis);
    let response = service.list_collections(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

