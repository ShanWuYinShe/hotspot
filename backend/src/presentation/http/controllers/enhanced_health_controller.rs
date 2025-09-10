/// 增强的健康检查HTTP控制器
/// 
/// 提供详细的系统健康状态检查，包括数据库、缓存、任务队列等组件
use super::{Controller, ok_response};
use axum::Json;
use axum::response::{Response, IntoResponse};
use serde_json::json;
use std::sync::Arc;
use std::time::Instant;
use tracing::warn;

use crate::bootstrap::HybridConfigManager;
use crate::ports::cache::Cache;
use crate::ports::jobs::JobDispatcher;

// 使用 JobDispatcher trait 而不是 JobDispatcherPort 类型别名
use crate::utils::performance_monitor;

/// 增强的健康检查控制器
#[derive(Clone)]
pub struct EnhancedHealthController<C: Cache, J: JobDispatcher> {
    config_manager: Arc<HybridConfigManager>,
    cache: Option<Arc<C>>,
    job_dispatcher: Option<Arc<J>>,
}

impl<C: Cache, J: JobDispatcher> EnhancedHealthController<C, J> {
    /// 创建新的增强健康检查控制器
    pub fn new(
        config_manager: Arc<HybridConfigManager>,
        cache: Option<Arc<C>>,
        job_dispatcher: Option<Arc<J>>,
    ) -> Self {
        Self {
            config_manager,
            cache,
            job_dispatcher,
        }
    }

    /// 基础健康检查端点
    pub async fn health_check(&self) -> Response {
        let response = json!({
            "status": "healthy",
            "version": env!("CARGO_PKG_VERSION"),
            "timestamp": chrono::Utc::now().to_rfc3339(),
            "uptime": self.get_uptime_info()
        });
        
        ok_response(response)
    }

    /// 详细健康检查端点
    pub async fn detailed_health_check(&self) -> Response {
        let start_time = Instant::now();
        
        // 检查各个组件
        let database_check = self.check_database().await;
        let cache_check = self.check_cache().await;
        let job_queue_check = self.check_job_queue().await;
        
        // 获取性能监控数据
        let performance_monitor = performance_monitor::get_global_performance_monitor();
        let performance_report = performance_monitor.get_performance_report();
        
        let check_duration = start_time.elapsed();
        
        // 确定整体状态
        let component_healthy = database_check.is_healthy && cache_check.is_healthy && job_queue_check.is_healthy;
        let performance_healthy = performance_report.average_response_time < 500.0;
        
        let overall_status = if component_healthy && performance_healthy {
            "healthy"
        } else if component_healthy {
            "degraded"
        } else {
            "unhealthy"
        };

        let response = json!({
            "status": overall_status,
            "version": env!("CARGO_PKG_VERSION"),
            "timestamp": chrono::Utc::now().to_rfc3339(),
            "check_duration_ms": check_duration.as_millis(),
            "uptime": self.get_uptime_info(),
            "checks": {
                "database": database_check,
                "cache": cache_check,
                "job_queue": job_queue_check
            },
            "performance": {
                "total_queries": performance_report.total_queries,
                "average_response_time": performance_report.average_response_time,
                "slow_queries_count": performance_report.slow_queries.len(),
                "geo_query_count": performance_report.geo_query_count,
                "recommendations": performance_report.recommendations
            },
            "system_info": self.get_system_info()
        });

        if overall_status == "unhealthy" {
            {
                let error_response = json!({
                    "code": 5000,
                    "msg": "System unhealthy",
                    "data": response
                });
                (axum::http::StatusCode::OK, Json(error_response)).into_response()
            }
        } else {
            ok_response(response)
        }
    }

    /// 就绪检查端点
    pub async fn readiness_check(&self) -> Response {
        // 就绪检查主要关注关键依赖是否可用
        let database_check = self.check_database().await;
        
        if database_check.is_healthy {
            let response = json!({
                "status": "ready",
                "timestamp": chrono::Utc::now().to_rfc3339(),
                "checks": {
                    "database": "ready"
                }
            });
            ok_response(response)
        } else {
            let response = json!({
                "status": "not_ready",
                "timestamp": chrono::Utc::now().to_rfc3339(),
                "reason": "Database not available"
            });
            {
                let error_response = json!({
                    "code": 5000,
                    "msg": "Service not ready",
                    "data": response
                });
                (axum::http::StatusCode::OK, Json(error_response)).into_response()
            }
        }
    }

    /// 存活检查端点
    pub async fn liveness_check(&self) -> Response {
        // 存活检查只检查应用本身是否还在运行
        let response = json!({
            "status": "alive",
            "timestamp": chrono::Utc::now().to_rfc3339(),
            "pid": std::process::id()
        });
        
        ok_response(response)
    }

    /// 检查数据库状态
    async fn check_database(&self) -> HealthCheckResult {
        let start_time = Instant::now();
        
        match self.config_manager.check_health().await {
            Ok(_) => {
                let duration = start_time.elapsed();
                let metrics = self.config_manager.get_db_metrics();
                
                HealthCheckResult {
                    is_healthy: true,
                    status: "healthy".to_string(),
                    response_time_ms: duration.as_millis() as u64,
                    details: Some(json!({
                        "pg_pool_size": metrics.pg_pool_size,
                        "pg_pool_max_size": metrics.pg_pool_max_size,
                        "redis_available": metrics.redis_available
                    })),
                    error: None,
                }
            }
            Err(e) => {
                let duration = start_time.elapsed();
                warn!("Database health check failed: {:?}", e);
                
                HealthCheckResult {
                    is_healthy: false,
                    status: "unhealthy".to_string(),
                    response_time_ms: duration.as_millis() as u64,
                    details: None,
                    error: Some(format!("Database connection failed: {}", e)),
                }
            }
        }
    }

    /// 检查缓存状态
    async fn check_cache(&self) -> HealthCheckResult {
        let start_time = Instant::now();
        
        if let Some(cache) = &self.cache {
            match cache.get_stats().await {
                Ok(stats) => {
                    let duration = start_time.elapsed();
                    
                    HealthCheckResult {
                        is_healthy: true,
                        status: "healthy".to_string(),
                        response_time_ms: duration.as_millis() as u64,
                        details: Some(json!({
                            "hits": stats.hit_count,
                            "misses": stats.miss_count,
                            "entries": stats.total_keys,
                            "hit_rate": if stats.hit_count + stats.miss_count > 0 { 
                       stats.hit_count as f64 / (stats.hit_count + stats.miss_count) as f64 
                   } else { 
                       0.0 
                   },
                            "memory_usage": stats.memory_usage
                        })),
                        error: None,
                    }
                }
                Err(e) => {
                    let duration = start_time.elapsed();
                    warn!("Cache health check failed: {:?}", e);
                    
                    HealthCheckResult {
                        is_healthy: false,
                        status: "unhealthy".to_string(),
                        response_time_ms: duration.as_millis() as u64,
                        details: None,
                        error: Some(format!("Cache check failed: {:?}", e)),
                    }
                }
            }
        } else {
            HealthCheckResult {
                is_healthy: true,
                status: "not_configured".to_string(),
                response_time_ms: 0,
                details: Some(json!({"message": "Cache not configured"})),
                error: None,
            }
        }
    }

    /// 检查任务队列状态
    async fn check_job_queue(&self) -> HealthCheckResult {
        let start_time = Instant::now();
        
        if let Some(job_dispatcher) = &self.job_dispatcher {
            match job_dispatcher.get_queue_stats().await {
                Ok(stats) => {
                    let duration = start_time.elapsed();
                    
                    HealthCheckResult {
                        is_healthy: true,
                        status: "healthy".to_string(),
                        response_time_ms: duration.as_millis() as u64,
                        details: Some(json!({
                            "queued_jobs": stats.queued_jobs,
                            "running_jobs": stats.running_jobs,
                            "completed_jobs": stats.completed_jobs,
                            "failed_jobs": stats.failed_jobs,
                            "avg_processing_time_ms": stats.avg_processing_time_ms
                        })),
                        error: None,
                    }
                }
                Err(e) => {
                    let duration = start_time.elapsed();
                    warn!("Job queue health check failed: {:?}", e);
                    
                    HealthCheckResult {
                        is_healthy: false,
                        status: "unhealthy".to_string(),
                        response_time_ms: duration.as_millis() as u64,
                        details: None,
                        error: Some(format!("Job queue check failed: {:?}", e)),
                    }
                }
            }
        } else {
            HealthCheckResult {
                is_healthy: true,
                status: "not_configured".to_string(),
                response_time_ms: 0,
                details: Some(json!({"message": "Job queue not configured"})),
                error: None,
            }
        }
    }

    /// 获取运行时间信息
    fn get_uptime_info(&self) -> serde_json::Value {
        // 这里可以实现更复杂的运行时间跟踪
        json!({
            "started_at": "unknown", // 可以在应用启动时记录
            "uptime_seconds": "unknown"
        })
    }

    /// 获取系统信息
    fn get_system_info(&self) -> serde_json::Value {
        json!({
            "rust_version": std::env::var("RUSTC_VERSION").unwrap_or_else(|_| "unknown".to_string()),
            "target": std::env::var("TARGET").unwrap_or_else(|_| "unknown".to_string()),
            "profile": if cfg!(debug_assertions) { "debug" } else { "release" },
            "features": self.get_enabled_features()
        })
    }

    /// 获取启用的功能特性
    fn get_enabled_features(&self) -> Vec<&'static str> {
        let mut features = vec![];
        
        #[cfg(feature = "postgres")]
        features.push("postgres");
        
        #[cfg(feature = "cache_redis")]
        features.push("cache_redis");
        
        #[cfg(feature = "cache_in_memory")]
        features.push("cache_in_memory");
        
        #[cfg(feature = "jobs_lite")]
        features.push("jobs_lite");
        
        #[cfg(feature = "jobs_redis")]
        features.push("jobs_redis");
        
        #[cfg(feature = "ws")]
        features.push("ws");
        
        features
    }
}

/// 健康检查结果
#[derive(Debug, Clone)]
struct HealthCheckResult {
    is_healthy: bool,
    status: String,
    response_time_ms: u64,
    details: Option<serde_json::Value>,
    error: Option<String>,
}

impl serde::Serialize for HealthCheckResult {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut state = serializer.serialize_struct("HealthCheckResult", 5)?;
        state.serialize_field("is_healthy", &self.is_healthy)?;
        state.serialize_field("status", &self.status)?;
        state.serialize_field("response_time_ms", &self.response_time_ms)?;
        if let Some(details) = &self.details {
            state.serialize_field("details", details)?;
        }
        if let Some(error) = &self.error {
            state.serialize_field("error", error)?;
        }
        state.end()
    }
}

#[async_trait::async_trait]
impl<C: Cache, J: JobDispatcher> Controller for EnhancedHealthController<C, J> {
    fn name(&self) -> &'static str {
        "EnhancedHealthController"
    }
}
