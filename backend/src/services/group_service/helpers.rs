use super::GroupService;
use crate::{
    dto::response::{
        AdminGroupItem, AdminGroupStats, GroupInfo, GroupItem, GroupMemberItem, RelativeLocation,
        UserInfo,
    },
    error::{AppError, Result},
    repositories::group::member_repo::GroupMemberDetailRow,
    repositories::group::settings_repo::GroupSettingsRow,
    repositories::group_repo::{AdminGroupSearchRow, GroupRow, GroupSearchRow},
    types::enums::{GroupJoinMethod, GroupRole},
};
use std::collections::HashMap;

impl GroupService {
    /// 检查用户是否是群成员，返回角色
    pub(super) async fn require_member(&self, group_id: i64, user_id: i64) -> Result<GroupRole> {
        self.member_repo
            .get_member_role(group_id, user_id)
            .await?
            .ok_or_else(|| AppError::NotGroupMember("你不是该群成员".to_string()))
    }

    /// 检查用户是否是群管理员或群主
    pub(super) async fn require_admin_or_owner(
        &self,
        group_id: i64,
        user_id: i64,
    ) -> Result<GroupRole> {
        let role = self.require_member(group_id, user_id).await?;
        if role != GroupRole::Admin && role != GroupRole::Owner {
            return Err(AppError::Forbidden("需要管理员或群主权限".to_string()));
        }
        Ok(role)
    }

    /// 群管理统一鉴权：群主/群管理员可操作；系统管理员需具备对应 RBAC 权限。
    pub(super) async fn require_group_admin_or_system_permission(
        &self,
        group_id: i64,
        user_id: i64,
        permission_key: &str,
    ) -> Result<()> {
        if !self.group_repo.exists(group_id).await? {
            return Err(AppError::GroupNotFound("群组不存在".to_string()));
        }

        if self.require_admin_or_owner(group_id, user_id).await.is_ok() {
            return Ok(());
        }

        if !self.admin_rbac_repo.is_admin(user_id).await? {
            return Err(AppError::Forbidden("需要管理员或群主权限".to_string()));
        }

        if self
            .admin_rbac_repo
            .user_has_permission(user_id, permission_key)
            .await?
        {
            return Ok(());
        }

        Err(AppError::AdminPermissionDenied(format!(
            "缺少权限: {}",
            permission_key
        )))
    }

    /// 检查用户是否是群主
    pub(super) async fn require_owner(&self, group_id: i64, user_id: i64) -> Result<GroupRole> {
        let role = self.require_member(group_id, user_id).await?;
        if role != GroupRole::Owner {
            return Err(AppError::Forbidden("需要群主权限".to_string()));
        }
        Ok(role)
    }

    pub(super) async fn build_user_info(&self, user_id: i64) -> Result<UserInfo> {
        let user = self.user_repo.find_by_id(user_id).await?;
        if let Some(u) = user {
            let avatar_url = self
                .media_service
                .sign_url_if_path(u.avatar_url.as_deref())
                .await;
            Ok(UserInfo::new(
                u.user_id,
                u.nickname
                    .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                avatar_url,
            ))
        } else {
            Ok(UserInfo::new(
                user_id,
                crate::constants::UNKNOWN_USER_NICKNAME,
                None,
            ))
        }
    }

    pub(super) fn join_method_to_string(join_method: GroupJoinMethod) -> String {
        join_method.as_str().to_string()
    }

    pub(super) fn join_method_from_str(join_method: &str) -> GroupJoinMethod {
        match join_method {
            "open" => GroupJoinMethod::Open,
            "approval" => GroupJoinMethod::Approval,
            "invite_only" => GroupJoinMethod::InviteOnly,
            _ => GroupJoinMethod::Open,
        }
    }

    pub(super) fn build_location(
        latitude: Option<f64>,
        longitude: Option<f64>,
        location_name: Option<&str>,
        distance: Option<f64>,
        bearing_degrees: Option<f64>,
    ) -> RelativeLocation {
        match (latitude, longitude, location_name) {
            (Some(lat), Some(lon), Some(name)) => {
                if let Some(d) = distance {
                    RelativeLocation::with_distance(
                        lat,
                        lon,
                        name,
                        d,
                        bearing_degrees.unwrap_or(0.0),
                    )
                } else {
                    RelativeLocation::without_distance(lat, lon, name)
                }
            }
            _ => RelativeLocation::without_distance(0.0, 0.0, "未知"),
        }
    }

    pub(super) async fn map_search_rows(
        &self,
        groups: Vec<GroupSearchRow>,
    ) -> Result<Vec<GroupItem>> {
        use std::collections::HashMap;

        let group_ids: Vec<i64> = groups.iter().map(|g| g.group_id).collect();
        let tags_map: HashMap<i64, Vec<String>> = self
            .tag_repo
            .get_group_tags_batch(&group_ids)
            .await
            .unwrap_or_default();
        let settings_map = self.settings_repo.find_by_groups(&group_ids).await?;

        let mut items = Vec::with_capacity(groups.len());
        for g in groups {
            let tags = tags_map.get(&g.group_id).cloned();
            let settings = settings_map.get(&g.group_id);
            let has_password = settings.and_then(|s| s.password_hash.as_ref()).is_some();
            let location = Self::build_location(
                g.latitude,
                g.longitude,
                g.location_name.as_deref(),
                g.distance,
                g.bearing_degrees,
            );
            items.push(GroupItem {
                group: GroupInfo {
                    group_id: g.group_id.to_string(),
                    name: g.name,
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(g.avatar_url.as_deref())
                        .await,
                    members_count: g.member_count,
                },
                description: g.description,
                tags,
                has_password,
                location,
                last_activity_at: None,
                created_at: None,
                owner_id: None,
                my_state: None,
                friends_in_group: None,
                join_method: None,
                is_searchable: None,
                max_members: None,
            });
        }
        Ok(items)
    }

    pub(super) async fn map_member_detail(
        &self,
        viewer_id: Option<i64>,
        can_view_admin_fields: bool,
        m: GroupMemberDetailRow,
    ) -> Result<GroupMemberItem> {
        let nickname = m
            .user_nickname
            .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string());
        let avatar_url = self
            .media_service
            .sign_url_if_path(m.avatar_url.as_deref())
            .await;
        let can_view_muted = can_view_admin_fields || viewer_id == Some(m.user_id);
        Ok(GroupMemberItem {
            user: UserInfo::new(m.user_id, nickname, avatar_url),
            role: m.role,
            joined_at: m.joined_at.to_rfc3339(),
            muted_until: if can_view_muted {
                m.muted_until.map(|t| t.to_rfc3339())
            } else {
                None
            },
            presence_status: None,
            last_online: None,
        })
    }

    pub(super) async fn build_user_infos(
        &self,
        user_ids: &[i64],
    ) -> Result<HashMap<i64, UserInfo>> {
        let mut map = HashMap::new();
        if user_ids.is_empty() {
            return Ok(map);
        }

        let profiles = self.user_repo.find_profiles(user_ids).await?;
        for profile in profiles {
            let avatar_url = self
                .media_service
                .sign_url_if_path(profile.avatar_url.as_deref())
                .await;
            let nickname = profile
                .nickname
                .clone()
                .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string());
            map.insert(
                profile.user_id,
                UserInfo::new(profile.user_id, nickname, avatar_url),
            );
        }

        Ok(map)
    }

    pub(super) async fn build_group_item_from_admin_row(
        &self,
        row: &AdminGroupSearchRow,
        tags: Option<Vec<String>>,
        settings: Option<&GroupSettingsRow>,
    ) -> Result<GroupItem> {
        let avatar_url = self
            .media_service
            .sign_url_if_path(row.avatar_url.as_deref())
            .await;

        let location = Self::build_location(
            row.latitude,
            row.longitude,
            row.location_name.as_deref(),
            None,
            None,
        );

        let has_password = settings.and_then(|s| s.password_hash.as_ref()).is_some();
        let join_method = settings.map(|s| Self::join_method_from_str(&s.join_method));
        let is_searchable = settings.map(|s| s.is_searchable);
        let max_members = settings
            .and_then(|s| s.max_members)
            .unwrap_or(row.max_members);

        Ok(GroupItem {
            group: GroupInfo {
                group_id: row.group_id.to_string(),
                name: row.name.clone(),
                avatar_url,
                members_count: row.member_count,
            },
            description: row.description.clone(),
            tags,
            has_password,
            location,
            last_activity_at: None,
            created_at: Some(row.created_at.to_rfc3339()),
            owner_id: Some(row.owner_id.to_string()),
            my_state: None,
            friends_in_group: None,
            join_method,
            is_searchable,
            max_members: Some(max_members),
        })
    }

    pub(super) async fn build_group_item_from_row(
        &self,
        row: &GroupRow,
        members_count: i32,
        tags: Option<Vec<String>>,
        settings: Option<&GroupSettingsRow>,
    ) -> Result<GroupItem> {
        let avatar_url = self
            .media_service
            .sign_url_if_path(row.avatar_url.as_deref())
            .await;

        let location = Self::build_location(
            row.latitude,
            row.longitude,
            row.location_name.as_deref(),
            None,
            None,
        );
        let has_password = settings.and_then(|s| s.password_hash.as_ref()).is_some();
        let join_method = settings.map(|s| Self::join_method_from_str(&s.join_method));
        let is_searchable = settings.map(|s| s.is_searchable);
        let max_members = settings
            .and_then(|s| s.max_members)
            .unwrap_or(row.max_members);

        Ok(GroupItem {
            group: GroupInfo {
                group_id: row.group_id.to_string(),
                name: row.name.clone(),
                avatar_url,
                members_count,
            },
            description: row.description.clone(),
            tags,
            has_password,
            location,
            last_activity_at: None,
            created_at: Some(row.created_at.to_rfc3339()),
            owner_id: Some(row.owner_id.to_string()),
            my_state: None,
            friends_in_group: None,
            join_method,
            is_searchable,
            max_members: Some(max_members),
        })
    }

    pub(super) fn compose_admin_group_item(
        &self,
        ctx: AdminGroupContext,
        user_map: &HashMap<i64, UserInfo>,
    ) -> AdminGroupItem {
        let owner = user_map.get(&ctx.owner_id).cloned().unwrap_or_else(|| {
            UserInfo::new(ctx.owner_id, crate::constants::UNKNOWN_USER_NICKNAME, None)
        });

        let admin_ids = if ctx.admin_ids.is_empty() {
            vec![ctx.owner_id]
        } else {
            ctx.admin_ids
        };

        let mut admins = Vec::with_capacity(admin_ids.len());
        for admin_id in admin_ids {
            let info = user_map.get(&admin_id).cloned().unwrap_or_else(|| {
                UserInfo::new(admin_id, crate::constants::UNKNOWN_USER_NICKNAME, None)
            });
            admins.push(info);
        }

        AdminGroupItem {
            group: ctx.group,
            owner,
            admins,
            stats: ctx.stats,
        }
    }
}

pub(super) struct AdminGroupContext {
    pub group: GroupItem,
    pub owner_id: i64,
    pub admin_ids: Vec<i64>,
    pub stats: AdminGroupStats,
}
