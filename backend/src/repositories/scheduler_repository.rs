//! 调度器相关数据仓库实现

use chrono::{Utc, Duration};
use uuid::Uuid;
use tracing::error;

use super::{Repository, RepositoryError, RepositoryResult};

/// 定时任务相关查询接口
pub trait SchedulerRepository: Repository {
    fn find_inactive_groups(&self) -> impl std::future::Future<Output = RepositoryResult<Vec<(Uuid, String, chrono::DateTime<Utc>)>>> + Send;
}

/// PostgreSQL实现的调度器仓库
#[derive(Debug)]
pub struct PgSchedulerRepository {
    /// 数据库连接池
    pool: sqlx::PgPool,
}

impl PgSchedulerRepository {
    /// 创建新的PostgreSQL调度器仓库
    pub fn new(pool: sqlx::PgPool) -> Self {
        Self { pool }
    }
}

impl Repository for PgSchedulerRepository {
    fn name(&self) -> &str {
        "PgSchedulerRepository"
    }
}

impl SchedulerRepository for PgSchedulerRepository {
    fn find_inactive_groups(&self) -> impl std::future::Future<Output = RepositoryResult<Vec<(Uuid, String, chrono::DateTime<Utc>)>>> + Send {
        let pool = self.pool.clone();
        async move {
            // 查询标准:
            // 1. 群组在最近7天内有活动
            // 2. 群组有至少1个成员超过3天没活动
            let three_days_ago = Utc::now() - Duration::days(3);
            let seven_days_ago = Utc::now() - Duration::days(7);

            #[derive(Debug)]
            struct InactiveGroupRow {
                group_id: uuid::Uuid,
                name: String,
                last_active_at: chrono::DateTime<Utc>,
            }

            let rows = sqlx::query_as!(
                InactiveGroupRow,
                r#"
                SELECT DISTINCT g.group_id as "group_id!", g.name as "name!", g.last_active_at as "last_active_at!"
                FROM groups g
                JOIN group_members gm ON g.group_id = gm.group_id
                WHERE 
                    g.last_active_at > $1
                    AND EXISTS (
                        SELECT 1 
                        FROM group_members gm2 
                        WHERE gm2.group_id = g.group_id 
                        AND (gm2.last_active IS NULL OR gm2.last_active < $2)
                        AND gm2.notification_level != 'muted'
                    )
                LIMIT 50
                "#,
                seven_days_ago,
                three_days_ago
            )
            .fetch_all(&pool)
            .await
            .map_err(|e| {
                error!("查询需要发送摘要的群组失败: {:?}", e);
                RepositoryError::Database(e)
            })?;

            Ok(rows
                .into_iter()
                .map(|row| (row.group_id, row.name, row.last_active_at))
                .collect())
        }
    }
}