/// 好友系统领域实体
/// 
/// 功能职责：
/// - 定义好友关系和好友请求的核心业务实体
/// - 实现好友请求发送、接受、拒绝的业务规则
/// - 管理好友关系的状态和生命周期
/// - 提供好友关系的验证和权限控制逻辑
/// 
/// 业务规则：
/// - 好友请求的有效期和状态管理
/// - 好友关系的双向性和一致性
/// - 好友请求的防重复发送机制
/// - 好友删除的双向处理
/// - 好友关系对消息和通知的影响
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;
use sqlx::Type;

/// 好友关系状态（从用户角度看的状态）
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum FriendshipStatus {
    /// 不是好友
    NotFriends,
    /// 已发送好友请求（等待对方回应）
    RequestSent,
    /// 收到好友请求（等待自己回应）
    RequestReceived,
    /// 已是好友
    Friends,
    /// 已被对方屏蔽
    Blocked,
}

/// 数据库中的好友关系状态（对应 PostgreSQL 枚举）
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Type)]
#[sqlx(type_name = "friendship_status")]
pub enum DbFriendshipStatus {
    /// 等待处理的好友请求
    #[sqlx(rename = "pending")]
    Pending,
    /// 已接受的好友关系
    #[sqlx(rename = "accepted")]
    Accepted,
    /// 已屏蔽
    #[sqlx(rename = "blocked")]
    Blocked,
    /// 已拒绝
    #[sqlx(rename = "declined")]
    Declined,
}

/// 好友请求状态
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum FriendRequestStatus {
    /// 待处理
    Pending,
    /// 已接受
    Accepted,
    /// 已拒绝
    Rejected,
    /// 已过期
    Expired,
}

/// 好友请求实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FriendRequest {
    /// 请求ID
    pub request_id: String,
    /// 发送者ID
    pub requester_id: i64,
    /// 目标用户ID
    pub target_id: i64,
    /// 请求消息
    pub message: Option<String>,
    /// 请求状态
    pub status: FriendRequestStatus,
    /// 创建时间
    pub created_at: DateTime<Utc>,
    /// 更新时间
    pub updated_at: DateTime<Utc>,
    /// 过期时间
    pub expires_at: DateTime<Utc>,
}

/// 好友关系实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Friendship {
    /// 关系ID
    pub friendship_id: String,
    /// 用户1 ID（较小的用户ID）
    pub user1_id: i64,
    /// 用户2 ID（较大的用户ID）
    pub user2_id: i64,
    /// 建立时间
    pub created_at: DateTime<Utc>,
}

/// 好友资料（用于好友列表显示）
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FriendProfile {
    /// 好友用户ID
    pub user_id: i64,
    /// 好友昵称
    pub nickname: String,
    /// 好友头像
    pub avatar_url: Option<String>,
    /// 最后活跃时间
    pub last_active_at: Option<DateTime<Utc>>,
    /// 成为好友的时间
    pub friendship_created_at: DateTime<Utc>,
    /// 是否在线
    pub is_online: bool,
    /// 关系记录ID（业务ID）
    pub relationship_id: String,
    /// 关系行自增ID（用于游标分页）
    pub relationship_row_id: i64,
    /// 拉黑原因（仅拉黑列表使用）
    pub block_reason: Option<String>,
}

/// 好友请求详情（包含用户信息）
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FriendRequestDetail {
    /// 请求ID
    pub request_id: String,
    /// 发送者ID
    pub requester_id: i64,
    /// 发送者昵称
    pub requester_nickname: String,
    /// 发送者头像
    pub requester_avatar: Option<String>,
    /// 目标用户ID
    pub target_id: i64,
    /// 目标用户昵称
    pub target_nickname: String,
    /// 请求消息
    pub message: Option<String>,
    /// 请求状态
    pub status: FriendRequestStatus,
    /// 创建时间
    pub created_at: DateTime<Utc>,
    /// 过期时间
    pub expires_at: DateTime<Utc>,
}

impl FriendRequest {
    /// 创建新的好友请求
    pub fn new(requester_id: i64, target_id: i64, message: Option<String>) -> Self {
        let now = Utc::now();
        let expires_at = now + chrono::Duration::days(30); // 30天过期
        
        Self {
            request_id: Uuid::new_v4().to_string(),
            requester_id,
            target_id,
            message,
            status: FriendRequestStatus::Pending,
            created_at: now,
            updated_at: now,
            expires_at,
        }
    }
    
    /// 验证好友请求消息
    pub fn validate_message(message: &str) -> Result<(), String> {
        let message = message.trim();
        
        if message.len() > 200 {
            return Err("好友请求消息不能超过200个字符".to_string());
        }
        
        Ok(())
    }
    
    /// 接受好友请求
    pub fn accept(&mut self) -> Result<(), String> {
        if self.status != FriendRequestStatus::Pending {
            return Err("只能接受待处理的好友请求".to_string());
        }
        
        if Utc::now() > self.expires_at {
            self.status = FriendRequestStatus::Expired;
            return Err("好友请求已过期".to_string());
        }
        
        self.status = FriendRequestStatus::Accepted;
        self.updated_at = Utc::now();
        Ok(())
    }
    
    /// 拒绝好友请求
    pub fn reject(&mut self) -> Result<(), String> {
        if self.status != FriendRequestStatus::Pending {
            return Err("只能拒绝待处理的好友请求".to_string());
        }
        
        self.status = FriendRequestStatus::Rejected;
        self.updated_at = Utc::now();
        Ok(())
    }
    
    /// 检查是否已过期
    pub fn is_expired(&self) -> bool {
        Utc::now() > self.expires_at
    }
}

impl Friendship {
    /// 创建新的好友关系
    pub fn new(user1_id: i64, user2_id: i64) -> Self {
        // 确保user1_id < user2_id，保持一致性
        let (user1, user2) = if user1_id < user2_id {
            (user1_id, user2_id)
        } else {
            (user2_id, user1_id)
        };
        
        Self {
            friendship_id: Uuid::new_v4().to_string(),
            user1_id: user1,
            user2_id: user2,
            created_at: Utc::now(),
        }
    }
    
    /// 检查是否包含指定用户
    pub fn contains_user(&self, user_id: i64) -> bool {
        self.user1_id == user_id || self.user2_id == user_id
    }
    
    /// 获取对方用户ID
    pub fn get_friend_id(&self, user_id: i64) -> Option<i64> {
        if self.user1_id == user_id {
            Some(self.user2_id)
        } else if self.user2_id == user_id {
            Some(self.user1_id)
        } else {
            None
        }
    }
}
