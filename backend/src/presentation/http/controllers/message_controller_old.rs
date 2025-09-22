/// 统一消息HTTP控制器 - 统一会话架构
/// 
/// 处理统一消息相关的HTTP请求，群聊和私聊使用相同的底层实现
/// 核心设计理念：
/// - 群聊和私聊都是"会话"，区别仅在于可见性和访问权限
/// - 私聊也需要对应的群组（visibility='private_chat'）
/// - 统一的API接口，前端通过conversation_type区分显示
use axum::extract::{State, Extension};
use axum::response::Response;
use axum::Json;
use serde::{Deserialize, Serialize};
use crate::utils::JsSafeI64;

use crate::application::services::message_application_service::MessageApplicationService;
// 使用统一的Message实体，符合需求文档的统一消息架构
// 消息类型使用字符串表示，不需要特定的枚举类型

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
    
    // ===== 统一消息API =====
    
    /// 发送消息（统一接口）- 符合需求文档的统一消息架构
    /// 私聊和群聊都通过group_id统一管理
    pub async fn send_message(
        State(controller): State<Self>,
        Extension(current_user): Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<SendMessageRequest>,
    ) -> Response {
        // 使用统一的group_id架构，私聊也是群组
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
                    message_id: message.message_id,
                    group_id: message.group_id,
                    sent_at: message.created_at.to_rfc3339(),
                    message: "消息发送成功".to_string(),
                };
                created_response(response)
            },
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取消息历史（统一接口）- 符合需求文档的统一消息架构
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
                        message_id: msg.message_id,
                        group_id: msg.group_id,
                        sender_id: msg.sender_id.map(|id| id.to_string()),
                        sender_name: msg.sender_name,
                        content: msg.content,
                        message_type: msg.message_type,
                        system_event_type: msg.system_event_type,
                        reply_to_message_id: msg.reply_to_message_id,
                        is_edited: msg.is_edited,
                        reaction_count: msg.reaction_count,
                        mentioned_users: msg.mentioned_users,
                        affected_users: msg.affected_users,
                        sent_at: msg.created_at.to_rfc3339(),
                        edited_at: msg.edited_at.map(|dt| dt.to_rfc3339()),
                        latitude: msg.latitude,
                        longitude: msg.longitude,
                    })
                    .collect();
                
                paginated_response(response_messages, messages.has_more, messages.next_cursor)
            }
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 删除消息（统一接口）- 符合需求文档的统一消息架构
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
                        last_message: group.last_message.map(|msg| LastMessage {
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
    
    // ===== 兼容性API已移除，请使用统一消息接口 =====
    
    /// 发送私聊消息
    pub async fn send_direct_message(
        State(controller): State<Self>,
        Json(request): Json<UserSendRequest>,
    ) -> Response {
        let result = controller.message_service
            .create_direct_message(
                request.sender_id.value(),
                request.recipient_id.value(),
                request.content,
                request.message_type.unwrap_or_default(),
                request.latitude.unwrap_or(0.0),
                request.longitude.unwrap_or(0.0),
            )
            .await;
        
        match result {
            Ok(message) => created_response(DirectMessageResponse::from(message)),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取私聊消息历史
    pub async fn get_direct_messages(
        State(controller): State<Self>,
        Json(request): Json<UserHistoryRequest>,
    ) -> Response {
        let conversation_id = if let Some(id) = request.conversation_id {
            id
        } else {
            // 如果没有提供conversation_id，需要通过用户ID获取或创建会话
            // 这里暂时返回错误，实际应该实现获取会话ID的逻辑
            return map_domain_error_to_http_response(
                crate::domain::DomainError::BusinessRuleViolation { 
                    rule: "conversation_id is required".to_string() 
                }
            );
        };
        
        let result = controller.message_service
            .get_direct_messages_by_conversation(
                &conversation_id,
                request.cursor.map(|c| c.value()).unwrap_or(0),
                request.limit.unwrap_or(20),
            )
            .await;
        
        match result {
            Ok(paginated) => {
                let items: Vec<DirectMessageResponse> = paginated.items
                    .into_iter()
                    .map(DirectMessageResponse::from)
                    .collect();
                paginated_response(items, paginated.has_more, paginated.next_cursor)
            }
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 删除私聊消息
    pub async fn delete_direct_message(
        State(controller): State<Self>,
        Json(request): Json<DeleteRequest>,
    ) -> Response {
        let result = controller.message_service
            .delete_message(request.message_id.value(), request.user_id.value())
            .await;
        
        match result {
            Ok(()) => ok_response(serde_json::json!({"success": true})),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    // ===== 群组消息API =====
    
    /// 发送群组消息
    pub async fn send_group_message(
        State(controller): State<Self>,
        Json(request): Json<GroupSendRequest>,
    ) -> Response {
        let result = controller.message_service
            .create_group_message(
                request.sender_id.value(),
                request.group_id,
                request.content,
                request.message_type.unwrap_or_default(),
                request.latitude.unwrap_or(0.0),
                request.longitude.unwrap_or(0.0),
            )
            .await;
        
        match result {
            Ok(message) => created_response(GroupMessageResponse::from(message)),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取群组消息历史
    pub async fn get_group_messages(
        State(controller): State<Self>,
        Json(request): Json<GroupHistoryRequest>,
    ) -> Response {
        let result = controller.message_service
            .get_group_messages(
                &request.group_id,
                request.cursor.map(|c| c.value()).unwrap_or(0),
                request.limit.unwrap_or(20),
            )
            .await;
        
        match result {
            Ok(paginated) => {
                let items: Vec<GroupMessageResponse> = paginated.items
                    .into_iter()
                    .map(GroupMessageResponse::from)
                    .collect();
                paginated_response(items, paginated.has_more, paginated.next_cursor)
            }
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 删除群组消息
    pub async fn delete_group_message(
        State(controller): State<Self>,
        Json(request): Json<DeleteRequest>,
    ) -> Response {
        let result = controller.message_service
            .delete_message(request.message_id.value(), request.user_id.value())
            .await;
        
        match result {
            Ok(()) => ok_response(serde_json::json!({"success": true})),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 标记群组消息已读
    pub async fn mark_group_messages_read(
        State(controller): State<Self>,
        Json(request): Json<MarkReadRequest>,
    ) -> Response {
        let group_id = match &request.group_id {
            Some(id) => id,
            None => return map_domain_error_to_http_response(
                crate::domain::DomainError::BusinessRuleViolation {
                    rule: "群组ID不能为空".to_string(),
                }
            ),
        };
        
        match controller.message_service.mark_group_messages_read(
            request.user_id.into(),
            group_id,
            request.last_read_message_id.as_ref().map(|id| id.to_string()).as_deref(),
        ).await {
            Ok(_) => ok_response(serde_json::json!({"success": true})),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取群组未读消息数
    pub async fn get_group_unread_count(
        State(controller): State<Self>,
        Json(request): Json<UnreadCountRequest>,
    ) -> Response {
        match controller.message_service.get_group_unread_count(
            request.user_id.into(),
            request.group_id.as_deref(),
        ).await {
            Ok(count) => ok_response(serde_json::json!({"unread_count": count})),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 标记私聊消息已读
    pub async fn mark_direct_messages_read(
        State(controller): State<Self>,
        Json(request): Json<MarkReadRequest>,
    ) -> Response {
        let conversation_id = match &request.conversation_id {
            Some(id) => id,
            None => return map_domain_error_to_http_response(
                crate::domain::DomainError::BusinessRuleViolation {
                    rule: "会话ID不能为空".to_string(),
                }
            ),
        };
        
        match controller.message_service.mark_direct_messages_read(
            request.user_id.into(),
            conversation_id,
            request.last_read_message_id.as_ref().map(|id| id.to_string()).as_deref(),
        ).await {
            Ok(_) => ok_response(serde_json::json!({"success": true})),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取私聊未读消息数
    pub async fn get_direct_unread_count(
        State(controller): State<Self>,
        Json(request): Json<UnreadCountRequest>,
    ) -> Response {
        match controller.message_service.get_direct_unread_count(
            request.user_id.into(),
            request.conversation_id.as_deref(),
        ).await {
            Ok(count) => ok_response(serde_json::json!({"unread_count": count})),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
}

// ===== 请求和响应数据结构 =====

/// 发送私聊消息请求
#[derive(Debug, Deserialize)]
pub struct UserSendRequest {
    pub sender_id: JsSafeI64,
    pub recipient_id: JsSafeI64,
    pub content: String,
    pub message_type: Option<DirectMessageType>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
}

/// 获取私聊消息历史请求
#[derive(Debug, Deserialize)]
pub struct UserHistoryRequest {
    pub conversation_id: Option<String>,
    pub cursor: Option<JsSafeI64>,
    pub limit: Option<i32>,
}

/// 发送群组消息请求
#[derive(Debug, Deserialize)]
pub struct GroupSendRequest {
    pub sender_id: JsSafeI64,
    pub group_id: String,
    pub content: String,
    pub message_type: Option<GroupMessageType>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
}

/// 获取群组消息历史请求
#[derive(Debug, Deserialize)]
pub struct GroupHistoryRequest {
    pub group_id: String,
    pub cursor: Option<JsSafeI64>,
    pub limit: Option<i32>,
}

/// 删除消息请求
#[derive(Debug, Deserialize)]
pub struct DeleteRequest {
    pub message_id: JsSafeI64,
    pub user_id: JsSafeI64,
}

// 旧的类型定义已移除，使用新的统一架构类型

/// 私聊消息响应
#[derive(Debug, Serialize)]
pub struct DirectMessageResponse {
    pub message_id: String,
    pub conversation_id: String,
    pub sender_id: JsSafeI64,
    pub recipient_id: JsSafeI64,
    pub content: String,
    pub message_type: String,
    pub created_at: String,
    pub read_at: Option<String>,
    pub latitude: f64,
    pub longitude: f64,
}

impl From<DirectMessage> for DirectMessageResponse {
    fn from(message: DirectMessage) -> Self {
        Self {
            message_id: message.message_id,
            conversation_id: message.conversation_id,
            sender_id: JsSafeI64::new(message.sender_id),
            recipient_id: JsSafeI64::new(message.recipient_id),
            content: message.content,
            message_type: message.message_type.as_str().to_string(),
            created_at: message.created_at.to_rfc3339(),
            read_at: message.read_at.map(|dt| dt.to_rfc3339()),
            latitude: message.latitude,
            longitude: message.longitude,
        }
    }
}

/// 群组消息响应
#[derive(Debug, Serialize)]
pub struct GroupMessageResponse {
    pub message_id: String,
    pub group_id: String,
    pub sender_id: JsSafeI64,
    pub content: String,
    pub message_type: String,
    pub created_at: String,
    pub latitude: f64,
    pub longitude: f64,
}

impl From<GroupMessage> for GroupMessageResponse {
    fn from(message: GroupMessage) -> Self {
        Self {
            message_id: message.message_id,
            group_id: message.group_id,
            sender_id: JsSafeI64::new(message.sender_id),
            content: message.content,
            message_type: message.message_type.as_str().to_string(),
            created_at: message.created_at.to_rfc3339(),
            latitude: message.latitude,
            longitude: message.longitude,
        }
    }
}

// ===== 统一消息API数据结构 =====

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
#[derive(Debug, Serialize)]
pub struct DeleteMessageResponse {
    pub success: bool,
    pub message: String,
}

/// 标记已读响应 - 符合需求文档命名规范
#[derive(Debug, Serialize)]
pub struct MarkReadResponse {
    pub success: bool,
    pub message: String,
}


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
    pub last_message: Option<LastMessage>,
    pub unread_count: i32,
    pub last_activity_at: String,
    pub is_muted: bool,
    pub is_archived: bool,
    pub member_count: i32,
}

/// 最后一条消息信息
#[derive(Debug, Serialize)]
pub struct LastMessage {
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
    pub groups: Option<Vec<GroupUnreadCount>>,
}

/// 各群组未读数
#[derive(Debug, Serialize)]
pub struct GroupUnreadCount {
    pub group_id: String,
    pub unread_count: i32,
}

// 实现转换方法
impl From<DirectMessage> for UnifiedMessageResponse {
    fn from(message: DirectMessage) -> Self {
        Self {
            message_id: message.message_id,
            conversation_id: message.conversation_id.unwrap_or_default(),
            conversation_type: "private".to_string(),
            sender_id: JsSafeI64::new(message.sender_id),
            sender_name: None, // 需要从服务层获取
            content: message.content,
            message_type: message.message_type.as_str().to_string(),
            created_at: message.created_at.to_rfc3339(),
            latitude: message.latitude,
            longitude: message.longitude,
        }
    }
}

impl From<GroupMessage> for UnifiedMessageResponse {
    fn from(message: GroupMessage) -> Self {
        Self {
            message_id: message.message_id,
            conversation_id: message.conversation_id.unwrap_or_default(),
            conversation_type: "group".to_string(),
            sender_id: JsSafeI64::new(message.sender_id),
            sender_name: None, // 需要从服务层获取
            content: message.content,
            message_type: message.message_type.as_str().to_string(),
            created_at: message.created_at.to_rfc3339(),
            latitude: message.latitude,
            longitude: message.longitude,
        }
    }
}

impl From<crate::domain::Conversation> for ConversationResponse {
    fn from(conversation: crate::domain::Conversation) -> Self {
        Self {
            conversation_id: format!("conv_{}", conversation.id), // 临时转换，需要实际的conversation_id
            conversation_type: "private".to_string(), // 需要从数据库获取实际类型
            group_id: format!("group_{}", conversation.user1_id), // 临时转换，需要实际的group_id
            name: None,
            description: None,
            avatar_url: None,
            participant_count: 2, // 私聊默认2人
            unread_count: 0, // 需要从服务层获取
            last_message_preview: conversation.last_message_preview,
            last_message_at: conversation.last_message_at.map(|t| t.to_rfc3339()),
            created_at: conversation.created_at.to_rfc3339(),
        }
    }
}