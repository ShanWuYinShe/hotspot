use crate::dto::response::RelativeLocation;
use redis::aio::ConnectionManager;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use uuid::Uuid;

use crate::events::event_types::AppEvent;
use crate::types::enums::ResponseStatus;

use crate::{
    constants::{DEFAULT_LATITUDE, DEFAULT_LOCATION_NAME, DEFAULT_LONGITUDE, DEFAULT_PAGE_LIMIT},
    dto::{
        request::{
            AdminGroupDetailRequest, ApplyToGroupRequest, ApproveGroupApplicationRequest,
            BanMemberRequest, BannedMembersRequest, CancelApplicationRequest,
            CreateGroupAnnouncementRequest, CreateGroupRequest, DeleteGroupAnnouncementRequest,
            GetMyGroupStateRequest, GroupAnnouncementsRequest, GroupApplicationsRequest,
            GroupMembersRequest, GroupOpenTheDoorRequest, GroupSettingsRequest, JoinGroupRequest,
            LeaveGroupRequest, MuteMemberRequest, MyGroupsRequest,
            RejectGroupApplicationRequest, RemoveMemberRequest, SearchGroupsByKeywordRequest,
            SearchGroupsByLocationRequest, SearchGroupsByTagRequest, SetMemberRoleRequest,
            TransferOwnershipRequest, UnbanMemberRequest, UnmuteMemberRequest,
            UpdateGroupAnnouncementRequest, UpdateGroupRequest, UpdateGroupMySettingsRequest,
        },
        response::{
            AdminGroupDetail, AdminGroupSettingsSummary, AdminGroupStats, ApplyToGroupData,
            BannedMemberItem, GroupAnnouncementItem, GroupApplicationItem, GroupInfo,
            GroupMemberItem, GroupMyStateData, GroupSearchItem, GroupSettings,
            GroupSettingsData, PaginatedData, PaginationMeta, UserInfo,
        },
    },
    error::{AppError, Result},
    repositories::{
        chat_unread_repo::ChatUnreadRepository,
        group_application_repo::{GroupApplicationRepository, GroupApplicationRow},
        group_member_repo::{GroupMemberDetailRow, GroupMemberRepository},
        group_message_repo::GroupMessageRepository,
        group_repo::GroupRepository,
        user_repo::UserRepository,
    },
    types::enums::{ApplicationStatus, GroupJoinMethod, GroupRole},
    utils::{
        bloom_filter::ResourceExistenceChecker,
        cache::{cache_keys, CacheManager},
        validation::Validator,
    },
    EventBus,
};

#[derive(Clone)]
pub struct GroupService {
    event_bus: Arc<EventBus>,
    group_repo: GroupRepository,
    group_message_repo: GroupMessageRepository,
    member_repo: GroupMemberRepository,
    application_repo: GroupApplicationRepository,
    user_repo: UserRepository,
    unread_repo: ChatUnreadRepository,
    _redis: ConnectionManager,
    presence_service: crate::services::presence_service::PresenceService,
    media_service: crate::services::media_service::MediaService,
    cache_manager: CacheManager,
}

use crate::dto::request::{AdminDismissGroupRequest, AdminGroupSearchRequest};
use crate::dto::response::AdminGroupSearchItem;

#[derive(Debug, Clone, Serialize, Deserialize)]
struct CachedGroupMembers {
    members: Vec<GroupMemberDetailRow>,
    next_cursor: Option<String>,
}

impl GroupService {
    pub fn new(
        event_bus: Arc<EventBus>,
        group_repo: GroupRepository,
        group_message_repo: GroupMessageRepository,
        member_repo: GroupMemberRepository,
        application_repo: GroupApplicationRepository,
        user_repo: UserRepository,
        unread_repo: ChatUnreadRepository,
        redis: ConnectionManager,
        presence_service: crate::services::presence_service::PresenceService,
        media_service: crate::services::media_service::MediaService,
        cache_manager: CacheManager,
    ) -> Self {
        Self {
            event_bus,
            group_repo,
            group_message_repo,
            member_repo,
            application_repo,
            user_repo,
            unread_repo,
            _redis: redis,
            presence_service,
            media_service,
            cache_manager,
        }
    }

    async fn get_group_cached(
        &self,
        group_id: Uuid,
    ) -> Result<crate::repositories::group_repo::GroupRow> {
        let cache_key = group_id.to_string();
        let cached = self
            .cache_manager
            .get_or_set_with_lock(cache_keys::GROUP_INFO, &cache_key, || async {
                let group = self
                    .group_repo
                    .find_by_id(group_id)
                    .await
                    .map_err(|e| Box::new(e) as Box<dyn std::error::Error + Send + Sync>)?;
                Ok(group)
            })
            .await
            .map_err(|e| AppError::InternalError(e.to_string()))?;

        cached.ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))
    }

    async fn invalidate_group_caches(&self, group_id: Uuid) {
        let cache_key = group_id.to_string();
        if let Err(e) = self
            .cache_manager
            .delete(cache_keys::GROUP_INFO, &cache_key)
            .await
        {
            tracing::warn!(group_id = %cache_key, error = %e, "delete group info cache failed");
        }
        let ids = vec![cache_key];
        if let Err(e) = self
            .cache_manager
            .delete_batch(cache_keys::GROUP_MEMBERS, &ids)
            .await
        {
            tracing::warn!(error = %e, "delete group members cache failed");
        }
    }

    async fn invalidate_group_members_cache(&self, group_id: Uuid) {
        let ids = vec![group_id.to_string()];
        if let Err(e) = self
            .cache_manager
            .delete_batch(cache_keys::GROUP_MEMBERS, &ids)
            .await
        {
            tracing::warn!(error = %e, "delete group members cache failed");
        }
    }

    async fn get_group_members_cached(
        &self,
        group_id: Uuid,
        role: Option<&str>,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<GroupMemberDetailRow>, Option<String>)> {
        if role.is_some() || cursor.is_some() || limit != DEFAULT_PAGE_LIMIT as i32 {
            return self
                .member_repo
                .list_by_group(group_id, role, limit, cursor)
                .await;
        }

        let cache_key = group_id.to_string();
        let cached = self
            .cache_manager
            .get_or_set_with_lock_with_ttl(
                cache_keys::GROUP_MEMBERS,
                &cache_key,
                30,
                10,
                || async {
                    let (members, next_cursor) = self
                        .member_repo
                        .list_by_group(group_id, None, limit, None)
                        .await
                        .map_err(|e| Box::new(e) as Box<dyn std::error::Error + Send + Sync>)?;
                    Ok(Some(CachedGroupMembers {
                        members,
                        next_cursor,
                    }))
                },
            )
            .await
            .map_err(|e| AppError::InternalError(e.to_string()))?;

        if let Some(data) = cached {
            return Ok((data.members, data.next_cursor));
        }

        Ok((Vec::new(), None))
    }

    pub async fn create_group(&self, owner_id: i64, req: CreateGroupRequest) -> Result<()> {
        let _user = self
            .user_repo
            .find_by_id(owner_id)
            .await?
            .ok_or(AppError::UserNotFound("User not found".to_string()))?;

        let _ = self.user_repo.get_current_location(owner_id).await?;

        let join_method = req.join_method.unwrap_or(GroupJoinMethod::Open);
        let is_searchable = req.is_searchable.unwrap_or(true);
        let max_members = req.max_members.unwrap_or(500);

        if let Some(location) = req.location.as_ref() {
            Validator::validate_coordinates(location.latitude, location.longitude)?;
        }

        let mut avatar_url = None;
        if let Some(file_id_str) = req.avatar_file_id {
            if let Ok(file_id) = Uuid::parse_str(&file_id_str) {
                if let Ok(file_data) = self
                    .media_service
                    .validate_file(file_id, owner_id, "group_avatar")
                    .await
                {
                    avatar_url = Some(file_data);
                    let _ = self.media_service.mark_as_used(file_id).await;
                }
            }
        }

        let group_type = if is_searchable { "public" } else { "private" };

        let group = self
            .group_repo
            .create(
                req.name.clone(),
                req.description.clone().unwrap_or_default(),
                avatar_url,
                owner_id,
                join_method.as_str(),
                group_type,
                max_members,
            )
            .await?;
        if let Err(e) = ResourceExistenceChecker::new(self._redis.clone())
            .record_group(&group.group_id.to_string())
            .await
        {
            tracing::warn!(group_id = %group.group_id, error = %e, "record_group bloom failed");
        }

        self.member_repo
            .add_member(group.group_id, owner_id, GroupRole::Owner)
            .await?;

        Ok(())
    }

    /// 按关键字搜索群组（全局）
    pub async fn search_by_keyword(
        &self,
        req: SearchGroupsByKeywordRequest,
    ) -> Result<PaginatedData<GroupSearchItem>> {
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

        let mut items = Vec::with_capacity(results.len());
        for g in results {
            items.push(GroupSearchItem {
                group_id: g.group_id.to_string(),
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
                tags: None,
                members_count: g.member_count,
                has_password: g.has_password,
                location: RelativeLocation {
                    latitude: DEFAULT_LATITUDE,
                    longitude: DEFAULT_LONGITUDE,
                    location_name: DEFAULT_LOCATION_NAME.to_string(),
                    distance: None,
                    bearing_degrees: None,
                },
                created_at: None,
                owner_id: None,
                my_state: None,
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
    ) -> Result<PaginatedData<GroupSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = req.cursor;

        let lat = req.reference_point.latitude;
        let lng = req.reference_point.longitude;
        Validator::validate_coordinates(lat, lng)?;

        // 通过标签搜索群组

        let (results, next_cursor) = self
            .group_repo
            .search_by_tag(&req.tag, lat, lng, limit, cursor)
            .await?;

        let mut items = Vec::with_capacity(results.len());
        for g in results {
            items.push(GroupSearchItem {
                group_id: g.group_id.to_string(),
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
                tags: None,
                members_count: g.member_count,
                has_password: g.has_password,
                location: RelativeLocation {
                    latitude: DEFAULT_LATITUDE,
                    longitude: DEFAULT_LONGITUDE,
                    location_name: DEFAULT_LOCATION_NAME.to_string(),
                    distance: None,
                    bearing_degrees: None,
                },
                created_at: None,
                owner_id: None,
                my_state: None,
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
    ) -> Result<PaginatedData<GroupSearchItem>> {
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

        let mut items = Vec::with_capacity(results.len());
        for g in results {
            items.push(GroupSearchItem {
                group_id: g.group_id.to_string(),
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
                tags: None,
                members_count: g.member_count,
                has_password: g.has_password,
                location: RelativeLocation {
                    latitude: DEFAULT_LATITUDE,
                    longitude: DEFAULT_LONGITUDE,
                    location_name: DEFAULT_LOCATION_NAME.to_string(),
                    distance: None,
                    bearing_degrees: None,
                },
                created_at: None,
                owner_id: None,
                my_state: None,
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
    ) -> Result<GroupSearchItem> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        let group = self.get_group_cached(group_id).await?;
        let members_count = self.member_repo.count_members(group_id).await? as i32;
        let has_password = false;

        // 获取用户相关信息 (Zero-Lookup)
        let my_state = if let Some(uid) = user_id {
            let member = self.member_repo.find_member(group_id, uid).await?;
            if let Some(m) = member {
                let unread = 0i64;
                Some(GroupMyStateData {
                    role: m.role,
                    is_joined: true,
                    unread_count: unread as i32,
                    is_muted: false,
                    is_pinned: false,
                    joined_at: Some(m.joined_at.to_rfc3339()),
                })
            } else {
                None
            }
        } else {
            None
        };
        Ok(GroupSearchItem {
            group_id: group.group_id.to_string(),
            name: group.name.clone(),
            description: if group.description.as_ref().map(|d| d.is_empty()).unwrap_or(true) {
                None
            } else {
                group.description.clone()
            },
            avatar_url: self
                .media_service
                .sign_url_if_path(group.avatar_url.as_deref())
                .await,
            tags: None,
            members_count,
            has_password,
            location: RelativeLocation {
                latitude: DEFAULT_LATITUDE,
                longitude: DEFAULT_LONGITUDE,
                location_name: DEFAULT_LOCATION_NAME.to_string(),
                distance: None,
                bearing_degrees: None,
            },
            created_at: Some(group.created_at.to_rfc3339()),
            owner_id: Some(group.owner_id),
            my_state,
        })
    }

    /// 获取群组设置
    pub async fn get_settings(&self, req: GroupSettingsRequest) -> Result<GroupSettingsData> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        let group = self.get_group_cached(group_id).await?;

        Ok(GroupSettingsData {
            group_id: group.group_id.to_string(),
            settings: GroupSettings {
                join_method: GroupJoinMethod::Open,
                is_searchable: true,
                max_members: 500,
                location: RelativeLocation {
                    latitude: DEFAULT_LATITUDE,
                    distance: None,
                    bearing_degrees: None,
                    longitude: DEFAULT_LONGITUDE,
                    location_name: DEFAULT_LOCATION_NAME.to_string(),
                },
            },
            has_password: false,
            created_at: group.created_at.to_rfc3339(),
        })
    }

    pub async fn get_joined(
        &self,
        user_id: i64,
        req: MyGroupsRequest,
    ) -> Result<PaginatedData<GroupSearchItem>> {
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
        let group_ids: Vec<Uuid> = memberships.iter().map(|m| m.group_id).collect();

        // 3. 批量查询（并行执行）
        // 使用 find_by_ids_with_distance 获取详情，传入 0,0 作为坐标（此处不关心距离）
        let groups = self
            .group_repo
            .find_by_ids_with_distance(&group_ids, 0.0, 0.0)
            .await?;
        let unread_counts: std::collections::HashMap<String, i64> =
            std::collections::HashMap::new();

        // 4. 构建映射
        let group_map: std::collections::HashMap<Uuid, _> =
            groups.into_iter().map(|g| (g.group_id, g)).collect();

        // 5. 组装结果
        let mut items = Vec::with_capacity(memberships.len());
        for m in &memberships {
            if let Some(g) = group_map.get(&m.group_id) {
                let unread_count = unread_counts
                    .get(&g.group_id.to_string())
                    .copied()
                    .unwrap_or(0);
                items.push(GroupSearchItem {
                    group_id: g.group_id.to_string(),
                    name: g.name.clone(),
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(g.avatar_url.as_deref())
                        .await,
                    description: if g.description.is_empty() {
                        None
                    } else {
                        Some(g.description.clone())
                    },
                    tags: None,
                    has_password: g.has_password,
                    location: RelativeLocation {
                        latitude: DEFAULT_LATITUDE,
                        longitude: DEFAULT_LONGITUDE,
                        location_name: DEFAULT_LOCATION_NAME.to_string(),
                        distance: None,
                        bearing_degrees: None,
                    },
                    members_count: g.member_count,
                    created_at: Some(g.created_at.to_rfc3339()),
                    owner_id: Some(g.owner_id),
                    my_state: Some(GroupMyStateData {
                        role: m.role,
                        is_joined: true,
                        unread_count: unread_count as i32,
                        is_muted: false,
                        is_pinned: false,
                        joined_at: Some(m.joined_at.to_rfc3339()),
                    }),
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

    pub async fn update_group(&self, user_id: i64, req: UpdateGroupRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
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

        let _ = avatar_url.clone();
        self.group_repo
            .update(group_id, req.name, req.description, avatar_url)
            .await?;

        let _ = (
            req.password,
            req.join_method,
            req.is_searchable,
            req.max_members,
        );

        // 如果提供了位置信息，更新群组位置
        if let Some(loc) = &req.location {
            Validator::validate_coordinates(loc.latitude, loc.longitude)?;
        }

        // 失效群组缓存
        self.invalidate_group_caches(group_id).await;

        let _ = (updated_name, updated_description, has_avatar_update);

        Ok(())
    }

    pub async fn delete_group(
        &self,
        user_id: i64,
        req: crate::dto::request::DeleteGroupRequest,
    ) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
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

        // 失效群组缓存
        self.invalidate_group_caches(group_id).await;

        let _ = (group_id, user_id);

        Ok(())
    }

    pub async fn get_members(
        &self,
        req: GroupMembersRequest,
    ) -> Result<PaginatedData<GroupMemberItem>> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = req.cursor;

        // 转换 GroupRole 枚举为字符串用于数据库查询
        let role = req.role.as_ref().map(|r| r.as_str());
        let (members, next_cursor) = self
            .get_group_members_cached(group_id, role, limit, cursor)
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
                role: m.role,
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

    pub async fn join_group(&self, user_id: i64, req: JoinGroupRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        // 检查群组是否存在
        if !self.group_repo.exists(group_id).await? {
            return Err(AppError::GroupNotFound("Group not found".to_string()));
        }

        // 检查是否被封禁
        if self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::AlreadyInGroup);
        }

        self.member_repo
            .add_member(group_id, user_id, GroupRole::Member)
            .await?;

        // 失效群组成员缓存
        self.invalidate_group_members_cache(group_id).await;

        // 获取最新成员数
        let member_count = self.member_repo.count_members(group_id).await?;

        let _ = (group_id, user_id, member_count);

        Ok(())
    }

    pub async fn apply_to_group(
        &self,
        user_id: i64,
        req: ApplyToGroupRequest,
    ) -> Result<ApplyToGroupData> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        if self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::AlreadyInGroup);
        }

        let app = self
            .application_repo
            .create(group_id, user_id, req.message)
            .await?;

        Ok(ApplyToGroupData {
            request_id: app.request_id.to_string(),
        })
    }

    pub async fn set_role(
        &self,
        user_id: i64,
        req: SetMemberRoleRequest,
    ) -> Result<GroupMemberItem> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) {
            return Err(AppError::Forbidden("Only owner can set roles".to_string()));
        }

        let old_role = self
            .member_repo
            .get_member_role(group_id, target_uid)
            .await?
            .ok_or_else(|| AppError::NotFound("Member not found".to_string()))?;
        // 使用 SetableGroupRole 的转换方法获取 GroupRole
        let new_role = req.role.to_group_role();

        self.member_repo
            .update_role(group_id, target_uid, new_role)
            .await?;

        // 记录角色变更历史
        let _ = (group_id, target_uid, user_id, old_role, new_role);

        // 返回更新后的成员信息
        self.get_member_item(group_id, target_uid).await
    }

    /// 获取成员详情（内部辅助方法）
    async fn get_member_item(&self, group_id: Uuid, member_id: i64) -> Result<GroupMemberItem> {
        let member = self
            .member_repo
            .find_member(group_id, member_id)
            .await?
            .ok_or_else(|| AppError::NotFound("Member not found".to_string()))?;

        let user = self
            .user_repo
            .find_by_id(member_id)
            .await?
            .ok_or_else(|| AppError::NotFound("User not found".to_string()))?;

        Ok(GroupMemberItem {
            user: UserInfo {
                user_id: user.user_id,
                nickname: user.nickname,
                avatar_url: user.avatar_url,
                user_type: user.user_type,
            },
            role: member.role,
            joined_at: member.joined_at.to_rfc3339(),
            muted_until: member.muted_until.map(|t| t.to_rfc3339()),
            is_online: None, // 隐私受控，需要根据设置判断
        })
    }

    pub async fn mute_member(
        &self,
        user_id: i64,
        req: MuteMemberRequest,
    ) -> Result<GroupMemberItem> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        let until = chrono::Utc::now() + chrono::Duration::minutes(req.duration_minutes as i64);
        self.member_repo
            .mute_member(group_id, target_uid, until)
            .await?;

        let _ = (group_id, target_uid, until);

        // 返回更新后的成员信息
        self.get_member_item(group_id, target_uid).await
    }

    pub async fn unmute_member(
        &self,
        user_id: i64,
        req: UnmuteMemberRequest,
    ) -> Result<GroupMemberItem> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.member_repo.unmute_member(group_id, target_uid).await?;

        let _ = (group_id, target_uid);

        // 返回更新后的成员信息
        self.get_member_item(group_id, target_uid).await
    }

    /// 封禁成员
    pub async fn ban_member(
        &self,
        user_id: i64,
        req: BanMemberRequest,
    ) -> Result<BannedMemberItem> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id;

        // 检查权限
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        // 从群组移除成员
        self.member_repo.remove_member(group_id, target_uid).await?;

        // 获取最新成员数
        let member_count = self.member_repo.count_members(group_id).await?;

        let _ = (group_id, target_uid, user_id, member_count);

        // 获取被封禁用户信息
        let user = self
            .user_repo
            .find_by_id(target_uid)
            .await?
            .ok_or_else(|| AppError::NotFound("User not found".to_string()))?;

        Ok(BannedMemberItem {
            user: UserInfo {
                user_id: user.user_id,
                nickname: user.nickname,
                avatar_url: user.avatar_url,
                user_type: user.user_type,
            },
            reason: req.reason,
            banned_at: chrono::Utc::now().to_rfc3339(),
        })
    }

    /// 解除封禁成员
    pub async fn unban_member(&self, user_id: i64, req: UnbanMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id;

        // 检查权限
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        Ok(())
    }

    /// 获取封禁成员列表
    pub async fn get_banned_list(
        &self,
        user_id: i64,
        req: BannedMembersRequest,
    ) -> Result<PaginatedData<BannedMemberItem>> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;

        // 检查权限（群主或管理员可以查看封禁列表）
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        Ok(PaginatedData {
            items: Vec::new(),
            pagination: PaginationMeta {
                has_more: false,
                next_cursor: None,
            },
        })
    }

    pub async fn remove_member(&self, user_id: i64, req: RemoveMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let target_uid = req.user_id;

        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.member_repo.remove_member(group_id, target_uid).await?;

        // 获取最新成员数
        let member_count = self.member_repo.count_members(group_id).await?;

        let _ = (group_id, target_uid, user_id, member_count);

        Ok(())
    }

    pub async fn leave_group(&self, user_id: i64, req: LeaveGroupRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
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

        // 获取最新成员数
        let member_count = self.member_repo.count_members(group_id).await?;

        let _ = (group_id, user_id, member_count);

        Ok(())
    }

    pub async fn get_applications(
        &self,
        _user_id: i64,
        req: GroupApplicationsRequest,
    ) -> Result<PaginatedData<GroupApplicationItem>> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("Invalid Group ID".to_string()))?;
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = req.cursor;

        let status = req.status.and_then(|filter| {
            use crate::types::enums::{ApplicationStatus, ApplicationStatusFilter};
            match filter {
                ApplicationStatusFilter::Any => None,
                ApplicationStatusFilter::Specific(s) => Some(match s {
                    ApplicationStatus::Pending => ApplicationStatus::Pending,
                    ApplicationStatus::Accepted => ApplicationStatus::Accepted,
                    ApplicationStatus::Rejected => ApplicationStatus::Rejected,
                }),
            }
        });
        let (apps, next_cursor) = self
            .application_repo
            .list_by_group(group_id, status, limit, cursor)
            .await?;

        let mut user_ids: Vec<i64> = apps.iter().map(|a| a.user_id).collect();
        user_ids.sort_unstable();
        user_ids.dedup();
        let profiles = self.user_repo.find_profiles(&user_ids).await?;
        let profile_map: std::collections::HashMap<i64, _> =
            profiles.into_iter().map(|u| (u.user_id, u)).collect();

        let mut items = Vec::with_capacity(apps.len());
        for a in apps {
            let user = profile_map.get(&a.user_id);
            let avatar_url = self
                .media_service
                .sign_url_if_path(user.and_then(|u| u.avatar_url.as_deref()))
                .await;

            items.push(GroupApplicationItem {
                request_id: a.request_id.to_string(),
                user: UserInfo {
                    user_id: a.user_id,
                    nickname: user
                        .map(|u| u.nickname.clone())
                        .unwrap_or_else(|| "Unknown".to_string()),
                    avatar_url,
                    user_type: user
                        .map(|u| u.user_type)
                        .unwrap_or(crate::types::enums::UserType::Normal),
                },
                message: a.reason,
                status: a.status,
                created_at: a.created_at.to_rfc3339(),
                updated_at: a.updated_at.map(|t| t.to_rfc3339()),
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

    async fn build_application_item(
        &self,
        app: GroupApplicationRow,
    ) -> Result<GroupApplicationItem> {
        let user = self.user_repo.find_by_id(app.user_id).await?;
        let avatar_url = self
            .media_service
            .sign_url_if_path(user.as_ref().and_then(|u| u.avatar_url.as_deref()))
            .await;

        Ok(GroupApplicationItem {
            request_id: app.request_id.to_string(),
            user: UserInfo {
                user_id: app.user_id,
                nickname: user.as_ref()
                    .map(|u| u.nickname.clone())
                    .unwrap_or_else(|| "Unknown".to_string()),
                avatar_url,
                user_type: user
                    .map(|u| u.user_type)
                    .unwrap_or(crate::types::enums::UserType::Normal),
            },
            message: app.reason,
            status: app.status,
            created_at: app.created_at.to_rfc3339(),
            updated_at: app.updated_at.map(|t| t.to_rfc3339()),
        })
    }

    pub async fn approve_application(
        &self,
        user_id: i64,
        req: ApproveGroupApplicationRequest,
    ) -> Result<GroupApplicationItem> {
        let request_id = Uuid::parse_str(&req.request_id)
            .map_err(|_| AppError::BadRequest("Invalid Request ID".to_string()))?;
        let app = self
            .application_repo
            .find_by_request_id(request_id)
            .await?
            .ok_or(AppError::NotFound("Application not found".to_string()))?;

        let role = self
            .member_repo
            .get_member_role(app.group_id, user_id)
            .await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.application_repo
            .update_status_by_request_id(request_id, ApplicationStatus::Accepted)
            .await?;
        self.member_repo
            .add_member(app.group_id, app.user_id, GroupRole::Member)
            .await?;

        self.invalidate_group_members_cache(app.group_id).await;

        let group_name = self
            .get_group_cached(app.group_id)
            .await
            .map(|g| g.name)
            .unwrap_or_else(|_| "Unknown".to_string());

        self.event_bus.publish(AppEvent::GroupApplicationResponse {
            request_id,
            group_id: app.group_id,
            group_name,
            user_id: app.user_id,
            status: ResponseStatus::Accepted,
        });

        let member_count = self.member_repo.count_members(app.group_id).await?;
        self.event_bus.publish(AppEvent::GroupMemberAdded {
            group_id: app.group_id,
            user_id: app.user_id,
            added_by: user_id,
            role: GroupRole::Member,
            member_count,
        });

        self.build_application_item(app).await
    }

    pub async fn reject_application(
        &self,
        user_id: i64,
        req: RejectGroupApplicationRequest,
    ) -> Result<GroupApplicationItem> {
        if let Some(reason) = req.reason.as_deref() {
            let reason = reason.trim();
            if !reason.is_empty() {
                Validator::validate_length(reason, "拒绝原因", 1, 200)?;
            }
        }

        let request_id = Uuid::parse_str(&req.request_id)
            .map_err(|_| AppError::BadRequest("Invalid Request ID".to_string()))?;
        let app = self
            .application_repo
            .find_by_request_id(request_id)
            .await?
            .ok_or(AppError::NotFound("Application not found".to_string()))?;

        let role = self
            .member_repo
            .get_member_role(app.group_id, user_id)
            .await?;
        if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
            return Err(AppError::Forbidden("Permission denied".to_string()));
        }

        self.application_repo
            .update_status_by_request_id(request_id, ApplicationStatus::Rejected)
            .await?;

        let group_name = self
            .get_group_cached(app.group_id)
            .await
            .map(|g| g.name)
            .unwrap_or_else(|_| "Unknown".to_string());

        self.event_bus.publish(AppEvent::GroupApplicationResponse {
            request_id,
            group_id: app.group_id,
            group_name,
            user_id: app.user_id,
            status: ResponseStatus::Rejected,
        });

        self.build_application_item(app).await
    }

    pub async fn cancel_application(
        &self,
        user_id: i64,
        req: CancelApplicationRequest,
    ) -> Result<()> {
        let request_id = Uuid::parse_str(&req.request_id)
            .map_err(|_| AppError::BadRequest("Invalid Request ID".to_string()))?;
        let app = self
            .application_repo
            .find_by_request_id(request_id)
            .await?
            .ok_or(AppError::NotFound("Application not found".to_string()))?;

        if app.user_id != user_id {
            return Err(AppError::Forbidden("只能取消自己的申请".to_string()));
        }

        if app.status != ApplicationStatus::Pending {
            return Err(AppError::BadRequest("只能取消待处理的申请".to_string()));
        }

        self.application_repo
            .delete_by_request_id(request_id)
            .await?;

        Ok(())
    }

    pub async fn transfer_ownership(
        &self,
        user_id: i64,
        req: TransferOwnershipRequest,
    ) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
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
            .update_role(group_id, user_id, GroupRole::Admin)
            .await?;
        self.member_repo
            .update_role(group_id, new_owner, GroupRole::Owner)
            .await?;

        Ok(())
    }

    // ==================== 管理员群组管理 (Admin) ====================

    pub async fn admin_search_groups(
        &self,
        req: &AdminGroupSearchRequest,
    ) -> Result<PaginatedData<AdminGroupSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let cursor = req.cursor.clone();

        let group_id = if let Some(gid_str) = &req.group_id {
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
            items.push(AdminGroupSearchItem {
                group_id: g.group_id.to_string(),
                name: g.name,
                avatar_url: self
                    .media_service
                    .sign_url_if_path(g.avatar_url.as_deref())
                    .await,
                members_count: g.member_count as i32,
                location: None,
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

    pub async fn admin_group_detail(
        &self,
        req: &AdminGroupDetailRequest,
    ) -> Result<AdminGroupDetail> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        let group = self.get_group_cached(group_id).await?;

        let members_count = self.member_repo.count_members(group_id).await? as i32;
        let join_method = GroupJoinMethod::Open;
        let is_searchable = true;
        let max_members = 500;
        let has_password = false;

        let owner_profile = self
            .user_repo
            .find_profiles(&[group.owner_id])
            .await?
            .into_iter()
            .next()
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;
        let owner = UserInfo {
            user_id: owner_profile.user_id,
            nickname: owner_profile.nickname,
            avatar_url: self
                .media_service
                .sign_url_if_path(owner_profile.avatar_url.as_deref())
                .await,
            user_type: owner_profile.user_type,
        };

        let admin_ids = self.member_repo.list_admin_user_ids(group_id).await?;
        let admin_profiles = self.user_repo.find_profiles(&admin_ids).await?;
        let admins = if admin_profiles.is_empty() {
            None
        } else {
            let mut items = Vec::with_capacity(admin_profiles.len());
            for profile in admin_profiles {
                items.push(UserInfo {
                    user_id: profile.user_id,
                    nickname: profile.nickname,
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(profile.avatar_url.as_deref())
                        .await,
                    user_type: profile.user_type,
                });
            }
            Some(items)
        };

        let messages_count = self.group_message_repo.count_by_group(group_id).await? as i32;
        let active_members_today = self
            .group_message_repo
            .count_active_members_today(group_id)
            .await? as i32;

        Ok(AdminGroupDetail {
            group_id: group.group_id.to_string(),
            name: group.name,
            avatar_url: self
                .media_service
                .sign_url_if_path(group.avatar_url.as_deref())
                .await,
            members_count,
            location: None,
            description: if group.description.is_empty() {
                None
            } else {
                Some(group.description)
            },
            tags: None,
            max_members,
            owner,
            created_at: group.created_at.to_rfc3339(),
            settings: AdminGroupSettingsSummary {
                join_method,
                is_searchable,
                has_password,
            },
            admins,
            stats: AdminGroupStats {
                messages_count,
                active_members_today,
            },
            dismissal_history: None,
        })
    }

    pub async fn admin_dismiss_group(
        &self,
        admin_id: i64,
        req: &AdminDismissGroupRequest,
    ) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
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
        let _ = (user_id, req);
        Err(AppError::BadRequest("群公告暂不可用".to_string()))
    }

    /// 获取群公告列表
    pub async fn list_announcements(
        &self,
        req: &GroupAnnouncementsRequest,
    ) -> Result<PaginatedData<GroupAnnouncementItem>> {
        Ok(PaginatedData {
            items: Vec::new(),
            pagination: PaginationMeta {
                has_more: false,
                next_cursor: None,
            },
        })
    }

    /// 更新群公告
    pub async fn update_announcement(
        &self,
        user_id: i64,
        req: &UpdateGroupAnnouncementRequest,
    ) -> Result<GroupAnnouncementItem> {
        let _ = (user_id, req);
        Err(AppError::BadRequest("群公告暂不可用".to_string()))
    }

    /// 删除群公告
    pub async fn delete_announcement(
        &self,
        user_id: i64,
        req: &DeleteGroupAnnouncementRequest,
    ) -> Result<()> {
        let _ = (user_id, req);
        Ok(())
    }

    /// 获取我在群内的状态
    pub async fn get_my_state(
        &self,
        user_id: i64,
        req: &GetMyGroupStateRequest,
    ) -> Result<GroupMyStateData> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        // 验证群组是否存在
        if !self.group_repo.exists(group_id).await? {
            return Err(AppError::GroupNotFound("群组不存在".to_string()));
        }

        // 获取成员信息
        let member = self
            .member_repo
            .find_member(group_id, user_id)
            .await?
            .ok_or_else(|| AppError::NotGroupMember("您不是群组成员".to_string()))?;

        // 获取未读消息数
        let unread_count = 0;

        // 获取用户群组偏好设置
        Ok(GroupMyStateData {
            role: member.role,
            is_joined: true,
            unread_count,
            is_muted: false,
            is_pinned: false,
            joined_at: Some(member.joined_at.to_rfc3339()),
        })
    }

    /// 更新我的群组偏好设置
    pub async fn update_my_settings(
        &self,
        user_id: i64,
        req: &UpdateGroupMySettingsRequest,
    ) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        // 验证群组是否存在
        if !self.group_repo.exists(group_id).await? {
            return Err(AppError::GroupNotFound("群组不存在".to_string()));
        }

        // 验证用户是否是群组成员
        if !self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::NotGroupMember("您不是群组成员".to_string()));
        }

        let _ = (group_id, user_id, req);
        Ok(())
    }
}
