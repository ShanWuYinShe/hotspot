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

use crate::{
    ports::CommentRepository,
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

    /// POST /api/v1/protected/comments/add - 添加评论
    pub async fn add_comment(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<AddCommentRequest>,
    ) -> Result<Json<ApiResponse<AddCommentResponse>>, StatusCode> {
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

        // TODO: 实现添加评论逻辑
        // 1. 验证瞬间存在
        // 2. 创建评论记录
        // 3. 更新瞬间评论数
        // 4. 发送通知给瞬间作者

        let response = AddCommentResponse {
            comment_id: "temp_comment_id".to_string(), // TODO: 使用实际的评论ID
            content: request.content.trim().to_string(),
            created_at: chrono::Utc::now().to_rfc3339(),
            message: "评论添加成功".to_string(),
        };

        Ok(Json(ApiResponse::success(response)))
    }

    /// POST /api/v1/protected/comments/reply - 回复评论
    pub async fn reply_comment(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<ReplyCommentRequest>,
    ) -> Result<Json<ApiResponse<ReplyCommentResponse>>, StatusCode> {
        // 验证请求参数
        if request.content.trim().is_empty() {
            return Ok(Json(ApiResponse::error(1000, "回复内容不能为空")));
        }

        if request.content.len() > 1000 {
            return Ok(Json(ApiResponse::error(1000, "回复内容不能超过1000个字符")));
        }

        // 验证comment_id格式
        if Uuid::parse_str(&request.comment_id).is_err() {
            return Ok(Json(ApiResponse::error(1000, "无效的评论ID格式")));
        }

        // TODO: 实现回复评论逻辑
        // 1. 验证父评论存在
        // 2. 创建回复记录
        // 3. 更新父评论回复数
        // 4. 发送通知给父评论作者

        let response = ReplyCommentResponse {
            comment_id: "temp_reply_id".to_string(), // TODO: 使用实际的回复ID
            parent_comment_id: request.comment_id,
            content: request.content.trim().to_string(),
            created_at: chrono::Utc::now().to_rfc3339(),
            message: "回复添加成功".to_string(),
        };

        Ok(Json(ApiResponse::success(response)))
    }

    /// POST /api/v1/protected/comments/delete - 删除评论
    pub async fn delete_comment(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<DeleteCommentRequest>,
    ) -> Result<Json<ApiResponse<DeleteCommentResponse>>, StatusCode> {
        // 验证comment_id格式
        if Uuid::parse_str(&request.comment_id).is_err() {
            return Ok(Json(ApiResponse::error(1000, "无效的评论ID格式")));
        }

        // TODO: 实现删除评论逻辑
        // 1. 验证评论存在且用户有权限删除
        // 2. 软删除评论记录
        // 3. 更新相关统计数据
        // 4. 级联处理回复（如果有）

        let response = DeleteCommentResponse {
            success: true,
            message: "评论删除成功".to_string(),
        };

        Ok(Json(ApiResponse::success(response)))
    }

    /// POST /api/v1/public/comments/list - 获取评论列表
    pub async fn get_comments(
        State(controller): State<Self>,
        Json(request): Json<GetCommentsRequest>,
    ) -> Result<Json<ApiResponse<PaginatedResponse<CommentResponse>>>, StatusCode> {
        // 验证moment_id格式
        if Uuid::parse_str(&request.moment_id).is_err() {
            return Ok(Json(ApiResponse::error(1000, "无效的生活瞬间ID格式")));
        }

        // TODO: 实现获取评论列表逻辑
        // 1. 查询瞬间的评论
        // 2. 分页处理
        // 3. 关联用户信息
        // 4. 返回评论列表

        let comments = vec![
            CommentResponse {
                comment_id: "example_comment_id".to_string(),
                moment_id: request.moment_id.clone(),
                user_id: "example_user_id".to_string(),
                nickname: "示例用户".to_string(),
                user_avatar: Some("https://example.com/avatar.jpg".to_string()),
                content: "这是一条示例评论".to_string(),
                parent_comment_id: None,
                created_at: chrono::Utc::now().to_rfc3339(),
                replies_count: 2,
            }
        ];

        let paginated_response = PaginatedResponse {
            items: comments,
            has_more: false,
            next_cursor: None,
        };

        Ok(Json(ApiResponse::success(paginated_response)))
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
