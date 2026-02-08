use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;
use std::collections::HashMap;

use crate::error::Result;
use crate::types::enums::UserType;

/// 管理员用户Row - 严格按照数据库 schema
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct AdminUserRow {
    pub user_id: i64,
    pub role_id: Option<Uuid>,
    pub is_super_admin: bool,
    pub created_at: DateTime<Utc>,
}

/// 管理员角色Row
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct AdminRoleRow {
    pub role_id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub is_system: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// 管理员权限Row
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct AdminPermissionRow {
    pub permission_key: String,  // 如: user:create
    pub name: String,
    pub description: Option<String>,
}

#[derive(Debug, Clone, FromRow)]
pub struct AdminPermissionListRow {
    pub key: String,
    pub description: String,
}

/// 用户权限Row
#[derive(Debug, Clone, FromRow)]
pub struct UserPermissionRow {
    pub permission_key: String,
    pub name: String,
}

#[derive(Debug, Clone, FromRow)]
pub struct AdminUserInfoRow {
    pub user_id: i64,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub user_type: UserType,
}

#[derive(Debug, Clone)]
pub struct AdminWithRolesRow {
    pub user: AdminUserInfoRow,
    pub is_super: bool,
    pub roles: Vec<AdminRoleRow>,
}

#[derive(Debug, Clone, FromRow)]
struct AdminWithRoleRowRaw {
    pub user_id: i64,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub user_type: UserType,
    pub is_super_admin: bool,
    pub role_id: Option<Uuid>,
    pub role_name: Option<String>,
    pub role_description: Option<String>,
    pub role_is_system: Option<bool>,
    pub role_created_at: Option<DateTime<Utc>>,
    pub role_updated_at: Option<DateTime<Utc>>,
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

    /// 检查用户是否为管理员
    pub async fn is_admin(&self, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            "SELECT EXISTS(SELECT 1 FROM admin_users WHERE user_id = $1) as \"exists!\"",
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 检查用户是否为超级管理员
    pub async fn is_super_admin(&self, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            "SELECT COALESCE(is_super_admin, FALSE) as \"is_super_admin!\" FROM admin_users WHERE user_id = $1",
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(result.map(|r| r.is_super_admin).unwrap_or(false))
    }

    /// 是否存在超级管理员
    pub async fn super_admin_exists(&self) -> Result<bool> {
        let result = sqlx::query!(
            "SELECT EXISTS(SELECT 1 FROM admin_users WHERE COALESCE(is_super_admin, FALSE) = TRUE) as \"exists!\""
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 获取管理员信息
    pub async fn get_admin_info(&self, user_id: i64) -> Result<Option<AdminUserRow>> {
        let row = sqlx::query_as!(
            AdminUserRow,
            "SELECT user_id, role_id, COALESCE(is_super_admin, FALSE) as \"is_super_admin!\", created_at FROM admin_users WHERE user_id = $1",
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 获取管理员详情（含角色）
    pub async fn get_admin_user(&self, user_id: i64) -> Result<Option<AdminWithRolesRow>> {
        let rows = sqlx::query_as!(
            AdminWithRoleRowRaw,
            r#"
            SELECT u.user_id as "user_id!",
                   COALESCE(u.nickname, '') as "nickname!",
                   u.avatar_url as "avatar_url?",
                   u.user_type as "user_type!: UserType",
                   COALESCE(au.is_super_admin, FALSE) as "is_super_admin!",
                   r.role_id as "role_id?",
                   r.name as "role_name?",
                   r.description as "role_description?",
                   r.is_system as "role_is_system?",
                   r.created_at as "role_created_at?",
                   r.updated_at as "role_updated_at?"
            FROM admin_users au
            JOIN users u ON u.user_id = au.user_id
            LEFT JOIN admin_roles r ON au.role_id = r.role_id
            WHERE u.user_id = $1
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        if rows.is_empty() {
            return Ok(None);
        }

        let mut map: HashMap<i64, AdminWithRolesRow> = HashMap::new();
        for row in rows {
            let entry = map.entry(row.user_id).or_insert_with(|| AdminWithRolesRow {
                user: AdminUserInfoRow {
                    user_id: row.user_id,
                    nickname: row.nickname.clone(),
                    avatar_url: row.avatar_url.clone(),
                    user_type: row.user_type,
                },
                is_super: row.is_super_admin,
                roles: Vec::new(),
            });

            if let (Some(role_id), Some(role_name), Some(role_created_at), Some(role_updated_at)) = (
                row.role_id,
                row.role_name.clone(),
                row.role_created_at,
                row.role_updated_at,
            ) {
                entry.roles.push(AdminRoleRow {
                    role_id,
                    name: role_name,
                    description: row.role_description.clone(),
                    is_system: row.role_is_system.unwrap_or(false),
                    created_at: role_created_at,
                    updated_at: role_updated_at,
                });
            }
        }

        Ok(map.remove(&user_id))
    }

    /// 添加或更新管理员
    pub async fn upsert_admin_user(&self, user_id: i64, is_super: bool) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO admin_users (user_id, is_super_admin)
            VALUES ($1, $2)
            ON CONFLICT (user_id) DO UPDATE SET is_super_admin = EXCLUDED.is_super_admin
            "#,
            user_id,
            is_super
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 移除管理员
    pub async fn remove_admin_user(&self, user_id: i64) -> Result<()> {
        sqlx::query!("DELETE FROM admin_users WHERE user_id = $1", user_id)
            .execute(&self.pool)
            .await?;

        Ok(())
    }

    /// 确保用户类型为 admin
    pub async fn ensure_user_type_admin(&self, user_id: i64) -> Result<()> {
        sqlx::query!("UPDATE users SET user_type = 'admin' WHERE user_id = $1", user_id)
            .execute(&self.pool)
            .await?;

        Ok(())
    }

    /// 撤销用户 admin 类型
    pub async fn revoke_user_type_admin(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            "UPDATE users SET user_type = 'normal' WHERE user_id = $1 AND user_type = 'admin'",
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 检查用户是否有特定权限
    pub async fn user_has_permission(&self, user_id: i64, permission_key: &str) -> Result<bool> {
        if self.is_super_admin(user_id).await? {
            return Ok(true);
        }

        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM admin_users u
                JOIN admin_role_permissions rp ON u.role_id = rp.role_id
                WHERE u.user_id = $1 AND rp.permission_key = $2
                UNION
                SELECT 1 FROM admin_user_permissions
                WHERE user_id = $1 AND permission_key = $2
            ) as "exists!"
            "#,
            user_id,
            permission_key
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 权限列表
    pub async fn list_permissions(&self) -> Result<Vec<AdminPermissionListRow>> {
        let rows = sqlx::query_as!(
            AdminPermissionListRow,
            r#"
            SELECT permission_key as "key!",
                   COALESCE(description, '') as "description!"
            FROM admin_permissions
            ORDER BY permission_key
            "#
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 获取所有角色
    pub async fn list_roles(&self) -> Result<Vec<AdminRoleRow>> {
        let rows = sqlx::query_as!(
            AdminRoleRow,
            r#"
            SELECT role_id,
                   name,
                   description,
                   COALESCE(is_system, FALSE) as "is_system!",
                   created_at,
                   updated_at
            FROM admin_roles
            ORDER BY name
            "#
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 获取角色权限 keys
    pub async fn get_role_permissions(&self, role_id: Uuid) -> Result<Vec<String>> {
        let rows = sqlx::query_scalar!(
            r#"
            SELECT permission_key
            FROM admin_role_permissions
            WHERE role_id = $1
            ORDER BY permission_key
            "#,
            role_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 创建角色
    pub async fn create_role(&self, name: &str, description: Option<&str>) -> Result<Uuid> {
        let role_id = sqlx::query_scalar!(
            r#"
            INSERT INTO admin_roles (name, description, is_system)
            VALUES ($1, $2, FALSE)
            RETURNING role_id
            "#,
            name,
            description
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(role_id)
    }

    /// 更新角色
    pub async fn update_role(
        &self,
        role_id: Uuid,
        name: Option<&str>,
        description: Option<Option<&str>>,
    ) -> Result<()> {
        let (desc_set, desc_value) = match description {
            None => (false, None),
            Some(None) => (true, None),
            Some(Some(v)) => (true, Some(v)),
        };

        sqlx::query!(
            r#"
            UPDATE admin_roles
            SET name = COALESCE($2, name),
                description = CASE WHEN $3 THEN $4 ELSE description END,
                updated_at = NOW()
            WHERE role_id = $1 AND COALESCE(is_system, FALSE) = FALSE
            "#,
            role_id,
            name,
            desc_set,
            desc_value
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 删除角色
    pub async fn delete_role(&self, role_id: Uuid) -> Result<()> {
        sqlx::query!(
            "DELETE FROM admin_roles WHERE role_id = $1 AND COALESCE(is_system, FALSE) = FALSE",
            role_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 设置角色权限（覆盖）
    pub async fn set_role_permissions(
        &self,
        role_id: Uuid,
        permission_keys: &[String],
    ) -> Result<()> {
        let mut tx = self.pool.begin().await?;
        sqlx::query!(
            "DELETE FROM admin_role_permissions WHERE role_id = $1",
            role_id
        )
        .execute(&mut *tx)
        .await?;

        if !permission_keys.is_empty() {
            sqlx::query!(
                r#"
                INSERT INTO admin_role_permissions (role_id, permission_key)
                SELECT $1, UNNEST($2::text[])
                "#,
                role_id,
                permission_keys
            )
            .execute(&mut *tx)
            .await?;
        }

        tx.commit().await?;
        Ok(())
    }

    /// 管理员列表
    pub async fn list_admins(&self) -> Result<Vec<AdminWithRolesRow>> {
        let rows = sqlx::query_as!(
            AdminWithRoleRowRaw,
            r#"
            SELECT u.user_id as "user_id!",
                   COALESCE(u.nickname, '') as "nickname!",
                   u.avatar_url as "avatar_url?",
                   u.user_type as "user_type!: UserType",
                   COALESCE(au.is_super_admin, FALSE) as "is_super_admin!",
                   r.role_id as "role_id?",
                   r.name as "role_name?",
                   r.description as "role_description?",
                   r.is_system as "role_is_system?",
                   r.created_at as "role_created_at?",
                   r.updated_at as "role_updated_at?"
            FROM admin_users au
            JOIN users u ON u.user_id = au.user_id
            LEFT JOIN admin_roles r ON au.role_id = r.role_id
            ORDER BY au.created_at DESC
            "#
        )
        .fetch_all(&self.pool)
        .await?;

        let mut map: HashMap<i64, AdminWithRolesRow> = HashMap::new();
        for row in rows {
            let entry = map.entry(row.user_id).or_insert_with(|| AdminWithRolesRow {
                user: AdminUserInfoRow {
                    user_id: row.user_id,
                    nickname: row.nickname.clone(),
                    avatar_url: row.avatar_url.clone(),
                    user_type: row.user_type,
                },
                is_super: row.is_super_admin,
                roles: Vec::new(),
            });

            if let (Some(role_id), Some(role_name), Some(role_created_at), Some(role_updated_at)) = (
                row.role_id,
                row.role_name.clone(),
                row.role_created_at,
                row.role_updated_at,
            ) {
                entry.roles.push(AdminRoleRow {
                    role_id,
                    name: role_name,
                    description: row.role_description.clone(),
                    is_system: row.role_is_system.unwrap_or(false),
                    created_at: role_created_at,
                    updated_at: role_updated_at,
                });
            }
        }

        Ok(map.into_values().collect())
    }

    /// 获取用户权限列表
    pub async fn list_user_permissions(&self, user_id: i64) -> Result<Vec<String>> {
        let rows = sqlx::query_scalar!(
            r#"
            SELECT DISTINCT permission_key FROM (
                SELECT rp.permission_key
                FROM admin_users u
                JOIN admin_role_permissions rp ON u.role_id = rp.role_id
                WHERE u.user_id = $1
                UNION
                SELECT permission_key
                FROM admin_user_permissions
                WHERE user_id = $1
            ) t
            ORDER BY permission_key
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().flatten().collect())
    }

    /// 获取用户角色列表
    pub async fn list_user_roles(&self, user_id: i64) -> Result<Vec<AdminRoleRow>> {
        let rows = sqlx::query_as!(
            AdminRoleRow,
            r#"
            SELECT r.role_id,
                   r.name,
                   r.description,
                   COALESCE(r.is_system, FALSE) as "is_system!",
                   r.created_at,
                   r.updated_at
            FROM admin_users u
            JOIN admin_roles r ON u.role_id = r.role_id
            WHERE u.user_id = $1
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 覆盖设置管理员角色（当前 schema 仅支持单角色）
    pub async fn replace_admin_roles(&self, user_id: i64, role_ids: &[Uuid]) -> Result<()> {
        let role_id = role_ids.first().copied();
        sqlx::query!(
            "UPDATE admin_users SET role_id = $2 WHERE user_id = $1",
            user_id,
            role_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}
