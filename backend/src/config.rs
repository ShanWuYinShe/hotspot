//! 应用配置 — 从环境变量加载

use std::env;

/// 从环境变量读取字符串，缺失时使用默认值
fn env_or(key: &str, default: &str) -> String {
    env::var(key).unwrap_or_else(|_| default.to_string())
}

/// 从环境变量读取并解析为目标类型，缺失时使用默认值
fn env_parse<T: std::str::FromStr>(key: &str, default: &str) -> Result<T, String> {
    env_or(key, default)
        .parse()
        .map_err(|_| format!("Invalid {key}"))
}

/// 从环境变量读取并解析，缺失时返回错误
fn env_require<T: std::str::FromStr>(key: &str) -> Result<T, String> {
    env::var(key)
        .map_err(|_| format!("{key} must be set"))?
        .parse()
        .map_err(|_| format!("Invalid {key}"))
}

#[derive(Debug, Clone)]
pub struct Config {
    pub database_url: String,
    pub redis_url: String,
    pub jwt_secret: String,
    pub jwt_access_token_expiration: i64,
    pub jwt_refresh_token_expiration: i64,
    pub server_host: String,
    pub server_port: u16,
    /// Cookie Secure 标志（生产环境应为 true）
    pub cookie_secure: bool,

    // Database Pool
    pub db_pool_max_connections: u32,
    pub db_pool_min_connections: u32,
    pub db_pool_acquire_timeout_secs: u64,
    pub db_pool_idle_timeout_secs: u64,
    pub db_pool_max_lifetime_secs: u64,
    pub db_pool_test_before_acquire: bool,

    // Rate Limit — 全局
    pub rate_limit_global_per_hour: u32,
    pub rate_limit_user_per_5minutes: u32,
    pub rate_limit_user_per_hour: u32,
    /// Redis 故障时: true = fail-open（放行）, false = fail-closed（拒绝）
    pub rate_limit_fail_open: bool,
    // Rate Limit — 端点级
    pub rate_limit_register_per_10minutes: u32,
    pub rate_limit_login_per_10minutes: u32,
    pub rate_limit_auth_reset_per_10minutes: u32,
    pub rate_limit_guest_per_10minutes: u32,
    pub rate_limit_refresh_per_10minutes: u32,
    pub rate_limit_moments_create_per_minute: u32,
    pub rate_limit_comments_create_per_minute: u32,
    pub rate_limit_reports_create_per_hour: u32,

    /// 超级管理员 bootstrap
    pub super_admin_login_id: Option<String>,
    pub super_admin_password: Option<String>,

    // S3 / MinIO
    pub s3_endpoint: String,
    pub s3_region: String,
    pub s3_bucket: String,
    pub s3_access_key: String,
    pub s3_secret_key: String,
    pub s3_force_path_style: bool,
    /// S3 对外 URL 前缀（相对路径 /s3 用于本地 MinIO；完整 URL 用于生产）
    pub s3_public_base: String,

    /// 可选：密码重置验证码外部投递 Webhook（邮件/短信网关）
    pub auth_reset_delivery_webhook_url: Option<String>,
    /// 可选：投递 Webhook 的 Bearer Token
    pub auth_reset_delivery_webhook_token: Option<String>,
    /// 投递 Webhook 超时（毫秒）
    pub auth_reset_delivery_timeout_ms: u64,
}

impl Config {
    pub fn from_env() -> Result<Self, String> {
        dotenvy::dotenv().ok();

        Ok(Config {
            database_url: env_require("DATABASE_URL")?,
            redis_url: env_or("REDIS_URL", "redis://localhost:6379"),
            jwt_secret: env_require("JWT_SECRET")?,
            jwt_access_token_expiration: env_parse("JWT_ACCESS_TOKEN_EXPIRATION", "900")?,
            jwt_refresh_token_expiration: env_parse("JWT_REFRESH_TOKEN_EXPIRATION", "2592000")?,
            server_host: env_or("SERVER_HOST", "0.0.0.0"),
            server_port: env_parse("SERVER_PORT", "5496")?,
            cookie_secure: env_parse("COOKIE_SECURE", "false")?,

            db_pool_max_connections: env_parse("DB_POOL_MAX_CONNECTIONS", "20")?,
            db_pool_min_connections: env_parse("DB_POOL_MIN_CONNECTIONS", "5")?,
            db_pool_acquire_timeout_secs: env_parse("DB_POOL_ACQUIRE_TIMEOUT_SECS", "10")?,
            db_pool_idle_timeout_secs: env_parse("DB_POOL_IDLE_TIMEOUT_SECS", "600")?,
            db_pool_max_lifetime_secs: env_parse("DB_POOL_MAX_LIFETIME_SECS", "1800")?,
            db_pool_test_before_acquire: env_parse("DB_POOL_TEST_BEFORE_ACQUIRE", "true")?,

            rate_limit_global_per_hour: env_parse("RATE_LIMIT_GLOBAL_PER_HOUR", "20000")?,
            rate_limit_user_per_5minutes: env_parse("RATE_LIMIT_USER_PER_5MINUTES", "500")?,
            rate_limit_user_per_hour: env_parse("RATE_LIMIT_USER_PER_HOUR", "5000")?,
            rate_limit_fail_open: env_parse("RATE_LIMIT_FAIL_OPEN", "true")?,
            rate_limit_register_per_10minutes: env_parse(
                "RATE_LIMIT_REGISTER_PER_10MINUTES",
                "10",
            )?,
            rate_limit_login_per_10minutes: env_parse("RATE_LIMIT_LOGIN_PER_10MINUTES", "30")?,
            rate_limit_auth_reset_per_10minutes: env_parse(
                "RATE_LIMIT_AUTH_RESET_PER_10MINUTES",
                "10",
            )?,
            rate_limit_guest_per_10minutes: env_parse("RATE_LIMIT_GUEST_PER_10MINUTES", "60")?,
            rate_limit_refresh_per_10minutes: env_parse("RATE_LIMIT_REFRESH_PER_10MINUTES", "120")?,
            rate_limit_moments_create_per_minute: env_parse(
                "RATE_LIMIT_MOMENTS_CREATE_PER_MINUTE",
                "20",
            )?,
            rate_limit_comments_create_per_minute: env_parse(
                "RATE_LIMIT_COMMENTS_CREATE_PER_MINUTE",
                "60",
            )?,
            rate_limit_reports_create_per_hour: env_parse(
                "RATE_LIMIT_REPORTS_CREATE_PER_HOUR",
                "10",
            )?,

            super_admin_login_id: env::var("SUPER_ADMIN_LOGIN_ID").ok(),
            super_admin_password: env::var("SUPER_ADMIN_PASSWORD").ok(),

            s3_endpoint: env_or("S3_ENDPOINT", "http://localhost:9000"),
            s3_region: env_or("S3_REGION", "us-east-1"),
            s3_bucket: env_or("S3_BUCKET", "hotspot"),
            s3_access_key: env_require("S3_ACCESS_KEY")?,
            s3_secret_key: env_require("S3_SECRET_KEY")?,
            s3_force_path_style: env_parse("S3_FORCE_PATH_STYLE", "true")?,
            s3_public_base: env_require("S3_PUBLIC_BASE")?,
            auth_reset_delivery_webhook_url: env::var("AUTH_RESET_DELIVERY_WEBHOOK_URL")
                .ok()
                .filter(|value| !value.trim().is_empty()),
            auth_reset_delivery_webhook_token: env::var("AUTH_RESET_DELIVERY_WEBHOOK_TOKEN")
                .ok()
                .filter(|value| !value.trim().is_empty()),
            auth_reset_delivery_timeout_ms: env_parse("AUTH_RESET_DELIVERY_TIMEOUT_MS", "3000")?,
        })
    }

    pub fn server_addr(&self) -> String {
        format!("{}:{}", self.server_host, self.server_port)
    }
}
