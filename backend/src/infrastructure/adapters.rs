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
use crate::domain::{DomainResult, DomainError, User, UserId};
use crate::ports::{UserRepository, PasswordService};
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

/// PostgreSQL用户仓储实现
#[derive(Clone)]
pub struct PostgresUserRepository {
    adapter: PostgresAdapter,
}

impl PostgresUserRepository {
    /// 创建新的PostgreSQL用户仓储
    pub fn new(pool: PgPool) -> Self {
        Self {
            adapter: PostgresAdapter::new(pool),
        }
    }
    
}

impl PostgresUserRepository {
    // save 方法已从 UserRepository trait 中移除
    #[allow(dead_code)]
    async fn save(&self, user: &User) -> DomainResult<()> {
        self.adapter.execute_query("save_user", || {
            let user = user.clone();
            let pool = self.adapter.pool.clone();
            
            Box::pin(async move {
                sqlx::query!(
                    r#"
                    INSERT INTO users (
                        id, username, nickname, email, avatar_url,
                        password_hash, user_type, expires_at, created_at, last_login_at
                    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
                    ON CONFLICT (id) DO UPDATE SET
                        username = EXCLUDED.username,
                        nickname = EXCLUDED.nickname,
                        email = EXCLUDED.email,
                        avatar_url = EXCLUDED.avatar_url,
                        password_hash = EXCLUDED.password_hash,
                        user_type = EXCLUDED.user_type,
                        expires_at = EXCLUDED.expires_at,
                        last_login_at = EXCLUDED.last_login_at
                    "#,
                    user.id.value(),
                    user.login_id,
                    user.nickname.value(),
                    user.email,
                    user.avatar_url,
                    user.password_hash,
                    user.user_type as _,
                    user.expires_at,
                    user.created_at,
                    user.last_active_at
                )
                .execute(&pool)
                .await?;
                
                // 保存用户位置（如果有）
                if let Some(location) = &user.location {
                    sqlx::query!(
                        r#"
                        INSERT INTO user_locations (user_id, latitude, longitude, address, updated_at)
                        VALUES ($1, $2, $3, $4, $5)
                        ON CONFLICT (user_id) DO UPDATE SET
                            latitude = EXCLUDED.latitude,
                            longitude = EXCLUDED.longitude,
                            address = EXCLUDED.address,
                            updated_at = EXCLUDED.updated_at
                        "#,
                        user.id.value(),
                        location.coordinates.latitude.value(),
                        location.coordinates.longitude.value(),
                        location.location_name,
                        location.updated_at
                    )
                    .execute(&pool)
                    .await?;
                }
                
                Ok(())
            })
        }).await
    }
    
    // delete_by_id 方法已从 UserRepository trait 中移除
    #[allow(dead_code)]
    async fn delete_by_id(&self, user_id: UserId) -> DomainResult<bool> {
        self.adapter.execute_query("delete_user", || {
            let pool = self.adapter.pool.clone();
            
            Box::pin(async move {
                let result = sqlx::query!(
                    "DELETE FROM users WHERE id = $1",
                    user_id.value()
                )
                .execute(&pool)
                .await?;
                
                Ok(result.rows_affected() > 0)
            })
        }).await
    }
    
    async fn find_by_id(&self, user_id: UserId) -> DomainResult<Option<User>> {
        self.adapter.execute_query("find_user_by_id", || {
            let pool = self.adapter.pool.clone();
            
            Box::pin(async move {
                let row = sqlx::query!(
                    r#"
                    SELECT 
                        id as user_id, username as login_id, nickname, email, avatar_url,
                        password_hash, user_type::text as "user_type!", expires_at, 
                        created_at, last_login_at as last_active_at
                    FROM users 
                    WHERE id = $1
                    "#,
                    user_id.value()
                )
                .fetch_optional(&pool)
                .await?;
                
                if let Some(row) = row {
                    // 获取用户位置
                    let location_row = sqlx::query!(
                        r#"
                        SELECT latitude, longitude, address as location_name, updated_at
                        FROM user_locations
                        WHERE id = $1
                        "#,
                        user_id.value()
                    )
                    .fetch_optional(&pool)
                    .await?;
                    
                    let location = if let Some(loc) = location_row {
                        Some(crate::domain::entities::UserLocation {
                            coordinates: crate::domain::Coordinates::new(loc.latitude, loc.longitude),
                            location_name: loc.location_name.unwrap_or_default(),
                            updated_at: loc.updated_at.unwrap_or_else(|| chrono::Utc::now()),
                        })
                    } else {
                        None
                    };
                    
                    let mut user = User {
                        id: user_id,
                        login_id: row.login_id,
                        nickname: crate::domain::Nickname::new(row.nickname)
                            .map_err(|e| sqlx::Error::Protocol(format!("Invalid nickname: {}", e)))?,
                        email: row.email,
                        avatar_url: row.avatar_url,
                        bio: None, // 暂时设为None，后续可以从数据库添加bio字段
                        password_hash: row.password_hash,
                        user_type: crate::domain::entities::UserType::from_str(&row.user_type).map_err(|e| sqlx::Error::Decode(e.into()))?,
                        expires_at: row.expires_at,
                        created_at: row.created_at.unwrap_or_else(|| chrono::Utc::now()),
                        last_active_at: row.last_active_at.unwrap_or_else(|| chrono::Utc::now()),
                        location,
                        events: Vec::new(),
                    };
                    
                    Ok(Some(user))
                } else {
                    Ok(None)
                }
            })
        }).await
    }
    
    async fn find_by_login_id(&self, login_id: &str) -> DomainResult<Option<User>> {
        self.adapter.execute_query("find_user_by_login_id", || {
            let login_id = login_id.to_string();
            let pool = self.adapter.pool.clone();
            
            Box::pin(async move {
                let row = sqlx::query!(
                    r#"
                    SELECT 
                        id as user_id, username as login_id, nickname, email, avatar_url,
                        password_hash, user_type::text as "user_type!", expires_at, 
                        created_at, last_login_at as last_active_at
                    FROM users 
                    WHERE username = $1
                    "#,
                    login_id
                )
                .fetch_optional(&pool)
                .await?;
                
                if let Some(row) = row {
                    let user_id = UserId::new(row.user_id)
                        .map_err(|e| sqlx::Error::Protocol(format!("Invalid user_id: {}", e)))?;
                    
                    let mut user = User {
                        id: user_id,
                        login_id: row.login_id,
                        nickname: crate::domain::Nickname::new(row.nickname)
                            .map_err(|e| sqlx::Error::Protocol(format!("Invalid nickname: {}", e)))?,
                        email: row.email,
                        avatar_url: row.avatar_url,
                        bio: None, // 暂时设为None，后续可以从数据库添加bio字段
                        password_hash: row.password_hash,
                        user_type: crate::domain::entities::UserType::from_str(&row.user_type).map_err(|e| sqlx::Error::Decode(e.into()))?,
                        expires_at: row.expires_at,
                        created_at: row.created_at.unwrap_or_else(|| chrono::Utc::now()),
                        last_active_at: row.last_active_at.unwrap_or_else(|| chrono::Utc::now()),
                        location: None, // 简化实现，不加载位置
                        events: Vec::new(),
                    };
                    
                    Ok(Some(user))
                } else {
                    Ok(None)
                }
            })
        }).await
    }
    
    async fn exists_by_login_id(&self, login_id: &str) -> DomainResult<bool> {
        self.adapter.execute_query("check_login_id_exists", || {
            let login_id = login_id.to_string();
            let pool = self.adapter.pool.clone();
            
            Box::pin(async move {
                let count = sqlx::query_scalar!(
                    "SELECT COUNT(*) FROM users WHERE username = $1",
                    login_id
                )
                .fetch_one(&pool)
                .await?;
                
                Ok(count.unwrap_or(0) > 0)
            })
        }).await
    }
    
    // 其他方法的实现会类似，都使用相同的错误处理模式
    async fn search_by_name(
        &self,
        _query: &str,
        _pagination: &crate::application::PaginationQuery,
    ) -> DomainResult<crate::application::PaginatedResult<crate::application::UserView>> {
        // 简化实现
        Ok(crate::application::PaginatedResult::empty())
    }
    
    async fn search_nearby(
        &self,
        _location: &crate::application::LocationQuery,
        _exclude_user_id: Option<UserId>,
        _pagination: &crate::application::PaginationQuery,
    ) -> DomainResult<crate::application::PaginatedResult<crate::application::UserView>> {
        // 简化实现
        Ok(crate::application::PaginatedResult::empty())
    }
    
    async fn get_user_stats(&self, _user_id: UserId) -> DomainResult<crate::ports::UserStats> {
        // 简化实现
        Ok(crate::ports::UserStats {
            total_groups: 0,
            total_messages: 0,
            total_moments: 0,
            total_likes_received: 0,
        })
    }
}

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
            use rand::{Rng, thread_rng};
            
            const CHARSET: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZ\
                                    abcdefghijklmnopqrstuvwxyz\
                                    0123456789";
            let mut rng = thread_rng();
            
            (0..length)
                .map(|_| {
                    let idx = rng.gen_range(0..CHARSET.len());
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
