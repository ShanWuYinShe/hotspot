use std::collections::HashMap;

use super::GroupService;
use crate::{
    constants,
    dto::{
        request::{
            CreateGroupAnnouncementRequest, DeleteGroupAnnouncementRequest,
            GroupAnnouncementsRequest, UpdateGroupAnnouncementRequest,
        },
        response::{GroupAnnouncementItem, UserInfo},
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    utils::validation::Validator,
};

impl GroupService {
    pub async fn create_announcement(
        &self,
        user_id: i64,
        req: &CreateGroupAnnouncementRequest,
    ) -> Result<GroupAnnouncementItem> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let _ = self.require_admin_or_owner(group_id, user_id).await?;

        let announcement = self
            .announcement_repo
            .create(
                group_id,
                req.title.clone(),
                req.content.clone(),
                user_id,
                req.is_pinned.unwrap_or(false),
            )
            .await?;

        let created_by = self.build_user_info(user_id).await?;
        self.event_bus.publish(AppEvent::GroupAnnouncementCreated {
            announcement_id: announcement.announcement_id,
            group_id,
            title: announcement.title.clone(),
            content: announcement.content.clone(),
            created_by_user_id: user_id,
            created_by_nickname: Some(created_by.nickname.clone()),
            created_by_avatar_url: created_by.avatar_url.clone(),
            is_pinned: announcement.is_pinned,
        });

        Ok(GroupAnnouncementItem {
            announcement_id: announcement.announcement_id.to_string(),
            group_id: group_id.to_string(),
            title: announcement.title,
            content: announcement.content,
            created_by,
            created_at: announcement.created_at.to_rfc3339(),
            is_pinned: announcement.is_pinned,
        })
    }

    pub async fn list_announcements(
        &self,
        req: &GroupAnnouncementsRequest,
    ) -> Result<crate::dto::response::PaginatedData<GroupAnnouncementItem>> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let limit = constants::DEFAULT_PAGE_LIMIT;
        let announcements = self
            .announcement_repo
            .find_by_group_id(group_id, limit)
            .await?;

        let user_ids: Vec<i64> = announcements.iter().map(|a| a.publisher_id).collect();
        let users = self.user_repo.find_profiles(&user_ids).await?;
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        let mut items = Vec::with_capacity(announcements.len());
        for a in announcements {
            let user = user_map.get(&a.publisher_id);
            let avatar_url = self
                .media_service
                .sign_url_if_path(user.and_then(|u| u.avatar_url.as_deref()))
                .await;
            items.push(GroupAnnouncementItem {
                announcement_id: a.announcement_id.to_string(),
                group_id: group_id.to_string(),
                title: a.title,
                content: a.content,
                created_by: UserInfo::new(
                    a.publisher_id,
                    user.and_then(|u| u.nickname.clone())
                        .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                    avatar_url,
                ),
                created_at: a.created_at.to_rfc3339(),
                is_pinned: a.is_pinned,
            });
        }

        Ok(crate::dto::response::PaginatedData {
            items,
            next_cursor: None,
        })
    }

    pub async fn update_announcement(
        &self,
        user_id: i64,
        req: &UpdateGroupAnnouncementRequest,
    ) -> Result<GroupAnnouncementItem> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let announcement_id = Validator::parse_uuid(&req.announcement_id, "公告ID")?;
        let announcement = self
            .announcement_repo
            .find_by_id(announcement_id)
            .await?
            .ok_or_else(|| AppError::NotFound("公告不存在".to_string()))?;

        if announcement.group_id != group_id {
            return Err(AppError::BadRequest("公告不属于该群组".to_string()));
        }

        let _ = self
            .require_admin_or_owner(announcement.group_id, user_id)
            .await?;

        self.announcement_repo
            .update(
                announcement_id,
                req.title.clone(),
                req.content.clone(),
                req.is_pinned,
            )
            .await?;

        self.event_bus.publish(AppEvent::GroupAnnouncementUpdated {
            announcement_id,
            group_id: announcement.group_id,
            title: req.title.clone(),
            content: req.content.clone(),
            is_pinned: req.is_pinned,
        });

        Ok(GroupAnnouncementItem {
            announcement_id: announcement_id.to_string(),
            group_id: announcement.group_id.to_string(),
            title: req.title.clone().unwrap_or(announcement.title),
            content: req.content.clone().unwrap_or(announcement.content),
            created_by: self.build_user_info(announcement.publisher_id).await?,
            created_at: announcement.created_at.to_rfc3339(),
            is_pinned: req.is_pinned.unwrap_or(announcement.is_pinned),
        })
    }

    pub async fn delete_announcement(
        &self,
        user_id: i64,
        req: &DeleteGroupAnnouncementRequest,
    ) -> Result<()> {
        let announcement_id = Validator::parse_uuid(&req.announcement_id, "公告ID")?;
        let announcement = self
            .announcement_repo
            .find_by_id(announcement_id)
            .await?
            .ok_or_else(|| AppError::NotFound("公告不存在".to_string()))?;

        let _ = self
            .require_admin_or_owner(announcement.group_id, user_id)
            .await?;

        self.announcement_repo.delete(announcement_id).await?;
        self.event_bus.publish(AppEvent::GroupAnnouncementDeleted {
            announcement_id,
            group_id: announcement.group_id,
        });
        Ok(())
    }
}
