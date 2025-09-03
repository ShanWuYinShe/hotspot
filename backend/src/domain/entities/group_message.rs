/// 群组消息领域实体
/// 
/// 表示群组内的消息

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// 群组消息类型枚举
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum GroupMessageType {
    /// 文本消息
    Text,
    /// 图片消息
    Image,
    /// 系统消息
    System,
}

impl GroupMessageType {
    pub fn as_str(&self) -> &'static str {
        match self {
            GroupMessageType::Text => "text",
            GroupMessageType::Image => "image",
            GroupMessageType::System => "system",
        }
    }
    
    pub fn from_str(s: &str) -> Self {
        match s {
            "text" => GroupMessageType::Text,
            "image" => GroupMessageType::Image,
            "system" => GroupMessageType::System,
            _ => GroupMessageType::Text, // 默认为文本消息
        }
    }
}

/// 群组消息实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupMessage {
    pub id: i64,
    pub message_id: String, // UUID as String
    pub group_id: String, // UUID as String
    pub sender_id: i64,
    pub content: String,
    pub message_type: GroupMessageType,
    pub created_at: DateTime<Utc>,
    pub latitude: f64,
    pub longitude: f64,
}

impl GroupMessage {
    /// 创建新的文本消息
    pub fn new_text(
        group_id: String,
        sender_id: i64,
        content: String,
        latitude: f64,
        longitude: f64,
    ) -> Self {
        Self {
            id: 0, // 将由数据库生成
            message_id: uuid::Uuid::new_v4().to_string(),
            group_id,
            sender_id,
            content,
            message_type: GroupMessageType::Text,
            created_at: Utc::now(),
            latitude,
            longitude,
        }
    }
    
    /// 创建新的图片消息
    pub fn new_image(
        group_id: String,
        sender_id: i64,
        image_url: String,
        latitude: f64,
        longitude: f64,
    ) -> Self {
        Self {
            id: 0, // 将由数据库生成
            message_id: uuid::Uuid::new_v4().to_string(),
            group_id,
            sender_id,
            content: image_url,
            message_type: GroupMessageType::Image,
            created_at: Utc::now(),
            latitude,
            longitude,
        }
    }
    
    /// 创建新的系统消息
    pub fn new_system(
        group_id: String,
        content: String,
        latitude: f64,
        longitude: f64,
    ) -> Self {
        Self {
            id: 0, // 将由数据库生成
            message_id: uuid::Uuid::new_v4().to_string(),
            group_id,
            sender_id: 0, // 系统消息没有发送者
            content,
            message_type: GroupMessageType::System,
            created_at: Utc::now(),
            latitude,
            longitude,
        }
    }
    
    /// 检查是否为文本消息
    pub fn is_text(&self) -> bool {
        self.message_type == GroupMessageType::Text
    }
    
    /// 检查是否为图片消息
    pub fn is_image(&self) -> bool {
        self.message_type == GroupMessageType::Image
    }
    
    /// 检查是否为系统消息
    pub fn is_system(&self) -> bool {
        self.message_type == GroupMessageType::System
    }
    
    /// 获取消息预览（用于通知）
    pub fn get_preview(&self) -> String {
        match self.message_type {
            GroupMessageType::Text => {
                if self.content.len() > 50 {
                    format!("{}...", &self.content[..50])
                } else {
                    self.content.clone()
                }
            }
            GroupMessageType::Image => "[图片]".to_string(),
            GroupMessageType::System => self.content.clone(),
        }
    }
    
    /// 验证消息数据的完整性
    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        
        if self.content.is_empty() {
            errors.push("消息内容不能为空".to_string());
        }
        
        if self.content.len() > 2000 {
            errors.push("消息内容长度不能超过2000个字符".to_string());
        }
        
        if self.group_id.is_empty() {
            errors.push("群组ID不能为空".to_string());
        }
        
        if !self.is_system() && self.sender_id <= 0 {
            errors.push("非系统消息必须有有效的发送者ID".to_string());
        }
        
        if self.latitude < -90.0 || self.latitude > 90.0 {
            errors.push("纬度必须在-90到90之间".to_string());
        }
        
        if self.longitude < -180.0 || self.longitude > 180.0 {
            errors.push("经度必须在-180到180之间".to_string());
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
}
