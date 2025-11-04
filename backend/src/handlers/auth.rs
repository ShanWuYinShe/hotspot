use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            CreateTempRequest, LoginRequest, RegisterRequest, RefreshTokenRequest,
            RequestPasswordResetRequest, VerifyResetCodeRequest, ResetPasswordRequest,
            CheckStatusRequest, SetPasswordRequest, ListSessionsRequest, RevokeSessionRequest,
        },
        response::{
            ApiResponse, AuthResponse, RequestPasswordResetResponse, VerifyResetCodeResponse,
            ResetPasswordResponse, CheckStatusResponse, SetPasswordResponse,
            SessionResponse, PaginatedResponse, RevokeSessionResponse,
        },
    },
    error::{AppError, Result},
    middleware::auth::BearerToken,
    services::auth_service::AuthService,
    state::AppState,
    utils::validation::Validator,
};

/// POST /api/v1/public/auth/register
pub async fn register(
    State(state): State<AppState>,
    Json(req): Json<RegisterRequest>,
) -> Result<Json<ApiResponse<AuthResponse>>> {
    // 参数验证
    Validator::validate_not_empty(&req.login_id, "登录ID")?;
    Validator::validate_not_empty(&req.nickname, "昵称")?;
    Validator::validate_length(&req.nickname, "昵称", 1, 100)?;
    
    if req.password_hash.len() != 64 {
        return Err(AppError::BadRequest("密码哈希格式错误".to_string()));
    }

    // 调用 Service
    let auth_service = AuthService::new(
        &state.db,
        &state.redis,
        state.config.jwt_secret.clone(),
        state.config.jwt_access_token_expiration,
        state.config.jwt_refresh_token_expiration,
    );
    let auth_response = auth_service.register(&req).await?;

    Ok(Json(ApiResponse::success(auth_response)))
}

/// POST /api/v1/public/auth/login
pub async fn login(
    State(state): State<AppState>,
    Json(req): Json<LoginRequest>,
) -> Result<Json<ApiResponse<AuthResponse>>> {
    // 参数验证
    Validator::validate_not_empty(&req.login_id, "登录ID")?;
    
    if req.password_hash.len() != 64 {
        return Err(AppError::BadRequest("密码哈希格式错误".to_string()));
    }

    // 调用 Service
    let auth_service = AuthService::new(
        &state.db,
        &state.redis,
        state.config.jwt_secret.clone(),
        state.config.jwt_access_token_expiration,
        state.config.jwt_refresh_token_expiration,
    );
    let auth_response = auth_service.login(&req).await?;

    Ok(Json(ApiResponse::success(auth_response)))
}

/// POST /api/v1/public/auth/create_temp
pub async fn create_temp(
    State(state): State<AppState>,
    Json(req): Json<CreateTempRequest>,
) -> Result<Json<ApiResponse<AuthResponse>>> {
    // 参数验证
    Validator::validate_not_empty(&req.nickname, "昵称")?;
    Validator::validate_length(&req.nickname, "昵称", 1, 100)?;

    // 调用 Service
    let auth_service = AuthService::new(
        &state.db,
        &state.redis,
        state.config.jwt_secret.clone(),
        state.config.jwt_access_token_expiration,
        state.config.jwt_refresh_token_expiration,
    );
    let auth_response = auth_service.create_temp(&req).await?;

    Ok(Json(ApiResponse::success(auth_response)))
}

/// POST /api/v1/public/auth/refresh_token
pub async fn refresh_token(
    State(state): State<AppState>,
    Json(req): Json<RefreshTokenRequest>,
) -> Result<Json<ApiResponse<AuthResponse>>> {
    // 调用 Service
    let auth_service = AuthService::new(
        &state.db,
        &state.redis,
        state.config.jwt_secret.clone(),
        state.config.jwt_access_token_expiration,
        state.config.jwt_refresh_token_expiration,
    );
    let auth_response = auth_service.refresh_token(&req).await?;

    Ok(Json(ApiResponse::success(auth_response)))
}

/// POST /api/v1/protected/auth/logout
pub async fn logout(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    BearerToken(token): BearerToken,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    // 调用 Service
    let auth_service = AuthService::new(
        &state.db,
        &state.redis,
        state.config.jwt_secret.clone(),
        state.config.jwt_access_token_expiration,
        state.config.jwt_refresh_token_expiration,
    );
    auth_service.logout(user_id, &token).await?;

    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "登出成功"
    }))))
}

/// POST /api/v1/public/auth/request_password_reset
pub async fn request_password_reset(
    State(state): State<AppState>,
    Json(req): Json<RequestPasswordResetRequest>,
) -> Result<Json<ApiResponse<RequestPasswordResetResponse>>> {
    let auth_service = AuthService::new(
        &state.db,
        &state.redis,
        state.config.jwt_secret.clone(),
        state.config.jwt_access_token_expiration,
        state.config.jwt_refresh_token_expiration,
    );
    let response = auth_service.request_password_reset(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/auth/verify_reset_code
pub async fn verify_reset_code(
    State(state): State<AppState>,
    Json(req): Json<VerifyResetCodeRequest>,
) -> Result<Json<ApiResponse<VerifyResetCodeResponse>>> {
    let auth_service = AuthService::new(
        &state.db,
        &state.redis,
        state.config.jwt_secret.clone(),
        state.config.jwt_access_token_expiration,
        state.config.jwt_refresh_token_expiration,
    );
    let response = auth_service.verify_reset_code(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/auth/reset_password
pub async fn reset_password(
    State(state): State<AppState>,
    Json(req): Json<ResetPasswordRequest>,
) -> Result<Json<ApiResponse<ResetPasswordResponse>>> {
    let auth_service = AuthService::new(
        &state.db,
        &state.redis,
        state.config.jwt_secret.clone(),
        state.config.jwt_access_token_expiration,
        state.config.jwt_refresh_token_expiration,
    );
    let response = auth_service.reset_password(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/auth/check_status
pub async fn check_status(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    BearerToken(token): BearerToken,
    Json(_req): Json<CheckStatusRequest>,
) -> Result<Json<ApiResponse<CheckStatusResponse>>> {
    let auth_service = AuthService::new(
        &state.db,
        &state.redis,
        state.config.jwt_secret.clone(),
        state.config.jwt_access_token_expiration,
        state.config.jwt_refresh_token_expiration,
    );
    
    let response = auth_service.check_status(user_id, &token).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/auth/set_password
pub async fn set_password(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(password_req): Json<SetPasswordRequest>,
) -> Result<Json<ApiResponse<SetPasswordResponse>>> {
    let auth_service = AuthService::new(
        &state.db,
        &state.redis,
        state.config.jwt_secret.clone(),
        state.config.jwt_access_token_expiration,
        state.config.jwt_refresh_token_expiration,
    );
    let response = auth_service.set_password(user_id, &password_req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/auth/list_sessions
pub async fn list_sessions(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(list_req): Json<ListSessionsRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<SessionResponse>>>> {
    let auth_service = AuthService::new(
        &state.db,
        &state.redis,
        state.config.jwt_secret.clone(),
        state.config.jwt_access_token_expiration,
        state.config.jwt_refresh_token_expiration,
    );
    let response = auth_service.list_sessions(user_id, &list_req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/auth/revoke_session
pub async fn revoke_session(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(revoke_req): Json<RevokeSessionRequest>,
) -> Result<Json<ApiResponse<RevokeSessionResponse>>> {
    let auth_service = AuthService::new(
        &state.db,
        &state.redis,
        state.config.jwt_secret.clone(),
        state.config.jwt_access_token_expiration,
        state.config.jwt_refresh_token_expiration,
    );
    let response = auth_service.revoke_session(user_id, &revoke_req).await?;
    Ok(Json(ApiResponse::success(response)))
}

