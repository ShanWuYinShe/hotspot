/// HotSpot 统一认证系统
/// 
/// 功能职责：
/// - JWT token 验证和解析
/// - 用户身份认证和授权检查
/// - 公开路径管理和访问控制
/// - 认证错误处理和响应格式化
/// 
/// 核心特性：
/// - 统一的认证中间件，替代所有旧的认证逻辑
/// - 支持临时用户和正式用户的双重身份体系
/// - 集中的公开路径配置，便于维护
/// - 完整的JWT生命周期管理（创建、验证、刷新）
/// 
/// 使用方式：
/// - 通过 require_auth 中间件保护路由
/// - 通过 CurrentUser 扩展获取当前用户信息
/// - 通过 jwt_utils 模块管理JWT token
use axum::{
    body::Body,
    extract::State,
    http::{header, Request, StatusCode},
    middleware::Next,
    response::{IntoResponse, Response},
    Json,
};
use jsonwebtoken::{decode, DecodingKey, Validation, Algorithm};
use serde::{Deserialize, Serialize};
use tracing::{debug, warn, error};

/// JWT配置
#[derive(Clone, Debug)]
pub struct JwtConfig {
    pub secret: String,
}

impl JwtConfig {
    pub fn new(secret: String) -> Self {
        Self { secret }
    }
    
    /// 检查路径是否为公开路径（不需要认证）
    pub fn is_public_path(&self, path: &str) -> bool {
        // 公开路径列表
        const PUBLIC_PATHS: &[&str] = &[
            "/health",
            "/health/detailed", 
            "/ready",
            "/alive",
            "/api/v1/auth/register",
            "/api/v1/auth/login",
            "/api/v1/auth/create_temp",
            "/api/v1/auth/refresh_token",
            "/uploads",
        ];
        
        // 检查精确匹配
        if PUBLIC_PATHS.contains(&path) {
            return true;
        }
        
        // 检查前缀匹配
        if path.starts_with("/uploads/") || path.starts_with("/ws") {
            return true;
        }
        
        false
    }
}

/// 认证后的用户信息
#[derive(Debug, Clone, Copy)]
pub struct CurrentUser {
    pub user_id: i64,
    pub is_temporary: bool,
    pub nickname: Option<&'static str>, // 可选，用于日志记录
}

/// JWT Claims结构
#[derive(Debug, Serialize, Deserialize)]
pub struct JwtClaims {
    pub sub: i64,           // 用户ID
    pub temp: bool,         // 是否为临时用户
    pub exp: usize,         // 过期时间
    pub iat: usize,         // 签发时间
    pub nickname: Option<String>, // 用户昵称（可选）
}

/// 统一认证中间件
/// 
/// 功能：
/// 1. 检查公开路径，直接放行
/// 2. 验证Authorization Bearer token
/// 3. 解析JWT claims
/// 4. 注入CurrentUser到请求扩展
pub async fn require_auth(
    State(jwt_config): State<JwtConfig>,
    mut request: Request<Body>,
    next: Next,
) -> Result<Response, Response> {
    let path = request.uri().path();
    let method = request.method();
    
    debug!("Auth middleware: {} {}", method, path);

    // 检查是否为公开路径
    if jwt_config.is_public_path(path) {
        debug!("Public path detected, skipping auth: {}", path);
        return Ok(next.run(request).await);
    }

    debug!("Protected path, checking auth: {}", path);

    // 获取Authorization header
    let auth_header = match request.headers().get(header::AUTHORIZATION) {
        Some(header) => match header.to_str() {
            Ok(h) => h,
            Err(_) => {
                warn!("Invalid Authorization header format for {}", path);
                return Err(create_auth_error("认证头格式无效"));
            }
        },
        None => {
            warn!("Missing Authorization header for {}", path);
            return Err(create_auth_error("缺少认证头"));
        }
    };

    // 验证Bearer格式
    if !auth_header.starts_with("Bearer ") {
        warn!("Invalid Bearer format for {}: {}", path, auth_header);
        return Err(create_auth_error("认证头格式无效"));
    }

    // 提取token
    let token = auth_header.trim_start_matches("Bearer ").trim();
    if token.is_empty() {
        warn!("Empty token for {}", path);
        return Err(create_auth_error("令牌为空"));
    }

    debug!("Token extracted for {}, length: {}", path, token.len());

    // 验证JWT token
    let claims = match verify_jwt_token(token, &jwt_config.secret) {
        Ok(claims) => claims,
        Err(e) => {
            warn!("JWT verification failed for {}: {:?}", path, e);
            return Err(create_auth_error(&e.to_string()));
        }
    };

    debug!("Token verified successfully for {}, user_id: {}, temp: {}", 
           path, claims.sub, claims.temp);

    // 注入用户信息到请求扩展
    request.extensions_mut().insert(CurrentUser {
        user_id: claims.sub,
        is_temporary: claims.temp,
        nickname: None, // 可以后续从数据库查询填充
    });

    // 继续处理请求
    Ok(next.run(request).await)
}

/// JWT token验证
fn verify_jwt_token(token: &str, secret: &str) -> Result<JwtClaims, AuthError> {
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
            error!("JWT decode error: {:?}", e);
            Err(AuthError::InvalidToken)
        }
    }
}

/// 创建认证错误响应
fn create_auth_error(message: &str) -> Response {
    let error_response = ErrorResponse {
        code: 1002, // 未授权错误码
        msg: message.to_string(),
        data: None::<()>,
    };
    
    (StatusCode::OK, Json(error_response)).into_response()
}

/// 统一错误响应格式
#[derive(Serialize)]
struct ErrorResponse<T> {
    code: i32,
    msg: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    data: Option<T>,
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

/// JWT工具函数
pub mod jwt_utils {
    use super::*;
    use chrono::{Duration, Utc};
    use jsonwebtoken::{encode, EncodingKey, Header};

    /// 创建JWT token
    pub fn create_jwt_token(
        user_id: i64,
        is_temporary: bool,
        nickname: Option<String>,
        secret: &str,
        expires_in_hours: i64,
    ) -> Result<String, AuthError> {
        let now = Utc::now();
        let exp = (now + Duration::hours(expires_in_hours)).timestamp() as usize;
        let iat = now.timestamp() as usize;

        let claims = JwtClaims {
            sub: user_id,
            temp: is_temporary,
            exp,
            iat,
            nickname,
        };

        let header = Header::new(Algorithm::HS256);
        let key = EncodingKey::from_secret(secret.as_ref());

        encode(&header, &claims, &key)
            .map_err(|_| AuthError::InvalidToken)
    }

    /// 刷新JWT token
    pub fn refresh_jwt_token(
        old_token: &str,
        secret: &str,
        new_expires_in_hours: i64,
    ) -> Result<String, AuthError> {
        // 验证旧token（允许过期）
        let mut validation = Validation::new(Algorithm::HS256);
        validation.validate_exp = false; // 允许过期token用于刷新
        
        let key = DecodingKey::from_secret(secret.as_ref());
        let token_data = decode::<JwtClaims>(old_token, &key, &validation)
            .map_err(|_| AuthError::InvalidToken)?;

        // 创建新token
        create_jwt_token(
            token_data.claims.sub,
            token_data.claims.temp,
            token_data.claims.nickname,
            secret,
            new_expires_in_hours,
        )
    }
}

/// 从请求扩展中获取当前用户
pub fn get_current_user(request: &Request<Body>) -> Option<CurrentUser> {
    request.extensions().get::<CurrentUser>().copied()
}

/// 检查当前用户是否为管理员
pub fn is_admin_user(user: &CurrentUser) -> bool {
    // 管理员用户ID范围：9000000000000000 - 9999999999999999
    user.user_id >= 9000000000000000 && user.user_id <= 9999999999999999
}

/// 检查当前用户是否为临时用户
pub fn is_temporary_user(user: &CurrentUser) -> bool {
    user.is_temporary
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_public_path_detection() {
        let jwt_config = JwtConfig::new("test_secret".to_string());
        
        // 测试公开路径
        assert!(jwt_config.is_public_path("/health"));
        assert!(jwt_config.is_public_path("/api/v1/auth/login"));
        assert!(jwt_config.is_public_path("/uploads/avatar.jpg"));
        assert!(jwt_config.is_public_path("/ws"));
        
        // 测试受保护路径
        assert!(!jwt_config.is_public_path("/api/v1/user/profile"));
        assert!(!jwt_config.is_public_path("/api/v1/group/create"));
        assert!(!jwt_config.is_public_path("/api/v1/chat/send"));
    }

    #[test]
    fn test_jwt_token_creation() {
        let secret = "test_secret";
        let token = jwt_utils::create_jwt_token(
            12345,
            false,
            Some("test_user".to_string()),
            secret,
            24,
        ).unwrap();
        
        assert!(!token.is_empty());
        
        // 验证token
        let claims = verify_jwt_token(&token, secret).unwrap();
        assert_eq!(claims.sub, 12345);
        assert!(!claims.temp);
        assert_eq!(claims.nickname, Some("test_user".to_string()));
    }
}