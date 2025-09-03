/// HTTP响应映射器
/// 
/// 负责将领域模型转换为HTTP响应格式

use serde::{Deserialize, Serialize};
use crate::domain::{User, Group, DirectMessage, GroupMessage, Moment};

/// 用户响应模型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserResponse {
    pub id: i64,
    pub nickname: String,
    pub user_type: String,
    pub has_location: bool,
    pub created_at: String,
    pub updated_at: String,
}

impl From<User> for UserResponse {
    fn from(user: User) -> Self {
        Self {
            id: user.id,
            nickname: user.nickname.clone(),
            user_type: user.user_type.as_str().to_string(),
            has_location: false, // 默认为false，需要通过with_location方法设置
            created_at: user.created_at.to_rfc3339(),
            updated_at: user.updated_at.to_rfc3339(),
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
    pub id: i64,
    pub login_id: Option<String>,
    pub nickname: String,
    pub user_type: String,
    pub location: Option<LocationResponse>,
    pub created_at: String,
    pub updated_at: String,
}

impl From<User> for UserProfileResponse {
    fn from(user: User) -> Self {
        Self {
            id: user.id,
            login_id: user.login_id,
            nickname: user.nickname,
            user_type: user.user_type.as_str().to_string(),
            location: None, // 默认为None，需要通过with_location方法设置
            created_at: user.created_at.to_rfc3339(),
            updated_at: user.updated_at.to_rfc3339(),
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
            latitude: location.latitude,
            longitude: location.longitude,
            address: Some(location.location_name),
        }
    }
}

/// 群组响应模型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupResponse {
    pub id: i64,
    pub name: String,
    pub description: Option<String>,
    pub creator_id: i64,
    pub member_count: i32,
    pub location: Option<LocationResponse>,
    pub created_at: String,
    pub updated_at: String,
}

impl From<Group> for GroupResponse {
    fn from(group: Group) -> Self {
        let location = Some(LocationResponse {
            latitude: group.latitude,
            longitude: group.longitude,
            address: Some(group.location_name.clone()),
        });

        Self {
            id: group.id,
            name: group.name,
            description: group.description,
            creator_id: group.owner_id,
            member_count: 0, // 默认为0，需要通过with_member_count方法设置
            location,
            created_at: group.created_at.to_rfc3339(),
            updated_at: group.updated_at.to_rfc3339(),
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
    pub id: i64,
    pub group_id: i64,
    pub sender_id: i64,
    pub message_type: String,
    pub content: String,
    pub created_at: String,
    pub updated_at: String,
}

impl From<DirectMessage> for MessageResponse {
    fn from(message: DirectMessage) -> Self {
        Self {
            id: 0, // DirectMessage使用UUID，这里需要转换
            group_id: 0, // 私聊消息没有group_id
            sender_id: message.sender_id,
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
            id: 0, // GroupMessage使用UUID，这里需要转换
            group_id: 0, // GroupMessage使用UUID group_id，这里需要转换
            sender_id: message.sender_id,
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
    pub id: i64,
    pub user_id: i64,
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
            id: moment.id,
            user_id: moment.user_id,
            content: moment.content.unwrap_or_default(),
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
