/// 好友关系管理控制器
/// 
/// 功能职责：
/// - 处理好友申请、接受、拒绝等操作
/// - 管理用户关注、取消关注功能
/// - 处理用户拉黑、取消拉黑操作
/// - 提供好友列表、拉黑列表查询
/// 
/// API端点：
/// - POST /api/v1/protected/user/add_friend - 发送好友申请
/// - POST /api/v1/protected/user/accept_friend - 接受好友申请
/// - POST /api/v1/protected/user/reject_friend - 拒绝好友申请
/// - POST /api/v1/protected/user/follow_user - 关注用户
/// - POST /api/v1/protected/user/unfollow_user - 取消关注
/// - POST /api/v1/protected/user/block_user - 拉黑用户
/// - POST /api/v1/protected/user/friends_list - 获取好友列表
/// - POST /api/v1/protected/user/blocked_list - 获取拉黑列表

use axum::{extract::{State, Extension}, http::StatusCode, response::IntoResponse, Json};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tracing::debug;
use utoipa::ToSchema;

use crate::application::services::friend_application_service::FriendApplicationService;
use crate::presentation::http::models::ApiResponse;
use crate::presentation::http::auth::CurrentUser;

/// 好友关系管理控制器
#[derive(Clone)]
pub struct FriendController {
    friend_service: Arc<dyn FriendApplicationService>,
}

impl FriendController {
    pub fn new(friend_service: Arc<dyn FriendApplicationService>) -> Self {
        Self { friend_service }
    }
}

// ==================== 请求/响应类型定义 ====================

/// 发送好友申请请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct AddFriendRequest {
    /// 目标用户ID
    pub to_user_id: String,
    /// 申请消息（可选）
    pub message: Option<String>,
}

/// 发送好友申请响应
#[derive(Debug, Serialize, ToSchema)]
pub struct AddFriendResponse {
    /// 是否发送成功
    pub success: bool,
    /// 响应消息
    pub message: String,
    /// 关系记录ID
    pub relationship_id: String,
}

/// 接受好友申请请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct AcceptFriendRequest {
    /// 关系记录ID
    pub relationship_id: String,
}

/// 接受好友申请响应
#[derive(Debug, Serialize, ToSchema)]
pub struct AcceptFriendResponse {
    /// 是否接受成功
    pub success: bool,
    /// 响应消息
    pub message: String,
}

/// 拒绝好友申请请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct RejectFriendRequest {
    /// 关系记录ID
    pub relationship_id: String,
}

/// 拒绝好友申请响应
#[derive(Debug, Serialize, ToSchema)]
pub struct RejectFriendResponse {
    /// 是否拒绝成功
    pub success: bool,
    /// 响应消息
    pub message: String,
}

/// 关注用户请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct FollowRequest {
    /// 目标用户ID
    pub to_user_id: String,
}

/// 关注用户响应
#[derive(Debug, Serialize, ToSchema)]
pub struct FollowResponse {
    /// 是否关注成功
    pub success: bool,
    /// 响应消息
    pub message: String,
    /// 关系记录ID
    pub relationship_id: String,
}

/// 取消关注请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct UnfollowRequest {
    /// 目标用户ID
    pub to_user_id: String,
}

/// 取消关注响应
#[derive(Debug, Serialize, ToSchema)]
pub struct UnfollowResponse {
    /// 是否取消成功
    pub success: bool,
    /// 响应消息
    pub message: String,
}

/// 拉黑用户请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct BlockRequest {
    /// 目标用户ID
    pub to_user_id: String,
    /// 拉黑原因（可选）
    pub reason: Option<String>,
}

/// 拉黑用户响应
#[derive(Debug, Serialize, ToSchema)]
pub struct BlockResponse {
    /// 是否拉黑成功
    pub success: bool,
    /// 响应消息
    pub message: String,
    /// 关系记录ID
    pub relationship_id: String,
}

/// 获取好友列表请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct GetFriendsListRequest {
    /// 返回数量限制，默认20
    pub limit: Option<u32>,
    /// 分页游标
    pub cursor: Option<String>,
    /// 状态过滤：'pending'|'accepted'|'all'，默认'accepted'
    pub status: Option<String>,
}

/// 获取好友列表响应
#[derive(Debug, Serialize, ToSchema)]
pub struct GetFriendsListResponse {
    /// 好友用户ID
    pub user_id: String,
    /// 好友昵称
    pub nickname: String,
    /// 好友头像URL
    pub avatar_url: Option<String>,
    /// 好友个人简介
    pub bio: Option<String>,
    /// 关系类型：'friend'|'pending_sent'|'pending_received'
    pub relationship_type: String,
    /// 关系记录ID
    pub relationship_id: String,
    /// 建立关系时间（ISO 8601）
    pub created_at: String,
    /// 好友最后活跃时间（ISO 8601）
    pub last_active_at: Option<String>,
    /// 是否在线（可选）
    pub is_online: Option<bool>,
}

/// 获取拉黑列表请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct GetBlockedListRequest {
    /// 返回数量限制，默认20
    pub limit: Option<u32>,
    /// 分页游标
    pub cursor: Option<String>,
}

/// 获取拉黑列表响应
#[derive(Debug, Serialize, ToSchema)]
pub struct GetBlockedListResponse {
    /// 被拉黑用户ID
    pub user_id: String,
    /// 被拉黑用户昵称
    pub nickname: String,
    /// 被拉黑用户头像URL
    pub avatar_url: Option<String>,
    /// 关系记录ID
    pub relationship_id: String,
    /// 拉黑时间（ISO 8601）
    pub blocked_at: String,
    /// 拉黑原因
    pub reason: Option<String>,
}

// ==================== 控制器实现 ====================

impl FriendController {
    /// POST /api/v1/protected/user/add_friend - 发送好友申请
    pub async fn add_friend(
        State(controller): State<FriendController>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<AddFriendRequest>,
    ) -> axum::response::Response {
        debug!("用户 {} 发送好友申请给 {}", user.user_id, request.to_user_id);

        // 解析目标用户ID
        let target_id = match request.to_user_id.parse::<i64>() {
            Ok(id) => id,
            Err(_) => {
                let error_response = ApiResponse {
                    code: 400,
                    msg: "无效的用户ID格式".to_string(),
                    data: None::<()>,
                };
                return (StatusCode::OK, Json(error_response)).into_response();
            }
        };

        // 调用好友应用服务发送好友申请
        match controller.friend_service.send_friend_request(user.user_id, target_id, request.message).await {
            Ok(friend_request) => {
                let response = AddFriendResponse {
                    success: true,
                    message: "好友申请发送成功".to_string(),
                    relationship_id: friend_request.request_id,
                };

                let api_response = ApiResponse {
                    code: 0,
                    msg: "好友申请发送成功".to_string(),
                    data: Some(response),
                };

                (StatusCode::OK, Json(api_response)).into_response()
            }
            Err(err) => {
                let (status_code, error_msg) = match err {
                    crate::domain::DomainError::ValidationError { reason, .. } => (StatusCode::BAD_REQUEST, reason),
                    crate::domain::DomainError::NotFound { resource } => (StatusCode::NOT_FOUND, format!("{}不存在", resource)),
                    _ => (StatusCode::INTERNAL_SERVER_ERROR, "服务器内部错误".to_string()),
                };

                let error_response = ApiResponse {
                    code: status_code.as_u16() as i32,
                    msg: error_msg,
                    data: None::<()>,
                };

                // 统一HTTP 200策略
                (StatusCode::OK, Json(error_response)).into_response()
            }
        }
    }

    /// POST /api/v1/protected/user/accept_friend - 接受好友申请
    pub async fn accept_friend(
        State(controller): State<FriendController>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<AcceptFriendRequest>,
    ) -> axum::response::Response {
        debug!("用户 {} 接受好友申请 {}", user.user_id, request.relationship_id);

        // 调用好友应用服务接受好友申请
        match controller.friend_service.accept_friend_request(request.relationship_id, user.user_id).await {
            Ok(_friendship) => {
                let response = AcceptFriendResponse {
                    success: true,
                    message: "好友申请已接受".to_string(),
                };

                let api_response = ApiResponse {
                    code: 0,
                    msg: "好友申请已接受".to_string(),
                    data: Some(response),
                };

                (StatusCode::OK, Json(api_response)).into_response()
            }
            Err(err) => {
                let (status_code, error_msg) = match err {
                    crate::domain::DomainError::ValidationError { reason, .. } => (StatusCode::BAD_REQUEST, reason),
                    crate::domain::DomainError::NotFound { resource } => (StatusCode::NOT_FOUND, format!("{}不存在", resource)),
                    _ => (StatusCode::INTERNAL_SERVER_ERROR, "服务器内部错误".to_string()),
                };

                let error_response = ApiResponse {
                    code: status_code.as_u16() as i32,
                    msg: error_msg,
                    data: None::<()>,
                };

                (StatusCode::OK, Json(error_response)).into_response()
            }
        }
    }

    /// POST /api/v1/protected/user/reject_friend - 拒绝好友申请
    pub async fn reject_friend(
        State(controller): State<FriendController>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<RejectFriendRequest>,
    ) -> axum::response::Response {
        debug!("用户 {} 拒绝好友申请 {}", user.user_id, request.relationship_id);
        match controller.friend_service.reject_friend_request(request.relationship_id.clone(), user.user_id).await {
            Ok(_) => {
                let response = RejectFriendResponse { success: true, message: "好友申请已拒绝".to_string() };
                let api_response = ApiResponse { code: 0, msg: "好友申请已拒绝".to_string(), data: Some(response) };
                (StatusCode::OK, Json(api_response)).into_response()
            }
            Err(err) => crate::presentation::http::controllers::map_domain_error_to_http_response(err),
        }
    }

    /// POST /api/v1/protected/user/follow_user - 关注用户
    pub async fn follow_user(
        State(controller): State<FriendController>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<FollowRequest>,
    ) -> axum::response::Response {
        debug!("用户 {} 关注用户 {}", user.user_id, request.to_user_id);
        let target_id = request.to_user_id.parse::<i64>().unwrap_or(0);
        match controller.friend_service.follow_user(user.user_id, target_id).await {
            Ok(_) => {
                // 从仓储层返回关系ID更合适，但应用服务目前返回 ()，先返回空ID
                let response = FollowResponse { success: true, message: "关注成功".to_string(), relationship_id: String::new() };
                let api_response = ApiResponse { code: 0, msg: "关注成功".to_string(), data: Some(response) };
                (StatusCode::OK, Json(api_response)).into_response()
            }
            Err(err) => crate::presentation::http::controllers::map_domain_error_to_http_response(err),
        }
    }

    /// POST /api/v1/protected/user/unfollow_user - 取消关注
    pub async fn unfollow_user(
        State(controller): State<FriendController>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<UnfollowRequest>,
    ) -> axum::response::Response {
        debug!("用户 {} 取消关注用户 {}", user.user_id, request.to_user_id);
        let target_id = request.to_user_id.parse::<i64>().unwrap_or(0);
        match controller.friend_service.unfollow_user(user.user_id, target_id).await {
            Ok(_) => {
                let response = UnfollowResponse { success: true, message: "取消关注成功".to_string() };
                let api_response = ApiResponse { code: 0, msg: "取消关注成功".to_string(), data: Some(response) };
                (StatusCode::OK, Json(api_response)).into_response()
            }
            Err(err) => crate::presentation::http::controllers::map_domain_error_to_http_response(err),
        }
    }

    /// POST /api/v1/protected/user/block_user - 拉黑用户
    pub async fn block_user(
        State(controller): State<FriendController>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<BlockRequest>,
    ) -> axum::response::Response {
        debug!("用户 {} 拉黑用户 {}", user.user_id, request.to_user_id);
        let target_id = request.to_user_id.parse::<i64>().unwrap_or(0);
        match controller.friend_service.block_user(user.user_id, target_id, request.reason.clone()).await {
            Ok(_) => {
                let response = BlockResponse { success: true, message: "用户已拉黑".to_string(), relationship_id: String::new() };
                let api_response = ApiResponse { code: 0, msg: "用户已拉黑".to_string(), data: Some(response) };
                (StatusCode::OK, Json(api_response)).into_response()
            }
            Err(err) => crate::presentation::http::controllers::map_domain_error_to_http_response(err),
        }
    }

    /// POST /api/v1/protected/user/friends_list - 获取好友列表
    pub async fn get_friends_list(
        State(controller): State<FriendController>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<GetFriendsListRequest>,
    ) -> axum::response::Response {
        debug!("用户 {} 获取好友列表", user.user_id);
        let limit = request.limit.unwrap_or(20).min(100) as i32;
        let cursor = request
            .cursor
            .as_deref()
            .and_then(|s| s.parse::<i64>().ok())
            .unwrap_or(0);
        match controller.friend_service.get_friends_list(user.user_id, cursor, limit).await {
            Ok(list) => {
                let has_more = (list.len() as i32) > limit;
                let items: Vec<GetFriendsListResponse> = list.into_iter().take(limit as usize).map(|f| GetFriendsListResponse {
                    user_id: f.user_id.to_string(),
                    nickname: f.nickname,
                    avatar_url: f.avatar_url,
                    bio: None,
                    relationship_type: "friend".to_string(),
                    relationship_id: f.relationship_id,
                    created_at: f.friendship_created_at.to_rfc3339(),
                    last_active_at: f.last_active_at.map(|dt| dt.to_rfc3339()),
                    is_online: Some(f.is_online),
                }).collect();

                let paginated = crate::presentation::http::models::PaginatedResponse { items, has_more, next_cursor: None };
                let api_response = ApiResponse { code: 0, msg: "获取好友列表成功".to_string(), data: Some(paginated) };
                (StatusCode::OK, Json(api_response)).into_response()
            }
            Err(err) => crate::presentation::http::controllers::map_domain_error_to_http_response(err),
        }
    }

    /// POST /api/v1/protected/user/blocked_list - 获取拉黑列表
    pub async fn get_blocked_list(
        State(controller): State<FriendController>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<GetBlockedListRequest>,
    ) -> axum::response::Response {
        debug!("用户 {} 获取拉黑列表", user.user_id);
        let limit = request.limit.unwrap_or(20).min(100) as i32;
        let cursor = request
            .cursor
            .as_deref()
            .and_then(|s| s.parse::<i64>().ok())
            .unwrap_or(0);
        match controller.friend_service.get_blocked_list(user.user_id, cursor, limit).await {
            Ok(list) => {
                let has_more = (list.len() as i32) > limit;
                let items: Vec<GetBlockedListResponse> = list.into_iter().take(limit as usize).map(|f| GetBlockedListResponse {
                    user_id: f.user_id.to_string(),
                    nickname: f.nickname,
                    avatar_url: f.avatar_url,
                    relationship_id: f.relationship_id,
                    blocked_at: f.friendship_created_at.to_rfc3339(),
                    reason: f.block_reason,
                }).collect();

                let paginated = crate::presentation::http::models::PaginatedResponse { items, has_more, next_cursor: None };
                let api_response = ApiResponse { code: 0, msg: "获取拉黑列表成功".to_string(), data: Some(paginated) };
                (StatusCode::OK, Json(api_response)).into_response()
            }
            Err(err) => crate::presentation::http::controllers::map_domain_error_to_http_response(err),
        }
    }
}
