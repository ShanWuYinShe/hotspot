use redis::Client as RedisClient;
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreateGroupRequest, UpdateGroupRequest, JoinGroupRequest, LeaveGroupRequest,
            InviteMemberRequest, RemoveMemberRequest, MuteMemberRequest, UnmuteMemberRequest,
            BanMemberRequest, SetMemberRoleRequest, TransferOwnershipRequest,
            SearchGroupByNameRequest, SearchGroupByLocationRequest,
        },
        response::{GroupResponse, GroupMemberResponse, PaginatedResponse},
    },
    error::{AppError, Result},
    models::{
        group::Group,
        group_member::GroupMember,
    },
    repositories::{
        group_repo::GroupRepository,
        group_member_repo::GroupMemberRepository,
    },
    utils::password::PasswordUtil,
};

pub struct GroupService<'a> {
    db: &'a PgPool,
    #[allow(dead_code)]
    redis: &'a RedisClient,
}

impl<'a> GroupService<'a> {
    pub fn new(db: &'a PgPool, redis: &'a RedisClient) -> Self {
        Self { db, redis }
    }

    /// 创建群组
    pub async fn create_group(&self, owner_id: i64, req: &CreateGroupRequest) -> Result<String> {
        let group_repo = GroupRepository::new(self.db);
        let member_repo = GroupMemberRepository::new(self.db);

        // 验证坐标
        if let (Some(lat), Some(lon)) = (req.latitude, req.longitude) {
            crate::utils::validation::Validator::validate_coordinates(lat, lon)?;
        }

        // 处理密码（私有群组）
        let password_hash = if req.group_type == "private" {
            if let Some(password) = &req.password {
                Some(PasswordUtil::hash_client_sha256(password)?)
            } else {
                return Err(AppError::BadRequest("私有群组必须设置密码".to_string()));
            }
        } else {
            None
        };

        let group_id = Uuid::new_v4();
        let group = Group {
            group_id,
            name: req.name.clone(),
            description: req.description.clone(),
            group_type: req.group_type.clone(),
            owner_id,
            avatar_url: req.avatar_url.clone(),
            latitude: req.latitude,
            longitude: req.longitude,
            max_members: req.max_members.unwrap_or(500),
            password_hash,
            created_at: chrono::Utc::now().naive_utc(),
            deleted_at: None,
        };

        group_repo.create(&group).await?;

        // 自动添加创建者为群主
        let owner_member = GroupMember {
            id: 0,
            group_id,
            user_id: owner_id,
            role: "owner".to_string(),
            nickname: None,
            status: "active".to_string(),
            joined_at: chrono::Utc::now().naive_utc(),
            left_at: None,
            muted_until: None,
        };
        member_repo.create(&owner_member).await?;

        Ok(group_id.to_string())
    }

    /// 更新群组信息
    pub async fn update_group(
        &self,
        user_id: i64,
        group_id: Uuid,
        req: &UpdateGroupRequest,
    ) -> Result<()> {
        let group_repo = GroupRepository::new(self.db);
        let member_repo = GroupMemberRepository::new(self.db);

        // 检查权限（群主或管理员）
        let role = member_repo.get_member_role(group_id, user_id).await?;
        if role.as_deref() != Some("owner") && role.as_deref() != Some("admin") {
            return Err(AppError::Forbidden("无权限修改群组信息".to_string()));
        }

        let mut group = group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;

        // 更新字段
        if let Some(name) = &req.name {
            group.name = name.clone();
        }
        if let Some(description) = &req.description {
            group.description = description.clone();
        }
        if let Some(avatar_url) = &req.avatar_url {
            group.avatar_url = avatar_url.clone();
        }
        if let Some(max_members) = req.max_members {
            group.max_members = max_members;
        }

        group_repo.update(&group).await?;

        Ok(())
    }

    /// 解散群组
    pub async fn delete_group(&self, user_id: i64, group_id: Uuid) -> Result<()> {
        let group_repo = GroupRepository::new(self.db);

        let group = group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;

        // 只有群主可以解散群组
        if group.owner_id != user_id {
            return Err(AppError::Forbidden("只有群主可以解散群组".to_string()));
        }

        group_repo.soft_delete(group_id).await?;

        Ok(())
    }

    /// 搜索群组（按名称）
    pub async fn search_by_name(
        &self,
        req: &SearchGroupByNameRequest,
    ) -> Result<PaginatedResponse<GroupResponse>> {
        let group_repo = GroupRepository::new(self.db);

        let limit = req.limit.unwrap_or(20).min(100);
        let offset = req.cursor.unwrap_or(0);

        let groups = group_repo.search_by_name(&req.name, limit + 1, offset).await?;

        let has_more = groups.len() > limit as usize;
        let items: Vec<GroupResponse> = groups
            .into_iter()
            .take(limit as usize)
            .map(|g| GroupResponse {
                id: g.group_id.to_string(),
                name: g.name,
                description: g.description,
                group_type: g.group_type,
                owner_id: g.owner_id.to_string(),
                avatar_url: g.avatar_url,
                member_count: 0, // TODO: 查询成员数
                latitude: g.latitude,
                longitude: g.longitude,
                created_at: g.created_at.and_utc().timestamp(),
            })
            .collect();

        let next_cursor = if has_more {
            Some(offset + limit)
        } else {
            None
        };

        Ok(PaginatedResponse {
            items,
            has_more,
            next_cursor,
        })
    }

    /// 搜索群组（按位置）
    pub async fn search_by_location(
        &self,
        req: &SearchGroupByLocationRequest,
    ) -> Result<PaginatedResponse<GroupResponse>> {
        let group_repo = GroupRepository::new(self.db);

        crate::utils::validation::Validator::validate_coordinates(req.latitude, req.longitude)?;
        crate::utils::validation::Validator::validate_radius(req.radius_meters)?;

        let limit = req.limit.unwrap_or(20).min(100);
        let offset = req.cursor.unwrap_or(0);

        let groups = group_repo
            .search_by_location(
                req.latitude,
                req.longitude,
                req.radius_meters,
                limit + 1,
                offset,
            )
            .await?;

        let has_more = groups.len() > limit as usize;
        let items: Vec<GroupResponse> = groups
            .into_iter()
            .take(limit as usize)
            .map(|(g, _distance)| GroupResponse {
                id: g.group_id.to_string(),
                name: g.name,
                description: g.description,
                group_type: g.group_type,
                owner_id: g.owner_id.to_string(),
                avatar_url: g.avatar_url,
                member_count: 0, // TODO: 查询成员数
                latitude: g.latitude,
                longitude: g.longitude,
                created_at: g.created_at.and_utc().timestamp(),
            })
            .collect();

        let next_cursor = if has_more {
            Some(offset + limit)
        } else {
            None
        };

        Ok(PaginatedResponse {
            items,
            has_more,
            next_cursor,
        })
    }

    /// 加入群组
    pub async fn join_group(&self, user_id: i64, req: &JoinGroupRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        let group_repo = GroupRepository::new(self.db);
        let member_repo = GroupMemberRepository::new(self.db);

        let group = group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;

        // 检查是否已经是成员
        if member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::BadRequest("已经是群成员".to_string()));
        }

        // 检查群组类型和密码
        if group.group_type == "private" {
            if let Some(password) = &req.password {
                let hashed = PasswordUtil::hash_client_sha256(password)?;
                if group.password_hash.as_ref() != Some(&hashed) {
                    return Err(AppError::Unauthorized("群组密码错误".to_string()));
                }
            } else {
                return Err(AppError::BadRequest("私有群组需要密码".to_string()));
            }
        }

        // 检查人数限制
        let member_count = group_repo.get_member_count(group_id).await?;
        if member_count >= group.max_members as i64 {
            return Err(AppError::BadRequest("群组已满".to_string()));
        }

        // 添加成员
        let member = GroupMember {
            id: 0,
            group_id,
            user_id,
            role: "member".to_string(),
            nickname: None,
            status: "active".to_string(),
            joined_at: chrono::Utc::now().naive_utc(),
            left_at: None,
            muted_until: None,
        };
        member_repo.create(&member).await?;

        Ok(())
    }

    /// 退出群组
    pub async fn leave_group(&self, user_id: i64, req: &LeaveGroupRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        let group_repo = GroupRepository::new(self.db);
        let member_repo = GroupMemberRepository::new(self.db);

        let group = group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;

        // 群主不能直接退出，需要先转让群主
        if group.owner_id == user_id {
            return Err(AppError::BadRequest("群主需要先转让群主身份才能退出".to_string()));
        }

        member_repo.leave_group(group_id, user_id).await?;

        Ok(())
    }

    /// 移除成员
    pub async fn remove_member(&self, operator_id: i64, req: &RemoveMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        let target_user_id = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;

        let member_repo = GroupMemberRepository::new(self.db);

        // 检查操作者权限
        let operator_role = member_repo.get_member_role(group_id, operator_id).await?;
        if operator_role.as_deref() != Some("owner") && operator_role.as_deref() != Some("admin") {
            return Err(AppError::Forbidden("无权限移除成员".to_string()));
        }

        // 检查目标用户角色（管理员不能移除群主）
        let target_role = member_repo.get_member_role(group_id, target_user_id).await?;
        if target_role.as_deref() == Some("owner") {
            return Err(AppError::Forbidden("不能移除群主".to_string()));
        }

        member_repo.remove_member(group_id, target_user_id).await?;

        Ok(())
    }

    /// 禁言成员
    pub async fn mute_member(&self, operator_id: i64, req: &MuteMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        let target_user_id = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;

        let member_repo = GroupMemberRepository::new(self.db);

        // 检查操作者权限
        let operator_role = member_repo.get_member_role(group_id, operator_id).await?;
        if operator_role.as_deref() != Some("owner") && operator_role.as_deref() != Some("admin") {
            return Err(AppError::Forbidden("无权限禁言成员".to_string()));
        }

        let muted_until = chrono::Utc::now().naive_utc()
            + chrono::Duration::seconds(req.duration_seconds);

        member_repo.mute_member(group_id, target_user_id, muted_until).await?;

        Ok(())
    }

    /// 解除禁言
    pub async fn unmute_member(&self, operator_id: i64, req: &UnmuteMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        let target_user_id = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;

        let member_repo = GroupMemberRepository::new(self.db);

        // 检查操作者权限
        let operator_role = member_repo.get_member_role(group_id, operator_id).await?;
        if operator_role.as_deref() != Some("owner") && operator_role.as_deref() != Some("admin") {
            return Err(AppError::Forbidden("无权限解除禁言".to_string()));
        }

        member_repo.unmute_member(group_id, target_user_id).await?;

        Ok(())
    }

    /// 设置成员角色
    pub async fn set_member_role(&self, operator_id: i64, req: &SetMemberRoleRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        let target_user_id = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;

        let member_repo = GroupMemberRepository::new(self.db);

        // 只有群主可以设置角色
        let operator_role = member_repo.get_member_role(group_id, operator_id).await?;
        if operator_role.as_deref() != Some("owner") {
            return Err(AppError::Forbidden("只有群主可以设置成员角色".to_string()));
        }

        // 验证角色
        if req.role != "admin" && req.role != "member" {
            return Err(AppError::BadRequest("无效的角色".to_string()));
        }

        member_repo.update_role(group_id, target_user_id, &req.role).await?;

        Ok(())
    }

    /// 转让群主
    pub async fn transfer_ownership(
        &self,
        owner_id: i64,
        req: &TransferOwnershipRequest,
    ) -> Result<()> {
        let group_id = Uuid::parse_str(&req.group_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        let new_owner_id = req.new_owner_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;

        let group_repo = GroupRepository::new(self.db);
        let member_repo = GroupMemberRepository::new(self.db);

        let group = group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;

        // 只有群主可以转让
        if group.owner_id != owner_id {
            return Err(AppError::Forbidden("只有群主可以转让群主身份".to_string()));
        }

        // 检查新群主是否为成员
        if !member_repo.is_member(group_id, new_owner_id).await? {
            return Err(AppError::BadRequest("新群主必须是群成员".to_string()));
        }

        // 更新群组owner_id
        group_repo.transfer_ownership(group_id, new_owner_id).await?;

        // 更新成员角色
        member_repo.update_role(group_id, owner_id, "member").await?;
        member_repo.update_role(group_id, new_owner_id, "owner").await?;

        Ok(())
    }

    /// 获取群组成员列表
    pub async fn list_members(
        &self,
        group_id: Uuid,
        limit: i64,
        offset: i64,
    ) -> Result<PaginatedResponse<GroupMemberResponse>> {
        let member_repo = GroupMemberRepository::new(self.db);

        let members = member_repo.list_by_group(group_id, limit + 1, offset).await?;

        let has_more = members.len() > limit as usize;
        let items: Vec<GroupMemberResponse> = members
            .into_iter()
            .take(limit as usize)
            .map(|m| GroupMemberResponse {
                user_id: m.user_id.to_string(),
                role: m.role,
                nickname: m.nickname,
                joined_at: m.joined_at.and_utc().timestamp(),
            })
            .collect();

        let next_cursor = if has_more {
            Some(offset + limit)
        } else {
            None
        };

        Ok(PaginatedResponse {
            items,
            has_more,
            next_cursor,
        })
    }

    // TODO: 实现以下功能
    // - invite_member (邀请成员)
    // - ban_member (封禁成员)
    // - list_my_groups (我的群组列表)
}
