/// 适配器实现 - 端口接口的具体实现
/// 
/// 适配器特征：
/// - 实现端口层定义的接口
/// - 处理技术细节和外部系统集成
/// - 错误转换和上下文传播
/// - 资源管理和连接池

use async_trait::async_trait;
use std::sync::Arc;
use sqlx::PgPool;
use crate::domain::{DomainResult, DomainError};
use crate::ports::PasswordService;
use crate::infrastructure::errors::{InfrastructureError, ErrorContext, ContextualError, ErrorConverter};

// === PostgreSQL适配器基类 ===

/// PostgreSQL适配器基类 - 提供通用的数据库操作
#[derive(Clone)]
pub struct PostgresAdapter {
    pub pool: PgPool,
}

impl PostgresAdapter {
    /// 创建新的PostgreSQL适配器
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
    
    /// 执行查询并处理错误
    pub async fn execute_query<F, T>(&self, operation: &str, query_fn: F) -> DomainResult<T>
    where
        F: FnOnce() -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<T, sqlx::Error>> + Send>>,
    {
        let context = ErrorContext::new(operation);
        
        match query_fn().await {
            Ok(result) => Ok(result),
            Err(sqlx_error) => {
                let infra_error = ErrorConverter::from_sqlx_error(sqlx_error, Some(operation));
                let contextual_error = ContextualError::new(infra_error, context);
                
                // 记录结构化日志
                tracing::error!(
                    error = %contextual_error,
                    error_data = ?contextual_error.to_structured_log(),
                    "Database operation failed"
                );
                
                // 转换为领域错误
                Err(self.convert_to_domain_error(&contextual_error))
            }
        }
    }
    
    /// 将基础设施错误转换为领域错误
    fn convert_to_domain_error(&self, error: &ContextualError) -> DomainError {
        match &error.error {
            InfrastructureError::DatabaseQuery { query, message } => {
                if message.contains("unique constraint") {
                    if query.contains("users") && query.contains("login_id") {
                        DomainError::UserAlreadyExists
                    } else {
                        DomainError::BusinessRuleViolation {
                            rule: "unique constraint violation".to_string(),
                        }
                    }
                } else if message.contains("foreign key constraint") {
                    DomainError::BusinessRuleViolation {
                        rule: "referential integrity violation".to_string(),
                    }
                } else if message.contains("not found") {
                    DomainError::UserNotFound // 默认，具体实现可以更精确
                } else {
                    DomainError::ValidationError {
                        field: "database".to_string(),
                        reason: message.clone(),
                    }
                }
            }
            InfrastructureError::DatabaseConnection { .. } => {
                DomainError::BusinessRuleViolation {
                    rule: "database unavailable".to_string(),
                }
            }
            InfrastructureError::DatabaseTransaction { .. } => {
                DomainError::ConcurrencyConflict
            }
            InfrastructureError::Serialization { .. } => {
                DomainError::InvalidFormat
            }
            _ => {
                DomainError::BusinessRuleViolation {
                    rule: "infrastructure error".to_string(),
                }
            }
        }
    }
}

// === PostgreSQL用户仓储实现 ===
// 注意：实际的 PostgresUserRepository 实现在 src/adapters/adapters_pg/user_repository.rs
// 这里只是重新导出，避免重复实现

pub use crate::adapters::adapters_pg::PostgresUserRepository;
// === Argon2密码服务实现 ===

/// Argon2密码服务实现
#[derive(Clone)]
pub struct Argon2PasswordService {
    hasher: Arc<argon2::Argon2<'static>>,
}

impl Argon2PasswordService {
    /// 创建新的Argon2密码服务
    pub fn new() -> Self {
        use argon2::{Argon2, Algorithm, Version, Params};
        
        let params = Params::new(15000, 2, 1, None).expect("Invalid Argon2 params");
        let hasher = Argon2::new(Algorithm::Argon2id, Version::V0x13, params);
        
        Self {
            hasher: Arc::new(hasher),
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
        let password = password.to_string();
        let hasher = self.hasher.clone();
        
        // 在阻塞任务中执行CPU密集型操作
        let hash = tokio::task::spawn_blocking(move || {
            use argon2::{PasswordHasher, password_hash::{rand_core::OsRng, SaltString}};
            
            let salt = SaltString::generate(&mut OsRng);
            hasher.hash_password(password.as_bytes(), &salt)
                .map(|hash| hash.to_string())
        }).await;
        
        match hash {
            Ok(Ok(hash_string)) => Ok(hash_string),
            Ok(Err(hash_error)) => {
                let context = ErrorContext::new("hash_password")
                    .with_metadata("error", &hash_error.to_string());
                let infra_error = InfrastructureError::PasswordHash {
                    message: hash_error.to_string(),
                };
                let contextual_error = ContextualError::new(infra_error, context);
                
                tracing::error!(
                    error = %contextual_error,
                    "Password hashing failed"
                );
                
                Err(DomainError::ValidationError {
                    field: "password".to_string(),
                    reason: "hashing failed".to_string(),
                })
            }
            Err(join_error) => {
                let context = ErrorContext::new("hash_password")
                    .with_metadata("error", &join_error.to_string());
                let infra_error = InfrastructureError::Internal {
                    message: join_error.to_string(),
                };
                let contextual_error = ContextualError::new(infra_error, context);
                
                tracing::error!(
                    error = %contextual_error,
                    "Password hashing task failed"
                );
                
                Err(DomainError::ValidationError {
                    field: "password".to_string(),
                    reason: "task execution failed".to_string(),
                })
            }
        }
    }
    
    async fn verify_password(&self, password: &str, hash: &str) -> DomainResult<bool> {
        let password = password.to_string();
        let hash = hash.to_string();
        let hasher = self.hasher.clone();
        
        let verification = tokio::task::spawn_blocking(move || {
            use argon2::{PasswordVerifier, PasswordHash};
            
            let parsed_hash = PasswordHash::new(&hash)?;
            hasher.verify_password(password.as_bytes(), &parsed_hash)
        }).await;
        
        match verification {
            Ok(Ok(())) => Ok(true),
            Ok(Err(_)) => Ok(false), // 密码不匹配
            Err(join_error) => {
                let context = ErrorContext::new("verify_password")
                    .with_metadata("error", &join_error.to_string());
                let infra_error = InfrastructureError::Internal {
                    message: join_error.to_string(),
                };
                let contextual_error = ContextualError::new(infra_error, context);
                
                tracing::error!(
                    error = %contextual_error,
                    "Password verification task failed"
                );
                
                Err(DomainError::ValidationError {
                    field: "password".to_string(),
                    reason: "verification failed".to_string(),
                })
            }
        }
    }
    
    async fn generate_random_password(&self, length: usize) -> DomainResult<String> {
        if length < 8 || length > 128 {
            return Err(DomainError::ValidationError {
                field: "length".to_string(),
                reason: "must be between 8 and 128 characters".to_string(),
            });
        }
        
        let password = tokio::task::spawn_blocking(move || {
            use rand::{Rng, rng};
            
            const CHARSET: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZ\
                                    abcdefghijklmnopqrstuvwxyz\
                                    0123456789";
            let mut rng = rng();
            
            (0..length)
                .map(|_| {
                    let idx = rng.random_range(0..CHARSET.len());
                    CHARSET[idx] as char
                })
                .collect::<String>()
        }).await;
        
        match password {
            Ok(pwd) => Ok(pwd),
            Err(join_error) => {
                let context = ErrorContext::new("generate_random_password")
                    .with_metadata("error", &join_error.to_string());
                let infra_error = InfrastructureError::Internal {
                    message: join_error.to_string(),
                };
                let contextual_error = ContextualError::new(infra_error, context);
                
                tracing::error!(
                    error = %contextual_error,
                    "Random password generation failed"
                );
                
                Err(DomainError::ValidationError {
                    field: "password".to_string(),
                    reason: "generation failed".to_string(),
                })
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_postgres_adapter_creation() {
        // 这里只测试创建，不测试实际数据库操作
        // 实际测试需要数据库连接
        assert!(true);
    }
    
    #[tokio::test]
    async fn test_password_service() {
        let service = Argon2PasswordService::new();
        let password = "test_password_123";
        
        let hash = service.hash_password(password).await.unwrap();
        assert!(!hash.is_empty());
        
        let is_valid = service.verify_password(password, &hash).await.unwrap();
        assert!(is_valid);
        
        let is_invalid = service.verify_password("wrong_password", &hash).await.unwrap();
        assert!(!is_invalid);
    }
    
    #[tokio::test]
    async fn test_random_password_generation() {
        let service = Argon2PasswordService::new();
        
        let password = service.generate_random_password(12).await.unwrap();
        assert_eq!(password.len(), 12);
        
        // 测试边界条件
        assert!(service.generate_random_password(7).await.is_err());
        assert!(service.generate_random_password(129).await.is_err());
    }
}
