use axum::extract::ws::Message;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Instant;
use tokio::sync::{mpsc, RwLock};

use crate::websocket::types::WsMessage;

/// WebSocket连接信息
pub struct Connection {
    pub connection_id: String,
    pub sender: mpsc::UnboundedSender<Message>,
    pub last_ping: Arc<RwLock<Instant>>,
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
            sender,
            last_ping: Arc::new(RwLock::new(Instant::now())),
        });

        // 添加到用户连接列表
        let mut connections = self.connections.write().await;
        connections
            .entry(user_id)
            .or_insert_with(Vec::new)
            .push(connection);

        // 添加到连接映射
        let mut connection_map = self.connection_map.write().await;
        connection_map.insert(connection_id.clone(), user_id);

        tracing::debug!(
            "WebSocket连接已添加: user_id={}, connection_id={}",
            user_id,
            connection_id
        );
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

            tracing::debug!(
                "WebSocket连接已移除: user_id={}, connection_id={}",
                user_id,
                connection_id
            );
        }
    }

    /// 发送已序列化的 JSON 到指定用户的所有连接（内部方法，&str 版本，用于 send_to_users）
    async fn send_to_user_prepared(&self, user_id: i64, json_str: &str) -> Result<(), String> {
        let connections = self.connections.read().await;

        if let Some(user_connections) = connections.get(&user_id) {
            for conn in user_connections {
                if let Err(e) = conn.sender.send(Message::Text(json_str.to_string().into())) {
                    tracing::error!("发送消息失败: user_id={}, error={}", user_id, e);
                }
            }
            Ok(())
        } else {
            Err(format!("用户不在线: user_id={}", user_id))
        }
    }

    /// 发送已序列化的 JSON 到指定用户（String 版本，最后一个连接移动所有权，避免一次 clone）
    async fn send_to_user_prepared_owned(
        &self,
        user_id: i64,
        json_str: String,
    ) -> Result<(), String> {
        let connections = self.connections.read().await;

        if let Some(user_connections) = connections.get(&user_id) {
            if let Some((last, rest)) = user_connections.split_last() {
                for conn in rest {
                    if let Err(e) = conn.sender.send(Message::Text(json_str.clone().into())) {
                        tracing::error!("发送消息失败: user_id={}, error={}", user_id, e);
                    }
                }
                if let Err(e) = last.sender.send(Message::Text(json_str.into())) {
                    tracing::error!("发送消息失败: user_id={}, error={}", user_id, e);
                }
            }
            Ok(())
        } else {
            Err(format!("用户不在线: user_id={}", user_id))
        }
    }

    /// 发送消息给指定用户的所有连接
    pub async fn send_to_user(&self, user_id: i64, message: &WsMessage) -> Result<(), String> {
        let json_str =
            serde_json::to_string(message).map_err(|e| format!("序列化消息失败: {}", e))?;
        self.send_to_user_prepared_owned(user_id, json_str).await
    }

    /// 发送消息给多个用户（序列化一次，分发给所有用户）
    pub async fn send_to_users(&self, user_ids: &[i64], message: &WsMessage) {
        let json_str = match serde_json::to_string(message) {
            Ok(s) => s,
            Err(e) => {
                tracing::error!("序列化 WebSocket 消息失败: {}", e);
                return;
            }
        };
        for user_id in user_ids {
            // 发送失败（常见原因：对方离线）不记录日志，避免噪音；真正的发送异常会在 send_to_user_prepared 内部记录 error
            let _ = self.send_to_user_prepared(*user_id, &json_str).await;
        }
    }

    /// 断开指定用户的所有 WebSocket 连接
    pub async fn disconnect_user(&self, user_id: i64) {
        let connection_ids: Vec<String> = {
            let connections = self.connections.read().await;
            connections
                .get(&user_id)
                .map(|conns| conns.iter().map(|c| c.connection_id.clone()).collect())
                .unwrap_or_default()
        };
        for conn_id in connection_ids {
            self.remove_connection(&conn_id).await;
        }
    }

    /// 获取所有在线用户ID
    pub async fn get_online_user_ids(&self) -> Vec<i64> {
        let connections = self.connections.read().await;
        connections.keys().copied().collect()
    }

    /// 获取用户的连接数
    pub async fn get_user_connection_count(&self, user_id: i64) -> usize {
        let connections = self.connections.read().await;
        connections.get(&user_id).map(|v| v.len()).unwrap_or(0)
    }

    /// 更新连接的最后心跳时间
    pub async fn update_ping(&self, connection_id: &str) -> Result<(), String> {
        let connection_map = self.connection_map.read().await;
        if let Some(&user_id) = connection_map.get(connection_id) {
            let connections = self.connections.read().await;
            if let Some(user_connections) = connections.get(&user_id) {
                for conn in user_connections {
                    if conn.connection_id == connection_id {
                        let mut last_ping = conn.last_ping.write().await;
                        *last_ping = Instant::now();
                        return Ok(());
                    }
                }
            }
        }
        Err(format!("连接不存在: connection_id={}", connection_id))
    }

    /// 发送消息给指定连接（用于心跳超时等场景下向单个连接发送 Error 后断开）
    pub async fn send_to_connection(&self, connection_id: &str, message: &WsMessage) {
        let json_str = match serde_json::to_string(message) {
            Ok(s) => s,
            Err(e) => {
                tracing::error!("序列化 WebSocket 消息失败: {}", e);
                return;
            }
        };
        let connections = self.connections.read().await;
        for user_connections in connections.values() {
            for conn in user_connections {
                if conn.connection_id == connection_id {
                    let _ = conn.sender.send(Message::Text(json_str.into()));
                    return;
                }
            }
        }
    }

    /// 检查连接是否超时（超过 WS_CONNECTION_TIMEOUT_SECS 秒未收到心跳）
    pub async fn check_timeout(&self) -> Vec<String> {
        let mut timeout_connections = Vec::new();
        let connections = self.connections.read().await;
        let timeout_duration =
            std::time::Duration::from_secs(crate::constants::WS_CONNECTION_TIMEOUT_SECS);

        for user_connections in connections.values() {
            for conn in user_connections {
                let last_ping = conn.last_ping.read().await;
                if last_ping.elapsed() > timeout_duration {
                    timeout_connections.push(conn.connection_id.clone());
                }
            }
        }

        timeout_connections
    }
}

impl Default for ConnectionManager {
    fn default() -> Self {
        Self::new()
    }
}
