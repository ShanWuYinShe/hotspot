use redis::aio::ConnectionManager;
use sqlx::PgPool;
use std::str::FromStr;
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    constants::{DEFAULT_LATITUDE, DEFAULT_LOCATION_NAME, DEFAULT_LONGITUDE},
    dto::{
        request::{
            Bounds,
            AcceptInviteRequest, ApplyToGroupRequest, ApproveApplicationRequest, BanMemberRequest,
            BannedMembersRequest, CancelApplicationRequest, CreateGroupChatRequest,
            GroupApplicationsRequest, GroupMembersRequest, GroupOpenTheDoorRequest,
            GroupSettingsRequest, InviteMemberRequest, JoinGroupRequest, LeaveGroupRequest,
            MuteMemberRequest, MyGroupChatsRequest, MyInvitesRequest, RejectApplicationRequest,
            RejectInviteRequest, RemoveMemberRequest, SearchGroupsByKeywordRequest,
            SearchGroupsByLocationRequest, SearchGroupsByTagRequest, SetMemberRoleRequest,
            TransferOwnershipRequest, UnbanMemberRequest, UnmuteMemberRequest,
            UpdateGroupChatRequest,
        },
        response::{
            ApplyToGroupData, BannedMemberItem, CreateGroupChatData, GeoData, GroupApplicationItem,
            GroupInfo, GroupInviteItem, GroupMemberItem, GroupSearchItem, GroupSettings,
            GroupSettingsData, InviteMemberData, JoinedGroupItem, PaginatedData, PaginationMeta,
            UserInfo,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::{
        chat_unread_repo::ChatUnreadRepository, group_application_repo::GroupApplicationRepository,
        group_bans_repo::GroupBansRepository, group_invite_repo::GroupInviteRepository,
        group_member_history_repo::GroupMemberHistoryRepository,
        group_member_repo::GroupMemberRepository, group_repo::GroupRepository,
        group_settings_repo::GroupSettingsRepository, tag_repo::TagRepository,
        user_repo::UserRepository,
    },
    types::enums::{InviteStatus, MemberRole},
    utils::{password::PasswordUtil, validation::Validator},
    EventBus,
};

#[derive(Clone)]
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
    file_repo: crate::repositories::file_repo::FileRepository,
    unread_repo: ChatUnreadRepository,
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
            file_repo: crate::repositories::file_repo::FileRepository::new(pool.clone()),
            unread_repo: ChatUnreadRepository::new(pool.clone()),
            _redis: redis.clone(),
        }
    }

    pub async fn create_group(
        &self,
        owner_id: i64,
        req: CreateGroupChatRequest,
    ) -> Result<CreateGroupChatData> {
        let user = self
            .user_repo
            .find_by_id(owner_id)
            .await?
            .ok_or(AppError::NotFound("User not found".to_string()))?;

        let lat = user.current_latitude.unwrap_or(DEFAULT_LATITUDE);
        let lng = user.current_longitude.unwrap_or(DEFAULT_LONGITUDE);
        let loc_name = user
            .current_location_name
            .clone()
            .unwrap_or_else(|| DEFAULT_LOCATION_NAME.to_string());

        // 获取设置（使用默认值如果未提供）
        let settings = req.settings.unwrap_or_default();

        // 如果提供了密码，先进行哈希处理
        let hashed_password = if let Some(pwd) = &settings.password {
            Some(PasswordUtil::hash_password(pwd)?)
        } else {
            None
        };

        // 解析 avatar_file_id 为 avatar_url
        let mut avatar_url = None;
        if let Some(file_id_str) = req.avatar_file_id {
            if let Ok(file_id) = Uuid::parse_str(&file_id_str) {
                // 验证文件所有权和分类 ('group_avatar')
                if let Ok(url) = self
                    .file_repo
                    .validate_file(file_id, owner_id, "group_avatar")
                    .await
                {
                    avatar_url = Some(url);
                    // 标记文件已使用
                    let _ = self.file_repo.mark_as_used(file_id).await;
                }
            }
        }

        let group = self
            .group_repo
            .create(
                req.name.clone(),
                req.description.clone(),
                avatar_url,
                owner_id,
                Some(lat),
                Some(lng),
                Some(loc_name),
            )
            .await?;

        self.member_repo
            .add_member(group.group_id, owner_id, MemberRole::Owner)
            .await?;

        // 记录群主加入历史
        self.history_repo
            .record_join(group.group_id, owner_id, None)
            .await?;

        self.settings_repo
            .create(
                group.group_id,
                &settings.join_method,
                settings.is_searchable,
                settings.max_members,
                hashed_password.as_deref(),
            )
            .await?;

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
        let cursor = req.cursor;

        Validator::validate_coordinates(req.latitude, req.longitude)?;

        let (results, next_cursor) = self
            .group_repo
            .search_by_name(&req.kw, req.latitude, req.longitude, limit, cursor)
            .await?;

        // 批量获取标签
        let group_ids: Vec<Uuid> = results.iter().map(|g| g.group_id).collect();
        let tags_map = self.tag_repo.get_group_tags_batch(&group_ids).await?;

        let items: Vec<GroupSearchItem> = results
            .into_iter()
            .map(|g| {
                let tags = tags_map.get(&g.group_id).cloned();
                let tags = if tags.as_ref().map(|t| t.is_empty()).unwrap_or(true) {
                    None
                } else {
                    tags
                };

                GroupSearchItem {
                    id: g.group_id.to_string(),
                    name: g.name,
                    description: g.description,
                    avatar_url: g.avatar_url,
                    tags,
                    members_count: g.member_count,
                    has_password: g.has_password,
                    geo: GeoData {
                        latitude: g.latitude,
                        longitude: g.longitude,
                        location_name: g.location_name,
                        distance: g.distance as i32,
                        degrees: g.degrees,
                    },
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

    /// 按标签搜索群组（全局）
    pub async fn search_by_tag(
        &self,
        req: SearchGroupsByTagRequest,
    ) -> Result<PaginatedData<GroupSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = req.cursor;

        let lat = req.latitude;
        let lng = req.longitude;
        Validator::validate_coordinates(lat, lng)?;

        // 通过标签搜索群组 (Reuse search_by_name logic or implement search_by_tag in repo?)
        // Currently relying on search_by_name for text search, but tag search is specific.
        // Wait, GroupRepo doesn't have search_by_tag?
        // TagRepo has search_users_by_tag.
        // GroupRepo SHOULD have search_by_tag or search_by_name supports tags?
        // search_by_name is `name ILIKE $3 OR description ILIKE $3`. Not tag specific.
        // But TagRepo has `get_group_tags`.
        // I should add `search_by_tag` to GroupRepo? Or use `TagRepo` to find group IDs and then fetch groups?
        // TagRepo usually handles tag-based search.
        // But for now, let's use search_by_name as a fallback or assume TagRepo handles it if implemented.
        // Actually, previous code called `search_by_name` with `req.tag` as keyword. That searches name/desc, not tags.
        // Tag search is implemented in repo (search_by_tag).
        // I will keep it as is (using `search_by_name`) but update signatures.

        let (results, next_cursor) = self
            .group_repo
            .search_by_tag(&req.tag, lat, lng, limit, cursor)
            .await?;

        // 批量获取标签
        let group_ids: Vec<Uuid> = results.iter().map(|g| g.group_id).collect();
        let tags_map = self.tag_repo.get_group_tags_batch(&group_ids).await?;

        let items: Vec<GroupSearchItem> = results
            .into_iter()
            .map(|g| {
                let tags = tags_map.get(&g.group_id).cloned();
                let tags = if tags.as_ref().map(|t| t.is_empty()).unwrap_or(true) {
                    None
                } else {
                    tags
                };

                GroupSearchItem {
                    id: g.group_id.to_string(),
                    name: g.name,
                    description: g.description,
                    avatar_url: g.avatar_url,
                    tags,
                    members_count: g.member_count,
                    has_password: g.has_password,
                    geo: GeoData {
                        latitude: g.latitude,
                        longitude: g.longitude,
                        location_name: g.location_name,
                        distance: g.distance as i32,
                        degrees: g.degrees,
                    },
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

    /// 按位置搜索群组（范围过滤）
    pub async fn search_by_location(
        &self,
        req: SearchGroupsByLocationRequest,
    ) -> Result<PaginatedData<GroupSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = req.cursor;

        let lat = req.latitude;
        let lng = req.longitude;
        Validator::validate_coordinates(lat, lng)?;
        Validator::validate_radius_required(req.radius, 50000.0)?;

        let (results, next_cursor) = self
            .group_repo
            .search_by_location(req.kw.as_deref(), lat, lng, req.radius, limit, cursor)
            .await?;

        // 批量获取标签
        let group_ids: Vec<Uuid> = results.iter().map(|g| g.group_id).collect();
        let tags_map = self.tag_repo.get_group_tags_batch(&group_ids).await?;

        let items: Vec<GroupSearchItem> = results
            .into_iter()
            .map(|g| {
                let tags = tags_map.get(&g.group_id).cloned();
                let tags = if tags.as_ref().map(|t| t.is_empty()).unwrap_or(true) {
                    None
                } else {
                    tags
                };

                GroupSearchItem {
                    id: g.group_id.to_string(),
                    name: g.name,
                    description: g.description,
                    avatar_url: g.avatar_url,
                    tags,
                    members_count: g.member_count,
                    has_password: g.has_password,
                    geo: GeoData {
                        latitude: g.latitude,
                        longitude: g.longitude,
                        location_name: g.location_name,
                        distance: g.distance as i32,
                        degrees: g.degrees,
                    },
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

    /// 按视口 bounds 获取群组（用于地图：替代 search_by_location）
    pub async fn search_by_bounds(
        &self,
        bounds: Bounds,
        latitude: f64,
        longitude: f64,
        limit: Option<i32>,
    ) -> Result<Vec<GroupSearchItem>> {
        Validator::validate_coordinates(latitude, longitude)?;
        Validator::validate_coordinates(bounds.north, bounds.east)?;
        Validator::validate_coordinates(bounds.south, bounds.west)?;
        let limit = Validator::validate_limit(limit, 200)?;

        let results = self
            .group_repo
            .search_by_bounds(
                bounds.west,
                bounds.south,
                bounds.east,
                bounds.north,
                latitude,
                longitude,
                limit,
            )
            .await?;

        let group_ids: Vec<Uuid> = results.iter().map(|g| g.group_id).collect();
        let tags_map = self.tag_repo.get_group_tags_batch(&group_ids).await?;

        Ok(results
            .into_iter()
            .map(|g| {
                let tags = tags_map.get(&g.group_id).cloned();
                let tags = if tags.as_ref().map(|t| t.is_empty()).unwrap_or(true) {
                    None
                } else {
                    tags
                };

                GroupSearchItem {
                    id: g.group_id.to_string(),
                    name: g.name,
                    description: g.description,
                    avatar_url: g.avatar_url,
                    tags,
                    members_count: g.member_count,
                    has_password: g.has_password,
                    geo: GeoData {
                        latitude: g.latitude,
                        longitude: g.longitude,
                        location_name: g.location_name,
                        distance: g.distance as i32,
                        degrees: g.degrees,
                    },
                }
            })
            .collect())
    }

    /// 获取群组详情（通过 ID 获取指定群组的详情）
    pub async fn open_the_door(&self, req: &GroupOpenTheDoorRequest) -> Result<GroupSearchItem> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        Validator::validate_coordinates(req.latitude, req.longitude)?;

        let groups = self
            .group_repo
            .find_by_ids_with_distance(&[group_id], req.latitude, req.longitude)
            .await?;

        let group = groups
            .first()
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;

        // 获取标签
        let tag_repo = TagRepository::new(self.db.clone());
        let tags = tag_repo.get_group_tags(group_id).await?;
        let tags = if tags.is_empty() { None } else { Some(tags) };

        Ok(GroupSearchItem {
            id: group.group_id.to_string(),
            name: group.name.clone(),
            description: group.description.clone(),
            avatar_url: group.avatar_url.clone(),
            tags,
            members_count: group.member_count,
            has_password: group.has_password,
            geo: GeoData {
                latitude: group.latitude,
                longitude: group.longitude,
                location_name: group.location_name.clone(),
                distance: group.distance as i32,
                degrees: group.degrees,
            },
        })
    }

    /// 获取群组设置
    pub async fn get_settings(&self, req: GroupSettingsRequest) -> Result<GroupSettingsData> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        let group = self
            .group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;

        let settings = self.settings_repo.find_by_group(group_id).await?;
        let has_password = settings
            .as_ref()
            .map(|s| s.password_hash.is_some())
            .unwrap_or(false);

        Ok(GroupSettingsData {
            id: group.group_id.to_string(),
            settings: GroupSettings {
                join_method: Some(
                    settings
                        .as_ref()
                        .map(|s| s.join_method.clone())
                        .unwrap_or_else(|| "open".to_string()),
                ),
                is_searchable: Some(settings.as_ref().map(|s| s.is_searchable).unwrap_or(true)),
                max_members: settings.as_ref().and_then(|s| s.max_members),
                // 从 group_chats 表获取位置信息
                longitude: group.longitude,
                latitude: group.latitude,
                location_name: group.location_name.clone(),
            },
            has_password,
            created_at: group.created_at.to_rfc3339(),
        })
    }

    pub async fn get_joined(
        &self,
        user_id: i64,
        req: MyGroupChatsRequest,
    ) -> Result<PaginatedData<JoinedGroupItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = req.cursor;

        // 1. 获取成员关系列表
        let (memberships, next_cursor) = self
            .member_repo
            .list_by_user(user_id, limit, cursor)
            .await?;

        if memberships.is_empty() {
            return Ok(PaginatedData {
                items: Vec::new(),
                pagination: PaginationMeta {
                    has_more: false,
                    next_cursor: None,
                },
            });
        }

        // 2. 收集 ID
        let group_ids: Vec<Uuid> = memberships.iter().map(|m| m.group_chat_id).collect();

        // 3. 批量查询（并行执行）
        // 使用 find_by_ids_with_distance 获取详情，传入 0,0 作为坐标（此处不关心距离）
        let (groups, unread_counts) = tokio::try_join!(
            self.group_repo
                .find_by_ids_with_distance(&group_ids, 0.0, 0.0),
            self.unread_repo
                .get_unread_count_batch("group_chat", &group_ids, user_id)
        )?;

        // 4. 构建映射
        let group_map: std::collections::HashMap<Uuid, _> =
            groups.into_iter().map(|g| (g.group_id, g)).collect();

        // 5. 组装结果
        let mut items = Vec::with_capacity(memberships.len());
        for m in &memberships {
            if let Some(g) = group_map.get(&m.group_chat_id) {
                let unread_count = unread_counts.get(&g.group_id).copied().unwrap_or(0);

                items.push(JoinedGroupItem {
                    group_info: GroupInfo {
                        id: g.group_id.to_string(),
                        name: g.name.clone(),
                        avatar_url: g.avatar_url.clone(),
                        members_count: g.member_count,
                    },
                    my_role: m.role.to_string(),
                    unread_count: unread_count as i32,
                    last_activity_at: g.updated_at.unwrap_or(g.created_at).to_rfc3339(),
                });
            }
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
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        // 保留变更字段（用于 WS 推送）
        let updated_name = req.name.clone();
        let updated_description = req.description.clone();
        let has_avatar_update = req.avatar_file_id.is_some();

        // 解析 avatar_file_id 为 avatar_url
        let mut avatar_url = req.avatar_file_id.clone();
        if let Some(file_id_str) = &req.avatar_file_id {
            if let Ok(file_id) = Uuid::parse_str(file_id_str) {
                // 验证文件所有权和分类 ('group_avatar')
                if let Ok(url) = self
                    .file_repo
                    .validate_file(file_id, user_id, "group_avatar")
                    .await
                {
                    avatar_url = Some(url);
                    // 标记文件已使用
                    let _ = self.file_repo.mark_as_used(file_id).await;
                }
            }
        }

        let avatar_url_for_event = avatar_url.clone();
        self.group_repo
            .update(group_id, req.name, req.description, avatar_url)
            .await?;

        if let Some(s) = req.settings {
            // password：输入为明文；存储为 bcrypt hash（不回显）
            let password_hash = if let Some(pwd) = s.password.as_deref() {
                Some(PasswordUtil::hash_password(pwd)?)
            } else {
                None
            };

            // settings_repo.update 签名: (group_chat_id, join_method?, is_searchable?, max_members?, password_hash?)
            self.settings_repo
                .update(
                    group_id,
                    s.join_method.as_deref(),
                    s.is_searchable,
                    s.max_members,
                    password_hash.as_deref(),
                )
                .await?;

            // 如果提供了位置信息，更新群组位置
            if let (Some(lat), Some(lng)) = (s.latitude, s.longitude) {
                // 如果提供了位置名称，则使用提供的名称，否则保留原名称或使用默认名称
                // 这里为了简化，如果 location_name 不存在，可以使用默认值或者当前值
                // 实际上我们应该查询当前值，但这里可以要求客户端传完整的 location_name 如果传了 lat/lng
                // 或者我们可以先查询当前群组信息
                let loc_name = s
                    .location_name
                    .unwrap_or_else(|| DEFAULT_LOCATION_NAME.to_string());

                self.group_repo
                    .update_location(group_id, lat, lng, loc_name)
                    .await?;
            }
        }

        // 更新标签
        if let Some(tags) = &req.tags {
            let tag_repo = TagRepository::new(self.db.clone());
            tag_repo.set_group_tags(group_id, tags).await?;
        }

        // 发布群组资料更新事件（群聊订阅者可实时刷新 UI）
        if updated_name.is_some() || updated_description.is_some() || has_avatar_update {
            self.event_bus.publish(AppEvent::GroupProfileUpdated {
                group_id,
                name: updated_name,
                description: updated_description,
                avatar_url: avatar_url_for_event,
            });
        }

        Ok(())
    }

    pub async fn delete_group(
        &self,
        user_id: i64,
        req: crate::dto::request::DeleteGroupChatRequest,
    ) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        // 验证所有权
        let owner_id = self
            .group_repo
            .get_owner(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("Group not found".to_string()))?;

        if owner_id != user_id {
            return Err(AppError::Forbidden(
                "Only owner can delete group".to_string(),
            ));
        }

        self.group_repo.soft_delete(group_id).await?;

        // 发布群组解散事件
        self.event_bus.publish(AppEvent::GroupDismissed {
            group_id,
            dismissed_by: user_id,
        });

        Ok(())
    }

    pub async fn get_members(
        &self,
        req: GroupMembersRequest,
    ) -> Result<PaginatedData<GroupMemberItem>> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor = req.cursor;

        let role = req.role.as_deref();
        let (members, next_cursor) = self
            .member_repo
            .list_by_group(group_id, role, limit as i32, cursor)
            .await?;

        let items = members
            .into_iter()
            .map(|m| GroupMemberItem {
                user: UserInfo {
                    user_id: m.user_id,
                    nickname: m.nickname.unwrap_or_default(),
                    avatar_url: m.avatar_url,
                    user_type: m.user_type.unwrap_or(crate::types::enums::UserType::Normal),
                },
                role: m.role.to_string(),
                joined_at: m.joined_at.to_rfc3339(),
                muted_until: m.muted_until.map(|t| t.to_rfc3339()),
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

    pub async fn invite_member(
        &self,
        user_id: i64,
        req: InviteMemberRequest,
    ) -> Result<InviteMemberData> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id;

        // Check permission
        if !self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::Forbidden("Not a member".to_string()));
        }

        let invite = self
            .invite_repo
            .create(group_id, user_id, target_uid, req.message)
            .await?;

        // 发布“群组邀请已发送”事件（用于 WS 通知被邀请者）
        let group_name = self
            .group_repo
            .find_by_id(group_id)
            .await?
            .map(|g| g.name)
            .unwrap_or_else(|| "Unknown".to_string());
        self.event_bus.publish(AppEvent::GroupInviteSent {
            invite_id: invite.invite_id,
            group_id,
            group_name,
            from_user_id: user_id,
            to_user_id: target_uid,
            message: invite.message.clone(),
        });

        Ok(InviteMemberData {
            id: invite.invite_id.to_string(),
            expires_at: invite.expires_at.to_rfc3339(),
        })
    }

    pub async fn join_group(&self, user_id: i64, req: JoinGroupRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        // 检查群组是否存在
        if !self.group_repo.exists(group_id).await? {
            return Err(AppError::NotFound("Group not found".to_string()));
        }

        // 检查是否被封禁
        if self.bans_repo.is_banned(group_id, user_id).await? {
            return Err(AppError::Forbidden(
                "您已被封禁，无法加入此群组".to_string(),
            ));
        }

        if self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::BadRequest("Already a member".to_string()));
        }

        // 获取群组设置以检查密码
        let settings = self.settings_repo.find_by_group(group_id).await?;

        if let Some(password) = &req.password {
            if let Some(stored_hash) = settings.as_ref().and_then(|s| s.password_hash.as_ref()) {
                if !PasswordUtil::verify_password(password, stored_hash)? {
                    return Err(AppError::GroupPasswordError("群组密码错误".to_string()));
                }
            }
        } else if settings
            .as_ref()
            .and_then(|s| s.password_hash.as_ref())
            .is_some()
        {
            return Err(AppError::InvalidInput("该群组需要密码才能加入".to_string()));
        }

        self.member_repo
            .add_member(group_id, user_id, MemberRole::Member)
            .await?;

        // 记录加入历史
        self.history_repo
            .record_join(group_id, user_id, None)
            .await?;

        // 发布成员加入事件（群聊订阅者可实时看到成员变化）
        self.event_bus.publish(AppEvent::GroupMemberAdded {
            group_id,
            user_id,
            added_by: user_id,
        });

        Ok(())
    }

    pub async fn apply_to_group(
        &self,
        user_id: i64,
        req: ApplyToGroupRequest,
    ) -> Result<ApplyToGroupData> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        if self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::BadRequest("Already a member".to_string()));
        }

        let app = self
            .application_repo
            .create(group_id, user_id, req.message)
            .await?;

        // 发布“收到入群申请”事件（通知群主/管理员）
        self.event_bus.publish(AppEvent::GroupApplicationReceived {
            application_id: app.application_id,
            group_id,
            applicant_id: user_id,
            message: app.message.clone(),
        });

        Ok(ApplyToGroupData {
            id: app.application_id.to_string(),
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

        let old_role = self
            .member_repo
            .get_member_role(group_id, target_uid)
            .await?
            .ok_or_else(|| AppError::NotFound("Member not found".to_string()))?;
        let new_role = MemberRole::from_str(&req.role).map_err(AppError::BadRequest)?;

        self.member_repo
            .update_role(group_id, target_uid, new_role)
            .await?;

        // 记录角色变更历史
        self.history_repo
            .record_role_changed(group_id, target_uid, user_id, old_role, new_role)
            .await?;

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
        self.member_repo
            .mute_member(group_id, target_uid, until)
            .await?;

        // 记录禁言历史
        self.history_repo
            .record_muted(group_id, target_uid, user_id, until)
            .await?;

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
        self.history_repo
            .record_unmuted(group_id, target_uid, user_id)
            .await?;

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
        self.bans_repo
            .ban(group_id, target_uid, user_id, req.reason.clone())
            .await?;

        // 记录封禁历史
        self.history_repo
            .record_banned(group_id, target_uid, user_id, req.reason.clone())
            .await?;

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
        req: BannedMembersRequest,
    ) -> Result<PaginatedData<BannedMemberItem>> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        // 检查权限（群主或管理员可以查看封禁列表）
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = req.cursor;

        let (bans, next_cursor) = self
            .bans_repo
            .list_by_group(group_id, limit, cursor)
            .await?;

        let items: Vec<_> = bans
            .into_iter()
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
                has_more: next_cursor.is_some(),
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
        self.history_repo
            .record_removed(group_id, target_uid, user_id, None)
            .await?;

        // 发布成员离开事件（被移除）
        self.event_bus.publish(AppEvent::GroupMemberRemoved {
            group_id,
            user_id: target_uid,
            removed_by: user_id,
        });

        Ok(())
    }

    pub async fn leave_group(&self, user_id: i64, req: LeaveGroupRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        let owner_id = self
            .group_repo
            .get_owner(group_id)
            .await?
            .ok_or(AppError::NotFound("Group not found".to_string()))?;

        if owner_id == user_id {
            return Err(AppError::BadRequest("Owner cannot leave".to_string()));
        }

        self.member_repo.leave_group(group_id, user_id).await?;

        // 记录离开历史
        self.history_repo.record_leave(group_id, user_id).await?;

        // 发布成员离开事件（主动退出）
        self.event_bus.publish(AppEvent::GroupMemberRemoved {
            group_id,
            user_id,
            removed_by: user_id,
        });

        Ok(())
    }

    pub async fn get_applications(
        &self,
        _user_id: i64,
        req: GroupApplicationsRequest,
    ) -> Result<PaginatedData<GroupApplicationItem>> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = req.cursor;

        let status = req
            .status
            .as_ref()
            .map(|s| {
                InviteStatus::from_str(s)
                    .map_err(|_| AppError::BadRequest("无效的 status".to_string()))
            })
            .transpose()?;
        let (apps, next_cursor) = self
            .application_repo
            .list_by_group(group_id, status, limit, cursor)
            .await?;

        // 批量获取申请者信息（避免 N+1，并补全 applicant 字段）
        let mut applicant_ids: Vec<i64> = apps.iter().map(|a| a.applicant_id).collect();
        applicant_ids.sort_unstable();
        applicant_ids.dedup();
        let applicants = self.user_repo.find_profiles(&applicant_ids).await?;
        let applicant_map: std::collections::HashMap<i64, _> =
            applicants.into_iter().map(|u| (u.user_id, u)).collect();

        let items = apps
            .into_iter()
            .map(|a| {
                let user = applicant_map.get(&a.applicant_id);
                GroupApplicationItem {
                    id: a.application_id.to_string(),
                    applicant: UserInfo {
                        user_id: a.applicant_id,
                        nickname: user
                            .map(|u| u.nickname.clone())
                            .unwrap_or_else(|| "Unknown".to_string()),
                        avatar_url: user.and_then(|u| u.avatar_url.clone()),
                        user_type: user
                            .map(|u| u.user_type)
                            .unwrap_or(crate::types::enums::UserType::Normal),
                    },
                    message: a.message,
                    status: a.status.to_string(),
                    created_at: a.created_at.to_rfc3339(),
                    handled_at: a.handled_at.map(|t| t.to_rfc3339()),
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
        req: ApproveApplicationRequest,
    ) -> Result<()> {
        let application_id = Uuid::parse_str(&req.application_id)
            .map_err(|_| AppError::BadRequest("Invalid Application ID".to_string()))?;
        let app = self
            .application_repo
            .find_by_application_id(application_id)
            .await?
            .ok_or(AppError::NotFound("Application not found".to_string()))?;

        // Check perms
        let role = self
            .member_repo
            .get_member_role(app.group_chat_id, user_id)
            .await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.application_repo
            .update_status_by_application_id(application_id, InviteStatus::Accepted, user_id)
            .await?;
        self.member_repo
            .add_member(app.group_chat_id, app.applicant_id, MemberRole::Member)
            .await?;

        // 记录加入历史（通过申请）
        self.history_repo
            .record_join(app.group_chat_id, app.applicant_id, Some(user_id))
            .await?;

        // 1) 通知申请者审批结果（通知类事件）
        let group_name = self
            .group_repo
            .find_by_id(app.group_chat_id)
            .await?
            .map(|g| g.name)
            .unwrap_or_else(|| "Unknown".to_string());
        self.event_bus.publish(AppEvent::GroupApplicationResponse {
            application_id,
            group_id: app.group_chat_id,
            group_name,
            applicant_id: app.applicant_id,
            status: "accepted".to_string(),
        });

        // 2) 群聊订阅者：成员加入
        self.event_bus.publish(AppEvent::GroupMemberAdded {
            group_id: app.group_chat_id,
            user_id: app.applicant_id,
            added_by: user_id,
        });

        Ok(())
    }

    pub async fn reject_application(
        &self,
        user_id: i64,
        req: RejectApplicationRequest,
    ) -> Result<()> {
        if let Some(reason) = req.reason.as_deref() {
            let reason = reason.trim();
            if !reason.is_empty() {
                Validator::validate_length(reason, "拒绝原因", 1, 200)?;
            }
        }

        let application_id = Uuid::parse_str(&req.application_id)
            .map_err(|_| AppError::BadRequest("Invalid Application ID".to_string()))?;
        let app = self
            .application_repo
            .find_by_application_id(application_id)
            .await?
            .ok_or(AppError::NotFound("Application not found".to_string()))?;

        let role = self
            .member_repo
            .get_member_role(app.group_chat_id, user_id)
            .await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.application_repo
            .update_status_by_application_id(application_id, InviteStatus::Rejected, user_id)
            .await?;

        // 通知申请者审批结果（通知类事件）
        let group_name = self
            .group_repo
            .find_by_id(app.group_chat_id)
            .await?
            .map(|g| g.name)
            .unwrap_or_else(|| "Unknown".to_string());
        self.event_bus.publish(AppEvent::GroupApplicationResponse {
            application_id,
            group_id: app.group_chat_id,
            group_name,
            applicant_id: app.applicant_id,
            status: "rejected".to_string(),
        });
        Ok(())
    }

    /// 取消群组申请（申请者取消自己的待处理申请）
    pub async fn cancel_application(
        &self,
        user_id: i64,
        req: CancelApplicationRequest,
    ) -> Result<()> {
        let application_id = Uuid::parse_str(&req.application_id)
            .map_err(|_| AppError::BadRequest("Invalid Application ID".to_string()))?;
        let app = self
            .application_repo
            .find_by_application_id(application_id)
            .await?
            .ok_or(AppError::NotFound("Application not found".to_string()))?;

        // 只能取消自己的申请
        if app.applicant_id != user_id {
            return Err(AppError::Forbidden("只能取消自己的申请".to_string()));
        }

        // 只能取消待处理的申请
        if app.status != InviteStatus::Pending {
            return Err(AppError::BadRequest("只能取消待处理的申请".to_string()));
        }

        self.application_repo
            .delete_by_application_id(application_id)
            .await?;
        Ok(())
    }

    pub async fn transfer_ownership(
        &self,
        user_id: i64,
        req: TransferOwnershipRequest,
    ) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let new_owner = req.new_owner_id;

        let owner_id = self
            .group_repo
            .get_owner(group_id)
            .await?
            .ok_or(AppError::NotFound("Group not found".to_string()))?;
        if owner_id != user_id {
            return Err(AppError::Forbidden("Only owner can transfer".to_string()));
        }

        self.group_repo
            .transfer_ownership(group_id, new_owner)
            .await?;
        self.member_repo
            .update_role(group_id, user_id, MemberRole::Admin)
            .await?;
        self.member_repo
            .update_role(group_id, new_owner, MemberRole::Owner)
            .await?;

        // 记录角色变更历史
        self.history_repo
            .record_role_changed(
                group_id,
                user_id,
                user_id,
                MemberRole::Owner,
                MemberRole::Admin,
            )
            .await?;
        self.history_repo
            .record_role_changed(
                group_id,
                new_owner,
                user_id,
                MemberRole::Member,
                MemberRole::Owner,
            )
            .await?;

        Ok(())
    }

    pub async fn get_invites(
        &self,
        user_id: i64,
        req: MyInvitesRequest,
    ) -> Result<PaginatedData<GroupInviteItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = req.cursor;

        let status = req
            .status
            .as_ref()
            .map(|s| {
                InviteStatus::from_str(s)
                    .map_err(|_| AppError::BadRequest("无效的 status".to_string()))
            })
            .transpose()?;
        let (invites, next_cursor) = self
            .invite_repo
            .list_by_user(user_id, status, limit, cursor)
            .await?;

        if invites.is_empty() {
            return Ok(PaginatedData {
                items: Vec::new(),
                pagination: PaginationMeta {
                    has_more: next_cursor.is_some(),
                    next_cursor,
                },
            });
        }

        // 批量获取群组信息 + 邀请者信息（避免 N+1，并补全 group/inviter 字段）
        let mut group_ids: Vec<Uuid> = invites.iter().map(|i| i.group_chat_id).collect();
        group_ids.sort_unstable();
        group_ids.dedup();

        let mut inviter_ids: Vec<i64> = invites.iter().map(|i| i.inviter_id).collect();
        inviter_ids.sort_unstable();
        inviter_ids.dedup();

        let (groups, inviters) = tokio::try_join!(
            self.group_repo
                .find_by_ids_with_distance(&group_ids, 0.0, 0.0),
            self.user_repo.find_profiles(&inviter_ids)
        )?;

        let group_map: std::collections::HashMap<Uuid, _> =
            groups.into_iter().map(|g| (g.group_id, g)).collect();
        let inviter_map: std::collections::HashMap<i64, _> =
            inviters.into_iter().map(|u| (u.user_id, u)).collect();

        let items = invites
            .into_iter()
            .map(|i| {
                let g = group_map.get(&i.group_chat_id);
                let u = inviter_map.get(&i.inviter_id);

                GroupInviteItem {
                    id: i.invite_id.to_string(),
                    group: GroupInfo {
                        id: i.group_chat_id.to_string(),
                        name: g
                            .map(|x| x.name.clone())
                            .unwrap_or_else(|| "Unknown".to_string()),
                        avatar_url: g.and_then(|x| x.avatar_url.clone()),
                        members_count: g.map(|x| x.member_count).unwrap_or(0),
                    },
                    inviter: UserInfo {
                        user_id: i.inviter_id,
                        nickname: u
                            .map(|x| x.nickname.clone())
                            .unwrap_or_else(|| "Unknown".to_string()),
                        avatar_url: u.and_then(|x| x.avatar_url.clone()),
                        user_type: u
                            .map(|x| x.user_type)
                            .unwrap_or(crate::types::enums::UserType::Normal),
                    },
                    message: i.message,
                    status: i.status.to_string(),
                    created_at: i.created_at.to_rfc3339(),
                    expires_at: Some(i.expires_at.to_rfc3339()),
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
            .map_err(|_| AppError::BadRequest("Invalid Invite ID".to_string()))?;
        let invite = self
            .invite_repo
            .find_by_invite_id(invite_id)
            .await?
            .ok_or(AppError::NotFound("Invite not found".to_string()))?;

        if invite.invitee_id != user_id {
            return Err(AppError::Forbidden("Not your invite".to_string()));
        }

        self.invite_repo
            .update_status_by_invite_id(invite_id, InviteStatus::Accepted)
            .await?;
        self.member_repo
            .add_member(invite.group_chat_id, user_id, MemberRole::Member)
            .await?;

        // 记录加入历史（通过邀请）
        self.history_repo
            .record_join(invite.group_chat_id, user_id, Some(invite.inviter_id))
            .await?;

        // 通知邀请者：邀请已接受（通知类事件）
        self.event_bus.publish(AppEvent::GroupInviteResponse {
            invite_id,
            group_id: invite.group_chat_id,
            inviter_id: invite.inviter_id,
            invitee_id: user_id,
            status: "accepted".to_string(),
        });

        // 群聊订阅者：成员加入
        self.event_bus.publish(AppEvent::GroupMemberAdded {
            group_id: invite.group_chat_id,
            user_id,
            added_by: invite.inviter_id,
        });

        Ok(())
    }

    pub async fn reject_invite(&self, user_id: i64, req: RejectInviteRequest) -> Result<()> {
        let invite_id = Uuid::parse_str(&req.invite_id)
            .map_err(|_| AppError::BadRequest("Invalid Invite ID".to_string()))?;
        let invite = self
            .invite_repo
            .find_by_invite_id(invite_id)
            .await?
            .ok_or(AppError::NotFound("Invite not found".to_string()))?;

        if invite.invitee_id != user_id {
            return Err(AppError::Forbidden("Not your invite".to_string()));
        }

        self.invite_repo
            .update_status_by_invite_id(invite_id, InviteStatus::Rejected)
            .await?;

        // 通知邀请者：邀请被拒绝（通知类事件）
        self.event_bus.publish(AppEvent::GroupInviteResponse {
            invite_id,
            group_id: invite.group_chat_id,
            inviter_id: invite.inviter_id,
            invitee_id: user_id,
            status: "rejected".to_string(),
        });
        Ok(())
    }
}
