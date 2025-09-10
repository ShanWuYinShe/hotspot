/// APM (Application Performance Monitoring) 服务
/// 
/// 提供全面的应用性能监控，包括请求追踪、性能分析、错误监控和系统指标收集
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use async_trait::async_trait;
use serde::{Serialize, Deserialize};
use tracing::{info, warn, debug, error};
use tokio::sync::RwLock;
use uuid::Uuid;

use crate::domain::DomainResult;

/// APM配置
#[derive(Debug, Clone)]
pub struct ApmConfig {
    /// 启用APM监控
    pub enable_apm: bool,
    /// 采样率 (0.0-1.0)
    pub sampling_rate: f64,
    /// 慢请求阈值（毫秒）
    pub slow_request_threshold_ms: u64,
    /// 错误请求阈值（毫秒）
    pub error_request_threshold_ms: u64,
    /// 追踪保留时间（秒）
    pub trace_retention_seconds: u64,
    /// 最大追踪数量
    pub max_traces: usize,
    /// 启用数据库监控
    pub enable_db_monitoring: bool,
    /// 启用缓存监控
    pub enable_cache_monitoring: bool,
    /// 启用外部服务监控
    pub enable_external_monitoring: bool,
}

impl Default for ApmConfig {
    fn default() -> Self {
        Self {
            enable_apm: true,
            sampling_rate: 1.0, // 100%采样，生产环境可调低
            slow_request_threshold_ms: 1000, // 1秒
            error_request_threshold_ms: 5000, // 5秒
            trace_retention_seconds: 3600, // 1小时
            max_traces: 10000,
            enable_db_monitoring: true,
            enable_cache_monitoring: true,
            enable_external_monitoring: true,
        }
    }
}

/// 追踪上下文
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TraceContext {
    /// 追踪ID
    pub trace_id: String,
    /// 跨度ID
    pub span_id: String,
    /// 父跨度ID
    pub parent_span_id: Option<String>,
    /// 操作名称
    pub operation_name: String,
    /// 开始时间
    pub start_time: u64,
    /// 结束时间
    pub end_time: Option<u64>,
    /// 持续时间（微秒）
    pub duration_micros: Option<u64>,
    /// 状态
    pub status: TraceStatus,
    /// 标签
    pub tags: HashMap<String, String>,
    /// 日志
    pub logs: Vec<TraceLog>,
    /// 错误信息
    pub error: Option<TraceError>,
}

impl TraceContext {
    /// 创建新的追踪上下文
    pub fn new(operation_name: String) -> Self {
        Self {
            trace_id: Uuid::new_v4().to_string(),
            span_id: Uuid::new_v4().to_string(),
            parent_span_id: None,
            operation_name,
            start_time: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap_or_default()
                .as_micros() as u64,
            end_time: None,
            duration_micros: None,
            status: TraceStatus::Ok,
            tags: HashMap::new(),
            logs: Vec::new(),
            error: None,
        }
    }

    /// 创建子跨度
    pub fn create_child_span(&self, operation_name: String) -> Self {
        Self {
            trace_id: self.trace_id.clone(),
            span_id: Uuid::new_v4().to_string(),
            parent_span_id: Some(self.span_id.clone()),
            operation_name,
            start_time: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap_or_default()
                .as_micros() as u64,
            end_time: None,
            duration_micros: None,
            status: TraceStatus::Ok,
            tags: HashMap::new(),
            logs: Vec::new(),
            error: None,
        }
    }

    /// 添加标签
    pub fn add_tag(&mut self, key: String, value: String) {
        self.tags.insert(key, value);
    }

    /// 添加日志
    pub fn add_log(&mut self, level: LogLevel, message: String) {
        self.logs.push(TraceLog {
            timestamp: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap_or_default()
                .as_micros() as u64,
            level,
            message,
            fields: HashMap::new(),
        });
    }

    /// 设置错误
    pub fn set_error(&mut self, error: TraceError) {
        self.status = TraceStatus::Error;
        self.error = Some(error);
    }

    /// 完成跨度
    pub fn finish(&mut self) {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_micros() as u64;
        self.end_time = Some(now);
        self.duration_micros = Some(now - self.start_time);
    }

    /// 获取持续时间（毫秒）
    pub fn duration_ms(&self) -> Option<f64> {
        self.duration_micros.map(|micros| micros as f64 / 1000.0)
    }
}

/// 追踪状态
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum TraceStatus {
    /// 正常
    Ok,
    /// 错误
    Error,
    /// 超时
    Timeout,
    /// 取消
    Cancelled,
}

/// 追踪日志
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TraceLog {
    /// 时间戳（微秒）
    pub timestamp: u64,
    /// 日志级别
    pub level: LogLevel,
    /// 消息
    pub message: String,
    /// 字段
    pub fields: HashMap<String, String>,
}

/// 日志级别
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum LogLevel {
    Debug,
    Info,
    Warn,
    Error,
}

/// 追踪错误
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TraceError {
    /// 错误类型
    pub error_type: String,
    /// 错误消息
    pub message: String,
    /// 堆栈跟踪
    pub stack_trace: Option<String>,
}

/// 性能指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceMetrics {
    /// 请求总数
    pub total_requests: u64,
    /// 成功请求数
    pub successful_requests: u64,
    /// 错误请求数
    pub error_requests: u64,
    /// 平均响应时间（毫秒）
    pub avg_response_time_ms: f64,
    /// P50响应时间（毫秒）
    pub p50_response_time_ms: f64,
    /// P90响应时间（毫秒）
    pub p90_response_time_ms: f64,
    /// P95响应时间（毫秒）
    pub p95_response_time_ms: f64,
    /// P99响应时间（毫秒）
    pub p99_response_time_ms: f64,
    /// 吞吐量（请求/秒）
    pub throughput_rps: f64,
    /// 错误率
    pub error_rate: f64,
    /// 慢请求数
    pub slow_requests: u64,
    /// 时间窗口（秒）
    pub time_window_seconds: u64,
}

/// 数据库性能指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseMetrics {
    /// 查询总数
    pub total_queries: u64,
    /// 慢查询数
    pub slow_queries: u64,
    /// 平均查询时间（毫秒）
    pub avg_query_time_ms: f64,
    /// 连接池状态
    pub connection_pool_status: ConnectionPoolStatus,
    /// 查询类型分布
    pub query_type_distribution: HashMap<String, u64>,
}

/// 连接池状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConnectionPoolStatus {
    /// 活跃连接数
    pub active_connections: u32,
    /// 空闲连接数
    pub idle_connections: u32,
    /// 最大连接数
    pub max_connections: u32,
    /// 等待连接数
    pub waiting_connections: u32,
}

/// 缓存性能指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CacheMetrics {
    /// 缓存命中数
    pub hits: u64,
    /// 缓存未命中数
    pub misses: u64,
    /// 命中率
    pub hit_rate: f64,
    /// 平均响应时间（毫秒）
    pub avg_response_time_ms: f64,
    /// 缓存大小
    pub cache_size: u64,
    /// 内存使用量（字节）
    pub memory_usage_bytes: u64,
}

/// 系统资源指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemResourceMetrics {
    /// CPU使用率（百分比）
    pub cpu_usage_percent: f64,
    /// 内存使用率（百分比）
    pub memory_usage_percent: f64,
    /// 磁盘使用率（百分比）
    pub disk_usage_percent: f64,
    /// 网络IO（字节/秒）
    pub network_io_bytes_per_sec: f64,
    /// 文件描述符使用数
    pub file_descriptors_used: u32,
    /// 线程数
    pub thread_count: u32,
}

/// APM监控服务接口
#[async_trait]
pub trait ApmService: Send + Sync {
    /// 开始追踪
    async fn start_trace(&self, operation_name: String) -> DomainResult<TraceContext>;
    
    /// 完成追踪
    async fn finish_trace(&self, trace: TraceContext) -> DomainResult<()>;
    
    /// 记录数据库操作
    async fn record_database_operation(
        &self,
        trace_id: &str,
        query: &str,
        duration_ms: u64,
        success: bool,
    ) -> DomainResult<()>;
    
    /// 记录缓存操作
    async fn record_cache_operation(
        &self,
        trace_id: &str,
        operation: &str,
        key: &str,
        hit: bool,
        duration_ms: u64,
    ) -> DomainResult<()>;
    
    /// 记录外部服务调用
    async fn record_external_service_call(
        &self,
        trace_id: &str,
        service_name: &str,
        endpoint: &str,
        duration_ms: u64,
        status_code: u16,
    ) -> DomainResult<()>;
    
    /// 获取性能指标
    async fn get_performance_metrics(&self, time_window_seconds: u64) -> DomainResult<PerformanceMetrics>;
    
    /// 获取数据库指标
    async fn get_database_metrics(&self) -> DomainResult<DatabaseMetrics>;
    
    /// 获取缓存指标
    async fn get_cache_metrics(&self) -> DomainResult<CacheMetrics>;
    
    /// 获取系统资源指标
    async fn get_system_metrics(&self) -> DomainResult<SystemResourceMetrics>;
    
    /// 获取追踪详情
    async fn get_trace_details(&self, trace_id: &str) -> DomainResult<Option<TraceContext>>;
    
    /// 搜索追踪
    async fn search_traces(
        &self,
        query: TraceSearchQuery,
    ) -> DomainResult<Vec<TraceContext>>;
    
    /// 清理过期追踪
    async fn cleanup_expired_traces(&self) -> DomainResult<u64>;
}

/// 追踪搜索查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TraceSearchQuery {
    /// 操作名称
    pub operation_name: Option<String>,
    /// 最小持续时间（毫秒）
    pub min_duration_ms: Option<u64>,
    /// 最大持续时间（毫秒）
    pub max_duration_ms: Option<u64>,
    /// 状态
    pub status: Option<TraceStatus>,
    /// 开始时间
    pub start_time: Option<u64>,
    /// 结束时间
    pub end_time: Option<u64>,
    /// 标签过滤
    pub tags: HashMap<String, String>,
    /// 限制数量
    pub limit: Option<usize>,
}

/// 内存APM服务实现
pub struct InMemoryApmService {
    config: ApmConfig,
    traces: Arc<RwLock<HashMap<String, TraceContext>>>,
    performance_data: Arc<RwLock<Vec<TraceContext>>>,
}

impl InMemoryApmService {
    /// 创建新的内存APM服务
    pub fn new(config: ApmConfig) -> Self {
        let service = Self {
            config,
            traces: Arc::new(RwLock::new(HashMap::new())),
            performance_data: Arc::new(RwLock::new(Vec::new())),
        };

        // 启动清理任务
        if service.config.enable_apm {
            service.start_cleanup_task();
        }

        service
    }

    /// 启动清理任务
    fn start_cleanup_task(&self) {
        let traces = Arc::clone(&self.traces);
        let performance_data = Arc::clone(&self.performance_data);
        let retention_seconds = self.config.trace_retention_seconds;
        let max_traces = self.config.max_traces;

        tokio::spawn(async move {
            let mut interval = tokio::time::interval(Duration::from_secs(300)); // 5分钟清理一次
            loop {
                interval.tick().await;

                // 清理过期追踪
                Self::cleanup_expired_data(&traces, &performance_data, retention_seconds, max_traces).await;

                debug!("APM cleanup task completed");
            }
        });
    }

    /// 清理过期数据
    async fn cleanup_expired_data(
        traces: &Arc<RwLock<HashMap<String, TraceContext>>>,
        performance_data: &Arc<RwLock<Vec<TraceContext>>>,
        retention_seconds: u64,
        max_traces: usize,
    ) {
        let cutoff_time = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_micros() as u64
            - (retention_seconds * 1_000_000); // 转换为微秒

        // 清理过期追踪
        let mut traces_guard = traces.write().await;
        traces_guard.retain(|_, trace| trace.start_time >= cutoff_time);

        // 清理过期性能数据
        let mut perf_guard = performance_data.write().await;
        perf_guard.retain(|trace| trace.start_time >= cutoff_time);

        // 限制数据量
        if perf_guard.len() > max_traces {
            let excess = perf_guard.len() - max_traces;
            perf_guard.drain(0..excess);
        }
    }

    /// 检查是否应该采样
    fn should_sample(&self) -> bool {
        if !self.config.enable_apm {
            return false;
        }
        
        use rand::Rng;
        let mut rng = rand::rng();
        rng.random::<f64>() < self.config.sampling_rate
    }

    /// 计算百分位数
    fn calculate_percentile(values: &mut [f64], percentile: f64) -> f64 {
        if values.is_empty() {
            return 0.0;
        }
        
        values.sort_by(|a, b| a.partial_cmp(b).unwrap());
        let index = ((values.len() - 1) as f64 * percentile) as usize;
        values[index.min(values.len() - 1)]
    }
}

#[async_trait]
impl ApmService for InMemoryApmService {
    async fn start_trace(&self, operation_name: String) -> DomainResult<TraceContext> {
        if !self.should_sample() {
            // 返回一个禁用的追踪上下文
            let mut trace = TraceContext::new(operation_name);
            trace.add_tag("sampled".to_string(), "false".to_string());
            return Ok(trace);
        }

        let trace = TraceContext::new(operation_name);
        
        // 存储追踪上下文
        let mut traces = self.traces.write().await;
        traces.insert(trace.trace_id.clone(), trace.clone());

        debug!(
            trace_id = %trace.trace_id,
            operation = %trace.operation_name,
            "Started APM trace"
        );

        Ok(trace)
    }

    async fn finish_trace(&self, mut trace: TraceContext) -> DomainResult<()> {
        trace.finish();

        // 检查是否为慢请求
        if let Some(duration_ms) = trace.duration_ms() {
            if duration_ms > self.config.slow_request_threshold_ms as f64 {
                warn!(
                    trace_id = %trace.trace_id,
                    operation = %trace.operation_name,
                    duration_ms = duration_ms,
                    "Slow request detected"
                );
                trace.add_tag("slow_request".to_string(), "true".to_string());
            }

            if duration_ms > self.config.error_request_threshold_ms as f64 {
                error!(
                    trace_id = %trace.trace_id,
                    operation = %trace.operation_name,
                    duration_ms = duration_ms,
                    "Very slow request detected"
                );
                trace.add_tag("very_slow_request".to_string(), "true".to_string());
            }
        }

        // 更新追踪数据
        let mut traces = self.traces.write().await;
        traces.insert(trace.trace_id.clone(), trace.clone());

        // 添加到性能数据
        let mut performance_data = self.performance_data.write().await;
        performance_data.push(trace.clone());

        debug!(
            trace_id = %trace.trace_id,
            duration_ms = trace.duration_ms(),
            "Finished APM trace"
        );

        Ok(())
    }

    async fn record_database_operation(
        &self,
        trace_id: &str,
        query: &str,
        duration_ms: u64,
        success: bool,
    ) -> DomainResult<()> {
        if !self.config.enable_db_monitoring {
            return Ok(());
        }

        let mut traces = self.traces.write().await;
        if let Some(trace) = traces.get_mut(trace_id) {
            trace.add_tag("db.query".to_string(), query.to_string());
            trace.add_tag("db.duration_ms".to_string(), duration_ms.to_string());
            trace.add_tag("db.success".to_string(), success.to_string());

            if duration_ms > 100 { // 慢查询阈值
                trace.add_log(
                    LogLevel::Warn,
                    format!("Slow database query: {}ms", duration_ms),
                );
            }

            if !success {
                trace.add_log(
                    LogLevel::Error,
                    "Database operation failed".to_string(),
                );
            }
        }

        debug!(
            trace_id = trace_id,
            query = query,
            duration_ms = duration_ms,
            success = success,
            "Recorded database operation"
        );

        Ok(())
    }

    async fn record_cache_operation(
        &self,
        trace_id: &str,
        operation: &str,
        key: &str,
        hit: bool,
        duration_ms: u64,
    ) -> DomainResult<()> {
        if !self.config.enable_cache_monitoring {
            return Ok(());
        }

        let mut traces = self.traces.write().await;
        if let Some(trace) = traces.get_mut(trace_id) {
            trace.add_tag("cache.operation".to_string(), operation.to_string());
            trace.add_tag("cache.key".to_string(), key.to_string());
            trace.add_tag("cache.hit".to_string(), hit.to_string());
            trace.add_tag("cache.duration_ms".to_string(), duration_ms.to_string());

            trace.add_log(
                LogLevel::Debug,
                format!("Cache {} {}: {}", operation, if hit { "hit" } else { "miss" }, key),
            );
        }

        debug!(
            trace_id = trace_id,
            operation = operation,
            key = key,
            hit = hit,
            duration_ms = duration_ms,
            "Recorded cache operation"
        );

        Ok(())
    }

    async fn record_external_service_call(
        &self,
        trace_id: &str,
        service_name: &str,
        endpoint: &str,
        duration_ms: u64,
        status_code: u16,
    ) -> DomainResult<()> {
        if !self.config.enable_external_monitoring {
            return Ok(());
        }

        let mut traces = self.traces.write().await;
        if let Some(trace) = traces.get_mut(trace_id) {
            trace.add_tag("external.service".to_string(), service_name.to_string());
            trace.add_tag("external.endpoint".to_string(), endpoint.to_string());
            trace.add_tag("external.duration_ms".to_string(), duration_ms.to_string());
            trace.add_tag("external.status_code".to_string(), status_code.to_string());

            let log_level = if status_code >= 400 {
                LogLevel::Error
            } else if duration_ms > 1000 {
                LogLevel::Warn
            } else {
                LogLevel::Info
            };

            trace.add_log(
                log_level,
                format!("External service call: {} {} ({}ms, status: {})", 
                    service_name, endpoint, duration_ms, status_code),
            );
        }

        debug!(
            trace_id = trace_id,
            service = service_name,
            endpoint = endpoint,
            duration_ms = duration_ms,
            status_code = status_code,
            "Recorded external service call"
        );

        Ok(())
    }

    async fn get_performance_metrics(&self, time_window_seconds: u64) -> DomainResult<PerformanceMetrics> {
        let cutoff_time = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_micros() as u64
            - (time_window_seconds * 1_000_000);

        let performance_data = self.performance_data.read().await;
        let recent_traces: Vec<_> = performance_data
            .iter()
            .filter(|trace| trace.start_time >= cutoff_time && trace.end_time.is_some())
            .collect();

        let total_requests = recent_traces.len() as u64;
        let successful_requests = recent_traces
            .iter()
            .filter(|trace| trace.status == TraceStatus::Ok)
            .count() as u64;
        let error_requests = total_requests - successful_requests;

        let mut durations: Vec<f64> = recent_traces
            .iter()
            .filter_map(|trace| trace.duration_ms())
            .collect();

        let avg_response_time_ms = if !durations.is_empty() {
            durations.iter().sum::<f64>() / durations.len() as f64
        } else {
            0.0
        };

        let p50_response_time_ms = Self::calculate_percentile(&mut durations, 0.5);
        let p90_response_time_ms = Self::calculate_percentile(&mut durations, 0.9);
        let p95_response_time_ms = Self::calculate_percentile(&mut durations, 0.95);
        let p99_response_time_ms = Self::calculate_percentile(&mut durations, 0.99);

        let throughput_rps = if time_window_seconds > 0 {
            total_requests as f64 / time_window_seconds as f64
        } else {
            0.0
        };

        let error_rate = if total_requests > 0 {
            error_requests as f64 / total_requests as f64
        } else {
            0.0
        };

        let slow_requests = durations
            .iter()
            .filter(|&&duration| duration > self.config.slow_request_threshold_ms as f64)
            .count() as u64;

        Ok(PerformanceMetrics {
            total_requests,
            successful_requests,
            error_requests,
            avg_response_time_ms,
            p50_response_time_ms,
            p90_response_time_ms,
            p95_response_time_ms,
            p99_response_time_ms,
            throughput_rps,
            error_rate,
            slow_requests,
            time_window_seconds,
        })
    }

    async fn get_database_metrics(&self) -> DomainResult<DatabaseMetrics> {
        // 简化实现，实际应该从数据库连接池获取真实指标
        Ok(DatabaseMetrics {
            total_queries: 1000,
            slow_queries: 10,
            avg_query_time_ms: 25.5,
            connection_pool_status: ConnectionPoolStatus {
                active_connections: 5,
                idle_connections: 15,
                max_connections: 20,
                waiting_connections: 0,
            },
            query_type_distribution: {
                let mut dist = HashMap::new();
                dist.insert("SELECT".to_string(), 800);
                dist.insert("INSERT".to_string(), 150);
                dist.insert("UPDATE".to_string(), 40);
                dist.insert("DELETE".to_string(), 10);
                dist
            },
        })
    }

    async fn get_cache_metrics(&self) -> DomainResult<CacheMetrics> {
        // 简化实现，实际应该从缓存服务获取真实指标
        Ok(CacheMetrics {
            hits: 8500,
            misses: 1500,
            hit_rate: 0.85,
            avg_response_time_ms: 2.3,
            cache_size: 50000,
            memory_usage_bytes: 1024 * 1024 * 128, // 128MB
        })
    }

    async fn get_system_metrics(&self) -> DomainResult<SystemResourceMetrics> {
        // 简化实现，实际应该从系统监控获取真实指标
        Ok(SystemResourceMetrics {
            cpu_usage_percent: 45.2,
            memory_usage_percent: 68.7,
            disk_usage_percent: 23.1,
            network_io_bytes_per_sec: 1024.0 * 512.0, // 512KB/s
            file_descriptors_used: 256,
            thread_count: 32,
        })
    }

    async fn get_trace_details(&self, trace_id: &str) -> DomainResult<Option<TraceContext>> {
        let traces = self.traces.read().await;
        Ok(traces.get(trace_id).cloned())
    }

    async fn search_traces(&self, query: TraceSearchQuery) -> DomainResult<Vec<TraceContext>> {
        let traces = self.traces.read().await;
        let mut results: Vec<TraceContext> = traces
            .values()
            .filter(|trace| {
                // 操作名称过滤
                if let Some(ref op_name) = query.operation_name {
                    if !trace.operation_name.contains(op_name) {
                        return false;
                    }
                }

                // 持续时间过滤
                if let Some(duration_ms) = trace.duration_ms() {
                    if let Some(min_duration) = query.min_duration_ms {
                        if duration_ms < min_duration as f64 {
                            return false;
                        }
                    }
                    if let Some(max_duration) = query.max_duration_ms {
                        if duration_ms > max_duration as f64 {
                            return false;
                        }
                    }
                }

                // 状态过滤
                if let Some(ref status) = query.status {
                    if trace.status != *status {
                        return false;
                    }
                }

                // 时间范围过滤
                if let Some(start_time) = query.start_time {
                    if trace.start_time < start_time {
                        return false;
                    }
                }
                if let Some(end_time) = query.end_time {
                    if let Some(trace_end_time) = trace.end_time {
                        if trace_end_time > end_time {
                            return false;
                        }
                    }
                }

                // 标签过滤
                for (key, value) in &query.tags {
                    if trace.tags.get(key) != Some(value) {
                        return false;
                    }
                }

                true
            })
            .cloned()
            .collect();

        // 按开始时间排序（最新的在前）
        results.sort_by(|a, b| b.start_time.cmp(&a.start_time));

        // 限制结果数量
        if let Some(limit) = query.limit {
            results.truncate(limit);
        }

        Ok(results)
    }

    async fn cleanup_expired_traces(&self) -> DomainResult<u64> {
        let cutoff_time = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_micros() as u64
            - (self.config.trace_retention_seconds * 1_000_000);

        let mut cleaned_count = 0u64;

        // 清理追踪
        let mut traces = self.traces.write().await;
        let original_len = traces.len();
        traces.retain(|_, trace| trace.start_time >= cutoff_time);
        cleaned_count += (original_len - traces.len()) as u64;

        // 清理性能数据
        let mut performance_data = self.performance_data.write().await;
        let original_len = performance_data.len();
        performance_data.retain(|trace| trace.start_time >= cutoff_time);
        cleaned_count += (original_len - performance_data.len()) as u64;

        info!(cleaned_count = cleaned_count, "Cleaned up expired APM traces");
        Ok(cleaned_count)
    }
}

/// 空APM服务（用于禁用APM监控）
pub struct NullApmService;

#[async_trait]
impl ApmService for NullApmService {
    async fn start_trace(&self, operation_name: String) -> DomainResult<TraceContext> {
        Ok(TraceContext::new(operation_name))
    }

    async fn finish_trace(&self, _trace: TraceContext) -> DomainResult<()> {
        Ok(())
    }

    async fn record_database_operation(
        &self,
        _trace_id: &str,
        _query: &str,
        _duration_ms: u64,
        _success: bool,
    ) -> DomainResult<()> {
        Ok(())
    }

    async fn record_cache_operation(
        &self,
        _trace_id: &str,
        _operation: &str,
        _key: &str,
        _hit: bool,
        _duration_ms: u64,
    ) -> DomainResult<()> {
        Ok(())
    }

    async fn record_external_service_call(
        &self,
        _trace_id: &str,
        _service_name: &str,
        _endpoint: &str,
        _duration_ms: u64,
        _status_code: u16,
    ) -> DomainResult<()> {
        Ok(())
    }

    async fn get_performance_metrics(&self, time_window_seconds: u64) -> DomainResult<PerformanceMetrics> {
        Ok(PerformanceMetrics {
            total_requests: 0,
            successful_requests: 0,
            error_requests: 0,
            avg_response_time_ms: 0.0,
            p50_response_time_ms: 0.0,
            p90_response_time_ms: 0.0,
            p95_response_time_ms: 0.0,
            p99_response_time_ms: 0.0,
            throughput_rps: 0.0,
            error_rate: 0.0,
            slow_requests: 0,
            time_window_seconds,
        })
    }

    async fn get_database_metrics(&self) -> DomainResult<DatabaseMetrics> {
        Ok(DatabaseMetrics {
            total_queries: 0,
            slow_queries: 0,
            avg_query_time_ms: 0.0,
            connection_pool_status: ConnectionPoolStatus {
                active_connections: 0,
                idle_connections: 0,
                max_connections: 0,
                waiting_connections: 0,
            },
            query_type_distribution: HashMap::new(),
        })
    }

    async fn get_cache_metrics(&self) -> DomainResult<CacheMetrics> {
        Ok(CacheMetrics {
            hits: 0,
            misses: 0,
            hit_rate: 0.0,
            avg_response_time_ms: 0.0,
            cache_size: 0,
            memory_usage_bytes: 0,
        })
    }

    async fn get_system_metrics(&self) -> DomainResult<SystemResourceMetrics> {
        Ok(SystemResourceMetrics {
            cpu_usage_percent: 0.0,
            memory_usage_percent: 0.0,
            disk_usage_percent: 0.0,
            network_io_bytes_per_sec: 0.0,
            file_descriptors_used: 0,
            thread_count: 0,
        })
    }

    async fn get_trace_details(&self, _trace_id: &str) -> DomainResult<Option<TraceContext>> {
        Ok(None)
    }

    async fn search_traces(&self, _query: TraceSearchQuery) -> DomainResult<Vec<TraceContext>> {
        Ok(Vec::new())
    }

    async fn cleanup_expired_traces(&self) -> DomainResult<u64> {
        Ok(0)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_apm_service() {
        let config = ApmConfig::default();
        let service = InMemoryApmService::new(config);

        // 测试追踪
        let mut trace = service.start_trace("test_operation".to_string()).await.unwrap();
        trace.add_tag("user_id".to_string(), "123".to_string());
        trace.add_log(LogLevel::Info, "Test log message".to_string());

        // 模拟一些操作
        tokio::time::sleep(Duration::from_millis(10)).await;

        service.finish_trace(trace.clone()).await.unwrap();

        // 测试获取追踪详情
        let details = service.get_trace_details(&trace.trace_id).await.unwrap();
        assert!(details.is_some());
        assert_eq!(details.unwrap().operation_name, "test_operation");

        // 测试性能指标
        let metrics = service.get_performance_metrics(3600).await.unwrap();
        assert_eq!(metrics.total_requests, 1);
    }

    #[test]
    fn test_trace_context() {
        let mut trace = TraceContext::new("test_op".to_string());
        trace.add_tag("key".to_string(), "value".to_string());
        trace.add_log(LogLevel::Info, "Test message".to_string());
        
        let child = trace.create_child_span("child_op".to_string());
        assert_eq!(child.trace_id, trace.trace_id);
        assert_eq!(child.parent_span_id, Some(trace.span_id.clone()));

        trace.finish();
        assert!(trace.end_time.is_some());
        assert!(trace.duration_micros.is_some());
    }
}
