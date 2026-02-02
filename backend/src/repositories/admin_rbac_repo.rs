use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::{error::Result, repositories::user_repo::UserProfileRow, types::enums::UserType};

#[derive(Debug, Clone, FromRow)]
pub struct AdminUserRow {
    pub is_super: bool,
}

#[derive(Debug, Clone, FromRow)]
pub struct AdminPermissionRow {
    pub key: String,
    pub description: String,
}

#[derive(Debug, Clone, FromRow)]
pub struct AdminRoleRow {
    pub role_id: Uuid,
    pub name: String,
    pub description: Option<String>,
}

#[derive(Debug, Clone)]
pub struct AdminWithRolesRow {
    pub user: UserProfileRow,
    pub is_super: bool,
    pub roles: Vec<AdminRoleRow>,
}

#[derive(Debug, Clone)]
pub struct AdminRbacRepository {
    pool: PgPool,
}

impl AdminRbacRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub fn pool(&self) -> &PgPool {
        &self.pool
    }

    /// 检查是否存在超级管理员
    pub async fn super_admin_exists(&self) -> Result<bool> {
        let exists = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(SELECT 1 FROM admin_users WHERE is_super = TRUE)
            "#,
        )
        .fetch_one(&self.pool)
        .await?
        .unwrap_or(false);
        Ok(exists)
    }

    pub async fn get_admin_user(&self, user_id: i64) -> Result<Option<AdminUserRow>> {
        let row = sqlx::query_as!(
            AdminUserRow,
            r#"
            SELECT is_super
            FROM admin_users
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn list_permissions(&self) -> Result<Vec<AdminPermissionRow>> {
        let rows = sqlx::query_as!(
            AdminPermissionRow,
            r#"
            SELECT key, description
            FROM admin_permissions
            ORDER BY key ASC
            "#,
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    pub async fn list_roles(&self) -> Result<Vec<AdminRoleRow>> {
        let rows = sqlx::query_as!(
            AdminRoleRow,
            r#"
            SELECT role_id, name, description
            FROM admin_roles
            ORDER BY name ASC
            "#,
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    pub async fn get_role_by_id(&self, role_id: Uuid) -> Result<Option<AdminRoleRow>> {
        let row = sqlx::query_as!(
            AdminRoleRow,
            r#"
            SELECT role_id, name, description
            FROM admin_roles
            WHERE role_id = $1
            "#,
            role_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn get_role_permissions(&self, role_id: Uuid) -> Result<Vec<String>> {
        let rows = sqlx::query_scalar!(
            r#"
            SELECT permission_key
            FROM admin_role_permissions
            WHERE role_id = $1
            ORDER BY permission_key ASC
            "#,
            role_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    pub async fn user_has_permission(&self, user_id: i64, permission_key: &str) -> Result<bool> {
        // 超级管理员：直接放行
        let is_super = sqlx::query_scalar!(
            r#"
            SELECT is_super
            FROM admin_users
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?
        .unwrap_or(false);

        if is_super {
            return Ok(true);
        }

        let has = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1
                FROM admin_user_roles ur
                JOIN admin_role_permissions rp ON rp.role_id = ur.role_id
                WHERE ur.user_id = $1 AND rp.permission_key = $2
            )
            "#,
            user_id,
            permission_key
        )
        .fetch_one(&self.pool)
        .await?
        .unwrap_or(false);

        Ok(has)
    }

    pub async fn list_user_roles(&self, user_id: i64) -> Result<Vec<AdminRoleRow>> {
        // 使用 JOIN 一次性查询所有角色信息，避免 N+1 查询
        let roles = sqlx::query_as!(
            AdminRoleRow,
            r#"
            SELECT r.role_id, r.name, r.description
            FROM admin_user_roles aur
            JOIN admin_roles r ON r.role_id = aur.role_id
            WHERE aur.user_id = $1
            ORDER BY r.name ASC
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(roles)
    }

    pub async fn list_user_permissions(&self, user_id: i64) -> Result<Vec<String>> {
        let is_super = sqlx::query_scalar!(
            r#"
            SELECT is_super
            FROM admin_users
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?
        .unwrap_or(false);

        if is_super {
            // 超级管理员拥有全部权限；这里返回 '*' 代表全量（文档会约定该语义）
            return Ok(vec!["*".to_string()]);
        }

        let rows = sqlx::query_scalar!(
            r#"
            SELECT DISTINCT rp.permission_key
            FROM admin_user_roles ur
            JOIN admin_role_permissions rp ON rp.role_id = ur.role_id
            WHERE ur.user_id = $1
            ORDER BY rp.permission_key ASC
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    pub async fn ensure_user_type_admin(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE users
            SET user_type = $2::user_type
            WHERE user_id = $1 AND user_type <> $2::user_type AND is_banned = FALSE
            "#,
            user_id,
            UserType::Admin.as_str() as _
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn revoke_user_type_admin(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE users
            SET user_type = $2::user_type
            WHERE user_id = $1 AND user_type = $3::user_type AND is_banned = FALSE
            "#,
            user_id,
            UserType::Normal.as_str() as _,
            UserType::Admin.as_str() as _
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn upsert_admin_user(&self, user_id: i64, is_super: bool) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO admin_users (user_id, is_super)
            VALUES ($1, $2)
            ON CONFLICT (user_id) DO UPDATE SET is_super = EXCLUDED.is_super
            "#,
            user_id,
            is_super
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn replace_admin_roles(&self, user_id: i64, role_ids: &[Uuid]) -> Result<()> {
        // 清空旧角色
        sqlx::query!(
            r#"
            DELETE FROM admin_user_roles
            WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        // 批量插入新的角色关联（避免循环插入）
        if !role_ids.is_empty() {
            sqlx::query!(
                r#"
                INSERT INTO admin_user_roles (user_id, role_id)
                SELECT $1, unnest($2::uuid[])
                ON CONFLICT (user_id, role_id) DO NOTHING
                "#,
                user_id,
                role_ids
            )
            .execute(&self.pool)
            .await?;
        }

        Ok(())
    }

    pub async fn remove_admin_user(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM admin_user_roles WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        sqlx::query!(
            r#"
            DELETE FROM admin_users WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    pub async fn create_role(&self, name: &str, description: Option<&str>) -> Result<Uuid> {
        let role_id = sqlx::query_scalar!(
            r#"
            INSERT INTO admin_roles (name, description)
            VALUES ($1, $2)
            RETURNING role_id
            "#,
            name,
            description
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(role_id)
    }

    pub async fn update_role(
        &self,
        role_id: Uuid,
        name: Option<&str>,
        description: Option<Option<&str>>,
    ) -> Result<()> {
        // description:
        // - None => keep current description
        // - Some(None) => set NULL
        // - Some(Some(v)) => set v
        match description {
            None => {
                sqlx::query!(
                    r#"
                    UPDATE admin_roles
                    SET name = COALESCE($2, name)
                    WHERE role_id = $1
                    "#,
                    role_id,
                    name
                )
                .execute(&self.pool)
                .await?;
            }
            Some(desc) => {
                sqlx::query!(
                    r#"
                    UPDATE admin_roles
                    SET name = COALESCE($2, name),
                        description = $3
                    WHERE role_id = $1
                    "#,
                    role_id,
                    name,
                    desc
                )
                .execute(&self.pool)
                .await?;
            }
        }

        Ok(())
    }

    pub async fn delete_role(&self, role_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM admin_role_permissions WHERE role_id = $1
            "#,
            role_id
        )
        .execute(&self.pool)
        .await?;

        sqlx::query!(
            r#"
            DELETE FROM admin_user_roles WHERE role_id = $1
            "#,
            role_id
        )
        .execute(&self.pool)
        .await?;

        sqlx::query!(
            r#"
            DELETE FROM admin_roles WHERE role_id = $1
            "#,
            role_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    pub async fn set_role_permissions(
        &self,
        role_id: Uuid,
        permission_keys: &[String],
    ) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM admin_role_permissions WHERE role_id = $1
            "#,
            role_id
        )
        .execute(&self.pool)
        .await?;

        // 批量插入权限（避免循环插入）
        if !permission_keys.is_empty() {
            sqlx::query!(
                r#"
                INSERT INTO admin_role_permissions (role_id, permission_key)
                SELECT $1, unnest($2::text[])
                ON CONFLICT (role_id, permission_key) DO NOTHING
                "#,
                role_id,
                permission_keys
            )
            .execute(&self.pool)
            .await?;
        }

        Ok(())
    }

    pub async fn list_admins(&self) -> Result<Vec<AdminWithRolesRow>> {
        // 步骤1：一次性查询所有管理员用户和 is_super 信息
        #[derive(Debug, FromRow)]
        struct AdminUserInfo {
            user_id: i64,
            nickname: String,
            avatar_url: Option<String>,
            bio: String,
            user_type: UserType,
            is_super: bool,
        }

        let admin_users = sqlx::query_as!(
            AdminUserInfo,
            r#"
            SELECT u.user_id, u.nickname, u.avatar_url, u.bio, u.user_type as "user_type: UserType", au.is_super
            FROM admin_users au
            JOIN users u ON u.user_id = au.user_id
            WHERE u.is_banned = FALSE
            ORDER BY u.user_id ASC
            "#,
        )
        .fetch_all(&self.pool)
        .await?;

        if admin_users.is_empty() {
            return Ok(Vec::new());
        }

        // 步骤2：批量查询所有管理员的角色信息
        let user_ids: Vec<i64> = admin_users.iter().map(|u| u.user_id).collect();

        #[derive(Debug, FromRow)]
        struct UserRoleRow {
            user_id: i64,
            role_id: Uuid,
            role_name: String,
            role_description: Option<String>,
        }

        let user_roles = sqlx::query_as!(
            UserRoleRow,
            r#"
            SELECT aur.user_id, r.role_id, r.name as role_name, r.description as role_description
            FROM admin_user_roles aur
            JOIN admin_roles r ON r.role_id = aur.role_id
            WHERE aur.user_id = ANY($1)
            ORDER BY aur.user_id ASC, r.name ASC
            "#,
            &user_ids
        )
        .fetch_all(&self.pool)
        .await?;

        // 步骤3：按 user_id 分组角色数据
        let mut roles_by_user: std::collections::HashMap<i64, Vec<AdminRoleRow>> =
            std::collections::HashMap::new();

        for ur in user_roles {
            roles_by_user
                .entry(ur.user_id)
                .or_insert_with(Vec::new)
                .push(AdminRoleRow {
                    role_id: ur.role_id,
                    name: ur.role_name,
                    description: ur.role_description,
                });
        }

        // 步骤4：组装最终结果
        let result: Vec<AdminWithRolesRow> = admin_users
            .into_iter()
            .map(|admin_user| {
                let roles = roles_by_user
                    .remove(&admin_user.user_id)
                    .unwrap_or_default();

                AdminWithRolesRow {
                    user: UserProfileRow {
                        user_id: admin_user.user_id,
                        nickname: admin_user.nickname,
                        avatar_url: admin_user.avatar_url,
                        bio: admin_user.bio,
                        user_type: admin_user.user_type,
                    },
                    is_super: admin_user.is_super,
                    roles,
                }
            })
            .collect();

        Ok(result)
    }
}
