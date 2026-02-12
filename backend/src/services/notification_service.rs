use crate::{
    error::Result,
    repositories::notification_repo::{
        NotificationRepository, NotificationType as RepoNotificationType,
    },
    repositories::user_repo::UserRepository,
};

#[derive(Clone)]
pub struct NotificationService {
    repo: NotificationRepository,
    user_repo: UserRepository,
}

impl NotificationService {
    pub fn new(repo: NotificationRepository, user_repo: UserRepository) -> Self {
        Self { repo, user_repo }
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
            )
            .await?;

        Ok(())
    }
}
