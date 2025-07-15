// 群组角色常量
pub mod group_roles {
    pub const MEMBER: &str = "member";
    pub const ADMIN: &str = "admin";
    pub const OWNER: &str = "owner";
    
    // 获取所有有效角色列表
    pub fn all_roles() -> Vec<&'static str> {
        vec![MEMBER, ADMIN, OWNER]
    }
    
    // 检查角色是否有效
    pub fn is_valid_role(role: &str) -> bool {
        all_roles().contains(&role)
    }
} 