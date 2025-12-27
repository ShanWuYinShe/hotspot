use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            CommentLikesRequest, CommentRepliesRequest, CreateCommentRequest, DeleteCommentRequest,
            LikeCommentRequest, MomentCommentsRequest, UnlikeCommentRequest,
        },
        response::{
            ApiResponse, CommentLikeItem, CommentListItem, CreateCommentData, PaginatedData,
        },
    },
    error::Result,
    state::AppState,
};

// ==================== Public APIs ====================

pub async fn get_comments(
    State(state): State<AppState>,
    Json(req): Json<MomentCommentsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<CommentListItem>>>> {
    let response = state.comment_service.list_comments(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn get_replies(
    State(state): State<AppState>,
    Json(req): Json<CommentRepliesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<CommentListItem>>>> {
    let response = state.comment_service.list_replies(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn get_likes(
    State(state): State<AppState>,
    Json(req): Json<CommentLikesRequest>,
) -> Result<Json<ApiResponse<PaginatedData<CommentLikeItem>>>> {
    let response = state.comment_service.get_likes_list(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

// ==================== Protected APIs ====================

pub async fn create(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CreateCommentRequest>,
) -> Result<Json<ApiResponse<CreateCommentData>>> {
    let response = state.comment_service.add_comment(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn delete(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<DeleteCommentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.comment_service.delete_comment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn like(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<LikeCommentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.comment_service.like_comment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn unlike(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<UnlikeCommentRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.comment_service.unlike_comment(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
