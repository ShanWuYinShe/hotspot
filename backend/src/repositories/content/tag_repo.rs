//! 标签仓库
//!
//! 适配新 Schema: tags(tag_id UUID, name, usage_count)

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use std::collections::HashMap;
use uuid::Uuid;

use crate::error::{AppError, Result};

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct TagRow {
    pub tag_id: Uuid,
    pub name: String,
    pub usage_count: Option<i32>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, FromRow)]
pub struct AdminTagRow {
    pub tag_id: Uuid,
    pub name: String,
    pub usage_count: i32,
    pub created_at: DateTime<Utc>,
    pub last_used_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Copy)]
pub enum AdminTagSortOrder {
    UsageCountDesc,
    UsageCountAsc,
    CreatedAtDesc,
    LastUsedAtDesc,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AdminTagCursor {
    UsageCountDesc {
        usage_count: i32,
        tag_id: Uuid,
    },
    UsageCountAsc {
        usage_count: i32,
        tag_id: Uuid,
    },
    CreatedAtDesc {
        created_at: DateTime<Utc>,
        tag_id: Uuid,
    },
    LastUsedAtDesc {
        last_used_at: DateTime<Utc>,
        tag_id: Uuid,
    },
}

#[derive(Debug, Clone)]
pub struct TagRepository {
    pool: PgPool,
}

impl TagRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 批量获取或创建（单次 INSERT + SELECT，保持输入顺序）
    pub async fn find_or_create_many(&self, names: &[String]) -> Result<Vec<Uuid>> {
        if names.is_empty() {
            return Ok(vec![]);
        }

        sqlx::query!(
            r#"
            INSERT INTO tags (name)
            SELECT unnest($1::text[])
            ON CONFLICT (name) DO NOTHING
            "#,
            names
        )
        .execute(&self.pool)
        .await?;

        let rows = sqlx::query!(
            r#"
            SELECT t.tag_id
            FROM unnest($1::text[]) WITH ORDINALITY AS u(name, ord)
            JOIN tags t ON t.name = u.name
            ORDER BY u.ord
            "#,
            names
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| r.tag_id).collect())
    }

    /// 设置用户标签
    pub async fn set_user_tags(&self, user_id: i64, tag_names: &[String]) -> Result<()> {
        let tag_ids = self.find_or_create_many(tag_names).await?;

        let mut tx = self.pool.begin().await?;

        sqlx::query!("DELETE FROM user_tags WHERE user_id = $1", user_id)
            .execute(&mut *tx)
            .await?;

        if !tag_ids.is_empty() {
            sqlx::query!(
                r#"
                INSERT INTO user_tags (user_id, tag_id)
                SELECT $1, unnest($2::uuid[])
                ON CONFLICT DO NOTHING
                "#,
                user_id,
                &tag_ids
            )
            .execute(&mut *tx)
            .await?;
        }

        tx.commit().await?;
        Ok(())
    }

    /// 设置瞬间标签
    pub async fn set_moment_tags(&self, moment_id: Uuid, tag_names: &[String]) -> Result<()> {
        let tag_ids = self.find_or_create_many(tag_names).await?;

        let mut tx = self.pool.begin().await?;

        sqlx::query!("DELETE FROM moment_tags WHERE moment_id = $1", moment_id)
            .execute(&mut *tx)
            .await?;

        if !tag_ids.is_empty() {
            sqlx::query!(
                r#"
                INSERT INTO moment_tags (moment_id, tag_id)
                SELECT $1, unnest($2::uuid[])
                ON CONFLICT DO NOTHING
                "#,
                moment_id,
                &tag_ids
            )
            .execute(&mut *tx)
            .await?;
        }

        tx.commit().await?;
        Ok(())
    }

    /// 获取用户标签名列表
    pub async fn get_user_tags(&self, user_id: i64) -> Result<Vec<String>> {
        let rows = sqlx::query_scalar!(
            r#"
            SELECT t.name
            FROM tags t
            JOIN user_tags ut ON t.tag_id = ut.tag_id
            WHERE ut.user_id = $1
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

    /// 批量获取瞬间标签
    pub async fn get_moment_tags_batch(
        &self,
        moment_ids: &[Uuid],
    ) -> Result<HashMap<Uuid, Vec<String>>> {
        if moment_ids.is_empty() {
            return Ok(HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT mt.moment_id as "moment_id!", t.name as "name!"
            FROM moment_tags mt
            JOIN tags t ON t.tag_id = mt.tag_id
            WHERE mt.moment_id = ANY($1)
            ORDER BY mt.moment_id ASC, t.name ASC
            "#,
            moment_ids
        )
        .fetch_all(&self.pool)
        .await?;

        let mut map: HashMap<Uuid, Vec<String>> = HashMap::new();
        for row in rows {
            map.entry(row.moment_id).or_default().push(row.name);
        }

        Ok(map)
    }

    /// 获取热门标签
    pub async fn get_trending(&self, limit: i64) -> Result<Vec<TagRow>> {
        let rows = sqlx::query_as!(
            TagRow,
            r#"
            SELECT tag_id, name, usage_count, created_at, updated_at
            FROM tags
            ORDER BY usage_count DESC
            LIMIT $1
            "#,
            limit
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

    /// 标签前缀建议
    pub async fn suggest(&self, prefix: &str, limit: i64) -> Result<Vec<TagRow>> {
        let pattern = format!("{}%", prefix);
        let rows = sqlx::query_as!(
            TagRow,
            r#"
            SELECT tag_id, name, usage_count, created_at, updated_at
            FROM tags
            WHERE name ILIKE $1
            ORDER BY usage_count DESC NULLS LAST, name ASC
            LIMIT $2
            "#,
            pattern,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 删除标签（管理员）
    pub async fn delete(&self, name: &str) -> Result<bool> {
        let result = sqlx::query!("DELETE FROM tags WHERE name = $1", name)
            .execute(&self.pool)
            .await?;

        Ok(result.rows_affected() > 0)
    }

    /// 设置群组标签
    pub async fn set_group_tags(&self, group_id: i64, tag_names: &[String]) -> Result<()> {
        let tag_ids = self.find_or_create_many(tag_names).await?;
        let mut tx = self.pool.begin().await?;

        sqlx::query!("DELETE FROM group_tags WHERE group_id = $1", group_id)
            .execute(&mut *tx)
            .await?;

        if !tag_ids.is_empty() {
            sqlx::query!(
                r#"
                INSERT INTO group_tags (group_id, tag_id)
                SELECT $1, unnest($2::uuid[])
                ON CONFLICT DO NOTHING
                "#,
                group_id,
                &tag_ids
            )
            .execute(&mut *tx)
            .await?;
        }

        tx.commit().await?;
        Ok(())
    }

    /// 获取群组标签名列表
    pub async fn get_group_tags(&self, group_id: i64) -> Result<Vec<String>> {
        let rows = sqlx::query_scalar!(
            r#"
            SELECT t.name
            FROM tags t
            JOIN group_tags gt ON t.tag_id = gt.tag_id
            WHERE gt.group_id = $1
            "#,
            group_id
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

    /// 批量获取多个群组的标签
    pub async fn get_group_tags_batch(
        &self,
        group_ids: &[i64],
    ) -> Result<HashMap<i64, Vec<String>>> {
        if group_ids.is_empty() {
            return Ok(HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT gt.group_id as "group_id!", t.name as "name!"
            FROM group_tags gt
            JOIN tags t ON gt.tag_id = t.tag_id
            WHERE gt.group_id = ANY($1)
            ORDER BY gt.group_id ASC, t.name ASC
            "#,
            group_ids
        )
        .fetch_all(&self.pool)
        .await?;

        let mut map: HashMap<i64, Vec<String>> = HashMap::new();
        for row in rows {
            map.entry(row.group_id).or_default().push(row.name);
        }

        Ok(map)
    }

    pub async fn admin_search(
        &self,
        keyword: Option<&str>,
        sort_by: AdminTagSortOrder,
        limit: i64,
        cursor: Option<AdminTagCursor>,
    ) -> Result<Vec<AdminTagRow>> {
        let keyword_pattern = keyword.map(|k| format!("%{}%", k));
        let keyword_filter = keyword_pattern.as_deref();
        let rows = match sort_by {
            AdminTagSortOrder::UsageCountDesc => match cursor {
                Some(AdminTagCursor::UsageCountDesc {
                    usage_count,
                    tag_id,
                }) => {
                    sqlx::query_as!(
                        AdminTagRow,
                        r#"
                        SELECT tag_id, name, usage_count as "usage_count!",
                               created_at, updated_at as "last_used_at!"
                        FROM tags
                        WHERE ($1::text IS NULL OR name ILIKE $1)
                          AND (usage_count < $2 OR (usage_count = $2 AND tag_id < $3))
                        ORDER BY usage_count DESC, tag_id DESC
                        LIMIT $4
                        "#,
                        keyword_filter,
                        usage_count,
                        tag_id,
                        limit
                    )
                    .fetch_all(&self.pool)
                    .await?
                }
                Some(_) => {
                    return Err(AppError::CursorInvalidOrExpired(
                        "无效的标签游标".to_string(),
                    ))
                }
                None => {
                    sqlx::query_as!(
                        AdminTagRow,
                        r#"
                        SELECT tag_id, name, usage_count as "usage_count!",
                               created_at, updated_at as "last_used_at!"
                        FROM tags
                        WHERE ($1::text IS NULL OR name ILIKE $1)
                        ORDER BY usage_count DESC, tag_id DESC
                        LIMIT $2
                        "#,
                        keyword_filter,
                        limit
                    )
                    .fetch_all(&self.pool)
                    .await?
                }
            },
            AdminTagSortOrder::UsageCountAsc => match cursor {
                Some(AdminTagCursor::UsageCountAsc {
                    usage_count,
                    tag_id,
                }) => {
                    sqlx::query_as!(
                        AdminTagRow,
                        r#"
                        SELECT tag_id, name, usage_count as "usage_count!",
                               created_at, updated_at as "last_used_at!"
                        FROM tags
                        WHERE ($1::text IS NULL OR name ILIKE $1)
                          AND (usage_count > $2 OR (usage_count = $2 AND tag_id > $3))
                        ORDER BY usage_count ASC, tag_id ASC
                        LIMIT $4
                        "#,
                        keyword_filter,
                        usage_count,
                        tag_id,
                        limit
                    )
                    .fetch_all(&self.pool)
                    .await?
                }
                Some(_) => {
                    return Err(AppError::CursorInvalidOrExpired(
                        "无效的标签游标".to_string(),
                    ))
                }
                None => {
                    sqlx::query_as!(
                        AdminTagRow,
                        r#"
                        SELECT tag_id, name, usage_count as "usage_count!",
                               created_at, updated_at as "last_used_at!"
                        FROM tags
                        WHERE ($1::text IS NULL OR name ILIKE $1)
                        ORDER BY usage_count ASC, tag_id ASC
                        LIMIT $2
                        "#,
                        keyword_filter,
                        limit
                    )
                    .fetch_all(&self.pool)
                    .await?
                }
            },
            AdminTagSortOrder::CreatedAtDesc => match cursor {
                Some(AdminTagCursor::CreatedAtDesc { created_at, tag_id }) => {
                    sqlx::query_as!(
                        AdminTagRow,
                        r#"
                        SELECT tag_id, name, usage_count as "usage_count!",
                               created_at, updated_at as "last_used_at!"
                        FROM tags
                        WHERE ($1::text IS NULL OR name ILIKE $1)
                          AND (created_at < $2 OR (created_at = $2 AND tag_id < $3))
                        ORDER BY created_at DESC, tag_id DESC
                        LIMIT $4
                        "#,
                        keyword_filter,
                        created_at,
                        tag_id,
                        limit
                    )
                    .fetch_all(&self.pool)
                    .await?
                }
                Some(_) => {
                    return Err(AppError::CursorInvalidOrExpired(
                        "无效的标签游标".to_string(),
                    ))
                }
                None => {
                    sqlx::query_as!(
                        AdminTagRow,
                        r#"
                        SELECT tag_id, name, usage_count as "usage_count!",
                               created_at, updated_at as "last_used_at!"
                        FROM tags
                        WHERE ($1::text IS NULL OR name ILIKE $1)
                        ORDER BY created_at DESC, tag_id DESC
                        LIMIT $2
                        "#,
                        keyword_filter,
                        limit
                    )
                    .fetch_all(&self.pool)
                    .await?
                }
            },
            AdminTagSortOrder::LastUsedAtDesc => match cursor {
                Some(AdminTagCursor::LastUsedAtDesc {
                    last_used_at,
                    tag_id,
                }) => {
                    sqlx::query_as!(
                        AdminTagRow,
                        r#"
                        SELECT tag_id, name, usage_count as "usage_count!",
                               created_at, updated_at as "last_used_at!"
                        FROM tags
                        WHERE ($1::text IS NULL OR name ILIKE $1)
                          AND (updated_at < $2 OR (updated_at = $2 AND tag_id < $3))
                        ORDER BY updated_at DESC, tag_id DESC
                        LIMIT $4
                        "#,
                        keyword_filter,
                        last_used_at,
                        tag_id,
                        limit
                    )
                    .fetch_all(&self.pool)
                    .await?
                }
                Some(_) => {
                    return Err(AppError::CursorInvalidOrExpired(
                        "无效的标签游标".to_string(),
                    ))
                }
                None => {
                    sqlx::query_as!(
                        AdminTagRow,
                        r#"
                        SELECT tag_id, name, usage_count as "usage_count!",
                               created_at, updated_at as "last_used_at!"
                        FROM tags
                        WHERE ($1::text IS NULL OR name ILIKE $1)
                        ORDER BY updated_at DESC, tag_id DESC
                        LIMIT $2
                        "#,
                        keyword_filter,
                        limit
                    )
                    .fetch_all(&self.pool)
                    .await?
                }
            },
        };

        Ok(rows)
    }
}
