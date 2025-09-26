/// PostgreSQL 通知仓储实现
/// 
/// 功能职责：
/// - 实现通知记录的数据库操作
/// - 管理通知的发送和状态更新
/// - 提供通知查询和统计
/// - 支持通知设置管理

use async_trait::async_trait;
use sqlx::PgPool;
use std::sync::Arc;
use std::str::FromStr;
use tracing::{debug, error, warn};

use crate::domain::{DomainResult, DomainError};

/// 通知类型
#[derive(Debug, Clone, PartialEq)]
pub enum NotificationType {
    FriendRequest,
    FriendAccepted,
    NewMessage,
    GroupInvite,
    MomentLike,
    MomentComment,
    System,
    Security,
}

impl NotificationType {
    pub fn as_str(&self) -> &str {
        match self {
            NotificationType::FriendRequest => "friend_request",
            NotificationType::FriendAccepted => "friend_accepted",
            NotificationType::NewMessage => "new_message",
            NotificationType::GroupInvite => "group_invite",
            NotificationType::MomentLike => "moment_like",
            NotificationType::MomentComment => "moment_comment",
            NotificationType::System => "system",
            NotificationType::Security => "security",
        }
    }
    
    pub fn from_str(s: &str) -> Result<Self, String> {
        match s {
            "friend_request" => Ok(NotificationType::FriendRequest),
            "friend_accepted" => Ok(NotificationType::FriendAccepted),
            "new_message" => Ok(NotificationType::NewMessage),
            "group_invite" => Ok(NotificationType::GroupInvite),
            "moment_like" => Ok(NotificationType::MomentLike),
            "moment_comment" => Ok(NotificationType::MomentComment),
            "system" => Ok(NotificationType::System),
            "security" => Ok(NotificationType::Security),
            _ => Err(format!("Unknown notification type: {}", s)),
        }
    }
}

impl FromStr for NotificationType {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        Self::from_str(s)
    }
}

/// 通知状态
#[derive(Debug, Clone, PartialEq)]
pub enum NotificationStatus {
    Unread,
    Read,
    Dismissed,
}

impl NotificationStatus {
    pub fn as_str(&self) -> &str {
        match self {
            NotificationStatus::Unread => "unread",
            NotificationStatus::Read => "read",
            NotificationStatus::Dismissed => "dismissed",
        }
    }
    
    pub fn from_str(s: &str) -> Result<Self, String> {
        match s {
            "unread" => Ok(NotificationStatus::Unread),
            "read" => Ok(NotificationStatus::Read),
            "dismissed" => Ok(NotificationStatus::Dismissed),
            _ => Err(format!("Unknown notification status: {}", s)),
        }
    }
}

/// 通知优先级
#[derive(Debug, Clone, PartialEq)]
pub enum NotificationPriority {
    Low,
    Normal,
    High,
    Urgent,
}

/// 通知记录
#[derive(Debug, Clone)]
pub struct Notification {
    pub notification_id: String,
    pub user_id: i64,
    pub notification_type: NotificationType,
    pub title: String,
    pub content: String,
    pub data: Option<serde_json::Value>,
    pub status: NotificationStatus,
    pub priority: NotificationPriority,
    pub sender_id: Option<i64>,
    pub related_id: Option<String>,
    pub expires_at: Option<chrono::DateTime<chrono::Utc>>,
    pub read_at: Option<chrono::DateTime<chrono::Utc>>,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub is_deleted: bool,
}

/// 通知设置
#[derive(Debug, Clone)]
pub struct NotificationSettings {
    pub user_id: i64,
    pub friend_requests: bool,
    pub messages: bool,
    pub group_invites: bool,
    pub moment_interactions: bool,
    pub system_notifications: bool,
    pub push_enabled: bool,
    pub email_enabled: bool,
    pub quiet_hours_enabled: bool,
    pub quiet_hours_start: Option<chrono::NaiveTime>,
    pub quiet_hours_end: Option<chrono::NaiveTime>,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
}

/// 通知仓储接口
#[async_trait]
pub trait NotificationRepository: Send + Sync {
    /// 创建通知
    async fn create_notification(&self, notification: Notification) -> DomainResult<Notification>;
    
    /// 批量创建通知
    async fn create_notifications(&self, notifications: Vec<Notification>) -> DomainResult<Vec<Notification>>;
    
    /// 获取用户通知列表
    async fn get_user_notifications(
        &self,
        user_id: i64,
        status: Option<NotificationStatus>,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Notification>>;
    
    /// 标记通知为已读
    async fn mark_notification_read(&self, notification_id: String, user_id: i64) -> DomainResult<()>;
    
    /// 批量标记通知为已读
    async fn mark_notifications_read(&self, notification_ids: Vec<String>, user_id: i64) -> DomainResult<()>;
    
    /// 标记所有通知为已读
    async fn mark_all_notifications_read(&self, user_id: i64) -> DomainResult<()>;
    
    /// 获取未读通知数量
    async fn get_unread_count(&self, user_id: i64) -> DomainResult<i64>;
    
    /// 删除通知
    async fn delete_notification(&self, notification_id: String, user_id: i64) -> DomainResult<()>;
    
    /// 清理过期通知
    async fn cleanup_expired_notifications(&self) -> DomainResult<i64>;
    
    /// 获取通知设置
    async fn get_notification_settings(&self, user_id: i64) -> DomainResult<Option<NotificationSettings>>;
    
    /// 更新通知设置
    async fn update_notification_settings(&self, settings: NotificationSettings) -> DomainResult<NotificationSettings>;
    
    /// 根据ID查找通知
    async fn find_by_id(&self, notification_id: String) -> DomainResult<Option<Notification>>;
}

/// PostgreSQL 通知仓储实现
pub struct PgNotificationRepository {
    pool: Arc<PgPool>,
}

impl PgNotificationRepository {
    /// 创建新的通知仓储
    pub fn new(pool: Arc<PgPool>) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl NotificationRepository for PgNotificationRepository {
    /// 创建通知
    async fn create_notification(&self, notification: Notification) -> DomainResult<Notification> {
        debug!("创建通知: 用户={}, 类型={:?}, 标题={}", 
               notification.user_id, notification.notification_type, notification.title);
        
        let mut tx = self.pool.begin().await.map_err(|e| {
            error!("开始事务失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        
        // 检查用户是否启用了此类型的通知
        let settings = sqlx::query!(
            "SELECT 
                system_notifications, message_notifications, group_invite_notifications,
                friend_request_notifications, moment_like_notifications, moment_comment_notifications,
                mention_notifications, push_enabled
             FROM notification_settings 
             WHERE user_id = $1",
            notification.user_id
        )
        .fetch_optional(&mut *tx)
        .await
        .map_err(|e| {
            error!("查询通知设置失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        
        // 如果用户没有设置，使用默认设置（全部启用）
        let should_notify = if let Some(settings) = settings {
            match notification.notification_type {
                NotificationType::System => settings.system_notifications,
                NotificationType::NewMessage => settings.message_notifications,
                NotificationType::GroupInvite => settings.group_invite_notifications,
                NotificationType::FriendRequest => settings.friend_request_notifications,
                NotificationType::FriendAccepted => settings.friend_request_notifications,
                NotificationType::MomentLike => settings.moment_like_notifications,
                NotificationType::MomentComment => settings.moment_comment_notifications,
                NotificationType::Security => true, // 安全通知总是发送
            }
        } else {
            true // 默认启用所有通知
        };
        
        if !should_notify {
            debug!("用户已禁用此类型通知，跳过创建");
            return Ok(notification);
        }
        
        // 创建通知记录
        let notification_type_str = notification.notification_type.as_str();
        let status_str = notification.status.as_str();
        let priority_val = match notification.priority { NotificationPriority::Low => 1, NotificationPriority::Normal => 2, NotificationPriority::High => 3, NotificationPriority::Urgent => 4 };
        let push_data = notification.data.clone().unwrap_or(serde_json::json!({}));
        
        sqlx::query!(
            r#"
            INSERT INTO notifications (
                notification_id, user_id, sender_id, notification_type, title, content,
                related_id, push_data, status, priority, expires_at, created_at
            ) VALUES (
                $1, $2, $3, $4::text::notification_type_enum, $5, $6,
                $7, $8, $9::text::notification_status_enum, $10, $11, $12
            )
            "#,
            notification.notification_id,
            notification.user_id,
            notification.sender_id,
            notification_type_str,
            notification.title,
            notification.content,
            notification.related_id,
            push_data,
            status_str,
            priority_val,
            notification.expires_at,
            notification.created_at
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("创建通知记录失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        
        debug!("通知创建成功: {}", notification.notification_id);
        Ok(notification)
    }
    
    /// 批量创建通知
    async fn create_notifications(&self, notifications: Vec<Notification>) -> DomainResult<Vec<Notification>> {
        debug!("批量创建 {} 个通知", notifications.len());
        
        let mut tx = self.pool.begin().await.map_err(|e| {
            error!("开始事务失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        
        for n in &notifications {
            let notification_type_str = n.notification_type.as_str();
            let status_str = n.status.as_str();
            let priority_val = match n.priority { NotificationPriority::Low => 1, NotificationPriority::Normal => 2, NotificationPriority::High => 3, NotificationPriority::Urgent => 4 };
            let push_data = n.data.clone().unwrap_or(serde_json::json!({}));
            sqlx::query!(
                r#"
                INSERT INTO notifications (
                    notification_id, user_id, sender_id, notification_type, title, content,
                    related_id, push_data, status, priority, expires_at, created_at
                ) VALUES (
                    $1, $2, $3, $4::text::notification_type_enum, $5, $6,
                    $7, $8, $9::text::notification_status_enum, $10, $11, $12
                )
                "#,
                n.notification_id,
                n.user_id,
                n.sender_id,
                notification_type_str,
                n.title,
                n.content,
                n.related_id,
                push_data,
                status_str,
                priority_val,
                n.expires_at,
                n.created_at
            )
            .execute(&mut *tx)
            .await
            .map_err(|e| {
                error!("批量创建通知失败: {}", e);
                DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
            })?;
        }
        
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        
        Ok(notifications)
    }
    
    /// 获取用户通知列表
    async fn get_user_notifications(
        &self,
        user_id: i64,
        status: Option<NotificationStatus>,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Notification>> {
        debug!("获取用户 {} 的通知列表，状态过滤: {:?}, 游标: {}, 限制: {}", 
               user_id, status, cursor, limit);
        
        let status_filter = status.map(|s| match s {
            NotificationStatus::Unread => "unread",
            NotificationStatus::Read => "read", 
            NotificationStatus::Dismissed => "dismissed",
        });
        
        let notifications = if let Some(status_str) = status_filter {
            let rows = sqlx::query!(
                "SELECT n.notification_id, n.user_id, n.sender_id, n.notification_type::text as \"notification_type!\", n.title, n.content,
                        n.related_id, n.push_data as \"push_data: Option<serde_json::Value>\",
                        n.status::text as \"status!\", n.priority, n.read_at, n.expires_at, n.created_at
                 FROM notifications n
                 WHERE n.user_id = $1 AND n.status::text = $2 AND n.id > $3 AND n.is_deleted = false
                 ORDER BY n.id ASC
                 LIMIT $4",
                user_id,
                status_str,
                cursor,
                limit as i64
            )
            .fetch_all(&*self.pool)
            .await
            .map_err(|e| {
                error!("查询用户通知列表失败: {}", e);
                DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
            })?;
            
            let mut list = Vec::new();
            for row in rows { 
                let notification_type = match row.notification_type.as_str() {
                    "system" => NotificationType::System,
                    "new_message" => NotificationType::NewMessage,
                    "group_invite" => NotificationType::GroupInvite,
                    "friend_request" => NotificationType::FriendRequest,
                    "friend_accepted" => NotificationType::FriendAccepted,
                    "moment_like" => NotificationType::MomentLike,
                    "moment_comment" => NotificationType::MomentComment,
                    "security" => NotificationType::Security,
                    _ => { warn!("未知的通知类型: {}", row.notification_type); continue; }
                };
                let status = match row.status.as_str() {
                    "unread" => NotificationStatus::Unread,
                    "read" => NotificationStatus::Read,
                    "dismissed" => NotificationStatus::Dismissed,
                    _ => { warn!("未知的通知状态: {}", row.status); continue; }
                };
                let priority = match row.priority { 1 => NotificationPriority::Low, 2 => NotificationPriority::Normal, 3 => NotificationPriority::High, 4 => NotificationPriority::Urgent, _ => NotificationPriority::Normal };
                let data: Option<serde_json::Value> = row.push_data;
                let notification = Notification {
                    notification_id: row.notification_id,
                    user_id: row.user_id,
                    sender_id: row.sender_id,
                    notification_type,
                    title: row.title,
                    content: row.content,
                    related_id: row.related_id,
                    data,
                    status,
                    priority,
                    read_at: row.read_at,
                    expires_at: row.expires_at,
                    created_at: row.created_at,
                    is_deleted: false,
                };
                list.push(notification);
            }
            list
        } else {
            let rows = sqlx::query!(
                "SELECT n.notification_id, n.user_id, n.sender_id, n.notification_type::text as \"notification_type!\", n.title, n.content,
                        n.related_id, n.push_data as \"push_data: Option<serde_json::Value>\",
                        n.status::text as \"status!\", n.priority, n.read_at, n.expires_at, n.created_at
                 FROM notifications n
                 WHERE n.user_id = $1 AND n.id > $2 AND n.is_deleted = false
                 ORDER BY n.id ASC
                 LIMIT $3",
                user_id,
                cursor,
                limit as i64
            )
            .fetch_all(&*self.pool)
            .await
            .map_err(|e| {
                error!("查询用户通知列表失败: {}", e);
                DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
            })?;
            
            let mut list = Vec::new();
            for row in rows { 
                let notification_type = match row.notification_type.as_str() {
                    "system" => NotificationType::System,
                    "new_message" => NotificationType::NewMessage,
                    "group_invite" => NotificationType::GroupInvite,
                    "friend_request" => NotificationType::FriendRequest,
                    "friend_accepted" => NotificationType::FriendAccepted,
                    "moment_like" => NotificationType::MomentLike,
                    "moment_comment" => NotificationType::MomentComment,
                    "security" => NotificationType::Security,
                    _ => { warn!("未知的通知类型: {}", row.notification_type); continue; }
                };
                let status = match row.status.as_str() {
                    "unread" => NotificationStatus::Unread,
                    "read" => NotificationStatus::Read,
                    "dismissed" => NotificationStatus::Dismissed,
                    _ => { warn!("未知的通知状态: {}", row.status); continue; }
                };
                let priority = match row.priority { 1 => NotificationPriority::Low, 2 => NotificationPriority::Normal, 3 => NotificationPriority::High, 4 => NotificationPriority::Urgent, _ => NotificationPriority::Normal };
                let data: Option<serde_json::Value> = row.push_data;
                let notification = Notification {
                    notification_id: row.notification_id,
                    user_id: row.user_id,
                    sender_id: row.sender_id,
                    notification_type,
                    title: row.title,
                    content: row.content,
                    related_id: row.related_id,
                    data,
                    status,
                    priority,
                    read_at: row.read_at,
                    expires_at: row.expires_at,
                    created_at: row.created_at,
                    is_deleted: false,
                };
                list.push(notification);
            }
            list
        };
        for notification in &notifications { debug!("查询到通知: {} - {}", notification.notification_id, notification.title); }
        Ok(notifications)
    }
    
    /// 标记通知为已读
    async fn mark_notification_read(&self, notification_id: String, user_id: i64) -> DomainResult<()> {
        debug!("标记通知 {} 为已读，用户: {}", notification_id, user_id);
        
        let now = chrono::Utc::now();
        let rows_affected = sqlx::query!(
            "UPDATE notifications 
             SET status = 'read', read_at = $1, updated_at = $2
             WHERE notification_id = $3 AND user_id = $4 AND status = 'unread'",
            now,
            now,
            notification_id,
            user_id
        )
        .execute(&*self.pool)
        .await
        .map_err(|e| {
            error!("标记通知已读失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?
        .rows_affected();
        
        if rows_affected == 0 {
            return Err(DomainError::NotFound {
                resource: "通知".to_string(),
            });
        }
        
        debug!("通知标记已读成功");
        Ok(())
    }
    
    /// 批量标记通知为已读
    async fn mark_notifications_read(&self, notification_ids: Vec<String>, user_id: i64) -> DomainResult<()> {
        debug!("批量标记 {} 个通知为已读，用户: {}", notification_ids.len(), user_id);
        
        if notification_ids.is_empty() { return Ok(()); }
        let now = chrono::Utc::now();
        let _res = sqlx::query!(
            r#"UPDATE notifications 
               SET status = 'read', read_at = $1, updated_at = $1
               WHERE user_id = $2 AND status = 'unread' AND notification_id = ANY($3)
            "#,
            now,
            user_id,
            &notification_ids[..]
        )
        .execute(&*self.pool)
        .await
        .map_err(|e| {
            error!("批量标记通知已读失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        Ok(())
    }
    
    /// 标记所有通知为已读
    async fn mark_all_notifications_read(&self, user_id: i64) -> DomainResult<()> {
        debug!("标记用户 {} 的所有通知为已读", user_id);
        let now = chrono::Utc::now();
        sqlx::query!(
            r#"UPDATE notifications 
               SET status = 'read', read_at = $1, updated_at = $1
               WHERE user_id = $2 AND status = 'unread' AND is_deleted = false
            "#,
            now,
            user_id
        )
        .execute(&*self.pool)
        .await
        .map_err(|e| {
            error!("标记所有通知为已读失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        Ok(())
    }
    
    /// 获取未读通知数量
    async fn get_unread_count(&self, user_id: i64) -> DomainResult<i64> {
        debug!("获取用户 {} 的未读通知数量", user_id);
        
        let count = sqlx::query_scalar!(
            "SELECT COUNT(*) as \"count!\" FROM notifications 
             WHERE user_id = $1 AND status = 'unread' AND is_deleted = false",
            user_id
        )
        .fetch_one(&*self.pool)
        .await
        .map_err(|e| {
            error!("查询未读通知数量失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        
        debug!("用户 {} 有 {} 个未读通知", user_id, count);
        Ok(count)
    }
    
    /// 删除通知（软删除）
    async fn delete_notification(&self, notification_id: String, user_id: i64) -> DomainResult<()> {
        debug!("删除通知 {}，用户: {}", notification_id, user_id);
        let now = chrono::Utc::now();
        let res = sqlx::query!(
            r#"UPDATE notifications 
               SET is_deleted = true, deleted_at = $1, updated_at = $1
               WHERE notification_id = $2 AND user_id = $3 AND is_deleted = false
            "#,
            now,
            notification_id,
            user_id
        )
        .execute(&*self.pool)
        .await
        .map_err(|e| {
            error!("删除通知失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        if res.rows_affected() == 0 {
            return Err(DomainError::NotFound { resource: "通知".to_string() });
        }
        Ok(())
    }
    
    /// 清理过期通知（软删除）
    async fn cleanup_expired_notifications(&self) -> DomainResult<i64> {
        debug!("清理过期通知");
        let now = chrono::Utc::now();
        let res = sqlx::query!(
            r#"UPDATE notifications 
               SET is_deleted = true, deleted_at = $1, updated_at = $1
               WHERE expires_at IS NOT NULL AND expires_at < $1 AND is_deleted = false
            "#,
            now
        )
        .execute(&*self.pool)
        .await
        .map_err(|e| {
            error!("清理过期通知失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        Ok(res.rows_affected() as i64)
    }
    
    /// 获取通知设置
    async fn get_notification_settings(&self, user_id: i64) -> DomainResult<Option<NotificationSettings>> {
        debug!("获取用户 {} 的通知设置", user_id);
        let row = sqlx::query!(
            r#"SELECT user_id,
                       push_enabled, email_enabled,
                       system_notifications, message_notifications, group_invite_notifications,
                       friend_request_notifications, moment_like_notifications, moment_comment_notifications,
                       quiet_hours_enabled, quiet_start_time, quiet_end_time,
                       created_at, updated_at
                FROM notification_settings
                WHERE user_id = $1"#,
            user_id
        )
        .fetch_optional(&*self.pool)
        .await
        .map_err(|e| {
            error!("查询通知设置失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        if let Some(s) = row {
            let settings = NotificationSettings {
                user_id: s.user_id,
                friend_requests: s.friend_request_notifications,
                messages: s.message_notifications,
                group_invites: s.group_invite_notifications,
                moment_interactions: s.moment_like_notifications || s.moment_comment_notifications,
                system_notifications: s.system_notifications,
                push_enabled: s.push_enabled,
                email_enabled: s.email_enabled,
                quiet_hours_enabled: s.quiet_hours_enabled,
                quiet_hours_start: s.quiet_start_time,
                quiet_hours_end: s.quiet_end_time,
                created_at: s.created_at,
                updated_at: s.updated_at,
            };
            Ok(Some(settings))
        } else {
            Ok(None)
        }
    }
    
    /// 更新通知设置（UPSERT）
    async fn update_notification_settings(&self, settings: NotificationSettings) -> DomainResult<NotificationSettings> {
        debug!("更新用户 {} 的通知设置", settings.user_id);
        let now = chrono::Utc::now();
        sqlx::query!(
            r#"INSERT INTO notification_settings (
                    user_id, push_enabled, email_enabled,
                    system_notifications, message_notifications, group_invite_notifications,
                    friend_request_notifications, moment_like_notifications, moment_comment_notifications,
                    quiet_hours_enabled, quiet_start_time, quiet_end_time, created_at, updated_at
                ) VALUES (
                    $1, $2, $3,
                    $4, $5, $6,
                    $7, $8, $9,
                    $10, $11, $12, $13, $13
                )
                ON CONFLICT (user_id) DO UPDATE SET
                    push_enabled = EXCLUDED.push_enabled,
                    email_enabled = EXCLUDED.email_enabled,
                    system_notifications = EXCLUDED.system_notifications,
                    message_notifications = EXCLUDED.message_notifications,
                    group_invite_notifications = EXCLUDED.group_invite_notifications,
                    friend_request_notifications = EXCLUDED.friend_request_notifications,
                    moment_like_notifications = EXCLUDED.moment_like_notifications,
                    moment_comment_notifications = EXCLUDED.moment_comment_notifications,
                    quiet_hours_enabled = EXCLUDED.quiet_hours_enabled,
                    quiet_start_time = EXCLUDED.quiet_start_time,
                    quiet_end_time = EXCLUDED.quiet_end_time,
                    updated_at = EXCLUDED.updated_at
            "#,
            settings.user_id,
            settings.push_enabled,
            settings.email_enabled,
            settings.system_notifications,
            settings.messages,
            settings.group_invites,
            settings.friend_requests,
            settings.moment_interactions,
            settings.moment_interactions,
            settings.quiet_hours_enabled,
            settings.quiet_hours_start,
            settings.quiet_hours_end,
            now
        )
        .execute(&*self.pool)
        .await
        .map_err(|e| {
            error!("更新通知设置失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        Ok(NotificationSettings { updated_at: now, ..settings })
    }
    
    /// 根据ID查找通知
    async fn find_by_id(&self, notification_id: String) -> DomainResult<Option<Notification>> {
        debug!("查找通知: {}", notification_id);
        let row = sqlx::query!(
            r#"SELECT notification_id, user_id, sender_id, notification_type::text as "notification_type!",
                       title, content, related_id, push_data as "push_data: Option<serde_json::Value>",
                       status::text as "status!", priority, read_at, expires_at, created_at, is_deleted
                FROM notifications 
                WHERE notification_id = $1"#,
            notification_id
        )
        .fetch_optional(&*self.pool)
        .await
        .map_err(|e| {
            error!("查询通知失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        if let Some(r) = row {
            let ntype = NotificationType::from_str(&r.notification_type)
                .map_err(|msg| DomainError::BusinessRuleViolation { rule: msg })?;
            let nstatus = NotificationStatus::from_str(&r.status)
                .map_err(|msg| DomainError::BusinessRuleViolation { rule: msg })?;
            let priority = match r.priority { 1 => NotificationPriority::Low, 2 => NotificationPriority::Normal, 3 => NotificationPriority::High, 4 => NotificationPriority::Urgent, _ => NotificationPriority::Normal };
            let notif = Notification {
                notification_id: r.notification_id,
                user_id: r.user_id,
                notification_type: ntype,
                title: r.title,
                content: r.content,
                data: r.push_data,
                status: nstatus,
                priority,
                sender_id: r.sender_id,
                related_id: r.related_id,
                expires_at: r.expires_at,
                read_at: r.read_at,
                created_at: r.created_at,
                is_deleted: r.is_deleted,
            };
            Ok(Some(notif))
        } else {
            Ok(None)
        }
    }
}

impl Clone for PgNotificationRepository {
    fn clone(&self) -> Self {
        Self {
            pool: Arc::clone(&self.pool),
        }
    }
}

/// 默认通知设置
impl Default for NotificationSettings {
    fn default() -> Self {
        Self {
            user_id: 0,
            friend_requests: true,
            messages: true,
            group_invites: true,
            moment_interactions: true,
            system_notifications: true,
            push_enabled: true,
            email_enabled: false,
            quiet_hours_enabled: false,
            quiet_hours_start: None,
            quiet_hours_end: None,
            created_at: chrono::Utc::now(),
            updated_at: chrono::Utc::now(),
        }
    }
}
