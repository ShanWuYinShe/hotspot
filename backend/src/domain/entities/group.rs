/// 群组领域实体 - 三层数据结构设计
/// 
/// 架构说明：
/// 1. Row结构体：仅在Repository层使用，对应数据库字段
/// 2. Domain模型：核心业务模型，在Application和Presentation层传递
/// 3. API模型：在Presentation层定义，用于HTTP响应序列化
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use crate::domain::{values::*, events::*, DomainError, DomainResult};

// ===== 核心实体 =====

/// 群组实体 - 群组聚合根
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Group {
    /// 群组ID（聚合根标识）
    pub id: GroupId,
    /// 群组名称
    pub name: GroupName,
    /// 群组描述
    pub description: Option<String>,
    /// 群主ID
    pub owner_id: UserId,
    /// 群组位置
    pub location: GroupLocation,
    /// 群组类型
    pub group_type: GroupType,
    /// 群组可见性
    pub visibility: GroupVisibility,
    /// 是否公开
    pub is_public: bool,
    /// 最大成员数
    pub max_members: u32,
    /// 当前成员数
    pub member_count: u32,
    /// 创建时间
    pub created_at: DateTime<Utc>,
    /// 最后活跃时间
    pub last_active_at: DateTime<Utc>,
    /// 群组成员
    pub members: Vec<GroupMember>,
    /// 领域事件（待发布）
    #[serde(skip)]
    pub events: Vec<DomainEvent>,
}

/// 群组位置信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupLocation {
    /// 地理坐标
    pub coordinates: Coordinates,
    /// 位置名称
    pub location_name: String,
    /// 活动半径（米）
    pub radius: f64,
}

/// 群组成员
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupMember {
    /// 用户ID
    pub user_id: UserId,
    /// 成员角色
    pub role: GroupRole,
    /// 加入时间
    pub joined_at: DateTime<Utc>,
}

/// 群组角色
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum GroupRole {
    /// 群主
    Owner,
    /// 管理员
    Admin,
    /// 普通成员
    Member,
}

impl Group {
    /// 创建新群组
    pub fn new(
        id: GroupId,
        name: GroupName,
        owner_id: UserId,
        location: GroupLocation,
        group_type: GroupType,
        visibility: GroupVisibility,
    ) -> Self {
        let now = Utc::now();
        
        let mut group = Self {
            id: id.clone(),
            name: name.clone(),
            description: None,
            owner_id,
            location,
            group_type,
            visibility,
            is_public: visibility.is_public(),
            max_members: 500,
            member_count: 1, // 创建者自动成为成员
            created_at: now,
            last_active_at: now,
            members: vec![GroupMember {
                user_id: owner_id,
                role: GroupRole::Owner,
                joined_at: now,
            }],
            events: Vec::new(),
        };
        
        // 发布群组创建事件
        group.add_event(DomainEvent::GroupCreated {
            group_id: id,
            owner_id,
            created_at: now,
        });
        
        group
    }
    
    /// 添加领域事件
    pub fn add_event(&mut self, event: DomainEvent) {
        self.events.push(event);
    }
    
    /// 获取并清空领域事件
    pub fn take_events(&mut self) -> Vec<DomainEvent> {
        std::mem::take(&mut self.events)
    }
    
    /// 添加成员
    pub fn add_member(&mut self, user_id: UserId, role: GroupRole) -> DomainResult<()> {
        // 检查是否已经是成员
        if self.members.iter().any(|m| m.user_id == user_id) {
            return Err(DomainError::BusinessRuleViolation {
                rule: "User is already a member of this group".to_string(),
            });
        }
        
        // 检查成员数量限制
        if self.member_count >= self.max_members {
            return Err(DomainError::BusinessRuleViolation {
                rule: "Group has reached maximum member limit".to_string(),
            });
        }
        
        let now = Utc::now();
        
        self.members.push(GroupMember {
            user_id,
            role,
            joined_at: now,
        });
        
        self.member_count += 1;
        self.last_active_at = now;
        
        self.add_event(DomainEvent::GroupMemberJoined {
            group_id: self.id.clone(),
            user_id,
            joined_at: now,
        });
        
        Ok(())
    }
    
    /// 移除成员
    pub fn remove_member(&mut self, user_id: UserId) -> DomainResult<()> {
        // 不能移除群主
        if user_id == self.owner_id {
            return Err(DomainError::BusinessRuleViolation {
                rule: "Cannot remove group owner".to_string(),
            });
        }
        
        // 查找并移除成员
        if let Some(pos) = self.members.iter().position(|m| m.user_id == user_id) {
            self.members.remove(pos);
            self.member_count -= 1;
            self.last_active_at = Utc::now();
            
            self.add_event(DomainEvent::GroupMemberLeft {
                group_id: self.id.clone(),
                user_id,
                left_at: Utc::now(),
            });
            
            Ok(())
        } else {
            Err(DomainError::BusinessRuleViolation {
                rule: "User is not a member of this group".to_string(),
            })
        }
    }
    
    /// 更新群组信息
    pub fn update_info(
        &mut self,
        name: Option<GroupName>,
        description: Option<String>,
    ) -> DomainResult<()> {
        let mut updated = false;
        
        if let Some(new_name) = name {
            if self.name != new_name {
                self.name = new_name;
                updated = true;
            }
        }
        
        if description.is_some() && self.description != description {
            self.description = description;
            updated = true;
        }
        
        if updated {
            self.last_active_at = Utc::now();
            
            self.add_event(DomainEvent::GroupUpdated {
                group_id: self.id.clone(),
                updated_at: Utc::now(),
                updated_by: self.owner_id, // 简化处理，实际应该传入操作者ID
            });
        }
        
        Ok(())
    }
    
    /// 检查用户是否为成员
    pub fn is_member(&self, user_id: UserId) -> bool {
        self.members.iter().any(|m| m.user_id == user_id)
    }
    
    /// 检查用户是否为管理员或群主
    pub fn is_admin(&self, user_id: UserId) -> bool {
        self.members
            .iter()
            .find(|m| m.user_id == user_id)
            .map(|m| matches!(m.role, GroupRole::Owner | GroupRole::Admin))
            .unwrap_or(false)
    }
}

// ===== 领域模型层 (Domain Models) =====

/// 群组基础信息（最常用的轻量级模型）
/// 用途：群组列表、搜索结果、群组展示等90%的场景
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupProfile {
    /// 业务群组ID (UUID)
    pub group_id: String,
    /// 群组名称
    pub name: String,
    /// 群组描述
    pub description: Option<String>,
    /// 群主ID
    pub owner_id: i64,
    /// 纬度
    pub latitude: f64,
    /// 经度
    pub longitude: f64,
    /// 位置名称
    pub location_name: String,
    /// 成员数量
    pub member_count: i32,
    /// 是否公开
    pub is_public: bool,
    /// 最后活跃时间
    pub last_active_at: DateTime<Utc>,
}

/// 群组认证信息（仅用于加入群组时的密码验证）
/// 用途：群组密码验证、权限检查等认证相关操作
/// 注意：不实现Serialize，防止意外序列化敏感信息
#[derive(Debug, Clone)]
pub struct GroupAuth {
    /// 业务群组ID (UUID)
    pub group_id: String,
    /// 群组名称
    pub name: String,
    /// 群组密码哈希（仅验证时需要）
    pub password_hash: Option<String>,
    /// 是否公开
    pub is_public: bool,
    /// 群主ID
    pub owner_id: i64,
}

/// 群组详细信息（包含管理信息，仅特定场景使用）
/// 用途：群组管理页面、设置页面等需要完整信息的场景
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupDetail {
    /// 业务群组ID (UUID)
    pub group_id: String,
    /// 群组名称
    pub name: String,
    /// 群组描述
    pub description: Option<String>,
    /// 群主ID
    pub owner_id: i64,
    /// 纬度
    pub latitude: f64,
    /// 经度
    pub longitude: f64,
    /// 位置名称
    pub location_name: String,
    /// 成员数量
    pub member_count: i32,
    /// 最大成员数
    pub max_members: i32,
    /// 是否公开
    pub is_public: bool,
    /// 创建时间
    pub created_at: DateTime<Utc>,
    /// 更新时间
    pub updated_at: DateTime<Utc>,
    /// 最后活跃时间
    pub last_active_at: DateTime<Utc>,
}

/// 群组搜索结果（专用于搜索场景）
/// 用途：搜索API返回，包含距离、成员状态等搜索相关字段
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupSearchResult {
    /// 业务群组ID (UUID)
    pub group_id: String,
    /// 群组名称
    pub name: String,
    /// 群组描述
    pub description: Option<String>,
    /// 群主ID
    pub owner_id: i64,
    /// 纬度
    pub latitude: f64,
    /// 经度
    pub longitude: f64,
    /// 位置名称
    pub location_name: String,
    /// 成员数量
    pub member_count: i32,
    /// 是否公开
    pub is_public: bool,
    /// 最后活跃时间
    pub last_active_at: DateTime<Utc>,
    /// 距离（米，仅地理位置搜索时有值）
    pub distance: Option<f64>,
    /// 是否需要密码
    pub is_password_required: bool,
    /// 当前用户是否为成员
    pub is_member: bool,
    /// 当前用户在群组中的角色
    pub user_role: Option<String>,
}

// ===== 领域模型实现 =====

impl GroupProfile {
    /// 创建新的群组资料
    pub fn new(
        group_id: String,
        name: String,
        description: Option<String>,
        owner_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: String,
        member_count: i32,
        is_public: bool,
        last_active_at: DateTime<Utc>,
    ) -> Self {
        Self {
            group_id,
            name,
            description,
            owner_id,
            latitude,
            longitude,
            location_name,
            member_count,
            is_public,
            last_active_at,
        }
    }

    /// 判断是否为公开群组
    pub fn is_public(&self) -> bool {
        self.is_public
    }

    /// 判断用户是否为群主
    pub fn is_owner(&self, user_id: i64) -> bool {
        self.owner_id == user_id
    }
}

impl GroupAuth {
    /// 创建新的群组认证信息
    pub fn new(
        group_id: String,
        name: String,
        password_hash: Option<String>,
        is_public: bool,
        owner_id: i64,
    ) -> Self {
        Self {
            group_id,
            name,
            password_hash,
            is_public,
            owner_id,
        }
    }

    /// 验证是否需要密码
    pub fn requires_password(&self) -> bool {
        !self.is_public && self.password_hash.is_some()
    }

    /// 获取密码哈希（仅用于验证）
    pub fn get_password_hash(&self) -> Option<&str> {
        self.password_hash.as_deref()
    }

    /// 判断用户是否为群主
    pub fn is_owner(&self, user_id: i64) -> bool {
        self.owner_id == user_id
    }
}

impl GroupDetail {
    /// 创建新的群组详细信息
    pub fn new(
        group_id: String,
        name: String,
        description: Option<String>,
        owner_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: String,
        member_count: i32,
        max_members: i32,
        is_public: bool,
        created_at: DateTime<Utc>,
        updated_at: DateTime<Utc>,
        last_active_at: DateTime<Utc>,
    ) -> Self {
        Self {
            group_id,
            name,
            description,
            owner_id,
            latitude,
            longitude,
            location_name,
            member_count,
            max_members,
            is_public,
            created_at,
            updated_at,
            last_active_at,
        }
    }

    /// 转换为群组资料
    pub fn to_profile(&self) -> GroupProfile {
        GroupProfile {
            group_id: self.group_id.clone(),
            name: self.name.clone(),
            description: self.description.clone(),
            owner_id: self.owner_id,
            latitude: self.latitude,
            longitude: self.longitude,
            location_name: self.location_name.clone(),
            member_count: self.member_count,
            is_public: self.is_public,
            last_active_at: self.last_active_at,
        }
    }

    /// 判断是否已满员
    pub fn is_full(&self) -> bool {
        self.member_count >= self.max_members
    }

    /// 判断用户是否为群主
    pub fn is_owner(&self, user_id: i64) -> bool {
        self.owner_id == user_id
    }
}

impl GroupSearchResult {
    /// 创建新的群组搜索结果
    pub fn new(
        group_id: String,
        name: String,
        description: Option<String>,
        owner_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: String,
        member_count: i32,
        is_public: bool,
        last_active_at: DateTime<Utc>,
        distance: Option<f64>,
        is_password_required: bool,
        is_member: bool,
        user_role: Option<String>,
    ) -> Self {
        Self {
            group_id,
            name,
            description,
            owner_id,
            latitude,
            longitude,
            location_name,
            member_count,
            is_public,
            last_active_at,
            distance,
            is_password_required,
            is_member,
            user_role,
        }
    }

    /// 从群组资料创建搜索结果
    pub fn from_profile(
        profile: GroupProfile,
        distance: Option<f64>,
        is_password_required: bool,
        is_member: bool,
        user_role: Option<String>,
    ) -> Self {
        Self {
            group_id: profile.group_id,
            name: profile.name,
            description: profile.description,
            owner_id: profile.owner_id,
            latitude: profile.latitude,
            longitude: profile.longitude,
            location_name: profile.location_name,
            member_count: profile.member_count,
            is_public: profile.is_public,
            last_active_at: profile.last_active_at,
            distance,
            is_password_required,
            is_member,
            user_role,
        }
    }

    /// 转换为群组资料
    pub fn to_profile(&self) -> GroupProfile {
        GroupProfile {
            group_id: self.group_id.clone(),
            name: self.name.clone(),
            description: self.description.clone(),
            owner_id: self.owner_id,
            latitude: self.latitude,
            longitude: self.longitude,
            location_name: self.location_name.clone(),
            member_count: self.member_count,
            is_public: self.is_public,
            last_active_at: self.last_active_at,
        }
    }

    /// 判断用户是否为群主
    pub fn is_owner(&self, user_id: i64) -> bool {
        self.owner_id == user_id
    }
}

// ===== 类型转换实现 =====

impl From<GroupDetail> for GroupProfile {
    fn from(detail: GroupDetail) -> Self {
        detail.to_profile()
    }
}

impl From<GroupSearchResult> for GroupProfile {
    fn from(result: GroupSearchResult) -> Self {
        result.to_profile()
    }
}