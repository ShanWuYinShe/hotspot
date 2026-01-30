use futures_util::future;
use std::str::FromStr;
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    dto::{
        request::{GetNotificationsRequest, MarkNotificationsReadRequest},
        response::{
            NotificationItem, PaginatedData, PaginationMeta, UnreadNotificationCountData, UserInfo,
        },
        types::StringOrNumberId,
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::notification_repo::{
        NotificationRepository, NotificationType as RepoNotificationType,
    },
    repositories::user_repo::UserRepository,
    types::enums::{NotificationFilterType, NotificationTargetType, NotificationType},
    utils::validation::Validator,
    EventBus,
};

/// 将 Repository 层的 NotificationType 转换为 DTO 层的 NotificationType
fn map_repo_notification_type(repo_type: &RepoNotificationType) -> NotificationType {
    match repo_type {
        RepoNotificationType::MomentLike => NotificationType::MomentLike,
        RepoNotificationType::MomentComment => NotificationType::MomentComment,
        RepoNotificationType::CommentReply => NotificationType::CommentReply,
        RepoNotificationType::NewFollower => NotificationType::NewFollower,
        RepoNotificationType::MutualFollow => NotificationType::MutualFollow,
        RepoNotificationType::FollowRequest => NotificationType::FollowRequest,
        RepoNotificationType::FollowRequestHandled => NotificationType::FollowRequestHandled,
        RepoNotificationType::GroupInvite => NotificationType::GroupInvite,
        RepoNotificationType::GroupInviteResponse => NotificationType::GroupInviteResponse,
        RepoNotificationType::GroupApplication => NotificationType::GroupApplication,
        RepoNotificationType::GroupApplicationResponse => {
            NotificationType::GroupApplicationResponse
        }
        RepoNotificationType::GroupAnnouncement => NotificationType::GroupAnnouncement,
        RepoNotificationType::SystemAnnouncement => NotificationType::SystemAnnouncement,
        RepoNotificationType::MessageMention => NotificationType::MessageMention,
    }
}

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

        let type_filters = req.r#type.map(|filter| {
            let types = match filter {
                NotificationFilterType::Like => vec![NotificationType::MomentLike],
                NotificationFilterType::Comment => {
                    vec![
                        NotificationType::MomentComment,
                        NotificationType::CommentReply,
                    ]
                }
                NotificationFilterType::Follow => vec![
                    NotificationType::NewFollower,
                    NotificationType::MutualFollow,
                    NotificationType::FollowRequest,
                    NotificationType::FollowRequestHandled,
                ],
                NotificationFilterType::System => vec![
                    NotificationType::GroupInvite,
                    NotificationType::GroupInviteResponse,
                    NotificationType::GroupApplication,
                    NotificationType::GroupApplicationResponse,
                    NotificationType::GroupAnnouncement,
                    NotificationType::SystemAnnouncement,
                ],
                NotificationFilterType::Mention => vec![NotificationType::MessageMention],
            };
            types.into_iter().map(|t| t.to_string()).collect()
        });

        let (rows, next_cursor) = self
            .repo
            .list_by_user(user_id, limit, cursor, type_filters)
            .await?;

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

                let (target_text_preview, target_media_url) = match row.target_preview.clone() {
                    Some(preview) => {
                        if preview.starts_with("uploads/") {
                            let signed = self.media_service.sign_url_if_path(Some(&preview)).await;
                            (None, signed)
                        } else if preview.starts_with("http://") || preview.starts_with("https://")
                        {
                            (None, Some(preview))
                        } else {
                            (Some(preview), None)
                        }
                    }
                    None => (None, None),
                };

                let target_type = parse_notification_target_type(row.target_type)?;
                let target_id = parse_notification_target_id(target_type, row.target_id)?;

                Ok(NotificationItem {
                    notification_id: row.notification_id.to_string(),
                    notification_type: map_repo_notification_type(&row.r#type),
                    title: Some(row.title),
                    content: Some(row.content),
                    created_at: row.created_at.to_rfc3339(),
                    is_read: row.is_read,
                    sender,
                    target_type,
                    target_id,
                    reference_id: row.reference_id,
                    target_text_preview,
                    target_media_url,
                })
            }))
            .await
            .into_iter()
            .collect::<Result<Vec<_>>>()?;

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
                RepoNotificationType::MomentLike => breakdown.like += count,
                RepoNotificationType::MomentComment | RepoNotificationType::CommentReply => {
                    breakdown.comment += count
                }
                RepoNotificationType::NewFollower
                | RepoNotificationType::MutualFollow
                | RepoNotificationType::FollowRequest
                | RepoNotificationType::FollowRequestHandled => breakdown.follow += count,
                RepoNotificationType::MessageMention => breakdown.mention += count,
                RepoNotificationType::GroupInvite
                | RepoNotificationType::GroupInviteResponse
                | RepoNotificationType::GroupApplication
                | RepoNotificationType::GroupApplicationResponse
                | RepoNotificationType::GroupAnnouncement
                | RepoNotificationType::SystemAnnouncement => breakdown.system += count,
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
                NotificationFilterType::Like => {
                    self.repo
                        .mark_type_read(user_id, RepoNotificationType::MomentLike)
                        .await?
                }
                NotificationFilterType::Comment => {
                    self.repo
                        .mark_type_read(user_id, RepoNotificationType::MomentComment)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, RepoNotificationType::CommentReply)
                        .await?;
                }
                NotificationFilterType::Follow => {
                    self.repo
                        .mark_type_read(user_id, RepoNotificationType::NewFollower)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, RepoNotificationType::MutualFollow)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, RepoNotificationType::FollowRequest)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, RepoNotificationType::FollowRequestHandled)
                        .await?;
                }
                NotificationFilterType::Mention => {
                    self.repo
                        .mark_type_read(user_id, RepoNotificationType::MessageMention)
                        .await?
                }
                NotificationFilterType::System => {
                    self.repo
                        .mark_type_read(user_id, RepoNotificationType::GroupInvite)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, RepoNotificationType::GroupInviteResponse)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, RepoNotificationType::GroupApplication)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, RepoNotificationType::GroupApplicationResponse)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, RepoNotificationType::GroupAnnouncement)
                        .await?;
                    self.repo
                        .mark_type_read(user_id, RepoNotificationType::SystemAnnouncement)
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
        r#type: RepoNotificationType,
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

fn parse_notification_target_type(raw: Option<String>) -> Result<Option<NotificationTargetType>> {
    match raw {
        None => Ok(None),
        Some(value) => NotificationTargetType::from_str(&value)
            .map(Some)
            .map_err(|_| AppError::InternalError("通知数据不合法: target_type 无效".to_string())),
    }
}

/// 解析通知目标 ID
fn parse_notification_target_id(
    target_type: Option<NotificationTargetType>,
    target_id: Option<String>,
) -> Result<Option<StringOrNumberId>> {
    match (target_type, target_id) {
        (Some(NotificationTargetType::User), Some(value)) => value
            .parse::<i64>()
            .map(|id| Some(StringOrNumberId::Number(id)))
            .map_err(|_| {
                AppError::InternalError("通知数据不合法: user target_id 非数字".to_string())
            }),
        (Some(NotificationTargetType::User), None) => Ok(None),
        (_, Some(value)) => Ok(Some(StringOrNumberId::String(value))),
        (_, None) => Ok(None),
    }
}
