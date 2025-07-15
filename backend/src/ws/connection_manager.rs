//! WebSocket连接管理器
//! 
//! 专门负责WebSocket连接的生命周期管理

use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::{RwLock, mpsc};
use uuid::Uuid;
use tracing::{info, warn, debug};

use crate::ws::{
    message_types::ServerMessage,
    errors::{WsError, WsResult},
    constants::connection::MAX_CONNECTIONS,
};

/// WebSocket连接信息
#[derive(Debug, Clone)]
pub struct ConnectionInfo {
    pub user_id: i64,
    #[allow(dead_code)]
    pub connection_id: Uuid,
    #[allow(dead_code)]
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub last_activity: chrono::DateTime<chrono::Utc>,
}

/// WebSocket连接管理器
/// 
/// 职责：
/// - 管理WebSocket连接的生命周期
/// - 维护连接到用户的映射关系
/// - 提供连接查询和统计功能
#[derive(Debug)]
pub struct ConnectionManager {
    /// 活跃连接 connection_id -> ConnectionInfo
    connections: Arc<RwLock<HashMap<Uuid, ConnectionInfo>>>,
    /// 用户到连接的映射 user_id -> Vec<connection_id>
    user_connections: Arc<RwLock<HashMap<i64, Vec<Uuid>>>>,
    /// 消息发送通道 connection_id -> Sender<ServerMessage>
    message_senders: Arc<RwLock<HashMap<Uuid, mpsc::UnboundedSender<ServerMessage>>>>,
}

impl ConnectionManager {
    /// 创建新的连接管理器
    pub fn new() -> Self {
        Self {
            connections: Arc::new(RwLock::new(HashMap::new())),
            user_connections: Arc::new(RwLock::new(HashMap::new())),
            message_senders: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// 添加新连接
    pub async fn add_connection(
        &self,
        connection_id: Uuid,
        user_id: i64,
        sender: mpsc::UnboundedSender<ServerMessage>,
    ) -> WsResult<()> {
        // 检查连接数限制
        {
            let connections = self.connections.read().await;
            if connections.len() >= MAX_CONNECTIONS {
                return Err(WsError::connection_limit_exceeded());
            }
        }

        let now = chrono::Utc::now();
        let connection_info = ConnectionInfo {
            user_id,
            connection_id,
            created_at: now,
            last_activity: now,
        };

        // 存储连接信息
        self.connections.write().await.insert(connection_id, connection_info);
        
        // 更新用户连接映射
        self.user_connections
            .write()
            .await
            .entry(user_id)
            .or_insert_with(Vec::new)
            .push(connection_id);

        // 存储消息发送器
        self.message_senders.write().await.insert(connection_id, sender);

        info!("WebSocket连接已添加: connection_id={}, user_id={}", connection_id, user_id);
        Ok(())
    }

    /// 移除连接
    pub async fn remove_connection(&self, connection_id: Uuid) -> Option<ConnectionInfo> {
        let connection_info = {
            let mut connections = self.connections.write().await;
            connections.remove(&connection_id)
        };

        if let Some(info) = &connection_info {
            // 从用户连接映射中移除
            let mut user_connections = self.user_connections.write().await;
            if let Some(user_conns) = user_connections.get_mut(&info.user_id) {
                user_conns.retain(|&id| id != connection_id);
                if user_conns.is_empty() {
                    user_connections.remove(&info.user_id);
                }
            }

            // 移除消息发送器
            self.message_senders.write().await.remove(&connection_id);

            info!("WebSocket连接已移除: connection_id={}, user_id={}", connection_id, info.user_id);
        }

        connection_info
    }

    /// 更新连接活动时间
    pub async fn update_activity(&self, connection_id: Uuid) {
        let mut connections = self.connections.write().await;
        if let Some(connection) = connections.get_mut(&connection_id) {
            connection.last_activity = chrono::Utc::now();
        }
    }

    /// 获取连接信息
    pub async fn get_connection(&self, connection_id: Uuid) -> Option<ConnectionInfo> {
        let connections = self.connections.read().await;
        connections.get(&connection_id).cloned()
    }

    /// 获取用户的所有连接
    pub async fn get_user_connections(&self, user_id: i64) -> Vec<Uuid> {
        let user_connections = self.user_connections.read().await;
        user_connections.get(&user_id).cloned().unwrap_or_default()
    }

    /// 获取所有在线用户
    pub async fn get_online_users(&self) -> Vec<i64> {
        let user_connections = self.user_connections.read().await;
        user_connections.keys().copied().collect()
    }

    /// 向特定连接发送消息
    pub async fn send_to_connection(&self, connection_id: Uuid, message: ServerMessage) -> WsResult<()> {
        let senders = self.message_senders.read().await;
        if let Some(sender) = senders.get(&connection_id) {
            sender.send(message).map_err(|e| {
                warn!("发送消息到连接失败: connection_id={}, error={}", connection_id, e);
                WsError::internal_error(&format!("消息发送失败: {}", e))
            })?;
            
            // 更新活动时间
            drop(senders);
            self.update_activity(connection_id).await;
            Ok(())
        } else {
            Err(WsError::internal_error(&format!("连接不存在: {}", connection_id)))
        }
    }

    /// 向用户的所有连接发送消息
    pub async fn send_to_user(&self, user_id: i64, message: ServerMessage) -> WsResult<()> {
        let connection_ids = self.get_user_connections(user_id).await;
        if connection_ids.is_empty() {
            debug!("用户 {} 没有活跃连接", user_id);
            return Ok(());
        }

        let mut errors = Vec::new();
        for connection_id in connection_ids {
            if let Err(e) = self.send_to_connection(connection_id, message.clone()).await {
                errors.push(e);
            }
        }

        if !errors.is_empty() {
            warn!("向用户 {} 发送消息时部分连接失败: {:?}", user_id, errors);
        }

        Ok(())
    }

    /// 清理不活跃的连接
    #[allow(dead_code)]
    pub async fn cleanup_inactive_connections(&self, timeout_secs: u64) -> usize {
        let timeout = chrono::Duration::seconds(timeout_secs as i64);
        let now = chrono::Utc::now();
        let mut to_remove = Vec::new();

        {
            let connections = self.connections.read().await;
            for (connection_id, info) in connections.iter() {
                if now.signed_duration_since(info.last_activity) > timeout {
                    to_remove.push(*connection_id);
                }
            }
        }

        for connection_id in &to_remove {
            self.remove_connection(*connection_id).await;
        }

        to_remove.len()
    }

    /// 获取连接统计信息
    #[allow(dead_code)]
    pub async fn get_stats(&self) -> ConnectionStats {
        let connections = self.connections.read().await;
        let user_connections = self.user_connections.read().await;

        ConnectionStats {
            total_connections: connections.len(),
            active_users: user_connections.len(),
        }
    }
}

/// 连接统计信息
#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct ConnectionStats {
    pub total_connections: usize,
    pub active_users: usize,
}

impl Default for ConnectionManager {
    fn default() -> Self {
        Self::new()
    }
}
