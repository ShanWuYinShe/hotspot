use futures_util::future;
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    dto::{
        request::{GetNotificationsRequest, MarkNotificationsReadRequest},
        response::{
            NotificationItem, PaginatedData, PaginationMeta, UnreadNotificationCountData, UserInfo,
        },
    },
    error::Result,
    events::event_types::AppEvent,
    repositories::notification_repo::{NotificationRepository, NotificationType},
    repositories::user_repo::UserRepository,
    types::enums::NotificationApiType,
    utils::validation::Validator,
    EventBus,
};

#[derive(Clone)]
pub struct NotificationService {
    repo: NotificationRepository,
    user_repo: UserRepository,
    event_bus: Arc<EventBus>,
    media_service: crate::services::media_service::MediaService,
}

impl NotificationService {
    pub fn new(
        repo: NotificationRepository,
        user_repo: UserRepository,
        event_bus: Arc<EventBus>,
        media_service: crate::services::media_service::MediaService,
    ) -> Self {
        Self {
            repo,
            user_repo,
            event_bus,
            media_service,
        }
    }

    /// 获取通知列表
    pub async fn list_notifications(
        &self,
        user_id: i64,
        req: GetNotificationsRequest,
    ) -> Result<PaginatedData<NotificationItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let cursor = req.cursor;

        let (rows, next_cursor) = self.repo.list_by_user(user_id, limit, cursor).await?;

        // 使用 futures_util::future::join_all 来并发签名所有 avatar_url
        let items: Vec<NotificationItem> =
            future::join_all(rows.into_iter().map(|row| async move {
                let sender = if let Some(uid) = row.sender_id {
                    let avatar_url = self
                        .media_service
                        .sign_url_if_path(row.sender_avatar_url.as_deref())
                        .await;
                    Some(UserInfo {
                        user_id: uid,
                        nickname: row.sender_nickname.unwrap_or_default(),
                        avatar_url,
                        user_type: crate::types::enums::UserType::Normal, // 简化：通知快照不存 UserType，假设为 Normal
                    })
                } else {
                    None
                };

                NotificationItem {
                    notification_id: row.notification_id.to_string(),
                    r#type: row.r#type.to_string(),
                    title: row.title,
                    content: row.content,
                    created_at: row.created_at.to_rfc3339(),
                    read: row.is_read,
                    sender,
                    target_type: row.target_type,
                    target_id: row.target_id,
                    reference_id: row.reference_id,
                    target_preview: row.target_preview,
                }
            }))
            .await;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    /// 获取未读计数
    pub async fn get_unread_count(&self, user_id: i64) -> Result<UnreadNotificationCountData> {
        let total = self.repo.count_unread(user_id).await?;
        let breakdown_rows = self.repo.count_unread_breakdown(user_id).await?;

        let mut breakdown = crate::dto::response::UnreadNotificationBreakdown {
            like: 0,
            comment: 0,
            follow: 0,
            system: 0,
            mention: 0,
        };

        for (r#type, count) in breakdown_rows {
            match r#type {
                NotificationType::Like => breakdown.like += count,
                NotificationType::Comment | NotificationType::Reply => breakdown.comment += count,
                NotificationType::Follow => breakdown.follow += count,
                NotificationType::Mention => breakdown.mention += count,
                NotificationType::System
                | NotificationType::GroupInvite
                | NotificationType::GroupApply
                | NotificationType::GroupKick
                | NotificationType::GroupQuit
                | NotificationType::GroupDismiss
                | NotificationType::GroupAdmin => breakdown.system += count,
            }
        }

        Ok(UnreadNotificationCountData {
            total: total as i64,
            breakdown: Some(breakdown),
        })
    }

    /// 标记已读
    pub async fn mark_read(&self, user_id: i64, req: MarkNotificationsReadRequest) -> Result<()> {
        // 1. 按 ID 列表标记
        if let Some(ids) = req.notification_ids {
            if !ids.is_empty() {
                let uuids: Vec<Uuid> = ids
                    .iter()
                    .filter_map(|id| Uuid::parse_str(id).ok())
                    .collect();
                if !uuids.is_empty() {
                    self.repo.mark_read(user_id, &uuids).await?;
                }
            }
            // 如果 ids 为空但字段存在，根据 API 定义暂不做操作
            return self.publish_unread_update(user_id).await;
        }

        // 2. 按类型标记
        if let Some(type_enum) = req.r#type {
            match type_enum {
                NotificationApiType::Like => {
                    self.repo
                        .mark_type_read(user_id, NotificationType::Like)
                        .await?
                }
                NotificationApiType::Comment => {
                    self.repo
                        .mark_type_read(user_id, NotificationType::Comment)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, NotificationType::Reply)
                        .await?;
                }
                NotificationApiType::Follow => {
                    self.repo
                        .mark_type_read(user_id, NotificationType::Follow)
                        .await?
                }
                NotificationApiType::Mention => {
                    self.repo
                        .mark_type_read(user_id, NotificationType::Mention)
                        .await?
                }
                NotificationApiType::System => {
                    self.repo
                        .mark_type_read(user_id, NotificationType::System)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, NotificationType::GroupInvite)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, NotificationType::GroupApply)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, NotificationType::GroupKick)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, NotificationType::GroupQuit)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, NotificationType::GroupDismiss)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, NotificationType::GroupAdmin)
                        .await?;
                }
            }
            return self.publish_unread_update(user_id).await;
        }

        // 3. 全部标记已读
        self.repo.mark_all_read(user_id).await?;
        self.publish_unread_update(user_id).await
    }

    async fn publish_unread_update(&self, user_id: i64) -> Result<()> {
        // 计算未读总数
        let total = self.repo.count_unread(user_id).await?;

        // 推送 NotificationReadStateUpdated 事件 (Self-Sync)
        self.event_bus
            .publish(AppEvent::NotificationReadStateUpdated {
                user_id,
                total_unread_count: total as i32,
            });

        Ok(())
    }

    /// 内部调用：创建通知 (由 EventHandler 调用)
    #[allow(clippy::too_many_arguments)]
    pub async fn create_notification(
        &self,
        user_id: i64,
        r#type: NotificationType,
        title: String,
        content: String,
        sender_id: Option<i64>,
        target_type: Option<String>,
        target_id: Option<String>,
        reference_id: Option<String>,
        target_preview: Option<String>,
    ) -> Result<()> {
        // 获取发送者快照
        let (nickname, avatar_url) = if let Some(sid) = sender_id {
            if let Ok(Some(user)) = self.user_repo.find_by_id(sid).await {
                (Some(user.nickname), user.avatar_url)
            } else {
                (None, None)
            }
        } else {
            (None, None)
        };

        self.repo
            .create(
                user_id,
                r#type,
                title,
                content,
                sender_id,
                nickname,
                avatar_url,
                target_type,
                target_id,
                reference_id,
                target_preview,
            )
            .await?;

        Ok(())
    }
}
