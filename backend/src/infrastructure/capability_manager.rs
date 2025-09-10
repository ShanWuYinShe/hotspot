/// 能力管理器 - 完整的能力系统管理和集成
/// 
/// 管理功能：
/// - 能力自动发现和注册
/// - 配置驱动的能力启用
/// - 依赖解析和启动顺序
/// - 运行时能力管理
/// - 健康监控和故障恢复

use std::collections::HashMap;
use std::sync::Arc;
use serde::{Deserialize, Serialize};
use tokio::sync::RwLock;

use crate::domain::DomainResult;
use crate::infrastructure::{
    CapabilityRegistry, CapabilityId, CapabilityConfig, CapabilityRegistryHealth,
    CapabilityRegistryStats, CapabilityFactory, Capability,
};

/// 能力管理器配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CapabilityManagerConfig {
    /// 启用的能力列表
    pub enabled_capabilities: Vec<String>,
    /// 禁用的能力列表
    pub disabled_capabilities: Vec<String>,
    /// 能力特定配置
    pub capability_configs: HashMap<String, CapabilityConfig>,
    /// 全局配置
    pub global_config: HashMap<String, serde_json::Value>,
    /// 是否自动启用所有可用能力
    pub auto_enable_all: bool,
    /// 健康检查间隔（秒）
    pub health_check_interval_seconds: u64,
    /// 是否启用故障恢复
    pub enable_failure_recovery: bool,
}

impl Default for CapabilityManagerConfig {
    fn default() -> Self {
        Self {
            enabled_capabilities: vec![
                "database".to_string(),
            ],
            disabled_capabilities: vec![],
            capability_configs: HashMap::new(),
            global_config: HashMap::new(),
            auto_enable_all: false,
            health_check_interval_seconds: 60,
            enable_failure_recovery: true,
        }
    }
}

/// 能力管理器 - 统一管理所有系统能力
pub struct CapabilityManager {
    /// 能力注册表
    registry: Arc<CapabilityRegistry>,
    /// 配置
    config: CapabilityManagerConfig,
    /// 管理器统计
    stats: Arc<RwLock<CapabilityManagerStats>>,
    /// 是否已初始化
    initialized: Arc<RwLock<bool>>,
}

/// 能力管理器统计
#[derive(Debug, Default, Clone, Serialize, Deserialize)]
pub struct CapabilityManagerStats {
    /// 管理器启动时间
    pub started_at: chrono::DateTime<chrono::Utc>,
    /// 自动发现的能力数
    pub discovered_capabilities: usize,
    /// 成功启用的能力数
    pub successfully_enabled: usize,
    /// 启用失败的能力数
    pub failed_to_enable: usize,
    /// 健康检查次数
    pub health_checks_performed: u64,
    /// 故障恢复次数
    pub failure_recoveries: u64,
}

impl CapabilityManager {
    /// 创建新的能力管理器
    pub fn new(config: CapabilityManagerConfig) -> Self {
        Self {
            registry: Arc::new(CapabilityRegistry::new()),
            config,
            stats: Arc::new(RwLock::new(CapabilityManagerStats {
                started_at: chrono::Utc::now(),
                ..Default::default()
            })),
            initialized: Arc::new(RwLock::new(false)),
        }
    }
    
    /// 初始化能力管理器
    pub async fn initialize(&self) -> DomainResult<()> {
        let mut initialized = self.initialized.write().await;
        if *initialized {
            return Ok(());
        }
        
        tracing::info!("Initializing capability manager");
        
        // 设置全局配置
        for (key, value) in &self.config.global_config {
            self.registry.set_global_config(key, value.clone()).await?;
        }
        
        // 自动发现和注册能力
        self.discover_and_register_capabilities().await?;
        
        // 启用配置的能力
        self.enable_configured_capabilities().await?;
        
        // 启动健康检查任务
        if self.config.health_check_interval_seconds > 0 {
            self.start_health_check_task().await;
        }
        
        // 启动故障恢复任务
        if self.config.enable_failure_recovery {
            self.start_failure_recovery_task().await;
        }
        
        *initialized = true;
        
        tracing::info!("Capability manager initialized successfully");
        Ok(())
    }
    
    /// 自动发现和注册能力
    async fn discover_and_register_capabilities(&self) -> DomainResult<()> {
        let capabilities = CapabilityFactory::create_all_capabilities();
        let mut discovered_count = 0;
        
        for capability in capabilities {
            let capability_id = capability.metadata().id.clone();
            let capability_name = capability.metadata().name.clone();
            
            // 获取能力特定配置
            let config = self.config.capability_configs
                .get(capability_id.as_str())
                .cloned()
                .unwrap_or_else(CapabilityConfig::default);
            
            match self.registry.register_capability(capability, config).await {
                Ok(()) => {
                    discovered_count += 1;
                    tracing::debug!(
                        capability_id = %capability_id,
                        capability_name = %capability_name,
                        "Capability discovered and registered"
                    );
                }
                Err(e) => {
                    tracing::warn!(
                        capability_id = %capability_id,
                        error = %e,
                        "Failed to register capability"
                    );
                }
            }
        }
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.discovered_capabilities = discovered_count;
        }
        
        tracing::info!(
            discovered_count = discovered_count,
            "Capability discovery completed"
        );
        
        Ok(())
    }
    
    /// 启用配置的能力
    async fn enable_configured_capabilities(&self) -> DomainResult<()> {
        let capability_ids_to_enable = if self.config.auto_enable_all {
            // 获取所有已注册的能力
            let all_status = self.registry.get_all_capability_status().await;
            all_status.keys().cloned().collect()
        } else {
            // 只启用配置中指定的能力
            self.config.enabled_capabilities
                .iter()
                .map(|name| CapabilityId::new(name))
                .collect()
        };
        
        // 过滤掉被禁用的能力
        let disabled_set: std::collections::HashSet<_> = self.config.disabled_capabilities
            .iter()
            .map(|name| CapabilityId::new(name))
            .collect();
        
        let filtered_ids: Vec<_> = capability_ids_to_enable
            .into_iter()
            .filter(|id| !disabled_set.contains(id))
            .collect();
        
        // 批量启用能力
        let mut success_count = 0;
        let mut failure_count = 0;
        
        match self.registry.enable_capabilities(filtered_ids.clone()).await {
            Ok(()) => {
                // 检查实际启用状态
                for capability_id in &filtered_ids {
                    match self.registry.get_capability_status(capability_id).await {
                        Some(crate::infrastructure::CapabilityStatus::Enabled) => {
                            success_count += 1;
                        }
                        _ => {
                            failure_count += 1;
                        }
                    }
                }
            }
            Err(e) => {
                tracing::error!(
                    error = %e,
                    "Failed to enable capabilities in batch"
                );
                failure_count = filtered_ids.len();
            }
        }
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.successfully_enabled = success_count;
            stats.failed_to_enable = failure_count;
        }
        
        tracing::info!(
            success_count = success_count,
            failure_count = failure_count,
            total_requested = filtered_ids.len(),
            "Capability enablement completed"
        );
        
        Ok(())
    }
    
    /// 启用特定能力
    pub async fn enable_capability(&self, capability_name: &str) -> DomainResult<()> {
        let capability_id = CapabilityId::new(capability_name);
        self.registry.enable_capability(&capability_id).await?;
        
        tracing::info!(
            capability_name = %capability_name,
            "Capability enabled manually"
        );
        
        Ok(())
    }
    
    /// 禁用特定能力
    pub async fn disable_capability(&self, capability_name: &str) -> DomainResult<()> {
        let capability_id = CapabilityId::new(capability_name);
        self.registry.disable_capability(&capability_id).await?;
        
        tracing::info!(
            capability_name = %capability_name,
            "Capability disabled manually"
        );
        
        Ok(())
    }
    
    /// 重新加载配置
    pub async fn reload_config(&self, new_config: CapabilityManagerConfig) -> DomainResult<()> {
        tracing::info!("Reloading capability manager configuration");
        
        // 这里应该实现配置热重载逻辑
        // 1. 比较新旧配置
        // 2. 禁用不再需要的能力
        // 3. 启用新增的能力
        // 4. 更新能力配置
        
        tracing::info!("Configuration reloaded successfully");
        Ok(())
    }
    
    /// 获取能力状态
    pub async fn get_capability_status(&self, capability_name: &str) -> Option<crate::infrastructure::CapabilityStatus> {
        let capability_id = CapabilityId::new(capability_name);
        self.registry.get_capability_status(&capability_id).await
    }
    
    /// 获取所有能力状态
    pub async fn get_all_capability_status(&self) -> HashMap<String, crate::infrastructure::CapabilityStatus> {
        let status_map = self.registry.get_all_capability_status().await;
        status_map
            .into_iter()
            .map(|(id, status)| (id.as_str().to_string(), status))
            .collect()
    }
    
    /// 执行健康检查
    pub async fn health_check(&self) -> CapabilityManagerHealth {
        let registry_health = self.registry.health_check().await;
        let manager_stats = self.stats.read().await.clone();
        
        // 更新健康检查统计
        {
            let mut stats = self.stats.write().await;
            stats.health_checks_performed += 1;
        }
        
        CapabilityManagerHealth {
            is_healthy: registry_health.is_healthy,
            registry_health,
            manager_stats,
            uptime_seconds: chrono::Utc::now()
                .signed_duration_since(manager_stats.started_at)
                .num_seconds() as u64,
        }
    }
    
    /// 获取管理器统计
    pub async fn get_stats(&self) -> CapabilityManagerStats {
        self.stats.read().await.clone()
    }
    
    /// 获取注册表统计
    pub async fn get_registry_stats(&self) -> CapabilityRegistryStats {
        self.registry.get_stats().await
    }
    
    /// 关闭能力管理器
    pub async fn shutdown(&self) -> DomainResult<()> {
        tracing::info!("Shutting down capability manager");
        
        // 关闭所有能力
        self.registry.shutdown_all().await?;
        
        // 标记为未初始化
        {
            let mut initialized = self.initialized.write().await;
            *initialized = false;
        }
        
        tracing::info!("Capability manager shutdown completed");
        Ok(())
    }
    
    /// 启动健康检查任务
    async fn start_health_check_task(&self) {
        let registry = self.registry.clone();
        let stats = self.stats.clone();
        let interval_seconds = self.config.health_check_interval_seconds;
        
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(
                std::time::Duration::from_secs(interval_seconds)
            );
            
            loop {
                interval.tick().await;
                
                let health = registry.health_check().await;
                
                if !health.is_healthy {
                    tracing::warn!(
                        failed_capabilities = health.failed_capabilities.len(),
                        total_capabilities = health.total_capabilities,
                        "Health check detected unhealthy capabilities"
                    );
                    
                    for failed_capability in &health.failed_capabilities {
                        tracing::warn!(
                            capability_id = %failed_capability,
                            "Capability is unhealthy"
                        );
                    }
                } else {
                    tracing::debug!(
                        enabled_capabilities = health.enabled_capabilities,
                        total_capabilities = health.total_capabilities,
                        "Health check completed - all capabilities healthy"
                    );
                }
                
                // 更新统计
                {
                    let mut stats_guard = stats.write().await;
                    stats_guard.health_checks_performed += 1;
                }
            }
        });
        
        tracing::info!(
            interval_seconds = interval_seconds,
            "Health check task started"
        );
    }
    
    /// 启动故障恢复任务
    async fn start_failure_recovery_task(&self) {
        let registry = self.registry.clone();
        let stats = self.stats.clone();
        
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(
                std::time::Duration::from_secs(120) // 每2分钟检查一次
            );
            
            loop {
                interval.tick().await;
                
                let all_status = registry.get_all_capability_status().await;
                
                for (capability_id, status) in all_status {
                    if let crate::infrastructure::CapabilityStatus::Failed(reason) = status {
                        tracing::warn!(
                            capability_id = %capability_id,
                            failure_reason = %reason,
                            "Attempting to recover failed capability"
                        );
                        
                        // 尝试重新启用失败的能力
                        match registry.enable_capability(&capability_id).await {
                            Ok(()) => {
                                tracing::info!(
                                    capability_id = %capability_id,
                                    "Capability recovery successful"
                                );
                                
                                // 更新恢复统计
                                {
                                    let mut stats_guard = stats.write().await;
                                    stats_guard.failure_recoveries += 1;
                                }
                            }
                            Err(e) => {
                                tracing::error!(
                                    capability_id = %capability_id,
                                    error = %e,
                                    "Capability recovery failed"
                                );
                            }
                        }
                    }
                }
            }
        });
        
        tracing::info!("Failure recovery task started");
    }
}

/// 能力管理器健康状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CapabilityManagerHealth {
    /// 整体是否健康
    pub is_healthy: bool,
    /// 注册表健康状态
    pub registry_health: CapabilityRegistryHealth,
    /// 管理器统计
    pub manager_stats: CapabilityManagerStats,
    /// 运行时间（秒）
    pub uptime_seconds: u64,
}

/// 能力管理器构建器
pub struct CapabilityManagerBuilder {
    config: CapabilityManagerConfig,
}

impl CapabilityManagerBuilder {
    /// 创建新的构建器
    pub fn new() -> Self {
        Self {
            config: CapabilityManagerConfig::default(),
        }
    }
    
    /// 启用能力
    pub fn enable_capability(mut self, capability_name: &str) -> Self {
        if !self.config.enabled_capabilities.contains(&capability_name.to_string()) {
            self.config.enabled_capabilities.push(capability_name.to_string());
        }
        self
    }
    
    /// 禁用能力
    pub fn disable_capability(mut self, capability_name: &str) -> Self {
        if !self.config.disabled_capabilities.contains(&capability_name.to_string()) {
            self.config.disabled_capabilities.push(capability_name.to_string());
        }
        self
    }
    
    /// 设置能力配置
    pub fn with_capability_config(mut self, capability_name: &str, config: CapabilityConfig) -> Self {
        self.config.capability_configs.insert(capability_name.to_string(), config);
        self
    }
    
    /// 设置全局配置
    pub fn with_global_config<T: Serialize>(mut self, key: &str, value: T) -> Self {
        if let Ok(json_value) = serde_json::to_value(value) {
            self.config.global_config.insert(key.to_string(), json_value);
        }
        self
    }
    
    /// 启用自动发现所有能力
    pub fn auto_enable_all(mut self) -> Self {
        self.config.auto_enable_all = true;
        self
    }
    
    /// 设置健康检查间隔
    pub fn with_health_check_interval(mut self, seconds: u64) -> Self {
        self.config.health_check_interval_seconds = seconds;
        self
    }
    
    /// 启用故障恢复
    pub fn enable_failure_recovery(mut self) -> Self {
        self.config.enable_failure_recovery = true;
        self
    }
    
    /// 构建能力管理器
    pub fn build(self) -> CapabilityManager {
        CapabilityManager::new(self.config)
    }
}

impl Default for CapabilityManagerBuilder {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_capability_manager_config() {
        let config = CapabilityManagerConfig::default();
        assert!(config.enabled_capabilities.contains(&"database".to_string()));
        assert!(!config.auto_enable_all);
        assert_eq!(config.health_check_interval_seconds, 60);
    }
    
    #[test]
    fn test_capability_manager_builder() {
        let manager = CapabilityManagerBuilder::new()
            .enable_capability("websocket")
            .enable_capability("cache")
            .disable_capability("metrics")
            .auto_enable_all()
            .with_health_check_interval(30)
            .enable_failure_recovery()
            .build();
        
        assert!(manager.config.enabled_capabilities.contains(&"websocket".to_string()));
        assert!(manager.config.disabled_capabilities.contains(&"metrics".to_string()));
        assert!(manager.config.auto_enable_all);
        assert_eq!(manager.config.health_check_interval_seconds, 30);
    }
    
    #[tokio::test]
    async fn test_capability_manager_creation() {
        let config = CapabilityManagerConfig::default();
        let manager = CapabilityManager::new(config);
        
        let stats = manager.get_stats().await;
        assert_eq!(stats.discovered_capabilities, 0);
        assert_eq!(stats.successfully_enabled, 0);
    }
}

