use axum::Router;

use crate::state::AppState;

/// Protected 系统路由
///
/// 额度敏感的系统能力需要登录后调用，避免匿名刷第三方额度。
pub fn routes() -> Router<AppState> {
    Router::new()
}
