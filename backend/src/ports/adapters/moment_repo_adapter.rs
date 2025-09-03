//! 动态仓储适配器 - 将 PgMomentRepository 适配为 MomentRepoPort

use std::future::Future;
use std::pin::Pin;
use std::sync::Arc;

use crate::ports::repositories::{RepoPortError, RepoPortResult, MomentRepoPort};
use crate::repositories::moment_repository::PgMomentRepository;

pub struct MomentRepoAdapter {
    repo: Arc<PgMomentRepository>,
}

impl MomentRepoAdapter {
    pub fn new(repo: Arc<PgMomentRepository>) -> Self {
        Self { repo }
    }
}

impl MomentRepoPort for MomentRepoAdapter {
    fn name(&self) -> &str {
        "MomentRepoAdapter"
    }

    fn like_moment(&self, moment_id: &str, user_id: i64) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>> {
        let repo = self.repo.clone();
        let moment_id = moment_id.to_string();
        Box::pin(async move {
            match repo.like_moment(&moment_id, user_id).await {
                Ok(liked) => Ok(liked),
                Err(e) => Err(RepoPortError::from_repository_error(e)),
            }
        })
    }

    fn unlike_moment(&self, moment_id: &str, user_id: i64) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>> {
        let repo = self.repo.clone();
        let moment_id = moment_id.to_string();
        Box::pin(async move {
            match repo.unlike_moment(&moment_id, user_id).await {
                Ok(unliked) => Ok(unliked),
                Err(e) => Err(RepoPortError::from_repository_error(e)),
            }
        })
    }
}
