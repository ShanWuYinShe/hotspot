/// 数据库连接池监控系统
/// 
/// 提供数据库连接池状态、性能统计和健康检查功能
use sqlx::PgPool;
use std::time::{Duration, Instant};
use serde_json::Value;

/// 数据库连接池统计信息
#[derive(Debug, Clone)]
pub struct DbPoolStats {
    /// 连接池大小
    pub pool_size: u32,
    /// 空闲连接数
    pub idle_connections: usize,
    /// 活跃连接数  
    pub active_connections: usize,
    /// 等待连接的任务数
    pub waiting_tasks: usize,
    /// 连接池是否已关闭
    pub is_closed: bool,
    /// 统计时间
    pub timestamp: std::time::SystemTime,
}

/// 数据库性能统计
#[derive(Debug, Clone)]
pub struct DbPerformanceStats {
    /// 总查询数
    pub total_queries: u64,
    /// 成功查询数
    pub successful_queries: u64,
    /// 失败查询数
    pub failed_queries: u64,
    /// 平均查询时间（毫秒）
    pub avg_query_time_ms: f64,
    /// 最慢查询时间（毫秒）
    pub slowest_query_ms: u64,
    /// 最快查询时间（毫秒）
    pub fastest_query_ms: u64,
    /// 慢查询数（>100ms）
    pub slow_queries_count: u64,
    /// 统计时间范围
    pub stats_duration: Duration,
    /// 最后更新时间
    pub last_updated: std::time::SystemTime,
}

impl DbPerformanceStats {
    pub fn new() -> Self {
        Self {
            total_queries: 0,
            successful_queries: 0,
            failed_queries: 0,
            avg_query_time_ms: 0.0,
            slowest_query_ms: 0,
            fastest_query_ms: u64::MAX,
            slow_queries_count: 0,
            stats_duration: Duration::from_secs(0),
            last_updated: std::time::SystemTime::now(),
        }
    }

    /// 记录查询性能
    pub fn record_query(&mut self, duration: Duration, success: bool) {
        let duration_ms = duration.as_millis() as u64;
        
        self.total_queries += 1;
        if success {
            self.successful_queries += 1;
        } else {
            self.failed_queries += 1;
        }
        
        // 更新最快和最慢查询时间
        if duration_ms > self.slowest_query_ms {
            self.slowest_query_ms = duration_ms;
        }
        if duration_ms < self.fastest_query_ms {
            self.fastest_query_ms = duration_ms;
        }
        
        // 慢查询统计
        if duration_ms > 100 {
            self.slow_queries_count += 1;
        }
        
        // 计算平均查询时间
        self.avg_query_time_ms = (self.avg_query_time_ms * (self.total_queries - 1) as f64 + duration_ms as f64) / self.total_queries as f64;
        
        self.last_updated = std::time::SystemTime::now();
    }

    /// 获取成功率百分比
    pub fn success_rate(&self) -> f64 {
        if self.total_queries == 0 {
            0.0
        } else {
            (self.successful_queries as f64 / self.total_queries as f64) * 100.0
        }
    }

    /// 获取失败率百分比
    pub fn failure_rate(&self) -> f64 {
        if self.total_queries == 0 {
            0.0
        } else {
            (self.failed_queries as f64 / self.total_queries as f64) * 100.0
        }
    }

    /// 获取慢查询率百分比
    pub fn slow_query_rate(&self) -> f64 {
        if self.total_queries == 0 {
            0.0
        } else {
            (self.slow_queries_count as f64 / self.total_queries as f64) * 100.0
        }
    }
}

/// 数据库监控器
pub struct DbMonitor {
    pool: PgPool,
    performance_stats: std::sync::Arc<std::sync::Mutex<DbPerformanceStats>>,
}

impl DbMonitor {
    pub fn new(pool: PgPool) -> Self {
        Self {
            pool,
            performance_stats: std::sync::Arc::new(std::sync::Mutex::new(DbPerformanceStats::new())),
        }
    }

    /// 获取连接池统计信息
    pub fn get_pool_stats(&self) -> DbPoolStats {
        DbPoolStats {
            pool_size: self.pool.size(),
            idle_connections: self.pool.num_idle(),
            active_connections: (self.pool.size() as usize).saturating_sub(self.pool.num_idle()),
            waiting_tasks: 0, // SQLx 不直接提供这个信息
            is_closed: self.pool.is_closed(),
            timestamp: std::time::SystemTime::now(),
        }
    }

    /// 获取性能统计信息
    pub fn get_performance_stats(&self) -> DbPerformanceStats {
        self.performance_stats.lock().unwrap().clone()
    }

    /// 记录查询性能
    pub fn record_query_performance(&self, duration: Duration, success: bool) {
        let mut stats = self.performance_stats.lock().unwrap();
        stats.record_query(duration, success);
    }

    /// 执行数据库健康检查
    pub async fn health_check(&self) -> Result<Value, sqlx::Error> {
        let start = Instant::now();
        
        // 执行简单的查询来测试连接
        let result = sqlx::query!("SELECT 1 as health_check")
            .fetch_one(&self.pool)
            .await;
        
        let duration = start.elapsed();
        let is_healthy = result.is_ok();
        
        // 记录健康检查性能
        self.record_query_performance(duration, is_healthy);
        
        let pool_stats = self.get_pool_stats();
        let perf_stats = self.get_performance_stats();
        
        Ok(serde_json::json!({
            "database": {
                "status": if is_healthy { "healthy" } else { "unhealthy" },
                "response_time_ms": duration.as_millis(),
                "pool": {
                    "size": pool_stats.pool_size,
                    "idle_connections": pool_stats.idle_connections,
                    "active_connections": pool_stats.active_connections,
                    "is_closed": pool_stats.is_closed
                },
                "performance": {
                    "total_queries": perf_stats.total_queries,
                    "success_rate": perf_stats.success_rate(),
                    "failure_rate": perf_stats.failure_rate(),
                    "avg_query_time_ms": perf_stats.avg_query_time_ms,
                    "slow_query_rate": perf_stats.slow_query_rate(),
                    "slowest_query_ms": perf_stats.slowest_query_ms,
                    "fastest_query_ms": if perf_stats.fastest_query_ms == u64::MAX { 0 } else { perf_stats.fastest_query_ms }
                }
            }
        }))
    }

    /// 获取详细的数据库状态报告
    pub async fn get_detailed_report(&self) -> Value {
        let pool_stats = self.get_pool_stats();
        let perf_stats = self.get_performance_stats();
        
        // 尝试获取数据库版本信息
        let db_version = sqlx::query!("SELECT version() as version")
            .fetch_one(&self.pool)
            .await
            .map(|row| row.version)
            .unwrap_or_else(|_| "Unknown".to_string());

        serde_json::json!({
            "database_info": {
                "version": db_version,
                "pool_configuration": {
                    "max_connections": pool_stats.pool_size,
                    "current_idle": pool_stats.idle_connections,
                    "current_active": pool_stats.active_connections,
                    "pool_closed": pool_stats.is_closed
                }
            },
            "performance_metrics": {
                "query_statistics": {
                    "total_queries": perf_stats.total_queries,
                    "successful_queries": perf_stats.successful_queries,
                    "failed_queries": perf_stats.failed_queries,
                    "success_rate_percent": perf_stats.success_rate(),
                    "failure_rate_percent": perf_stats.failure_rate()
                },
                "timing_statistics": {
                    "average_query_time_ms": perf_stats.avg_query_time_ms,
                    "fastest_query_ms": if perf_stats.fastest_query_ms == u64::MAX { 0 } else { perf_stats.fastest_query_ms },
                    "slowest_query_ms": perf_stats.slowest_query_ms,
                    "slow_queries_count": perf_stats.slow_queries_count,
                    "slow_query_rate_percent": perf_stats.slow_query_rate()
                }
            },
            "health_status": {
                "overall_status": if pool_stats.is_closed { "unhealthy" } else { "healthy" },
                "connection_pool_health": if pool_stats.idle_connections > 0 { "good" } else { "warning" },
                "performance_health": if perf_stats.slow_query_rate() < 10.0 { "good" } else { "warning" }
            },
            "timestamp": std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap_or_default()
                .as_secs()
        })
    }

    /// 重置性能统计
    pub fn reset_performance_stats(&self) {
        let mut stats = self.performance_stats.lock().unwrap();
        *stats = DbPerformanceStats::new();
    }

    /// 检查连接池是否健康
    pub fn is_pool_healthy(&self) -> bool {
        let stats = self.get_pool_stats();
        !stats.is_closed && stats.idle_connections > 0
    }

    /// 获取连接池利用率
    pub fn get_pool_utilization(&self) -> f64 {
        let stats = self.get_pool_stats();
        if stats.pool_size == 0 {
            0.0
        } else {
            (stats.active_connections as f64 / stats.pool_size as f64) * 100.0
        }
    }
}

/// 全局数据库监控器实例
static GLOBAL_DB_MONITOR: std::sync::OnceLock<DbMonitor> = std::sync::OnceLock::new();

/// 初始化全局数据库监控器
pub fn init_db_monitor(pool: PgPool) {
    GLOBAL_DB_MONITOR.set(DbMonitor::new(pool)).ok();
}

/// 获取全局数据库监控器
pub fn get_db_monitor() -> Option<&'static DbMonitor> {
    GLOBAL_DB_MONITOR.get()
}

/// 记录数据库查询性能的便捷函数
pub fn record_db_query_performance(duration: Duration, success: bool) {
    if let Some(monitor) = get_db_monitor() {
        monitor.record_query_performance(duration, success);
    }
}

/// 获取数据库健康状态的便捷函数
pub async fn get_db_health() -> Option<Value> {
    if let Some(monitor) = get_db_monitor() {
        monitor.health_check().await.ok()
    } else {
        None
    }
}
