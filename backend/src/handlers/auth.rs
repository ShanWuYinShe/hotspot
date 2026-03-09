use crate::services::auth_service::AuthIssue;
use crate::{
    dto::{
        request::{
            ChangePasswordRequest, CheckStatusRequest, GuestRequest, LoginRequest, LogoutRequest,
            RefreshTokenRequest, RegisterRequest, RequestPasswordResetRequest,
            ResetPasswordRequest, RevokeAllSessionsRequest, RevokeSessionRequest, SessionsRequest,
        },
        response::{ApiResponse, CheckStatusData, RevokeAllResult, SessionsListData},
    },
    error::{ApiError, Result},
    middleware::auth::{AuthenticatedUser, BearerToken, OptionalAuthenticatedUser},
    state::AppState,
    types::enums::ClientType,
    utils::validation::Validator,
    utils::web_cookie::{
        build_access_token_clear_cookie, build_access_token_set_cookie,
        build_csrf_token_clear_cookie, build_csrf_token_set_cookie,
        build_refresh_token_clear_cookie, build_refresh_token_set_cookie, generate_csrf_token,
        get_cookie_value,
    },
};
use axum::{
    extract::State,
    http::{header, HeaderMap, HeaderValue},
    response::IntoResponse,
    Json,
};

fn append_set_cookie(headers: &mut HeaderMap, cookie: String) -> Result<()> {
    headers.append(
        header::SET_COOKIE,
        HeaderValue::from_str(&cookie).map_err(|_| ApiError::internal("Cookie header error"))?,
    );
    Ok(())
}

/// 统一设置认证相关的 Cookie（access_token + refresh_token + csrf_token）
fn append_auth_cookies(
    headers: &mut HeaderMap,
    access_token_raw: &str,
    access_token_expiration: i64,
    refresh_token_raw: &str,
    refresh_token_expiration: i64,
    secure: bool,
) -> Result<()> {
    let csrf_token = generate_csrf_token();

    // access_token Cookie: Path=/api (所有 API 请求自动携带)
    let access_cookie =
        build_access_token_set_cookie(access_token_raw, access_token_expiration, secure);

    // refresh_token Cookie: Path=/api/v1/auth/refresh_token (仅刷新 API 携带)
    let refresh_cookie =
        build_refresh_token_set_cookie(refresh_token_raw, refresh_token_expiration, secure);

    // CSRF token Cookie
    let csrf_cookie = build_csrf_token_set_cookie(&csrf_token, refresh_token_expiration, secure);

    append_set_cookie(headers, access_cookie)?;
    append_set_cookie(headers, refresh_cookie)?;
    append_set_cookie(headers, csrf_cookie)?;
    Ok(())
}

/// 统一构建认证响应（设置 Cookie、按 client_type 处理 data）
///
/// spec: Web 端 data 为 `{}`（tokens 仅通过 Set-Cookie 传递）
///       App 端 data 包含完整 AuthTokenPairApp
fn build_auth_response(
    auth_issue: AuthIssue,
    client_type: ClientType,
    config: &crate::config::Config,
) -> Result<(HeaderMap, Json<ApiResponse<serde_json::Value>>)> {
    let mut headers = HeaderMap::new();
    if client_type == ClientType::Web {
        append_auth_cookies(
            &mut headers,
            &auth_issue.data.access_token,
            config.jwt_access_token_expiration,
            &auth_issue.refresh_token_raw,
            config.jwt_refresh_token_expiration,
            config.cookie_secure,
        )?;
    }
    let data = if client_type == ClientType::Web {
        // spec: Web 端 data = {} (空对象)
        serde_json::json!({})
    } else {
        serde_json::to_value(&auth_issue.data)
            .map_err(|_| ApiError::internal("序列化认证数据失败"))?
    };
    Ok((headers, Json(ApiResponse::success(data))))
}

/// Web 端敏感操作的 CSRF 校验（仅当请求携带 hs_csrf_token Cookie 时触发）。
/// App 端不使用 Cookie，不携带 hs_csrf_token，因此自动跳过。
fn validate_csrf_if_web(headers: &HeaderMap) -> Result<()> {
    if get_cookie_value(headers, "hs_csrf_token").is_some() {
        validate_csrf_token(headers)?;
    }
    Ok(())
}

fn validate_csrf_token(headers: &HeaderMap) -> Result<()> {
    if let Some(fetch_site) = headers.get("Sec-Fetch-Site").and_then(|v| v.to_str().ok()) {
        if fetch_site.eq_ignore_ascii_case("cross-site") {
            return Err(ApiError::Forbidden("CSRF：跨站请求被拒绝".to_string()));
        }
    }

    let csrf_cookie = get_cookie_value(headers, "hs_csrf_token")
        .ok_or_else(|| ApiError::Forbidden("CSRF token 缺失".to_string()))?;
    let csrf_header = headers
        .get("X-CSRF-Token")
        .and_then(|v| v.to_str().ok())
        .ok_or_else(|| ApiError::Forbidden("CSRF token 缺失".to_string()))?;

    if csrf_cookie != csrf_header {
        return Err(ApiError::Forbidden("CSRF token 无效".to_string()));
    }

    Ok(())
}

pub async fn register(
    State(state): State<AppState>,
    headers: axum::http::HeaderMap,
    Json(req): Json<RegisterRequest>,
) -> Result<impl IntoResponse> {
    // 参数验证
    Validator::validate_login_id(&req.login_id)?;
    Validator::validate_nickname(&req.nickname)?;
    Validator::validate_password(&req.password, "密码")?;

    let user_agent = headers
        .get(axum::http::header::USER_AGENT)
        .and_then(|v| v.to_str().ok());
    let device_fingerprint = headers
        .get("X-Device-Fingerprint")
        .and_then(|v| v.to_str().ok());
    let ip_address = None;

    // client_type 已在请求中强类型化，直接使用
    let client_type = req.client_type;

    // 调用 Service
    let auth_issue = state
        .auth_service
        .register(
            &req,
            client_type,
            ip_address,
            user_agent,
            device_fingerprint,
        )
        .await?;

    build_auth_response(auth_issue, client_type, &state.config)
}

pub async fn login(
    State(state): State<AppState>,
    headers: axum::http::HeaderMap,
    Json(req): Json<LoginRequest>,
) -> Result<impl IntoResponse> {
    // 参数验证
    Validator::validate_login_id(&req.login_id)?;
    Validator::validate_password(&req.password, "密码")?;

    let user_agent = headers
        .get(axum::http::header::USER_AGENT)
        .and_then(|v| v.to_str().ok());
    let device_fingerprint = headers
        .get("X-Device-Fingerprint")
        .and_then(|v| v.to_str().ok());
    let ip_address = None;

    // client_type 已在请求中强类型化，直接使用
    let client_type = req.client_type;

    // 调用 Service
    let auth_issue = state
        .auth_service
        .login(
            &req,
            client_type,
            ip_address,
            user_agent,
            device_fingerprint,
        )
        .await?;

    build_auth_response(auth_issue, client_type, &state.config)
}

pub async fn create_guest(
    State(state): State<AppState>,
    headers: axum::http::HeaderMap,
    Json(req): Json<GuestRequest>,
) -> Result<impl IntoResponse> {
    // 参数验证
    Validator::validate_nickname(&req.nickname)?;

    let user_agent = headers
        .get(axum::http::header::USER_AGENT)
        .and_then(|v| v.to_str().ok());
    let device_fingerprint = headers
        .get("X-Device-Fingerprint")
        .and_then(|v| v.to_str().ok());
    let ip_address = None;

    // client_type 已在请求中强类型化，直接使用
    let client_type = req.client_type;

    // 调用 Service
    let auth_issue = state
        .auth_service
        .create_guest(
            &req,
            client_type,
            ip_address,
            user_agent,
            device_fingerprint,
        )
        .await?;

    build_auth_response(auth_issue, client_type, &state.config)
}

pub async fn refresh_token(
    State(state): State<AppState>,
    headers: axum::http::HeaderMap,
    Json(req): Json<RefreshTokenRequest>,
) -> Result<impl IntoResponse> {
    // 获取客户端信息用于环境绑定检测
    let user_agent = headers
        .get(axum::http::header::USER_AGENT)
        .and_then(|v| v.to_str().ok());
    let device_fingerprint = headers
        .get("X-Device-Fingerprint")
        .and_then(|v| v.to_str().ok());

    // IP 获取逻辑通常在反向代理后比较复杂，这里简化处理或通过中间件获取
    // 实际项目中可以从 extensions 获取 ConnectInfo 或 X-Forwarded-For
    let ip_address = None;

    // client_type 已在请求中强类型化，直接使用
    let client_type = req.client_type;

    if client_type == ClientType::Web {
        validate_csrf_token(&headers)?;
    }

    // refresh_token 投递规则（强制）：
    // - app：必须在 body 传 refresh_token
    // - web：必须不传（从 HttpOnly cookie 读取），若传入则视为参数错误
    let refresh_token_raw = match client_type {
        ClientType::App => req
            .refresh_token
            .as_deref()
            .filter(|token| !token.trim().is_empty())
            .ok_or_else(|| ApiError::BadRequest("app 场景必须提供 refresh_token".to_string()))?
            .to_string(),
        ClientType::Web => {
            if req.refresh_token.is_some() {
                return Err(ApiError::BadRequest(
                    "web 场景不得在 body 中传递 refresh_token".to_string(),
                ));
            }
            get_cookie_value(&headers, "refresh_token")
                .ok_or_else(|| ApiError::RefreshTokenMissing)?
        }
    };

    // 调用 Service
    let auth_issue = state
        .auth_service
        .refresh_token(
            &refresh_token_raw,
            client_type,
            ip_address,
            user_agent,
            device_fingerprint,
        )
        .await?;

    build_auth_response(auth_issue, client_type, &state.config)
}

pub async fn logout(
    State(state): State<AppState>,
    headers: axum::http::HeaderMap,
    user: AuthenticatedUser,
    BearerToken(token): BearerToken,
    Json(_req): Json<LogoutRequest>,
) -> Result<impl IntoResponse> {
    // CSRF 校验（Web 端敏感操作）
    validate_csrf_if_web(&headers)?;

    // 调用 Service
    state.auth_service.logout(user.user_id, &token).await?;

    // 清除所有认证相关 Cookie（web）
    let clear_access_cookie = build_access_token_clear_cookie(state.config.cookie_secure);
    let clear_refresh_cookie = build_refresh_token_clear_cookie(state.config.cookie_secure);
    let clear_csrf_cookie = build_csrf_token_clear_cookie(state.config.cookie_secure);

    let mut resp_headers = HeaderMap::new();
    append_set_cookie(&mut resp_headers, clear_access_cookie)?;
    append_set_cookie(&mut resp_headers, clear_refresh_cookie)?;
    append_set_cookie(&mut resp_headers, clear_csrf_cookie)?;

    Ok((resp_headers, Json(ApiResponse::success_empty())))
}

pub async fn request_password_reset(
    State(state): State<AppState>,
    Json(req): Json<RequestPasswordResetRequest>,
) -> Result<Json<ApiResponse<()>>> {
    // 参数验证
    Validator::validate_login_id(&req.login_id)?;

    state.auth_service.request_password_reset(&req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn reset_password(
    State(state): State<AppState>,
    headers: axum::http::HeaderMap,
    Json(req): Json<ResetPasswordRequest>,
) -> Result<impl IntoResponse> {
    // 参数验证
    Validator::validate_login_id(&req.login_id)?;
    Validator::validate_not_empty(&req.verification_code, "验证码")?;
    Validator::validate_password(&req.new_password, "新密码")?;

    let user_agent = headers
        .get(axum::http::header::USER_AGENT)
        .and_then(|v| v.to_str().ok());
    let device_fingerprint = headers
        .get("X-Device-Fingerprint")
        .and_then(|v| v.to_str().ok());
    let ip_address = None;
    let client_type = req.client_type;

    let auth_issue = state
        .auth_service
        .reset_password(
            &req,
            client_type,
            ip_address,
            user_agent,
            device_fingerprint,
        )
        .await?;

    build_auth_response(auth_issue, client_type, &state.config)
}

pub async fn check_status(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    headers: axum::http::HeaderMap,
    Json(_req): Json<CheckStatusRequest>,
) -> Result<Json<ApiResponse<CheckStatusData>>> {
    let token = headers
        .get("Authorization")
        .and_then(|h| h.to_str().ok())
        .and_then(|h| h.strip_prefix("Bearer "));
    let response = state.auth_service.check_status(user.user_id, token).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn change_password(
    State(state): State<AppState>,
    headers: axum::http::HeaderMap,
    user: AuthenticatedUser,
    Json(req): Json<ChangePasswordRequest>,
) -> Result<Json<ApiResponse<()>>> {
    // CSRF 校验（Web 端敏感操作）
    validate_csrf_if_web(&headers)?;
    Validator::validate_not_empty(&req.old_password, "原密码")?;
    Validator::validate_password(&req.new_password, "新密码")?;

    let session_id = user
        .session_id
        .ok_or_else(|| ApiError::Unauthorized("当前会话不可用".to_string()))?;
    state
        .auth_service
        .change_password(user.user_id, session_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn list_sessions(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(list_req): Json<SessionsRequest>,
) -> Result<Json<ApiResponse<SessionsListData>>> {
    let response = state
        .auth_service
        .list_sessions(user.user_id, &list_req, user.session_id)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn revoke_session(
    State(state): State<AppState>,
    headers: axum::http::HeaderMap,
    user: AuthenticatedUser,
    Json(revoke_req): Json<RevokeSessionRequest>,
) -> Result<Json<ApiResponse<()>>> {
    // CSRF 校验（Web 端敏感操作）
    validate_csrf_if_web(&headers)?;

    let session_id = user
        .session_id
        .ok_or_else(|| ApiError::Unauthorized("当前会话不可用".to_string()))?;
    state
        .auth_service
        .revoke_session(user.user_id, session_id, &revoke_req)
        .await?;

    Ok(Json(ApiResponse::success_empty()))
}

/// 撤销当前用户所有其他会话（不含当前会话）
pub async fn revoke_all_sessions(
    State(state): State<AppState>,
    headers: axum::http::HeaderMap,
    user: AuthenticatedUser,
    Json(_req): Json<RevokeAllSessionsRequest>,
) -> Result<Json<ApiResponse<RevokeAllResult>>> {
    // CSRF 校验（Web 端敏感操作）
    validate_csrf_if_web(&headers)?;

    let session_id = user
        .session_id
        .ok_or_else(|| ApiError::Unauthorized("当前会话不可用".to_string()))?;
    let revoked_count = state
        .auth_service
        .revoke_all_sessions(user.user_id, session_id)
        .await?;

    Ok(Json(ApiResponse::success(RevokeAllResult {
        revoked_count,
    })))
}
