use uuid::Uuid;

use crate::{
    config::Config,
    error::{AppError, Result},
    repositories::{
        admin::rbac_repo::{AdminRbacRepository, AdminRoleRow, AdminWithRolesRow},
        user::settings_repo::UserSettingsRepository,
        user_repo::UserRepository,
    },
    utils::password::PasswordUtil,
};

#[derive(Clone)]
pub struct AdminRbacService {
    repo: AdminRbacRepository,
    user_repo: UserRepository,
    user_settings_repo: UserSettingsRepository,
}

impl AdminRbacService {
    pub fn new(
        repo: AdminRbacRepository,
        user_repo: UserRepository,
        user_settings_repo: UserSettingsRepository,
    ) -> Self {
        Self {
            repo,
            user_repo,
            user_settings_repo,
        }
    }

    pub async fn ensure_super_admin_bootstrap(&self, config: &Config) -> Result<()> {
        // 启动即对齐权限 key 目录，避免 read/write 与历史 view/edit 键名不一致。
        self.repo.ensure_permission_catalog_alignment().await?;

        // 首先检查是否存在超级管理员
        let existing = self.repo.super_admin_exists().await?;

        // 如果已存在超级管理员，直接返回
        if existing {
            tracing::info!("Super admin already exists, bootstrap skipped");
            return Ok(());
        }

        // 如果没有超级管理员，必须通过 SUPER_ADMIN_PASSWORD 环境变量创建
        let password = match &config.super_admin_password {
            Some(p) if !p.is_empty() => p.clone(),
            _ => {
                return Err(AppError::InternalError(
                    "系统启动失败：未检测到超级管理员，且 SUPER_ADMIN_PASSWORD 环境变量未设置。请设置 SUPER_ADMIN_PASSWORD 环境变量以创建初始超级管理员。".to_string()
                ));
            }
        };

        let login_id = config
            .super_admin_login_id
            .clone()
            .unwrap_or("superadmin".to_string());

        // 如果 login_id 已存在用户，直接将其升级为超级管理员
        if let Some(user) = self.user_repo.find_by_identifier(&login_id).await? {
            self.repo.upsert_admin_user(user.user_id, true).await?;
            tracing::warn!(
                "SUPER ADMIN bootstrap: promoted existing user_id={} login_id={}",
                user.user_id,
                login_id
            );
            return Ok(());
        }

        // 创建新的超级管理员账号
        let password_hash = PasswordUtil::hash_password(&password)?;

        let user = self
            .user_repo
            .create(
                login_id.clone(),
                password_hash,
                Some("Super Admin".to_string()),
                None,
                None,
            )
            .await?;

        // 创建默认 user_settings 并禁用搜索/私聊
        self.user_settings_repo.create(user.user_id).await?;
        use crate::repositories::user::settings_repo::UpdateSettingsParams;
        use crate::types::enums::PrivateChatPermission;
        let params = UpdateSettingsParams {
            searchable: Some(false),
            phone_searchable: Some(false),
            nickname_searchable: Some(false),
            location_searchable: Some(false),
            allow_private_chat: Some(PrivateChatPermission::Nobody),
            ..Default::default()
        };
        self.user_settings_repo
            .update(user.user_id, &params)
            .await?;

        // 标记为超级管理员
        self.repo.upsert_admin_user(user.user_id, true).await?;

        tracing::warn!(
            "SUPER ADMIN bootstrap: created user_id={} login_id={}",
            user.user_id,
            login_id
        );

        Ok(())
    }

    pub async fn is_admin(&self, user_id: i64) -> Result<bool> {
        self.repo.is_admin(user_id).await
    }

    pub async fn is_super_admin(&self, user_id: i64) -> Result<bool> {
        self.repo.is_super_admin(user_id).await
    }

    pub async fn has_permission(&self, user_id: i64, permission_key: &str) -> Result<bool> {
        self.repo.user_has_permission(user_id, permission_key).await
    }

    pub async fn require_permission(&self, user_id: i64, permission_key: &str) -> Result<()> {
        // 必须是管理员（此方法由 admin_auth_middleware 调用，已确保是管理员）
        // 再校验权限（超级管理员会自动放行）
        let ok = self
            .repo
            .user_has_permission(user_id, permission_key)
            .await?;
        if !ok {
            return Err(AppError::AdminPermissionDenied(format!(
                "缺少权限: {}",
                permission_key
            )));
        }
        Ok(())
    }

    pub async fn list_permissions(&self) -> Result<Vec<(String, String)>> {
        Ok(self
            .repo
            .list_permissions()
            .await?
            .into_iter()
            .map(|p| (p.key, p.description))
            .collect())
    }

    pub async fn list_roles(&self) -> Result<Vec<(AdminRoleRow, Vec<String>)>> {
        let roles = self.repo.list_roles().await?;
        let mut out = Vec::with_capacity(roles.len());
        for r in roles {
            let perms = self.repo.get_role_permissions(r.role_id).await?;
            out.push((r, perms));
        }
        Ok(out)
    }

    pub async fn create_role(&self, name: &str, description: Option<&str>) -> Result<Uuid> {
        let started_at = std::time::Instant::now();
        tracing::info!(
            role_name = %name,
            has_description = description.is_some(),
            "admin_rbac.create_role started"
        );
        let role_id = self.repo.create_role(name, description).await?;
        tracing::info!(
            role_id = %role_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "admin_rbac.create_role completed"
        );
        Ok(role_id)
    }

    pub async fn update_role(
        &self,
        role_id: Uuid,
        name: Option<&str>,
        description: Option<Option<&str>>,
    ) -> Result<()> {
        let started_at = std::time::Instant::now();
        tracing::info!(
            role_id = %role_id,
            has_name = name.is_some(),
            has_description = description.is_some(),
            "admin_rbac.update_role started"
        );
        self.repo.update_role(role_id, name, description).await?;
        tracing::info!(
            role_id = %role_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "admin_rbac.update_role completed"
        );
        Ok(())
    }

    pub async fn delete_role(&self, role_id: Uuid) -> Result<()> {
        let started_at = std::time::Instant::now();
        tracing::info!(role_id = %role_id, "admin_rbac.delete_role started");
        self.repo.delete_role(role_id).await?;
        tracing::info!(
            role_id = %role_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "admin_rbac.delete_role completed"
        );
        Ok(())
    }

    pub async fn set_role_permissions(
        &self,
        operator_id: i64,
        role_id: Uuid,
        permission_keys: &[String],
    ) -> Result<()> {
        let started_at = std::time::Instant::now();
        tracing::info!(
            operator_id,
            role_id = %role_id,
            permission_count = permission_keys.len(),
            has_wildcard = permission_keys.iter().any(|key| key == "*"),
            "admin_rbac.set_role_permissions started"
        );
        if permission_keys.iter().any(|key| key == "*") && !self.is_super_admin(operator_id).await?
        {
            tracing::warn!(
                operator_id,
                role_id = %role_id,
                "admin_rbac.set_role_permissions rejected: wildcard requires super admin"
            );
            return Err(AppError::AdminPermissionDenied(
                "只有超级管理员可以设置包含 '*' 的角色权限".to_string(),
            ));
        }

        self.repo
            .set_role_permissions(role_id, permission_keys)
            .await?;
        tracing::info!(
            operator_id,
            role_id = %role_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "admin_rbac.set_role_permissions completed"
        );
        Ok(())
    }

    pub async fn list_admins(
        &self,
        limit: Option<i32>,
        cursor: Option<String>,
    ) -> Result<(Vec<AdminWithRolesRow>, Option<String>)> {
        let limit = crate::utils::validation::Validator::validate_limit(limit, 50)? as usize;
        self.repo.list_admins(limit, cursor.as_deref()).await
    }

    pub async fn get_me(&self, user_id: i64) -> Result<(bool, Vec<String>, Vec<AdminRoleRow>)> {
        // 一次查询获取管理员信息（同时完成存在性校验和 is_super 判断）
        let admin = self
            .repo
            .get_admin_user(user_id)
            .await?
            .ok_or_else(|| AppError::Forbidden("需要管理员权限".to_string()))?;

        let permissions = self.repo.list_user_permissions(user_id).await?;
        let roles = self.repo.list_user_roles(user_id).await?;
        Ok((admin.is_super, permissions, roles))
    }

    pub async fn grant_admin(
        &self,
        operator_id: i64,
        target_user_id: i64,
        is_super: bool,
        role_ids: &[Uuid],
    ) -> Result<()> {
        let started_at = std::time::Instant::now();
        tracing::info!(
            operator_id,
            target_user_id,
            is_super,
            role_count = role_ids.len(),
            "admin_rbac.grant_admin started"
        );
        let operator_is_super = self.is_super_admin(operator_id).await?;

        if is_super {
            if !operator_is_super {
                tracing::warn!(
                    operator_id,
                    target_user_id,
                    "admin_rbac.grant_admin rejected: only super admin can grant super"
                );
                return Err(AppError::Forbidden(
                    "只有超级管理员可以授予超级管理员权限".to_string(),
                ));
            }
            if !role_ids.is_empty() {
                tracing::warn!(
                    operator_id,
                    target_user_id,
                    "admin_rbac.grant_admin rejected: super grant cannot include role_ids"
                );
                return Err(AppError::InvalidFormat(
                    "grant_all_permissions 与 role_ids 互斥".to_string(),
                ));
            }
        } else {
            if role_ids.is_empty() {
                tracing::warn!(
                    operator_id,
                    target_user_id,
                    "admin_rbac.grant_admin rejected: role_ids is empty"
                );
                return Err(AppError::MissingParameter("role_ids".to_string()));
            }
            if !operator_is_super && self.repo.roles_have_permission(role_ids, "*").await? {
                tracing::warn!(
                    operator_id,
                    target_user_id,
                    "admin_rbac.grant_admin rejected: wildcard role requires super admin"
                );
                return Err(AppError::AdminPermissionDenied(
                    "只有超级管理员可以授予包含 '*' 的角色".to_string(),
                ));
            }
        }

        self.repo
            .upsert_admin_user(target_user_id, is_super)
            .await?;
        self.repo
            .replace_admin_roles(target_user_id, role_ids)
            .await?;
        tracing::info!(
            operator_id,
            target_user_id,
            is_super,
            role_count = role_ids.len(),
            elapsed_ms = started_at.elapsed().as_millis(),
            "admin_rbac.grant_admin completed"
        );
        Ok(())
    }

    pub async fn revoke_admin(&self, operator_id: i64, target_user_id: i64) -> Result<()> {
        let started_at = std::time::Instant::now();
        tracing::info!(
            operator_id,
            target_user_id,
            "admin_rbac.revoke_admin started"
        );
        // 只有超级管理员可以撤销超级管理员
        if self.is_super_admin(target_user_id).await? && !self.is_super_admin(operator_id).await? {
            tracing::warn!(
                operator_id,
                target_user_id,
                "admin_rbac.revoke_admin rejected: only super admin can revoke super admin"
            );
            return Err(AppError::Forbidden(
                "只有超级管理员可以撤销超级管理员权限".to_string(),
            ));
        }

        self.repo.remove_admin_user(target_user_id).await?;
        tracing::info!(
            operator_id,
            target_user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "admin_rbac.revoke_admin completed"
        );
        Ok(())
    }
}
