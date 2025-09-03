/// 领域事件模块
/// 
/// 定义系统中的领域事件，用于实现事件驱动架构

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// 事件基础trait
pub trait DomainEvent {
    /// 获取事件ID
    fn event_id(&self) -> &str;
    
    /// 获取事件类型
    fn event_type(&self) -> &str;
    
    /// 获取事件发生时间
    fn occurred_at(&self) -> DateTime<Utc>;
    
    /// 获取聚合根ID
    fn aggregate_id(&self) -> i64;
}

/// 用户相关事件
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum UserEvent {
    /// 用户已注册
    UserRegistered {
        event_id: String,
        user_id: i64,
        login_id: String,
        nickname: String,
        occurred_at: DateTime<Utc>,
    },
    /// 临时用户已创建
    TempUserCreated {
        event_id: String,
        user_id: i64,
        nickname: String,
        occurred_at: DateTime<Utc>,
    },
    /// 用户已升级
    UserUpgraded {
        event_id: String,
        user_id: i64,
        login_id: String,
        occurred_at: DateTime<Utc>,
    },
    /// 用户位置已更新
    UserLocationUpdated {
        event_id: String,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        address: Option<String>,
        occurred_at: DateTime<Utc>,
    },
    /// 用户资料已更新
    UserProfileUpdated {
        event_id: String,
        user_id: i64,
        occurred_at: DateTime<Utc>,
    },
    /// 用户密码已设置
    UserPasswordSet {
        event_id: String,
        user_id: i64,
        occurred_at: DateTime<Utc>,
    },
    /// 用户已删除
    UserDeleted {
        event_id: String,
        user_id: i64,
        occurred_at: DateTime<Utc>,
    },
}

impl DomainEvent for UserEvent {
    fn event_id(&self) -> &str {
        match self {
            UserEvent::UserRegistered { event_id, .. }
            | UserEvent::TempUserCreated { event_id, .. }
            | UserEvent::UserUpgraded { event_id, .. }
            | UserEvent::UserLocationUpdated { event_id, .. }
            | UserEvent::UserProfileUpdated { event_id, .. }
            | UserEvent::UserPasswordSet { event_id, .. }
            | UserEvent::UserDeleted { event_id, .. } => event_id,
        }
    }
    
    fn event_type(&self) -> &str {
        match self {
            UserEvent::UserRegistered { .. } => "UserRegistered",
            UserEvent::TempUserCreated { .. } => "TempUserCreated",
            UserEvent::UserUpgraded { .. } => "UserUpgraded",
            UserEvent::UserLocationUpdated { .. } => "UserLocationUpdated",
            UserEvent::UserProfileUpdated { .. } => "UserProfileUpdated",
            UserEvent::UserPasswordSet { .. } => "UserPasswordSet",
            UserEvent::UserDeleted { .. } => "UserDeleted",
        }
    }
    
    fn occurred_at(&self) -> DateTime<Utc> {
        match self {
            UserEvent::UserRegistered { occurred_at, .. }
            | UserEvent::TempUserCreated { occurred_at, .. }
            | UserEvent::UserUpgraded { occurred_at, .. }
            | UserEvent::UserLocationUpdated { occurred_at, .. }
            | UserEvent::UserProfileUpdated { occurred_at, .. }
            | UserEvent::UserPasswordSet { occurred_at, .. }
            | UserEvent::UserDeleted { occurred_at, .. } => *occurred_at,
        }
    }
    
    fn aggregate_id(&self) -> i64 {
        match self {
            UserEvent::UserRegistered { user_id, .. }
            | UserEvent::TempUserCreated { user_id, .. }
            | UserEvent::UserUpgraded { user_id, .. }
            | UserEvent::UserLocationUpdated { user_id, .. }
            | UserEvent::UserProfileUpdated { user_id, .. }
            | UserEvent::UserPasswordSet { user_id, .. }
            | UserEvent::UserDeleted { user_id, .. } => *user_id,
        }
    }
}

/// 群组相关事件
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum GroupEvent {
    /// 群组已创建
    GroupCreated {
        event_id: String,
        group_id: i64,
        creator_id: i64,
        name: String,
        occurred_at: DateTime<Utc>,
    },
    /// 成员已加入
    MemberJoined {
        event_id: String,
        group_id: i64,
        user_id: i64,
        role: String,
        occurred_at: DateTime<Utc>,
    },
    /// 成员已离开
    MemberLeft {
        event_id: String,
        group_id: i64,
        user_id: i64,
        occurred_at: DateTime<Utc>,
    },
    /// 群组信息已更新
    GroupInfoUpdated {
        event_id: String,
        group_id: i64,
        updated_by: i64,
        occurred_at: DateTime<Utc>,
    },
}

impl DomainEvent for GroupEvent {
    fn event_id(&self) -> &str {
        match self {
            GroupEvent::GroupCreated { event_id, .. }
            | GroupEvent::MemberJoined { event_id, .. }
            | GroupEvent::MemberLeft { event_id, .. }
            | GroupEvent::GroupInfoUpdated { event_id, .. } => event_id,
        }
    }
    
    fn event_type(&self) -> &str {
        match self {
            GroupEvent::GroupCreated { .. } => "GroupCreated",
            GroupEvent::MemberJoined { .. } => "MemberJoined",
            GroupEvent::MemberLeft { .. } => "MemberLeft",
            GroupEvent::GroupInfoUpdated { .. } => "GroupInfoUpdated",
        }
    }
    
    fn occurred_at(&self) -> DateTime<Utc> {
        match self {
            GroupEvent::GroupCreated { occurred_at, .. }
            | GroupEvent::MemberJoined { occurred_at, .. }
            | GroupEvent::MemberLeft { occurred_at, .. }
            | GroupEvent::GroupInfoUpdated { occurred_at, .. } => *occurred_at,
        }
    }
    
    fn aggregate_id(&self) -> i64 {
        match self {
            GroupEvent::GroupCreated { group_id, .. }
            | GroupEvent::MemberJoined { group_id, .. }
            | GroupEvent::MemberLeft { group_id, .. }
            | GroupEvent::GroupInfoUpdated { group_id, .. } => *group_id,
        }
    }
}

/// 消息相关事件
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MessageEvent {
    /// 消息已发送
    MessageSent {
        event_id: String,
        message_id: i64,
        group_id: i64,
        sender_id: i64,
        occurred_at: DateTime<Utc>,
    },
    /// 消息已删除
    MessageDeleted {
        event_id: String,
        message_id: i64,
        group_id: i64,
        deleted_by: i64,
        occurred_at: DateTime<Utc>,
    },
}

impl DomainEvent for MessageEvent {
    fn event_id(&self) -> &str {
        match self {
            MessageEvent::MessageSent { event_id, .. }
            | MessageEvent::MessageDeleted { event_id, .. } => event_id,
        }
    }
    
    fn event_type(&self) -> &str {
        match self {
            MessageEvent::MessageSent { .. } => "MessageSent",
            MessageEvent::MessageDeleted { .. } => "MessageDeleted",
        }
    }
    
    fn occurred_at(&self) -> DateTime<Utc> {
        match self {
            MessageEvent::MessageSent { occurred_at, .. }
            | MessageEvent::MessageDeleted { occurred_at, .. } => *occurred_at,
        }
    }
    
    fn aggregate_id(&self) -> i64 {
        match self {
            MessageEvent::MessageSent { message_id, .. }
            | MessageEvent::MessageDeleted { message_id, .. } => *message_id,
        }
    }
}

/// 瞬间相关事件
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MomentEvent {
    /// 瞬间已创建
    MomentCreated {
        event_id: String,
        moment_id: i64,
        user_id: i64,
        occurred_at: DateTime<Utc>,
    },
    /// 瞬间已点赞
    MomentLiked {
        event_id: String,
        moment_id: i64,
        user_id: i64,
        liked_by: i64,
        occurred_at: DateTime<Utc>,
    },
    /// 瞬间已删除
    MomentDeleted {
        event_id: String,
        moment_id: i64,
        user_id: i64,
        occurred_at: DateTime<Utc>,
    },
}

impl DomainEvent for MomentEvent {
    fn event_id(&self) -> &str {
        match self {
            MomentEvent::MomentCreated { event_id, .. }
            | MomentEvent::MomentLiked { event_id, .. }
            | MomentEvent::MomentDeleted { event_id, .. } => event_id,
        }
    }
    
    fn event_type(&self) -> &str {
        match self {
            MomentEvent::MomentCreated { .. } => "MomentCreated",
            MomentEvent::MomentLiked { .. } => "MomentLiked",
            MomentEvent::MomentDeleted { .. } => "MomentDeleted",
        }
    }
    
    fn occurred_at(&self) -> DateTime<Utc> {
        match self {
            MomentEvent::MomentCreated { occurred_at, .. }
            | MomentEvent::MomentLiked { occurred_at, .. }
            | MomentEvent::MomentDeleted { occurred_at, .. } => *occurred_at,
        }
    }
    
    fn aggregate_id(&self) -> i64 {
        match self {
            MomentEvent::MomentCreated { moment_id, .. }
            | MomentEvent::MomentLiked { moment_id, .. }
            | MomentEvent::MomentDeleted { moment_id, .. } => *moment_id,
        }
    }
}

/// 事件工厂，用于创建事件
pub struct EventFactory;

impl EventFactory {
    /// 生成唯一的事件ID
    pub fn generate_event_id() -> String {
        use uuid::Uuid;
        Uuid::new_v4().to_string()
    }
    
    /// 创建用户注册事件
    pub fn user_registered(user_id: i64, login_id: String, nickname: String) -> UserEvent {
        UserEvent::UserRegistered {
            event_id: Self::generate_event_id(),
            user_id,
            login_id,
            nickname,
            occurred_at: Utc::now(),
        }
    }
    
    /// 创建临时用户创建事件
    pub fn temp_user_created(user_id: i64, nickname: String) -> UserEvent {
        UserEvent::TempUserCreated {
            event_id: Self::generate_event_id(),
            user_id,
            nickname,
            occurred_at: Utc::now(),
        }
    }
    
    /// 创建群组创建事件
    pub fn group_created(group_id: i64, creator_id: i64, name: String) -> GroupEvent {
        GroupEvent::GroupCreated {
            event_id: Self::generate_event_id(),
            group_id,
            creator_id,
            name,
            occurred_at: Utc::now(),
        }
    }
    
    /// 创建消息发送事件
    pub fn message_sent(message_id: i64, group_id: i64, sender_id: i64) -> MessageEvent {
        MessageEvent::MessageSent {
            event_id: Self::generate_event_id(),
            message_id,
            group_id,
            sender_id,
            occurred_at: Utc::now(),
        }
    }
}

