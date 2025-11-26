use axum::{routing::post, Router};

use crate::{handlers::private_chats, state::AppState};

/// Protected 私聊管理路由
///
/// 这些接口需要认证，必须提供有效的 access_token
/// 严格按照 api_spec.mdc 定义
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/create", post(private_chats::create)) // ✅ 发起私聊
        .route("/get_list", post(private_chats::get_list)) // ✅ 获取私聊列表
        .route("/open_the_door", post(private_chats::open_the_door)) // ✅ 获取私聊详情
        .route("/update_settings", post(private_chats::update_settings)) // ✅ 更新私聊设置
        .route("/delete", post(private_chats::delete)) // ✅ 删除私聊
}
