/// Actor系统 - 基于Actor模式的并发安全设计
/// 
/// Actor特征：
/// - 消息驱动
/// - 状态隔离
/// - 并发安全
/// - 容错处理
/// - 监督机制

use async_trait::async_trait;
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::{mpsc, RwLock, oneshot};
use uuid::Uuid;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use crate::domain::{DomainResult, DomainError};

// === Actor系统核心定义 ===

/// Actor ID - 全局唯一标识符
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct ActorId(Uuid);

impl ActorId {
    /// 生成新的Actor ID
    pub fn new() -> Self {
        Self(Uuid::new_v4())
    }
    
    /// 从UUID创建Actor ID
    pub fn from_uuid(uuid: Uuid) -> Self {
        Self(uuid)
    }
    
    /// 获取UUID值
    pub fn value(&self) -> Uuid {
        self.0
    }
    
    /// 转换为字符串
    pub fn to_string(&self) -> String {
        self.0.to_string()
    }
}

impl Default for ActorId {
    fn default() -> Self {
        Self::new()
    }
}

impl std::fmt::Display for ActorId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}

/// Actor消息特征 - 所有Actor消息必须实现此特征
pub trait ActorMessage: Send + Sync + std::fmt::Debug + std::any::Any {
    /// 获取消息类型名称
    fn message_type(&self) -> &'static str;
    
    /// 获取消息优先级（0-255，数字越小优先级越高）
    fn priority(&self) -> u8 {
        128 // 默认中等优先级
    }
    
    /// 是否需要响应
    fn requires_response(&self) -> bool {
        false
    }
    
    /// 转换为Any trait以支持downcast
    fn as_any(&self) -> &dyn std::any::Any;
}

/// Actor消息包装器
#[derive(Debug)]
pub struct MessageEnvelope {
    /// 消息ID
    pub message_id: Uuid,
    /// 发送者Actor ID
    pub sender_id: Option<ActorId>,
    /// 接收者Actor ID
    pub recipient_id: ActorId,
    /// 消息内容
    pub message: Box<dyn ActorMessage>,
    /// 发送时间
    pub sent_at: DateTime<Utc>,
    /// 响应通道（如果需要响应）
    pub response_sender: Option<oneshot::Sender<ActorResponse>>,
}

/// Actor响应
#[derive(Debug, Clone)]
pub enum ActorResponse {
    /// 成功响应
    Success(Option<serde_json::Value>),
    /// 错误响应
    Error(String),
    /// 超时响应
    Timeout,
}

/// Actor特征 - 所有Actor必须实现此特征
#[async_trait]
pub trait Actor: Send + Sync {
    /// 获取Actor ID
    fn actor_id(&self) -> ActorId;
    
    /// 获取Actor类型名称
    fn actor_type(&self) -> &'static str;
    
    /// 处理消息
    async fn handle_message(&mut self, envelope: MessageEnvelope) -> DomainResult<Option<ActorResponse>>;
    
    /// Actor启动时调用
    async fn on_start(&mut self) -> DomainResult<()> {
        Ok(())
    }
    
    /// Actor停止时调用
    async fn on_stop(&mut self) -> DomainResult<()> {
        Ok(())
    }
    
    /// Actor出错时调用
    async fn on_error(&mut self, error: &DomainError) -> DomainResult<()> {
        tracing::error!(
            actor_id = %self.actor_id(),
            actor_type = self.actor_type(),
            error = %error,
            "Actor error occurred"
        );
        Ok(())
    }
    
    /// 获取Actor状态（用于监控）
    async fn get_status(&self) -> ActorStatus {
        ActorStatus::Running
    }
}

/// Actor状态
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ActorStatus {
    /// 初始化中
    Initializing,
    /// 运行中
    Running,
    /// 暂停
    Paused,
    /// 停止中
    Stopping,
    /// 已停止
    Stopped,
    /// 错误状态
    Error(String),
}

/// Actor引用 - 用于向Actor发送消息
#[derive(Clone)]
pub struct ActorRef {
    actor_id: ActorId,
    message_sender: mpsc::UnboundedSender<MessageEnvelope>,
}

impl ActorRef {
    /// 创建新的Actor引用
    pub fn new(actor_id: ActorId, message_sender: mpsc::UnboundedSender<MessageEnvelope>) -> Self {
        Self {
            actor_id,
            message_sender,
        }
    }
    
    /// 获取Actor ID
    pub fn actor_id(&self) -> ActorId {
        self.actor_id.clone()
    }
    
    /// 发送消息（不等待响应）
    pub async fn send(&self, message: Box<dyn ActorMessage>) -> DomainResult<()> {
        let envelope = MessageEnvelope {
            message_id: Uuid::new_v4(),
            sender_id: None,
            recipient_id: self.actor_id.clone(),
            message,
            sent_at: Utc::now(),
            response_sender: None,
        };
        
        self.message_sender.send(envelope).map_err(|e| {
            DomainError::BusinessRuleViolation {
                rule: format!("Failed to send message to actor {}: {}", self.actor_id, e),
            }
        })
    }
    
    /// 发送消息并等待响应
    pub async fn ask(&self, message: Box<dyn ActorMessage>, timeout_ms: u64) -> DomainResult<ActorResponse> {
        let (response_sender, response_receiver) = oneshot::channel();
        
        let envelope = MessageEnvelope {
            message_id: Uuid::new_v4(),
            sender_id: None,
            recipient_id: self.actor_id.clone(),
            message,
            sent_at: Utc::now(),
            response_sender: Some(response_sender),
        };
        
        self.message_sender.send(envelope).map_err(|e| {
            DomainError::BusinessRuleViolation {
                rule: format!("Failed to send message to actor {}: {}", self.actor_id, e),
            }
        })?;
        
        // 等待响应或超时
        let timeout_duration = std::time::Duration::from_millis(timeout_ms);
        match tokio::time::timeout(timeout_duration, response_receiver).await {
            Ok(Ok(response)) => Ok(response),
            Ok(Err(_)) => Ok(ActorResponse::Error("Response channel closed".to_string())),
            Err(_) => Ok(ActorResponse::Timeout),
        }
    }
}

/// Actor上下文 - 提供Actor运行时环境
#[allow(dead_code)]
pub struct ActorContext {
    /// Actor系统引用
    system: Arc<ActorSystem>,
    /// 当前Actor ID
    actor_id: ActorId,
    /// 消息发送器
    #[allow(dead_code)]
    message_sender: mpsc::UnboundedSender<MessageEnvelope>,
}

impl ActorContext {
    /// 创建新的Actor上下文
    pub fn new(
        system: Arc<ActorSystem>,
        actor_id: ActorId,
        message_sender: mpsc::UnboundedSender<MessageEnvelope>,
    ) -> Self {
        Self {
            system,
            actor_id,
            message_sender,
        }
    }
    
    /// 获取当前Actor ID
    pub fn actor_id(&self) -> ActorId {
        self.actor_id.clone()
    }
    
    /// 创建子Actor
    pub async fn spawn_child<A>(&self, actor: A) -> DomainResult<ActorRef>
    where
        A: Actor + 'static,
    {
        self.system.spawn_actor(actor).await
    }
    
    /// 获取Actor引用
    pub async fn get_actor_ref(&self, actor_id: ActorId) -> Option<ActorRef> {
        self.system.get_actor_ref(actor_id).await
    }
    
    /// 向其他Actor发送消息
    pub async fn send_to(&self, actor_id: ActorId, message: Box<dyn ActorMessage>) -> DomainResult<()> {
        if let Some(actor_ref) = self.get_actor_ref(actor_id.clone()).await {
            actor_ref.send(message).await
        } else {
            Err(DomainError::BusinessRuleViolation {
                rule: format!("Actor {} not found", actor_id),
            })
        }
    }
    
    /// 停止Actor
    pub async fn stop_actor(&self, actor_id: ActorId) -> DomainResult<()> {
        self.system.stop_actor(actor_id).await
    }
}

/// Actor系统 - 管理所有Actor的生命周期
pub struct ActorSystem {
    /// 系统名称
    name: String,
    /// Actor注册表
    actors: Arc<RwLock<HashMap<ActorId, ActorRef>>>,
    /// Actor状态
    actor_status: Arc<RwLock<HashMap<ActorId, ActorStatus>>>,
    /// 系统统计
    stats: Arc<RwLock<ActorSystemStats>>,
}

/// Actor系统统计
#[derive(Debug, Default, Clone)]
pub struct ActorSystemStats {
    /// 总Actor数量
    pub total_actors: u64,
    /// 运行中的Actor数量
    pub running_actors: u64,
    /// 已处理的消息数量
    pub messages_processed: u64,
    /// 失败的消息数量
    pub messages_failed: u64,
    /// 平均消息处理时间（毫秒）
    pub avg_processing_time_ms: f64,
}

impl ActorSystem {
    /// 创建新的Actor系统
    pub fn new(name: &str) -> Arc<Self> {
        Arc::new(Self {
            name: name.to_string(),
            actors: Arc::new(RwLock::new(HashMap::new())),
            actor_status: Arc::new(RwLock::new(HashMap::new())),
            stats: Arc::new(RwLock::new(ActorSystemStats::default())),
        })
    }
    
    /// 启动Actor
    pub async fn spawn_actor<A>(self: &Arc<Self>, mut actor: A) -> DomainResult<ActorRef>
    where
        A: Actor + 'static,
    {
        let actor_id = actor.actor_id();
        let (message_sender, mut message_receiver) = mpsc::unbounded_channel();
        let actor_ref = ActorRef::new(actor_id.clone(), message_sender.clone());
        
        // 注册Actor
        {
            let mut actors = self.actors.write().await;
            actors.insert(actor_id.clone(), actor_ref.clone());
        }
        
        // 设置初始状态
        {
            let mut status = self.actor_status.write().await;
            status.insert(actor_id.clone(), ActorStatus::Initializing);
        }
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.total_actors += 1;
        }
        
        // 创建Actor上下文
        let _context = ActorContext::new(self.clone(), actor_id.clone(), message_sender);
        
        // 启动Actor消息处理循环
        let system_clone = self.clone();
        let actor_id_clone = actor_id.clone();
        
        tokio::spawn(async move {
            // 调用Actor启动方法
            if let Err(e) = actor.on_start().await {
                tracing::error!(
                    actor_id = %actor_id_clone,
                    error = %e,
                    "Actor failed to start"
                );
                
                // 设置错误状态
                let mut status = system_clone.actor_status.write().await;
                status.insert(actor_id_clone.clone(), ActorStatus::Error(e.to_string()));
                return;
            }
            
            // 设置运行状态
            {
                let mut status = system_clone.actor_status.write().await;
                status.insert(actor_id_clone.clone(), ActorStatus::Running);
                
                let mut stats = system_clone.stats.write().await;
                stats.running_actors += 1;
            }
            
            tracing::info!(
                actor_id = %actor_id_clone,
                actor_type = actor.actor_type(),
                "Actor started successfully"
            );
            
            // 消息处理循环
            while let Some(envelope) = message_receiver.recv().await {
                let start_time = std::time::Instant::now();
                let message_type = envelope.message.message_type();
                
                tracing::debug!(
                    actor_id = %actor_id_clone,
                    message_id = %envelope.message_id,
                    message_type = message_type,
                    "Processing message"
                );
                
                // 处理消息
                let result = actor.handle_message(envelope).await;
                let processing_time = start_time.elapsed();
                
                // 更新统计
                {
                    let mut stats = system_clone.stats.write().await;
                    stats.messages_processed += 1;
                    
                    // 更新平均处理时间
                    let new_time_ms = processing_time.as_millis() as f64;
                    if stats.avg_processing_time_ms == 0.0 {
                        stats.avg_processing_time_ms = new_time_ms;
                    } else {
                        stats.avg_processing_time_ms = 
                            (stats.avg_processing_time_ms * 0.9) + (new_time_ms * 0.1);
                    }
                    
                    if result.is_err() {
                        stats.messages_failed += 1;
                    }
                }
                
                // 处理错误
                if let Err(e) = &result {
                    tracing::error!(
                        actor_id = %actor_id_clone,
                        message_type = message_type,
                        error = %e,
                        processing_time_ms = processing_time.as_millis(),
                        "Message processing failed"
                    );
                    
                    // 调用错误处理方法
                    if let Err(error_handling_error) = actor.on_error(e).await {
                        tracing::error!(
                            actor_id = %actor_id_clone,
                            error = %error_handling_error,
                            "Actor error handling failed"
                        );
                    }
                } else {
                    tracing::debug!(
                        actor_id = %actor_id_clone,
                        message_type = message_type,
                        processing_time_ms = processing_time.as_millis(),
                        "Message processed successfully"
                    );
                }
            }
            
            // Actor停止
            tracing::info!(
                actor_id = %actor_id_clone,
                "Actor message loop ended, stopping actor"
            );
            
            // 调用停止方法
            if let Err(e) = actor.on_stop().await {
                tracing::error!(
                    actor_id = %actor_id_clone,
                    error = %e,
                    "Actor failed to stop gracefully"
                );
            }
            
            // 更新状态和统计
            {
                let mut status = system_clone.actor_status.write().await;
                status.insert(actor_id_clone.clone(), ActorStatus::Stopped);
                
                let mut stats = system_clone.stats.write().await;
                if stats.running_actors > 0 {
                    stats.running_actors -= 1;
                }
            }
            
            // 从注册表中移除
            {
                let mut actors = system_clone.actors.write().await;
                actors.remove(&actor_id_clone);
            }
            
            tracing::info!(
                actor_id = %actor_id_clone,
                "Actor stopped successfully"
            );
        });
        
        Ok(actor_ref)
    }
    
    /// 获取Actor引用
    pub async fn get_actor_ref(&self, actor_id: ActorId) -> Option<ActorRef> {
        let actors = self.actors.read().await;
        actors.get(&actor_id).cloned()
    }
    
    /// 停止Actor
    pub async fn stop_actor(&self, actor_id: ActorId) -> DomainResult<()> {
        if let Some(actor_ref) = self.get_actor_ref(actor_id.clone()).await {
            // 关闭消息通道，这将导致Actor消息循环结束
            drop(actor_ref.message_sender);
            
            // 设置停止状态
            {
                let mut status = self.actor_status.write().await;
                status.insert(actor_id.clone(), ActorStatus::Stopping);
            }
            
            tracing::info!(
                actor_id = %actor_id,
                "Actor stop initiated"
            );
            
            Ok(())
        } else {
            Err(DomainError::BusinessRuleViolation {
                rule: format!("Actor {} not found", actor_id),
            })
        }
    }
    
    /// 停止所有Actor
    pub async fn shutdown(&self) -> DomainResult<()> {
        tracing::info!(
            system_name = %self.name,
            "Shutting down actor system"
        );
        
        let actor_ids: Vec<ActorId> = {
            let actors = self.actors.read().await;
            actors.keys().cloned().collect()
        };
        
        for actor_id in actor_ids {
            if let Err(e) = self.stop_actor(actor_id.clone()).await {
                tracing::error!(
                    actor_id = %actor_id,
                    error = %e,
                    "Failed to stop actor during shutdown"
                );
            }
        }
        
        // 等待所有Actor停止
        let mut attempts = 0;
        while attempts < 30 { // 最多等待30秒
            let running_count = {
                let actors = self.actors.read().await;
                actors.len()
            };
            
            if running_count == 0 {
                break;
            }
            
            tokio::time::sleep(std::time::Duration::from_secs(1)).await;
            attempts += 1;
        }
        
        let final_count = {
            let actors = self.actors.read().await;
            actors.len()
        };
        
        if final_count > 0 {
            tracing::warn!(
                system_name = %self.name,
                remaining_actors = final_count,
                "Some actors did not stop gracefully"
            );
        } else {
            tracing::info!(
                system_name = %self.name,
                "All actors stopped successfully"
            );
        }
        
        Ok(())
    }
    
    /// 获取系统统计
    pub async fn get_stats(&self) -> ActorSystemStats {
        self.stats.read().await.clone()
    }
    
    /// 获取所有Actor状态
    pub async fn get_all_actor_status(&self) -> HashMap<ActorId, ActorStatus> {
        self.actor_status.read().await.clone()
    }
    
    /// 获取系统健康状态
    pub async fn get_health(&self) -> ActorSystemHealth {
        let stats = self.get_stats().await;
        let actor_status = self.get_all_actor_status().await;
        
        let error_actors = actor_status.values()
            .filter(|status| matches!(status, ActorStatus::Error(_)))
            .count();
        
        let is_healthy = error_actors == 0 && stats.running_actors > 0;
        
        ActorSystemHealth {
            is_healthy,
            total_actors: stats.total_actors,
            running_actors: stats.running_actors,
            error_actors: error_actors as u64,
            messages_processed: stats.messages_processed,
            messages_failed: stats.messages_failed,
            avg_processing_time_ms: stats.avg_processing_time_ms,
        }
    }
}

/// Actor系统健康状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActorSystemHealth {
    pub is_healthy: bool,
    pub total_actors: u64,
    pub running_actors: u64,
    pub error_actors: u64,
    pub messages_processed: u64,
    pub messages_failed: u64,
    pub avg_processing_time_ms: f64,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_actor_id_creation() {
        let actor_id = ActorId::new();
        assert!(!actor_id.value().is_nil());
        
        let uuid = Uuid::new_v4();
        let actor_id2 = ActorId::from_uuid(uuid);
        assert_eq!(actor_id2.value(), uuid);
    }
    
    #[test]
    fn test_actor_system_creation() {
        let system = ActorSystem::new("test_system");
        assert_eq!(system.name, "test_system");
    }
    
    #[tokio::test]
    async fn test_actor_system_stats() {
        let system = ActorSystem::new("test_system");
        let stats = system.get_stats().await;
        
        assert_eq!(stats.total_actors, 0);
        assert_eq!(stats.running_actors, 0);
        assert_eq!(stats.messages_processed, 0);
        assert_eq!(stats.messages_failed, 0);
    }
}
