/// 群组领域实体
/// 
/// 包含群组的核心业务逻辑和状态

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use crate::domain::values::GroupRole;

/// 群组实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Group {
    /// 数据库自增主键
    pub id: i64,
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
    /// 群组密码哈希（可选）
    pub password_hash: Option<String>,
    /// 创建时间
    pub created_at: DateTime<Utc>,
    /// 更新时间
    pub updated_at: DateTime<Utc>,
    /// 最后活跃时间
    pub last_active_at: DateTime<Utc>,
}

impl Group {
    /// 创建新群组
    pub fn new(
        group_id: String,
        name: String,
        description: Option<String>,
        owner_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: String,
        password_hash: Option<String>,
    ) -> Self {
        let now = Utc::now();
        Self {
            id: 0, // 将由数据库生成
            group_id,
            name,
            description,
            owner_id,
            latitude,
            longitude,
            location_name,
            member_count: 1, // 创建者默认为第一个成员
            max_members: 100, // 默认最大成员数
            is_public: true, // 默认为公开群组
            password_hash,
            created_at: now,
            updated_at: now,
            last_active_at: now,
        }
    }
    
    /// 检查群组是否有密码保护
    pub fn is_password_protected(&self) -> bool {
        self.password_hash.is_some()
    }
    
    /// 更新最后活跃时间
    pub fn update_last_active(&mut self) {
        self.last_active_at = Utc::now();
    }
    
    /// 设置密码
    pub fn set_password(&mut self, password_hash: Option<String>) {
        self.password_hash = password_hash;
        self.updated_at = Utc::now();
    }
    
    /// 更新群组信息
    pub fn update_info(&mut self, name: Option<String>, description: Option<String>) {
        if let Some(name) = name {
            self.name = name;
        }
        if description.is_some() {
            self.description = description;
        }
        self.updated_at = Utc::now();
    }
    
    /// 更新位置信息
    pub fn update_location(&mut self, latitude: f64, longitude: f64, location_name: String) {
        self.latitude = latitude;
        self.longitude = longitude;
        self.location_name = location_name;
        self.updated_at = Utc::now();
    }

    /// 检查是否有位置信息
    pub fn has_location(&self) -> bool {
        // Group实体总是有位置信息，因为创建时需要位置
        true
    }
    
    /// 验证群组数据的完整性
    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        
        if self.name.is_empty() {
            errors.push("群组名称不能为空".to_string());
        }
        
        if self.name.len() > 100 {
            errors.push("群组名称长度不能超过100个字符".to_string());
        }
        
        if let Some(ref description) = self.description {
            if description.len() > 500 {
                errors.push("群组描述长度不能超过500个字符".to_string());
            }
        }
        
        if self.latitude < -90.0 || self.latitude > 90.0 {
            errors.push("纬度必须在-90到90之间".to_string());
        }
        
        if self.longitude < -180.0 || self.longitude > 180.0 {
            errors.push("经度必须在-180到180之间".to_string());
        }
        
        if self.location_name.is_empty() {
            errors.push("位置名称不能为空".to_string());
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
}

/// 群组成员实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupMember {
    /// 数据库自增主键
    pub id: i64,
    /// 群组ID (UUID)
    pub group_id: String,
    /// 用户ID
    pub user_id: i64,
    /// 成员角色
    pub role: GroupRole,
    /// 加入时间
    pub joined_at: DateTime<Utc>,
    /// 通知级别
    pub notification_level: String,
    /// 最后活跃时间
    pub last_active: Option<DateTime<Utc>>,
}

impl GroupMember {
    /// 创建新的群组成员
    pub fn new(group_id: String, user_id: i64, role: GroupRole) -> Self {
        let now = Utc::now();
        Self {
            id: 0, // 将由数据库生成
            group_id,
            user_id,
            role,
            joined_at: now,
            notification_level: "all".to_string(),
            last_active: Some(now),
        }
    }
    
    /// 检查是否为群主
    pub fn is_owner(&self) -> bool {
        self.role == GroupRole::Owner
    }
    
    /// 检查是否为管理员或群主
    pub fn is_admin_or_owner(&self) -> bool {
        matches!(self.role, GroupRole::Admin | GroupRole::Owner)
    }
    
    /// 检查是否可以管理群组
    pub fn can_manage_group(&self) -> bool {
        self.is_admin_or_owner()
    }
    
    /// 检查是否可以邀请成员
    pub fn can_invite_members(&self) -> bool {
        self.is_admin_or_owner()
    }
    
    /// 检查是否可以移除成员
    pub fn can_remove_members(&self) -> bool {
        self.is_admin_or_owner()
    }
    
    /// 更新角色
    pub fn update_role(&mut self, new_role: GroupRole) {
        self.role = new_role;
    }
    
    /// 更新通知级别
    pub fn update_notification_level(&mut self, level: String) -> Result<(), String> {
        if !matches!(level.as_str(), "all" | "mentions" | "none") {
            return Err("无效的通知级别".to_string());
        }
        self.notification_level = level;
        Ok(())
    }
    
    /// 更新最后活跃时间
    pub fn update_last_active(&mut self) {
        self.last_active = Some(Utc::now());
    }
}
