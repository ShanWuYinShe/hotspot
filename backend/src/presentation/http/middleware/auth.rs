/// 认证中间件
/// 
/// 迁移自传统middleware/auth.rs，适配新的端口适配器架构
/// 使用新的应用状态和JWT验证逻辑

use axum::{
    body::Body,
    extract::{Request, State},
    http::{header, Method, StatusCode},
    middleware::Next,
    response::Response,
};
use std::sync::Arc;
use tracing::{debug, warn, error};
use serde::{Deserialize, Serialize};

// 移除未使用的导入
use crate::ports::UserRepositoryPort;

/// 认证用户信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthUser {
    pub user_id: i64,
    pub nickname: String,
}

/// JWT Claims
#[derive(Debug, Serialize, Deserialize)]
pub struct JwtClaims {
    pub sub: i64,  // 用户ID
    pub exp: usize, // 过期时间
    pub iat: usize, // 签发时间
}

/// JWT认证配置
#[derive(Debug, Clone)]
pub struct JwtConfig {
    pub jwt_secret: String,
    pub public_paths: Vec<String>,
}

impl Default for JwtConfig {
    fn default() -> Self {
        Self {
            jwt_secret: "default_secret".to_string(),
            public_paths: vec![
                "/api/v1/user/register".to_string(),
                "/api/v1/user/login".to_string(),
                "/api/v1/user/create_temp".to_string(),
                "/api/v1/user/refresh_token".to_string(),
                "/api/v1/health".to_string(),
                "/api/v1/metrics".to_string(),
                "/docs".to_string(),
                "/swagger-ui".to_string(),
            ],
        }
    }
}

impl JwtConfig {
    /// 检查路径是否为公开路径
    pub fn is_public_path(&self, path: &str) -> bool {
        self.public_paths.iter().any(|p| path.starts_with(p))
    }
}

/// 认证中间件状态
#[derive(Clone)]
pub struct AuthMiddlewareState<U: UserRepositoryPort> {
    pub user_repository: Arc<U>,
    pub auth_config: JwtConfig,
}

impl<U: UserRepositoryPort> AuthMiddlewareState<U> {
    pub fn new(user_repository: Arc<U>, auth_config: JwtConfig) -> Self {
        Self {
            user_repository,
            auth_config,
        }
    }
}

/// 认证中间件
/// 
/// 验证JWT令牌并注入用户信息到请求扩展中
pub async fn auth_middleware<U: UserRepositoryPort + 'static>(
    State(state): State<AuthMiddlewareState<U>>,
    mut request: Request<Body>,
    next: Next,
) -> Result<Response<Body>, AuthError> {
    // 处理OPTIONS预检请求
    if request.method() == Method::OPTIONS {
        return Ok(Response::builder()
            .status(StatusCode::OK)
            .body(Body::empty())
            .unwrap());
    }

    let path = request.uri().path();
    let method = request.method();
    
    debug!("Auth middleware: {} {}", method, path);

    // 特别处理WebSocket路径 - 直接放行（WebSocket有自己的认证逻辑）
    if path.starts_with("/ws") || path.starts_with("/ws/") {
        debug!("WebSocket path detected, skipping auth: {}", path);
        return Ok(next.run(request).await);
    }

    // 如果是公开路径，直接放行
    if state.auth_config.is_public_path(path) {
        debug!("Public path detected, skipping auth: {}", path);
        return Ok(next.run(request).await);
    }
    
    debug!("Protected path, checking auth: {}", path);

    // 获取认证头中的token
    let auth_header = match request.headers().get(header::AUTHORIZATION) {
        Some(header) => match header.to_str() {
            Ok(h) => h,
            Err(_) => {
                warn!("Invalid Authorization header format for {}", path);
                return Err(AuthError::InvalidHeaderFormat);
            }
        },
        None => {
            warn!("Missing Authorization header for {}", path);
            return Err(AuthError::MissingAuthHeader);
        }
    };

    // 验证Bearer格式
    if !auth_header.starts_with("Bearer ") {
        warn!("Invalid Bearer format for {}: {}", path, auth_header);
        return Err(AuthError::InvalidBearerFormat);
    }

    // 提取token
    let token = auth_header.trim_start_matches("Bearer ").trim();
    if token.is_empty() {
        warn!("Empty token for {}", path);
        return Err(AuthError::EmptyToken);
    }

    debug!("Token extracted for {}, length: {}", path, token.len());

    // 验证token
    let claims = verify_jwt_token(token, &state.auth_config.jwt_secret)?;
    debug!("Token verified successfully for {}, user_id: {}", path, claims.sub);

    // 获取用户ID
    let user_id = claims.sub;

    // 验证用户是否存在
    let user = match state.user_repository.find_by_id(user_id).await {
        Ok(Some(user)) => user,
        Ok(None) => {
            warn!("User not found: {}", user_id);
            return Err(AuthError::UserNotFound);
        }
        Err(e) => {
            error!("Failed to query user: {:?}", e);
            return Err(AuthError::DatabaseError);
        }
    };

    // 注入用户信息到请求扩展
    request.extensions_mut().insert(AuthUser {
        user_id: user.id,
        nickname: user.nickname,
    });

    // 执行请求
    Ok(next.run(request).await)
}

/// JWT令牌验证
fn verify_jwt_token(token: &str, secret: &str) -> Result<JwtClaims, AuthError> {
    use jsonwebtoken::{decode, DecodingKey, Validation, Algorithm};
    
    let validation = Validation::new(Algorithm::HS256);
    let key = DecodingKey::from_secret(secret.as_ref());
    
    match decode::<JwtClaims>(token, &key, &validation) {
        Ok(token_data) => {
            // 检查令牌是否过期
            let now = chrono::Utc::now().timestamp() as usize;
            if token_data.claims.exp < now {
                return Err(AuthError::TokenExpired);
            }
            Ok(token_data.claims)
        }
        Err(e) => {
            warn!("JWT verification failed: {:?}", e);
            Err(AuthError::InvalidToken)
        }
    }
}

/// 认证错误类型
#[derive(Debug, thiserror::Error)]
pub enum AuthError {
    #[error("缺少认证头")]
    MissingAuthHeader,
    
    #[error("认证头格式无效")]
    InvalidHeaderFormat,
    
    #[error("Bearer格式无效")]
    InvalidBearerFormat,
    
    #[error("令牌为空")]
    EmptyToken,
    
    #[error("令牌无效")]
    InvalidToken,
    
    #[error("令牌已过期")]
    TokenExpired,
    
    #[error("用户不存在")]
    UserNotFound,
    
    #[error("数据库错误")]
    DatabaseError,
}

impl axum::response::IntoResponse for AuthError {
    fn into_response(self) -> Response {
        let (error_code, message) = match self {
            AuthError::MissingAuthHeader 
            | AuthError::InvalidHeaderFormat 
            | AuthError::InvalidBearerFormat 
            | AuthError::EmptyToken 
            | AuthError::InvalidToken 
            | AuthError::TokenExpired => (1002, self.to_string()), // 1002表示未授权
            AuthError::UserNotFound => (1002, "用户不存在".to_string()),
            AuthError::DatabaseError => (5000, "服务器内部错误".to_string()), // 5000表示服务器内部错误
        };

        let body = serde_json::json!({
            "code": error_code,
            "msg": message,
            "data": null
        });

        // 根据hotspot.md规范，统一返回HTTP 200状态码
        (StatusCode::OK, axum::Json(body)).into_response()
    }
}
