//! PostgreSQL事件存储 - 生产级事件存储实现
//! 
//! 特性：
//! - 事务安全的事件存储
//! - 事件版本控制
//! - 高性能查询
//! - 事件快照支持
//! - 分区表优化

#![cfg(feature = "event_store")]

use async_trait::async_trait;
use sqlx::PgPool;
use chrono::{DateTime, Utc};
// use uuid::Uuid; // 暂时未使用
use crate::domain::DomainResult;
use crate::infrastructure::event_system::{EventStore, EventEnvelope, EventId};
use crate::infrastructure::errors::{ErrorContext, ContextualError, ErrorConverter};

/// PostgreSQL事件存储实现
#[derive(Clone)]
pub struct PostgresEventStore {
    pool: PgPool,
}

/// 事件存储行结构 - 对应events表
#[derive(Debug, sqlx::FromRow)]
struct EventRow {
    /// 事件ID
    event_id: String,
    /// 事件类型
    event_type: String,
    /// 事件版本
    version: i32,
    /// 聚合根ID
    aggregate_id: String,
    /// 聚合根类型
    aggregate_type: String,
    /// 事件数据（JSON格式）
    event_data: serde_json::Value,
    /// 事件元数据（JSON格式）
    metadata: serde_json::Value,
    /// 事件发生时间
    occurred_at: DateTime<Utc>,
    /// 因果关系ID
    causation_id: Option<String>,
    /// 相关性ID
    correlation_id: Option<String>,
    /// 用户ID
    user_id: Option<i64>,
    /// 会话ID
    session_id: Option<String>,
}

impl PostgresEventStore {
    /// 创建新的PostgreSQL事件存储
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
    
    /// 初始化事件存储表结构
    pub async fn initialize(&self) -> DomainResult<()> {
        let context = ErrorContext::new("initialize_event_store");
        
        match self.create_tables().await {
            Ok(()) => Ok(()),
            Err(sqlx_error) => {
                let infra_error = ErrorConverter::from_sqlx_error(sqlx_error, Some("initialize_event_store"));
                let contextual_error = ContextualError::new(infra_error, context);
                
                tracing::error!(
                    error = %contextual_error,
                    "Failed to initialize event store"
                );
                
                Err(crate::domain::DomainError::BusinessRuleViolation {
                    rule: "event store initialization failed".to_string(),
                })
            }
        }
    }
    
    /// 创建事件存储相关表
    async fn create_tables(&self) -> Result<(), sqlx::Error> {
        // 创建事件表
        sqlx::query!(
            r#"
            CREATE TABLE IF NOT EXISTS events (
                id BIGSERIAL PRIMARY KEY,
                event_id UUID NOT NULL UNIQUE,
                event_type VARCHAR(255) NOT NULL,
                version INTEGER NOT NULL DEFAULT 1,
                aggregate_id VARCHAR(255) NOT NULL,
                aggregate_type VARCHAR(255) NOT NULL,
                event_data JSONB NOT NULL,
                metadata JSONB NOT NULL DEFAULT '{}',
                occurred_at TIMESTAMPTZ NOT NULL,
                occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                causation_id UUID,
                correlation_id VARCHAR(255),
                user_id BIGINT,
                session_id VARCHAR(255),
                
                -- 索引优化
                CONSTRAINT events_event_id_unique UNIQUE (event_id),
                CONSTRAINT events_aggregate_version_unique UNIQUE (aggregate_id, version)
            )
            "#
        )
        .execute(&self.pool)
        .await?;
        
        // 创建索引（分别执行以避免多命令问题）
        sqlx::query!("CREATE INDEX IF NOT EXISTS idx_events_aggregate_id ON events(aggregate_id)")
            .execute(&self.pool).await?;
        sqlx::query!("CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type)")
            .execute(&self.pool).await?;
        sqlx::query!("CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events(occurred_at)")
            .execute(&self.pool).await?;
        sqlx::query!("CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(user_id) WHERE user_id IS NOT NULL")
            .execute(&self.pool).await?;
        sqlx::query!("CREATE INDEX IF NOT EXISTS idx_events_correlation_id ON events(correlation_id) WHERE correlation_id IS NOT NULL")
            .execute(&self.pool).await?;
        sqlx::query!("CREATE INDEX IF NOT EXISTS idx_events_aggregate_type_occurred ON events(aggregate_type, occurred_at)")
            .execute(&self.pool).await?;
        sqlx::query!("CREATE INDEX IF NOT EXISTS idx_events_event_data_gin ON events USING GIN (event_data)")
            .execute(&self.pool).await?;
        sqlx::query!("CREATE INDEX IF NOT EXISTS idx_events_metadata_gin ON events USING GIN (metadata)")
            .execute(&self.pool).await?;
        
        // 创建事件快照表（用于优化聚合根重建）
        sqlx::query!(
            r#"
            CREATE TABLE IF NOT EXISTS event_snapshots (
                id BIGSERIAL PRIMARY KEY,
                aggregate_id VARCHAR(255) NOT NULL,
                aggregate_type VARCHAR(255) NOT NULL,
                version INTEGER NOT NULL,
                snapshot_data JSONB NOT NULL,
                occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                
                CONSTRAINT event_snapshots_aggregate_unique UNIQUE (aggregate_id, aggregate_type)
            )
            "#
        )
        .execute(&self.pool)
        .await?;
        
        // 创建事件处理进度表（用于跟踪事件处理器的处理进度）
        sqlx::query!(
            r#"
            CREATE TABLE IF NOT EXISTS event_processing_progress (
                id BIGSERIAL PRIMARY KEY,
                handler_name VARCHAR(255) NOT NULL UNIQUE,
                last_processed_event_id UUID,
                last_processed_at TIMESTAMPTZ,
                occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )
            "#
        )
        .execute(&self.pool)
        .await?;
        
        Ok(())
    }
    
    /// 执行查询并处理错误
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
                
                tracing::error!(
                    error = %contextual_error,
                    "Event store operation failed"
                );
                
                Err(crate::domain::DomainError::BusinessRuleViolation {
                    rule: format!("event store operation failed: {}", contextual_error),
                })
            }
        }
    }
    
    /// 将事件包装器转换为数据库行
    fn envelope_to_row(&self, envelope: &EventEnvelope) -> Result<EventRowInsert, sqlx::Error> {
        let event_data = serde_json::from_slice::<serde_json::Value>(&envelope.data)
            .map_err(|e| sqlx::Error::Protocol(format!("Failed to parse event data as JSON: {}", e)))?;
        
        let metadata = serde_json::to_value(&envelope.metadata.metadata)
            .map_err(|e| sqlx::Error::Protocol(format!("Failed to serialize metadata: {}", e)))?;
        
        Ok(EventRowInsert {
            event_id: envelope.metadata.event_id.value().to_string(),
            event_type: envelope.metadata.event_type.clone(),
            version: envelope.metadata.version as i32,
            aggregate_id: envelope.metadata.aggregate_id.clone(),
            aggregate_type: envelope.metadata.aggregate_type.clone(),
            event_data,
            metadata,
            occurred_at: envelope.metadata.occurred_at,
            causation_id: envelope.metadata.causation_id.as_ref().map(|id| id.value().to_string()),
            correlation_id: envelope.metadata.correlation_id.clone(),
            user_id: envelope.metadata.user_id.as_ref().map(|id| id.value()),
            session_id: envelope.metadata.session_id.clone(),
        })
    }
    
    /// 将数据库行转换为事件包装器
    fn row_to_envelope(&self, row: EventRow) -> Result<EventEnvelope, sqlx::Error> {
        let event_data = serde_json::to_vec(&row.event_data)
            .map_err(|e| sqlx::Error::Protocol(format!("Failed to serialize event data: {}", e)))?;
        
        let metadata_map: std::collections::HashMap<String, String> = 
            serde_json::from_value(row.metadata)
                .unwrap_or_default();
        
        let metadata = crate::infrastructure::event_system::EventMetadata {
            event_id: EventId::from_string(row.event_id),
            event_type: row.event_type.clone(),
            version: row.version as u32,
            aggregate_id: row.aggregate_id,
            aggregate_type: row.aggregate_type,
            occurred_at: row.occurred_at,
            causation_id: row.causation_id.map(EventId::from_string),
            correlation_id: row.correlation_id,
            user_id: row.user_id.map(crate::domain::UserId::from_trusted),
            session_id: row.session_id,
            metadata: metadata_map,
        };
        
        Ok(EventEnvelope {
            metadata,
            data: event_data,
            event_type_info: row.event_type,
        })
    }
}

/// 事件插入数据结构
struct EventRowInsert {
    event_id: String,
    event_type: String,
    version: i32,
    aggregate_id: String,
    aggregate_type: String,
    event_data: serde_json::Value,
    metadata: serde_json::Value,
    occurred_at: DateTime<Utc>,
    causation_id: Option<String>,
    correlation_id: Option<String>,
    user_id: Option<i64>,
    session_id: Option<String>,
}

#[async_trait]
impl EventStore for PostgresEventStore {
    /// 保存单个事件
    async fn save_event(&self, envelope: EventEnvelope) -> DomainResult<()> {
        self.execute_query("save_event", async {
            let row = self.envelope_to_row(&envelope)?;
            
            sqlx::query!(
                r#"
                INSERT INTO events (
                    event_id, event_type, version, aggregate_id, aggregate_type,
                    event_data, metadata, occurred_at, causation_id, correlation_id,
                    user_id, session_id
                ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
                "#,
                row.event_id,
                row.event_type,
                row.version,
                row.aggregate_id,
                row.aggregate_type,
                row.event_data,
                row.metadata,
                row.occurred_at,
                row.causation_id,
                row.correlation_id,
                row.user_id,
                row.session_id
            )
            .execute(&self.pool)
            .await?;
            
            Ok(())
        }).await
    }
    
    /// 批量保存事件（事务安全）
    async fn save_events(&self, envelopes: Vec<EventEnvelope>) -> DomainResult<()> {
        if envelopes.is_empty() {
            return Ok(());
        }
        
        self.execute_query("save_events_batch", async {
            let mut tx = self.pool.begin().await?;
            
            for envelope in envelopes {
                let row = self.envelope_to_row(&envelope)?;
                
                sqlx::query!(
                    r#"
                    INSERT INTO events (
                        event_id, event_type, version, aggregate_id, aggregate_type,
                        event_data, metadata, occurred_at, causation_id, correlation_id,
                        user_id, session_id
                    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
                    "#,
                    row.event_id,
                    row.event_type,
                    row.version,
                    row.aggregate_id,
                    row.aggregate_type,
                    row.event_data,
                    row.metadata,
                    row.occurred_at,
                    row.causation_id,
                    row.correlation_id,
                    row.user_id,
                    row.session_id
                )
                .execute(&mut *tx)
                .await?;
            }
            
            tx.commit().await?;
            Ok(())
        }).await
    }
    
    /// 根据聚合根ID获取事件流
    async fn get_events_for_aggregate(
        &self,
        aggregate_id: &str,
        from_version: Option<u32>,
    ) -> DomainResult<Vec<EventEnvelope>> {
        self.execute_query("get_events_for_aggregate", async {
            let min_version = from_version.unwrap_or(0) as i32;
            
            let rows = sqlx::query_as!(
                EventRow,
                r#"
                SELECT 
                    event_id as "event_id!",
                    event_type as "event_type!",
                    version as "version!",
                    aggregate_id as "aggregate_id!",
                    aggregate_type as "aggregate_type!",
                    event_data as "event_data!",
                    metadata as "metadata!",
                    occurred_at as "occurred_at!",
                    causation_id,
                    correlation_id,
                    user_id,
                    session_id
                FROM events 
                WHERE aggregate_id = $1 AND version >= $2
                ORDER BY version ASC
                "#,
                aggregate_id,
                min_version
            )
            .fetch_all(&self.pool)
            .await?;
            
            let mut envelopes = Vec::new();
            for row in rows {
                let envelope = self.row_to_envelope(row)?;
                envelopes.push(envelope);
            }
            
            Ok(envelopes)
        }).await
    }
    
    /// 获取所有事件（用于重放）
    async fn get_all_events(
        &self,
        from_timestamp: Option<DateTime<Utc>>,
        limit: Option<usize>,
    ) -> DomainResult<Vec<EventEnvelope>> {
        self.execute_query("get_all_events", async {
            let min_time = from_timestamp.unwrap_or_else(|| DateTime::from_timestamp(0, 0).unwrap());
            let max_count = limit.unwrap_or(1000).min(10000) as i64; // 限制最大10000条
            
            let rows = sqlx::query_as!(
                EventRow,
                r#"
                SELECT 
                    event_id as "event_id!",
                    event_type as "event_type!",
                    version as "version!",
                    aggregate_id as "aggregate_id!",
                    aggregate_type as "aggregate_type!",
                    event_data as "event_data!",
                    metadata as "metadata!",
                    occurred_at as "occurred_at!",
                    causation_id,
                    correlation_id,
                    user_id,
                    session_id
                FROM events 
                WHERE occurred_at >= $1
                ORDER BY occurred_at ASC, id ASC
                LIMIT $2
                "#,
                min_time,
                max_count
            )
            .fetch_all(&self.pool)
            .await?;
            
            let mut envelopes = Vec::new();
            for row in rows {
                let envelope = self.row_to_envelope(row)?;
                envelopes.push(envelope);
            }
            
            Ok(envelopes)
        }).await
    }
    
    /// 根据事件类型获取事件
    async fn get_events_by_type(
        &self,
        event_type: &str,
        from_timestamp: Option<DateTime<Utc>>,
        limit: Option<usize>,
    ) -> DomainResult<Vec<EventEnvelope>> {
        self.execute_query("get_events_by_type", async {
            let min_time = from_timestamp.unwrap_or_else(|| DateTime::from_timestamp(0, 0).unwrap());
            let max_count = limit.unwrap_or(1000).min(10000) as i64;
            
            let rows = sqlx::query_as!(
                EventRow,
                r#"
                SELECT 
                    event_id as "event_id!",
                    event_type as "event_type!",
                    version as "version!",
                    aggregate_id as "aggregate_id!",
                    aggregate_type as "aggregate_type!",
                    event_data as "event_data!",
                    metadata as "metadata!",
                    occurred_at as "occurred_at!",
                    causation_id,
                    correlation_id,
                    user_id,
                    session_id
                FROM events 
                WHERE event_type = $1 AND occurred_at >= $2
                ORDER BY occurred_at ASC, id ASC
                LIMIT $3
                "#,
                event_type,
                min_time,
                max_count
            )
            .fetch_all(&self.pool)
            .await?;
            
            let mut envelopes = Vec::new();
            for row in rows {
                let envelope = self.row_to_envelope(row)?;
                envelopes.push(envelope);
            }
            
            Ok(envelopes)
        }).await
    }
}

/// 事件存储扩展功能
impl PostgresEventStore {
    /// 保存聚合根快照
    pub async fn save_snapshot(
        &self,
        aggregate_id: &str,
        aggregate_type: &str,
        version: u32,
        snapshot_data: &serde_json::Value,
    ) -> DomainResult<()> {
        self.execute_query("save_snapshot", async {
            sqlx::query!(
                r#"
                INSERT INTO event_snapshots (aggregate_id, aggregate_type, version, snapshot_data)
                VALUES ($1, $2, $3, $4)
                ON CONFLICT (aggregate_id, aggregate_type) DO UPDATE SET
                    version = EXCLUDED.version,
                    snapshot_data = EXCLUDED.snapshot_data,
                    created_at = NOW()
                "#,
                aggregate_id,
                aggregate_type,
                version as i32,
                snapshot_data
            )
            .execute(&self.pool)
            .await?;
            
            Ok(())
        }).await
    }
    
    /// 获取聚合根快照
    pub async fn get_snapshot(
        &self,
        aggregate_id: &str,
        aggregate_type: &str,
    ) -> DomainResult<Option<(u32, serde_json::Value)>> {
        self.execute_query("get_snapshot", async {
            let row = sqlx::query!(
                r#"
                SELECT version as "version!", snapshot_data as "snapshot_data!"
                FROM event_snapshots
                WHERE aggregate_id = $1 AND aggregate_type = $2
                "#,
                aggregate_id,
                aggregate_type
            )
            .fetch_optional(&self.pool)
            .await?;
            
            if let Some(row) = row {
                Ok(Some((row.version as u32, row.snapshot_data)))
            } else {
                Ok(None)
            }
        }).await
    }
    
    /// 更新事件处理进度
    pub async fn update_processing_progress(
        &self,
        handler_name: &str,
        last_processed_event_id: EventId,
    ) -> DomainResult<()> {
        self.execute_query("update_processing_progress", async {
            sqlx::query!(
                r#"
                INSERT INTO event_processing_progress (handler_name, last_processed_event_id, last_processed_at)
                VALUES ($1, $2, NOW())
                ON CONFLICT (handler_name) DO UPDATE SET
                    last_processed_event_id = EXCLUDED.last_processed_event_id,
                    last_processed_at = EXCLUDED.last_processed_at,
                    updated_at = NOW()
                "#,
                handler_name,
                last_processed_event_id.value().to_string().parse::<i64>().unwrap_or(0)
            )
            .execute(&self.pool)
            .await?;
            
            Ok(())
        }).await
    }
    
    /// 获取事件处理进度
    pub async fn get_processing_progress(&self, handler_name: &str) -> DomainResult<Option<EventId>> {
        self.execute_query("get_processing_progress", async {
            let row = sqlx::query!(
                "SELECT last_processed_event_id FROM event_processing_progress WHERE handler_name = $1",
                handler_name
            )
            .fetch_optional(&self.pool)
            .await?;
            
            if let Some(row) = row {
                let event_id = row.last_processed_event_id;
                Ok(Some(EventId::from_uuid(uuid::Uuid::parse_str(&event_id.to_string()).unwrap())))
            } else {
                Ok(None)
            }
        }).await
    }
    
    /// 获取事件统计信息
    pub async fn get_event_statistics(&self) -> DomainResult<EventStatistics> {
        self.execute_query("get_event_statistics", async {
            let stats = sqlx::query!(
                r#"
                SELECT 
                    COUNT(*) as "total_events!",
                    COUNT(DISTINCT aggregate_id) as "unique_aggregates!",
                    COUNT(DISTINCT event_type) as "unique_event_types!",
                    MIN(occurred_at) as "first_event_at",
                    MAX(occurred_at) as "last_event_at"
                FROM events
                "#
            )
            .fetch_one(&self.pool)
            .await?;
            
            Ok(EventStatistics {
                total_events: stats.total_events as u64,
                unique_aggregates: stats.unique_aggregates as u64,
                unique_event_types: stats.unique_event_types as u64,
                first_event_at: stats.first_event_at,
                last_event_at: stats.last_event_at,
            })
        }).await
    }
}

/// 事件统计信息
#[derive(Debug, Clone)]
pub struct EventStatistics {
    pub total_events: u64,
    pub unique_aggregates: u64,
    pub unique_event_types: u64,
    pub first_event_at: Option<DateTime<Utc>>,
    pub last_event_at: Option<DateTime<Utc>>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_event_statistics_creation() {
        let stats = EventStatistics {
            total_events: 1000,
            unique_aggregates: 50,
            unique_event_types: 10,
            first_event_at: Some(chrono::Utc::now()),
            last_event_at: Some(chrono::Utc::now()),
        };
        
        assert_eq!(stats.total_events, 1000);
        assert_eq!(stats.unique_aggregates, 50);
        assert_eq!(stats.unique_event_types, 10);
    }
}

