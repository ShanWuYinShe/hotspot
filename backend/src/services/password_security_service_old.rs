/// 密码安全服务 - 实现双重哈希密码安全架构
/// 
/// 根据HotSpot后端需求文档实现的密码安全架构：
/// 1. 客户端：使用SHA-256计算 password_hash = SHA256(password + nonce + timestamp)
/// 2. 服务端：使用Argon2计算 stored_hash = Argon2(password_hash)
/// 
/// 安全特性：
/// - 双重哈希保护：SHA-256 + Argon2
/// - Nonce防重放攻击
/// - 时间戳验证
/// - 密码不明文传输
use std::sync::Arc;
use sha2::{Sha256, Digest};
use argon2::{Argon2, PasswordHash, PasswordHasher, PasswordVerifier};
use argon2::password_hash::{rand_core::OsRng, SaltString};
use chrono::Utc;
use tracing::{info, warn, error};

use crate::domain::{DomainResult, DomainError};
use crate::services::nonce_service::{NonceService, NonceVerification};

/// 密码安全服务
pub struct PasswordSecurityService {
    nonce_service: Arc<NonceService>,
    argon2: Argon2<'static>,
}

/// 密码验证请求 - 重新设计的安全架构
#[derive(Debug, Clone)]
pub struct PasswordVerificationRequest {
    pub password_hash: String,   // 密码哈希：SHA256(password)
    pub transport_hash: String,  // 传输哈希：SHA256(password_hash + nonce + timestamp)
    pub nonce_id: String,
    pub timestamp: i64,
}

// 重复定义已移除，使用上面的PasswordVerificationRequest

/// 密码验证结果
#[derive(Debug, Clone)]
pub struct PasswordVerificationResult {
    pub is_valid: bool,
    pub user_id: Option<i64>,
    pub nonce_consumed: bool,
}

impl PasswordSecurityService {
    /// 创建新的密码安全服务
    pub fn new(nonce_service: Arc<NonceService>) -> Self {
        Self {
            nonce_service,
            argon2: Argon2::default(),
        }
    }
    
    /// 验证客户端密码哈希并生成存储哈希
    /// 
    /// 流程：
    /// 1. 验证nonce的有效性
    /// 2. 重新计算客户端应该发送的哈希值
    /// 3. 验证客户端哈希是否正确
    /// 4. 使用Argon2对客户端哈希进行二次哈希
    pub async fn hash_password_with_nonce(
        &self,
        request: PasswordHashRequest,
        original_password: &str,  // 仅用于验证，实际场景中不会有
        login_id: &str,
    ) -> DomainResult<String> {
        // 1. 获取nonce数据
        let nonce_data = self.nonce_service
            .get_nonce(&request.nonce_id)
            .await?
            .ok_or_else(|| DomainError::BusinessRuleViolation {
                rule: "Nonce不存在或已过期".to_string(),
            })?;
        
        // 2. 验证nonce绑定的login_id
        if nonce_data.login_id != login_id {
            return Err(DomainError::BusinessRuleViolation {
                rule: "Nonce与login_id不匹配".to_string(),
            });
        }
        
        // 3. 重新计算期望的客户端哈希
        let expected_hash = self.calculate_client_hash(
            original_password,
            &nonce_data.nonce,
            request.timestamp,
        )?;
        
        // 4. 验证客户端发送的哈希是否正确
        if request.password_hash != expected_hash {
            warn!("客户端密码哈希验证失败: login_id={}", login_id);
            return Err(DomainError::BusinessRuleViolation {
                rule: "密码哈希验证失败".to_string(),
            });
        }
        
        // 5. 验证时间戳（允许一定偏差）
        let time_diff = (request.timestamp - nonce_data.timestamp).abs();
        if time_diff > 30000 { // 30秒
            return Err(DomainError::BusinessRuleViolation {
                rule: "时间戳偏差过大".to_string(),
            });
        }
        
        // 6. 使用Argon2对客户端哈希进行二次哈希
        let stored_hash = self.argon2_hash(&request.password_hash).await?;
        
        info!("密码双重哈希完成: login_id={}", login_id);
        Ok(stored_hash)
    }
    
    /// 验证密码
    /// 
    /// 流程：
    /// 1. 验证并消费nonce
    /// 2. 从数据库获取存储的Argon2哈希
    /// 3. 使用Argon2验证客户端哈希
    pub async fn verify_password_with_nonce(
        &self,
        request: PasswordVerificationRequest,
        stored_argon2_hash: &str,
    ) -> DomainResult<PasswordVerificationResult> {
        // 1. 验证并消费nonce
        let nonce_verification = NonceVerification {
            nonce_id: request.nonce_id.clone(),
            nonce: "".to_string(), // 这里需要从nonce服务获取
            login_id: request.login_id.clone(),
            timestamp: request.timestamp,
        };
        
        // 先获取nonce数据以验证
        let nonce_data = self.nonce_service
            .get_nonce(&request.nonce_id)
            .await?
            .ok_or_else(|| DomainError::BusinessRuleViolation {
                rule: "Nonce不存在或已过期".to_string(),
            })?;
        
        let nonce_verification = NonceVerification {
            nonce_id: request.nonce_id,
            nonce: nonce_data.nonce,
            login_id: request.login_id,
            timestamp: request.timestamp,
        };
        
        let nonce_valid = self.nonce_service
            .verify_and_consume_nonce(nonce_verification)
            .await?;
        
        if !nonce_valid {
            warn!("Nonce验证失败");
            return Ok(PasswordVerificationResult {
                is_valid: false,
                user_id: None,
                nonce_consumed: false,
            });
        }
        
        // 2. 使用Argon2验证客户端哈希
        let password_valid = self.argon2_verify(&request.password_hash, stored_argon2_hash).await?;
        
        if password_valid {
            info!("密码验证成功: login_id={}", nonce_data.login_id);
        } else {
            warn!("密码验证失败: login_id={}", nonce_data.login_id);
        }
        
        Ok(PasswordVerificationResult {
            is_valid: password_valid,
            user_id: None, // 需要调用方设置
            nonce_consumed: true,
        })
    }
    
    /// 计算客户端应该发送的SHA-256哈希
    /// 
    /// 公式：SHA256(password + nonce + timestamp)
    pub fn calculate_client_hash(
        &self,
        password: &str,
        nonce: &str,
        timestamp: i64,
    ) -> DomainResult<String> {
        let input = format!("{}{}{}", password, nonce, timestamp);
        let mut hasher = Sha256::new();
        hasher.update(input.as_bytes());
        let result = hasher.finalize();
        
        // 转换为十六进制字符串
        let hash_hex = format!("{:x}", result);
        Ok(hash_hex)
    }
    
    /// 验证客户端哈希格式
    pub fn validate_client_hash(&self, hash: &str) -> DomainResult<bool> {
        // SHA-256哈希应该是64个十六进制字符
        if hash.len() != 64 {
            return Ok(false);
        }
        
        // 验证是否为有效的十六进制字符串
        let is_hex = hash.chars().all(|c| c.is_ascii_hexdigit());
        Ok(is_hex)
    }
    
    /// 生成随机密码
    pub fn generate_random_password(&self, length: usize) -> DomainResult<String> {
        use rand::Rng;
        
        const CHARSET: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZ\
                                 abcdefghijklmnopqrstuvwxyz\
                                 0123456789!@#$%^&*";
        
        let mut rng = rand::rng();
        let password: String = (0..length)
            .map(|_| {
                let idx = rng.gen_range(0..CHARSET.len());
                CHARSET[idx] as char
            })
            .collect();
        
        Ok(password)
    }
    
    /// 获取密码安全配置
    pub fn get_security_config(&self) -> serde_json::Value {
        serde_json::json!({
            "client_hash_algorithm": "SHA-256",
            "server_hash_algorithm": "Argon2id",
            "nonce_ttl_seconds": 300,
            "timestamp_tolerance_ms": 30000,
            "min_password_length": 8,
            "max_password_length": 128,
            "hash_format": "hex"
        })
    }
    
    // ===== 私有辅助方法 =====
    
    /// 使用Argon2对密码哈希进行二次哈希
    async fn argon2_hash(&self, password_hash: &str) -> DomainResult<String> {
        let salt = SaltString::generate(&mut OsRng);
        
        let argon2_hash = self.argon2
            .hash_password(password_hash.as_bytes(), &salt)
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Argon2哈希失败: {}", e),
            })?
            .to_string();
        
        Ok(argon2_hash)
    }
    
    /// 使用Argon2验证密码哈希
    async fn argon2_verify(&self, password_hash: &str, stored_hash: &str) -> DomainResult<bool> {
        let parsed_hash = PasswordHash::new(stored_hash)
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("无效的Argon2哈希格式: {}", e),
            })?;
        
        let is_valid = self.argon2
            .verify_password(password_hash.as_bytes(), &parsed_hash)
            .is_ok();
        
        Ok(is_valid)
    }
}

/// 密码安全服务错误类型
#[derive(Debug)]
pub enum PasswordSecurityError {
    InvalidHash { reason: String },
    NonceError { reason: String },
    ValidationFailed { reason: String },
    HashingFailed { reason: String },
}

impl std::fmt::Display for PasswordSecurityError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            PasswordSecurityError::InvalidHash { reason } => {
                write!(f, "无效的密码哈希: {}", reason)
            }
            PasswordSecurityError::NonceError { reason } => {
                write!(f, "Nonce错误: {}", reason)
            }
            PasswordSecurityError::ValidationFailed { reason } => {
                write!(f, "验证失败: {}", reason)
            }
            PasswordSecurityError::HashingFailed { reason } => {
                write!(f, "哈希计算失败: {}", reason)
            }
        }
    }
}

impl std::error::Error for PasswordSecurityError {}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::services::nonce_service::{NonceService, CacheService};
    use std::collections::HashMap;
    use std::sync::Mutex;
    
    // Mock缓存服务
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
    async fn test_password_double_hashing() {
        let cache_service = Arc::new(MockCacheService::new());
        let nonce_service = Arc::new(NonceService::new(cache_service));
        let password_service = PasswordSecurityService::new(nonce_service.clone());
        
        let login_id = "test@example.com";
        let password = "test_password_123";
        
        // 1. 生成nonce
        let nonce_data = nonce_service.generate_nonce(login_id).await.unwrap();
        
        // 2. 模拟客户端计算哈希
        let client_hash = password_service
            .calculate_client_hash(password, &nonce_data.nonce, nonce_data.timestamp)
            .unwrap();
        
        // 3. 验证哈希格式
        assert!(password_service.validate_client_hash(&client_hash).unwrap());
        
        // 4. 服务端处理密码哈希
        let hash_request = PasswordHashRequest {
            password_hash: client_hash.clone(),
            nonce_id: nonce_data.nonce_id.clone(),
            timestamp: nonce_data.timestamp,
        };
        
        let stored_hash = password_service
            .hash_password_with_nonce(hash_request, password, login_id)
            .await
            .unwrap();
        
        // 5. 验证存储的哈希不等于客户端哈希（经过Argon2二次哈希）
        assert_ne!(stored_hash, client_hash);
        assert!(stored_hash.starts_with("$argon2"));
        
        println!("客户端哈希: {}", client_hash);
        println!("存储哈希: {}", stored_hash);
    }
    
    #[tokio::test]
    async fn test_client_hash_calculation() {
        let cache_service = Arc::new(MockCacheService::new());
        let nonce_service = Arc::new(NonceService::new(cache_service));
        let password_service = PasswordSecurityService::new(nonce_service);
        
        let password = "test123";
        let nonce = "abc123xyz";
        let timestamp = 1640995200000i64; // 固定时间戳
        
        let hash1 = password_service
            .calculate_client_hash(password, nonce, timestamp)
            .unwrap();
        
        let hash2 = password_service
            .calculate_client_hash(password, nonce, timestamp)
            .unwrap();
        
        // 相同输入应该产生相同哈希
        assert_eq!(hash1, hash2);
        assert_eq!(hash1.len(), 64); // SHA-256产生64个十六进制字符
        
        // 不同输入应该产生不同哈希
        let hash3 = password_service
            .calculate_client_hash("different", nonce, timestamp)
            .unwrap();
        
        assert_ne!(hash1, hash3);
    }
}

