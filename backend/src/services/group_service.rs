use redis::aio::ConnectionManager;
use std::str::FromStr;
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    constants::{DEFAULT_LATITUDE, DEFAULT_LOCATION_NAME, DEFAULT_LONGITUDE},
    dto::{
        request::{
            AcceptInviteRequest, ApplyToGroupRequest, ApproveGroupApplicationRequest,
            BanMemberRequest, BannedMembersRequest, CancelApplicationRequest,
            CreateGroupAnnouncementRequest, CreateGroupChatRequest, DeleteGroupAnnouncementRequest,
            GroupAnnouncementsRequest, GroupApplicationsRequest, GroupMembersRequest,
            GroupOpenTheDoorRequest, GroupSettingsRequest, InviteMemberRequest, JoinGroupRequest,
            LeaveGroupRequest, MuteMemberRequest, MyGroupChatsRequest, MyInvitesRequest,
            RejectGroupApplicationRequest, RejectInviteRequest, RemoveMemberRequest,
            SearchGroupsByKeywordRequest, SearchGroupsByLocationRequest, SearchGroupsByTagRequest,
            SetMemberRoleRequest, TransferOwnershipRequest, UnbanMemberRequest,
            UnmuteMemberRequest, UpdateGroupAnnouncementRequest, UpdateGroupChatRequest,
        },
        response::{
            ApplyToGroupData, BannedMemberItem, CreateGroupChatData, GroupAnnouncementItem,
            GroupApplicationItem, GroupDetailData, GroupInfo, GroupInviteItem, GroupItem,
            GroupMemberItem, GroupSettings, GroupSettingsData, InviteMemberData, JoinedGroupItem,
            PaginatedData, PaginationMeta, RelativeLocation, UserInfo,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::{
        chat_unread_repo::ChatUnreadRepository,
        group_announcement_repo::GroupAnnouncementRepository,
        group_application_repo::GroupApplicationRepository, group_bans_repo::GroupBansRepository,
        group_invite_repo::GroupInviteRepository,
        group_member_history_repo::GroupMemberHistoryRepository,
        group_member_repo::GroupMemberRepository, group_repo::GroupRepository,
        group_settings_repo::GroupSettingsRepository, tag_repo::TagRepository,
        user_repo::UserRepository,
    },
    types::enums::{GroupMemberRemovalReason, InviteStatus, MemberRole},
    utils::{password::PasswordUtil, validation::Validator},
    EventBus,
};

#[derive(Clone)]
pub struct GroupService {
    event_bus: Arc<EventBus>,
    group_repo: GroupRepository,
    member_repo: GroupMemberRepository,
    history_repo: GroupMemberHistoryRepository,
    application_repo: GroupApplicationRepository,
    bans_repo: GroupBansRepository,
    invite_repo: GroupInviteRepository,
    settings_repo: GroupSettingsRepository,
    announcement_repo: GroupAnnouncementRepository,
    tag_repo: TagRepository,
    user_repo: UserRepository,
    unread_repo: ChatUnreadRepository,
    _redis: ConnectionManager,
    presence_service: crate::services::presence_service::PresenceService,
    media_service: crate::services::media_service::MediaService,
}

use crate::dto::request::{AdminDismissGroupRequest, AdminGroupSearchRequest};
use crate::dto::response::AdminGroupItem;

impl GroupService {
    pub fn new(
        event_bus: Arc<EventBus>,
        group_repo: GroupRepository,
        member_repo: GroupMemberRepository,
        history_repo: GroupMemberHistoryRepository,
        application_repo: GroupApplicationRepository,
        bans_repo: GroupBansRepository,
        invite_repo: GroupInviteRepository,
        settings_repo: GroupSettingsRepository,
        announcement_repo: GroupAnnouncementRepository,
        tag_repo: TagRepository,
        user_repo: UserRepository,
        unread_repo: ChatUnreadRepository,
        redis: ConnectionManager,
        presence_service: crate::services::presence_service::PresenceService,
        media_service: crate::services::media_service::MediaService,
    ) -> Self {
        Self {
            event_bus,
            group_repo,
            member_repo,
            history_repo,
            application_repo,
            bans_repo,
            invite_repo,
            settings_repo,
            announcement_repo,
            tag_repo,
            user_repo,
            unread_repo,
            _redis: redis,
            presence_service,
            media_service,
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
            .ok_or(AppError::UserNotFound("User not found".to_string()))?;

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
                if let Ok(file_data) = self
                    .media_service
                    .validate_file(file_id, owner_id, "group_avatar")
                    .await
                {
                    // 存储 storage_path 到 DB
                    avatar_url = Some(file_data);
                    // 标记文件已使用
                    let _ = self.media_service.mark_as_used(file_id).await;
                }
            }
        }

        let group = self
            .group_repo
            .create(
                req.name.clone(),
                req.description.clone().unwrap_or_default(), // Convert Option<String> to String
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
                self.tag_repo.set_group_tags(group.group_id, tags).await?;
            }
        }

        Ok(CreateGroupChatData {
            group_chat_id: group.group_id.to_string(),
        })
    }

    /// 按关键字搜索群组（全局）
    pub async fn search_by_keyword(
        &self,
        req: SearchGroupsByKeywordRequest,
    ) -> Result<PaginatedData<GroupItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = req.cursor;

        Validator::validate_coordinates(
            req.reference_point.latitude,
            req.reference_point.longitude,
        )?;

        let (results, next_cursor) = self
            .group_repo
            .search_by_name(
                &req.kw,
                req.reference_point.latitude,
                req.reference_point.longitude,
                limit,
                cursor,
            )
            .await?;

        // 批量获取标签
        let group_ids: Vec<Uuid> = results.iter().map(|g| g.group_id).collect();
        let tags_map = self.tag_repo.get_group_tags_batch(&group_ids).await?;

        let mut items = Vec::with_capacity(results.len());
        for g in results {
            let tags = tags_map.get(&g.group_id).cloned();
            let tags = if tags.as_ref().map(|t| t.is_empty()).unwrap_or(true) {
                None
            } else {
                tags
            };

            items.push(GroupItem {
                group_chat_id: g.group_id.to_string(),
                name: g.name,
                description: if g.description.is_empty() {
                    None
                } else {
                    Some(g.description)
                },
                avatar_url: self
                    .media_service
                    .sign_url_if_path(g.avatar_url.as_deref())
                    .await,
                tags,
                members_count: g.member_count,
                has_password: g.has_password,
                location: Some(RelativeLocation {
                    latitude: g.latitude,
                    longitude: g.longitude,
                    location_name: g.location_name,
                    distance: g.distance,
                    bearing_degrees: g.bearing_degrees,
                }),
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

    /// 按标签搜索群组（全局）
    pub async fn search_by_tag(
        &self,
        req: SearchGroupsByTagRequest,
    ) -> Result<PaginatedData<GroupItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = req.cursor;

        let lat = req.reference_point.latitude;
        let lng = req.reference_point.longitude;
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

        let mut items = Vec::with_capacity(results.len());
        for g in results {
            let tags = tags_map.get(&g.group_id).cloned();
            let tags = if tags.as_ref().map(|t| t.is_empty()).unwrap_or(true) {
                None
            } else {
                tags
            };

            items.push(GroupItem {
                group_chat_id: g.group_id.to_string(),
                name: g.name,
                description: if g.description.is_empty() {
                    None
                } else {
                    Some(g.description)
                },
                avatar_url: self
                    .media_service
                    .sign_url_if_path(g.avatar_url.as_deref())
                    .await,
                tags,
                members_count: g.member_count,
                has_password: g.has_password,
                location: Some(RelativeLocation {
                    latitude: g.latitude,
                    longitude: g.longitude,
                    location_name: g.location_name,
                    distance: g.distance,
                    bearing_degrees: g.bearing_degrees,
                }),
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

    /// 按位置搜索群组（范围过滤）
    pub async fn search_by_location(
        &self,
        req: SearchGroupsByLocationRequest,
    ) -> Result<PaginatedData<GroupItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = req.cursor;

        let lat = req.reference_point.latitude;
        let lng = req.reference_point.longitude;
        Validator::validate_coordinates(lat, lng)?;
        Validator::validate_radius_required(req.radius, 50000.0)?;

        let (results, next_cursor) = self
            .group_repo
            .search_by_location(req.kw.as_deref(), lat, lng, req.radius, limit, cursor)
            .await?;

        // 批量获取标签
        let group_ids: Vec<Uuid> = results.iter().map(|g| g.group_id).collect();
        let tags_map = self.tag_repo.get_group_tags_batch(&group_ids).await?;

        let mut items = Vec::with_capacity(results.len());
        for g in results {
            let tags = tags_map.get(&g.group_id).cloned();
            let tags = if tags.as_ref().map(|t| t.is_empty()).unwrap_or(true) {
                None
            } else {
                tags
            };

            items.push(GroupItem {
                group_chat_id: g.group_id.to_string(),
                name: g.name,
                description: if g.description.is_empty() {
                    None
                } else {
                    Some(g.description)
                },
                avatar_url: self
                    .media_service
                    .sign_url_if_path(g.avatar_url.as_deref())
                    .await,
                tags,
                members_count: g.member_count,
                has_password: g.has_password,
                location: Some(RelativeLocation {
                    latitude: g.latitude,
                    longitude: g.longitude,
                    location_name: g.location_name,
                    distance: g.distance,
                    bearing_degrees: g.bearing_degrees,
                }),
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

    /// 获取群组详情（通过 ID 获取指定群组的详情）
    pub async fn open_the_door(
        &self,
        req: &GroupOpenTheDoorRequest,
        user_id: Option<i64>,
    ) -> Result<GroupDetailData> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        Validator::validate_coordinates(
            req.reference_point.latitude,
            req.reference_point.longitude,
        )?;

        let groups = self
            .group_repo
            .find_by_ids_with_distance(
                &[group_id],
                req.reference_point.latitude,
                req.reference_point.longitude,
            )
            .await?;

        let group = groups
            .first()
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;

        // 获取标签
        let tags = self.tag_repo.get_group_tags(group_id).await?;
        let tags = if tags.is_empty() { None } else { Some(tags) };

        // 获取用户相关信息 (Zero-Lookup)
        let (my_role, is_joined, unread_count, is_muted, is_pinned) = if let Some(uid) = user_id {
            let member = self.member_repo.find_member(group_id, uid).await?;
            if let Some(m) = member {
                let unread = self
                    .unread_repo
                    .get_unread_count("group_chat", &group_id.to_string(), uid)
                    .await?;
                // TODO: Fetch is_muted/is_pinned from DB once columns are added
                (
                    Some(m.role.to_string()),
                    Some(true),
                    Some(unread as i32),
                    Some(false),
                    Some(false),
                )
            } else {
                (None, Some(false), None, None, None)
            }
        } else {
            (None, None, None, None, None)
        };

        let group_item = GroupItem {
            group_chat_id: group.group_id.to_string(),
            name: group.name.clone(),
            description: if group.description.is_empty() {
                None
            } else {
                Some(group.description.clone())
            },
            avatar_url: self
                .media_service
                .sign_url_if_path(group.avatar_url.as_deref())
                .await,
            tags,
            members_count: group.member_count,
            has_password: group.has_password,
            location: Some(RelativeLocation {
                latitude: group.latitude,
                longitude: group.longitude,
                location_name: group.location_name.clone(),
                distance: group.distance,
                bearing_degrees: group.bearing_degrees,
            }),
        };

        Ok(GroupDetailData {
            group: group_item,
            my_role,
            is_joined,
            unread_count,
            is_muted,
            is_pinned,
        })
    }

    /// 获取群组设置
    pub async fn get_settings(&self, req: GroupSettingsRequest) -> Result<GroupSettingsData> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        let group = self
            .group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;

        let settings = self.settings_repo.find_by_group(group_id).await?;
        let has_password = settings
            .as_ref()
            .map(|s| s.password_hash.is_some())
            .unwrap_or(false);

        Ok(GroupSettingsData {
            group_chat_id: group.group_id.to_string(),
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
        let group_id_strs: Vec<String> = group_ids.iter().map(|id| id.to_string()).collect();
        let (groups, unread_counts) = tokio::try_join!(
            self.group_repo
                .find_by_ids_with_distance(&group_ids, 0.0, 0.0),
            self.unread_repo
                .get_unread_count_batch("group_chat", &group_id_strs, user_id)
        )?;

        // 4. 构建映射
        let group_map: std::collections::HashMap<Uuid, _> =
            groups.into_iter().map(|g| (g.group_id, g)).collect();

        // 5. 组装结果
        let mut items = Vec::with_capacity(memberships.len());
        for m in &memberships {
            if let Some(g) = group_map.get(&m.group_chat_id) {
                let unread_count = unread_counts
                    .get(&g.group_id.to_string())
                    .copied()
                    .unwrap_or(0);

                items.push(JoinedGroupItem {
                    group_info: GroupInfo {
                        group_chat_id: g.group_id.to_string(),
                        name: g.name.clone(),
                        avatar_url: self
                            .media_service
                            .sign_url_if_path(g.avatar_url.as_deref())
                            .await,
                        members_count: g.member_count,
                        location: Some(crate::dto::response::Location {
                            latitude: g.latitude,
                            longitude: g.longitude,
                            location_name: g.location_name.clone(),
                        }),
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
        let group_id = Uuid::parse_str(&req.group_chat_id)
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
                if let Ok(file_data) = self
                    .media_service
                    .validate_file(file_id, user_id, "group_avatar")
                    .await
                {
                    // 存储 storage_path 到 DB
                    avatar_url = Some(file_data);
                    // 标记文件已使用
                    let _ = self.media_service.mark_as_used(file_id).await;
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
            self.tag_repo.set_group_tags(group_id, tags).await?;
        }

        // 发布群组资料更新事件（群聊订阅者可实时刷新 UI）
        if updated_name.is_some() || updated_description.is_some() || has_avatar_update {
            self.event_bus.publish(AppEvent::GroupProfileUpdated {
                group_id,
                name: updated_name,
                description: updated_description,
                avatar_url: self
                    .media_service
                    .sign_url_if_path(avatar_url_for_event.as_deref())
                    .await,
            });
        }

        Ok(())
    }

    pub async fn delete_group(
        &self,
        user_id: i64,
        req: crate::dto::request::DeleteGroupChatRequest,
    ) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        // 验证所有权
        let owner_id = self
            .group_repo
            .get_owner(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("Group not found".to_string()))?;

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
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor = req.cursor;

        let role = req.role.as_deref();
        let (members, next_cursor) = self
            .member_repo
            .list_by_group(group_id, role, limit as i32, cursor)
            .await?;

        // 批量获取在线状态
        let user_ids: Vec<i64> = members.iter().map(|m| m.user_id).collect();
        let online_status_map = self
            .presence_service
            .batch_get_online_status_with_privacy(&user_ids)
            .await;

        let mut items = Vec::with_capacity(members.len());
        for m in members {
            let is_online = online_status_map
                .get(&m.user_id)
                .and_then(|(online, _)| *online);

            items.push(GroupMemberItem {
                user: UserInfo {
                    user_id: m.user_id,
                    nickname: m.nickname.unwrap_or_default(),
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(m.avatar_url.as_deref())
                        .await,
                    user_type: m.user_type.unwrap_or(crate::types::enums::UserType::Normal),
                },
                role: m.role.to_string(),
                joined_at: m.joined_at.to_rfc3339(),
                muted_until: m.muted_until.map(|t| t.to_rfc3339()),
                is_online,
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

    pub async fn invite_member(
        &self,
        user_id: i64,
        req: InviteMemberRequest,
    ) -> Result<InviteMemberData> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
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
        let group_info = self
            .group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("Group not found".to_string()))?;

        self.event_bus.publish(AppEvent::GroupInviteSent {
            invite_id: invite.invite_id,
            group_id,
            group_name: group_info.name,
            group_avatar_url: self
                .media_service
                .sign_url_if_path(group_info.avatar_url.as_deref())
                .await,
            from_user_id: user_id,
            to_user_id: target_uid,
            message: invite.message.clone(),
        });

        Ok(InviteMemberData {
            invite_id: invite.invite_id.to_string(),
            expires_at: invite.expires_at.to_rfc3339(),
        })
    }

    pub async fn join_group(&self, user_id: i64, req: JoinGroupRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        // 检查群组是否存在
        if !self.group_repo.exists(group_id).await? {
            return Err(AppError::GroupNotFound("Group not found".to_string()));
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

        // 获取最新成员数
        let member_count = self.member_repo.count_members(group_id).await?;

        // 发布成员加入事件（群聊订阅者可实时看到成员变化）
        self.event_bus.publish(AppEvent::GroupMemberAdded {
            group_id,
            user_id,
            added_by: user_id,
            role: MemberRole::Member,
            member_count,
        });

        Ok(())
    }

    pub async fn apply_to_group(
        &self,
        user_id: i64,
        req: ApplyToGroupRequest,
    ) -> Result<ApplyToGroupData> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
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
            application_id: app.application_id.to_string(),
        })
    }

    pub async fn set_role(&self, user_id: i64, req: SetMemberRoleRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
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
            old_role,
            new_role,
        });

        Ok(())
    }

    pub async fn mute_member(&self, user_id: i64, req: MuteMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
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
        let group_id = Uuid::parse_str(&req.group_chat_id)
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
        let group_id = Uuid::parse_str(&req.group_chat_id)
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

        // 获取最新成员数
        let member_count = self.member_repo.count_members(group_id).await?;

        // 发布成员离开事件（被封禁）
        self.event_bus.publish(AppEvent::GroupMemberRemoved {
            group_id,
            user_id: target_uid,
            removed_by: user_id,
            member_count,
            reason: GroupMemberRemovalReason::Banned,
        });

        Ok(())
    }

    /// 解除封禁成员
    pub async fn unban_member(&self, user_id: i64, req: UnbanMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
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
        let group_id = Uuid::parse_str(&req.group_chat_id)
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

        let mut items = Vec::with_capacity(bans.len());
        for b in bans {
            items.push(BannedMemberItem {
                user: UserInfo {
                    user_id: b.user_id,
                    nickname: b.nickname.unwrap_or_default(),
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(b.avatar_url.as_deref())
                        .await,
                    user_type: b.user_type.unwrap_or(crate::types::enums::UserType::Normal),
                },
                reason: b.reason,
                banned_at: b.banned_at.to_rfc3339(),
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

    pub async fn remove_member(&self, user_id: i64, req: RemoveMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
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

        // 获取最新成员数
        let member_count = self.member_repo.count_members(group_id).await?;

        // 发布成员离开事件（被移除）
        self.event_bus.publish(AppEvent::GroupMemberRemoved {
            group_id,
            user_id: target_uid,
            removed_by: user_id,
            member_count,
            reason: GroupMemberRemovalReason::Kicked,
        });

        Ok(())
    }

    pub async fn leave_group(&self, user_id: i64, req: LeaveGroupRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        let owner_id = self
            .group_repo
            .get_owner(group_id)
            .await?
            .ok_or(AppError::GroupNotFound("Group not found".to_string()))?;

        if owner_id == user_id {
            return Err(AppError::BadRequest("Owner cannot leave".to_string()));
        }

        self.member_repo.leave_group(group_id, user_id).await?;

        // 记录离开历史
        self.history_repo.record_leave(group_id, user_id).await?;

        // 获取最新成员数
        let member_count = self.member_repo.count_members(group_id).await?;

        // 发布成员离开事件（主动退出）
        self.event_bus.publish(AppEvent::GroupMemberRemoved {
            group_id,
            user_id,
            removed_by: user_id,
            member_count,
            reason: GroupMemberRemovalReason::Left,
        });

        Ok(())
    }

    pub async fn get_applications(
        &self,
        _user_id: i64,
        req: GroupApplicationsRequest,
    ) -> Result<PaginatedData<GroupApplicationItem>> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
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

        let mut items = Vec::with_capacity(apps.len());
        for a in apps {
            let user = applicant_map.get(&a.applicant_id);
            items.push(GroupApplicationItem {
                application_id: a.application_id.to_string(),
                applicant: UserInfo {
                    user_id: a.applicant_id,
                    nickname: user
                        .map(|u| u.nickname.clone())
                        .unwrap_or_else(|| "Unknown".to_string()),
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(user.and_then(|u| u.avatar_url.as_deref()))
                        .await,
                    user_type: user
                        .map(|u| u.user_type)
                        .unwrap_or(crate::types::enums::UserType::Normal),
                },
                message: a.message,
                status: a.status.to_string(),
                created_at: a.created_at.to_rfc3339(),
                handled_at: a.handled_at.map(|t| t.to_rfc3339()),
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

    pub async fn approve_application(
        &self,
        user_id: i64,
        req: ApproveGroupApplicationRequest,
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
            status: InviteStatus::Accepted,
        });

        // 2) 群聊订阅者：成员加入
        let member_count = self.member_repo.count_members(app.group_chat_id).await?;
        self.event_bus.publish(AppEvent::GroupMemberAdded {
            group_id: app.group_chat_id,
            user_id: app.applicant_id,
            added_by: user_id,
            role: MemberRole::Member,
            member_count,
        });

        Ok(())
    }

    pub async fn reject_application(
        &self,
        user_id: i64,
        req: RejectGroupApplicationRequest,
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
            status: InviteStatus::Rejected,
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
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let new_owner = req.new_owner_id;

        let owner_id = self
            .group_repo
            .get_owner(group_id)
            .await?
            .ok_or(AppError::GroupNotFound("Group not found".to_string()))?;
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

        let mut items = Vec::with_capacity(invites.len());
        for i in invites {
            let g = group_map.get(&i.group_chat_id);
            let u = inviter_map.get(&i.inviter_id);

            items.push(GroupInviteItem {
                invite_id: i.invite_id.to_string(),
                group: GroupInfo {
                    group_chat_id: i.group_chat_id.to_string(),
                    name: g
                        .map(|x| x.name.clone())
                        .unwrap_or_else(|| "Unknown".to_string()),
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(g.and_then(|x| x.avatar_url.as_deref()))
                        .await,
                    members_count: g.map(|x| x.member_count).unwrap_or(0),
                    location: g.map(|x| crate::dto::response::Location {
                        latitude: x.latitude,
                        longitude: x.longitude,
                        location_name: x.location_name.clone(),
                    }),
                },
                inviter: UserInfo {
                    user_id: i.inviter_id,
                    nickname: u
                        .map(|x| x.nickname.clone())
                        .unwrap_or_else(|| "Unknown".to_string()),
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(u.and_then(|x| x.avatar_url.as_deref()))
                        .await,
                    user_type: u
                        .map(|x| x.user_type)
                        .unwrap_or(crate::types::enums::UserType::Normal),
                },
                message: i.message,
                status: i.status.to_string(),
                created_at: i.created_at.to_rfc3339(),
                expires_at: Some(i.expires_at.to_rfc3339()),
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
            status: InviteStatus::Accepted,
        });

        // 群聊订阅者：成员加入
        let member_count = self.member_repo.count_members(invite.group_chat_id).await?;
        self.event_bus.publish(AppEvent::GroupMemberAdded {
            group_id: invite.group_chat_id,
            user_id,
            added_by: invite.inviter_id,
            role: MemberRole::Member,
            member_count,
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
            status: InviteStatus::Rejected,
        });
        Ok(())
    }

    // ==================== 管理员群组管理 (Admin) ====================

    pub async fn admin_search_groups(
        &self,
        req: &AdminGroupSearchRequest,
    ) -> Result<PaginatedData<AdminGroupItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let cursor = req.cursor.clone();

        let group_id = if let Some(gid_str) = &req.group_chat_id {
            Some(
                Uuid::parse_str(gid_str)
                    .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?,
            )
        } else {
            None
        };

        let (groups, next_cursor) = self
            .group_repo
            .admin_search(group_id, req.name.as_deref(), req.owner_id, limit, cursor)
            .await?;

        let mut items = Vec::with_capacity(groups.len());
        for g in groups {
            items.push(AdminGroupItem {
                group_chat_id: g.group_id.to_string(),
                name: g.name,
                avatar_url: self
                    .media_service
                    .sign_url_if_path(g.avatar_url.as_deref())
                    .await,
                members_count: g.member_count as i32,
                owner: UserInfo {
                    user_id: g.owner_id,
                    nickname: g.owner_nickname,
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(g.owner_avatar_url.as_deref())
                        .await,
                    user_type: g.owner_user_type,
                },
                created_at: g.created_at.to_rfc3339(),
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

    pub async fn admin_dismiss_group(
        &self,
        admin_id: i64,
        req: &AdminDismissGroupRequest,
    ) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        // 检查群组是否存在
        if !self.group_repo.exists(group_id).await? {
            return Err(AppError::GroupNotFound("Group not found".to_string()));
        }

        self.group_repo
            .dismiss(group_id, req.reason.as_deref())
            .await?;

        // 发布群组解散事件
        self.event_bus.publish(AppEvent::GroupDismissed {
            group_id,
            dismissed_by: admin_id,
        });

        Ok(())
    }

    // ==================== 群公告管理 ====================

    /// 发布群公告
    pub async fn create_announcement(
        &self,
        user_id: i64,
        req: &CreateGroupAnnouncementRequest,
    ) -> Result<GroupAnnouncementItem> {
        let group_id = Uuid::from_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        // 验证用户是否是群主或管理员
        let role = self
            .member_repo
            .get_member_role(group_id, user_id)
            .await?
            .ok_or_else(|| AppError::NotGroupMember("您不是群组成员".to_string()))?;

        if !matches!(role, MemberRole::Owner | MemberRole::Admin) {
            return Err(AppError::Forbidden(
                "只有群主和管理员可以发布公告".to_string(),
            ));
        }

        // 验证标题和内容长度
        Validator::validate_not_empty(&req.title, "标题")?;
        Validator::validate_length(&req.title, "标题", 1, 100)?;
        Validator::validate_not_empty(&req.content, "内容")?;
        Validator::validate_length(&req.content, "内容", 1, 2000)?;

        // 创建公告
        let announcement = self
            .announcement_repo
            .create(
                group_id,
                req.title.clone(),
                req.content.clone(),
                user_id,
                req.is_pinned.unwrap_or(false),
            )
            .await
            .map_err(|e| AppError::InternalError(e.to_string()))?;

        // 获取创建者信息
        let creator = self
            .user_repo
            .find_profiles(&[user_id])
            .await?
            .into_iter()
            .next()
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

        let announcement_item = GroupAnnouncementItem {
            announcement_id: announcement.announcement_id.to_string(),
            group_chat_id: announcement.group_chat_id.to_string(),
            title: announcement.title.clone(),
            content: announcement.content.clone(),
            created_by: UserInfo {
                user_id: creator.user_id,
                nickname: creator.nickname,
                avatar_url: self
                    .media_service
                    .sign_url_if_path(creator.avatar_url.as_deref())
                    .await,
                user_type: creator.user_type,
            },
            created_at: announcement.created_at.to_rfc3339(),
            is_pinned: announcement.is_pinned,
        };

        // 发布群公告创建事件
        let announcement_uuid = Uuid::from_str(&announcement.announcement_id)
            .map_err(|_| AppError::InternalError("无效的公告ID".to_string()))?;
        self.event_bus.publish(AppEvent::GroupAnnouncementCreated {
            announcement_id: announcement_uuid,
            group_id,
            title: announcement.title,
            content: announcement.content,
            created_by_user_id: user_id,
            is_pinned: announcement.is_pinned,
        });

        Ok(announcement_item)
    }

    /// 获取群公告列表
    pub async fn list_announcements(
        &self,
        req: &GroupAnnouncementsRequest,
    ) -> Result<PaginatedData<GroupAnnouncementItem>> {
        let group_id = Uuid::from_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        // 验证群组是否存在
        let _group = self
            .group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;

        let limit = req.limit.unwrap_or(20).min(100) as i64;
        let announcements = self
            .announcement_repo
            .list_by_group(group_id, limit, req.cursor.clone())
            .await
            .map_err(|e| AppError::InternalError(e.to_string()))?;

        // 获取所有创建者信息
        let creator_ids: Vec<i64> = announcements.iter().map(|a| a.created_by).collect();
        let creators = self.user_repo.find_profiles(&creator_ids).await?;
        let creators_map: std::collections::HashMap<i64, _> =
            creators.into_iter().map(|c| (c.user_id, c)).collect();

        // 简化实现：暂不支持真正的cursor分页
        let has_more = announcements.len() >= limit as usize;

        let mut items = Vec::with_capacity(announcements.len());
        for announcement in announcements {
            if let Some(creator) = creators_map.get(&announcement.created_by) {
                items.push(GroupAnnouncementItem {
                    announcement_id: announcement.announcement_id.to_string(),
                    group_chat_id: announcement.group_chat_id.to_string(),
                    title: announcement.title,
                    content: announcement.content,
                    created_by: UserInfo {
                        user_id: creator.user_id,
                        nickname: creator.nickname.clone(),
                        avatar_url: self
                            .media_service
                            .sign_url_if_path(creator.avatar_url.as_deref())
                            .await,
                        user_type: creator.user_type,
                    },
                    created_at: announcement.created_at.to_rfc3339(),
                    is_pinned: announcement.is_pinned,
                });
            }
        }

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor: None, // 暂不支持cursor
            },
        })
    }

    /// 更新群公告
    pub async fn update_announcement(
        &self,
        user_id: i64,
        req: &UpdateGroupAnnouncementRequest,
    ) -> Result<()> {
        let announcement_id = Uuid::from_str(&req.announcement_id)
            .map_err(|_| AppError::BadRequest("无效的公告ID".to_string()))?;

        // 获取公告信息
        let announcement = self
            .announcement_repo
            .find_by_id(announcement_id)
            .await
            .map_err(|e| AppError::InternalError(e.to_string()))?
            .ok_or_else(|| AppError::NotFound("公告不存在".to_string()))?;

        // 验证用户是否是群主或管理员
        let group_chat_uuid = Uuid::from_str(&announcement.group_chat_id)
            .map_err(|_| AppError::InternalError("无效的群组ID".to_string()))?;
        let role = self
            .member_repo
            .get_member_role(group_chat_uuid, user_id)
            .await?
            .ok_or_else(|| AppError::NotGroupMember("您不是群组成员".to_string()))?;

        if !matches!(role, MemberRole::Owner | MemberRole::Admin) {
            return Err(AppError::Forbidden(
                "只有群主和管理员可以编辑公告".to_string(),
            ));
        }

        // 验证标题和内容长度（如果提供）
        if let Some(title) = &req.title {
            Validator::validate_not_empty(title, "标题")?;
            Validator::validate_length(title, "标题", 1, 100)?;
        }
        if let Some(content) = &req.content {
            Validator::validate_not_empty(content, "内容")?;
            Validator::validate_length(content, "内容", 1, 2000)?;
        }

        // 更新公告
        self.announcement_repo
            .update(
                announcement_id,
                req.title.clone(),
                req.content.clone(),
                req.is_pinned,
            )
            .await
            .map_err(|e| AppError::InternalError(e.to_string()))?;

        // 发布群公告更新事件
        let group_chat_uuid = Uuid::from_str(&announcement.group_chat_id)
            .map_err(|_| AppError::InternalError("无效的群组ID".to_string()))?;
        self.event_bus.publish(AppEvent::GroupAnnouncementUpdated {
            announcement_id,
            group_id: group_chat_uuid,
            title: req.title.clone(),
            content: req.content.clone(),
            is_pinned: req.is_pinned,
        });

        Ok(())
    }

    /// 删除群公告
    pub async fn delete_announcement(
        &self,
        user_id: i64,
        req: &DeleteGroupAnnouncementRequest,
    ) -> Result<()> {
        let announcement_id = Uuid::from_str(&req.announcement_id)
            .map_err(|_| AppError::BadRequest("无效的公告ID".to_string()))?;

        // 获取公告信息
        let announcement = self
            .announcement_repo
            .find_by_id(announcement_id)
            .await
            .map_err(|e| AppError::InternalError(e.to_string()))?
            .ok_or_else(|| AppError::NotFound("公告不存在".to_string()))?;

        // 验证用户是否是群主或管理员
        let group_chat_uuid = Uuid::from_str(&announcement.group_chat_id)
            .map_err(|_| AppError::InternalError("无效的群组ID".to_string()))?;
        let role = self
            .member_repo
            .get_member_role(group_chat_uuid, user_id)
            .await?
            .ok_or_else(|| AppError::NotGroupMember("您不是群组成员".to_string()))?;

        if !matches!(role, MemberRole::Owner | MemberRole::Admin) {
            return Err(AppError::Forbidden(
                "只有群主和管理员可以删除公告".to_string(),
            ));
        }

        // 删除公告
        self.announcement_repo
            .delete(announcement_id)
            .await
            .map_err(|e| AppError::InternalError(e.to_string()))?;

        // 发布群公告删除事件
        self.event_bus.publish(AppEvent::GroupAnnouncementDeleted {
            announcement_id,
            group_id: group_chat_uuid,
        });

        Ok(())
    }
}
