/// 用户消息已读状态领域实体
/// 
/// 记录每个用户在每个会话/群组中最后已读的消息位置
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// 用户消息已读状态实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserMessageReadStatus {
    pub id: i64,
    pub user_id: i64,
    pub conversation_id: Option<String>, // 私聊会话ID (UUID as String)
    pub group_id: Option<String>, // 群聊ID (UUID as String)
    pub last_read_message_id: i64, // 最后已读消息的自增ID
    pub updated_at: DateTime<Utc>,
}

impl UserMessageReadStatus {
    /// 创建新的私聊已读状态
    pub fn new_for_conversation(
        user_id: i64,
        conversation_id: String,
        last_read_message_id: i64,
    ) -> Self {
        Self {
            id: 0, // 将由数据库生成
            user_id,
            conversation_id: Some(conversation_id),
            group_id: None,
            last_read_message_id,
            updated_at: Utc::now(),
        }
    }
    
    /// 创建新的群聊已读状态
    pub fn new_for_group(
        user_id: i64,
        group_id: String,
        last_read_message_id: i64,
    ) -> Self {
        Self {
            id: 0, // 将由数据库生成
            user_id,
            conversation_id: None,
            group_id: Some(group_id),
            last_read_message_id,
            updated_at: Utc::now(),
        }
    }
    
    /// 更新已读消息位置
    pub fn update_last_read(&mut self, message_id: i64) {
        if message_id > self.last_read_message_id {
            self.last_read_message_id = message_id;
            self.updated_at = Utc::now();
        }
    }
    
    /// 检查是否为私聊已读状态
    pub fn is_conversation_status(&self) -> bool {
        self.conversation_id.is_some() && self.group_id.is_none()
    }
    
    /// 检查是否为群聊已读状态
    pub fn is_group_status(&self) -> bool {
        self.conversation_id.is_none() && self.group_id.is_some()
    }
    
    /// 验证已读状态数据的完整性
    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        
        if self.user_id <= 0 {
            errors.push("用户ID必须为正数".to_string());
        }
        
        if self.last_read_message_id <= 0 {
            errors.push("最后已读消息ID必须为正数".to_string());
        }
        
        // 确保只设置了conversation_id或group_id中的一个
        match (self.conversation_id.as_ref(), self.group_id.as_ref()) {
            (Some(_), None) => {
                // 私聊已读状态，正确
            }
            (None, Some(_)) => {
                // 群聊已读状态，正确
            }
            (Some(_), Some(_)) => {
                errors.push("不能同时设置会话ID和群组ID".to_string());
            }
            (None, None) => {
                errors.push("必须设置会话ID或群组ID中的一个".to_string());
            }
        }
        
        if let Some(conversation_id) = &self.conversation_id {
            if conversation_id.is_empty() {
                errors.push("会话ID不能为空".to_string());
            }
        }
        
        if let Some(group_id) = &self.group_id {
            if group_id.is_empty() {
                errors.push("群组ID不能为空".to_string());
            }
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
}

