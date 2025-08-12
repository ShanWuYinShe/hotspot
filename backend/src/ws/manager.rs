//! WebSocket管理器
//! 
//! 专注于WebSocket连接的基础管理，不处理业务逻辑的消息通知
//! 
//! ## 职责分工
//! 
//! ### WebSocketManager (本模块)
//! - 连接生命周期管理 (添加/移除连接)
//! - 房间订阅/取消订阅管理
//! - 基础协议处理 (ping/pong)
//! - 连接状态查询和统计
//! - 内部事件日志记录
//! 
//! ### WebSocketNotificationService 
//! - 业务逻辑相关的WebSocket通知
//! - 消息、群组事件、系统通知等的发送
//! - 由Controllers调用来发送特定的业务通知
//! 
//! ## 使用建议
//! - Controllers应该使用 `WebSocketNotificationService` 发送业务通知
//! - 仅在需要连接管理或状态查询时直接使用 `WebSocketManager`
//! - 避免在业务逻辑中直接调用 `WebSocketManager` 的消息发送方法

use crate::ws::{
    constants::connection::MAX_CONNECTIONS,
    errors::{WsError, WsResult},
    message_types::{ClientMessage, ServerMessage, InternalEvent, RoomId},
    connection_manager::ConnectionManager,
    room_manager::RoomManager,
    message_router::MessageRouter,
};
use crate::utils::permission::PermissionService;
#[cfg(feature = "audit")]
use crate::utils::audit::{AuditService, AuditEvent, NoopAuditSink};
use crate::services::notification::WebSocketNotificationService;
use crate::ws::hooks::{WsHookService, NoopWsHooks, WsHooks};
use std::sync::Arc;
use tokio::sync::mpsc;
use uuid::Uuid;
use tracing::{info, debug, error, warn};
use crate::utils::metrics::AppMetrics;
use sqlx::PgPool;
#[cfg(feature = "ws_offline_notifications")]
#[derive(Clone)]
struct WebSocketManagerForTasks {
    notification_service: Arc<WebSocketNotificationService>,
}

#[cfg(feature = "ws_offline_notifications")]
impl WebSocketManagerForTasks {
    async fn deliver_offline_notifications(&self, user_id: i64) {
        if let Ok(notifications) = self.notification_service.get_offline_notifications(user_id).await {
            for n in notifications {
                if let Ok(value) = serde_json::from_str::<serde_json::Value>(&n) {
                    if let Ok(data) = serde_json::from_value::<crate::ws::message_types::SystemNotificationData>(value.clone()) {
                        let _ = self.notification_service.notify_system_event_to_user(user_id, data).await;
                    }
                }
            }
            let _ = self.notification_service.clear_offline_notifications(user_id).await;
        }
    }
}

/// WebSocket管理器统计信息
#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct WebSocketStats {
    /// 总连接数
    pub total_connections: usize,
    /// 活跃连接数
    pub active_connections: usize,
    /// 总用户数
    pub total_users: usize,
    /// 总房间数
    pub total_rooms: usize,
}

/// WebSocket管理器
/// 
/// 专注于WebSocket连接的基础管理功能：
/// - 连接生命周期管理 (添加/移除连接)
/// - 房间订阅/取消订阅
/// - 基础消息路由 (ping/pong, 房间管理)
/// - 连接状态统计
/// 
/// 注意：业务逻辑的WebSocket通知现在由 WebSocketNotificationService 处理
pub struct WebSocketManager {
    /// 连接管理器
    connection_manager: Arc<ConnectionManager>,
    /// 房间管理器
    room_manager: Arc<RoomManager>,
    /// 消息路由器
    #[allow(dead_code)]
    message_router: Arc<MessageRouter>,
    /// WebSocket 通知服务（用于在订阅用户房间后下发离线通知）
    #[cfg(feature = "ws_offline_notifications")]
    notification_service: Arc<WebSocketNotificationService>,
    /// 指标
    #[allow(dead_code)]
    metrics: Option<Arc<AppMetrics>>,
    /// 权限服务
    permission_service: PermissionService,
    /// 审计服务（可选，默认None，通过 feature gate 控制）
    #[cfg(feature = "audit")]
    audit: Option<AuditService<NoopAuditSink>>,
    /// WebSocket 事件钩子（默认 Noop）
    hooks: WsHookService<NoopWsHooks>,
}

impl WebSocketManager {
    #[cfg(feature = "ws_offline_notifications")]
    fn clone_for_tasks(&self) -> WebSocketManagerForTasks {
        WebSocketManagerForTasks {
            notification_service: self.notification_service.clone(),
        }
    }

    // 仅用于审计时启用的内部事件记录等，其他辅助方法通过 feature gate 控制
    /// 创建新的WebSocket管理器
    pub fn new(
        message_router: Arc<MessageRouter>,
        connection_manager: Arc<ConnectionManager>,
        room_manager: Arc<RoomManager>,
        db_pool: PgPool,
        notification_service: Arc<WebSocketNotificationService>,
        metrics: Option<Arc<AppMetrics>>,
        #[cfg(feature = "audit")]
        audit: Option<AuditService<NoopAuditSink>>,
    ) -> Self {
        let permission_service = PermissionService::new(db_pool.clone());
        #[cfg(not(feature = "ws_offline_notifications"))]
        let _ = &notification_service;
        Self {
            connection_manager,
            room_manager,
            message_router,
            #[cfg(feature = "ws_offline_notifications")]
            notification_service,
            metrics,
            permission_service,
            #[cfg(feature = "audit")]
            audit,
            hooks: WsHookService::new(Arc::new(NoopWsHooks::default())),
        }
    }

    /// 指标：记录收到一条客户端消息
    #[allow(dead_code)]
    pub fn record_ws_message_received(&self) {
        if let Some(m) = &self.metrics {
            m.record_ws_message_received();
        }
    }

    /// 添加新连接
    /// 
    /// # 参数
    /// * `connection_id` - 连接ID
    /// * `user_id` - 用户ID
    /// * `sender` - 消息发送通道
    pub async fn add_connection(
        &self,
        connection_id: Uuid,
        user_id: i64,
        sender: mpsc::UnboundedSender<ServerMessage>,
    ) -> WsResult<()> {
        debug!("添加WebSocket连接: connection_id={}, user_id={}", connection_id, user_id);

        // 添加到连接管理器
        self.connection_manager
            .add_connection(connection_id, user_id, sender)
            .await?;

        info!("WebSocket连接已添加: connection_id={}, user_id={}", connection_id, user_id);
        // 审计：连接建立
        #[cfg(feature = "audit")]
        if let Some(audit) = &self.audit {
            let ev = AuditEvent::now("ws_connection", "connect")
                .with_actor(user_id)
                .with_target(connection_id.to_string());
            audit.submit(ev).await;
        }
        // 更新连接数量指标
        if let Some(m) = &self.metrics {
            let stats = self.connection_manager.get_stats().await;
            m.update_ws_metrics(stats.total_connections);
        }
        Ok(())
    }

    /// 移除连接
    /// 
    /// # 参数
    /// * `connection_id` - 连接ID
    pub async fn remove_connection(&self, connection_id: Uuid) -> WsResult<()> {
        debug!("移除WebSocket连接: connection_id={}", connection_id);

        // 获取连接信息
        let connection_info = self.connection_manager
            .get_connection(connection_id)
            .await;

        if let Some(_info) = connection_info {
            // 从所有房间中移除
            let user_rooms = self.room_manager.get_connection_rooms(connection_id).await;
            for room_id in user_rooms {
                self.room_manager
                    .unsubscribe_from_room(connection_id, &room_id)
                    .await?;
            }

            // 清理事件总线中的订阅（使用实际存在的方法） // Removed EventBus call
            // self.event_bus
            //     .unsubscribe_all(connection_id, Some(info.user_id.to_string()))
            //     .await?;
        }

        // 从连接管理器中移除
        self.connection_manager.remove_connection(connection_id).await;

        info!("WebSocket连接已移除: connection_id={}", connection_id);
        // 钩子
        self.hooks.inner().on_disconnect(connection_id);
        // 审计：连接断开
        #[cfg(feature = "audit")]
        if let Some(audit) = &self.audit {
            let ev = AuditEvent::now("ws_connection", "disconnect")
                .with_target(connection_id.to_string());
            audit.submit(ev).await;
        }
        // 更新连接数量指标
        if let Some(m) = &self.metrics {
            let stats = self.connection_manager.get_stats().await;
            m.update_ws_metrics(stats.total_connections);
        }
        Ok(())
    }

    /// 订阅房间
    /// 
    /// # 参数
    /// * `connection_id` - 连接ID
    /// * `room_id` - 房间ID
    pub async fn subscribe_to_room(&self, connection_id: Uuid, room_id: RoomId) -> WsResult<()> {
        debug!("连接 {} 订阅房间 {:?}", connection_id, room_id);

        // 检查连接是否存在
        let _connection_info = self.connection_manager
            .get_connection(connection_id)
            .await
            .ok_or_else(|| WsError::internal_error(&format!("连接不存在: {}", connection_id)))?;

        // 添加到房间管理器
        self.room_manager
            .subscribe_to_room(connection_id, room_id.clone())
            .await?;

        // 添加到事件总线（使用实际存在的方法） // Removed EventBus call
        // self.event_bus
        //     .subscribe_room(connection_id, Some(connection_info.user_id.to_string()), room_id.clone())
        //     .await?;

        info!("连接 {} 已订阅房间 {:?}", connection_id, room_id);
        // 钩子
        self.hooks.inner().on_subscribe(connection_id, &room_id);
        Ok(())
    }

    /// 取消订阅房间
    /// 
    /// # 参数
    /// * `connection_id` - 连接ID
    /// * `room_id` - 房间ID
    pub async fn unsubscribe_from_room(&self, connection_id: Uuid, room_id: &RoomId) -> WsResult<()> {
        debug!("连接 {} 取消订阅房间 {:?}", connection_id, room_id);

        // 检查连接是否存在
        let _connection_info = self.connection_manager
            .get_connection(connection_id)
            .await
            .ok_or_else(|| WsError::internal_error(&format!("连接不存在: {}", connection_id)))?;

        // 从房间管理器中移除
        self.room_manager
            .unsubscribe_from_room(connection_id, room_id)
            .await?;

        // 从事件总线中移除（使用实际存在的方法） // Removed EventBus call
        // self.event_bus
        //     .unsubscribe_room(connection_id, Some(connection_info.user_id.to_string()), room_id)
        //     .await?;

        info!("连接 {} 已取消订阅房间 {:?}", connection_id, room_id);
        // 钩子
        self.hooks.inner().on_unsubscribe(connection_id, room_id);
        Ok(())
    }

    /// 向特定连接发送消息 (内部使用)
    /// 
    /// 注意：业务逻辑应该使用 WebSocketNotificationService
    pub(crate) async fn send_to_connection(&self, connection_id: Uuid, message: ServerMessage) -> WsResult<()> {
        let res = self.connection_manager
            .send_to_connection(connection_id, message)
            .await;
        if res.is_ok() {
            if let Some(m) = &self.metrics {
                m.record_ws_message_sent();
            }
        }
        res
    }

    /// 获取消息路由器引用 (供 WebSocketNotificationService 使用)
    #[allow(dead_code)]
    pub fn message_router(&self) -> Arc<MessageRouter> {
        self.message_router.clone()
    }

    /// 获取连接管理器引用 (供检查连接状态使用)
    #[allow(dead_code)]
    pub fn connection_manager(&self) -> Arc<ConnectionManager> {
        self.connection_manager.clone()
    }

    /// 检查用户是否在线
    /// 
    /// # 参数
    /// * `user_id` - 用户ID
    /// 
    /// # 返回值
    /// 返回用户是否有活跃连接
    #[allow(dead_code)]
    pub async fn is_user_online(&self, user_id: i64) -> bool {
        let connections = self.connection_manager.get_user_connections(user_id).await;
        !connections.is_empty()
    }

    /// 获取用户的连接数量
    /// 
    /// # 参数
    /// * `user_id` - 用户ID
    /// 
    /// # 返回值
    /// 返回用户的活跃连接数量
    #[allow(dead_code)]
    pub async fn get_user_connection_count(&self, user_id: i64) -> usize {
        let connections = self.connection_manager.get_user_connections(user_id).await;
        connections.len()
    }

    /// 提交一个审计事件（未启用 audit 特性时为空操作）
    pub async fn audit_ws(
        &self,
        _category: &'static str,
        _action: &'static str,
        _actor_user_id: Option<i64>,
        _target: Option<String>,
        _metadata: Option<serde_json::Value>,
    ) {
        #[cfg(feature = "audit")]
        if let Some(audit) = &self.audit {
            let mut ev = AuditEvent::now(_category, _action);
            if let Some(uid) = _actor_user_id { ev = ev.with_actor(uid); }
            if let Some(tgt) = _target { ev = ev.with_target(tgt); }
            if let Some(meta) = _metadata { ev = ev.with_metadata(meta); }
            audit.submit(ev).await;
        }
    }

    /// 处理客户端连接管理相关消息
    /// 
    /// 专注处理：ping/pong、房间订阅/取消订阅
    /// 业务逻辑相关的消息应该通过其他方式处理
    /// 
    /// # 参数
    /// * `connection_id` - 连接ID
    /// * `message` - 客户端消息
    pub async fn handle_client_message(
        &self,
        connection_id: Uuid,
        message: ClientMessage,
    ) -> WsResult<()> {
        // 验证连接是否存在
        let _conn_info = self.connection_manager.get_connection(connection_id).await
            .ok_or_else(|| {
                error!("处理客户端消息失败: 连接 {} 不存在", connection_id);
                WsError::internal_error("连接信息丢失")
            })?;

        match message {
            // 基础连接管理
            ClientMessage::Ping => {
                debug!("收到Ping，回复Pong: connection_id={}", connection_id);
                let pong_msg = ServerMessage::Pong;
                self.send_to_connection(connection_id, pong_msg).await?;
            }
            ClientMessage::Connect { token: _ } => {
                // 连接消息在握手阶段处理，这里不应该收到
                warn!("收到意外的连接消息: connection_id={}", connection_id);
            }
            
            // 房间管理
            ClientMessage::JoinGroup { group_id } => {
                debug!("用户加入群组房间: group_id={}, connection_id={}", group_id, connection_id);
                // 权限校验：必须是群组成员
                let conn_info = self.connection_manager.get_connection(connection_id).await
                    .ok_or_else(|| WsError::internal_error(&format!("连接不存在: {}", connection_id)))?;
                // 直接使用 DB 权限校验
                if let Err(e) = self.permission_service.require_group_member(&group_id, conn_info.user_id).await {
                    warn!("群组订阅拒绝: user_id={}, group_id={}, error={:?}", conn_info.user_id, group_id, e);
                    if let Some(m) = &self.metrics { m.record_ws_subscription("group", "forbidden"); }
                    #[cfg(feature = "audit")]
                    if let Some(audit) = &self.audit {
                        let ev = AuditEvent::now("ws_permission", "deny")
                            .with_actor(conn_info.user_id)
                            .with_target(format!("group:{}", group_id))
                            .with_metadata(serde_json::json!({"reason": "not_member"}));
                        audit.submit(ev).await;
                    }
                    return Err(WsError::forbidden("需要群组成员权限"));
                }
                let room_id = RoomId::Group(group_id);
                self.subscribe_to_room(connection_id, room_id).await?;
                if let Some(m) = &self.metrics { m.record_ws_subscription("group", "ok"); }
                #[cfg(feature = "audit")]
                if let Some(audit) = &self.audit {
                    let ev = AuditEvent::now("ws_subscription", "subscribe")
                        .with_actor(conn_info.user_id)
                        .with_target(format!("group:{}", group_id));
                    audit.submit(ev).await;
                }
            }
            ClientMessage::LeaveGroup { group_id } => {
                debug!("用户离开群组房间: group_id={}, connection_id={}", group_id, connection_id);
                let room_id = RoomId::Group(group_id);
                self.unsubscribe_from_room(connection_id, &room_id).await?;
                if let Some(_conn_info) = self.connection_manager.get_connection(connection_id).await {
                    #[cfg(feature = "audit")]
                    if let Some(audit) = &self.audit {
                        let ev = AuditEvent::now("ws_subscription", "unsubscribe")
                            .with_actor(conn_info.user_id)
                            .with_target(format!("group:{}", group_id));
                        audit.submit(ev).await;
                    }
                }
            }
            ClientMessage::JoinConversation { conversation_id } => {
                debug!("用户加入对话房间: conversation_id={}, connection_id={}", conversation_id, connection_id);
                // 权限校验：必须是会话参与者
                let conn_info = self.connection_manager.get_connection(connection_id).await
                    .ok_or_else(|| WsError::internal_error(&format!("连接不存在: {}", connection_id)))?;
                if let Err(e) = self.permission_service.require_conversation_member(&conversation_id, conn_info.user_id).await {
                    warn!("会话订阅拒绝: user_id={}, conversation_id={}, error={:?}", conn_info.user_id, conversation_id, e);
                    if let Some(m) = &self.metrics { m.record_ws_subscription("conversation", "forbidden"); }
                    #[cfg(feature = "audit")]
                    if let Some(audit) = &self.audit {
                        let ev = AuditEvent::now("ws_permission", "deny")
                            .with_actor(conn_info.user_id)
                            .with_target(format!("conversation:{}", conversation_id))
                            .with_metadata(serde_json::json!({"reason": "not_participant"}));
                        audit.submit(ev).await;
                    }
                    return Err(WsError::forbidden("需要会话参与者权限"));
                }
                let room_id = RoomId::Conversation(conversation_id);
                self.subscribe_to_room(connection_id, room_id).await?;
                if let Some(m) = &self.metrics { m.record_ws_subscription("conversation", "ok"); }
                #[cfg(feature = "audit")]
                if let Some(audit) = &self.audit {
                    let ev = AuditEvent::now("ws_subscription", "subscribe")
                        .with_actor(conn_info.user_id)
                        .with_target(format!("conversation:{}", conversation_id));
                    audit.submit(ev).await;
                }
            }
            ClientMessage::LeaveConversation { conversation_id } => {
                debug!("用户离开对话房间: conversation_id={}, connection_id={}", conversation_id, connection_id);
                let room_id = RoomId::Conversation(conversation_id);
                self.unsubscribe_from_room(connection_id, &room_id).await?;
                if let Some(_conn_info) = self.connection_manager.get_connection(connection_id).await {
                    #[cfg(feature = "audit")]
                    if let Some(audit) = &self.audit {
                        let ev = AuditEvent::now("ws_subscription", "unsubscribe")
                            .with_actor(conn_info.user_id)
                            .with_target(format!("conversation:{}", conversation_id));
                        audit.submit(ev).await;
                    }
                }
            }
            ClientMessage::JoinUserRoom => {
                debug!("用户加入私有房间: connection_id={}", connection_id);
                let conn_info = self.connection_manager.get_connection(connection_id).await
                    .ok_or_else(|| WsError::internal_error(&format!("连接不存在: {}", connection_id)))?;
                let room_id = RoomId::User(conn_info.user_id);
                self.subscribe_to_room(connection_id, room_id).await?;
                if let Some(m) = &self.metrics { m.record_ws_subscription("user", "ok"); }
                 #[cfg(feature = "audit")]
                 if let Some(audit) = &self.audit {
                    let ev = AuditEvent::now("ws_subscription", "subscribe")
                        .with_actor(conn_info.user_id)
                        .with_target(format!("user:{}", conn_info.user_id));
                    audit.submit(ev).await;
                 }
                // 订阅后尝试下发并清理离线通知
                #[cfg(feature = "ws_offline_notifications")]
                let this = self.clone_for_tasks();
                #[cfg(feature = "ws_offline_notifications")]
                let uid = conn_info.user_id;
                tokio::spawn(async move {
                    #[cfg(feature = "ws_offline_notifications")]
                    this.deliver_offline_notifications(uid).await;
                });
            }
            ClientMessage::LeaveUserRoom => {
                debug!("用户离开私有房间: connection_id={}", connection_id);
                if let Some(conn_info) = self.connection_manager.get_connection(connection_id).await {
                    let room_id = RoomId::User(conn_info.user_id);
                    self.unsubscribe_from_room(connection_id, &room_id).await?;
                    #[cfg(feature = "audit")]
                    if let Some(audit) = &self.audit {
                        let ev = AuditEvent::now("ws_subscription", "unsubscribe")
                            .with_actor(conn_info.user_id)
                            .with_target(format!("user:{}", conn_info.user_id));
                        audit.submit(ev).await;
                    }
                }
            }
        }

        Ok(())
    }

    /// 记录内部事件 (仅用于审计和调试)
    /// 
    /// # 参数
    /// * `event` - 内部事件
    /// 
    /// # 注意
    /// 此方法仅用于日志记录，实际的WebSocket通知由WebSocketNotificationService处理
    /// 考虑在未来版本中移除此方法，直接在相关位置记录日志
    #[allow(dead_code)]
    pub async fn log_internal_event(&self, event: InternalEvent) -> WsResult<()> {
        match event {
            InternalEvent::UserConnected { user_id, session_id } => {
                info!("📱 用户连接: user_id={}, session_id={}", user_id, session_id);
            }
            InternalEvent::UserDisconnected { user_id, session_id } => {
                info!("📱 用户断开: user_id={}, session_id={}", user_id, session_id);
            }
            InternalEvent::MessageSent {
                message_id,
                room_id,
                room_type,
                sender_id,
                content: _,
                message_type: _,
            } => {
                debug!(
                    "💬 消息发送: room={:?}({:?}), msg_id={}, sender={}",
                    room_id, room_type, message_id, sender_id
                );
            }
            InternalEvent::GroupEvent { group_id, event_type, actor_id, target_id } => {
                debug!(
                    "👥 群组事件: group={}, type={:?}, actor={}, target={:?}",
                    group_id, event_type, actor_id, target_id
                );
            }
            InternalEvent::UserLocationUpdate { user_id, latitude, longitude, location_name } => {
                debug!(
                    "📍 位置更新: user={}, pos=({:.4},{:.4}), location={:?}",
                    user_id, latitude, longitude, location_name
                );
            }
        }
        Ok(())
    }



    /// 获取统计信息
    #[allow(dead_code)]
    pub async fn get_stats(&self) -> WebSocketStats {
        let connection_stats = self.connection_manager.get_stats().await;
        let room_stats = self.room_manager.get_stats().await;

        WebSocketStats {
            total_connections: connection_stats.total_connections,
            active_connections: connection_stats.total_connections, // 假设所有连接都是活跃的
            total_users: connection_stats.active_users,
            total_rooms: room_stats.active_rooms,
            // event_bus_rooms: event_stats.total_rooms, // Removed EventBus stats
            // event_bus_users: event_stats.total_users, // Removed EventBus stats
        }
    }

    /// 清理非活跃连接
    /// 
    /// # 参数
    /// * `inactive_timeout_ms` - 非活跃超时时间（毫秒）
    /// 
    /// # 返回值
    /// 返回清理的连接数
    #[allow(dead_code)]
    pub async fn cleanup_inactive_connections(&self, inactive_timeout_ms: u64) -> usize {
        let timeout_secs = inactive_timeout_ms / 1000; // 转换为秒
        // 先由 ConnectionManager 计算出需要移除的连接，再通过 WebSocketManager 完整移除（包含房间订阅）
        let to_remove = self.connection_manager.find_inactive_connections(timeout_secs).await;
        let mut removed = 0usize;
        for connection_id in to_remove {
            if self.remove_connection(connection_id).await.is_ok() {
                removed += 1;
            }
        }
        if removed > 0 {
            info!("清理了 {} 个非活跃连接", removed);
            if let Some(m) = &self.metrics { m.record_ws_cleanup_removed(removed); }
        }
        removed
    }

    /// 清理空房间（无订阅者）
    #[allow(dead_code)]
    pub async fn cleanup_empty_rooms(&self) -> usize {
        self.room_manager.cleanup_empty_rooms().await
    }

    /// 检查是否达到连接限制
    #[allow(dead_code)]
    pub async fn is_connection_limit_reached(&self) -> bool {
        let stats = self.connection_manager.get_stats().await;
        stats.total_connections >= MAX_CONNECTIONS
    }
}

impl Default for WebSocketManager {
    fn default() -> Self {
        panic!("WebSocketManager::default() is not supported due to service dependencies. Please construct explicitly.");
    }
}
