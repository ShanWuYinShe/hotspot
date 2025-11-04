use redis::aio::ConnectionManager;
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreateGroupRequest, UpdateGroupRequest, JoinGroupRequest, LeaveGroupRequest,
            RemoveMemberRequest, MuteMemberRequest, UnmuteMemberRequest,
            SetMemberRoleRequest, TransferOwnershipRequest,
            SearchGroupByNameRequest, SearchGroupByLocationRequest,
        },
        response::{GroupResponse, MemberResponse, PaginatedResponse},
    },
    error::{AppError, Result},
    models::{
        group::Group,
        group_member::GroupMember,
    },
    repositories::{
        group_repo::GroupRepository,
        group_member_repo::GroupMemberRepository,
        room_location_history_repo::RoomLocationHistoryRepository,
    },
    types::enums::{RoomType, MemberRole, InviteStatus},
    utils::{password::PasswordUtil, geo::GeoUtil},
};

pub struct GroupService {
    group_repo: GroupRepository,
    member_repo: GroupMemberRepository,
    room_location_history_repo: RoomLocationHistoryRepository,
    #[allow(dead_code)]
    redis: ConnectionManager,
}

impl GroupService {
    pub fn new(pool: &PgPool, redis: &ConnectionManager) -> Self {
        Self {
            group_repo: GroupRepository::new(pool.clone()),
            member_repo: GroupMemberRepository::new(pool.clone()),
            room_location_history_repo: RoomLocationHistoryRepository::new(pool.clone()),
            redis: redis.clone(),
        }
    }

    /// 创建群组
    pub async fn create_group(&self, owner_id: i64, req: &CreateGroupRequest) -> Result<String> {
        // 验证坐标（必填）
        GeoUtil::validate_coordinates(req.latitude, req.longitude)?;

        // 处理密码（私有群组）
        let password_hash = if req.group_type == "private_group" {
            if let Some(password_hash) = &req.password_hash {
                Some(password_hash.clone())
            } else {
                return Err(AppError::BadRequest("私有群组必须设置密码".to_string()));
            }
        } else {
            None
        };

        let group_id = Uuid::new_v4();
        let group_type = RoomType::from_str(&req.group_type)
            .map_err(|e| AppError::BadRequest(e))?;
        
        let group = Group {
            id: None,
            group_id,
            group_type,
            name: req.name.clone(),
            description: req.description.clone(),
            avatar_url: req.avatar_url.clone(),
            owner_id,
            password_hash,
            max_members: Some(req.max_members.unwrap_or(500)),
            created_at: chrono::Utc::now(),
            last_active_at: None,
            is_deleted: false,
            deleted_at: None,
        };

        self.group_repo.create(&group).await?;

        // 插入群组位置历史
        self.room_location_history_repo
            .create(
                group_id,
                req.latitude,
                req.longitude,
                &req.location_name,
            )
            .await?;

        // 自动添加创建者为群主
        let owner_member = GroupMember {
            id: 0, // 自增ID，插入时不使用
            group_id,
            user_id: owner_id,
            role: MemberRole::Owner,
            nickname: None,
            is_banned: false,
            is_muted: false,
            muted_until: None,
            joined_at: chrono::Utc::now(),
            unread_count: 0,
            last_read_at: None,
            is_deleted: false,
            deleted_at: None,
        };
        self.member_repo.create(&owner_member).await?;

        Ok(group_id.to_string())
    }

    /// 更新群组信息
    pub async fn update_group(
        &self,
        user_id: i64,
        group_id: Uuid,
        req: &UpdateGroupRequest,
    ) -> Result<()> {

        // 检查权限（群主或管理员）
        let role = self.member_repo.get_member_role(group_id, user_id).await?;
        if role != Some(MemberRole::Owner) && role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("无权限修改群组信息".to_string()));
        }

        let mut group = self.group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;

        // 更新字段
        if let Some(name) = &req.name {
            group.name = name.clone();
        }
        if let Some(description) = &req.description {
            group.description = Some(description.clone());
        }
        if let Some(avatar_url) = &req.avatar_url {
            group.avatar_url = Some(avatar_url.clone());
        }

        self.group_repo.update(&group).await?;

        Ok(())
    }

    /// 解散群组
    pub async fn delete_group(&self, user_id: i64, group_id: Uuid) -> Result<()> {

        let group = self.group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;

        // 只有群主可以解散群组
        if group.owner_id != user_id {
            return Err(AppError::Forbidden("只有群主可以解散群组".to_string()));
        }

        self.group_repo.soft_delete(group_id).await?;

        Ok(())
    }

    /// 搜索群组（按名称）
    pub async fn search_by_name(
        &self,
        req: &SearchGroupByNameRequest,
    ) -> Result<PaginatedResponse<GroupResponse>> {
        let limit = req.limit.unwrap_or(20).min(100) as i64;

        // 使用游标分页：多取一条判断has_more
        let groups = self.group_repo.search_by_name(&req.name, req.cursor.clone(), limit + 1).await?;

        let has_more = groups.len() > limit as usize;
        let groups_to_return: Vec<Group> = groups.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的group_id（业务ID，即room_id）
        let next_cursor = if has_more {
            groups_to_return.last().map(|g| g.group_id.to_string())
        } else {
            None
        };

        let items: Vec<GroupResponse> = groups_to_return
            .into_iter()
            .map(|g| GroupResponse {
                id: g.group_id.to_string(),
                name: g.name,
                description: g.description,
                group_type: g.group_type.to_string(),
                owner_id: g.owner_id.to_string(),
                owner_name: String::new(), // TODO: 查询owner昵称
                location_name: String::new(), // TODO: 从 room_location_history 查询
                avatar_url: g.avatar_url,
                member_count: 0, // TODO: 查询成员数
                is_password_required: g.password_hash.is_some(),
                created_at: g.created_at,
                is_member: false, // TODO: 检查当前用户是否为成员
            })
            .collect();

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
        GeoUtil::validate_coordinates(req.latitude, req.longitude)?;
        let radius = req.radius.unwrap_or(5000.0).min(50000.0);
        let limit = req.limit.unwrap_or(20).min(100) as i64;

        // 使用 room_location_history_repo 搜索附近群组
        let nearby_rooms = self
            .room_location_history_repo
            .search_nearby_rooms(req.latitude, req.longitude, radius as f64, limit as i64)
            .await?;

        let mut items = Vec::new();
        for (room_id, _distance) in nearby_rooms {
            // 获取群组详情
            if let Some(group) = self.group_repo.find_by_id(room_id).await? {
                // 获取最新位置信息
                if let Some(location) = self
                    .room_location_history_repo
                    .get_latest_by_room_id(room_id)
                    .await?
                {
                    items.push(GroupResponse {
                        id: group.group_id.to_string(),
                        group_type: group.group_type.to_string(),
                        name: group.name,
                        description: group.description,
                        avatar_url: group.avatar_url,
                        owner_id: group.owner_id.to_string(),
                        owner_name: String::new(), // TODO: 查询群主昵称
                        location_name: location.location_name,
                        member_count: 0, // TODO: 查询成员数
                        is_password_required: group.password_hash.is_some(),
                        created_at: group.created_at,
                        is_member: false, // TODO: 检查当前用户是否为成员
                    });
                }
            }
        }

        // 游标分页逻辑（简化版）
        let has_more = items.len() >= limit as usize;
        let next_cursor = if has_more {
            items.last().map(|item| item.id.clone())
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
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;


        let group = self.group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;

        // 检查是否已经是成员
        if self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::BadRequest("已经是群成员".to_string()));
        }

        // 检查群组类型和密码
        if group.group_type == RoomType::PrivateGroup {
            if let Some(password_hash) = &req.password_hash {
                // 客户端已经用SHA256哈希过了，服务端验证bcrypt哈希
                if let Some(stored_hash) = &group.password_hash {
                    if !PasswordUtil::verify_password(password_hash, stored_hash)? {
                        return Err(AppError::Unauthorized("群组密码错误".to_string()));
                    }
                } else {
                    return Err(AppError::Unauthorized("该群组需要密码".to_string()));
                }
            } else if group.password_hash.is_some() {
                return Err(AppError::BadRequest("私有群组需要密码".to_string()));
            }
        }

        // 检查人数限制
        let member_count = self.group_repo.get_member_count(group_id).await?;
        if let Some(max_members) = group.max_members {
            if member_count >= max_members as i64 {
                return Err(AppError::BadRequest("群组已满".to_string()));
            }
        }

        // 添加成员
        let member = GroupMember {
            id: 0, // 自增ID
            group_id,
            user_id,
            role: MemberRole::Member,
            nickname: None,
            is_banned: false,
            is_muted: false,
            muted_until: None,
            joined_at: chrono::Utc::now(),
            unread_count: 0,
            last_read_at: None,
            is_deleted: false,
            deleted_at: None,
        };
        self.member_repo.create(&member).await?;

        Ok(())
    }

    /// 退出群组
    pub async fn leave_group(&self, user_id: i64, req: &LeaveGroupRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;


        let group = self.group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;

        // 群主不能直接退出，需要先转让群主
        if group.owner_id == user_id {
            return Err(AppError::BadRequest("群主需要先转让群主身份才能退出".to_string()));
        }

        self.member_repo.leave_group(group_id, user_id).await?;

        Ok(())
    }

    /// 移除成员
    pub async fn remove_member(&self, operator_id: i64, req: &RemoveMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        let target_user_id = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;


        // 检查操作者权限
        let operator_role = self.member_repo.get_member_role(group_id, operator_id).await?;
        if operator_role != Some(MemberRole::Owner) && operator_role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("无权限移除成员".to_string()));
        }

        // 检查目标用户角色（管理员不能移除群主）
        let target_role = self.member_repo.get_member_role(group_id, target_user_id).await?;
        if target_role == Some(MemberRole::Owner) {
            return Err(AppError::Forbidden("不能移除群主".to_string()));
        }

        self.member_repo.remove_member(group_id, target_user_id).await?;

        Ok(())
    }

    /// 禁言成员
    pub async fn mute_member(&self, operator_id: i64, req: &MuteMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        let target_user_id = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;


        // 检查操作者权限
        let operator_role = self.member_repo.get_member_role(group_id, operator_id).await?;
        if operator_role != Some(MemberRole::Owner) && operator_role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("无权限禁言成员".to_string()));
        }

        let muted_until = chrono::Utc::now()
            + chrono::Duration::seconds(req.duration);

        self.member_repo.mute_member(group_id, target_user_id, muted_until).await?;

        Ok(())
    }

    /// 解除禁言
    pub async fn unmute_member(&self, operator_id: i64, req: &UnmuteMemberRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        let target_user_id = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;


        // 检查操作者权限
        let operator_role = self.member_repo.get_member_role(group_id, operator_id).await?;
        if operator_role != Some(MemberRole::Owner) && operator_role != Some(MemberRole::Admin) {
            return Err(AppError::Forbidden("无权限解除禁言".to_string()));
        }

        self.member_repo.unmute_member(group_id, target_user_id).await?;

        Ok(())
    }

    /// 设置成员角色
    pub async fn set_member_role(&self, operator_id: i64, req: &SetMemberRoleRequest) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        let target_user_id = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;


        // 只有群主可以设置角色
        let operator_role = self.member_repo.get_member_role(group_id, operator_id).await?;
        if operator_role != Some(MemberRole::Owner) {
            return Err(AppError::Forbidden("只有群主可以设置成员角色".to_string()));
        }

        // 验证并转换角色
        let new_role = MemberRole::from_str(&req.role)
            .map_err(|e| AppError::BadRequest(e))?;
        
        if new_role == MemberRole::Owner {
            return Err(AppError::BadRequest("不能直接设置为群主，请使用转移群主功能".to_string()));
        }

        self.member_repo.update_role(group_id, target_user_id, new_role).await?;

        Ok(())
    }

    /// 转让群主
    pub async fn transfer_ownership(
        &self,
        owner_id: i64,
        req: &TransferOwnershipRequest,
    ) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        let new_owner_id = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;


        let group = self.group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;

        // 只有群主可以转让
        if group.owner_id != owner_id {
            return Err(AppError::Forbidden("只有群主可以转让群主身份".to_string()));
        }

        // 检查新群主是否为成员
        if !self.member_repo.is_member(group_id, new_owner_id).await? {
            return Err(AppError::BadRequest("新群主必须是群成员".to_string()));
        }

        // 更新群组owner_id
        self.group_repo.transfer_ownership(group_id, new_owner_id).await?;

        // 更新成员角色
        self.member_repo.update_role(group_id, owner_id, MemberRole::Member).await?;
        self.member_repo.update_role(group_id, new_owner_id, MemberRole::Owner).await?;

        Ok(())
    }

    /// 获取群组成员列表
    pub async fn list_members(
        &self,
        group_id: Uuid,
        cursor: Option<String>,
        limit: i64,
    ) -> Result<PaginatedResponse<MemberResponse>> {
        // 使用游标分页：多取一条判断has_more
        let members = self.member_repo.list_by_group(group_id, cursor, limit + 1).await?;

        let has_more = members.len() > limit as usize;
        let members_to_return: Vec<GroupMember> = members.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的user_id（业务ID）
        let next_cursor = if has_more {
            members_to_return.last().map(|m| m.user_id.to_string())
        } else {
            None
        };

        let items: Vec<MemberResponse> = members_to_return
            .into_iter()
            .map(|m| MemberResponse {
                user_id: m.user_id.to_string(),
                nickname: m.nickname.unwrap_or_else(|| String::from("未知用户")),
                avatar_url: None, // TODO: 查询用户头像
                role: m.role.to_string(),
                is_online: false, // TODO: 查询在线状态
                last_seen: None, // TODO: 查询最后在线时间
                joined_at: m.joined_at,
                is_muted: m.is_muted,
                muted_until: m.muted_until,
                is_banned: m.is_banned,
            })
            .collect();

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

    /// 按ID获取群组详情
    pub async fn search_by_id(
        &self,
        req: &crate::dto::request::SearchGroupByIdRequest,
    ) -> Result<crate::dto::response::GroupDetailResponse> {
        use crate::dto::response::GroupDetailResponse;
        
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        
        let room = self.group_repo
            .find_by_id(group_id)
            .await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;
        
        // 获取群组成员数量
        let member_count = self.group_repo.get_member_count(group_id).await? as i32;
        
        Ok(GroupDetailResponse {
            id: room.group_id.to_string(),
            group_type: room.group_type.as_str().to_string(),
            name: room.name.clone(),
            description: room.description.clone(),
            avatar_url: room.avatar_url.clone(),
            owner_id: room.owner_id.to_string(),
            owner_name: String::new(), // TODO: 查询用户昵称
            latitude: room.latitude,
            longitude: room.longitude,
            location_name: room.location_name.clone(),
            member_count,
            max_members: room.max_members,
            is_password_required: room.password_hash.is_some(),
            created_at: room.created_at,
            last_active_at: room.last_active_at.unwrap_or(room.created_at),
            my_role: None, // TODO: 查询当前用户角色
            is_member: false, // TODO: 查询是否为成员
        })
    }

    /// 更新群组资料
    pub async fn update_profile(
        &self,
        user_id: i64,
        req: &crate::dto::request::UpdateGroupProfileRequest,
    ) -> Result<crate::dto::response::UpdateGroupProfileResponse> {
        use crate::dto::response::UpdateGroupProfileResponse;
        
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        
        // 检查权限（仅群主或管理员可修改）
        let member = self.member_repo.find_by_group_and_user(group_id, user_id).await?
            .ok_or_else(|| AppError::Forbidden("您不是群组成员".to_string()))?;
        
        if member.role != MemberRole::Owner && member.role != MemberRole::Admin {
            return Err(AppError::Forbidden("您没有权限修改群组资料".to_string()));
        }
        
        // 获取当前群组信息
        let mut group = self.group_repo.find_by_id(group_id).await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;
        
        // 更新字段
        if let Some(name) = &req.name {
            group.name = name.clone();
        }
        if let Some(description) = &req.description {
            group.description = Some(description.clone());
        }
        if let Some(avatar_url) = &req.avatar_url {
            group.avatar_url = Some(avatar_url.clone());
        }
        
        // 保存更新
        self.group_repo.update(&group).await?;
        
        Ok(UpdateGroupProfileResponse {
            success: true,
            message: "群组资料更新成功".to_string(),
        })
    }

    /// 更新群组位置
    pub async fn update_location(
        &self,
        user_id: i64,
        req: &crate::dto::request::UpdateGroupLocationRequest,
    ) -> Result<crate::dto::response::UpdateGroupLocationResponse> {
        use crate::dto::response::UpdateGroupLocationResponse;
        
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        
        // 检查权限（仅群主或管理员可修改）
        let member = self.member_repo.find_by_group_and_user(group_id, user_id).await?
            .ok_or_else(|| AppError::Forbidden("您不是群组成员".to_string()))?;
        
        if member.role != MemberRole::Owner && member.role != MemberRole::Admin {
            return Err(AppError::Forbidden("您没有权限修改群组位置".to_string()));
        }
        
        // 获取当前群组信息
        let mut group = self.group_repo.find_by_id(group_id).await?
            .ok_or_else(|| AppError::NotFound("群组不存在".to_string()))?;
        
        // 更新位置信息
        group.latitude = req.latitude;
        group.longitude = req.longitude;
        group.location_name = req.location_name.clone();
        
        // 保存更新
        self.group_repo.update(&group).await?;
        
        Ok(UpdateGroupLocationResponse {
            success: true,
            message: "群组位置更新成功".to_string(),
        })
    }

    /// 更新群组隐私设置
    pub async fn update_privacy(
        &self,
        user_id: i64,
        req: &crate::dto::request::UpdatePrivacyRequest,
    ) -> Result<crate::dto::response::UpdatePrivacyResponse> {
        use crate::dto::response::UpdatePrivacyResponse;
        
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        
        // 检查权限（仅群主可修改）
        let member = self.member_repo.find_by_group_and_user(group_id, user_id).await?
            .ok_or_else(|| AppError::Forbidden("您不是群组成员".to_string()))?;
        
        if member.role != MemberRole::Owner {
            return Err(AppError::Forbidden("仅群主可修改隐私设置".to_string()));
        }
        
        // 更新隐私设置
        // TODO: 实现隐私设置更新逻辑
        
        Ok(UpdatePrivacyResponse {
            success: true,
            message: "隐私设置更新成功".to_string(),
        })
    }

    /// 更新群组通知设置
    pub async fn update_notifications(
        &self,
        user_id: i64,
        req: &crate::dto::request::UpdateNotificationsRequest,
    ) -> Result<crate::dto::response::UpdateNotificationsResponse> {
        use crate::dto::response::UpdateNotificationsResponse;
        
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        
        // 检查是否为成员
        let _member = self.member_repo.find_by_group_and_user(group_id, user_id).await?
            .ok_or_else(|| AppError::Forbidden("您不是群组成员".to_string()))?;
        
        // 更新通知设置
        // TODO: 实现通知设置更新逻辑
        
        Ok(UpdateNotificationsResponse {
            success: true,
            message: "通知设置更新成功".to_string(),
        })
    }

    /// 封禁/解封成员
    pub async fn ban_member(
        &self,
        operator_id: i64,
        req: &crate::dto::request::BanMemberRequest,
    ) -> Result<()> {
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        
        let target_user_id = req.user_id.parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;
        
        // 检查权限（仅群主或管理员可操作）
        let operator_member = self.member_repo.find_by_group_and_user(group_id, operator_id).await?
            .ok_or_else(|| AppError::Forbidden("您不是群组成员".to_string()))?;
        
        if operator_member.role != MemberRole::Owner && operator_member.role != MemberRole::Admin {
            return Err(AppError::Forbidden("您没有权限封禁成员".to_string()));
        }
        
        // 检查目标用户是否为成员
        let target_member = self.member_repo.find_by_group_and_user(group_id, target_user_id).await?
            .ok_or_else(|| AppError::NotFound("目标用户不是群组成员".to_string()))?;
        
        // 不能封禁群主
        if target_member.role == MemberRole::Owner {
            return Err(AppError::Forbidden("不能封禁群主".to_string()));
        }
        
        // 管理员不能封禁管理员
        if operator_member.role == MemberRole::Admin && target_member.role == MemberRole::Admin {
            return Err(AppError::Forbidden("管理员不能封禁其他管理员".to_string()));
        }
        
        // 执行封禁/解封操作
        match req.action.as_str() {
            "ban" => {
                // TODO: 实现封禁逻辑
                Ok(())
            }
            "unban" => {
                // TODO: 实现解封逻辑
                Ok(())
            }
            _ => Err(AppError::BadRequest("无效的操作类型".to_string())),
        }
    }

    /// 邀请成员加入群组
    pub async fn invite_members(
        &self,
        inviter_id: i64,
        req: &crate::dto::request::InviteRequest,
    ) -> Result<crate::dto::response::InviteMembersResponse> {
        use crate::dto::response::InviteMembersResponse;
        
        let group_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        
        // 检查邀请者是否为成员
        let _inviter_member = self.member_repo.find_by_group_and_user(group_id, inviter_id).await?
            .ok_or_else(|| AppError::Forbidden("您不是群组成员".to_string()))?;
        
        // TODO: 实现邀请逻辑
        
        Ok(InviteMembersResponse {
            success: true,
            message: "邀请已发送".to_string(),
            results: vec![],
        })
    }

    /// 接受群组邀请
    pub async fn accept_invite(
        &self,
        user_id: i64,
        req: &crate::dto::request::AcceptInviteRequest,
    ) -> Result<crate::dto::response::AcceptInviteResponse> {
        use crate::dto::response::AcceptInviteResponse;
        
        let invite_id = req.id.clone();
        
        // TODO: 实现接受邀请逻辑
        
        Ok(AcceptInviteResponse {
            success: true,
            message: "已成功加入群组".to_string(),
            group: crate::dto::response::GroupInfo {
                id: String::new(),
                group_type: String::new(),
                name: String::new(),
            },
        })
    }

    /// 拒绝群组邀请
    pub async fn reject_invite(
        &self,
        user_id: i64,
        req: &crate::dto::request::RejectInviteRequest,
    ) -> Result<crate::dto::response::RejectInviteResponse> {
        use crate::dto::response::RejectInviteResponse;
        
        let _invite_id = req.id.clone();
        
        // TODO: 实现拒绝邀请逻辑
        
        Ok(RejectInviteResponse {
            success: true,
            message: "已拒绝邀请".to_string(),
        })
    }

    /// 获取我的群组邀请列表
    pub async fn list_invites(
        &self,
        user_id: i64,
        req: &crate::dto::request::ListInvitesRequest,
    ) -> Result<crate::dto::response::PaginatedResponse<crate::dto::response::GroupInviteResponse>> {
        use crate::dto::response::{PaginatedResponse, GroupInviteResponse};
        
        let _status = req.status.clone();
        let _limit = req.limit.unwrap_or(20).min(100);
        
        // TODO: 实现列表查询逻辑
        
        Ok(PaginatedResponse {
            items: vec![],
            has_more: false,
            next_cursor: None,
        })
    }

    /// 申请加入群组（私密群）
    pub async fn apply_to_group(
        &self,
        user_id: i64,
        req: &crate::dto::request::ApplyToGroupRequest,
    ) -> Result<crate::dto::response::ApplyToGroupResponse> {
        use crate::dto::response::ApplyToGroupResponse;
        
        let _group_id = uuid::Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
        
        // TODO: 实现申请逻辑
        
        Ok(ApplyToGroupResponse {
            success: true,
            message: "申请已提交".to_string(),
            application_id: uuid::Uuid::new_v4().to_string(),
        })
    }

    /// 审核入群申请
    pub async fn review_application(
        &self,
        reviewer_id: i64,
        req: &crate::dto::request::ReviewApplicationRequest,
    ) -> Result<crate::dto::response::ReviewApplicationResponse> {
        use crate::dto::response::ReviewApplicationResponse;
        
        let _application_id = uuid::Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的申请ID".to_string()))?;
        
        // TODO: 实现审核逻辑
        
        Ok(ReviewApplicationResponse {
            success: true,
            message: "审核完成".to_string(),
        })
    }

    /// 获取我的群组列表
    pub async fn list_my_groups(
        &self,
        user_id: i64,
        req: &crate::dto::request::ListGroupsRequest,
    ) -> Result<crate::dto::response::PaginatedResponse<crate::dto::response::MyGroupResponse>> {
        use crate::dto::response::{PaginatedResponse, MyGroupResponse};
        
        let _group_type = req.group_type.clone();
        let _limit = req.limit.unwrap_or(20).min(100);
        
        // TODO: 实现我的群组列表逻辑
        
        Ok(PaginatedResponse {
            items: vec![],
            has_more: false,
            next_cursor: None,
        })
    }
}
