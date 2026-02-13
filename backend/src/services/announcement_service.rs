use std::str::FromStr;
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            AdminAnnouncementListRequest, AdminUpdateAnnouncementRequest,
            CreateAnnouncementRequest, DeleteAnnouncementRequest,
        },
        response::{CreateAnnouncementData, PaginatedData, SystemAnnouncementItem},
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

    pub async fn update_announcement(&self, req: AdminUpdateAnnouncementRequest) -> Result<()> {
        let announcement_id = Uuid::parse_str(&req.announcement_id)
            .map_err(|_| AppError::BadRequest("无效的公告ID".to_string()))?;

        if !self.repo.exists(announcement_id).await? {
            return Err(AppError::NotFound("公告不存在".to_string()));
        }

        self.repo
            .update(
                announcement_id,
                req.title.as_deref(),
                req.content.as_deref(),
                req.severity.map(|s| s.as_str()),
                req.is_active,
            )
            .await?;

        Ok(())
    }

    pub async fn delete_announcement(&self, req: DeleteAnnouncementRequest) -> Result<()> {
        let announcement_id = Uuid::parse_str(&req.announcement_id)
            .map_err(|_| AppError::BadRequest("无效的公告ID".to_string()))?;

        if !self.repo.exists(announcement_id).await? {
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

    /// 公开公告列表（非分页，返回条数由调用方传入，通常来自 ResponseDefaults 配置）
    pub async fn list_public_announcements(
        &self,
        count: i64,
    ) -> Result<Vec<SystemAnnouncementItem>> {
        let (rows, _next_cursor) = self.repo.list(count, None).await?;

        let items: Vec<SystemAnnouncementItem> = rows
            .into_iter()
            .map(|row| SystemAnnouncementItem {
                announcement_id: row.announcement_id.to_string(),
                title: row.title,
                content: row.content,
                severity: AnnouncementSeverity::from_str(&row.severity)
                    .unwrap_or(AnnouncementSeverity::Info),
                action_url: row.action_url,
                created_at: row.created_at.to_rfc3339(),
            })
            .collect();

        Ok(items)
    }

    async fn list_announcements(
        &self,
        limit_opt: Option<i32>,
        cursor: Option<String>,
    ) -> Result<PaginatedData<SystemAnnouncementItem>> {
        let limit = Validator::validate_limit(limit_opt, 50)? as i64;
        let (rows, next_cursor) = self.repo.list(limit, cursor.as_deref()).await?;

        let items: Vec<SystemAnnouncementItem> = rows
            .into_iter()
            .map(|row| SystemAnnouncementItem {
                announcement_id: row.announcement_id.to_string(),
                title: row.title,
                content: row.content,
                severity: AnnouncementSeverity::from_str(&row.severity)
                    .unwrap_or(AnnouncementSeverity::Info),
                action_url: row.action_url,
                created_at: row.created_at.to_rfc3339(),
            })
            .collect();

        Ok(PaginatedData {
            items,
            next_cursor,
        })
    }
}
