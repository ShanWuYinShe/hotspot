//! 统一WebSocket连接管理器
//! 
//! 提供完整的WebSocket连接管理功能，包括：
//! - 连接生命周期管理
//! - 用户认证和会话管理
//! - 房间管理和消息路由
//! - 心跳检测和连接监控

use std::collections::{HashMap, HashSet};
use std::sync::Arc;
use tokio::sync::{RwLock, mpsc, broadcast};
use tokio::time::Duration;
use axum::extract::ws::{WebSocket, Message};
use futures_util::{SinkExt, StreamExt};
use tracing::{info, error, debug};
use serde_json;
use uuid::Uuid;
use chrono::Utc;

use super::unified_message_types::*;
use crate::domain::DomainResult;

/// WebSocket连接管理器
pub struct UnifiedConnectionManager {
    /// 活跃连接映射：连接ID -> 连接信息
    connections: Arc<RwLock<HashMap<String, ConnectionInfo>>>,
    /// 用户连接映射：用户ID -> 连接ID集合
    user_connections: Arc<RwLock<HashMap<String, HashSet<String>>>>,
    /// 房间管理：房间ID -> 房间信息
    rooms: Arc<RwLock<HashMap<String, RoomInfo>>>,
    /// 消息发送通道：连接ID -> 发送通道
    senders: Arc<RwLock<HashMap<String, mpsc::UnboundedSender<Message>>>>,
    /// 广播通道（用于全局消息）
    broadcast_tx: broadcast::Sender<ServerMessage>,
    /// 连接统计
    stats: Arc<RwLock<ConnectionStats>>,
    /// 心跳检测间隔
    heartbeat_interval: Duration,
    /// 连接超时时间
    connection_timeout: Duration,
}

impl UnifiedConnectionManager {
    /// 创建新的连接管理器
    pub fn new() -> Self {
        let (broadcast_tx, _) = broadcast::channel(1000);
        
        Self {
            connections: Arc::new(RwLock::new(HashMap::new())),
            user_connections: Arc::new(RwLock::new(HashMap::new())),
            rooms: Arc::new(RwLock::new(HashMap::new())),
            senders: Arc::new(RwLock::new(HashMap::new())),
            broadcast_tx,
            stats: Arc::new(RwLock::new(ConnectionStats {
                total_connections: 0,
                authenticated_connections: 0,
                active_rooms: 0,
                messages_sent_per_minute: 0.0,
                messages_received_per_minute: 0.0,
                average_latency_ms: 0.0,
                error_rate: 0.0,
            })),
            heartbeat_interval: Duration::from_secs(30),
            connection_timeout: Duration::from_secs(300), // 5分钟
        }
    }
    
    /// 处理新的WebSocket连接
    pub async fn handle_connection(&self, socket: WebSocket, connection_id: String) {
        let session_id = Uuid::new_v4().to_string();
        let connection_info = ConnectionInfo::new(connection_id.clone(), session_id.clone());
        
        // 注册连接
        {
            let mut connections = self.connections.write().await;
            connections.insert(connection_id.clone(), connection_info);
        }
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.total_connections += 1;
        }
        
        info!("新WebSocket连接建立: {}", connection_id);
        
        // 分离发送和接收
        let (mut sender, mut receiver) = socket.split();
        let (tx, mut rx) = mpsc::unbounded_channel();
        
        // 注册发送通道
        {
            let mut senders = self.senders.write().await;
            senders.insert(connection_id.clone(), tx);
        }
        
        // 发送连接确认消息
        let connected_msg = ServerMessage::Connected {
            user_id: "".to_string(), // 未认证时为空
            session_id: session_id.clone(),
        };
        
        if let Err(e) = self.send_to_connection(&connection_id, connected_msg).await {
            error!("发送连接确认消息失败: {}", e);
        }
        
        // 克隆必要的数据用于异步任务
        let manager = Arc::new(self.clone());
        let connection_id_clone = connection_id.clone();
        
        // 启动发送任务
        let send_task = {
            let connection_id = connection_id.clone();
            tokio::spawn(async move {
                while let Some(message) = rx.recv().await {
                    if let Err(e) = sender.send(message).await {
                        error!("发送消息失败 (连接: {}): {}", connection_id, e);
                        break;
                    }
                }
            })
        };
        
        // 启动接收任务
        let receive_task = {
            let manager = manager.clone();
            let connection_id = connection_id.clone();
            tokio::spawn(async move {
                while let Some(msg) = receiver.next().await {
                    match msg {
                        Ok(Message::Text(text)) => {
                            if let Err(e) = manager.handle_text_message(&connection_id, text.to_string()).await {
                                error!("处理文本消息失败: {}", e);
                            }
                        }
                        Ok(Message::Binary(data)) => {
                            debug!("收到二进制消息 (连接: {}): {} bytes", connection_id, data.len());
                        }
                        Ok(Message::Ping(data)) => {
                            if let Err(e) = manager.handle_ping(&connection_id, data.to_vec()).await {
                                error!("处理Ping消息失败: {}", e);
                            }
                        }
                        Ok(Message::Pong(_)) => {
                            manager.handle_pong(&connection_id).await;
                        }
                        Ok(Message::Close(_)) => {
                            info!("连接关闭 (连接: {})", connection_id);
                            break;
                        }
                        Err(e) => {
                            error!("WebSocket错误 (连接: {}): {}", connection_id, e);
                            break;
                        }
                    }
                }
            })
        };
        
        // 启动心跳检测任务
        let heartbeat_task = {
            let manager = manager.clone();
            let connection_id = connection_id.clone();
            tokio::spawn(async move {
                let mut interval = tokio::time::interval(manager.heartbeat_interval);
                loop {
                    interval.tick().await;
                    if !manager.is_connection_alive(&connection_id).await {
                        info!("连接心跳超时，断开连接: {}", connection_id);
                        break;
                    }
                }
            })
        };
        
        // 等待任务完成
        tokio::select! {
            _ = send_task => {},
            _ = receive_task => {},
            _ = heartbeat_task => {},
        }
        
        // 清理连接
        self.cleanup_connection(&connection_id_clone).await;
    }
    
    /// 处理文本消息
    async fn handle_text_message(&self, connection_id: &str, text: String) -> DomainResult<()> {
        debug!("收到文本消息 (连接: {}): {}", connection_id, text);
        
        // 解析客户端消息
        let client_message: ClientMessage = serde_json::from_str(&text)
            .map_err(|e| crate::domain::DomainError::ValidationError { field: "message".to_string(), reason: format!("消息格式错误: {}", e) })?;
        
        // 处理不同类型的消息
        match client_message {
            ClientMessage::Auth { token } => {
                self.handle_auth(connection_id, token).await?;
            }
            ClientMessage::Ping => {
                self.send_pong(connection_id).await?;
            }
            ClientMessage::JoinGroup { group_id } => {
                self.handle_join_group(connection_id, group_id).await?;
            }
            ClientMessage::LeaveGroup { group_id } => {
                self.handle_leave_group(connection_id, group_id).await?;
            }
            ClientMessage::SendMessage { 
                group_id, content, message_type, reply_to_message_id, 
                mentioned_users, latitude, longitude 
            } => {
                self.handle_send_message(
                    connection_id, group_id, content, message_type,
                    reply_to_message_id, mentioned_users, latitude, longitude
                ).await?;
            }
            ClientMessage::MarkRead { group_id, message_id } => {
                self.handle_mark_read(connection_id, group_id, message_id).await?;
            }
            ClientMessage::Typing { group_id, is_typing } => {
                self.handle_typing(connection_id, group_id, is_typing).await?;
            }
            ClientMessage::UpdateStatus { status, location } => {
                self.handle_update_status(connection_id, status, location).await?;
            }
        }
        
        Ok(())
    }
    
    /// 处理用户认证
    async fn handle_auth(&self, connection_id: &str, token: String) -> DomainResult<()> {
        // TODO: 实现JWT token验证
        // 这里应该调用认证服务验证token并获取用户信息
        
        // 临时实现：假设token格式为 "user_id:session"
        let parts: Vec<&str> = token.split(':').collect();
        if parts.len() != 2 {
            let error_msg = ServerMessage::AuthError {
                error: "Invalid token format".to_string(),
            };
            self.send_to_connection(connection_id, error_msg).await?;
            return Ok(());
        }
        
        let user_id = parts[0].to_string();
        let nickname = format!("User_{}", user_id); // 临时昵称
        
        // 更新连接信息
        {
            let mut connections = self.connections.write().await;
            if let Some(conn) = connections.get_mut(connection_id) {
                conn.authenticate(user_id.clone());
            }
        }
        
        // 更新用户连接映射
        {
            let mut user_connections = self.user_connections.write().await;
            user_connections
                .entry(user_id.clone())
                .or_insert_with(HashSet::new)
                .insert(connection_id.to_string());
        }
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.authenticated_connections += 1;
        }
        
        // 发送认证成功消息
        let success_msg = ServerMessage::AuthSuccess {
            user_id: user_id.clone(),
            nickname,
        };
        self.send_to_connection(connection_id, success_msg).await?;
        
        // 自动加入用户私有房间
        self.join_user_room(connection_id, &user_id).await?;
        
        info!("用户认证成功: {} (连接: {})", user_id, connection_id);
        Ok(())
    }
    
    /// 处理加入群组
    async fn handle_join_group(&self, connection_id: &str, group_id: String) -> DomainResult<()> {
        // 获取用户ID
        let user_id = {
            let connections = self.connections.read().await;
            connections.get(connection_id)
                .and_then(|conn| conn.user_id.clone())
                .ok_or_else(|| crate::domain::DomainError::AuthenticationError { reason: "未认证的连接".to_string() })?
        };
        
        // 加入群组房间
        self.join_room(connection_id, &group_id, RoomType::Group).await?;
        
        // 更新连接的订阅群组列表
        {
            let mut connections = self.connections.write().await;
            if let Some(conn) = connections.get_mut(connection_id) {
                conn.join_group(group_id.clone());
            }
        }
        
        info!("用户 {} 加入群组 {}", user_id, group_id);
        Ok(())
    }
    
    /// 处理离开群组
    async fn handle_leave_group(&self, connection_id: &str, group_id: String) -> DomainResult<()> {
        // 获取用户ID
        let user_id = {
            let connections = self.connections.read().await;
            connections.get(connection_id)
                .and_then(|conn| conn.user_id.clone())
                .ok_or_else(|| crate::domain::DomainError::AuthenticationError { reason: "未认证的连接".to_string() })?
        };
        
        // 离开群组房间
        self.leave_room(connection_id, &group_id).await?;
        
        // 更新连接的订阅群组列表
        {
            let mut connections = self.connections.write().await;
            if let Some(conn) = connections.get_mut(connection_id) {
                conn.leave_group(&group_id);
            }
        }
        
        info!("用户 {} 离开群组 {}", user_id, group_id);
        Ok(())
    }
    
    /// 处理发送消息
    async fn handle_send_message(
        &self,
        connection_id: &str,
        group_id: String,
        content: String,
        message_type: String,
        reply_to_message_id: Option<String>,
        mentioned_users: Option<Vec<String>>,
        latitude: Option<f64>,
        longitude: Option<f64>,
    ) -> DomainResult<()> {
        // 获取发送者信息
        let (user_id, nickname) = {
            let connections = self.connections.read().await;
            let conn = connections.get(connection_id)
                .ok_or_else(|| crate::domain::DomainError::AuthenticationError { reason: "连接不存在".to_string() })?;
            
            let user_id = conn.user_id.clone()
                .ok_or_else(|| crate::domain::DomainError::AuthenticationError { reason: "未认证的连接".to_string() })?;
            
            // TODO: 从数据库获取真实昵称
            let nickname = format!("User_{}", user_id);
            
            (user_id, nickname)
        };
        
        // 创建消息载荷
        let message_payload = MessagePayload {
            message_id: Uuid::new_v4().to_string(),
            group_id: group_id.clone(),
            sender_id: user_id,
            sender_nickname: nickname,
            sender_avatar: None, // TODO: 从数据库获取头像
            content,
            message_type,
            reply_to_message_id,
            mentioned_users: mentioned_users.unwrap_or_default(),
            created_at: Utc::now(),
            edited_at: None,
            latitude,
            longitude,
            media_attachments: Vec::new(), // TODO: 处理媒体附件
        };
        
        // 广播消息到群组
        let new_message = ServerMessage::NewMessage {
            message: message_payload,
        };
        
        self.broadcast_to_room(&group_id, new_message, Some(connection_id)).await?;
        
        Ok(())
    }
    
    /// 处理标记已读
    async fn handle_mark_read(
        &self,
        connection_id: &str,
        group_id: String,
        message_id: Option<String>,
    ) -> DomainResult<()> {
        // 获取用户信息
        let user_id = {
            let connections = self.connections.read().await;
            connections.get(connection_id)
                .and_then(|conn| conn.user_id.clone())
                .ok_or_else(|| crate::domain::DomainError::AuthenticationError { reason: "未认证的连接".to_string() })?
        };
        
        // 广播已读状态
        if let Some(msg_id) = message_id {
            let read_message = ServerMessage::MessageRead {
                group_id: group_id.clone(),
                user_id,
                message_id: msg_id,
                read_at: Utc::now(),
            };
            
            self.broadcast_to_room(&group_id, read_message, Some(connection_id)).await?;
        }
        
        Ok(())
    }
    
    /// 处理正在输入状态
    async fn handle_typing(
        &self,
        connection_id: &str,
        group_id: String,
        is_typing: bool,
    ) -> DomainResult<()> {
        // 获取用户信息
        let (user_id, nickname) = {
            let connections = self.connections.read().await;
            let conn = connections.get(connection_id)
                .ok_or_else(|| crate::domain::DomainError::AuthenticationError { reason: "连接不存在".to_string() })?;
            
            let user_id = conn.user_id.clone()
                .ok_or_else(|| crate::domain::DomainError::AuthenticationError { reason: "未认证的连接".to_string() })?;
            
            let nickname = format!("User_{}", user_id);
            
            (user_id, nickname)
        };
        
        // 广播正在输入状态
        let typing_message = ServerMessage::TypingIndicator {
            group_id: group_id.clone(),
            user_id,
            nickname,
            is_typing,
        };
        
        self.broadcast_to_room(&group_id, typing_message, Some(connection_id)).await?;
        
        Ok(())
    }
    
    /// 处理用户状态更新
    async fn handle_update_status(
        &self,
        connection_id: &str,
        status: UserStatus,
        location: Option<LocationUpdate>,
    ) -> DomainResult<()> {
        // 获取用户信息
        let (user_id, nickname) = {
            let connections = self.connections.read().await;
            let conn = connections.get(connection_id)
                .ok_or_else(|| crate::domain::DomainError::AuthenticationError { reason: "连接不存在".to_string() })?;
            
            let user_id = conn.user_id.clone()
                .ok_or_else(|| crate::domain::DomainError::AuthenticationError { reason: "未认证的连接".to_string() })?;
            
            let nickname = format!("User_{}", user_id);
            
            (user_id, nickname)
        };
        
        // 广播用户状态更新
        match status {
            UserStatus::Online => {
                let online_message = ServerMessage::UserOnline {
                    user_id: user_id.clone(),
                    nickname: nickname.clone(),
                    status,
                };
                self.broadcast_to_user_contacts(&user_id, online_message).await?;
            }
            UserStatus::Offline => {
                let offline_message = ServerMessage::UserOffline {
                    user_id: user_id.clone(),
                    nickname,
                };
                self.broadcast_to_user_contacts(&user_id, offline_message).await?;
            }
            _ => {
                let status_message = ServerMessage::UserOnline {
                    user_id: user_id.clone(),
                    nickname,
                    status,
                };
                self.broadcast_to_user_contacts(&user_id, status_message).await?;
            }
        }
        
        // 如果有位置更新，广播位置信息
        if let Some(loc) = location {
            let location_message = ServerMessage::UserLocationUpdate {
                user_id: user_id.clone(),
                nickname: format!("User_{}", user_id),
                location: loc,
            };
            self.broadcast_to_user_contacts(&user_id, location_message).await?;
        }
        
        Ok(())
    }
    
    /// 发送Pong消息
    async fn send_pong(&self, connection_id: &str) -> DomainResult<()> {
        let pong_message = ServerMessage::Pong;
        self.send_to_connection(connection_id, pong_message).await
    }
    
    /// 处理Ping消息
    async fn handle_ping(&self, connection_id: &str, _data: Vec<u8>) -> DomainResult<()> {
        // 更新心跳时间
        {
            let mut connections = self.connections.write().await;
            if let Some(conn) = connections.get_mut(connection_id) {
                conn.update_ping();
            }
        }
        
        // 发送Pong响应
        let senders = self.senders.read().await;
        if let Some(sender) = senders.get(connection_id) {
            let _ = sender.send(Message::Pong(Vec::new().into()));
        }
        
        Ok(())
    }
    
    /// 处理Pong消息
    async fn handle_pong(&self, connection_id: &str) {
        // 更新心跳时间
        let mut connections = self.connections.write().await;
        if let Some(conn) = connections.get_mut(connection_id) {
            conn.update_ping();
        }
    }
    
    /// 检查连接是否存活
    async fn is_connection_alive(&self, connection_id: &str) -> bool {
        let connections = self.connections.read().await;
        if let Some(conn) = connections.get(connection_id) {
            if let Some(last_ping) = conn.last_ping_at {
                let elapsed = Utc::now().signed_duration_since(last_ping);
                elapsed.num_seconds() < self.connection_timeout.as_secs() as i64
            } else {
                // 如果从未收到心跳，检查连接时间
                let elapsed = Utc::now().signed_duration_since(conn.connected_at);
                elapsed.num_seconds() < self.connection_timeout.as_secs() as i64
            }
        } else {
            false
        }
    }
    
    /// 加入房间
    async fn join_room(&self, connection_id: &str, room_id: &str, room_type: RoomType) -> DomainResult<()> {
        let user_id = {
            let connections = self.connections.read().await;
            connections.get(connection_id)
                .and_then(|conn| conn.user_id.clone())
                .ok_or_else(|| crate::domain::DomainError::AuthenticationError { reason: "未认证的连接".to_string() })?
        };
        
        // 创建或更新房间
        {
            let mut rooms = self.rooms.write().await;
            let room = rooms.entry(room_id.to_string())
                .or_insert_with(|| RoomInfo::new(room_id.to_string(), room_type));
            room.add_user(user_id);
        }
        
        Ok(())
    }
    
    /// 离开房间
    async fn leave_room(&self, connection_id: &str, room_id: &str) -> DomainResult<()> {
        let user_id = {
            let connections = self.connections.read().await;
            connections.get(connection_id)
                .and_then(|conn| conn.user_id.clone())
                .ok_or_else(|| crate::domain::DomainError::AuthenticationError { reason: "未认证的连接".to_string() })?
        };
        
        // 从房间移除用户
        {
            let mut rooms = self.rooms.write().await;
            if let Some(room) = rooms.get_mut(room_id) {
                room.remove_user(&user_id);
                
                // 如果房间为空，删除房间
                if room.connected_users.is_empty() {
                    rooms.remove(room_id);
                }
            }
        }
        
        Ok(())
    }
    
    /// 加入用户私有房间
    async fn join_user_room(&self, connection_id: &str, user_id: &str) -> DomainResult<()> {
        let room_id = format!("user_{}", user_id);
        self.join_room(connection_id, &room_id, RoomType::User).await
    }
    
    /// 发送消息到指定连接
    async fn send_to_connection(&self, connection_id: &str, message: ServerMessage) -> DomainResult<()> {
        let senders = self.senders.read().await;
        if let Some(sender) = senders.get(connection_id) {
            let json_message = serde_json::to_string(&message)
                .map_err(|e| crate::domain::DomainError::SerializationError { reason: e.to_string() })?;
            
            let _ = sender.send(Message::Text(json_message.into()));
        }
        
        Ok(())
    }
    
    /// 广播消息到房间
    async fn broadcast_to_room(&self, room_id: &str, message: ServerMessage, exclude_connection: Option<&str>) -> DomainResult<()> {
        let room_users = {
            let rooms = self.rooms.read().await;
            rooms.get(room_id)
                .map(|room| room.connected_users.clone())
                .unwrap_or_default()
        };
        
        for user_id in room_users {
            self.send_to_user(&user_id, message.clone(), exclude_connection).await?;
        }
        
        Ok(())
    }
    
    /// 发送消息到用户的所有连接
    async fn send_to_user(&self, user_id: &str, message: ServerMessage, exclude_connection: Option<&str>) -> DomainResult<()> {
        let user_connections = {
            let user_connections = self.user_connections.read().await;
            user_connections.get(user_id).cloned().unwrap_or_default()
        };
        
        for connection_id in user_connections {
            if let Some(exclude) = exclude_connection {
                if connection_id == exclude {
                    continue;
                }
            }
            
            self.send_to_connection(&connection_id, message.clone()).await?;
        }
        
        Ok(())
    }
    
    /// 广播消息到用户的联系人
    async fn broadcast_to_user_contacts(&self, user_id: &str, message: ServerMessage) -> DomainResult<()> {
        // TODO: 实现获取用户联系人列表的逻辑
        // 这里应该查询数据库获取用户的好友列表，然后向他们广播消息
        
        // 临时实现：只发送给用户自己
        self.send_to_user(user_id, message, None).await
    }
    
    /// 清理连接
    async fn cleanup_connection(&self, connection_id: &str) {
        let user_id = {
            let mut connections = self.connections.write().await;
            connections.remove(connection_id)
                .and_then(|conn| conn.user_id)
        };
        
        // 清理发送通道
        {
            let mut senders = self.senders.write().await;
            senders.remove(connection_id);
        }
        
        // 清理用户连接映射
        let has_user = user_id.is_some();
        if let Some(user_id) = user_id {
            let mut user_connections = self.user_connections.write().await;
            if let Some(connections) = user_connections.get_mut(&user_id) {
                connections.remove(connection_id);
                if connections.is_empty() {
                    user_connections.remove(&user_id);
                }
            }
        }
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.total_connections = stats.total_connections.saturating_sub(1);
            if has_user {
                stats.authenticated_connections = stats.authenticated_connections.saturating_sub(1);
            }
        }
        
        info!("连接已清理: {}", connection_id);
    }
    
    /// 获取连接统计
    pub async fn get_stats(&self) -> ConnectionStats {
        let stats = self.stats.read().await;
        let rooms = self.rooms.read().await;
        
        ConnectionStats {
            total_connections: stats.total_connections,
            authenticated_connections: stats.authenticated_connections,
            active_rooms: rooms.len(),
            messages_sent_per_minute: stats.messages_sent_per_minute,
            messages_received_per_minute: stats.messages_received_per_minute,
            average_latency_ms: stats.average_latency_ms,
            error_rate: stats.error_rate,
        }
    }
    
    /// 获取房间统计
    pub async fn get_room_stats(&self) -> Vec<RoomStats> {
        let rooms = self.rooms.read().await;
        rooms.values().map(|room| RoomStats {
            room_id: room.room_id.clone(),
            room_type: room.room_type.clone(),
            connected_users: room.connected_users.len(),
            messages_per_minute: 0.0, // TODO: 实现消息统计
            created_at: room.created_at,
            last_activity_at: room.last_activity_at,
        }).collect()
    }
}

// 实现Clone trait以支持Arc共享
impl Clone for UnifiedConnectionManager {
    fn clone(&self) -> Self {
        Self {
            connections: self.connections.clone(),
            user_connections: self.user_connections.clone(),
            rooms: self.rooms.clone(),
            senders: self.senders.clone(),
            broadcast_tx: self.broadcast_tx.clone(),
            stats: self.stats.clone(),
            heartbeat_interval: self.heartbeat_interval,
            connection_timeout: self.connection_timeout,
        }
    }
}
