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
use tracing::{debug, info};
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
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(
            pagination.and_then(|p| p.limit),
            crate::constants::MAX_PAGE_LIMIT,
        )? as i64;
        let rows = self
            .repo
            .list_by_user(
                user_id,
                req.notification_type,
                pagination.and_then(|p| p.cursor.as_deref()),
                limit,
            )
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
        let started_at = std::time::Instant::now();
        info!(
            user_id,
            requested_count = notification_ids.len(),
            "notification.mark_as_read started"
        );
        if notification_ids.len() > crate::constants::MAX_PAGE_LIMIT as usize {
            return Err(AppError::OutOfRange(format!(
                "notification_ids 最多 {} 个",
                crate::constants::MAX_PAGE_LIMIT
            )));
        }
        let mut uuids: Vec<Uuid> = Vec::with_capacity(notification_ids.len());
        let mut invalid_id_count = 0usize;
        for id in notification_ids {
            match Uuid::parse_str(id) {
                Ok(v) => uuids.push(v),
                Err(_) => invalid_id_count += 1,
            }
        }
        if invalid_id_count > 0 {
            debug!(
                user_id,
                invalid_id_count, "通知标记已读请求包含无法解析的 notification_id，按契约静默忽略"
            );
        }
        if uuids.is_empty() {
            debug!(
                user_id,
                "通知标记已读请求无有效 notification_id，直接返回成功"
            );
            info!(
                user_id,
                requested_count = notification_ids.len(),
                valid_count = 0,
                elapsed_ms = started_at.elapsed().as_millis(),
                "notification.mark_as_read completed (noop_no_valid_ids)"
            );
            return Ok(());
        }

        self.repo.mark_as_read(user_id, &uuids).await?;
        debug!(user_id, marked_count = uuids.len(), "通知标记已读成功");
        info!(
            user_id,
            requested_count = notification_ids.len(),
            valid_count = uuids.len(),
            elapsed_ms = started_at.elapsed().as_millis(),
            "notification.mark_as_read completed"
        );
        Ok(())
    }

    /// 标记全部已读
    pub async fn mark_all_read(&self, user_id: i64) -> Result<()> {
        let started_at = std::time::Instant::now();
        info!(user_id, "notification.mark_all_read started");
        self.repo.mark_all_read(user_id).await?;
        debug!(user_id, "通知全部已读成功");
        info!(
            user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "notification.mark_all_read completed"
        );
        Ok(())
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
            description_preview: preview.to_string(),
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
                moment_summary: moment_summary_from_preview(target_id, target_preview),
            },
            NT::CommentReply => {
                let comment_id = payload
                    .get("comment_id")
                    .and_then(|v| v.as_str())
                    .unwrap_or(reference_id)
                    .to_string();
                NotificationPayload::CommentReply {
                    actor: actor(payload),
                    comment_id,
                    replied_comment_preview: target_preview.to_string(),
                    reply_id: reference_id.to_string(),
                    reply_content_preview: content.to_string(),
                    moment_id: target_id.to_string(),
                    moment_summary: moment_summary_from_preview(target_id, target_preview),
                }
            }
            NT::Mentioned => {
                let target_type = payload
                    .get("target_type")
                    .and_then(|v| v.as_str())
                    .and_then(|s| s.parse::<crate::types::enums::MentionTargetType>().ok())
                    .unwrap_or(crate::types::enums::MentionTargetType::Moment);
                NotificationPayload::Mentioned {
                    actor: actor(payload),
                    target_type,
                    moment_id: target_id.to_string(),
                    moment_summary: moment_summary_from_preview(target_id, target_preview),
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
                group_avatar_url: payload
                    .get("group_avatar_url")
                    .and_then(|v| v.as_str())
                    .map(String::from),
                invite_id: reference_id.to_string(),
                message: payload
                    .get("message")
                    .and_then(|v| v.as_str())
                    .map(String::from),
            },
            NT::GroupInviteAccepted => NotificationPayload::GroupInviteAccepted {
                group_id: target_id.to_string(),
                group_name: content
                    .strip_prefix("接受了群组邀请: ")
                    .unwrap_or(content)
                    .to_string(),
                invite_id: reference_id.to_string(),
                user: actor(payload),
            },
            NT::GroupApplicationReceived => NotificationPayload::GroupApplicationReceived {
                applicant: actor(payload),
                group_id: target_id.to_string(),
                group_name: content
                    .strip_prefix("申请加入群组: ")
                    .unwrap_or(content)
                    .to_string(),
                application_id: reference_id.to_string(),
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
        let started_at = std::time::Instant::now();
        debug!(
            user_id,
            notification_type = ?r#type,
            sender_id = ?sender_id,
            has_target_type = target_type.is_some(),
            has_target_id = target_id.is_some(),
            has_reference_id = reference_id.is_some(),
            "notification.create started"
        );
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
        debug!(
            user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "notification.create completed"
        );

        Ok(())
    }
}
