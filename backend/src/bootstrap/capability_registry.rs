/// 能力注册表
/// 
/// 统一管理和装配系统能力，支持编译期feature选择和运行时配置控制
use std::collections::HashMap;
use crate::domain::DomainResult;
use serde::{Deserialize, Serialize};

/// 能力类型枚举
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum CapabilityType {
    /// 数据库连接
    Database,
    /// 缓存服务
    Cache,
    /// 任务队列
    JobQueue,
    /// 通知传输
    NotificationTransport,
    /// 通知策略
    NotificationPolicy,
    /// 地理位置服务
    GeoService,
    /// 监控指标
    Metrics,
    /// WebSocket管理
    WebSocket,
}

/// 能力状态
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum CapabilityStatus {
    /// 未注册
    NotRegistered,
    /// 已注册但未启用
    Registered,
    /// 已启用
    Enabled,
    /// 已禁用
    Disabled,
    /// 出现错误
    Error(String),
}

/// 能力元数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CapabilityMetadata {
    pub capability_type: CapabilityType,
    pub name: String,
    pub version: String,
    pub description: String,
    pub dependencies: Vec<CapabilityType>,
    pub feature_flag: Option<String>,
    pub status: CapabilityStatus,
}

/// 能力注册表配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CapabilityRegistryConfig {
    /// 默认启用的能力
    pub default_capabilities: Vec<CapabilityType>,
    /// 可选能力配置
    pub optional_capabilities: HashMap<CapabilityType, bool>,
    /// 能力特定配置
    pub capability_configs: HashMap<CapabilityType, serde_json::Value>,
}

impl Default for CapabilityRegistryConfig {
    fn default() -> Self {
        Self {
            default_capabilities: vec![
                CapabilityType::Database,
                CapabilityType::Cache,
                CapabilityType::JobQueue,
                CapabilityType::NotificationTransport,
                CapabilityType::WebSocket,
            ],
            optional_capabilities: HashMap::new(),
            capability_configs: HashMap::new(),
        }
    }
}

/// 能力注册表
#[derive(Debug)]
#[derive(Clone)]
pub struct CapabilityRegistry {
    capabilities: HashMap<CapabilityType, CapabilityMetadata>,
    config: CapabilityRegistryConfig,
}

impl Default for CapabilityRegistry {
    fn default() -> Self {
        Self::new(CapabilityRegistryConfig::default())
    }
}

impl CapabilityRegistry {
    /// 创建新的能力注册表
    pub fn new(config: CapabilityRegistryConfig) -> Self {
        Self {
            capabilities: HashMap::new(),
            config,
        }
    }
    
    /// 创建默认的能力注册表
    pub fn with_default_config() -> Self {
        Self::new(CapabilityRegistryConfig::default())
    }
    
    /// 注册能力
    pub fn register_capability(&mut self, metadata: CapabilityMetadata) -> DomainResult<()> {
        // 检查依赖
        for dependency in &metadata.dependencies {
            if !self.is_capability_available(dependency) {
                return Err(crate::domain::DomainError::BusinessRuleViolation {
                    rule: format!("Missing dependency: {:?} for {:?}", dependency, metadata.capability_type),
                });
            }
        }
        
        // 检查feature flag
        if let Some(ref feature_flag) = metadata.feature_flag {
            if !self.is_feature_enabled(feature_flag) {
                log::debug!("Capability {:?} not registered due to disabled feature: {}", 
                           metadata.capability_type, feature_flag);
                return Ok(());
            }
        }
        
        self.capabilities.insert(metadata.capability_type.clone(), metadata);
        Ok(())
    }
    
    /// 启用能力
    pub fn enable_capability(&mut self, capability_type: CapabilityType) -> DomainResult<()> {
        // 先获取依赖信息，避免同时持有可变和不可变借用
        let dependencies = {
            if let Some(metadata) = self.capabilities.get(&capability_type) {
                metadata.dependencies.clone()
            } else {
                return Err(crate::domain::DomainError::BusinessRuleViolation {
                    rule: format!("Capability {:?} not found", capability_type),
                });
            }
        };
        
        // 检查依赖是否都已启用
        for dependency in &dependencies {
            if !self.is_capability_enabled(dependency) {
                return Err(crate::domain::DomainError::BusinessRuleViolation {
                    rule: format!("Dependency {:?} not enabled for {:?}", dependency, capability_type),
                });
            }
        }
        
        // 更新状态
        if let Some(metadata) = self.capabilities.get_mut(&capability_type) {
            metadata.status = CapabilityStatus::Enabled;
            log::info!("Enabled capability: {:?}", capability_type);
            Ok(())
        } else {
            Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: format!("Capability {:?} not registered", capability_type),
            })
        }
    }
    
    /// 禁用能力
    pub fn disable_capability(&mut self, capability_type: CapabilityType) -> DomainResult<()> {
        // 检查是否有其他能力依赖此能力
        let dependents: Vec<_> = self.capabilities
            .iter()
            .filter(|(_, metadata)| {
                metadata.dependencies.contains(&capability_type) && 
                metadata.status == CapabilityStatus::Enabled
            })
            .map(|(ct, _)| ct.clone())
            .collect();
        
        if !dependents.is_empty() {
            return Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: format!("Cannot disable {:?}: required by {:?}", capability_type, dependents),
            });
        }
        
        if let Some(metadata) = self.capabilities.get_mut(&capability_type) {
            metadata.status = CapabilityStatus::Disabled;
            log::info!("Disabled capability: {:?}", capability_type);
            Ok(())
        } else {
            Err(crate::domain::DomainError::BusinessRuleViolation {
                rule: format!("Capability {:?} not registered", capability_type),
            })
        }
    }
    
    /// 检查能力是否可用（已注册）
    pub fn is_capability_available(&self, capability_type: &CapabilityType) -> bool {
        self.capabilities.contains_key(capability_type)
    }
    
    /// 检查能力是否已启用
    pub fn is_capability_enabled(&self, capability_type: &CapabilityType) -> bool {
        self.capabilities
            .get(capability_type)
            .map(|metadata| metadata.status == CapabilityStatus::Enabled)
            .unwrap_or(false)
    }
    
    /// 获取能力元数据
    pub fn get_capability_metadata(&self, capability_type: &CapabilityType) -> Option<&CapabilityMetadata> {
        self.capabilities.get(capability_type)
    }
    
    /// 获取所有已启用的能力
    pub fn get_enabled_capabilities(&self) -> Vec<&CapabilityMetadata> {
        self.capabilities
            .values()
            .filter(|metadata| metadata.status == CapabilityStatus::Enabled)
            .collect()
    }
    
    /// 获取能力配置
    pub fn get_capability_config(&self, capability_type: &CapabilityType) -> Option<&serde_json::Value> {
        self.config.capability_configs.get(capability_type)
    }
    
    /// 设置能力配置
    pub fn set_capability_config(&mut self, capability_type: CapabilityType, config: serde_json::Value) {
        self.config.capability_configs.insert(capability_type, config);
    }
    
    /// 自动启用默认能力
    pub fn enable_default_capabilities(&mut self) -> DomainResult<Vec<String>> {
        let mut results = Vec::new();
        
        for capability_type in &self.config.default_capabilities.clone() {
            match self.enable_capability(capability_type.clone()) {
                Ok(()) => {
                    results.push(format!("Enabled default capability: {:?}", capability_type));
                }
                Err(e) => {
                    results.push(format!("Failed to enable {:?}: {}", capability_type, e));
                    log::error!("Failed to enable default capability {:?}: {}", capability_type, e);
                }
            }
        }
        
        Ok(results)
    }
    
    /// 验证能力配置的完整性
    pub fn validate_configuration(&self) -> DomainResult<Vec<String>> {
        let mut issues = Vec::new();
        
        for (capability_type, metadata) in &self.capabilities {
            // 检查依赖
            for dependency in &metadata.dependencies {
                if !self.is_capability_available(dependency) {
                    issues.push(format!("Capability {:?} missing dependency: {:?}", 
                                       capability_type, dependency));
                }
            }
            
            // 检查配置
            if self.config.capability_configs.get(capability_type).is_none() {
                log::debug!("No config found for capability: {:?}", capability_type);
            }
        }
        
        Ok(issues)
    }
    
    /// 生成能力状态报告
    pub fn generate_status_report(&self) -> CapabilityStatusReport {
        let mut enabled = Vec::new();
        let mut disabled = Vec::new();
        let mut errors = Vec::new();
        
        for (capability_type, metadata) in &self.capabilities {
            match &metadata.status {
                CapabilityStatus::Enabled => enabled.push(capability_type.clone()),
                CapabilityStatus::Disabled => disabled.push(capability_type.clone()),
                CapabilityStatus::Error(err) => errors.push((capability_type.clone(), err.clone())),
                _ => {}
            }
        }
        
        CapabilityStatusReport {
            total_capabilities: self.capabilities.len(),
            enabled_capabilities: enabled,
            disabled_capabilities: disabled,
            error_capabilities: errors,
        }
    }
    
    /// 检查feature flag是否启用
    fn is_feature_enabled(&self, feature_flag: &str) -> bool {
        // 检查编译时feature
        match feature_flag {
            "postgres" => true, // PostgreSQL 是核心功能，始终启用
            "cache_memory" => cfg!(any(feature = "cache_memory", not(feature = "cache_redis"))),
            "cache_redis" => cfg!(feature = "cache_redis"),
            "jobs_memory" => cfg!(any(feature = "jobs_memory", not(feature = "jobs_redis"))),
            "jobs_redis" => cfg!(feature = "jobs_redis"),
            "ws" => true, // WebSocket 是核心功能，始终启用
            "policy_friend" => true, // 通知策略是核心功能，始终启用
            "policy_group" => true, // 通知策略是核心功能，始终启用
            "policy_nearby" => true, // 通知策略是核心功能，始终启用
            "metrics_basic" => cfg!(feature = "metrics_basic"),
            "metrics_prometheus" => cfg!(feature = "metrics_prometheus"),
            _ => {
                log::warn!("Unknown feature flag: {}", feature_flag);
                false
            }
        }
    }
}

/// 能力状态报告
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CapabilityStatusReport {
    pub total_capabilities: usize,
    pub enabled_capabilities: Vec<CapabilityType>,
    pub disabled_capabilities: Vec<CapabilityType>,
    pub error_capabilities: Vec<(CapabilityType, String)>,
}

/// 便捷的能力创建宏
#[macro_export]
macro_rules! capability_metadata {
    ($type:expr, $name:expr, $version:expr, $description:expr) => {
        CapabilityMetadata {
            capability_type: $type,
            name: $name.to_string(),
            version: $version.to_string(),
            description: $description.to_string(),
            dependencies: Vec::new(),
            feature_flag: None,
            status: CapabilityStatus::Registered,
        }
    };
    
    ($type:expr, $name:expr, $version:expr, $description:expr, deps: [$($dep:expr),*]) => {
        CapabilityMetadata {
            capability_type: $type,
            name: $name.to_string(),
            version: $version.to_string(),
            description: $description.to_string(),
            dependencies: vec![$($dep),*],
            feature_flag: None,
            status: CapabilityStatus::Registered,
        }
    };
    
    ($type:expr, $name:expr, $version:expr, $description:expr, feature: $feature:expr) => {
        CapabilityMetadata {
            capability_type: $type,
            name: $name.to_string(),
            version: $version.to_string(),
            description: $description.to_string(),
            dependencies: Vec::new(),
            feature_flag: Some($feature.to_string()),
            status: CapabilityStatus::Registered,
        }
    };
    
    ($type:expr, $name:expr, $version:expr, $description:expr, deps: [$($dep:expr),*], feature: $feature:expr) => {
        CapabilityMetadata {
            capability_type: $type,
            name: $name.to_string(),
            version: $version.to_string(),
            description: $description.to_string(),
            dependencies: vec![$($dep),*],
            feature_flag: Some($feature.to_string()),
            status: CapabilityStatus::Registered,
        }
    };
}

