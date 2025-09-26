/// 用户数据访问层 - 三层数据结构实现
/// 
/// 架构说明：
/// 1. Row结构体：仅在此文件内使用，对应数据库字段
/// 2. Domain模型：返回给上层的业务模型
/// 3. 查询优化：只查询必要字段，使用sqlx!宏和非NULL标记
use sqlx::PgPool;
use chrono::{DateTime, Utc};
use crate::domain::{
    DomainError, DomainResult, UserProfile, UserAuth, UserDetail, UserSearchResult
};
use crate::domain::entities::UserType;
use crate::ports::UserRepository;
use super::query_optimizer::PgQueryOptimizer;

// ===== 数据库行结构体 (Repository层专用) =====

/// 用户资料查询的轻量级数据库行结构
#[derive(Debug, sqlx::FromRow)]
struct UserProfileRow {
    user_id: i64,  // 修改为i64类型，与数据库表结构一致
    nickname: String,
    avatar_url: Option<String>,
    last_active_at: Option<DateTime<Utc>>,
    // expires_at字段在实际数据库表中不存在，临时用户通过其他方式判断
}


/// 用户认证查询的数据库行结构（仅用于认证场景）
#[derive(Debug, sqlx::FromRow)]
struct UserAuthRow {
    user_id: i64,
    login_id: Option<String>,
    nickname: String,
    password_hash: Option<String>,
    avatar_url: Option<String>,
}

/// 用户详细信息查询的数据库行结构
#[derive(Debug, sqlx::FromRow)]
struct UserDetailRow {
    user_id: i64,
    login_id: Option<String>,
    nickname: String,
    email: Option<String>,
    avatar_url: Option<String>,
    created_at: DateTime<Utc>,
    last_active_at: Option<DateTime<Utc>>,
}

/// 用户搜索结果的数据库行结构
#[derive(Debug, sqlx::FromRow)]
struct UserSearchRow {
    user_id: i64,
    nickname: String,
    avatar_url: Option<String>,
    last_active_at: Option<DateTime<Utc>>,  // 修改为Option类型
    distance: Option<f64>, // 仅地理位置搜索时有值
}

// ===== Row到Domain的转换实现 =====

impl UserProfileRow {
    fn to_domain(self) -> UserProfile {
        // 暂时将所有用户设为Regular类型，后续可通过其他字段判断
        let user_type = UserType::Regular;

        UserProfile::new(
            self.user_id,
            self.nickname,
            self.avatar_url,
            user_type,
            self.last_active_at.unwrap_or_else(|| chrono::Utc::now()),  // 提供默认值
        )
    }
}

impl UserAuthRow {
    fn to_domain(self) -> UserAuth {
        // 暂时将所有用户设为Regular类型，后续可通过其他字段判断
        let user_type = UserType::Regular;

        UserAuth::new(
            self.user_id,
            self.login_id,
            self.nickname,
            self.password_hash,
            user_type,
            self.avatar_url,
        )
    }
}

impl UserDetailRow {
    fn to_domain(self) -> UserDetail {
        // 暂时将所有用户设为Regular类型，后续可通过其他字段判断
        let user_type = UserType::Regular;

        UserDetail::new(
            self.user_id,
            self.login_id,
            self.nickname,
            self.email,
            self.avatar_url,
            user_type,
            self.last_active_at.unwrap_or_else(|| chrono::Utc::now()),
            self.created_at,
            None, // expires_at - 临时用户功能暂未实现
        )
    }
}

impl UserSearchRow {
    fn to_domain(self, has_location: bool) -> UserSearchResult {
        // 暂时将所有用户设为Regular类型，后续可通过其他字段判断
        let user_type = UserType::Regular;

        UserSearchResult::new(
            self.user_id,
            self.nickname,
            self.avatar_url,
            user_type,
            self.last_active_at.unwrap_or_else(|| chrono::Utc::now()),  // 提供默认值
            self.distance,
            has_location,
        )
    }
}

// ===== Repository实现 =====

#[derive(Clone)]
pub struct PostgresUserRepository {
    pool: PgPool,
    query_optimizer: PgQueryOptimizer,
}

impl PostgresUserRepository {
    pub fn new(pool: PgPool) -> Self {
        // 强制启用查询优化器
        let query_optimizer = PgQueryOptimizer::new(pool.clone());
        Self {
            pool,
            query_optimizer,
        }
    }
    
    /// 执行查询并进行性能监控
    async fn execute_query_with_monitoring<F, T>(
        &self,
        operation_name: &str,
        query_fn: F,
    ) -> DomainResult<T>
    where
        F: std::future::Future<Output = DomainResult<T>>,
    {
        // 强制使用查询优化器进行监控和优化
        let params_summary = std::collections::HashMap::new(); // 可以根据需要添加参数信息
        
        let (result, metrics) = self.query_optimizer.execute_with_monitoring(
            operation_name,
            params_summary,
            query_fn
        ).await?;
        
        // 记录位置查询监控（如果是位置相关查询）
        if operation_name.contains("location") || operation_name.contains("nearby") {
            let metrics_collector = crate::utils::metrics::get_app_metrics();
            if let Some(ref perf_metrics) = metrics {
                metrics_collector.record_location_query(perf_metrics.execution_time_ms as f64 / 1000.0);
            }
        }
        
        // 记录性能指标
        if let Some(ref perf_metrics) = metrics {
            if perf_metrics.execution_time_ms > 100 {
                tracing::warn!(
                    operation = operation_name,
                    duration_ms = perf_metrics.execution_time_ms,
                    query_type = perf_metrics.query_type,
                    used_index = perf_metrics.used_index,
                    repository = "user_repository",
                    "慢查询检测"
                );
            } else {
                tracing::debug!(
                    operation = operation_name,
                    duration_ms = perf_metrics.execution_time_ms,
                    query_type = perf_metrics.query_type,
                    used_index = perf_metrics.used_index,
                    repository = "user_repository",
                    "查询完成"
                );
            }
        }
        
        Ok(result)
    }

    /// 🔒 专门的密码验证方法：只在需要验证密码时查询密码哈希
    pub async fn find_user_password_hash(&self, user_id: i64) -> DomainResult<Option<String>> {
        let password_hash = sqlx::query_scalar!(
            r#"
            SELECT password_hash
            FROM users
            WHERE id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(password_hash.flatten())
    }

    /// 检查用户位置是否存在
    async fn check_user_location(&self, user_id: i64) -> DomainResult<bool> {
        let exists = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM users 
                WHERE id = $1 AND latitude IS NOT NULL AND longitude IS NOT NULL
            ) as "exists!"
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(exists)
    }
}

#[async_trait::async_trait]
impl UserRepository for PostgresUserRepository {
    /// 获取用户基础资料（轻量级）
    async fn find_user_profile(&self, user_id: i64) -> DomainResult<Option<UserProfile>> {
        self.execute_query_with_monitoring("find_user_profile", async {
            let user_row = sqlx::query_as!(
                UserProfileRow,
                r#"
                SELECT id as "user_id!", nickname as "nickname!", 
                       avatar_url, updated_at as last_active_at
                FROM users
                WHERE id = $1
                "#,
                user_id
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;

            Ok(user_row.map(|row| row.to_domain()))
        }).await
    }

    /// 获取用户认证信息（仅用于认证）
    async fn find_user_auth(&self, login_id: &str) -> DomainResult<Option<UserAuth>> {
        self.execute_query_with_monitoring("find_user_auth", async {
            let user_row = sqlx::query_as!(
                UserAuthRow,
                r#"
                SELECT id as "user_id!", login_id, nickname as "nickname!", 
                       password_hash, avatar_url
                FROM users
                WHERE login_id = $1
                "#,
                login_id
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;

            Ok(user_row.map(|row| row.to_domain()))
        }).await
    }

    /// 按昵称搜索用户资料（轻量级）
    async fn search_user_profiles_by_name(
        &self,
        keyword: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<UserSearchResult>> {
        let search_pattern = format!("%{}%", keyword);

        let user_rows = sqlx::query_as!(
            UserSearchRow,
            r#"
            SELECT id as "user_id!", nickname as "nickname!", 
                   avatar_url, updated_at as last_active_at,
                   NULL::DOUBLE PRECISION as distance
            FROM users
            WHERE nickname ILIKE $1
              AND ($2 = 0 OR id < $2)
            ORDER BY id DESC
            LIMIT $3
            "#,
            search_pattern,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        let mut results = Vec::new();
        for user_row in user_rows {
            let has_location = self.check_user_location(user_row.user_id).await?;
            results.push(user_row.to_domain(has_location));
        }

        Ok(results)
    }

    /// 按位置搜索附近用户
    async fn find_nearby_users_with_distance(
        &self,
        latitude: f64,
        longitude: f64,
        radius_meters: f64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<UserSearchResult>> {
        let user_rows = sqlx::query_as!(
            UserSearchRow,
            r#"
            SELECT 
                u.id as "user_id!",
                u.nickname as "nickname!",
                u.avatar_url,
                u.updated_at as last_active_at,
                ST_Distance(u.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) as distance
            FROM users u
            WHERE u.latitude IS NOT NULL AND u.longitude IS NOT NULL
              AND ST_DWithin(u.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
              AND ($4 = 0 OR u.id < $4)
            ORDER BY distance ASC, u.id DESC
            LIMIT $5
            "#,
            longitude,
            latitude,
            radius_meters,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(user_rows.into_iter().map(|row| row.to_domain(true)).collect())
    }

    /// 获取用户详细信息
    async fn find_user_detail(&self, user_id: i64) -> DomainResult<Option<UserDetail>> {
        self.execute_query_with_monitoring("find_user_detail", async {
            let user_row = sqlx::query_as!(
                UserDetailRow,
                r#"
                SELECT id as "user_id!", login_id, nickname as "nickname!", 
                       NULL::TEXT as email, avatar_url, 
                       created_at as "created_at!", 
                       updated_at as last_active_at
                FROM users
                WHERE id = $1
                "#,
                user_id
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;

            Ok(user_row.map(|row| row.to_domain()))
        }).await
    }

    /// 创建新用户
    async fn create_user(
        &self,
        login_id: Option<&str>,
        nickname: &str,
        password_hash: Option<&str>,
        _email: Option<&str>,
        _expires_at: Option<chrono::DateTime<chrono::Utc>>,
    ) -> DomainResult<UserProfile> {
        self.execute_query_with_monitoring("create_user", async {
            let user_row = sqlx::query_as!(
                UserProfileRow,
                r#"
                INSERT INTO users (login_id, nickname, password_hash)
                VALUES ($1, $2, $3)
                RETURNING id as "user_id!", nickname as "nickname!", 
                          avatar_url, updated_at as last_active_at
                "#,
                login_id,
                nickname,
                password_hash
            )
            .fetch_one(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;

            Ok(user_row.to_domain())
        }).await
    }

    /// 更新用户资料
    async fn update_user_profile(
        &self,
        user_id: i64,
        nickname: Option<&str>,
        avatar_url: Option<&str>,
    ) -> DomainResult<UserProfile> {
        let user_row = sqlx::query_as!(
            UserProfileRow,
            r#"
            UPDATE users 
            SET nickname = COALESCE($2, nickname),
                avatar_url = COALESCE($3, avatar_url),
                updated_at = NOW()
            WHERE id = $1
            RETURNING id as "user_id!", nickname as "nickname!", 
                      avatar_url, updated_at as last_active_at
            "#,
            user_id,
            nickname,
            avatar_url
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(user_row.to_domain())
    }

    /// 设置用户密码
    async fn set_user_password(&self, user_id: i64, password_hash: &str) -> DomainResult<()> {
        sqlx::query!(
            r#"
            UPDATE users 
            SET password_hash = $2, updated_at = NOW()
            WHERE id = $1
            "#,
            user_id,
            password_hash
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(())
    }

    /// 升级临时用户为正式用户
    async fn upgrade_temp_user(
        &self,
        user_id: i64,
        login_id: &str,
        password_hash: &str,
    ) -> DomainResult<UserProfile> {
        let user_row = sqlx::query_as!(
            UserProfileRow,
            r#"
            UPDATE users 
            SET login_id = $2, 
                password_hash = $3, 
                updated_at = NOW()
            WHERE id = $1
            RETURNING id as "user_id!", nickname as "nickname!", 
                      avatar_url, updated_at as last_active_at
            "#,
            user_id,
            login_id,
            password_hash
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(user_row.to_domain())
    }

    /// 删除用户
    async fn delete_user(&self, user_id: i64) -> DomainResult<()> {
        sqlx::query!(
            r#"
            DELETE FROM users WHERE id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(())
    }

    /// 更新用户位置
    async fn update_user_location(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: &str,
    ) -> DomainResult<()> {
        self.execute_query_with_monitoring("update_user_location", async {
            sqlx::query!(
                r#"
                UPDATE users SET 
                    latitude = $2,
                    longitude = $3,
                    location_name = $4,
                    location_updated_at = NOW(),
                    location = ST_SetSRID(ST_MakePoint($3, $2), 4326)
                WHERE id = $1
                "#,
                user_id,
                latitude,
                longitude,
                location_name
            )
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;

            Ok(())
        }).await
    }

    /// 查找用户位置
    async fn find_user_location(&self, user_id: i64) -> DomainResult<Option<(f64, f64, String)>> {
        self.execute_query_with_monitoring("find_user_location", async {
            let location = sqlx::query!(
                r#"
                SELECT latitude as "latitude!", longitude as "longitude!", 
                       COALESCE(location_name, '') as "location_name!"
                FROM users
                WHERE id = $1 AND latitude IS NOT NULL AND longitude IS NOT NULL
                "#,
                user_id
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;

            Ok(location.map(|loc| (loc.latitude, loc.longitude, loc.location_name)))
        }).await
    }

    /// 获取用户总数
    async fn get_total_user_count(&self) -> DomainResult<i64> {
        let count = sqlx::query_scalar!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM users
            "#
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(count)
    }

    /// 按ID范围获取用户数量
    async fn get_user_count_by_id_range(&self, start_id: i64, end_id: i64) -> DomainResult<i64> {
        let count = sqlx::query_scalar!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM users
            WHERE id >= $1 AND id <= $2
            "#,
            start_id,
            end_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(count)
    }

    /// 获取指定范围内的最大用户ID
    async fn get_max_user_id_by_range(&self, start_id: i64, end_id: i64) -> DomainResult<Option<i64>> {
        let max_id = sqlx::query_scalar!(
            r#"
            SELECT MAX(id)
            FROM users
            WHERE id >= $1 AND id <= $2
            "#,
            start_id,
            end_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(max_id)
    }
}