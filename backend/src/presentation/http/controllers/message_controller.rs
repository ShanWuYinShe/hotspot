/// 统一消息HTTP控制器 - 符合需求文档的统一消息架构
/// 
/// 核心设计理念：
/// - 私聊和群聊都通过group_id统一管理
/// - 私聊也是群组，group_type='private_chat'
/// - 统一的API接口，只有6个消息端点
/// - 完全符合需求文档的XxxRequest/XxxResponse命名规范

use axum::extract::{State, Extension};
use axum::response::Response;
use axum::Json;
use serde::{Deserialize, Serialize};

use crate::application::services::message_application_service::MessageApplicationService;
use super::{ok_response, created_response, map_domain_error_to_http_response, paginated_response};

/// 消息控制器
#[derive(Clone)]
pub struct MessageController {
    message_service: MessageApplicationService,
}

impl MessageController {
    /// 创建消息控制器
    pub fn new(message_service: MessageApplicationService) -> Self {
        Self { message_service }
    }
    
    // ===== 统一消息API（6个端点）=====
    
    /// 发送消息（统一接口）- 符合需求文档的统一消息架构
    /// POST /api/v1/protected/chat/send
    pub async fn send_message(
        State(controller): State<Self>,
        Extension(current_user): Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<SendMessageRequest>,
    ) -> Response {
        let result = controller.message_service
            .send_message_to_group(
                request.group_id,
                current_user.user_id,
                request.content,
                request.message_type.unwrap_or("text".to_string()),
                request.reply_to_message_id,
                request.mentioned_users.unwrap_or_default(),
                request.latitude,
                request.longitude,
            )
            .await;
        
        match result {
            Ok(message) => {
                let response = SendMessageResponse {
                    message_id: message.id.to_string(),
                    group_id: message.group_id.to_string(),
                    sent_at: message.created_at.to_rfc3339(),
                    message: "消息发送成功".to_string(),
                };
                created_response(response)
            },
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取消息历史（统一接口）- 符合需求文档的统一消息架构
    /// POST /api/v1/protected/chat/history
    pub async fn get_message_history(
        State(controller): State<Self>,
        Extension(current_user): Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<GetHistoryRequest>,
    ) -> Response {
        let result = controller.message_service
            .get_group_messages(
                request.group_id,
                current_user.user_id,
                request.cursor,
                request.limit.unwrap_or(50),
                request.include_system_messages.unwrap_or(true),
            )
            .await;
        
        match result {
            Ok(messages) => {
                let response_messages: Vec<GetHistoryResponse> = messages.items
                    .into_iter()
                    .map(|msg| GetHistoryResponse {
                        message_id: msg.id.to_string(),
                        group_id: msg.group_id.to_string(),
                        sender_id: Some(msg.sender_id.to_string()),
                        sender_name: None, // TODO: 需要从用户服务获取发送者姓名
                        content: msg.content,
                        message_type: msg.message_type.to_string(),
                        system_event_type: None, // TODO: 系统事件类型字段
                        reply_to_message_id: msg.reply_to_message_id.map(|id| id.to_string()),
                        is_edited: false, // TODO: 编辑状态字段
                        reaction_count: 0, // TODO: 反应计数字段
                        mentioned_users: vec![], // TODO: 提及用户字段
                        affected_users: vec![], // TODO: 受影响用户字段
                        sent_at: msg.created_at.to_rfc3339(),
                        edited_at: None, // TODO: 编辑时间字段
                        latitude: None, // TODO: 位置纬度字段
                        longitude: None, // TODO: 位置经度字段
                    })
                    .collect();
                
                paginated_response(response_messages, messages.has_more, messages.next_cursor)
            }
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 删除消息（统一接口）- 符合需求文档的统一消息架构
    /// POST /api/v1/protected/chat/delete
    pub async fn delete_message(
        State(controller): State<Self>,
        Extension(current_user): Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<DeleteMessageRequest>,
    ) -> Response {
        let result = controller.message_service
            .delete_message(
                request.message_id,
                current_user.user_id,
                request.delete_for_everyone.unwrap_or(false),
            )
            .await;
        
        match result {
            Ok(_) => {
                let response = DeleteMessageResponse {
                    success: true,
                    message: "消息删除成功".to_string(),
                };
                ok_response(response)
            },
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 标记消息已读（统一接口）- 符合需求文档的统一消息架构
    /// POST /api/v1/protected/chat/mark-read
    pub async fn mark_messages_read(
        State(controller): State<Self>,
        Extension(current_user): Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<MarkReadRequest>,
    ) -> Response {
        let result = controller.message_service
            .mark_group_messages_read(
                request.group_id,
                current_user.user_id,
                request.message_id,
            )
            .await;
        
        match result {
            Ok(_) => {
                let response = MarkReadResponse {
                    success: true,
                    message: "标记已读成功".to_string(),
                };
                ok_response(response)
            },
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取未读消息数（统一接口）- 符合需求文档的统一消息架构
    /// POST /api/v1/protected/chat/unread_count
    pub async fn get_unread_count(
        State(controller): State<Self>,
        Extension(current_user): Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<UnreadCountRequest>,
    ) -> Response {
        let result = if let Some(group_id) = request.group_id {
            // 获取指定群组的未读数
            controller.message_service
                .get_group_unread_count(group_id, current_user.user_id)
                .await
                .map(|count| UnreadCountResponse {
                    total_unread: count,
                    groups: None,
                })
        } else {
            // 获取所有群组的未读数
            controller.message_service
                .get_all_groups_unread_count(current_user.user_id)
                .await
        };
        
        match result {
            Ok(response) => ok_response(response),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取会话列表（统一接口）- 符合需求文档的统一消息架构
    /// POST /api/v1/protected/chat/conversations
    pub async fn get_conversations(
        State(controller): State<Self>,
        Extension(current_user): Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<ConversationsRequest>,
    ) -> Response {
        let result = controller.message_service
            .get_user_groups(
                current_user.user_id,
                request.cursor,
                request.limit.unwrap_or(20),
                request.include_archived.unwrap_or(false),
            )
            .await;
        
        match result {
            Ok(conversations) => {
                let response_conversations: Vec<ConversationsResponse> = conversations.items
                    .into_iter()
                    .map(|group| ConversationsResponse {
                        group_id: group.group_id,
                        group_name: group.name,
                        group_type: group.group_type,
                        group_avatar: group.avatar_url,
                        last_message: group.last_message.map(|msg| LastMessageResponse {
                            message_id: msg.message_id,
                            sender_id: msg.sender_id,
                            sender_name: msg.sender_name,
                            content: msg.content,
                            message_type: msg.message_type,
                            sent_at: msg.sent_at,
                        }),
                        unread_count: group.unread_count,
                        last_activity_at: group.last_activity_at,
                        is_muted: group.is_muted,
                        is_archived: group.is_archived,
                        member_count: group.member_count,
                    })
                    .collect();
                
                paginated_response(response_conversations, conversations.has_more, conversations.next_cursor)
            }
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
}

// ===== API类型定义（符合XxxRequest/XxxResponse命名规范）=====

/// 发送消息请求 - 符合需求文档的统一消息架构
#[derive(Debug, Deserialize)]
pub struct SendMessageRequest {
    pub group_id: String,        // 统一使用group_id，私聊也是群组
    pub content: String,
    pub message_type: Option<String>,
    pub reply_to_message_id: Option<String>,
    pub mentioned_users: Option<Vec<String>>,
    pub latitude: Option<f64>,   // 位置消息
    pub longitude: Option<f64>,  // 位置消息
}

/// 获取消息历史请求 - 符合需求文档的统一消息架构
#[derive(Debug, Deserialize)]
pub struct GetHistoryRequest {
    pub group_id: String,        // 统一使用group_id
    pub limit: Option<i32>,
    pub cursor: Option<String>,
    pub include_system_messages: Option<bool>,
}

/// 删除消息请求 - 符合需求文档命名规范
#[derive(Debug, Deserialize)]
pub struct DeleteMessageRequest {
    pub message_id: String,
    pub delete_for_everyone: Option<bool>,
}

/// 标记消息已读请求 - 符合需求文档命名规范
#[derive(Debug, Deserialize)]
pub struct MarkReadRequest {
    pub group_id: String,        // 统一使用group_id
    pub message_id: Option<String>, // 最后已读消息ID，不提供则标记所有为已读
}

/// 获取未读消息数请求 - 符合需求文档命名规范
#[derive(Debug, Deserialize)]
pub struct UnreadCountRequest {
    pub group_id: Option<String>, // 可选，不提供则获取所有群组的未读数
}

/// 获取会话列表请求 - 符合需求文档命名规范
#[derive(Debug, Deserialize)]
pub struct ConversationsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
    pub include_archived: Option<bool>,
}

/// 发送消息响应 - 符合需求文档命名规范
#[derive(Debug, Serialize)]
pub struct SendMessageResponse {
    pub message_id: String,
    pub group_id: String,
    pub sent_at: String,
    pub message: String,
}

/// 删除消息响应 - 符合需求文档命名规范
pub type DeleteMessageResponse = crate::presentation::http::models::common_responses::DeleteResponse;

/// 标记已读响应 - 符合需求文档命名规范
pub type MarkReadResponse = crate::presentation::http::models::common_responses::MarkResponse;

/// 消息历史响应 - 符合需求文档命名规范和统一架构
#[derive(Debug, Serialize)]
pub struct GetHistoryResponse {
    pub message_id: String,
    pub group_id: String,           // 统一使用group_id
    pub sender_id: Option<String>,  // 系统消息可为null
    pub sender_name: Option<String>,
    pub content: String,
    pub message_type: String,       // text|image|video|audio|file|location|system
    pub system_event_type: Option<String>, // 系统事件类型
    pub reply_to_message_id: Option<String>,
    pub is_edited: bool,
    pub reaction_count: i32,
    pub mentioned_users: Vec<String>,
    pub affected_users: Vec<String>, // 系统消息用
    pub sent_at: String,            // 发送时间
    pub edited_at: Option<String>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
}

/// 会话列表响应 - 符合需求文档命名规范
#[derive(Debug, Serialize)]
pub struct ConversationsResponse {
    pub group_id: String,           // 统一使用group_id
    pub group_name: String,
    pub group_type: String,         // private_chat|public_chat
    pub group_avatar: Option<String>,
    pub last_message: Option<LastMessageResponse>,
    pub unread_count: i32,
    pub last_activity_at: String,
    pub is_muted: bool,
    pub is_archived: bool,
    pub member_count: i32,
}

/// 最后一条消息信息
#[derive(Debug, Serialize)]
pub struct LastMessageResponse {
    pub message_id: String,
    pub sender_id: String,
    pub sender_name: String,
    pub content: String,
    pub message_type: String,
    pub sent_at: String,
}

/// 未读消息数响应 - 符合需求文档命名规范
#[derive(Debug, Serialize)]
pub struct UnreadCountResponse {
    pub total_unread: i32,
    pub groups: Option<Vec<GroupUnreadCountResponse>>,
}

/// 各群组未读数
#[derive(Debug, Serialize)]
pub struct GroupUnreadCountResponse {
    pub group_id: String,
    pub unread_count: i32,
}
