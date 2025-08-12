use crate::{
    models::api::user::AuthUser,
    repositories::PgUserRepository,
    services::{
        MomentService, ConfigService, GroupService, HealthService, MessageService, 
        UserService,
        notification::WebSocketNotificationService,
    },
    utils::{jwt::verify_jwt_token, response::AppError},
    ws::manager::WebSocketManager,
};
#[cfg(feature = "jobs")]
use crate::jobs::JobService;
use axum::{
    body::Body,
    extract::{Request, State},
    http::{header, Method, StatusCode},
    middleware::Next,
    response::Response,
};
use std::sync::Arc;
use tracing::error;
use crate::utils::metrics::AppMetrics;
use crate::utils::events::EventBus;

pub struct AppState {
    pub config_service: Arc<ConfigService>,
    pub user_service: Arc<UserService>,
    pub user_repository: Arc<PgUserRepository>,
    pub group_service: Arc<GroupService>,
    pub message_service: Arc<MessageService>,
    pub moment_service: Arc<MomentService>,
    pub ws_manager: Arc<WebSocketManager>,
    pub websocket_notification_service: Arc<WebSocketNotificationService>,
    #[cfg(feature = "jobs")]
    pub job_service: Option<Arc<JobService>>,
    pub health_service: Arc<HealthService>,
    pub metrics: Arc<AppMetrics>,
    pub event_bus: Arc<EventBus>,
}

// 认证中间件，用于验证请求中的JWT令牌
pub async fn auth_middleware(
    State(state): State<Arc<AppState>>,
    mut request: Request<Body>,
    next: Next,
) -> Result<Response<Body>, AppError> {
    // 处理OPTIONS预检请求
    if request.method() == Method::OPTIONS {
        return Ok(Response::builder()
            .status(StatusCode::OK)
            .body(Body::empty())
            .unwrap());
    }

    // 从路径判断是否需要认证
    let path = request.uri().path();
    let method = request.method();
    
    tracing::debug!("Auth middleware: {} {}", method, path);

    // 特别处理WebSocket路径 - 直接放行
    if path.starts_with("/ws") || path.starts_with("/ws/") {
        tracing::debug!("WebSocket path detected, skipping auth: {}", path);
        return Ok(next.run(request).await);
    }

    // 如果是公开路径，直接放行
    if state.config_service.is_public_path(path) {
        tracing::debug!("Public path detected, skipping auth: {}", path);
        return Ok(next.run(request).await);
    }
    
    tracing::debug!("Protected path, checking auth: {}", path);

    // 获取认证头中的token
    let auth_header = match request.headers().get(header::AUTHORIZATION) {
        Some(header) => match header.to_str() {
            Ok(h) => h,
            Err(_) => {
                tracing::warn!("Invalid Authorization header format for {}", path);
                return Err(AppError::Unauthorized("认证头格式无效".to_string()));
            }
        },
        None => {
            tracing::warn!("Missing Authorization header for {}", path);
            return Err(AppError::Unauthorized("缺少认证头".to_string()));
        }
    };

    // 验证Bearer格式
    if !auth_header.starts_with("Bearer ") {
        tracing::warn!("Invalid Bearer format for {}: {}", path, auth_header);
        return Err(AppError::Unauthorized("认证头格式无效".to_string()));
    }

    // 提取token
    let token = auth_header.trim_start_matches("Bearer ").trim();
    if token.is_empty() {
        tracing::warn!("Empty token for {}", path);
        return Err(AppError::Unauthorized("令牌为空".to_string()));
    }

    tracing::debug!("Token extracted for {}, length: {}", path, token.len());

    // 验证token - 从config_service获取jwt_secret
    let jwt_secret = &state.config_service.config.auth.jwt_secret;
    let claims = match verify_jwt_token(token, jwt_secret) {
        Ok(claims) => {
            tracing::debug!("Token verified successfully for {}, user_id: {}", path, claims.sub);
            claims
        }
        Err(e) => {
            tracing::warn!("Token verification failed for {}: {:?}", path, e);
            return Err(AppError::Unauthorized("令牌无效".to_string()));
        }
    };

    // 获取用户ID
    let user_id = claims.sub;

    // 获取用户信息 - 使用专门用于认证的get_auth_info方法
    let user = state
        .user_service
        .get_auth_info(user_id)
        .await
        .map_err(|e| {
            error!("查询用户失败: {:?}", e);
            AppError::InternalServerError("服务器错误".to_string())
        })?;

    request.extensions_mut().insert(AuthUser {
        user_id: user.user_id,
        nickname: user.nickname,
    });

    // 执行请求
    Ok(next.run(request).await)
}
