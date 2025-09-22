/// PostgreSQL 通知仓储实现
/// 
/// 功能职责：
/// - 实现通知记录的数据库操作
/// - 管理通知的发送和状态更新
/// - 提供通知查询和统计
/// - 支持通知设置管理

use async_trait::async_trait;
use sqlx::{PgPool, Row};
use std::sync::Arc;
use tracing::{debug, error, info, warn};
use uuid::Uuid;

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

/// 通知状态
#[derive(Debug, Clone, PartialEq)]
pub enum NotificationStatus {
    Unread,
    Read,
    Dismissed,
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
    pub quiet_hours_start: Option<chrono::NaiveTime>,
    pub quiet_hours_end: Option<chrono::NaiveTime>,
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
            DomainError::DatabaseError { message: e.to_string() }
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
            DomainError::DatabaseError { message: e.to_string() }
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
        let notification_type_str = match notification.notification_type {
            NotificationType::System => "system",
            NotificationType::NewMessage => "new_message",
            NotificationType::GroupInvite => "group_invite",
            NotificationType::FriendRequest => "friend_request",
            NotificationType::FriendAccepted => "friend_accepted",
            NotificationType::MomentLike => "moment_like",
            NotificationType::MomentComment => "moment_comment",
            NotificationType::Security => "security",
        };
        
        let status_str = match notification.status {
            NotificationStatus::Unread => "unread",
            NotificationStatus::Read => "read",
            NotificationStatus::Dismissed => "dismissed",
        };
        
        let priority_val = match notification.priority {
            NotificationPriority::Low => 1,
            NotificationPriority::Normal => 2,
            NotificationPriority::High => 3,
            NotificationPriority::Urgent => 4,
        };
        
        sqlx::query!(
            "INSERT INTO notifications (
                notification_id, user_id, sender_id, notification_type, title, content,
                related_type, related_id, push_data, image_url, action_url,
                status, priority, expires_at, created_at, updated_at
             ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)",
            notification.notification_id,
            notification.user_id,
            notification.sender_id,
            notification_type_str,
            notification.title,
            notification.content,
            notification.related_type,
            notification.related_id,
            serde_json::to_value(&notification.data).unwrap_or_default(),
            notification.image_url,
            notification.action_url,
            status_str,
            priority_val,
            notification.expires_at,
            notification.created_at,
            notification.updated_at
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("创建通知记录失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?;
        
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?;
        
        debug!("通知创建成功: {}", notification.notification_id);
        Ok(notification)
    }
    
    /// 批量创建通知
    async fn create_notifications(&self, notifications: Vec<Notification>) -> DomainResult<Vec<Notification>> {
        debug!("批量创建 {} 个通知", notifications.len());
        
        // TODO: 实现批量通知创建
        // 1. 批量检查用户通知设置
        // 2. 批量插入通知记录
        // 3. 批量触发推送通知
        // 4. 批量更新未读计数
        
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
        
        let rows = if let Some(status_str) = status_filter {
            sqlx::query!(
                "SELECT n.notification_id, n.user_id, n.sender_id, n.notification_type, n.title, n.content,
                        n.related_type, n.related_id, n.push_data, n.image_url, n.action_url,
                        n.status, n.priority, n.read_at, n.expires_at, n.created_at, n.updated_at
                 FROM notifications n
                 WHERE n.user_id = $1 AND n.status = $2 AND n.id > $3 AND n.is_deleted = false
                 ORDER BY n.id ASC
                 LIMIT $4",
                user_id,
                status_str,
                cursor,
                limit
            )
            .fetch_all(&*self.pool)
            .await
        } else {
            sqlx::query!(
                "SELECT n.notification_id, n.user_id, n.sender_id, n.notification_type, n.title, n.content,
                        n.related_type, n.related_id, n.push_data, n.image_url, n.action_url,
                        n.status, n.priority, n.read_at, n.expires_at, n.created_at, n.updated_at
                 FROM notifications n
                 WHERE n.user_id = $1 AND n.id > $2 AND n.is_deleted = false
                 ORDER BY n.id ASC
                 LIMIT $3",
                user_id,
                cursor,
                limit
            )
            .fetch_all(&*self.pool)
            .await
        };
        
        let rows = rows.map_err(|e| {
            error!("查询用户通知列表失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?;
        
        let mut notifications = Vec::new();
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
                _ => {
                    warn!("未知的通知类型: {}", row.notification_type);
                    continue;
                }
            };
            
            let status = match row.status.as_str() {
                "unread" => NotificationStatus::Unread,
                "read" => NotificationStatus::Read,
                "dismissed" => NotificationStatus::Dismissed,
                _ => {
                    warn!("未知的通知状态: {}", row.status);
                    continue;
                }
            };
            
            let priority = match row.priority {
                1 => NotificationPriority::Low,
                2 => NotificationPriority::Normal,
                3 => NotificationPriority::High,
                4 => NotificationPriority::Urgent,
                _ => NotificationPriority::Normal,
            };
            
            let data: serde_json::Value = row.push_data.unwrap_or_default();
            
            let notification = Notification {
                notification_id: row.notification_id,
                user_id: row.user_id,
                sender_id: row.sender_id,
                notification_type,
                title: row.title,
                content: row.content,
                related_type: row.related_type,
                related_id: row.related_id,
                data,
                image_url: row.image_url,
                action_url: row.action_url,
                status,
                priority,
                read_at: row.read_at,
                expires_at: row.expires_at,
                created_at: row.created_at,
                updated_at: row.updated_at,
            };
            notifications.push(notification);
        }
        
        debug!("获取到 {} 个通知", notifications.len());
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
            DomainError::DatabaseError { message: e.to_string() }
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
        
        // TODO: 实现批量标记通知已读
        // 1. 验证所有通知属于当前用户
        // 2. 批量更新通知状态
        // 3. 更新未读计数
        
        Ok(())
    }
    
    /// 标记所有通知为已读
    async fn mark_all_notifications_read(&self, user_id: i64) -> DomainResult<()> {
        debug!("标记用户 {} 的所有通知为已读", user_id);
        
        // TODO: 实现标记所有通知已读
        // 1. 更新用户所有未读通知状态
        // 2. 重置未读计数为0
        
        Ok(())
    }
    
    /// 获取未读通知数量
    async fn get_unread_count(&self, user_id: i64) -> DomainResult<i64> {
        debug!("获取用户 {} 的未读通知数量", user_id);
        
        let count = sqlx::query_scalar!(
            "SELECT COUNT(*) FROM notifications 
             WHERE user_id = $1 AND status = 'unread' AND is_deleted = false",
            user_id
        )
        .fetch_one(&*self.pool)
        .await
        .map_err(|e| {
            error!("查询未读通知数量失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?;
        
        let count = count.unwrap_or(0);
        debug!("用户 {} 有 {} 个未读通知", user_id, count);
        Ok(count)
    }
    
    /// 删除通知
    async fn delete_notification(&self, notification_id: String, user_id: i64) -> DomainResult<()> {
        debug!("删除通知 {}，用户: {}", notification_id, user_id);
        
        // TODO: 实现通知删除
        // 1. 验证通知属于当前用户
        // 2. 软删除或硬删除通知
        // 3. 更新未读计数（如果是未读通知）
        
        Ok(())
    }
    
    /// 清理过期通知
    async fn cleanup_expired_notifications(&self) -> DomainResult<i64> {
        debug!("清理过期通知");
        
        // TODO: 实现过期通知清理
        // 1. 查找已过期的通知
        // 2. 批量删除过期通知
        // 3. 返回清理的数量
        
        Ok(0)
    }
    
    /// 获取通知设置
    async fn get_notification_settings(&self, user_id: i64) -> DomainResult<Option<NotificationSettings>> {
        debug!("获取用户 {} 的通知设置", user_id);
        
        // TODO: 实现通知设置查询
        // 1. 查询user_notification_settings表
        // 2. 如果不存在则返回默认设置
        
        Ok(None)
    }
    
    /// 更新通知设置
    async fn update_notification_settings(&self, settings: NotificationSettings) -> DomainResult<NotificationSettings> {
        debug!("更新用户 {} 的通知设置", settings.user_id);
        
        // TODO: 实现通知设置更新
        // 1. 插入或更新通知设置
        // 2. 验证设置的有效性
        // 3. 返回更新后的设置
        
        Ok(settings)
    }
    
    /// 根据ID查找通知
    async fn find_by_id(&self, notification_id: String) -> DomainResult<Option<Notification>> {
        debug!("查找通知: {}", notification_id);
        
        // TODO: 实现根据ID查找通知
        // 1. 查询notifications表
        // 2. 返回通知记录或None
        
        Ok(None)
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
            quiet_hours_start: None,
            quiet_hours_end: None,
            updated_at: chrono::Utc::now(),
        }
    }
}
