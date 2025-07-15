//! 管理员路由
//! 提供系统管理相关的API接口

use axum::{
    Router,
    routing::post,
};
use std::sync::Arc;

use crate::{
    controllers::user_id_admin_controller,
    middleware::auth::AppState,
};

/// 管理员路由
pub fn admin_routes() -> Router<Arc<AppState>> {
    Router::new()
        // 用户ID分区管理接口 - 全部改为POST方法
        .route("/user-id/stats", post(user_id_admin_controller::get_partition_stats))
        .route("/user-id/validate", post(user_id_admin_controller::validate_user_id))
        .route("/user-id/info", post(user_id_admin_controller::get_user_type_info))
        .route("/user-id/ranges", post(user_id_admin_controller::get_user_type_ranges))
        .route("/user-id/generate", post(user_id_admin_controller::generate_user_ids))
        .route("/user-id/format", post(user_id_admin_controller::format_user_id))
}
