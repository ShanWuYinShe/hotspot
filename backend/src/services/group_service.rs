//! 群组服务层实现
//!
//! 核心业务逻辑：群组生命周期、成员管理、申请审核

use chrono::{Duration, Utc};
use redis::aio::ConnectionManager as RedisConnectionManager;
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            AcceptInviteRequest, AdminDismissGroupRequest, AdminGroupDetailRequest,
            AdminGroupSearchRequest, ApplyToGroupRequest, ApproveGroupApplicationRequest,
            BanMemberRequest, BannedMembersRequest, CancelApplicationRequest,
            CreateGroupAnnouncementRequest, CreateGroupChatRequest, DeleteGroupAnnouncementRequest,
            DeleteGroupChatRequest, GetMyGroupStateRequest, GroupAnnouncementsRequest,
            GroupApplicationsRequest, GroupMembersRequest, GroupOpenTheDoorRequest,
            GroupSettingsRequest, InviteMemberRequest, JoinGroupRequest, LeaveGroupRequest,
            MuteMemberRequest, MyGroupChatsRequest, MyInvitesRequest,
            RejectGroupApplicationRequest, RejectInviteRequest, RemoveMemberRequest,
            SearchGroupsByKeywordRequest, SearchGroupsByLocationRequest, SearchGroupsByTagRequest,
            SetMemberRoleRequest, TransferOwnershipRequest, UnbanMemberRequest,
            UnmuteMemberRequest, UpdateGroupAnnouncementRequest, UpdateGroupChatRequest,
            UpdateGroupMySettingsRequest,
        },
        response::{
            AdminGroupDetail, AdminGroupSearchItem, AdminGroupSettingsSummary, AdminGroupStats,
            ApplyToGroupData, BannedMemberItem, GroupAnnouncementItem, GroupApplicationItem,
            GroupDetailData, GroupInfo, GroupInviteItem, GroupMemberItem, GroupMyStateData,
            GroupSearchItem, GroupSettings as GroupSettingsDto, GroupSettingsData, InviteMemberData,
            MyGroupItem, PaginatedData, PaginationMeta, RelativeLocation, UserInfo,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::{
        chat_unread_repo::ChatUnreadRepository,
        group_announcement_repo::GroupAnnouncementRepository,
        group_application_repo::GroupApplicationRepository,
        group_bans_repo::GroupBansRepository,
        group_invite_repo::GroupInviteRepository,
        group_member_history_repo::GroupMemberHistoryRepository,
        group_member_repo::{GroupMemberDetailRow, GroupMemberRepository},
        group_message_repo::GroupMessageRepository,
        group_repo::{GroupRepository, GroupSearchRow},
        group_settings_repo::GroupSettingsRepository,
        tag_repo::TagRepository,
        user_repo::UserRepository,
    },
    types::enums::{
        ApplicationStatus, ApplicationStatusFilter, GroupJoinMethod, GroupRole, MemberLeaveReason,
        ResponseStatus, SetableGroupRole, UserType,
    },
    utils::{
        cache::{cache_keys, CacheManager},
        password::PasswordUtil,
        validation::Validator,
    },
    EventBus,
};

#[derive(Clone)]
pub struct GroupService {
    event_bus: Arc<EventBus>,
    group_repo: GroupRepository,
    message_repo: GroupMessageRepository,
    member_repo: GroupMemberRepository,
    member_history_repo: GroupMemberHistoryRepository,
    application_repo: GroupApplicationRepository,
    bans_repo: GroupBansRepository,
    invite_repo: GroupInviteRepository,
    settings_repo: GroupSettingsRepository,
    announcement_repo: GroupAnnouncementRepository,
    tag_repo: TagRepository,
    user_repo: UserRepository,
    unread_repo: ChatUnreadRepository,
    redis: RedisConnectionManager,
    presence_service: crate::services::presence_service::PresenceService,
    media_service: crate::services::media_service::MediaService,
    cache_manager: CacheManager,
}

impl GroupService {
    pub fn new(
        event_bus: Arc<EventBus>,
        group_repo: GroupRepository,
        message_repo: GroupMessageRepository,
        member_repo: GroupMemberRepository,
        member_history_repo: GroupMemberHistoryRepository,
        application_repo: GroupApplicationRepository,
        bans_repo: GroupBansRepository,
        invite_repo: GroupInviteRepository,
        settings_repo: GroupSettingsRepository,
        announcement_repo: GroupAnnouncementRepository,
        tag_repo: TagRepository,
        user_repo: UserRepository,
        unread_repo: ChatUnreadRepository,
        redis: RedisConnectionManager,
        presence_service: crate::services::presence_service::PresenceService,
        media_service: crate::services::media_service::MediaService,
        cache_manager: CacheManager,
    ) -> Self {
        Self {
            event_bus,
            group_repo,
            message_repo,
            member_repo,
            member_history_repo,
            application_repo,
            bans_repo,
            invite_repo,
            settings_repo,
            announcement_repo,
            tag_repo,
            user_repo,
            unread_repo,
            redis,
            presence_service,
            media_service,
            cache_manager,
        }
    }

    fn parse_group_id(group_chat_id: &str) -> Result<i64> {
        group_chat_id
            .parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))
    }

    async fn build_user_info(&self, user_id: i64) -> Result<UserInfo> {
        let user = self.user_repo.find_by_id(user_id).await?;
        if let Some(u) = user {
            let avatar_url = self.media_service.sign_url_if_path(u.avatar_url.as_deref()).await;
            Ok(UserInfo {
                user_id: u.user_id,
                nickname: u.nickname.unwrap_or_else(|| "未知用户".to_string()),
                avatar_url,
                user_type: u.user_type,
            })
        } else {
            Ok(UserInfo {
                user_id,
                nickname: "未知用户".to_string(),
                avatar_url: None,
                user_type: UserType::Normal,
            })
        }
    }

    fn join_method_to_string(join_method: GroupJoinMethod) -> String {
        join_method.as_str().to_string()
    }

    fn join_method_from_str(join_method: &str) -> GroupJoinMethod {
        match join_method {
            "open" => GroupJoinMethod::Open,
            "approval" => GroupJoinMethod::Approval,
            "invite_only" => GroupJoinMethod::InviteOnly,
            "free" => GroupJoinMethod::Open,
            "apply" => GroupJoinMethod::Approval,
            "invite" => GroupJoinMethod::InviteOnly,
            _ => GroupJoinMethod::Open,
        }
    }

    fn join_type_from_join_method(join_method: GroupJoinMethod) -> String {
        match join_method {
            GroupJoinMethod::Open => "free".to_string(),
            GroupJoinMethod::Approval => "apply".to_string(),
            GroupJoinMethod::InviteOnly => "invite".to_string(),
        }
    }

    async fn build_location(&self, distance: Option<f64>) -> RelativeLocation {
        RelativeLocation {
            latitude: 0.0,
            longitude: 0.0,
            location_name: "未知".to_string(),
            distance,
            bearing_degrees: None,
        }
    }

    // ==================== 搜索与发现 ====================

    pub async fn search_by_keyword(
        &self,
        req: SearchGroupsByKeywordRequest,
    ) -> Result<PaginatedData<GroupSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let (groups, next_cursor) = self
            .group_repo
            .search_by_keyword(
                &req.kw,
                req.reference_point.longitude,
                req.reference_point.latitude,
                limit,
                req.cursor,
            )
            .await?;

        let items = self.map_search_rows(groups).await?;
        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    pub async fn search_by_tag(
        &self,
        req: SearchGroupsByTagRequest,
    ) -> Result<PaginatedData<GroupSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let (groups, next_cursor) = self
            .group_repo
            .search_by_tag(
                &req.tag,
                req.reference_point.longitude,
                req.reference_point.latitude,
                limit,
                req.cursor,
            )
            .await?;

        let items = self.map_search_rows(groups).await?;
        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    pub async fn search_by_location(
        &self,
        req: SearchGroupsByLocationRequest,
    ) -> Result<PaginatedData<GroupSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let (groups, next_cursor) = self
            .group_repo
            .search_nearby(
                req.reference_point.longitude,
                req.reference_point.latitude,
                req.radius,
                limit,
                req.cursor,
            )
            .await?;

        let items = self.map_search_rows(groups).await?;
        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    async fn map_search_rows(&self, groups: Vec<GroupSearchRow>) -> Result<Vec<GroupSearchItem>> {
        let mut items = Vec::with_capacity(groups.len());
        for g in groups {
            let tags = self.tag_repo.get_group_tags(g.group_id).await.ok();
            let settings = self.settings_repo.find_by_group(g.group_id).await?;
            let has_password = settings.as_ref().and_then(|s| s.password_hash.as_ref()).is_some();
            let location = self.build_location(g.distance).await;
            items.push(GroupSearchItem {
                group_chat_id: g.group_id.to_string(),
                name: g.name,
                description: g.description,
                avatar_url: self.media_service.sign_url_if_path(g.avatar_url.as_deref()).await,
                tags,
                members_count: g.member_count,
                has_password,
                location,
            });
        }
        Ok(items)
    }

    // ==================== 群组详情/设置 ====================

    pub async fn open_the_door(
        &self,
        req: &GroupOpenTheDoorRequest,
        user_id: Option<i64>,
    ) -> Result<GroupDetailData> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let group = self
            .group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;

        let tags = self.tag_repo.get_group_tags(group_id).await.ok();
        let settings = self.settings_repo.find_by_group(group_id).await?;
        let has_password = settings.as_ref().and_then(|s| s.password_hash.as_ref()).is_some();

        let members_count = self.member_repo.count_members(group_id).await? as i32;

        let my_state = if let Some(uid) = user_id {
            if let Some(m) = self.member_repo.find_member(group_id, uid).await? {
                let unread_count = self
                    .unread_repo
                    .get_unread_count("group", &group_id.to_string(), uid)
                    .await
                    .unwrap_or(0);
                let prefs = self.settings_repo.find_user_preferences(group_id, uid).await?;
                Some(GroupMyStateData {
                    role: m.role,
                    is_joined: true,
                    unread_count: unread_count as i32,
                    is_muted: prefs.as_ref().map(|p| p.is_muted).unwrap_or(false),
                    is_pinned: prefs.as_ref().map(|p| p.is_pinned).unwrap_or(false),
                    joined_at: Some(m.joined_at.to_rfc3339()),
                })
            } else {
                None
            }
        } else {
            None
        };

        Ok(GroupDetailData {
            group_chat_id: group.group_id.to_string(),
            name: group.name,
            description: group.description,
            avatar_url: self.media_service.sign_url_if_path(group.avatar_url.as_deref()).await,
            tags,
            members_count,
            has_password,
            location: self.build_location(None).await,
            created_at: group.created_at.to_rfc3339(),
            owner_id: group.owner_id,
            my_state,
        })
    }

    pub async fn get_settings(&self, req: GroupSettingsRequest) -> Result<GroupSettingsData> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let group = self
            .group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;
        let settings = self.settings_repo.find_by_group(group_id).await?;

        let join_method = settings
            .as_ref()
            .map(|s| Self::join_method_from_str(&s.join_method))
            .unwrap_or(GroupJoinMethod::Open);
        let is_searchable = settings.as_ref().map(|s| s.is_searchable).unwrap_or(true);
        let max_members = settings
            .as_ref()
            .and_then(|s| s.max_members)
            .unwrap_or(group.max_members);
        let has_password = settings.as_ref().and_then(|s| s.password_hash.as_ref()).is_some();

        Ok(GroupSettingsData {
            group_chat_id: group_id.to_string(),
            settings: GroupSettingsDto {
                join_method,
                is_searchable,
                max_members,
                location: self.build_location(None).await,
            },
            has_password,
            created_at: group.created_at.to_rfc3339(),
        })
    }

    pub async fn get_members(
        &self,
        req: GroupMembersRequest,
    ) -> Result<PaginatedData<GroupMemberItem>> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let limit = Validator::validate_limit(req.limit, 100)? as usize;

        let (rows, next_cursor) = self
            .member_repo
            .list_members(group_id, limit, req.cursor.as_deref())
            .await?;

        let items = rows
            .into_iter()
            .map(|m| self.map_member_detail(m))
            .collect::<Result<Vec<_>>>()?;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    fn map_member_detail(&self, m: GroupMemberDetailRow) -> Result<GroupMemberItem> {
        let nickname = m.user_nickname.unwrap_or_else(|| "未知用户".to_string());
        Ok(GroupMemberItem {
            user: UserInfo {
                user_id: m.user_id,
                nickname,
                avatar_url: m.avatar_url,
                user_type: m.user_type.unwrap_or(UserType::Normal),
            },
            role: m.role,
            joined_at: m.joined_at.to_rfc3339(),
            muted_until: m.muted_until.map(|t| t.to_rfc3339()),
            is_online: None,
        })
    }

    // ==================== 群组管理 ====================

    pub async fn create_group(&self, owner_id: i64, req: CreateGroupChatRequest) -> Result<()> {
        Validator::validate_length(&req.name, "群组名称", 1, 100)?;

        let (lon, lat) = req
            .location
            .as_ref()
            .map(|l| (Some(l.longitude), Some(l.latitude)))
            .unwrap_or((None, None));

        let join_method = req.join_method.unwrap_or(GroupJoinMethod::Open);
        let join_method_str = Self::join_method_to_string(join_method);

        let group = self
            .group_repo
            .create(
                req.name,
                owner_id,
                lon,
                lat,
                "public".to_string(),
                Self::join_type_from_join_method(join_method),
            )
            .await?;

        let avatar_url = if let Some(fid) = req.avatar_file_id {
            let fid_uuid = Uuid::parse_str(&fid)
                .map_err(|_| AppError::BadRequest("无效的文件ID".to_string()))?;
            Some(
                self.media_service
                    .validate_file(fid_uuid, owner_id, "group_avatar")
                    .await?,
            )
        } else {
            None
        };

        if req.description.is_some() || avatar_url.is_some() || req.max_members.is_some() {
            self.group_repo
                .update_profile(
                    group.group_id,
                    None,
                    req.description.as_deref(),
                    avatar_url.as_deref(),
                    req.max_members,
                )
                .await?;
        }

        let password_hash = if let Some(pwd) = req.password.as_deref() {
            if pwd.is_empty() {
                None
            } else {
                Some(PasswordUtil::hash_password(pwd)?)
            }
        } else {
            None
        };

        self.settings_repo
            .create(
                group.group_id,
                &join_method_str,
                req.is_searchable.unwrap_or(true),
                req.max_members,
                password_hash.as_deref(),
            )
            .await?;

        if let Some(tags) = req.tags {
            self.tag_repo.set_group_tags(group.group_id, &tags).await?;
        }

        self.member_repo
            .add_member(group.group_id, owner_id, GroupRole::Owner, None)
            .await?;
        let _ = self
            .member_history_repo
            .record_join(group.group_id, owner_id, Some(owner_id))
            .await;

        self.event_bus.publish(AppEvent::GroupCreated {
            group_id: group.group_id,
            creator_id: owner_id,
        });

        Ok(())
    }

    pub async fn get_joined(
        &self,
        user_id: i64,
        req: MyGroupChatsRequest,
    ) -> Result<PaginatedData<MyGroupItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as usize;
        let (rows, next_cursor) = self
            .member_repo
            .list_user_groups(user_id, limit, req.cursor.as_deref())
            .await?;

        let group_ids: Vec<i64> = rows.iter().map(|r| r.group_id).collect();
        let group_id_strs: Vec<String> = group_ids.iter().map(|id| id.to_string()).collect();
        let unread_counts = self
            .unread_repo
            .get_unread_count_batch("group", &group_id_strs, user_id)
            .await
            .unwrap_or_default();

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            let unread_count = unread_counts
                .get(&row.group_id.to_string())
                .copied()
                .unwrap_or(0) as i32;
            let tags = self.tag_repo.get_group_tags(row.group_id).await.ok();
            items.push(MyGroupItem {
                group_info: GroupInfo {
                    group_chat_id: row.group_id.to_string(),
                    name: row.name,
                    avatar_url: self.media_service.sign_url_if_path(row.avatar_url.as_deref()).await,
                    members_count: row.members_count,
                },
                description: row.description,
                tags,
                has_password: row.has_password,
                location: self.build_location(None).await,
                last_activity_at: row.joined_at.to_rfc3339(),
                role: GroupRole::Member,
                unread_count,
            });
        }

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    pub async fn update_group(&self, user_id: i64, req: UpdateGroupChatRequest) -> Result<()> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        let avatar_url = if let Some(fid) = req.avatar_file_id.as_deref() {
            let fid_uuid = Uuid::parse_str(fid)
                .map_err(|_| AppError::BadRequest("无效的文件ID".to_string()))?;
            Some(
                self.media_service
                    .validate_file(fid_uuid, user_id, "group_avatar")
                    .await?,
            )
        } else {
            None
        };

        self.group_repo
            .update_profile(
                group_id,
                req.name.as_deref(),
                req.description.as_deref(),
                avatar_url.as_deref(),
                req.max_members,
            )
            .await?;

        if let Some(tags) = req.tags {
            self.tag_repo.set_group_tags(group_id, &tags).await?;
        }

        if req.join_method.is_some()
            || req.is_searchable.is_some()
            || req.max_members.is_some()
            || req.password.is_some()
        {
            let join_method = req.join_method.map(Self::join_method_to_string);
            match req.password.as_deref() {
                Some("") => {
                    self.settings_repo.clear_password(group_id).await?;
                    self.settings_repo
                        .update(
                            group_id,
                            join_method.as_deref(),
                            req.is_searchable,
                            req.max_members,
                            None,
                        )
                        .await?;
                }
                Some(pwd) => {
                    let password_hash = PasswordUtil::hash_password(pwd)?;
                    self.settings_repo
                        .update(
                            group_id,
                            join_method.as_deref(),
                            req.is_searchable,
                            req.max_members,
                            Some(&password_hash),
                        )
                        .await?;
                }
                None => {
                    self.settings_repo
                        .update(
                            group_id,
                            join_method.as_deref(),
                            req.is_searchable,
                            req.max_members,
                            None,
                        )
                        .await?;
                }
            }
        }

        self.event_bus.publish(AppEvent::GroupProfileUpdated {
            group_id,
            name: req.name,
            description: req.description,
            avatar_url,
        });

        let _ = self
            .cache_manager
            .delete(cache_keys::GROUP_INFO, &group_id.to_string())
            .await;
        Ok(())
    }

    pub async fn delete_group(&self, user_id: i64, req: DeleteGroupChatRequest) -> Result<()> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) {
            return Err(AppError::Forbidden("只有群主可解散群组".to_string()));
        }

        self.group_repo.dismiss(group_id).await?;
        self.event_bus.publish(AppEvent::GroupDismissed {
            group_id,
            dismissed_by: user_id,
        });
        Ok(())
    }

    pub async fn transfer_ownership(
        &self,
        user_id: i64,
        req: TransferOwnershipRequest,
    ) -> Result<()> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) {
            return Err(AppError::Forbidden("只有群主可转让".to_string()));
        }

        if !self
            .member_repo
            .is_member(group_id, req.new_owner_id)
            .await?
        {
            return Err(AppError::NotGroupMember("目标用户不在群组中".to_string()));
        }

        self.group_repo
            .update_owner(group_id, req.new_owner_id)
            .await?;
        self.member_repo
            .update_role(group_id, req.new_owner_id, GroupRole::Owner)
            .await?;
        self.member_repo
            .update_role(group_id, user_id, GroupRole::Member)
            .await?;
        Ok(())
    }

    // ==================== 成员操作 ====================

    pub async fn invite_member(
        &self,
        user_id: i64,
        req: InviteMemberRequest,
    ) -> Result<InviteMemberData> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        if !self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::NotGroupMember("不是群组成员".to_string()));
        }

        if self.member_repo.is_member(group_id, req.user_id).await? {
            return Err(AppError::AlreadyInGroup);
        }

        let invite = self
            .invite_repo
            .create(group_id, user_id, req.user_id, req.message.clone())
            .await?;

        let group = self
            .group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;

        self.event_bus.publish(AppEvent::GroupInviteSent {
            invite_id: invite.invite_id,
            group_id,
            group_name: group.name,
            group_avatar_url: group.avatar_url,
            from_user_id: user_id,
            to_user_id: req.user_id,
            message: req.message,
        });

        Ok(InviteMemberData {
            invite_id: invite.invite_id.to_string(),
            expires_at: invite
                .expires_at
                .unwrap_or(invite.created_at)
                .to_rfc3339(),
        })
    }

    pub async fn join_group(&self, user_id: i64, req: JoinGroupRequest) -> Result<()> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let group = self
            .group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;

        if self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::AlreadyInGroup);
        }

        if self.bans_repo.is_banned(group_id, user_id).await? {
            return Err(AppError::Forbidden("已被封禁".to_string()));
        }

        let settings = self.settings_repo.find_by_group(group_id).await?;
        if let Some(settings) = &settings {
            if let Some(pwd_hash) = settings.password_hash.as_deref() {
                let pwd = req.password.as_deref().unwrap_or("");
                if !PasswordUtil::verify_password(pwd, pwd_hash)? {
                    return Err(AppError::GroupPasswordError("群组密码错误".to_string()));
                }
            }
            if let Some(max) = settings.max_members {
                let count = self.member_repo.count_members(group_id).await? as i32;
                if count >= max {
                    return Err(AppError::GroupFull);
                }
            }
            let join_method = Self::join_method_from_str(&settings.join_method);
            if join_method == GroupJoinMethod::Approval {
                return Err(AppError::BusinessRuleViolation("需要申请加入".to_string()));
            }
            if join_method == GroupJoinMethod::InviteOnly {
                return Err(AppError::BusinessRuleViolation("仅邀请加入".to_string()));
            }
        }

        self.member_repo
            .add_member(group_id, user_id, GroupRole::Member, None)
            .await?;
        let _ = self
            .member_history_repo
            .record_join(group_id, user_id, Some(user_id))
            .await;

        let member_count = self.member_repo.count_members(group_id).await?;
        self.event_bus.publish(AppEvent::GroupMemberAdded {
            group_id,
            user_id,
            added_by: user_id,
            role: GroupRole::Member,
            member_count,
        });

        let _ = group;
        Ok(())
    }

    pub async fn apply_to_group(
        &self,
        user_id: i64,
        req: ApplyToGroupRequest,
    ) -> Result<ApplyToGroupData> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        if self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::AlreadyInGroup);
        }
        let application = self
            .application_repo
            .create(group_id, user_id, req.message.clone())
            .await?;

        self.event_bus.publish(AppEvent::GroupApplicationReceived {
            application_id: application.application_id,
            group_id,
            applicant_id: user_id,
            message: application.message.clone(),
        });

        Ok(ApplyToGroupData {
            application_id: application.application_id.to_string(),
        })
    }

    pub async fn set_role(
        &self,
        user_id: i64,
        req: SetMemberRoleRequest,
    ) -> Result<GroupMemberItem> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) {
            return Err(AppError::Forbidden("只有群主可设置角色".to_string()));
        }

        let target_role = match req.role {
            SetableGroupRole::Admin => GroupRole::Admin,
            SetableGroupRole::Member => GroupRole::Member,
        };

        self.member_repo
            .update_role(group_id, req.user_id, target_role)
            .await?;
        self.member_history_repo
            .record_role_changed(group_id, req.user_id, user_id, GroupRole::Member, target_role)
            .await
            .ok();

        self.event_bus.publish(AppEvent::GroupMemberRoleChanged {
            group_id,
            user_id: req.user_id,
            old_role: GroupRole::Member,
            new_role: target_role,
        });

        let user_info = self.build_user_info(req.user_id).await?;
        Ok(GroupMemberItem {
            user: user_info,
            role: target_role,
            joined_at: Utc::now().to_rfc3339(),
            muted_until: None,
            is_online: None,
        })
    }

    pub async fn mute_member(
        &self,
        user_id: i64,
        req: MuteMemberRequest,
    ) -> Result<GroupMemberItem> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        let muted_until = Utc::now() + Duration::minutes(req.duration_minutes as i64);
        self.member_repo
            .set_mute(group_id, req.user_id, true, Some(muted_until))
            .await?;
        let _ = self
            .member_history_repo
            .record_muted(group_id, req.user_id, user_id, muted_until)
            .await;

        self.event_bus.publish(AppEvent::GroupMemberMuted {
            group_id,
            user_id: req.user_id,
            muted_until: muted_until.to_rfc3339(),
        });

        let user_info = self.build_user_info(req.user_id).await?;
        Ok(GroupMemberItem {
            user: user_info,
            role: GroupRole::Member,
            joined_at: Utc::now().to_rfc3339(),
            muted_until: Some(muted_until.to_rfc3339()),
            is_online: None,
        })
    }

    pub async fn unmute_member(
        &self,
        user_id: i64,
        req: UnmuteMemberRequest,
    ) -> Result<GroupMemberItem> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        self.member_repo
            .set_mute(group_id, req.user_id, false, None)
            .await?;
        let _ = self
            .member_history_repo
            .record_unmuted(group_id, req.user_id, user_id)
            .await;

        self.event_bus.publish(AppEvent::GroupMemberUnmuted {
            group_id,
            user_id: req.user_id,
        });

        let user_info = self.build_user_info(req.user_id).await?;
        Ok(GroupMemberItem {
            user: user_info,
            role: GroupRole::Member,
            joined_at: Utc::now().to_rfc3339(),
            muted_until: None,
            is_online: None,
        })
    }

    pub async fn ban_member(
        &self,
        user_id: i64,
        req: BanMemberRequest,
    ) -> Result<BannedMemberItem> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        self.bans_repo
            .ban(group_id, req.user_id, user_id, req.reason.clone())
            .await?;
        let _ = self
            .member_history_repo
            .record_banned(group_id, req.user_id, user_id, req.reason.clone())
            .await;

        let user_info = self.build_user_info(req.user_id).await?;
        Ok(BannedMemberItem {
            user: user_info,
            reason: req.reason,
            banned_at: Utc::now().to_rfc3339(),
        })
    }

    pub async fn unban_member(&self, user_id: i64, req: UnbanMemberRequest) -> Result<()> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        self.bans_repo.unban(group_id, req.user_id, user_id).await?;
        Ok(())
    }

    pub async fn get_banned_list(
        &self,
        user_id: i64,
        req: BannedMembersRequest,
    ) -> Result<PaginatedData<BannedMemberItem>> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role.is_none() {
            return Err(AppError::NotGroupMember("不是群组成员".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let (rows, next_cursor) =
            self.bans_repo.list_by_group(group_id, limit, req.cursor).await?;
        let items = rows
            .into_iter()
            .map(|r| BannedMemberItem {
                user: UserInfo {
                    user_id: r.user_id,
                    nickname: r.nickname.unwrap_or_else(|| "未知用户".to_string()),
                    avatar_url: r.avatar_url,
                    user_type: r.user_type.unwrap_or(UserType::Normal),
                },
                reason: r.reason,
                banned_at: r.banned_at.to_rfc3339(),
            })
            .collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    pub async fn remove_member(&self, user_id: i64, req: RemoveMemberRequest) -> Result<()> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        self.member_repo.remove_member(group_id, req.user_id).await?;
        let _ = self
            .member_history_repo
            .record_removed(group_id, req.user_id, user_id, None)
            .await;

        let member_count = self.member_repo.count_members(group_id).await?;
        self.event_bus.publish(AppEvent::GroupMemberRemoved {
            group_id,
            user_id: req.user_id,
            removed_by: user_id,
            member_count,
            reason: MemberLeaveReason::Kick,
        });
        Ok(())
    }

    pub async fn leave_group(&self, user_id: i64, req: LeaveGroupRequest) -> Result<()> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role == Some(GroupRole::Owner) {
            return Err(AppError::BusinessRuleViolation("群主不能退出".to_string()));
        }

        self.member_repo.remove_member(group_id, user_id).await?;
        let _ = self.member_history_repo.record_leave(group_id, user_id).await;
        let member_count = self.member_repo.count_members(group_id).await?;
        self.event_bus.publish(AppEvent::GroupMemberRemoved {
            group_id,
            user_id,
            removed_by: user_id,
            member_count,
            reason: MemberLeaveReason::Quit,
        });
        Ok(())
    }

    // ==================== 申请管理 ====================

    pub async fn get_applications(
        &self,
        user_id: i64,
        req: GroupApplicationsRequest,
    ) -> Result<PaginatedData<GroupApplicationItem>> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let status = match req.status.unwrap_or(ApplicationStatusFilter::All) {
            ApplicationStatusFilter::All => None,
            ApplicationStatusFilter::Pending => Some(ApplicationStatus::Pending),
            ApplicationStatusFilter::Accepted => Some(ApplicationStatus::Accepted),
            ApplicationStatusFilter::Rejected => Some(ApplicationStatus::Rejected),
        };
        let (rows, next_cursor) = self
            .application_repo
            .list_by_group(group_id, status, limit, req.cursor)
            .await?;

        let applicant_ids: Vec<i64> = rows.iter().map(|r| r.applicant_id).collect();
        let handler_ids: Vec<i64> = rows.iter().filter_map(|r| r.handled_by).collect();
        let users = self.user_repo.find_profiles(&applicant_ids).await?;
        let handlers = self.user_repo.find_profiles(&handler_ids).await?;
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();
        let handler_map: std::collections::HashMap<i64, _> =
            handlers.into_iter().map(|u| (u.user_id, u)).collect();

        let items = rows
            .into_iter()
            .map(|r| {
                let applicant = user_map.get(&r.applicant_id);
                let handled_by = r
                    .handled_by
                    .and_then(|id| handler_map.get(&id))
                    .map(|u| UserInfo {
                        user_id: u.user_id,
                        nickname: u.nickname.clone().unwrap_or_else(|| "未知用户".to_string()),
                        avatar_url: u.avatar_url.clone(),
                        user_type: u.user_type,
                    });
                GroupApplicationItem {
                    application_id: r.application_id.to_string(),
                    applicant: UserInfo {
                        user_id: r.applicant_id,
                        nickname: applicant
                            .and_then(|u| u.nickname.clone())
                            .unwrap_or_else(|| "未知用户".to_string()),
                        avatar_url: applicant.and_then(|u| u.avatar_url.clone()),
                        user_type: applicant.map(|u| u.user_type).unwrap_or(UserType::Normal),
                    },
                    message: r.message,
                    status: r.status,
                    created_at: r.created_at.to_rfc3339(),
                    handled_at: r.handled_at.map(|t| t.to_rfc3339()),
                    handled_by,
                }
            })
            .collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    pub async fn approve_application(
        &self,
        user_id: i64,
        req: ApproveGroupApplicationRequest,
    ) -> Result<GroupApplicationItem> {
        let app_id = Uuid::parse_str(&req.application_id)
            .map_err(|_| AppError::BadRequest("无效的申请ID".to_string()))?;
        let application = self
            .application_repo
            .find_by_application_id(app_id)
            .await?
            .ok_or_else(|| AppError::NotFound("申请不存在".to_string()))?;

        let role = self
            .member_repo
            .get_member_role(application.group_id, user_id)
            .await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        self.application_repo
            .update_status_by_application_id(app_id, ApplicationStatus::Accepted, user_id)
            .await?;
        self.member_repo
            .add_member(application.group_id, application.applicant_id, GroupRole::Member, None)
            .await?;

        self.event_bus.publish(AppEvent::GroupApplicationResponse {
            application_id: app_id,
            group_id: application.group_id,
            group_name: "群组".to_string(),
            applicant_id: application.applicant_id,
            status: ResponseStatus::Accepted,
        });

        let applicant = self.build_user_info(application.applicant_id).await?;
        Ok(GroupApplicationItem {
            application_id: application.application_id.to_string(),
            applicant,
            message: application.message,
            status: ApplicationStatus::Accepted,
            created_at: application.created_at.to_rfc3339(),
            handled_at: Some(Utc::now().to_rfc3339()),
            handled_by: Some(self.build_user_info(user_id).await?),
        })
    }

    pub async fn reject_application(
        &self,
        user_id: i64,
        req: RejectGroupApplicationRequest,
    ) -> Result<GroupApplicationItem> {
        let app_id = Uuid::parse_str(&req.application_id)
            .map_err(|_| AppError::BadRequest("无效的申请ID".to_string()))?;
        let application = self
            .application_repo
            .find_by_application_id(app_id)
            .await?
            .ok_or_else(|| AppError::NotFound("申请不存在".to_string()))?;

        let role = self
            .member_repo
            .get_member_role(application.group_id, user_id)
            .await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        self.application_repo
            .update_status_by_application_id(app_id, ApplicationStatus::Rejected, user_id)
            .await?;
        self.event_bus.publish(AppEvent::GroupApplicationResponse {
            application_id: app_id,
            group_id: application.group_id,
            group_name: "群组".to_string(),
            applicant_id: application.applicant_id,
            status: ResponseStatus::Rejected,
        });

        let applicant = self.build_user_info(application.applicant_id).await?;
        Ok(GroupApplicationItem {
            application_id: application.application_id.to_string(),
            applicant,
            message: application.message,
            status: ApplicationStatus::Rejected,
            created_at: application.created_at.to_rfc3339(),
            handled_at: Some(Utc::now().to_rfc3339()),
            handled_by: Some(self.build_user_info(user_id).await?),
        })
    }

    pub async fn cancel_application(
        &self,
        user_id: i64,
        req: CancelApplicationRequest,
    ) -> Result<()> {
        let app_id = Uuid::parse_str(&req.application_id)
            .map_err(|_| AppError::BadRequest("无效的申请ID".to_string()))?;
        let application = self
            .application_repo
            .find_by_application_id(app_id)
            .await?
            .ok_or_else(|| AppError::NotFound("申请不存在".to_string()))?;
        if application.applicant_id != user_id {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }
        self.application_repo.delete_by_application_id(app_id).await?;
        Ok(())
    }

    // ==================== 邀请管理 ====================

    pub async fn get_invites(
        &self,
        user_id: i64,
        req: MyInvitesRequest,
    ) -> Result<PaginatedData<GroupInviteItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i32;
        let status = req.status.map(|s| match s {
            ApplicationStatusFilter::Pending => ApplicationStatus::Pending,
            ApplicationStatusFilter::Accepted => ApplicationStatus::Accepted,
            ApplicationStatusFilter::Rejected => ApplicationStatus::Rejected,
            ApplicationStatusFilter::All => ApplicationStatus::Pending,
        });

        let (rows, next_cursor) =
            self.invite_repo.list_by_user(user_id, status, limit, req.cursor).await?;
        let group_ids: Vec<i64> = rows.iter().map(|r| r.group_id).collect();
        let inviter_ids: Vec<i64> = rows.iter().map(|r| r.inviter_id).collect();
        let groups = self.group_repo.list_by_ids(&group_ids).await?;
        let users = self.user_repo.find_profiles(&inviter_ids).await?;
        let group_map: std::collections::HashMap<i64, _> =
            groups.into_iter().map(|g| (g.group_id, g)).collect();
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        let items = rows
            .into_iter()
            .map(|r| {
                let group = group_map.get(&r.group_id);
                let inviter = user_map.get(&r.inviter_id);
                GroupInviteItem {
                    invite_id: r.invite_id.to_string(),
                    group: GroupInfo {
                        group_chat_id: r.group_id.to_string(),
                        name: group
                            .map(|g| g.name.clone())
                            .unwrap_or_else(|| "群组".to_string()),
                        avatar_url: group.and_then(|g| g.avatar_url.clone()),
                        members_count: group.map(|g| g.max_members).unwrap_or(0),
                    },
                    inviter: UserInfo {
                        user_id: r.inviter_id,
                        nickname: inviter
                            .and_then(|u| u.nickname.clone())
                            .unwrap_or_else(|| "未知用户".to_string()),
                        avatar_url: inviter.and_then(|u| u.avatar_url.clone()),
                        user_type: inviter.map(|u| u.user_type).unwrap_or(UserType::Normal),
                    },
                    message: r.message,
                    status: r.status,
                    created_at: r.created_at.to_rfc3339(),
                    expires_at: r.expires_at.map(|t| t.to_rfc3339()),
                }
            })
            .collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    pub async fn accept_invite(&self, user_id: i64, req: AcceptInviteRequest) -> Result<()> {
        let invite_id = Uuid::parse_str(&req.invite_id)
            .map_err(|_| AppError::BadRequest("无效的邀请ID".to_string()))?;
        let invite = self
            .invite_repo
            .find_by_invite_id(invite_id)
            .await?
            .ok_or_else(|| AppError::NotFound("邀请不存在".to_string()))?;

        if invite.invitee_id != user_id {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        self.invite_repo
            .update_status_by_invite_id(invite_id, ApplicationStatus::Accepted)
            .await?;
        self.member_repo
            .add_member(invite.group_id, user_id, GroupRole::Member, Some(invite.inviter_id))
            .await?;

        self.event_bus.publish(AppEvent::GroupInviteResponse {
            invite_id,
            group_id: invite.group_id,
            inviter_id: invite.inviter_id,
            invitee_id: invite.invitee_id,
            status: ResponseStatus::Accepted,
        });

        Ok(())
    }

    pub async fn reject_invite(&self, user_id: i64, req: RejectInviteRequest) -> Result<()> {
        let invite_id = Uuid::parse_str(&req.invite_id)
            .map_err(|_| AppError::BadRequest("无效的邀请ID".to_string()))?;
        let invite = self
            .invite_repo
            .find_by_invite_id(invite_id)
            .await?
            .ok_or_else(|| AppError::NotFound("邀请不存在".to_string()))?;
        if invite.invitee_id != user_id {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        self.invite_repo
            .update_status_by_invite_id(invite_id, ApplicationStatus::Rejected)
            .await?;
        self.event_bus.publish(AppEvent::GroupInviteResponse {
            invite_id,
            group_id: invite.group_id,
            inviter_id: invite.inviter_id,
            invitee_id: invite.invitee_id,
            status: ResponseStatus::Rejected,
        });
        Ok(())
    }

    // ==================== 群公告 ====================

    pub async fn create_announcement(
        &self,
        user_id: i64,
        req: &CreateGroupAnnouncementRequest,
    ) -> Result<GroupAnnouncementItem> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

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

        self.event_bus.publish(AppEvent::GroupAnnouncementCreated {
            announcement_id: announcement.announcement_id,
            group_id,
            title: announcement.title.clone(),
            content: announcement.content.clone(),
            created_by_user_id: user_id,
            is_pinned: announcement.is_pinned,
        });

        Ok(GroupAnnouncementItem {
            announcement_id: announcement.announcement_id.to_string(),
            group_chat_id: group_id.to_string(),
            title: announcement.title,
            content: announcement.content,
            created_by: self.build_user_info(user_id).await?,
            created_at: announcement.created_at.to_rfc3339(),
            is_pinned: announcement.is_pinned,
        })
    }

    pub async fn list_announcements(
        &self,
        req: &GroupAnnouncementsRequest,
    ) -> Result<PaginatedData<GroupAnnouncementItem>> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let announcements = self
            .announcement_repo
            .list_by_group(group_id, limit, req.cursor.clone())
            .await?;

        let user_ids: Vec<i64> = announcements.iter().map(|a| a.publisher_id).collect();
        let users = self.user_repo.find_profiles(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        let items = announcements
            .into_iter()
            .map(|a| {
                let user = user_map.get(&a.publisher_id);
                GroupAnnouncementItem {
                    announcement_id: a.announcement_id.to_string(),
                    group_chat_id: group_id.to_string(),
                    title: a.title,
                    content: a.content,
                    created_by: UserInfo {
                        user_id: a.publisher_id,
                        nickname: user
                            .and_then(|u| u.nickname.clone())
                            .unwrap_or_else(|| "未知用户".to_string()),
                        avatar_url: user.and_then(|u| u.avatar_url.clone()),
                        user_type: user.map(|u| u.user_type).unwrap_or(UserType::Normal),
                    },
                    created_at: a.created_at.to_rfc3339(),
                    is_pinned: a.is_pinned,
                }
            })
            .collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: false,
                next_cursor: None,
            },
        })
    }

    pub async fn update_announcement(
        &self,
        user_id: i64,
        req: &UpdateGroupAnnouncementRequest,
    ) -> Result<GroupAnnouncementItem> {
        let announcement_id = Uuid::parse_str(&req.announcement_id)
            .map_err(|_| AppError::BadRequest("无效的公告ID".to_string()))?;
        let announcement = self
            .announcement_repo
            .find_by_id(announcement_id)
            .await?
            .ok_or_else(|| AppError::NotFound("公告不存在".to_string()))?;

        let role = self
            .member_repo
            .get_member_role(announcement.group_id, user_id)
            .await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        self.announcement_repo
            .update(announcement_id, req.title.clone(), req.content.clone(), req.is_pinned)
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
            group_chat_id: announcement.group_id.to_string(),
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
        let announcement_id = Uuid::parse_str(&req.announcement_id)
            .map_err(|_| AppError::BadRequest("无效的公告ID".to_string()))?;
        let announcement = self
            .announcement_repo
            .find_by_id(announcement_id)
            .await?
            .ok_or_else(|| AppError::NotFound("公告不存在".to_string()))?;

        let role = self
            .member_repo
            .get_member_role(announcement.group_id, user_id)
            .await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        self.announcement_repo.delete(announcement_id).await?;
        self.event_bus.publish(AppEvent::GroupAnnouncementDeleted {
            announcement_id,
            group_id: announcement.group_id,
        });
        Ok(())
    }

    pub async fn get_my_state(
        &self,
        user_id: i64,
        req: &GetMyGroupStateRequest,
    ) -> Result<GroupMyStateData> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let member = self
            .member_repo
            .find_member(group_id, user_id)
            .await?
            .ok_or_else(|| AppError::NotGroupMember("不是群组成员".to_string()))?;
        let unread_count = self
            .unread_repo
            .get_unread_count("group", &group_id.to_string(), user_id)
            .await
            .unwrap_or(0);
        let prefs = self.settings_repo.find_user_preferences(group_id, user_id).await?;
        Ok(GroupMyStateData {
            role: member.role,
            is_joined: true,
            unread_count: unread_count as i32,
            is_muted: prefs.as_ref().map(|p| p.is_muted).unwrap_or(false),
            is_pinned: prefs.as_ref().map(|p| p.is_pinned).unwrap_or(false),
            joined_at: Some(member.joined_at.to_rfc3339()),
        })
    }

    pub async fn update_my_settings(
        &self,
        user_id: i64,
        req: &UpdateGroupMySettingsRequest,
    ) -> Result<()> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        self.settings_repo
            .upsert_user_preferences(group_id, user_id, req.is_muted, req.is_pinned)
            .await?;
        Ok(())
    }

    // ==================== 管理员群组 ====================

    pub async fn admin_search_groups(
        &self,
        req: &AdminGroupSearchRequest,
    ) -> Result<PaginatedData<AdminGroupSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let group_id = req
            .group_chat_id
            .as_deref()
            .and_then(|id| id.parse::<i64>().ok());
        let (rows, next_cursor) = self
            .group_repo
            .admin_search(group_id, req.name.clone(), req.owner_id, limit, req.cursor.clone())
            .await?;

        let owner_ids: Vec<i64> = rows.iter().map(|r| r.owner_id).collect();
        let owners = self.user_repo.find_profiles(&owner_ids).await?;
        let owner_map: std::collections::HashMap<i64, _> =
            owners.into_iter().map(|u| (u.user_id, u)).collect();

        let items = rows
            .into_iter()
            .map(|r| {
                let owner = owner_map.get(&r.owner_id);
                AdminGroupSearchItem {
                    group_chat_id: r.group_id.to_string(),
                    name: r.name,
                    avatar_url: r.avatar_url,
                    members_count: r.member_count,
                    location: None,
                    owner: UserInfo {
                        user_id: r.owner_id,
                        nickname: owner
                            .and_then(|u| u.nickname.clone())
                            .unwrap_or_else(|| "未知用户".to_string()),
                        avatar_url: owner.and_then(|u| u.avatar_url.clone()),
                        user_type: owner.map(|u| u.user_type).unwrap_or(UserType::Normal),
                    },
                    created_at: r.created_at.to_rfc3339(),
                }
            })
            .collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    pub async fn admin_group_detail(
        &self,
        req: &AdminGroupDetailRequest,
    ) -> Result<AdminGroupDetail> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        let group = self
            .group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;
        let members_count = self.member_repo.count_members(group_id).await? as i32;
        let owner = self.build_user_info(group.owner_id).await?;
        let settings = self.settings_repo.find_by_group(group_id).await?;
        let tags = self.tag_repo.get_group_tags(group_id).await.ok();

        Ok(AdminGroupDetail {
            group_chat_id: group_id.to_string(),
            name: group.name,
            avatar_url: self.media_service.sign_url_if_path(group.avatar_url.as_deref()).await,
            members_count,
            location: None,
            description: group.description,
            tags,
            max_members: group.max_members,
            owner,
            created_at: group.created_at.to_rfc3339(),
            settings: AdminGroupSettingsSummary {
                join_method: settings
                    .as_ref()
                    .map(|s| Self::join_method_from_str(&s.join_method))
                    .unwrap_or(GroupJoinMethod::Open),
                is_searchable: settings.as_ref().map(|s| s.is_searchable).unwrap_or(true),
                has_password: settings
                    .as_ref()
                    .and_then(|s| s.password_hash.as_ref())
                    .is_some(),
            },
            admins: None,
            stats: AdminGroupStats {
                messages_count: 0,
                active_members_today: 0,
            },
            dismissal_history: None,
        })
    }

    pub async fn admin_dismiss_group(
        &self,
        admin_user_id: i64,
        req: &AdminDismissGroupRequest,
    ) -> Result<()> {
        let group_id = Self::parse_group_id(&req.group_chat_id)?;
        self.group_repo.dismiss(group_id).await?;
        self.event_bus.publish(AppEvent::GroupDismissed {
            group_id,
            dismissed_by: admin_user_id,
        });
        Ok(())
    }
}
