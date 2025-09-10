/// 配置适配器
/// 
/// 提供统一的配置管理，直接使用新架构
use std::sync::Arc;
use std::env;
use sqlx::PgPool;
// use crate::domain::DomainResult; // 暂时不需要
use crate::bootstrap::app_state::{
    AppConfig, 
    DatabaseConfig,
    CacheConfig,
    JobQueueConfig,
    NotificationConfig,
    WebSocketConfig,
    AuthConfig,
    MetricsConfig,
};

/// 认证配置
/// 服务器配置
#[derive(Debug, Clone)]
pub struct ServerConfig {
    pub port: u16,
    pub host: String,
}

/// CORS配置
#[derive(Debug, Clone)]
pub struct CorsConfig {
    pub allowed_origins: Vec<String>,
    pub allowed_methods: Vec<String>,
    pub allowed_headers: Vec<String>,
}

/// 环境类型
#[derive(Debug, Clone, PartialEq)]
pub enum Environment {
    Development,
    Production,
    Test,
}

// 移除了LegacyAppConfig，直接使用新架构配置

/// 数据库指标
#[derive(Debug, Clone)]
pub struct DbMetrics {
    pub pg_pool_size: u32,
    pub pg_pool_max_size: u32,
    pub redis_available: bool,
}

/// 数据库连接管理
pub struct Database {
    pool: PgPool,
}

impl Database {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
    
    pub fn pool(&self) -> &PgPool {
        &self.pool
    }
    
    pub fn pg_pool(&self) -> &PgPool {
        &self.pool
    }
    
    pub async fn check_health(&self) -> Result<(), sqlx::Error> {
        sqlx::query("SELECT 1").execute(&self.pool).await?;
        Ok(())
    }
    
    pub fn get_db_metrics(&self) -> DbMetrics {
        DbMetrics {
            pg_pool_size: 10, // 从连接池获取实际值
            pg_pool_max_size: 20, // 从连接池获取实际值
            redis_available: false, // 根据实际情况设置
        }
    }
}

/// 配置管理器
/// 
/// 使用新架构配置
pub struct HybridConfigManager {
    config: AppConfig,
    database: Arc<Database>,
}

impl HybridConfigManager {
    /// 从环境变量创建配置管理器
    pub async fn from_env() -> Result<Self, Box<dyn std::error::Error>> {
        let database_url = env::var("DATABASE_URL")
            .unwrap_or_else(|_| "postgresql://localhost/hotspot".to_string());
        
        let pool = PgPool::connect(&database_url).await?;
        let database = Arc::new(Database::new(pool));
        
        let database_config = DatabaseConfig {
            url: database_url,
            max_connections: env::var("DB_MAX_CONNECTIONS")
                .unwrap_or_else(|_| "10".to_string())
                .parse()
                .unwrap_or(10),
            min_connections: env::var("DB_MIN_CONNECTIONS")
                .unwrap_or_else(|_| "1".to_string())
                .parse()
                .unwrap_or(1),
            connect_timeout: env::var("DB_CONNECT_TIMEOUT")
                .unwrap_or_else(|_| "30".to_string())
                .parse()
                .unwrap_or(30),
            idle_timeout: env::var("DB_IDLE_TIMEOUT")
                .unwrap_or_else(|_| "600".to_string())
                .parse()
                .unwrap_or(600),
        };
        
        let cache_config = CacheConfig {
            cache_type: if env::var("REDIS_URL").is_ok() { "redis".to_string() } else { "memory".to_string() },
            redis_url: env::var("REDIS_URL").ok(),
            default_ttl: env::var("CACHE_TTL")
                .unwrap_or_else(|_| "3600".to_string())
                .parse()
                .unwrap_or(3600),
            max_memory_size: Some(env::var("MEMORY_CACHE_SIZE")
                .unwrap_or_else(|_| "1000".to_string())
                .parse()
                .unwrap_or(1000)),
        };
        
        let job_queue_config = JobQueueConfig {
            queue_type: env::var("JOB_QUEUE_TYPE")
                .unwrap_or_else(|_| "lite".to_string()),
            redis_url: env::var("JOB_REDIS_URL").ok(),
            max_retries: env::var("JOB_MAX_RETRIES")
                .unwrap_or_else(|_| "3".to_string())
                .parse()
                .unwrap_or(3),
            retry_delay: env::var("JOB_RETRY_DELAY")
                .unwrap_or_else(|_| "1000".to_string())
                .parse()
                .unwrap_or(1000),
            max_concurrent_jobs: env::var("JOB_MAX_CONCURRENT")
                .unwrap_or_else(|_| "4".to_string())
                .parse()
                .unwrap_or(4),
        };
        
        let websocket_config = WebSocketConfig {
            auth_timeout_seconds: env::var("WS_AUTH_TIMEOUT")
                .unwrap_or_else(|_| "30".to_string())
                .parse()
                .unwrap_or(30),
            heartbeat_interval_seconds: env::var("WS_HEARTBEAT_INTERVAL")
                .unwrap_or_else(|_| "30".to_string())
                .parse()
                .unwrap_or(30),
            idle_timeout_seconds: env::var("WS_IDLE_TIMEOUT")
                .unwrap_or_else(|_| "300".to_string())
                .parse()
                .unwrap_or(300),
            max_connections_per_user: env::var("WS_MAX_CONNECTIONS_PER_USER")
                .unwrap_or_else(|_| "5".to_string())
                .parse()
                .unwrap_or(5),
        };
        
        let notification_config = NotificationConfig {
            enable_friend_notifications: env::var("ENABLE_FRIEND_NOTIFICATIONS")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .unwrap_or(true),
            enable_group_notifications: env::var("ENABLE_GROUP_NOTIFICATIONS")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .unwrap_or(true),
            enable_nearby_notifications: env::var("ENABLE_NEARBY_NOTIFICATIONS")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .unwrap_or(true),
            nearby_radius_km: env::var("NEARBY_RADIUS_KM")
                .unwrap_or_else(|_| "5.0".to_string())
                .parse()
                .unwrap_or(5.0),
            enable_global_broadcast: env::var("ENABLE_GLOBAL_BROADCAST")
                .unwrap_or_else(|_| "false".to_string())
                .parse()
                .unwrap_or(false),
        };
        
        let metrics_config = MetricsConfig {
            enable_basic_metrics: env::var("ENABLE_BASIC_METRICS")
                .unwrap_or_else(|_| "true".to_string())
                .parse()
                .unwrap_or(true),
            enable_prometheus: env::var("ENABLE_PROMETHEUS")
                .unwrap_or_else(|_| "false".to_string())
                .parse()
                .unwrap_or(false),
            prometheus_endpoint: env::var("PROMETHEUS_ENDPOINT")
                .unwrap_or_else(|_| "/metrics".to_string()),
            collection_interval_seconds: env::var("METRICS_INTERVAL")
                .unwrap_or_else(|_| "60".to_string())
                .parse()
                .unwrap_or(60),
        };
        
        let auth_config = AuthConfig {
            jwt_secret: env::var("JWT_SECRET")
                .unwrap_or_else(|_| "your-secret-key-change-in-production".to_string()),
            jwt_expiry_hours: env::var("JWT_EXPIRY_HOURS")
                .unwrap_or_else(|_| "24".to_string())
                .parse()
                .unwrap_or(24),
            temp_user_expiry_hours: env::var("TEMP_USER_EXPIRY_HOURS")
                .unwrap_or_else(|_| "72".to_string())
                .parse()
                .unwrap_or(72),
        };
        
        let new_config = AppConfig {
            database: database_config.clone(),
            cache: cache_config.clone(),
            job_queue: job_queue_config.clone(),
            notification: notification_config,
            websocket: websocket_config.clone(),
            auth: auth_config.clone(),
            metrics: metrics_config,
        };
        
        let _server_config = ServerConfig {
            port: env::var("PORT")
                .unwrap_or_else(|_| "3000".to_string())
                .parse()
                .unwrap_or(3000),
            host: env::var("HOST")
                .unwrap_or_else(|_| "0.0.0.0".to_string()),
        };
        
        let _cors_config = CorsConfig {
            allowed_origins: vec!["*".to_string()],
            allowed_methods: vec!["GET".to_string(), "POST".to_string(), "PUT".to_string(), "DELETE".to_string()],
            allowed_headers: vec!["*".to_string()],
        };
        
        let _environment = match env::var("ENVIRONMENT").as_deref() {
            Ok("production") => Environment::Production,
            Ok("test") => Environment::Test,
            _ => Environment::Development,
        };
        
        Ok(Self {
            config: new_config,
            database,
        })
    }
    
    /// 获取配置
    pub fn config(&self) -> &AppConfig {
        &self.config
    }
    
    /// 获取数据库连接
    pub fn database(&self) -> Arc<Database> {
        self.database.clone()
    }
    
    /// 检查数据库健康状态
    pub async fn check_health(&self) -> Result<(), Box<dyn std::error::Error>> {
        self.database.check_health().await?;
        Ok(())
    }
    
    /// 获取数据库指标
    pub fn get_db_metrics(&self) -> DbMetrics {
        DbMetrics {
            pg_pool_size: 10, // 从连接池获取实际值
            pg_pool_max_size: self.config.database.max_connections as u32,
            redis_available: self.config.cache.redis_url.is_some(),
        }
    }
    
    /// 验证并打印配置
    pub fn validate_and_print(&self) {
        tracing::info!("🔧 配置验证:");
        tracing::info!("  📊 数据库: {} 连接", self.config.database.max_connections);
        tracing::info!("  🗄️  缓存: {}", if self.config.cache.redis_url.is_some() { "Redis" } else { "内存" });
        tracing::info!("  🔌 WebSocket: 启用");
        tracing::info!("  ⚙️  任务队列: {}", if self.config.job_queue.queue_type == "lite" { "Lite队列" } else { "Redis队列" });
        tracing::info!("  🌍 环境: Development"); // 暂时硬编码
        tracing::info!("  🚪 端口: {}", env::var("SERVER_PORT").unwrap_or("8080".to_string()).parse::<u16>().unwrap_or(8080));
    }
}