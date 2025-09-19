/// Nonce服务 - 管理认证随机数的生成、存储和验证
/// 
/// 功能职责：
/// - 生成安全的随机nonce
/// - 在Redis中存储nonce（5分钟TTL）
/// - 验证nonce的有效性和一次性使用
/// - 防重放攻击保护
/// 
/// 安全特性：
/// - nonce一次性使用（验证后立即删除）
/// - 5分钟自动过期
/// - 32字符随机字符串
/// - 与用户login_id绑定
use std::sync::Arc;
use serde::{Deserialize, Serialize};
use chrono::Utc;
use tracing::{info, warn, error};
use rand::Rng;

use crate::domain::DomainResult;
/// 简单缓存服务接口 - 专为nonce服务设计
#[async_trait::async_trait]
pub trait CacheService: Send + Sync {
    async fn get(&self, key: &str) -> Result<Option<String>, String>;
    async fn set(&self, key: &str, value: &str, ttl: Option<u64>) -> Result<(), String>;
    async fn delete(&self, key: &str) -> Result<(), String>;
    async fn exists(&self, key: &str) -> Result<bool, String>;
    async fn expire(&self, key: &str, ttl: u64) -> Result<(), String>;
}

/// Nonce数据结构
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NonceData {
    pub nonce_id: String,
    pub nonce: String,
    pub login_id: String,
    pub timestamp: i64,
    pub expires_at: i64,
}

/// Nonce验证请求
#[derive(Debug, Clone)]
pub struct NonceVerification {
    pub nonce_id: String,
    pub nonce: String,
    pub login_id: String,
    pub timestamp: i64,
}

/// Nonce服务
pub struct NonceService {
    cache_service: Arc<dyn CacheService>,
}

impl NonceService {
    /// 创建新的Nonce服务
    pub fn new(cache_service: Arc<dyn CacheService>) -> Self {
        Self { cache_service }
    }
    
    /// 生成新的nonce
    pub async fn generate_nonce(&self, login_id: &str) -> DomainResult<NonceData> {
        let nonce_id = uuid::Uuid::new_v4().to_string();
        let nonce = Self::generate_random_string(32);
        let timestamp = Utc::now().timestamp_millis();
        let expires_at = timestamp + (5 * 60 * 1000); // 5分钟过期
        
        let nonce_data = NonceData {
            nonce_id: nonce_id.clone(),
            nonce: nonce.clone(),
            login_id: login_id.to_string(),
            timestamp,
            expires_at,
        };
        
        // 存储到Redis，5分钟TTL
        let cache_key = format!("nonce:{}", nonce_id);
        let serialized_data = serde_json::to_string(&nonce_data)
            .map_err(|e| crate::domain::DomainError::BusinessRuleViolation {
                rule: format!("Nonce序列化失败: {}", e),
            })?;
        
        self.cache_service
            .set(&cache_key, &serialized_data, Some(300)) // 5分钟 = 300秒
            .await
            .map_err(|e| crate::domain::DomainError::BusinessRuleViolation {
                rule: format!("Nonce存储失败: {}", e),
            })?;
        
        info!("生成nonce成功: nonce_id={}, login_id={}", nonce_id, login_id);
        Ok(nonce_data)
    }
    
    /// 获取nonce值（不消费）
    pub async fn get_nonce(&self, nonce_id: &str) -> DomainResult<Option<NonceData>> {
        let cache_key = format!("nonce:{}", nonce_id);
        
        match self.cache_service.get(&cache_key).await {
            Ok(Some(cached_data)) => {
                match serde_json::from_str::<NonceData>(&cached_data) {
                    Ok(nonce_data) => {
                        // 检查是否过期
                        let current_time = Utc::now().timestamp_millis();
                        if current_time > nonce_data.expires_at {
                            warn!("Nonce已过期: nonce_id={}", nonce_id);
                            // 删除过期的nonce
                            let _ = self.cache_service.delete(&cache_key).await;
                            return Ok(None);
                        }
                        Ok(Some(nonce_data))
                    }
                    Err(e) => {
                        error!("Nonce反序列化失败: nonce_id={}, error={}", nonce_id, e);
                        Ok(None)
                    }
                }
            }
            Ok(None) => {
                warn!("Nonce不存在: nonce_id={}", nonce_id);
                Ok(None)
            }
            Err(e) => {
                error!("获取nonce失败: nonce_id={}, error={}", nonce_id, e);
                Err(crate::domain::DomainError::BusinessRuleViolation {
                    rule: format!("获取nonce失败: {}", e),
                })
            }
        }
    }
    
    /// 验证并消费nonce（一次性使用）
    pub async fn verify_and_consume_nonce(&self, verification: NonceVerification) -> DomainResult<bool> {
        let cache_key = format!("nonce:{}", verification.nonce_id);
        
        // 从Redis获取nonce数据
        let cached_data = self.cache_service
            .get(&cache_key)
            .await
            .map_err(|e| crate::domain::DomainError::BusinessRuleViolation {
                rule: format!("Nonce获取失败: {}", e),
            })?;
        
        let cached_data = match cached_data {
            Some(data) => data,
            None => {
                warn!("Nonce不存在或已过期: nonce_id={}", verification.nonce_id);
                return Ok(false);
            }
        };
        
        // 反序列化nonce数据
        let nonce_data: NonceData = serde_json::from_str(&cached_data)
            .map_err(|e| crate::domain::DomainError::BusinessRuleViolation {
                rule: format!("Nonce反序列化失败: {}", e),
            })?;
        
        // 验证nonce数据
        let is_valid = self.validate_nonce_data(&nonce_data, &verification);
        
        if is_valid {
            // 验证成功，立即删除nonce（一次性使用）
            if let Err(e) = self.cache_service.delete(&cache_key).await {
                error!("删除已使用的nonce失败: {}", e);
                // 不返回错误，因为验证已经成功
            }
            
            info!("Nonce验证成功并已消费: nonce_id={}, login_id={}", 
                  verification.nonce_id, verification.login_id);
        } else {
            warn!("Nonce验证失败: nonce_id={}, login_id={}", 
                  verification.nonce_id, verification.login_id);
        }
        
        Ok(is_valid)
    }
    
    /// 清理过期的nonce（可选，Redis TTL会自动处理）
    pub async fn cleanup_expired_nonces(&self) -> DomainResult<u32> {
        // Redis的TTL机制会自动清理过期的nonce
        // 这个方法主要用于统计和监控
        info!("Nonce清理完成（由Redis TTL自动处理）");
        Ok(0)
    }
    
    /// 获取nonce统计信息
    pub async fn get_nonce_stats(&self) -> DomainResult<serde_json::Value> {
        // 这里可以实现nonce使用统计
        // 目前返回基本信息
        Ok(serde_json::json!({
            "service": "NonceService",
            "ttl_seconds": 300,
            "max_length": 32,
            "status": "active"
        }))
    }
    
    // ===== 私有辅助方法 =====
    
    /// 验证nonce数据的完整性
    fn validate_nonce_data(&self, stored: &NonceData, verification: &NonceVerification) -> bool {
        // 1. 验证nonce值匹配
        if stored.nonce != verification.nonce {
            warn!("Nonce值不匹配");
            return false;
        }
        
        // 2. 验证login_id匹配
        if stored.login_id != verification.login_id {
            warn!("Login ID不匹配");
            return false;
        }
        
        // 3. 验证时间戳（允许一定的时间偏差，比如30秒）
        let time_diff = (verification.timestamp - stored.timestamp).abs();
        if time_diff > 30000 { // 30秒 = 30000毫秒
            warn!("时间戳偏差过大: {}ms", time_diff);
            return false;
        }
        
        // 4. 验证是否过期
        let current_time = Utc::now().timestamp_millis();
        if current_time > stored.expires_at {
            warn!("Nonce已过期");
            return false;
        }
        
        true
    }
    
    /// 生成指定长度的随机字符串
    fn generate_random_string(length: usize) -> String {
        const CHARSET: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZ\
                                 abcdefghijklmnopqrstuvwxyz\
                                 0123456789";
        let mut rng = rand::rng();
        (0..length)
            .map(|_| {
                let idx = rng.gen_range(0..CHARSET.len());
                CHARSET[idx] as char
            })
            .collect()
    }
}

/// Nonce服务的错误类型
#[derive(Debug)]
pub enum NonceError {
    NotFound,
    ValidationFailed { reason: String },
    CacheError { source: String },
    SerializationError { source: String },
}

impl std::fmt::Display for NonceError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            NonceError::NotFound => write!(f, "Nonce不存在或已过期"),
            NonceError::ValidationFailed { reason } => write!(f, "Nonce验证失败: {}", reason),
            NonceError::CacheError { source } => write!(f, "缓存操作失败: {}", source),
            NonceError::SerializationError { source } => write!(f, "序列化错误: {}", source),
        }
    }
}

impl std::error::Error for NonceError {}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;
    use std::sync::Mutex;
    
    // Mock缓存服务用于测试
    struct MockCacheService {
        data: Arc<Mutex<HashMap<String, String>>>,
    }
    
    impl MockCacheService {
        fn new() -> Self {
            Self {
                data: Arc::new(Mutex::new(HashMap::new())),
            }
        }
    }
    
    #[async_trait::async_trait]
    impl CacheService for MockCacheService {
        async fn get(&self, key: &str) -> Result<Option<String>, String> {
            let data = self.data.lock().unwrap();
            Ok(data.get(key).cloned())
        }
        
        async fn set(&self, key: &str, value: &str, _ttl: Option<u64>) -> Result<(), String> {
            let mut data = self.data.lock().unwrap();
            data.insert(key.to_string(), value.to_string());
            Ok(())
        }
        
        async fn delete(&self, key: &str) -> Result<(), String> {
            let mut data = self.data.lock().unwrap();
            data.remove(key);
            Ok(())
        }
        
        async fn exists(&self, key: &str) -> Result<bool, String> {
            let data = self.data.lock().unwrap();
            Ok(data.contains_key(key))
        }
        
        async fn expire(&self, _key: &str, _ttl: u64) -> Result<(), String> {
            Ok(())
        }
    }
    
    #[tokio::test]
    async fn test_generate_and_verify_nonce() {
        let cache_service = Arc::new(MockCacheService::new());
        let nonce_service = NonceService::new(cache_service);
        
        // 生成nonce
        let login_id = "test@example.com";
        let nonce_data = nonce_service.generate_nonce(login_id).await.unwrap();
        
        assert_eq!(nonce_data.login_id, login_id);
        assert_eq!(nonce_data.nonce.len(), 32);
        
        // 验证nonce
        let verification = NonceVerification {
            nonce_id: nonce_data.nonce_id.clone(),
            nonce: nonce_data.nonce.clone(),
            login_id: login_id.to_string(),
            timestamp: nonce_data.timestamp,
        };
        
        let is_valid = nonce_service.verify_and_consume_nonce(verification).await.unwrap();
        assert!(is_valid);
        
        // 再次验证应该失败（一次性使用）
        let verification2 = NonceVerification {
            nonce_id: nonce_data.nonce_id,
            nonce: nonce_data.nonce,
            login_id: login_id.to_string(),
            timestamp: nonce_data.timestamp,
        };
        
        let is_valid2 = nonce_service.verify_and_consume_nonce(verification2).await.unwrap();
        assert!(!is_valid2);
    }
}
