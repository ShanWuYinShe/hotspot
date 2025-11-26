use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            CreateCommentRequest, DeleteCommentRequest, LikeCommentRequest,
            ListCommentsRequest, ListRepliesRequest, GetCommentLikesRequest,
        },
        response::{
            ApiResponse, PaginatedData, CommentListItem, CreateCommentData, CommentLikeItem,
        },
    },
    error::Result,
    middleware::auth::OptionalUserId,
    services::comment_service::CommentService,
    state::AppState,
};

/// POST /api/v1/protected/comments/add
pub async fn add(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CreateCommentRequest>,
) -> Result<Json<ApiResponse<CreateCommentData>>> {
    let service = CommentService::new(&state.db, &state.redis);
    let response = service.add_comment(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/comments/remove
pub async fn remove(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteCommentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = CommentService::new(&state.db, &state.redis);
    service.delete_comment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/protected/comments/toggle_like
pub async fn toggle_like(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<LikeCommentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = CommentService::new(&state.db, &state.redis);
    service.toggle_like(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

/// POST /api/v1/public/comments/get_list
pub async fn get_list(
    State(state): State<AppState>,
    OptionalUserId(current_user_id): OptionalUserId,
    Json(req): Json<ListCommentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<CommentListItem>>>> {
    let service = CommentService::new(&state.db, &state.redis);
    let response = service.list_comments(&req, current_user_id).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/comments/get_replies
pub async fn get_replies(
    State(state): State<AppState>,
    OptionalUserId(current_user_id): OptionalUserId,
    Json(req): Json<ListRepliesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<CommentListItem>>>> {
    let service = CommentService::new(&state.db, &state.redis);
    let response = service.list_replies(&req, current_user_id).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/comments/get_likes
pub async fn get_likes(
    State(state): State<AppState>,
    Json(req): Json<GetCommentLikesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<CommentLikeItem>>>> {
    let service = CommentService::new(&state.db, &state.redis);
    let response = service.get_likes(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}
