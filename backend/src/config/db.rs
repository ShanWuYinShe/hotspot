use redis::{Client as RedisClient, aio::ConnectionManager};
use sqlx::{PgPool, postgres::PgPoolOptions};
use std::sync::Arc;
use tracing::{info, warn, error};

use super::app_config::{AppConfig, DatabaseConfig, CacheConfig};

#[derive(Clone)]
pub struct Database {
    pub pg_pool: PgPool,
    pub redis_manager: Option<ConnectionManager>,
    pub redis_client: Option<Arc<RedisClient>>,
}

impl std::fmt::Debug for Database {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Database")
            .field("pg_pool", &"PgPool")
            .field("redis_manager", &self.redis_manager.is_some())
            .field("redis_client", &self.redis_client.is_some())
            .finish()
    }
}

impl Database {
    /// 从配置连接数据库
    pub async fn from_config(config: &AppConfig) -> Result<Self, Box<dyn std::error::Error>> {
        Self::connect(&config.database, &config.cache).await
    }
    
    /// 连接数据库和Redis
    pub async fn connect(
        db_config: &DatabaseConfig,
        cache_config: &CacheConfig,
    ) -> Result<Self, Box<dyn std::error::Error>> {
        // 连接PostgreSQL
        info!("连接PostgreSQL数据库...");
        info!("连接池配置: 最小连接数={}, 最大连接数={}, 连接超时={}秒, 空闲超时={}秒, 最大生命周期={}秒",
            db_config.min_connections,
            db_config.max_connections,
            db_config.connect_timeout.as_secs(),
            db_config.idle_timeout.as_secs(),
            db_config.max_lifetime.as_secs()
        );
        
        let pg_pool = PgPoolOptions::new()
            .max_connections(db_config.max_connections)
            .min_connections(db_config.min_connections)
            .max_lifetime(db_config.max_lifetime)
            .idle_timeout(db_config.idle_timeout)
            .acquire_timeout(db_config.connect_timeout)
            // 启用连接前健康检查
            .test_before_acquire(true)
            // 连接数据库
            .connect(&db_config.url)
            .await?;
            
        info!("PostgreSQL连接成功, 连接池空闲连接数: {}", pg_pool.num_idle());

        // 连接Redis（如果配置了）
        let (redis_client, redis_manager) = if let Some(redis_url) = &cache_config.redis_url {
            info!("连接Redis服务器...");
            match RedisClient::open(redis_url.clone()) {
                Ok(client) => {
                    // 测试Redis连接
                    match client.get_multiplexed_async_connection().await {
                        Ok(_) => {
                            info!("Redis连接成功");
                            
                            // 创建连接管理器
                            match ConnectionManager::new(client.clone()).await {
                                Ok(manager) => {
                                    info!("Redis连接管理器初始化成功");
                                    (Some(Arc::new(client)), Some(manager))
                                },
                                Err(e) => {
                                    warn!("Redis连接管理器初始化失败: {}. 缓存功能将不可用", e);
                                    (Some(Arc::new(client)), None)
                                }
                            }
                        },
                        Err(e) => {
                            warn!("Redis连接测试失败: {}. 缓存功能将不可用", e);
                            (None, None)
                        }
                    }
                },
                Err(e) => {
                    warn!("Redis客户端创建失败: {}. 缓存功能将不可用", e);
                    (None, None)
                }
            }
        } else {
            warn!("Redis未配置，缓存功能将不可用");
            (None, None)
        };

        Ok(Self {
            pg_pool,
            redis_client,
            redis_manager,
        })
    }
    
    /// 检查数据库健康状态
    pub async fn check_health(&self) -> Result<(), Box<dyn std::error::Error>> {
        // 检查PostgreSQL连接
        match sqlx::query("SELECT 1").execute(&self.pg_pool).await {
            Ok(_) => info!("PostgreSQL健康检查成功"),
            Err(e) => {
                error!("PostgreSQL健康检查失败: {}", e);
                return Err(Box::new(e));
            }
        }
        
        // 检查Redis连接（如果配置了）
        if let Some(manager) = &self.redis_manager {
            let mut conn = manager.clone();
            match redis::cmd("PING").query_async::<String>(&mut conn).await {
                Ok(pong) => info!("Redis健康检查成功: {:?}", pong),
                Err(e) => {
                    warn!("Redis健康检查失败: {}. 部分缓存功能可能不可用", e);
                    // Redis连接失败不应该阻止整个应用程序启动
                }
            }
        }
        
        Ok(())
    }
    
    /// 获取数据库连接池指标
    #[allow(dead_code)]
    pub fn get_db_metrics(&self) -> DbMetrics {
        DbMetrics {
            pg_pool_size: self.pg_pool.size(),
            pg_pool_max_size: self.pg_pool.options().get_max_connections() as u32,
            redis_available: self.redis_manager.is_some(),
        }
    }
}

/// 数据库连接池指标
#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct DbMetrics {
    pub pg_pool_size: u32,
    pub pg_pool_max_size: u32,
    pub redis_available: bool,
}
