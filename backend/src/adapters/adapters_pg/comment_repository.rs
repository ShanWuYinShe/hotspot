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
use sqlx::{PgPool, Row};
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
    fn row_to_comment(row: sqlx::postgres::PgRow) -> Comment {
        Comment {
            comment_id: row.get::<Uuid, _>("comment_id"),
            moment_id: row.get::<i64, _>("moment_id"),
            user_id: row.get("user_id"),
            content: row.get("content"),
            created_at: row.get("created_at"),
            updated_at: row.get("updated_at"),
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

            let row = sqlx::query(
                r#"
                INSERT INTO moment_comments (moment_id, user_id, content, parent_comment_id, created_at, updated_at)
                VALUES ($1, $2, $3, $4, $5, $6)
                RETURNING comment_id, moment_id, user_id, content, parent_comment_id, created_at, updated_at
                "#
            )
            .bind(moment_id_i64)
            .bind(user_id)
            .bind(content)
            .bind(parent_id_i64)
            .bind(now)
            .bind(now)
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
            let row = sqlx::query(
                r#"
                SELECT comment_id, moment_id, user_id, content, parent_comment_id, created_at, updated_at
                FROM moment_comments
                WHERE comment_id = $1
                "#
            )
            .bind(comment_id)
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
            FROM moment_comments c
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
                moment_id: row.moment_id,
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
                c.created_at as "created_at!",
                c.updated_at as "updated_at!",
                u.nickname as user_nickname,
                u.avatar_url as user_avatar
            FROM moment_comments c
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
                moment_id: row.moment_id,
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
            DELETE FROM moment_comments
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
        let row = sqlx::query(
            r#"
            UPDATE moment_comments
            SET content = $3, updated_at = $4
            WHERE comment_id = $1 AND user_id = $2
            RETURNING comment_id, moment_id, user_id, content, parent_comment_id, created_at, updated_at
            "#
        )
        .bind(comment_id)
        .bind(user_id)
        .bind(content)
        .bind(Utc::now())
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
        let _parent_uuid = Uuid::parse_str(parent_comment_id)
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
                c.created_at as "created_at!",
                c.updated_at as "updated_at!",
                u.nickname as user_nickname,
                u.avatar_url as user_avatar
            FROM moment_comments c
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
                moment_id: row.moment_id,
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
            FROM moment_comments
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

