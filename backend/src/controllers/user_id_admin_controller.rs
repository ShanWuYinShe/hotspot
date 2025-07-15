//! 用户ID分区系统管理控制器
//! 提供用户ID分区相关的管理和查询接口

use axum::{
    debug_handler,
    extract::{State, Extension},
    Json,
};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tracing::{info, warn};

use crate::{
    middleware::auth::AppState,
    models::api::user::AuthUser,
    repositories::user_repository::UserRepository,
    services::user::service::{UserTypeInfo, UserTypeRangeInfo},
    utils::{
        response::{success_response, ApiResponse, AppError},
        UserIdPartition, UserType,
    },
};

/// 用户ID分区统计信息
#[derive(Debug, Serialize)]
pub struct UserIdPartitionStats {
    pub total_users: i64,
    pub regular_users: i64,
    pub temporary_users: i64,
    pub admin_users: i64,
    pub system_users: i64,
    pub type_ranges: Vec<UserTypeRangeInfo>,
}

/// 用户ID验证请求
#[derive(Debug, Deserialize)]
pub struct ValidateUserIdRequest {
    pub user_id: i64,
}

/// 用户ID验证响应
#[derive(Debug, Serialize)]
pub struct ValidateUserIdResponse {
    pub is_valid: bool,
    pub user_info: Option<UserTypeInfo>,
    pub error: Option<String>,
}

/// 批量生成用户ID请求
#[derive(Debug, Deserialize)]
pub struct GenerateUserIdsRequest {
    pub user_type: String, // "regular", "temporary", "admin", "system", "test"
    pub count: usize,      // 生成数量，最多100
    pub start_sequence: Option<i64>, // 起始序列号，可选
}

/// 批量生成用户ID响应
#[derive(Debug, Serialize)]
pub struct GenerateUserIdsResponse {
    pub user_type: UserType,
    pub generated_ids: Vec<GeneratedUserId>,
    pub success_count: usize,
    pub error_count: usize,
}

/// 生成的用户ID信息
#[derive(Debug, Serialize)]
pub struct GeneratedUserId {
    pub user_id: i64,
    pub sequence: i64,
    pub formatted: String,
}

/// 获取用户ID分区系统统计信息
/// 
/// 管理员专用接口，显示各用户类型的统计信息和容量情况
#[debug_handler]
pub async fn get_partition_stats(
    State(app_state): State<Arc<AppState>>,
    Extension(_current_user): Extension<AuthUser>, // 需要认证，但不限制用户类型（便于测试）
) -> Result<Json<ApiResponse<UserIdPartitionStats>>, AppError> {
    info!("获取用户ID分区统计信息");

    // 从数据库获取各类型用户数量
    let user_repository = &app_state.user_repository;
    
    let total_users = user_repository.get_total_user_count().await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;
        
    let regular_users = user_repository.get_user_count_by_type(UserType::Regular).await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;
        
    let temporary_users = user_repository.get_user_count_by_type(UserType::Temporary).await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;
        
    let admin_users = user_repository.get_user_count_by_type(UserType::Admin).await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;
        
    let system_users = user_repository.get_user_count_by_type(UserType::System).await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

    let stats = UserIdPartitionStats {
        total_users,
        regular_users,
        temporary_users,
        admin_users,
        system_users,
        type_ranges: crate::services::user::service::UserService::get_user_type_ranges(),
    };

    Ok(success_response(stats))
}

/// 验证用户ID的有效性和类型信息
#[debug_handler]
pub async fn validate_user_id(
    State(app_state): State<Arc<AppState>>,
    Extension(_current_user): Extension<AuthUser>,
    Json(request): Json<ValidateUserIdRequest>,
) -> Result<Json<ApiResponse<ValidateUserIdResponse>>, AppError> {
    info!("验证用户ID: {}", request.user_id);

    let user_service = &app_state.user_service;
    
    let (is_valid, user_info, error) = if user_service.is_valid_user_id(request.user_id) {
        match user_service.get_user_type_info(request.user_id) {
            Ok(info) => (true, Some(info), None),
            Err(e) => (false, None, Some(e.to_string())),
        }
    } else {
        (false, None, Some("无效的用户ID格式".to_string()))
    };

    let response = ValidateUserIdResponse {
        is_valid,
        user_info,
        error,
    };

    Ok(success_response(response))
}

/// 根据用户ID获取用户类型信息
#[debug_handler]
pub async fn get_user_type_info(
    State(app_state): State<Arc<AppState>>,
    Extension(_current_user): Extension<AuthUser>,
    Json(request): Json<ValidateUserIdRequest>, // 改为从请求体获取user_id
) -> Result<Json<ApiResponse<UserTypeInfo>>, AppError> {
    info!("获取用户类型信息: {}", request.user_id);

    let user_service = &app_state.user_service;
    let user_info = user_service.get_user_type_info(request.user_id)?;

    Ok(success_response(user_info))
}

/// 获取所有用户类型的范围信息
#[debug_handler]
pub async fn get_user_type_ranges(
    Extension(_current_user): Extension<AuthUser>,
    Json(_request): Json<serde_json::Value>, // 添加空的请求体参数
) -> Result<Json<ApiResponse<Vec<UserTypeRangeInfo>>>, AppError> {
    info!("获取用户类型范围信息");

    let ranges = crate::services::user::UserService::get_user_type_ranges();
    Ok(success_response(ranges))
}

/// 批量生成指定类型的用户ID（测试用）
/// 
/// 注意：这是一个测试接口，生产环境中应该谨慎使用
#[debug_handler]
pub async fn generate_user_ids(
    Extension(_current_user): Extension<AuthUser>,
    Json(request): Json<GenerateUserIdsRequest>,
) -> Result<Json<ApiResponse<GenerateUserIdsResponse>>, AppError> {
    info!("批量生成用户ID: 类型={}, 数量={}", request.user_type, request.count);

    // 限制生成数量
    if request.count > 100 {
        return Err(AppError::BadRequest("单次生成数量不能超过100".to_string()));
    }

    // 解析用户类型
    let user_type = match request.user_type.as_str() {
        "regular" => UserType::Regular,
        "temporary" => UserType::Temporary,
        "admin" => UserType::Admin,
        "system" => UserType::System,
        _ => return Err(AppError::BadRequest("无效的用户类型".to_string())),
    };

    let mut generated_ids = Vec::new();
    let mut success_count = 0;
    let mut error_count = 0;
    
    let start_seq = request.start_sequence.unwrap_or(1);

    for i in 0..request.count {
        let sequence = start_seq + i as i64;
        
        match UserIdPartition::generate_user_id(user_type, sequence) {
            Ok(user_id) => {
                generated_ids.push(GeneratedUserId {
                    user_id,
                    sequence,
                    formatted: UserIdPartition::format_user_id(user_id),
                });
                success_count += 1;
            }
            Err(e) => {
                warn!("生成用户ID失败: sequence={}, error={}", sequence, e);
                error_count += 1;
            }
        }
    }

    let response = GenerateUserIdsResponse {
        user_type,
        generated_ids,
        success_count,
        error_count,
    };

    Ok(success_response(response))
}

/// 格式化用户ID请求
#[derive(Debug, Deserialize)]
pub struct FormatUserIdRequest {
    pub user_id: i64,
}

/// 格式化用户ID响应
#[derive(Debug, Serialize)]
pub struct FormatUserIdResponse {
    pub user_id: i64,
    pub formatted: String,
    pub user_type: UserType,
}

/// 格式化用户ID显示
#[debug_handler]
pub async fn format_user_id(
    State(app_state): State<Arc<AppState>>,
    Extension(_current_user): Extension<AuthUser>,
    Json(request): Json<FormatUserIdRequest>,
) -> Result<Json<ApiResponse<FormatUserIdResponse>>, AppError> {
    info!("格式化用户ID: {}", request.user_id);

    let user_service = &app_state.user_service;
    let user_info = user_service.get_user_type_info(request.user_id)?;
    let formatted = format!("{:016}", request.user_id);

    let response = FormatUserIdResponse {
        user_id: request.user_id,
        formatted,
        user_type: user_info.user_type,
    };

    Ok(success_response(response))
}
