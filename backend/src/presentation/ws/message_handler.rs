/// WebSocket消息处理器
/// 
/// 负责处理不同类型的WebSocket消息
use crate::presentation::ws::{WSMessageType, MessageData, WSEventType};

/// WebSocket消息处理器
pub struct WebSocketMessageHandler;

impl WebSocketMessageHandler {
    /// 创建用户上线消息
    pub fn create_user_online_message(user_id: i64, nickname: String) -> WSEventType {
        WSEventType::UserOnline { user_id, nickname }
    }
    
    /// 创建用户下线消息
    pub fn create_user_offline_message(user_id: i64) -> WSEventType {
        WSEventType::UserOffline { user_id }
    }
    
    /// 创建新消息通知
    pub fn create_message_notification(group_id: String, message: MessageData) -> WSEventType {
        WSEventType::NewMessage { group_id, message }
    }
    
    /// 创建群组信息更新通知
    pub fn create_group_update_notification(group_id: String, name: String) -> WSEventType {
        WSEventType::GroupInfoUpdated { group_id, name }
    }
    
    /// 创建用户加入群组通知
    pub fn create_user_joined_group_message(user_id: i64, group_id: String, nickname: String) -> WSEventType {
        WSEventType::UserJoinedGroup { user_id, group_id, nickname }
    }
    
    /// 创建用户离开群组通知
    pub fn create_user_left_group_message(user_id: i64, group_id: String) -> WSEventType {
        WSEventType::UserLeftGroup { user_id, group_id }
    }
    
    /// 将事件转换为WebSocket消息
    pub fn event_to_message(event: WSEventType) -> Result<String, serde_json::Error> {
        serde_json::to_string(&event)
    }
    
    /// 创建系统通知消息
    pub fn create_system_notification(message: String) -> WSMessageType {
        WSMessageType::SystemNotification { message }
    }
    
    /// 创建错误消息
    pub fn create_error_message(code: u32, message: String) -> WSMessageType {
        WSMessageType::Error { code, message }
    }
}

