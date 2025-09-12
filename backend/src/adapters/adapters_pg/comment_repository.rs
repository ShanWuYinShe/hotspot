/// PostgreSQL评论仓储实现
/// 
/// 功能职责：
/// - 实现评论的数据库CRUD操作
/// - 提供评论查询和分页功能
/// - 管理评论回复的层级关系
/// - 支持评论统计和计数功能
use async_trait::async_trait;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    domain::{DomainResult, DomainError, Comment},
    ports::repository_ports::CommentRepository,
};
use super::query_optimizer::PgQueryOptimizer;

/// 评论数据库行结构
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CommentRow {
    pub comment_id: String,
    pub moment_id: i64,
    pub user_id: i64,
    pub content: String,
    pub parent_comment_id: Option<i64>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// PostgreSQL评论仓储
#[derive(Clone)]
pub struct PgCommentRepository {
    pool: PgPool,
    query_optimizer: PgQueryOptimizer,
}

impl PgCommentRepository {
    /// 创建新的评论仓储实例
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
        
        // 记录性能指标
        if let Some(ref perf_metrics) = metrics {
            if perf_metrics.execution_time_ms > 100 {
                tracing::warn!(
                    operation = operation_name,
                    duration_ms = perf_metrics.execution_time_ms,
                    query_type = perf_metrics.query_type,
                    used_index = perf_metrics.used_index,
                    repository = "comment_repository",
                    "慢查询检测"
                );
            } else {
                tracing::debug!(
                    operation = operation_name,
                    duration_ms = perf_metrics.execution_time_ms,
                    query_type = perf_metrics.query_type,
                    used_index = perf_metrics.used_index,
                    repository = "comment_repository",
                    "查询完成"
                );
            }
        }
        
        Ok(result)
    }

    /// 将数据库行转换为领域模型
    fn row_to_comment(row: CommentRow) -> Comment {
        Comment {
            comment_id: Uuid::parse_str(&row.comment_id).unwrap_or_else(|_| Uuid::new_v4()),
            moment_id: Uuid::parse_str(&row.moment_id.to_string()).unwrap_or_else(|_| Uuid::new_v4()),
            user_id: row.user_id,
            content: row.content,
            created_at: row.created_at,
            updated_at: row.updated_at,
        }
    }
}

#[async_trait]
impl CommentRepository for PgCommentRepository {
    /// 创建评论
    async fn create_comment(
        &self,
        moment_id: &str,
        user_id: i64,
        content: &str,
        parent_comment_id: Option<&str>,
    ) -> DomainResult<Comment> {
        self.execute_query_with_monitoring("create_comment", async {
            // moment_id 现在是 BIGINT，需要解析为 i64
            let moment_id_i64 = moment_id.parse::<i64>()
                .map_err(|_| DomainError::BusinessRuleViolation {
                    rule: "无效的生活瞬间ID格式".to_string(),
                })?;
            
            let parent_id_i64 = if let Some(parent_id) = parent_comment_id {
                Some(parent_id.parse::<i64>()
                    .map_err(|_| DomainError::BusinessRuleViolation {
                    rule: "无效的父评论ID格式".to_string(),
                })?)
            } else {
                None
            };

            let now = Utc::now();

            let row = sqlx::query_as!(
                CommentRow,
                r#"
                INSERT INTO comments (moment_id, user_id, content, parent_comment_id, created_at, updated_at)
                VALUES ($1, $2, $3, $4, $5, $6)
                RETURNING comment_id as "comment_id!", moment_id as "moment_id!", user_id as "user_id!", content as "content!", parent_comment_id, created_at as "created_at!", updated_at as "updated_at!"
                "#,
                moment_id_i64,
                user_id,
                content,
                parent_id_i64,
                now,
                now
            )
            .fetch_one(&self.pool)
            .await
            .map_err(|e| {
                tracing::error!("创建评论失败: {:?}", e);
                DomainError::BusinessRuleViolation {
                    rule: "创建评论失败".to_string(),
                }
            })?;

            Ok(Self::row_to_comment(row))
        }).await
    }

    /// 按ID查找评论
    async fn find_comment_by_id(&self, comment_id: &str) -> DomainResult<Option<Comment>> {
        self.execute_query_with_monitoring("find_comment_by_id", async {
            let row = sqlx::query_as!(
                CommentRow,
                r#"
                SELECT comment_id as "comment_id!", moment_id as "moment_id!", user_id as "user_id!", content as "content!", parent_comment_id, created_at as "created_at!", updated_at as "updated_at!"
                FROM comments
                WHERE comment_id = $1
                "#,
                comment_id
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| {
                tracing::error!("查询评论失败: {:?}", e);
                DomainError::BusinessRuleViolation {
                    rule: "查询评论失败".to_string(),
                }
            })?;

            Ok(row.map(Self::row_to_comment))
        }).await
    }

    /// 获取生活瞬间的评论列表
    async fn find_comments_by_moment(
        &self,
        moment_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Comment>> {
        let moment_id_i64 = moment_id.parse::<i64>()
            .map_err(|_| DomainError::BusinessRuleViolation {
                rule: "无效的生活瞬间ID格式".to_string(),
            })?;

        let rows = sqlx::query!(
            r#"
            SELECT 
                c.comment_id as "comment_id!",
                c.moment_id as "moment_id!",
                c.user_id as "user_id!",
                c.content as "content!",
                c.created_at as "created_at!",
                c.updated_at as "updated_at!",
                u.nickname as "user_nickname!",
                u.avatar_url as user_avatar
            FROM comments c
            JOIN users u ON c.user_id = u.id
            WHERE c.moment_id = $1 
            AND ($2 = 0 OR c.user_id > $2)
            ORDER BY c.created_at ASC
            LIMIT $3
            "#,
            moment_id_i64,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            tracing::error!("查询评论列表失败: {:?}", e);
            DomainError::BusinessRuleViolation {
                rule: "查询评论列表失败".to_string(),
            }
        })?;

        let comments = rows
            .into_iter()
            .map(|row| Comment {
                comment_id: uuid::Uuid::parse_str(&row.comment_id).unwrap_or_default(),
                moment_id: uuid::Uuid::from_u128(row.moment_id as u128),
                user_id: row.user_id,
                content: row.content,
                created_at: row.created_at,
                updated_at: row.updated_at,
            })
            .collect();

        Ok(comments)
    }

    /// 获取用户的评论历史
    async fn find_comments_by_user(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Comment>> {
        let rows = sqlx::query!(
            r#"
            SELECT 
                c.comment_id,
                c.moment_id,
                c.user_id,
                c.content,
                c.created_at,
                c.updated_at,
                u.nickname as user_nickname,
                u.avatar_url as user_avatar
            FROM comments c
            JOIN users u ON c.user_id = u.id
            WHERE c.user_id = $1 
            AND ($2 = 0 OR c.user_id > $2)
            ORDER BY c.created_at DESC
            LIMIT $3
            "#,
            user_id,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            tracing::error!("查询用户评论历史失败: {:?}", e);
            DomainError::BusinessRuleViolation {
                rule: "查询用户评论历史失败".to_string(),
            }
        })?;

        let comments = rows
            .into_iter()
            .map(|row| Comment {
                comment_id: uuid::Uuid::parse_str(&row.comment_id).unwrap_or_default(),
                moment_id: uuid::Uuid::from_u128(row.moment_id as u128),
                user_id: row.user_id,
                content: row.content,
                created_at: row.created_at,
                updated_at: row.updated_at,
            })
            .collect();

        Ok(comments)
    }

    /// 删除评论
    async fn delete_comment(&self, comment_id: &str, user_id: i64) -> DomainResult<()> {
        let result = sqlx::query!(
            r#"
            DELETE FROM comments
            WHERE comment_id = $1 AND user_id = $2
            "#,
            comment_id,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            tracing::error!("删除评论失败: {:?}", e);
            DomainError::BusinessRuleViolation {
                rule: "删除评论失败".to_string(),
            }
        })?;

        if result.rows_affected() == 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "评论不存在或无权限删除".to_string(),
            });
        }

        Ok(())
    }

    /// 更新评论内容
    async fn update_comment(
        &self,
        comment_id: &str,
        user_id: i64,
        content: &str,
    ) -> DomainResult<Comment> {
        let row = sqlx::query_as!(
            CommentRow,
            r#"
            UPDATE comments
            SET content = $3, updated_at = $4
            WHERE comment_id = $1 AND user_id = $2
            RETURNING comment_id as "comment_id!", moment_id as "moment_id!", user_id as "user_id!", content as "content!", parent_comment_id, created_at as "created_at!", updated_at as "updated_at!"
            "#,
            comment_id,
            user_id,
            content,
            Utc::now()
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| {
            tracing::error!("更新评论失败: {:?}", e);
            DomainError::BusinessRuleViolation {
                rule: "更新评论失败".to_string(),
            }
        })?;

        match row {
            Some(row) => Ok(Self::row_to_comment(row)),
            None => Err(DomainError::BusinessRuleViolation {
                rule: "评论不存在或无权限修改".to_string(),
            }),
        }
    }

    /// 获取评论回复列表
    async fn find_comment_replies(
        &self,
        parent_comment_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Comment>> {
        let parent_uuid = Uuid::parse_str(parent_comment_id)
            .map_err(|_| DomainError::BusinessRuleViolation {
                rule: "无效的父评论ID格式".to_string(),
            })?;

        let rows = sqlx::query!(
            r#"
            SELECT 
                c.comment_id,
                c.moment_id,
                c.user_id,
                c.content,
                c.created_at,
                c.updated_at,
                u.nickname as user_nickname,
                u.avatar_url as user_avatar
            FROM comments c
            JOIN users u ON c.user_id = u.id
            WHERE c.parent_comment_id = $1 
            AND ($2 = 0 OR c.user_id > $2)
            ORDER BY c.created_at ASC
            LIMIT $3
            "#,
            parent_comment_id.parse::<i64>().unwrap_or(0),
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            tracing::error!("查询评论回复失败: {:?}", e);
            DomainError::BusinessRuleViolation {
                rule: "查询评论回复失败".to_string(),
            }
        })?;

        let comments = rows
            .into_iter()
            .map(|row| Comment {
                comment_id: uuid::Uuid::parse_str(&row.comment_id).unwrap_or_default(),
                moment_id: uuid::Uuid::from_u128(row.moment_id as u128),
                user_id: row.user_id,
                content: row.content,
                created_at: row.created_at,
                updated_at: row.updated_at,
            })
            .collect();

        Ok(comments)
    }

    /// 获取生活瞬间评论数
    async fn get_moment_comments_count(&self, moment_id: &str) -> DomainResult<i64> {
        let moment_id_i64 = moment_id.parse::<i64>()
            .map_err(|_| DomainError::BusinessRuleViolation {
                rule: "无效的生活瞬间ID格式".to_string(),
            })?;

        let count = sqlx::query_scalar!(
            r#"
            SELECT COUNT(*)::bigint as "count!"
            FROM comments
            WHERE moment_id = $1
            "#,
            moment_id_i64
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            tracing::error!("查询评论数量失败: {:?}", e);
            DomainError::BusinessRuleViolation {
                rule: "查询评论数量失败".to_string(),
            }
        })?;

        Ok(count)
    }
}

