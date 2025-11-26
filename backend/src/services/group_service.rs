use redis::aio::ConnectionManager;
use sqlx::PgPool;
use uuid::Uuid;
use std::str::FromStr;

use crate::{
    constants::{DEFAULT_LATITUDE, DEFAULT_LONGITUDE, DEFAULT_LOCATION_NAME},
    dto::{
        request::{
            CreateGroupChatRequest, UpdateGroupChatRequest, JoinGroupRequest, LeaveGroupRequest,
            RemoveMemberRequest, MuteMemberRequest, UnmuteMemberRequest, BanMemberRequest,
            SetMemberRoleRequest, TransferOwnershipRequest,
            DiscoverGroupChatsRequest, OpenTheDoorRequest, GetMembersRequest,
            InviteMemberRequest, ApplyToGroupRequest, GetGroupApplicationsRequest,
            ApproveGroupApplicationRequest, RejectGroupApplicationRequest,
            GetGroupInvitesRequest, AcceptGroupInviteRequest, RejectGroupInviteRequest,
            GetJoinedGroupChatsRequest,
        },
        response::{
            CreateGroupChatData, DiscoveredGroupItem, JoinedGroupItem, GroupDetailData, 
            GroupMemberItem, PaginatedData, PaginationMeta, UserInfo, GroupSettings,
            GroupInfo, InviteMemberData, ApplyToGroupData, GroupApplicationItem, GroupInviteItem,
        },
    },
    error::{AppError, Result},
    repositories::{
        group_repo::GroupRepository,
        group_member_repo::{GroupMemberRepository, GroupMemberRow},
        group_application_repo::GroupApplicationRepository,
        group_invite_repo::GroupInviteRepository,
        group_settings_repo::GroupSettingsRepository,
        user_repo::UserRepository,
    },
    types::enums::{MemberRole, InviteStatus},
    utils::{password::PasswordUtil, geo::GeoUtil},
};

pub struct GroupService {
    group_repo: GroupRepository,
    member_repo: GroupMemberRepository,
    application_repo: GroupApplicationRepository,
    invite_repo: GroupInviteRepository,
    settings_repo: GroupSettingsRepository,
    user_repo: UserRepository,
    #[allow(dead_code)]
    redis: ConnectionManager,
}

impl GroupService {
    pub fn new(pool: &PgPool, redis: &ConnectionManager) -> Self {
        Self {
            group_repo: GroupRepository::new(pool.clone()),
            member_repo: GroupMemberRepository::new(pool.clone()),
            application_repo: GroupApplicationRepository::new(pool.clone()),
            invite_repo: GroupInviteRepository::new(pool.clone()),
            settings_repo: GroupSettingsRepository::new(pool.clone()),
            user_repo: UserRepository::new(pool.clone()),
            redis: redis.clone(),
        }
    }

    pub async fn create_group(&self, owner_id: i64, req: CreateGroupChatRequest) -> Result<CreateGroupChatData> {
        let user = self.user_repo.find_by_id(owner_id).await?
            .ok_or(AppError::NotFound("User not found".to_string()))?;
        
        let lat = user.current_latitude.unwrap_or(DEFAULT_LATITUDE);
        let lng = user.current_longitude.unwrap_or(DEFAULT_LONGITUDE);
        let loc_name = user.current_location_name.clone().unwrap_or_else(|| DEFAULT_LOCATION_NAME.to_string());

        // 如果提供了密码，先进行哈希处理
        let hashed_password = if let Some(pwd) = &req.settings.password_hash {
            Some(PasswordUtil::hash_password(pwd)?)
        } else {
            None
        };

        let group = self.group_repo.create(
            req.name.clone(),
            req.description.clone(),
            req.avatar_url.clone(),
            owner_id,
            Some(lat),
            Some(lng),
            Some(loc_name),
        ).await?;

        self.member_repo.create(group.group_id, owner_id, MemberRole::Owner).await?;

        self.settings_repo.create(
            group.group_id,
            &req.settings.join_method,
            req.settings.is_searchable,
            req.settings.max_members,
            hashed_password.as_deref(),
        ).await?;

        Ok(CreateGroupChatData {
            group_chat_id: group.group_id.to_string(),
        })
    }

    pub async fn discover(
        &self,
        req: DiscoverGroupChatsRequest,
    ) -> Result<PaginatedData<DiscoveredGroupItem>> {
        let limit = req.limit.unwrap_or(20).min(100) as i32;
        
        // 使用优化后的批量查询方法，避免 N+1 问题
        let items = if let (Some(lat), Some(lng)) = (req.latitude, req.longitude) {
            GeoUtil::validate_coordinates(lat, lng)?;
            let radius = req.radius.unwrap_or(5000.0);
            
            // 一次查询获取群组信息、距离和成员数量
            let results = self.group_repo
                .search_nearby_with_details(lat, lng, radius, req.keyword.as_deref(), limit)
                .await?;
            
            results.into_iter().map(|(g, dist, member_count)| {
                DiscoveredGroupItem {
                    group_chat_id: g.group_id.to_string(),
                    name: g.name,
                    avatar_url: g.avatar_url,
                    member_count,
                    is_member: false,
                    location_name: g.location_name.unwrap_or_default(),
                    distance: dist as i32,
                }
            }).collect()
        } else if let Some(kw) = &req.keyword {
            // 一次查询获取群组信息和成员数量
            let results = self.group_repo
                .search_by_name_with_member_count(kw, limit, None)
                .await?;
            
            results.into_iter().map(|(g, member_count)| {
                DiscoveredGroupItem {
                    group_chat_id: g.group_id.to_string(),
                    name: g.name,
                    avatar_url: g.avatar_url,
                    member_count,
                    is_member: false,
                    location_name: g.location_name.unwrap_or_default(),
                    distance: 0,
                }
            }).collect()
        } else {
            vec![]
        };

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: false,
                next_cursor: None,
            },
        })
    }

    pub async fn get_joined(
        &self,
        user_id: i64,
        req: GetJoinedGroupChatsRequest,
    ) -> Result<PaginatedData<JoinedGroupItem>> {
        let limit = req.limit.unwrap_or(20).min(100) as i32;

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
                        group_chat_id: g.group_id.to_string(),
                        name: g.name.clone(),
                        avatar_url: g.avatar_url.clone(),
                        member_count: member_count as i32,
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

    pub async fn open_the_door(
        &self,
        user_id: i64,
        req: OpenTheDoorRequest,
    ) -> Result<GroupDetailData> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        
        let group = self.group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("Group not found".to_string()))?;
        
        let member_count = self.group_repo.get_member_count(group_id).await?;
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        let settings = self.settings_repo.find_by_group(group_id).await?;

        let settings_data = settings.as_ref();
        Ok(GroupDetailData {
            group_chat_id: group.group_id.to_string(),
            name: group.name,
            description: group.description,
            avatar_url: group.avatar_url,
            member_count: member_count as i32,
            my_role: role.map(|r| r.to_string()).unwrap_or_else(|| "none".to_string()),
            settings: GroupSettings {
                join_method: settings_data.map(|s| s.join_method.clone()).unwrap_or_else(|| "open".to_string()),
                is_searchable: settings_data.map(|s| s.is_searchable).unwrap_or(true),
                max_members: settings_data.and_then(|s| s.max_members),
                password_hash: settings_data.and_then(|s| s.password_hash.clone()),
            },
            created_at: group.created_at.to_rfc3339(),
            is_member: role.is_some(),
        })
    }

    pub async fn update_group(&self, user_id: i64, req: UpdateGroupChatRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.group_repo.update(
            group_id,
            req.name,
            req.description,
            req.avatar_url
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

        Ok(())
    }

    pub async fn delete_group(&self, user_id: i64, req: crate::dto::request::DeleteGroupChatRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        let group = self.group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("Group not found".to_string()))?;

        if group.owner_id != user_id {
            return Err(AppError::Forbidden("Only owner can delete group".to_string()));
        }

        self.group_repo.soft_delete(group_id).await?;
        Ok(())
    }

    pub async fn get_members(&self, req: GetMembersRequest) -> Result<PaginatedData<GroupMemberItem>> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        
        let limit = req.limit.unwrap_or(20).min(100) as i64;

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
                user_id: m.user_id.to_string(),
                nickname: m.nickname.unwrap_or_default(),
                avatar_url: m.avatar_url,
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
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("Invalid User ID".to_string()))?;

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
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        // 检查群组是否存在
        if self.group_repo.find_by_id(group_id).await?.is_none() {
            return Err(AppError::NotFound("Group not found".to_string()));
        }

        if self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::BadRequest("Already a member".to_string()));
        }

        // 获取群组设置以检查密码
        let settings = self.settings_repo.find_by_group(group_id).await?;
        
        if let Some(hash) = &req.password_hash {
            if let Some(stored_hash) = settings.as_ref().and_then(|s| s.password_hash.as_ref()) {
                if !PasswordUtil::verify_password(hash, stored_hash)? {
                    return Err(AppError::Unauthorized("Incorrect password".to_string()));
                }
            }
        } else if settings.as_ref().and_then(|s| s.password_hash.as_ref()).is_some() {
            return Err(AppError::BadRequest("Password required".to_string()));
        }

        self.member_repo.create(group_id, user_id, MemberRole::Member).await?;
        Ok(())
    }

    pub async fn apply_to_group(&self, user_id: i64, req: ApplyToGroupRequest) -> Result<ApplyToGroupData> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
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
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("Invalid User ID".to_string()))?;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) {
            return Err(AppError::Forbidden("Only owner can set roles".to_string()));
        }

        let new_role = MemberRole::from_str(&req.role).map_err(|e| AppError::BadRequest(e))?;
        self.member_repo.update_role(group_id, target_uid, new_role).await?;
        Ok(())
    }

    pub async fn mute_member(&self, user_id: i64, req: MuteMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("Invalid User ID".to_string()))?;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        let until = chrono::Utc::now() + chrono::Duration::minutes(req.duration_minutes as i64);
        self.member_repo.mute_member(group_id, target_uid, until).await?;
        Ok(())
    }

    pub async fn unmute_member(&self, user_id: i64, req: UnmuteMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("Invalid User ID".to_string()))?;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.member_repo.unmute_member(group_id, target_uid).await?;
        Ok(())
    }

    pub async fn ban_member(&self, user_id: i64, req: BanMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("Invalid User ID".to_string()))?;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.member_repo.remove_member(group_id, target_uid).await?;
        // TODO: Add to blacklist table (omitted for brevity)
        Ok(())
    }

    pub async fn remove_member(&self, user_id: i64, req: RemoveMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("Invalid User ID".to_string()))?;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.member_repo.remove_member(group_id, target_uid).await?;
        Ok(())
    }

    pub async fn leave_group(&self, user_id: i64, req: LeaveGroupRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let group = self.group_repo.find_by_id(group_id).await?.ok_or(AppError::NotFound("Group not found".to_string()))?;

        if group.owner_id == user_id {
            return Err(AppError::BadRequest("Owner cannot leave".to_string()));
        }

        self.member_repo.leave_group(group_id, user_id).await?;
        Ok(())
    }

    pub async fn get_applications(&self, _user_id: i64, req: GetGroupApplicationsRequest) -> Result<PaginatedData<GroupApplicationItem>> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let limit = req.limit.unwrap_or(20).min(100);
        
        // 解析 cursor（必须是 UUID）
        let cursor = match &req.cursor {
            Some(c) => Some(Uuid::parse_str(c).map_err(|_| AppError::BadRequest("无效的 cursor".to_string()))?),
            None => None,
        };
        
        let status = req.status.as_ref().and_then(|s| InviteStatus::from_str(s).ok());
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
                user_id: a.applicant_id.to_string(),
                nickname: "Unknown".to_string(), // Need join
                avatar_url: None,
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
        self.member_repo.create(app.group_chat_id, app.applicant_id, MemberRole::Member).await?;
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

    pub async fn transfer_ownership(&self, user_id: i64, req: TransferOwnershipRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let new_owner = req.new_owner_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("Invalid User ID".to_string()))?;

        let group = self.group_repo.find_by_id(group_id).await?.ok_or(AppError::NotFound("Group not found".to_string()))?;
        if group.owner_id != user_id {
            return Err(AppError::Forbidden("Only owner can transfer".to_string()));
        }

        self.group_repo.transfer_ownership(group_id, new_owner).await?;
        self.member_repo.update_role(group_id, user_id, MemberRole::Admin).await?;
        self.member_repo.update_role(group_id, new_owner, MemberRole::Owner).await?;
        Ok(())
    }

    pub async fn get_invites(&self, user_id: i64, req: GetGroupInvitesRequest) -> Result<PaginatedData<GroupInviteItem>> {
        let limit = req.limit.unwrap_or(20).min(100);
        
        // 解析 cursor（必须是 UUID）
        let cursor = match &req.cursor {
            Some(c) => Some(Uuid::parse_str(c).map_err(|_| AppError::BadRequest("无效的 cursor".to_string()))?),
            None => None,
        };
        
        let status = req.status.as_ref().and_then(|s| InviteStatus::from_str(s).ok());
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
                group_chat_id: i.group_chat_id.to_string(),
                name: "Unknown".to_string(),
                avatar_url: None,
                member_count: 0,
            },
            inviter: UserInfo {
                user_id: i.inviter_id.to_string(),
                nickname: "Unknown".to_string(),
                avatar_url: None,
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
        self.member_repo.create(invite.group_chat_id, user_id, MemberRole::Member).await?;
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
