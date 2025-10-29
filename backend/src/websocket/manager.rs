use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::{mpsc, RwLock};
use futures_util::stream::SplitSink;
use tokio_tungstenite::{WebSocketStream, tungstenite::Message};
use tokio::net::TcpStream;

use crate::websocket::types::WsMessage;

type WsSender = SplitSink<WebSocketStream<TcpStream>, Message>;

/// WebSocket连接信息
pub struct Connection {
    pub connection_id: String,
    pub user_id: i64,
    pub sender: mpsc::UnboundedSender<Message>,
}

/// WebSocket连接管理器
#[derive(Clone)]
pub struct ConnectionManager {
    // user_id -> Vec<Connection>
    connections: Arc<RwLock<HashMap<i64, Vec<Arc<Connection>>>>>,
    // connection_id -> user_id
    connection_map: Arc<RwLock<HashMap<String, i64>>>,
}

impl ConnectionManager {
    pub fn new() -> Self {
        Self {
            connections: Arc::new(RwLock::new(HashMap::new())),
            connection_map: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// 添加连接
    pub async fn add_connection(
        &self,
        connection_id: String,
        user_id: i64,
        sender: mpsc::UnboundedSender<Message>,
    ) {
        let connection = Arc::new(Connection {
            connection_id: connection_id.clone(),
            user_id,
            sender,
        });

        // 添加到用户连接列表
        let mut connections = self.connections.write().await;
        connections
            .entry(user_id)
            .or_insert_with(Vec::new)
            .push(connection);

        // 添加到连接映射
        let mut connection_map = self.connection_map.write().await;
        connection_map.insert(connection_id, user_id);

        tracing::info!("WebSocket连接已添加: user_id={}, connection_id={}", user_id, connection_id);
    }

    /// 移除连接
    pub async fn remove_connection(&self, connection_id: &str) {
        // 从连接映射中获取user_id
        let mut connection_map = self.connection_map.write().await;
        if let Some(user_id) = connection_map.remove(connection_id) {
            // 从用户连接列表中移除
            let mut connections = self.connections.write().await;
            if let Some(user_connections) = connections.get_mut(&user_id) {
                user_connections.retain(|conn| conn.connection_id != connection_id);
                
                // 如果用户没有其他连接，移除该用户
                if user_connections.is_empty() {
                    connections.remove(&user_id);
                }
            }

            tracing::info!("WebSocket连接已移除: user_id={}, connection_id={}", user_id, connection_id);
        }
    }

    /// 发送消息给指定用户的所有连接
    pub async fn send_to_user(&self, user_id: i64, message: &WsMessage) -> Result<(), String> {
        let connections = self.connections.read().await;
        
        if let Some(user_connections) = connections.get(&user_id) {
            let json_str = serde_json::to_string(message)
                .map_err(|e| format!("序列化消息失败: {}", e))?;
            
            let ws_message = Message::Text(json_str);
            
            for connection in user_connections {
                if let Err(e) = connection.sender.send(ws_message.clone()) {
                    tracing::error!("发送消息失败: user_id={}, error={}", user_id, e);
                }
            }
            
            Ok(())
        } else {
            Err(format!("用户不在线: user_id={}", user_id))
        }
    }

    /// 发送消息给多个用户
    pub async fn send_to_users(&self, user_ids: &[i64], message: &WsMessage) {
        for user_id in user_ids {
            if let Err(e) = self.send_to_user(*user_id, message).await {
                tracing::debug!("{}", e);
            }
        }
    }

    /// 广播消息给所有在线用户
    pub async fn broadcast(&self, message: &WsMessage) -> Result<(), String> {
        let connections = self.connections.read().await;
        
        let json_str = serde_json::to_string(message)
            .map_err(|e| format!("序列化消息失败: {}", e))?;
        
        let ws_message = Message::Text(json_str);
        
        for user_connections in connections.values() {
            for connection in user_connections {
                if let Err(e) = connection.sender.send(ws_message.clone()) {
                    tracing::error!("广播消息失败: user_id={}, error={}", connection.user_id, e);
                }
            }
        }
        
        Ok(())
    }

    /// 检查用户是否在线
    pub async fn is_user_online(&self, user_id: i64) -> bool {
        let connections = self.connections.read().await;
        connections.contains_key(&user_id)
    }

    /// 获取在线用户数量
    pub async fn online_user_count(&self) -> usize {
        let connections = self.connections.read().await;
        connections.len()
    }

    /// 获取在线用户列表
    pub async fn get_online_users(&self) -> Vec<i64> {
        let connections = self.connections.read().await;
        connections.keys().copied().collect()
    }

    /// 获取用户的连接数
    pub async fn get_user_connection_count(&self, user_id: i64) -> usize {
        let connections = self.connections.read().await;
        connections.get(&user_id).map(|v| v.len()).unwrap_or(0)
    }
}

impl Default for ConnectionManager {
    fn default() -> Self {
        Self::new()
    }
}
