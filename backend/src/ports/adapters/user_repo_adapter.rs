//! 用户仓储适配器 - 将 PgUserRepository 适配为 UserRepoPort

use std::future::Future;
use std::pin::Pin;
use std::sync::Arc;

use crate::ports::repositories::{RepoPortError, RepoPortResult, UserRepoPort};
use crate::repositories::user_repository::{PgUserRepository, UserRepository};

pub struct UserRepoAdapter {
    repo: Arc<PgUserRepository>,
}

impl UserRepoAdapter {
    pub fn new(repo: Arc<PgUserRepository>) -> Self {
        Self { repo }
    }
}

impl UserRepoPort for UserRepoAdapter {
    fn name(&self) -> &str {
        "UserRepoAdapter"
    }

    fn is_login_id_taken(&self, login_id: &str) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>> {
        let repo = self.repo.clone();
        let login_id = login_id.to_string();
        Box::pin(async move {
            match repo.is_login_id_taken(&login_id).await {
                Ok(taken) => Ok(taken),
                Err(e) => Err(RepoPortError::from_repository_error(e)),
            }
        })
    }

    fn is_password_set(&self, user_id: i64) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>> {
        let repo = self.repo.clone();
        Box::pin(async move {
            match repo.is_password_set(user_id).await {
                Ok(is_set) => Ok(is_set),
                Err(e) => Err(RepoPortError::from_repository_error(e)),
            }
        })
    }

    fn is_temp_user(&self, user_id: i64) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>> {
        let repo = self.repo.clone();
        Box::pin(async move {
            match repo.is_temp_user(user_id).await {
                Ok(is_temp) => Ok(is_temp),
                Err(e) => Err(RepoPortError::from_repository_error(e)),
            }
        })
    }
}
