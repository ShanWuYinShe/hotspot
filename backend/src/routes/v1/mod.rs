pub mod public;
pub mod protected;

use axum::Router;
use crate::state::AppState;

pub fn public_routes() -> Router<AppState> {
    Router::new()
        .nest("/public/auth", public::auth::routes())
        .nest("/public/user", public::user::routes())
        .nest("/public/group_chats", public::group::routes())
        .nest("/public/moments", public::moment::routes())
        .nest("/public/comments", public::comment::routes())
        .nest("/public/system", public::system::routes())
        .nest("/public/leaderboard", public::leaderboard::routes())
        .nest("/public/tags", public::tag::routes())
}

pub fn protected_routes() -> Router<AppState> {
    Router::new()
        .nest("/protected/auth", protected::auth::routes())
        .nest("/protected/user", protected::user::routes())
        .nest("/protected/group_chats", protected::group::routes())
        .nest("/protected/private_chats", protected::private_chats::routes())
        .nest("/protected/private_messages", protected::private_messages::routes())
        .nest("/protected/group_messages", protected::group_messages::routes())
        .nest("/protected/messages", protected::messages::routes())
        .nest("/protected/moments", protected::moment::routes())
        .nest("/protected/comments", protected::comment::routes())
        .nest("/protected/reports", protected::report::routes())
        .nest("/protected/inbox", protected::inbox::routes())
        .nest("/protected/subscriptions", protected::subscription::routes())
        .nest("/protected/feed", protected::feed::routes())
        .nest("/protected/media", protected::media::routes())
        .nest("/protected/leaderboard", protected::leaderboard::routes())
        // Admin routes: /api/v1/admin/...
        .nest("/admin", protected::admin::admin_routes())
        .nest("/admin/config", protected::system_config::routes())
}
