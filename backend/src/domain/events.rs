/// 领域事件定义 - 业务重要时刻的记录
/// 
/// 事件特征：
/// - 不可变
/// - 过去时态命名
/// - 包含事件发生的上下文信息
/// - 用于触发副作用和保持系统一致性

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use crate::domain::{
    values::*,
    entities::{UserType, UserLocation},
};

/// 领域事件枚举
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum DomainEvent {
    // === 用户相关事件 ===
    
    /// 用户创建事件
    UserCreated {
        user_id: UserId,
        user_type: UserType,
        created_at: DateTime<Utc>,
    },
    
    /// 用户升级事件（临时用户 -> 注册用户）
    UserUpgraded {
        user_id: UserId,
        upgraded_at: DateTime<Utc>,
    },
    
    /// 用户资料更新事件
    UserProfileUpdated {
        user_id: UserId,
        updated_at: DateTime<Utc>,
    },
    
    /// 用户位置更新事件
    UserLocationUpdated {
        user_id: UserId,
        old_location: Option<UserLocation>,
        new_location: UserLocation,
        updated_at: DateTime<Utc>,
    },
    
    /// 用户密码设置事件
    UserPasswordSet {
        user_id: UserId,
        set_at: DateTime<Utc>,
    },
    
    /// 用户登录事件
    UserLoggedIn {
        user_id: UserId,
        login_at: DateTime<Utc>,
        ip_address: Option<String>,
        user_agent: Option<String>,
    },
    
    /// 用户登出事件
    UserLoggedOut {
        user_id: UserId,
        logout_at: DateTime<Utc>,
    },
    
    /// 用户删除事件
    UserDeleted {
        user_id: UserId,
        deleted_at: DateTime<Utc>,
    },
    
    // === 群组相关事件 ===
    
    /// 群组创建事件
    GroupCreated {
        group_id: GroupId,
        owner_id: UserId,
        created_at: DateTime<Utc>,
    },
    
    /// 群组成员加入事件
    GroupMemberJoined {
        group_id: GroupId,
        user_id: UserId,
        joined_at: DateTime<Utc>,
    },
    
    /// 群组成员离开事件
    GroupMemberLeft {
        group_id: GroupId,
        user_id: UserId,
        left_at: DateTime<Utc>,
    },
    
    /// 群组成员被踢出事件
    GroupMemberKicked {
        group_id: GroupId,
        user_id: UserId,
        kicked_by: UserId,
        kicked_at: DateTime<Utc>,
        reason: Option<String>,
    },
    
    /// 群组所有权转让事件
    GroupOwnershipTransferred {
        group_id: GroupId,
        old_owner_id: UserId,
        new_owner_id: UserId,
        transferred_at: DateTime<Utc>,
    },
    
    /// 群组信息更新事件
    GroupUpdated {
        group_id: GroupId,
        updated_by: UserId,
        updated_at: DateTime<Utc>,
    },
    
    /// 群组删除事件
    GroupDeleted {
        group_id: GroupId,
        deleted_by: UserId,
        deleted_at: DateTime<Utc>,
    },
    
    // === 消息相关事件 ===
    
    /// 消息发送事件
    MessageSent {
        message_id: MessageId,
        sender_id: UserId,
        recipient_type: MessageRecipientType,
        recipient_id: String, // 可能是用户ID或群组ID
        sent_at: DateTime<Utc>,
    },
    
    /// 消息已读事件
    MessageRead {
        message_id: MessageId,
        reader_id: UserId,
        read_at: DateTime<Utc>,
    },
    
    /// 消息删除事件
    MessageDeleted {
        message_id: MessageId,
        deleted_by: UserId,
        deleted_at: DateTime<Utc>,
    },
    
    // === 瞬间相关事件 ===
    
    /// 瞬间创建事件
    MomentCreated {
        moment_id: MomentId,
        user_id: UserId,
        location: Coordinates,
        created_at: DateTime<Utc>,
    },
    
    /// 瞬间点赞事件
    MomentLiked {
        moment_id: MomentId,
        user_id: UserId,
        liked_at: DateTime<Utc>,
    },
    
    /// 瞬间取消点赞事件
    MomentUnliked {
        moment_id: MomentId,
        user_id: UserId,
        unliked_at: DateTime<Utc>,
    },
    
    /// 瞬间删除事件
    MomentDeleted {
        moment_id: MomentId,
        deleted_by: UserId,
        deleted_at: DateTime<Utc>,
    },
    
    // === 系统事件 ===
    
    /// 系统通知事件
    SystemNotificationSent {
        notification_id: String,
        recipient_id: UserId,
        notification_type: String,
        sent_at: DateTime<Utc>,
    },
    
    /// 数据清理事件
    DataCleanupPerformed {
        cleanup_type: String,
        records_affected: u64,
        performed_at: DateTime<Utc>,
    },
}

/// 消息接收者类型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MessageRecipientType {
    /// 私聊消息
    User,
    /// 群组消息
    Group,
}

impl DomainEvent {
    /// 获取事件ID（用于去重和追踪）
    pub fn event_id(&self) -> String {
        use std::collections::hash_map::DefaultHasher;
        use std::hash::{Hash, Hasher};
        
        let mut hasher = DefaultHasher::new();
        self.hash(&mut hasher);
        format!("{:x}", hasher.finish())
    }
    
    /// 获取事件发生时间
    pub fn occurred_at(&self) -> DateTime<Utc> {
        match self {
            Self::UserCreated { created_at, .. } => *created_at,
            Self::UserUpgraded { upgraded_at, .. } => *upgraded_at,
            Self::UserProfileUpdated { updated_at, .. } => *updated_at,
            Self::UserLocationUpdated { updated_at, .. } => *updated_at,
            Self::UserPasswordSet { set_at, .. } => *set_at,
            Self::UserLoggedIn { login_at, .. } => *login_at,
            Self::UserLoggedOut { logout_at, .. } => *logout_at,
            Self::UserDeleted { deleted_at, .. } => *deleted_at,
            
            Self::GroupCreated { created_at, .. } => *created_at,
            Self::GroupMemberJoined { joined_at, .. } => *joined_at,
            Self::GroupMemberLeft { left_at, .. } => *left_at,
            Self::GroupMemberKicked { kicked_at, .. } => *kicked_at,
            Self::GroupOwnershipTransferred { transferred_at, .. } => *transferred_at,
            Self::GroupUpdated { updated_at, .. } => *updated_at,
            Self::GroupDeleted { deleted_at, .. } => *deleted_at,
            
            Self::MessageSent { sent_at, .. } => *sent_at,
            Self::MessageRead { read_at, .. } => *read_at,
            Self::MessageDeleted { deleted_at, .. } => *deleted_at,
            
            Self::MomentCreated { created_at, .. } => *created_at,
            Self::MomentLiked { liked_at, .. } => *liked_at,
            Self::MomentUnliked { unliked_at, .. } => *unliked_at,
            Self::MomentDeleted { deleted_at, .. } => *deleted_at,
            
            Self::SystemNotificationSent { sent_at, .. } => *sent_at,
            Self::DataCleanupPerformed { performed_at, .. } => *performed_at,
        }
    }
    
    /// 获取事件类型名称
    pub fn event_type(&self) -> &'static str {
        match self {
            Self::UserCreated { .. } => "UserCreated",
            Self::UserUpgraded { .. } => "UserUpgraded",
            Self::UserProfileUpdated { .. } => "UserProfileUpdated",
            Self::UserLocationUpdated { .. } => "UserLocationUpdated",
            Self::UserPasswordSet { .. } => "UserPasswordSet",
            Self::UserLoggedIn { .. } => "UserLoggedIn",
            Self::UserLoggedOut { .. } => "UserLoggedOut",
            Self::UserDeleted { .. } => "UserDeleted",
            
            Self::GroupCreated { .. } => "GroupCreated",
            Self::GroupMemberJoined { .. } => "GroupMemberJoined",
            Self::GroupMemberLeft { .. } => "GroupMemberLeft",
            Self::GroupMemberKicked { .. } => "GroupMemberKicked",
            Self::GroupOwnershipTransferred { .. } => "GroupOwnershipTransferred",
            Self::GroupUpdated { .. } => "GroupUpdated",
            Self::GroupDeleted { .. } => "GroupDeleted",
            
            Self::MessageSent { .. } => "MessageSent",
            Self::MessageRead { .. } => "MessageRead",
            Self::MessageDeleted { .. } => "MessageDeleted",
            
            Self::MomentCreated { .. } => "MomentCreated",
            Self::MomentLiked { .. } => "MomentLiked",
            Self::MomentUnliked { .. } => "MomentUnliked",
            Self::MomentDeleted { .. } => "MomentDeleted",
            
            Self::SystemNotificationSent { .. } => "SystemNotificationSent",
            Self::DataCleanupPerformed { .. } => "DataCleanupPerformed",
        }
    }
    
    /// 获取相关的用户ID（如果有）
    pub fn related_user_ids(&self) -> Vec<UserId> {
        match self {
            Self::UserCreated { user_id, .. }
            | Self::UserUpgraded { user_id, .. }
            | Self::UserProfileUpdated { user_id, .. }
            | Self::UserLocationUpdated { user_id, .. }
            | Self::UserPasswordSet { user_id, .. }
            | Self::UserLoggedIn { user_id, .. }
            | Self::UserLoggedOut { user_id, .. }
            | Self::UserDeleted { user_id, .. }
            | Self::SystemNotificationSent { recipient_id: user_id, .. } => vec![*user_id],
            
            Self::GroupCreated { owner_id, .. } => vec![*owner_id],
            Self::GroupMemberJoined { user_id, .. }
            | Self::GroupMemberLeft { user_id, .. } => vec![*user_id],
            Self::GroupMemberKicked { user_id, kicked_by, .. } => vec![*user_id, *kicked_by],
            Self::GroupOwnershipTransferred { old_owner_id, new_owner_id, .. } => {
                vec![*old_owner_id, *new_owner_id]
            }
            Self::GroupUpdated { updated_by, .. } => vec![*updated_by],
            Self::GroupDeleted { deleted_by, .. } => vec![*deleted_by],
            
            Self::MessageSent { sender_id, .. } => vec![*sender_id],
            Self::MessageDeleted { deleted_by, .. } => vec![*deleted_by],
            Self::MessageRead { reader_id, .. } => vec![*reader_id],
            
            Self::MomentCreated { user_id, .. }
            | Self::MomentLiked { user_id, .. }
            | Self::MomentUnliked { user_id, .. } => vec![*user_id],
            Self::MomentDeleted { deleted_by, .. } => vec![*deleted_by],
            
            Self::DataCleanupPerformed { .. } => vec![],
        }
    }
    
    /// 检查事件是否与指定用户相关
    pub fn is_related_to_user(&self, user_id: UserId) -> bool {
        self.related_user_ids().contains(&user_id)
    }
    
    /// 检查是否为用户事件
    pub fn is_user_event(&self) -> bool {
        matches!(
            self,
            Self::UserCreated { .. }
                | Self::UserUpgraded { .. }
                | Self::UserProfileUpdated { .. }
                | Self::UserLocationUpdated { .. }
                | Self::UserPasswordSet { .. }
                | Self::UserLoggedIn { .. }
                | Self::UserLoggedOut { .. }
                | Self::UserDeleted { .. }
        )
    }
    
    /// 检查是否为群组事件
    pub fn is_group_event(&self) -> bool {
        matches!(
            self,
            Self::GroupCreated { .. }
                | Self::GroupMemberJoined { .. }
                | Self::GroupMemberLeft { .. }
                | Self::GroupMemberKicked { .. }
                | Self::GroupOwnershipTransferred { .. }
                | Self::GroupUpdated { .. }
                | Self::GroupDeleted { .. }
        )
    }
    
    /// 检查是否为消息事件
    pub fn is_message_event(&self) -> bool {
        matches!(
            self,
            Self::MessageSent { .. } | Self::MessageRead { .. } | Self::MessageDeleted { .. }
        )
    }
}

// 为了支持哈希计算，需要实现Hash trait
impl std::hash::Hash for DomainEvent {
    fn hash<H: std::hash::Hasher>(&self, state: &mut H) {
        self.event_type().hash(state);
        self.occurred_at().hash(state);
        
        // 根据事件类型哈希相关的ID
        match self {
            Self::UserCreated { user_id, .. } => user_id.hash(state),
            Self::GroupCreated { group_id, .. } => group_id.hash(state),
            Self::MessageSent { message_id, .. } => message_id.hash(state),
            Self::MomentCreated { moment_id, .. } => moment_id.hash(state),
            _ => {
                // 对于其他事件，哈希相关的用户ID
                for user_id in self.related_user_ids() {
                    user_id.hash(state);
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_event_id_generation() {
        let user_id = UserId::new(1).unwrap();
        let event = DomainEvent::UserCreated {
            user_id,
            user_type: UserType::Registered,
            created_at: Utc::now(),
        };
        
        let event_id = event.event_id();
        assert!(!event_id.is_empty());
        
        // 相同事件应该产生相同的ID
        let same_event = DomainEvent::UserCreated {
            user_id,
            user_type: UserType::Registered,
            created_at: event.occurred_at(),
        };
        assert_eq!(event.event_id(), same_event.event_id());
    }
    
    #[test]
    fn test_event_classification() {
        let user_id = UserId::new(1).unwrap();
        let group_id = GroupId::new();
        
        let user_event = DomainEvent::UserCreated {
            user_id,
            user_type: UserType::Registered,
            created_at: Utc::now(),
        };
        assert!(user_event.is_user_event());
        assert!(!user_event.is_group_event());
        
        let group_event = DomainEvent::GroupCreated {
            group_id,
            owner_id: user_id,
            created_at: Utc::now(),
        };
        assert!(group_event.is_group_event());
        assert!(!group_event.is_user_event());
    }
    
    #[test]
    fn test_related_user_ids() {
        let user_id = UserId::new(1).unwrap();
        let other_user_id = UserId::new(2).unwrap();
        
        let transfer_event = DomainEvent::GroupOwnershipTransferred {
            group_id: GroupId::new(),
            old_owner_id: user_id,
            new_owner_id: other_user_id,
            transferred_at: Utc::now(),
        };
        
        let related_users = transfer_event.related_user_ids();
        assert_eq!(related_users.len(), 2);
        assert!(related_users.contains(&user_id));
        assert!(related_users.contains(&other_user_id));
    }
}

// === 事件类型别名（向后兼容） ===

/// 用户事件（向后兼容别名）
pub type UserEvent = DomainEvent;

/// 群组事件（向后兼容别名）
pub type GroupEvent = DomainEvent;

/// 消息事件（向后兼容别名）
pub type MessageEvent = DomainEvent;

/// 瞬间事件（向后兼容别名）
pub type MomentEvent = DomainEvent;

// === 事件工厂 ===

/// 事件工厂 - 用于生成事件ID和创建事件
pub struct EventFactory;

impl EventFactory {
    /// 生成事件ID
    pub fn generate_event_id() -> String {
        uuid::Uuid::new_v4().to_string()
    }
    
    /// 生成事件时间戳
    pub fn generate_timestamp() -> chrono::DateTime<chrono::Utc> {
        chrono::Utc::now()
    }
}
