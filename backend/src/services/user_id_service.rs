/// 用户ID服务 - 实现用户ID编码规范和生成策略
/// 
/// 根据HotSpot后端需求文档实现的用户ID编码规范：
/// - 64位整数用户ID
/// - 高8位表示用户类型：1=临时用户，2=正式用户，3=系统用户
/// - 低56位为实际的用户序号
/// 
/// 功能特性：
/// - 类型安全的用户ID生成
/// - 用户类型判断和提取
/// - ID范围管理和验证
/// - 统计和监控支持
use std::sync::Arc;
use chrono::Utc;
use tracing::info;
use tokio::sync::Mutex;

use crate::domain::{DomainResult, DomainError};
use crate::domain::entities::user::{UserType, UserIdEncoder};

/// 用户ID生成器
pub struct UserIdService {
    /// 当前序号计数器（按用户类型分别维护）
    counters: Arc<Mutex<UserIdCounters>>,
}

/// 用户ID计数器
#[derive(Debug, Clone)]
struct UserIdCounters {
    /// 临时用户计数器
    temp_counter: i64,
    /// 正式用户计数器
    regular_counter: i64,
    /// 系统用户计数器
    system_counter: i64,
}

/// 用户ID生成请求
#[derive(Debug, Clone)]
pub struct UserIdGenerationRequest {
    pub user_type: UserType,
    pub batch_size: Option<u32>,  // 批量生成数量
}

/// 用户ID生成结果
#[derive(Debug, Clone)]
pub struct UserIdGenerationResult {
    pub user_ids: Vec<i64>,
    pub user_type: UserType,
    pub generated_count: u32,
    pub next_available_id: i64,
}

/// 用户ID信息
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct UserIdInfo {
    pub user_id: i64,
    pub user_type: UserType,
    pub type_code: u8,
    pub base_id: i64,
    pub is_valid: bool,
    pub type_name: String,
}

/// 用户ID统计信息
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct UserIdStatistics {
    pub total_users: u64,
    pub temp_users: u64,
    pub regular_users: u64,
    pub system_users: u64,
    pub next_temp_id: i64,
    pub next_regular_id: i64,
    pub next_system_id: i64,
    pub max_base_id: i64,
    pub generated_at: chrono::DateTime<chrono::Utc>,
}

impl UserIdService {
    /// 创建新的用户ID服务
    pub fn new() -> Self {
        Self {
            counters: Arc::new(Mutex::new(UserIdCounters {
                temp_counter: 1,
                regular_counter: 1,
                system_counter: 1,
            })),
        }
    }
    
    /// 从现有最大ID初始化计数器
    pub async fn initialize_from_max_ids(
        &self,
        max_temp_id: Option<i64>,
        max_regular_id: Option<i64>,
        max_system_id: Option<i64>,
    ) -> DomainResult<()> {
        let mut counters = self.counters.lock().await;
        
        // 从现有最大ID中提取基础ID并设置下一个计数器
        if let Some(max_id) = max_temp_id {
            counters.temp_counter = UserIdEncoder::extract_base_id(max_id) + 1;
        }
        
        if let Some(max_id) = max_regular_id {
            counters.regular_counter = UserIdEncoder::extract_base_id(max_id) + 1;
        }
        
        if let Some(max_id) = max_system_id {
            counters.system_counter = UserIdEncoder::extract_base_id(max_id) + 1;
        }
        
        info!("用户ID计数器初始化完成: temp={}, regular={}, system={}", 
              counters.temp_counter, counters.regular_counter, counters.system_counter);
        
        Ok(())
    }
    
    /// 生成单个用户ID
    pub async fn generate_user_id(&self, user_type: UserType) -> DomainResult<i64> {
        let mut counters = self.counters.lock().await;
        
        let base_id = match user_type {
            UserType::Temporary => {
                let id = counters.temp_counter;
                counters.temp_counter += 1;
                id
            }
            UserType::Regular => {
                let id = counters.regular_counter;
                counters.regular_counter += 1;
                id
            }
            UserType::System => {
                let id = counters.system_counter;
                counters.system_counter += 1;
                id
            }
        };
        
        // 检查基础ID是否超出范围（56位最大值）
        let max_base_id = (1i64 << 56) - 1;
        if base_id > max_base_id {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("用户ID基础序号超出范围: {}", base_id),
            });
        }
        
        let encoded_id = UserIdEncoder::encode_user_id(base_id, user_type);
        
        info!("生成用户ID: type={:?}, base_id={}, encoded_id={}", 
              user_type, base_id, encoded_id);
        
        Ok(encoded_id)
    }
    
    /// 批量生成用户ID
    pub async fn generate_user_ids(&self, request: UserIdGenerationRequest) -> DomainResult<UserIdGenerationResult> {
        let batch_size = request.batch_size.unwrap_or(1);
        if batch_size == 0 || batch_size > 1000 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "批量生成数量必须在1-1000之间".to_string(),
            });
        }
        
        let mut user_ids = Vec::with_capacity(batch_size as usize);
        
        for _ in 0..batch_size {
            let user_id = self.generate_user_id(request.user_type).await?;
            user_ids.push(user_id);
        }
        
        let next_available_id = self.get_next_available_id(request.user_type).await?;
        
        Ok(UserIdGenerationResult {
            user_ids,
            user_type: request.user_type,
            generated_count: batch_size,
            next_available_id,
        })
    }
    
    /// 获取用户ID信息
    pub fn get_user_id_info(&self, user_id: i64) -> DomainResult<UserIdInfo> {
        let user_type = UserIdEncoder::get_user_type(user_id)
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("无效的用户ID: {}", e),
            })?;
        
        let type_code = user_type.type_code();
        let base_id = UserIdEncoder::extract_base_id(user_id);
        let is_valid = self.validate_user_id(user_id);
        
        Ok(UserIdInfo {
            user_id,
            user_type,
            type_code,
            base_id,
            is_valid,
            type_name: user_type.as_str().to_string(),
        })
    }
    
    /// 验证用户ID格式
    pub fn validate_user_id(&self, user_id: i64) -> bool {
        // 检查用户ID是否为正数
        if user_id <= 0 {
            return false;
        }
        
        // 检查用户类型是否有效
        if UserIdEncoder::get_user_type(user_id).is_err() {
            return false;
        }
        
        // 检查基础ID是否在有效范围内
        let base_id = UserIdEncoder::extract_base_id(user_id);
        if base_id <= 0 || base_id >= (1i64 << 56) {
            return false;
        }
        
        true
    }
    
    /// 获取用户类型范围
    pub fn get_user_type_ranges(&self) -> serde_json::Value {
        serde_json::json!({
            "temporary": {
                "type_code": 1,
                "min_id": UserIdEncoder::encode_user_id(1, UserType::Temporary),
                "max_id": UserIdEncoder::encode_user_id((1i64 << 56) - 1, UserType::Temporary),
                "description": "临时用户ID范围"
            },
            "regular": {
                "type_code": 2,
                "min_id": UserIdEncoder::encode_user_id(1, UserType::Regular),
                "max_id": UserIdEncoder::encode_user_id((1i64 << 56) - 1, UserType::Regular),
                "description": "正式用户ID范围"
            },
            "system": {
                "type_code": 3,
                "min_id": UserIdEncoder::encode_user_id(1, UserType::System),
                "max_id": UserIdEncoder::encode_user_id((1i64 << 56) - 1, UserType::System),
                "description": "系统用户ID范围"
            }
        })
    }
    
    /// 获取用户ID统计信息
    pub async fn get_statistics(&self) -> DomainResult<UserIdStatistics> {
        let counters = self.counters.lock().await;
        
        Ok(UserIdStatistics {
            total_users: (counters.temp_counter + counters.regular_counter + counters.system_counter - 3) as u64,
            temp_users: (counters.temp_counter - 1) as u64,
            regular_users: (counters.regular_counter - 1) as u64,
            system_users: (counters.system_counter - 1) as u64,
            next_temp_id: UserIdEncoder::encode_user_id(counters.temp_counter, UserType::Temporary),
            next_regular_id: UserIdEncoder::encode_user_id(counters.regular_counter, UserType::Regular),
            next_system_id: UserIdEncoder::encode_user_id(counters.system_counter, UserType::System),
            max_base_id: (1i64 << 56) - 1,
            generated_at: Utc::now(),
        })
    }
    
    /// 格式化用户ID显示
    pub fn format_user_id(&self, user_id: i64) -> DomainResult<String> {
        let info = self.get_user_id_info(user_id)?;
        
        Ok(format!(
            "ID: {} | 类型: {} ({}) | 基础ID: {} | 有效: {}",
            user_id,
            info.type_name,
            info.type_code,
            info.base_id,
            if info.is_valid { "是" } else { "否" }
        ))
    }
    
    /// 检查用户ID是否为特定类型
    pub fn is_user_type(&self, user_id: i64, expected_type: UserType) -> bool {
        match UserIdEncoder::get_user_type(user_id) {
            Ok(user_type) => user_type == expected_type,
            Err(_) => false,
        }
    }
    
    /// 获取下一个可用ID
    async fn get_next_available_id(&self, user_type: UserType) -> DomainResult<i64> {
        let counters = self.counters.lock().await;
        
        let next_base_id = match user_type {
            UserType::Temporary => counters.temp_counter,
            UserType::Regular => counters.regular_counter,
            UserType::System => counters.system_counter,
        };
        
        Ok(UserIdEncoder::encode_user_id(next_base_id, user_type))
    }
}

impl Default for UserIdService {
    fn default() -> Self {
        Self::new()
    }
}

/// 用户ID服务错误类型
#[derive(Debug)]
pub enum UserIdServiceError {
    InvalidId { user_id: i64, reason: String },
    CounterOverflow { user_type: UserType },
    BatchSizeInvalid { size: u32 },
    InitializationFailed { reason: String },
}

impl std::fmt::Display for UserIdServiceError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            UserIdServiceError::InvalidId { user_id, reason } => {
                write!(f, "无效的用户ID {}: {}", user_id, reason)
            }
            UserIdServiceError::CounterOverflow { user_type } => {
                write!(f, "用户类型 {:?} 的ID计数器溢出", user_type)
            }
            UserIdServiceError::BatchSizeInvalid { size } => {
                write!(f, "无效的批量生成数量: {}", size)
            }
            UserIdServiceError::InitializationFailed { reason } => {
                write!(f, "用户ID服务初始化失败: {}", reason)
            }
        }
    }
}

impl std::error::Error for UserIdServiceError {}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_user_id_generation() {
        let service = UserIdService::new();
        
        // 测试生成不同类型的用户ID
        let temp_id = service.generate_user_id(UserType::Temporary).await.unwrap();
        let regular_id = service.generate_user_id(UserType::Regular).await.unwrap();
        let system_id = service.generate_user_id(UserType::System).await.unwrap();
        
        // 验证ID格式
        assert!(service.validate_user_id(temp_id));
        assert!(service.validate_user_id(regular_id));
        assert!(service.validate_user_id(system_id));
        
        // 验证用户类型
        assert!(service.is_user_type(temp_id, UserType::Temporary));
        assert!(service.is_user_type(regular_id, UserType::Regular));
        assert!(service.is_user_type(system_id, UserType::System));
        
        // 验证ID不相等
        assert_ne!(temp_id, regular_id);
        assert_ne!(regular_id, system_id);
        assert_ne!(temp_id, system_id);
    }
    
    #[tokio::test]
    async fn test_user_id_info() {
        let service = UserIdService::new();
        
        let user_id = service.generate_user_id(UserType::Regular).await.unwrap();
        let info = service.get_user_id_info(user_id).unwrap();
        
        assert_eq!(info.user_id, user_id);
        assert_eq!(info.user_type, UserType::Regular);
        assert_eq!(info.type_code, 2);
        assert_eq!(info.base_id, 1); // 第一个生成的ID
        assert!(info.is_valid);
        assert_eq!(info.type_name, "regular");
    }
    
    #[tokio::test]
    async fn test_batch_generation() {
        let service = UserIdService::new();
        
        let request = UserIdGenerationRequest {
            user_type: UserType::Temporary,
            batch_size: Some(5),
        };
        
        let result = service.generate_user_ids(request).await.unwrap();
        
        assert_eq!(result.user_ids.len(), 5);
        assert_eq!(result.generated_count, 5);
        assert_eq!(result.user_type, UserType::Temporary);
        
        // 验证所有ID都是有效的临时用户ID
        for user_id in &result.user_ids {
            assert!(service.validate_user_id(*user_id));
            assert!(service.is_user_type(*user_id, UserType::Temporary));
        }
        
        // 验证ID是连续的
        for i in 1..result.user_ids.len() {
            let prev_base = UserIdEncoder::extract_base_id(result.user_ids[i-1]);
            let curr_base = UserIdEncoder::extract_base_id(result.user_ids[i]);
            assert_eq!(curr_base, prev_base + 1);
        }
    }
    
    #[tokio::test]
    async fn test_statistics() {
        let service = UserIdService::new();
        
        // 生成一些用户ID
        service.generate_user_id(UserType::Temporary).await.unwrap();
        service.generate_user_id(UserType::Temporary).await.unwrap();
        service.generate_user_id(UserType::Regular).await.unwrap();
        service.generate_user_id(UserType::System).await.unwrap();
        
        let stats = service.get_statistics().await.unwrap();
        
        assert_eq!(stats.temp_users, 2);
        assert_eq!(stats.regular_users, 1);
        assert_eq!(stats.system_users, 1);
        assert_eq!(stats.total_users, 4);
    }
}





