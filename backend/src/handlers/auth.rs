use axum::{
    extract::State,
    http::{header, HeaderMap, HeaderValue},
    response::IntoResponse,
    Extension, Json,
};

use crate::{
    dto::{
        request::{
            ChangePasswordRequest, CheckStatusRequest, GuestRequest, LoginRequest, LogoutRequest,
            RefreshTokenRequest, RegisterRequest, RequestPasswordResetRequest,
            ResetPasswordRequest, RevokeSessionRequest, SessionsRequest, VerifyResetCodeRequest,
        },
        response::{
            ApiResponse, CheckStatusData, PaginatedData, RequestPasswordResetData, SessionItem,
            VerifyResetCodeData,
        },
    },
    error::{AppError, Result},
    middleware::auth::BearerToken,
    state::AppState,
    utils::validation::Validator,
    utils::web_cookie::{
        build_refresh_token_clear_cookie, build_refresh_token_set_cookie, get_cookie_value,
    },
};

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

    let client_type =
        crate::services::auth_service::ClientType::from_opt_str(req.client_type.as_deref());

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
    if client_type == crate::services::auth_service::ClientType::Web {
        let secure = false; // 生产建议改为配置项
        let set_cookie = build_refresh_token_set_cookie(
            &auth_issue.refresh_token_raw,
            state.config.jwt_refresh_token_expiration,
            secure,
        );
        resp_headers.insert(
            header::SET_COOKIE,
            HeaderValue::from_str(&set_cookie).map_err(|_| AppError::Internal)?,
        );
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

    let client_type =
        crate::services::auth_service::ClientType::from_opt_str(req.client_type.as_deref());

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
    if client_type == crate::services::auth_service::ClientType::Web {
        let secure = false; // 生产建议改为配置项
        let set_cookie = build_refresh_token_set_cookie(
            &auth_issue.refresh_token_raw,
            state.config.jwt_refresh_token_expiration,
            secure,
        );
        resp_headers.insert(
            header::SET_COOKIE,
            HeaderValue::from_str(&set_cookie).map_err(|_| AppError::Internal)?,
        );
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

    let client_type =
        crate::services::auth_service::ClientType::from_opt_str(req.client_type.as_deref());

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
    if client_type == crate::services::auth_service::ClientType::Web {
        let secure = false; // 生产建议改为配置项
        let set_cookie = build_refresh_token_set_cookie(
            &auth_issue.refresh_token_raw,
            state.config.jwt_refresh_token_expiration,
            secure,
        );
        resp_headers.insert(
            header::SET_COOKIE,
            HeaderValue::from_str(&set_cookie).map_err(|_| AppError::Internal)?,
        );
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

    // client_type 推断：明确传 web/app；否则如果携带 refresh cookie，按 web 处理；默认 app
    let mut client_type =
        crate::services::auth_service::ClientType::from_opt_str(req.client_type.as_deref());
    if req.client_type.is_none() && get_cookie_value(&headers, "hs_refresh_token").is_some() {
        client_type = crate::services::auth_service::ClientType::Web;
    }

    // refresh_token 来源：body 优先；否则从 cookie 读取（web）
    let refresh_token_raw = if let Some(t) = req.refresh_token.as_deref() {
        t.to_string()
    } else {
        get_cookie_value(&headers, "hs_refresh_token")
            .ok_or_else(|| AppError::TokenMissing("缺少 refresh_token".to_string()))?
    };

    // 生产同源模式（推荐由 Nginx 统一入口）：不在后端做 Origin 白名单校验
    // - CORS/Origin/Referer 等安全策略应在反向代理层按环境统一配置
    // - Web refresh_token 使用 HttpOnly Cookie，依赖同源部署与 Cookie SameSite 策略降低 CSRF 风险

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
    if client_type == crate::services::auth_service::ClientType::Web {
        let secure = false; // 生产建议改为配置项
        let set_cookie = build_refresh_token_set_cookie(
            &auth_issue.refresh_token_raw,
            state.config.jwt_refresh_token_expiration,
            secure,
        );
        resp_headers.insert(
            header::SET_COOKIE,
            HeaderValue::from_str(&set_cookie).map_err(|_| AppError::Internal)?,
        );
    }

    Ok((resp_headers, Json(ApiResponse::success(auth_issue.data))))
}

pub async fn logout(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    BearerToken(token): BearerToken,
    Json(_req): Json<LogoutRequest>,
) -> Result<impl IntoResponse> {
    // 调用 Service
    state.auth_service.logout(user_id, &token).await?;

    // 清除 refresh cookie（web）
    let secure = false; // 生产建议改为配置项
    let clear_cookie = build_refresh_token_clear_cookie(secure);
    let mut resp_headers = HeaderMap::new();
    resp_headers.insert(
        header::SET_COOKIE,
        HeaderValue::from_str(&clear_cookie).map_err(|_| AppError::Internal)?,
    );

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
    Extension(user_id): Extension<i64>,
    BearerToken(token): BearerToken,
    Json(_req): Json<CheckStatusRequest>,
) -> Result<Json<ApiResponse<CheckStatusData>>> {
    let response = state.auth_service.check_status(user_id, &token).await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn change_password(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<ChangePasswordRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.auth_service.change_password(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}

pub async fn list_sessions(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Extension(session_id): Extension<Option<uuid::Uuid>>,
    Json(list_req): Json<SessionsRequest>,
) -> Result<Json<ApiResponse<PaginatedData<SessionItem>>>> {
    let response = state
        .auth_service
        .list_sessions(user_id, &list_req, session_id)
        .await?;
    Ok(Json(ApiResponse::success(response)))
}

pub async fn revoke_session(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(revoke_req): Json<RevokeSessionRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state
        .auth_service
        .revoke_session(user_id, &revoke_req)
        .await?;
    Ok(Json(ApiResponse::success_empty()))
}
