use chrono::{DateTime, Utc};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::{
    error::Result,
    repositories::user_repo::UserProfileRow,
    types::enums::UserType,
};

#[derive(Debug, Clone, FromRow)]
pub struct AdminUserRow {
    pub user_id: i64,
    pub is_super: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
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
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
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

    pub async fn get_admin_user(&self, user_id: i64) -> Result<Option<AdminUserRow>> {
        let row = sqlx::query_as::<_, AdminUserRow>(
            r#"
            SELECT user_id, is_super, created_at, updated_at
            FROM admin_users
            WHERE user_id = $1
            "#,
        )
        .bind(user_id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn list_permissions(&self) -> Result<Vec<AdminPermissionRow>> {
        let rows = sqlx::query_as::<_, AdminPermissionRow>(
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
        let rows = sqlx::query_as::<_, AdminRoleRow>(
            r#"
            SELECT role_id, name, description, created_at, updated_at
            FROM admin_roles
            ORDER BY name ASC
            "#,
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    pub async fn get_role_by_id(&self, role_id: Uuid) -> Result<Option<AdminRoleRow>> {
        let row = sqlx::query_as::<_, AdminRoleRow>(
            r#"
            SELECT role_id, name, description, created_at, updated_at
            FROM admin_roles
            WHERE role_id = $1
            "#,
        )
        .bind(role_id)
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn get_role_permissions(&self, role_id: Uuid) -> Result<Vec<String>> {
        let rows = sqlx::query_scalar::<_, String>(
            r#"
            SELECT permission_key
            FROM admin_role_permissions
            WHERE role_id = $1
            ORDER BY permission_key ASC
            "#,
        )
        .bind(role_id)
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    pub async fn user_has_permission(&self, user_id: i64, permission_key: &str) -> Result<bool> {
        // 超级管理员：直接放行
        let is_super = sqlx::query_scalar::<_, bool>(
            r#"
            SELECT is_super
            FROM admin_users
            WHERE user_id = $1
            "#,
        )
        .bind(user_id)
        .fetch_optional(&self.pool)
        .await?
        .unwrap_or(false);

        if is_super {
            return Ok(true);
        }

        let has = sqlx::query_scalar::<_, bool>(
            r#"
            SELECT EXISTS(
                SELECT 1
                FROM admin_user_roles ur
                JOIN admin_role_permissions rp ON rp.role_id = ur.role_id
                WHERE ur.user_id = $1 AND rp.permission_key = $2
            )
            "#,
        )
        .bind(user_id)
        .bind(permission_key)
        .fetch_one(&self.pool)
        .await?;

        Ok(has)
    }

    pub async fn list_user_roles(&self, user_id: i64) -> Result<Vec<AdminRoleRow>> {
        let role_ids = sqlx::query_scalar::<_, Uuid>(
            r#"
            SELECT role_id
            FROM admin_user_roles
            WHERE user_id = $1
            "#,
        )
        .bind(user_id)
        .fetch_all(&self.pool)
        .await?;

        let mut roles = Vec::new();
        for rid in role_ids {
            if let Some(role) = self.get_role_by_id(rid).await? {
                roles.push(role);
            }
        }
        Ok(roles)
    }

    pub async fn list_user_permissions(&self, user_id: i64) -> Result<Vec<String>> {
        let is_super = sqlx::query_scalar::<_, bool>(
            r#"
            SELECT is_super
            FROM admin_users
            WHERE user_id = $1
            "#,
        )
        .bind(user_id)
        .fetch_optional(&self.pool)
        .await?
        .unwrap_or(false);

        if is_super {
            // 超级管理员拥有全部权限；这里返回 '*' 代表全量（文档会约定该语义）
            return Ok(vec!["*".to_string()]);
        }

        let rows = sqlx::query_scalar::<_, String>(
            r#"
            SELECT DISTINCT rp.permission_key
            FROM admin_user_roles ur
            JOIN admin_role_permissions rp ON rp.role_id = ur.role_id
            WHERE ur.user_id = $1
            ORDER BY rp.permission_key ASC
            "#,
        )
        .bind(user_id)
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    pub async fn ensure_user_type_admin(&self, user_id: i64) -> Result<()> {
        sqlx::query(
            r#"
            UPDATE users
            SET user_type = $2::user_type
            WHERE user_id = $1 AND user_type <> $2::user_type AND is_deleted = FALSE
            "#,
        )
        .bind(user_id)
        .bind(UserType::Admin.as_str())
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn revoke_user_type_admin(&self, user_id: i64) -> Result<()> {
        sqlx::query(
            r#"
            UPDATE users
            SET user_type = $2::user_type
            WHERE user_id = $1 AND user_type = $3::user_type AND is_deleted = FALSE
            "#,
        )
        .bind(user_id)
        .bind(UserType::Normal.as_str())
        .bind(UserType::Admin.as_str())
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn upsert_admin_user(&self, user_id: i64, is_super: bool) -> Result<()> {
        sqlx::query(
            r#"
            INSERT INTO admin_users (user_id, is_super)
            VALUES ($1, $2)
            ON CONFLICT (user_id) DO UPDATE SET is_super = EXCLUDED.is_super
            "#,
        )
        .bind(user_id)
        .bind(is_super)
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn replace_admin_roles(&self, user_id: i64, role_ids: &[Uuid]) -> Result<()> {
        // 清空旧角色
        sqlx::query(
            r#"
            DELETE FROM admin_user_roles
            WHERE user_id = $1
            "#,
        )
        .bind(user_id)
        .execute(&self.pool)
        .await?;

        // 重新插入
        for role_id in role_ids {
            sqlx::query(
                r#"
                INSERT INTO admin_user_roles (user_id, role_id)
                VALUES ($1, $2)
                ON CONFLICT (user_id, role_id) DO NOTHING
                "#,
            )
            .bind(user_id)
            .bind(*role_id)
            .execute(&self.pool)
            .await?;
        }

        Ok(())
    }

    pub async fn remove_admin_user(&self, user_id: i64) -> Result<()> {
        sqlx::query(
            r#"
            DELETE FROM admin_user_roles WHERE user_id = $1
            "#,
        )
        .bind(user_id)
        .execute(&self.pool)
        .await?;

        sqlx::query(
            r#"
            DELETE FROM admin_users WHERE user_id = $1
            "#,
        )
        .bind(user_id)
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    pub async fn create_role(&self, name: &str, description: Option<&str>) -> Result<Uuid> {
        let role_id = sqlx::query_scalar::<_, Uuid>(
            r#"
            INSERT INTO admin_roles (name, description)
            VALUES ($1, $2)
            RETURNING role_id
            "#,
        )
        .bind(name)
        .bind(description)
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
                sqlx::query(
                    r#"
                    UPDATE admin_roles
                    SET name = COALESCE($2, name)
                    WHERE role_id = $1
                    "#,
                )
                .bind(role_id)
                .bind(name)
                .execute(&self.pool)
                .await?;
            }
            Some(desc) => {
                sqlx::query(
                    r#"
                    UPDATE admin_roles
                    SET name = COALESCE($2, name),
                        description = $3
                    WHERE role_id = $1
                    "#,
                )
                .bind(role_id)
                .bind(name)
                .bind(desc)
                .execute(&self.pool)
                .await?;
            }
        }

        Ok(())
    }

    pub async fn delete_role(&self, role_id: Uuid) -> Result<()> {
        sqlx::query(
            r#"
            DELETE FROM admin_role_permissions WHERE role_id = $1
            "#,
        )
        .bind(role_id)
        .execute(&self.pool)
        .await?;

        sqlx::query(
            r#"
            DELETE FROM admin_user_roles WHERE role_id = $1
            "#,
        )
        .bind(role_id)
        .execute(&self.pool)
        .await?;

        sqlx::query(
            r#"
            DELETE FROM admin_roles WHERE role_id = $1
            "#,
        )
        .bind(role_id)
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    pub async fn set_role_permissions(&self, role_id: Uuid, permission_keys: &[String]) -> Result<()> {
        sqlx::query(
            r#"
            DELETE FROM admin_role_permissions WHERE role_id = $1
            "#,
        )
        .bind(role_id)
        .execute(&self.pool)
        .await?;

        for key in permission_keys {
            sqlx::query(
                r#"
                INSERT INTO admin_role_permissions (role_id, permission_key)
                VALUES ($1, $2)
                ON CONFLICT (role_id, permission_key) DO NOTHING
                "#,
            )
            .bind(role_id)
            .bind(key)
            .execute(&self.pool)
            .await?;
        }

        Ok(())
    }

    pub async fn list_admins(&self) -> Result<Vec<AdminWithRolesRow>> {
        let users = sqlx::query_as::<_, UserProfileRow>(
            r#"
            SELECT u.user_id, u.nickname, u.avatar_url, u.bio, u.user_type
            FROM admin_users au
            JOIN users u ON u.user_id = au.user_id
            WHERE u.is_deleted = FALSE
            ORDER BY u.user_id ASC
            "#,
        )
        .fetch_all(&self.pool)
        .await?;

        let mut result = Vec::with_capacity(users.len());
        for user in users {
            let is_super = sqlx::query_scalar::<_, bool>(
                r#"
                SELECT is_super
                FROM admin_users
                WHERE user_id = $1
                "#,
            )
            .bind(user.user_id)
            .fetch_one(&self.pool)
            .await?;

            let role_ids = sqlx::query_scalar::<_, Uuid>(
                r#"
                SELECT role_id
                FROM admin_user_roles
                WHERE user_id = $1
                "#,
            )
            .bind(user.user_id)
            .fetch_all(&self.pool)
            .await?;

            let mut roles = Vec::new();
            for rid in role_ids {
                if let Some(role) = self.get_role_by_id(rid).await? {
                    roles.push(role);
                }
            }

            result.push(AdminWithRolesRow { user, is_super, roles });
        }

        Ok(result)
    }
}

