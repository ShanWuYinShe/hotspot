use redis::aio::ConnectionManager;
use sqlx::PgPool;
use std::sync::Arc;
use uuid::Uuid;
use std::str::FromStr;

use crate::{
    constants::{DEFAULT_LATITUDE, DEFAULT_LONGITUDE, DEFAULT_LOCATION_NAME},
    dto::{
        request::{
            CreateGroupChatRequest, UpdateGroupChatRequest, JoinGroupRequest, LeaveGroupRequest,
            RemoveMemberRequest, MuteMemberRequest, UnmuteMemberRequest,
            BanMemberRequest, UnbanMemberRequest, GetBannedMembersListRequest,
            SetMemberRoleRequest, TransferOwnershipRequest,
            SearchGroupsByKeywordRequest, SearchGroupsByTagRequest, SearchGroupsByLocationRequest,
            GroupOpenTheDoorRequest, GetGroupSettingsRequest, GetMembersRequest,
            InviteMemberRequest, ApplyToGroupRequest, GetGroupApplicationsListRequest,
            ApproveGroupApplicationRequest, RejectGroupApplicationRequest, CancelGroupApplicationRequest,
            GetGroupInvitesListRequest, AcceptGroupInviteRequest, RejectGroupInviteRequest,
            GetJoinedGroupChatsRequest,
        },
        response::{
            CreateGroupChatData, GroupSearchItem, JoinedGroupItem, GroupSettingsData, GeoData,
            GroupMemberItem, BannedMemberItem, PaginatedData, PaginationMeta, UserInfo, GroupSettings,
            GroupInfo, InviteMemberData, ApplyToGroupData, GroupApplicationItem, GroupInviteItem,
        },
    },
    error::{AppError, Result},
    events::events::AppEvent,
    repositories::{
        group_repo::GroupRepository,
        group_member_repo::{GroupMemberRepository, GroupMemberRow},
        group_member_history_repo::GroupMemberHistoryRepository,
        group_application_repo::GroupApplicationRepository,
        group_bans_repo::GroupBansRepository,
        group_invite_repo::GroupInviteRepository,
        group_settings_repo::GroupSettingsRepository,
        tag_repo::TagRepository,
        user_repo::UserRepository,
    },
    types::enums::{MemberRole, InviteStatus},
    utils::{password::PasswordUtil, validation::Validator},
    EventBus,
};

pub struct GroupService {
    db: PgPool,
    event_bus: Arc<EventBus>,
    group_repo: GroupRepository,
    member_repo: GroupMemberRepository,
    history_repo: GroupMemberHistoryRepository,
    application_repo: GroupApplicationRepository,
    bans_repo: GroupBansRepository,
    invite_repo: GroupInviteRepository,
    settings_repo: GroupSettingsRepository,
    tag_repo: TagRepository,
    user_repo: UserRepository,
    _redis: ConnectionManager,
}

impl GroupService {
    pub fn new(pool: &PgPool, redis: &ConnectionManager, event_bus: Arc<EventBus>) -> Self {
        Self {
            db: pool.clone(),
            event_bus,
            group_repo: GroupRepository::new(pool.clone()),
            member_repo: GroupMemberRepository::new(pool.clone()),
            history_repo: GroupMemberHistoryRepository::new(pool.clone()),
            application_repo: GroupApplicationRepository::new(pool.clone()),
            bans_repo: GroupBansRepository::new(pool.clone()),
            invite_repo: GroupInviteRepository::new(pool.clone()),
            settings_repo: GroupSettingsRepository::new(pool.clone()),
            tag_repo: TagRepository::new(pool.clone()),
            user_repo: UserRepository::new(pool.clone()),
            _redis: redis.clone(),
        }
    }

    pub async fn create_group(&self, owner_id: i64, req: CreateGroupChatRequest) -> Result<CreateGroupChatData> {
        let user = self.user_repo.find_by_id(owner_id).await?
            .ok_or(AppError::NotFound("User not found".to_string()))?;
        
        let lat = user.current_latitude.unwrap_or(DEFAULT_LATITUDE);
        let lng = user.current_longitude.unwrap_or(DEFAULT_LONGITUDE);
        let loc_name = user.current_location_name.clone().unwrap_or_else(|| DEFAULT_LOCATION_NAME.to_string());

        // 获取设置（使用默认值如果未提供）
        let settings = req.settings.unwrap_or_default();

        // 如果提供了密码，先进行哈希处理
        let hashed_password = if let Some(pwd) = &settings.password_hash {
            Some(PasswordUtil::hash_password(pwd)?)
        } else {
            None
        };

        // TODO: 将 avatar_file_id 解析为 avatar_url（需要查询 uploaded_files 表）
        let avatar_url = req.avatar_file_id.clone();

        let group = self.group_repo.create(
            req.name.clone(),
            req.description.clone(),
            avatar_url,
            owner_id,
            Some(lat),
            Some(lng),
            Some(loc_name),
        ).await?;

        self.member_repo.add_member(group.group_id, owner_id, MemberRole::Owner).await?;
        
        // 记录群主加入历史
        self.history_repo.record_join(group.group_id, owner_id, None).await?;

        self.settings_repo.create(
            group.group_id,
            &settings.join_method,
            settings.is_searchable,
            settings.max_members,
            hashed_password.as_deref(),
        ).await?;

        // 设置群组标签
        if let Some(tags) = &req.tags {
            if !tags.is_empty() {
                let tag_repo = TagRepository::new(self.db.clone());
                tag_repo.set_group_tags(group.group_id, tags).await?;
            }
        }

        Ok(CreateGroupChatData {
            id: group.group_id.to_string(),
        })
    }

    /// 按关键字搜索群组（全局）
    pub async fn search_by_keyword(
        &self,
        req: SearchGroupsByKeywordRequest,
    ) -> Result<PaginatedData<GroupSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        
        Validator::validate_coordinates(req.latitude, req.longitude)?;
        
        let results = self.group_repo
            .search_by_name(&req.kw, req.latitude, req.longitude, limit)
            .await?;
        
        // 批量获取标签
        let group_ids: Vec<Uuid> = results.iter().map(|(g, _, _, _)| g.group_id).collect();
        let tags_map = self.tag_repo.get_group_tags_batch(&group_ids).await?;
        
        let items: Vec<GroupSearchItem> = results.into_iter().map(|(g, dist, member_count, has_password)| {
            let tags = tags_map.get(&g.group_id).cloned();
            let tags = if tags.as_ref().map(|t| t.is_empty()).unwrap_or(true) { None } else { tags };
            
            GroupSearchItem {
                id: g.group_id.to_string(),
                name: g.name,
                description: g.description,
                avatar_url: g.avatar_url,
                tags,
                members_count: member_count,
                has_password,
                geo: GeoData {
                    latitude: g.latitude.unwrap_or(0.0),
                    longitude: g.longitude.unwrap_or(0.0),
                    location_name: g.location_name.unwrap_or_default(),
                    distance: dist as i32,
                    degrees: 0.0, // 待 Repository 层实现 PostGIS 计算
                },
            }
        }).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: false,
                next_cursor: None,
            },
        })
    }

    /// 按标签搜索群组（全局）
    pub async fn search_by_tag(
        &self,
        req: SearchGroupsByTagRequest,
    ) -> Result<PaginatedData<GroupSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        
        let lat = req.latitude;
        let lng = req.longitude;
        Validator::validate_coordinates(lat, lng)?;
        
        // 通过标签搜索群组
        let results = self.group_repo
            .search_by_name(&req.tag, lat, lng, limit)
            .await?;
        
        // 批量获取标签
        let group_ids: Vec<Uuid> = results.iter().map(|(g, _, _, _)| g.group_id).collect();
        let tags_map = self.tag_repo.get_group_tags_batch(&group_ids).await?;
        
        let items: Vec<GroupSearchItem> = results.into_iter().map(|(g, dist, member_count, has_password)| {
            let tags = tags_map.get(&g.group_id).cloned();
            let tags = if tags.as_ref().map(|t| t.is_empty()).unwrap_or(true) { None } else { tags };
            
            GroupSearchItem {
                id: g.group_id.to_string(),
                name: g.name,
                description: g.description,
                avatar_url: g.avatar_url,
                tags,
                members_count: member_count,
                has_password,
                geo: GeoData {
                    latitude: g.latitude.unwrap_or(0.0),
                    longitude: g.longitude.unwrap_or(0.0),
                    location_name: g.location_name.unwrap_or_default(),
                    distance: dist as i32,
                    degrees: 0.0, // 待 Repository 层实现 PostGIS 计算
                },
            }
        }).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: false,
                next_cursor: None,
            },
        })
    }

    /// 按位置搜索群组（范围过滤）
    pub async fn search_by_location(
        &self,
        req: SearchGroupsByLocationRequest,
    ) -> Result<PaginatedData<GroupSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        
        let lat = req.latitude;
        let lng = req.longitude;
        Validator::validate_coordinates(lat, lng)?;
        Validator::validate_radius_required(req.radius, 50000.0)?;
        
        let results = self.group_repo
            .discover(lat, lng, req.radius, req.kw.as_deref(), limit)
            .await?;
        
        // 批量获取标签
        let group_ids: Vec<Uuid> = results.iter().map(|(g, _, _, _)| g.group_id).collect();
        let tags_map = self.tag_repo.get_group_tags_batch(&group_ids).await?;
        
        let items: Vec<GroupSearchItem> = results.into_iter().map(|(g, dist, member_count, has_password)| {
            let tags = tags_map.get(&g.group_id).cloned();
            let tags = if tags.as_ref().map(|t| t.is_empty()).unwrap_or(true) { None } else { tags };
            
            GroupSearchItem {
                id: g.group_id.to_string(),
                name: g.name,
                description: g.description,
                avatar_url: g.avatar_url,
                tags,
                members_count: member_count,
                has_password,
                geo: GeoData {
                    latitude: g.latitude.unwrap_or(0.0),
                    longitude: g.longitude.unwrap_or(0.0),
                    location_name: g.location_name.unwrap_or_default(),
                    distance: dist as i32,
                    degrees: 0.0, // 待 Repository 层实现 PostGIS 计算
                },
            }
        }).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: false,
                next_cursor: None,
            },
        })
    }

    /// 获取群组详情（通过 ID 获取指定群组的详情）
    pub async fn open_the_door(&self, req: &GroupOpenTheDoorRequest) -> Result<GroupSearchItem> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        Validator::validate_coordinates(req.latitude, req.longitude)?;

        let group = self.group_repo.find_by_id(group_id).await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;

        let member_count = self.group_repo.get_member_count(group_id).await? as i32;
        let settings = self.settings_repo.find_by_group(group_id).await?;
        let has_password = settings.as_ref().and_then(|s| s.password_hash.as_ref()).is_some();

        // TODO: distance 和 degrees 应由 PostGIS 计算
        let distance = 0;
        let degrees = 0.0;

        // 获取标签
        let tag_repo = TagRepository::new(self.db.clone());
        let tags = tag_repo.get_group_tags(group_id).await?;
        let tags = if tags.is_empty() { None } else { Some(tags) };

        Ok(GroupSearchItem {
            id: group.group_id.to_string(),
            name: group.name,
            description: group.description,
            avatar_url: group.avatar_url,
            tags,
            members_count: member_count,
            has_password,
            geo: GeoData {
                latitude: group.latitude.unwrap_or(DEFAULT_LATITUDE),
                longitude: group.longitude.unwrap_or(DEFAULT_LONGITUDE),
                location_name: group.location_name.unwrap_or_else(|| DEFAULT_LOCATION_NAME.to_string()),
                distance,
                degrees,
            },
        })
    }

    /// 获取群组设置
    pub async fn get_settings(
        &self,
        req: GetGroupSettingsRequest,
    ) -> Result<GroupSettingsData> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        let group = self.group_repo.find_by_id(group_id).await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;

        let settings = self.settings_repo.find_by_group(group_id).await?;
        let has_password = settings.as_ref().map(|s| s.password_hash.is_some()).unwrap_or(false);

        Ok(GroupSettingsData {
            id: group.group_id.to_string(),
            settings: GroupSettings {
                join_method: Some(settings.as_ref().map(|s| s.join_method.clone()).unwrap_or_else(|| "open".to_string())),
                is_searchable: Some(settings.as_ref().map(|s| s.is_searchable).unwrap_or(true)),
                max_members: settings.as_ref().and_then(|s| s.max_members),
                password_hash: None, // 不返回密码哈希
                // TODO: 需要添加数据库迁移，将 longitude/latitude/location_name 字段添加到 group_settings 表
                longitude: None,
                latitude: None,
                location_name: None,
            },
            has_password,
            created_at: group.created_at.to_rfc3339(),
        })
    }

    pub async fn get_joined(
        &self,
        user_id: i64,
        req: GetJoinedGroupChatsRequest,
    ) -> Result<PaginatedData<JoinedGroupItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;

        // 解析 cursor（必须是 UUID）
        let cursor = match &req.cursor {
            Some(c) => Some(Uuid::parse_str(c).map_err(|_| AppError::BadRequest("无效的 cursor".to_string()))?),
            None => None,
        };

        let memberships = self.member_repo.list_by_user(user_id, limit + 1, cursor).await?;
        
        let has_more = memberships.len() > limit as usize;
        let memberships_to_return: Vec<_> = memberships.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            memberships_to_return.last().map(|m| m.cursor_id.to_string())
        } else {
            None
        };

        let mut items = Vec::new();
        for m in &memberships_to_return {
            if let Some(g) = self.group_repo.find_by_id(m.group_chat_id).await? {
                let member_count = self.group_repo.get_member_count(g.group_id).await?;
                items.push(JoinedGroupItem {
                    group_info: GroupInfo {
                        id: g.group_id.to_string(),
                        name: g.name.clone(),
                        avatar_url: g.avatar_url.clone(),
                        members_count: member_count as i32,
                    },
                    my_role: m.role.to_string(),
                    unread_count: 0,
                    last_activity_at: g.updated_at.unwrap_or(g.created_at).to_rfc3339(),
                });
            }
        }

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    pub async fn update_group(&self, user_id: i64, req: UpdateGroupChatRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        // TODO: 将 avatar_file_id 解析为 avatar_url（需要查询 uploaded_files 表）
        let avatar_url = req.avatar_file_id;

        self.group_repo.update(
            group_id,
            req.name,
            req.description,
            avatar_url
        ).await?;

        if let Some(s) = req.settings {
            // settings_repo.update 签名: (group_chat_id, join_method?, is_searchable?, max_members?, password_hash?)
            self.settings_repo.update(
                group_id,
                s.join_method.as_deref(),
                s.is_searchable,
                s.max_members,
                s.password_hash.as_deref(),
            ).await?;
        }

        // 更新标签
        if let Some(tags) = &req.tags {
            let tag_repo = TagRepository::new(self.db.clone());
            tag_repo.set_group_tags(group_id, tags).await?;
        }

        Ok(())
    }

    pub async fn delete_group(&self, user_id: i64, req: crate::dto::request::DeleteGroupChatRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        let group = self.group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("Group not found".to_string()))?;

        if group.owner_id != user_id {
            return Err(AppError::Forbidden("Only owner can delete group".to_string()));
        }

        self.group_repo.soft_delete(group_id).await?;
        
        // 发布群组解散事件
        self.event_bus.publish(AppEvent::GroupDismissed {
            group_id,
            dismissed_by: user_id,
        });
        
        Ok(())
    }

    pub async fn get_members(&self, req: GetMembersRequest) -> Result<PaginatedData<GroupMemberItem>> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        
        let limit = Validator::validate_limit(req.limit, 100)? as i64;

        // 解析 cursor（必须是 UUID）
        let cursor = match &req.cursor {
            Some(c) => Some(Uuid::parse_str(c).map_err(|_| AppError::BadRequest("无效的 cursor".to_string()))?),
            None => None,
        };

        let members = self.member_repo.list_by_group(group_id, limit as i32 + 1, cursor).await?;

        let has_more = members.len() > limit as usize;
        let members_to_return: Vec<GroupMemberRow> = members.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            members_to_return.last().map(|m| m.cursor_id.to_string())
        } else {
            None
        };

        let items = members_to_return.into_iter().map(|m| GroupMemberItem {
            user: UserInfo {
                user_id: m.user_id,
                nickname: m.nickname.unwrap_or_default(),
                avatar_url: m.avatar_url,
                user_type: m.user_type.unwrap_or(crate::types::enums::UserType::Normal),
            },
            role: m.role.to_string(),
            joined_at: m.joined_at.to_rfc3339(),
            muted_until: m.muted_until.map(|t| t.to_rfc3339()),
        }).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta { has_more, next_cursor },
        })
    }

    pub async fn invite_member(&self, user_id: i64, req: InviteMemberRequest) -> Result<InviteMemberData> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id;

        // Check permission
        if !self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::Forbidden("Not a member".to_string()));
        }

        let invite = self.invite_repo.create(group_id, user_id, target_uid, req.message).await?;

        Ok(InviteMemberData {
            id: invite.id.to_string(),
            expires_at: invite.expires_at.to_rfc3339(),
        })
    }

    pub async fn join_group(&self, user_id: i64, req: JoinGroupRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        // 检查群组是否存在
        if self.group_repo.find_by_id(group_id).await?.is_none() {
            return Err(AppError::NotFound("Group not found".to_string()));
        }

        // 检查是否被封禁
        if self.bans_repo.is_banned(group_id, user_id).await? {
            return Err(AppError::Forbidden("您已被封禁，无法加入此群组".to_string()));
        }

        if self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::BadRequest("Already a member".to_string()));
        }

        // 获取群组设置以检查密码
        let settings = self.settings_repo.find_by_group(group_id).await?;
        
        if let Some(hash) = &req.password_hash {
            if let Some(stored_hash) = settings.as_ref().and_then(|s| s.password_hash.as_ref()) {
                if !PasswordUtil::verify_password(hash, stored_hash)? {
                    return Err(AppError::GroupPasswordError("群组密码错误".to_string()));
                }
            }
        } else if settings.as_ref().and_then(|s| s.password_hash.as_ref()).is_some() {
            return Err(AppError::InvalidInput("该群组需要密码才能加入".to_string()));
        }

        self.member_repo.add_member(group_id, user_id, MemberRole::Member).await?;
        
        // 记录加入历史
        self.history_repo.record_join(group_id, user_id, None).await?;
        
        Ok(())
    }

    pub async fn apply_to_group(&self, user_id: i64, req: ApplyToGroupRequest) -> Result<ApplyToGroupData> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        if self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::BadRequest("Already a member".to_string()));
        }

        let app = self.application_repo.create(group_id, user_id, req.message).await?;

        Ok(ApplyToGroupData {
            id: app.id.to_string(),
        })
    }

    pub async fn set_role(&self, user_id: i64, req: SetMemberRoleRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) {
            return Err(AppError::Forbidden("Only owner can set roles".to_string()));
        }

        let old_role = self.member_repo.get_member_role(group_id, target_uid).await?
            .ok_or_else(|| AppError::NotFound("Member not found".to_string()))?;
        let new_role = MemberRole::from_str(&req.role).map_err(|e| AppError::BadRequest(e))?;
        
        self.member_repo.update_role(group_id, target_uid, new_role).await?;
        
        // 记录角色变更历史
        self.history_repo.record_role_changed(group_id, target_uid, user_id, old_role, new_role).await?;
        
        // 发布角色变更事件
        self.event_bus.publish(AppEvent::GroupMemberRoleChanged {
            group_id,
            user_id: target_uid,
            old_role: old_role.to_string(),
            new_role: new_role.to_string(),
        });
        
        Ok(())
    }

    pub async fn mute_member(&self, user_id: i64, req: MuteMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        let until = chrono::Utc::now() + chrono::Duration::minutes(req.duration_minutes as i64);
        self.member_repo.mute_member(group_id, target_uid, until).await?;
        
        // 记录禁言历史
        self.history_repo.record_muted(group_id, target_uid, user_id, until).await?;
        
        // 发布禁言事件
        self.event_bus.publish(AppEvent::GroupMemberMuted {
            group_id,
            user_id: target_uid,
            muted_until: until.to_rfc3339(),
        });
        
        Ok(())
    }

    pub async fn unmute_member(&self, user_id: i64, req: UnmuteMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.member_repo.unmute_member(group_id, target_uid).await?;
        
        // 记录解除禁言历史
        self.history_repo.record_unmuted(group_id, target_uid, user_id).await?;
        
        // 发布解除禁言事件
        self.event_bus.publish(AppEvent::GroupMemberUnmuted {
            group_id,
            user_id: target_uid,
        });
        
        Ok(())
    }

    /// 封禁成员
    pub async fn ban_member(&self, user_id: i64, req: BanMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id;

        // 检查权限
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        // 从群组移除成员
        self.member_repo.remove_member(group_id, target_uid).await?;
        
        // 添加到封禁表
        self.bans_repo.ban(group_id, target_uid, user_id, req.reason.clone()).await?;
        
        // 记录封禁历史
        self.history_repo.record_banned(group_id, target_uid, user_id, req.reason.clone()).await?;

        Ok(())
    }

    /// 解除封禁成员
    pub async fn unban_member(&self, user_id: i64, req: UnbanMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id;

        // 检查权限
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.bans_repo.unban(group_id, target_uid, user_id).await?;

        Ok(())
    }

    /// 获取封禁成员列表
    pub async fn get_banned_list(
        &self,
        user_id: i64,
        req: GetBannedMembersListRequest,
    ) -> Result<PaginatedData<BannedMemberItem>> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        // 检查权限（群主或管理员可以查看封禁列表）
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;

        let bans = self.bans_repo.list_by_group(group_id, limit + 1, cursor).await?;
        let has_more = bans.len() > limit as usize;
        
        // 先获取 cursor，再消费 bans
        let next_cursor = if has_more {
            bans.get(limit as usize - 1).map(|b| b.cursor_id.to_string())
        } else {
            None
        };

        let items: Vec<_> = bans
            .into_iter()
            .take(limit as usize)
            .map(|b| BannedMemberItem {
                user: UserInfo {
                    user_id: b.user_id,
                    nickname: b.nickname.unwrap_or_default(),
                    avatar_url: b.avatar_url,
                    user_type: b.user_type.unwrap_or(crate::types::enums::UserType::Normal),
                },
                reason: b.reason,
                banned_at: b.banned_at.to_rfc3339(),
            })
            .collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    pub async fn remove_member(&self, user_id: i64, req: RemoveMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.member_repo.remove_member(group_id, target_uid).await?;
        
        // 记录被移除历史
        self.history_repo.record_removed(group_id, target_uid, user_id, None).await?;
        
        Ok(())
    }

    pub async fn leave_group(&self, user_id: i64, req: LeaveGroupRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let group = self.group_repo.find_by_id(group_id).await?.ok_or(AppError::NotFound("Group not found".to_string()))?;

        if group.owner_id == user_id {
            return Err(AppError::BadRequest("Owner cannot leave".to_string()));
        }

        self.member_repo.leave_group(group_id, user_id).await?;
        
        // 记录离开历史
        self.history_repo.record_leave(group_id, user_id).await?;
        
        Ok(())
    }

    pub async fn get_applications(&self, _user_id: i64, req: GetGroupApplicationsListRequest) -> Result<PaginatedData<GroupApplicationItem>> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let limit = Validator::validate_limit(req.limit, 100)?;
        
        // 解析 cursor（必须是 UUID）
        let cursor = match &req.cursor {
            Some(c) => Some(Uuid::parse_str(c).map_err(|_| AppError::BadRequest("无效的 cursor".to_string()))?),
            None => None,
        };
        
        let status = req.status.as_ref()
            .map(|s| InviteStatus::from_str(s).map_err(|_| AppError::BadRequest("无效的 status".to_string())))
            .transpose()?;
        let apps = self.application_repo.list_by_group(group_id, status, limit as i32 + 1, cursor).await?;
        
        let has_more = apps.len() > limit as usize;
        let apps_to_return: Vec<_> = apps.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            apps_to_return.last().map(|a| a.cursor_id.to_string())
        } else {
            None
        };

        let items = apps_to_return.into_iter().map(|a| GroupApplicationItem {
            id: a.cursor_id.to_string(),
            applicant: UserInfo {
                user_id: a.applicant_id,
                nickname: "Unknown".to_string(), // Need join
                avatar_url: None,
                user_type: crate::types::enums::UserType::Normal,
            },
            message: a.message,
            status: a.status.to_string(),
            created_at: a.created_at.to_rfc3339(),
            handled_at: a.handled_at.map(|t| t.to_rfc3339()),
        }).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta { has_more, next_cursor },
        })
    }

    pub async fn approve_application(&self, user_id: i64, req: ApproveGroupApplicationRequest) -> Result<()> {
        let app_id = req.application_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("Invalid Application ID".to_string()))?;
        let app = self.application_repo.find_by_id(app_id).await?.ok_or(AppError::NotFound("Application not found".to_string()))?;
        
        // Check perms
        let role = self.member_repo.get_member_role(app.group_chat_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.application_repo.update_status(app_id, InviteStatus::Accepted, user_id).await?;
        self.member_repo.add_member(app.group_chat_id, app.applicant_id, MemberRole::Member).await?;
        
        // 记录加入历史（通过申请）
        self.history_repo.record_join(app.group_chat_id, app.applicant_id, Some(user_id)).await?;
        
        Ok(())
    }

    pub async fn reject_application(&self, user_id: i64, req: RejectGroupApplicationRequest) -> Result<()> {
        let app_id = req.application_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("Invalid Application ID".to_string()))?;
        let app = self.application_repo.find_by_id(app_id).await?.ok_or(AppError::NotFound("Application not found".to_string()))?;
        
        let role = self.member_repo.get_member_role(app.group_chat_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.application_repo.update_status(app_id, InviteStatus::Rejected, user_id).await?;
        Ok(())
    }

    /// 取消群组申请（申请者取消自己的待处理申请）
    pub async fn cancel_application(&self, user_id: i64, req: CancelGroupApplicationRequest) -> Result<()> {
        let app_id = req.application_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("Invalid Application ID".to_string()))?;
        let app = self.application_repo.find_by_id(app_id).await?
            .ok_or(AppError::NotFound("Application not found".to_string()))?;
        
        // 只能取消自己的申请
        if app.applicant_id != user_id {
            return Err(AppError::Forbidden("只能取消自己的申请".to_string()));
        }

        // 只能取消待处理的申请
        if app.status != InviteStatus::Pending {
            return Err(AppError::BadRequest("只能取消待处理的申请".to_string()));
        }

        self.application_repo.delete(app_id).await?;
        Ok(())
    }

    pub async fn transfer_ownership(&self, user_id: i64, req: TransferOwnershipRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let new_owner = req.new_owner_id;

        let group = self.group_repo.find_by_id(group_id).await?.ok_or(AppError::NotFound("Group not found".to_string()))?;
        if group.owner_id != user_id {
            return Err(AppError::Forbidden("Only owner can transfer".to_string()));
        }

        self.group_repo.transfer_ownership(group_id, new_owner).await?;
        self.member_repo.update_role(group_id, user_id, MemberRole::Admin).await?;
        self.member_repo.update_role(group_id, new_owner, MemberRole::Owner).await?;
        
        // 记录角色变更历史
        self.history_repo.record_role_changed(group_id, user_id, user_id, MemberRole::Owner, MemberRole::Admin).await?;
        self.history_repo.record_role_changed(group_id, new_owner, user_id, MemberRole::Member, MemberRole::Owner).await?;
        
        Ok(())
    }

    pub async fn get_invites(&self, user_id: i64, req: GetGroupInvitesListRequest) -> Result<PaginatedData<GroupInviteItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        
        // 解析 cursor（必须是 UUID）
        let cursor = match &req.cursor {
            Some(c) => Some(Uuid::parse_str(c).map_err(|_| AppError::BadRequest("无效的 cursor".to_string()))?),
            None => None,
        };
        
        let status = req.status.as_ref()
            .map(|s| InviteStatus::from_str(s).map_err(|_| AppError::BadRequest("无效的 status".to_string())))
            .transpose()?;
        let invites = self.invite_repo.list_by_user(user_id, status, limit as i32 + 1, cursor).await?;

        let has_more = invites.len() > limit as usize;
        let invites_to_return: Vec<_> = invites.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            invites_to_return.last().map(|i| i.cursor_id.to_string())
        } else {
            None
        };

        let items = invites_to_return.into_iter().map(|i| GroupInviteItem {
            id: i.cursor_id.to_string(),
            group: GroupInfo {
                id: i.group_chat_id.to_string(),
                name: "Unknown".to_string(),
                avatar_url: None,
                members_count: 0,
            },
            inviter: UserInfo {
                user_id: i.inviter_id,
                nickname: "Unknown".to_string(),
                avatar_url: None,
                user_type: crate::types::enums::UserType::Normal,
            },
            message: i.message,
            status: i.status.to_string(),
            created_at: i.created_at.to_rfc3339(),
            expires_at: Some(i.expires_at.to_rfc3339()),
        }).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta { has_more, next_cursor },
        })
    }

    pub async fn accept_invite(&self, user_id: i64, req: AcceptGroupInviteRequest) -> Result<()> {
        let invite_id = req.invite_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("Invalid Invite ID".to_string()))?;
        let invite = self.invite_repo.find_by_id(invite_id).await?.ok_or(AppError::NotFound("Invite not found".to_string()))?;

        if invite.invitee_id != user_id {
            return Err(AppError::Forbidden("Not your invite".to_string()));
        }

        self.invite_repo.update_status(invite_id, InviteStatus::Accepted).await?;
        self.member_repo.add_member(invite.group_chat_id, user_id, MemberRole::Member).await?;
        
        // 记录加入历史（通过邀请）
        self.history_repo.record_join(invite.group_chat_id, user_id, Some(invite.inviter_id)).await?;
        
        Ok(())
    }

    pub async fn reject_invite(&self, user_id: i64, req: RejectGroupInviteRequest) -> Result<()> {
        let invite_id = req.invite_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("Invalid Invite ID".to_string()))?;
        let invite = self.invite_repo.find_by_id(invite_id).await?.ok_or(AppError::NotFound("Invite not found".to_string()))?;

        if invite.invitee_id != user_id {
            return Err(AppError::Forbidden("Not your invite".to_string()));
        }

        self.invite_repo.update_status(invite_id, InviteStatus::Rejected).await?;
        Ok(())
    }
}
