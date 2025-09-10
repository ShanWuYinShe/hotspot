/// WebSocket连接优化器
/// 
/// 提供WebSocket连接池管理、性能优化和连接监控功能
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};
use tokio::sync::RwLock;
use tracing::{debug, warn, info};
use serde::{Serialize, Deserialize};

/// WebSocket连接优化配置
#[derive(Debug, Clone)]
pub struct ConnectionOptimizerConfig {
    /// 最大连接数
    pub max_connections: usize,
    /// 连接空闲超时（秒）
    pub idle_timeout_seconds: u64,
    /// 心跳间隔（秒）
    pub heartbeat_interval_seconds: u64,
    /// 连接清理间隔（秒）
    pub cleanup_interval_seconds: u64,
    /// 每个用户最大连接数
    pub max_connections_per_user: usize,
    /// 是否启用连接压缩
    pub enable_compression: bool,
    /// 消息缓冲区大小
    pub message_buffer_size: usize,
    /// 是否启用连接统计
    pub enable_connection_stats: bool,
}

impl Default for ConnectionOptimizerConfig {
    fn default() -> Self {
        Self {
            max_connections: 10000,
            idle_timeout_seconds: 300, // 5分钟
            heartbeat_interval_seconds: 30,
            cleanup_interval_seconds: 60,
            max_connections_per_user: 5,
            enable_compression: true,
            message_buffer_size: 1024,
            enable_connection_stats: true,
        }
    }
}

/// 连接信息
#[derive(Debug, Clone)]
pub struct ConnectionInfo {
    pub connection_id: String,
    pub user_id: Option<i64>,
    pub connected_at: Instant,
    pub last_activity: Instant,
    pub message_count: u64,
    pub bytes_sent: u64,
    pub bytes_received: u64,
    pub client_ip: Option<String>,
    pub user_agent: Option<String>,
    pub is_authenticated: bool,
    pub subscribed_rooms: Vec<String>,
}

impl ConnectionInfo {
    pub fn new(connection_id: String) -> Self {
        let now = Instant::now();
        Self {
            connection_id,
            user_id: None,
            connected_at: now,
            last_activity: now,
            message_count: 0,
            bytes_sent: 0,
            bytes_received: 0,
            client_ip: None,
            user_agent: None,
            is_authenticated: false,
            subscribed_rooms: Vec::new(),
        }
    }

    pub fn update_activity(&mut self) {
        self.last_activity = Instant::now();
    }

    pub fn add_message(&mut self, bytes: usize, is_outbound: bool) {
        self.message_count += 1;
        if is_outbound {
            self.bytes_sent += bytes as u64;
        } else {
            self.bytes_received += bytes as u64;
        }
        self.update_activity();
    }

    pub fn is_idle(&self, timeout: Duration) -> bool {
        self.last_activity.elapsed() > timeout
    }

    pub fn authenticate(&mut self, user_id: i64) {
        self.user_id = Some(user_id);
        self.is_authenticated = true;
        self.update_activity();
    }

    pub fn subscribe_room(&mut self, room_id: String) {
        if !self.subscribed_rooms.contains(&room_id) {
            self.subscribed_rooms.push(room_id);
        }
    }

    pub fn unsubscribe_room(&mut self, room_id: &str) {
        self.subscribed_rooms.retain(|r| r != room_id);
    }
}

/// 连接统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConnectionStats {
    pub total_connections: usize,
    pub authenticated_connections: usize,
    pub idle_connections: usize,
    pub connections_per_user: HashMap<i64, usize>,
    pub average_connection_duration: f64,
    pub total_messages: u64,
    pub total_bytes_sent: u64,
    pub total_bytes_received: u64,
    pub peak_connections: usize,
    pub connection_rate: f64, // 每分钟新连接数
}

/// WebSocket连接优化器
pub struct ConnectionOptimizer {
    config: ConnectionOptimizerConfig,
    connections: Arc<RwLock<HashMap<String, ConnectionInfo>>>,
    user_connections: Arc<RwLock<HashMap<i64, Vec<String>>>>,
    stats: Arc<Mutex<ConnectionStats>>,
    peak_connections: Arc<Mutex<usize>>,
    recent_connections: Arc<Mutex<Vec<Instant>>>,
}

impl ConnectionOptimizer {
    /// 创建新的连接优化器
    pub fn new(config: ConnectionOptimizerConfig) -> Self {
        let optimizer = Self {
            config,
            connections: Arc::new(RwLock::new(HashMap::new())),
            user_connections: Arc::new(RwLock::new(HashMap::new())),
            stats: Arc::new(Mutex::new(ConnectionStats {
                total_connections: 0,
                authenticated_connections: 0,
                idle_connections: 0,
                connections_per_user: HashMap::new(),
                average_connection_duration: 0.0,
                total_messages: 0,
                total_bytes_sent: 0,
                total_bytes_received: 0,
                peak_connections: 0,
                connection_rate: 0.0,
            })),
            peak_connections: Arc::new(Mutex::new(0)),
            recent_connections: Arc::new(Mutex::new(Vec::new())),
        };

        optimizer
    }

    /// 添加新连接
    pub async fn add_connection(&self, connection_id: String, client_ip: Option<String>, user_agent: Option<String>) -> Result<(), String> {
        let mut connections = self.connections.write().await;
        
        // 检查连接数限制
        if connections.len() >= self.config.max_connections {
            warn!("达到最大连接数限制: {}", self.config.max_connections);
            return Err("服务器连接数已满".to_string());
        }

        let mut connection_info = ConnectionInfo::new(connection_id.clone());
        connection_info.client_ip = client_ip;
        connection_info.user_agent = user_agent;

        connections.insert(connection_id.clone(), connection_info);
        
        // 更新统计
        self.update_connection_stats().await;
        
        // 记录新连接
        {
            let mut recent = self.recent_connections.lock().unwrap();
            recent.push(Instant::now());
            
            // 保持最近1小时的连接记录
            let one_hour_ago = Instant::now() - Duration::from_secs(3600);
            recent.retain(|&time| time > one_hour_ago);
        }

        debug!("新WebSocket连接: {}", connection_id);
        Ok(())
    }

    /// 移除连接
    pub async fn remove_connection(&self, connection_id: &str) {
        let mut connections = self.connections.write().await;
        
        if let Some(connection_info) = connections.remove(connection_id) {
            // 如果是认证用户，从用户连接映射中移除
            if let Some(user_id) = connection_info.user_id {
                let mut user_connections = self.user_connections.write().await;
                if let Some(user_conn_list) = user_connections.get_mut(&user_id) {
                    user_conn_list.retain(|id| id != connection_id);
                    if user_conn_list.is_empty() {
                        user_connections.remove(&user_id);
                    }
                }
            }
            
            debug!("移除WebSocket连接: {}, 连接时长: {:?}", 
                   connection_id, connection_info.last_activity.duration_since(connection_info.connected_at));
        }
        
        // 更新统计
        self.update_connection_stats().await;
    }

    /// 认证连接
    pub async fn authenticate_connection(&self, connection_id: &str, user_id: i64) -> Result<(), String> {
        let mut connections = self.connections.write().await;
        
        if let Some(connection_info) = connections.get_mut(connection_id) {
            // 检查用户连接数限制
            let user_connections = self.user_connections.read().await;
            if let Some(user_conn_list) = user_connections.get(&user_id) {
                if user_conn_list.len() >= self.config.max_connections_per_user {
                    warn!("用户{}达到最大连接数限制: {}", user_id, self.config.max_connections_per_user);
                    return Err("用户连接数已达上限".to_string());
                }
            }
            drop(user_connections);

            connection_info.authenticate(user_id);
            
            // 添加到用户连接映射
            let mut user_connections = self.user_connections.write().await;
            user_connections.entry(user_id)
                .or_insert_with(Vec::new)
                .push(connection_id.to_string());
            
            debug!("连接{}认证成功，用户ID: {}", connection_id, user_id);
            Ok(())
        } else {
            Err("连接不存在".to_string())
        }
    }

    /// 更新连接活动
    pub async fn update_connection_activity(&self, connection_id: &str, message_bytes: usize, is_outbound: bool) {
        let mut connections = self.connections.write().await;
        
        if let Some(connection_info) = connections.get_mut(connection_id) {
            connection_info.add_message(message_bytes, is_outbound);
        }
    }

    /// 订阅房间
    pub async fn subscribe_room(&self, connection_id: &str, room_id: String) -> Result<(), String> {
        let mut connections = self.connections.write().await;
        
        if let Some(connection_info) = connections.get_mut(connection_id) {
            connection_info.subscribe_room(room_id.clone());
            debug!("连接{}订阅房间: {}", connection_id, room_id);
            Ok(())
        } else {
            Err("连接不存在".to_string())
        }
    }

    /// 取消订阅房间
    pub async fn unsubscribe_room(&self, connection_id: &str, room_id: &str) -> Result<(), String> {
        let mut connections = self.connections.write().await;
        
        if let Some(connection_info) = connections.get_mut(connection_id) {
            connection_info.unsubscribe_room(room_id);
            debug!("连接{}取消订阅房间: {}", connection_id, room_id);
            Ok(())
        } else {
            Err("连接不存在".to_string())
        }
    }

    /// 获取用户的所有连接
    pub async fn get_user_connections(&self, user_id: i64) -> Vec<String> {
        let user_connections = self.user_connections.read().await;
        user_connections.get(&user_id)
            .map(|connections| connections.clone())
            .unwrap_or_default()
    }

    /// 获取房间的所有连接
    pub async fn get_room_connections(&self, room_id: &str) -> Vec<String> {
        let connections = self.connections.read().await;
        connections.values()
            .filter(|conn| conn.subscribed_rooms.contains(&room_id.to_string()))
            .map(|conn| conn.connection_id.clone())
            .collect()
    }

    /// 清理空闲连接
    pub async fn cleanup_idle_connections(&self) -> usize {
        let idle_timeout = Duration::from_secs(self.config.idle_timeout_seconds);
        let mut connections = self.connections.write().await;
        let mut user_connections = self.user_connections.write().await;
        
        let initial_count = connections.len();
        let mut removed_connections = Vec::new();
        
        // 找出空闲连接
        connections.retain(|connection_id, connection_info| {
            if connection_info.is_idle(idle_timeout) {
                removed_connections.push((connection_id.clone(), connection_info.user_id));
                false
            } else {
                true
            }
        });
        
        // 从用户连接映射中移除
        for (connection_id, user_id) in removed_connections {
            if let Some(user_id) = user_id {
                if let Some(user_conn_list) = user_connections.get_mut(&user_id) {
                    user_conn_list.retain(|id| id != &connection_id);
                    if user_conn_list.is_empty() {
                        user_connections.remove(&user_id);
                    }
                }
            }
            debug!("清理空闲连接: {}", connection_id);
        }
        
        let cleaned_count = initial_count - connections.len();
        if cleaned_count > 0 {
            info!("清理了{}个空闲WebSocket连接", cleaned_count);
        }
        
        // 更新统计
        drop(connections);
        drop(user_connections);
        self.update_connection_stats().await;
        
        cleaned_count
    }

    /// 获取连接统计
    pub async fn get_connection_stats(&self) -> ConnectionStats {
        self.update_connection_stats().await;
        self.stats.lock().unwrap().clone()
    }

    /// 更新连接统计
    async fn update_connection_stats(&self) {
        let connections = self.connections.read().await;
        let mut stats = self.stats.lock().unwrap();
        
        stats.total_connections = connections.len();
        stats.authenticated_connections = connections.values()
            .filter(|conn| conn.is_authenticated)
            .count();
        
        let idle_timeout = Duration::from_secs(self.config.idle_timeout_seconds);
        stats.idle_connections = connections.values()
            .filter(|conn| conn.is_idle(idle_timeout))
            .count();
        
        // 统计每个用户的连接数
        stats.connections_per_user.clear();
        for connection in connections.values() {
            if let Some(user_id) = connection.user_id {
                *stats.connections_per_user.entry(user_id).or_insert(0) += 1;
            }
        }
        
        // 计算平均连接时长
        if !connections.is_empty() {
            let total_duration: Duration = connections.values()
                .map(|conn| conn.last_activity.duration_since(conn.connected_at))
                .sum();
            stats.average_connection_duration = total_duration.as_secs_f64() / connections.len() as f64;
        }
        
        // 统计消息和字节数
        stats.total_messages = connections.values().map(|conn| conn.message_count).sum();
        stats.total_bytes_sent = connections.values().map(|conn| conn.bytes_sent).sum();
        stats.total_bytes_received = connections.values().map(|conn| conn.bytes_received).sum();
        
        // 更新峰值连接数
        let mut peak = self.peak_connections.lock().unwrap();
        if stats.total_connections > *peak {
            *peak = stats.total_connections;
        }
        stats.peak_connections = *peak;
        
        // 计算连接速率（每分钟新连接数）
        let recent = self.recent_connections.lock().unwrap();
        let one_minute_ago = Instant::now() - Duration::from_secs(60);
        let recent_count = recent.iter().filter(|&&time| time > one_minute_ago).count();
        stats.connection_rate = recent_count as f64;
    }

    /// 启动清理任务
    #[allow(dead_code)]
    fn start_cleanup_task(self: Arc<Self>) {
        let cleanup_optimizer = self.clone();
        
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(
                Duration::from_secs(cleanup_optimizer.config.cleanup_interval_seconds)
            );
            
            loop {
                interval.tick().await;
                cleanup_optimizer.cleanup_idle_connections().await;
            }
        });
        
        info!("WebSocket连接清理任务已启动");
    }

    /// 获取连接详情
    pub async fn get_connection_info(&self, connection_id: &str) -> Option<ConnectionInfo> {
        let connections = self.connections.read().await;
        connections.get(connection_id).cloned()
    }

    /// 强制断开用户的所有连接
    pub async fn disconnect_user(&self, user_id: i64) -> usize {
        let user_connections = self.get_user_connections(user_id).await;
        let disconnect_count = user_connections.len();
        
        for connection_id in user_connections {
            self.remove_connection(&connection_id).await;
        }
        
        if disconnect_count > 0 {
            info!("强制断开用户{}的{}个连接", user_id, disconnect_count);
        }
        
        disconnect_count
    }

    /// 广播消息到房间
    pub async fn get_room_broadcast_targets(&self, room_id: &str, exclude_connection: Option<&str>) -> Vec<String> {
        let connections = self.connections.read().await;
        connections.values()
            .filter(|conn| {
                conn.subscribed_rooms.contains(&room_id.to_string()) &&
                exclude_connection.map_or(true, |exclude| conn.connection_id != exclude)
            })
            .map(|conn| conn.connection_id.clone())
            .collect()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_connection_management() {
        let config = ConnectionOptimizerConfig::default();
        let optimizer = ConnectionOptimizer::new(config);
        
        // 添加连接
        let conn_id = "test_connection_1".to_string();
        assert!(optimizer.add_connection(conn_id.clone(), Some("127.0.0.1".to_string()), None).await.is_ok());
        
        // 认证连接
        assert!(optimizer.authenticate_connection(&conn_id, 123).await.is_ok());
        
        // 订阅房间
        assert!(optimizer.subscribe_room(&conn_id, "room_1".to_string()).await.is_ok());
        
        // 获取用户连接
        let user_connections = optimizer.get_user_connections(123).await;
        assert_eq!(user_connections.len(), 1);
        assert_eq!(user_connections[0], conn_id);
        
        // 获取房间连接
        let room_connections = optimizer.get_room_connections("room_1").await;
        assert_eq!(room_connections.len(), 1);
        assert_eq!(room_connections[0], conn_id);
        
        // 移除连接
        optimizer.remove_connection(&conn_id).await;
        let user_connections = optimizer.get_user_connections(123).await;
        assert_eq!(user_connections.len(), 0);
    }

    #[tokio::test]
    async fn test_connection_limits() {
        let mut config = ConnectionOptimizerConfig::default();
        config.max_connections_per_user = 2;
        let optimizer = ConnectionOptimizer::new(config);
        
        // 添加第一个连接
        let conn_id1 = "test_connection_1".to_string();
        assert!(optimizer.add_connection(conn_id1.clone(), None, None).await.is_ok());
        assert!(optimizer.authenticate_connection(&conn_id1, 123).await.is_ok());
        
        // 添加第二个连接
        let conn_id2 = "test_connection_2".to_string();
        assert!(optimizer.add_connection(conn_id2.clone(), None, None).await.is_ok());
        assert!(optimizer.authenticate_connection(&conn_id2, 123).await.is_ok());
        
        // 尝试添加第三个连接（应该失败）
        let conn_id3 = "test_connection_3".to_string();
        assert!(optimizer.add_connection(conn_id3.clone(), None, None).await.is_ok());
        assert!(optimizer.authenticate_connection(&conn_id3, 123).await.is_err());
    }

    #[test]
    fn test_connection_info() {
        let mut conn_info = ConnectionInfo::new("test_conn".to_string());
        
        // 测试活动更新
        conn_info.add_message(100, true);
        assert_eq!(conn_info.message_count, 1);
        assert_eq!(conn_info.bytes_sent, 100);
        assert_eq!(conn_info.bytes_received, 0);
        
        // 测试认证
        conn_info.authenticate(123);
        assert_eq!(conn_info.user_id, Some(123));
        assert!(conn_info.is_authenticated);
        
        // 测试房间订阅
        conn_info.subscribe_room("room_1".to_string());
        assert_eq!(conn_info.subscribed_rooms.len(), 1);
        assert!(conn_info.subscribed_rooms.contains(&"room_1".to_string()));
        
        conn_info.unsubscribe_room("room_1");
        assert_eq!(conn_info.subscribed_rooms.len(), 0);
    }
}
