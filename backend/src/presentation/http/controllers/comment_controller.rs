/// 评论控制器
/// 
/// 功能职责：
/// - 处理评论相关的HTTP请求
/// - 实现评论的创建、删除、更新、查询功能
/// - 管理评论的权限验证和数据转换
/// - 提供评论回复和层级管理功能
/// 
/// API端点：
/// - POST /api/v1/protected/comments/create - 创建评论
/// - POST /api/v1/protected/comments/{comment_id}/delete - 删除评论
/// - POST /api/v1/protected/comments/{comment_id}/update - 更新评论
/// - POST /api/v1/public/comments/moment/{moment_id} - 获取生活瞬间评论
/// - POST /api/v1/public/comments/user/{user_id} - 获取用户评论历史
use axum::{
    extract::{Path, State, Extension},
    http::StatusCode,
    response::Json,
};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    ports::CommentRepository,
    utils::response::ApiResponse,
    presentation::http::models::common::{PaginatedResponse, PaginationMeta},
    utils::JsSafeI64,
    presentation::http::auth::CurrentUser,
};

/// 创建评论请求
#[derive(Debug, Deserialize)]
pub struct CreateCommentRequest {
    /// 生活瞬间ID
    pub moment_id: String,
    /// 评论内容
    pub content: String,
    /// 父评论ID（回复评论时使用）
    pub parent_comment_id: Option<String>,
}

/// 更新评论请求
#[derive(Debug, Deserialize)]
pub struct UpdateCommentRequest {
    /// 新的评论内容
    pub content: String,
}

/// 获取评论列表请求
#[derive(Debug, Deserialize)]
pub struct GetCommentsRequest {
    /// 游标（用于分页）
    pub cursor: Option<i64>,
    /// 每页数量
    pub limit: Option<i32>,
}

/// 评论响应
#[derive(Debug, Serialize)]
pub struct CommentResponse {
    /// 评论ID
    pub comment_id: String,
    /// 生活瞬间ID
    pub moment_id: String,
    /// 评论用户ID
    pub user_id: JsSafeI64,
    /// 评论内容
    pub content: String,
    /// 创建时间
    pub created_at: String,
    /// 更新时间
    pub updated_at: String,
}

/// 评论详情响应
#[derive(Debug, Serialize)]
pub struct CommentDetailResponse {
    /// 评论ID
    pub comment_id: String,
    /// 生活瞬间ID
    pub moment_id: String,
    /// 评论用户ID
    pub user_id: JsSafeI64,
    /// 评论用户昵称
    pub user_nickname: String,
    /// 评论用户头像
    pub user_avatar: Option<String>,
    /// 评论内容
    pub content: String,
    /// 创建时间
    pub created_at: String,
    /// 更新时间
    pub updated_at: String,
}

/// 评论控制器
pub struct CommentController<R>
where
    R: CommentRepository,
{
    comment_repository: Arc<R>,
}

impl<R> CommentController<R>
where
    R: CommentRepository,
{
    /// 创建新的评论控制器
    pub fn new(comment_repository: Arc<R>) -> Self {
        Self {
            comment_repository,
        }
    }

    /// 创建评论
    pub async fn create_comment(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<CreateCommentRequest>,
    ) -> Result<Json<ApiResponse<CommentResponse>>, StatusCode> {
        // 验证请求参数
        if request.content.trim().is_empty() {
            return Ok(Json(ApiResponse::error(1000, "评论内容不能为空")));
        }

        if request.content.len() > 1000 {
            return Ok(Json(ApiResponse::error(1000, "评论内容不能超过1000个字符")));
        }

        // 验证moment_id格式
        if Uuid::parse_str(&request.moment_id).is_err() {
            return Ok(Json(ApiResponse::error(1000, "无效的生活瞬间ID格式")));
        }

        // 如果有父评论ID，验证格式
        if let Some(ref parent_id) = request.parent_comment_id {
            if Uuid::parse_str(parent_id).is_err() {
                return Ok(Json(ApiResponse::error(1000, "无效的父评论ID格式")));
            }
        }

        // 创建评论
        match controller
            .comment_repository
            .create_comment(
                &request.moment_id,
                current_user.user_id,
                &request.content.trim(),
                request.parent_comment_id.as_deref(),
            )
            .await
        {
            Ok(comment) => {
                let response = CommentResponse {
                    comment_id: comment.comment_id.to_string(),
                    moment_id: comment.moment_id.to_string(),
                    user_id: JsSafeI64::new(comment.user_id),
                    content: comment.content,
                    created_at: comment.created_at.to_rfc3339(),
                    updated_at: comment.updated_at.to_rfc3339(),
                };
                Ok(Json(ApiResponse::success(response)))
            }
            Err(e) => {
                tracing::error!("创建评论失败: {:?}", e);
                Ok(Json(ApiResponse::error(5000, "创建评论失败")))
            }
        }
    }

    /// 删除评论
    pub async fn delete_comment(
        State(controller): State<Self>,
        Path(comment_id): Path<String>,
        Extension(current_user): Extension<CurrentUser>,
    ) -> Result<Json<ApiResponse<()>>, StatusCode> {
        // 验证comment_id格式
        if Uuid::parse_str(&comment_id).is_err() {
            return Ok(Json(ApiResponse::error(1000, "无效的评论ID格式")));
        }

        // 删除评论
        match controller
            .comment_repository
            .delete_comment(&comment_id, current_user.user_id)
            .await
        {
            Ok(_) => Ok(Json(ApiResponse::success(()))),
            Err(e) => {
                tracing::error!("删除评论失败: {:?}", e);
                match e.to_string().as_str() {
                    msg if msg.contains("not found") => {
                        Ok(Json(ApiResponse::error(1004, "评论不存在")))
                    }
                    msg if msg.contains("permission") => {
                        Ok(Json(ApiResponse::error(1003, "没有权限删除此评论")))
                    }
                    _ => Ok(Json(ApiResponse::error(5000, "删除评论失败"))),
                }
            }
        }
    }

    /// 更新评论
    pub async fn update_comment(
        State(controller): State<Self>,
        Path(comment_id): Path<String>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<UpdateCommentRequest>,
    ) -> Result<Json<ApiResponse<CommentResponse>>, StatusCode> {
        // 验证请求参数
        if request.content.trim().is_empty() {
            return Ok(Json(ApiResponse::error(1000, "评论内容不能为空")));
        }

        if request.content.len() > 1000 {
            return Ok(Json(ApiResponse::error(1000, "评论内容不能超过1000个字符")));
        }

        // 验证comment_id格式
        if Uuid::parse_str(&comment_id).is_err() {
            return Ok(Json(ApiResponse::error(1000, "无效的评论ID格式")));
        }

        // 更新评论
        match controller
            .comment_repository
            .update_comment(&comment_id, current_user.user_id, &request.content.trim())
            .await
        {
            Ok(comment) => {
                let response = CommentResponse {
                    comment_id: comment.comment_id.to_string(),
                    moment_id: comment.moment_id.to_string(),
                    user_id: JsSafeI64::new(comment.user_id),
                    content: comment.content,
                    created_at: comment.created_at.to_rfc3339(),
                    updated_at: comment.updated_at.to_rfc3339(),
                };
                Ok(Json(ApiResponse::success(response)))
            }
            Err(e) => {
                tracing::error!("更新评论失败: {:?}", e);
                match e.to_string().as_str() {
                    msg if msg.contains("not found") => {
                        Ok(Json(ApiResponse::error(1004, "评论不存在")))
                    }
                    msg if msg.contains("permission") => {
                        Ok(Json(ApiResponse::error(1003, "没有权限修改此评论")))
                    }
                    _ => Ok(Json(ApiResponse::error(5000, "更新评论失败"))),
                }
            }
        }
    }

    /// 获取生活瞬间的评论列表
    pub async fn get_moment_comments(
        State(controller): State<Self>,
        Path(moment_id): Path<String>,
        Json(request): Json<GetCommentsRequest>,
    ) -> Result<Json<ApiResponse<PaginatedResponse<CommentDetailResponse>>>, StatusCode> {
        // 验证moment_id格式
        if Uuid::parse_str(&moment_id).is_err() {
            return Ok(Json(ApiResponse::error(1000, "无效的生活瞬间ID格式")));
        }

        let cursor = request.cursor.unwrap_or(0);
        let limit = request.limit.unwrap_or(20).min(100).max(1);

        // 获取评论列表
        match controller
            .comment_repository
            .find_comments_by_moment(&moment_id, cursor, limit + 1)
            .await
        {
            Ok(comments) => {
                let has_more = comments.len() > limit as usize;
                let mut items = comments;
                if has_more {
                    items.pop(); // 移除多查询的一条记录
                }

                let next_cursor = items.last().map(|c| JsSafeI64::new(c.user_id)); // 使用user_id作为游标

                let response_items: Vec<CommentDetailResponse> = items
                    .into_iter()
                    .map(|comment| CommentDetailResponse {
                        comment_id: comment.comment_id.to_string(),
                        moment_id: comment.moment_id.to_string(),
                        user_id: JsSafeI64::new(comment.user_id),
                        user_nickname: comment.user_nickname,
                        user_avatar: comment.user_avatar,
                        content: comment.content,
                        created_at: comment.created_at.to_rfc3339(),
                        updated_at: comment.updated_at.to_rfc3339(),
                    })
                    .collect();

                let pagination = PaginationMeta {
                    has_more,
                    next_cursor,
                };

                let response = PaginatedResponse {
                    items: response_items,
                    pagination,
                };

                Ok(Json(ApiResponse::success(response)))
            }
            Err(e) => {
                tracing::error!("获取评论列表失败: {:?}", e);
                Ok(Json(ApiResponse::error(5000, "获取评论列表失败")))
            }
        }
    }

    /// 获取用户评论历史
    pub async fn get_user_comments(
        State(controller): State<Self>,
        Path(user_id): Path<i64>,
        Json(request): Json<GetCommentsRequest>,
    ) -> Result<Json<ApiResponse<PaginatedResponse<CommentDetailResponse>>>, StatusCode> {
        let cursor = request.cursor.unwrap_or(0);
        let limit = request.limit.unwrap_or(20).min(100).max(1);

        // 获取用户评论历史
        match controller
            .comment_repository
            .find_comments_by_user(user_id, cursor, limit + 1)
            .await
        {
            Ok(comments) => {
                let has_more = comments.len() > limit as usize;
                let mut items = comments;
                if has_more {
                    items.pop(); // 移除多查询的一条记录
                }

                let next_cursor = items.last().map(|c| JsSafeI64::new(c.user_id));

                let response_items: Vec<CommentDetailResponse> = items
                    .into_iter()
                    .map(|comment| CommentDetailResponse {
                        comment_id: comment.comment_id.to_string(),
                        moment_id: comment.moment_id.to_string(),
                        user_id: JsSafeI64::new(comment.user_id),
                        user_nickname: comment.user_nickname,
                        user_avatar: comment.user_avatar,
                        content: comment.content,
                        created_at: comment.created_at.to_rfc3339(),
                        updated_at: comment.updated_at.to_rfc3339(),
                    })
                    .collect();

                let pagination = PaginationMeta {
                    has_more,
                    next_cursor,
                };

                let response = PaginatedResponse {
                    items: response_items,
                    pagination,
                };

                Ok(Json(ApiResponse::success(response)))
            }
            Err(e) => {
                tracing::error!("获取用户评论历史失败: {:?}", e);
                Ok(Json(ApiResponse::error(5000, "获取用户评论历史失败")))
            }
        }
    }
}

impl<R> Clone for CommentController<R>
where
    R: CommentRepository,
{
    fn clone(&self) -> Self {
        Self {
            comment_repository: Arc::clone(&self.comment_repository),
        }
    }
}
