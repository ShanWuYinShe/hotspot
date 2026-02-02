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

    // Database Pool 配置
    pub db_pool_max_connections: u32,
    pub db_pool_min_connections: u32,
    pub db_pool_acquire_timeout_secs: u64,
    pub db_pool_idle_timeout_secs: u64,
    pub db_pool_max_lifetime_secs: u64,
    pub db_pool_test_before_acquire: bool,

    // Rate Limit 配置
    pub rate_limit_ip_per_hour: u32, // 单IP限流（防止共享IP滥用）
    pub rate_limit_user_per_5minutes: u32, // 单用户短期限流（防突发攻击）
    pub rate_limit_user_per_hour: u32, // 单用户长期限流（补充防护）
    /// Redis 故障时的限流降级策略
    /// - true: fail-open（允许请求通过，优先保证可用性）
    /// - false: fail-closed（拒绝请求，优先保证安全性）
    pub rate_limit_fail_open: bool,
    // 端点级限速配置
    pub rate_limit_send_code_per_minute: u32,
    pub rate_limit_verify_code_per_10minutes: u32,
    pub rate_limit_register_per_10minutes: u32,
    pub rate_limit_login_per_10minutes: u32,
    pub rate_limit_guest_per_10minutes: u32,
    pub rate_limit_refresh_per_10minutes: u32,
    pub rate_limit_reset_request_per_minute: u32,
    pub rate_limit_reset_verify_per_10minutes: u32,
    pub rate_limit_reset_per_10minutes: u32,
    pub rate_limit_moments_create_per_minute: u32,
    pub rate_limit_comments_create_per_minute: u32,
    pub rate_limit_reports_create_per_hour: u32,
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
                .unwrap_or_else(|_| "5496".to_string())
                .parse()
                .map_err(|_| "Invalid SERVER_PORT".to_string())?,
            _max_upload_size: env::var("MAX_UPLOAD_SIZE")
                .unwrap_or_else(|_| "10485760".to_string())
                .parse()
                .map_err(|_| "Invalid MAX_UPLOAD_SIZE".to_string())?,

            // Database Pool 配置（分布式环境推荐值：假设 PostgreSQL max_connections=100, 3个实例）
            db_pool_max_connections: env::var("DB_POOL_MAX_CONNECTIONS")
                .unwrap_or_else(|_| "20".to_string())
                .parse()
                .map_err(|_| "Invalid DB_POOL_MAX_CONNECTIONS".to_string())?,
            db_pool_min_connections: env::var("DB_POOL_MIN_CONNECTIONS")
                .unwrap_or_else(|_| "5".to_string())
                .parse()
                .map_err(|_| "Invalid DB_POOL_MIN_CONNECTIONS".to_string())?,
            db_pool_acquire_timeout_secs: env::var("DB_POOL_ACQUIRE_TIMEOUT_SECS")
                .unwrap_or_else(|_| "10".to_string())
                .parse()
                .map_err(|_| "Invalid DB_POOL_ACQUIRE_TIMEOUT_SECS".to_string())?,
            db_pool_idle_timeout_secs: env::var("DB_POOL_IDLE_TIMEOUT_SECS")
                .unwrap_or_else(|_| "600".to_string())
                .parse()
                .map_err(|_| "Invalid DB_POOL_IDLE_TIMEOUT_SECS".to_string())?,
            db_pool_max_lifetime_secs: env::var("DB_POOL_MAX_LIFETIME_SECS")
                .unwrap_or_else(|_| "1800".to_string())
                .parse()
                .map_err(|_| "Invalid DB_POOL_MAX_LIFETIME_SECS".to_string())?,
            db_pool_test_before_acquire: env::var("DB_POOL_TEST_BEFORE_ACQUIRE")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .map_err(|_| "Invalid DB_POOL_TEST_BEFORE_ACQUIRE".to_string())?,

            rate_limit_ip_per_hour: env::var("RATE_LIMIT_IP_PER_HOUR")
                .unwrap_or_else(|_| "20000".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_IP_PER_HOUR".to_string())?,
            rate_limit_user_per_5minutes: env::var("RATE_LIMIT_USER_PER_5MINUTES")
                .unwrap_or_else(|_| "500".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_USER_PER_5MINUTES".to_string())?,
            rate_limit_user_per_hour: env::var("RATE_LIMIT_USER_PER_HOUR")
                .unwrap_or_else(|_| "5000".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_USER_PER_HOUR".to_string())?,
            rate_limit_fail_open: env::var("RATE_LIMIT_FAIL_OPEN")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_FAIL_OPEN".to_string())?,
            rate_limit_send_code_per_minute: env::var("RATE_LIMIT_SEND_CODE_PER_MINUTE")
                .unwrap_or_else(|_| "5".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_SEND_CODE_PER_MINUTE".to_string())?,
            rate_limit_verify_code_per_10minutes: env::var("RATE_LIMIT_VERIFY_CODE_PER_10MINUTES")
                .unwrap_or_else(|_| "30".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_VERIFY_CODE_PER_10MINUTES".to_string())?,
            rate_limit_register_per_10minutes: env::var("RATE_LIMIT_REGISTER_PER_10MINUTES")
                .unwrap_or_else(|_| "10".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_REGISTER_PER_10MINUTES".to_string())?,
            rate_limit_login_per_10minutes: env::var("RATE_LIMIT_LOGIN_PER_10MINUTES")
                .unwrap_or_else(|_| "30".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_LOGIN_PER_10MINUTES".to_string())?,
            rate_limit_guest_per_10minutes: env::var("RATE_LIMIT_GUEST_PER_10MINUTES")
                .unwrap_or_else(|_| "60".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_GUEST_PER_10MINUTES".to_string())?,
            rate_limit_refresh_per_10minutes: env::var("RATE_LIMIT_REFRESH_PER_10MINUTES")
                .unwrap_or_else(|_| "120".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_REFRESH_PER_10MINUTES".to_string())?,
            rate_limit_reset_request_per_minute: env::var("RATE_LIMIT_RESET_REQUEST_PER_MINUTE")
                .unwrap_or_else(|_| "5".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_RESET_REQUEST_PER_MINUTE".to_string())?,
            rate_limit_reset_verify_per_10minutes: env::var(
                "RATE_LIMIT_RESET_VERIFY_PER_10MINUTES",
            )
            .unwrap_or_else(|_| "20".to_string())
            .parse()
            .map_err(|_| "Invalid RATE_LIMIT_RESET_VERIFY_PER_10MINUTES".to_string())?,
            rate_limit_reset_per_10minutes: env::var("RATE_LIMIT_RESET_PER_10MINUTES")
                .unwrap_or_else(|_| "10".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_RESET_PER_10MINUTES".to_string())?,
            rate_limit_moments_create_per_minute: env::var("RATE_LIMIT_MOMENTS_CREATE_PER_MINUTE")
                .unwrap_or_else(|_| "20".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_MOMENTS_CREATE_PER_MINUTE".to_string())?,
            rate_limit_comments_create_per_minute: env::var(
                "RATE_LIMIT_COMMENTS_CREATE_PER_MINUTE",
            )
            .unwrap_or_else(|_| "60".to_string())
            .parse()
            .map_err(|_| "Invalid RATE_LIMIT_COMMENTS_CREATE_PER_MINUTE".to_string())?,
            rate_limit_reports_create_per_hour: env::var("RATE_LIMIT_REPORTS_CREATE_PER_HOUR")
                .unwrap_or_else(|_| "10".to_string())
                .parse()
                .map_err(|_| "Invalid RATE_LIMIT_REPORTS_CREATE_PER_HOUR".to_string())?,
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
