/// 瞬间仓储PostgreSQL适配器（待实现）
use async_trait::async_trait;
use sqlx::{PgPool, Row};
use crate::domain::{Moment, DomainResult, DomainError};
use crate::ports::MomentRepository;
use chrono::{DateTime, Utc};
use super::query_optimizer::PgQueryOptimizer;

/// 瞬间查询结果行结构体
#[derive(Debug)]
struct MomentRow {
    pub id: i64,
    pub moment_id: String,
    pub user_id: i64,
    pub content: String,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Clone)]
pub struct PgMomentRepository {
    pool: PgPool,
    query_optimizer: PgQueryOptimizer,
}

impl PgMomentRepository {
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
        let params_summary = std::collections::HashMap::new();
        
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
                    repository = "moment_repository",
                    "慢查询检测"
                );
            } else {
                tracing::debug!(
                    operation = operation_name,
                    duration_ms = perf_metrics.execution_time_ms,
                    query_type = perf_metrics.query_type,
                    used_index = perf_metrics.used_index,
                    repository = "moment_repository",
                    "查询完成"
                );
            }
        }
        
        Ok(result)
    }
}

#[async_trait]
impl MomentRepository for PgMomentRepository {
    async fn find_by_id(&self, moment_id: &str) -> DomainResult<Option<Moment>> {
        self.execute_query_with_monitoring("find_by_id", async {
            let moment_uuid = uuid::Uuid::parse_str(moment_id)
                .map_err(|_| DomainError::BusinessRuleViolation {
                    rule: "Invalid moment ID format".to_string(),
                })?;
                
            let row = sqlx::query_as!(
                MomentRow,
                r#"
                SELECT id as "id!", moment_id as "moment_id!", user_id as "user_id!", content, 
                       latitude as "latitude!", longitude as "longitude!", location_name as "location_name!", 
                       created_at as "created_at!"
                FROM moments
                WHERE moment_id = $1
                "#,
                moment_uuid.to_string()
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

            Ok(row.map(|r| Moment {
                id: r.id,
                moment_id: r.moment_id.to_string(),
                user_id: r.user_id,
                content: r.content,
                latitude: r.latitude,
                longitude: r.longitude,
                location_name: r.location_name,
                created_at: r.created_at,
            }))
        }).await
    }

    async fn create_moment(
        &self,
        user_id: i64,
        description: &str,
        latitude: f64,
        longitude: f64,
        location_name: &str,
        _tags: &[String],
    ) -> DomainResult<Moment> {
        self.execute_query_with_monitoring("create_moment", async {
            let row = sqlx::query_as!(
                MomentRow,
                r#"
                INSERT INTO moments (moment_id, user_id, content, latitude, longitude, location_name, created_at)
                VALUES ($1, $2, $3, $4, $5, $6, $7)
                RETURNING id as "id!", moment_id as "moment_id!", user_id as "user_id!", content, 
                          latitude as "latitude!", longitude as "longitude!", location_name as "location_name!", 
                          created_at as "created_at!"
                "#,
                uuid::Uuid::new_v4().to_string(), // 生成新的 UUID
                user_id,
                description,
                latitude,
                longitude,
                location_name,
                chrono::Utc::now()
            )
            .fetch_one(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

            Ok(Moment {
                id: row.id,
                moment_id: row.moment_id.to_string(),
                user_id: row.user_id,
                content: row.content,
                latitude: row.latitude,
                longitude: row.longitude,
                location_name: row.location_name,
                created_at: row.created_at,
            })
        }).await
    }

    async fn update_moment(
        &self,
        moment_id: &str,
        user_id: i64,
        description: Option<&str>,
        _tags: Option<&[String]>,
    ) -> DomainResult<Moment> {
        let moment_uuid = uuid::Uuid::parse_str(moment_id)
            .map_err(|_| DomainError::BusinessRuleViolation {
                rule: "Invalid moment ID format".to_string(),
            })?;
            
        let row = sqlx::query(
            r#"
            UPDATE moments
            SET content = COALESCE($2, content)
            WHERE moment_id = $1 AND user_id = $3
            RETURNING id, moment_id, user_id, content, 
                      latitude, longitude, location_name, 
                      created_at
            "#
        )
        .bind(moment_uuid.to_string())
        .bind(description)
        .bind(user_id)
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        if let Some(r) = row {
            Ok(Moment {
                id: r.get("id"),
                moment_id: r.get::<uuid::Uuid, _>("moment_id").to_string(),
                user_id: r.get("user_id"),
                content: r.get("content"),
                latitude: r.get("latitude"),
                longitude: r.get("longitude"),
                location_name: r.get("location_name"),
                created_at: r.get("created_at"),
            })
        } else {
            Err(DomainError::BusinessRuleViolation { rule: "Moment not found".to_string() })
        }
    }

    async fn delete_moment(&self, moment_id: &str, user_id: i64) -> DomainResult<()> {
        let moment_uuid = uuid::Uuid::parse_str(moment_id)
            .map_err(|_| DomainError::BusinessRuleViolation {
                rule: "Invalid moment ID format".to_string(),
            })?;
            
        let rows = sqlx::query(
            "DELETE FROM moments WHERE moment_id = $1 AND user_id = $2"
        )
        .bind(moment_uuid.to_string())
        .bind(user_id)
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?
        .rows_affected();
        if rows == 0 {
            return Err(DomainError::BusinessRuleViolation { rule: "Moment not found".to_string() });
        }
        Ok(())
    }

    async fn find_user_moments(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Moment>> {
        let rows = sqlx::query(
            r#"
            SELECT id, moment_id, user_id, content, 
                   latitude, longitude, location_name, 
                   created_at
            FROM moments
            WHERE user_id = $1 AND ($2 = 0 OR id < $2)
            ORDER BY id DESC
            LIMIT $3
            "#
        )
        .bind(user_id)
        .bind(cursor as i32)
        .bind(limit as i64)
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        let items = rows.into_iter().map(|r| Moment {
            id: r.get("id"),
            moment_id: r.get::<uuid::Uuid, _>("moment_id").to_string(),
            user_id: r.get("user_id"),
            content: r.get("content"),
            latitude: r.get("latitude"),
            longitude: r.get("longitude"),
            location_name: r.get("location_name"),
            created_at: r.get("created_at"),
        }).collect();
        Ok(items)
    }

    async fn find_nearby_moments(
        &self,
        latitude: f64,
        longitude: f64,
        radius_km: f64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Moment>> {
        let rows = sqlx::query(
            r#"
            SELECT id, moment_id, user_id, content, 
                   latitude, longitude, location_name, 
                   created_at,
                   ST_Distance(
                       location,
                       ST_Point($2, $1)::geography
                   ) / 1000.0 as distance_km
            FROM moments
            WHERE ($4 = 0 OR id < $4)
              AND ST_DWithin(
                    location,
                    ST_Point($2, $1)::geography,
                    $3 * 1000
              )
            ORDER BY id DESC
            LIMIT $5
            "#
        )
        .bind(latitude)
        .bind(longitude)
        .bind(radius_km as i32)
        .bind(cursor as i32)
        .bind(limit as i64)
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        let items = rows.into_iter().map(|r| Moment {
            id: r.get("id"),
            moment_id: r.get::<uuid::Uuid, _>("moment_id").to_string(),
            user_id: r.get("user_id"),
            content: r.get("content"),
            latitude: r.get("latitude"),
            longitude: r.get("longitude"),
            location_name: r.get("location_name"),
            created_at: r.get("created_at"),
        }).collect();
        Ok(items)
    }

    async fn like_moment(&self, moment_id: &str, _user_id: i64) -> DomainResult<()> {
        let moment_uuid = uuid::Uuid::parse_str(moment_id)
            .map_err(|_| DomainError::BusinessRuleViolation {
                rule: "Invalid moment ID format".to_string(),
            })?;
            
        // 更新点赞计数（简化实现，不记录具体点赞用户）
        sqlx::query(
            "UPDATE moments SET like_count = like_count + 1 WHERE moment_id = $1"
        )
        .bind(moment_uuid.to_string())
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;
        
        Ok(())
    }

    async fn unlike_moment(&self, moment_id: &str, _user_id: i64) -> DomainResult<()> {
        let moment_uuid = uuid::Uuid::parse_str(moment_id)
            .map_err(|_| DomainError::BusinessRuleViolation {
                rule: "Invalid moment ID format".to_string(),
            })?;
            
        // 减少点赞计数（简化实现，不记录具体点赞用户）
        sqlx::query(
            "UPDATE moments SET like_count = GREATEST(like_count - 1, 0) WHERE moment_id = $1"
        )
        .bind(moment_uuid.to_string())
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;
        
        Ok(())
    }

    async fn search_by_tags(
        &self,
        tags: &[String],
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Moment>> {
        // 简单实现：暂时返回空结果，因为需要标签表结构支持
        log::info!("按标签搜索瞬间: {:?}, cursor: {}, limit: {}", tags, cursor, limit);
        Ok(vec![])
    }

    /// 检查用户是否已点赞生活瞬间
    async fn is_moment_liked(&self, moment_id: &str, user_id: i64) -> DomainResult<bool> {
        // 简化实现：由于不记录具体点赞用户，总是返回false
        // 在实际应用中，可以考虑添加单独的用户点赞记录表或使用其他方式跟踪
        let _ = moment_id;
        let _ = user_id;
        Ok(false)
    }

    /// 获取生活瞬间点赞数
    async fn get_moment_likes_count(&self, moment_id: &str) -> DomainResult<i64> {
        let moment_uuid = uuid::Uuid::parse_str(moment_id)
            .map_err(|_| DomainError::BusinessRuleViolation {
                rule: "Invalid moment ID format".to_string(),
            })?;
            
        let count = sqlx::query_scalar::<_, i64>(
            "SELECT like_count FROM moments WHERE moment_id = $1"
        )
        .bind(moment_uuid.to_string())
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?
        .unwrap_or(0);
        
        Ok(count)
    }
}

