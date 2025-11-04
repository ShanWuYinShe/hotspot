use chrono::Utc;
use redis::aio::ConnectionManager;
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreateGroupRequest, JoinRequest, LeaveRequest, StartChatRequest,
            UpdateGroupProfileRequest, UpdatePrivacyRequest,
        },
        response::{MyGroupResponse, PaginatedResponse},
    },
    error::{AppError, Result},
    models::{
        chat_room::ChatRoom,
        group_member::GroupMember,
    },
    repositories::{
        chat_room_repo::ChatRoomRepository, group_member_repo::GroupMemberRepository,
        room_location_history_repo::RoomLocationHistoryRepository,
    },
    types::enums::{MemberRole, RoomType},
    utils::{geo::GeoUtil, password::PasswordUtil},
};

pub struct ChatRoomService {
    room_repo: ChatRoomRepository,
    member_repo: GroupMemberRepository,
    location_history_repo: RoomLocationHistoryRepository,
    redis: ConnectionManager,
}

impl ChatRoomService {
    pub fn new(db: &PgPool, redis: &ConnectionManager) -> Self {
        Self {
            room_repo: ChatRoomRepository::new(db.clone()),
            member_repo: GroupMemberRepository::new(db.clone()),
            location_history_repo: RoomLocationHistoryRepository::new(db.clone()),
            redis: redis.clone(),
        }
    }

    /// 发起私聊（创建 direct 类型的聊天室）
    pub async fn start_chat(&self, user_id: i64, req: &StartChatRequest) -> Result<String> {
        let other_user_id = req
            .user_id
            .parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的用户ID".to_string()))?;

        // 检查是否已存在私聊
        if let Some(existing_room) = self
            .room_repo
            .find_direct_chat(user_id, other_user_id)
            .await?
        {
            return Ok(existing_room.room_id.to_string());
        }

        // 创建新的私聊室
        let room_id = Uuid::new_v4();
        let room = ChatRoom {
            id: 0, // 由数据库自动生成
            room_id,
            room_type: RoomType::Direct,
            name: String::new(), // 私聊不需要名称
            description: None,
            avatar_url: None,
            owner_id: user_id,
            password_hash: None,
            max_members: 2,
            created_at: Utc::now(),
            last_message_at: None,
            is_deleted: false,
            deleted_at: None,
        };

        self.room_repo.create(&room).await?;

        // 添加两个成员
        let member1 = GroupMember {
            id: 0,
            group_id: room_id,
            user_id,
            role: MemberRole::Owner,
            nickname: None,
            is_banned: false,
            is_muted: false,
            muted_until: None,
            joined_at: Utc::now(),
            unread_count: 0,
            last_read_at: None,
            is_deleted: false,
            deleted_at: None,
        };
        self.member_repo.add_member(&member1).await?;

        let member2 = GroupMember {
            id: 0,
            group_id: room_id,
            user_id: other_user_id,
            role: MemberRole::Member,
            nickname: None,
            is_banned: false,
            is_muted: false,
            muted_until: None,
            joined_at: Utc::now(),
            unread_count: 0,
            last_read_at: None,
            is_deleted: false,
            deleted_at: None,
        };
        self.member_repo.add_member(&member2).await?;

        Ok(room_id.to_string())
    }

    /// 创建群组（public_group 或 private_group）
    pub async fn create_group(&self, owner_id: i64, req: &CreateGroupRequest) -> Result<String> {
        // 验证坐标
        GeoUtil::validate_coordinates(req.latitude, req.longitude)?;

        // 处理密码（仅公开群可设置密码）
        let password_hash = if let Some(pwd_hash) = &req.password_hash {
            Some(PasswordUtil::hash_password(pwd_hash)?)
        } else {
            None
        };

        let group_id = Uuid::new_v4();
        let group_type = if req.group_type == "public_group" {
            RoomType::PublicGroup
        } else if req.group_type == "private_group" {
            RoomType::PrivateGroup
        } else {
            return Err(AppError::BadRequest("无效的群组类型".to_string()));
        };

        let room = ChatRoom {
            id: 0, // 由数据库自动生成
            room_id: group_id,
            room_type: group_type,
            name: req.name.clone(),
            description: req.description.clone(),
            avatar_url: req.avatar_url.clone(),
            owner_id,
            password_hash,
            max_members: req.max_members.unwrap_or(500),
            created_at: Utc::now(),
            last_message_at: None,
            is_deleted: false,
            deleted_at: None,
        };

        self.room_repo.create(&room).await?;

        // 插入群组位置历史
        self.location_history_repo
            .create(group_id, req.latitude, req.longitude, &req.location_name)
            .await?;

        // 自动添加创建者为群主
        let owner_member = GroupMember {
            id: 0,
            group_id,
            user_id: owner_id,
            role: MemberRole::Owner,
            nickname: None,
            is_banned: false,
            is_muted: false,
            muted_until: None,
            joined_at: Utc::now(),
            unread_count: 0,
            last_read_at: None,
            is_deleted: false,
            deleted_at: None,
        };
        self.member_repo.add_member(&owner_member).await?;

        // 添加初始成员（如果有）
        if let Some(initial_members) = &req.initial_members {
            for member_id_str in initial_members {
                if let Ok(member_id) = member_id_str.parse::<i64>() {
                    if member_id != owner_id {
                        let new_member = GroupMember {
                            id: 0,
                            group_id,
                            user_id: member_id,
                            role: MemberRole::Member,
                            nickname: None,
                            is_banned: false,
                            is_muted: false,
                            muted_until: None,
                            joined_at: Utc::now(),
                            unread_count: 0,
                            last_read_at: None,
                            is_deleted: false,
                            deleted_at: None,
                        };
                        let _ = self.member_repo.add_member(&new_member).await;
                    }
                }
            }
        }

        Ok(group_id.to_string())
    }

    /// 加入公开群组
    pub async fn join_group(&self, user_id: i64, req: &JoinRequest) -> Result<()> {
        let room_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        let room = self
            .room_repo
            .find_by_room_id(room_id)
            .await?
            .ok_or(AppError::NotFound("群组不存在".to_string()))?;

        // 只有公开群可以直接加入
        if room.room_type != RoomType::PublicGroup {
            return Err(AppError::BadRequest("只能加入公开群组".to_string()));
        }

        // 检查密码（如果需要）
        if let Some(hash) = &room.password_hash {
            if let Some(pwd_hash) = &req.password_hash {
                if !PasswordUtil::verify_password(pwd_hash, hash)? {
                    return Err(AppError::Unauthorized("密码错误".to_string()));
                }
            } else {
                return Err(AppError::BadRequest("需要密码".to_string()));
            }
        }

        // 检查是否已经是成员
        if self.member_repo.is_member(room_id, user_id).await? {
            return Err(AppError::Conflict("已经是群成员".to_string()));
        }

        // 添加成员
        let new_member = GroupMember {
            id: 0,
            group_id: room_id,
            user_id,
            role: MemberRole::Member,
            nickname: None,
            is_banned: false,
            is_muted: false,
            muted_until: None,
            joined_at: Utc::now(),
            unread_count: 0,
            last_read_at: None,
            is_deleted: false,
            deleted_at: None,
        };
        self.member_repo.add_member(&new_member).await?;

        Ok(())
    }

    /// 离开群组/私聊
    pub async fn leave_room(&self, user_id: i64, req: &LeaveRequest) -> Result<()> {
        let room_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的房间ID".to_string()))?;

        let room = self
            .room_repo
            .find_by_room_id(room_id)
            .await?
            .ok_or(AppError::NotFound("房间不存在".to_string()))?;

        // 群主不能离开（需要转移所有权或解散）
        if room.owner_id == user_id && room.room_type != RoomType::Direct {
            return Err(AppError::BadRequest(
                "群主不能直接离开，请先转移所有权".to_string(),
            ));
        }

        self.member_repo.remove_member(room_id, user_id).await?;

        Ok(())
    }

    /// 更新群组资料
    pub async fn update_profile(
        &self,
        user_id: i64,
        req: &UpdateGroupProfileRequest,
    ) -> Result<()> {
        let room_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        // 检查权限（只有群主或管理员可以修改）
        let member = self
            .member_repo
            .find_by_room_and_user(room_id, user_id)
            .await?
            .ok_or(AppError::Forbidden("无权限".to_string()))?;

        if member.role != MemberRole::Owner && member.role != MemberRole::Admin {
            return Err(AppError::Forbidden("只有群主或管理员可以修改".to_string()));
        }

        self.room_repo
            .update_profile(room_id, req.name.as_deref(), req.description.as_deref(), req.avatar_url.as_deref())
            .await?;

        Ok(())
    }

    /// 更新群组隐私设置
    pub async fn update_privacy(&self, user_id: i64, req: &UpdatePrivacyRequest) -> Result<()> {
        let room_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        // 检查是否为群主
        let room = self
            .room_repo
            .find_by_room_id(room_id)
            .await?
            .ok_or(AppError::NotFound("群组不存在".to_string()))?;

        if room.owner_id != user_id {
            return Err(AppError::Forbidden("只有群主可以修改隐私设置".to_string()));
        }

        // 处理群组类型转换
        let room_type = if let Some(gt) = &req.group_type {
            match gt.as_str() {
                "public_group" => Some(RoomType::PublicGroup),
                "private_group" => Some(RoomType::PrivateGroup),
                _ => return Err(AppError::BadRequest("无效的群组类型".to_string())),
            }
        } else {
            None
        };

        // 处理密码更新
        let password_hash = if let Some(pwd_hash) = &req.password_hash {
            if pwd_hash.is_empty() {
                None // 空字符串表示移除密码
            } else {
                Some(PasswordUtil::hash_password(pwd_hash)?)
            }
        } else {
            None // 不提供表示不修改
        };

        self.room_repo
            .update_privacy(room_id, room_type, password_hash, req.max_members)
            .await?;

        Ok(())
    }

    /// 获取我的聊天室列表（私聊+群聊）
    pub async fn list_my_rooms(
        &self,
        user_id: i64,
        room_type_str: Option<String>,
        limit: Option<i32>,
        cursor: Option<String>,
    ) -> Result<PaginatedResponse<MyGroupResponse>> {
        // 验证并设置默认limit
        let limit = limit.unwrap_or(20).min(100).max(1) as i64;
        
        // 转换 room_type 字符串为枚举
        let room_type = if let Some(rt_str) = room_type_str {
            match rt_str.as_str() {
                "public_group" => Some(RoomType::PublicGroup),
                "private_group" => Some(RoomType::PrivateGroup),
                "direct" => Some(RoomType::Direct),
                _ => None,
            }
        } else {
            None
        };

        let rooms = self
            .room_repo
            .list_user_rooms(user_id, room_type, cursor.clone(), limit)
            .await?;

        // 判断是否还有更多数据
        let has_more = rooms.len() as i64 >= limit;
        let next_cursor = if has_more && !rooms.is_empty() {
            Some(rooms.last().unwrap().room_id.to_string())
        } else {
            None
        };

        // TODO: 构造 MyGroupResponse（需要获取未读数、最后消息等）
        let items = rooms
            .into_iter()
            .map(|room| MyGroupResponse {
                id: room.room_id.to_string(),
                group_type: format!("{:?}", room.room_type).to_lowercase(),
                name: room.name,
                description: room.description,
                avatar_url: room.avatar_url,
                owner_id: room.owner_id.to_string(),
                latitude: 0.0,  // TODO: 从location_history获取
                longitude: 0.0,
                location_name: String::new(),
                member_count: 0, // TODO: 统计成员数
                my_role: "member".to_string(), // TODO: 从member表获取
                unread_count: 0,
                notifications_enabled: true,
                mention_only: false,
                joined_at: room.created_at.to_rfc3339(),
                last_active_at: room.last_message_at.unwrap_or(room.created_at).to_rfc3339(),
                last_message: None,
            })
            .collect();

        Ok(PaginatedResponse {
            items,
            has_more,
            next_cursor,
        })
    }
}


