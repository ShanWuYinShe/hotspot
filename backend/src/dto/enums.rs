// src/dto/enums.rs - DTO层枚举类型定义
// 说明：所有枚举类型与 @api_spec.mdc 严格一致
// 1. 大部分枚举直接从 types::enums 重新导出
// 2. 仅定义少量DTO特有的枚举类型（不存储在数据库中）

use serde::{Deserialize, Serialize};

// ============================================================
// 重新导出数据库层的枚举类型
// ============================================================
pub use crate::types::enums::{
    FeedChannel,
    FeedItemType,
    InviteStatus,
    JoinMethod,
    LeaderboardScope,
    LeaderboardTimeframe,
    MemberRole,
    MessageType,
    NotificationCategory,
    NotificationSeverity,
    Platform as DevicePlatform, // Re-export Platform as DevicePlatform to match API spec naming if needed, or just use Platform? Spec uses 'ios' | 'android' | 'web' string union in DTO usually.
                                // Spec 3.13.5 uses platform: 'ios' | 'android' | 'web'.
                                // Let's keep it as Platform or map it.
    PrivacyLevel,
    ReactionType,
    RelationshipStatus,
    RelationshipType,
    ReportStatus,
    ReportTargetType,
    RoomType,
};

// ============================================================
// DTO特有的枚举类型（不存储在数据库中）
// ============================================================

// 收藏操作
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum FavoriteAction {
    Add,
    Remove,
}

// 信息流反馈操作
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum FeedAction {
    Hide,
    NotInterested,
    Clicked,
    Viewed,
}

// 文件分类
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum FileCategory {
    Avatar,
    GroupImage,
    MomentImage,
    File,
}

// 点赞操作
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum LikeAction {
    Like,
    Unlike,
}

// 举报处理操作
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum ReportAction {
    Resolve,
    Dismiss,
    Escalate,
}

// 上传状态
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum UploadStatus {
    InProgress,
    Completed,
    Failed,
}

// 通用操作类型（用于批量操作等场景）
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum ActionType {
    Add,
    Remove,
    Like,
    Unlike,
    Register,
    Unregister,
}

// 好友申请状态方向（发送/接收）
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum FriendRequestStatus {
    Sent,     // 我发送的好友申请
    Received, // 我收到的好友申请
}

// 系统健康检查状态
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum HealthStatus {
    Healthy,
    Degraded,
    Unhealthy,
}

// 配置值类型
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum ConfigValueType {
    String,
    Number,
    Boolean,
}

// 标签趋势
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum TagTrend {
    Rising,
    Stable,
    Falling,
}
