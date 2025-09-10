/// 数据库连接池优化模块
/// 
/// 提供智能的数据库连接池配置和监控功能
use sqlx::{PgPool, postgres::PgPoolOptions};
use std::time::Duration;
use tracing::{info, warn, debug};
use serde::{Serialize, Deserialize};

/// 数据库连接池优化配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DbOptimizerConfig {
    /// 环境类型
    pub environment: Environment,
    /// 预期并发连接数
    pub expected_concurrent_connections: u32,
    /// 应用类型
    pub application_type: ApplicationType,
    /// 是否启用自动调优
    pub enable_auto_tuning: bool,
}

/// 环境类型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum Environment {
    Development,
    Testing,
    Staging,
    Production,
}

/// 应用类型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ApplicationType {
    WebApi,
    BackgroundWorker,
    DataProcessor,
    Mixed,
}

/// 优化后的数据库配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OptimizedDbConfig {
    pub min_connections: u32,
    pub max_connections: u32,
    pub connect_timeout: Duration,
    pub idle_timeout: Duration,
    pub max_lifetime: Duration,
    pub acquire_timeout: Duration,
    pub test_before_acquire: bool,
}

/// 连接池统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PoolStats {
    pub size: u32,
    pub idle: u32,
    pub connections: u32,
    pub acquire_count: u64,
    pub acquire_slow_count: u64,
    pub max_lifetime_closed_count: u64,
    pub idle_closed_count: u64,
}

/// 数据库优化器
pub struct DbOptimizer {
    config: DbOptimizerConfig,
}

impl DbOptimizer {
    /// 创建新的数据库优化器
    pub fn new(config: DbOptimizerConfig) -> Self {
        Self { config }
    }

    /// 根据环境和应用类型生成优化配置
    pub fn generate_optimized_config(&self) -> OptimizedDbConfig {
        match (&self.config.environment, &self.config.application_type) {
            // 开发环境配置
            (Environment::Development, _) => OptimizedDbConfig {
                min_connections: 2,
                max_connections: 10,
                connect_timeout: Duration::from_secs(10),
                idle_timeout: Duration::from_secs(300), // 5分钟
                max_lifetime: Duration::from_secs(1800), // 30分钟
                acquire_timeout: Duration::from_secs(30),
                test_before_acquire: false,
            },
            
            // 测试环境配置
            (Environment::Testing, _) => OptimizedDbConfig {
                min_connections: 1,
                max_connections: 5,
                connect_timeout: Duration::from_secs(5),
                idle_timeout: Duration::from_secs(60), // 1分钟
                max_lifetime: Duration::from_secs(600), // 10分钟
                acquire_timeout: Duration::from_secs(10),
                test_before_acquire: true,
            },
            
            // 预发布环境配置
            (Environment::Staging, ApplicationType::WebApi) => OptimizedDbConfig {
                min_connections: 5,
                max_connections: 20,
                connect_timeout: Duration::from_secs(10),
                idle_timeout: Duration::from_secs(600), // 10分钟
                max_lifetime: Duration::from_secs(3600), // 1小时
                acquire_timeout: Duration::from_secs(30),
                test_before_acquire: true,
            },
            
            // 生产环境 - Web API
            (Environment::Production, ApplicationType::WebApi) => {
                let base_connections = self.calculate_base_connections();
                OptimizedDbConfig {
                    min_connections: base_connections / 2,
                    max_connections: base_connections,
                    connect_timeout: Duration::from_secs(10),
                    idle_timeout: Duration::from_secs(900), // 15分钟
                    max_lifetime: Duration::from_secs(7200), // 2小时
                    acquire_timeout: Duration::from_secs(30),
                    test_before_acquire: true,
                }
            },
            
            // 生产环境 - 后台任务
            (Environment::Production, ApplicationType::BackgroundWorker) => OptimizedDbConfig {
                min_connections: 2,
                max_connections: 15,
                connect_timeout: Duration::from_secs(30),
                idle_timeout: Duration::from_secs(1800), // 30分钟
                max_lifetime: Duration::from_secs(14400), // 4小时
                acquire_timeout: Duration::from_secs(60),
                test_before_acquire: true,
            },
            
            // 生产环境 - 数据处理
            (Environment::Production, ApplicationType::DataProcessor) => OptimizedDbConfig {
                min_connections: 5,
                max_connections: 30,
                connect_timeout: Duration::from_secs(15),
                idle_timeout: Duration::from_secs(600), // 10分钟
                max_lifetime: Duration::from_secs(3600), // 1小时
                acquire_timeout: Duration::from_secs(45),
                test_before_acquire: true,
            },
            
            // 生产环境 - 混合应用
            (Environment::Production, ApplicationType::Mixed) => {
                let base_connections = self.calculate_base_connections();
                OptimizedDbConfig {
                    min_connections: base_connections / 3,
                    max_connections: base_connections + 10,
                    connect_timeout: Duration::from_secs(15),
                    idle_timeout: Duration::from_secs(900), // 15分钟
                    max_lifetime: Duration::from_secs(5400), // 1.5小时
                    acquire_timeout: Duration::from_secs(45),
                    test_before_acquire: true,
                }
            },
            
            // 其他环境默认配置
            _ => OptimizedDbConfig {
                min_connections: 3,
                max_connections: 15,
                connect_timeout: Duration::from_secs(10),
                idle_timeout: Duration::from_secs(600),
                max_lifetime: Duration::from_secs(3600),
                acquire_timeout: Duration::from_secs(30),
                test_before_acquire: true,
            },
        }
    }

    /// 计算基础连接数
    fn calculate_base_connections(&self) -> u32 {
        let cpu_cores = num_cpus::get() as u32;
        let expected = self.config.expected_concurrent_connections;
        
        // 基于CPU核心数和预期并发数计算
        let calculated = match self.config.application_type {
            ApplicationType::WebApi => {
                // Web API: CPU核心数 * 2 + 预期并发数 / 10
                cpu_cores * 2 + expected / 10
            },
            ApplicationType::BackgroundWorker => {
                // 后台任务: CPU核心数 + 5
                cpu_cores + 5
            },
            ApplicationType::DataProcessor => {
                // 数据处理: CPU核心数 * 3
                cpu_cores * 3
            },
            ApplicationType::Mixed => {
                // 混合应用: CPU核心数 * 2 + 预期并发数 / 15
                cpu_cores * 2 + expected / 15
            },
        };
        
        // 限制在合理范围内
        calculated.clamp(5, 100)
    }

    /// 创建优化的连接池
    pub async fn create_optimized_pool(&self, database_url: &str) -> Result<PgPool, sqlx::Error> {
        let config = self.generate_optimized_config();
        
        info!("🔧 创建优化的数据库连接池");
        info!("📊 连接池配置: 最小连接数={}, 最大连接数={}", config.min_connections, config.max_connections);
        info!("⏱️  超时配置: 连接超时={}s, 空闲超时={}s, 最大生命周期={}s", 
            config.connect_timeout.as_secs(),
            config.idle_timeout.as_secs(),
            config.max_lifetime.as_secs()
        );
        
        let pool_options = PgPoolOptions::new()
            .min_connections(config.min_connections)
            .max_connections(config.max_connections)
            .idle_timeout(config.idle_timeout)
            .max_lifetime(config.max_lifetime)
            .acquire_timeout(config.acquire_timeout)
            .test_before_acquire(config.test_before_acquire);

        // 注意：sqlx的PgPoolOptions没有connect_timeout方法，连接超时通过其他方式设置

        // 创建连接池
        let pool = pool_options.connect(database_url).await?;
        
        info!("✅ 数据库连接池创建成功");
        
        // 预热连接池
        if matches!(self.config.environment, Environment::Production) {
            self.warm_up_pool(&pool).await;
        }
        
        Ok(pool)
    }

    /// 预热连接池
    async fn warm_up_pool(&self, pool: &PgPool) {
        info!("🔥 预热数据库连接池...");
        
        let config = self.generate_optimized_config();
        let warm_up_connections = config.min_connections;
        
        let mut handles = Vec::new();
        
        for i in 0..warm_up_connections {
            let pool_clone = pool.clone();
            let handle = tokio::spawn(async move {
                match sqlx::query("SELECT 1").fetch_one(&pool_clone).await {
                    Ok(_) => debug!("连接 {} 预热成功", i + 1),
                    Err(e) => warn!("连接 {} 预热失败: {}", i + 1, e),
                }
            });
            handles.push(handle);
        }
        
        // 等待所有预热连接完成
        for handle in handles {
            let _ = handle.await;
        }
        
        info!("✅ 连接池预热完成");
    }

    /// 获取连接池统计信息
    pub fn get_pool_stats(&self, pool: &PgPool) -> PoolStats {
        PoolStats {
            size: pool.size(),
            idle: pool.num_idle() as u32,
            connections: (pool.size() as i32 - pool.num_idle() as i32).max(0) as u32,
            acquire_count: 0, // sqlx 不直接提供这些统计信息
            acquire_slow_count: 0,
            max_lifetime_closed_count: 0,
            idle_closed_count: 0,
        }
    }

    /// 监控连接池健康状态
    pub async fn monitor_pool_health(&self, pool: &PgPool) -> PoolHealthStatus {
        let stats = self.get_pool_stats(pool);
        let config = self.generate_optimized_config();
        
        // 检查连接池使用率
        let usage_rate = (stats.connections as f64 / config.max_connections as f64) * 100.0;
        
        // 执行健康检查查询
        let health_check_result = sqlx::query("SELECT 1")
            .fetch_one(pool)
            .await;
        
        let status = match health_check_result {
            Ok(_) => {
                if usage_rate > 90.0 {
                    PoolHealthStatus::Warning
                } else if usage_rate > 95.0 {
                    PoolHealthStatus::Critical
                } else {
                    PoolHealthStatus::Healthy
                }
            },
            Err(_) => PoolHealthStatus::Critical,
        };
        
        if matches!(status, PoolHealthStatus::Warning | PoolHealthStatus::Critical) {
            warn!("🚨 连接池健康状态: {:?}, 使用率: {:.1}%", status, usage_rate);
        } else {
            debug!("✅ 连接池健康状态: {:?}, 使用率: {:.1}%", status, usage_rate);
        }
        
        status
    }

    /// 自动调优连接池
    pub async fn auto_tune_pool(&self, pool: &PgPool) -> Option<OptimizedDbConfig> {
        if !self.config.enable_auto_tuning {
            return None;
        }
        
        let stats = self.get_pool_stats(pool);
        let current_config = self.generate_optimized_config();
        
        // 基于统计信息调整配置
        let mut new_config = current_config.clone();
        
        // 如果连接使用率持续很高，建议增加最大连接数
        let usage_rate = stats.connections as f64 / current_config.max_connections as f64;
        
        if usage_rate > 0.8 {
            new_config.max_connections = ((current_config.max_connections as f64) * 1.2) as u32;
            new_config.min_connections = ((current_config.min_connections as f64) * 1.1) as u32;
            
            info!("🔧 自动调优: 增加连接池大小 max={}, min={}", 
                new_config.max_connections, new_config.min_connections);
        } else if usage_rate < 0.3 {
            new_config.max_connections = ((current_config.max_connections as f64) * 0.9) as u32;
            new_config.min_connections = ((current_config.min_connections as f64) * 0.9) as u32;
            
            info!("🔧 自动调优: 减少连接池大小 max={}, min={}", 
                new_config.max_connections, new_config.min_connections);
        }
        
        // 确保最小值合理
        new_config.max_connections = new_config.max_connections.max(5);
        new_config.min_connections = new_config.min_connections.max(1).min(new_config.max_connections);
        
        if new_config.max_connections != current_config.max_connections ||
           new_config.min_connections != current_config.min_connections {
            Some(new_config)
        } else {
            None
        }
    }
}

/// 连接池健康状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum PoolHealthStatus {
    Healthy,
    Warning,
    Critical,
}

/// 便捷函数：创建默认的开发环境优化器
pub fn create_dev_optimizer() -> DbOptimizer {
    DbOptimizer::new(DbOptimizerConfig {
        environment: Environment::Development,
        expected_concurrent_connections: 10,
        application_type: ApplicationType::WebApi,
        enable_auto_tuning: false,
    })
}

/// 便捷函数：创建生产环境优化器
pub fn create_production_optimizer(expected_concurrent: u32) -> DbOptimizer {
    DbOptimizer::new(DbOptimizerConfig {
        environment: Environment::Production,
        expected_concurrent_connections: expected_concurrent,
        application_type: ApplicationType::Mixed,
        enable_auto_tuning: true,
    })
}
