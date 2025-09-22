/// 通知推送控制器
/// 
/// 功能职责：
/// - 处理通知发送和管理
/// - 提供通知列表查询
/// - 管理通知已读状态
/// - 处理通知设置和偏好
/// 
/// API端点：
/// - POST /api/v1/protected/notifications/send - 发送通知
/// - POST /api/v1/protected/notifications/list - 获取通知列表
/// - POST /api/v1/protected/notifications/mark_read - 标记通知已读
/// - POST /api/v1/protected/notifications/unread_count - 获取未读通知数
/// - POST /api/v1/protected/notifications/settings - 获取通知设置
/// - POST /api/v1/protected/notifications/update_settings - 更新通知设置

use axum::{extract::State, http::StatusCode, response::IntoResponse, Json};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tracing::debug;
use utoipa::ToSchema;

use crate::application::services::notification_application_service::NotificationApplicationService;
use crate::presentation::http::models::ApiResponse;
use crate::presentation::http::auth::CurrentUser;
use crate::adapters::adapters_pg::notification_repository::{NotificationType, NotificationStatus, NotificationPriority};

/// 通知推送控制器
#[derive(Clone)]
pub struct NotificationController {
    notification_service: Arc<dyn NotificationApplicationService>,
}

impl NotificationController {
    pub fn new(notification_service: Arc<dyn NotificationApplicationService>) -> Self {
        Self { notification_service }
    }
}

// ==================== 请求/响应类型定义 ====================

/// 发送通知请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct SendNotificationRequest {
    /// 接收者ID数组
    pub user_ids: Vec<String>,
    /// 通知类型
    pub r#type: String,
    /// 通知标题
    pub title: String,
    /// 通知内容
    pub content: String,
    /// 扩展数据（JSON）
    pub data: Option<serde_json::Value>,
}

/// 发送通知响应
#[derive(Debug, Serialize, ToSchema)]
pub struct SendNotificationResponse {
    /// 是否发送成功
    pub success: bool,
    /// 响应消息
    pub message: String,
    /// 成功发送数量
    pub sent_count: u32,
}

/// 获取通知列表请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct GetNotificationsRequest {
    /// 通知类型过滤
    pub r#type: Option<String>,
    /// 已读状态过滤
    pub is_read: Option<bool>,
    /// 返回数量限制，默认20
    pub limit: Option<u32>,
    /// 分页游标
    pub cursor: Option<String>,
}

/// 通知通用响应类型（用于查询类API）
#[derive(Debug, Serialize, ToSchema)]
pub struct NotificationResponse {
    /// 通知ID
    pub notification_id: String,
    /// 接收者ID
    pub user_id: String,
    /// 通知类型：'friend_request'|'group_invite'|'message'|'moment_like'|'comment'|'system'
    pub r#type: String,
    /// 通知标题
    pub title: String,
    /// 通知内容
    pub content: String,
    /// 扩展数据（JSON）
    pub data: Option<serde_json::Value>,
    /// 是否已读
    pub is_read: bool,
    /// 创建时间（ISO 8601）
    pub created_at: String,
    /// 已读时间（ISO 8601）
    pub read_at: Option<String>,
}

/// 标记通知已读请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct MarkNotificationReadRequest {
    /// 通知ID数组
    pub notification_ids: Vec<String>,
}

/// 标记通知已读响应
#[derive(Debug, Serialize, ToSchema)]
pub struct MarkNotificationReadResponse {
    /// 是否标记成功
    pub success: bool,
    /// 响应消息
    pub message: String,
    /// 标记数量
    pub marked_count: u32,
}

/// 获取未读通知数请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct GetUnreadNotificationCountRequest {
    /// 通知类型过滤
    pub r#type: Option<String>,
}

/// 获取未读通知数响应
#[derive(Debug, Serialize, ToSchema)]
pub struct GetUnreadNotificationCountResponse {
    /// 总未读数量
    pub total_unread: u32,
    /// 按类型分组的未读数（仅当未指定type时返回）
    pub by_type: Option<Vec<NotificationTypeCount>>,
}

/// 通知类型计数
#[derive(Debug, Serialize, ToSchema)]
pub struct NotificationTypeCount {
    /// 通知类型
    pub r#type: String,
    /// 未读数量
    pub unread_count: u32,
}

/// 获取通知设置请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct GetNotificationSettingsRequest {
    // 空请求体
}

/// 获取通知设置响应
#[derive(Debug, Serialize, ToSchema)]
pub struct GetNotificationSettingsResponse {
    /// 推送通知开关
    pub push_enabled: bool,
    /// 邮件通知开关
    pub email_enabled: bool,
    /// 好友请求通知
    pub friend_request: bool,
    /// 群组邀请通知
    pub group_invite: bool,
    /// 消息通知
    pub message: bool,
    /// 瞬间点赞通知
    pub moment_like: bool,
    /// 评论通知
    pub comment: bool,
    /// 系统通知
    pub system: bool,
}

/// 更新通知设置请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct UpdateNotificationSettingsRequest {
    /// 推送通知开关
    pub push_enabled: Option<bool>,
    /// 邮件通知开关
    pub email_enabled: Option<bool>,
    /// 好友请求通知
    pub friend_request: Option<bool>,
    /// 群组邀请通知
    pub group_invite: Option<bool>,
    /// 消息通知
    pub message: Option<bool>,
    /// 瞬间点赞通知
    pub moment_like: Option<bool>,
    /// 评论通知
    pub comment: Option<bool>,
    /// 系统通知
    pub system: Option<bool>,
}

/// 更新通知设置响应
#[derive(Debug, Serialize, ToSchema)]
pub struct UpdateNotificationSettingsResponse {
    /// 是否更新成功
    pub success: bool,
    /// 响应消息
    pub message: String,
}

// ==================== 控制器实现 ====================

impl NotificationController {
    /// POST /api/v1/protected/notifications/send - 发送通知
    pub async fn send_notification(
        State(controller): State<Self>,
        user: CurrentUser,
        Json(request): Json<SendNotificationRequest>,
    ) -> impl IntoResponse {
        debug!(
            "用户 {} 发送通知给 {} 个用户，类型：{}",
            user.user_id,
            request.user_ids.len(),
            request.r#type
        );

        // TODO: 实现发送通知逻辑
        // 1. 验证发送权限（可能需要管理员权限）
        // 2. 验证接收者用户存在
        // 3. 根据用户通知设置过滤
        // 4. 创建通知记录
        // 5. 通过WebSocket实时推送
        // 6. 发送离线推送（APNs/FCM）

        let response = SendNotificationResponse {
            success: true,
            message: "通知发送成功".to_string(),
            sent_count: request.user_ids.len() as u32,
        };

        let api_response = ApiResponse {
            code: 0,
            msg: "通知发送成功".to_string(),
            data: Some(response),
        };

        (StatusCode::OK, Json(api_response))
    }

    /// POST /api/v1/protected/notifications/list - 获取通知列表
    pub async fn get_notifications(
        State(controller): State<Self>,
        user: CurrentUser,
        Json(request): Json<GetNotificationsRequest>,
    ) -> impl IntoResponse {
        debug!("用户 {} 获取通知列表", user.user_id);

        // TODO: 实现获取通知列表逻辑
        // 1. 查询用户的通知记录
        // 2. 根据类型和已读状态过滤
        // 3. 分页处理
        // 4. 返回通知详情

        // 解析状态过滤条件
        let status_filter = request.status.and_then(|s| match s.as_str() {
            "unread" => Some(crate::adapters::adapters_pg::notification_repository::NotificationStatus::Unread),
            "read" => Some(crate::adapters::adapters_pg::notification_repository::NotificationStatus::Read),
            "dismissed" => Some(crate::adapters::adapters_pg::notification_repository::NotificationStatus::Dismissed),
            _ => None,
        });

        // 解析游标
        let cursor = request.cursor.unwrap_or("0".to_string()).parse::<i64>().unwrap_or(0);
        let limit = request.limit.unwrap_or(20).min(100) as i32; // 限制最大100条

        // 调用通知应用服务获取通知列表
        match controller.notification_service.get_user_notifications(
            user.user_id,
            status_filter,
            cursor,
            limit,
        ).await {
            Ok(notifications_data) => {
                let notifications: Vec<NotificationResponse> = notifications_data
                    .into_iter()
                    .map(|notification| NotificationResponse {
                        notification_id: notification.notification_id,
                        user_id: user.user_id.to_string(),
                        r#type: match notification.notification_type {
                            crate::adapters::adapters_pg::notification_repository::NotificationType::System => "system".to_string(),
                            crate::adapters::adapters_pg::notification_repository::NotificationType::NewMessage => "new_message".to_string(),
                            crate::adapters::adapters_pg::notification_repository::NotificationType::GroupInvite => "group_invite".to_string(),
                            crate::adapters::adapters_pg::notification_repository::NotificationType::FriendRequest => "friend_request".to_string(),
                            crate::adapters::adapters_pg::notification_repository::NotificationType::FriendAccepted => "friend_accepted".to_string(),
                            crate::adapters::adapters_pg::notification_repository::NotificationType::MomentLike => "moment_like".to_string(),
                            crate::adapters::adapters_pg::notification_repository::NotificationType::MomentComment => "moment_comment".to_string(),
                            crate::adapters::adapters_pg::notification_repository::NotificationType::Security => "security".to_string(),
                        },
                        title: notification.title,
                        content: notification.content,
                        data: Some(notification.data),
                        is_read: matches!(notification.status, crate::adapters::adapters_pg::notification_repository::NotificationStatus::Read),
                        created_at: notification.created_at.to_rfc3339(),
                        read_at: notification.read_at.map(|t| t.to_rfc3339()),
                    })
                    .collect();

                // 判断是否还有更多数据
                let has_more = notifications.len() == limit as usize;
                let next_cursor = if has_more && !notifications.is_empty() {
                    Some((cursor + limit as i64).to_string())
                } else {
                    None
                };

                let paginated_response = crate::presentation::http::models::PaginatedResponse {
                    items: notifications,
                    has_more,
                    next_cursor,
                };

                let api_response = ApiResponse {
                    code: 0,
                    msg: "获取通知列表成功".to_string(),
                    data: Some(paginated_response),
                };

                (StatusCode::OK, Json(api_response))
            }
            Err(err) => {
                let (status_code, error_msg) = match err {
                    crate::domain::DomainError::ValidationError { reason, .. } => (StatusCode::BAD_REQUEST, reason),
                    _ => (StatusCode::INTERNAL_SERVER_ERROR, "服务器内部错误".to_string()),
                };

                let error_response = ApiResponse {
                    code: if status_code == StatusCode::BAD_REQUEST { 400 } else { 500 },
                    msg: error_msg,
                    data: None::<()>,
                };

                (status_code, Json(error_response))
            }
        }
    }

    /// POST /api/v1/protected/notifications/mark_read - 标记通知已读
    pub async fn mark_notification_read(
        State(controller): State<Self>,
        user: CurrentUser,
        Json(request): Json<MarkNotificationReadRequest>,
    ) -> impl IntoResponse {
        debug!(
            "用户 {} 标记 {} 个通知为已读",
            user.user_id,
            request.notification_ids.len()
        );

        // 调用通知应用服务批量标记已读
        match controller.notification_service.mark_notifications_read(
            request.notification_ids.clone(),
            user.user_id,
        ).await {
            Ok(_) => {
                let response = MarkNotificationReadResponse {
                    success: true,
                    message: "通知已标记为已读".to_string(),
                    marked_count: request.notification_ids.len() as u32,
                };

                let api_response = ApiResponse {
                    code: 0,
                    msg: "标记已读成功".to_string(),
                    data: Some(response),
                };

                (StatusCode::OK, Json(api_response))
            }
            Err(err) => {
                let (status_code, error_msg) = match err {
                    crate::domain::DomainError::NotFound { resource } => (StatusCode::NOT_FOUND, format!("{}不存在", resource)),
                    crate::domain::DomainError::ValidationError { reason, .. } => (StatusCode::BAD_REQUEST, reason),
                    _ => (StatusCode::INTERNAL_SERVER_ERROR, "服务器内部错误".to_string()),
                };

                let error_response = ApiResponse {
                    code: if status_code == StatusCode::BAD_REQUEST { 400 } else if status_code == StatusCode::NOT_FOUND { 404 } else { 500 },
                    msg: error_msg,
                    data: None::<()>,
                };

                (status_code, Json(error_response))
            }
        }
    }

    /// POST /api/v1/protected/notifications/unread_count - 获取未读通知数
    pub async fn get_unread_notification_count(
        State(controller): State<Self>,
        user: CurrentUser,
        Json(request): Json<GetUnreadNotificationCountRequest>,
    ) -> impl IntoResponse {
        debug!("用户 {} 获取未读通知数", user.user_id);

        // 调用通知应用服务获取未读通知数量
        match controller.notification_service.get_unread_count(user.user_id).await {
            Ok(total_count) => {
                // TODO: 如果需要按类型统计，需要在Repository中实现相应方法
                let by_type = if request.r#type.is_none() {
                    // 暂时返回None，实际应该查询各类型的未读数量
                    None
                } else {
                    None
                };

                let response = GetUnreadNotificationCountResponse {
                    total_unread: total_count as u32,
                    by_type,
                };

                let api_response = ApiResponse {
                    code: 0,
                    msg: "获取未读通知数成功".to_string(),
                    data: Some(response),
                };

                (StatusCode::OK, Json(api_response))
            }
            Err(err) => {
                let error_response = ApiResponse {
                    code: 500,
                    msg: "服务器内部错误".to_string(),
                    data: None::<()>,
                };

                (StatusCode::INTERNAL_SERVER_ERROR, Json(error_response))
            }
        }
    }

    /// POST /api/v1/protected/notifications/settings - 获取通知设置
    pub async fn get_notification_settings(
        State(controller): State<Self>,
        user: CurrentUser,
        Json(_request): Json<GetNotificationSettingsRequest>,
    ) -> impl IntoResponse {
        debug!("用户 {} 获取通知设置", user.user_id);

        // TODO: 实现获取通知设置逻辑
        // 1. 查询用户的通知偏好设置
        // 2. 返回各类通知的开关状态

        let response = GetNotificationSettingsResponse {
            push_enabled: true,
            email_enabled: false,
            friend_request: true,
            group_invite: true,
            message: true,
            moment_like: true,
            comment: true,
            system: true,
        };

        let api_response = ApiResponse {
            code: 0,
            msg: "获取通知设置成功".to_string(),
            data: Some(response),
        };

        (StatusCode::OK, Json(api_response))
    }

    /// POST /api/v1/protected/notifications/update_settings - 更新通知设置
    pub async fn update_notification_settings(
        State(controller): State<Self>,
        user: CurrentUser,
        Json(request): Json<UpdateNotificationSettingsRequest>,
    ) -> impl IntoResponse {
        debug!("用户 {} 更新通知设置", user.user_id);

        // TODO: 实现更新通知设置逻辑
        // 1. 验证设置参数
        // 2. 更新用户的通知偏好
        // 3. 保存到数据库
        // 4. 可能需要更新推送服务的订阅

        let response = UpdateNotificationSettingsResponse {
            success: true,
            message: "通知设置更新成功".to_string(),
        };

        let api_response = ApiResponse {
            code: 0,
            msg: "通知设置更新成功".to_string(),
            data: Some(response),
        };

        (StatusCode::OK, Json(api_response))
    }
}
