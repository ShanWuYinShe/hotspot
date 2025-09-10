/// 健康检查HTTP控制器
use super::{Controller, ok_response};
use axum::Json;
use axum::response::{Response, IntoResponse};
use serde_json::json;
use std::time::Instant;


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

#[derive(Clone)]
pub struct HealthController;

impl HealthController {
    pub fn new() -> Self {
        Self
    }
    
    /// 健康检查端点
    pub async fn health_check() -> Response {
        let response = json!({
            "status": "healthy",
            "version": env!("CARGO_PKG_VERSION"),
            "timestamp": chrono::Utc::now().to_rfc3339()
        });
        
        ok_response(response)
    }
    
    /// 详细健康检查端点
    pub async fn detailed_health_check() -> Response {
        let start_time = Instant::now();
        
        // 基础系统检查
        let system_check = Self::check_system_resources().await;
        let memory_check = Self::check_memory_usage().await;
        
        let check_duration = start_time.elapsed();
        
        // 确定整体状态
        let overall_status = if system_check.is_healthy && memory_check.is_healthy {
            "healthy"
        } else if system_check.is_healthy || memory_check.is_healthy {
            "degraded"
        } else {
            "unhealthy"
        };

        let response = json!({
            "status": overall_status,
            "version": env!("CARGO_PKG_VERSION"),
            "timestamp": chrono::Utc::now().to_rfc3339(),
            "check_duration_ms": check_duration.as_millis(),
            "uptime": Self::get_uptime_info(),
            "checks": {
                "system": system_check,
                "memory": memory_check
            },
            "system_info": Self::get_system_info()
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
    pub async fn readiness_check() -> Response {
        let response = json!({
            "status": "ready",
            "timestamp": chrono::Utc::now().to_rfc3339(),
            "checks": {
                "application": "ready"
            }
        });
        
        ok_response(response)
    }
    
    /// 存活检查端点
    pub async fn liveness_check() -> Response {
        let response = json!({
            "status": "alive",
            "timestamp": chrono::Utc::now().to_rfc3339(),
            "pid": std::process::id()
        });
        
        ok_response(response)
    }
    
    /// 检查系统资源
    async fn check_system_resources() -> HealthCheckResult {
        let start_time = Instant::now();
        
        // 检查CPU核心数和可用内存
        let cpu_count = num_cpus::get();
        let available_memory = Self::get_available_memory();
        
        let duration = start_time.elapsed();
        
        let is_healthy = cpu_count > 0 && available_memory > 100; // 至少100MB内存
        
        HealthCheckResult {
            is_healthy,
            status: if is_healthy { "healthy".to_string() } else { "unhealthy".to_string() },
            response_time_ms: duration.as_millis() as u64,
            details: Some(json!({
                "cpu_cores": cpu_count,
                "available_memory_mb": available_memory,
                "load_average": Self::get_load_average()
            })),
            error: if !is_healthy { Some("System resources insufficient".to_string()) } else { None },
        }
    }
    
    /// 检查内存使用情况
    async fn check_memory_usage() -> HealthCheckResult {
        let start_time = Instant::now();
        
        let available_memory = Self::get_available_memory();
        let total_memory = Self::get_total_memory();
        let memory_usage_percent = if total_memory > 0 {
            ((total_memory - available_memory) as f64 / total_memory as f64) * 100.0
        } else {
            0.0
        };
        
        let duration = start_time.elapsed();
        
        let is_healthy = memory_usage_percent < 90.0; // 内存使用率低于90%
        
        HealthCheckResult {
            is_healthy,
            status: if is_healthy { "healthy".to_string() } else { "warning".to_string() },
            response_time_ms: duration.as_millis() as u64,
            details: Some(json!({
                "total_memory_mb": total_memory,
                "available_memory_mb": available_memory,
                "memory_usage_percent": memory_usage_percent
            })),
            error: if !is_healthy { Some("High memory usage".to_string()) } else { None },
        }
    }
    
    /// 获取可用内存 (MB)
    fn get_available_memory() -> u64 {
        use sysinfo::System;
        let mut sys = System::new_all();
        sys.refresh_memory();
        sys.available_memory() / 1024 / 1024 // 转换为MB
    }
    
    /// 获取总内存 (MB)
    fn get_total_memory() -> u64 {
        use sysinfo::System;
        let mut sys = System::new_all();
        sys.refresh_memory();
        sys.total_memory() / 1024 / 1024 // 转换为MB
    }
    
    /// 获取系统负载平均值
    fn get_load_average() -> serde_json::Value {
        #[cfg(unix)]
        {
            // 尝试从 /proc/loadavg 读取负载信息
            if let Ok(content) = std::fs::read_to_string("/proc/loadavg") {
                let parts: Vec<&str> = content.split_whitespace().collect();
                if parts.len() >= 3 {
                    if let (Ok(one), Ok(five), Ok(fifteen)) = (
                        parts[0].parse::<f64>(),
                        parts[1].parse::<f64>(),
                        parts[2].parse::<f64>(),
                    ) {
                        return json!({
                            "one_minute": one,
                            "five_minutes": five,
                            "fifteen_minutes": fifteen
                        });
                    }
                }
            }
            json!({
                "one_minute": 0.0,
                "five_minutes": 0.0,
                "fifteen_minutes": 0.0,
                "note": "Load average not available"
            })
        }
        #[cfg(not(unix))]
        {
            json!({
                "one_minute": 0.0,
                "five_minutes": 0.0,
                "fifteen_minutes": 0.0,
                "note": "Load average not available on this platform"
            })
        }
    }
    
    /// 获取运行时间信息
    fn get_uptime_info() -> serde_json::Value {
        json!({
            "started_at": "unknown", // 可以在应用启动时记录
            "uptime_seconds": "unknown"
        })
    }
    
    /// 获取系统信息
    fn get_system_info() -> serde_json::Value {
        json!({
            "rust_version": std::env::var("RUSTC_VERSION").unwrap_or_else(|_| "unknown".to_string()),
            "target": std::env::var("TARGET").unwrap_or_else(|_| "unknown".to_string()),
            "profile": if cfg!(debug_assertions) { "debug" } else { "release" },
            "os": std::env::consts::OS,
            "arch": std::env::consts::ARCH
        })
    }
}

impl Default for HealthController {
    fn default() -> Self {
        Self::new()
    }
}

#[async_trait::async_trait]
impl Controller for HealthController {
    fn name(&self) -> &'static str {
        "HealthController"
    }
}

