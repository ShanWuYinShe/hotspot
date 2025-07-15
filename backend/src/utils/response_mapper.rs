use crate::models::api::message::{DirectMessageHistoryItem, GroupMessageHistoryItem};
use crate::repositories::message_repository::{MessageRow, GroupMessageRow};
use crate::utils::response::PaginatedResponse;

/// 响应映射辅助工具
pub struct ResponseMapper;

impl ResponseMapper {
    /// 将MessageRow转换为DirectMessageHistoryItem
    #[allow(dead_code)]
    pub fn map_message_to_history_item(msg: MessageRow) -> DirectMessageHistoryItem {
        DirectMessageHistoryItem {
            message_id: msg.message_id.to_string(),
            conversation_id: msg.conversation_id.to_string(),
            sender_id: msg.sender_id.to_string(),
            sender_name: String::new(), // 可在后续版本中补充
            recipient_id: msg.recipient_id.to_string(),
            content: msg.content,
            message_type: msg.message_type,
            sent_at: msg.created_at,
            read_at: msg.read_at,
            latitude: msg.latitude,
            longitude: msg.longitude,
        }
    }
    
    /// 将GroupMessageRow转换为GroupMessageHistoryItem
    pub fn map_group_message_to_history_item(msg: GroupMessageRow) -> GroupMessageHistoryItem {
        GroupMessageHistoryItem {
            message_id: msg.message_id.to_string(),
            group_id: msg.group_id.to_string(),
            sender_id: msg.sender_id.to_string(),
            sender_name: String::new(), // 可在后续版本中补充
            content: msg.content,
            message_type: msg.message_type,
            sent_at: msg.created_at,
            latitude: msg.latitude,
            longitude: msg.longitude,
        }
    }
    
    /// 创建分页响应
    pub fn create_paginated_response<T>(
        items: Vec<T>,
        has_more: bool,
        next_cursor: Option<i64>,
    ) -> PaginatedResponse<T> {
        PaginatedResponse {
            items,
            pagination: crate::utils::response::PaginationMeta {
                has_more,
                next_cursor,
            },
        }
    }
    
    /// 批量映射消息到历史项目
    #[allow(dead_code)]
    pub fn map_messages_to_history_items(messages: Vec<MessageRow>) -> Vec<DirectMessageHistoryItem> {
        messages
            .into_iter()
            .map(Self::map_message_to_history_item)
            .collect()
    }
    
    /// 批量映射群组消息到历史项目
    pub fn map_group_messages_to_history_items(messages: Vec<GroupMessageRow>) -> Vec<GroupMessageHistoryItem> {
        messages
            .into_iter()
            .map(Self::map_group_message_to_history_item)
            .collect()
    }
}
