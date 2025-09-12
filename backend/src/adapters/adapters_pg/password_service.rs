/// 密码服务适配器
/// 
/// 基于argon2的密码哈希和验证实现
use async_trait::async_trait;
use argon2::{Argon2, PasswordHash, PasswordHasher, PasswordVerifier};
use argon2::password_hash::{rand_core::OsRng, SaltString};
use crate::domain::{DomainResult, DomainError};
use crate::ports::PasswordService;

/// Argon2密码服务实现
#[derive(Clone)]
pub struct Argon2PasswordService {
    argon2: Argon2<'static>,
}

impl Argon2PasswordService {
    /// 创建新的Argon2密码服务
    pub fn new() -> Self {
        Self {
            argon2: Argon2::default(),
        }
    }
}

impl Default for Argon2PasswordService {
    fn default() -> Self {
        Self::new()
    }
}

#[async_trait]
impl PasswordService for Argon2PasswordService {
    async fn hash_password(&self, password: &str) -> DomainResult<String> {
        let salt = SaltString::generate(&mut OsRng);
        
        let password_hash = self.argon2
            .hash_password(password.as_bytes(), &salt)
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("密码哈希失败: {}", e),
            })?
            .to_string();
        
        Ok(password_hash)
    }
    
    async fn verify_password(&self, password: &str, hash: &str) -> DomainResult<bool> {
        let parsed_hash = PasswordHash::new(hash)
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("无效的密码哈希格式: {}", e),
            })?;
        
        let is_valid = self.argon2
            .verify_password(password.as_bytes(), &parsed_hash)
            .is_ok();
        
        Ok(is_valid)
    }

    async fn generate_random_password(&self, length: usize) -> DomainResult<String> {
        use rand::Rng;
        
        const CHARSET: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZ\
                                 abcdefghijklmnopqrstuvwxyz\
                                 0123456789";
        
        let mut rng = rand::thread_rng();
        let password: String = (0..length)
            .map(|_| {
                let idx = rng.random_range(0..CHARSET.len());
                CHARSET[idx] as char
            })
            .collect();
        
        Ok(password)
    }
}

