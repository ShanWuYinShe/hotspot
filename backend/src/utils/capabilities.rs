/// 能力注册表
/// 
/// 集中管理各种基础设施能力组件，兼容新架构
/// 提供统一的能力发现和注入机制
use std::sync::Arc;

use crate::ports::{
    UserRepository, GroupRepository, MomentRepository,
    EventPublisher, NotificationTransport,
};

/// 能力注册表 - 集中管理各种基础设施能力
#[derive(Clone, Default)]
pub struct CapabilityRegistry {
    // 仓储端口能力
    user_repo_port: Option<Arc<dyn UserRepository>>,
    group_repo_port: Option<Arc<dyn GroupRepository>>,

    moment_repo_port: Option<Arc<dyn MomentRepository>>,
    
    // 基础设施端口能力
    event_publisher_port: Option<Arc<dyn EventPublisher>>,
    notification_transport_port: Option<Arc<dyn NotificationTransport>>,
    
    // 其他能力（预留扩展）
    #[allow(dead_code)]
    _placeholder: Option<()>, // 占位字段，避免空结构体
}

impl CapabilityRegistry {
    /// 创建新的能力注册表
    pub fn new() -> Self {
        Self::default()
    }
    
    /// 注册用户仓储端口
    pub fn with_user_repo_port(mut self, port: Arc<dyn UserRepository>) -> Self {
        self.user_repo_port = Some(port);
        self
    }
    
    /// 获取用户仓储端口
    pub fn user_repo_port(&self) -> Option<Arc<dyn UserRepository>> {
        self.user_repo_port.clone()
    }
    
    /// 注册群组仓储端口
    pub fn with_group_repo_port(mut self, port: Arc<dyn GroupRepository>) -> Self {
        self.group_repo_port = Some(port);
        self
    }
    
    /// 获取群组仓储端口
    pub fn group_repo_port(&self) -> Option<Arc<dyn GroupRepository>> {
        self.group_repo_port.clone()
    }
    

    /// 注册瞬间仓储端口
    pub fn with_moment_repo_port(mut self, port: Arc<dyn MomentRepository>) -> Self {
        self.moment_repo_port = Some(port);
        self
    }
    
    /// 获取瞬间仓储端口
    pub fn moment_repo_port(&self) -> Option<Arc<dyn MomentRepository>> {
        self.moment_repo_port.clone()
    }
    
    /// 注册事件发布端口
    pub fn with_event_publisher_port(mut self, port: Arc<dyn EventPublisher>) -> Self {
        self.event_publisher_port = Some(port);
        self
    }
    
    /// 获取事件发布端口
    pub fn event_publisher_port(&self) -> Option<Arc<dyn EventPublisher>> {
        self.event_publisher_port.clone()
    }
    
    /// 注册通知传输端口
    pub fn with_notification_transport_port(mut self, port: Arc<dyn NotificationTransport>) -> Self {
        self.notification_transport_port = Some(port);
        self
    }
    
    /// 获取通知传输端口
    pub fn notification_transport_port(&self) -> Option<Arc<dyn NotificationTransport>> {
        self.notification_transport_port.clone()
    }
    
    /// 检查是否有必要的能力
    pub fn has_required_capabilities(&self) -> bool {
        self.user_repo_port.is_some() && 
        self.event_publisher_port.is_some()
    }
    
    /// 获取能力统计信息
    pub fn get_capability_stats(&self) -> CapabilityStats {
        CapabilityStats {
            user_repo_available: self.user_repo_port.is_some(),
            group_repo_available: self.group_repo_port.is_some(),
            message_repo_available: false, // 已迁移到新架构
            moment_repo_available: self.moment_repo_port.is_some(),
            event_publisher_available: self.event_publisher_port.is_some(),
            notification_transport_available: self.notification_transport_port.is_some(),
        }
    }
}

/// 能力统计信息
#[derive(Debug, Clone)]
pub struct CapabilityStats {
    pub user_repo_available: bool,
    pub group_repo_available: bool,
    pub message_repo_available: bool,
    pub moment_repo_available: bool,
    pub event_publisher_available: bool,
    pub notification_transport_available: bool,
}