/// 内容安全控制器
/// 
/// 功能职责：
/// - 处理用户举报功能
/// - 管理举报记录和处理流程
/// - 提供管理员审核接口
/// - 处理用户屏蔽操作
/// 
/// API端点：
/// - POST /api/v1/protected/reports/create - 提交举报
/// - POST /api/v1/protected/reports/my_reports - 我的举报记录
/// - POST /api/v1/protected/reports/list - 举报列表（管理员）
/// - POST /api/v1/protected/reports/handle - 处理举报（管理员）
/// - POST /api/v1/protected/users/block - 屏蔽用户

use axum::{extract::{State, Extension}, http::StatusCode, response::IntoResponse, Json};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tracing::debug;
use utoipa::ToSchema;

use crate::application::services::security_application_service::SecurityApplicationService;
use crate::application::services::user_application_service;
use crate::presentation::http::models::ApiResponse;
use crate::presentation::http::auth::CurrentUser;
use crate::adapters::adapters_pg::report_repository::{ReportTargetType, ReportReason};

/// 内容安全控制器
#[derive(Clone)]
pub struct SecurityController {
    security_service: Arc<dyn SecurityApplicationService>,
    user_service: Arc<dyn user_application_service::UserApplicationServicePort>,
}

impl SecurityController {
    pub fn new(
        security_service: Arc<dyn SecurityApplicationService>,
        user_service: Arc<dyn user_application_service::UserApplicationServicePort>,
    ) -> Self {
        Self { security_service, user_service }
    }
}

// ==================== 请求/响应类型定义 ====================

/// 提交举报请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct CreateReportRequest {
    /// 举报目标类型：'user'|'moment'|'comment'|'message'|'group'
    pub target_type: String,
    /// 举报目标ID
    pub target_id: String,
    /// 举报原因：'spam'|'harassment'|'inappropriate_content'|'fake_account'|'violence'|'other'
    pub reason: String,
    /// 详细描述
    pub description: Option<String>,
}

/// 提交举报响应
#[derive(Debug, Serialize, ToSchema)]
pub struct CreateReportResponse {
    /// 举报记录ID
    pub report_id: String,
    /// 响应消息
    pub message: String,
    /// 创建时间（ISO 8601）
    pub created_at: String,
}

/// 我的举报记录请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct GetMyReportsRequest {
    /// 返回数量限制，默认20
    pub limit: Option<u32>,
    /// 分页游标
    pub cursor: Option<String>,
}

/// 举报列表请求（管理员）
#[derive(Debug, Deserialize, ToSchema)]
pub struct GetReportsRequest {
    /// 状态过滤：'pending'|'reviewing'|'resolved'|'dismissed'
    pub status: Option<String>,
    /// 目标类型过滤
    pub target_type: Option<String>,
    /// 返回数量限制，默认20
    pub limit: Option<u32>,
    /// 分页游标
    pub cursor: Option<String>,
}

/// 举报通用响应类型（用于查询类API）
#[derive(Debug, Serialize, ToSchema)]
pub struct ReportResponse {
    /// 举报记录ID
    pub report_id: String,
    /// 举报者ID
    pub reporter_id: String,
    /// 举报目标类型
    pub target_type: String,
    /// 举报目标ID
    pub target_id: String,
    /// 举报原因
    pub reason: String,
    /// 详细描述
    pub description: Option<String>,
    /// 处理状态：'pending'|'reviewing'|'resolved'|'dismissed'
    pub status: String,
    /// 创建时间（ISO 8601）
    pub created_at: String,
    /// 处理时间（ISO 8601）
    pub handled_at: Option<String>,
    /// 处理备注
    pub handler_note: Option<String>,
}

/// 处理举报请求（管理员）
#[derive(Debug, Deserialize, ToSchema)]
pub struct HandleReportRequest {
    /// 举报记录ID
    pub report_id: String,
    /// 处理动作：'resolve'|'dismiss'|'escalate'
    pub action: String,
    /// 处理备注
    pub note: Option<String>,
}

/// 处理举报响应（管理员）
#[derive(Debug, Serialize, ToSchema)]
pub struct HandleReportResponse {
    /// 是否处理成功
    pub success: bool,
    /// 响应消息
    pub message: String,
    /// 处理时间（ISO 8601）
    pub handled_at: String,
}

/// 屏蔽用户请求
#[derive(Debug, Deserialize, ToSchema)]
pub struct BlockUserRequest {
    /// 要屏蔽的用户ID
    pub user_id: String,
    /// 屏蔽原因
    pub reason: Option<String>,
}

/// 屏蔽用户响应
#[derive(Debug, Serialize, ToSchema)]
pub struct BlockUserResponse {
    /// 是否屏蔽成功
    pub success: bool,
    /// 响应消息
    pub message: String,
}

// ==================== 控制器实现 ====================

impl SecurityController {
    /// POST /api/v1/protected/reports/create - 提交举报
    pub async fn create_report(
        State(controller): State<Self>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<CreateReportRequest>,
    ) -> axum::response::Response {
        debug!(
            "用户 {} 提交举报：类型={}, 目标={}, 原因={}",
            user.user_id, request.target_type, request.target_id, request.reason
        );

        // 解析举报目标类型
        let target_type = match request.target_type.as_str() {
            "user" => ReportTargetType::User,
            "moment" => ReportTargetType::Moment,
            "comment" => ReportTargetType::Comment,
            "message" => ReportTargetType::Message,
            "group" => ReportTargetType::Group,
            _ => {
                let error_response = ApiResponse {
                    code: 400,
                    msg: "无效的举报目标类型".to_string(),
                    data: None::<()>,
                };
                return (StatusCode::OK, Json(error_response)).into_response();
            }
        };

        // 解析举报原因
        let reason = match request.reason.as_str() {
            "spam" => ReportReason::Spam,
            "harassment" => ReportReason::Harassment,
            "inappropriate_content" => ReportReason::InappropriateContent,
            "fake_account" => ReportReason::FakeAccount,
            "violence" => ReportReason::Violence,
            "other" => ReportReason::Other,
            _ => {
                let error_response = ApiResponse {
                    code: 400,
                    msg: "无效的举报原因".to_string(),
                    data: None::<()>,
                };
                return (StatusCode::OK, Json(error_response)).into_response();
            }
        };

        // 调用安全应用服务创建举报
        match controller.security_service.create_report(
            user.user_id,
            target_type,
            request.target_id,
            reason,
            request.description,
        ).await {
            Ok(report) => {
                let response = CreateReportResponse {
                    report_id: report.report_id,
                    message: "举报提交成功，我们将尽快处理".to_string(),
                    created_at: report.created_at.to_rfc3339(),
                };

                let api_response = ApiResponse {
                    code: 0,
                    msg: "举报提交成功".to_string(),
                    data: Some(response),
                };

                (StatusCode::OK, Json(api_response)).into_response()
            }
            Err(err) => {
                let (status_code, error_msg) = match err {
                    crate::domain::DomainError::ValidationError { reason, .. } => (StatusCode::BAD_REQUEST, reason),
                    crate::domain::DomainError::NotFound { resource } => (StatusCode::NOT_FOUND, format!("{}不存在", resource)),
                    _ => (StatusCode::INTERNAL_SERVER_ERROR, "服务器内部错误".to_string()),
                };

                let error_response = ApiResponse {
                    code: status_code.as_u16() as i32,
                    msg: error_msg,
                    data: None::<()>,
                };

                (StatusCode::OK, Json(error_response)).into_response()
            }
        }
    }

    /// POST /api/v1/protected/reports/my_reports - 我的举报记录
    pub async fn get_my_reports(
        State(_controller): State<Self>,
        Extension(user): Extension<CurrentUser>,
        Json(_request): Json<GetMyReportsRequest>,
    ) -> axum::response::Response {
        debug!("用户 {} 获取举报记录", user.user_id);

        // TODO: 实现获取用户举报记录逻辑
        // 1. 查询用户的举报记录
        // 2. 分页处理
        // 3. 返回举报状态和处理结果

        let reports = vec![
            ReportResponse {
                report_id: "example_report_id".to_string(),
                reporter_id: user.user_id.to_string(),
                target_type: "user".to_string(),
                target_id: "example_target_id".to_string(),
                reason: "spam".to_string(),
                description: Some("发送垃圾信息".to_string()),
                status: "pending".to_string(),
                created_at: chrono::Utc::now().to_rfc3339(),
                handled_at: None,
                handler_note: None,
            }
        ];

        let paginated_response = crate::presentation::http::models::PaginatedResponse {
            items: reports,
            has_more: false,
            next_cursor: None,
        };

        let api_response = ApiResponse {
            code: 0,
            msg: "获取举报记录成功".to_string(),
            data: Some(paginated_response),
        };

        (StatusCode::OK, Json(api_response)).into_response()
    }

    /// POST /api/v1/protected/reports/list - 举报列表（管理员）
    pub async fn get_reports(
        State(_controller): State<Self>,
        Extension(user): Extension<CurrentUser>,
        Json(_request): Json<GetReportsRequest>,
    ) -> axum::response::Response {
        debug!("管理员 {} 获取举报列表", user.user_id);

        // TODO: 验证管理员权限
        // TODO: 实现获取举报列表逻辑
        // 1. 验证用户是管理员
        // 2. 根据状态和类型过滤
        // 3. 分页查询举报记录
        // 4. 返回详细的举报信息

        let reports = vec![
            ReportResponse {
                report_id: "admin_report_id".to_string(),
                reporter_id: "reporter_user_id".to_string(),
                target_type: "moment".to_string(),
                target_id: "reported_moment_id".to_string(),
                reason: "inappropriate_content".to_string(),
                description: Some("包含不当内容".to_string()),
                status: "pending".to_string(),
                created_at: chrono::Utc::now().to_rfc3339(),
                handled_at: None,
                handler_note: None,
            }
        ];

        let paginated_response = crate::presentation::http::models::PaginatedResponse {
            items: reports,
            has_more: false,
            next_cursor: None,
        };

        let api_response = ApiResponse {
            code: 0,
            msg: "获取举报列表成功".to_string(),
            data: Some(paginated_response),
        };

        (StatusCode::OK, Json(api_response)).into_response()
    }

    /// POST /api/v1/protected/reports/handle - 处理举报（管理员）
    pub async fn handle_report(
        State(_controller): State<Self>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<HandleReportRequest>,
    ) -> axum::response::Response {
        debug!(
            "管理员 {} 处理举报 {}，动作：{}",
            user.user_id, request.report_id, request.action
        );

        // TODO: 验证管理员权限
        // TODO: 实现处理举报逻辑
        // 1. 验证用户是管理员
        // 2. 验证举报记录存在
        // 3. 根据处理动作执行相应操作
        // 4. 更新举报状态
        // 5. 记录处理日志
        // 6. 通知举报者处理结果（如需要）

        let response = HandleReportResponse {
            success: true,
            message: "举报处理完成".to_string(),
            handled_at: chrono::Utc::now().to_rfc3339(),
        };

        let api_response = ApiResponse {
            code: 0,
            msg: "举报处理成功".to_string(),
            data: Some(response),
        };

        (StatusCode::OK, Json(api_response)).into_response()
    }

    /// POST /api/v1/protected/users/block - 屏蔽用户
    pub async fn block_user(
        State(controller): State<Self>,
        Extension(user): Extension<CurrentUser>,
        Json(request): Json<BlockUserRequest>,
    ) -> axum::response::Response {
        debug!("用户 {} 屏蔽用户 {}", user.user_id, request.user_id);

        // 验证目标用户存在
        match controller.security_service.get_user_profile(request.user_id.parse::<i64>().unwrap_or(0)).await {
            Ok(target_user) => {
                let _target_user = match target_user {
                    Some(user) => user,
                    None => {
                        let api_response = ApiResponse {
                            code: 404,
                            msg: "目标用户不存在".to_string(),
                            data: None::<()>,
                        };
                        return (StatusCode::OK, Json(api_response)).into_response();
                    }
                };

                // 调用安全服务屏蔽用户
                match controller.security_service.block_user(user.user_id, request.user_id.parse::<i64>().unwrap_or(0), request.reason.clone()).await {
                    Ok(_) => {
                        let response = BlockUserResponse {
                            success: true,
                            message: "用户已屏蔽".to_string(),
                        };

                        let api_response = ApiResponse {
                            code: 0,
                            msg: "用户屏蔽成功".to_string(),
                            data: Some(response),
                        };

                        (StatusCode::OK, Json(api_response)).into_response()
                    }
                    Err(err) => crate::presentation::http::controllers::map_domain_error_to_http_response(err),
                }
            }
            Err(err) => crate::presentation::http::controllers::map_domain_error_to_http_response(err),
        }
    }
}
