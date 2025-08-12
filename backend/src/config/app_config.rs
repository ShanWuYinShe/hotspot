use std::env;
use std::path::Path;
use std::time::Duration;
use tracing::{info, warn};

/// 配置默认值常量
mod config_defaults {
    use std::time::Duration;
    
    // 数据库默认配置
    pub const DEFAULT_DB_MAX_CONNECTIONS: u32 = 10;
    pub const DEFAULT_DB_MIN_CONNECTIONS: u32 = 1;
    pub const DEFAULT_DB_MAX_LIFETIME_SECS: u64 = 1800; // 30分钟
    pub const DEFAULT_DB_IDLE_TIMEOUT_SECS: u64 = 600;  // 10分钟
    pub const DEFAULT_DB_CONNECT_TIMEOUT_SECS: u64 = 10;
    
    // WebSocket清理默认配置
    pub const DEFAULT_WS_CLEANUP_INTERVAL_SECS: u64 = 600; // 10分钟
    pub const DEFAULT_WS_INACTIVE_TIMEOUT_SECS: u64 = 1800; // 30分钟
    
    // JWT默认配置
    pub const DEFAULT_JWT_ACCESS_TOKEN_EXPIRY_SECS: u64 = 3600;    // 1小时
    pub const DEFAULT_JWT_REFRESH_TOKEN_EXPIRY_SECS: u64 = 2592000; // 30天
    
    // 服务器默认配置
    pub const DEFAULT_HOST: &str = "0.0.0.0";
    pub const DEFAULT_PORT: u16 = 8080;
    #[allow(dead_code)]
    pub const DEFAULT_CORS_ENABLED: bool = true;
    
    // 环境默认值
    #[allow(dead_code)]
    pub const DEFAULT_ENVIRONMENT: &str = "development";
    
    #[allow(dead_code)]
    pub fn default_db_max_lifetime() -> Duration {
        Duration::from_secs(DEFAULT_DB_MAX_LIFETIME_SECS)
    }
    
    #[allow(dead_code)]
    pub fn default_db_idle_timeout() -> Duration {
        Duration::from_secs(DEFAULT_DB_IDLE_TIMEOUT_SECS)
    }
    
    #[allow(dead_code)]
    pub fn default_db_connect_timeout() -> Duration {
        Duration::from_secs(DEFAULT_DB_CONNECT_TIMEOUT_SECS)
    }
}

/// 环境枚举
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Environment {
    Development,
    Production,
}

impl Environment {
    pub fn from_string(s: &str) -> Self {
        match s.to_lowercase().as_str() {
            "production" | "prod" => Self::Production,
            _ => Self::Development,
        }
    }
    
    pub fn is_dev(&self) -> bool {
        matches!(self, Self::Development)
    }
    
    pub fn is_prod(&self) -> bool {
        matches!(self, Self::Production)
    }
}

/// 数据库配置
#[derive(Debug, Clone)]
pub struct DatabaseConfig {
    pub url: String,
    pub max_connections: u32,
    pub min_connections: u32,
    pub max_lifetime: Duration,
    pub idle_timeout: Duration,
    pub connect_timeout: Duration,
}

/// 缓存配置
#[derive(Debug, Clone)]
pub struct CacheConfig {
    pub redis_url: Option<String>,
}

/// 认证配置
#[derive(Debug, Clone)]
pub struct AuthConfig {
    pub jwt_secret: String,
    pub jwt_expires_in: Duration,
    pub refresh_token_expires_in: Duration,
}

/// 服务器配置
#[derive(Debug, Clone)]
pub struct ServerConfig {
    pub host: String,
    pub port: u16,
}

/// WebSocket 配置
#[derive(Debug, Clone)]
pub struct WebSocketConfig {
    /// 调度器清理间隔（秒）
    pub cleanup_interval_secs: u64,
    /// 判定为不活跃并清理的超时（秒）
    pub inactive_timeout_secs: u64,
}

/// 作业队列配置
#[derive(Debug, Clone)]
pub struct JobQueueConfig {
    pub amqp_url: Option<String>,
    pub enabled: bool,
}

/// CORS配置
#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct CorsConfig {
    /// 是否为开发模式CORS（完全开放）
    pub permissive_mode: bool,
    /// 允许的HTTP方法 - 只允许POST和OPTIONS
    pub allowed_methods: Vec<String>,
    /// 允许的域名
    pub allowed_origins: Vec<String>,
    /// 允许的请求头
    pub allowed_headers: Vec<String>,
}

impl Default for CorsConfig {
    fn default() -> Self {
        Self {
            permissive_mode: cfg!(debug_assertions),
            allowed_methods: vec![
                "POST".to_string(),
                "OPTIONS".to_string(), // 预检请求必须保留
            ],
            allowed_origins: vec!["*".to_string()],
            allowed_headers: vec!["*".to_string()],
        }
    }
}

/// 应用配置
#[derive(Debug, Clone)]
pub struct AppConfig {
    pub environment: Environment,
    pub database: DatabaseConfig,
    pub cache: CacheConfig,
    pub auth: AuthConfig,
    pub server: ServerConfig,
    pub job_queue: JobQueueConfig,
    pub websocket: WebSocketConfig,
}

impl AppConfig {
    /// 从环境变量加载配置
    pub fn from_env() -> Result<Self, Box<dyn std::error::Error>> {
        // 加载.env文件 (如果存在)
        if Path::new(".env").exists() {
            match dotenv::dotenv() {
                Ok(_) => info!("已加载 .env 文件"),
                Err(e) => warn!("无法加载 .env 文件: {}", e),
            }
        } else if Path::new(".env.local").exists() {
            match dotenv::from_filename(".env.local") {
                Ok(_) => info!("已加载 .env.local 文件"),
                Err(e) => warn!("无法加载 .env.local 文件: {}", e),
            }
        }
        
        // 环境
        let environment = Environment::from_string(
            &env::var("APP_ENV").unwrap_or_else(|_| "development".to_string())
        );
        
        info!("应用环境: {:?}", environment);
        
        // 数据库配置
        let database_config = DatabaseConfig {
            url: env::var("DATABASE_URL")?,
            max_connections: env::var("DATABASE_MAX_CONNECTIONS")
                .unwrap_or_else(|_| config_defaults::DEFAULT_DB_MAX_CONNECTIONS.to_string())
                .parse()
                .unwrap_or(config_defaults::DEFAULT_DB_MAX_CONNECTIONS),
            min_connections: env::var("DATABASE_MIN_CONNECTIONS")
                .unwrap_or_else(|_| config_defaults::DEFAULT_DB_MIN_CONNECTIONS.to_string())
                .parse()
                .unwrap_or(config_defaults::DEFAULT_DB_MIN_CONNECTIONS),
            max_lifetime: Duration::from_secs(
                env::var("DATABASE_MAX_LIFETIME_SECS")
                    .unwrap_or_else(|_| config_defaults::DEFAULT_DB_MAX_LIFETIME_SECS.to_string()) // 30分钟
                    .parse()
                    .unwrap_or(config_defaults::DEFAULT_DB_MAX_LIFETIME_SECS),
            ),
            idle_timeout: Duration::from_secs(
                env::var("DATABASE_IDLE_TIMEOUT_SECS")
                    .unwrap_or_else(|_| config_defaults::DEFAULT_DB_IDLE_TIMEOUT_SECS.to_string()) // 10分钟
                    .parse()
                    .unwrap_or(config_defaults::DEFAULT_DB_IDLE_TIMEOUT_SECS),
            ),
            connect_timeout: Duration::from_secs(
                env::var("DATABASE_CONNECT_TIMEOUT_SECS")
                    .unwrap_or_else(|_| config_defaults::DEFAULT_DB_CONNECT_TIMEOUT_SECS.to_string())
                    .parse()
                    .unwrap_or(config_defaults::DEFAULT_DB_CONNECT_TIMEOUT_SECS),
            ),
        };
        
        // 缓存配置
        let cache_config = CacheConfig {
            redis_url: env::var("REDIS_URL").ok(),
        };
        
        // 认证配置
        let auth_config = AuthConfig {
            jwt_secret: env::var("JWT_SECRET").unwrap_or_else(|_| {
                if !environment.is_dev() {
                    warn!("JWT_SECRET 未设置，使用不安全的默认值");
                }
                "unsafe_jwt_secret_please_change_in_production".to_string()
            }),
            jwt_expires_in: Duration::from_secs(
                env::var("JWT_EXPIRES_IN_SECS")
                    .unwrap_or_else(|_| config_defaults::DEFAULT_JWT_ACCESS_TOKEN_EXPIRY_SECS.to_string()) // 1小时
                    .parse()
                    .unwrap_or(config_defaults::DEFAULT_JWT_ACCESS_TOKEN_EXPIRY_SECS),
            ),
            refresh_token_expires_in: Duration::from_secs(
                env::var("REFRESH_TOKEN_EXPIRES_IN_SECS")
                    .unwrap_or_else(|_| config_defaults::DEFAULT_JWT_REFRESH_TOKEN_EXPIRY_SECS.to_string()) // 30天
                    .parse()
                    .unwrap_or(config_defaults::DEFAULT_JWT_REFRESH_TOKEN_EXPIRY_SECS),
            ),
        };
        
        // 服务器配置
        let server_config = ServerConfig {
            host: env::var("HOST").unwrap_or_else(|_| config_defaults::DEFAULT_HOST.to_string()),
            port: env::var("PORT")
                .unwrap_or_else(|_| config_defaults::DEFAULT_PORT.to_string())
                .parse()
                .unwrap_or(config_defaults::DEFAULT_PORT),
        };
        
        // WebSocket 配置
        let websocket_config = WebSocketConfig {
            cleanup_interval_secs: env::var("WS_CLEANUP_INTERVAL_SECS")
                .unwrap_or_else(|_| config_defaults::DEFAULT_WS_CLEANUP_INTERVAL_SECS.to_string())
                .parse()
                .unwrap_or(config_defaults::DEFAULT_WS_CLEANUP_INTERVAL_SECS),
            inactive_timeout_secs: env::var("WS_INACTIVE_TIMEOUT_SECS")
                .unwrap_or_else(|_| config_defaults::DEFAULT_WS_INACTIVE_TIMEOUT_SECS.to_string())
                .parse()
                .unwrap_or(config_defaults::DEFAULT_WS_INACTIVE_TIMEOUT_SECS),
        };
        
        // 作业队列配置
        let job_queue_config = JobQueueConfig {
            amqp_url: env::var("AMQP_URL").ok(),
            enabled: env::var("JOB_QUEUE_ENABLED")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .unwrap_or(true),
        };
        
        Ok(Self {
            environment,
            database: database_config,
            cache: cache_config,
            auth: auth_config,
            server: server_config,
            job_queue: job_queue_config,
            websocket: websocket_config,
        })
    }
    
    /// 检查并输出配置信息
    pub fn validate_and_print(&self) {
        info!("===== 应用配置信息 =====");
        info!("环境: {:?}", self.environment);
        info!("数据库: {}", self.database.url);
        info!("数据库连接池: min={}, max={}", self.database.min_connections, self.database.max_connections);
        
        if let Some(redis_url) = &self.cache.redis_url {
            info!("Redis: {}", redis_url);
        } else {
            warn!("Redis未配置，某些功能可能不可用");
        }
        
        info!("服务器: {}:{}", self.server.host, self.server.port);
        info!(
            "WebSocket清理: interval={}s, inactive_timeout={}s",
            self.websocket.cleanup_interval_secs,
            self.websocket.inactive_timeout_secs
        );
        
        if !self.environment.is_prod() && self.auth.jwt_secret == "unsafe_jwt_secret_please_change_in_production" {
            warn!("使用默认JWT密钥，不建议在生产环境中使用");
        }
        
        if let Some(amqp_url) = &self.job_queue.amqp_url {
            if self.job_queue.enabled {
                info!("作业队列已启用: {}", amqp_url);
            } else {
                info!("作业队列已配置但未启用");
            }
        } else {
            warn!("作业队列未配置，异步任务将不可用");
        }
        
        info!("=======================");
    }
}