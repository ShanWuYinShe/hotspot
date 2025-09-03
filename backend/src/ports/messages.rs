//! Message 仓储端口（最小子集示例）

use std::future::Future;
use std::pin::Pin;
use uuid::Uuid;

use super::repositories::RepoPortResult;

pub trait MessageRepoPort: Send + Sync {
    fn name(&self) -> &str;
    fn check_user_in_conversation(&self, conversation_id: Uuid, user_id: i64) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>>;
    fn check_user_in_group(&self, group_id: Uuid, user_id: i64) -> Pin<Box<dyn Future<Output = RepoPortResult<bool>> + Send>>;
}


