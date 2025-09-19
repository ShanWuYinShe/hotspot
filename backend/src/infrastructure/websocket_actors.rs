/// WebSocket Actor系统 - 基于Actor模式的WebSocket管理
/// 
/// Actor类型：
/// - ConnectionActor: 管理单个WebSocket连接
/// - RoomActor: 管理房间和群组
/// - BroadcastActor: 处理广播消息
/// - PresenceActor: 管理用户在线状态
/// - NotificationActor: 处理通知分发

#[allow(dead_code)]

use async_trait::async_trait;
use std::collections::{HashMap, HashSet};
use std::sync::Arc;
use tokio::sync::{/* RwLock, */ Mutex}; // RwLock 暂时未使用
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use axum::extract::ws::{WebSocket, Message as WsMessage};
use futures::SinkExt;

use crate::domain::{DomainResult, DomainError, UserId, GroupId};
use crate::infrastructure::actor_system::{
    Actor, ActorMessage, ActorId, MessageEnvelope, ActorResponse,
};

// === WebSocket消息定义 ===

/// WebSocket消息类型
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type", content = "data")]
pub enum WebSocketMessageType {
    /// 连接建立
    Connected {
        connection_id: String,
        user_id: Option<UserId>,
    },
    /// 连接断开
    Disconnected {
        connection_id: String,
        reason: String,
    },
    /// 加入房间
    JoinRoom {
        room_id: String,
        user_id: UserId,
    },
    /// 离开房间
    LeaveRoom {
        room_id: String,
        user_id: UserId,
    },
    /// 发送消息到房间
    SendToRoom {
        room_id: String,
        message: String,
        sender_id: UserId,
    },
    /// 发送私人消息
    SendPrivate {
        recipient_id: UserId,
        message: String,
        sender_id: UserId,
    },
    /// 广播消息
    Broadcast {
        message: String,
        sender_id: Option<UserId>,
        exclude_users: Vec<UserId>,
    },
    /// 用户状态更新
    UserStatusUpdate {
        user_id: UserId,
        status: UserPresenceStatus,
    },
    /// 通知消息
    Notification {
        notification_id: String,
        recipient_id: UserId,
        title: String,
        content: String,
        notification_type: String,
    },
}

/// 用户在线状态
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum UserPresenceStatus {
    /// 在线
    Online,
    /// 离开
    Away,
    /// 忙碌
    Busy,
    /// 离线
    Offline,
}

/// WebSocket Actor消息
#[derive(Debug)]
pub struct WebSocketActorMessage {
    pub message_type: WebSocketMessageType,
    pub connection_id: Option<String>,
    pub timestamp: DateTime<Utc>,
}

impl ActorMessage for WebSocketActorMessage {
    fn message_type(&self) -> &'static str {
        match &self.message_type {
            WebSocketMessageType::Connected { .. } => "Connected",
            WebSocketMessageType::Disconnected { .. } => "Disconnected",
            WebSocketMessageType::JoinRoom { .. } => "JoinRoom",
            WebSocketMessageType::LeaveRoom { .. } => "LeaveRoom",
            WebSocketMessageType::SendToRoom { .. } => "SendToRoom",
            WebSocketMessageType::SendPrivate { .. } => "SendPrivate",
            WebSocketMessageType::Broadcast { .. } => "Broadcast",
            WebSocketMessageType::UserStatusUpdate { .. } => "UserStatusUpdate",
            WebSocketMessageType::Notification { .. } => "Notification",
        }
    }
    
    fn priority(&self) -> u8 {
        match &self.message_type {
            WebSocketMessageType::Connected { .. } | 
            WebSocketMessageType::Disconnected { .. } => 50, // 高优先级
            WebSocketMessageType::Notification { .. } => 80, // 中高优先级
            WebSocketMessageType::SendPrivate { .. } => 100, // 中等优先级
            WebSocketMessageType::SendToRoom { .. } => 120, // 中低优先级
            WebSocketMessageType::Broadcast { .. } => 150, // 低优先级
            _ => 128, // 默认优先级
        }
    }
    
    fn as_any(&self) -> &dyn std::any::Any {
        self
    }
}

// === 连接Actor ===

/// WebSocket连接Actor - 管理单个WebSocket连接
pub struct ConnectionActor {
    actor_id: ActorId,
    connection_id: String,
    user_id: Option<UserId>,
    websocket: Option<Arc<Mutex<WebSocket>>>,  // 使用Arc<Mutex<>>包装以实现Sync
    connected_at: DateTime<Utc>,
    last_activity: DateTime<Utc>,
    rooms: HashSet<String>,
    message_count: u64,
}

impl ConnectionActor {
    /// 创建新的连接Actor
    pub fn new(connection_id: String, websocket: WebSocket, user_id: Option<UserId>) -> Self {
        Self {
            actor_id: ActorId::new(),
            connection_id,
            user_id,
            websocket: Some(Arc::new(Mutex::new(websocket))),  // 使用Arc<Mutex<>>包装
            connected_at: Utc::now(),
            last_activity: Utc::now(),
            rooms: HashSet::new(),
            message_count: 0,
        }
    }
    
    /// 发送WebSocket消息
    async fn send_websocket_message(&mut self, message: &str) -> DomainResult<()> {
        if let Some(websocket_arc) = &self.websocket {
            let mut websocket = websocket_arc.lock().await;
            match websocket.send(WsMessage::Text(message.to_string().into())).await {
                Ok(()) => {
                    self.message_count += 1;
                    self.last_activity = Utc::now();
                    Ok(())
                }
                Err(e) => {
                    tracing::error!(
                        connection_id = %self.connection_id,
                        error = %e,
                        "Failed to send WebSocket message"
                    );
                    Err(DomainError::BusinessRuleViolation {
                        rule: format!("WebSocket send failed: {}", e),
                    })
                }
            }
        } else {
            Err(DomainError::BusinessRuleViolation {
                rule: "WebSocket connection not available".to_string(),
            })
        }
    }
    
    /// 加入房间
    async fn join_room(&mut self, room_id: String) -> DomainResult<()> {
        self.rooms.insert(room_id.clone());
        
        tracing::info!(
            connection_id = %self.connection_id,
            user_id = ?self.user_id,
            room_id = %room_id,
            "Connection joined room"
        );
        
        Ok(())
    }
    
    /// 离开房间
    async fn leave_room(&mut self, room_id: &str) -> DomainResult<()> {
        self.rooms.remove(room_id);
        
        tracing::info!(
            connection_id = %self.connection_id,
            user_id = ?self.user_id,
            room_id = %room_id,
            "Connection left room"
        );
        
        Ok(())
    }
    
    /// 获取连接统计信息
    pub fn get_connection_stats(&self) -> ConnectionStats {
        ConnectionStats {
            connection_id: self.connection_id.clone(),
            user_id: self.user_id,
            connected_at: self.connected_at,
            last_activity: self.last_activity,
            rooms_count: self.rooms.len(),
            message_count: self.message_count,
            is_active: Utc::now().signed_duration_since(self.last_activity).num_minutes() < 5,
        }
    }
}

#[async_trait]
impl Actor for ConnectionActor {
    fn actor_id(&self) -> ActorId {
        self.actor_id.clone()
    }
    
    fn actor_type(&self) -> &'static str {
        "ConnectionActor"
    }
    
    async fn handle_message(&mut self, envelope: MessageEnvelope) -> DomainResult<Option<ActorResponse>> {
        if let Some(ws_message) = envelope.message.as_any().downcast_ref::<WebSocketActorMessage>() {
            match &ws_message.message_type {
                WebSocketMessageType::JoinRoom { room_id, .. } => {
                    self.join_room(room_id.clone()).await?;
                    Ok(Some(ActorResponse::Success(None)))
                }
                
                WebSocketMessageType::LeaveRoom { room_id, .. } => {
                    self.leave_room(&room_id).await?;
                    Ok(Some(ActorResponse::Success(None)))
                }
                
                WebSocketMessageType::SendToRoom { message, .. } |
                WebSocketMessageType::SendPrivate { message, .. } |
                WebSocketMessageType::Notification { content: message, .. } => {
                    self.send_websocket_message(&message).await?;
                    Ok(Some(ActorResponse::Success(None)))
                }
                
                WebSocketMessageType::Disconnected { .. } => {
                    // 清理连接
                    self.websocket = None;
                    tracing::info!(
                        connection_id = %self.connection_id,
                        "Connection actor disconnected"
                    );
                    Ok(Some(ActorResponse::Success(None)))
                }
                
                _ => {
                    tracing::debug!(
                        connection_id = %self.connection_id,
                        message_type = ws_message.message_type(),
                        "Unhandled message type for connection actor"
                    );
                    Ok(None)
                }
            }
        } else {
            Err(DomainError::ValidationError {
                field: "message".to_string(),
                reason: "Invalid message type for ConnectionActor".to_string(),
            })
        }
    }
    
    async fn on_start(&mut self) -> DomainResult<()> {
        tracing::info!(
            connection_id = %self.connection_id,
            user_id = ?self.user_id,
            "Connection actor started"
        );
        Ok(())
    }
    
    async fn on_stop(&mut self) -> DomainResult<()> {
        // 清理WebSocket连接
        if let Some(websocket_arc) = &self.websocket {
            let mut websocket = websocket_arc.lock().await;
            let _ = websocket.close().await;
        }
        
        tracing::info!(
            connection_id = %self.connection_id,
            message_count = self.message_count,
            duration_minutes = Utc::now().signed_duration_since(self.connected_at).num_minutes(),
            "Connection actor stopped"
        );
        
        Ok(())
    }
}

/// 连接统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConnectionStats {
    pub connection_id: String,
    pub user_id: Option<UserId>,
    pub connected_at: DateTime<Utc>,
    pub last_activity: DateTime<Utc>,
    pub rooms_count: usize,
    pub message_count: u64,
    pub is_active: bool,
}

// === 房间Actor ===

/// 房间Actor - 管理房间成员和消息分发
pub struct RoomActor {
    actor_id: ActorId,
    room_id: String,
    room_type: RoomType,
    members: HashMap<UserId, String>, // user_id -> connection_id
    created_at: DateTime<Utc>,
    message_count: u64,
    max_members: Option<usize>,
}

/// 房间类型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RoomType {
    /// 群组房间
    Group { group_id: GroupId },
    /// 私聊房间
    Private { participants: Vec<UserId> },
    /// 系统房间
    System { name: String },
}

impl RoomActor {
    /// 创建新的房间Actor
    pub fn new(room_id: String, room_type: RoomType, max_members: Option<usize>) -> Self {
        Self {
            actor_id: ActorId::new(),
            room_id,
            room_type,
            members: HashMap::new(),
            created_at: Utc::now(),
            message_count: 0,
            max_members,
        }
    }
    
    /// 添加成员
    async fn add_member(&mut self, user_id: UserId, connection_id: String) -> DomainResult<()> {
        // 检查房间容量
        if let Some(max) = self.max_members {
            if self.members.len() >= max {
                return Err(DomainError::BusinessRuleViolation {
                    rule: format!("Room {} is full (max: {})", self.room_id, max),
                });
            }
        }
        
        self.members.insert(user_id, connection_id.clone());
        
        tracing::info!(
            room_id = %self.room_id,
            user_id = %user_id.value(),
            connection_id = %connection_id,
            member_count = self.members.len(),
            "User joined room"
        );
        
        Ok(())
    }
    
    /// 移除成员
    async fn remove_member(&mut self, user_id: UserId) -> DomainResult<()> {
        if let Some(connection_id) = self.members.remove(&user_id) {
            tracing::info!(
                room_id = %self.room_id,
                user_id = %user_id.value(),
                connection_id = %connection_id,
                member_count = self.members.len(),
                "User left room"
            );
        }
        
        Ok(())
    }
    
    /// 广播消息到房间所有成员
    async fn broadcast_to_room(
        &mut self,
        _message: &str,
        sender_id: Option<UserId>,
        exclude_users: &[UserId],
    ) -> DomainResult<u32> {
        let mut sent_count = 0;
        let exclude_set: HashSet<_> = exclude_users.iter().collect();
        
        for (user_id, connection_id) in &self.members {
            // 跳过排除的用户
            if exclude_set.contains(user_id) {
                continue;
            }
            
            // 跳过发送者（如果指定）
            if let Some(sender) = sender_id {
                if *user_id == sender {
                    continue;
                }
            }
            
            // 这里需要向连接Actor发送消息
            // 在实际实现中，需要通过ActorContext获取连接Actor的引用
            tracing::debug!(
                room_id = %self.room_id,
                user_id = %user_id.value(),
                connection_id = %connection_id,
                "Broadcasting message to room member"
            );
            
            sent_count += 1;
        }
        
        self.message_count += 1;
        
        tracing::info!(
            room_id = %self.room_id,
            sent_count = sent_count,
            total_members = self.members.len(),
            "Message broadcasted to room"
        );
        
        Ok(sent_count)
    }
    
    /// 获取房间统计信息
    pub fn get_room_stats(&self) -> RoomStats {
        RoomStats {
            room_id: self.room_id.clone(),
            room_type: self.room_type.clone(),
            member_count: self.members.len(),
            message_count: self.message_count,
            created_at: self.created_at,
            max_members: self.max_members,
        }
    }
}

#[async_trait]
impl Actor for RoomActor {
    fn actor_id(&self) -> ActorId {
        self.actor_id.clone()
    }
    
    fn actor_type(&self) -> &'static str {
        "RoomActor"
    }
    
    async fn handle_message(&mut self, envelope: MessageEnvelope) -> DomainResult<Option<ActorResponse>> {
        if let Some(ws_message) = envelope.message.as_any().downcast_ref::<WebSocketActorMessage>() {
            match &ws_message.message_type {
                WebSocketMessageType::JoinRoom { user_id, .. } => {
                    let connection_id = ws_message.connection_id
                        .clone()
                        .unwrap_or_else(|| "unknown".to_string());
                    
                    self.add_member(*user_id, connection_id).await?;
                    Ok(Some(ActorResponse::Success(None)))
                }
                
                WebSocketMessageType::LeaveRoom { user_id, .. } => {
                    self.remove_member(*user_id).await?;
                    Ok(Some(ActorResponse::Success(None)))
                }
                
                WebSocketMessageType::SendToRoom { message, sender_id, .. } => {
                    let sent_count = self.broadcast_to_room(&message, Some(*sender_id), &[]).await?;
                    
                    Ok(Some(ActorResponse::Success(Some(serde_json::json!({
                        "sent_count": sent_count
                    })))))
                }
                
                _ => {
                    tracing::debug!(
                        room_id = %self.room_id,
                        message_type = ws_message.message_type(),
                        "Unhandled message type for room actor"
                    );
                    Ok(None)
                }
            }
        } else {
            Err(DomainError::ValidationError {
                field: "message".to_string(),
                reason: "Invalid message type for RoomActor".to_string(),
            })
        }
    }
    
    async fn on_start(&mut self) -> DomainResult<()> {
        tracing::info!(
            room_id = %self.room_id,
            room_type = ?self.room_type,
            max_members = ?self.max_members,
            "Room actor started"
        );
        Ok(())
    }
    
    async fn on_stop(&mut self) -> DomainResult<()> {
        tracing::info!(
            room_id = %self.room_id,
            member_count = self.members.len(),
            message_count = self.message_count,
            duration_minutes = Utc::now().signed_duration_since(self.created_at).num_minutes(),
            "Room actor stopped"
        );
        Ok(())
    }
}

/// 房间统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RoomStats {
    pub room_id: String,
    pub room_type: RoomType,
    pub member_count: usize,
    pub message_count: u64,
    pub created_at: DateTime<Utc>,
    pub max_members: Option<usize>,
}

// === 在线状态Actor ===

/// 在线状态Actor - 管理用户在线状态
pub struct PresenceActor {
    actor_id: ActorId,
    user_presence: HashMap<UserId, UserPresence>,
    status_change_count: u64,
}

/// 用户在线信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserPresence {
    pub user_id: UserId,
    pub status: UserPresenceStatus,
    pub last_seen: DateTime<Utc>,
    pub connection_count: u32,
    pub current_rooms: HashSet<String>,
}

impl PresenceActor {
    /// 创建新的在线状态Actor
    pub fn new() -> Self {
        Self {
            actor_id: ActorId::new(),
            user_presence: HashMap::new(),
            status_change_count: 0,
        }
    }
    
    /// 更新用户状态
    async fn update_user_status(&mut self, user_id: UserId, status: UserPresenceStatus) -> DomainResult<()> {
        let presence = self.user_presence.entry(user_id).or_insert_with(|| UserPresence {
            user_id,
            status: UserPresenceStatus::Offline,
            last_seen: Utc::now(),
            connection_count: 0,
            current_rooms: HashSet::new(),
        });
        
        let old_status = presence.status.clone();
        presence.status = status.clone();
        presence.last_seen = Utc::now();
        
        if old_status != status {
            self.status_change_count += 1;
            
            tracing::info!(
                user_id = %user_id.value(),
                old_status = ?old_status,
                new_status = ?status,
                "User status updated"
            );
        }
        
        Ok(())
    }
    
    /// 用户连接
    async fn user_connected(&mut self, user_id: UserId, room_id: Option<String>) -> DomainResult<()> {
        let presence = self.user_presence.entry(user_id).or_insert_with(|| UserPresence {
            user_id,
            status: UserPresenceStatus::Online,
            last_seen: Utc::now(),
            connection_count: 0,
            current_rooms: HashSet::new(),
        });
        
        presence.connection_count += 1;
        presence.status = UserPresenceStatus::Online;
        presence.last_seen = Utc::now();
        
        if let Some(room) = room_id {
            presence.current_rooms.insert(room);
        }
        
        tracing::info!(
            user_id = %user_id.value(),
            connection_count = presence.connection_count,
            "User connected"
        );
        
        Ok(())
    }
    
    /// 用户断开连接
    async fn user_disconnected(&mut self, user_id: UserId, room_id: Option<String>) -> DomainResult<()> {
        if let Some(presence) = self.user_presence.get_mut(&user_id) {
            if presence.connection_count > 0 {
                presence.connection_count -= 1;
            }
            
            if let Some(room) = room_id {
                presence.current_rooms.remove(&room);
            }
            
            // 如果没有活跃连接，设置为离线
            if presence.connection_count == 0 {
                presence.status = UserPresenceStatus::Offline;
            }
            
            presence.last_seen = Utc::now();
            
            tracing::info!(
                user_id = %user_id.value(),
                connection_count = presence.connection_count,
                status = ?presence.status,
                "User disconnected"
            );
        }
        
        Ok(())
    }
    
    /// 获取在线用户列表
    pub fn get_online_users(&self) -> Vec<UserPresence> {
        self.user_presence
            .values()
            .filter(|p| p.status != UserPresenceStatus::Offline)
            .cloned()
            .collect()
    }
    
    /// 获取在线状态统计
    pub fn get_presence_stats(&self) -> PresenceStats {
        let mut stats = PresenceStats {
            total_users: self.user_presence.len(),
            online_users: 0,
            away_users: 0,
            busy_users: 0,
            offline_users: 0,
            status_change_count: self.status_change_count,
        };
        
        for presence in self.user_presence.values() {
            match presence.status {
                UserPresenceStatus::Online => stats.online_users += 1,
                UserPresenceStatus::Away => stats.away_users += 1,
                UserPresenceStatus::Busy => stats.busy_users += 1,
                UserPresenceStatus::Offline => stats.offline_users += 1,
            }
        }
        
        stats
    }
}

#[async_trait]
impl Actor for PresenceActor {
    fn actor_id(&self) -> ActorId {
        self.actor_id.clone()
    }
    
    fn actor_type(&self) -> &'static str {
        "PresenceActor"
    }
    
    async fn handle_message(&mut self, envelope: MessageEnvelope) -> DomainResult<Option<ActorResponse>> {
        if let Some(ws_message) = envelope.message.as_any().downcast_ref::<WebSocketActorMessage>() {
            match &ws_message.message_type {
                WebSocketMessageType::Connected { user_id: Some(user_id), .. } => {
                    self.user_connected(*user_id, None).await?;
                    Ok(Some(ActorResponse::Success(None)))
                }
                
                WebSocketMessageType::Disconnected { .. } => {
                    // 需要从消息中提取user_id，这里简化处理
                    Ok(Some(ActorResponse::Success(None)))
                }
                
                WebSocketMessageType::UserStatusUpdate { user_id, status } => {
                    self.update_user_status(*user_id, status.clone()).await?;
                    Ok(Some(ActorResponse::Success(None)))
                }
                
                WebSocketMessageType::JoinRoom { room_id, user_id } => {
                    self.user_connected(*user_id, Some(room_id.clone())).await?;
                    Ok(Some(ActorResponse::Success(None)))
                }
                
                WebSocketMessageType::LeaveRoom { room_id, user_id } => {
                    self.user_disconnected(*user_id, Some(room_id.clone())).await?;
                    Ok(Some(ActorResponse::Success(None)))
                }
                
                _ => Ok(None)
            }
        } else {
            Err(DomainError::ValidationError {
                field: "message".to_string(),
                reason: "Invalid message type for PresenceActor".to_string(),
            })
        }
    }
    
    async fn on_start(&mut self) -> DomainResult<()> {
        tracing::info!("Presence actor started");
        Ok(())
    }
    
    async fn on_stop(&mut self) -> DomainResult<()> {
        tracing::info!(
            total_users = self.user_presence.len(),
            status_changes = self.status_change_count,
            "Presence actor stopped"
        );
        Ok(())
    }
}

/// 在线状态统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PresenceStats {
    pub total_users: usize,
    pub online_users: usize,
    pub away_users: usize,
    pub busy_users: usize,
    pub offline_users: usize,
    pub status_change_count: u64,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_websocket_message_priority() {
        let connected_msg = WebSocketActorMessage {
            message_type: WebSocketMessageType::Connected {
                connection_id: "test".to_string(),
                user_id: None,
            },
            connection_id: None,
            timestamp: Utc::now(),
        };
        
        assert_eq!(connected_msg.priority(), 50);
        assert_eq!(connected_msg.message_type(), "Connected");
    }
    
    #[test]
    fn test_room_actor_creation() {
        let room_type = RoomType::Group {
            group_id: GroupId::new("test-group".to_string()).unwrap(),
        };
        
        let room_actor = RoomActor::new(
            "test_room".to_string(),
            room_type,
            Some(100),
        );
        
        assert_eq!(room_actor.room_id, "test_room");
        assert_eq!(room_actor.members.len(), 0);
        assert_eq!(room_actor.max_members, Some(100));
    }
    
    #[test]
    fn test_presence_actor_stats() {
        let presence_actor = PresenceActor::new();
        let stats = presence_actor.get_presence_stats();
        
        assert_eq!(stats.total_users, 0);
        assert_eq!(stats.online_users, 0);
        assert_eq!(stats.offline_users, 0);
    }
}
