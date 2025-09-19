/// 增强监控系统
/// 
/// 功能职责：
/// - 业务指标收集和分析
/// - 性能监控和告警
/// - 系统健康检查
/// - 自动化运维支持
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;
use serde::{Serialize, Deserialize};
use tracing::{info, warn, debug, error};

use crate::domain::DomainResult;
use crate::domain::DomainError;
use crate::infrastructure::{
    QueryPerformanceOptimizer,
    monitoring::MetricsCollector
};
use crate::ws::WebSocketPerformanceOptimizer;

/// 增强监控系统
#[derive(Clone)]
pub struct EnhancedMonitoringSystem {
    business_metrics: Arc<RwLock<BusinessMetrics>>,
    system_metrics: Arc<RwLock<SystemMetrics>>,
    alert_manager: Arc<AlertManager>,
    config: MonitoringConfig,
    query_optimizer: Option<Arc<QueryPerformanceOptimizer>>,
    ws_optimizer: Option<Arc<WebSocketPerformanceOptimizer>>,
}

/// 业务指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BusinessMetrics {
    /// 用户指标
    pub user_metrics: UserMetrics,
    /// 群组指标
    pub group_metrics: GroupMetrics,
    /// 消息指标
    pub message_metrics: MessageMetrics,
    /// 生活瞬间指标
    pub moment_metrics: MomentMetrics,
    /// 最后更新时间
    pub last_updated: chrono::DateTime<chrono::Utc>,
}

/// 用户指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserMetrics {
    pub total_users: u64,
    pub active_users_1d: u64,
    pub active_users_7d: u64,
    pub active_users_30d: u64,
    pub new_registrations_1d: u64,
    pub temp_users: u64,
    pub user_retention_rate: f64,
}

/// 群组指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupMetrics {
    pub total_groups: u64,
    pub active_groups_1d: u64,
    pub new_groups_1d: u64,
    pub average_group_size: f64,
    pub largest_group_size: u64,
    pub group_engagement_rate: f64,
}

/// 消息指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageMetrics {
    pub total_messages_1d: u64,
    pub group_messages_1d: u64,
    pub direct_messages_1d: u64,
    pub messages_per_user: f64,
    pub peak_message_rate: f64,
    pub message_delivery_rate: f64,
}

/// 生活瞬间指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MomentMetrics {
    pub total_moments_1d: u64,
    pub moments_with_location: u64,
    pub moments_with_images: u64,
    pub average_likes_per_moment: f64,
    pub moment_engagement_rate: f64,
}

/// 系统指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemMetrics {
    /// 数据库性能
    pub database_performance: DatabasePerformanceMetrics,
    /// WebSocket性能
    pub websocket_performance: WebSocketPerformanceMetrics,
    /// API性能
    pub api_performance: ApiPerformanceMetrics,
    /// 资源使用
    pub resource_usage: ResourceUsageMetrics,
    /// 最后更新时间
    pub last_updated: chrono::DateTime<chrono::Utc>,
}

/// 数据库性能指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabasePerformanceMetrics {
    pub connection_pool_utilization: f64,
    pub average_query_time_ms: f64,
    pub slow_queries_count: u64,
    pub failed_queries_count: u64,
    pub total_queries_1h: u64,
}

/// WebSocket性能指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebSocketPerformanceMetrics {
    pub active_connections: usize,
    pub messages_per_second: f64,
    pub connection_success_rate: f64,
    pub average_latency_ms: f64,
    pub memory_usage_mb: f64,
}

/// API性能指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiPerformanceMetrics {
    pub requests_per_second: f64,
    pub average_response_time_ms: f64,
    pub error_rate: f64,
    pub p95_response_time_ms: f64,
    pub total_requests_1h: u64,
}

/// 资源使用指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceUsageMetrics {
    pub cpu_usage_percent: f64,
    pub memory_usage_mb: f64,
    pub memory_usage_percent: f64,
    pub disk_usage_percent: f64,
    pub network_io_mbps: f64,
}

/// 监控配置
#[derive(Debug, Clone)]
pub struct MonitoringConfig {
    pub collection_interval: Duration,
    pub retention_days: u32,
    pub alert_thresholds: AlertThresholds,
    pub enable_detailed_logging: bool,
}

/// 告警阈值
#[derive(Debug, Clone)]
pub struct AlertThresholds {
    pub cpu_usage_critical: f64,
    pub memory_usage_critical: f64,
    pub error_rate_critical: f64,
    pub response_time_critical: f64,
    pub connection_failure_rate_critical: f64,
}

impl Default for MonitoringConfig {
    fn default() -> Self {
        Self {
            collection_interval: Duration::from_secs(60),
            retention_days: 30,
            alert_thresholds: AlertThresholds {
                cpu_usage_critical: 80.0,
                memory_usage_critical: 85.0,
                error_rate_critical: 5.0,
                response_time_critical: 1000.0,
                connection_failure_rate_critical: 10.0,
            },
            enable_detailed_logging: true,
        }
    }
}

/// 告警管理器
#[derive(Clone)]
pub struct AlertManager {
    active_alerts: Arc<RwLock<HashMap<String, Alert>>>,
    alert_history: Arc<RwLock<Vec<Alert>>>,
    config: AlertConfig,
}

/// 告警信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Alert {
    pub id: String,
    pub alert_type: AlertType,
    pub severity: AlertSeverity,
    pub title: String,
    pub description: String,
    pub metric_value: f64,
    pub threshold: f64,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub resolved_at: Option<chrono::DateTime<chrono::Utc>>,
    pub is_resolved: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AlertType {
    Performance,
    Resource,
    Business,
    Security,
    System,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AlertSeverity {
    Info,
    Warning,
    Critical,
    Emergency,
}

/// 告警配置
#[derive(Debug, Clone)]
pub struct AlertConfig {
    pub max_active_alerts: usize,
    pub alert_cooldown: Duration,
    pub enable_notifications: bool,
}

impl Default for AlertConfig {
    fn default() -> Self {
        Self {
            max_active_alerts: 100,
            alert_cooldown: Duration::from_secs(300), // 5分钟
            enable_notifications: true,
        }
    }
}

impl EnhancedMonitoringSystem {
    /// 创建新的增强监控系统
    pub fn new(config: MonitoringConfig) -> Self {
        Self {
            business_metrics: Arc::new(RwLock::new(BusinessMetrics::default())),
            system_metrics: Arc::new(RwLock::new(SystemMetrics::default())),
            alert_manager: Arc::new(AlertManager::new(AlertConfig::default())),
            config,
            query_optimizer: None,
            ws_optimizer: None,
        }
    }

    /// 设置查询优化器
    pub fn with_query_optimizer(mut self, optimizer: Arc<QueryPerformanceOptimizer>) -> Self {
        self.query_optimizer = Some(optimizer);
        self
    }

    /// 设置WebSocket优化器
    pub fn with_websocket_optimizer(mut self, optimizer: Arc<WebSocketPerformanceOptimizer>) -> Self {
        self.ws_optimizer = Some(optimizer);
        self
    }

    /// 启动监控
    pub async fn start_monitoring(&self) {
        let system = self.clone();
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(system.config.collection_interval);
            
            loop {
                interval.tick().await;
                if let Err(e) = system.collect_all_metrics().await {
                    error!("收集监控指标失败: {:?}", e);
                }
            }
        });

        info!("增强监控系统已启动，收集间隔: {:?}", self.config.collection_interval);
    }

    /// 收集所有指标
    async fn collect_all_metrics(&self) -> DomainResult<()> {
        // 收集业务指标
        self.collect_business_metrics().await?;
        
        // 收集系统指标
        self.collect_system_metrics().await?;
        
        // 检查告警条件
        self.check_alert_conditions().await?;
        
        Ok(())
    }

    /// 收集业务指标
    async fn collect_business_metrics(&self) -> DomainResult<()> {
        // 这里应该从数据库查询实际的业务指标
        // 为了演示，使用模拟数据
        let mut metrics = self.business_metrics.write().await;
        
        metrics.user_metrics = UserMetrics {
            total_users: 1000, // 从数据库查询
            active_users_1d: 150,
            active_users_7d: 400,
            active_users_30d: 800,
            new_registrations_1d: 10,
            temp_users: 50,
            user_retention_rate: 75.5,
        };
        
        metrics.group_metrics = GroupMetrics {
            total_groups: 200,
            active_groups_1d: 50,
            new_groups_1d: 5,
            average_group_size: 8.5,
            largest_group_size: 150,
            group_engagement_rate: 65.2,
        };
        
        metrics.message_metrics = MessageMetrics {
            total_messages_1d: 5000,
            group_messages_1d: 3500,
            direct_messages_1d: 1500,
            messages_per_user: 33.3,
            peak_message_rate: 50.0,
            message_delivery_rate: 99.8,
        };
        
        metrics.moment_metrics = MomentMetrics {
            total_moments_1d: 300,
            moments_with_location: 280,
            moments_with_images: 200,
            average_likes_per_moment: 5.2,
            moment_engagement_rate: 45.8,
        };
        
        metrics.last_updated = chrono::Utc::now();
        
        debug!("业务指标已更新");
        Ok(())
    }

    /// 收集系统指标
    async fn collect_system_metrics(&self) -> DomainResult<()> {
        let mut metrics = self.system_metrics.write().await;
        
        // 收集数据库性能指标
        if let Some(query_optimizer) = &self.query_optimizer {
            let pool_health = query_optimizer.check_connection_pool_health().await?;
            let query_stats = query_optimizer.get_query_stats().await;
            
            metrics.database_performance = DatabasePerformanceMetrics {
                connection_pool_utilization: pool_health.utilization_percent,
                average_query_time_ms: 25.0, // 从查询统计计算
                slow_queries_count: query_stats.values().map(|s| s.slow_query_count).sum(),
                failed_queries_count: 0, // 需要从统计中获取
                total_queries_1h: query_stats.values().map(|s| s.execution_count).sum(),
            };
        } else {
            metrics.database_performance = DatabasePerformanceMetrics {
                connection_pool_utilization: 0.0,
                average_query_time_ms: 0.0,
                slow_queries_count: 0,
                failed_queries_count: 0,
                total_queries_1h: 0,
            };
        }
        
        // 收集WebSocket性能指标
        if let Some(ws_optimizer) = &self.ws_optimizer {
            let ws_stats = ws_optimizer.get_performance_stats().await;
            let connection_health = ws_optimizer.get_connection_health().await;
            
            metrics.websocket_performance = WebSocketPerformanceMetrics {
                active_connections: connection_health.total_connections,
                messages_per_second: ws_stats.message_stats.messages_per_second,
                connection_success_rate: connection_health.connection_success_rate,
                average_latency_ms: connection_health.average_latency_ms,
                memory_usage_mb: ws_stats.memory_stats.total_memory_mb,
            };
        } else {
            metrics.websocket_performance = WebSocketPerformanceMetrics {
                active_connections: 0,
                messages_per_second: 0.0,
                connection_success_rate: 0.0,
                average_latency_ms: 0.0,
                memory_usage_mb: 0.0,
            };
        }
        
        // 收集API性能指标（模拟数据）
        metrics.api_performance = ApiPerformanceMetrics {
            requests_per_second: 100.0,
            average_response_time_ms: 150.0,
            error_rate: 0.5,
            p95_response_time_ms: 300.0,
            total_requests_1h: 360000,
        };
        
        // 收集资源使用指标（模拟数据）
        metrics.resource_usage = ResourceUsageMetrics {
            cpu_usage_percent: 45.0,
            memory_usage_mb: 512.0,
            memory_usage_percent: 60.0,
            disk_usage_percent: 35.0,
            network_io_mbps: 10.5,
        };
        
        metrics.last_updated = chrono::Utc::now();
        
        debug!("系统指标已更新");
        Ok(())
    }

    /// 检查告警条件
    async fn check_alert_conditions(&self) -> DomainResult<()> {
        let system_metrics = self.system_metrics.read().await;
        let thresholds = &self.config.alert_thresholds;
        
        // 检查CPU使用率
        if system_metrics.resource_usage.cpu_usage_percent > thresholds.cpu_usage_critical {
            self.alert_manager.create_alert(
                AlertType::Resource,
                AlertSeverity::Critical,
                "CPU使用率过高".to_string(),
                format!("CPU使用率 {:.1}% 超过阈值 {:.1}%", 
                    system_metrics.resource_usage.cpu_usage_percent,
                    thresholds.cpu_usage_critical
                ),
                system_metrics.resource_usage.cpu_usage_percent,
                thresholds.cpu_usage_critical,
            ).await;
        }
        
        // 检查内存使用率
        if system_metrics.resource_usage.memory_usage_percent > thresholds.memory_usage_critical {
            self.alert_manager.create_alert(
                AlertType::Resource,
                AlertSeverity::Critical,
                "内存使用率过高".to_string(),
                format!("内存使用率 {:.1}% 超过阈值 {:.1}%", 
                    system_metrics.resource_usage.memory_usage_percent,
                    thresholds.memory_usage_critical
                ),
                system_metrics.resource_usage.memory_usage_percent,
                thresholds.memory_usage_critical,
            ).await;
        }
        
        // 检查API错误率
        if system_metrics.api_performance.error_rate > thresholds.error_rate_critical {
            self.alert_manager.create_alert(
                AlertType::Performance,
                AlertSeverity::Critical,
                "API错误率过高".to_string(),
                format!("API错误率 {:.1}% 超过阈值 {:.1}%", 
                    system_metrics.api_performance.error_rate,
                    thresholds.error_rate_critical
                ),
                system_metrics.api_performance.error_rate,
                thresholds.error_rate_critical,
            ).await;
        }
        
        Ok(())
    }

    /// 获取业务指标
    pub async fn get_business_metrics(&self) -> BusinessMetrics {
        self.business_metrics.read().await.clone()
    }

    /// 获取系统指标
    pub async fn get_system_metrics(&self) -> SystemMetrics {
        self.system_metrics.read().await.clone()
    }

    /// 获取活跃告警
    pub async fn get_active_alerts(&self) -> Vec<Alert> {
        self.alert_manager.get_active_alerts().await
    }

    /// 生成监控报告
    pub async fn generate_monitoring_report(&self) -> MonitoringReport {
        let business_metrics = self.get_business_metrics().await;
        let system_metrics = self.get_system_metrics().await;
        let active_alerts = self.get_active_alerts().await;
        
        MonitoringReport {
            business_metrics,
            system_metrics,
            active_alerts,
            generated_at: chrono::Utc::now(),
        }
    }
}

impl AlertManager {
    pub fn new(config: AlertConfig) -> Self {
        Self {
            active_alerts: Arc::new(RwLock::new(HashMap::new())),
            alert_history: Arc::new(RwLock::new(Vec::new())),
            config,
        }
    }

    pub async fn create_alert(
        &self,
        alert_type: AlertType,
        severity: AlertSeverity,
        title: String,
        description: String,
        metric_value: f64,
        threshold: f64,
    ) {
        let alert_id = format!("{:?}_{}", alert_type, chrono::Utc::now().timestamp());
        
        // 检查是否已存在相同类型的活跃告警
        {
            let active_alerts = self.active_alerts.read().await;
            if active_alerts.contains_key(&alert_id) {
                return; // 避免重复告警
            }
        }
        
        let alert = Alert {
            id: alert_id.clone(),
            alert_type,
            severity,
            title: title.clone(),
            description,
            metric_value,
            threshold,
            created_at: chrono::Utc::now(),
            resolved_at: None,
            is_resolved: false,
        };
        
        // 添加到活跃告警
        {
            let mut active_alerts = self.active_alerts.write().await;
            active_alerts.insert(alert_id, alert.clone());
        }
        
        // 添加到历史记录
        {
            let mut history = self.alert_history.write().await;
            history.push(alert.clone());
        }
        
        warn!("创建告警: {} - {}", title, alert.description);
    }

    pub async fn get_active_alerts(&self) -> Vec<Alert> {
        self.active_alerts.read().await.values().cloned().collect()
    }
}

impl Default for BusinessMetrics {
    fn default() -> Self {
        Self {
            user_metrics: UserMetrics {
                total_users: 0,
                active_users_1d: 0,
                active_users_7d: 0,
                active_users_30d: 0,
                new_registrations_1d: 0,
                temp_users: 0,
                user_retention_rate: 0.0,
            },
            group_metrics: GroupMetrics {
                total_groups: 0,
                active_groups_1d: 0,
                new_groups_1d: 0,
                average_group_size: 0.0,
                largest_group_size: 0,
                group_engagement_rate: 0.0,
            },
            message_metrics: MessageMetrics {
                total_messages_1d: 0,
                group_messages_1d: 0,
                direct_messages_1d: 0,
                messages_per_user: 0.0,
                peak_message_rate: 0.0,
                message_delivery_rate: 0.0,
            },
            moment_metrics: MomentMetrics {
                total_moments_1d: 0,
                moments_with_location: 0,
                moments_with_images: 0,
                average_likes_per_moment: 0.0,
                moment_engagement_rate: 0.0,
            },
            last_updated: chrono::Utc::now(),
        }
    }
}

impl Default for SystemMetrics {
    fn default() -> Self {
        Self {
            database_performance: DatabasePerformanceMetrics {
                connection_pool_utilization: 0.0,
                average_query_time_ms: 0.0,
                slow_queries_count: 0,
                failed_queries_count: 0,
                total_queries_1h: 0,
            },
            websocket_performance: WebSocketPerformanceMetrics {
                active_connections: 0,
                messages_per_second: 0.0,
                connection_success_rate: 0.0,
                average_latency_ms: 0.0,
                memory_usage_mb: 0.0,
            },
            api_performance: ApiPerformanceMetrics {
                requests_per_second: 0.0,
                average_response_time_ms: 0.0,
                error_rate: 0.0,
                p95_response_time_ms: 0.0,
                total_requests_1h: 0,
            },
            resource_usage: ResourceUsageMetrics {
                cpu_usage_percent: 0.0,
                memory_usage_mb: 0.0,
                memory_usage_percent: 0.0,
                disk_usage_percent: 0.0,
                network_io_mbps: 0.0,
            },
            last_updated: chrono::Utc::now(),
        }
    }
}

/// 监控报告
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitoringReport {
    pub business_metrics: BusinessMetrics,
    pub system_metrics: SystemMetrics,
    pub active_alerts: Vec<Alert>,
    pub generated_at: chrono::DateTime<chrono::Utc>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_monitoring_system_creation() {
        let system = EnhancedMonitoringSystem::new(MonitoringConfig::default());
        let business_metrics = system.get_business_metrics().await;
        assert_eq!(business_metrics.user_metrics.total_users, 0);
    }

    #[tokio::test]
    async fn test_alert_creation() {
        let alert_manager = AlertManager::new(AlertConfig::default());
        
        alert_manager.create_alert(
            AlertType::Performance,
            AlertSeverity::Warning,
            "测试告警".to_string(),
            "这是一个测试告警".to_string(),
            85.0,
            80.0,
        ).await;
        
        let alerts = alert_manager.get_active_alerts().await;
        assert_eq!(alerts.len(), 1);
        assert_eq!(alerts[0].title, "测试告警");
    }
}
