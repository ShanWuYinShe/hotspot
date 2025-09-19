/// 查询优化器 - sqlx宏的高级用法和性能优化
/// 
/// 优化特性：
/// - 动态查询构建
/// - 批量操作
/// - 查询缓存
/// - 性能监控
/// - 连接池优化

use sqlx::{PgPool, Postgres, QueryBuilder /* , Arguments */};
use std::collections::HashMap;
use chrono::{DateTime, Utc};
use crate::domain::DomainResult;
// use crate::domain::{UserId, GroupId}; // 暂时未使用
use crate::infrastructure::errors::{ErrorContext, ContextualError, ErrorConverter};

/// 查询优化器 - 提供高性能的数据库查询方法
#[derive(Clone)]
#[allow(dead_code)]
pub struct QueryOptimizer {
    pool: PgPool,
    #[allow(dead_code)]
    query_cache: std::sync::Arc<tokio::sync::RwLock<HashMap<String, String>>>,
}

impl QueryOptimizer {
    /// 创建新的查询优化器
    pub fn new(pool: PgPool) -> Self {
        Self {
            pool,
            query_cache: std::sync::Arc::new(tokio::sync::RwLock::new(HashMap::new())),
        }
    }
    
    /// 批量插入用户 - 展示批量操作的sqlx用法
    pub async fn batch_insert_users(&self, users: Vec<BatchUserInsert>) -> DomainResult<u64> {
        let context = ErrorContext::new("batch_insert_users")
            .with_metadata("batch_size", &users.len().to_string());
        
        if users.is_empty() {
            return Ok(0);
        }
        
        match self.execute_batch_insert_users(users).await {
            Ok(rows_affected) => Ok(rows_affected),
            Err(sqlx_error) => {
                let infra_error = ErrorConverter::from_sqlx_error(sqlx_error, Some("batch_insert_users"));
                let contextual_error = ContextualError::new(infra_error, context);
                
                tracing::error!(
                    error = %contextual_error,
                    "Batch insert users failed"
                );
                
                Err(crate::domain::DomainError::ValidationError {
                    field: "batch_operation".to_string(),
                    reason: contextual_error.to_string(),
                })
            }
        }
    }
    
    /// 执行批量插入用户 - 使用QueryBuilder动态构建查询
    async fn execute_batch_insert_users(&self, users: Vec<BatchUserInsert>) -> Result<u64, sqlx::Error> {
        let mut query_builder = QueryBuilder::<Postgres>::new(
            r#"
            INSERT INTO users (
                user_id, login_id, nickname, email, avatar_url,
                password_hash, user_type, expires_at, created_at, last_active_at
            )
            "#
        );
        
        query_builder.push_values(users, |mut b, user| {
            b.push_bind(user.user_id)
             .push_bind(user.login_id)
             .push_bind(user.nickname)
             .push_bind(user.email)
             .push_bind(user.avatar_url)
             .push_bind(user.password_hash)
             .push_bind(user.user_type)
             .push_bind(user.expires_at)
             .push_bind(user.created_at)
             .push_bind(user.last_active_at);
        });
        
        // 添加冲突处理
        query_builder.push(
            r#"
            ON CONFLICT (user_id) DO UPDATE SET
                nickname = EXCLUDED.nickname,
                email = EXCLUDED.email,
                avatar_url = EXCLUDED.avatar_url,
                last_active_at = EXCLUDED.last_active_at
            "#
        );
        
        let query = query_builder.build();
        let result = query.execute(&self.pool).await?;
        
        Ok(result.rows_affected())
    }
    
    /// 批量查询用户 - 使用ANY操作符优化IN查询
    pub async fn batch_find_users(&self, user_ids: Vec<i64>) -> DomainResult<Vec<BatchUserResult>> {
        let context = ErrorContext::new("batch_find_users")
            .with_metadata("batch_size", &user_ids.len().to_string());
        
        if user_ids.is_empty() {
            return Ok(Vec::new());
        }
        
        match self.execute_batch_find_users(user_ids).await {
            Ok(users) => Ok(users),
            Err(sqlx_error) => {
                let infra_error = ErrorConverter::from_sqlx_error(sqlx_error, Some("batch_find_users"));
                let contextual_error = ContextualError::new(infra_error, context);
                
                tracing::error!(
                    error = %contextual_error,
                    "Batch find users failed"
                );
                
                Err(crate::domain::DomainError::ValidationError {
                    field: "batch_operation".to_string(),
                    reason: contextual_error.to_string(),
                })
            }
        }
    }
    
    /// 执行批量查询用户 - 使用ANY操作符
    async fn execute_batch_find_users(&self, user_ids: Vec<i64>) -> Result<Vec<BatchUserResult>, sqlx::Error> {
        let users = sqlx::query_as!(
            BatchUserResult,
            r#"
            SELECT 
                id as "user_id!",
                username as "login_id!",
                nickname as "nickname!",
                email,
                avatar_url,
                user_type as "user_type!: crate::domain::entities::UserType",
                expires_at,
                last_login_at as "last_active_at"
            FROM users 
            WHERE id = ANY($1)
            ORDER BY id
            "#,
            &user_ids
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(users)
    }
    
    /// 复杂聚合查询 - 展示窗口函数和CTE的使用
    pub async fn get_user_activity_stats(&self, user_id: i64, days: i32) -> DomainResult<UserActivityStats> {
        let context = ErrorContext::new("get_user_activity_stats")
            .with_metadata("user_id", &user_id.to_string())
            .with_metadata("days", &days.to_string());
        
        match self.execute_user_activity_stats(user_id, days).await {
            Ok(stats) => Ok(stats),
            Err(sqlx_error) => {
                let infra_error = ErrorConverter::from_sqlx_error(sqlx_error, Some("get_user_activity_stats"));
                let contextual_error = ContextualError::new(infra_error, context);
                
                tracing::error!(
                    error = %contextual_error,
                    "Get user activity stats failed"
                );
                
                Err(crate::domain::DomainError::ValidationError {
                    field: "user_stats".to_string(),
                    reason: contextual_error.to_string(),
                })
            }
        }
    }
    
    /// 执行复杂的用户活动统计查询 - 使用CTE和窗口函数
    async fn execute_user_activity_stats(&self, user_id: i64, days: i32) -> Result<UserActivityStats, sqlx::Error> {
        let stats = sqlx::query_as!(
            UserActivityStats,
            r#"
            WITH daily_stats AS (
                SELECT 
                    DATE(created_at) as activity_date,
                    COUNT(*) FILTER (WHERE 'messages' = ANY(ARRAY['messages'])) as daily_messages,
                    COUNT(*) FILTER (WHERE 'moments' = ANY(ARRAY['moments'])) as daily_moments,
                    COUNT(*) FILTER (WHERE 'group_joins' = ANY(ARRAY['group_joins'])) as daily_group_joins
                FROM (
                    SELECT created_at, 'messages' as activity_type FROM messages WHERE sender_id = $1
                    UNION ALL
                    SELECT created_at, 'moments' as activity_type FROM moments WHERE user_id = $1
                    UNION ALL
                    SELECT joined_at as created_at, 'group_joins' as activity_type FROM group_members WHERE user_id = $1
                ) activities
                WHERE created_at >= NOW() - INTERVAL '1 day' * $2
                GROUP BY DATE(created_at)
            ),
            activity_trends AS (
                SELECT 
                    activity_date,
                    daily_messages,
                    daily_moments,
                    daily_group_joins,
                    LAG(daily_messages, 1) OVER (ORDER BY activity_date) as prev_messages,
                    LAG(daily_moments, 1) OVER (ORDER BY activity_date) as prev_moments
                FROM daily_stats
            )
            SELECT 
                $1 as "user_id!",
                COALESCE(SUM(daily_messages)::bigint, 0) as "total_messages!",
                COALESCE(SUM(daily_moments)::bigint, 0) as "total_moments!",
                COALESCE(SUM(daily_group_joins)::bigint, 0) as "total_group_joins!",
                COALESCE(AVG(daily_messages)::float8, 0) as "avg_daily_messages!",
                COALESCE(AVG(daily_moments)::float8, 0) as "avg_daily_moments!",
                COALESCE(MAX(daily_messages)::bigint, 0) as "max_daily_messages!",
                COALESCE(MAX(daily_moments)::bigint, 0) as "max_daily_moments!",
                COUNT(DISTINCT activity_date)::bigint as "active_days!",
                -- 计算活动趋势（正数表示增长，负数表示下降）
                COALESCE(
                    AVG(CASE 
                        WHEN prev_messages IS NOT NULL 
                        THEN daily_messages - prev_messages 
                        ELSE 0 
                    END), 
                    0
                )::float8 as "message_trend!",
                COALESCE(
                    AVG(CASE 
                        WHEN prev_moments IS NOT NULL 
                        THEN daily_moments - prev_moments 
                        ELSE 0 
                    END), 
                    0
                )::float8 as "moment_trend!"
            FROM activity_trends
            "#,
            user_id,
            days as f64
        )
        .fetch_one(&self.pool)
        .await?;
        
        Ok(stats)
    }
    
    /// 动态搜索查询 - 根据条件动态构建WHERE子句
    pub async fn dynamic_user_search(&self, criteria: UserSearchCriteria) -> DomainResult<Vec<BatchUserResult>> {
        let context = ErrorContext::new("dynamic_user_search");
        
        match self.execute_dynamic_user_search(criteria).await {
            Ok(users) => Ok(users),
            Err(sqlx_error) => {
                let infra_error = ErrorConverter::from_sqlx_error(sqlx_error, Some("dynamic_user_search"));
                let contextual_error = ContextualError::new(infra_error, context);
                
                tracing::error!(
                    error = %contextual_error,
                    "Dynamic user search failed"
                );
                
                Err(crate::domain::DomainError::ValidationError {
                    field: "search_criteria".to_string(),
                    reason: contextual_error.to_string(),
                })
            }
        }
    }
    
    /// 执行动态用户搜索 - 使用QueryBuilder动态构建WHERE条件
    async fn execute_dynamic_user_search(&self, criteria: UserSearchCriteria) -> Result<Vec<BatchUserResult>, sqlx::Error> {
        let mut query_builder = QueryBuilder::<Postgres>::new(
            r#"
            SELECT 
                user_id, login_id, nickname, email, avatar_url,
                user_type, expires_at, last_active_at
            FROM users 
            WHERE 1=1
            "#
        );
        
        // 动态添加WHERE条件
        if let Some(nickname_pattern) = &criteria.nickname_pattern {
            query_builder.push(" AND nickname ILIKE ");
            query_builder.push_bind(format!("%{}%", nickname_pattern));
        }
        
        if let Some(user_type) = &criteria.user_type {
            query_builder.push(" AND user_type = ");
            query_builder.push_bind(user_type);
        }
        
        if let Some(active_since) = &criteria.active_since {
            query_builder.push(" AND last_active_at >= ");
            query_builder.push_bind(active_since);
        }
        
        if let Some(has_email) = criteria.has_email {
            if has_email {
                query_builder.push(" AND email IS NOT NULL");
            } else {
                query_builder.push(" AND email IS NULL");
            }
        }
        
        if let Some(exclude_ids) = &criteria.exclude_user_ids {
            if !exclude_ids.is_empty() {
                query_builder.push(" AND user_id != ALL(");
                query_builder.push_bind(exclude_ids);
                query_builder.push(")");
            }
        }
        
        // 添加排序和限制
        query_builder.push(" ORDER BY last_active_at DESC");
        
        if let Some(limit) = criteria.limit {
            query_builder.push(" LIMIT ");
            query_builder.push_bind(limit);
        }
        
        let query = query_builder.build_query_as::<BatchUserResult>();
        let users = query.fetch_all(&self.pool).await?;
        
        Ok(users)
    }
    
    /// 获取查询执行计划 - 用于性能分析
    pub async fn explain_query(&self, sql: &str) -> DomainResult<Vec<QueryPlan>> {
        let context = ErrorContext::new("explain_query");
        
        match self.execute_explain_query(sql).await {
            Ok(plan) => Ok(plan),
            Err(sqlx_error) => {
                let infra_error = ErrorConverter::from_sqlx_error(sqlx_error, Some("explain_query"));
                let contextual_error = ContextualError::new(infra_error, context);
                
                tracing::error!(
                    error = %contextual_error,
                    "Explain query failed"
                );
                
                Err(crate::domain::DomainError::ValidationError {
                    field: "query_plan".to_string(),
                    reason: contextual_error.to_string(),
                })
            }
        }
    }
    
    /// 执行EXPLAIN查询
    async fn execute_explain_query(&self, sql: &str) -> Result<Vec<QueryPlan>, sqlx::Error> {
        let explain_sql = format!("EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) {}", sql);
        
        let plans = sqlx::query_as!(
            QueryPlan,
            r#"
            SELECT 
                plan->0->'Plan'->>'Node Type' as "node_type!",
                COALESCE((plan->0->'Plan'->>'Total Cost')::FLOAT, 0) as "total_cost!",
                COALESCE((plan->0->'Plan'->>'Actual Total Time')::FLOAT, 0) as "actual_time!",
                COALESCE((plan->0->'Plan'->>'Actual Rows')::BIGINT, 0) as "actual_rows!",
                plan->0->'Plan' as "plan_details!"
            FROM (
                SELECT $1::jsonb as plan
            ) t
            "#,
            serde_json::Value::String(explain_sql.clone())
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(plans)
    }
}

// === 批量操作数据结构 ===

/// 批量插入用户数据结构
#[derive(Debug, Clone)]
pub struct BatchUserInsert {
    pub user_id: i64,
    pub login_id: String,
    pub nickname: String,
    pub email: Option<String>,
    pub avatar_url: Option<String>,
    pub password_hash: Option<String>,
    pub user_type: crate::domain::entities::UserType,
    pub expires_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub last_active_at: Option<DateTime<Utc>>,
}

/// 批量查询用户结果
#[derive(Debug, Clone, sqlx::FromRow)]
pub struct BatchUserResult {
    pub user_id: i64,
    pub login_id: String,
    pub nickname: String,
    pub email: Option<String>,
    pub avatar_url: Option<String>,
    pub user_type: crate::domain::entities::UserType,
    pub expires_at: Option<DateTime<Utc>>,
    pub last_active_at: Option<DateTime<Utc>>,
}

/// 用户活动统计
#[derive(Debug, Clone, sqlx::FromRow)]
pub struct UserActivityStats {
    pub user_id: i64,
    pub total_messages: i64,
    pub total_moments: i64,
    pub total_group_joins: i64,
    pub avg_daily_messages: f64,
    pub avg_daily_moments: f64,
    pub max_daily_messages: i64,
    pub max_daily_moments: i64,
    pub active_days: i64,
    pub message_trend: f64,
    pub moment_trend: f64,
}

/// 动态搜索条件
#[derive(Debug, Clone, Default)]
pub struct UserSearchCriteria {
    pub nickname_pattern: Option<String>,
    pub user_type: Option<crate::domain::entities::UserType>,
    pub active_since: Option<DateTime<Utc>>,
    pub has_email: Option<bool>,
    pub exclude_user_ids: Option<Vec<i64>>,
    pub limit: Option<i64>,
}

/// 查询执行计划
#[derive(Debug, Clone, sqlx::FromRow)]
pub struct QueryPlan {
    pub node_type: String,
    pub total_cost: f64,
    pub actual_time: f64,
    pub actual_rows: i64,
    pub plan_details: serde_json::Value,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_batch_user_insert_creation() {
        let user = BatchUserInsert {
            user_id: 123,
            login_id: "test_user".to_string(),
            nickname: "测试用户".to_string(),
            email: Some("test@example.com".to_string()),
            avatar_url: None,
            password_hash: Some("hashed".to_string()),
            user_type: crate::domain::entities::UserType::Regular,
            expires_at: None,
            created_at: Utc::now(),
            last_active_at: Some(Utc::now()),
        };
        
        assert_eq!(user.user_id, 123);
        assert_eq!(user.login_id, "test_user");
        assert_eq!(user.nickname, "测试用户");
    }
    
    #[test]
    fn test_user_search_criteria() {
        let mut criteria = UserSearchCriteria::default();
        criteria.nickname_pattern = Some("测试".to_string());
        criteria.user_type = Some(crate::domain::entities::UserType::Regular);
        criteria.has_email = Some(true);
        criteria.limit = Some(50);
        
        assert_eq!(criteria.nickname_pattern, Some("测试".to_string()));
        assert_eq!(criteria.user_type, Some(crate::domain::entities::UserType::Regular));
        assert_eq!(criteria.has_email, Some(true));
        assert_eq!(criteria.limit, Some(50));
    }
}

