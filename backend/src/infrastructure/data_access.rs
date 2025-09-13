/// 数据访问层 - 完全基于sqlx宏和极简Option使用
/// 
/// 设计原则：
/// - 强制使用sqlx::query!、query_as!、query_scalar!宏
/// - Option字段极简化：只有数据库字段确实可能为NULL时才使用Option<T>
/// - 强制sqlx非NULL标记：对于确保非NULL的字段使用"field_name!"语法
/// - 分层数据结构：Row结构体 → Domain模型转换
/// - 查询优化：只查询必要字段，避免SELECT *

use async_trait::async_trait;
use sqlx::PgPool;
use chrono::{DateTime, Utc};
use crate::domain::{DomainResult, DomainError, UserId, User};
use crate::ports::UserRepository;
use crate::infrastructure::errors::{ErrorContext, ContextualError, ErrorConverter};

// === 数据库行结构体（Repository层专用）===

/// 用户基础信息行结构 - 对应users表的核心字段
/// 
/// Option字段极简原则：
/// - user_id, login_id, nickname: NOT NULL字段，使用非Option类型
/// - email, avatar_url: 可为NULL字段，使用Option类型
/// - expires_at: 可为NULL字段（正式用户为NULL，临时用户有值）
#[derive(Debug, sqlx::FromRow)]
struct UserRow {
    /// 用户ID - NOT NULL主键
    user_id: i64,
    /// 登录ID - NOT NULL唯一标识
    login_id: String,
    /// 昵称 - NOT NULL，有默认值
    nickname: String,
    /// 邮箱 - 可为NULL
    email: Option<String>,
    /// 头像URL - 可为NULL
    avatar_url: Option<String>,
    /// 密码哈希 - 可为NULL（临时用户可能没有密码）
    password_hash: Option<String>,
    /// 用户类型 - NOT NULL枚举
    user_type: crate::domain::entities::UserType,
    /// 过期时间 - 可为NULL（正式用户为NULL）
    expires_at: Option<DateTime<Utc>>,
    /// 创建时间 - NOT NULL，有默认值
    created_at: DateTime<Utc>,
    /// 最后活跃时间 - 可为NULL
    last_active_at: Option<DateTime<Utc>>,
}

/// 用户位置信息行结构 - 对应user_locations表
#[derive(Debug, sqlx::FromRow)]
struct UserLocationRow {
    /// 用户ID - NOT NULL外键
    user_id: i64,
    /// 纬度 - NOT NULL，有默认值
    latitude: f64,
    /// 经度 - NOT NULL，有默认值
    longitude: f64,
    /// 位置名称 - NOT NULL，有默认值
    location_name: String,
    /// 更新时间 - NOT NULL，有默认值
    updated_at: DateTime<Utc>,
}

/// 用户统计信息行结构 - 聚合查询结果
#[derive(Debug, sqlx::FromRow)]
struct UserStatsRow {
    /// 用户ID - NOT NULL
    user_id: i64,
    /// 群组数量 - COUNT结果，必定非NULL
    group_count: i64,
    /// 消息数量 - COUNT结果，必定非NULL
    message_count: i64,
    /// 动态数量 - COUNT结果，必定非NULL
    moment_count: i64,
    /// 收到的点赞数 - COUNT结果，必定非NULL
    likes_received: i64,
}

/// 用户搜索结果行结构 - 带距离信息的搜索结果
#[derive(Debug, sqlx::FromRow)]
struct UserSearchRow {
    /// 用户ID - NOT NULL
    user_id: i64,
    /// 昵称 - NOT NULL
    nickname: String,
    /// 头像URL - 可为NULL
    avatar_url: Option<String>,
    /// 最后活跃时间 - 可为NULL
    last_active_at: Option<DateTime<Utc>>,
    /// 用户类型 - NOT NULL
    user_type: crate::domain::entities::UserType,
    /// 距离（米）- 地理搜索时非NULL，名称搜索时为NULL
    distance_meters: Option<f64>,
}

// === Row到Domain模型转换 ===

impl UserRow {
    /// 转换为领域用户模型
    fn to_domain(self, location: Option<crate::domain::entities::UserLocation>) -> Result<User, sqlx::Error> {
        Ok(User {
            id: UserId::new(self.user_id).map_err(|e| sqlx::Error::Protocol(format!("Invalid user_id: {}", e)))?,
            login_id: Some(self.login_id),
            nickname: crate::domain::Nickname::new(self.nickname).map_err(|e| sqlx::Error::Protocol(format!("Invalid nickname: {}", e)))?,
            email: self.email,
            avatar_url: self.avatar_url,
            bio: None, // 当前数据库表没有bio字段
            password_hash: self.password_hash,
            user_type: self.user_type,
            expires_at: self.expires_at,
            created_at: self.created_at,
            last_active_at: self.last_active_at.unwrap_or_else(|| chrono::Utc::now()),
            location,
            events: Vec::new(), // 新创建的用户没有待处理事件
        })
    }
}

impl UserLocationRow {
    /// 转换为领域位置模型
    fn to_domain(self) -> crate::domain::entities::UserLocation {
        crate::domain::entities::UserLocation {
            coordinates: crate::domain::Coordinates::new(self.latitude, self.longitude)
                .expect("数据库中的坐标应该是有效的"),
            location_name: self.location_name,
            updated_at: self.updated_at,
        }
    }
}

impl UserStatsRow {
    /// 转换为用户统计信息
    fn to_domain(self) -> crate::ports::UserStats {
        crate::ports::UserStats {
            total_groups: self.group_count as u32,
            total_messages: self.message_count as u32,
            total_moments: self.moment_count as u32,
            total_likes_received: self.likes_received as u32,
        }
    }
}

impl UserSearchRow {
    /// 转换为用户搜索结果
    fn to_domain(self) -> crate::application::UserView {
        crate::application::UserView {
            user_id: self.user_id.to_string(),
            login_id: None, // 搜索结果不包含敏感信息
            nickname: self.nickname,
            avatar_url: self.avatar_url,
            email: None, // 搜索结果不包含敏感信息
            bio: None,
            is_temp: false, // 默认值
            created_at: chrono::Utc::now().to_rfc3339(),
            last_active_at: self.last_active_at.unwrap_or_else(|| chrono::Utc::now()).to_rfc3339(),
            latitude: None,
            longitude: None,
            location_name: None,
            distance: self.distance_meters,
        }
    }
}

// === PostgreSQL数据访问实现 ===

/// PostgreSQL用户仓储实现 - 完全基于sqlx宏
#[derive(Clone)]
pub struct PgUserRepository {
    pool: PgPool,
}

impl PgUserRepository {
    /// 创建新的PostgreSQL用户仓储
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
    
    /// 执行查询并处理错误 - 统一的错误处理模式
    async fn execute_query<F, T>(&self, operation: &str, query_fn: F) -> DomainResult<T>
    where
        F: std::future::Future<Output = Result<T, sqlx::Error>>,
    {
        let context = ErrorContext::new(operation);
        
        match query_fn.await {
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
            crate::infrastructure::errors::InfrastructureError::DatabaseQuery { message, .. } => {
                if message.contains("unique constraint") {
                    if message.contains("login_id") {
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
                    DomainError::UserNotFound
                } else {
                    DomainError::ValidationError {
                        field: "database".to_string(),
                        reason: message.clone(),
                    }
                }
            }
            crate::infrastructure::errors::InfrastructureError::DatabaseConnection { .. } => {
                DomainError::BusinessRuleViolation {
                    rule: "database unavailable".to_string(),
                }
            }
            crate::infrastructure::errors::InfrastructureError::DatabaseTransaction { .. } => {
                DomainError::ConcurrencyConflict
            }
            _ => {
                DomainError::BusinessRuleViolation {
                    rule: "infrastructure error".to_string(),
                }
            }
        }
    }
}

impl PgUserRepository {
    /// 保存用户 - 使用UPSERT模式
    async fn save(&self, user: &User) -> DomainResult<()> {
        self.execute_query("save_user", async {
            // 保存用户基础信息
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
            .execute(&self.pool)
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
                .execute(&self.pool)
                .await?;
            }
            
            Ok(())
        }).await
    }
    
    /// 根据ID删除用户
    async fn delete_by_id(&self, user_id: UserId) -> DomainResult<bool> {
        self.execute_query("delete_user_by_id", async {
            let result = sqlx::query!(
                "DELETE FROM users WHERE id = $1",
                user_id.value()
            )
            .execute(&self.pool)
            .await?;
            
            Ok(result.rows_affected() > 0)
        }).await
    }
    
    /// 根据ID查找用户 - 使用sqlx!宏和非NULL标记
    async fn find_by_id(&self, user_id: UserId) -> DomainResult<Option<User>> {
        self.execute_query("find_user_by_id", async {
            // 查询用户基础信息 - 使用"!"标记确保非NULL字段
            let user_row = sqlx::query_as!(
                UserRow,
                r#"
                SELECT 
                    id as "user_id!",
                    username as "login_id!",
                    nickname as "nickname!",
                    email,
                    avatar_url,
                    password_hash,
                    user_type as "user_type!: crate::domain::entities::UserType",
                    expires_at,
                    created_at as "created_at!",
                    last_login_at as "last_active_at"
                FROM users 
                WHERE id = $1
                "#,
                user_id.value()
            )
            .fetch_optional(&self.pool)
            .await?;
            
            if let Some(user_row) = user_row {
                // 查询用户位置信息
                let location_row = sqlx::query_as!(
                    UserLocationRow,
                    r#"
                    SELECT 
                        id as "user_id!",
                        latitude as "latitude!",
                        longitude as "longitude!",
                        address as "location_name!",
                        updated_at as "updated_at!"
                    FROM user_locations
                    WHERE id = $1
                    "#,
                    user_id.value()
                )
                .fetch_optional(&self.pool)
                .await?;
                
                let location = location_row.map(|row| row.to_domain());
                let user = user_row.to_domain(location)?;
                
                Ok(Some(user))
            } else {
                Ok(None)
            }
        }).await
    }
    
    /// 根据登录ID查找用户
    async fn find_by_login_id(&self, login_id: &str) -> DomainResult<Option<User>> {
        self.execute_query("find_user_by_login_id", async {
            let user_row = sqlx::query_as!(
                UserRow,
                r#"
                SELECT 
                    id as "user_id!",
                    username as "login_id!",
                    nickname as "nickname!",
                    email,
                    avatar_url,
                    password_hash,
                    user_type as "user_type!: crate::domain::entities::UserType",
                    expires_at,
                    created_at as "created_at!",
                    last_login_at as "last_active_at"
                FROM users 
                WHERE username = $1
                "#,
                login_id
            )
            .fetch_optional(&self.pool)
            .await?;
            
            if let Some(user_row) = user_row {
                // 简化实现：不加载位置信息
                let user = user_row.to_domain(None)?;
                Ok(Some(user))
            } else {
                Ok(None)
            }
        }).await
    }
    
    /// 检查登录ID是否存在 - 使用query_scalar!宏
    async fn exists_by_login_id(&self, login_id: &str) -> DomainResult<bool> {
        self.execute_query("check_login_id_exists", async {
            let count = sqlx::query_scalar!(
                "SELECT COUNT(*) as count FROM users WHERE username = $1",
                login_id
            )
            .fetch_one(&self.pool)
            .await?;
            
            // COUNT(*)结果必定非NULL，但sqlx可能返回Option，使用unwrap_or处理
            Ok(count.unwrap_or(0) > 0)
        }).await
    }
    
    /// 按名称搜索用户 - 分页查询
    async fn search_by_name(
        &self,
        query: &str,
        pagination: &crate::application::PaginationQuery,
    ) -> DomainResult<crate::application::PaginatedResult<crate::application::UserView>> {
        self.execute_query("search_users_by_name", async {
            let limit = pagination.get_limit(); // 使用统一的方法
            let cursor = pagination.get_cursor();
            
            // 使用ILIKE进行模糊搜索，按user_id排序实现游标分页
            let rows = sqlx::query_as!(
                UserSearchRow,
                r#"
                SELECT 
                    id as "user_id!",
                    nickname as "nickname!",
                    avatar_url,
                    last_login_at as "last_active_at",
                    user_type as "user_type!: crate::domain::entities::UserType",
                    NULL::FLOAT8 as distance_meters
                FROM users 
                WHERE nickname ILIKE $1 
                  AND ($2 = 0 OR id > $2)
                ORDER BY id ASC
                LIMIT $3
                "#,
                format!("%{}%", query),
                cursor as i32,
                limit + 1 // 多查询一条判断是否有更多数据
            )
            .fetch_all(&self.pool)
            .await?;
            
            let has_more = rows.len() > limit as usize;
            let mut items: Vec<_> = rows.into_iter()
                .take(limit as usize)
                .map(|row| row.to_domain())
                .collect();
            
            let next_cursor = if has_more && !items.is_empty() {
                Some(items.last().unwrap().user_id.clone())
            } else {
                None
            };
            
            Ok(crate::application::PaginatedResult {
                items,
                has_more,
                next_cursor: next_cursor.map(|c| c.to_string()),
            })
        }).await
    }
    
    /// 搜索附近用户 - 地理位置查询
    async fn search_nearby(
        &self,
        location: &crate::application::LocationQuery,
        exclude_user_id: Option<UserId>,
        pagination: &crate::application::PaginationQuery,
    ) -> DomainResult<crate::application::PaginatedResult<crate::application::UserView>> {
        self.execute_query("search_nearby_users", async {
            let limit = pagination.limit.unwrap_or(20).min(100) as i64;
            let cursor = pagination.cursor.unwrap_or(0);
            let exclude_id = exclude_user_id.map(|id| id.value()).unwrap_or(-1);
            
            // 使用PostGIS进行地理距离查询
            let rows = sqlx::query_as!(
                UserSearchRow,
                r#"
                SELECT 
                    u.id as "user_id!",
                    u.nickname as "nickname!",
                    u.avatar_url,
                    u.last_login_at as "last_active_at!",
                    u.user_type as "user_type!: crate::domain::entities::UserType",
                    ST_Distance(
                        ul.location,
                        ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
                    ) as distance_meters
                FROM users u
                INNER JOIN user_locations ul ON u.id = ul.user_id
                WHERE u.id != $3
                  AND ($4 = 0 OR u.id > $4)
                  AND ST_DWithin(
                      ul.location,
                      ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                      $5
                  )
                ORDER BY distance_meters ASC, u.id ASC
                LIMIT $6
                "#,
                location.center.longitude.value(),
                location.center.latitude.value(),
                exclude_id,
                cursor as i32,
                location.radius * 1000.0, // 转换为米
                limit + 1
            )
            .fetch_all(&self.pool)
            .await?;
            
            let has_more = rows.len() > limit as usize;
            let mut items: Vec<_> = rows.into_iter()
                .take(limit as usize)
                .map(|row| row.to_domain())
                .collect();
            
            let next_cursor = if has_more && !items.is_empty() {
                Some(items.last().unwrap().user_id.clone())
            } else {
                None
            };
            
            Ok(crate::application::PaginatedResult {
                items,
                has_more,
                next_cursor: next_cursor.map(|c| c.to_string()),
            })
        }).await
    }
    
    /// 获取用户统计信息 - 聚合查询
    async fn get_user_stats(&self, user_id: UserId) -> DomainResult<crate::ports::UserStats> {
        self.execute_query("get_user_stats", async {
            let user_id_val = user_id.value();
            let stats = sqlx::query_as!(
                UserStatsRow,
                r#"
                SELECT 
                    $1::BIGINT as "user_id!",
                    COALESCE(
                        (SELECT COUNT(*) FROM group_members WHERE user_id = $1), 
                        0
                    ) as "group_count!",
                    COALESCE(
                        (SELECT COUNT(*) FROM messages WHERE sender_id = $1), 
                        0
                    ) as "message_count!",
                    COALESCE(
                        (SELECT COUNT(*) FROM moments WHERE user_id = $1), 
                        0
                    ) as "moment_count!",
                    0 as "likes_received!"
                "#,
                user_id_val
            )
            .fetch_one(&self.pool)
            .await?;
            
            Ok(stats.to_domain())
        }).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_user_row_conversion() {
        let user_row = UserRow {
            user_id: 123,
            login_id: "test_user".to_string(),
            nickname: "测试用户".to_string(),
            email: Some("test@example.com".to_string()),
            avatar_url: None,
            password_hash: Some("hashed_password".to_string()),
            user_type: crate::domain::entities::UserType::Regular,
            expires_at: None,
            created_at: Utc::now(),
            last_active_at: Some(Utc::now()),
        };
        
        let user = user_row.to_domain(None).unwrap();
        assert_eq!(user.id.value(), 123);
        assert_eq!(user.login_id, Some("test_user".to_string()));
        assert_eq!(user.nickname.value(), "测试用户");
        assert_eq!(user.email, Some("test@example.com".to_string()));
        assert_eq!(user.user_type, crate::domain::entities::UserType::Regular);
    }
    
    #[test]
    fn test_user_stats_conversion() {
        let stats_row = UserStatsRow {
            user_id: 123,
            group_count: 5,
            message_count: 100,
            moment_count: 20,
            likes_received: 50,
        };
        
        let stats = stats_row.to_domain();
        assert_eq!(stats.total_groups, 5);
        assert_eq!(stats.total_messages, 100);
        assert_eq!(stats.total_moments, 20);
        assert_eq!(stats.total_likes_received, 50);
    }
}

