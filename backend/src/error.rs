use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde_json::json;
use thiserror::Error;

#[derive(Debug, Error)]
pub enum AppError {
    #[error("参数错误: {0}")]
    BadRequest(String),

    #[error("认证失败: {0}")]
    Unauthorized(String),

    #[error("权限不足")]
    Forbidden,

    #[error("资源不存在")]
    NotFound,

    #[error("操作冲突: {0}")]
    Conflict(String),

    #[error("数据库错误: {0}")]
    Database(#[from] sqlx::Error),

    #[error("Redis错误: {0}")]
    Redis(#[from] redis::RedisError),

    #[error("JWT错误: {0}")]
    Jwt(#[from] jsonwebtoken::errors::Error),

    #[error("密码哈希错误")]
    BcryptError(#[from] bcrypt::BcryptError),

    #[error("序列化错误: {0}")]
    Serialization(#[from] serde_json::Error),

    #[error("内部服务器错误")]
    Internal,
}

impl AppError {
    fn error_code(&self) -> i32 {
        match self {
            Self::BadRequest(_) => 1000,
            Self::Unauthorized(_) => 1002,
            Self::Forbidden => 1003,
            Self::NotFound => 1004,
            Self::Conflict(_) => 1005,
            Self::Database(_) => 5001,
            Self::Redis(_) => 5001,
            Self::Jwt(_) => 1002,
            Self::BcryptError(_) => 5000,
            Self::Serialization(_) => 5000,
            Self::Internal => 5000,
        }
    }
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let status = StatusCode::OK; // 统一返回 200
        let code = self.error_code();
        let message = self.to_string();

        // 记录错误日志
        if code >= 5000 {
            tracing::error!(error = %self, "Internal server error");
        } else {
            tracing::warn!(error = %self, "Client error");
        }

        let body = Json(json!({
            "code": code,
            "message": message,
            "data": null,
        }));

        (status, body).into_response()
    }
}

pub type Result<T> = std::result::Result<T, AppError>;

