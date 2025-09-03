//! 数据库查询性能监控工具
//!
//! 提供查询执行时间监控、慢查询检测和性能分析功能

use std::time::{Duration, Instant};
use tracing::{info, warn, error};

/// 查询性能监控器
pub struct QueryMonitor {
    query_name: String,
    start_time: Instant,
    slow_query_threshold: Duration,
}

impl QueryMonitor {
    /// 创建新的查询监控器
    pub fn new(query_name: &str) -> Self {
        Self {
            query_name: query_name.to_string(),
            start_time: Instant::now(),
            slow_query_threshold: Duration::from_millis(1000), // 默认1秒为慢查询阈值
        }
    }
    
    /// 设置慢查询阈值
    pub fn with_slow_threshold(mut self, threshold: Duration) -> Self {
        self.slow_query_threshold = threshold;
        self
    }
    
    /// 完成查询监控并记录性能指标
    pub fn finish(self) -> Duration {
        let duration = self.start_time.elapsed();
        
        if duration >= self.slow_query_threshold {
            warn!(
                "慢查询检测: {} 耗时 {}ms (阈值: {}ms)",
                self.query_name,
                duration.as_millis(),
                self.slow_query_threshold.as_millis()
            );
        } else {
            info!(
                "查询完成: {} 耗时 {}ms",
                self.query_name,
                duration.as_millis()
            );
        }
        
        duration
    }
    
    /// 记录查询错误
    pub fn error(self, err: &dyn std::error::Error) {
        let duration = self.start_time.elapsed();
        error!(
            "查询失败: {} 耗时 {}ms, 错误: {}",
            self.query_name,
            duration.as_millis(),
            err
        );
    }
}

/// 查询性能统计
#[derive(Debug, Clone)]
pub struct QueryStats {
    pub total_queries: u64,
    pub slow_queries: u64,
    pub avg_duration_ms: f64,
    pub max_duration_ms: u64,
    pub min_duration_ms: u64,
}

impl Default for QueryStats {
    fn default() -> Self {
        Self {
            total_queries: 0,
            slow_queries: 0,
            avg_duration_ms: 0.0,
            max_duration_ms: 0,
            min_duration_ms: u64::MAX,
        }
    }
}

impl QueryStats {
    /// 更新统计信息
    pub fn update(&mut self, duration: Duration, is_slow: bool) {
        let duration_ms = duration.as_millis() as u64;
        
        self.total_queries += 1;
        if is_slow {
            self.slow_queries += 1;
        }
        
        // 更新最大最小值
        self.max_duration_ms = self.max_duration_ms.max(duration_ms);
        if self.min_duration_ms == u64::MAX {
            self.min_duration_ms = duration_ms;
        } else {
            self.min_duration_ms = self.min_duration_ms.min(duration_ms);
        }
        
        // 更新平均值
        self.avg_duration_ms = (self.avg_duration_ms * (self.total_queries - 1) as f64 + duration_ms as f64) / self.total_queries as f64;
    }
    
    /// 获取慢查询比例
    pub fn slow_query_ratio(&self) -> f64 {
        if self.total_queries == 0 {
            0.0
        } else {
            self.slow_queries as f64 / self.total_queries as f64
        }
    }
}

/// 查询性能监控宏
#[macro_export]
macro_rules! monitor_query {
    ($query_name:expr, $query_block:block) => {{
        let monitor = $crate::utils::query_monitor::QueryMonitor::new($query_name);
        let result = $query_block;
        match &result {
            Ok(_) => {
                monitor.finish();
            }
            Err(e) => {
                monitor.error(e);
            }
        }
        result
    }};
    
    ($query_name:expr, $threshold:expr, $query_block:block) => {{
        let monitor = $crate::utils::query_monitor::QueryMonitor::new($query_name)
            .with_slow_threshold($threshold);
        let result = $query_block;
        match &result {
            Ok(_) => {
                monitor.finish();
            }
            Err(e) => {
                monitor.error(e);
            }
        }
        result
    }};
}
