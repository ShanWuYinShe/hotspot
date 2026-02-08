use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor};

/// Feed反馈Row - 严格按照数据库 schema
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct FeedFeedbackRow {
    pub feedback_id: Uuid,
    pub user_id: i64,
    pub content_type: String,  // moment, user, group
    pub content_id: String,
    pub action: String,  // like, dislike, click, view
    pub reason: Option<String>,
    pub created_at: DateTime<Utc>,
}

/// Feed项Row - 用于feed流展示
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct FeedItemRow {
    pub item_id: Uuid,  // Internal feed item ID
    pub user_id: i64,
    pub content_type: String,
    pub content_id: String,
    pub action: String,
    pub score: f64,  // Feed ranking score
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct FeedRepository {
    pool: PgPool,
}

impl FeedRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 记录反馈
    pub async fn record_feedback(
        &self,
        user_id: i64,
        content_type: String,
        content_id: String,
        action: String,
        reason: Option<String>,
    ) -> Result<FeedFeedbackRow> {
        let row = sqlx::query_as!(
            FeedFeedbackRow,
            r#"
            INSERT INTO feed_feedback (user_id, content_type, content_id, action, reason)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING feedback_id as "feedback_id!", user_id as "user_id!",
                      content_type as "content_type!", content_id as "content_id!",
                      action as "action!", reason, created_at as "created_at!"
            "#,
            user_id,
            content_type,
            content_id,
            action,
            reason
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 检查用户是否对内容执行过操作
    pub async fn has_action(
        &self,
        user_id: i64,
        content_id: &str,
        action: &str,
    ) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM feed_feedback
            WHERE user_id = $1 AND content_id = $2 AND action = $3
            ) as "exists!"
            "#,
            user_id,
            content_id,
            action
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 批量获取用户对内容的操作
    pub async fn batch_check_actions(
        &self,
        user_id: i64,
        content_ids: &[String],
        action: &str,
    ) -> Result<std::collections::HashMap<String, bool>> {
        if content_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT DISTINCT content_id as "content_id!"
            FROM feed_feedback
            WHERE user_id = $1 AND content_id = ANY($2) AND action = $3
            "#,
            user_id,
            content_ids,
            action
        )
        .fetch_all(&self.pool)
        .await?;

        let acted_set: std::collections::HashSet<String> = rows.into_iter().map(|r| r.content_id).collect();
        
        let mut result = std::collections::HashMap::new();
        for content_id in content_ids {
            result.insert(content_id.clone(), acted_set.contains(content_id));
        }

        Ok(result)
    }

    /// 获取用户的行为历史
    pub async fn get_user_history(
        &self,
        user_id: i64,
        content_type: &str,
        action: &str,
        limit: i64,
    ) -> Result<Vec<String>> {
        let rows = sqlx::query!(
            r#"
            SELECT content_id as "content_id!"
            FROM feed_feedback
            WHERE user_id = $1 AND content_type = $2 AND action = $3
            ORDER BY created_at DESC
            LIMIT $4
            "#,
            user_id,
            content_type,
            action,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| r.content_id).collect())
    }

    /// 统计用户对某内容的操作次数
    pub async fn count_actions(
        &self,
        user_id: i64,
        content_id: &str,
        action: &str,
    ) -> Result<i64> {
        let result = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM feed_feedback
            WHERE user_id = $1 AND content_id = $2 AND action = $3
            "#,
            user_id,
            content_id,
            action
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.count)
    }

    /// 删除反馈记录
    pub async fn delete_feedback(&self, feedback_id: Uuid) -> Result<()> {
        sqlx::query!(
            "DELETE FROM feed_feedback WHERE feedback_id = $1",
            feedback_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 清除用户对某内容的操作
    pub async fn clear_actions(
        &self,
        user_id: i64,
        content_id: &str,
        action: &str,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM feed_feedback
            WHERE user_id = $1 AND content_id = $2 AND action = $3
            "#,
            user_id,
            content_id,
            action
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}
