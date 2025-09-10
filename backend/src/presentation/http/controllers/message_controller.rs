/// 消息HTTP控制器 - 新架构
/// 
/// 处理消息相关的HTTP请求，使用新的实体和仓库架构
use axum::extract::State;
use axum::response::Response;
use axum::Json;
use serde::{Deserialize, Serialize};
use crate::utils::JsSafeI64;

use crate::application::MessageApplicationService;
use crate::domain::{DirectMessage, GroupMessage};

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
    
    // ===== 私聊消息API =====
    
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
                request.message_type.unwrap_or_default().into(),
                request.latitude.unwrap_or(35.16),
                request.longitude.unwrap_or(112.68),
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
            // 根据用户ID获取或创建会话
            match controller.message_service.get_pg_repository()
                .get_or_create_conversation(request.user_id.value(), request.peer_id.value())
                .await {
                Ok(conv) => conv.conversation_id,
                Err(error) => return map_domain_error_to_http_response(error),
            }
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
        let result = controller.message_service.get_pg_repository()
            .delete_message(request.message_id.value())
            .await;
        
        match result {
            Ok(_) => ok_response(serde_json::json!({"success": true})),
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
                request.group_id,
                request.sender_id.value(),
                request.content,
                request.message_type.unwrap_or_default().into(),
                request.latitude.unwrap_or(35.16),
                request.longitude.unwrap_or(112.68),
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
            .get_group_messages_by_group(
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
        let result = controller.message_service.get_pg_repository()
            .delete_message(request.message_id.value())
            .await;
        
        match result {
            Ok(_) => ok_response(serde_json::json!({"success": true})),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    // ===== 会话管理API =====
    
    /// 获取会话列表
    pub async fn get_conversations(
        State(controller): State<Self>,
        Json(request): Json<ConversationsRequest>,
    ) -> Response {
        let result = controller.message_service.get_pg_repository()
            .list_conversations(
                request.user_id.value(),
                request.cursor.map(|c| c.value()).unwrap_or(0),
                request.limit.unwrap_or(20),
            )
            .await;
        
        match result {
            Ok((conversations, has_more, next_cursor)) => {
                let responses: Vec<ConversationSummaryResponse> = conversations
                    .into_iter()
                    .map(|conv_summary| ConversationSummaryResponse {
                        conversation_id: conv_summary.conversation_id,
                        peer_id: JsSafeI64::new(conv_summary.peer_id),
                        last_message_preview: conv_summary.last_message_preview,
                        last_message_at: conv_summary.last_message_at.to_rfc3339(),
                    })
                    .collect();
                
                paginated_response(responses, has_more, next_cursor)
            }
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    // ===== 消息状态管理API =====
    
    /// 标记群组消息已读
    pub async fn mark_group_messages_read(
        State(controller): State<Self>,
        Json(request): Json<GroupMarkReadRequest>,
    ) -> Response {
        let result = controller.message_service.get_pg_repository()
            .mark_group_messages_read(
                &request.group_id,
                request.user_id.value(),
                request.last_read_message_id.as_deref(),
            )
            .await;
        
        match result {
            Ok(_) => ok_response(serde_json::json!({"success": true})),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 标记私聊消息已读
    pub async fn mark_user_messages_read(
        State(controller): State<Self>,
        Json(request): Json<UserMarkReadRequest>,
    ) -> Response {
        let result = controller.message_service.get_pg_repository()
            .mark_direct_messages_read(
                &request.conversation_id,
                request.user_id.value(),
                request.last_read_message_id.as_deref(),
            )
            .await;
        
        match result {
            Ok(_) => ok_response(serde_json::json!({"success": true})),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取群组未读消息数
    pub async fn get_group_unread_count(
        State(controller): State<Self>,
        Json(request): Json<GroupUnreadCountRequest>,
    ) -> Response {
        let result = controller.message_service.get_pg_repository()
            .get_group_unread_count(&request.group_id, request.user_id.value())
            .await;
        
        match result {
            Ok(count) => ok_response(serde_json::json!({
                "group_id": request.group_id,
                "unread_count": count
            })),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取私聊未读消息数
    pub async fn get_user_unread_count(
        State(controller): State<Self>,
        Json(request): Json<UserUnreadCountRequest>,
    ) -> Response {
        if let Some(conversation_id) = request.conversation_id {
            // 获取指定会话的未读数
            let result = controller.message_service.get_pg_repository()
                .get_conversation_unread_count(&conversation_id, request.user_id.value())
                .await;
            
            match result {
                Ok(count) => ok_response(serde_json::json!({
                    "conversation_id": conversation_id,
                    "unread_count": count
                })),
                Err(error) => map_domain_error_to_http_response(error),
            }
        } else {
            // 获取用户所有会话的总未读数
            let result = controller.message_service.get_pg_repository()
                .get_user_total_unread_count(request.user_id.value())
                .await;
            
            match result {
                Ok(count) => ok_response(serde_json::json!({
                    "total_unread_count": count
                })),
                Err(error) => map_domain_error_to_http_response(error),
            }
        }
    }
}

// ===== 请求结构体 =====

#[derive(Debug, Deserialize)]
pub struct UserSendRequest {
    pub sender_id: JsSafeI64,
    pub recipient_id: JsSafeI64,
    pub content: String,
    pub message_type: Option<MessageTypeRequest>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
}

#[derive(Debug, Deserialize)]
pub struct GroupSendRequest {
    pub group_id: String,
    pub sender_id: JsSafeI64,
    pub content: String,
    pub message_type: Option<MessageTypeRequest>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
}

#[derive(Debug, Deserialize)]
pub struct UserHistoryRequest {
    pub user_id: JsSafeI64,
    pub peer_id: JsSafeI64,
    pub conversation_id: Option<String>,
    pub cursor: Option<JsSafeI64>,
    pub limit: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct GroupHistoryRequest {
    pub group_id: String,
    pub cursor: Option<JsSafeI64>,
    pub limit: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteRequest {
    pub message_id: JsSafeI64,
}

#[derive(Debug, Deserialize)]
pub struct ConversationsRequest {
    pub user_id: JsSafeI64,
    pub cursor: Option<JsSafeI64>,
    pub limit: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct GroupMarkReadRequest {
    pub group_id: String,
    pub user_id: JsSafeI64,
    pub last_read_message_id: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UserMarkReadRequest {
    pub conversation_id: String,
    pub user_id: JsSafeI64,
    pub last_read_message_id: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GroupUnreadCountRequest {
    pub group_id: String,
    pub user_id: JsSafeI64,
}

#[derive(Debug, Deserialize)]
pub struct UserUnreadCountRequest {
    pub user_id: JsSafeI64,
    pub conversation_id: Option<String>,
}

#[derive(Debug, Deserialize, Default)]
#[serde(rename_all = "lowercase")]
pub enum MessageTypeRequest {
    #[default]
    Text,
    Image,
    System,
}

impl From<MessageTypeRequest> for crate::domain::DirectMessageType {
    fn from(msg_type: MessageTypeRequest) -> Self {
        match msg_type {
            MessageTypeRequest::Text => crate::domain::DirectMessageType::Text,
            MessageTypeRequest::Image => crate::domain::DirectMessageType::Image,
            MessageTypeRequest::System => crate::domain::DirectMessageType::System,
        }
    }
}

impl From<MessageTypeRequest> for crate::domain::GroupMessageType {
    fn from(msg_type: MessageTypeRequest) -> Self {
        match msg_type {
            MessageTypeRequest::Text => crate::domain::GroupMessageType::Text,
            MessageTypeRequest::Image => crate::domain::GroupMessageType::Image,
            MessageTypeRequest::System => crate::domain::GroupMessageType::System,
        }
    }
}

// ===== 响应结构体 =====

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
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
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
            latitude: Some(message.latitude),
            longitude: Some(message.longitude),
        }
    }
}

#[derive(Debug, Serialize)]
pub struct GroupMessageResponse {
    pub message_id: String,
    pub group_id: String,
    pub sender_id: JsSafeI64,
    pub content: String,
    pub message_type: String,
    pub created_at: String,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
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
            latitude: Some(message.latitude),
            longitude: Some(message.longitude),
        }
    }
}

#[derive(Debug, Serialize)]
pub struct ConversationResponse {
    pub conversation_id: String,
    pub user1_id: JsSafeI64,
    pub user2_id: JsSafeI64,
    pub created_at: String,
    pub updated_at: String,
    pub last_message_at: Option<String>,
    pub last_message_preview: Option<String>,
}

impl From<crate::domain::Conversation> for ConversationResponse {
    fn from(conversation: crate::domain::Conversation) -> Self {
        Self {
            conversation_id: conversation.conversation_id,
            user1_id: JsSafeI64::new(conversation.user1_id),
            user2_id: JsSafeI64::new(conversation.user2_id),
            created_at: conversation.created_at.to_rfc3339(),
            updated_at: conversation.updated_at.to_rfc3339(),
            last_message_at: Some(conversation.last_message_at.to_rfc3339()),
            last_message_preview: conversation.last_message_preview,
        }
    }
}

#[derive(Debug, Serialize)]
pub struct ConversationSummaryResponse {
    pub conversation_id: String,
    pub peer_id: JsSafeI64,
    pub last_message_preview: Option<String>,
    pub last_message_at: String,
}
