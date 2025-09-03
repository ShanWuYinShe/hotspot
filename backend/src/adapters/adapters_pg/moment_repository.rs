/// 瞬间仓储PostgreSQL适配器（待实现）

use async_trait::async_trait;
use sqlx::PgPool;
use crate::domain::{Moment, DomainResult, DomainError};
use crate::ports::MomentRepositoryPort;

#[derive(Clone)]
#[allow(dead_code)]
pub struct PgMomentRepository {
    #[allow(dead_code)]
    pool: PgPool,
}

impl PgMomentRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl MomentRepositoryPort for PgMomentRepository {
    async fn find_by_id(&self, moment_id: i64) -> DomainResult<Option<Moment>> {
        let row = sqlx::query!(
            r#"
            SELECT id as "id!", moment_id as "moment_id!", user_id as "user_id!", content, 
                   latitude as "latitude!", longitude as "longitude!", location_name as "location_name!", 
                   created_at as "created_at!"
            FROM moments
            WHERE id = $1
            "#,
            moment_id
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
    }

    async fn create_moment(&self, moment: &Moment) -> DomainResult<Moment> {
        let row = sqlx::query!(
            r#"
            INSERT INTO moments (moment_id, user_id, content, latitude, longitude, location_name, created_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            RETURNING id as "id!", moment_id as "moment_id!", user_id as "user_id!", content, 
                      latitude as "latitude!", longitude as "longitude!", location_name as "location_name!", 
                      created_at as "created_at!"
            "#,
            uuid::Uuid::parse_str(&moment.moment_id).map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Invalid UUID: {}", e) })?,
            moment.user_id,
            moment.content.as_ref(),
            moment.latitude,
            moment.longitude,
            moment.location_name,
            moment.created_at
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
    }

    async fn update_moment(&self, moment: &Moment) -> DomainResult<Moment> {
        let row = sqlx::query!(
            r#"
            UPDATE moments
            SET content = $2,
                latitude = $3,
                longitude = $4,
                location_name = $5
            WHERE id = $1
            RETURNING id as "id!", moment_id as "moment_id!", user_id as "user_id!", content, 
                      latitude as "latitude!", longitude as "longitude!", location_name as "location_name!", 
                      created_at as "created_at!"
            "#,
            moment.id,
            moment.content.as_ref(),
            moment.latitude,
            moment.longitude,
            moment.location_name
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        if let Some(r) = row {
            Ok(Moment {
                id: r.id,
                moment_id: r.moment_id.to_string(),
                user_id: r.user_id,
                content: r.content,
                latitude: r.latitude,
                longitude: r.longitude,
                location_name: r.location_name,
                created_at: r.created_at,
            })
        } else {
            Err(DomainError::BusinessRuleViolation { rule: "Moment not found".to_string() })
        }
    }

    async fn delete_moment(&self, moment_id: i64) -> DomainResult<()> {
        let rows = sqlx::query!(
            "DELETE FROM moments WHERE id = $1",
            moment_id
        )
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
        let rows = sqlx::query!(
            r#"
            SELECT id as "id!", moment_id as "moment_id!", user_id as "user_id!", content, 
                   latitude as "latitude!", longitude as "longitude!", location_name as "location_name!", 
                   created_at as "created_at!"
            FROM moments
            WHERE user_id = $1 AND ($2 = 0 OR id < $2)
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            (cursor as i32),
            (limit as i64)
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        let items = rows.into_iter().map(|r| Moment {
            id: r.id,
            moment_id: r.moment_id.to_string(),
            user_id: r.user_id,
            content: r.content,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            created_at: r.created_at,
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
        let rows = sqlx::query!(
            r#"
            SELECT id as "id!", moment_id as "moment_id!", user_id as "user_id!", content, 
                   latitude as "latitude!", longitude as "longitude!", location_name as "location_name!", 
                   created_at as "created_at!",
                   ST_Distance(
                       geom,
                       ST_Point($2, $1)::geography
                   ) / 1000.0 as distance_km
            FROM moments
            WHERE ($4 = 0 OR id < $4)
              AND ST_DWithin(
                    geom,
                    ST_Point($2, $1)::geography,
                    $3 * 1000
              )
            ORDER BY id DESC
            LIMIT $5
            "#,
            latitude,
            longitude,
            (radius_km as i32),
            (cursor as i32),
            (limit as i64)
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        let items = rows.into_iter().map(|r| Moment {
            id: r.id,
            moment_id: r.moment_id.to_string(),
            user_id: r.user_id,
            content: r.content,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            created_at: r.created_at,
        }).collect();
        Ok(items)
    }

    async fn like_moment(&self, moment_id: i64, user_id: i64) -> DomainResult<()> {
        // 简单实现：插入点赞记录（如果不存在）
        sqlx::query!(
            "INSERT INTO moment_likes (moment_id, user_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT (moment_id, user_id) DO NOTHING",
            moment_id as i64,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;
        
        Ok(())
    }

    async fn unlike_moment(&self, moment_id: i64, user_id: i64) -> DomainResult<()> {
        // 简单实现：删除点赞记录
        sqlx::query!(
            "DELETE FROM moment_likes WHERE moment_id = $1 AND user_id = $2",
            moment_id as i64,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;
        
        Ok(())
    }

    async fn search_by_tags(
        &self,
        tags: Vec<String>,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Moment>> {
        // 简单实现：暂时返回空结果，因为需要标签表结构支持
        log::info!("按标签搜索瞬间: {:?}, cursor: {}, limit: {}", tags, cursor, limit);
        Ok(vec![])
    }
}

