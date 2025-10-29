use axum::{extract::{Request, State}, Json};

use crate::{
    dto::{
        request::{CreateTempRequest, LoginRequest, RegisterRequest, RefreshTokenRequest},
        response::{ApiResponse, AuthResponse},
    },
    error::{AppError, Result},
    middleware::auth::extract_user_id,
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
    req: Request,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;
    
    // 提取 token
    let token = req
        .headers()
        .get("Authorization")
        .and_then(|h| h.to_str().ok())
        .and_then(|s| s.strip_prefix("Bearer "))
        .ok_or_else(|| AppError::Unauthorized("缺少Token".to_string()))?;

    // 调用 Service
    let auth_service = AuthService::new(
        &state.db,
        &state.redis,
        state.config.jwt_secret.clone(),
        state.config.jwt_access_token_expiration,
        state.config.jwt_refresh_token_expiration,
    );
    auth_service.logout(user_id, token).await?;

    Ok(Json(ApiResponse::success(serde_json::json!({
        "success": true,
        "message": "登出成功"
    }))))
}

// TODO: 实现其他认证端点
// - check_status
// - set_password
// - request_password_reset
// - verify_reset_code
// - reset_password
// - list_sessions
// - revoke_session

