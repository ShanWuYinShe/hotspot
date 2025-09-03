/// 群组角色值对象
/// 
/// 定义群组中用户的角色和权限

use serde::{Deserialize, Serialize};
use sqlx::Type;

/// 群组角色枚举，对应数据库中的 group_role 类型
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Type)]
#[sqlx(type_name = "group_role", rename_all = "lowercase")]
pub enum GroupRole {
    /// 群主
    Owner,
    /// 管理员
    Admin,
    /// 普通成员
    Member,
}

impl GroupRole {
    /// 转换为字符串
    pub fn as_str(&self) -> &'static str {
        match self {
            GroupRole::Owner => "owner",
            GroupRole::Admin => "admin",
            GroupRole::Member => "member",
        }
    }
    
    /// 从字符串解析（保留向后兼容性）
    pub fn from_str_option(s: &str) -> Option<Self> {
        match s {
            "owner" => Some(GroupRole::Owner),
            "admin" => Some(GroupRole::Admin),
            "member" => Some(GroupRole::Member),
            _ => None,
        }
    }
    
    /// 获取所有角色
    pub fn all() -> Vec<GroupRole> {
        vec![GroupRole::Owner, GroupRole::Admin, GroupRole::Member]
    }
    
    /// 检查角色权限等级
    pub fn permission_level(&self) -> u8 {
        match self {
            GroupRole::Owner => 3,
            GroupRole::Admin => 2,
            GroupRole::Member => 1,
        }
    }
    
    /// 检查是否有管理权限
    pub fn has_admin_permission(&self) -> bool {
        matches!(self, GroupRole::Owner | GroupRole::Admin)
    }
    
    /// 检查是否为群主
    pub fn is_owner(&self) -> bool {
        matches!(self, GroupRole::Owner)
    }
    
    /// 检查是否可以邀请成员
    pub fn can_invite_members(&self) -> bool {
        self.has_admin_permission()
    }
    
    /// 检查是否可以移除成员
    pub fn can_remove_members(&self) -> bool {
        self.has_admin_permission()
    }
    
    /// 检查是否可以修改群组信息
    pub fn can_modify_group(&self) -> bool {
        self.has_admin_permission()
    }
    
    /// 检查是否可以解散群组
    pub fn can_dissolve_group(&self) -> bool {
        self.is_owner()
    }
    
    /// 检查是否可以升级其他成员为管理员
    pub fn can_promote_to_admin(&self) -> bool {
        self.is_owner()
    }
    
    /// 检查是否可以移除管理员权限
    pub fn can_demote_admin(&self) -> bool {
        self.is_owner()
    }
    
    /// 检查是否可以对目标角色执行操作
    pub fn can_manage_role(&self, target_role: GroupRole) -> bool {
        self.permission_level() > target_role.permission_level()
    }
}

impl std::fmt::Display for GroupRole {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

impl From<GroupRole> for String {
    fn from(role: GroupRole) -> Self {
        role.as_str().to_string()
    }
}

impl std::str::FromStr for GroupRole {
    type Err = String;
    
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            "owner" => Ok(GroupRole::Owner),
            "admin" => Ok(GroupRole::Admin),
            "member" => Ok(GroupRole::Member),
            _ => Err(format!("Invalid group role: {}", s)),
        }
    }
}

