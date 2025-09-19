/// 统一消息HTTP控制器 - 统一会话架构
/// 
/// 处理统一消息相关的HTTP请求，群聊和私聊使用相同的底层实现
/// 核心设计理念：
/// - 群聊和私聊都是"会话"，区别仅在于可见性和访问权限
/// - 私聊也需要对应的群组（visibility='private_chat'）
/// - 统一的API接口，前端通过conversation_type区分显示
use axum::extract::State;
use axum::response::Response;
use axum::Json;
use serde::{Deserialize, Serialize};
use crate::utils::JsSafeI64;

use crate::application::services::message_application_service::MessageApplicationService;
use crate::domain::{DirectMessage, GroupMessage};
use crate::domain::entities::{DirectMessageType, GroupMessageType};

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
    
    /// 发送消息（统一接口）
    /// 支持群聊和私聊，通过conversation_id区分
    pub async fn send_message(
        State(controller): State<Self>,
        Json(request): Json<UnifiedSendRequest>,
    ) -> Response {
        // 根据conversation_id发送消息
        // 底层服务会自动处理群聊和私聊的逻辑差异
        let result = controller.message_service
            .send_message_to_conversation(
                request.conversation_id,
                request.sender_id.value(),
                request.content,
                request.message_type.unwrap_or_default(),
                request.latitude,
                request.longitude,
            )
            .await;
        
        match result {
            Ok(message) => created_response(UnifiedMessageResponse::from(message)),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取消息历史（统一接口）
    pub async fn get_message_history(
        State(controller): State<Self>,
        Json(request): Json<UnifiedHistoryRequest>,
    ) -> Response {
        let result = controller.message_service
            .get_conversation_messages(
                request.conversation_id,
                request.user_id.value(),
                request.cursor.map(|c| c.value()).unwrap_or(0),
                request.limit.unwrap_or(20),
            )
            .await;
        
        match result {
            Ok(messages) => {
                let response_messages: Vec<UnifiedMessageResponse> = messages.items
                    .into_iter()
                    .map(UnifiedMessageResponse::from)
                    .collect();
                
                paginated_response(crate::application::PaginatedResult {
                    items: response_messages,
                    has_more: messages.has_more,
                    next_cursor: messages.next_cursor.map(JsSafeI64::new),
                })
            }
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 删除消息（统一接口）
    pub async fn delete_message(
        State(controller): State<Self>,
        Json(request): Json<UnifiedDeleteRequest>,
    ) -> Response {
        let result = controller.message_service
            .delete_message(
                request.message_id,
                request.user_id.value(),
            )
            .await;
        
        match result {
            Ok(_) => ok_response("消息删除成功"),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 标记消息已读（统一接口）
    pub async fn mark_messages_read(
        State(controller): State<Self>,
        Json(request): Json<UnifiedMarkReadRequest>,
    ) -> Response {
        let result = controller.message_service
            .mark_conversation_messages_read(
                request.conversation_id,
                request.user_id.value(),
                request.message_id,
            )
            .await;
        
        match result {
            Ok(_) => ok_response("消息已标记为已读"),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取未读消息数（统一接口）
    pub async fn get_unread_count(
        State(controller): State<Self>,
        Json(request): Json<UnifiedUnreadCountRequest>,
    ) -> Response {
        let result = controller.message_service
            .get_conversation_unread_count(
                request.conversation_id,
                request.user_id.value(),
            )
            .await;
        
        match result {
            Ok(count) => ok_response(UnreadCountResponse { unread_count: count }),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取会话列表
    pub async fn get_conversations(
        State(controller): State<Self>,
        Json(request): Json<ConversationsRequest>,
    ) -> Response {
        let result = controller.message_service
            .find_conversations_by_user(
                request.user_id.value(),
                request.cursor.map(|c| c.value()).unwrap_or(0),
                request.limit.unwrap_or(20),
            )
            .await;
        
        match result {
            Ok(conversations) => {
                let response_conversations: Vec<ConversationResponse> = conversations.items
                    .into_iter()
                    .map(ConversationResponse::from)
                    .collect();
                
                paginated_response(crate::application::PaginatedResult {
                    items: response_conversations,
                    has_more: conversations.has_more,
                    next_cursor: conversations.next_cursor.map(JsSafeI64::new),
                })
            }
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    // ===== 兼容性API（保留旧接口，内部调用统一接口）=====
    
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

/// 标记消息已读请求
#[derive(Debug, Deserialize)]
pub struct MarkReadRequest {
    pub user_id: JsSafeI64,
    pub group_id: Option<String>,
    pub conversation_id: Option<String>,
    pub last_read_message_id: Option<JsSafeI64>,
}

/// 获取未读消息数请求
#[derive(Debug, Deserialize)]
pub struct UnreadCountRequest {
    pub user_id: JsSafeI64,
    pub group_id: Option<String>,
    pub conversation_id: Option<String>,
}

/// 获取会话列表请求
#[derive(Debug, Deserialize)]
pub struct ConversationsRequest {
    pub user_id: JsSafeI64,
    pub cursor: Option<JsSafeI64>,
    pub limit: Option<i32>,
}

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

/// 统一发送消息请求
#[derive(Debug, Deserialize)]
pub struct UnifiedSendRequest {
    pub conversation_id: String,
    pub sender_id: JsSafeI64,
    pub content: String,
    pub message_type: Option<String>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
}

/// 统一消息历史请求
#[derive(Debug, Deserialize)]
pub struct UnifiedHistoryRequest {
    pub conversation_id: String,
    pub user_id: JsSafeI64,
    pub cursor: Option<JsSafeI64>,
    pub limit: Option<i32>,
}

/// 统一删除消息请求
#[derive(Debug, Deserialize)]
pub struct UnifiedDeleteRequest {
    pub message_id: String,
    pub user_id: JsSafeI64,
}

/// 统一标记已读请求
#[derive(Debug, Deserialize)]
pub struct UnifiedMarkReadRequest {
    pub conversation_id: String,
    pub user_id: JsSafeI64,
    pub message_id: Option<String>, // 如果为None，标记所有消息为已读
}

/// 统一未读计数请求
#[derive(Debug, Deserialize)]
pub struct UnifiedUnreadCountRequest {
    pub conversation_id: String,
    pub user_id: JsSafeI64,
}


/// 统一消息响应
#[derive(Debug, Serialize)]
pub struct UnifiedMessageResponse {
    pub message_id: String,
    pub conversation_id: String,
    pub conversation_type: String, // 'group' 或 'private'
    pub sender_id: JsSafeI64,
    pub sender_name: Option<String>,
    pub content: String,
    pub message_type: String,
    pub created_at: String,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
}

/// 会话响应
#[derive(Debug, Serialize)]
pub struct ConversationResponse {
    pub conversation_id: String,
    pub conversation_type: String, // 'group' 或 'private'
    pub group_id: String,
    pub name: Option<String>,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub participant_count: i32,
    pub unread_count: i32,
    pub last_message_preview: Option<String>,
    pub last_message_at: Option<String>,
    pub created_at: String,
}

/// 未读计数响应
#[derive(Debug, Serialize)]
pub struct UnreadCountResponse {
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