//! 标签仓库 - 处理 tags 表及其关联表的数据操作

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::PgPool;
use uuid::Uuid;

use crate::error::Result;

use crate::utils::pagination::{decode_cursor, encode_cursor, IdCursor};

/// 标签行
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TagRow {
    pub id: i64,
    pub tag_id: Uuid,
    pub name: String,
    pub usage_count: i32,
    pub created_at: DateTime<Utc>,
}

/// 标签建议行
#[derive(Debug, Clone)]
pub struct TagSuggestRow {
    pub name: String,
    pub usage_count: i32,
}

#[derive(Debug, Clone)]
pub struct TagRepository {
    pool: PgPool,
}

impl TagRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    // ==================== 标签主表操作 ====================

    /// 批量查找或创建标签（返回标签ID列表）
    pub async fn find_or_create_many(&self, names: &[String]) -> Result<Vec<i64>> {
        if names.is_empty() {
            return Ok(vec![]);
        }

        // 真正的批量 upsert：一次 SQL 完成「创建缺失 + 查询现有」，并按输入顺序返回 tag_id
        let rows = sqlx::query!(
            r#"
            WITH input AS (
                SELECT *
                FROM unnest($1::text[]) WITH ORDINALITY AS t(name, ord)
            ),
            ins AS (
                INSERT INTO tags (name)
                SELECT DISTINCT name
                FROM input
                ON CONFLICT (name) DO NOTHING
            )
            SELECT tg.id, input.ord
            FROM input
            INNER JOIN tags tg ON tg.name = input.name
            ORDER BY input.ord ASC
            "#,
            names
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| r.id).collect())
    }

    /// 获取热门标签（基于指定时间段内的使用情况）
    pub async fn get_trending(&self, limit: i64, since: DateTime<Utc>) -> Result<Vec<TagRow>> {
        // 统计指定时间段内被使用的标签（基于 moment_tags 的创建时间）
        let rows = sqlx::query_as!(
            TagRow,
            r#"
            SELECT t.id, t.tag_id, t.name, t.usage_count, t.created_at
            FROM tags t
            WHERE EXISTS (
                SELECT 1 FROM moment_tags mt 
                WHERE mt.tag_id = t.id AND mt.created_at >= $2
            )
            ORDER BY (
                SELECT COUNT(*) FROM moment_tags mt 
                WHERE mt.tag_id = t.id AND mt.created_at >= $2
            ) DESC
            LIMIT $1
            "#,
            limit,
            since
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 标签搜索建议（前缀匹配）
    pub async fn suggest(&self, prefix: &str, limit: i64) -> Result<Vec<TagSuggestRow>> {
        let pattern = format!("{}%", prefix);
        let rows = sqlx::query!(
            r#"
            SELECT name, usage_count
            FROM tags
            WHERE name ILIKE $1
            ORDER BY usage_count DESC
            LIMIT $2
            "#,
            pattern,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows
            .into_iter()
            .map(|r| TagSuggestRow {
                name: r.name,
                usage_count: r.usage_count,
            })
            .collect())
    }

    /// 删除标签（管理员）
    pub async fn delete(&self, tag_name: &str) -> Result<bool> {
        let result = sqlx::query!(
            "DELETE FROM tags WHERE name = $1",
            tag_name
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() > 0)
    }

    // ==================== 用户标签操作 ====================

    /// 设置用户标签（替换所有现有标签）
    pub async fn set_user_tags(&self, user_id: i64, tag_names: &[String]) -> Result<()> {
        // 先删除所有现有标签
        sqlx::query!("DELETE FROM user_tags WHERE user_id = $1", user_id)
            .execute(&self.pool)
            .await?;

        // 添加新标签
        if !tag_names.is_empty() {
            let tag_ids = self.find_or_create_many(tag_names).await?;
            sqlx::query!(
                r#"
                INSERT INTO user_tags (user_id, tag_id)
                SELECT $1, x.tag_id
                FROM unnest($2::bigint[]) AS x(tag_id)
                ON CONFLICT DO NOTHING
                "#,
                user_id,
                &tag_ids
            )
            .execute(&self.pool)
            .await?;
        }

        Ok(())
    }

    /// 获取用户标签
    pub async fn get_user_tags(&self, user_id: i64) -> Result<Vec<String>> {
        let rows = sqlx::query!(
            r#"
            SELECT t.name
            FROM tags t
            INNER JOIN user_tags ut ON t.id = ut.tag_id
            WHERE ut.user_id = $1
            ORDER BY ut.created_at
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| r.name).collect())
    }

    /// 按标签搜索用户（返回用户ID列表）
    pub async fn search_users_by_tag(
        &self,
        tag_name: &str,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<i64>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query!(
            r#"
            SELECT ut.user_id
            FROM user_tags ut
            INNER JOIN tags t ON ut.tag_id = t.id
            WHERE t.name = $1 AND ut.user_id < $2
            ORDER BY ut.user_id DESC
            LIMIT $3
            "#,
            tag_name,
            last_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        let user_ids: Vec<i64> = rows.into_iter().map(|r| r.user_id).collect();

        let next_cursor = user_ids
            .last()
            .map(|last_id| encode_cursor(last_id))
            .transpose()?;

        Ok((user_ids, next_cursor))
    }

    // ==================== 群组标签操作 ====================

    /// 设置群组标签（替换所有现有标签）
    pub async fn set_group_tags(&self, group_chat_id: Uuid, tag_names: &[String]) -> Result<()> {
        // 先删除所有现有标签
        sqlx::query!(
            "DELETE FROM group_tags WHERE group_chat_id = $1",
            group_chat_id
        )
        .execute(&self.pool)
        .await?;

        // 添加新标签
        if !tag_names.is_empty() {
            let tag_ids = self.find_or_create_many(tag_names).await?;
            sqlx::query!(
                r#"
                INSERT INTO group_tags (group_chat_id, tag_id)
                SELECT $1, x.tag_id
                FROM unnest($2::bigint[]) AS x(tag_id)
                ON CONFLICT DO NOTHING
                "#,
                group_chat_id,
                &tag_ids
            )
            .execute(&self.pool)
            .await?;
        }

        Ok(())
    }

    /// 获取群组标签
    pub async fn get_group_tags(&self, group_chat_id: Uuid) -> Result<Vec<String>> {
        let rows = sqlx::query!(
            r#"
            SELECT t.name
            FROM tags t
            INNER JOIN group_tags gt ON t.id = gt.tag_id
            WHERE gt.group_chat_id = $1
            ORDER BY gt.created_at
            "#,
            group_chat_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| r.name).collect())
    }

    // ==================== 瞬间标签操作 ====================

    /// 设置瞬间标签（替换所有现有标签）
    pub async fn set_moment_tags(&self, moment_id: Uuid, tag_names: &[String]) -> Result<()> {
        // 先删除所有现有标签
        sqlx::query!("DELETE FROM moment_tags WHERE moment_id = $1", moment_id)
            .execute(&self.pool)
            .await?;

        // 添加新标签
        if !tag_names.is_empty() {
            let tag_ids = self.find_or_create_many(tag_names).await?;
            sqlx::query!(
                r#"
                INSERT INTO moment_tags (moment_id, tag_id)
                SELECT $1, x.tag_id
                FROM unnest($2::bigint[]) AS x(tag_id)
                ON CONFLICT DO NOTHING
                "#,
                moment_id,
                &tag_ids
            )
            .execute(&self.pool)
            .await?;
        }

        Ok(())
    }

    /// 获取瞬间标签
    pub async fn get_moment_tags(&self, moment_id: Uuid) -> Result<Vec<String>> {
        let rows = sqlx::query!(
            r#"
            SELECT t.name
            FROM tags t
            INNER JOIN moment_tags mt ON t.id = mt.tag_id
            WHERE mt.moment_id = $1
            ORDER BY mt.created_at
            "#,
            moment_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| r.name).collect())
    }

    /// 批量获取瞬间标签
    pub async fn get_moment_tags_batch(
        &self,
        moment_ids: &[Uuid],
    ) -> Result<std::collections::HashMap<Uuid, Vec<String>>> {
        if moment_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT mt.moment_id, t.name
            FROM moment_tags mt
            INNER JOIN tags t ON mt.tag_id = t.id
            WHERE mt.moment_id = ANY($1)
            ORDER BY mt.moment_id, mt.created_at
            "#,
            moment_ids
        )
        .fetch_all(&self.pool)
        .await?;

        let mut result: std::collections::HashMap<Uuid, Vec<String>> =
            std::collections::HashMap::new();
        for row in rows {
            result.entry(row.moment_id).or_default().push(row.name);
        }

        Ok(result)
    }

    /// 批量获取用户标签
    pub async fn get_user_tags_batch(
        &self,
        user_ids: &[i64],
    ) -> Result<std::collections::HashMap<i64, Vec<String>>> {
        if user_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT ut.user_id, t.name
            FROM user_tags ut
            INNER JOIN tags t ON ut.tag_id = t.id
            WHERE ut.user_id = ANY($1)
            ORDER BY ut.user_id, ut.created_at
            "#,
            user_ids
        )
        .fetch_all(&self.pool)
        .await?;

        let mut result: std::collections::HashMap<i64, Vec<String>> =
            std::collections::HashMap::new();
        for row in rows {
            result.entry(row.user_id).or_default().push(row.name);
        }

        Ok(result)
    }

    /// 批量获取群组标签
    pub async fn get_group_tags_batch(
        &self,
        group_ids: &[Uuid],
    ) -> Result<std::collections::HashMap<Uuid, Vec<String>>> {
        if group_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT gt.group_chat_id, t.name
            FROM group_tags gt
            INNER JOIN tags t ON gt.tag_id = t.id
            WHERE gt.group_chat_id = ANY($1)
            ORDER BY gt.group_chat_id, gt.created_at
            "#,
            group_ids
        )
        .fetch_all(&self.pool)
        .await?;

        let mut result: std::collections::HashMap<Uuid, Vec<String>> =
            std::collections::HashMap::new();
        for row in rows {
            result.entry(row.group_chat_id).or_default().push(row.name);
        }

        Ok(result)
    }
}
