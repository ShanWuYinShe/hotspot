use std::future::Future;
use std::pin::Pin;
use std::sync::Arc;
use uuid::Uuid;

use crate::ports::messages::MessageRepoPort;
use crate::ports::repositories::{RepoPortError, RepoPortResult};
use crate::repositories::PgMessageRepository;

#[derive(Clone)]
pub struct MessageRepoAdapter {
    inner: Arc<PgMessageRepository>,
}

impl MessageRepoAdapter {
    pub fn new(inner: Arc<PgMessageRepository>) -> Self { Self { inner } }
}

impl MessageRepoPort for MessageRepoAdapter {
    fn name(&self) -> &str { "MessageRepoAdapter" }

    fn check_user_in_conversation(&self, conversation_id: Uuid, user_id: i64) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>> {
        let inner = self.inner.clone();
        Box::pin(async move {
            inner.check_user_in_conversation(conversation_id, user_id).await
                .map_err(RepoPortError::from_repository_error)
        })
    }

    fn check_user_in_group(&self, group_id: Uuid, user_id: i64) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>> {
        let inner = self.inner.clone();
        Box::pin(async move {
            inner.check_user_in_group(group_id, user_id).await
                .map_err(RepoPortError::from_repository_error)
        })
    }
}


