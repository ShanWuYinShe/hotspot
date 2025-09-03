/// 好友控制器
/// 
/// 处理好友相关的HTTP请求

use axum::{
    extract::{Query, State},
    response::Response,
    Extension,
};
use serde::{Deserialize, Serialize};
use tracing::{debug, error};

use crate::application::services::FriendApplicationService;
use crate::presentation::http::auth::CurrentUser;
use crate::presentation::http::controllers::{ok_response, error_response};
use axum::http::StatusCode;
use crate::ports::repository_ports::{FriendRepositoryPort, UserRepositoryPort, FriendshipStatus};

/// 好友控制器
#[derive(Clone)]
pub struct FriendController<F, U>
where
    F: FriendRepositoryPort,
    U: UserRepositoryPort,
{
    friend_service: FriendApplicationService<F, U>,
}

impl<F, U> FriendController<F, U>
where
    F: FriendRepositoryPort,
    U: UserRepositoryPort,
{
    pub fn new(friend_service: FriendApplicationService<F, U>) -> Self {
        Self { friend_service }
    }

    /// 发送好友请求
    pub async fn send_friend_request(
        State(controller): State<FriendController<F, U>>,
        Extension(current_user): Extension<CurrentUser>,
        Query(params): Query<SendFriendRequestRequest>,
    ) -> Response {
        debug!("发送好友请求: {} -> {}", current_user.user_id, params.addressee_id);

        match controller.friend_service.send_friend_request(current_user.user_id, params.addressee_id).await {
            Ok(friendship) => {
                let response_data = FriendRequestResponse {
                    friendship_id: friendship.friendship_id,
                    requester_id: friendship.requester_id,
                    addressee_id: friendship.addressee_id,
                    status: friendship.status,
                    created_at: friendship.created_at.to_rfc3339(),
                };
                ok_response(response_data)
            }
            Err(e) => {
                error!("发送好友请求失败: {:?}", e);
                error_response(StatusCode::OK, &format!("发送好友请求失败: {}", e), None)
            }
        }
    }

    /// 接受好友请求
    pub async fn accept_friend_request(
        State(controller): State<FriendController<F, U>>,
        Extension(current_user): Extension<CurrentUser>,
        Query(params): Query<HandleFriendRequestRequest>,
    ) -> Response {
        debug!("接受好友请求: {} <- {}", current_user.user_id, params.requester_id);

        match controller.friend_service.accept_friend_request(params.requester_id, current_user.user_id).await {
            Ok(()) => {
                ok_response("好友请求已接受")
            }
            Err(e) => {
                error!("接受好友请求失败: {:?}", e);
                error_response(StatusCode::OK, &format!("接受好友请求失败: {}", e), None)
            }
        }
    }

    /// 拒绝好友请求
    pub async fn reject_friend_request(
        State(controller): State<FriendController<F, U>>,
        Extension(current_user): Extension<CurrentUser>,
        Query(params): Query<HandleFriendRequestRequest>,
    ) -> Response {
        debug!("拒绝好友请求: {} <- {}", current_user.user_id, params.requester_id);

        match controller.friend_service.reject_friend_request(params.requester_id, current_user.user_id).await {
            Ok(()) => {
                ok_response("好友请求已拒绝")
            }
            Err(e) => {
                error!("拒绝好友请求失败: {:?}", e);
                error_response(StatusCode::OK, &format!("拒绝好友请求失败: {}", e), None)
            }
        }
    }

    /// 删除好友关系
    pub async fn remove_friendship(
        State(controller): State<FriendController<F, U>>,
        Extension(current_user): Extension<CurrentUser>,
        Query(params): Query<RemoveFriendshipRequest>,
    ) -> Response {
        debug!("删除好友关系: {} <-> {}", current_user.user_id, params.friend_id);

        match controller.friend_service.remove_friendship(current_user.user_id, params.friend_id).await {
            Ok(()) => {
                ok_response("好友关系已删除")
            }
            Err(e) => {
                error!("删除好友关系失败: {:?}", e);
                error_response(StatusCode::OK, &format!("删除好友关系失败: {}", e), None)
            }
        }
    }

    /// 获取好友列表
    pub async fn get_friends(
        State(controller): State<FriendController<F, U>>,
        Extension(current_user): Extension<CurrentUser>,
        Query(params): Query<GetFriendsRequest>,
    ) -> Response {
        debug!("获取用户{}的好友列表", current_user.user_id);

        let cursor = params.cursor.unwrap_or(0);
        let limit = params.limit.unwrap_or(20).min(100);

        match controller.friend_service.get_friends(current_user.user_id, cursor, limit).await {
            Ok((friends, has_more, next_cursor)) => {
                let response_data = FriendsListResponse {
                    friends,
                    has_more,
                    next_cursor,
                };
                ok_response(response_data)
            }
            Err(e) => {
                error!("获取好友列表失败: {:?}", e);
                error_response(StatusCode::OK, &format!("获取好友列表失败: {}", e), None)
            }
        }
    }

    /// 获取收到的好友请求
    pub async fn get_received_friend_requests(
        State(controller): State<FriendController<F, U>>,
        Extension(current_user): Extension<CurrentUser>,
        Query(params): Query<GetFriendRequestsRequest>,
    ) -> Response {
        debug!("获取用户{}收到的好友请求", current_user.user_id);

        let cursor = params.cursor.unwrap_or(0);
        let limit = params.limit.unwrap_or(20).min(100);

        match controller.friend_service.get_received_friend_requests(current_user.user_id, cursor, limit).await {
            Ok((requests, has_more, next_cursor)) => {
                let response_data = FriendRequestsListResponse {
                    requests: requests.into_iter().map(|req| FriendRequestItem {
                        friendship_id: req.friendship_id,
                        requester_id: req.requester_id,
                        requester_nickname: req.requester_nickname,
                        requester_avatar_url: req.requester_avatar_url,
                        status: req.status,
                        created_at: req.created_at.to_rfc3339(),
                    }).collect(),
                    has_more,
                    next_cursor,
                };
                ok_response(response_data)
            }
            Err(e) => {
                error!("获取收到的好友请求失败: {:?}", e);
                error_response(StatusCode::OK, &format!("获取收到的好友请求失败: {}", e), None)
            }
        }
    }

    /// 获取发送的好友请求
    pub async fn get_sent_friend_requests(
        State(controller): State<FriendController<F, U>>,
        Extension(current_user): Extension<CurrentUser>,
        Query(params): Query<GetFriendRequestsRequest>,
    ) -> Response {
        debug!("获取用户{}发送的好友请求", current_user.user_id);

        let cursor = params.cursor.unwrap_or(0);
        let limit = params.limit.unwrap_or(20).min(100);

        match controller.friend_service.get_sent_friend_requests(current_user.user_id, cursor, limit).await {
            Ok((requests, has_more, next_cursor)) => {
                let response_data = FriendRequestsListResponse {
                    requests: requests.into_iter().map(|req| FriendRequestItem {
                        friendship_id: req.friendship_id,
                        requester_id: req.requester_id,
                        requester_nickname: req.requester_nickname,
                        requester_avatar_url: req.requester_avatar_url,
                        status: req.status,
                        created_at: req.created_at.to_rfc3339(),
                    }).collect(),
                    has_more,
                    next_cursor,
                };
                ok_response(response_data)
            }
            Err(e) => {
                error!("获取发送的好友请求失败: {:?}", e);
                error_response(StatusCode::OK, &format!("获取发送的好友请求失败: {}", e), None)
            }
        }
    }

    /// 检查好友关系
    pub async fn check_friendship(
        State(controller): State<FriendController<F, U>>,
        Extension(current_user): Extension<CurrentUser>,
        Query(params): Query<CheckFriendshipRequest>,
    ) -> Response {
        debug!("检查用户{}和{}的好友关系", current_user.user_id, params.user_id);

        match controller.friend_service.are_friends(current_user.user_id, params.user_id).await {
            Ok(are_friends) => {
                let response_data = FriendshipStatusResponse { are_friends };
                ok_response(response_data)
            }
            Err(e) => {
                error!("检查好友关系失败: {:?}", e);
                error_response(StatusCode::OK, &format!("检查好友关系失败: {}", e), None)
            }
        }
    }

    /// 获取待处理好友请求数量
    pub async fn get_pending_requests_count(
        State(controller): State<FriendController<F, U>>,
        Extension(current_user): Extension<CurrentUser>,
    ) -> Response {
        debug!("获取用户{}的待处理好友请求数量", current_user.user_id);

        match controller.friend_service.get_pending_requests_count(current_user.user_id).await {
            Ok(count) => {
                let response_data = PendingRequestsCountResponse { count };
                ok_response(response_data)
            }
            Err(e) => {
                error!("获取待处理好友请求数量失败: {:?}", e);
                error_response(StatusCode::OK, &format!("获取待处理好友请求数量失败: {}", e), None)
            }
        }
    }
}

// 请求和响应数据结构

#[derive(Debug, Deserialize)]
pub struct SendFriendRequestRequest {
    pub addressee_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct HandleFriendRequestRequest {
    pub requester_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct RemoveFriendshipRequest {
    pub friend_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct GetFriendsRequest {
    pub cursor: Option<i64>,
    pub limit: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct GetFriendRequestsRequest {
    pub cursor: Option<i64>,
    pub limit: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct CheckFriendshipRequest {
    pub user_id: i64,
}

#[derive(Debug, Serialize)]
pub struct FriendRequestResponse {
    pub friendship_id: String,
    pub requester_id: i64,
    pub addressee_id: i64,
    pub status: FriendshipStatus,
    pub created_at: String,
}

#[derive(Debug, Serialize)]
pub struct FriendsListResponse {
    pub friends: Vec<crate::application::queries::UserSearchResult>,
    pub has_more: bool,
    pub next_cursor: Option<i64>,
}

#[derive(Debug, Serialize)]
pub struct FriendRequestsListResponse {
    pub requests: Vec<FriendRequestItem>,
    pub has_more: bool,
    pub next_cursor: Option<i64>,
}

#[derive(Debug, Serialize)]
pub struct FriendRequestItem {
    pub friendship_id: String,
    pub requester_id: i64,
    pub requester_nickname: String,
    pub requester_avatar_url: Option<String>,
    pub status: FriendshipStatus,
    pub created_at: String,
}

#[derive(Debug, Serialize)]
pub struct FriendshipStatusResponse {
    pub are_friends: bool,
}

#[derive(Debug, Serialize)]
pub struct PendingRequestsCountResponse {
    pub count: i64,
}
