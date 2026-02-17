mod content_events;
mod group_events;
mod message_events;
mod system_events;
mod user_events;

pub use content_events::{persist_notification as content_persist, send_websocket as content_ws};
pub use group_events::{persist_notification as group_persist, send_websocket as group_ws};
pub use message_events::{persist_notification as message_persist, send_websocket as message_ws};
pub use system_events::{persist_notification as system_persist, send_websocket as system_ws};
pub use user_events::{persist_notification as user_persist, send_websocket as user_ws};

use crate::repositories::user_repo::UserRepository;

/// 获取用户信息，供各 handler 使用
pub async fn get_user_info(
    user_repo: &UserRepository,
    user_id: i64,
) -> crate::dto::response::UserInfo {
    match user_repo.find_by_id(user_id).await {
        Ok(Some(user)) => crate::dto::response::UserInfo::new(
            user.user_id,
            user.nickname.unwrap_or_default(),
            user.avatar_url,
        ),
        _ => crate::dto::response::UserInfo::new(
            user_id,
            crate::constants::UNKNOWN_USER_NICKNAME,
            None,
        ),
    }
}
