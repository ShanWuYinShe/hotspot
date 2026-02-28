use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use std::collections::{BTreeMap, BTreeSet, HashMap};
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

    fn canonical_permission_key(permission_key: &str) -> String {
        match permission_key {
            "admin.users.view" => "admin.users.read".to_string(),
            "admin.users.ban" | "admin.users.edit" | "admin.users.reset_password" => {
                "admin.users.write".to_string()
            }
            "admin.groups.view" => "admin.groups.read".to_string(),
            "admin.groups.dismiss" | "admin.groups.edit" => "admin.groups.write".to_string(),
            "admin.reports.view" => "admin.reports.read".to_string(),
            "admin.reports.handle" => "admin.reports.write".to_string(),
            "admin.config.view" => "admin.app_config.read".to_string(),
            "admin.config.edit" => "admin.app_config.write".to_string(),
            "admin.announcements.view" => "admin.announcements.read".to_string(),
            "admin.announcements.manage" => "admin.announcements.write".to_string(),
            "admin.audit.view" => "admin.audit.read".to_string(),
            "admin.rbac.view" => "admin.rbac.manage".to_string(),
            _ => permission_key.to_string(),
        }
    }

    fn legacy_permission_keys_for(canonical_key: &str) -> &'static [&'static str] {
        match canonical_key {
            "admin.users.read" => &["admin.users.view"],
            "admin.users.write" => &[
                "admin.users.ban",
                "admin.users.edit",
                "admin.users.reset_password",
            ],
            "admin.groups.read" => &["admin.groups.view"],
            "admin.groups.write" => &["admin.groups.dismiss", "admin.groups.edit"],
            "admin.reports.read" => &["admin.reports.view"],
            "admin.reports.write" => &["admin.reports.handle"],
            "admin.app_config.read" => &["admin.config.view"],
            "admin.app_config.write" => &["admin.config.edit"],
            "admin.announcements.read" => &["admin.announcements.view"],
            "admin.announcements.write" => &["admin.announcements.manage"],
            "admin.audit.read" => &["admin.audit.view"],
            "admin.rbac.manage" => &["admin.rbac.view"],
            _ => &[],
        }
    }

    fn compatible_permission_keys(permission_key: &str) -> Vec<String> {
        let canonical = Self::canonical_permission_key(permission_key);
        let mut keys: BTreeSet<String> = BTreeSet::new();
        keys.insert(canonical.clone());
        for legacy in Self::legacy_permission_keys_for(&canonical) {
            keys.insert((*legacy).to_string());
        }
        keys.into_iter().collect()
    }

    fn canonical_permission_catalog(
    ) -> &'static [(&'static str, &'static str, &'static str, &'static str)] {
        &[
            (
                "admin.rbac.manage",
                "管理员权限管理",
                "rbac",
                "创建/编辑/删除角色和权限",
            ),
            (
                "admin.reports.read",
                "查看举报列表",
                "reports",
                "查看举报列表",
            ),
            (
                "admin.reports.write",
                "处理举报",
                "reports",
                "审核和处理举报",
            ),
            (
                "admin.app_config.read",
                "查看应用配置",
                "config",
                "查看应用配置项",
            ),
            (
                "admin.app_config.write",
                "修改应用配置",
                "config",
                "修改应用配置项",
            ),
            ("admin.tags.read", "查看标签列表", "tags", "查看标签列表"),
            ("admin.tags.write", "管理标签", "tags", "删除/屏蔽标签"),
            (
                "admin.announcements.read",
                "查看公告列表",
                "announcements",
                "查看系统公告",
            ),
            (
                "admin.announcements.write",
                "管理公告",
                "announcements",
                "创建/编辑/删除系统公告",
            ),
            (
                "admin.users.read",
                "查看用户列表/详情",
                "users",
                "查看和搜索用户",
            ),
            (
                "admin.users.write",
                "管理用户账号",
                "users",
                "封禁/解封用户",
            ),
            (
                "admin.groups.read",
                "查看群组列表/详情",
                "groups",
                "查看和搜索群组",
            ),
            ("admin.groups.write", "管理群组", "groups", "解散群组"),
            (
                "admin.audit.read",
                "查看审计日志",
                "audit",
                "查看操作审计日志（不含敏感 IP）",
            ),
            (
                "admin.audit.read_ip",
                "查看审计日志 IP",
                "audit",
                "查看审计日志中的 IP 地址（需搭配 admin.audit.read）",
            ),
            (
                "admin.audit.export",
                "导出审计日志",
                "audit",
                "导出审计日志数据",
            ),
        ]
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

    /// 启动时对齐权限字典与历史权限 key，保证新旧 key 语义一致。
    pub async fn ensure_permission_catalog_alignment(&self) -> Result<()> {
        for (key, name, module, description) in Self::canonical_permission_catalog() {
            sqlx::query!(
                r#"
                INSERT INTO admin_permissions (permission_key, name, module, description)
                VALUES ($1, $2, $3, $4)
                ON CONFLICT (permission_key) DO UPDATE
                SET name = EXCLUDED.name,
                    module = EXCLUDED.module,
                    description = EXCLUDED.description
                "#,
                key,
                name,
                module,
                description
            )
            .execute(&self.pool)
            .await?;
        }

        // 将历史 key 的角色权限与用户直授权限同步到 canonical key（保留历史 key 以避免破坏已有数据）
        for (legacy_key, canonical_key) in [
            ("admin.users.view", "admin.users.read"),
            ("admin.users.ban", "admin.users.write"),
            ("admin.users.edit", "admin.users.write"),
            ("admin.users.reset_password", "admin.users.write"),
            ("admin.groups.view", "admin.groups.read"),
            ("admin.groups.dismiss", "admin.groups.write"),
            ("admin.groups.edit", "admin.groups.write"),
            ("admin.reports.view", "admin.reports.read"),
            ("admin.reports.handle", "admin.reports.write"),
            ("admin.config.view", "admin.app_config.read"),
            ("admin.config.edit", "admin.app_config.write"),
            ("admin.announcements.view", "admin.announcements.read"),
            ("admin.announcements.manage", "admin.announcements.write"),
            ("admin.audit.view", "admin.audit.read"),
            ("admin.rbac.view", "admin.rbac.manage"),
        ] {
            sqlx::query!(
                r#"
                INSERT INTO admin_role_permissions (role_id, permission_key)
                SELECT role_id, $2
                FROM admin_role_permissions
                WHERE permission_key = $1
                ON CONFLICT DO NOTHING
                "#,
                legacy_key,
                canonical_key
            )
            .execute(&self.pool)
            .await?;

            sqlx::query!(
                r#"
                INSERT INTO admin_user_permissions (user_id, permission_key)
                SELECT user_id, $2
                FROM admin_user_permissions
                WHERE permission_key = $1
                ON CONFLICT DO NOTHING
                "#,
                legacy_key,
                canonical_key
            )
            .execute(&self.pool)
            .await?;
        }

        Ok(())
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

    async fn user_has_permission_exact(&self, user_id: i64, permission_key: &str) -> Result<bool> {
        let result = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1
                FROM (
                    SELECT permission_key
                    FROM admin_user_permissions
                    WHERE user_id = $1
                    UNION
                    SELECT rp.permission_key
                    FROM admin_user_roles ur
                    JOIN admin_role_permissions rp ON ur.role_id = rp.role_id
                    WHERE ur.user_id = $1
                ) perms
                WHERE permission_key = $2
            ) as "exists!"
            "#,
            user_id,
            permission_key
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result)
    }

    /// 检查用户是否有特定权限（支持 canonical key + 历史 key 兼容）
    pub async fn user_has_permission(&self, user_id: i64, permission_key: &str) -> Result<bool> {
        // 超级管理员自动放行
        if self.is_super_admin(user_id).await? {
            return Ok(true);
        }

        for key in Self::compatible_permission_keys(permission_key) {
            if self.user_has_permission_exact(user_id, &key).await? {
                return Ok(true);
            }
        }

        Ok(false)
    }

    /// 检查用户是否具备 admin.audit.read_ip 权限（需同时具备 admin.audit.read）
    pub async fn user_has_read_ip_permission(&self, user_id: i64) -> Result<bool> {
        // 需要同时具备 admin.audit.read 和 admin.audit.read_ip 权限
        let has_audit_read = self
            .user_has_permission(user_id, "admin.audit.read")
            .await?;
        let has_read_ip = self
            .user_has_permission(user_id, "admin.audit.read_ip")
            .await?;

        Ok(has_audit_read && has_read_ip)
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
        let mut merged: BTreeMap<String, String> = BTreeMap::new();
        for row in rows {
            let key = Self::canonical_permission_key(&row.key);
            let description = if row.description.is_empty() {
                key.clone()
            } else {
                row.description
            };
            merged.entry(key).or_insert(description);
        }

        Ok(merged
            .into_iter()
            .map(|(key, description)| AdminPermissionListRow { key, description })
            .collect())
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
        let mut keys: BTreeSet<String> = BTreeSet::new();
        for row in rows {
            keys.insert(Self::canonical_permission_key(&row));
        }
        Ok(keys.into_iter().collect())
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
        let canonical_keys: Vec<String> = permission_keys
            .iter()
            .map(|key| Self::canonical_permission_key(key))
            .collect::<BTreeSet<_>>()
            .into_iter()
            .collect();

        let mut tx = self.pool.begin().await?;
        sqlx::query!(
            "DELETE FROM admin_role_permissions WHERE role_id = $1",
            role_id
        )
        .execute(&mut *tx)
        .await?;

        if !canonical_keys.is_empty() {
            sqlx::query!(
                r#"
                INSERT INTO admin_role_permissions (role_id, permission_key)
                SELECT $1, UNNEST($2::text[])
                "#,
                role_id,
                &canonical_keys
            )
            .execute(&mut *tx)
            .await?;
        }

        tx.commit().await?;
        Ok(())
    }

    /// 检查角色列表是否包含某权限
    pub async fn roles_have_permission(
        &self,
        role_ids: &[Uuid],
        permission_key: &str,
    ) -> Result<bool> {
        if role_ids.is_empty() {
            return Ok(false);
        }

        for key in Self::compatible_permission_keys(permission_key) {
            let result = sqlx::query!(
                r#"
                SELECT EXISTS(
                    SELECT 1 FROM admin_role_permissions
                    WHERE role_id = ANY($1) AND permission_key = $2
                ) as "exists!"
                "#,
                role_ids,
                key
            )
            .fetch_one(&self.pool)
            .await?;
            if result.exists {
                return Ok(true);
            }
        }

        Ok(false)
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
        let mut permissions: BTreeSet<String> = BTreeSet::new();
        for row in rows {
            if let Some(permission_key) = row {
                permissions.insert(Self::canonical_permission_key(&permission_key));
            }
        }
        Ok(permissions.into_iter().collect())
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
