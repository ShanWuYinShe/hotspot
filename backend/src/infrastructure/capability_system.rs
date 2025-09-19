/// 能力注册系统 - 编译期feature选择和运行时能力装配
/// 
/// 系统特征：
/// - 编译期feature门控
/// - 运行时能力发现
/// - 动态能力装配
/// - 能力依赖管理
/// - 配置驱动启用

use std::collections::{HashMap, HashSet};
use std::sync::Arc;
use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use tokio::sync::RwLock;

use crate::domain::{DomainResult, DomainError};

// === 能力系统核心定义 ===

/// 能力ID - 唯一标识一个能力
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct CapabilityId(String);

impl CapabilityId {
    /// 创建新的能力ID
    pub fn new(id: &str) -> Self {
        Self(id.to_string())
    }
    
    /// 获取ID字符串
    pub fn as_str(&self) -> &str {
        &self.0
    }
}

impl std::fmt::Display for CapabilityId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}

/// 能力版本
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Serialize, Deserialize)]
pub struct CapabilityVersion {
    pub major: u32,
    pub minor: u32,
    pub patch: u32,
}

impl CapabilityVersion {
    /// 创建新版本
    pub fn new(major: u32, minor: u32, patch: u32) -> Self {
        Self { major, minor, patch }
    }
    
    /// 检查版本兼容性
    pub fn is_compatible_with(&self, required: &CapabilityVersion) -> bool {
        // 主版本必须匹配，次版本必须大于等于要求
        self.major == required.major && 
        (self.minor > required.minor || 
         (self.minor == required.minor && self.patch >= required.patch))
    }
}

impl std::fmt::Display for CapabilityVersion {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}.{}.{}", self.major, self.minor, self.patch)
    }
}

/// 能力状态
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum CapabilityStatus {
    /// 可用但未启用
    Available,
    /// 已启用
    Enabled,
    /// 已禁用
    Disabled,
    /// 启用失败
    Failed(String),
    /// 不兼容
    Incompatible(String),
}

/// 能力元数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CapabilityMetadata {
    /// 能力ID
    pub id: CapabilityId,
    /// 能力名称
    pub name: String,
    /// 能力描述
    pub description: String,
    /// 能力版本
    pub version: CapabilityVersion,
    /// 依赖的能力
    pub dependencies: Vec<CapabilityDependency>,
    /// 提供的服务
    pub provides: Vec<String>,
    /// 需要的配置
    pub required_config: Vec<String>,
    /// 是否为核心能力
    pub is_core: bool,
    /// 编译期feature标志
    pub feature_flag: Option<String>,
}

/// 能力依赖
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CapabilityDependency {
    /// 依赖的能力ID
    pub capability_id: CapabilityId,
    /// 最小版本要求
    pub min_version: CapabilityVersion,
    /// 是否为可选依赖
    pub optional: bool,
}

/// 能力特征 - 所有能力必须实现此特征
#[async_trait]
pub trait Capability: Send + Sync {
    /// 获取能力元数据
    fn metadata(&self) -> &CapabilityMetadata;
    
    /// 初始化能力
    async fn initialize(&mut self, config: &CapabilityConfig) -> DomainResult<()>;
    
    /// 启动能力
    async fn start(&mut self) -> DomainResult<()>;
    
    /// 停止能力
    async fn stop(&mut self) -> DomainResult<()>;
    
    /// 检查能力健康状态
    async fn health_check(&self) -> CapabilityHealthStatus;
    
    /// 获取能力统计信息
    async fn get_stats(&self) -> CapabilityStats {
        CapabilityStats::default()
    }
    
    /// 处理配置更新
    async fn on_config_update(&mut self, _config: &CapabilityConfig) -> DomainResult<()> {
        Ok(())
    }
}

/// 能力配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CapabilityConfig {
    /// 配置键值对
    pub settings: HashMap<String, serde_json::Value>,
    /// 是否启用
    pub enabled: bool,
    /// 优先级（数字越小优先级越高）
    pub priority: u32,
}

impl CapabilityConfig {
    /// 创建新配置
    pub fn new() -> Self {
        Self {
            settings: HashMap::new(),
            enabled: true,
            priority: 100,
        }
    }
    
    /// 设置配置值
    pub fn set<T: Serialize>(&mut self, key: &str, value: T) -> DomainResult<()> {
        let json_value = serde_json::to_value(value).map_err(|e| {
            DomainError::ValidationError {
                field: key.to_string(),
                reason: format!("Failed to serialize config value: {}", e),
            }
        })?;
        
        self.settings.insert(key.to_string(), json_value);
        Ok(())
    }
    
    /// 获取配置值
    pub fn get<T: for<'de> Deserialize<'de>>(&self, key: &str) -> DomainResult<Option<T>> {
        if let Some(value) = self.settings.get(key) {
            let result = serde_json::from_value(value.clone()).map_err(|e| {
                DomainError::ValidationError {
                    field: key.to_string(),
                    reason: format!("Failed to deserialize config value: {}", e),
                }
            })?;
            Ok(Some(result))
        } else {
            Ok(None)
        }
    }
    
    /// 获取必需的配置值
    pub fn get_required<T: for<'de> Deserialize<'de>>(&self, key: &str) -> DomainResult<T> {
        self.get(key)?.ok_or_else(|| DomainError::ValidationError {
            field: key.to_string(),
            reason: "Required configuration value is missing".to_string(),
        })
    }
}

impl Default for CapabilityConfig {
    fn default() -> Self {
        Self::new()
    }
}

/// 能力健康状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CapabilityHealthStatus {
    /// 是否健康
    pub is_healthy: bool,
    /// 状态消息
    pub message: String,
    /// 最后检查时间
    pub last_check: chrono::DateTime<chrono::Utc>,
    /// 详细信息
    pub details: HashMap<String, serde_json::Value>,
}

impl CapabilityHealthStatus {
    /// 创建健康状态
    pub fn healthy(message: &str) -> Self {
        Self {
            is_healthy: true,
            message: message.to_string(),
            last_check: chrono::Utc::now(),
            details: HashMap::new(),
        }
    }
    
    /// 创建不健康状态
    pub fn unhealthy(message: &str) -> Self {
        Self {
            is_healthy: false,
            message: message.to_string(),
            last_check: chrono::Utc::now(),
            details: HashMap::new(),
        }
    }
    
    /// 添加详细信息
    pub fn with_detail<T: Serialize>(mut self, key: &str, value: T) -> Self {
        if let Ok(json_value) = serde_json::to_value(value) {
            self.details.insert(key.to_string(), json_value);
        }
        self
    }
}

/// 能力统计信息
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct CapabilityStats {
    /// 启动时间
    pub started_at: Option<chrono::DateTime<chrono::Utc>>,
    /// 运行时间（秒）
    pub uptime_seconds: u64,
    /// 处理的请求数
    pub requests_processed: u64,
    /// 错误数
    pub errors_count: u64,
    /// 自定义指标
    pub custom_metrics: HashMap<String, f64>,
}

/// 能力注册表项
pub struct CapabilityRegistration {
    /// 能力实例
    pub capability: Box<dyn Capability>,
    /// 当前状态
    pub status: CapabilityStatus,
    /// 配置
    pub config: CapabilityConfig,
    /// 注册时间
    pub registered_at: chrono::DateTime<chrono::Utc>,
    /// 最后状态更新时间
    pub last_status_update: chrono::DateTime<chrono::Utc>,
}

/// 能力注册系统 - 管理所有能力的生命周期
#[allow(dead_code)]
pub struct CapabilityRegistry {
    /// 注册的能力
    capabilities: Arc<RwLock<HashMap<CapabilityId, CapabilityRegistration>>>,
    /// 能力启动顺序
    #[allow(dead_code)]
    startup_order: Arc<RwLock<Vec<CapabilityId>>>,
    /// 全局配置
    global_config: Arc<RwLock<HashMap<String, serde_json::Value>>>,
    /// 系统统计
    stats: Arc<RwLock<CapabilityRegistryStats>>,
}

/// 能力注册系统统计
#[derive(Debug, Default, Clone, Serialize, Deserialize)]
pub struct CapabilityRegistryStats {
    /// 总注册能力数
    pub total_capabilities: usize,
    /// 已启用能力数
    pub enabled_capabilities: usize,
    /// 失败能力数
    pub failed_capabilities: usize,
    /// 系统启动时间
    pub system_started_at: chrono::DateTime<chrono::Utc>,
    /// 最后健康检查时间
    pub last_health_check: Option<chrono::DateTime<chrono::Utc>>,
}

impl CapabilityRegistry {
    /// 创建新的能力注册系统
    pub fn new() -> Self {
        Self {
            capabilities: Arc::new(RwLock::new(HashMap::new())),
            startup_order: Arc::new(RwLock::new(Vec::new())),
            global_config: Arc::new(RwLock::new(HashMap::new())),
            stats: Arc::new(RwLock::new(CapabilityRegistryStats {
                system_started_at: chrono::Utc::now(),
                ..Default::default()
            })),
        }
    }
    
    /// 注册能力
    pub async fn register_capability(
        &self,
        capability: Box<dyn Capability>,
        config: CapabilityConfig,
    ) -> DomainResult<()> {
        let metadata = capability.metadata().clone();
        let capability_id = metadata.id.clone();
        
        // 检查能力是否已注册
        {
            let capabilities = self.capabilities.read().await;
            if capabilities.contains_key(&capability_id) {
                return Err(DomainError::BusinessRuleViolation {
                    rule: format!("Capability {} is already registered", capability_id),
                });
            }
        }
        
        // 检查编译期feature
        if let Some(feature_flag) = &metadata.feature_flag {
            if !self.is_feature_enabled(feature_flag) {
                tracing::info!(
                    capability_id = %capability_id,
                    feature_flag = %feature_flag,
                    "Capability skipped due to disabled feature flag"
                );
                return Ok(());
            }
        }
        
        // 创建注册项
        let registration = CapabilityRegistration {
            capability,
            status: CapabilityStatus::Available,
            config,
            registered_at: chrono::Utc::now(),
            last_status_update: chrono::Utc::now(),
        };
        
        // 注册能力
        {
            let mut capabilities = self.capabilities.write().await;
            capabilities.insert(capability_id.clone(), registration);
        }
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.total_capabilities += 1;
        }
        
        tracing::info!(
            capability_id = %capability_id,
            capability_name = %metadata.name,
            version = %metadata.version,
            "Capability registered successfully"
        );
        
        Ok(())
    }
    
    /// 启用能力
    pub async fn enable_capability(&self, capability_id: &CapabilityId) -> DomainResult<()> {
        let mut capabilities = self.capabilities.write().await;
        
        if let Some(registration) = capabilities.get_mut(capability_id) {
            // 检查依赖
            let dependencies = registration.capability.metadata().dependencies.clone();
            drop(capabilities); // 释放可变借用
            let capabilities_read = self.capabilities.read().await;
            self.check_dependencies(&dependencies, &capabilities_read).await?;
            drop(capabilities_read); // 释放不可变借用
            let mut capabilities = self.capabilities.write().await; // 重新获取可变借用
            let registration = capabilities.get_mut(capability_id).unwrap();
            
            // 初始化能力
            match registration.capability.initialize(&registration.config).await {
                Ok(()) => {
                    // 启动能力
                    match registration.capability.start().await {
                        Ok(()) => {
                            registration.status = CapabilityStatus::Enabled;
                            registration.last_status_update = chrono::Utc::now();
                            
                            tracing::info!(
                                capability_id = %capability_id,
                                "Capability enabled successfully"
                            );
                        }
                        Err(e) => {
                            registration.status = CapabilityStatus::Failed(e.to_string());
                            registration.last_status_update = chrono::Utc::now();
                            
                            return Err(DomainError::BusinessRuleViolation {
                                rule: format!("Failed to start capability {}: {}", capability_id, e),
                            });
                        }
                    }
                }
                Err(e) => {
                    registration.status = CapabilityStatus::Failed(e.to_string());
                    registration.last_status_update = chrono::Utc::now();
                    
                    return Err(DomainError::BusinessRuleViolation {
                        rule: format!("Failed to initialize capability {}: {}", capability_id, e),
                    });
                }
            }
        } else {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("Capability {} not found", capability_id),
            });
        }
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.enabled_capabilities += 1;
        }
        
        Ok(())
    }
    
    /// 禁用能力
    pub async fn disable_capability(&self, capability_id: &CapabilityId) -> DomainResult<()> {
        let mut capabilities = self.capabilities.write().await;
        
        if let Some(registration) = capabilities.get_mut(capability_id) {
            if matches!(registration.status, CapabilityStatus::Enabled) {
                match registration.capability.stop().await {
                    Ok(()) => {
                        registration.status = CapabilityStatus::Disabled;
                        registration.last_status_update = chrono::Utc::now();
                        
                        tracing::info!(
                            capability_id = %capability_id,
                            "Capability disabled successfully"
                        );
                        
                        // 更新统计
                        let mut stats = self.stats.write().await;
                        if stats.enabled_capabilities > 0 {
                            stats.enabled_capabilities -= 1;
                        }
                    }
                    Err(e) => {
                        tracing::error!(
                            capability_id = %capability_id,
                            error = %e,
                            "Failed to stop capability"
                        );
                        
                        registration.status = CapabilityStatus::Failed(e.to_string());
                        registration.last_status_update = chrono::Utc::now();
                    }
                }
            }
        } else {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("Capability {} not found", capability_id),
            });
        }
        
        Ok(())
    }
    
    /// 批量启用能力
    pub async fn enable_capabilities(&self, capability_ids: Vec<CapabilityId>) -> DomainResult<()> {
        // 计算启动顺序
        let ordered_ids = self.calculate_startup_order(capability_ids).await?;
        
        // 按顺序启用能力
        for capability_id in ordered_ids {
            if let Err(e) = self.enable_capability(&capability_id).await {
                tracing::error!(
                    capability_id = %capability_id,
                    error = %e,
                    "Failed to enable capability during batch operation"
                );
                // 继续处理其他能力，不中断整个流程
            }
        }
        
        Ok(())
    }
    
    /// 启用所有可用能力
    pub async fn enable_all_capabilities(&self) -> DomainResult<()> {
        let capability_ids: Vec<CapabilityId> = {
            let capabilities = self.capabilities.read().await;
            capabilities.keys().cloned().collect()
        };
        
        self.enable_capabilities(capability_ids).await
    }
    
    /// 关闭所有能力
    pub async fn shutdown_all(&self) -> DomainResult<()> {
        let capability_ids: Vec<CapabilityId> = {
            let capabilities = self.capabilities.read().await;
            capabilities
                .iter()
                .filter(|(_, reg)| matches!(reg.status, CapabilityStatus::Enabled))
                .map(|(id, _)| id.clone())
                .collect()
        };
        
        // 逆序关闭能力
        for capability_id in capability_ids.into_iter().rev() {
            if let Err(e) = self.disable_capability(&capability_id).await {
                tracing::error!(
                    capability_id = %capability_id,
                    error = %e,
                    "Failed to disable capability during shutdown"
                );
            }
        }
        
        tracing::info!("All capabilities shutdown completed");
        Ok(())
    }
    
    /// 执行健康检查
    pub async fn health_check(&self) -> CapabilityRegistryHealth {
        let mut health = CapabilityRegistryHealth {
            is_healthy: true,
            capability_health: HashMap::new(),
            failed_capabilities: Vec::new(),
            total_capabilities: 0,
            enabled_capabilities: 0,
        };
        
        let capabilities = self.capabilities.read().await;
        health.total_capabilities = capabilities.len();
        
        for (capability_id, registration) in capabilities.iter() {
            if matches!(registration.status, CapabilityStatus::Enabled) {
                health.enabled_capabilities += 1;
                
                let capability_health = registration.capability.health_check().await;
                
                if !capability_health.is_healthy {
                    health.is_healthy = false;
                    health.failed_capabilities.push(capability_id.clone());
                }
                
                health.capability_health.insert(capability_id.clone(), capability_health);
            } else if matches!(registration.status, CapabilityStatus::Failed(_)) {
                health.is_healthy = false;
                health.failed_capabilities.push(capability_id.clone());
            }
        }
        
        // 更新统计
        {
            let mut stats = self.stats.write().await;
            stats.last_health_check = Some(chrono::Utc::now());
        }
        
        health
    }
    
    /// 获取能力状态
    pub async fn get_capability_status(&self, capability_id: &CapabilityId) -> Option<CapabilityStatus> {
        let capabilities = self.capabilities.read().await;
        capabilities.get(capability_id).map(|reg| reg.status.clone())
    }
    
    /// 获取所有能力状态
    pub async fn get_all_capability_status(&self) -> HashMap<CapabilityId, CapabilityStatus> {
        let capabilities = self.capabilities.read().await;
        capabilities
            .iter()
            .map(|(id, reg)| (id.clone(), reg.status.clone()))
            .collect()
    }
    
    /// 获取系统统计
    pub async fn get_stats(&self) -> CapabilityRegistryStats {
        self.stats.read().await.clone()
    }
    
    /// 设置全局配置
    pub async fn set_global_config<T: Serialize>(&self, key: &str, value: T) -> DomainResult<()> {
        let json_value = serde_json::to_value(value).map_err(|e| {
            DomainError::ValidationError {
                field: key.to_string(),
                reason: format!("Failed to serialize global config: {}", e),
            }
        })?;
        
        let mut global_config = self.global_config.write().await;
        global_config.insert(key.to_string(), json_value);
        
        Ok(())
    }
    
    /// 获取全局配置
    pub async fn get_global_config<T: for<'de> Deserialize<'de>>(&self, key: &str) -> DomainResult<Option<T>> {
        let global_config = self.global_config.read().await;
        
        if let Some(value) = global_config.get(key) {
            let result = serde_json::from_value(value.clone()).map_err(|e| {
                DomainError::ValidationError {
                    field: key.to_string(),
                    reason: format!("Failed to deserialize global config: {}", e),
                }
            })?;
            Ok(Some(result))
        } else {
            Ok(None)
        }
    }
    
    /// 检查编译期feature是否启用
    fn is_feature_enabled(&self, feature_flag: &str) -> bool {
        // 这里实现编译期feature检查逻辑
        // 在实际项目中，这些会通过 cfg! 宏在编译期确定
        match feature_flag {
            "websocket" => true, // WebSocket 是核心功能，始终启用
            "events" => true, // 事件系统是核心功能，始终启用
            "cache" => cfg!(any(feature = "cache_memory", feature = "cache_redis")),
            "metrics" => cfg!(any(feature = "metrics_basic", feature = "metrics_prometheus")),
            "email" => cfg!(feature = "email_notifications"),
            "notifications" => true, // 通知系统是核心功能，始终启用
            _ => true, // 默认启用未知feature
        }
    }
    
    /// 检查能力依赖
    async fn check_dependencies(
        &self,
        dependencies: &[CapabilityDependency],
        capabilities: &HashMap<CapabilityId, CapabilityRegistration>,
    ) -> DomainResult<()> {
        for dep in dependencies {
            if let Some(dep_registration) = capabilities.get(&dep.capability_id) {
                // 检查版本兼容性
                let dep_version = &dep_registration.capability.metadata().version;
                if !dep_version.is_compatible_with(&dep.min_version) {
                    return Err(DomainError::BusinessRuleViolation {
                        rule: format!(
                            "Dependency {} version {} is not compatible with required version {}",
                            dep.capability_id, dep_version, dep.min_version
                        ),
                    });
                }
                
                // 检查依赖是否已启用
                if !matches!(dep_registration.status, CapabilityStatus::Enabled) && !dep.optional {
                    return Err(DomainError::BusinessRuleViolation {
                        rule: format!("Required dependency {} is not enabled", dep.capability_id),
                    });
                }
            } else if !dep.optional {
                return Err(DomainError::BusinessRuleViolation {
                    rule: format!("Required dependency {} is not registered", dep.capability_id),
                });
            }
        }
        
        Ok(())
    }
    
    /// 计算启动顺序（拓扑排序）
    async fn calculate_startup_order(&self, capability_ids: Vec<CapabilityId>) -> DomainResult<Vec<CapabilityId>> {
        let capabilities = self.capabilities.read().await;
        let mut ordered = Vec::new();
        let mut visited = HashSet::new();
        let mut visiting = HashSet::new();
        
        fn visit(
            capability_id: &CapabilityId,
            capabilities: &HashMap<CapabilityId, CapabilityRegistration>,
            ordered: &mut Vec<CapabilityId>,
            visited: &mut HashSet<CapabilityId>,
            visiting: &mut HashSet<CapabilityId>,
        ) -> DomainResult<()> {
            if visiting.contains(capability_id) {
                return Err(DomainError::BusinessRuleViolation {
                    rule: format!("Circular dependency detected involving {}", capability_id),
                });
            }
            
            if visited.contains(capability_id) {
                return Ok(());
            }
            
            visiting.insert(capability_id.clone());
            
            if let Some(registration) = capabilities.get(capability_id) {
                for dep in &registration.capability.metadata().dependencies {
                    if !dep.optional {
                        visit(&dep.capability_id, capabilities, ordered, visited, visiting)?;
                    }
                }
            }
            
            visiting.remove(capability_id);
            visited.insert(capability_id.clone());
            ordered.push(capability_id.clone());
            
            Ok(())
        }
        
        for capability_id in capability_ids {
            visit(&capability_id, &capabilities, &mut ordered, &mut visited, &mut visiting)?;
        }
        
        Ok(ordered)
    }
}

/// 能力注册系统健康状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CapabilityRegistryHealth {
    /// 整体是否健康
    pub is_healthy: bool,
    /// 各能力健康状态
    pub capability_health: HashMap<CapabilityId, CapabilityHealthStatus>,
    /// 失败的能力列表
    pub failed_capabilities: Vec<CapabilityId>,
    /// 总能力数
    pub total_capabilities: usize,
    /// 已启用能力数
    pub enabled_capabilities: usize,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_capability_version_compatibility() {
        let v1_0_0 = CapabilityVersion::new(1, 0, 0);
        let v1_1_0 = CapabilityVersion::new(1, 1, 0);
        let v1_0_1 = CapabilityVersion::new(1, 0, 1);
        let v2_0_0 = CapabilityVersion::new(2, 0, 0);
        
        assert!(v1_1_0.is_compatible_with(&v1_0_0));
        assert!(v1_0_1.is_compatible_with(&v1_0_0));
        assert!(!v1_0_0.is_compatible_with(&v1_1_0));
        assert!(!v2_0_0.is_compatible_with(&v1_0_0));
    }
    
    #[test]
    fn test_capability_config() {
        let mut config = CapabilityConfig::new();
        config.set("test_key", "test_value").unwrap();
        
        let value: String = config.get("test_key").unwrap().unwrap();
        assert_eq!(value, "test_value");
        
        let missing: Option<String> = config.get("missing_key").unwrap();
        assert!(missing.is_none());
    }
    
    #[tokio::test]
    async fn test_capability_registry_creation() {
        let registry = CapabilityRegistry::new();
        let stats = registry.get_stats().await;
        
        assert_eq!(stats.total_capabilities, 0);
        assert_eq!(stats.enabled_capabilities, 0);
    }
}
