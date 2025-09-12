/// WebSocket集成 - 完整的WebSocket服务集成示例

#[allow(dead_code)]
/// 
/// 集成特性：
/// - Actor系统初始化
/// - WebSocket服务配置
/// - HTTP升级处理
/// - 消息路由集成
/// - 事件系统集成

use std::sync::Arc;
use axum::{
    extract::{ws::WebSocketUpgrade, Query},
    response::Response,
    http::StatusCode,
};
use serde::{Deserialize, Serialize};
use uuid::Uuid;
use tokio::sync::RwLock;

use crate::domain::{DomainResult, DomainError, UserId, GroupId};
use crate::infrastructure::{
    WebSocketManager, WebSocketManagerConfig, WebSocketManagerHealth,
    RoomType, UserPresenceStatus,
    EventSystemIntegrator,
};

/// WebSocket服务配置
#[derive(Debug, Clone)]
pub struct WebSocketServiceConfig {
    /// WebSocket管理器配置
    pub manager_config: WebSocketManagerConfig,
    /// 是否启用事件集成
    pub enable_event_integration: bool,
    /// 是否启用认证
    pub enable_authentication: bool,
    /// 是否启用速率限制
    pub enable_rate_limiting: bool,
    /// 速率限制配置
    pub rate_limit_per_minute: u32,
}

impl Default for WebSocketServiceConfig {
    fn default() -> Self {
        Self {
            manager_config: WebSocketManagerConfig::default(),
            enable_event_integration: true,
            enable_authentication: true,
            enable_rate_limiting: true,
            rate_limit_per_minute: 60,
        }
    }
}

/// WebSocket连接查询参数
#[derive(Debug, Deserialize)]
pub struct WebSocketQuery {
    /// 用户ID
    pub user_id: Option<i64>,
    /// 认证令牌
    pub token: Option<String>,
    /// 连接类型
    pub connection_type: Option<String>,
}

/// WebSocket服务 - 完整的WebSocket服务实现
pub struct WebSocketService {
    /// WebSocket管理器
    manager: Arc<WebSocketManager>,
    /// 事件系统集成器
    event_integrator: Option<Arc<EventSystemIntegrator>>,
    /// 配置
    config: WebSocketServiceConfig,
    /// 速率限制器
    rate_limiter: Arc<RwLock<RateLimiter>>,
}

/// 简单的速率限制器
#[derive(Debug)]
pub struct RateLimiter {
    /// 用户请求计数 (user_id -> (count, last_reset))
    user_counts: std::collections::HashMap<UserId, (u32, chrono::DateTime<chrono::Utc>)>,
}

impl RateLimiter {
    /// 创建新的速率限制器
    pub fn new() -> Self {
        Self {
            user_counts: std::collections::HashMap::new(),
        }
    }
    
    /// 检查用户是否超过速率限制
    pub fn check_rate_limit(&mut self, user_id: UserId, limit_per_minute: u32) -> bool {
        let now = chrono::Utc::now();
        let entry = self.user_counts.entry(user_id).or_insert((0, now));
        
        // 如果超过一分钟，重置计数
        if now.signed_duration_since(entry.1).num_minutes() >= 1 {
            entry.0 = 0;
            entry.1 = now;
        }
        
        entry.0 += 1;
        entry.0 <= limit_per_minute
    }
    
    /// 清理过期的计数
    pub fn cleanup_expired(&mut self) {
        let now = chrono::Utc::now();
        self.user_counts.retain(|_, (_, last_reset)| {
            now.signed_duration_since(*last_reset).num_minutes() < 5
        });
    }
}

impl WebSocketService {
    /// 创建新的WebSocket服务
    pub async fn new(
        config: WebSocketServiceConfig,
        event_integrator: Option<Arc<EventSystemIntegrator>>,
    ) -> DomainResult<Arc<Self>> {
        let manager = WebSocketManager::new(config.manager_config.clone()).await?;
        
        let service = Arc::new(Self {
            manager,
            event_integrator,
            config: config.clone(),
            rate_limiter: Arc::new(RwLock::new(RateLimiter::new())),
        });
        
        // 启动清理任务
        service.start_cleanup_tasks().await;
        
        tracing::info!(
            enable_event_integration = config.enable_event_integration,
            enable_authentication = config.enable_authentication,
            enable_rate_limiting = config.enable_rate_limiting,
            "WebSocket service initialized"
        );
        
        Ok(service)
    }
    
    /// 处理WebSocket升级请求
    pub async fn handle_websocket_upgrade(
        &self,
        ws_upgrade: WebSocketUpgrade,
        query: Query<WebSocketQuery>,
    ) -> Result<Response, StatusCode> {
        let connection_id = Uuid::new_v4().to_string();
        
        // 验证用户身份
        let user_id = if self.config.enable_authentication {
            match self.authenticate_user(&query.token).await {
                Ok(uid) => uid,
                Err(_) => {
                    tracing::warn!(
                        connection_id = %connection_id,
                        "WebSocket authentication failed"
                    );
                    return Err(StatusCode::UNAUTHORIZED);
                }
            }
        } else {
            query.user_id.and_then(|id| UserId::new(id).ok())
        };
        
        // 检查速率限制
        if self.config.enable_rate_limiting {
            if let Some(uid) = user_id {
                let mut rate_limiter = self.rate_limiter.write().await;
                if !rate_limiter.check_rate_limit(uid, self.config.rate_limit_per_minute) {
                    tracing::warn!(
                        user_id = %uid.value(),
                        connection_id = %connection_id,
                        "WebSocket rate limit exceeded"
                    );
                    return Err(StatusCode::TOO_MANY_REQUESTS);
                }
            }
        }
        
        let manager = self.manager.clone();
        let event_integrator = self.event_integrator.clone();
        
        Ok(ws_upgrade.on_upgrade(move |websocket| async move {
            if let Err(e) = manager.handle_connection(connection_id.clone(), websocket, user_id).await {
                tracing::error!(
                    connection_id = %connection_id,
                    user_id = ?user_id,
                    error = %e,
                    "Failed to handle WebSocket connection"
                );
                return;
            }
            
            // 发布连接事件
            if let (Some(integrator), Some(uid)) = (event_integrator, user_id) {
                // 这里可以发布用户连接事件
                tracing::debug!(
                    user_id = %uid.value(),
                    connection_id = %connection_id,
                    "User connected via WebSocket"
                );
            }
            
            tracing::info!(
                connection_id = %connection_id,
                user_id = ?user_id,
                "WebSocket connection established successfully"
            );
        }))
    }
    
    /// 认证用户
    async fn authenticate_user(&self, token: &Option<String>) -> DomainResult<Option<UserId>> {
        // 简化的认证逻辑
        if let Some(token_str) = token {
            if token_str.starts_with("user_") {
                if let Ok(user_id) = token_str[5..].parse::<i64>() {
                    return Ok(UserId::new(user_id).ok());
                }
            }
        }
        
        Err(DomainError::BusinessRuleViolation {
            rule: "Invalid authentication token".to_string(),
        })
    }
    
    /// 创建群组房间
    pub async fn create_group_room(&self, group_id: GroupId, max_members: Option<usize>) -> DomainResult<()> {
        let room_id = format!("group_{}", group_id.value());
        let room_type = RoomType::Group { group_id: group_id.clone() };
        
        self.manager.get_or_create_room(&room_id, room_type, max_members).await?;
        
        tracing::info!(
            group_id = %group_id.value(),
            room_id = %room_id,
            max_members = ?max_members,
            "Group room created"
        );
        
        Ok(())
    }
    
    /// 创建私聊房间
    pub async fn create_private_room(&self, participants: Vec<UserId>) -> DomainResult<String> {
        if participants.len() != 2 {
            return Err(DomainError::ValidationError {
                field: "participants".to_string(),
                reason: "Private room must have exactly 2 participants".to_string(),
            });
        }
        
        // 生成房间ID（基于参与者ID的排序）
        let mut sorted_ids = participants.iter().map(|id| id.value()).collect::<Vec<_>>();
        sorted_ids.sort();
        let room_id = format!("private_{}_{}", sorted_ids[0], sorted_ids[1]);
        
        let room_type = RoomType::Private { participants };
        
        self.manager.get_or_create_room(&room_id, room_type, Some(2)).await?;
        
        tracing::info!(
            room_id = %room_id,
            participants = ?sorted_ids,
            "Private room created"
        );
        
        Ok(room_id)
    }
    
    /// 用户加入群组
    pub async fn user_join_group(&self, user_id: UserId, group_id: GroupId) -> DomainResult<()> {
        let room_id = format!("group_{}", group_id.value());
        
        // 确保房间存在
        let room_type = RoomType::Group { group_id: group_id.clone() };
        self.manager.get_or_create_room(&room_id, room_type, None).await?;
        
        // 用户加入房间
        self.manager.join_room(user_id, &room_id).await?;
        
        // 发布事件
        if let Some(integrator) = &self.event_integrator {
            integrator.publish_user_joined_group(user_id, group_id.clone(), "member".to_string()).await?;
        }
        
        tracing::info!(
            user_id = %user_id.value(),
            group_id = %group_id.value(),
            "User joined group via WebSocket"
        );
        
        Ok(())
    }
    
    /// 发送群组消息
    pub async fn send_group_message(
        &self,
        group_id: GroupId,
        message: &str,
        sender_id: UserId,
    ) -> DomainResult<()> {
        let room_id = format!("group_{}", group_id.value());
        
        self.manager.send_to_room(&room_id, message, sender_id).await?;
        
        tracing::debug!(
            group_id = %group_id.value(),
            sender_id = %sender_id.value(),
            message_length = message.len(),
            "Group message sent via WebSocket"
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
        self.manager.send_private_message(recipient_id, message, sender_id).await?;
        
        tracing::debug!(
            sender_id = %sender_id.value(),
            recipient_id = %recipient_id.value(),
            message_length = message.len(),
            "Private message sent via WebSocket"
        );
        
        Ok(())
    }
    
    /// 更新用户在线状态
    pub async fn update_user_presence(&self, user_id: UserId, status: UserPresenceStatus) -> DomainResult<()> {
        self.manager.update_user_status(user_id, status.clone()).await?;
        
        tracing::debug!(
            user_id = %user_id.value(),
            status = ?status,
            "User presence updated via WebSocket"
        );
        
        Ok(())
    }
    
    /// 断开用户连接
    pub async fn disconnect_user(&self, user_id: UserId, reason: &str) -> DomainResult<()> {
        // 这里需要找到用户的所有连接并断开
        // 简化实现，记录日志
        tracing::info!(
            user_id = %user_id.value(),
            reason = %reason,
            "User disconnection requested"
        );
        
        Ok(())
    }
    
    /// 获取服务统计信息
    pub async fn get_stats(&self) -> WebSocketServiceStats {
        let manager_stats = self.manager.get_stats().await;
        let connection_stats = self.manager.get_connection_stats().await;
        let room_stats = self.manager.get_room_stats().await;
        let presence_stats = self.manager.get_presence_stats().await;
        
        WebSocketServiceStats {
            manager_stats,
            connection_count: connection_stats.len(),
            room_count: room_stats.len(),
            presence_stats,
        }
    }
    
    /// 获取健康状态
    pub async fn get_health(&self) -> WebSocketServiceHealth {
        let manager_health = crate::infrastructure::WebSocketManagerHealthChecker::new(self.manager.clone())
            .check_health()
            .await;
        
        WebSocketServiceHealth {
            is_healthy: manager_health.is_healthy,
            manager_health,
            event_integration_enabled: self.event_integrator.is_some(),
            authentication_enabled: self.config.enable_authentication,
            rate_limiting_enabled: self.config.enable_rate_limiting,
        }
    }
    
    /// 关闭服务
    pub async fn shutdown(&self) -> DomainResult<()> {
        tracing::info!("Shutting down WebSocket service");
        
        self.manager.shutdown().await?;
        
        tracing::info!("WebSocket service shutdown completed");
        Ok(())
    }
    
    /// 启动清理任务
    async fn start_cleanup_tasks(&self) {
        let rate_limiter = self.rate_limiter.clone();
        
        // 启动速率限制器清理任务
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(std::time::Duration::from_secs(300)); // 每5分钟清理一次
            
            loop {
                interval.tick().await;
                
                {
                    let mut limiter = rate_limiter.write().await;
                    limiter.cleanup_expired();
                }
                
                tracing::debug!("Rate limiter cleanup completed");
            }
        });
    }
}

/// WebSocket服务统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebSocketServiceStats {
    pub manager_stats: crate::infrastructure::WebSocketManagerStats,
    pub connection_count: usize,
    pub room_count: usize,
    pub presence_stats: Option<crate::infrastructure::PresenceStats>,
}

/// WebSocket服务健康状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebSocketServiceHealth {
    pub is_healthy: bool,
    pub manager_health: WebSocketManagerHealth,
    pub event_integration_enabled: bool,
    pub authentication_enabled: bool,
    pub rate_limiting_enabled: bool,
}

/// WebSocket服务工厂
pub struct WebSocketServiceFactory;

impl WebSocketServiceFactory {
    /// 创建WebSocket服务
    pub async fn create_service(
        config: WebSocketServiceConfig,
        event_integrator: Option<Arc<EventSystemIntegrator>>,
    ) -> DomainResult<Arc<WebSocketService>> {
        WebSocketService::new(config, event_integrator).await
    }
    
    /// 创建默认WebSocket服务
    pub async fn create_default_service() -> DomainResult<Arc<WebSocketService>> {
        let config = WebSocketServiceConfig::default();
        WebSocketService::new(config, None).await
    }
    
    /// 创建带事件集成的WebSocket服务
    pub async fn create_service_with_events(
        event_integrator: Arc<EventSystemIntegrator>,
    ) -> DomainResult<Arc<WebSocketService>> {
        let config = WebSocketServiceConfig::default();
        WebSocketService::new(config, Some(event_integrator)).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_websocket_service_config() {
        let config = WebSocketServiceConfig::default();
        assert!(config.enable_event_integration);
        assert!(config.enable_authentication);
        assert!(config.enable_rate_limiting);
        assert_eq!(config.rate_limit_per_minute, 60);
    }
    
    #[test]
    fn test_rate_limiter() {
        let mut limiter = RateLimiter::new();
        let user_id = UserId::from_i64(123);
        
        // 测试正常情况
        assert!(limiter.check_rate_limit(user_id, 10));
        assert!(limiter.check_rate_limit(user_id, 10));
        
        // 测试超过限制
        for _ in 0..10 {
            limiter.check_rate_limit(user_id, 10);
        }
        assert!(!limiter.check_rate_limit(user_id, 10));
    }
    
    #[tokio::test]
    async fn test_websocket_service_creation() {
        let config = WebSocketServiceConfig::default();
        let service = WebSocketService::new(config, None).await.unwrap();
        
        let stats = service.get_stats().await;
        assert_eq!(stats.manager_stats.active_connections, 0);
        assert_eq!(stats.manager_stats.active_rooms, 0);
    }
}
