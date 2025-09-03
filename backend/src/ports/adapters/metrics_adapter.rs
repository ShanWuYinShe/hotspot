//! 指标适配器 - 将现有指标系统适配为 MetricsPort
//! 
//! 提供指标监控与端口架构的集成

use std::future::Future;
use std::pin::Pin;
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use std::collections::HashMap;

use crate::ports::metrics::{
    MetricsPort, MetricsPortResult, 
    Metric, MetricType, MetricValue, PortCallMetrics, CallTracker, Labels
};
use crate::utils::metrics::AppMetrics;

/// 指标适配器
#[derive(Clone)]
pub struct MetricsAdapter {
    metrics: Option<Arc<AppMetrics>>,
}

impl MetricsAdapter {
    pub fn new(metrics: Option<Arc<AppMetrics>>) -> Self {
        Self { metrics }
    }
}

impl MetricsPort for MetricsAdapter {
    fn name(&self) -> &str {
        "MetricsAdapter"
    }

    fn counter<'a>(
        &'a self,
        name: &'a str,
        value: u64,
        _labels: Labels,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<()>> + Send + 'a>> {
        let metrics = self.metrics.clone();
        let name = name.to_string();
        Box::pin(async move {
            if let Some(m) = metrics {
                // 使用现有的指标系统记录计数器
                match name.as_str() {
                    "ws_messages_sent_total" => {
                        for _ in 0..value {
                            m.record_ws_message_sent();
                        }
                    }
                    "ws_messages_received_total" => {
                        for _ in 0..value {
                            m.record_ws_message_received();
                        }
                    }
                    "api_requests_total" => {
                        for _ in 0..value {
                            m.record_http_request("POST", "/api", 200, 0.0);
                        }
                    }
                    _ => {
                        // 对于未知的计数器，记录为 HTTP 请求
                        for _ in 0..value {
                            m.record_http_request("GET", "/unknown", 200, 0.0);
                        }
                    }
                }
            }
            Ok(())
        })
    }

    fn gauge<'a>(
        &'a self,
        name: &'a str,
        _value: f64,
        _labels: Labels,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<()>> + Send + 'a>> {
        let _metrics = self.metrics.clone();
        let _name = name.to_string();
        Box::pin(async move {
            // 简化实现：仪表指标暂时不记录
            tracing::debug!("记录仪表指标: {} = {}", _name, _value);
            Ok(())
        })
    }

    fn histogram<'a>(
        &'a self,
        name: &'a str,
        value: f64,
        _labels: Labels,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<()>> + Send + 'a>> {
        let metrics = self.metrics.clone();
        let name = name.to_string();
        Box::pin(async move {
            if let Some(m) = metrics {
                // 将直方图值记录为位置查询时间
                if name.contains("duration") || name.contains("latency") {
                    m.record_location_query(value);
                }
            }
            Ok(())
        })
    }

    fn summary<'a>(
        &'a self,
        name: &'a str,
        _value: f64,
        _labels: Labels,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<()>> + Send + 'a>> {
        let _metrics = self.metrics.clone();
        let _name = name.to_string();
        Box::pin(async move {
            // 简化实现：摘要指标暂时不记录
            tracing::debug!("记录摘要指标: {} = {}", _name, _value);
            Ok(())
        })
    }

    fn record_port_call_start<'a>(
        &'a self,
        port_name: &'a str,
        method_name: &'a str,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<CallTracker>> + Send + 'a>> {
        let port_name = port_name.to_string();
        let method_name = method_name.to_string();
        Box::pin(async move {
            let tracker = CallTracker::new(port_name, method_name);
            tracing::debug!("开始端口调用跟踪: {}::{}", tracker.port_name, tracker.method_name);
            Ok(tracker)
        })
    }

    fn record_port_call_end<'a>(
        &'a self,
        tracker: CallTracker,
        success: bool,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<()>> + Send + 'a>> {
        let metrics = self.metrics.clone();
        Box::pin(async move {
            let duration_ms = tracker.duration_ms();
            
            if let Some(m) = metrics {
                // 记录 HTTP 请求
                let status = if success { 200 } else { 500 };
                m.record_http_request("POST", &format!("/{}", tracker.port_name), status, duration_ms as f64);
                
                // 根据成功/失败记录不同指标
                if !success {
                    tracing::warn!("端口调用失败: {}::{}, 耗时: {}ms", 
                        tracker.port_name, tracker.method_name, duration_ms);
                }
            }
            
            tracing::debug!("结束端口调用跟踪: {}::{}, 成功: {}, 耗时: {}ms", 
                tracker.port_name, tracker.method_name, success, duration_ms);
            
            Ok(())
        })
    }

    fn get_metrics<'a>(
        &'a self,
        _name_pattern: Option<&'a str>,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<Vec<Metric>>> + Send + 'a>> {
        let metrics = self.metrics.clone();
        Box::pin(async move {
            let mut result = Vec::new();
            
            if let Some(_m) = metrics {
                let timestamp = SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .unwrap_or_default()
                    .as_secs();
                
                // 简化实现：返回基本指标
                result.push(Metric {
                    name: "port_calls_total".to_string(),
                    help: "Total port calls".to_string(),
                    metric_type: MetricType::Counter,
                    value: MetricValue::Counter(0),
                    labels: HashMap::new(),
                    timestamp,
                });
                
                result.push(Metric {
                    name: "port_call_duration_avg".to_string(),
                    help: "Average port call duration".to_string(),
                    metric_type: MetricType::Gauge,
                    value: MetricValue::Gauge(0.0),
                    labels: HashMap::new(),
                    timestamp,
                });
            }
            
            Ok(result)
        })
    }

    fn get_port_call_metrics<'a>(
        &'a self,
        _port_name: Option<&'a str>,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<Vec<PortCallMetrics>>> + Send + 'a>> {
        let metrics = self.metrics.clone();
        Box::pin(async move {
            let mut result = Vec::new();
            
            if let Some(_m) = metrics {
                let timestamp = SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .unwrap_or_default()
                    .as_secs();
                
                // 简化实现：创建一个通用的端口调用指标
                result.push(PortCallMetrics {
                    port_name: "AllPorts".to_string(),
                    method_name: "all_methods".to_string(),
                    call_count: 0,
                    success_count: 0,
                    error_count: 0,
                    total_duration_ms: 0,
                    avg_duration_ms: 0.0,
                    min_duration_ms: 0,
                    max_duration_ms: 0,
                    last_call_time: timestamp,
                });
            }
            
            Ok(result)
        })
    }

    fn cleanup_expired_metrics<'a>(
        &'a self,
        _max_age: Duration,
    ) -> Pin<Box<dyn Future<Output = MetricsPortResult<usize>> + Send + 'a>> {
        Box::pin(async move {
            // 简化实现：暂时不清理过期指标
            tracing::debug!("清理过期指标（简化实现）");
            Ok(0)
        })
    }
}
