use axum::{
    Json, debug_handler,
    extract::{Extension, State},
};
use std::sync::Arc;
use tracing::{debug, info};

use crate::{
    middleware::auth::AppState,
    utils::{
        user_id_partition::UserIdPartition,
        constants::defaults::{
            DEFAULT_USER_NICKNAME, DEFAULT_TEMP_USER_NICKNAME, 
            MAX_SEARCH_RADIUS_METERS, MAX_SEARCH_KEYWORD_LENGTH,
            DEFAULT_PAGE_SIZE, MAX_PAGE_SIZE,
        },
        // constants::messages::WELCOME_TEMP_USER_MESSAGE
    },
    models::{
        api::user::AuthUser,
        api::user::{
            AuthResponse, CreateTempUserRequest, LoginRequest, RefreshTokenRequest,
            RefreshTokenResponse, RegisterRequest, SearchUserByIdRequest, SearchUserByIdResponse,
            SearchUserByLocationRequest, SearchUserByLocationResponse, SearchUserByNameRequest,
            SearchUserByNameResponse, SetPasswordRequest, UpdateLocationRequest, UserInfo,
        },
        user_settings::UserSettings,
    },
    utils::response::{ApiResponse, AppError, PaginatedResponse, PaginationMeta, success_response},
    validators::{common_validator, user_validator},
};

use serde_json::json;
use crate::utils::events::{DomainEvent};

/// 用户注册
///
/// 创建新用户账号，并返回用户信息和认证令牌
#[utoipa::path(
    post,
    path = "/api/v1/user/register",
    tag = "用户",
    request_body = RegisterRequest,
    responses(
        (status = 200, description = "用户注册成功", body = ApiResponse<AuthResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 409, description = "用户账号已存在", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn register(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<RegisterRequest>,
) -> Result<Json<ApiResponse<AuthResponse>>, AppError> {
    info!(
        "【Controller】开始用户注册处理: login_id={}",
        payload.login_id
    );
    debug!(
        "【Controller】注册请求参数: login_id={}, nickname={:?}",
        payload.login_id, payload.nickname
    );

    // 验证请求参数
    user_validator::validate_user_registration(
        &payload.login_id,
        Some(&payload.password),
        payload.nickname.as_deref(),
    )?;

    // 直接调用服务层的注册并生成令牌方法
    let user_login_result = state
        .user_service
        .signup_with_token(
            &payload.login_id,
            &payload.password,
            &payload.nickname.unwrap_or_else(|| DEFAULT_USER_NICKNAME.to_string()),
        )
        .await?;

    // 组装响应
    let response = AuthResponse {
        user_id: user_login_result.user_id,
        nickname: user_login_result.nickname.clone(),
        access_token: user_login_result.access_token,
        refresh_token: user_login_result.refresh_token,
        expires_at: user_login_result.expires_at,
    };

    // 发布领域事件（默认 Noop）
    state.event_bus.publish(DomainEvent::UserRegistered { user_id: user_login_result.user_id.into() }).await;

    // 分派欢迎邮件任务
    #[cfg(feature = "jobs")]
    if let Some(job_service) = &state.job_service {
        if let Err(e) = job_service.dispatch_welcome_email(
            user_login_result.user_id,
            user_login_result.nickname.clone(),
            None, // 暂时没有邮箱信息
        ).await {
            error!("分派欢迎邮件任务失败: {:?}", e);
        } else {
            info!("已分派欢迎邮件任务: user_id={}", user_login_result.user_id);
        }
    }

    info!(
        "【Controller】用户注册成功: user_id={}",
        user_login_result.user_id
    );
    Ok(success_response(response))
}

// 用户登录
#[debug_handler]
pub async fn login(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<LoginRequest>,
) -> Result<Json<ApiResponse<AuthResponse>>, AppError> {
    info!(
        "【Controller】开始用户登录处理: login_id={}",
        payload.login_id
    );

    // 1. 调用登录接口
    let user_login_result = state
        .user_service
        .login(&payload.login_id, &payload.password)
        .await?;

    // 2. 组装响应
    let response = AuthResponse {
        user_id: user_login_result.user_id,
        nickname: user_login_result.nickname,
        access_token: user_login_result.access_token,
        refresh_token: user_login_result.refresh_token,
        expires_at: user_login_result.expires_at,
    };

    info!(
        "【Controller】用户登录成功: user_id={}",
        user_login_result.user_id
    );
    Ok(success_response(response))
}

// 创建临时用户
#[debug_handler]
pub async fn create_temp_user(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<CreateTempUserRequest>,
) -> Result<Json<ApiResponse<AuthResponse>>, AppError> {
    info!(
        "【Controller】开始创建临时用户处理: nickname={:?}",
        payload.nickname
    );

    // 验证昵称（如果提供）
    if let Some(nickname) = &payload.nickname {
        user_validator::validate_nickname(nickname)?;
    }

    // 调用service层实现临时用户创建
    let user_login_result = state
        .user_service
        .create_temp_user(&payload.nickname.unwrap_or_else(|| DEFAULT_TEMP_USER_NICKNAME.to_string()))
        .await?;

    // 构建响应
    let response = AuthResponse {
        user_id: user_login_result.user_id,
        nickname: user_login_result.nickname.clone(),
        access_token: user_login_result.access_token,
        refresh_token: user_login_result.refresh_token,
        expires_at: user_login_result.expires_at,
    };

    // 分派欢迎通知任务
    #[cfg(feature = "jobs")]
    if let Some(job_service) = &state.job_service {
        if let Err(e) = job_service.dispatch_notification(
            user_login_result.user_id,
            "welcome",
            WELCOME_TEMP_USER_MESSAGE.replace("{}", &user_login_result.nickname),
        ).await {
            error!("分派欢迎通知任务失败: {:?}", e);
        } else {
            info!("已分派欢迎通知任务: user_id={}", user_login_result.user_id);
        }
    }

    // 发布领域事件（默认 Noop）
    state.event_bus.publish(DomainEvent::TempUserCreated { user_id: user_login_result.user_id.into() }).await;

    info!(
        "【Controller】临时用户创建成功: user_id={}",
        user_login_result.user_id
    );
    Ok(success_response(response))
}

// 根据位置搜索用户
#[debug_handler]
pub async fn search_user_by_location(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchUserByLocationRequest>,
) -> Result<Json<ApiResponse<SearchUserByLocationResponse>>, AppError> {
    info!(
        "【Controller】开始根据位置搜索用户: 用户ID={}, 经度={}, 纬度={}, 半径={}",
        current_user.user_id, payload.longitude, payload.latitude, payload.radius
    );

    // 验证位置搜索参数
    let (cursor, limit) = crate::utils::validation_pipeline::validators::validate_location_search(
        payload.latitude,
        payload.longitude,
        payload.radius,
        MAX_SEARCH_RADIUS_METERS,
        payload.pagination.cursor,
        payload.pagination.limit,
        DEFAULT_PAGE_SIZE,
        MAX_PAGE_SIZE
    )?;

    // 调用service层方法
    let (users, has_more, next_cursor) = state
        .user_service
        .find_nearby_users(
            payload.longitude,
            payload.latitude,
            payload.radius,
            cursor,
            limit,
            current_user.user_id,
        )
        .await?;

    // 转换为API响应模型
    let user_infos: Vec<UserInfo> = users
        .into_iter()
        .map(|user| UserInfo {
            user_id: user.user_id,
            nickname: user.nickname,
            last_active: user.last_active,
            latitude: user.latitude,
            longitude: user.longitude,
            distance: user.distance,
            location_name: user.location_name,
            last_active_at: user.last_active_at,
        })
        .collect();

    info!(
        "【Controller】根据位置搜索用户成功: 找到{}个用户",
        user_infos.len()
    );
    // 构建分页响应
    let response = PaginatedResponse {
        items: user_infos,
        pagination: PaginationMeta {
            has_more,
            next_cursor,
        },
    };

    Ok(success_response(response))
}

// 根据名称搜索用户
#[debug_handler]
pub async fn search_user_by_name(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchUserByNameRequest>,
) -> Result<Json<ApiResponse<SearchUserByNameResponse>>, AppError> {
    info!(
        "【Controller】开始根据名称搜索用户: 当前用户ID={}, 搜索关键词={}",
        current_user.user_id, payload.keyword
    );

    // 验证关键词搜索参数
    let (cursor, limit) = crate::utils::validation_pipeline::validators::validate_keyword_search(
        &payload.keyword,
        MAX_SEARCH_KEYWORD_LENGTH,
        payload.pagination.cursor,
        payload.pagination.limit,
        DEFAULT_PAGE_SIZE,
        MAX_PAGE_SIZE
    )?;

    // 调用service层方法
    let (users, has_more, next_cursor) = state
        .user_service
        .find_users_by_name(&payload.keyword, cursor, limit, current_user.user_id)
        .await?;

    // 转换为API响应模型
    let user_infos: Vec<UserInfo> = users
        .into_iter()
        .map(|user| UserInfo {
            user_id: user.user_id,
            nickname: user.nickname,
            last_active: user.last_active,
            latitude: user.latitude,
            longitude: user.longitude,
            distance: user.distance,
            location_name: user.location_name,
            last_active_at: user.last_active_at,
        })
        .collect();

    info!(
        "【Controller】根据名称搜索用户成功: 关键词={}, 找到{}个用户",
        payload.keyword,
        user_infos.len()
    );

    // 构建分页响应
    let response = PaginatedResponse {
        items: user_infos,
        pagination: PaginationMeta {
            has_more,
            next_cursor,
        },
    };

    Ok(success_response(response))
}

// 根据ID搜索用户
#[debug_handler]
pub async fn search_user_by_id(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SearchUserByIdRequest>,
) -> Result<Json<ApiResponse<SearchUserByIdResponse>>, AppError> {
    info!(
        "【Controller】开始根据ID搜索用户: 当前用户ID={}, 目标用户ID={}",
        current_user.user_id, payload.user_id
    );

    // 参数转换
    let user_id = payload.user_id;

    // 验证要查找的用户ID
    if !UserIdPartition::is_valid_user_id(user_id) {
        return Err(AppError::BadRequest("无效的用户ID".to_string()));
    }

    // 调用service层方法
    let user = state
        .user_service
        .get_user_by_id(user_id, current_user.user_id)
        .await?;

    info!(
        "【Controller】根据ID搜索用户成功: 用户ID={}, 昵称={}",
        user.user_id, user.nickname
    );

    // 转换为API响应模型
    let user_info = UserInfo {
        user_id: user.user_id,
        nickname: user.nickname,
        last_active: user.last_active,
        latitude: user.latitude,
        longitude: user.longitude,
        distance: user.distance,
        location_name: user.location_name,
        last_active_at: user.last_active_at,
    };

    // 直接返回UserInfo作为SearchUserByIdResponse
    Ok(success_response(user_info))
}

// 更新用户位置
#[debug_handler]
pub async fn update_location(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<UpdateLocationRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    info!(
        "【Controller】开始更新用户位置: 用户ID={}, 经度={}, 纬度={}, 位置名称={:?}",
        current_user.user_id, payload.longitude, payload.latitude, payload.location_name
    );

    // 验证位置参数
    common_validator::validate_location_params(payload.latitude, payload.longitude)?;

    common_validator::validate_location_name(&payload.location_name)?;

    // 调用service层方法
    state
        .user_service
        .update_location(
            current_user.user_id,
            payload.latitude,
            payload.longitude,
            &payload.location_name,
        )
        .await?;

    // 分派位置索引更新任务
    #[cfg(feature = "jobs")]
    if let Some(job_service) = &state.job_service {
        // 生成一个虚拟的生活瞬间ID用于位置更新
        let virtual_moment_id = format!("location_update_{}", current_user.user_id);
        if let Err(e) = job_service.dispatch_moment_index(
            virtual_moment_id,
            current_user.user_id,
            payload.longitude,
            payload.latitude,
        ).await {
            error!("分派位置索引任务失败: {:?}", e);
        } else {
            info!("已分派位置索引任务: user_id={}", current_user.user_id);
        }
    }

    info!(
        "【Controller】用户位置更新成功: 用户ID={}",
        current_user.user_id
    );
    Ok(success_response(()))
}

// 刷新令牌
#[debug_handler]
pub async fn refresh_token(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<RefreshTokenRequest>,
) -> Result<Json<ApiResponse<RefreshTokenResponse>>, AppError> {
    info!("【Controller】开始刷新令牌处理");
    debug!(
        "【Controller】刷新令牌请求: refresh_token={}",
        payload.refresh_token
    );

    // 调用service层方法
    let token_result = state
        .user_service
        .refresh_token(&payload.refresh_token)
        .await?;

    // 构建响应
    let response = RefreshTokenResponse {
        access_token: token_result.access_token,
        refresh_token: token_result.refresh_token,
        expires_at: token_result.expires_at,
    };

    info!(
        "【Controller】令牌刷新成功: 过期时间={}",
        token_result.expires_at
    );
    Ok(success_response(response))
}

// 设置/修改密码（仅正式用户可用）
#[debug_handler]
pub async fn set_password(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<SetPasswordRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    info!(
        "【Controller】开始处理设置/修改密码请求: 用户ID={}",
        current_user.user_id
    );
    
    // 验证是否为临时用户（虽然service层也会检查，但这里提前检查可以减少无效请求）
    if state.user_service.is_temp_user(current_user.user_id) {
        return Err(AppError::Forbidden(
            "临时用户不能设置或修改密码，请注册正式账号".into(),
        ));
    }
    
    // 验证新密码格式是否符合要求
    user_validator::validate_password(&payload.new_password)?;
    
    // 检查用户是否已设置密码
    let has_password = state.user_service.is_password_set(current_user.user_id).await?;
    debug!(
        "【Controller】用户密码状态检查: user_id={}, has_password={}",
        current_user.user_id, has_password
    );
    
    // 首次设置密码时不需要旧密码；修改密码时必须提供旧密码
    if has_password && payload.old_password.is_none() {
        return Err(AppError::BadRequest("修改密码时必须提供原密码".to_string()));
    }
    
    // 调用service层方法
    state
        .user_service
        .set_password(
            current_user.user_id, 
            &payload.new_password, 
            payload.old_password.as_deref()
        )
        .await?;
    
    info!(
        "【Controller】用户{}密码成功: user_id={}",
        if has_password { "修改" } else { "设置" },
        current_user.user_id
    );
    Ok(success_response(()))
}

/// 检查认证状态
///
/// 验证用户的认证状态并返回基本信息
#[utoipa::path(
    get,
    path = "/api/v1/user/auth/status",
    tag = "用户",
    responses(
        (status = 200, description = "认证状态检查成功", body = ApiResponse<AuthResponse>),
        (status = 401, description = "未认证", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn check_auth_status(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
) -> Result<Json<ApiResponse<AuthResponse>>, AppError> {
    info!("【Controller】检查认证状态: user_id={}", current_user.user_id);
    
    // 获取用户认证信息
    let auth_user = state
        .user_service
        .get_auth_info(current_user.user_id)
        .await
        .map_err(|e| AppError::InternalServerError(format!("获取用户认证信息失败: {:?}", e)))?;
    
    info!("【Controller】认证状态检查成功: user_id={}", current_user.user_id);
    Ok(success_response(AuthResponse {
        user_id: auth_user.user_id,
        nickname: auth_user.nickname,
        access_token: "verified".to_string(), // 状态检查时返回验证状态
        refresh_token: "verified".to_string(),
        expires_at: chrono::Utc::now() + chrono::Duration::hours(1), // 1小时后过期
    }))
}

/// 公开健康检查
/// 
/// 检查服务状态，不需要认证
#[utoipa::path(
    post,
    path = "/api/v1/auth/health",
    tag = "认证",
    responses(
        (status = 200, description = "服务状态正常", body = ApiResponse<serde_json::Value>)
    )
)]
#[debug_handler]
pub async fn public_health_check(
    State(state): State<Arc<AppState>>,
) -> Result<Json<ApiResponse<serde_json::Value>>, AppError> {
    info!("【Controller】公开健康检查");
    
    // 检查数据库连接
    let _db_health = state.health_service.check_database().await
        .map_err(|e| AppError::InternalServerError(format!("数据库健康检查失败: {:?}", e)))?;
    
    Ok(success_response(json!({
        "status": "healthy",
        "service": "hotspot-api",
        "timestamp": chrono::Utc::now(),
        "version": env!("CARGO_PKG_VERSION")
    })))
}

// 通知历史和未读数量相关功能已移除，这些应该通过WebSocket实时获取，而非HTTP API

/// 更新用户昵称
///
/// 更新当前用户的昵称
#[utoipa::path(
    post,
    path = "/api/v1/user/update_nickname",
    tag = "用户",
    request_body = UpdateNicknameRequest,
    responses(
        (status = 200, description = "昵称更新成功", body = ApiResponse<()>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 401, description = "未认证", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn update_nickname(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<crate::models::api::user::UpdateNicknameRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    info!("【Controller】更新昵称: user_id={}, new_nickname={}", 
          current_user.user_id, payload.nickname);

    // 验证昵称
    user_validator::validate_nickname(&payload.nickname)?;

    // 更新昵称
    state
        .user_service
        .update_nickname(current_user.user_id, &payload.nickname)
        .await
        .map_err(|e| AppError::InternalServerError(format!("更新昵称失败: {:?}", e)))?;

    info!("【Controller】昵称更新成功: user_id={}", current_user.user_id);
    Ok(success_response(()))
}

/// 升级临时账户
///
/// 将临时用户升级为正式用户
#[utoipa::path(
    post,
    path = "/api/v1/user/upgrade",
    tag = "用户",
    request_body = UpgradeAccountRequest,
    responses(
        (status = 200, description = "账户升级成功", body = ApiResponse<AuthResponse>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 401, description = "未认证", body = ApiResponse<()>),
        (status = 409, description = "登录ID已存在", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn upgrade_account(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<crate::models::api::user::UpgradeAccountRequest>,
) -> Result<Json<ApiResponse<AuthResponse>>, AppError> {
    info!("【Controller】升级账户: user_id={}, login_id={}", 
          current_user.user_id, payload.login_id);

    // 验证请求参数
    user_validator::validate_user_registration(
        &payload.login_id,
        Some(&payload.password),
        None,
    )?;

    // 升级账户
    let auth_response = state
        .user_service
        .upgrade_temp_user(current_user.user_id, &payload.login_id, &payload.password)
        .await
        .map_err(|e| AppError::InternalServerError(format!("升级账户失败: {:?}", e)))?;

    info!("【Controller】账户升级成功: user_id={}", current_user.user_id);
    Ok(success_response(auth_response))
}

/// 获取用户设置
///
/// 获取当前用户的设置信息
#[utoipa::path(
    get,
    path = "/api/v1/user/get_settings",
    tag = "用户",
    responses(
        (status = 200, description = "获取设置成功", body = ApiResponse<UserSettings>),
        (status = 401, description = "未认证", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn get_user_settings(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
) -> Result<Json<ApiResponse<UserSettings>>, AppError> {
    info!("【Controller】获取用户设置: user_id={}", current_user.user_id);

    let settings = state
        .user_service
        .get_user_settings(current_user.user_id)
        .await
        .map_err(|e| AppError::InternalServerError(format!("获取用户设置失败: {:?}", e)))?;

    info!("【Controller】获取用户设置成功: user_id={}", current_user.user_id);
    Ok(success_response(settings))
}

/// 更新用户设置
///
/// 更新当前用户的设置信息
#[utoipa::path(
    post,
    path = "/api/v1/user/update_settings",
    tag = "用户",
    request_body = UserSettingsRequest,
    responses(
        (status = 200, description = "设置更新成功", body = ApiResponse<()>),
        (status = 400, description = "请求参数错误", body = ApiResponse<()>),
        (status = 401, description = "未认证", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn update_user_settings(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
    Json(payload): Json<crate::models::api::user::UserSettingsRequest>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    info!("【Controller】更新用户设置: user_id={}", current_user.user_id);

    // 更新设置
    state
        .user_service
        .update_user_settings(current_user.user_id, payload)
        .await
        .map_err(|e| AppError::InternalServerError(format!("更新用户设置失败: {:?}", e)))?;

    info!("【Controller】用户设置更新成功: user_id={}", current_user.user_id);
    Ok(success_response(()))
}

/// 检查密码状态
///
/// 检查用户是否已设置密码
#[utoipa::path(
    get,
    path = "/api/v1/user/password-status",
    tag = "用户",
    responses(
        (status = 200, description = "检查成功", body = ApiResponse<PasswordStatusResponse>),
        (status = 401, description = "未认证", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn check_password_status(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
) -> Result<Json<ApiResponse<crate::models::api::user::PasswordStatusResponse>>, AppError> {
    info!("【Controller】检查密码状态: user_id={}", current_user.user_id);

    let is_password_set = state
        .user_service
        .is_password_set(current_user.user_id)
        .await
        .map_err(|e| AppError::InternalServerError(format!("检查密码状态失败: {:?}", e)))?;

    let response = crate::models::api::user::PasswordStatusResponse { is_password_set };
    
    info!("【Controller】密码状态检查成功: user_id={}, is_password_set={}", 
          current_user.user_id, is_password_set);
    Ok(success_response(response))
}

/// 删除用户账户
///
/// 删除当前用户账户及所有相关数据
#[utoipa::path(
    post,
    path = "/api/v1/user/delete",
    tag = "用户",
    responses(
        (status = 200, description = "账户删除成功", body = ApiResponse<()>),
        (status = 401, description = "未认证", body = ApiResponse<()>)
    )
)]
#[debug_handler]
pub async fn delete_account(
    Extension(current_user): Extension<AuthUser>,
    State(state): State<Arc<AppState>>,
) -> Result<Json<ApiResponse<()>>, AppError> {
    info!("【Controller】删除账户: user_id={}", current_user.user_id);

    // 删除用户账户
    state
        .user_service
        .delete_user(current_user.user_id)
        .await
        .map_err(|e| AppError::InternalServerError(format!("删除账户失败: {:?}", e)))?;

    info!("【Controller】账户删除成功: user_id={}", current_user.user_id);
    Ok(success_response(()))
}
