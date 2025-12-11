use axum::{routing::post, Router};

use crate::{handlers::{system, geocoding}, state::AppState};

/// Public 系统路由
/// 
/// 提供系统健康检查、地理编码等公开接口
pub fn routes() -> Router<AppState> {
    Router::new()
        // 系统健康检查
        .route("/health_check", post(system::health))
        // 逆地理编码（将坐标转换为地址）
        .route("/reverse_geocode", post(geocoding::reverse_geocode))
}










