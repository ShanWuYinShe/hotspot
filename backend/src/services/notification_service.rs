use crate::{
    dto::{
        request::NotificationListRequest,
        response::{
            MomentSummary, NotificationItem, NotificationPayload, PaginatedData, UnreadCountByType,
            UnreadCountResult, UserInfo,
        },
    },
    error::{AppError, Result},
    repositories::notification_repo::{
        NotificationListRow, NotificationRepository, NotificationType as RepoNotificationType,
    },
    repositories::user_repo::UserRepository,
    utils::validation::Validator,
};
use uuid::Uuid;

#[derive(Clone)]
pub struct NotificationService {
    repo: NotificationRepository,
    user_repo: UserRepository,
}

impl NotificationService {
    pub fn new(repo: NotificationRepository, user_repo: UserRepository) -> Self {
        Self { repo, user_repo }
    }

    /// 通知列表
    pub async fn list(
        &self,
        user_id: i64,
        req: NotificationListRequest,
    ) -> Result<PaginatedData<NotificationItem>> {
        let limit = Validator::validate_limit(
            req.limit.and_then(|x: i64| i32::try_from(x).ok()),
            crate::constants::MAX_PAGE_LIMIT,
        )? as i64;
        let rows = self
            .repo
            .list_by_user(user_id, req.notification_type, req.cursor.as_deref(), limit)
            .await?;

        let (items, next_cursor) = crate::utils::pagination::paginate_with_limit_plus_one(
            rows,
            limit as i32,
            |row: &NotificationListRow| {
                crate::utils::pagination::encode_cursor(&(row.created_at, row.notification_id))
            },
        )?;

        let items: Vec<NotificationItem> = items
            .into_iter()
            .filter_map(|row| self.row_to_item(row).ok())
            .collect();

        Ok(PaginatedData { items, next_cursor })
    }

    /// 未读数量
    pub async fn unread_count(&self, user_id: i64) -> Result<UnreadCountResult> {
        let (total, social, interaction, group, system) = self.repo.unread_count(user_id).await?;
        Ok(UnreadCountResult {
            total,
            by_type: UnreadCountByType {
                social,
                interaction,
                group,
                system,
            },
        })
    }

    /// 标记已读
    pub async fn mark_as_read(&self, user_id: i64, notification_ids: &[String]) -> Result<()> {
        if notification_ids.len() > crate::constants::MAX_PAGE_LIMIT as usize {
            return Err(AppError::OutOfRange(format!(
                "notification_ids 最多 {} 个",
                crate::constants::MAX_PAGE_LIMIT
            )));
        }
        let uuids: Vec<Uuid> = notification_ids
            .iter()
            .filter_map(|s| Uuid::parse_str(s).ok())
            .collect();
        self.repo.mark_as_read(user_id, &uuids).await
    }

    /// 标记全部已读
    pub async fn mark_all_read(&self, user_id: i64) -> Result<()> {
        self.repo.mark_all_read(user_id).await
    }

    fn row_to_item(&self, row: NotificationListRow) -> Result<NotificationItem> {
        let ntype: RepoNotificationType = row
            .notification_type
            .parse()
            .map_err(|_| AppError::InternalError("无效的 notification_type".to_string()))?;

        let payload = Self::payload_from_row(&ntype, &row.payload)?;

        Ok(NotificationItem {
            notification_id: row.notification_id.to_string(),
            notification_type: ntype,
            is_read: row.is_read,
            created_at: row.created_at.to_rfc3339(),
            payload,
        })
    }

    fn payload_from_row(
        ntype: &RepoNotificationType,
        payload: &serde_json::Value,
    ) -> Result<NotificationPayload> {
        let actor = |payload: &serde_json::Value| -> UserInfo {
            UserInfo::new(
                payload
                    .get("sender_id")
                    .and_then(|v| v.as_i64())
                    .unwrap_or(0),
                payload
                    .get("sender_nickname")
                    .and_then(|v| v.as_str())
                    .unwrap_or(""),
                payload
                    .get("sender_avatar_url")
                    .and_then(|v| v.as_str())
                    .map(String::from),
            )
        };

        let target_id = payload
            .get("target_id")
            .and_then(|v| v.as_str())
            .unwrap_or("");
        let reference_id = payload
            .get("reference_id")
            .and_then(|v| v.as_str())
            .unwrap_or("");
        let target_preview = payload
            .get("target_preview")
            .and_then(|v| v.as_str())
            .unwrap_or("");
        let content = payload
            .get("content")
            .and_then(|v| v.as_str())
            .unwrap_or("");

        let moment_summary_from_preview = |moment_id: &str, preview: &str| MomentSummary {
            moment_id: moment_id.to_string(),
            cover_url: Some(preview.to_string()).filter(|s| !s.is_empty()),
            text_preview: preview.to_string(),
        };

        use crate::types::enums::NotificationType as NT;
        let payload = match ntype {
            NT::NewFollower => NotificationPayload::NewFollower {
                actor: actor(payload),
                source: payload
                    .get("target_type")
                    .and_then(|v| v.as_str())
                    .and_then(|s| s.parse::<crate::types::enums::FollowSource>().ok()),
                source_moment_id: payload
                    .get("target_id")
                    .and_then(|v| v.as_str())
                    .map(String::from),
            },
            NT::MutualFollow => NotificationPayload::MutualFollow {
                actor: actor(payload),
            },
            NT::MomentLiked => NotificationPayload::MomentLiked {
                actor: actor(payload),
                moment_id: target_id.to_string(),
                moment_summary: moment_summary_from_preview(target_id, target_preview),
            },
            NT::MomentCommented => NotificationPayload::MomentCommented {
                actor: actor(payload),
                moment_id: target_id.to_string(),
                moment_summary: moment_summary_from_preview(target_id, target_preview),
                comment_id: reference_id.to_string(),
                comment_preview: content.to_string(),
            },
            NT::CommentLiked => NotificationPayload::CommentLiked {
                actor: actor(payload),
                comment_id: reference_id.to_string(),
                comment_preview: target_preview.to_string(),
                moment_id: target_id.to_string(),
                moment_summary: Some(moment_summary_from_preview(target_id, target_preview)),
            },
            NT::CommentReply => NotificationPayload::CommentReply {
                actor: actor(payload),
                comment_id: reference_id.to_string(),
                comment_preview: target_preview.to_string(),
                moment_id: target_id.to_string(),
                moment_summary: Some(moment_summary_from_preview(target_id, target_preview)),
            },
            NT::Mentioned => {
                let target_type = payload
                    .get("target_type")
                    .and_then(|v| v.as_str())
                    .unwrap_or("moment")
                    .to_string();
                NotificationPayload::Mentioned {
                    actor: actor(payload),
                    target_type,
                    moment_id: target_id.to_string(),
                    moment_summary: Some(moment_summary_from_preview(target_id, target_preview)),
                    comment_id: if reference_id.is_empty() {
                        None
                    } else {
                        Some(reference_id.to_string())
                    },
                    comment_preview: if content.is_empty() {
                        None
                    } else {
                        Some(content.to_string())
                    },
                }
            }
            NT::GroupInviteReceived => NotificationPayload::GroupInviteReceived {
                actor: actor(payload),
                group_id: target_id.to_string(),
                group_name: content
                    .strip_prefix("邀请你加入群组: ")
                    .unwrap_or(content)
                    .to_string(),
                invite_id: reference_id.to_string(),
            },
            NT::GroupApplicationAccepted => NotificationPayload::GroupApplicationAccepted {
                group_id: target_id.to_string(),
                group_name: content
                    .strip_prefix("已通过加入群组: ")
                    .unwrap_or(content)
                    .to_string(),
                application_id: reference_id.to_string(),
            },
            NT::SystemAnnouncement => NotificationPayload::SystemAnnouncement {
                announcement_id: reference_id.to_string(),
                severity: payload
                    .get("severity")
                    .and_then(|v| v.as_str())
                    .and_then(|s| s.parse::<crate::types::enums::AnnouncementSeverity>().ok())
                    .unwrap_or(crate::types::enums::AnnouncementSeverity::Info),
            },
        };

        Ok(payload)
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
        extra_payload: Option<serde_json::Value>,
    ) -> Result<()> {
        // 获取发送者快照
        let (nickname, avatar_url) = if let Some(sid) = sender_id {
            if let Ok(Some(user)) = self.user_repo.find_by_id(sid).await {
                (user.nickname, user.avatar_url)
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
                extra_payload,
            )
            .await?;

        Ok(())
    }
}
