/// WebSocket管理器 - 基于Actor系统的WebSocket服务
/// 
/// 管理功能：
/// - Actor系统初始化
/// - 连接生命周期管理
/// - 房间管理
/// - 消息路由
/// - 在线状态管理

use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;
use axum::extract::ws::WebSocket;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

use crate::domain::{DomainResult, DomainError, UserId};
use crate::infrastructure::{
    ActorSystem, ActorRef,
    ConnectionActor, RoomActor, PresenceActor,
    WebSocketActorMessage, WebSocketMessageType, UserPresenceStatus, RoomType,
    ConnectionStats, RoomStats, PresenceStats,
};

/// WebSocket管理器配置
#[derive(Debug, Clone)]
pub struct WebSocketManagerConfig {
    /// 最大连接数
    pub max_connections: usize,
    /// 最大房间数
    pub max_rooms: usize,
    /// 连接超时时间（秒）
    pub connection_timeout_seconds: u64,
    /// 心跳间隔（秒）
    pub heartbeat_interval_seconds: u64,
    /// 是否启用在线状态管理
    pub enable_presence: bool,
    /// 是否启用消息持久化
    pub enable_message_persistence: bool,
}

impl Default for WebSocketManagerConfig {
    fn default() -> Self {
        Self {
            max_connections: 10000,
            max_rooms: 1000,
            connection_timeout_seconds: 300, // 5分钟
            heartbeat_interval_seconds: 30,
            enable_presence: true,
            enable_message_persistence: false,
        }
    }
}

/// WebSocket管理器 - 基于Actor系统的WebSocket服务
pub struct WebSocketManager {
    /// Actor系统
    actor_system: Arc<ActorSystem>,
    /// 配置
    config: WebSocketManagerConfig,
    /// 连接映射 (connection_id -> actor_ref)
    connections: Arc<RwLock<HashMap<String, ActorRef>>>,
    /// 用户连接映射 (user_id -> connection_ids)
    user_connections: Arc<RwLock<HashMap<UserId, Vec<String>>>>,
    /// 房间映射 (room_id -> actor_ref)
    rooms: Arc<RwLock<HashMap<String, ActorRef>>>,
    /// 在线状态Actor
    presence_actor: Option<ActorRef>,
    /// 管理器统计
    stats: Arc<RwLock<WebSocketManagerStats>>,
}

/// WebSocket管理器统计
#[derive(Debug, Default, Clone, Serialize, Deserialize)]
pub struct WebSocketManagerStats {
    /// 总连接数
    pub total_connections: u64,
    /// 当前活跃连接数
    pub active_connections: u64,
    /// 总房间数
    pub total_rooms: u64,
    /// 当前活跃房间数
    pub active_rooms: u64,
    /// 已处理消息数
    pub messages_processed: u64,
    /// 连接错误数
    pub connection_errors: u64,
    /// 启动时间
    pub started_at: DateTime<Utc>,
}

impl WebSocketManager {
    /// 创建新的WebSocket管理器
    pub async fn new(config: WebSocketManagerConfig) -> DomainResult<Arc<Self>> {
        let actor_system = ActorSystem::new("websocket_system");
        
        // 创建在线状态Actor
        let presence_actor = if config.enable_presence {
            let presence = PresenceActor::new();
            Some(actor_system.spawn_actor(presence).await?)
        } else {
            None
        };
        
        let manager = Arc::new(Self {
            actor_system,
            config: config.clone(),
            connections: Arc::new(RwLock::new(HashMap::new())),
            user_connections: Arc::new(RwLock::new(HashMap::new())),
            rooms: Arc::new(RwLock::new(HashMap::new())),
            presence_actor,
            stats: Arc::new(RwLock::new(WebSocketManagerStats {
                started_at: Utc::now(),
                ..Default::default()
            })),
        });
        
        // 启动清理任务
        manager.start_cleanup_task().await;
        
        tracing::info!(
            max_connections = config.max_connections,
            max_rooms = config.max_rooms,
            enable_presence = config.enable_presence,
            "WebSocket manager initialized"
        );
        
        Ok(manager)
    }
    
    /// 处理新的WebSocket连接
    pub async fn handle_connection(
        &self,
        connection_id: String,
        websocket: WebSocket,
        user_id: Option<UserId>,
    ) -> DomainResult<()> {
        // 检查连接数限制
        {
            let connections = self.connections.read().await;
            if connections.len() >= self.config.max_connections {
                return Err(DomainError::BusinessRuleViolation {
                    rule: format!("Maximum connections limit reached: {}", self.config.max_connections),
                });
            }
        }
        
        // 创建连接Actor
        let connection_actor = ConnectionActor::new(
            connection_id.clone(),
            websocket,
            user_id,
        );
        
        let actor_ref = self.actor_system.spawn_actor(connection_actor).await?;
        
        // 注册连接
        {
            let mut connections = self.connections.write().await;
            connections.insert(connection_id.clone(), actor_ref.clone());
        }
        
        // 注册用户连接映射
        if let Some(uid) = user_id {
            let mut user_connections = self.user_connections.write().await;
            user_connections
                .entry(uid)
                .or_insert_with(Vec::new)
                .push(connection_id.clone());
        }
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.total_connections += 1;
            stats.active_connections += 1;
        }
        
        // 通知在线状态Actor
        if let Some(presence_ref) = &self.presence_actor {
            if let Some(uid) = user_id {
                let message = WebSocketActorMessage {
                    message_type: WebSocketMessageType::Connected {
                        connection_id: connection_id.clone(),
                        user_id: Some(uid),
                    },
                    connection_id: Some(connection_id.clone()),
                    timestamp: Utc::now(),
                };
                
                let _ = presence_ref.send(Box::new(message)).await;
            }
        }
        
        tracing::info!(
            connection_id = %connection_id,
            user_id = ?user_id,
            "WebSocket connection established"
        );
        
        Ok(())
    }
    
    /// 断开连接
    pub async fn disconnect(&self, connection_id: &str, reason: &str) -> DomainResult<()> {
        let actor_ref = {
            let mut connections = self.connections.write().await;
            connections.remove(connection_id)
        };
        
        if let Some(actor_ref) = actor_ref {
            // 发送断开消息
            let message = WebSocketActorMessage {
                message_type: WebSocketMessageType::Disconnected {
                    connection_id: connection_id.to_string(),
                    reason: reason.to_string(),
                },
                connection_id: Some(connection_id.to_string()),
                timestamp: Utc::now(),
            };
            
            let _ = actor_ref.send(Box::new(message)).await;
            
            // 停止Actor
            self.actor_system.stop_actor(actor_ref.actor_id()).await?;
            
            // 清理用户连接映射
            {
                let mut user_connections = self.user_connections.write().await;
                user_connections.retain(|_, connections| {
                    connections.retain(|conn_id| conn_id != connection_id);
                    !connections.is_empty()
                });
            }
            
            // 更新统计
            {
                let mut stats = self.stats.write().await;
                if stats.active_connections > 0 {
                    stats.active_connections -= 1;
                }
            }
            
            tracing::info!(
                connection_id = %connection_id,
                reason = %reason,
                "WebSocket connection disconnected"
            );
        }
        
        Ok(())
    }
    
    /// 创建或获取房间
    pub async fn get_or_create_room(
        &self,
        room_id: &str,
        room_type: RoomType,
        max_members: Option<usize>,
    ) -> DomainResult<ActorRef> {
        // 检查房间是否已存在
        {
            let rooms = self.rooms.read().await;
            if let Some(room_ref) = rooms.get(room_id) {
                return Ok(room_ref.clone());
            }
        }
        
        // 检查房间数限制
        {
            let rooms = self.rooms.read().await;
            if rooms.len() >= self.config.max_rooms {
                return Err(DomainError::BusinessRuleViolation {
                    rule: format!("Maximum rooms limit reached: {}", self.config.max_rooms),
                });
            }
        }
        
        // 创建新房间
        let room_actor = RoomActor::new(
            room_id.to_string(),
            room_type,
            max_members,
        );
        
        let actor_ref = self.actor_system.spawn_actor(room_actor).await?;
        
        // 注册房间
        {
            let mut rooms = self.rooms.write().await;
            rooms.insert(room_id.to_string(), actor_ref.clone());
        }
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.total_rooms += 1;
            stats.active_rooms += 1;
        }
        
        tracing::info!(
            room_id = %room_id,
            max_members = ?max_members,
            "Room created"
        );
        
        Ok(actor_ref)
    }
    
    /// 用户加入房间
    pub async fn join_room(&self, user_id: UserId, room_id: &str) -> DomainResult<()> {
        // 获取用户的连接
        let connection_ids = {
            let user_connections = self.user_connections.read().await;
            user_connections.get(&user_id).cloned().unwrap_or_default()
        };
        
        if connection_ids.is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("User {} has no active connections", user_id.value()),
            });
        }
        
        // 获取房间Actor
        let room_ref = {
            let rooms = self.rooms.read().await;
            rooms.get(room_id).cloned()
        };
        
        let room_ref = room_ref.ok_or_else(|| DomainError::BusinessRuleViolation {
            rule: format!("Room {} not found", room_id),
        })?;
        
        // 发送加入房间消息
        let message = WebSocketActorMessage {
            message_type: WebSocketMessageType::JoinRoom {
                room_id: room_id.to_string(),
                user_id,
            },
            connection_id: connection_ids.first().cloned(),
            timestamp: Utc::now(),
        };
        
        room_ref.send(Box::new(message)).await?;
        
        // 通知用户的所有连接
        for connection_id in &connection_ids {
            if let Some(connection_ref) = {
                let connections = self.connections.read().await;
                connections.get(connection_id).cloned()
            } {
                let join_message = WebSocketActorMessage {
                    message_type: WebSocketMessageType::JoinRoom {
                        room_id: room_id.to_string(),
                        user_id,
                    },
                    connection_id: Some(connection_id.clone()),
                    timestamp: Utc::now(),
                };
                
                let _ = connection_ref.send(Box::new(join_message)).await;
            }
        }
        
        tracing::info!(
            user_id = %user_id.value(),
            room_id = %room_id,
            connection_count = connection_ids.len(),
            "User joined room"
        );
        
        Ok(())
    }
    
    /// 用户离开房间
    pub async fn leave_room(&self, user_id: UserId, room_id: &str) -> DomainResult<()> {
        // 获取房间Actor
        let room_ref = {
            let rooms = self.rooms.read().await;
            rooms.get(room_id).cloned()
        };
        
        if let Some(room_ref) = room_ref {
            let message = WebSocketActorMessage {
                message_type: WebSocketMessageType::LeaveRoom {
                    room_id: room_id.to_string(),
                    user_id,
                },
                connection_id: None,
                timestamp: Utc::now(),
            };
            
            room_ref.send(Box::new(message)).await?;
            
            tracing::info!(
                user_id = %user_id.value(),
                room_id = %room_id,
                "User left room"
            );
        }
        
        Ok(())
    }
    
    /// 发送消息到房间
    pub async fn send_to_room(
        &self,
        room_id: &str,
        message: &str,
        sender_id: UserId,
    ) -> DomainResult<()> {
        let room_ref = {
            let rooms = self.rooms.read().await;
            rooms.get(room_id).cloned()
        };
        
        let room_ref = room_ref.ok_or_else(|| DomainError::BusinessRuleViolation {
            rule: format!("Room {} not found", room_id),
        })?;
        
        let ws_message = WebSocketActorMessage {
            message_type: WebSocketMessageType::SendToRoom {
                room_id: room_id.to_string(),
                message: message.to_string(),
                sender_id,
            },
            connection_id: None,
            timestamp: Utc::now(),
        };
        
        room_ref.send(Box::new(ws_message)).await?;
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.messages_processed += 1;
        }
        
        tracing::debug!(
            room_id = %room_id,
            sender_id = %sender_id.value(),
            message_length = message.len(),
            "Message sent to room"
        );
        
        Ok(())
    }
    
    /// 发送私人消息
    pub async fn send_private_message(
        &self,
        recipient_id: UserId,
        message: &str,
        sender_id: UserId,
    ) -> DomainResult<()> {
        // 获取接收者的连接
        let connection_ids = {
            let user_connections = self.user_connections.read().await;
            user_connections.get(&recipient_id).cloned().unwrap_or_default()
        };
        
        if connection_ids.is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("Recipient {} is not online", recipient_id.value()),
            });
        }
        
        // 发送到接收者的所有连接
        for connection_id in &connection_ids {
            if let Some(connection_ref) = {
                let connections = self.connections.read().await;
                connections.get(connection_id).cloned()
            } {
                let ws_message = WebSocketActorMessage {
                    message_type: WebSocketMessageType::SendPrivate {
                        recipient_id,
                        message: message.to_string(),
                        sender_id,
                    },
                    connection_id: Some(connection_id.clone()),
                    timestamp: Utc::now(),
                };
                
                let _ = connection_ref.send(Box::new(ws_message)).await;
            }
        }
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.messages_processed += 1;
        }
        
        tracing::debug!(
            sender_id = %sender_id.value(),
            recipient_id = %recipient_id.value(),
            message_length = message.len(),
            connection_count = connection_ids.len(),
            "Private message sent"
        );
        
        Ok(())
    }
    
    /// 更新用户状态
    pub async fn update_user_status(&self, user_id: UserId, status: UserPresenceStatus) -> DomainResult<()> {
        if let Some(presence_ref) = &self.presence_actor {
            let message = WebSocketActorMessage {
                message_type: WebSocketMessageType::UserStatusUpdate {
                    user_id,
                    status: status.clone(),
                },
                connection_id: None,
                timestamp: Utc::now(),
            };
            
            presence_ref.send(Box::new(message)).await?;
            
            tracing::debug!(
                user_id = %user_id.value(),
                status = ?status,
                "User status updated"
            );
        }
        
        Ok(())
    }
    
    /// 获取管理器统计信息
    pub async fn get_stats(&self) -> WebSocketManagerStats {
        self.stats.read().await.clone()
    }
    
    /// 获取所有连接统计
    pub async fn get_connection_stats(&self) -> Vec<ConnectionStats> {
        // 这里需要向所有连接Actor查询统计信息
        // 简化实现，返回空列表
        Vec::new()
    }
    
    /// 获取所有房间统计
    pub async fn get_room_stats(&self) -> Vec<RoomStats> {
        // 这里需要向所有房间Actor查询统计信息
        // 简化实现，返回空列表
        Vec::new()
    }
    
    /// 获取在线状态统计
    pub async fn get_presence_stats(&self) -> Option<PresenceStats> {
        // 这里需要向在线状态Actor查询统计信息
        // 简化实现，返回None
        None
    }
    
    /// 关闭管理器
    pub async fn shutdown(&self) -> DomainResult<()> {
        tracing::info!("Shutting down WebSocket manager");
        
        // 断开所有连接
        let connection_ids: Vec<String> = {
            let connections = self.connections.read().await;
            connections.keys().cloned().collect()
        };
        
        for connection_id in connection_ids {
            let _ = self.disconnect(&connection_id, "Server shutdown").await;
        }
        
        // 关闭Actor系统
        self.actor_system.shutdown().await?;
        
        tracing::info!("WebSocket manager shutdown completed");
        Ok(())
    }
    
    /// 启动清理任务
    async fn start_cleanup_task(&self) {
        let connections = self.connections.clone();
        let rooms = self.rooms.clone();
        let stats = self.stats.clone();
        let cleanup_interval = std::time::Duration::from_secs(60); // 每分钟清理一次
        
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(cleanup_interval);
            
            loop {
                interval.tick().await;
                
                // 清理断开的连接
                let mut disconnected_connections: Vec<String> = Vec::new();
                {
                    let connections_guard = connections.read().await;
                    for (connection_id, _) in connections_guard.iter() {
                        // 这里应该检查连接是否仍然活跃
                        // 简化实现，暂不清理
                    }
                }
                
                // 清理空房间
                let mut empty_rooms: Vec<String> = Vec::new();
                {
                    let rooms_guard = rooms.read().await;
                    for (room_id, _) in rooms_guard.iter() {
                        // 这里应该检查房间是否为空
                        // 简化实现，暂不清理
                    }
                }
                
                if !disconnected_connections.is_empty() || !empty_rooms.is_empty() {
                    tracing::debug!(
                        disconnected_connections = disconnected_connections.len(),
                        empty_rooms = empty_rooms.len(),
                        "Cleanup task completed"
                    );
                }
            }
        });
    }
}

/// WebSocket管理器健康检查
pub struct WebSocketManagerHealthChecker {
    manager: Arc<WebSocketManager>,
}

impl WebSocketManagerHealthChecker {
    /// 创建新的健康检查器
    pub fn new(manager: Arc<WebSocketManager>) -> Self {
        Self { manager }
    }
    
    /// 检查健康状态
    pub async fn check_health(&self) -> WebSocketManagerHealth {
        let stats = self.manager.get_stats().await;
        let actor_health = self.manager.actor_system.get_health().await;
        
        let is_healthy = actor_health.is_healthy && 
                        stats.active_connections < self.manager.config.max_connections as u64 &&
                        stats.active_rooms < self.manager.config.max_rooms as u64;
        
        WebSocketManagerHealth {
            is_healthy,
            active_connections: stats.active_connections,
            active_rooms: stats.active_rooms,
            messages_processed: stats.messages_processed,
            connection_errors: stats.connection_errors,
            actor_system_health: actor_health,
            uptime_seconds: Utc::now()
                .signed_duration_since(stats.started_at)
                .num_seconds() as u64,
        }
    }
}

/// WebSocket管理器健康状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebSocketManagerHealth {
    pub is_healthy: bool,
    pub active_connections: u64,
    pub active_rooms: u64,
    pub messages_processed: u64,
    pub connection_errors: u64,
    pub actor_system_health: crate::infrastructure::actor_system::ActorSystemHealth,
    pub uptime_seconds: u64,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_websocket_manager_config() {
        let config = WebSocketManagerConfig::default();
        assert_eq!(config.max_connections, 10000);
        assert_eq!(config.max_rooms, 1000);
        assert!(config.enable_presence);
    }
    
    #[tokio::test]
    async fn test_websocket_manager_creation() {
        let config = WebSocketManagerConfig::default();
        let manager = WebSocketManager::new(config).await.unwrap();
        
        let stats = manager.get_stats().await;
        assert_eq!(stats.active_connections, 0);
        assert_eq!(stats.active_rooms, 0);
    }
}
