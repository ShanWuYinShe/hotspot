pub mod public;
pub mod protected;

use axum::Router;
use crate::state::AppState;

pub fn routes() -> Router<AppState> {
    Router::new()
        // 公开端点
        .nest("/public/auth", public::auth::routes())
        .nest("/public/user", public::user::routes())
        .nest("/public/group", public::group::routes())
        .nest("/public/moment", public::moment::routes())
        .nest("/public/comment", public::comment::routes())
        .nest("/public/system", public::system::routes())
        .nest("/public/leaderboard", public::leaderboard::routes())
        
        // 受保护端点
        .nest("/protected/auth", protected::auth::routes())
        .nest("/protected/user", protected::user::routes())
        .nest("/protected/group", protected::group::routes())
        .nest("/protected/chat", protected::chat::routes())
        .nest("/protected/chat_messages", protected::chat_messages::routes())
        .nest("/protected/moment", protected::moment::routes())
        .nest("/protected/comment", protected::comment::routes())
        .nest("/protected/report", protected::report::routes())
        .nest("/protected/inbox", protected::inbox::routes())
        .nest("/protected/subscription", protected::subscription::routes())
        .nest("/protected/feed", protected::feed::routes())
        
        // 管理员端点（在 handler 中检查管理员权限）
        .nest("/protected/admin", protected::admin::admin_routes())
}

