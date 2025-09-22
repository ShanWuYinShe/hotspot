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

use axum::{extract::State, http::StatusCode, response::IntoResponse, Json};
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
        State(controller): State<Self>,
        user: CurrentUser,
        Json(request): Json<AddFriendRequest>,
    ) -> impl IntoResponse {
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
                return (StatusCode::BAD_REQUEST, Json(error_response));
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

                (StatusCode::OK, Json(api_response))
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

                (status_code, Json(error_response))
            }
        }
    }

    /// POST /api/v1/protected/user/accept_friend - 接受好友申请
    pub async fn accept_friend(
        State(controller): State<Self>,
        user: CurrentUser,
        Json(request): Json<AcceptFriendRequest>,
    ) -> impl IntoResponse {
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

                (StatusCode::OK, Json(api_response))
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

                (status_code, Json(error_response))
            }
        }
    }

    /// POST /api/v1/protected/user/reject_friend - 拒绝好友申请
    pub async fn reject_friend(
        State(controller): State<Self>,
        user: CurrentUser,
        Json(request): Json<RejectFriendRequest>,
    ) -> impl IntoResponse {
        debug!("用户 {} 拒绝好友申请 {}", user.user_id, request.relationship_id);

        // TODO: 实现拒绝好友申请逻辑
        // 1. 验证关系记录存在且状态为pending
        // 2. 更新关系状态为rejected
        // 3. 可选：发送通知给申请者

        let response = RejectFriendResponse {
            success: true,
            message: "好友申请已拒绝".to_string(),
        };

        let api_response = ApiResponse {
            code: 0,
            msg: "好友申请已拒绝".to_string(),
            data: Some(response),
        };

        (StatusCode::OK, Json(api_response))
    }

    /// POST /api/v1/protected/user/follow_user - 关注用户
    pub async fn follow_user(
        State(controller): State<Self>,
        user: CurrentUser,
        Json(request): Json<FollowRequest>,
    ) -> impl IntoResponse {
        debug!("用户 {} 关注用户 {}", user.user_id, request.to_user_id);

        // TODO: 实现关注用户逻辑
        // 1. 验证目标用户存在
        // 2. 检查是否已经关注
        // 3. 创建关注关系记录
        // 4. 发送通知给目标用户

        let response = FollowResponse {
            success: true,
            message: "关注成功".to_string(),
            relationship_id: "temp_follow_id".to_string(), // TODO: 使用实际的关系ID
        };

        let api_response = ApiResponse {
            code: 0,
            msg: "关注成功".to_string(),
            data: Some(response),
        };

        (StatusCode::OK, Json(api_response))
    }

    /// POST /api/v1/protected/user/unfollow_user - 取消关注
    pub async fn unfollow_user(
        State(controller): State<Self>,
        user: CurrentUser,
        Json(request): Json<UnfollowRequest>,
    ) -> impl IntoResponse {
        debug!("用户 {} 取消关注用户 {}", user.user_id, request.to_user_id);

        // TODO: 实现取消关注逻辑
        // 1. 验证关注关系存在
        // 2. 删除或标记关注关系为inactive
        // 3. 可选：发送通知给目标用户

        let response = UnfollowResponse {
            success: true,
            message: "取消关注成功".to_string(),
        };

        let api_response = ApiResponse {
            code: 0,
            msg: "取消关注成功".to_string(),
            data: Some(response),
        };

        (StatusCode::OK, Json(api_response))
    }

    /// POST /api/v1/protected/user/block_user - 拉黑用户
    pub async fn block_user(
        State(controller): State<Self>,
        user: CurrentUser,
        Json(request): Json<BlockRequest>,
    ) -> impl IntoResponse {
        debug!("用户 {} 拉黑用户 {}", user.user_id, request.to_user_id);

        // TODO: 实现拉黑用户逻辑
        // 1. 验证目标用户存在
        // 2. 检查是否已经拉黑
        // 3. 创建拉黑关系记录
        // 4. 自动解除好友关系（如果存在）
        // 5. 不发送通知给被拉黑用户（隐私保护）

        let response = BlockResponse {
            success: true,
            message: "用户已拉黑".to_string(),
            relationship_id: "temp_block_id".to_string(), // TODO: 使用实际的关系ID
        };

        let api_response = ApiResponse {
            code: 0,
            msg: "用户已拉黑".to_string(),
            data: Some(response),
        };

        (StatusCode::OK, Json(api_response))
    }

    /// POST /api/v1/protected/user/friends_list - 获取好友列表
    pub async fn get_friends_list(
        State(controller): State<Self>,
        user: CurrentUser,
        Json(request): Json<GetFriendsListRequest>,
    ) -> impl IntoResponse {
        debug!("用户 {} 获取好友列表", user.user_id);

        // TODO: 实现获取好友列表逻辑
        // 1. 根据status过滤关系状态
        // 2. 分页查询好友关系
        // 3. 关联用户信息
        // 4. 返回分页结果

        let friends = vec![
            GetFriendsListResponse {
                user_id: "example_friend_id".to_string(),
                nickname: "示例好友".to_string(),
                avatar_url: Some("https://example.com/avatar.jpg".to_string()),
                bio: Some("这是一个示例好友".to_string()),
                relationship_type: "friend".to_string(),
                relationship_id: "example_relationship_id".to_string(),
                created_at: chrono::Utc::now().to_rfc3339(),
                last_active_at: Some(chrono::Utc::now().to_rfc3339()),
                is_online: Some(true),
            }
        ];

        let paginated_response = crate::presentation::http::models::PaginatedResponse {
            items: friends,
            has_more: false,
            next_cursor: None,
        };

        let api_response = ApiResponse {
            code: 0,
            msg: "获取好友列表成功".to_string(),
            data: Some(paginated_response),
        };

        (StatusCode::OK, Json(api_response))
    }

    /// POST /api/v1/protected/user/blocked_list - 获取拉黑列表
    pub async fn get_blocked_list(
        State(controller): State<Self>,
        user: CurrentUser,
        Json(request): Json<GetBlockedListRequest>,
    ) -> impl IntoResponse {
        debug!("用户 {} 获取拉黑列表", user.user_id);

        // TODO: 实现获取拉黑列表逻辑
        // 1. 查询用户的拉黑关系
        // 2. 分页查询
        // 3. 关联被拉黑用户信息
        // 4. 返回分页结果

        let blocked_users = vec![
            GetBlockedListResponse {
                user_id: "example_blocked_id".to_string(),
                nickname: "被拉黑用户".to_string(),
                avatar_url: Some("https://example.com/blocked_avatar.jpg".to_string()),
                relationship_id: "example_block_relationship_id".to_string(),
                blocked_at: chrono::Utc::now().to_rfc3339(),
                reason: Some("垃圾信息".to_string()),
            }
        ];

        let paginated_response = crate::presentation::http::models::PaginatedResponse {
            items: blocked_users,
            has_more: false,
            next_cursor: None,
        };

        let api_response = ApiResponse {
            code: 0,
            msg: "获取拉黑列表成功".to_string(),
            data: Some(paginated_response),
        };

        (StatusCode::OK, Json(api_response))
    }
}
