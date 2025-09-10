/// 业务指标收集服务
/// 
/// 提供全面的业务指标收集、聚合和监控功能
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use async_trait::async_trait;
use serde::{Serialize, Deserialize};
use tracing::{info, warn, debug};
use tokio::sync::RwLock;

use crate::domain::DomainResult;

/// 业务指标配置
#[derive(Debug, Clone)]
pub struct MetricsConfig {
    /// 启用指标收集
    pub enable_metrics: bool,
    /// 指标聚合间隔（秒）
    pub aggregation_interval_seconds: u64,
    /// 指标保留时间（秒）
    pub retention_seconds: u64,
    /// 最大指标数量
    pub max_metrics_count: usize,
    /// 启用实时指标
    pub enable_realtime_metrics: bool,
    /// 慢操作阈值（毫秒）
    pub slow_operation_threshold_ms: u64,
}

impl Default for MetricsConfig {
    fn default() -> Self {
        Self {
            enable_metrics: true,
            aggregation_interval_seconds: 60, // 1分钟
            retention_seconds: 86400, // 24小时
            max_metrics_count: 10000,
            enable_realtime_metrics: true,
            slow_operation_threshold_ms: 1000, // 1秒
        }
    }
}

/// 指标类型
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum MetricType {
    /// 计数器（累计值）
    Counter,
    /// 计量器（瞬时值）
    Gauge,
    /// 直方图（分布统计）
    Histogram,
    /// 摘要（百分位数）
    Summary,
}

/// 业务指标数据点
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetricDataPoint {
    /// 指标名称
    pub name: String,
    /// 指标类型
    pub metric_type: MetricType,
    /// 指标值
    pub value: f64,
    /// 标签
    pub labels: HashMap<String, String>,
    /// 时间戳
    pub timestamp: u64,
    /// 描述
    pub description: Option<String>,
}

impl MetricDataPoint {
    /// 创建新的指标数据点
    pub fn new(name: String, metric_type: MetricType, value: f64) -> Self {
        Self {
            name,
            metric_type,
            value,
            labels: HashMap::new(),
            timestamp: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap_or_default()
                .as_secs(),
            description: None,
        }
    }

    /// 添加标签
    pub fn with_label(mut self, key: String, value: String) -> Self {
        self.labels.insert(key, value);
        self
    }

    /// 添加描述
    pub fn with_description(mut self, description: String) -> Self {
        self.description = Some(description);
        self
    }

    /// 生成指标键
    pub fn metric_key(&self) -> String {
        let mut key = self.name.clone();
        if !self.labels.is_empty() {
            let mut labels: Vec<_> = self.labels.iter().collect();
            labels.sort_by_key(|(k, _)| *k);
            let label_str = labels
                .iter()
                .map(|(k, v)| format!("{}={}", k, v))
                .collect::<Vec<_>>()
                .join(",");
            key.push_str(&format!("{{{}}}", label_str));
        }
        key
    }
}

/// 聚合指标统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AggregatedMetrics {
    /// 指标名称
    pub name: String,
    /// 指标类型
    pub metric_type: MetricType,
    /// 数据点数量
    pub count: u64,
    /// 最小值
    pub min: f64,
    /// 最大值
    pub max: f64,
    /// 平均值
    pub avg: f64,
    /// 总和
    pub sum: f64,
    /// 标准差
    pub std_dev: f64,
    /// 百分位数（P50, P90, P95, P99）
    pub percentiles: HashMap<String, f64>,
    /// 时间范围
    pub time_range: (u64, u64),
    /// 标签
    pub labels: HashMap<String, String>,
}

/// 业务事件类型
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum BusinessEventType {
    /// 用户相关事件
    UserRegistration,
    UserLogin,
    UserLogout,
    UserProfileUpdate,
    UserLocationUpdate,
    
    /// 群组相关事件
    GroupCreation,
    GroupJoin,
    GroupLeave,
    GroupMessageSent,
    
    /// 消息相关事件
    DirectMessageSent,
    MessageRead,
    MessageDeleted,
    
    /// 瞬间相关事件
    MomentCreated,
    MomentLiked,
    MomentCommented,
    MomentDeleted,
    
    /// 搜索相关事件
    UserSearch,
    GroupSearch,
    LocationSearch,
    GlobalSearch,
    
    /// 系统事件
    ApiRequest,
    DatabaseQuery,
    CacheHit,
    CacheMiss,
    ErrorOccurred,
}

/// 业务事件数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BusinessEvent {
    /// 事件类型
    pub event_type: BusinessEventType,
    /// 用户ID（如果适用）
    pub user_id: Option<i64>,
    /// 资源ID（如群组ID、消息ID等）
    pub resource_id: Option<i64>,
    /// 事件属性
    pub properties: HashMap<String, String>,
    /// 执行时间（毫秒）
    pub duration_ms: Option<u64>,
    /// 时间戳
    pub timestamp: u64,
    /// 成功标志
    pub success: bool,
    /// 错误信息（如果失败）
    pub error_message: Option<String>,
}

impl BusinessEvent {
    /// 创建新的业务事件
    pub fn new(event_type: BusinessEventType) -> Self {
        Self {
            event_type,
            user_id: None,
            resource_id: None,
            properties: HashMap::new(),
            duration_ms: None,
            timestamp: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap_or_default()
                .as_secs(),
            success: true,
            error_message: None,
        }
    }

    /// 设置用户ID
    pub fn with_user_id(mut self, user_id: i64) -> Self {
        self.user_id = Some(user_id);
        self
    }

    /// 设置资源ID
    pub fn with_resource_id(mut self, resource_id: i64) -> Self {
        self.resource_id = Some(resource_id);
        self
    }

    /// 添加属性
    pub fn with_property(mut self, key: String, value: String) -> Self {
        self.properties.insert(key, value);
        self
    }

    /// 设置执行时间
    pub fn with_duration(mut self, duration_ms: u64) -> Self {
        self.duration_ms = Some(duration_ms);
        self
    }

    /// 标记为失败
    pub fn with_error(mut self, error_message: String) -> Self {
        self.success = false;
        self.error_message = Some(error_message);
        self
    }
}

/// 实时指标快照
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RealtimeMetricsSnapshot {
    /// 快照时间
    pub timestamp: u64,
    /// 活跃用户数
    pub active_users: u64,
    /// 在线用户数
    pub online_users: u64,
    /// 每分钟API请求数
    pub requests_per_minute: f64,
    /// 平均响应时间（毫秒）
    pub avg_response_time_ms: f64,
    /// 错误率
    pub error_rate: f64,
    /// 数据库查询数
    pub database_queries: u64,
    /// 缓存命中率
    pub cache_hit_rate: f64,
    /// 系统负载指标
    pub system_metrics: SystemMetrics,
}

/// 系统指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemMetrics {
    /// CPU使用率
    pub cpu_usage_percent: f64,
    /// 内存使用率
    pub memory_usage_percent: f64,
    /// 磁盘使用率
    pub disk_usage_percent: f64,
    /// 网络IO（字节/秒）
    pub network_io_bytes_per_sec: f64,
}

/// 业务指标服务接口
#[async_trait]
pub trait MetricsService: Send + Sync {
    /// 记录指标数据点
    async fn record_metric(&self, metric: MetricDataPoint) -> DomainResult<()>;
    
    /// 记录业务事件
    async fn record_event(&self, event: BusinessEvent) -> DomainResult<()>;
    
    /// 获取聚合指标
    async fn get_aggregated_metrics(
        &self,
        name: &str,
        start_time: u64,
        end_time: u64,
    ) -> DomainResult<Vec<AggregatedMetrics>>;
    
    /// 获取实时指标快照
    async fn get_realtime_snapshot(&self) -> DomainResult<RealtimeMetricsSnapshot>;
    
    /// 获取业务事件统计
    async fn get_event_statistics(
        &self,
        event_type: BusinessEventType,
        start_time: u64,
        end_time: u64,
    ) -> DomainResult<EventStatistics>;
    
    /// 清理过期指标
    async fn cleanup_expired_metrics(&self) -> DomainResult<u64>;
    
    /// 获取热门指标
    async fn get_top_metrics(&self, limit: usize) -> DomainResult<Vec<String>>;
}

/// 事件统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EventStatistics {
    /// 事件类型
    pub event_type: BusinessEventType,
    /// 总事件数
    pub total_events: u64,
    /// 成功事件数
    pub successful_events: u64,
    /// 失败事件数
    pub failed_events: u64,
    /// 成功率
    pub success_rate: f64,
    /// 平均执行时间（毫秒）
    pub avg_duration_ms: f64,
    /// 每小时事件分布
    pub hourly_distribution: HashMap<u8, u64>,
    /// 用户分布（前10名）
    pub top_users: Vec<(i64, u64)>,
    /// 时间范围
    pub time_range: (u64, u64),
}

/// 内存指标服务实现
pub struct InMemoryMetricsService {
    config: MetricsConfig,
    metrics: Arc<RwLock<HashMap<String, Vec<MetricDataPoint>>>>,
    events: Arc<RwLock<Vec<BusinessEvent>>>,
    realtime_stats: Arc<RwLock<RealtimeMetricsSnapshot>>,
}

impl InMemoryMetricsService {
    /// 创建新的内存指标服务
    pub fn new(config: MetricsConfig) -> Self {
        let service = Self {
            config,
            metrics: Arc::new(RwLock::new(HashMap::new())),
            events: Arc::new(RwLock::new(Vec::new())),
            realtime_stats: Arc::new(RwLock::new(RealtimeMetricsSnapshot {
                timestamp: SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .unwrap_or_default()
                    .as_secs(),
                active_users: 0,
                online_users: 0,
                requests_per_minute: 0.0,
                avg_response_time_ms: 0.0,
                error_rate: 0.0,
                database_queries: 0,
                cache_hit_rate: 0.0,
                system_metrics: SystemMetrics {
                    cpu_usage_percent: 0.0,
                    memory_usage_percent: 0.0,
                    disk_usage_percent: 0.0,
                    network_io_bytes_per_sec: 0.0,
                },
            })),
        };

        // 启动后台聚合任务
        if service.config.enable_metrics {
            service.start_aggregation_task();
        }

        service
    }

    /// 启动指标聚合任务
    fn start_aggregation_task(&self) {
        let metrics = Arc::clone(&self.metrics);
        let events = Arc::clone(&self.events);
        let realtime_stats = Arc::clone(&self.realtime_stats);
        let interval = Duration::from_secs(self.config.aggregation_interval_seconds);
        let retention = Duration::from_secs(self.config.retention_seconds);

        tokio::spawn(async move {
            let mut interval_timer = tokio::time::interval(interval);
            loop {
                interval_timer.tick().await;

                // 清理过期指标
                Self::cleanup_expired_data(&metrics, &events, retention).await;

                // 更新实时统计
                Self::update_realtime_stats(&metrics, &events, &realtime_stats).await;

                debug!("Metrics aggregation task completed");
            }
        });
    }

    /// 清理过期数据
    async fn cleanup_expired_data(
        metrics: &Arc<RwLock<HashMap<String, Vec<MetricDataPoint>>>>,
        events: &Arc<RwLock<Vec<BusinessEvent>>>,
        retention: Duration,
    ) {
        let cutoff_time = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs()
            .saturating_sub(retention.as_secs());

        // 清理过期指标
        let mut metrics_guard = metrics.write().await;
        for (_, data_points) in metrics_guard.iter_mut() {
            data_points.retain(|point| point.timestamp >= cutoff_time);
        }
        metrics_guard.retain(|_, data_points| !data_points.is_empty());

        // 清理过期事件
        let mut events_guard = events.write().await;
        events_guard.retain(|event| event.timestamp >= cutoff_time);
    }

    /// 更新实时统计
    async fn update_realtime_stats(
        _metrics: &Arc<RwLock<HashMap<String, Vec<MetricDataPoint>>>>,
        events: &Arc<RwLock<Vec<BusinessEvent>>>,
        realtime_stats: &Arc<RwLock<RealtimeMetricsSnapshot>>,
    ) {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs();

        let one_minute_ago = now.saturating_sub(60);

        // 计算最近一分钟的统计
        let events_guard = events.read().await;
        let recent_events: Vec<_> = events_guard
            .iter()
            .filter(|event| event.timestamp >= one_minute_ago)
            .collect();

        let api_requests = recent_events
            .iter()
            .filter(|event| event.event_type == BusinessEventType::ApiRequest)
            .count() as f64;

        let avg_response_time = if !recent_events.is_empty() {
            recent_events
                .iter()
                .filter_map(|event| event.duration_ms)
                .map(|d| d as f64)
                .sum::<f64>()
                / recent_events.len() as f64
        } else {
            0.0
        };

        let error_count = recent_events
            .iter()
            .filter(|event| !event.success)
            .count() as f64;

        let error_rate = if !recent_events.is_empty() {
            error_count / recent_events.len() as f64
        } else {
            0.0
        };

        // 更新实时统计
        let mut stats = realtime_stats.write().await;
        stats.timestamp = now;
        stats.requests_per_minute = api_requests;
        stats.avg_response_time_ms = avg_response_time;
        stats.error_rate = error_rate;

        info!(
            requests_per_minute = api_requests,
            avg_response_time_ms = avg_response_time,
            error_rate = error_rate,
            "Updated realtime metrics"
        );
    }

    /// 计算聚合指标
    fn calculate_aggregated_metrics(
        data_points: &[MetricDataPoint],
        start_time: u64,
        end_time: u64,
    ) -> Option<AggregatedMetrics> {
        if data_points.is_empty() {
            return None;
        }

        let filtered_points: Vec<_> = data_points
            .iter()
            .filter(|point| point.timestamp >= start_time && point.timestamp <= end_time)
            .collect();

        if filtered_points.is_empty() {
            return None;
        }

        let values: Vec<f64> = filtered_points.iter().map(|point| point.value).collect();
        let count = values.len() as u64;
        let sum = values.iter().sum::<f64>();
        let min = values.iter().fold(f64::INFINITY, |a, &b| a.min(b));
        let max = values.iter().fold(f64::NEG_INFINITY, |a, &b| a.max(b));
        let avg = sum / count as f64;

        // 计算标准差
        let variance = values.iter().map(|&x| (x - avg).powi(2)).sum::<f64>() / count as f64;
        let std_dev = variance.sqrt();

        // 计算百分位数
        let mut sorted_values = values.clone();
        sorted_values.sort_by(|a, b| a.partial_cmp(b).unwrap());
        let mut percentiles = HashMap::new();
        percentiles.insert("p50".to_string(), Self::percentile(&sorted_values, 0.5));
        percentiles.insert("p90".to_string(), Self::percentile(&sorted_values, 0.9));
        percentiles.insert("p95".to_string(), Self::percentile(&sorted_values, 0.95));
        percentiles.insert("p99".to_string(), Self::percentile(&sorted_values, 0.99));

        Some(AggregatedMetrics {
            name: filtered_points[0].name.clone(),
            metric_type: filtered_points[0].metric_type.clone(),
            count,
            min,
            max,
            avg,
            sum,
            std_dev,
            percentiles,
            time_range: (start_time, end_time),
            labels: filtered_points[0].labels.clone(),
        })
    }

    /// 计算百分位数
    fn percentile(sorted_values: &[f64], percentile: f64) -> f64 {
        if sorted_values.is_empty() {
            return 0.0;
        }

        let index = (percentile * (sorted_values.len() - 1) as f64) as usize;
        sorted_values[index.min(sorted_values.len() - 1)]
    }
}

#[async_trait]
impl MetricsService for InMemoryMetricsService {
    async fn record_metric(&self, metric: MetricDataPoint) -> DomainResult<()> {
        if !self.config.enable_metrics {
            return Ok(());
        }

        let key = metric.metric_key();
        let mut metrics = self.metrics.write().await;
        
        let data_points = metrics.entry(key.clone()).or_insert_with(Vec::new);
        data_points.push(metric.clone());

        // 限制数据点数量
        if data_points.len() > self.config.max_metrics_count {
            data_points.drain(0..data_points.len() - self.config.max_metrics_count);
        }

        debug!(
            metric_name = %metric.name,
            metric_type = ?metric.metric_type,
            value = metric.value,
            "Recorded metric"
        );

        Ok(())
    }

    async fn record_event(&self, event: BusinessEvent) -> DomainResult<()> {
        if !self.config.enable_metrics {
            return Ok(());
        }

        let mut events = self.events.write().await;
        events.push(event.clone());

        // 限制事件数量
        if events.len() > self.config.max_metrics_count {
            let excess = events.len() - self.config.max_metrics_count;
            events.drain(0..excess);
        }

        // 记录慢操作
        if let Some(duration) = event.duration_ms {
            if duration > self.config.slow_operation_threshold_ms {
                warn!(
                    event_type = ?event.event_type,
                    duration_ms = duration,
                    user_id = event.user_id,
                    "Slow operation detected"
                );
            }
        }

        debug!(
            event_type = ?event.event_type,
            user_id = event.user_id,
            success = event.success,
            "Recorded business event"
        );

        Ok(())
    }

    async fn get_aggregated_metrics(
        &self,
        name: &str,
        start_time: u64,
        end_time: u64,
    ) -> DomainResult<Vec<AggregatedMetrics>> {
        let metrics = self.metrics.read().await;
        let mut results = Vec::new();

        for (key, data_points) in metrics.iter() {
            if key.starts_with(name) {
                if let Some(aggregated) = Self::calculate_aggregated_metrics(data_points, start_time, end_time) {
                    results.push(aggregated);
                }
            }
        }

        Ok(results)
    }

    async fn get_realtime_snapshot(&self) -> DomainResult<RealtimeMetricsSnapshot> {
        let stats = self.realtime_stats.read().await;
        Ok(stats.clone())
    }

    async fn get_event_statistics(
        &self,
        event_type: BusinessEventType,
        start_time: u64,
        end_time: u64,
    ) -> DomainResult<EventStatistics> {
        let events = self.events.read().await;
        let filtered_events: Vec<_> = events
            .iter()
            .filter(|event| {
                event.event_type == event_type
                    && event.timestamp >= start_time
                    && event.timestamp <= end_time
            })
            .collect();

        let total_events = filtered_events.len() as u64;
        let successful_events = filtered_events.iter().filter(|event| event.success).count() as u64;
        let failed_events = total_events - successful_events;
        let success_rate = if total_events > 0 {
            successful_events as f64 / total_events as f64
        } else {
            0.0
        };

        let avg_duration_ms = if !filtered_events.is_empty() {
            filtered_events
                .iter()
                .filter_map(|event| event.duration_ms)
                .map(|d| d as f64)
                .sum::<f64>()
                / filtered_events.len() as f64
        } else {
            0.0
        };

        // 计算每小时分布
        let mut hourly_distribution = HashMap::new();
        for event in &filtered_events {
            let hour = ((event.timestamp % 86400) / 3600) as u8;
            *hourly_distribution.entry(hour).or_insert(0) += 1;
        }

        // 计算用户分布
        let mut user_counts: HashMap<i64, u64> = HashMap::new();
        for event in &filtered_events {
            if let Some(user_id) = event.user_id {
                *user_counts.entry(user_id).or_insert(0) += 1;
            }
        }
        let mut top_users: Vec<_> = user_counts.into_iter().collect();
        top_users.sort_by(|a, b| b.1.cmp(&a.1));
        top_users.truncate(10);

        Ok(EventStatistics {
            event_type,
            total_events,
            successful_events,
            failed_events,
            success_rate,
            avg_duration_ms,
            hourly_distribution,
            top_users,
            time_range: (start_time, end_time),
        })
    }

    async fn cleanup_expired_metrics(&self) -> DomainResult<u64> {
        let retention = Duration::from_secs(self.config.retention_seconds);
        let cutoff_time = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs()
            .saturating_sub(retention.as_secs());

        let mut cleaned_count = 0u64;

        // 清理指标
        let mut metrics = self.metrics.write().await;
        for (_, data_points) in metrics.iter_mut() {
            let original_len = data_points.len();
            data_points.retain(|point| point.timestamp >= cutoff_time);
            cleaned_count += (original_len - data_points.len()) as u64;
        }
        metrics.retain(|_, data_points| !data_points.is_empty());

        // 清理事件
        let mut events = self.events.write().await;
        let original_len = events.len();
        events.retain(|event| event.timestamp >= cutoff_time);
        cleaned_count += (original_len - events.len()) as u64;

        info!(cleaned_count = cleaned_count, "Cleaned up expired metrics");
        Ok(cleaned_count)
    }

    async fn get_top_metrics(&self, limit: usize) -> DomainResult<Vec<String>> {
        let metrics = self.metrics.read().await;
        let mut metric_counts: Vec<_> = metrics
            .iter()
            .map(|(name, data_points)| (name.clone(), data_points.len()))
            .collect();

        metric_counts.sort_by(|a, b| b.1.cmp(&a.1));
        metric_counts.truncate(limit);

        Ok(metric_counts.into_iter().map(|(name, _)| name).collect())
    }
}

/// 空指标服务（用于禁用指标收集）
pub struct NullMetricsService;

#[async_trait]
impl MetricsService for NullMetricsService {
    async fn record_metric(&self, _metric: MetricDataPoint) -> DomainResult<()> {
        Ok(())
    }

    async fn record_event(&self, _event: BusinessEvent) -> DomainResult<()> {
        Ok(())
    }

    async fn get_aggregated_metrics(
        &self,
        _name: &str,
        _start_time: u64,
        _end_time: u64,
    ) -> DomainResult<Vec<AggregatedMetrics>> {
        Ok(Vec::new())
    }

    async fn get_realtime_snapshot(&self) -> DomainResult<RealtimeMetricsSnapshot> {
        Ok(RealtimeMetricsSnapshot {
            timestamp: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap_or_default()
                .as_secs(),
            active_users: 0,
            online_users: 0,
            requests_per_minute: 0.0,
            avg_response_time_ms: 0.0,
            error_rate: 0.0,
            database_queries: 0,
            cache_hit_rate: 0.0,
            system_metrics: SystemMetrics {
                cpu_usage_percent: 0.0,
                memory_usage_percent: 0.0,
                disk_usage_percent: 0.0,
                network_io_bytes_per_sec: 0.0,
            },
        })
    }

    async fn get_event_statistics(
        &self,
        event_type: BusinessEventType,
        _start_time: u64,
        _end_time: u64,
    ) -> DomainResult<EventStatistics> {
        Ok(EventStatistics {
            event_type,
            total_events: 0,
            successful_events: 0,
            failed_events: 0,
            success_rate: 0.0,
            avg_duration_ms: 0.0,
            hourly_distribution: HashMap::new(),
            top_users: Vec::new(),
            time_range: (0, 0),
        })
    }

    async fn cleanup_expired_metrics(&self) -> DomainResult<u64> {
        Ok(0)
    }

    async fn get_top_metrics(&self, _limit: usize) -> DomainResult<Vec<String>> {
        Ok(Vec::new())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_metrics_service() {
        let config = MetricsConfig::default();
        let service = InMemoryMetricsService::new(config);

        // 测试记录指标
        let metric = MetricDataPoint::new(
            "test_counter".to_string(),
            MetricType::Counter,
            1.0,
        )
        .with_label("service".to_string(), "test".to_string());

        service.record_metric(metric).await.unwrap();

        // 测试记录事件
        let event = BusinessEvent::new(BusinessEventType::UserLogin)
            .with_user_id(123)
            .with_duration(50);

        service.record_event(event).await.unwrap();

        // 测试获取聚合指标
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();
        let metrics = service
            .get_aggregated_metrics("test_counter", now - 3600, now)
            .await
            .unwrap();

        assert!(!metrics.is_empty());

        // 测试获取实时快照
        let snapshot = service.get_realtime_snapshot().await.unwrap();
        assert!(snapshot.timestamp > 0);

        // 测试获取事件统计
        let stats = service
            .get_event_statistics(BusinessEventType::UserLogin, now - 3600, now)
            .await
            .unwrap();

        assert_eq!(stats.total_events, 1);
        assert_eq!(stats.successful_events, 1);
    }

    #[test]
    fn test_metric_data_point() {
        let metric = MetricDataPoint::new(
            "test_metric".to_string(),
            MetricType::Gauge,
            42.0,
        )
        .with_label("env".to_string(), "test".to_string())
        .with_description("Test metric".to_string());

        assert_eq!(metric.name, "test_metric");
        assert_eq!(metric.value, 42.0);
        assert_eq!(metric.labels.get("env"), Some(&"test".to_string()));
        assert_eq!(metric.description, Some("Test metric".to_string()));

        let key = metric.metric_key();
        assert!(key.contains("test_metric"));
        assert!(key.contains("env=test"));
    }

    #[test]
    fn test_business_event() {
        let event = BusinessEvent::new(BusinessEventType::GroupCreation)
            .with_user_id(456)
            .with_resource_id(789)
            .with_property("name".to_string(), "Test Group".to_string())
            .with_duration(200);

        assert_eq!(event.event_type, BusinessEventType::GroupCreation);
        assert_eq!(event.user_id, Some(456));
        assert_eq!(event.resource_id, Some(789));
        assert_eq!(event.duration_ms, Some(200));
        assert!(event.success);
        assert_eq!(
            event.properties.get("name"),
            Some(&"Test Group".to_string())
        );
    }
}
