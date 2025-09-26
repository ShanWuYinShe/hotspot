/// 统一消息领域实体
/// 
/// 根据HotSpot后端需求文档设计的统一消息架构
/// 私聊和群聊消息都通过groups表统一管理
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use crate::domain::{values::*, events::*, DomainError, DomainResult};

// ===== 核心实体 =====

/// 统一消息实体 - 消息聚合根
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Message {
    /// 消息ID（聚合根标识）
    pub id: MessageId,
    /// 群组ID（私聊也通过群组管理）
    pub group_id: GroupId,
    /// 发送者ID
    pub sender_id: UserId,
    /// 消息内容
    pub content: String,
    /// 消息类型
    pub message_type: MessageType,
    /// 媒体类型（仅媒体消息）
    pub media_type: Option<MediaType>,
    /// 媒体URL（仅媒体消息）
    pub media_url: Option<String>,
    /// 媒体大小（字节，仅媒体消息）
    pub media_size: Option<i64>,
    /// 回复的消息ID
    pub reply_to_message_id: Option<MessageId>,
    /// 创建时间
    pub created_at: DateTime<Utc>,
    /// 更新时间
    pub updated_at: DateTime<Utc>,
    /// 是否已删除（软删除）
    pub is_deleted: bool,
    /// 删除时间
    pub deleted_at: Option<DateTime<Utc>>,
    /// 领域事件（待发布）
    #[serde(skip)]
    pub events: Vec<DomainEvent>,
}

/// 消息类型枚举
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "message_type", rename_all = "lowercase")]
pub enum MessageType {
    /// 文本消息
    Text,
    /// 图片消息
    Image,
    /// 视频消息
    Video,
    /// 音频消息
    Audio,
    /// 文件消息
    File,
    /// 位置消息
    Location,
    /// 系统消息
    System,
}

impl std::str::FromStr for MessageType {
    type Err = String;
    
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "text" => Ok(MessageType::Text),
            "image" => Ok(MessageType::Image),
            "video" => Ok(MessageType::Video),
            "audio" => Ok(MessageType::Audio),
            "file" => Ok(MessageType::File),
            "location" => Ok(MessageType::Location),
            "system" => Ok(MessageType::System),
            _ => Err(format!("未知的消息类型: {}", s)),
        }
    }
}

impl std::fmt::Display for MessageType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            MessageType::Text => write!(f, "text"),
            MessageType::Image => write!(f, "image"),
            MessageType::Video => write!(f, "video"),
            MessageType::Audio => write!(f, "audio"),
            MessageType::File => write!(f, "file"),
            MessageType::Location => write!(f, "location"),
            MessageType::System => write!(f, "system"),
        }
    }
}

/// 媒体类型枚举
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "media_type", rename_all = "lowercase")]
pub enum MediaType {
    /// 图片
    Image,
    /// 视频
    Video,
    /// 音频
    Audio,
    /// 文档
    Document,
}

impl std::str::FromStr for MediaType {
    type Err = String;
    
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "image" => Ok(MediaType::Image),
            "video" => Ok(MediaType::Video),
            "audio" => Ok(MediaType::Audio),
            "document" => Ok(MediaType::Document),
            _ => Err(format!("未知的媒体类型: {}", s)),
        }
    }
}

impl std::fmt::Display for MediaType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            MediaType::Image => write!(f, "image"),
            MediaType::Video => write!(f, "video"),
            MediaType::Audio => write!(f, "audio"),
            MediaType::Document => write!(f, "document"),
        }
    }
}

impl Message {
    /// 创建新的文本消息
    pub fn new_text(
        id: MessageId,
        group_id: GroupId,
        sender_id: UserId,
        content: String,
    ) -> Self {
        let now = Utc::now();
        
        let mut message = Self {
            id: id.clone(),
            group_id: group_id.clone(),
            sender_id,
            content,
            message_type: MessageType::Text,
            media_type: None,
            media_url: None,
            media_size: None,
            reply_to_message_id: None,
            created_at: now,
            updated_at: now,
            is_deleted: false,
            deleted_at: None,
            events: Vec::new(),
        };
        
        // 发布消息创建事件
        message.add_event(DomainEvent::MessageSent {
            message_id: id,
            sender_id,
            recipient_type: crate::domain::events::MessageRecipientType::Group,
            recipient_id: group_id.to_string(),
            sent_at: now,
        });
        
        message
    }
    
    /// 创建新的媒体消息
    pub fn new_media(
        id: MessageId,
        group_id: GroupId,
        sender_id: UserId,
        message_type: MessageType,
        media_type: MediaType,
        media_url: String,
        media_size: i64,
        content: Option<String>,
    ) -> DomainResult<Self> {
        // 验证消息类型和媒体类型的匹配
        match (message_type, media_type) {
            (MessageType::Image, MediaType::Image) |
            (MessageType::Video, MediaType::Video) |
            (MessageType::Audio, MediaType::Audio) |
            (MessageType::File, MediaType::Document) => {},
            _ => return Err(DomainError::BusinessRuleViolation {
                rule: "Message type and media type mismatch".to_string(),
            }),
        }
        
        let now = Utc::now();
        
        let mut message = Self {
            id: id.clone(),
            group_id: group_id.clone(),
            sender_id,
            content: content.unwrap_or_default(),
            message_type,
            media_type: Some(media_type),
            media_url: Some(media_url),
            media_size: Some(media_size),
            reply_to_message_id: None,
            created_at: now,
            updated_at: now,
            is_deleted: false,
            deleted_at: None,
            events: Vec::new(),
        };
        
        // 发布消息创建事件
        message.add_event(DomainEvent::MessageSent {
            message_id: id,
            sender_id,
            recipient_type: crate::domain::events::MessageRecipientType::Group,
            recipient_id: group_id.to_string(),
            sent_at: now,
        });
        
        Ok(message)
    }
    
    /// 创建新的位置消息
    pub fn new_location(
        id: MessageId,
        group_id: GroupId,
        sender_id: UserId,
        latitude: f64,
        longitude: f64,
        location_name: String,
    ) -> Self {
        let now = Utc::now();
        let content = format!("{},{},{}", latitude, longitude, location_name);
        
        let mut message = Self {
            id: id.clone(),
            group_id: group_id.clone(),
            sender_id,
            content,
            message_type: MessageType::Location,
            media_type: None,
            media_url: None,
            media_size: None,
            reply_to_message_id: None,
            created_at: now,
            updated_at: now,
            is_deleted: false,
            deleted_at: None,
            events: Vec::new(),
        };
        
        // 发布消息创建事件
        message.add_event(DomainEvent::MessageSent {
            message_id: id,
            sender_id,
            recipient_type: crate::domain::events::MessageRecipientType::Group,
            recipient_id: group_id.to_string(),
            sent_at: now,
        });
        
        message
    }
    
    /// 创建新的系统消息
    pub fn new_system(
        id: MessageId,
        group_id: GroupId,
        content: String,
    ) -> Self {
        let now = Utc::now();
        
        let mut message = Self {
            id: id.clone(),
            group_id: group_id.clone(),
            sender_id: UserId::system(), // 系统用户ID
            content,
            message_type: MessageType::System,
            media_type: None,
            media_url: None,
            media_size: None,
            reply_to_message_id: None,
            created_at: now,
            updated_at: now,
            is_deleted: false,
            deleted_at: None,
            events: Vec::new(),
        };
        
        // 发布消息创建事件
        message.add_event(DomainEvent::MessageSent {
            message_id: id,
            sender_id: UserId::system(),
            recipient_type: crate::domain::events::MessageRecipientType::Group,
            recipient_id: group_id.to_string(),
            sent_at: now,
        });
        
        message
    }
    
    /// 添加领域事件
    pub fn add_event(&mut self, event: DomainEvent) {
        self.events.push(event);
    }
    
    /// 获取并清空领域事件
    pub fn take_events(&mut self) -> Vec<DomainEvent> {
        std::mem::take(&mut self.events)
    }
    
    /// 设置回复消息
    pub fn set_reply_to(&mut self, reply_to_message_id: MessageId) -> DomainResult<()> {
        if self.reply_to_message_id.is_some() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "Message already has a reply target".to_string(),
            });
        }
        
        self.reply_to_message_id = Some(reply_to_message_id);
        self.updated_at = Utc::now();
        
        Ok(())
    }
    
    /// 软删除消息
    pub fn soft_delete(&mut self) -> DomainResult<()> {
        if self.is_deleted {
            return Err(DomainError::BusinessRuleViolation {
                rule: "Message is already deleted".to_string(),
            });
        }
        
        let now = Utc::now();
        self.is_deleted = true;
        self.deleted_at = Some(now);
        self.updated_at = now;
        
        self.add_event(DomainEvent::MessageDeleted {
            message_id: self.id.clone(),
            deleted_by: self.sender_id, // 简化实现，使用发送者作为删除者
            deleted_at: now,
        });
        
        Ok(())
    }
    
    /// 检查是否为文本消息
    pub fn is_text(&self) -> bool {
        self.message_type == MessageType::Text
    }
    
    /// 检查是否为媒体消息
    pub fn is_media(&self) -> bool {
        matches!(self.message_type, MessageType::Image | MessageType::Video | MessageType::Audio | MessageType::File)
    }
    
    /// 检查是否为系统消息
    pub fn is_system(&self) -> bool {
        self.message_type == MessageType::System
    }
    
    /// 检查是否为回复消息
    pub fn is_reply(&self) -> bool {
        self.reply_to_message_id.is_some()
    }
    
    /// 获取消息预览
    pub fn get_preview(&self) -> String {
        match self.message_type {
            MessageType::Text => {
                if self.content.len() > 50 {
                    format!("{}...", &self.content[..50])
                } else {
                    self.content.clone()
                }
            }
            MessageType::Image => "[图片]".to_string(),
            MessageType::Video => "[视频]".to_string(),
            MessageType::Audio => "[语音]".to_string(),
            MessageType::File => "[文件]".to_string(),
            MessageType::Location => "[位置]".to_string(),
            MessageType::System => self.content.clone(),
        }
    }
}

// ===== 领域模型层 (Domain Models) =====

/// 消息基础信息（最常用的轻量级模型）
/// 用途：消息列表、聊天界面等90%的场景
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageProfile {
    /// 消息ID
    pub message_id: String,
    /// 群组ID
    pub group_id: String,
    /// 发送者ID
    pub sender_id: i64,
    /// 消息内容
    pub content: String,
    /// 消息类型
    pub message_type: MessageType,
    /// 媒体类型（仅媒体消息）
    pub media_type: Option<MediaType>,
    /// 媒体URL（仅媒体消息）
    pub media_url: Option<String>,
    /// 媒体大小（字节，仅媒体消息）
    pub media_size: Option<i64>,
    /// 回复的消息ID
    pub reply_to_message_id: Option<String>,
    /// 创建时间
    pub created_at: DateTime<Utc>,
    /// 是否已删除
    pub is_deleted: bool,
}

/// 消息详细信息（包含完整信息）
/// 用途：消息管理、审核等需要完整信息的场景
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageDetail {
    /// 消息ID
    pub message_id: String,
    /// 群组ID
    pub group_id: String,
    /// 发送者ID
    pub sender_id: i64,
    /// 消息内容
    pub content: String,
    /// 消息类型
    pub message_type: MessageType,
    /// 媒体类型（仅媒体消息）
    pub media_type: Option<MediaType>,
    /// 媒体URL（仅媒体消息）
    pub media_url: Option<String>,
    /// 媒体大小（字节，仅媒体消息）
    pub media_size: Option<i64>,
    /// 回复的消息ID
    pub reply_to_message_id: Option<String>,
    /// 创建时间
    pub created_at: DateTime<Utc>,
    /// 更新时间
    pub updated_at: DateTime<Utc>,
    /// 是否已删除
    pub is_deleted: bool,
    /// 删除时间
    pub deleted_at: Option<DateTime<Utc>>,
}

// ===== 领域模型实现 =====

impl MessageProfile {
    /// 创建新的消息资料
    pub fn new(
        message_id: String,
        group_id: String,
        sender_id: i64,
        content: String,
        message_type: MessageType,
        created_at: DateTime<Utc>,
    ) -> Self {
        Self {
            message_id,
            group_id,
            sender_id,
            content,
            message_type,
            media_type: None,
            media_url: None,
            media_size: None,
            reply_to_message_id: None,
            created_at,
            is_deleted: false,
        }
    }
    
    /// 获取消息预览
    pub fn get_preview(&self) -> String {
        match self.message_type {
            MessageType::Text => {
                if self.content.len() > 50 {
                    format!("{}...", &self.content[..50])
                } else {
                    self.content.clone()
                }
            }
            MessageType::Image => "[图片]".to_string(),
            MessageType::Video => "[视频]".to_string(),
            MessageType::Audio => "[语音]".to_string(),
            MessageType::File => "[文件]".to_string(),
            MessageType::Location => "[位置]".to_string(),
            MessageType::System => self.content.clone(),
        }
    }
    
    /// 检查是否为媒体消息
    pub fn is_media(&self) -> bool {
        matches!(self.message_type, MessageType::Image | MessageType::Video | MessageType::Audio | MessageType::File)
    }
    
    /// 检查是否为系统消息
    pub fn is_system(&self) -> bool {
        self.message_type == MessageType::System
    }
}

impl MessageDetail {
    /// 创建新的消息详细信息
    pub fn new(
        message_id: String,
        group_id: String,
        sender_id: i64,
        content: String,
        message_type: MessageType,
        created_at: DateTime<Utc>,
        updated_at: DateTime<Utc>,
    ) -> Self {
        Self {
            message_id,
            group_id,
            sender_id,
            content,
            message_type,
            media_type: None,
            media_url: None,
            media_size: None,
            reply_to_message_id: None,
            created_at,
            updated_at,
            is_deleted: false,
            deleted_at: None,
        }
    }
    
    /// 转换为消息资料
    pub fn to_profile(&self) -> MessageProfile {
        MessageProfile {
            message_id: self.message_id.clone(),
            group_id: self.group_id.clone(),
            sender_id: self.sender_id,
            content: self.content.clone(),
            message_type: self.message_type,
            media_type: self.media_type,
            media_url: self.media_url.clone(),
            media_size: self.media_size,
            reply_to_message_id: self.reply_to_message_id.clone(),
            created_at: self.created_at,
            is_deleted: self.is_deleted,
        }
    }
}

// ===== 类型转换实现 =====

impl From<MessageDetail> for MessageProfile {
    fn from(detail: MessageDetail) -> Self {
        detail.to_profile()
    }
}

// ===== 类型实现 =====

impl MessageType {
    pub fn as_str(&self) -> &'static str {
        match self {
            MessageType::Text => "text",
            MessageType::Image => "image",
            MessageType::Video => "video",
            MessageType::Audio => "audio",
            MessageType::File => "file",
            MessageType::Location => "location",
            MessageType::System => "system",
        }
    }
    
    pub fn from_str(s: &str) -> Result<Self, String> {
        match s {
            "text" => Ok(MessageType::Text),
            "image" => Ok(MessageType::Image),
            "video" => Ok(MessageType::Video),
            "audio" => Ok(MessageType::Audio),
            "file" => Ok(MessageType::File),
            "location" => Ok(MessageType::Location),
            "system" => Ok(MessageType::System),
            _ => Err(format!("Invalid message type: {}", s)),
        }
    }
}

impl MediaType {
    pub fn as_str(&self) -> &'static str {
        match self {
            MediaType::Image => "image",
            MediaType::Video => "video",
            MediaType::Audio => "audio",
            MediaType::Document => "document",
        }
    }
    
    pub fn from_str(s: &str) -> Result<Self, String> {
        match s {
            "image" => Ok(MediaType::Image),
            "video" => Ok(MediaType::Video),
            "audio" => Ok(MediaType::Audio),
            "document" => Ok(MediaType::Document),
            _ => Err(format!("Invalid media type: {}", s)),
        }
    }
}












