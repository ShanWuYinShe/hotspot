/// 评论互动控制器
/// 
/// 功能职责：
/// - 处理评论相关的HTTP请求
/// - 实现评论的创建、删除、回复、查询功能
/// - 管理评论的权限验证和数据转换
/// - 提供评论回复和层级管理功能
/// 
/// API端点（按需求文档）：
/// - POST /api/v1/protected/comments/add - 添加评论
/// - POST /api/v1/protected/comments/reply - 回复评论
/// - POST /api/v1/protected/comments/delete - 删除评论
/// - POST /api/v1/public/comments/list - 获取评论列表
use axum::{
    extract::{State, Extension},
    http::StatusCode,
    response::Json,
};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use uuid::Uuid;
use utoipa::ToSchema;
use tracing::error;

use crate::{
    presentation::http::models::{ApiResponse, PaginatedResponse},
    presentation::http::auth::CurrentUser,
};

// ==================== 请求/响应类型定义 ====================

/// 添加评论请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct AddCommentRequest {
    /// 瞬间ID
    pub moment_id: String,
    /// 评论内容
    pub content: String,
}

/// 添加评论响应
#[derive(Debug, Serialize, ToSchema)]
pub struct AddCommentResponse {
    /// 新创建的评论ID
    pub comment_id: String,
    /// 评论内容
    pub content: String,
    /// 创建时间（ISO 8601）
    pub created_at: String,
    /// 响应消息
    pub message: String,
}

/// 回复评论请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct ReplyCommentRequest {
    /// 父评论ID
    pub comment_id: String,
    /// 回复内容
    pub content: String,
}

/// 回复评论响应
#[derive(Debug, Serialize, ToSchema)]
pub struct ReplyCommentResponse {
    /// 新创建的回复ID
    pub comment_id: String,
    /// 父评论ID
    pub parent_comment_id: String,
    /// 回复内容
    pub content: String,
    /// 创建时间（ISO 8601）
    pub created_at: String,
    /// 响应消息
    pub message: String,
}

/// 删除评论请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct DeleteCommentRequest {
    /// 评论ID
    pub comment_id: String,
}

/// 删除评论响应
#[derive(Debug, Serialize, ToSchema)]
pub struct DeleteCommentResponse {
    /// 是否删除成功
    pub success: bool,
    /// 响应消息
    pub message: String,
}

/// 获取评论列表请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct GetCommentsRequest {
    /// 瞬间ID
    pub moment_id: String,
    /// 返回数量限制，默认20
    pub limit: Option<u32>,
    /// 分页游标
    pub cursor: Option<String>,
}

/// 评论通用响应类型（用于查询类API）
#[derive(Debug, Serialize, ToSchema)]
pub struct CommentResponse {
    /// 评论ID
    pub comment_id: String,
    /// 瞬间ID
    pub moment_id: String,
    /// 评论者ID
    pub user_id: String,
    /// 评论者昵称
    pub nickname: String,
    /// 评论者头像URL
    pub user_avatar: Option<String>,
    /// 评论内容
    pub content: String,
    /// 父评论ID（回复时）
    pub parent_comment_id: Option<String>,
    /// 创建时间（ISO 8601）
    pub created_at: String,
    /// 回复数量
    pub replies_count: u32,
}

/// 评论控制器
pub struct CommentController {
    comment_service: Arc<dyn crate::application::services::CommentApplicationService>,
}

impl CommentController {
    /// 创建新的评论控制器
    pub fn new(comment_service: Arc<dyn crate::application::services::CommentApplicationService>) -> Self {
        Self {
            comment_service,
        }
    }

    /// POST /api/v1/protected/comments/add - 添加评论
    pub async fn add_comment(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<AddCommentRequest>,
    ) -> Result<Json<ApiResponse<AddCommentResponse>>, StatusCode> {
        // 验证请求参数
        if request.content.trim().is_empty() {
            return Ok(Json(ApiResponse::error_for_type(1000, "评论内容不能为空".to_string())));
        }

        if request.content.len() > 1000 {
            return Ok(Json(ApiResponse::error_for_type(1000, "评论内容不能超过1000个字符".to_string())));
        }

        // 验证moment_id格式
        if Uuid::parse_str(&request.moment_id).is_err() {
            return Ok(Json(ApiResponse::error_for_type(1000, "无效的生活瞬间ID格式".to_string())));
        }

        // 调用应用服务创建评论
        match controller.comment_service.create_comment(
            &request.moment_id,
            current_user.user_id,
            &request.content,
            None,
        ).await {
            Ok(comment) => {
                let response = AddCommentResponse {
                    comment_id: comment.comment_id.to_string(),
                    content: comment.content,
                    created_at: comment.created_at.to_rfc3339(),
                    message: "评论添加成功".to_string(),
                };
                Ok(Json(ApiResponse::success(response)))
            },
            Err(error) => {
                error!("添加评论失败: {:?}", error);
                Ok(Json(ApiResponse::error_for_type(5000, "添加评论失败".to_string())))
            }
        }
    }

    /// POST /api/v1/protected/comments/reply - 回复评论
    pub async fn reply_comment(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<ReplyCommentRequest>,
    ) -> Result<Json<ApiResponse<ReplyCommentResponse>>, StatusCode> {
        // 验证请求参数
        if request.content.trim().is_empty() {
            return Ok(Json(ApiResponse::error_for_type(1000, "回复内容不能为空".to_string())));
        }

        if request.content.len() > 1000 {
            return Ok(Json(ApiResponse::error_for_type(1000, "回复内容不能超过1000个字符".to_string())));
        }

        // 验证comment_id格式
        if Uuid::parse_str(&request.comment_id).is_err() {
            return Ok(Json(ApiResponse::error_for_type(1000, "无效的评论ID格式".to_string())));
        }

        // 调用应用服务创建回复（需要先获取父评论信息获得moment_id）
        // 这里简化处理，实际应用中应该有获取父评论的方法
        match controller.comment_service.create_comment(
            "", // TODO: 需要从父评论获取moment_id
            current_user.user_id,
            &request.content,
            Some(&request.comment_id),
        ).await {
            Ok(comment) => {
                let response = ReplyCommentResponse {
                    comment_id: comment.comment_id.to_string(),
                    parent_comment_id: request.comment_id,
                    content: comment.content,
                    created_at: comment.created_at.to_rfc3339(),
                    message: "回复添加成功".to_string(),
                };
                Ok(Json(ApiResponse::success(response)))
            },
            Err(error) => {
                error!("回复评论失败: {:?}", error);
                Ok(Json(ApiResponse::error_for_type(5000, "回复评论失败".to_string())))
            }
        }
    }

    /// POST /api/v1/protected/comments/delete - 删除评论
    pub async fn delete_comment(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<DeleteCommentRequest>,
    ) -> Result<Json<ApiResponse<DeleteCommentResponse>>, StatusCode> {
        // 验证comment_id格式
        if Uuid::parse_str(&request.comment_id).is_err() {
            return Ok(Json(ApiResponse::error_for_type(1000, "无效的评论ID格式".to_string())));
        }

        // 调用应用服务删除评论
        match controller.comment_service.delete_comment(&request.comment_id, current_user.user_id).await {
            Ok(_) => {
                let response = DeleteCommentResponse {
                    success: true,
                    message: "评论删除成功".to_string(),
                };
                Ok(Json(ApiResponse::success(response)))
            },
            Err(error) => {
                error!("删除评论失败: {:?}", error);
                Ok(Json(ApiResponse::error_for_type(5000, "删除评论失败".to_string())))
            }
        }
    }

    /// POST /api/v1/public/comments/list - 获取评论列表
    pub async fn get_comments(
        State(controller): State<Self>,
        Json(request): Json<GetCommentsRequest>,
    ) -> Result<Json<ApiResponse<PaginatedResponse<CommentResponse>>>, StatusCode> {
        // 验证moment_id格式
        if Uuid::parse_str(&request.moment_id).is_err() {
            return Ok(Json(ApiResponse::error_for_type(1000, "无效的生活瞬间ID格式".to_string())));
        }

        // 调用应用服务获取评论列表
        let cursor = request.cursor.unwrap_or("0".to_string()).parse::<i64>().unwrap_or(0);
        let limit = request.limit.unwrap_or(20).min(100) as i32;
        
        match controller.comment_service.get_moment_comments(
            &request.moment_id,
            cursor,
            limit + 1,
        ).await {
            Ok(mut comments) => {
                let has_more = comments.len() > limit as usize;
                if has_more {
                    comments.pop();
                }
                
                let next_cursor = if has_more && !comments.is_empty() {
                    Some(comments.last().unwrap().comment_id.to_string())
                } else {
                    None
                };
                
                let comment_responses: Vec<CommentResponse> = comments
                    .into_iter()
                    .map(|comment| CommentResponse {
                        comment_id: comment.comment_id.to_string(),
                        moment_id: comment.moment_id.to_string(),
                        user_id: comment.user_id.to_string(),
                        nickname: "用户".to_string(), // TODO: 从用户服务获取昵称
                        user_avatar: None,
                        content: comment.content,
                        parent_comment_id: comment.parent_comment_id.map(|id| id.to_string()),
                        created_at: comment.created_at.to_rfc3339(),
                        replies_count: 0, // TODO: 计算回复数量
                    })
                    .collect();
                
                let paginated_response = PaginatedResponse {
                    items: comment_responses,
                    has_more,
                    next_cursor,
                };

                Ok(Json(ApiResponse::success(paginated_response)))
            },
            Err(error) => {
                error!("获取评论列表失败: {:?}", error);
                Ok(Json(ApiResponse::error_for_type(5000, "获取评论列表失败".to_string())))
            }
        }
    }
}

impl Clone for CommentController {
    fn clone(&self) -> Self {
        Self {
            comment_service: Arc::clone(&self.comment_service),
        }
    }
}
