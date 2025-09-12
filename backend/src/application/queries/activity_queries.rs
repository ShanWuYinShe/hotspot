/// 活动数据查询
/// 
/// 提供系统活动数据的查询功能，包括最近活动、用户活动等
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use crate::domain::{DomainError, DomainResult};
use super::PaginationQuery;

/// 获取最近活动查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetRecentActivitiesQuery {
    /// 分页参数
    #[serde(flatten)]
    pub pagination: PaginationQuery,
    /// 活动类型过滤
    pub activity_types: Option<Vec<ActivityType>>,
    /// 时间范围（小时）
    pub hours_range: Option<i32>,
}

/// 活动类型枚举
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum ActivityType {
    /// 用户注册
    UserRegistered,
    /// 群组创建
    GroupCreated,
    /// 群组加入
    GroupJoined,
    /// 生活瞬间发布
    MomentCreated,
    /// 生活瞬间点赞
    MomentLiked,
    /// 消息发送
    MessageSent,
    /// 用户位置更新
    LocationUpdated,
}

/// 活动数据项
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActivityItem {
    /// 活动ID
    pub id: String,
    /// 活动类型
    pub activity_type: ActivityType,
    /// 用户昵称
    pub user: String,
    /// 活动内容描述
    pub content: String,
    /// 活动时间
    pub time: DateTime<Utc>,
    /// 图标名称
    pub icon: String,
    /// 相关数据（可选）
    pub metadata: Option<serde_json::Value>,
}

/// 活动查询结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActivitiesResult {
    /// 活动列表
    pub activities: Vec<ActivityItem>,
    /// 分页信息
    pub has_more: bool,
    pub next_cursor: Option<i64>,
    /// 总数（可选）
    pub total_count: Option<i64>,
}

impl GetRecentActivitiesQuery {
    /// 创建默认查询
    pub fn new() -> Self {
        Self {
            pagination: PaginationQuery::default(),
            activity_types: None,
            hours_range: Some(24), // 默认24小时内的活动
        }
    }
    
    /// 验证查询参数
    pub fn validate(&self) -> DomainResult<()> {
        let mut errors = Vec::new();
        
        // 验证时间范围
        if let Some(hours) = self.hours_range {
            if hours <= 0 || hours > 168 { // 最多7天
                errors.push("时间范围必须在1-168小时之间".to_string());
            }
        }
        
        // 验证活动类型
        if let Some(ref types) = self.activity_types {
            if types.is_empty() {
                errors.push("活动类型列表不能为空".to_string());
            }
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(DomainError::BusinessRuleViolation {
                rule: errors.join("; "),
            })
        }
    }
    
    /// 获取实际的时间范围（小时）
    pub fn get_hours_range(&self) -> i32 {
        self.hours_range.unwrap_or(24)
    }
}

impl Default for GetRecentActivitiesQuery {
    fn default() -> Self {
        Self::new()
    }
}

impl ActivityItem {
    /// 创建用户注册活动
    pub fn user_registered(user_id: i64, nickname: String, created_at: DateTime<Utc>) -> Self {
        Self {
            id: format!("user_reg_{}", user_id),
            activity_type: ActivityType::UserRegistered,
            user: nickname.clone(),
            content: format!("{} 加入了 HotSpot", nickname),
            time: created_at,
            icon: "i-heroicons-user-plus".to_string(),
            metadata: Some(serde_json::json!({
                "user_id": user_id,
                "action": "register"
            })),
        }
    }
    
    /// 创建群组创建活动
    pub fn group_created(
        group_id: String, 
        group_name: String, 
        creator_name: String, 
        created_at: DateTime<Utc>
    ) -> Self {
        Self {
            id: format!("group_create_{}", group_id),
            activity_type: ActivityType::GroupCreated,
            user: creator_name.clone(),
            content: format!("{} 创建了群组「{}」", creator_name, group_name),
            time: created_at,
            icon: "i-heroicons-user-group".to_string(),
            metadata: Some(serde_json::json!({
                "group_id": group_id,
                "group_name": group_name,
                "action": "create_group"
            })),
        }
    }
    
    /// 创建群组加入活动
    pub fn group_joined(
        group_id: String,
        group_name: String,
        user_name: String,
        joined_at: DateTime<Utc>
    ) -> Self {
        Self {
            id: format!("group_join_{}_{}", group_id, chrono::Utc::now().timestamp()),
            activity_type: ActivityType::GroupJoined,
            user: user_name.clone(),
            content: format!("{} 加入了群组「{}」", user_name, group_name),
            time: joined_at,
            icon: "i-heroicons-arrow-right-on-rectangle".to_string(),
            metadata: Some(serde_json::json!({
                "group_id": group_id,
                "group_name": group_name,
                "action": "join_group"
            })),
        }
    }
    
    /// 创建生活瞬间发布活动
    pub fn moment_created(
        moment_id: String,
        user_name: String,
        content: String,
        created_at: DateTime<Utc>
    ) -> Self {
        let short_content = if content.len() > 30 {
            format!("{}...", &content[..30])
        } else {
            content.clone()
        };
        
        Self {
            id: format!("moment_create_{}", moment_id),
            activity_type: ActivityType::MomentCreated,
            user: user_name.clone(),
            content: format!("{} 分享了生活瞬间：{}", user_name, short_content),
            time: created_at,
            icon: "i-heroicons-camera".to_string(),
            metadata: Some(serde_json::json!({
                "moment_id": moment_id,
                "action": "create_moment"
            })),
        }
    }
    
    /// 创建生活瞬间点赞活动
    pub fn moment_liked(
        moment_id: String,
        liker_name: String,
        moment_author: String,
        liked_at: DateTime<Utc>
    ) -> Self {
        Self {
            id: format!("moment_like_{}_{}", moment_id, chrono::Utc::now().timestamp()),
            activity_type: ActivityType::MomentLiked,
            user: liker_name.clone(),
            content: format!("{} 点赞了 {} 的生活瞬间", liker_name, moment_author),
            time: liked_at,
            icon: "i-heroicons-heart".to_string(),
            metadata: Some(serde_json::json!({
                "moment_id": moment_id,
                "action": "like_moment"
            })),
        }
    }
    
    /// 创建位置更新活动
    pub fn location_updated(
        user_name: String,
        location_name: String,
        updated_at: DateTime<Utc>
    ) -> Self {
        Self {
            id: format!("location_update_{}_{}", user_name, updated_at.timestamp()),
            activity_type: ActivityType::LocationUpdated,
            user: user_name.clone(),
            content: format!("{} 更新了位置到 {}", user_name, location_name),
            time: updated_at,
            icon: "i-heroicons-map-pin".to_string(),
            metadata: Some(serde_json::json!({
                "location_name": location_name,
                "action": "update_location"
            })),
        }
    }
    
    /// 格式化时间为前端需要的字符串格式
    pub fn formatted_time(&self) -> String {
        let now = chrono::Utc::now();
        let duration = now.signed_duration_since(self.time);
        
        if let Ok(std_duration) = duration.to_std() {
            let seconds = std_duration.as_secs();
            
            if seconds < 60 {
                "刚刚".to_string()
            } else if seconds < 3600 {
                format!("{}分钟前", seconds / 60)
            } else if seconds < 86400 {
                format!("{}小时前", seconds / 3600)
            } else if seconds < 2592000 {
                format!("{}天前", seconds / 86400)
            } else {
                format!("{}个月前", seconds / 2592000)
            }
        } else {
            "未知时间".to_string()
        }
    }
}

/// 转换为前端需要的格式
impl From<ActivityItem> for serde_json::Value {
    fn from(activity: ActivityItem) -> Self {
        serde_json::json!({
            "id": activity.id,
            "user": activity.user,
            "content": activity.content,
            "time": activity.formatted_time(),
            "icon": activity.icon
        })
    }
}
