/// HTTP响应映射器
/// 
/// 负责将领域模型转换为HTTP响应格式
use serde::{Deserialize, Serialize};
use crate::domain::{UserProfile, GroupProfile, DirectMessage, GroupMessage, Moment};
use crate::utils::JsSafeI64;

/// 用户响应模型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserResponse {
    pub id: JsSafeI64,
    pub nickname: String,
    pub user_type: String,
    pub has_location: bool,
    pub created_at: String,
    pub updated_at: String,
}

impl From<UserProfile> for UserResponse {
    fn from(user: UserProfile) -> Self {
        Self {
            id: JsSafeI64::new(user.user_id),
            nickname: user.nickname.clone(),
            user_type: match user.user_type {
                crate::domain::UserType::Temporary => "temporary".to_string(),
                crate::domain::UserType::Regular => "registered".to_string(),
            },
            has_location: false, // 默认为false，需要通过with_location方法设置
            created_at: user.last_active_at.to_rfc3339(),
            updated_at: user.last_active_at.to_rfc3339(),
        }
    }
}

impl UserResponse {
    /// 创建带位置信息的用户响应
    pub fn with_location(mut self, has_location: bool) -> Self {
        self.has_location = has_location;
        self
    }
}

/// 用户详细信息响应模型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserProfileResponse {
    pub id: JsSafeI64,
    pub nickname: String,
    pub user_type: String,
    pub location: Option<LocationResponse>,
    pub created_at: String,
    pub updated_at: String,
}

impl From<crate::domain::UserDetail> for UserProfileResponse {
    fn from(user: crate::domain::UserDetail) -> Self {
        Self {
            id: JsSafeI64::new(user.user_id),
            nickname: user.nickname,
            user_type: match user.user_type {
                crate::domain::UserType::Temporary => "temporary".to_string(),
                crate::domain::UserType::Regular => "registered".to_string(),
            },
            location: None, // 默认为None，需要通过with_location方法设置
            created_at: user.created_at.to_rfc3339(),
            updated_at: user.created_at.to_rfc3339(), // 使用 created_at 作为 updated_at
        }
    }
}

impl UserProfileResponse {
    /// 创建带位置信息的用户资料响应
    pub fn with_location(mut self, location: Option<LocationResponse>) -> Self {
        self.location = location;
        self
    }
}

/// 位置响应模型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LocationResponse {
    pub latitude: f64,
    pub longitude: f64,
    pub address: Option<String>,
}

impl From<crate::domain::UserLocation> for LocationResponse {
    fn from(location: crate::domain::UserLocation) -> Self {
        Self {
            latitude: location.coordinates.latitude.value(),
            longitude: location.coordinates.longitude.value(),
            address: Some(location.location_name),
        }
    }
}

/// 群组响应模型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupResponse {
    pub id: JsSafeI64,
    pub name: String,
    pub description: Option<String>,
    pub creator_id: JsSafeI64,
    pub member_count: i32,
    pub location: Option<LocationResponse>,
    pub created_at: String,
    pub updated_at: String,
}

impl From<GroupProfile> for GroupResponse {
    fn from(group: GroupProfile) -> Self {
        let location = Some(LocationResponse {
            latitude: group.latitude,
            longitude: group.longitude,
            address: Some(group.location_name.clone()),
        });

        Self {
            id: JsSafeI64::new(group.group_id.parse().unwrap_or(0)), // 将 UUID 字符串转换为数字，这里可能需要重新设计
            name: group.name,
            description: group.description,
            creator_id: JsSafeI64::new(group.owner_id),
            member_count: group.member_count,
            location,
            created_at: group.last_active_at.to_rfc3339(),
            updated_at: group.last_active_at.to_rfc3339(),
        }
    }
}

impl GroupResponse {
    /// 设置群组成员数量
    pub fn with_member_count(mut self, member_count: i32) -> Self {
        self.member_count = member_count;
        self
    }
}

/// 消息响应模型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageResponse {
    pub id: JsSafeI64,
    pub group_id: JsSafeI64,
    pub sender_id: JsSafeI64,
    pub message_type: String,
    pub content: String,
    pub created_at: String,
    pub updated_at: String,
}

impl From<DirectMessage> for MessageResponse {
    fn from(message: DirectMessage) -> Self {
        Self {
            id: JsSafeI64::new(0), // DirectMessage使用UUID，这里需要转换
            group_id: JsSafeI64::new(0), // 私聊消息没有group_id
            sender_id: JsSafeI64::new(message.sender_id),
            message_type: message.message_type.as_str().to_string(),
            content: message.content,
            created_at: message.created_at.to_rfc3339(),
            updated_at: message.created_at.to_rfc3339(),
        }
    }
}

impl From<GroupMessage> for MessageResponse {
    fn from(message: GroupMessage) -> Self {
        Self {
            id: JsSafeI64::new(0), // GroupMessage使用UUID，这里需要转换
            group_id: JsSafeI64::new(0), // GroupMessage使用UUID group_id，这里需要转换
            sender_id: JsSafeI64::new(message.sender_id),
            message_type: message.message_type.as_str().to_string(),
            content: message.content,
            created_at: message.created_at.to_rfc3339(),
            updated_at: message.created_at.to_rfc3339(),
        }
    }
}

/// 瞬间响应模型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MomentResponse {
    pub id: JsSafeI64,
    pub user_id: JsSafeI64,
    pub content: String,
    pub image_url: Option<String>,
    pub like_count: i32,
    pub comment_count: i32,
    pub location: Option<LocationResponse>,
    pub created_at: String,
    pub updated_at: String,
}

impl From<Moment> for MomentResponse {
    fn from(moment: Moment) -> Self {
        let location = Some(LocationResponse {
            latitude: moment.latitude,
            longitude: moment.longitude,
            address: Some(moment.location_name.clone()),
        });

        Self {
            id: JsSafeI64::new(moment.id),
            user_id: JsSafeI64::new(moment.user_id),
            content: moment.content,
            image_url: None, // 默认为None，需要通过with_image_url方法设置
            like_count: 0, // 默认为0，需要通过with_like_count方法设置
            comment_count: 0, // 默认为0，需要通过with_comment_count方法设置
            location,
            created_at: moment.created_at.to_rfc3339(),
            updated_at: moment.created_at.to_rfc3339(), // 使用created_at作为updated_at
        }
    }
}

impl MomentResponse {
    /// 设置图片URL
    pub fn with_image_url(mut self, image_url: Option<String>) -> Self {
        self.image_url = image_url;
        self
    }
    
    /// 设置点赞数
    pub fn with_like_count(mut self, like_count: i32) -> Self {
        self.like_count = like_count;
        self
    }
    
    /// 设置评论数
    pub fn with_comment_count(mut self, comment_count: i32) -> Self {
        self.comment_count = comment_count;
        self
    }
}

/// 分页响应包装器
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaginatedResponse<T> {
    pub items: Vec<T>,
    pub pagination: PaginationMeta,
}

/// 分页元数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaginationMeta {
    pub has_more: bool,
    pub next_cursor: Option<i64>,
}

impl<T> PaginatedResponse<T> {
    pub fn new(items: Vec<T>, has_more: bool, next_cursor: Option<i64>) -> Self {
        Self {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        }
    }
}
