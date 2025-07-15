//! WebSocket房间管理器
//! 
//! 专门负责房间订阅和取消订阅管理

use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;
use uuid::Uuid;
use tracing::{debug, info};

use crate::ws::{
    message_types::RoomId,
    errors::WsResult,
};

/// 房间管理器
/// 
/// 职责：
/// - 管理房间订阅关系
/// - 维护房间到连接的映射
/// - 提供房间查询和统计功能
#[derive(Debug)]
pub struct RoomManager {
    /// 房间订阅 room_id -> Vec<connection_id>
    room_subscriptions: Arc<RwLock<HashMap<RoomId, Vec<Uuid>>>>,
    /// 连接订阅的房间 connection_id -> Vec<room_id>
    connection_rooms: Arc<RwLock<HashMap<Uuid, Vec<RoomId>>>>,
}

impl RoomManager {
    /// 创建新的房间管理器
    pub fn new() -> Self {
        Self {
            room_subscriptions: Arc::new(RwLock::new(HashMap::new())),
            connection_rooms: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    /// 订阅房间
    pub async fn subscribe_to_room(&self, connection_id: Uuid, room_id: RoomId) -> WsResult<()> {
        // 更新房间订阅映射
        {
            let mut room_subscriptions = self.room_subscriptions.write().await;
            let subscribers = room_subscriptions.entry(room_id.clone()).or_insert_with(Vec::new);
            if !subscribers.contains(&connection_id) {
                subscribers.push(connection_id);
            }
        }

        // 更新连接房间映射
        {
            let mut connection_rooms = self.connection_rooms.write().await;
            let rooms = connection_rooms.entry(connection_id).or_insert_with(Vec::new);
            if !rooms.contains(&room_id) {
                rooms.push(room_id.clone());
            }
        }

        debug!("连接已订阅房间: connection_id={}, room_id={:?}", connection_id, room_id);
        Ok(())
    }

    /// 取消订阅房间
    pub async fn unsubscribe_from_room(&self, connection_id: Uuid, room_id: &RoomId) -> WsResult<()> {
        // 更新房间订阅映射
        {
            let mut room_subscriptions = self.room_subscriptions.write().await;
            if let Some(subscribers) = room_subscriptions.get_mut(room_id) {
                subscribers.retain(|&id| id != connection_id);
                if subscribers.is_empty() {
                    room_subscriptions.remove(room_id);
                }
            }
        }

        // 更新连接房间映射
        {
            let mut connection_rooms = self.connection_rooms.write().await;
            if let Some(rooms) = connection_rooms.get_mut(&connection_id) {
                rooms.retain(|r| r != room_id);
                if rooms.is_empty() {
                    connection_rooms.remove(&connection_id);
                }
            }
        }

        debug!("连接已取消订阅房间: connection_id={}, room_id={:?}", connection_id, room_id);
        Ok(())
    }

    /// 连接断开时清理所有房间订阅
    #[allow(dead_code)]
    pub async fn cleanup_connection_subscriptions(&self, connection_id: Uuid) -> Vec<RoomId> {
        let rooms = {
            let mut connection_rooms = self.connection_rooms.write().await;
            connection_rooms.remove(&connection_id).unwrap_or_default()
        };

        // 从所有房间中移除该连接
        {
            let mut room_subscriptions = self.room_subscriptions.write().await;
            let mut empty_rooms = Vec::new();

            for room_id in &rooms {
                if let Some(subscribers) = room_subscriptions.get_mut(room_id) {
                    subscribers.retain(|&id| id != connection_id);
                    if subscribers.is_empty() {
                        empty_rooms.push(room_id.clone());
                    }
                }
            }

            // 清理空房间
            for room_id in empty_rooms {
                room_subscriptions.remove(&room_id);
            }
        }

        if !rooms.is_empty() {
            info!("已清理连接 {} 的房间订阅: {:?}", connection_id, rooms);
        }

        rooms
    }

    /// 获取房间的所有订阅者
    pub async fn get_room_subscribers(&self, room_id: &RoomId) -> Vec<Uuid> {
        let room_subscriptions = self.room_subscriptions.read().await;
        room_subscriptions.get(room_id).cloned().unwrap_or_default()
    }

    /// 获取连接订阅的所有房间
    pub async fn get_connection_rooms(&self, connection_id: Uuid) -> Vec<RoomId> {
        let connection_rooms = self.connection_rooms.read().await;
        connection_rooms.get(&connection_id).cloned().unwrap_or_default()
    }

    /// 检查连接是否订阅了指定房间
    #[allow(dead_code)]
    pub async fn is_subscribed(&self, connection_id: Uuid, room_id: &RoomId) -> bool {
        let connection_rooms = self.connection_rooms.read().await;
        if let Some(rooms) = connection_rooms.get(&connection_id) {
            rooms.contains(room_id)
        } else {
            false
        }
    }

    /// 获取房间统计信息
    #[allow(dead_code)]
    pub async fn get_stats(&self) -> RoomStats {
        let room_subscriptions = self.room_subscriptions.read().await;
        let connection_rooms = self.connection_rooms.read().await;

        let total_subscriptions: usize = room_subscriptions.values().map(|v| v.len()).sum();

        RoomStats {
            active_rooms: room_subscriptions.len(),
            total_subscriptions,
            connections_with_subscriptions: connection_rooms.len(),
        }
    }

    /// 清理空房间
    #[allow(dead_code)]
    pub async fn cleanup_empty_rooms(&self) -> usize {
        let mut room_subscriptions = self.room_subscriptions.write().await;
        let initial_count = room_subscriptions.len();
        
        room_subscriptions.retain(|_, subscribers| !subscribers.is_empty());
        
        let removed_count = initial_count - room_subscriptions.len();
        if removed_count > 0 {
            info!("已清理 {} 个空房间", removed_count);
        }
        
        removed_count
    }
}

/// 房间统计信息
#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct RoomStats {
    pub active_rooms: usize,
    pub total_subscriptions: usize,
    pub connections_with_subscriptions: usize,
}

impl Default for RoomManager {
    fn default() -> Self {
        Self::new()
    }
}
