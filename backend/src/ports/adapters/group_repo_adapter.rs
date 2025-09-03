use std::future::Future;
use std::pin::Pin;
use std::sync::Arc;
use uuid::Uuid;

use crate::ports::repositories::{GroupRepoPort, RepoPortError, RepoPortResult};
use crate::repositories::{PgGroupRepository, GroupRepository};

#[derive(Clone)]
pub struct GroupRepoAdapter { inner: Arc<PgGroupRepository> }

impl GroupRepoAdapter { pub fn new(inner: Arc<PgGroupRepository>) -> Self { Self { inner } } }

impl GroupRepoPort for GroupRepoAdapter {
    fn name(&self) -> &str { "GroupRepoAdapter" }
    fn check_group_exists(&self, group_id: Uuid) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>> {
        let inner = self.inner.clone();
        Box::pin(async move {
            // 使用 GroupRepository trait 方法
            inner.check_group_exists(group_id).await.map_err(RepoPortError::from_repository_error)
        })
    }
    fn check_user_in_group(&self, group_id: Uuid, user_id: i64) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>> {
        let inner = self.inner.clone();
        Box::pin(async move { inner.check_user_in_group(group_id, user_id).await.map_err(RepoPortError::from_repository_error) })
    }
}


