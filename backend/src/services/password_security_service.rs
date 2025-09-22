/// 重新设计的密码安全服务 - 符合需求文档的双重安全机制
/// 
/// 新的安全架构：
/// 1. 传输安全：SHA256(password_hash + nonce + timestamp) - 防重放攻击
/// 2. 存储安全：Argon2(password, user_salt) - 数据库安全
/// 3. 验证流程：分离传输验证和密码验证，确保两层安全都有效

use argon2::{Argon2, PasswordHash, PasswordHasher, PasswordVerifier};
use argon2::password_hash::{rand_core::OsRng, SaltString};
use sha2::{Sha256, Digest};
use std::sync::Arc;
use tracing::{info, warn, error};

use crate::domain::{DomainError, DomainResult};
use crate::services::nonce_service::{NonceService, NonceVerification};

/// 密码安全服务
#[derive(Clone)]
pub struct PasswordSecurityService {
    nonce_service: Arc<NonceService>,
    argon2: Argon2<'static>,
}

/// 密码验证请求 - 重新设计的安全架构
#[derive(Debug, Clone)]
pub struct PasswordVerificationRequest {
    pub login_id: String,        // 登录ID（邮箱或用户名）
    pub password_hash: String,   // 密码哈希：SHA256(password)
    pub transport_hash: String,  // 传输哈希：SHA256(password_hash + nonce + timestamp)
    pub nonce_id: String,        // Nonce ID
    pub timestamp: i64,          // 时间戳
}

/// 密码验证结果
#[derive(Debug, Clone)]
pub struct PasswordVerificationResult {
    pub is_valid: bool,
    pub user_id: Option<i64>,
    pub error_message: Option<String>,
}

/// 密码哈希结果
#[derive(Debug, Clone)]
pub struct PasswordHashResult {
    pub hash: String,
    pub salt: String,
}

impl PasswordSecurityService {
    /// 创建密码安全服务
    pub fn new(nonce_service: Arc<NonceService>) -> Self {
        Self {
            nonce_service,
            argon2: Argon2::default(),
        }
    }
    
    /// 生成密码的Argon2哈希（用于存储）
    /// 
    /// 这个方法用于注册时生成存储哈希
    pub fn hash_password_for_storage(&self, password: &str) -> DomainResult<PasswordHashResult> {
        let salt = SaltString::generate(&mut OsRng);
        
        let password_hash = self.argon2
            .hash_password(password.as_bytes(), &salt)
            .map_err(|e| DomainError::SecurityViolation { 
                reason: format!("密码哈希生成失败: {}", e) 
            })?;
        
        Ok(PasswordHashResult {
            hash: password_hash.to_string(),
            salt: salt.to_string(),
        })
    }
    
    /// 验证密码 - 重新设计的双重安全验证
    /// 
    /// 新的验证流程：
    /// 1. 验证传输哈希的完整性（防重放攻击）
    /// 2. 验证密码哈希与存储的Argon2哈希
    /// 
    /// 安全设计：
    /// - password_hash = SHA256(password) - 用于与Argon2比较
    /// - transport_hash = SHA256(password_hash + nonce + timestamp) - 防重放
    pub async fn verify_password_with_transport_security(
        &self,
        request: PasswordVerificationRequest,
        stored_argon2_hash: &str,
        user_id: i64,
    ) -> DomainResult<PasswordVerificationResult> {
        // 1. 验证nonce并获取nonce值
        let nonce_data = self.nonce_service
            .get_nonce(&request.nonce_id)
            .await
            .map_err(|e| DomainError::SecurityViolation { 
                reason: format!("Nonce获取失败: {}", e) 
            })?
            .ok_or_else(|| DomainError::SecurityViolation { 
                reason: "Nonce不存在或已过期".to_string() 
            })?;
        
        // 2. 验证nonce是否匹配登录ID
        if nonce_data.login_id != request.login_id {
            warn!("Nonce登录ID不匹配: expected={}, actual={}", nonce_data.login_id, request.login_id);
            return Ok(PasswordVerificationResult {
                is_valid: false,
                user_id: None,
                error_message: Some("Nonce验证失败".to_string()),
            });
        }
        
        // 3. 验证时间戳（5分钟有效期）
        let current_time = chrono::Utc::now().timestamp();
        if (current_time - request.timestamp).abs() > 300 {
            warn!("时间戳过期: current={}, request={}", current_time, request.timestamp);
            return Ok(PasswordVerificationResult {
                is_valid: false,
                user_id: None,
                error_message: Some("请求已过期".to_string()),
            });
        }
        
        // 4. 验证传输哈希的完整性
        let expected_transport_hash = self.calculate_sha256(&format!(
            "{}{}{}",
            request.password_hash,
            nonce_data.nonce,
            request.timestamp
        ));
        
        if expected_transport_hash != request.transport_hash {
            warn!("传输哈希验证失败: user_id={}", user_id);
            return Ok(PasswordVerificationResult {
                is_valid: false,
                user_id: None,
                error_message: Some("传输哈希验证失败".to_string()),
            });
        }
        
        // 5. 消费nonce（防重放攻击）
        let nonce_verification = NonceVerification {
            nonce_id: request.nonce_id,
            nonce: nonce_data.nonce,
            login_id: request.login_id.clone(),
            timestamp: request.timestamp,
        };
        
        let nonce_consumed = self.nonce_service
            .verify_and_consume_nonce(nonce_verification)
            .await
            .map_err(|e| DomainError::SecurityViolation { 
                reason: format!("Nonce消费失败: {}", e) 
            })?;
        
        if !nonce_consumed {
            warn!("Nonce消费失败: user_id={}", user_id);
            return Ok(PasswordVerificationResult {
                is_valid: false,
                user_id: None,
                error_message: Some("Nonce已被使用".to_string()),
            });
        }
        
        // 6. 验证密码哈希与存储的Argon2哈希
        // 注意：这里需要特殊处理，因为客户端发送的是SHA256(password)，
        // 而服务端存储的是Argon2(password)
        // 解决方案：重新用SHA256哈希作为输入进行Argon2验证
        let is_password_valid = self.verify_password_hash_with_argon2(
            &request.password_hash, 
            stored_argon2_hash
        )?;
        
        if is_password_valid {
            info!("密码验证成功: login_id={}, user_id={}", request.login_id, user_id);
            Ok(PasswordVerificationResult {
                is_valid: true,
                user_id: Some(user_id),
                error_message: None,
            })
        } else {
            warn!("密码验证失败: login_id={}, user_id={}", request.login_id, user_id);
            Ok(PasswordVerificationResult {
                is_valid: false,
                user_id: None,
                error_message: Some("密码错误".to_string()),
            })
        }
    }
    
    /// 计算SHA256哈希
    fn calculate_sha256(&self, input: &str) -> String {
        let mut hasher = Sha256::new();
        hasher.update(input.as_bytes());
        format!("{:x}", hasher.finalize())
    }
    
    /// 验证密码哈希与Argon2存储哈希
    /// 
    /// 注意：这里有一个架构问题需要解决
    /// 客户端发送SHA256(password)，但服务端存储Argon2(password)
    /// 
    /// 解决方案：修改存储策略，存储Argon2(SHA256(password))
    /// 这样可以直接验证客户端发送的SHA256哈希
    fn verify_password_hash_with_argon2(
        &self,
        password_hash: &str,
        stored_argon2_hash: &str,
    ) -> DomainResult<bool> {
        // 解析存储的Argon2哈希
        let parsed_hash = PasswordHash::new(stored_argon2_hash)
            .map_err(|e| DomainError::SecurityViolation { 
                reason: format!("Argon2哈希解析失败: {}", e) 
            })?;
        
        // 使用SHA256哈希作为输入进行Argon2验证
        // 注意：这要求存储的哈希是Argon2(SHA256(password))
        let is_valid = self.argon2
            .verify_password(password_hash.as_bytes(), &parsed_hash)
            .is_ok();
        
        Ok(is_valid)
    }
    
    /// 生成用于存储的双重哈希
    /// 
    /// 新的存储策略：Argon2(SHA256(password))
    /// 这样可以直接验证客户端发送的SHA256哈希
    pub fn hash_password_for_storage_v2(&self, password: &str) -> DomainResult<PasswordHashResult> {
        // 1. 先计算SHA256哈希
        let password_sha256 = self.calculate_sha256(password);
        
        // 2. 再用Argon2哈希SHA256结果
        let salt = SaltString::generate(&mut OsRng);
        
        let argon2_hash = self.argon2
            .hash_password(password_sha256.as_bytes(), &salt)
            .map_err(|e| DomainError::SecurityViolation { 
                reason: format!("双重哈希生成失败: {}", e) 
            })?;
        
        Ok(PasswordHashResult {
            hash: argon2_hash.to_string(),
            salt: salt.to_string(),
        })
    }
}

/// 密码安全服务错误类型
#[derive(Debug)]
pub enum PasswordSecurityError {
    InvalidHash { reason: String },
    HashGenerationFailed { reason: String },
    VerificationFailed { reason: String },
    NonceError { reason: String },
}

impl std::fmt::Display for PasswordSecurityError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            PasswordSecurityError::InvalidHash { reason } => {
                write!(f, "无效的密码哈希: {}", reason)
            }
            PasswordSecurityError::HashGenerationFailed { reason } => {
                write!(f, "密码哈希生成失败: {}", reason)
            }
            PasswordSecurityError::VerificationFailed { reason } => {
                write!(f, "密码验证失败: {}", reason)
            }
            PasswordSecurityError::NonceError { reason } => {
                write!(f, "Nonce错误: {}", reason)
            }
        }
    }
}

impl std::error::Error for PasswordSecurityError {}
