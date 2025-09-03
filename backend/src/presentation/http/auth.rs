use axum::Json;
use axum::extract::State;
use axum::http::header;
use axum::response::IntoResponse;
use axum::{body::Body, http::Request, middleware::Next, response::Response};
use jsonwebtoken::{DecodingKey, Validation, decode};
use serde::{Deserialize, Serialize};

/// 认证后的当前用户
#[derive(Debug, Clone, Copy)]
pub struct CurrentUser {
    pub user_id: i64,
    pub is_temporary: bool,
}

#[derive(Serialize)]
struct ErrorEnvelope<'a> {
    code: i32,
    msg: &'a str,
    data: Option<serde_json::Value>,
}

/// JWT 配置（通过中间件状态注入）
#[derive(Clone)]
pub struct JwtConfig {
    pub secret: String,
}

/// 认证中间件：校验 Authorization Bearer，并在通过后将 CurrentUser 注入到请求扩展中
pub async fn require_auth(
    State(jwt): State<JwtConfig>,
    mut request: Request<Body>,
    next: Next,
) -> Result<Response, Response> {
    let unauthorized = |msg: &str| {
        let body = ErrorEnvelope {
            code: 1,
            msg,
            data: None,
        };
        (axum::http::StatusCode::UNAUTHORIZED, Json(body)).into_response()
    };

    let auth_header = match request.headers().get(header::AUTHORIZATION) {
        Some(h) => match h.to_str() {
            Ok(s) => s.to_string(),
            Err(_) => return Err(unauthorized("认证头格式无效")),
        },
        None => return Err(unauthorized("缺少认证头")),
    };

    if !auth_header.starts_with("Bearer ") {
        return Err(unauthorized("认证头格式无效"));
    }
    let token = auth_header.trim_start_matches("Bearer ").trim();
    if token.is_empty() {
        return Err(unauthorized("令牌为空"));
    }

    #[derive(Debug, Deserialize)]
    struct TokenClaims {
        sub: i64,
        temp: bool,
        #[serde(rename = "exp")]
        _exp: usize,
        #[serde(rename = "iat")]
        _iat: usize,
    }
    let claims = match decode::<TokenClaims>(
        token,
        &DecodingKey::from_secret(jwt.secret.as_bytes()),
        &Validation::default(),
    ) {
        Ok(data) => data.claims,
        Err(_) => return Err(unauthorized("令牌无效或已过期")),
    };

    request.extensions_mut().insert(CurrentUser {
        user_id: claims.sub,
        is_temporary: claims.temp,
    });
    Ok(next.run(request).await)
}
