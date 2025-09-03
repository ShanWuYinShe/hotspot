//! 审计系统
//!
//! 提供完整的安全审计、操作日志和合规记录功能。
//! 支持多种审计输出方式：文件日志、数据库存储、消息队列等。

use std::sync::Arc;
use std::path::Path;
use tokio::fs::OpenOptions;
use tokio::io::AsyncWriteExt;
use tracing::{debug, warn, error};
use serde::{Serialize, Deserialize};
use chrono::{DateTime, Utc};

/// 审计事件类别
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AuditCategory {
    Authentication,  // 认证相关
    Authorization,   // 授权相关
    UserManagement,  // 用户管理
    GroupManagement, // 群组管理
    MessageSending,  // 消息发送
    DataAccess,      // 数据访问
    SystemAdmin,     // 系统管理
    Security,        // 安全事件
}

impl AuditCategory {
    pub fn as_str(&self) -> &'static str {
        match self {
            AuditCategory::Authentication => "authentication",
            AuditCategory::Authorization => "authorization",
            AuditCategory::UserManagement => "user_management",
            AuditCategory::GroupManagement => "group_management",
            AuditCategory::MessageSending => "message_sending",
            AuditCategory::DataAccess => "data_access",
            AuditCategory::SystemAdmin => "system_admin",
            AuditCategory::Security => "security",
        }
    }
}

/// 审计事件严重级别
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AuditSeverity {
    Info,     // 信息级别
    Warning,  // 警告级别
    Critical, // 严重级别
}

/// 审计事件
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditEvent {
    pub event_id: String,
    pub category: AuditCategory,
    pub action: String,
    pub severity: AuditSeverity,
    pub actor_user_id: Option<i64>,
    pub actor_ip: Option<String>,
    pub target_resource: Option<String>,
    pub target_id: Option<String>,
    pub metadata: Option<serde_json::Value>,
    pub occurred_at: DateTime<Utc>,
    pub session_id: Option<String>,
    pub user_agent: Option<String>,
}

impl AuditEvent {
    /// 创建新的审计事件
    pub fn new(category: AuditCategory, action: String, severity: AuditSeverity) -> Self {
        Self {
            event_id: uuid::Uuid::new_v4().to_string(),
            category,
            action,
            severity,
            actor_user_id: None,
            actor_ip: None,
            target_resource: None,
            target_id: None,
            metadata: None,
            occurred_at: Utc::now(),
            session_id: None,
            user_agent: None,
        }
    }

    /// 设置操作者用户ID
    pub fn with_actor(mut self, user_id: i64) -> Self {
        self.actor_user_id = Some(user_id);
        self
    }

    /// 设置操作者IP地址
    pub fn with_ip<T: ToString>(mut self, ip: T) -> Self {
        self.actor_ip = Some(ip.to_string());
        self
    }

    /// 设置目标资源
    pub fn with_target_resource<T: ToString>(mut self, resource: T) -> Self {
        self.target_resource = Some(resource.to_string());
        self
    }

    /// 设置目标ID
    pub fn with_target_id<T: ToString>(mut self, id: T) -> Self {
        self.target_id = Some(id.to_string());
        self
    }

    /// 设置元数据
    pub fn with_metadata(mut self, metadata: serde_json::Value) -> Self {
        self.metadata = Some(metadata);
        self
    }

    /// 设置会话ID
    pub fn with_session<T: ToString>(mut self, session_id: T) -> Self {
        self.session_id = Some(session_id.to_string());
        self
    }

    /// 设置用户代理
    pub fn with_user_agent<T: ToString>(mut self, user_agent: T) -> Self {
        self.user_agent = Some(user_agent.to_string());
        self
    }
}

/// 审计输出接口
#[async_trait::async_trait]
pub trait AuditSink: Send + Sync {
    async fn submit(&self, event: AuditEvent) -> Result<(), AuditError>;
    async fn flush(&self) -> Result<(), AuditError>;
}

/// 审计错误类型
#[derive(Debug, thiserror::Error)]
pub enum AuditError {
    #[error("IO错误: {0}")]
    IoError(#[from] std::io::Error),
    #[error("序列化错误: {0}")]
    SerializationError(#[from] serde_json::Error),
    #[error("审计系统不可用")]
    ServiceUnavailable,
}

/// 空实现（默认）
#[derive(Debug, Clone)]
pub struct NoopAuditSink;

impl NoopAuditSink {
    pub fn new() -> Self { Self }
}

impl Default for NoopAuditSink {
    fn default() -> Self {
        Self::new()
    }
}

#[async_trait::async_trait]
impl AuditSink for NoopAuditSink {
    async fn submit(&self, _event: AuditEvent) -> Result<(), AuditError> {
        Ok(())
    }
    
    async fn flush(&self) -> Result<(), AuditError> {
        Ok(())
    }
}

/// 文件审计输出
#[derive(Debug, Clone)]
pub struct FileAuditSink {
    file_path: Arc<std::path::PathBuf>,
}

impl FileAuditSink {
    pub fn new<P: AsRef<Path>>(file_path: P) -> Self {
        Self {
            file_path: Arc::new(file_path.as_ref().to_path_buf()),
        }
    }
}

#[async_trait::async_trait]
impl AuditSink for FileAuditSink {
    async fn submit(&self, event: AuditEvent) -> Result<(), AuditError> {
        let json_line = serde_json::to_string(&event)?;
        let log_line = format!("{}\n", json_line);
        
        let mut file = OpenOptions::new()
            .create(true)
            .append(true)
            .open(&*self.file_path)
            .await?;
            
        file.write_all(log_line.as_bytes()).await?;
        file.flush().await?;
        
        debug!("审计事件已写入文件: {} - {}", event.event_id, self.file_path.display());
        Ok(())
    }
    
    async fn flush(&self) -> Result<(), AuditError> {
        // 文件审计输出每次写入都会刷新，这里无需额外操作
        Ok(())
    }
}

/// 结构化日志审计输出
#[derive(Debug, Clone)]
pub struct LogAuditSink;

impl LogAuditSink {
    pub fn new() -> Self { Self }
}

#[async_trait::async_trait]
impl AuditSink for LogAuditSink {
    async fn submit(&self, event: AuditEvent) -> Result<(), AuditError> {
        match event.severity {
            AuditSeverity::Info => {
                tracing::info!(
                    event_id = %event.event_id,
                    category = %event.category.as_str(),
                    action = %event.action,
                    actor_user_id = ?event.actor_user_id,
                    actor_ip = ?event.actor_ip,
                    target_resource = ?event.target_resource,
                    target_id = ?event.target_id,
                    session_id = ?event.session_id,
                    "审计事件"
                );
            }
            AuditSeverity::Warning => {
                tracing::warn!(
                    event_id = %event.event_id,
                    category = %event.category.as_str(),
                    action = %event.action,
                    actor_user_id = ?event.actor_user_id,
                    actor_ip = ?event.actor_ip,
                    target_resource = ?event.target_resource,
                    target_id = ?event.target_id,
                    session_id = ?event.session_id,
                    "审计警告"
                );
            }
            AuditSeverity::Critical => {
                tracing::error!(
                    event_id = %event.event_id,
                    category = %event.category.as_str(),
                    action = %event.action,
                    actor_user_id = ?event.actor_user_id,
                    actor_ip = ?event.actor_ip,
                    target_resource = ?event.target_resource,
                    target_id = ?event.target_id,
                    session_id = ?event.session_id,
                    "审计严重事件"
                );
            }
        }
        Ok(())
    }
    
    async fn flush(&self) -> Result<(), AuditError> {
        Ok(())
    }
}

/// 组合审计输出（支持多个输出目标）
#[derive(Debug)]
pub struct CompositeAuditSink {
    sinks: Vec<Arc<dyn AuditSink>>,
}

impl CompositeAuditSink {
    pub fn new() -> Self {
        Self {
            sinks: Vec::new(),
        }
    }
    
    pub fn add_sink(mut self, sink: Arc<dyn AuditSink>) -> Self {
        self.sinks.push(sink);
        self
    }
}

#[async_trait::async_trait]
impl AuditSink for CompositeAuditSink {
    async fn submit(&self, event: AuditEvent) -> Result<(), AuditError> {
        let mut errors = Vec::new();
        
        for sink in &self.sinks {
            if let Err(e) = sink.submit(event.clone()).await {
                error!("审计输出失败: {}", e);
                errors.push(e);
            }
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(AuditError::ServiceUnavailable)
        }
    }
    
    async fn flush(&self) -> Result<(), AuditError> {
        for sink in &self.sinks {
            if let Err(e) = sink.flush().await {
                error!("审计输出刷新失败: {}", e);
            }
        }
        Ok(())
    }
}

/// 审计服务
#[derive(Clone)]
pub struct AuditService {
    sink: Arc<dyn AuditSink>,
}

impl AuditService {
    pub fn new(sink: Arc<dyn AuditSink>) -> Self {
        Self { sink }
    }
    
    /// 提交审计事件
    pub async fn submit(&self, event: AuditEvent) -> Result<(), AuditError> {
        self.sink.submit(event).await
    }
    
    /// 刷新审计输出
    pub async fn flush(&self) -> Result<(), AuditError> {
        self.sink.flush().await
    }
    
    /// 记录用户登录事件
    pub async fn log_user_login(&self, user_id: i64, ip: Option<String>, success: bool) -> Result<(), AuditError> {
        let event = AuditEvent::new(
            AuditCategory::Authentication,
            if success { "user_login_success".to_string() } else { "user_login_failed".to_string() },
            if success { AuditSeverity::Info } else { AuditSeverity::Warning },
        )
        .with_actor(user_id);
        
        let event = if let Some(ip) = ip {
            event.with_ip(ip)
        } else {
            event
        };
        
        self.submit(event).await
    }
    
    /// 记录用户注册事件
    pub async fn log_user_registration(&self, user_id: i64, ip: Option<String>) -> Result<(), AuditError> {
        let event = AuditEvent::new(
            AuditCategory::UserManagement,
            "user_registration".to_string(),
            AuditSeverity::Info,
        )
        .with_actor(user_id);
        
        let event = if let Some(ip) = ip {
            event.with_ip(ip)
        } else {
            event
        };
        
        self.submit(event).await
    }
    
    /// 记录群组创建事件
    pub async fn log_group_creation(&self, user_id: i64, group_id: &str) -> Result<(), AuditError> {
        let event = AuditEvent::new(
            AuditCategory::GroupManagement,
            "group_creation".to_string(),
            AuditSeverity::Info,
        )
        .with_actor(user_id)
        .with_target_resource("group")
        .with_target_id(group_id);
        
        self.submit(event).await
    }
    
    /// 记录权限违规事件
    pub async fn log_permission_violation(&self, user_id: Option<i64>, resource: &str, action: &str, ip: Option<String>) -> Result<(), AuditError> {
        let mut event = AuditEvent::new(
            AuditCategory::Security,
            format!("permission_violation_{}", action),
            AuditSeverity::Critical,
        )
        .with_target_resource(resource);
        
        if let Some(user_id) = user_id {
            event = event.with_actor(user_id);
        }
        
        if let Some(ip) = ip {
            event = event.with_ip(ip);
        }
        
        self.submit(event).await
    }
    
    /// 记录数据访问事件
    pub async fn log_data_access(&self, user_id: i64, resource: &str, action: &str, target_id: Option<&str>) -> Result<(), AuditError> {
        let mut event = AuditEvent::new(
            AuditCategory::DataAccess,
            format!("data_{}_{}", action, resource),
            AuditSeverity::Info,
        )
        .with_actor(user_id)
        .with_target_resource(resource);
        
        if let Some(target_id) = target_id {
            event = event.with_target_id(target_id);
        }
        
        self.submit(event).await
    }
}


