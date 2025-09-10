/// 业务指标监控系统
/// 
/// 收集和分析HotSpot应用的业务指标
use std::sync::Arc;
use std::collections::HashMap;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};
use tokio::sync::RwLock;
use serde::{Serialize, Deserialize};
use chrono::{DateTime, Utc};

/// 业务指标类型
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum BusinessMetricType {
    /// 用户相关指标
    UserRegistration,
    UserLogin,
    UserLocationUpdate,
    TempUserCreation,
    TempUserUpgrade,
    
    /// 群组相关指标
    GroupCreation,
    GroupJoin,
    GroupLeave,
    GroupMessage,
    
    /// 消息相关指标
    PrivateMessage,
    GroupMessageSent,
    MessageRead,
    
    /// 生活瞬间相关指标
    MomentCreation,
    MomentLike,
    MomentComment,
    
    /// WebSocket相关指标
    WebSocketConnection,
    WebSocketDisconnection,
    WebSocketMessage,
    
    /// 地理位置相关指标
    NearbyUserQuery,
    NearbyGroupQuery,
    LocationBasedSearch,
    
    /// 系统相关指标
    ApiRequest,
    ApiError,
    DatabaseQuery,
    CacheHit,
    CacheMiss,
}

/// 业务指标数据点
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BusinessMetricPoint {
    pub metric_type: BusinessMetricType,
    pub value: f64,
    pub timestamp: DateTime<Utc>,
    pub tags: HashMap<String, String>,
    pub user_id: Option<i64>,
}

/// 业务指标统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BusinessMetricStats {
    pub metric_type: BusinessMetricType,
    pub count: u64,
    pub sum: f64,
    pub average: f64,
    pub min: f64,
    pub max: f64,
    pub last_updated: DateTime<Utc>,
    pub hourly_counts: Vec<u64>, // 最近24小时的每小时计数
}

/// 实时业务指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RealtimeBusinessMetrics {
    pub active_users: u64,
    pub active_websocket_connections: u64,
    pub messages_per_minute: f64,
    pub api_requests_per_minute: f64,
    pub error_rate_percentage: f64,
    pub average_response_time_ms: f64,
    pub database_query_rate: f64,
    pub cache_hit_rate_percentage: f64,
}

/// 业务指标监控器
pub struct BusinessMetricsMonitor {
    metrics: Arc<RwLock<HashMap<BusinessMetricType, BusinessMetricStats>>>,
    recent_points: Arc<RwLock<Vec<BusinessMetricPoint>>>,
    start_time: Instant,
}

impl BusinessMetricsMonitor {
    /// 创建新的业务指标监控器
    pub fn new() -> Self {
        Self {
            metrics: Arc::new(RwLock::new(HashMap::new())),
            recent_points: Arc::new(RwLock::new(Vec::new())),
            start_time: Instant::now(),
        }
    }
    
    /// 记录业务指标
    pub async fn record_metric(
        &self,
        metric_type: BusinessMetricType,
        value: f64,
        tags: Option<HashMap<String, String>>,
        user_id: Option<i64>,
    ) {
        let point = BusinessMetricPoint {
            metric_type: metric_type.clone(),
            value,
            timestamp: Utc::now(),
            tags: tags.unwrap_or_default(),
            user_id,
        };
        
        // 添加到最近数据点
        {
            let mut recent_points = self.recent_points.write().await;
            recent_points.push(point.clone());
            
            // 保持最近1000个数据点
            if recent_points.len() > 1000 {
                recent_points.drain(0..recent_points.len() - 1000);
            }
        }
        
        // 更新统计信息
        {
            let mut metrics = self.metrics.write().await;
            let stats = metrics.entry(metric_type).or_insert_with(|| BusinessMetricStats {
                metric_type: point.metric_type.clone(),
                count: 0,
                sum: 0.0,
                average: 0.0,
                min: f64::MAX,
                max: f64::MIN,
                last_updated: Utc::now(),
                hourly_counts: vec![0; 24],
            });
            
            stats.count += 1;
            stats.sum += value;
            stats.average = stats.sum / stats.count as f64;
            stats.min = stats.min.min(value);
            stats.max = stats.max.max(value);
            stats.last_updated = point.timestamp;
            
            // 更新小时计数
            let hour = point.timestamp.hour() as usize;
            if hour < stats.hourly_counts.len() {
                stats.hourly_counts[hour] += 1;
            }
        }
    }
    
    /// 记录用户注册
    pub async fn record_user_registration(&self, user_id: i64, is_temp_user: bool) {
        let metric_type = if is_temp_user {
            BusinessMetricType::TempUserCreation
        } else {
            BusinessMetricType::UserRegistration
        };
        
        let mut tags = HashMap::new();
        tags.insert("user_type".to_string(), if is_temp_user { "temp" } else { "regular" }.to_string());
        
        self.record_metric(metric_type, 1.0, Some(tags), Some(user_id)).await;
    }
    
    /// 记录用户登录
    pub async fn record_user_login(&self, user_id: i64, login_method: &str) {
        let mut tags = HashMap::new();
        tags.insert("login_method".to_string(), login_method.to_string());
        
        self.record_metric(BusinessMetricType::UserLogin, 1.0, Some(tags), Some(user_id)).await;
    }
    
    /// 记录群组活动
    pub async fn record_group_activity(&self, activity_type: &str, group_id: &str, user_id: Option<i64>) {
        let metric_type = match activity_type {
            "create" => BusinessMetricType::GroupCreation,
            "join" => BusinessMetricType::GroupJoin,
            "leave" => BusinessMetricType::GroupLeave,
            "message" => BusinessMetricType::GroupMessage,
            _ => return,
        };
        
        let mut tags = HashMap::new();
        tags.insert("group_id".to_string(), group_id.to_string());
        
        self.record_metric(metric_type, 1.0, Some(tags), user_id).await;
    }
    
    /// 记录消息活动
    pub async fn record_message_activity(&self, message_type: &str, user_id: i64) {
        let metric_type = match message_type {
            "private" => BusinessMetricType::PrivateMessage,
            "group" => BusinessMetricType::GroupMessageSent,
            _ => return,
        };
        
        self.record_metric(metric_type, 1.0, None, Some(user_id)).await;
    }
    
    /// 记录WebSocket活动
    pub async fn record_websocket_activity(&self, activity_type: &str, user_id: Option<i64>) {
        let metric_type = match activity_type {
            "connect" => BusinessMetricType::WebSocketConnection,
            "disconnect" => BusinessMetricType::WebSocketDisconnection,
            "message" => BusinessMetricType::WebSocketMessage,
            _ => return,
        };
        
        self.record_metric(metric_type, 1.0, None, user_id).await;
    }
    
    /// 记录API请求
    pub async fn record_api_request(&self, endpoint: &str, method: &str, status_code: u16, response_time_ms: f64) {
        let mut tags = HashMap::new();
        tags.insert("endpoint".to_string(), endpoint.to_string());
        tags.insert("method".to_string(), method.to_string());
        tags.insert("status_code".to_string(), status_code.to_string());
        
        // 记录API请求
        self.record_metric(BusinessMetricType::ApiRequest, response_time_ms, Some(tags.clone()), None).await;
        
        // 如果是错误状态码，记录API错误
        if status_code >= 400 {
            self.record_metric(BusinessMetricType::ApiError, 1.0, Some(tags), None).await;
        }
    }
    
    /// 记录数据库查询
    pub async fn record_database_query(&self, query_type: &str, duration_ms: f64) {
        let mut tags = HashMap::new();
        tags.insert("query_type".to_string(), query_type.to_string());
        
        self.record_metric(BusinessMetricType::DatabaseQuery, duration_ms, Some(tags), None).await;
    }
    
    /// 记录缓存操作
    pub async fn record_cache_operation(&self, operation: &str, hit: bool) {
        let metric_type = if hit {
            BusinessMetricType::CacheHit
        } else {
            BusinessMetricType::CacheMiss
        };
        
        let mut tags = HashMap::new();
        tags.insert("operation".to_string(), operation.to_string());
        
        self.record_metric(metric_type, 1.0, Some(tags), None).await;
    }
    
    /// 获取指标统计
    pub async fn get_metric_stats(&self, metric_type: &BusinessMetricType) -> Option<BusinessMetricStats> {
        let metrics = self.metrics.read().await;
        metrics.get(metric_type).cloned()
    }
    
    /// 获取所有指标统计
    pub async fn get_all_metrics(&self) -> HashMap<BusinessMetricType, BusinessMetricStats> {
        let metrics = self.metrics.read().await;
        metrics.clone()
    }
    
    /// 获取实时业务指标
    pub async fn get_realtime_metrics(&self) -> RealtimeBusinessMetrics {
        let metrics = self.metrics.read().await;
        let recent_points = self.recent_points.read().await;
        
        // 计算最近1分钟的指标
        let one_minute_ago = Utc::now() - chrono::Duration::minutes(1);
        let recent_points_1min: Vec<_> = recent_points
            .iter()
            .filter(|p| p.timestamp > one_minute_ago)
            .collect();
        
        // 计算各种实时指标
        let messages_per_minute = recent_points_1min
            .iter()
            .filter(|p| matches!(p.metric_type, BusinessMetricType::PrivateMessage | BusinessMetricType::GroupMessageSent))
            .count() as f64;
        
        let api_requests_per_minute = recent_points_1min
            .iter()
            .filter(|p| matches!(p.metric_type, BusinessMetricType::ApiRequest))
            .count() as f64;
        
        let api_errors = recent_points_1min
            .iter()
            .filter(|p| matches!(p.metric_type, BusinessMetricType::ApiError))
            .count() as f64;
        
        let error_rate_percentage = if api_requests_per_minute > 0.0 {
            (api_errors / api_requests_per_minute) * 100.0
        } else {
            0.0
        };
        
        let api_response_times: Vec<f64> = recent_points_1min
            .iter()
            .filter(|p| matches!(p.metric_type, BusinessMetricType::ApiRequest))
            .map(|p| p.value)
            .collect();
        
        let average_response_time_ms = if !api_response_times.is_empty() {
            api_response_times.iter().sum::<f64>() / api_response_times.len() as f64
        } else {
            0.0
        };
        
        let cache_hits = recent_points_1min
            .iter()
            .filter(|p| matches!(p.metric_type, BusinessMetricType::CacheHit))
            .count() as f64;
        
        let cache_misses = recent_points_1min
            .iter()
            .filter(|p| matches!(p.metric_type, BusinessMetricType::CacheMiss))
            .count() as f64;
        
        let cache_hit_rate_percentage = if (cache_hits + cache_misses) > 0.0 {
            (cache_hits / (cache_hits + cache_misses)) * 100.0
        } else {
            0.0
        };
        
        let database_query_rate = recent_points_1min
            .iter()
            .filter(|p| matches!(p.metric_type, BusinessMetricType::DatabaseQuery))
            .count() as f64;
        
        // 获取活跃连接数（这里使用模拟值，实际应该从连接管理器获取）
        let active_websocket_connections = metrics
            .get(&BusinessMetricType::WebSocketConnection)
            .map(|s| s.count)
            .unwrap_or(0);
        
        RealtimeBusinessMetrics {
            active_users: 0, // 需要从用户会话管理器获取
            active_websocket_connections,
            messages_per_minute,
            api_requests_per_minute,
            error_rate_percentage,
            average_response_time_ms,
            database_query_rate,
            cache_hit_rate_percentage,
        }
    }
    
    /// 获取最近的数据点
    pub async fn get_recent_points(&self, limit: usize) -> Vec<BusinessMetricPoint> {
        let recent_points = self.recent_points.read().await;
        let start = if recent_points.len() > limit {
            recent_points.len() - limit
        } else {
            0
        };
        recent_points[start..].to_vec()
    }
    
    /// 清理旧数据
    pub async fn cleanup_old_data(&self, max_age: Duration) {
        let cutoff_time = Utc::now() - chrono::Duration::from_std(max_age).unwrap_or(chrono::Duration::hours(24));
        
        let mut recent_points = self.recent_points.write().await;
        recent_points.retain(|point| point.timestamp > cutoff_time);
    }
    
    /// 获取运行时间
    pub fn get_uptime(&self) -> Duration {
        self.start_time.elapsed()
    }
}

impl Default for BusinessMetricsMonitor {
    fn default() -> Self {
        Self::new()
    }
}

/// 全局业务指标监控器实例
use std::sync::OnceLock;
pub static GLOBAL_BUSINESS_METRICS: OnceLock<BusinessMetricsMonitor> = OnceLock::new();

/// 获取全局业务指标监控器
pub fn get_business_metrics() -> &'static BusinessMetricsMonitor {
    GLOBAL_BUSINESS_METRICS.get_or_init(|| BusinessMetricsMonitor::new())
}

/// 业务指标中间件宏
#[macro_export]
macro_rules! record_business_metric {
    ($metric_type:expr, $value:expr) => {
        tokio::spawn(async move {
            let monitor = crate::utils::business_metrics::get_business_metrics();
            monitor.record_metric($metric_type, $value, None, None).await;
        });
    };
    ($metric_type:expr, $value:expr, $tags:expr) => {
        tokio::spawn(async move {
            let monitor = crate::utils::business_metrics::get_business_metrics();
            monitor.record_metric($metric_type, $value, Some($tags), None).await;
        });
    };
    ($metric_type:expr, $value:expr, $tags:expr, $user_id:expr) => {
        tokio::spawn(async move {
            let monitor = crate::utils::business_metrics::get_business_metrics();
            monitor.record_metric($metric_type, $value, Some($tags), Some($user_id)).await;
        });
    };
}
