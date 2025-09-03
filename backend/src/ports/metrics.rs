//! 指标监控端口定义
//! 
//! 定义指标收集和监控的抽象接口，支持不同的监控实现

use std::future::Future;
use std::pin::Pin;
use std::time::{Duration, Instant};
use std::collections::HashMap;
use thiserror::Error;
use serde::{Serialize, Deserialize};

/// 指标端口错误类型
#[derive(Debug, Error)]
pub enum MetricsPortError {
    #[error("collection error: {0}")]
    Collection(String),
    #[error("storage error: {0}")]
    Storage(String),
    #[error("invalid metric name: {0}")]
    InvalidMetricName(String),
    #[error("invalid metric value: {0}")]
    InvalidMetricValue(String),
    #[error("other error: {0}")]
    Other(String),
}

pub type MetricsPortResult<T> = Result<T, MetricsPortError>;

/// 指标类型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MetricType {
    /// 计数器 - 只增不减的累计值
    Counter,
    /// 仪表 - 可增可减的瞬时值
    Gauge,
    /// 直方图 - 观察值的分布
    Histogram,
    /// 摘要 - 观察值的分位数统计
    Summary,
}

/// 指标值
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MetricValue {
    Counter(u64),
    Gauge(f64),
    Histogram(HistogramData),
    Summary(SummaryData),
}

/// 直方图数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HistogramData {
    pub count: u64,
    pub sum: f64,
    pub buckets: Vec<HistogramBucket>,
}

/// 直方图桶
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HistogramBucket {
    pub upper_bound: f64,
    pub count: u64,
}

/// 摘要数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SummaryData {
    pub count: u64,
    pub sum: f64,
    pub quantiles: Vec<Quantile>,
}

/// 分位数
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Quantile {
    pub quantile: f64,
    pub value: f64,
}

/// 指标标签
pub type Labels = HashMap<String, String>;

/// 指标数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Metric {
    pub name: String,
    pub help: String,
    pub metric_type: MetricType,
    pub value: MetricValue,
    pub labels: Labels,
    pub timestamp: u64,
}

/// 端口调用指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PortCallMetrics {
    /// 端口名称
    pub port_name: String,
    /// 方法名称
    pub method_name: String,
    /// 调用次数
    pub call_count: u64,
    /// 成功次数
    pub success_count: u64,
    /// 失败次数
    pub error_count: u64,
    /// 总耗时（毫秒）
    pub total_duration_ms: u64,
    /// 平均耗时（毫秒）
    pub avg_duration_ms: f64,
    /// 最小耗时（毫秒）
    pub min_duration_ms: u64,
    /// 最大耗时（毫秒）
    pub max_duration_ms: u64,
    /// 最后调用时间
    pub last_call_time: u64,
}

/// 指标监控端口 - 定义指标收集和查询的抽象接口
pub trait MetricsPort: Send + Sync {
    /// 获取端口名称
    fn name(&self) -> &str;

    /// 记录计数器指标
    fn counter<'a>(
        &'a self,
        name: &'a str,
        value: u64,
        labels: Labels,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<()>> + Send + 'a>>;

    /// 记录仪表指标
    fn gauge<'a>(
        &'a self,
        name: &'a str,
        value: f64,
        labels: Labels,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<()>> + Send + 'a>>;

    /// 记录直方图指标
    fn histogram<'a>(
        &'a self,
        name: &'a str,
        value: f64,
        labels: Labels,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<()>> + Send + 'a>>;

    /// 记录摘要指标
    fn summary<'a>(
        &'a self,
        name: &'a str,
        value: f64,
        labels: Labels,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<()>> + Send + 'a>>;

    /// 记录端口调用开始
    fn record_port_call_start<'a>(
        &'a self,
        port_name: &'a str,
        method_name: &'a str,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<CallTracker>> + Send + 'a>>;

    /// 记录端口调用结束
    fn record_port_call_end<'a>(
        &'a self,
        tracker: CallTracker,
        success: bool,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<()>> + Send + 'a>>;

    /// 获取指标数据
    fn get_metrics<'a>(
        &'a self,
        name_pattern: Option<&'a str>,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<Vec<Metric>>> + Send + 'a>>;

    /// 获取端口调用指标
    fn get_port_call_metrics<'a>(
        &'a self,
        port_name: Option<&'a str>,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<Vec<PortCallMetrics>>> + Send + 'a>>;

    /// 清理过期指标
    fn cleanup_expired_metrics<'a>(
        &'a self,
        max_age: Duration,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<usize>> + Send + 'a>>;
}

/// 调用跟踪器 - 用于跟踪端口调用的开始和结束
#[derive(Debug, Clone)]
pub struct CallTracker {
    pub port_name: String,
    pub method_name: String,
    pub start_time: Instant,
    pub labels: Labels,
}

impl CallTracker {
    /// 创建新的调用跟踪器
    pub fn new(port_name: String, method_name: String) -> Self {
        Self {
            port_name,
            method_name,
            start_time: Instant::now(),
            labels: HashMap::new(),
        }
    }

    /// 添加标签
    pub fn with_label(mut self, key: String, value: String) -> Self {
        self.labels.insert(key, value);
        self
    }

    /// 获取耗时（毫秒）
    pub fn duration_ms(&self) -> u64 {
        self.start_time.elapsed().as_millis() as u64
    }
}

impl MetricsPortError {
    /// 从收集错误创建
    pub fn from_collection_error(err: impl std::fmt::Display) -> Self {
        Self::Collection(err.to_string())
    }

    /// 从存储错误创建
    pub fn from_storage_error(err: impl std::fmt::Display) -> Self {
        Self::Storage(err.to_string())
    }

    /// 创建无效指标名称错误
    pub fn invalid_metric_name(name: impl Into<String>) -> Self {
        Self::InvalidMetricName(name.into())
    }

    /// 创建无效指标值错误
    pub fn invalid_metric_value(msg: impl Into<String>) -> Self {
        Self::InvalidMetricValue(msg.into())
    }
}

/// 指标工具函数
pub struct MetricsUtils;

impl MetricsUtils {
    /// 创建端口调用标签
    pub fn port_call_labels(port_name: &str, method_name: &str, success: bool) -> Labels {
        let mut labels = HashMap::new();
        labels.insert("port".to_string(), port_name.to_string());
        labels.insert("method".to_string(), method_name.to_string());
        labels.insert("status".to_string(), if success { "success" } else { "error" }.to_string());
        labels
    }

    /// 创建性能标签
    pub fn performance_labels(category: &str, operation: &str) -> Labels {
        let mut labels = HashMap::new();
        labels.insert("category".to_string(), category.to_string());
        labels.insert("operation".to_string(), operation.to_string());
        labels
    }

    /// 创建资源使用标签
    pub fn resource_labels(resource_type: &str, resource_name: &str) -> Labels {
        let mut labels = HashMap::new();
        labels.insert("type".to_string(), resource_type.to_string());
        labels.insert("name".to_string(), resource_name.to_string());
        labels
    }

    /// 验证指标名称
    pub fn validate_metric_name(name: &str) -> Result<(), MetricsPortError> {
        if name.is_empty() {
            return Err(MetricsPortError::invalid_metric_name("Metric name cannot be empty"));
        }
        if name.len() > 255 {
            return Err(MetricsPortError::invalid_metric_name("Metric name too long"));
        }
        if !name.chars().all(|c| c.is_alphanumeric() || c == '_' || c == ':') {
            return Err(MetricsPortError::invalid_metric_name("Invalid characters in metric name"));
        }
        Ok(())
    }
}
