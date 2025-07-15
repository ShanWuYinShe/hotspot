//! 群组消息摘要任务定义
use serde::{Deserialize, Serialize};
use uuid::Uuid;
use chrono::Utc;

/// 群组消息摘要任务数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupMessageDigestJob {
    pub group_id: Uuid,
    pub group_name: String,
    pub last_active_at: i64,
    pub min_inactivity_days: i32, // 最小不活跃天数，超过此天数才发送摘要
}

/// 群组成员活跃度更新任务数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupMemberActivityJob {
    pub group_id: Uuid,
    pub user_id: i64,
    pub activity_type: GroupActivityType,
    pub timestamp: i64,
}

/// 群组活动类型
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum GroupActivityType {
    MessageSent,       // 发送消息
    MessageRead,       // 阅读消息
    Join,              // 加入群组
    Leave,             // 离开群组
    Reaction,          // 消息回应
    EventCreate,       // 创建活动
    EventJoin,         // 参与活动
    MediaShare,        // 分享媒体
}

impl GroupActivityType {
    /// 获取活跃度分数
    pub fn get_score(&self) -> i32 {
        match self {
            Self::MessageSent => 10,
            Self::Join => 5,
            Self::EventCreate => 15,
            Self::EventJoin => 8,
            Self::MediaShare => 12,
            Self::Reaction => 3,
            _ => 1,
        }
    }
    
    /// 转换为字符串
    #[allow(dead_code)]
    pub fn to_string(&self) -> String {
        match self {
            Self::MessageSent => "message_sent".to_string(),
            Self::MessageRead => "message_read".to_string(),
            Self::Join => "join".to_string(),
            Self::Leave => "leave".to_string(),
            Self::Reaction => "reaction".to_string(),
            Self::EventCreate => "event_create".to_string(),
            Self::EventJoin => "event_join".to_string(),
            Self::MediaShare => "media_share".to_string(),
        }
    }
}

/// 创建群组消息摘要任务
#[allow(dead_code)]
pub fn create_group_message_digest_job(
    group_id: Uuid,
    group_name: String,
    last_active_at: i64,
    min_inactivity_days: i32,
) -> GroupMessageDigestJob {
    GroupMessageDigestJob {
        group_id,
        group_name,
        last_active_at,
        min_inactivity_days,
    }
}

/// 创建群组成员活跃度更新任务
#[allow(dead_code)]
pub fn create_group_member_activity_job(
    group_id: Uuid,
    user_id: i64,
    activity_type: GroupActivityType,
) -> GroupMemberActivityJob {
    GroupMemberActivityJob {
        group_id,
        user_id,
        activity_type,
        timestamp: Utc::now().timestamp(),
    }
}
