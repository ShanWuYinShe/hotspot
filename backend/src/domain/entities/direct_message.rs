/// 私聊消息领域实体
/// 
/// 表示两个用户之间的私聊消息
/// 私聊消息领域实体
/// 
/// 功能职责：
/// - 定义私聊消息的核心业务实体
/// - 实现消息发送、接收、删除的业务规则
/// - 管理消息状态和已读回执
/// - 提供消息内容的验证和加密逻辑
/// 
/// 业务规则：
/// - 消息必须属于特定的会话
/// - 消息内容的类型和长度限制
/// - 消息已读状态的维护和同步
/// - 消息删除的权限和影响范围
/// - 消息历史的保存和清理策略
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::Type;

/// 直接消息类型枚举
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Type, Default)]
#[sqlx(type_name = "message_type", rename_all = "lowercase")]
pub enum DirectMessageType {
    #[default]
    /// 文本消息
    Text,
    /// 图片消息
    Image,
    /// 系统消息
    System,
}

impl DirectMessageType {
    pub fn as_str(&self) -> &'static str {
        match self {
            DirectMessageType::Text => "text",
            DirectMessageType::Image => "image",
            DirectMessageType::System => "system",
        }
    }
    
    pub fn from_str(s: &str) -> Self {
        match s {
            "text" => DirectMessageType::Text,
            "image" => DirectMessageType::Image,
            "system" => DirectMessageType::System,
            _ => DirectMessageType::Text, // 默认为文本消息
        }
    }
}

/// 私聊消息实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DirectMessage {
    pub id: i64,
    pub message_id: String, // UUID as String
    pub conversation_id: String, // UUID as String
    pub sender_id: i64,
    pub recipient_id: i64,
    pub content: String,
    pub message_type: DirectMessageType,
    pub created_at: DateTime<Utc>,
    pub read_at: Option<DateTime<Utc>>,
    pub latitude: f64,
    pub longitude: f64,
}

impl DirectMessage {
    /// 创建新的文本消息
    pub fn new_text(
        conversation_id: String,
        sender_id: i64,
        recipient_id: i64,
        content: String,
        latitude: f64,
        longitude: f64,
    ) -> Self {
        Self {
            id: 0, // 将由数据库生成
            message_id: uuid::Uuid::new_v4().to_string(),
            conversation_id,
            sender_id,
            recipient_id,
            content,
            message_type: DirectMessageType::Text,
            created_at: Utc::now(),
            read_at: None,
            latitude,
            longitude,
        }
    }
    
    /// 创建新的图片消息
    pub fn new_image(
        conversation_id: String,
        sender_id: i64,
        recipient_id: i64,
        image_url: String,
        latitude: f64,
        longitude: f64,
    ) -> Self {
        Self {
            id: 0, // 将由数据库生成
            message_id: uuid::Uuid::new_v4().to_string(),
            conversation_id,
            sender_id,
            recipient_id,
            content: image_url,
            message_type: DirectMessageType::Image,
            created_at: Utc::now(),
            read_at: None,
            latitude,
            longitude,
        }
    }
    
    /// 创建新的系统消息
    pub fn new_system(
        conversation_id: String,
        content: String,
        latitude: f64,
        longitude: f64,
    ) -> Self {
        Self {
            id: 0, // 将由数据库生成
            message_id: uuid::Uuid::new_v4().to_string(),
            conversation_id,
            sender_id: 0, // 系统消息没有发送者
            recipient_id: 0, // 系统消息没有接收者
            content,
            message_type: DirectMessageType::System,
            created_at: Utc::now(),
            read_at: None,
            latitude,
            longitude,
        }
    }
    
    /// 标记消息为已读
    pub fn mark_as_read(&mut self) {
        if self.read_at.is_none() {
            self.read_at = Some(Utc::now());
        }
    }
    
    /// 检查消息是否已读
    pub fn is_read(&self) -> bool {
        self.read_at.is_some()
    }
    
    /// 检查是否为文本消息
    pub fn is_text(&self) -> bool {
        self.message_type == DirectMessageType::Text
    }
    
    /// 检查是否为图片消息
    pub fn is_image(&self) -> bool {
        self.message_type == DirectMessageType::Image
    }
    
    /// 检查是否为系统消息
    pub fn is_system(&self) -> bool {
        self.message_type == DirectMessageType::System
    }
    
    /// 获取消息预览（用于会话列表）
    pub fn get_preview(&self) -> String {
        match self.message_type {
            DirectMessageType::Text => {
                if self.content.len() > 50 {
                    format!("{}...", &self.content[..50])
                } else {
                    self.content.clone()
                }
            }
            DirectMessageType::Image => "[图片]".to_string(),
            DirectMessageType::System => self.content.clone(),
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
        
        if self.conversation_id.is_empty() {
            errors.push("会话ID不能为空".to_string());
        }
        
        if !self.is_system() {
            if self.sender_id <= 0 {
                errors.push("发送者ID必须为正数".to_string());
            }
            
            if self.recipient_id <= 0 {
                errors.push("接收者ID必须为正数".to_string());
            }
            
            if self.sender_id == self.recipient_id {
                errors.push("发送者和接收者不能是同一人".to_string());
            }
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
