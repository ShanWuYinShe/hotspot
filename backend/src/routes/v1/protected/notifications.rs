use axum::{routing::post, Router};

use crate::{handlers::notifications, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/", post(notifications::list_notifications)) // POST for list? API Spec says GET usually for list but spec says POST for all.
        // Wait, backend_doc.mdc says "所有 HTTP API 统一使用 POST".
        // api_spec_endpoints.mdc says: "POST /api/v1/protected/notifications - 获取通知列表"
        // Let's check api_spec_endpoints.mdc again for "list".
        // It says: "**POST /api/v1/protected/notifications - 获取通知列表**"
        // So I should use post. But my handler used `Query`.
        // If it is POST, params are usually in Body.
        // Let's check other list APIs. e.g. `list_moments`.
        // They use `Json` or `Query`?
        // api_spec_base.mdc says "POST ... search_by_xxx ... list ..."
        // Usually POST body is used for parameters.
        // Let's check `group_chats.rs` or `moments.rs`.
        .route("/unread_count", post(notifications::get_unread_count))
        .route("/mark_read", post(notifications::mark_read))
}
