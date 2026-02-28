use std::{collections::HashMap, str::FromStr, sync::Arc};
use tracing::{info, warn};

use crate::{
    constants::UNKNOWN_USER_NICKNAME,
    dto::{
        request::{
            AdminAnnouncementListRequest, AdminUpdateAnnouncementRequest,
            CreateAnnouncementRequest, DeleteAnnouncementRequest,
        },
        response::{
            AdminAnnouncementItem, CreateAnnouncementData, PaginatedData, SystemAnnouncementItem,
            UserInfo,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::{announcement_repo::AnnouncementRepository, user::user_repo::UserRepository},
    services::media_service::MediaService,
    types::enums::AnnouncementSeverity,
    utils::validation::Validator,
    EventBus,
};

#[derive(Clone)]
pub struct AnnouncementService {
    repo: AnnouncementRepository,
    user_repo: UserRepository,
    media_service: MediaService,
    event_bus: Arc<EventBus>,
}

impl AnnouncementService {
    pub fn new(
        repo: AnnouncementRepository,
        user_repo: UserRepository,
        media_service: MediaService,
        event_bus: Arc<EventBus>,
    ) -> Self {
        Self {
            repo,
            user_repo,
            media_service,
            event_bus,
        }
    }

    pub async fn create_announcement(
        &self,
        creator_id: i64,
        req: CreateAnnouncementRequest,
    ) -> Result<CreateAnnouncementData> {
        let started_at = std::time::Instant::now();
        info!(
            creator_id,
            severity = ?req.severity,
            title_len = req.title.chars().count(),
            content_len = req.content.chars().count(),
            send_push = req.send_push.unwrap_or(false),
            "admin_announcement.create started"
        );
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
        info!(
            creator_id,
            announcement_id = %announcement.announcement_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "admin_announcement.create completed"
        );

        Ok(CreateAnnouncementData {
            announcement_id: announcement.announcement_id.to_string(),
        })
    }

    pub async fn update_announcement(&self, req: AdminUpdateAnnouncementRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let announcement_id = Validator::parse_uuid(&req.announcement_id, "公告ID")?;
        info!(
            announcement_id = %announcement_id,
            has_title = req.title.is_some(),
            has_content = req.content.is_some(),
            has_severity = req.severity.is_some(),
            has_action_url = req.action_url.is_some(),
            "admin_announcement.update started"
        );

        if !self.repo.exists(announcement_id).await? {
            warn!(announcement_id = %announcement_id, "admin_announcement.update rejected: not found");
            return Err(AppError::NotFound("公告不存在".to_string()));
        }

        self.repo
            .update(
                announcement_id,
                req.title.as_deref(),
                req.content.as_deref(),
                req.severity.map(|s| s.as_str()),
                req.action_url.as_deref(),
            )
            .await?;
        info!(
            announcement_id = %announcement_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "admin_announcement.update completed"
        );

        Ok(())
    }

    pub async fn delete_announcement(&self, req: DeleteAnnouncementRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        let announcement_id = Validator::parse_uuid(&req.announcement_id, "公告ID")?;
        info!(announcement_id = %announcement_id, "admin_announcement.delete started");

        if !self.repo.exists(announcement_id).await? {
            warn!(announcement_id = %announcement_id, "admin_announcement.delete rejected: not found");
            return Err(AppError::NotFound("公告不存在".to_string()));
        }

        self.repo.delete(announcement_id).await?;
        info!(
            announcement_id = %announcement_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "admin_announcement.delete completed"
        );
        Ok(())
    }

    pub async fn list_admin_announcements(
        &self,
        req: AdminAnnouncementListRequest,
    ) -> Result<PaginatedData<AdminAnnouncementItem>> {
        let pagination = req.pagination.as_ref();
        let limit_input = pagination.and_then(|p| p.limit).unwrap_or(20);
        let limit = Validator::validate_limit(Some(limit_input), 50)? as i64;
        let cursor = pagination.and_then(|p| p.cursor.as_deref());

        let (rows, next_cursor) = self
            .repo
            .list_admin(req.severity.map(|s| s.as_str()), limit, cursor)
            .await?;

        let publishers: Vec<i64> = rows.iter().filter_map(|row| row.publisher_id).collect();
        let user_map = self.load_user_map(&publishers).await;

        let items = rows
            .into_iter()
            .map(|row| {
                let announcement = SystemAnnouncementItem {
                    announcement_id: row.announcement_id.to_string(),
                    title: row.title.clone(),
                    content: row.content.clone(),
                    severity: AnnouncementSeverity::from_str(&row.severity)
                        .unwrap_or(AnnouncementSeverity::Info),
                    action_url: row.action_url.clone(),
                    created_at: row.created_at.to_rfc3339(),
                };
                let created_by = row
                    .publisher_id
                    .and_then(|id| user_map.get(&id).cloned())
                    .unwrap_or_else(|| UserInfo::new(0, "System", None));
                AdminAnnouncementItem {
                    announcement,
                    created_by,
                    send_push: row.send_push,
                    push_sent_at: row.push_sent_at.map(|ts| ts.to_rfc3339()),
                    view_count: row.view_count,
                }
            })
            .collect();

        Ok(PaginatedData { items, next_cursor })
    }

    /// 公开公告列表（非分页，返回条数由调用方传入，通常来自 ResponseDefaults 配置）
    pub async fn list_public_announcements(
        &self,
        count: i64,
    ) -> Result<Vec<SystemAnnouncementItem>> {
        let (rows, _next_cursor) = self.repo.list(None, count, None).await?;

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

    async fn load_user_map(&self, user_ids: &[i64]) -> HashMap<i64, UserInfo> {
        let mut map = HashMap::new();
        if user_ids.is_empty() {
            return map;
        }

        if let Ok(profiles) = self.user_repo.find_profiles(user_ids).await {
            for profile in profiles {
                let avatar_url = self
                    .media_service
                    .sign_url_if_path(profile.avatar_url.as_deref())
                    .await;
                let nickname = profile
                    .nickname
                    .clone()
                    .unwrap_or_else(|| UNKNOWN_USER_NICKNAME.to_string());
                map.insert(
                    profile.user_id,
                    UserInfo::new(profile.user_id, nickname, avatar_url),
                );
            }
        }

        map
    }
}
