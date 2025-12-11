use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            CreateCommentRequest, DeleteCommentRequest,
            LikeCommentRequest, UnlikeCommentRequest,
            GetCommentsRequest, GetRepliesRequest, GetCommentLikesListRequest,
        },
        response::{
            ApiResponse, PaginatedData, CommentListItem, CreateCommentData, CommentLikeItem,
        },
    },
    error::Result,
    services::comment_service::CommentService,
    state::AppState,
};


// ==================== Public APIs ====================

pub async fn get_list(
    State(state): State<AppState>,
    Json(req): Json<GetCommentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<CommentListItem>>>> {
    let service = CommentService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.list_comments(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn get_replies(
    State(state): State<AppState>,
    Json(req): Json<GetRepliesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<CommentListItem>>>> {
    let service = CommentService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.list_replies(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn get_likes_list(
    State(state): State<AppState>,
    Json(req): Json<GetCommentLikesListRequest>,
) -> Result<Json<ApiResponse<PaginatedData<CommentLikeItem>>>> {
    let service = CommentService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.get_likes_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}


// ==================== Protected APIs ====================

pub async fn add(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CreateCommentRequest>,
) -> Result<Json<ApiResponse<CreateCommentData>>> {
    let service = CommentService::new(&state.db, &state.redis, state.event_bus.clone());
    let response = service.add_comment(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn remove(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteCommentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = CommentService::new(&state.db, &state.redis, state.event_bus.clone());
    service.delete_comment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn like_comment(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<LikeCommentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = CommentService::new(&state.db, &state.redis, state.event_bus.clone());
    service.like_comment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn unlike_comment(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnlikeCommentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = CommentService::new(&state.db, &state.redis, state.event_bus.clone());
    service.unlike_comment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


