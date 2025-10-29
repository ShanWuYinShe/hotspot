use axum::{extract::{Request, State}, Json};

use crate::{
    dto::{
        request::{
            AddCommentRequest, DeleteCommentRequest, LikeCommentRequest,
            UnlikeCommentRequest, ListCommentsRequest, ListRepliesRequest,
        },
        response::{ApiResponse, PaginatedResponse, CommentResponse},
    },
    error::Result,
    middleware::auth::extract_user_id,
    services::comment_service::CommentService,
    state::AppState,
};

/// POST /api/v1/protected/comment/add
pub async fn add(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<AddCommentRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = CommentService::new(&state.db, &state.redis);
    let comment_id = service.add_comment(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "评论添加成功",
        "id": comment_id
    }))))
}

/// POST /api/v1/protected/comment/delete
pub async fn delete(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<DeleteCommentRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = CommentService::new(&state.db, &state.redis);
    service.delete_comment(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "评论删除成功"
    }))))
}

/// POST /api/v1/protected/comment/like
pub async fn like(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<LikeCommentRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = CommentService::new(&state.db, &state.redis);
    service.like_comment(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "点赞成功"
    }))))
}

/// POST /api/v1/protected/comment/unlike
pub async fn unlike(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<UnlikeCommentRequest>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    let service = CommentService::new(&state.db, &state.redis);
    service.unlike_comment(user_id, &req_body).await?;
    
    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "已取消点赞"
    }))))
}

/// POST /api/v1/public/comment/list
pub async fn list(
    State(state): State<AppState>,
    request: Request,
    Json(req): Json<ListCommentsRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<CommentResponse>>>> {
    let current_user_id = extract_user_id(&request).ok();
    
    let service = CommentService::new(&state.db, &state.redis);
    let response = service.list_comments(&req, current_user_id).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/comment/replies
pub async fn replies(
    State(state): State<AppState>,
    request: Request,
    Json(req): Json<ListRepliesRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<CommentResponse>>>> {
    let current_user_id = extract_user_id(&request).ok();
    
    let service = CommentService::new(&state.db, &state.redis);
    let response = service.list_replies(&req, current_user_id).await?;
    
    Ok(Json(ApiResponse::success(response)))
}

