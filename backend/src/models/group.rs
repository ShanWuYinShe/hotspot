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
    
    /// 从字符串解析
    pub fn from_str(s: &str) -> Option<Self> {
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
