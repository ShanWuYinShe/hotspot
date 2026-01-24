//! 应用配置
//!
//! 部分配置项暂未使用，但已按规范定义

use std::env;

#[derive(Debug, Clone)]
pub struct Config {
    pub database_url: String,
    pub redis_url: String,
    pub jwt_secret: String,
    pub jwt_access_token_expiration: i64,
    pub jwt_refresh_token_expiration: i64,
    pub server_host: String,
    pub server_port: u16,
    pub _max_upload_size: usize,
    pub rate_limit_global_per_hour: u32,
    pub rate_limit_user_per_hour: u32,
    /// 超级管理员 bootstrap：登录ID（可选，默认 superadmin）
    pub super_admin_login_id: Option<String>,
    /// 超级管理员 bootstrap：初始密码（建议必须设置；未设置则不会自动创建）
    pub super_admin_password: Option<String>,

    // S3 Configuration (统一的 S3 endpoint，MinIO 模拟 AWS S3)
    pub s3_endpoint: String,
    pub s3_region: String,
    pub s3_bucket: String,
    pub s3_access_key: String,
    pub s3_secret_key: String,
    /// 是否强制 path-style（MinIO 建议 true，AWS S3 通常为 false）
    pub s3_force_path_style: bool,
    /// S3 对外 URL 前缀（用于生成 presigned URL 和公开访问链接）
    /// 必须配置，确保生成的 URL 在客户端可访问
    /// - 相对路径（如 /s3）：用于本地 MinIO，隐藏 bucket 名，通过 Nginx 转发
    /// - 完整 URL（如 https://cdn.example.com）：用于生产环境
    pub s3_public_base: String,
}

impl Config {
    pub fn from_env() -> Result<Self, String> {
        dotenvy::dotenv().ok();

        Ok(Config {
            database_url: env::var("DATABASE_URL")
                .map_err(|_| "DATABASE_URL must be set".to_string())?,
            redis_url: env::var("REDIS_URL")
                .unwrap_or_else(|_| "redis://localhost:6379".to_string()),
            jwt_secret: env::var("JWT_SECRET").map_err(|_| "JWT_SECRET must be set".to_string())?,
            jwt_access_token_expiration: env::var("JWT_ACCESS_TOKEN_EXPIRATION")
                .unwrap_or_else(|_| "900".to_string()) // 15分钟
                .parse()
                .map_err(|_| "Invalid JWT_ACCESS_TOKEN_EXPIRATION".to_string())?,
            jwt_refresh_token_expiration: env::var("JWT_REFRESH_TOKEN_EXPIRATION")
                .unwrap_or_else(|_| "2592000".to_string())
                .parse()
                .map_err(|_| "Invalid JWT_REFRESH_TOKEN_EXPIRATION".to_string())?,
            server_host: env::var("SERVER_HOST").unwrap_or_else(|_| "0.0.0.0".to_string()),
            server_port: env::var("SERVER_PORT")
                .unwrap_or_else(|_| "8080".to_string())
                .parse()
                .map_err(|_| "Invalid SERVER_PORT".to_string())?,
            _max_upload_size: env::var("MAX_UPLOAD_SIZE")
                .unwrap_or_else(|_| "10485760".to_string())
                .parse()
                .map_err(|_| "Invalid MAX_UPLOAD_SIZE".to_string())?,
            rate_limit_global_per_hour: env::var("RATE_LIMIT_GLOBAL_PER_HOUR")
                .unwrap_or_else(|_| "1000".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_GLOBAL_PER_HOUR".to_string())?,
            rate_limit_user_per_hour: env::var("RATE_LIMIT_USER_PER_HOUR")
                .unwrap_or_else(|_| "5000".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_USER_PER_HOUR".to_string())?,
            super_admin_login_id: env::var("SUPER_ADMIN_LOGIN_ID").ok(),
            super_admin_password: env::var("SUPER_ADMIN_PASSWORD").ok(),

            // S3 Defaults (统一的 S3 endpoint，MinIO 模拟 AWS S3)
            s3_endpoint: env::var("S3_ENDPOINT")
                .unwrap_or_else(|_| "http://localhost:9000".to_string()),
            s3_region: env::var("S3_REGION").unwrap_or_else(|_| "us-east-1".to_string()),
            s3_bucket: env::var("S3_BUCKET").unwrap_or_else(|_| "hotspot".to_string()),
            // 敏感信息强制要求配置，不使用默认值
            s3_access_key: env::var("S3_ACCESS_KEY")
                .map_err(|_| "S3_ACCESS_KEY must be set".to_string())?,
            s3_secret_key: env::var("S3_SECRET_KEY")
                .map_err(|_| "S3_SECRET_KEY must be set".to_string())?,
            s3_force_path_style: env::var("S3_FORCE_PATH_STYLE")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .map_err(|_| "Invalid S3_FORCE_PATH_STYLE".to_string())?,
            s3_public_base: env::var("S3_PUBLIC_BASE")
                .map_err(|_| "S3_PUBLIC_BASE must be set".to_string())?,
        })
    }

    pub fn server_addr(&self) -> String {
        format!("{}:{}", self.server_host, self.server_port)
    }
}
