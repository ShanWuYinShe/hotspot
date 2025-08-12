//! 审计接口预留
//!
//! 面向未来的安全/合规审计与可观测性需求，提供统一的审计事件接口与数据结构。
//! 默认实现为空操作，避免对现有业务产生影响；
//! 生产环境可替换为落库/消息队列/日志聚合等实现。

use std::sync::Arc;

#[derive(Debug, Clone)]
pub struct AuditEvent {
    pub category: &'static str,
    pub action: &'static str,
    pub actor_user_id: Option<i64>,
    pub target: Option<String>,
    pub metadata: Option<serde_json::Value>,
    pub occurred_at_unix_ms: i64,
}

impl AuditEvent {
    pub fn now(category: &'static str, action: &'static str) -> Self {
        Self {
            category,
            action,
            actor_user_id: None,
            target: None,
            metadata: None,
            occurred_at_unix_ms: chrono::Utc::now().timestamp_millis(),
        }
    }

    pub fn with_actor(mut self, user_id: i64) -> Self { self.actor_user_id = Some(user_id); self }
    pub fn with_target<T: ToString>(mut self, target: T) -> Self { self.target = Some(target.to_string()); self }
    pub fn with_metadata(mut self, metadata: serde_json::Value) -> Self { self.metadata = Some(metadata); self }
}

#[allow(dead_code)]
pub trait AuditSink: Send + Sync {
    async fn submit(&self, event: AuditEvent);
}

/// 空实现（默认）
#[derive(Debug, Clone)]
pub struct NoopAuditSink;

impl NoopAuditSink {
    pub fn new() -> Self { Self }
}

impl AuditSink for NoopAuditSink {
    async fn submit(&self, _event: AuditEvent) {}
}

/// 审计服务：封装 sink，便于替换
#[derive(Clone)]
pub struct AuditService<S: AuditSink + ?Sized> {
    sink: Arc<S>,
}

impl<S: AuditSink + ?Sized> AuditService<S> {
    pub fn new(sink: Arc<S>) -> Self { Self { sink } }
    pub async fn submit(&self, event: AuditEvent) { self.sink.submit(event).await }
}


