use uuid::Uuid;

use crate::{
    config::Config,
    error::{AppError, Result},
    repositories::{
        admin_rbac_repo::{AdminRbacRepository, AdminRoleRow, AdminWithRolesRow},
        user_repo::UserRepository,
        user_settings_repo::UserSettingsRepository,
    },
    types::enums::UserType,
    utils::{id_gen::IdGenerator, password::PasswordUtil},
};

#[derive(Clone)]
pub struct AdminRbacService {
    repo: AdminRbacRepository,
    user_repo: UserRepository,
    user_settings_repo: UserSettingsRepository,
}

impl AdminRbacService {
    pub fn new(repo: AdminRbacRepository, user_repo: UserRepository, user_settings_repo: UserSettingsRepository) -> Self {
        Self {
            repo,
            user_repo,
            user_settings_repo,
        }
    }

    pub async fn ensure_super_admin_bootstrap(&self, config: &Config) -> Result<()> {
        // 首先检查是否存在超级管理员
        let existing = sqlx::query_scalar::<_, bool>(
            r#"
            SELECT EXISTS(SELECT 1 FROM admin_users WHERE is_super = TRUE)
            "#,
        )
        .fetch_one(self.repo.pool())
        .await?;

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
            .unwrap_or_else(|| "superadmin".to_string());

        // 如果 login_id 已存在用户，直接将其升级为超级管理员
        if let Some(user) = self.user_repo.find_by_login_id(&login_id).await? {
            self.repo.upsert_admin_user(user.user_id, true).await?;
            self.repo.ensure_user_type_admin(user.user_id).await?;
            tracing::warn!(
                "SUPER ADMIN bootstrap: promoted existing user_id={} login_id={}",
                user.user_id,
                login_id
            );
            return Ok(());
        }

        // 创建新的超级管理员账号
        let user_id = IdGenerator::generate_user_id(self.repo.pool()).await?;
        let password_hash = PasswordUtil::hash_password(&password)?;

        let user = self
            .user_repo
            .create(
                user_id,
                UserType::Admin,
                Some(login_id.clone()),
                "Super Admin".to_string(),
                Some(password_hash),
            )
            .await?;

        // 创建默认 user_settings（使用 DB 默认值）
        self.user_settings_repo.create(user.user_id).await?;

        // 标记为超级管理员
        self.repo.upsert_admin_user(user.user_id, true).await?;
        self.repo.ensure_user_type_admin(user.user_id).await?;

        tracing::warn!(
            "SUPER ADMIN bootstrap: created user_id={} login_id={}",
            user.user_id,
            login_id
        );

        Ok(())
    }

    pub async fn is_admin(&self, user_id: i64) -> Result<bool> {
        Ok(self.repo.get_admin_user(user_id).await?.is_some())
    }

    pub async fn is_super_admin(&self, user_id: i64) -> Result<bool> {
        Ok(self
            .repo
            .get_admin_user(user_id)
            .await?
            .map(|u| u.is_super)
            .unwrap_or(false))
    }

    pub async fn require_permission(&self, user_id: i64, permission_key: &str) -> Result<()> {
        // 必须是管理员（此方法由 admin_auth_middleware 调用，已确保是管理员）
        // 再校验权限（超级管理员会自动放行）
        let ok = self.repo.user_has_permission(user_id, permission_key).await?;
        if !ok {
            return Err(AppError::AdminPermissionDenied(format!("缺少权限: {}", permission_key)));
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
        self.repo.create_role(name, description).await
    }

    pub async fn update_role(
        &self,
        role_id: Uuid,
        name: Option<&str>,
        description: Option<Option<&str>>,
    ) -> Result<()> {
        self.repo.update_role(role_id, name, description).await
    }

    pub async fn delete_role(&self, role_id: Uuid) -> Result<()> {
        self.repo.delete_role(role_id).await
    }

    pub async fn set_role_permissions(&self, role_id: Uuid, permission_keys: &[String]) -> Result<()> {
        self.repo.set_role_permissions(role_id, permission_keys).await
    }

    pub async fn list_admins(&self) -> Result<Vec<AdminWithRolesRow>> {
        self.repo.list_admins().await
    }

    pub async fn get_me(&self, user_id: i64) -> Result<(bool, Vec<String>, Vec<AdminRoleRow>)> {
        // 必须是管理员
        if !self.is_admin(user_id).await? {
            return Err(AppError::Forbidden("需要管理员权限".to_string()));
        }

        let is_super = self.is_super_admin(user_id).await?;
        let permissions = self.repo.list_user_permissions(user_id).await?;
        let roles = self.repo.list_user_roles(user_id).await?;
        Ok((is_super, permissions, roles))
    }

    pub async fn grant_admin(&self, operator_id: i64, target_user_id: i64, is_super: bool, role_ids: &[Uuid]) -> Result<()> {
        // 只有超级管理员可以授予/撤销超级管理员
        if is_super && !self.is_super_admin(operator_id).await? {
            return Err(AppError::Forbidden("只有超级管理员可以授予超级管理员权限".to_string()));
        }

        self.repo.upsert_admin_user(target_user_id, is_super).await?;
        self.repo.ensure_user_type_admin(target_user_id).await?;
        self.repo.replace_admin_roles(target_user_id, role_ids).await?;
        Ok(())
    }

    pub async fn revoke_admin(&self, operator_id: i64, target_user_id: i64) -> Result<()> {
        // 只有超级管理员可以撤销超级管理员
        if self.is_super_admin(target_user_id).await? && !self.is_super_admin(operator_id).await? {
            return Err(AppError::Forbidden("只有超级管理员可以撤销超级管理员权限".to_string()));
        }

        self.repo.remove_admin_user(target_user_id).await?;
        self.repo.revoke_user_type_admin(target_user_id).await?;
        Ok(())
    }
}

