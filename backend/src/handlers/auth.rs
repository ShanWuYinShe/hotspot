use axum::{
    extract::State,
    http::{header, HeaderMap, HeaderValue},
    response::IntoResponse,
    Json,
};
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            ChangePasswordRequest, CheckStatusRequest, GuestRequest, LoginRequest, LogoutRequest,
            RefreshTokenRequest, RegisterRequest, RequestPasswordResetRequest,
            ResetPasswordRequest, RevokeSessionRequest, SendVerificationCodeRequest,
            SessionsRequest, VerifyCodeRequest, VerifyResetCodeRequest,
        },
        response::{
            ApiResponse, CheckStatusData, PaginatedData, RequestPasswordResetData,
            SendVerificationCodeData, SessionItem, VerifyCodeData, VerifyResetCodeData,
        },
    },
    error::{AppError, Result},
    middleware::auth::{AuthenticatedUser, BearerToken, OptionalAuthenticatedUser},
    services::auth_service::ClientType,
    state::AppState,
    utils::validation::Validator,
    utils::web_cookie::{
        build_csrf_token_clear_cookie, build_csrf_token_set_cookie,
        build_refresh_token_clear_cookie, build_refresh_token_set_cookie, get_cookie_value,
    },
};

fn append_set_cookie(headers: &mut HeaderMap, cookie: String) -> Result<()> {
    headers.append(
        header::SET_COOKIE,
        HeaderValue::from_str(&cookie).map_err(|_| AppError::internal("Cookie header error"))?,
    );
    Ok(())
}

fn validate_csrf_token(headers: &HeaderMap) -> Result<()> {
    if let Some(fetch_site) = headers.get("Sec-Fetch-Site").and_then(|v| v.to_str().ok()) {
        if fetch_site.eq_ignore_ascii_case("cross-site") {
            return Err(AppError::Forbidden("CSRF：跨站请求被拒绝".to_string()));
        }
    }

    let csrf_cookie = get_cookie_value(headers, "hs_csrf_token")
        .ok_or_else(|| AppError::Forbidden("CSRF token 缺失".to_string()))?;
    let csrf_header = headers
        .get("X-CSRF-Token")
        .and_then(|v| v.to_str().ok())
        .ok_or_else(|| AppError::Forbidden("CSRF token 缺失".to_string()))?;

    if csrf_cookie != csrf_header {
        return Err(AppError::Forbidden("CSRF token 无效".to_string()));
    }

    Ok(())
}

pub async fn register(
    State(state): State<AppState>,
    headers: axum::http::HeaderMap,
    Json(req): Json<RegisterRequest>,
) -> Result<impl IntoResponse> {
    // 参数验证
    Validator::validate_not_empty(&req.login_id, "登录ID")?;
    Validator::validate_not_empty(&req.nickname, "昵称")?;
    Validator::validate_length(&req.nickname, "昵称", 1, 100)?;
    Validator::validate_not_empty(&req.password, "密码")?;
    Validator::validate_length(&req.password, "密码", 8, 128)?;

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

    let mut resp_headers = HeaderMap::new();
    if client_type == ClientType::Web {
        let secure = false; // 生产建议改为配置项
        let csrf_token = Uuid::new_v4().to_string();
        let set_cookie = build_refresh_token_set_cookie(
            &auth_issue.refresh_token_raw,
            state.config.jwt_refresh_token_expiration,
            secure,
        );
        let csrf_cookie = build_csrf_token_set_cookie(
            &csrf_token,
            state.config.jwt_refresh_token_expiration,
            secure,
        );
        append_set_cookie(&mut resp_headers, set_cookie)?;
        append_set_cookie(&mut resp_headers, csrf_cookie)?;
    }

    Ok((resp_headers, Json(ApiResponse::success(auth_issue.data))))
}

pub async fn login(
    State(state): State<AppState>,
    headers: axum::http::HeaderMap,
    Json(req): Json<LoginRequest>,
) -> Result<impl IntoResponse> {
    // 参数验证
    Validator::validate_not_empty(&req.login_id, "登录ID")?;
    Validator::validate_not_empty(&req.password, "密码")?;

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

    let mut resp_headers = HeaderMap::new();
    if client_type == ClientType::Web {
        let secure = false; // 生产建议改为配置项
        let csrf_token = Uuid::new_v4().to_string();
        let set_cookie = build_refresh_token_set_cookie(
            &auth_issue.refresh_token_raw,
            state.config.jwt_refresh_token_expiration,
            secure,
        );
        let csrf_cookie = build_csrf_token_set_cookie(
            &csrf_token,
            state.config.jwt_refresh_token_expiration,
            secure,
        );
        append_set_cookie(&mut resp_headers, set_cookie)?;
        append_set_cookie(&mut resp_headers, csrf_cookie)?;
    }

    Ok((resp_headers, Json(ApiResponse::success(auth_issue.data))))
}

pub async fn create_guest(
    State(state): State<AppState>,
    headers: axum::http::HeaderMap,
    Json(req): Json<GuestRequest>,
) -> Result<impl IntoResponse> {
    // 参数验证
    Validator::validate_not_empty(&req.nickname, "昵称")?;
    Validator::validate_length(&req.nickname, "昵称", 1, 100)?;

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

    let mut resp_headers = HeaderMap::new();
    if client_type == ClientType::Web {
        let secure = false; // 生产建议改为配置项
        let csrf_token = Uuid::new_v4().to_string();
        let set_cookie = build_refresh_token_set_cookie(
            &auth_issue.refresh_token_raw,
            state.config.jwt_refresh_token_expiration,
            secure,
        );
        let csrf_cookie = build_csrf_token_set_cookie(
            &csrf_token,
            state.config.jwt_refresh_token_expiration,
            secure,
        );
        append_set_cookie(&mut resp_headers, set_cookie)?;
        append_set_cookie(&mut resp_headers, csrf_cookie)?;
    }

    Ok((resp_headers, Json(ApiResponse::success(auth_issue.data))))
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
            .ok_or_else(|| AppError::BadRequest("app 场景必须提供 refresh_token".to_string()))?,
        ClientType::Web => {
            if req.refresh_token.is_some() {
                return Err(AppError::BadRequest(
                    "web 场景不得在 body 中传递 refresh_token".to_string(),
                ));
            }
            get_cookie_value(&headers, "hs_refresh_token")
                .ok_or_else(|| AppError::RefreshTokenMissing)?
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

    let mut resp_headers = HeaderMap::new();
    if client_type == ClientType::Web {
        let secure = false; // 生产建议改为配置项
        let csrf_token = Uuid::new_v4().to_string();
        let set_cookie = build_refresh_token_set_cookie(
            &auth_issue.refresh_token_raw,
            state.config.jwt_refresh_token_expiration,
            secure,
        );
        let csrf_cookie = build_csrf_token_set_cookie(
            &csrf_token,
            state.config.jwt_refresh_token_expiration,
            secure,
        );
        append_set_cookie(&mut resp_headers, set_cookie)?;
        append_set_cookie(&mut resp_headers, csrf_cookie)?;
    }

    Ok((resp_headers, Json(ApiResponse::success(auth_issue.data))))
}

pub async fn logout(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    BearerToken(token): BearerToken,
    Json(_req): Json<LogoutRequest>,
) -> Result<impl IntoResponse> {
    // 调用 Service
    state.auth_service.logout(user.user_id, &token).await?;

    // 清除 refresh cookie（web）
    let secure = false; // 生产建议改为配置项
    let clear_cookie = build_refresh_token_clear_cookie(secure);
    let clear_csrf_cookie = build_csrf_token_clear_cookie(secure);
    let mut resp_headers = HeaderMap::new();
    append_set_cookie(&mut resp_headers, clear_cookie)?;
    append_set_cookie(&mut resp_headers, clear_csrf_cookie)?;

    Ok((resp_headers, Json(ApiResponse::success_empty())))
}

pub async fn request_password_reset(
    State(state): State<AppState>,
    Json(req): Json<RequestPasswordResetRequest>,
) -> Result<Json<ApiResponse<RequestPasswordResetData>>> {
    let response = state.auth_service.request_password_reset(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn verify_reset_code(
    State(state): State<AppState>,
    Json(req): Json<VerifyResetCodeRequest>,
) -> Result<Json<ApiResponse<VerifyResetCodeData>>> {
    let response = state.auth_service.verify_reset_code(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn reset_password(
    State(state): State<AppState>,
    Json(req): Json<ResetPasswordRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.auth_service.reset_password(&req).await?;
    Ok(Json(ApiResponse::success_empty()))
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
    user: AuthenticatedUser,
    Json(req): Json<ChangePasswordRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .auth_service
        .change_password(user.user_id, &req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn list_sessions(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(list_req): Json<SessionsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<SessionItem>>>> {
    let response = state
        .auth_service
        .list_sessions(user.user_id, &list_req, user.session_id)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn revoke_session(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(revoke_req): Json<RevokeSessionRequest>,
) -> Result<Json<ApiResponse<()>>> {
    // 撤销会话
    state
        .auth_service
        .revoke_session(user.user_id, &revoke_req)
        .await?;

    // 清除认证缓存（异步执行，不阻塞响应）
    // 注意：由于 revoke_session 已经从 Redis 获取了 session_id，
    // 这里我们需要重新获取或者修改 API 以返回 session_id
    // 为了简化，我们使用更激进的策略：清除该用户的所有缓存
    tokio::spawn({
        let auth_cache = state.auth_cache.clone();
        let user_id = user.user_id;
        async move {
            auth_cache.invalidate_by_user(user_id).await;
        }
    });

    Ok(Json(ApiResponse::success_empty()))
}

pub async fn send_verification_code(
    State(state): State<AppState>,
    Json(req): Json<SendVerificationCodeRequest>,
) -> Result<Json<ApiResponse<SendVerificationCodeData>>> {
    // 参数验证
    Validator::validate_not_empty(&req.phone, "手机号")?;
    // r#type 已是强类型枚举，反序列化时自动验证

    // 调用 Service
    let response = state.auth_service.send_verification_code(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn verify_code(
    State(state): State<AppState>,
    Json(req): Json<VerifyCodeRequest>,
) -> Result<Json<ApiResponse<VerifyCodeData>>> {
    // 参数验证
    Validator::validate_not_empty(&req.code_id, "验证码会话ID")?;
    Validator::validate_not_empty(&req.code, "验证码")?;

    // 调用 Service
    let response = state.auth_service.verify_code(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}
