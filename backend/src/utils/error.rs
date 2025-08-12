// 更结构化的错误处理模块

use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde::{Serialize, Deserialize};
use thiserror::Error;
use tracing::{error, warn};

use crate::utils::response::ApiResponse;

// 使用thiserror简化错误定义
#[derive(Error, Debug)]
pub enum AppError {
    #[error("资源不存在: {0}")]
    NotFound(String),
    
    #[error("请求参数错误: {0}")]
    BadRequest(String),
    
    #[error("服务器内部错误: {0}")]
    InternalServerError(String),
    
    #[error("未授权: {0}")]
    Unauthorized(String),
    
    #[error("权限不足: {0}")]
    Forbidden(String),
    
    #[error("资源冲突: {0}")]
    Conflict(String),
    
    #[error("数据库错误: {0}")]
    DatabaseError(String),
    
    #[error("第三方服务错误: {0}")]
    ExternalServiceError(String),
    
    #[error("功能未实现: {0}")]
    #[allow(dead_code)]
    NotImplemented(String),
}

// 将具体的错误映射到AppError
impl From<sqlx::Error> for AppError {
    fn from(err: sqlx::Error) -> Self {
        error!("数据库错误: {:?}", err);
        match err {
            sqlx::Error::RowNotFound => Self::NotFound("未找到请求的资源".into()),
            _ => Self::DatabaseError(format!("数据库操作失败: {}", err)),
        }
    }
}

impl From<redis::RedisError> for AppError {
    fn from(err: redis::RedisError) -> Self {
        error!("Redis错误: {:?}", err);
        Self::ExternalServiceError(format!("Redis操作失败: {}", err))
    }
}

impl From<std::io::Error> for AppError {
    fn from(err: std::io::Error) -> Self {
        error!("IO错误: {:?}", err);
        Self::InternalServerError(format!("IO操作失败: {}", err))
    }
}

impl From<crate::repositories::RepositoryError> for AppError {
    fn from(err: crate::repositories::RepositoryError) -> Self {
        error!("仓库错误: {:?}", err);
        match err {
            crate::repositories::RepositoryError::NotFound => Self::NotFound("请求的资源不存在".into()),
            crate::repositories::RepositoryError::Database(e) => Self::DatabaseError(format!("数据库操作失败: {}", e)),
            crate::repositories::RepositoryError::InvalidArgument(msg) => Self::BadRequest(msg),
            crate::repositories::RepositoryError::UuidParse(err) => Self::BadRequest(format!("UUID格式错误: {}", err)),
            crate::repositories::RepositoryError::Other(msg) => Self::InternalServerError(msg),
        }
    }
}

// 用于在错误响应中返回的结构
#[derive(Debug, Serialize, Deserialize)]
pub struct ErrorDetail {
    pub code: i32,
    pub message: String,
    pub request_id: Option<String>,
    pub details: Option<Vec<String>>,
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        // 为每个错误记录追踪信息
        let (status_code, error_code, error_message) = match &self {
            AppError::NotFound(message) => {
                warn!("资源不存在: {}", message);
                (StatusCode::OK, 1004, message.clone())
            },
            AppError::BadRequest(message) => {
                warn!("请求参数错误: {}", message);
                (StatusCode::OK, 1000, message.clone())
            },
            AppError::InternalServerError(message) => {
                error!("服务器内部错误: {}", message);
                (StatusCode::OK, 5000, message.clone())
            },
            AppError::Unauthorized(message) => {
                warn!("未授权: {}", message);
                (StatusCode::OK, 1002, message.clone())
            },
            AppError::Forbidden(message) => {
                warn!("权限不足: {}", message);
                (StatusCode::OK, 1003, message.clone())
            },
            AppError::Conflict(message) => {
                warn!("资源冲突: {}", message);
                (StatusCode::OK, 1005, message.clone())
            },
            AppError::DatabaseError(message) => {
                error!("数据库错误: {}", message);
                (StatusCode::OK, 5001, "数据库操作失败".into())
            },
            AppError::ExternalServiceError(message) => {
                error!("第三方服务错误: {}", message);
                (StatusCode::OK, 5002, "外部服务调用失败".into())
            },
            AppError::NotImplemented(message) => {
                warn!("功能未实现: {}", message);
                (StatusCode::OK, 1006, message.clone())
            },
        };

        // 创建结构化的错误响应
        let body = Json(ApiResponse::<()> {
            code: error_code,
            msg: error_message,
            data: None,
        });

        // 统一返回HTTP状态码200，将实际错误码放在body中
        (status_code, body).into_response()
    }
}
