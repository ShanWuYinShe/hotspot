//! 群组服务层实现
//!
//! 核心业务逻辑：群组生命周期、成员管理、申请审核

use std::sync::Arc;

use crate::{
    repositories::{
        chat_unread_repo::ChatUnreadRepository,
        group_announcement_repo::GroupAnnouncementRepository,
        group_application_repo::GroupApplicationRepository, group_bans_repo::GroupBansRepository,
        group_invite_repo::GroupInviteRepository,
        group_member_history_repo::GroupMemberHistoryRepository,
        group_member_repo::GroupMemberRepository, group_repo::GroupRepository,
        group_settings_repo::GroupSettingsRepository, system::report_repo::ReportRepository,
        tag_repo::TagRepository, user_repo::UserRepository,
    },
    utils::cache::CacheManager,
    EventBus,
};

mod announcements;
mod applications;
mod crud;
mod helpers;
mod members;

#[derive(Clone)]
pub struct GroupService {
    pub(crate) event_bus: Arc<EventBus>,
    pub(crate) group_repo: GroupRepository,
    pub(crate) member_repo: GroupMemberRepository,
    pub(crate) member_history_repo: GroupMemberHistoryRepository,
    pub(crate) application_repo: GroupApplicationRepository,
    pub(crate) bans_repo: GroupBansRepository,
    pub(crate) invite_repo: GroupInviteRepository,
    pub(crate) settings_repo: GroupSettingsRepository,
    pub(crate) announcement_repo: GroupAnnouncementRepository,
    pub(crate) tag_repo: TagRepository,
    pub(crate) report_repo: ReportRepository,
    pub(crate) user_repo: UserRepository,
    pub(crate) unread_repo: ChatUnreadRepository,
    pub(crate) media_service: crate::services::media_service::MediaService,
    pub(crate) cache_manager: CacheManager,
}

impl GroupService {
    pub fn new(
        event_bus: Arc<EventBus>,
        group_repo: GroupRepository,
        member_repo: GroupMemberRepository,
        member_history_repo: GroupMemberHistoryRepository,
        application_repo: GroupApplicationRepository,
        bans_repo: GroupBansRepository,
        invite_repo: GroupInviteRepository,
        settings_repo: GroupSettingsRepository,
        announcement_repo: GroupAnnouncementRepository,
        tag_repo: TagRepository,
        report_repo: ReportRepository,
        user_repo: UserRepository,
        unread_repo: ChatUnreadRepository,
        media_service: crate::services::media_service::MediaService,
        cache_manager: CacheManager,
    ) -> Self {
        Self {
            event_bus,
            group_repo,
            member_repo,
            member_history_repo,
            application_repo,
            bans_repo,
            invite_repo,
            settings_repo,
            announcement_repo,
            tag_repo,
            report_repo,
            user_repo,
            unread_repo,
            media_service,
            cache_manager,
        }
    }
}
