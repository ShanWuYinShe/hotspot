/// WebSocket管理器
/// 
/// 迁移自传统ws模块，整合连接管理、房间管理、消息路由等功能
/// 适配新的端口适配器架构
use std::sync::Arc;
use std::collections::{HashMap, HashSet};
use axum::{
    extract::{WebSocketUpgrade, ws::{Message, WebSocket}},
    response::Response,
};
use futures::{sink::SinkExt, stream::StreamExt};
use serde_json;
use tracing::{info, error, debug, warn};
use uuid::Uuid;
use tokio::sync::{RwLock, mpsc};
use chrono::Utc;

use crate::domain::DomainResult;
use crate::presentation::ws::{WSMessageType, WSError};
use crate::ws::manager::WebSocketStats;
use crate::presentation::http::middleware::metrics::{
    record_ws_connection_opened, record_ws_connection_closed
};
use crate::ws::connection_manager::ConnectionStats;
use crate::ws::room_manager::RoomStats;
use crate::ports::UserRepository;

/// WebSocket连接信息
#[derive(Debug, Clone)]
pub struct ConnectionInfo {
    pub user_id: i64,
    pub connection_id: Uuid,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub last_activity: chrono::DateTime<chrono::Utc>,
}

/// 房间ID类型
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum RoomId {
    /// 用户私聊房间
    User(i64),
    /// 群组房间
    Group(i64),
    /// 系统通知房间
    System,
}

impl std::fmt::Display for RoomId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            RoomId::User(id) => write!(f, "user_{}", id),
            RoomId::Group(id) => write!(f, "group_{}", id),
            RoomId::System => write!(f, "system"),
        }
    }
}

impl RoomId {
    /// 从字符串解析房间ID
    pub fn from_string(s: &str) -> Option<Self> {
        if s == "system" {
            return Some(RoomId::System);
        }
        
        if let Some(id_str) = s.strip_prefix("user_") {
            if let Ok(id) = id_str.parse::<i64>() {
                return Some(RoomId::User(id));
            }
        }
        
        if let Some(id_str) = s.strip_prefix("group_") {
            if let Ok(id) = id_str.parse::<i64>() {
                return Some(RoomId::Group(id));
            }
        }
        
        None
    }
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
    /// 消息发送通道 connection_id -> Sender<WSMessageType>
    message_senders: Arc<RwLock<HashMap<Uuid, mpsc::UnboundedSender<WSMessageType>>>>,
}

impl Default for ConnectionManager {
    fn default() -> Self {
        Self::new()
    }
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
    
    /// 获取连接统计信息
    pub async fn get_stats(&self) -> ConnectionStats {
        let connections = self.connections.read().await;
        let user_connections = self.user_connections.read().await;
        
        ConnectionStats {
            total_connections: connections.len(),
            active_users: user_connections.len(),
        }
    }
    
    /// 添加新连接
    pub async fn add_connection(
        &self,
        connection_id: Uuid,
        user_id: i64,
        sender: mpsc::UnboundedSender<WSMessageType>,
    ) -> DomainResult<()> {
        // 记录连接打开监控
        record_ws_connection_opened();
        let connection_info = ConnectionInfo {
            user_id,
            connection_id,
            created_at: Utc::now(),
            last_activity: Utc::now(),
        };
        
        // 添加连接信息
        self.connections.write().await.insert(connection_id, connection_info);
        
        // 添加用户连接映射
        self.user_connections.write().await
            .entry(user_id)
            .or_insert_with(Vec::new)
            .push(connection_id);
        
        // 添加消息发送通道
        self.message_senders.write().await.insert(connection_id, sender);
        
        info!("添加WebSocket连接: {} (用户: {})", connection_id, user_id);
        Ok(())
    }
    
    /// 移除连接
    pub async fn remove_connection(&self, connection_id: Uuid) -> DomainResult<()> {
        // 记录连接关闭监控
        record_ws_connection_closed();
        
        let mut connections = self.connections.write().await;
        let mut user_connections = self.user_connections.write().await;
        let mut message_senders = self.message_senders.write().await;
        
        if let Some(connection_info) = connections.remove(&connection_id) {
            // 从用户连接映射中移除
            if let Some(user_conn_list) = user_connections.get_mut(&connection_info.user_id) {
                user_conn_list.retain(|&id| id != connection_id);
                if user_conn_list.is_empty() {
                    user_connections.remove(&connection_info.user_id);
                }
            }
            
            // 移除消息发送通道
            message_senders.remove(&connection_id);
            
            info!("移除WebSocket连接: {} (用户: {})", connection_id, connection_info.user_id);
        }
        
        Ok(())
    }
    
    /// 向用户的所有连接发送消息
    pub async fn send_to_user(&self, user_id: i64, message: WSMessageType) -> DomainResult<()> {
        let user_connections = self.user_connections.read().await;
        let message_senders = self.message_senders.read().await;
        
        if let Some(connection_ids) = user_connections.get(&user_id) {
            for &connection_id in connection_ids {
                if let Some(sender) = message_senders.get(&connection_id) {
                    if let Err(e) = sender.send(message.clone()) {
                        warn!("向连接 {} 发送消息失败: {:?}", connection_id, e);
                    }
                }
            }
        }
        
        Ok(())
    }
    
    /// 向特定连接发送消息
    pub async fn send_to_connection(&self, connection_id: Uuid, message: WSMessageType) -> DomainResult<()> {
        let message_senders = self.message_senders.read().await;
        
        if let Some(sender) = message_senders.get(&connection_id) {
            if let Err(e) = sender.send(message) {
                warn!("向连接 {} 发送消息失败: {:?}", connection_id, e);
            }
        }
        
        Ok(())
    }
    
    /// 获取在线用户数量
    pub async fn get_online_user_count(&self) -> usize {
        self.user_connections.read().await.len()
    }
    
    /// 获取连接总数
    pub async fn get_connection_count(&self) -> usize {
        self.connections.read().await.len()
    }
    
    /// 获取用户的连接列表
    pub async fn get_user_connections(&self, user_id: i64) -> Vec<Uuid> {
        self.user_connections.read().await
            .get(&user_id)
            .cloned()
            .unwrap_or_default()
    }
}

/// 房间管理器
/// 
/// 职责：
/// - 管理房间订阅关系
/// - 提供房间成员查询功能
#[derive(Debug)]
pub struct RoomManager {
    /// 房间订阅关系 room_id -> Set<connection_id>
    room_subscriptions: Arc<RwLock<HashMap<RoomId, HashSet<Uuid>>>>,
    /// 连接订阅关系 connection_id -> Set<room_id>
    connection_subscriptions: Arc<RwLock<HashMap<Uuid, HashSet<RoomId>>>>,
}

impl Default for RoomManager {
    fn default() -> Self {
        Self::new()
    }
}

impl RoomManager {
    /// 创建新的房间管理器
    pub fn new() -> Self {
        Self {
            room_subscriptions: Arc::new(RwLock::new(HashMap::new())),
            connection_subscriptions: Arc::new(RwLock::new(HashMap::new())),
        }
    }
    
    /// 获取房间统计信息
    pub async fn get_stats(&self) -> RoomStats {
        let room_subs = self.room_subscriptions.read().await;
        
        RoomStats {
            active_rooms: room_subs.len(),
            total_subscriptions: room_subs.values().map(|s| s.len()).sum(),
            connections_with_subscriptions: self.connection_subscriptions.read().await.len(),
        }
    }
    
    /// 订阅房间
    pub async fn subscribe(&self, connection_id: Uuid, room_id: RoomId) -> DomainResult<()> {
        let mut room_subs = self.room_subscriptions.write().await;
        let mut conn_subs = self.connection_subscriptions.write().await;
        
        // 添加房间订阅
        room_subs.entry(room_id.clone())
            .or_insert_with(HashSet::new)
            .insert(connection_id);
        
        // 添加连接订阅
        conn_subs.entry(connection_id)
            .or_insert_with(HashSet::new)
            .insert(room_id.clone());
        
        debug!("连接 {} 订阅房间: {}", connection_id, room_id);
        Ok(())
    }
    
    /// 取消订阅房间
    pub async fn unsubscribe(&self, connection_id: Uuid, room_id: &RoomId) -> DomainResult<()> {
        let mut room_subs = self.room_subscriptions.write().await;
        let mut conn_subs = self.connection_subscriptions.write().await;
        
        // 从房间订阅中移除
        if let Some(connections) = room_subs.get_mut(room_id) {
            connections.remove(&connection_id);
            if connections.is_empty() {
                room_subs.remove(room_id);
            }
        }
        
        // 从连接订阅中移除
        if let Some(rooms) = conn_subs.get_mut(&connection_id) {
            rooms.remove(room_id);
            if rooms.is_empty() {
                conn_subs.remove(&connection_id);
            }
        }
        
        debug!("连接 {} 取消订阅房间: {}", connection_id, room_id);
        Ok(())
    }
    
    /// 移除连接的所有订阅
    pub async fn remove_connection(&self, connection_id: Uuid) -> DomainResult<()> {
        let mut room_subs = self.room_subscriptions.write().await;
        let mut conn_subs = self.connection_subscriptions.write().await;
        
        // 获取连接订阅的所有房间
        if let Some(rooms) = conn_subs.remove(&connection_id) {
            // 从每个房间中移除该连接
            for room_id in rooms {
                if let Some(connections) = room_subs.get_mut(&room_id) {
                    connections.remove(&connection_id);
                    if connections.is_empty() {
                        room_subs.remove(&room_id);
                    }
                }
            }
        }
        
        Ok(())
    }
    
    /// 获取房间的所有订阅连接
    pub async fn get_room_connections(&self, room_id: &RoomId) -> Vec<Uuid> {
        self.room_subscriptions.read().await
            .get(room_id)
            .map(|connections| connections.iter().cloned().collect())
            .unwrap_or_default()
    }

    /// 向房间广播消息
    pub async fn broadcast_to_room(&self, room_id: &str, message: WSMessageType, connection_manager: &ConnectionManager) -> Result<(), WSError> {
        // 将字符串房间ID转换为RoomId枚举
        let room_id_enum = if room_id.starts_with("group_") {
            if let Ok(group_id) = room_id.strip_prefix("group_").unwrap_or("").parse::<i64>() {
                RoomId::Group(group_id)
            } else {
                return Err(WSError::InvalidRoomId { room_id: room_id.to_string() });
            }
        } else if room_id.starts_with("user_") {
            if let Ok(user_id) = room_id.strip_prefix("user_").unwrap_or("").parse::<i64>() {
                RoomId::User(user_id)
            } else {
                return Err(WSError::InvalidRoomId { room_id: room_id.to_string() });
            }
        } else if room_id == "system" {
            RoomId::System
        } else {
            return Err(WSError::InvalidRoomId { room_id: room_id.to_string() });
        };

        // 获取房间中的连接并广播消息
        let connections = self.get_room_connections(&room_id_enum).await;
        log::info!("向房间 {} 广播消息给 {} 个连接", room_id, connections.len());
        
        // 向房间中的所有连接发送消息
        for connection_id in connections {
            if let Err(e) = connection_manager.send_to_connection(connection_id, message.clone()).await {
                log::warn!("向连接 {} 发送消息失败: {:?}", connection_id, e);
            }
        }
        
        Ok(())
    }
}

/// WebSocket管理器
/// 
/// 整合连接管理、房间管理、消息路由等功能
pub struct WebSocketManager<U: UserRepository> {
    connection_manager: Arc<ConnectionManager>,
    room_manager: Arc<RoomManager>,
    user_repository: Arc<U>,
}

impl<U: UserRepository + 'static> WebSocketManager<U> {
    /// 创建新的WebSocket管理器
    pub fn new(user_repository: Arc<U>) -> Self {
        Self {
            connection_manager: Arc::new(ConnectionManager::new()),
            room_manager: Arc::new(RoomManager::new()),
            user_repository,
        }
    }
    
    /// 获取WebSocket统计信息
    pub async fn get_stats(&self) -> WebSocketStats {
        let connection_stats = self.connection_manager.get_stats().await;
        let room_stats = self.room_manager.get_stats().await;
        
        WebSocketStats {
            total_connections: connection_stats.total_connections,
            active_connections: connection_stats.total_connections, // 使用相同的值
            total_users: connection_stats.active_users,
            total_rooms: room_stats.active_rooms,
        }
    }
    
    /// 处理WebSocket升级请求
    pub async fn handle_upgrade(&self, ws: WebSocketUpgrade) -> Response {
        let connection_manager = self.connection_manager.clone();
        let room_manager = self.room_manager.clone();
        let user_repository = self.user_repository.clone();
        
        ws.on_upgrade(move |socket| {
            Self::handle_connection(socket, connection_manager, room_manager, user_repository)
        })
    }
    
    /// 处理单个WebSocket连接
    async fn handle_connection(
        socket: WebSocket,
        connection_manager: Arc<ConnectionManager>,
        room_manager: Arc<RoomManager>,
        user_repository: Arc<U>,
    ) {
        let (mut sender, mut receiver) = socket.split();
        let connection_id = Uuid::new_v4();
        
        info!("新的WebSocket连接建立: {}", connection_id);
        
        // 创建消息发送通道
        let (tx, mut rx) = mpsc::unbounded_channel::<WSMessageType>();
        
        // 启动消息发送任务
        let send_task = tokio::spawn(async move {
            while let Some(message) = rx.recv().await {
                if let Ok(json) = serde_json::to_string(&message) {
                    if let Err(e) = sender.send(Message::Text(json.into())).await {
                        error!("发送WebSocket消息失败: {:?}", e);
                        break;
                    }
                }
            }
        });
        
        // 等待认证消息
        let user_id = match Self::authenticate_connection(&mut receiver, &user_repository).await {
            Ok(user_id) => user_id,
            Err(e) => {
                error!("WebSocket认证失败: {:?}", e);
                let _ = tx.send(e.to_ws_message());
                return;
            }
        };
        
        // 注册连接
        if let Err(e) = connection_manager.add_connection(connection_id, user_id, tx.clone()).await {
            error!("注册WebSocket连接失败: {:?}", e);
            return;
        }
        
        // 发送认证成功消息
        let _ = tx.send(WSMessageType::SystemNotification {
            message: "认证成功".to_string(),
        });
        
        // 处理消息循环
        while let Some(msg) = receiver.next().await {
            match msg {
                Ok(Message::Text(text)) => {
                    if let Err(e) = Self::handle_text_message(
                        &text, 
                        user_id, 
                        connection_id,
                        &tx, 
                        &room_manager,
                        &connection_manager
                    ).await {
                        warn!("处理消息失败: {:?}", e);
                        let _ = tx.send(e.to_ws_message());
                    }
                }
                Ok(Message::Close(_)) => {
                    info!("WebSocket连接关闭: {}", connection_id);
                    break;
                }
                Ok(Message::Ping(_)) => {
                    // 响应ping
                    if let Err(e) = tx.send(WSMessageType::Pong) {
                        error!("发送pong失败: {:?}", e);
                        break;
                    }
                }
                Err(e) => {
                    error!("WebSocket错误: {:?}", e);
                    break;
                }
                _ => {}
            }
        }
        
        // 清理连接
        let _ = connection_manager.remove_connection(connection_id).await;
        let _ = room_manager.remove_connection(connection_id).await;
        send_task.abort();
        
        info!("WebSocket连接结束: {}", connection_id);
    }
    
    /// 认证WebSocket连接
    async fn authenticate_connection(
        receiver: &mut futures::stream::SplitStream<WebSocket>,
        user_repository: &Arc<U>,
    ) -> Result<i64, WSError> {
        // 等待认证消息，设置30秒超时
        let auth_timeout = tokio::time::Duration::from_secs(30);
        
        let auth_message = tokio::time::timeout(auth_timeout, async {
            while let Some(msg) = receiver.next().await {
                match msg {
                    Ok(Message::Text(text)) => return Ok(text),
                    Ok(_) => continue,
                    Err(e) => return Err(WSError::InternalError {
                        error: format!("接收消息错误: {:?}", e),
                    }),
                }
            }
            Err(WSError::ConnectionClosed)
        }).await;
        
        let text = match auth_message {
            Ok(Ok(text)) => text,
            Ok(Err(e)) => return Err(e),
            Err(_) => return Err(WSError::AuthenticationTimeout),
        };
        
        // 解析认证消息
        let auth_msg: WSMessageType = serde_json::from_str(&text)
            .map_err(|e| WSError::InvalidMessageFormat {
                message: format!("JSON解析错误: {}", e),
            })?;
        
        let token = match auth_msg {
            WSMessageType::Auth { token } => token,
            _ => return Err(WSError::AuthenticationFailed {
                reason: "期望认证消息".to_string(),
            }),
        };
        
        // 验证JWT令牌并获取用户ID
        let user_id = Self::verify_jwt_token(&token)?;
        
        // 验证用户是否存在
        match user_repository.as_ref().find_user_profile(user_id).await {
            Ok(Some(_)) => Ok(user_id),
            Ok(None) => Err(WSError::AuthenticationFailed {
                reason: "用户不存在".to_string(),
            }),
            Err(e) => Err(WSError::InternalError {
                error: format!("查询用户失败: {:?}", e),
            }),
        }
    }
    
    /// 验证JWT令牌
    fn verify_jwt_token(token: &str) -> Result<i64, WSError> {
        let jwt_secret = std::env::var("JWT_SECRET")
            .unwrap_or_else(|_| "your-secret-key-change-in-production".to_string());
        
        match crate::utils::jwt::verify_jwt_token(token, &jwt_secret) {
            Ok(claims) => Ok(claims.sub),
            Err(e) => {
                log::warn!("JWT验证失败: {}", e);
                Err(WSError::AuthenticationFailed {
                    reason: "无效的JWT令牌".to_string(),
                })
            }
        }
    }
    
    /// 处理文本消息
    async fn handle_text_message(
        text: &str,
        user_id: i64,
        connection_id: Uuid,
        sender: &mpsc::UnboundedSender<WSMessageType>,
        room_manager: &Arc<RoomManager>,
        connection_manager: &Arc<ConnectionManager>,
    ) -> Result<(), WSError> {
        let message: WSMessageType = serde_json::from_str(text)
            .map_err(|e| WSError::InvalidMessageFormat {
                message: format!("JSON解析错误: {}", e),
            })?;
        
        match message {
            WSMessageType::Ping => {
                let _ = sender.send(WSMessageType::Pong);
            }
            WSMessageType::Subscribe { room_id } => {
                if let Some(room) = RoomId::from_string(&room_id) {
                    room_manager.subscribe(connection_id, room).await
                        .map_err(|e| WSError::InternalError {
                            error: format!("订阅房间失败: {:?}", e),
                        })?;
                    debug!("用户 {} 订阅房间: {}", user_id, room_id);
                } else {
                    return Err(WSError::RoomNotFound { room_id });
                }
            }
            WSMessageType::Unsubscribe { room_id } => {
                if let Some(room) = RoomId::from_string(&room_id) {
                    room_manager.unsubscribe(connection_id, &room).await
                        .map_err(|e| WSError::InternalError {
                            error: format!("取消订阅房间失败: {:?}", e),
                        })?;
                    debug!("用户 {} 取消订阅房间: {}", user_id, room_id);
                } else {
                    return Err(WSError::RoomNotFound { room_id });
                }
            }
            WSMessageType::SendMessage { room_id, content } => {
                debug!("用户 {} 在房间 {} 发送消息: {}", user_id, room_id, content);
                // 简单实现：记录消息并向房间广播
                log::info!("消息发送: 用户{} -> 房间{}: {}", user_id, room_id, content);
                
                // 创建消息广播
                let broadcast_message = WSMessageType::MessageReceived {
                    room_id: room_id.clone(),
                    sender_id: user_id,
                    content: content.clone(),
                    timestamp: chrono::Utc::now().timestamp(),
                };
                
                // 向房间广播消息
                if let Err(e) = room_manager.broadcast_to_room(&room_id, broadcast_message, connection_manager).await {
                    log::error!("向房间 {} 广播消息失败: {:?}", room_id, e);
                }
            }
            _ => {
                return Err(WSError::InvalidMessageFormat {
                    message: "不支持的消息类型".to_string(),
                });
            }
        }
        
        Ok(())
    }
    
    /// 向用户发送消息
    pub async fn send_to_user(&self, user_id: i64, message: WSMessageType) -> DomainResult<()> {
        self.connection_manager.send_to_user(user_id, message).await
    }
    
    /// 向房间发送消息
    pub async fn send_to_room(&self, room_id: &RoomId, message: WSMessageType) -> DomainResult<()> {
        let connections = self.room_manager.get_room_connections(room_id).await;
        
        for connection_id in connections {
            let _ = self.connection_manager.send_to_connection(connection_id, message.clone()).await;
        }
        
        Ok(())
    }
    
    /// 获取在线用户数量
    pub async fn get_online_user_count(&self) -> usize {
        self.connection_manager.get_online_user_count().await
    }
    
    /// 获取连接总数
    pub async fn get_connection_count(&self) -> usize {
        self.connection_manager.get_connection_count().await
    }
}

impl<U: UserRepository + 'static> Clone for WebSocketManager<U> {
    fn clone(&self) -> Self {
        Self {
            connection_manager: self.connection_manager.clone(),
            room_manager: self.room_manager.clone(),
            user_repository: self.user_repository.clone(),
        }
    }
}
