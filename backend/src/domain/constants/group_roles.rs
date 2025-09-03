/// 群组角色常量
/// 
/// 定义群组中用户的角色类型和相关操作

/// 群组成员角色
pub const MEMBER: &str = "member";

/// 群组管理员角色
pub const ADMIN: &str = "admin";

/// 群组所有者角色
pub const OWNER: &str = "owner";

/// 获取所有有效角色列表
pub fn all_roles() -> Vec<&'static str> {
    vec![MEMBER, ADMIN, OWNER]
}

/// 检查角色是否有效
pub fn is_valid_role(role: &str) -> bool {
    all_roles().contains(&role)
}

/// 检查角色是否有管理权限
pub fn has_admin_permission(role: &str) -> bool {
    matches!(role, ADMIN | OWNER)
}

/// 检查角色是否有所有者权限
pub fn has_owner_permission(role: &str) -> bool {
    role == OWNER
}

/// 角色权重（用于权限比较）
pub fn role_weight(role: &str) -> u8 {
    match role {
        MEMBER => 1,
        ADMIN => 2,
        OWNER => 3,
        _ => 0,
    }
}

/// 检查是否可以修改目标用户角色
pub fn can_modify_role(current_role: &str, target_role: &str) -> bool {
    role_weight(current_role) > role_weight(target_role)
}
