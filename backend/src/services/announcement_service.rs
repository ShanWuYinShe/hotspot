use futures_util::future;
use std::str::FromStr;
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            AdminAnnouncementListRequest, CreateAnnouncementRequest, DeleteAnnouncementRequest,
            PublicAnnouncementListRequest,
        },
        response::{
            CreateAnnouncementData, PaginatedData, PaginationMeta, SystemAnnouncementItem, UserInfo,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::announcement_repo::AnnouncementRepository,
    types::enums::AnnouncementSeverity,
    utils::validation::Validator,
    EventBus,
};

#[derive(Clone)]
pub struct AnnouncementService {
    repo: AnnouncementRepository,
    event_bus: Arc<EventBus>,
    media_service: crate::services::media_service::MediaService,
}

impl AnnouncementService {
    pub fn new(
        repo: AnnouncementRepository,
        event_bus: Arc<EventBus>,
        media_service: crate::services::media_service::MediaService,
    ) -> Self {
        Self {
            repo,
            event_bus,
            media_service,
        }
    }

    pub async fn create_announcement(
        &self,
        creator_id: i64,
        req: CreateAnnouncementRequest,
    ) -> Result<CreateAnnouncementData> {
        // severity 已是强类型枚举，无需验证

        let announcement = self
            .repo
            .create(
                &req.title,
                &req.content,
                req.severity.as_str(),
                req.action_url.as_deref(),
                req.send_push.unwrap_or(false),
                Some(creator_id),
            )
            .await?;

        if announcement.send_push {
            let severity = AnnouncementSeverity::from_str(&announcement.severity)
                .unwrap_or(AnnouncementSeverity::Info);
            self.event_bus.publish(AppEvent::SystemAnnouncement {
                announcement_id: announcement.announcement_id.to_string(),
                title: announcement.title.clone(),
                content: announcement.content.clone(),
                severity,
                action_url: announcement.action_url.clone(),
                created_at: announcement.created_at.to_rfc3339(),
            });
        }

        Ok(CreateAnnouncementData {
            announcement_id: announcement.announcement_id.to_string(),
            created_at: announcement.created_at.to_rfc3339(),
        })
    }

    pub async fn delete_announcement(&self, req: DeleteAnnouncementRequest) -> Result<()> {
        let announcement_id = Uuid::parse_str(&req.announcement_id)
            .map_err(|_| AppError::BadRequest("无效的公告ID".to_string()))?;

        if self.repo.find_by_id(announcement_id).await?.is_none() {
            return Err(AppError::NotFound("公告不存在".to_string()));
        }

        self.repo.delete(announcement_id).await?;
        Ok(())
    }

    pub async fn list_admin_announcements(
        &self,
        req: AdminAnnouncementListRequest,
    ) -> Result<PaginatedData<SystemAnnouncementItem>> {
        self.list_announcements(req.limit, req.cursor).await
    }

    pub async fn list_public_announcements(
        &self,
        req: PublicAnnouncementListRequest,
    ) -> Result<PaginatedData<SystemAnnouncementItem>> {
        self.list_announcements(req.limit, req.cursor).await
    }

    async fn list_announcements(
        &self,
        limit_opt: Option<i32>,
        cursor: Option<String>,
    ) -> Result<PaginatedData<SystemAnnouncementItem>> {
        let limit = Validator::validate_limit(limit_opt, 50)? as i64;
        let (rows, next_cursor) = self.repo.list(limit, cursor.as_deref()).await?;

        // 使用 futures_util::future::join_all 来并发签名所有 avatar_url
        let items: Vec<SystemAnnouncementItem> =
            future::join_all(rows.into_iter().map(|row| async move {
                let publisher_id = row.publisher_id.unwrap_or_default();
                let created_by = if let (Some(nickname), Some(avatar_url)) =
                    (row.publisher_nickname.clone(), row.publisher_avatar_url.clone())
                {
                    let signed_avatar_url =
                        self.media_service.sign_url_if_path(Some(&avatar_url)).await;
                    UserInfo {
                        user_id: publisher_id,
                        nickname,
                        avatar_url: signed_avatar_url,
                        user_type: crate::types::enums::UserType::System,
                    }
                } else {
                    UserInfo {
                        user_id: publisher_id,
                        nickname: "Admin".to_string(),
                        avatar_url: None,
                        user_type: crate::types::enums::UserType::System,
                    }
                };

                SystemAnnouncementItem {
                    announcement_id: row.announcement_id.to_string(),
                    title: row.title,
                    content: row.content,
                    severity: AnnouncementSeverity::from_str(&row.severity)
                        .unwrap_or(AnnouncementSeverity::Info),
                    action_url: row.action_url,
                    created_at: row.created_at.to_rfc3339(),
                    created_by,
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
}
