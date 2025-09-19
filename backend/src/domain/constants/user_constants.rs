/// 用户相关常量
/// 群组角色：成员
pub const MEMBER: &str = "member";

/// 群组角色：管理员
pub const ADMIN: &str = "admin";

/// 群组角色：群主
pub const OWNER: &str = "owner";

/// 群组角色模块
pub mod group_roles {
    pub const MEMBER: &str = super::MEMBER;
    pub const ADMIN: &str = super::ADMIN;
    pub const OWNER: &str = super::OWNER;
}

/// 最小用户名长度
pub const MIN_USERNAME_LENGTH: usize = 2;

/// 最大用户名长度
pub const MAX_USERNAME_LENGTH: usize = 50;

/// 最小密码长度
pub const MIN_PASSWORD_LENGTH: usize = 8;

/// 最大密码长度
pub const MAX_PASSWORD_LENGTH: usize = 128;


