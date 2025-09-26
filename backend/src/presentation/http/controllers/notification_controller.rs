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

use axum::{extract::{State, Extension}, http::StatusCode, response::IntoResponse, Json};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tracing::debug;
use utoipa::ToSchema;

use crate::application::services::notification_application_service::NotificationApplicationService;
use crate::presentation::http::models::ApiResponse;
use crate::presentation::http::auth::CurrentUser;
use crate::adapters::adapters_pg::notification_repository::NotificationPriority;

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
        State(controller): State<NotificationController>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<SendNotificationRequest>,
    ) -> axum::response::Response {
        debug!(
            "用户 {} 发送通知给 {} 个用户，类型：{}",
            user.user_id,
            request.user_ids.len(),
            request.r#type
        );

        // 解析通知类型
        let notification_type = match request.r#type.parse() {
            Ok(nt) => nt,
            Err(_) => {
                let api_response = ApiResponse {
                    code: 400,
                    msg: "无效的通知类型".to_string(),
                    data: None::<()>,
                };
                return (StatusCode::OK, Json(api_response)).into_response();
            }
        };

        // 解析用户ID列表
        let user_ids: Vec<i64> = request.user_ids
            .iter()
            .filter_map(|id| id.parse().ok())
            .collect();

        if user_ids.is_empty() {
            let api_response = ApiResponse {
                code: 400,
                msg: "无效的用户ID列表".to_string(),
                data: None::<()>,
            };
            return (StatusCode::OK, Json(api_response)).into_response();
        }

        // 调用应用服务批量发送通知
        match controller.notification_service.send_bulk_notifications(
            user_ids,
            notification_type,
            request.title,
            request.content,
            request.data,
            NotificationPriority::Normal,
            Some(user.user_id),
        ).await {
            Ok(notifications) => {
                let response = SendNotificationResponse {
                    success: true,
                    message: "通知发送成功".to_string(),
                    sent_count: notifications.len() as u32,
                };

                let api_response = ApiResponse {
                    code: 0,
                    msg: "通知发送成功".to_string(),
                    data: Some(response),
                };

                (StatusCode::OK, Json(api_response)).into_response()
            },
            Err(err) => {
                debug!("发送通知失败: {:?}", err);
                let api_response = ApiResponse {
                    code: 5000,
                    msg: "通知发送失败".to_string(),
                    data: None::<()>,
                };
                (StatusCode::OK, Json(api_response)).into_response()
            }
        }
    }

    /// POST /api/v1/protected/notifications/list - 获取通知列表
    pub async fn get_notifications(
        State(controller): State<NotificationController>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<GetNotificationsRequest>,
    ) -> axum::response::Response {
        debug!("用户 {} 获取通知列表", user.user_id);

        // TODO: 实现获取通知列表逻辑
        // 1. 查询用户的通知记录
        // 2. 根据类型和已读状态过滤
        // 3. 分页处理
        // 4. 返回通知详情

        // 解析状态过滤条件
        let status_filter = match request.is_read {
            Some(true) => Some(crate::adapters::adapters_pg::notification_repository::NotificationStatus::Read),
            Some(false) => Some(crate::adapters::adapters_pg::notification_repository::NotificationStatus::Unread),
            None => None,
        };

        // 解析游标
        let cursor = request
            .cursor
            .as_deref()
            .and_then(|s| s.parse::<i64>().ok())
            .unwrap_or(0);
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
                        data: notification.data,
                        is_read: matches!(notification.status, crate::adapters::adapters_pg::notification_repository::NotificationStatus::Read),
                        created_at: notification.created_at.to_rfc3339(),
                        read_at: notification.read_at.map(|t| t.to_rfc3339()),
                    })
                    .collect();

                // 判断是否还有更多数据
                let has_more = notifications.len() == limit as usize;
                let next_cursor = if has_more && !notifications.is_empty() {
                    // 使用最后一个通知的ID作为游标
                    Some(notifications.last().unwrap().notification_id.clone())
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

                (StatusCode::OK, Json(api_response)).into_response()
            }
            Err(err) => crate::presentation::http::controllers::map_domain_error_to_http_response(err),
        }
    }

    /// POST /api/v1/protected/notifications/mark_read - 标记通知已读
    pub async fn mark_notification_read(
        State(controller): State<NotificationController>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<MarkNotificationReadRequest>,
    ) -> axum::response::Response {
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

                (StatusCode::OK, Json(api_response)).into_response()
            }
            Err(err) => crate::presentation::http::controllers::map_domain_error_to_http_response(err),
        }
    }

    /// POST /api/v1/protected/notifications/unread_count - 获取未读通知数
    pub async fn get_unread_notification_count(
        State(controller): State<NotificationController>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<GetUnreadNotificationCountRequest>,
    ) -> axum::response::Response {
        debug!("用户 {} 获取未读通知数", user.user_id);

        // 调用通知应用服务获取未读通知数量
        match controller.notification_service.get_unread_notification_count(user.user_id).await {
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

                (StatusCode::OK, Json(api_response)).into_response()
            }
            Err(err) => crate::presentation::http::controllers::map_domain_error_to_http_response(err),
        }
    }

    /// POST /api/v1/protected/notifications/settings - 获取通知设置
    pub async fn get_notification_settings(
        State(controller): State<NotificationController>,
        Extension(user): Extension<CurrentUser>,
        Json(_request): Json<GetNotificationSettingsRequest>,
    ) -> axum::response::Response {
        debug!("用户 {} 获取通知设置", user.user_id);

        // 调用通知应用服务获取通知设置
        match controller.notification_service.get_notification_settings(user.user_id).await {
            Ok(settings) => {
                let response = GetNotificationSettingsResponse {
                    push_enabled: settings.push_enabled,
                    email_enabled: settings.email_enabled,
                    friend_request: settings.friend_requests,
                    group_invite: settings.group_invites,
                    message: settings.messages,
                    moment_like: settings.moment_interactions,
                    comment: settings.moment_interactions, // 评论与瞬间交互共享设置
                    system: settings.system_notifications,
                };

                let api_response = ApiResponse {
                    code: 0,
                    msg: "获取通知设置成功".to_string(),
                    data: Some(response),
                };

                (StatusCode::OK, Json(api_response)).into_response()
            },
            Err(error) => crate::presentation::http::controllers::map_domain_error_to_http_response(error),
        }
    }

    /// POST /api/v1/protected/notifications/update_settings - 更新通知设置
    pub async fn update_notification_settings(
        State(controller): State<NotificationController>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<UpdateNotificationSettingsRequest>,
    ) -> axum::response::Response {
        debug!("用户 {} 更新通知设置", user.user_id);

        // 构建更新的通知设置
        use crate::adapters::adapters_pg::notification_repository::NotificationSettings;
        
        let updated_settings = NotificationSettings {
            user_id: user.user_id,
            push_enabled: request.push_enabled.unwrap_or(true),
            email_enabled: request.email_enabled.unwrap_or(false),
            friend_requests: request.friend_request.unwrap_or(true),
            group_invites: request.group_invite.unwrap_or(true),
            messages: request.message.unwrap_or(true),
            moment_interactions: request.moment_like.unwrap_or(true),
            system_notifications: request.system.unwrap_or(true),
            quiet_hours_enabled: false, // 静默时间功能暂未在API中暴露
            quiet_hours_start: None,
            quiet_hours_end: None,
            created_at: chrono::Utc::now(),
            updated_at: chrono::Utc::now(),
        };

        // 调用通知应用服务更新设置
        match controller.notification_service.update_notification_settings(
            user.user_id,
            updated_settings,
        ).await {
            Ok(_) => {
                let response = UpdateNotificationSettingsResponse {
                    success: true,
                    message: "通知设置更新成功".to_string(),
                };

                let api_response = ApiResponse {
                    code: 0,
                    msg: "通知设置更新成功".to_string(),
                    data: Some(response),
                };

                (StatusCode::OK, Json(api_response)).into_response()
            }
            Err(error) => crate::presentation::http::controllers::map_domain_error_to_http_response(error),
        }
    }
}
