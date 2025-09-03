/// 用户HTTP控制器
/// 
/// 处理用户相关的HTTP请求，调用用户应用服务

use axum::extract::State;
use axum::response::Response;
use axum::Json;
use serde::{Deserialize, Serialize};

use crate::application::{
    CommandHandler, QueryHandler,
    RegisterUserCommand, CreateTempUserCommand, UpgradeUserCommand,
    UpdateUserLocationCommand, UpdateUserProfileCommand, SetUserPasswordCommand,
    DeleteUserAccountCommand,
    GetUserByIdQuery, GetUserProfileQuery, FindNearbyUsersQuery,
    SearchUsersByNicknameQuery, CheckPasswordStatusQuery,
    PaginationQuery, UserApplicationServicePort,
};
use super::{Controller, ok_response, created_response, map_domain_error_to_http_response, paginated_response};

/// 用户控制器
#[derive(Clone)]
pub struct UserController<UserAppService> {
    user_service: UserAppService,
}

impl<UserAppService> UserController<UserAppService>
where
    UserAppService: CommandHandler<RegisterUserCommand, Result = crate::application::UserRegistrationResult> + 
                   CommandHandler<CreateTempUserCommand, Result = crate::application::UserRegistrationResult> +
                   CommandHandler<UpgradeUserCommand, Result = crate::application::UserCommandResult> +
                   CommandHandler<UpdateUserLocationCommand, Result = crate::application::LocationUpdateResult> +
                   CommandHandler<UpdateUserProfileCommand, Result = crate::application::UserCommandResult> +
                   CommandHandler<SetUserPasswordCommand, Result = crate::application::UserCommandResult> +
                   CommandHandler<DeleteUserAccountCommand, Result = crate::application::UserCommandResult> +
                   QueryHandler<GetUserByIdQuery, Result = Option<crate::domain::User>> +
                   QueryHandler<GetUserProfileQuery, Result = crate::application::UserProfile> +
                   QueryHandler<FindNearbyUsersQuery, Result = crate::application::PaginatedResult<crate::application::NearbyUser>> +
                   QueryHandler<SearchUsersByNicknameQuery, Result = crate::application::PaginatedResult<crate::application::UserSearchResult>> +
                   QueryHandler<CheckPasswordStatusQuery, Result = crate::application::PasswordStatus> +
                   UserApplicationServicePort +
                   Clone + Send + Sync + 'static,
{
    /// 创建用户控制器
    pub fn new(user_service: UserAppService) -> Self {
        Self { user_service }
    }
    
    // ===== 认证API接口 (7个端点) =====
    
    /// 用户注册
    pub async fn register_user(
        State(controller): State<Self>,
        Json(request): Json<RegisterUserRequest>,
    ) -> Response {
        let command = RegisterUserCommand {
            login_id: request.login_id,
            nickname: Some(request.nickname),
            password: request.password,
        };
        
        match controller.user_service.handle_command(command).await {
            Ok(result) => {
                let (access_token, expires_at) = match crate::utils::jwt::generate_jwt_token(
                    result.user.id,
                    &std::env::var("JWT_SECRET").unwrap_or_else(|_| "your-secret-key-change-in-production".to_string()),
                    3600, // 1小时
                    result.user.user_type == crate::domain::UserType::Temporary,
                ) {
                    Ok(token_data) => token_data,
                    Err(_) => return map_domain_error_to_http_response(
                        crate::domain::DomainError::BusinessRuleViolation { rule: "Token generation failed".to_string() }
                    ),
                };
                
                let response = AuthResponse {
                    user_id: result.user.id,
                    nickname: result.user.nickname.clone(),
                    login_id: result.user.login_id.clone(),
                    access_token,
                    refresh_token: "refresh_token_placeholder".to_string(),
                    expires_at,
                    is_temp: Some(result.user.user_type == crate::domain::UserType::Temporary),
                };
                created_response(response)
            }
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 用户登录
    pub async fn login(
        State(controller): State<Self>,
        Json(request): Json<LoginRequest>,
    ) -> Response {
        match controller.user_service.verify_user_password(&request.login_id, &request.password).await {
            Ok(user) => {
                // 生成JWT token对
                let jwt_secret = std::env::var("JWT_SECRET")
                    .unwrap_or_else(|_| "your-secret-key-change-in-production".to_string());
                let is_temp = user.user_type == crate::domain::UserType::Temporary;
                
                let (access_token, refresh_token, expires_at, _refresh_expires_at) = 
                    match crate::utils::jwt::generate_token_pair(user.id, &jwt_secret, is_temp) {
                        Ok(tokens) => tokens,
                        Err(_) => return map_domain_error_to_http_response(
                            crate::domain::DomainError::BusinessRuleViolation { rule: "Token generation failed".to_string() }
                        ),
                    };
                
                let response = AuthResponse {
                    user_id: user.id,
                    nickname: user.nickname,
                    login_id: user.login_id,
                    access_token,
                    refresh_token,
                    expires_at,
                    is_temp: Some(is_temp),
                };
                ok_response(response)
            }
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 创建临时用户
    pub async fn create_temp_user(
        State(controller): State<Self>,
        Json(request): Json<CreateTempUserRequest>,
    ) -> Response {
        let command = CreateTempUserCommand {
            nickname: request.nickname,
        };
        
        match controller.user_service.handle_command(command).await {
            Ok(result) => {
                let jwt_secret = std::env::var("JWT_SECRET")
                    .unwrap_or_else(|_| "your-secret-key-change-in-production".to_string());
                
                let (access_token, refresh_token, expires_at, _refresh_expires_at) = 
                    match crate::utils::jwt::generate_token_pair(result.user.id, &jwt_secret, true) {
                        Ok(tokens) => tokens,
                        Err(_) => return map_domain_error_to_http_response(
                            crate::domain::DomainError::BusinessRuleViolation { rule: "Token generation failed".to_string() }
                        ),
                    };
                
                let response = AuthResponse {
                    user_id: result.user.id,
                    nickname: result.user.nickname.clone(),
                    login_id: result.user.login_id.clone(),
                    access_token,
                    refresh_token,
                    expires_at,
                    is_temp: Some(true),
                };
                created_response(response)
            }
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 刷新Token
    pub async fn refresh_token(
        State(controller): State<Self>,
        Json(request): Json<RefreshTokenRequest>,
    ) -> Response {
        let jwt_secret = std::env::var("JWT_SECRET")
            .unwrap_or_else(|_| "your-secret-key-change-in-production".to_string());
        
        // 验证refresh token
        let refresh_claims = match crate::utils::jwt::verify_refresh_token(&request.refresh_token, &jwt_secret) {
            Ok(claims) => claims,
            Err(_) => return map_domain_error_to_http_response(
                crate::domain::DomainError::User(crate::domain::UserDomainError::InvalidCredentials)
            ),
        };
        
        // 验证用户是否存在
        let query = crate::application::GetUserByIdQuery { user_id: refresh_claims.sub };
        let user = match controller.user_service.handle_query(query).await {
            Ok(Some(user)) => user,
            Ok(None) => return map_domain_error_to_http_response(
                crate::domain::DomainError::User(crate::domain::UserDomainError::UserNotFound { user_id: refresh_claims.sub })
            ),
            Err(error) => return map_domain_error_to_http_response(error),
        };
        
        // 生成新的token对
        let is_temp = user.user_type == crate::domain::UserType::Temporary;
        let (access_token, refresh_token, expires_at, _refresh_expires_at) = 
            match crate::utils::jwt::generate_token_pair(user.id, &jwt_secret, is_temp) {
                Ok(tokens) => tokens,
                Err(_) => return map_domain_error_to_http_response(
                    crate::domain::DomainError::BusinessRuleViolation { rule: "Token generation failed".to_string() }
                ),
            };
        
        let response = AuthResponse {
            user_id: user.id,
            nickname: user.nickname,
            login_id: user.login_id,
            access_token,
            refresh_token,
            expires_at,
            is_temp: Some(is_temp),
        };
        ok_response(response)
    }
    
    /// 检查认证状态
    pub async fn check_auth_status(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<crate::presentation::http::auth::CurrentUser>,
    ) -> Response {
        let query = GetUserByIdQuery { user_id: current_user.user_id };
        
        match controller.user_service.handle_query(query).await {
            Ok(Some(user)) => {
                let jwt_secret = std::env::var("JWT_SECRET")
                    .unwrap_or_else(|_| "your-secret-key-change-in-production".to_string());
                let is_temp = user.user_type == crate::domain::UserType::Temporary;
                
                let (access_token, refresh_token, expires_at, _refresh_expires_at) = 
                    match crate::utils::jwt::generate_token_pair(user.id, &jwt_secret, is_temp) {
                        Ok(tokens) => tokens,
                        Err(_) => return map_domain_error_to_http_response(
                            crate::domain::DomainError::BusinessRuleViolation { rule: "Token generation failed".to_string() }
                        ),
                    };
                
                let response = AuthResponse {
                    user_id: user.id,
                    nickname: user.nickname,
                    login_id: user.login_id,
                    access_token,
                    refresh_token,
                    expires_at,
                    is_temp: Some(is_temp),
                };
                ok_response(response)
            }
            Ok(None) => map_domain_error_to_http_response(
                crate::domain::DomainError::User(crate::domain::UserDomainError::UserNotFound { user_id: current_user.user_id })
            ),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 设置密码
    pub async fn set_password(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<SetPasswordRequest>,
    ) -> Response {
        let command = SetUserPasswordCommand {
            user_id: current_user.user_id,
            password: request.new_password,
        };
        
        match controller.user_service.handle_command(command).await {
            Ok(_result) => ok_response(serde_json::json!({"success": true, "message": "密码设置成功"})),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 检查密码状态 (认证版本)
    pub async fn password_status(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<crate::presentation::http::auth::CurrentUser>,
    ) -> Response {
        let query = CheckPasswordStatusQuery { user_id: current_user.user_id };
        
        match controller.user_service.handle_query(query).await {
            Ok(status) => ok_response(PasswordStatusResult { has_password: status.has_password }),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    // ===== 用户数据API接口 (9个端点) =====
    
    /// 按昵称搜索用户 (POST版本)
    pub async fn search_by_name(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<SearchByNameRequest>,
    ) -> Response {
        let query = SearchUsersByNicknameQuery {
            keyword: request.query,
            requester_id: current_user.user_id,
            pagination: PaginationQuery {
                cursor: request.cursor,
                limit: request.limit.map(|v| v as i32),
            },
        };
        
        match controller.user_service.handle_query(query).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取用户资料 (POST版本)
    pub async fn search_by_id(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<SearchByIdRequest>,
    ) -> Response {
        let query = GetUserProfileQuery {
            user_id: request.user_id,
            requester_id: current_user.user_id,
        };
        
        match controller.user_service.handle_query(query).await {
            Ok(profile) => ok_response(profile),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 按位置搜索附近用户 (POST版本)
    pub async fn search_by_location(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<SearchByLocationRequest>,
    ) -> Response {
        let query = FindNearbyUsersQuery {
            user_id: current_user.user_id,
            latitude: request.latitude,
            longitude: request.longitude,
            radius_km: Some(request.radius / 1000.0), // 转换米到公里
            pagination: PaginationQuery {
                cursor: request.cursor,
                limit: request.limit.map(|v| v as i32),
            },
        };
        
        match controller.user_service.handle_query(query).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 更新用户位置 (POST版本)
    pub async fn update_location(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<UpdateLocationRequest>,
    ) -> Response {
        let command = UpdateUserLocationCommand {
            user_id: current_user.user_id,
            latitude: request.latitude,
            longitude: request.longitude,
            address: Some(request.location_name.unwrap_or_default()),
        };
        
        match controller.user_service.handle_command(command).await {
            Ok(_) => ok_response(serde_json::json!({"success": true})),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 更新用户资料 (POST版本)
    pub async fn update_profile(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<UpdateProfileRequest>,
    ) -> Response {
        let command = UpdateUserProfileCommand {
            user_id: current_user.user_id,
            nickname: request.nickname.clone(),
        };
        
        match controller.user_service.handle_command(command).await {
            Ok(_) => ok_response(serde_json::json!({"nickname": request.nickname.unwrap_or_default()})),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 账户升级 (POST版本)
    pub async fn upgrade_account(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<UpgradeAccountRequest>,
    ) -> Response {
        let command = UpgradeUserCommand {
            user_id: current_user.user_id,
            login_id: request.login_id.clone(),
            password: request.password,
        };
        
        match controller.user_service.handle_command(command).await {
            Ok(_result) => {
                // 获取升级后的用户信息
                let query = GetUserByIdQuery { user_id: current_user.user_id };
                let updated_user = match controller.user_service.handle_query(query).await {
                    Ok(Some(user)) => user,
                    Ok(None) => return map_domain_error_to_http_response(
                        crate::domain::DomainError::User(crate::domain::UserDomainError::UserNotFound { user_id: current_user.user_id })
                    ),
                    Err(error) => return map_domain_error_to_http_response(error),
                };
                
                // 生成新的JWT token
                let (access_token, expires_at) = match crate::utils::jwt::generate_jwt_token(
                    current_user.user_id,
                    &std::env::var("JWT_SECRET").unwrap_or_else(|_| "your-secret-key-change-in-production".to_string()),
                    3600,
                    false, // 升级后不再是临时用户
                ) {
                    Ok(token_data) => token_data,
                    Err(_) => return map_domain_error_to_http_response(
                        crate::domain::DomainError::BusinessRuleViolation { rule: "Token generation failed".to_string() }
                    ),
                };
                
                let response = AuthResponse {
                    user_id: current_user.user_id,
                    nickname: updated_user.nickname,
                    login_id: Some(request.login_id),
                    access_token,
                    refresh_token: "refresh_token_placeholder".to_string(),
                    expires_at,
                    is_temp: Some(false),
                };
                ok_response(response)
            }
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
    
    /// 获取用户设置
    pub async fn get_settings(
        State(_controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<crate::presentation::http::auth::CurrentUser>,
    ) -> Response {
        // 返回默认设置
        let default_settings = serde_json::json!({
            "notification_level": "normal",
            "message_notifications": true,
            "group_notifications": true,
            "system_notifications": true,
            "location_notifications": true,
            "quiet_hours_enabled": false,
            "quiet_hours_start": "22:00",
            "quiet_hours_end": "08:00",
            "notification_radius_km": 5.0,
            "sound_enabled": true,
            "vibration_enabled": true,
            "user_id": current_user.user_id
        });
        ok_response(default_settings)
    }
    
    /// 更新用户设置
    pub async fn update_settings(
        State(_controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<crate::presentation::http::auth::CurrentUser>,
        Json(request): Json<UpdateSettingsRequest>,
    ) -> Response {
        // 简单实现：返回更新后的设置（实际上只是回显请求的设置）
        let updated_settings = serde_json::json!({
            "user_id": current_user.user_id,
            "notification_level": request.notification_level.unwrap_or_else(|| "normal".to_string()),
            "message_notifications": request.message_notifications.unwrap_or(true),
            "group_notifications": request.group_notifications.unwrap_or(true),
            "system_notifications": request.system_notifications.unwrap_or(true),
            "location_notifications": request.location_notifications.unwrap_or(true),
            "quiet_hours_enabled": request.quiet_hours_enabled.unwrap_or(false),
            "quiet_hours_start": request.quiet_hours_start.unwrap_or_else(|| "22:00".to_string()),
            "quiet_hours_end": request.quiet_hours_end.unwrap_or_else(|| "08:00".to_string()),
            "notification_radius_km": request.notification_radius_km.unwrap_or(5.0),
            "sound_enabled": request.sound_enabled.unwrap_or(true),
            "vibration_enabled": request.vibration_enabled.unwrap_or(true),
            "success": true,
            "message": "设置更新成功"
        });
        ok_response(updated_settings)
    }
    
    /// 删除账户
    pub async fn delete_account(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<crate::presentation::http::auth::CurrentUser>,
    ) -> Response {
        let command = DeleteUserAccountCommand {
            user_id: current_user.user_id,
            requester_id: current_user.user_id,
        };
        
        match controller.user_service.handle_command(command).await {
            Ok(_result) => ok_response(serde_json::json!({"success": true, "message": "账户删除成功"})),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
}

#[async_trait::async_trait]
impl<UserAppService> Controller for UserController<UserAppService>
where
    UserAppService: Send + Sync,
{
    fn name(&self) -> &'static str {
        "UserController"
    }
}

// ===== 请求/响应数据结构 =====

#[derive(Debug, Deserialize)]
pub struct LoginRequest {
    pub login_id: String,
    pub password: String,
}

#[derive(Debug, Deserialize)]
pub struct RegisterUserRequest {
    pub login_id: String,
    pub nickname: String,
    pub password: String,
}

#[derive(Debug, Deserialize)]
pub struct CreateTempUserRequest {
    pub nickname: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct RefreshTokenRequest {
    pub refresh_token: String,
}

#[derive(Debug, Deserialize)]
pub struct SetPasswordRequest {
    pub new_password: String,
    pub old_password: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchByNameRequest {
    pub query: String,
    pub cursor: Option<i64>,
    pub limit: Option<i64>,
    pub requester_id: Option<i64>,
}

#[derive(Debug, Deserialize)]
pub struct SearchByIdRequest {
    pub user_id: i64,
    pub requester_id: Option<i64>,
}

#[derive(Debug, Deserialize)]
pub struct SearchByLocationRequest {
    pub latitude: f64,
    pub longitude: f64,
    pub radius: f64, // 单位：米
    pub cursor: Option<i64>,
    pub limit: Option<i64>,
    pub requester_id: Option<i64>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateLocationRequest {
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateProfileRequest {
    pub nickname: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpgradeAccountRequest {
    pub login_id: String,
    pub password: String,
}

#[derive(Debug, Deserialize)]
pub struct UpdateSettingsRequest {
    pub notification_level: Option<String>,
    pub message_notifications: Option<bool>,
    pub group_notifications: Option<bool>,
    pub system_notifications: Option<bool>,
    pub location_notifications: Option<bool>,
    pub quiet_hours_enabled: Option<bool>,
    pub quiet_hours_start: Option<String>,
    pub quiet_hours_end: Option<String>,
    pub notification_radius_km: Option<f64>,
    pub sound_enabled: Option<bool>,
    pub vibration_enabled: Option<bool>,
}

// ===== 响应数据结构 =====

#[derive(Debug, Serialize)]
pub struct AuthResponse {
    pub user_id: i64,
    pub nickname: String,
    pub login_id: Option<String>,
    pub access_token: String,
    pub refresh_token: String,
    pub expires_at: chrono::DateTime<chrono::Utc>,
    pub is_temp: Option<bool>,
}

#[derive(Debug, Serialize)]
pub struct PasswordStatusResult {
    pub has_password: bool,
}
