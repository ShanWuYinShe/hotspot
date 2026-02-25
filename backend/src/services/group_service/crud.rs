use std::collections::{HashMap, HashSet};

use tokio::{join, try_join};

use super::{helpers::AdminGroupContext, GroupService};
use crate::{
    dto::{
        request::{
            AdminDismissGroupRequest, AdminGroupDetailRequest, AdminGroupSearchRequest,
            CreateGroupChatRequest, DeleteGroupChatRequest, GroupOpenTheDoorRequest,
            MyGroupChatsRequest, SearchGroupsRequest, TransferOwnershipRequest,
            UpdateGroupChatRequest, UpdateGroupMySettingsRequest,
        },
        response::{
            AdminGroupItem, AdminGroupStats, GroupInfo, GroupItem, GroupMyStateData, PaginatedData,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    types::enums::{GroupJoinMethod, GroupRole},
    utils::{cache::cache_keys, password::PasswordUtil, validation::Validator},
};

impl GroupService {
    fn normalize_group_sort(sort_by: Option<&str>) -> &'static str {
        match sort_by.unwrap_or("relevance").to_lowercase().as_str() {
            "distance" => "distance",
            "newest" => "newest",
            "popular" => "popular",
            "relevance" => "relevance",
            _ => "relevance",
        }
    }

    pub async fn search(&self, req: &SearchGroupsRequest) -> Result<PaginatedData<GroupItem>> {
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i64;
        let cursor = pagination.and_then(|p| p.cursor.clone());
        let sort_by = Self::normalize_group_sort(req.sort_by.as_deref());
        let (groups, next_cursor) = if let Some(radius) = req.radius {
            self.group_repo
                .search_nearby(
                    req.reference_point.longitude,
                    req.reference_point.latitude,
                    radius,
                    limit,
                    cursor.clone(),
                    req.keyword.as_deref(),
                    sort_by,
                )
                .await?
        } else if let Some(ref tag) = req.tag {
            self.group_repo
                .search_by_tag(
                    tag,
                    req.reference_point.longitude,
                    req.reference_point.latitude,
                    limit,
                    cursor.clone(),
                    sort_by,
                )
                .await?
        } else if let Some(ref keyword) = req.keyword {
            self.group_repo
                .search_by_keyword(
                    keyword,
                    req.reference_point.longitude,
                    req.reference_point.latitude,
                    limit,
                    cursor.clone(),
                    sort_by,
                )
                .await?
        } else {
            return Err(AppError::BadRequest(
                "至少需要提供 keyword、tag、radius 中的一个搜索条件".to_string(),
            ));
        };

        let items = self.map_search_rows(groups).await?;
        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn open_the_door(
        &self,
        req: &GroupOpenTheDoorRequest,
        user_id: Option<i64>,
    ) -> Result<GroupItem> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let group = self
            .group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;

        let (tags_res, settings, members_count) = try_join!(
            self.tag_repo.get_group_tags(group_id),
            self.settings_repo.find_by_group(group_id),
            self.member_repo.count_members(group_id)
        )?;
        let tags = Some(tags_res);
        let members_count = members_count as i32;
        let has_password = settings
            .as_ref()
            .and_then(|s| s.password_hash.as_ref())
            .is_some();

        let my_state = if let Some(uid) = user_id {
            if let Some(m) = self.member_repo.find_member(group_id, uid).await? {
                let group_id_str = group_id.to_string();
                let (unread_count, prefs) = try_join!(
                    self.unread_repo
                        .get_unread_count("group", &group_id_str, uid),
                    self.settings_repo.find_user_preferences(group_id, uid)
                )?;
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

        Ok(GroupItem {
            group: GroupInfo {
                group_id: group.group_id.to_string(),
                name: group.name,
                avatar_url: self
                    .media_service
                    .sign_url_if_path(group.avatar_url.as_deref())
                    .await,
                members_count: members_count as i32,
            },
            description: group.description,
            tags,
            has_password,
            location: Self::build_location(
                group.latitude,
                group.longitude,
                group.location_name.as_deref(),
                None,
                None,
            ),
            last_activity_at: None,
            created_at: Some(group.created_at.to_rfc3339()),
            owner_id: Some(group.owner_id.to_string()),
            my_state,
            friends_in_group: None,
            join_method: None,
            is_searchable: None,
            max_members: None,
        })
    }

    pub async fn create_group(
        &self,
        owner_id: i64,
        req: CreateGroupChatRequest,
    ) -> Result<GroupItem> {
        Validator::validate_length(&req.name, "群组名称", 1, 100)?;

        let lon = Some(req.location.point.longitude);
        let lat = Some(req.location.point.latitude);

        let join_method = req.join_method.unwrap_or(GroupJoinMethod::Open);
        let join_method_str = Self::join_method_to_string(join_method);

        let mut group = self
            .group_repo
            .create(
                req.name.clone(),
                owner_id,
                lon,
                lat,
                "public".to_string(),
                Self::join_type_from_join_method(join_method),
            )
            .await?;

        let avatar_url = if let Some(fid) = req.avatar_file_id {
            let fid_uuid = Validator::parse_uuid(&fid, "文件ID")?;
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

        // spec: location_name 由客户端提供，必须完整存储
        let loc_point = req.location.point;
        self.group_repo
            .update_location(
                group.group_id,
                loc_point.longitude,
                loc_point.latitude,
                Some(&req.location.location_name),
            )
            .await?;
        group.latitude = Some(loc_point.latitude);
        group.longitude = Some(loc_point.longitude);
        group.location_name = Some(req.location.location_name.clone());

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

        let members_count = 1i32;
        let (tags_res, prefs) = try_join!(
            self.tag_repo.get_group_tags(group.group_id),
            self.settings_repo
                .find_user_preferences(group.group_id, owner_id)
        )?;
        let tags = Some(tags_res);
        let has_password = password_hash.is_some();
        Ok(GroupItem {
            group: GroupInfo {
                group_id: group.group_id.to_string(),
                name: group.name.clone(),
                avatar_url: self
                    .media_service
                    .sign_url_if_path(
                        avatar_url
                            .as_deref()
                            .or_else(|| group.avatar_url.as_deref()),
                    )
                    .await,
                members_count: members_count as i32,
            },
            description: req.description.or(group.description),
            tags,
            has_password,
            location: Self::build_location(
                group.latitude,
                group.longitude,
                group.location_name.as_deref(),
                None,
                None,
            ),
            last_activity_at: None,
            created_at: Some(group.created_at.to_rfc3339()),
            owner_id: Some(group.owner_id.to_string()),
            my_state: Some(GroupMyStateData {
                role: GroupRole::Owner,
                is_joined: true,
                unread_count: 0,
                is_muted: prefs.as_ref().map(|p| p.is_muted).unwrap_or(false),
                is_pinned: prefs.as_ref().map(|p| p.is_pinned).unwrap_or(false),
                joined_at: Some(group.created_at.to_rfc3339()),
            }),
            friends_in_group: None,
            join_method: Some(join_method),
            is_searchable: Some(req.is_searchable.unwrap_or(true)),
            max_members: req.max_members.or(Some(group.max_members)),
        })
    }

    pub async fn get_joined(
        &self,
        user_id: i64,
        req: MyGroupChatsRequest,
    ) -> Result<PaginatedData<GroupItem>> {
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as usize;
        let (rows, next_cursor) = self
            .member_repo
            .list_user_groups(user_id, limit, pagination.and_then(|p| p.cursor.as_deref()))
            .await?;

        let group_ids: Vec<i64> = rows.iter().map(|r| r.group_id).collect();
        let group_id_strs: Vec<String> = group_ids.iter().map(|id| id.to_string()).collect();
        let (unread_res, tags_res, roles_res) = join!(
            self.unread_repo
                .get_unread_count_batch("group", &group_id_strs, user_id),
            self.tag_repo.get_group_tags_batch(&group_ids),
            self.member_repo.batch_get_member_roles(&group_ids, user_id)
        );
        let unread_counts = unread_res.unwrap_or_default();
        let tags_map: HashMap<i64, Vec<String>> = tags_res.unwrap_or_default();
        let roles_map = roles_res?;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            let unread_count = unread_counts
                .get(&row.group_id.to_string())
                .copied()
                .unwrap_or(0) as i32;
            let tags = tags_map.get(&row.group_id).cloned();
            let role = roles_map
                .get(&row.group_id)
                .copied()
                .unwrap_or(GroupRole::Member);
            items.push(GroupItem {
                group: GroupInfo {
                    group_id: row.group_id.to_string(),
                    name: row.name,
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(row.avatar_url.as_deref())
                        .await,
                    members_count: row.members_count,
                },
                description: row.description,
                tags,
                has_password: row.has_password,
                location: Self::build_location(
                    row.latitude,
                    row.longitude,
                    row.location_name.as_deref(),
                    None,
                    None,
                ),
                last_activity_at: Some(row.joined_at.to_rfc3339()),
                created_at: None,
                owner_id: None,
                my_state: Some(GroupMyStateData {
                    role,
                    unread_count,
                    is_joined: true,
                    is_muted: false,
                    is_pinned: false,
                    joined_at: Some(row.joined_at.to_rfc3339()),
                }),
                friends_in_group: None,
                join_method: None,
                is_searchable: None,
                max_members: None,
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn update_group(&self, user_id: i64, req: UpdateGroupChatRequest) -> Result<()> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let _ = self.require_admin_or_owner(group_id, user_id).await?;

        let avatar_url = if let Some(fid) = req.avatar_file_id.as_deref() {
            let fid_uuid = Validator::parse_uuid(fid, "文件ID")?;
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

        if let Some(ref loc) = req.location {
            Validator::validate_coordinates(loc.point.latitude, loc.point.longitude)?;
            self.group_repo
                .update_location(
                    group_id,
                    loc.point.longitude,
                    loc.point.latitude,
                    Some(&loc.location_name),
                )
                .await?;
        }

        let event_tags = req.tags.clone();
        if let Some(tags) = req.tags {
            self.tag_repo.set_group_tags(group_id, &tags).await?;
        }

        if req.join_method.is_some()
            || req.is_searchable.is_some()
            || req.max_members.is_some()
            || req.password.is_some()
            || req.clear_password == Some(true)
        {
            if req.password.is_some() && req.clear_password == Some(true) {
                return Err(AppError::BadRequest(
                    "password 与 clear_password 互斥".to_string(),
                ));
            }

            let join_method = req.join_method.map(Self::join_method_to_string);
            if req.clear_password == Some(true) {
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
            } else if let Some(pwd) = req.password.as_deref() {
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
            } else {
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

        self.event_bus.publish(AppEvent::GroupProfileUpdated {
            group_id,
            updated_by: user_id,
            name: req.name,
            description: req.description,
            avatar_url,
            tags: event_tags,
            join_method: req.join_method.map(|jm| Self::join_method_to_string(jm)),
            is_searchable: req.is_searchable,
            max_members: req.max_members,
            has_password: if req.clear_password == Some(true) {
                Some(false)
            } else {
                req.password.as_ref().map(|_| true)
            },
        });

        let _ = self
            .cache_manager
            .delete(cache_keys::GROUP_INFO, &group_id.to_string())
            .await;
        Ok(())
    }

    pub async fn delete_group(&self, user_id: i64, req: DeleteGroupChatRequest) -> Result<()> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let _ = self.require_owner(group_id, user_id).await?;

        self.group_repo.dismiss(group_id).await?;
        self.event_bus.publish(AppEvent::GroupDismissed {
            group_id,
            dismissed_by: user_id,
            dismissed_by_owner: true,
        });
        Ok(())
    }

    pub async fn transfer_ownership(
        &self,
        user_id: i64,
        req: TransferOwnershipRequest,
    ) -> Result<()> {
        let new_owner_id = Validator::parse_user_id(&req.new_owner_id)?;
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let _ = self.require_owner(group_id, user_id).await?;

        if !self.member_repo.is_member(group_id, new_owner_id).await? {
            return Err(AppError::NotGroupMember("目标用户不在群组中".to_string()));
        }

        self.group_repo.update_owner(group_id, new_owner_id).await?;
        self.member_repo
            .update_role(group_id, new_owner_id, GroupRole::Owner)
            .await?;
        self.member_repo
            .update_role(group_id, user_id, GroupRole::Member)
            .await?;
        Ok(())
    }

    pub async fn update_my_settings(
        &self,
        user_id: i64,
        req: &UpdateGroupMySettingsRequest,
    ) -> Result<()> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        self.settings_repo
            .upsert_user_preferences(group_id, user_id, req.is_muted, req.is_pinned)
            .await?;
        Ok(())
    }

    pub async fn admin_search_groups(
        &self,
        req: &AdminGroupSearchRequest,
    ) -> Result<PaginatedData<AdminGroupItem>> {
        let limit_input = req.pagination.as_ref().and_then(|p| p.limit).unwrap_or(20);
        let limit = Validator::validate_limit(Some(limit_input), 50)? as i64;
        let cursor = req.pagination.as_ref().and_then(|p| p.cursor.clone());
        let group_id = req
            .group_id
            .as_deref()
            .and_then(|id| Validator::parse_group_id(id).ok());
        let owner_id = req
            .owner_id
            .as_deref()
            .and_then(|id| Validator::parse_user_id(id).ok());
        let (rows, next_cursor) = self
            .group_repo
            .admin_search(group_id, req.name.clone(), owner_id, limit, cursor)
            .await?;

        let group_ids: Vec<i64> = rows.iter().map(|r| r.group_id).collect();
        let tags_map = self.tag_repo.get_group_tags_batch(&group_ids).await?;
        let settings_map = self.settings_repo.find_by_groups(&group_ids).await?;
        let admin_map = self.member_repo.list_admin_members(&group_ids).await?;
        let report_stats = self.report_repo.count_group_reports(&group_ids).await?;

        let mut contexts = Vec::with_capacity(group_ids.len());
        let mut user_ids = HashSet::new();

        for row in rows {
            let tags = tags_map.get(&row.group_id).cloned();
            let settings = settings_map.get(&row.group_id);
            let group_item = self
                .build_group_item_from_admin_row(&row, tags, settings)
                .await?;
            let admin_ids = admin_map.get(&row.group_id).cloned().unwrap_or_default();

            user_ids.insert(row.owner_id);
            for admin_id in &admin_ids {
                user_ids.insert(*admin_id);
            }

            let (reports_count, last_reported_at) = report_stats
                .get(&row.group_id)
                .cloned()
                .unwrap_or((0, None));
            contexts.push(AdminGroupContext {
                group: group_item,
                owner_id: row.owner_id,
                admin_ids,
                stats: AdminGroupStats {
                    messages_count: row.messages_count,
                    active_members_today: 0,
                    reports_count: if reports_count > 0 {
                        Some(reports_count as i32)
                    } else {
                        None
                    },
                    last_reported_at: last_reported_at.map(|ts| ts.to_rfc3339()),
                },
            });
        }

        let user_info_map = self
            .build_user_infos(&user_ids.into_iter().collect::<Vec<_>>())
            .await?;

        let items = contexts
            .into_iter()
            .map(|ctx| self.compose_admin_group_item(ctx, &user_info_map))
            .collect();

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn admin_group_detail(
        &self,
        req: &AdminGroupDetailRequest,
    ) -> Result<AdminGroupItem> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        let group = self
            .group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::GroupNotFound("群组不存在".to_string()))?;
        let (members_count, settings, tags_res) = try_join!(
            self.member_repo.count_members(group_id),
            self.settings_repo.find_by_group(group_id),
            self.tag_repo.get_group_tags(group_id)
        )?;
        let members_count = members_count as i32;
        let tags = Some(tags_res);
        let group_item = self
            .build_group_item_from_row(&group, members_count, tags, settings.as_ref())
            .await?;

        let admin_map = self.member_repo.list_admin_members(&[group_id]).await?;
        let admin_ids = admin_map.get(&group_id).cloned().unwrap_or_default();
        let report_stats = self.report_repo.count_group_reports(&[group_id]).await?;
        let (reports_count, last_reported_at) =
            report_stats.get(&group_id).cloned().unwrap_or((0, None));

        let mut user_ids = HashSet::new();
        user_ids.insert(group.owner_id);
        for admin_id in &admin_ids {
            user_ids.insert(*admin_id);
        }

        let user_info_map = self
            .build_user_infos(&user_ids.into_iter().collect::<Vec<_>>())
            .await?;
        let stats = AdminGroupStats {
            messages_count: self.group_repo.get_message_count(group_id).await?,
            active_members_today: 0,
            reports_count: if reports_count > 0 {
                Some(reports_count as i32)
            } else {
                None
            },
            last_reported_at: last_reported_at.map(|ts| ts.to_rfc3339()),
        };

        let context = AdminGroupContext {
            group: group_item,
            owner_id: group.owner_id,
            admin_ids,
            stats,
        };

        Ok(self.compose_admin_group_item(context, &user_info_map))
    }

    pub async fn admin_dismiss_group(
        &self,
        admin_user_id: i64,
        req: &AdminDismissGroupRequest,
    ) -> Result<()> {
        let group_id = Validator::parse_group_id(&req.group_id)?;
        self.group_repo.dismiss(group_id).await?;
        self.event_bus.publish(AppEvent::GroupDismissed {
            group_id,
            dismissed_by: admin_user_id,
            dismissed_by_owner: false,
        });
        Ok(())
    }
}
