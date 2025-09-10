/// 好友控制器
/// 
/// 功能职责：
/// - 处理好友相关的HTTP请求
/// - 实现好友请求发送、接受、拒绝功能
/// - 管理好友关系和好友列表
/// - 提供好友状态查询功能
/// 
/// API端点：
/// - POST /api/v1/protected/friends/send_request - 发送好友请求
/// - POST /api/v1/protected/friends/accept_request - 接受好友请求
/// - POST /api/v1/protected/friends/reject_request - 拒绝好友请求
/// - POST /api/v1/protected/friends/remove_friendship - 移除好友关系
/// - POST /api/v1/protected/friends/list - 获取好友列表
/// - POST /api/v1/protected/friends/received_requests - 获取收到的好友请求
/// - POST /api/v1/protected/friends/sent_requests - 获取发送的好友请求
/// - POST /api/v1/protected/friends/check_friendship - 检查好友关系状态
/// - POST /api/v1/protected/friends/pending_count - 获取待处理请求数量
use axum::{
    extract::{State, Extension},
    http::StatusCode,
    response::Json,
};
use serde::{Deserialize, Serialize};

use crate::{
    application::services::FriendApplicationService,
    domain::{FriendRequest, Friendship, FriendProfile, FriendshipStatus, FriendRequestStatus},
    utils::response::ApiResponse,
    presentation::http::models::common::{PaginatedResponse, PaginationMeta},
    utils::JsSafeI64,
    presentation::http::auth::CurrentUser,
};

/// 发送好友请求
#[derive(Debug, Deserialize)]
pub struct SendFriendRequestRequest {
    /// 目标用户ID
    pub target_id: JsSafeI64,
    /// 请求消息（可选）
    pub message: Option<String>,
}

/// 处理好友请求
#[derive(Debug, Deserialize)]
pub struct HandleFriendRequestRequest {
    /// 好友请求ID
    pub request_id: String,
}

/// 移除好友关系
#[derive(Debug, Deserialize)]
pub struct RemoveFriendshipRequest {
    /// 好友用户ID
    pub friend_id: JsSafeI64,
}

/// 检查好友关系状态
#[derive(Debug, Deserialize)]
pub struct CheckFriendshipRequest {
    /// 目标用户ID
    pub user_id: JsSafeI64,
}

/// 分页请求
#[derive(Debug, Deserialize)]
pub struct PaginationRequest {
    /// 游标
    pub cursor: Option<i64>,
    /// 限制数量
    pub limit: Option<i32>,
}

/// 好友请求响应
#[derive(Debug, Serialize)]
pub struct FriendRequestResponse {
    /// 请求ID
    pub request_id: String,
    /// 请求者ID
    pub requester_id: JsSafeI64,
    /// 目标用户ID
    pub target_id: JsSafeI64,
    /// 请求状态
    pub status: String,
    /// 请求消息
    pub message: Option<String>,
    /// 创建时间
    pub created_at: String,
    /// 响应时间
    pub responded_at: Option<String>,
}

/// 好友关系响应
#[derive(Debug, Serialize)]
pub struct FriendshipResponse {
    /// 好友关系ID
    pub friendship_id: String,
    /// 用户ID1
    pub user_id_1: JsSafeI64,
    /// 用户ID2
    pub user_id_2: JsSafeI64,
    /// 关系状态
    pub status: String,
    /// 创建时间
    pub created_at: String,
    /// 更新时间
    pub updated_at: String,
}

/// 好友资料响应
#[derive(Debug, Serialize)]
pub struct FriendProfileResponse {
    /// 用户ID
    pub user_id: JsSafeI64,
    /// 昵称
    pub nickname: String,
    /// 头像URL
    pub avatar_url: Option<String>,
    /// 最后活跃时间
    pub last_active_at: String,
    /// 成为好友的时间
    pub friendship_created_at: String,
}

/// 好友关系状态响应
#[derive(Debug, Serialize)]
pub struct FriendshipStatusResponse {
    /// 好友关系状态
    pub status: String,
}

/// 待处理请求数量响应
#[derive(Debug, Serialize)]
pub struct PendingRequestsCountResponse {
    /// 待处理请求数量
    pub count: JsSafeI64,
}

/// 好友控制器
#[derive(Clone)]
pub struct FriendController<S>
where
    S: FriendApplicationService + Clone,
{
    friend_service: S,
}

impl<S> FriendController<S>
where
    S: FriendApplicationService + Clone,
{
    /// 创建新的好友控制器实例
    pub fn new(friend_service: S) -> Self {
        Self { friend_service }
    }

    /// 将好友请求领域模型转换为响应模型
    fn friend_request_to_response(request: &FriendRequest) -> FriendRequestResponse {
        FriendRequestResponse {
            request_id: request.request_id.to_string(),
            requester_id: JsSafeI64::new(request.requester_id),
            target_id: JsSafeI64::new(request.target_id),
            status: match request.status {
                FriendRequestStatus::Pending => "pending".to_string(),
                FriendRequestStatus::Accepted => "accepted".to_string(),
                FriendRequestStatus::Rejected => "rejected".to_string(),
                FriendRequestStatus::Expired => "expired".to_string(),
            },
            message: request.message.clone(),
            created_at: request.created_at.to_rfc3339(),
            responded_at: None, // TODO: FriendRequest 没有 responded_at 字段
        }
    }

    /// 将好友关系领域模型转换为响应模型
    fn friendship_to_response(friendship: &Friendship) -> FriendshipResponse {
        FriendshipResponse {
            friendship_id: friendship.friendship_id.to_string(),
            user_id_1: JsSafeI64::new(friendship.user1_id),
            user_id_2: JsSafeI64::new(friendship.user2_id),
            status: "friends".to_string(), // Friendship 结构体表示已建立的好友关系
            created_at: friendship.created_at.to_rfc3339(),
            updated_at: friendship.created_at.to_rfc3339(), // 使用 created_at 作为 updated_at
        }
    }

    /// 将好友资料领域模型转换为响应模型
    fn friend_profile_to_response(profile: &FriendProfile) -> FriendProfileResponse {
        FriendProfileResponse {
            user_id: JsSafeI64::new(profile.user_id),
            nickname: profile.nickname.clone(),
            avatar_url: profile.avatar_url.clone(),
            last_active_at: profile.last_active_at.map(|dt| dt.to_rfc3339()).unwrap_or_default(),
            friendship_created_at: profile.friendship_created_at.to_rfc3339(),
        }
    }

    /// 发送好友请求
    pub async fn send_friend_request(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<SendFriendRequestRequest>,
    ) -> Result<Json<ApiResponse<FriendRequestResponse>>, StatusCode> {
        let result = controller
            .friend_service
            .send_friend_request(
                current_user.user_id,
                request.target_id.value(),
                request.message.as_deref(),
            )
            .await;

        match result {
            Ok(friend_request) => {
                let response = Self::friend_request_to_response(&friend_request);
                Ok(Json(ApiResponse::success(response)))
            }
            Err(e) => {
                tracing::error!("发送好友请求失败: {:?}", e);
                Err(StatusCode::INTERNAL_SERVER_ERROR)
            }
        }
    }

    /// 接受好友请求
    pub async fn accept_friend_request(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<HandleFriendRequestRequest>,
    ) -> Result<Json<ApiResponse<FriendshipResponse>>, StatusCode> {
        let result = controller
            .friend_service
            .accept_friend_request(&request.request_id, current_user.user_id)
            .await;

        match result {
            Ok(friendship) => {
                let response = Self::friendship_to_response(&friendship);
                Ok(Json(ApiResponse::success(response)))
            }
            Err(e) => {
                tracing::error!("接受好友请求失败: {:?}", e);
                Err(StatusCode::INTERNAL_SERVER_ERROR)
            }
        }
    }

    /// 拒绝好友请求
    pub async fn reject_friend_request(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<HandleFriendRequestRequest>,
    ) -> Result<Json<ApiResponse<()>>, StatusCode> {
        let result = controller
            .friend_service
            .reject_friend_request(&request.request_id, current_user.user_id)
            .await;

        match result {
            Ok(_) => Ok(Json(ApiResponse::success(()))),
            Err(e) => {
                tracing::error!("拒绝好友请求失败: {:?}", e);
                Err(StatusCode::INTERNAL_SERVER_ERROR)
            }
        }
    }

    /// 移除好友关系
    pub async fn remove_friendship(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<RemoveFriendshipRequest>,
    ) -> Result<Json<ApiResponse<()>>, StatusCode> {
        let result = controller
            .friend_service
            .remove_friendship(current_user.user_id, request.friend_id.value())
            .await;

        match result {
            Ok(_) => Ok(Json(ApiResponse::success(()))),
            Err(e) => {
                tracing::error!("移除好友关系失败: {:?}", e);
                Err(StatusCode::INTERNAL_SERVER_ERROR)
            }
        }
    }

    /// 获取好友列表
    pub async fn get_friends(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<PaginationRequest>,
    ) -> Result<Json<ApiResponse<PaginatedResponse<FriendProfileResponse>>>, StatusCode> {
        let cursor = request.cursor.unwrap_or(0);
        let limit = request.limit.unwrap_or(20).min(100);

        let result = controller
            .friend_service
            .get_friends(current_user.user_id, cursor, limit + 1)
            .await;

        match result {
            Ok(friends) => {
                let has_more = friends.len() > limit as usize;
                let items: Vec<FriendProfileResponse> = friends
                    .iter()
                    .take(limit as usize)
                    .map(Self::friend_profile_to_response)
                    .collect();

                let next_cursor = if has_more {
                    items.last().map(|item| JsSafeI64::new(item.user_id.value()))
                } else {
                    None
                };

                let pagination = PaginationMeta {
                    has_more,
                    next_cursor,
                };

                let response = PaginatedResponse { items, pagination };
                Ok(Json(ApiResponse::success(response)))
            }
            Err(e) => {
                tracing::error!("获取好友列表失败: {:?}", e);
                Err(StatusCode::INTERNAL_SERVER_ERROR)
            }
        }
    }

    /// 获取收到的好友请求
    pub async fn get_received_friend_requests(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<PaginationRequest>,
    ) -> Result<Json<ApiResponse<PaginatedResponse<FriendRequestResponse>>>, StatusCode> {
        let cursor = request.cursor.unwrap_or(0);
        let limit = request.limit.unwrap_or(20).min(100);

        let result = controller
            .friend_service
            .get_received_friend_requests(current_user.user_id, cursor, limit + 1)
            .await;

        match result {
            Ok(requests) => {
                let has_more = requests.len() > limit as usize;
                let items: Vec<FriendRequestResponse> = requests
                    .iter()
                    .take(limit as usize)
                    .map(Self::friend_request_to_response)
                    .collect();

                let next_cursor = if has_more {
                    items.last().map(|item| JsSafeI64::new(item.requester_id.value()))
                } else {
                    None
                };

                let pagination = PaginationMeta {
                    has_more,
                    next_cursor,
                };

                let response = PaginatedResponse { items, pagination };
                Ok(Json(ApiResponse::success(response)))
            }
            Err(e) => {
                tracing::error!("获取收到的好友请求失败: {:?}", e);
                Err(StatusCode::INTERNAL_SERVER_ERROR)
            }
        }
    }

    /// 获取发送的好友请求
    pub async fn get_sent_friend_requests(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<PaginationRequest>,
    ) -> Result<Json<ApiResponse<PaginatedResponse<FriendRequestResponse>>>, StatusCode> {
        let cursor = request.cursor.unwrap_or(0);
        let limit = request.limit.unwrap_or(20).min(100);

        let result = controller
            .friend_service
            .get_sent_friend_requests(current_user.user_id, cursor, limit + 1)
            .await;

        match result {
            Ok(requests) => {
                let has_more = requests.len() > limit as usize;
                let items: Vec<FriendRequestResponse> = requests
                    .iter()
                    .take(limit as usize)
                    .map(Self::friend_request_to_response)
                    .collect();

                let next_cursor = if has_more {
                    items.last().map(|item| JsSafeI64::new(item.target_id.value()))
                } else {
                    None
                };

                let pagination = PaginationMeta {
                    has_more,
                    next_cursor,
                };

                let response = PaginatedResponse { items, pagination };
                Ok(Json(ApiResponse::success(response)))
            }
            Err(e) => {
                tracing::error!("获取发送的好友请求失败: {:?}", e);
                Err(StatusCode::INTERNAL_SERVER_ERROR)
            }
        }
    }

    /// 检查好友关系状态
    pub async fn check_friendship(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(request): Json<CheckFriendshipRequest>,
    ) -> Result<Json<ApiResponse<FriendshipStatusResponse>>, StatusCode> {
        let result = controller
            .friend_service
            .check_friendship(current_user.user_id, request.user_id.value())
            .await;

        match result {
            Ok(status) => {
                let status_str = match status {
                    FriendshipStatus::NotFriends => "not_friends",
                    FriendshipStatus::RequestSent => "request_sent",
                    FriendshipStatus::RequestReceived => "request_received",
                    FriendshipStatus::Friends => "friends",
                    FriendshipStatus::Blocked => "blocked",
                };

                let response = FriendshipStatusResponse {
                    status: status_str.to_string(),
                };
                Ok(Json(ApiResponse::success(response)))
            }
            Err(e) => {
                tracing::error!("检查好友关系状态失败: {:?}", e);
                Err(StatusCode::INTERNAL_SERVER_ERROR)
            }
        }
    }

    /// 获取待处理的好友请求数量
    pub async fn get_pending_requests_count(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
    ) -> Result<Json<ApiResponse<PendingRequestsCountResponse>>, StatusCode> {
        let result = controller
            .friend_service
            .get_pending_requests_count(current_user.user_id)
            .await;

        match result {
            Ok(count) => {
                let response = PendingRequestsCountResponse { count: JsSafeI64::new(count) };
                Ok(Json(ApiResponse::success(response)))
            }
            Err(e) => {
                tracing::error!("获取待处理好友请求数量失败: {:?}", e);
                Err(StatusCode::INTERNAL_SERVER_ERROR)
            }
        }
    }
}
