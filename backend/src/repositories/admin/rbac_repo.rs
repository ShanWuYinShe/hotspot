use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use std::collections::HashMap;
use uuid::Uuid;

use crate::error::Result;
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

#[derive(Debug, Clone, FromRow)]
pub struct AdminPermissionListRow {
    pub key: String,
    pub description: String,
}

#[derive(Debug, Clone, FromRow)]
pub struct AdminUserInfoRow {
    pub user_id: i64,
    pub nickname: String,
    pub avatar_url: Option<String>,
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
    pub is_super_admin: bool,
    pub au_created_at: DateTime<Utc>,
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

    /// 检查用户是否为管理员（轻量级，仅判断存在性）
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

    /// 获取管理员详情（含角色）
    pub async fn get_admin_user(&self, user_id: i64) -> Result<Option<AdminWithRolesRow>> {
        let rows = sqlx::query_as!(
            AdminWithRoleRowRaw,
            r#"
            SELECT u.user_id as "user_id!",
                   COALESCE(u.nickname, '') as "nickname!",
                   u.avatar_url as "avatar_url?",
                   COALESCE(au.is_super_admin, FALSE) as "is_super_admin!",
                   au.created_at as "au_created_at!",
                   r.role_id as "role_id?",
                   r.name as "role_name?",
                   r.description as "role_description?",
                   r.is_system as "role_is_system?",
                   r.created_at as "role_created_at?",
                   r.updated_at as "role_updated_at?"
            FROM admin_users au
            JOIN users u ON u.user_id = au.user_id
            LEFT JOIN admin_user_roles aur ON aur.user_id = au.user_id
            LEFT JOIN admin_roles r ON aur.role_id = r.role_id
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

    /// 检查用户是否有特定权限
    pub async fn user_has_permission(&self, user_id: i64, permission_key: &str) -> Result<bool> {
        if self.is_super_admin(user_id).await? {
            return Ok(true);
        }

        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM admin_user_roles ur
                JOIN admin_role_permissions rp ON ur.role_id = rp.role_id
                WHERE ur.user_id = $1 AND rp.permission_key = $2
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

    /// 管理员列表（分页）
    pub async fn list_admins(
        &self,
        limit: usize,
        cursor: Option<&str>,
    ) -> crate::utils::pagination::Paginated<AdminWithRolesRow> {
        use crate::utils::pagination::{decode_cursor, encode_cursor, TimeCursor};

        let cursor_val = cursor.and_then(|c| decode_cursor::<TimeCursor>(c).ok());
        let query_limit = limit + 1;

        let rows = if let Some((cursor_time, cursor_id)) = cursor_val {
            sqlx::query_as!(
                AdminWithRoleRowRaw,
                r#"
                SELECT u.user_id as "user_id!",
                       COALESCE(u.nickname, '') as "nickname!",
                       u.avatar_url as "avatar_url?",
                       COALESCE(au.is_super_admin, FALSE) as "is_super_admin!",
                       au.created_at as "au_created_at!",
                       r.role_id as "role_id?",
                       r.name as "role_name?",
                       r.description as "role_description?",
                       r.is_system as "role_is_system?",
                       r.created_at as "role_created_at?",
                       r.updated_at as "role_updated_at?"
                FROM admin_users au
                JOIN users u ON u.user_id = au.user_id
                LEFT JOIN admin_user_roles aur ON aur.user_id = au.user_id
                LEFT JOIN admin_roles r ON aur.role_id = r.role_id
                WHERE (au.created_at, au.user_id) < ($1, $2)
                ORDER BY au.created_at DESC, au.user_id DESC
                LIMIT $3
                "#,
                cursor_time,
                cursor_id,
                query_limit as i64
            )
            .fetch_all(&self.pool)
            .await?
        } else {
            sqlx::query_as!(
                AdminWithRoleRowRaw,
                r#"
                SELECT u.user_id as "user_id!",
                       COALESCE(u.nickname, '') as "nickname!",
                       u.avatar_url as "avatar_url?",
                       COALESCE(au.is_super_admin, FALSE) as "is_super_admin!",
                       au.created_at as "au_created_at!",
                       r.role_id as "role_id?",
                       r.name as "role_name?",
                       r.description as "role_description?",
                       r.is_system as "role_is_system?",
                       r.created_at as "role_created_at?",
                       r.updated_at as "role_updated_at?"
                FROM admin_users au
                JOIN users u ON u.user_id = au.user_id
                LEFT JOIN admin_user_roles aur ON aur.user_id = au.user_id
                LEFT JOIN admin_roles r ON aur.role_id = r.role_id
                ORDER BY au.created_at DESC, au.user_id DESC
                LIMIT $1
                "#,
                query_limit as i64
            )
            .fetch_all(&self.pool)
            .await?
        };

        let mut map: HashMap<i64, AdminWithRolesRow> = HashMap::new();
        for row in &rows {
            let entry = map.entry(row.user_id).or_insert_with(|| AdminWithRolesRow {
                user: AdminUserInfoRow {
                    user_id: row.user_id,
                    nickname: row.nickname.clone(),
                    avatar_url: row.avatar_url.clone(),
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

        let mut order_keys: Vec<(DateTime<Utc>, i64)> =
            rows.iter().map(|r| (r.au_created_at, r.user_id)).collect();
        order_keys.dedup_by(|a, b| a.1 == b.1);
        let mut by_user = map;
        let mut result: Vec<AdminWithRolesRow> = order_keys
            .into_iter()
            .filter_map(|(_, uid)| by_user.remove(&uid))
            .collect();

        let has_more = result.len() > limit;
        if has_more {
            result.truncate(limit);
        }
        let next_cursor = if has_more {
            result
                .last()
                .and_then(|last| rows.iter().find(|r| r.user_id == last.user.user_id))
                .and_then(|r| encode_cursor(&(r.au_created_at, r.user_id)).ok())
        } else {
            None
        };

        Ok((result, next_cursor))
    }

    /// 获取用户权限列表
    pub async fn list_user_permissions(&self, user_id: i64) -> Result<Vec<String>> {
        let rows = sqlx::query_scalar!(
            r#"
            SELECT DISTINCT permission_key FROM (
                SELECT rp.permission_key
                FROM admin_user_roles ur
                JOIN admin_role_permissions rp ON ur.role_id = rp.role_id
                WHERE ur.user_id = $1
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
            FROM admin_user_roles ur
            JOIN admin_roles r ON ur.role_id = r.role_id
            WHERE ur.user_id = $1
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 覆盖设置管理员角色（当前 schema 仅支持单角色）
    pub async fn replace_admin_roles(&self, user_id: i64, role_ids: &[Uuid]) -> Result<()> {
        let mut tx = self.pool.begin().await?;
        sqlx::query!("DELETE FROM admin_user_roles WHERE user_id = $1", user_id)
            .execute(&mut *tx)
            .await?;

        if !role_ids.is_empty() {
            sqlx::query!(
                r#"
                INSERT INTO admin_user_roles (user_id, role_id)
                SELECT $1, UNNEST($2::uuid[])
                "#,
                user_id,
                role_ids
            )
            .execute(&mut *tx)
            .await?;
        }

        tx.commit().await?;
        Ok(())
    }
}
