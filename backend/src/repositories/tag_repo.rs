//! 标签仓库
//!
//! 适配新 Schema: tags(tag_id UUID, name, usage_count)

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use std::collections::HashMap;
use uuid::Uuid;

use crate::error::Result;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct TagRow {
    pub tag_id: Uuid,
    pub name: String,
    pub usage_count: Option<i32>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct TagRepository {
    pool: PgPool,
}

impl TagRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 获取或创建标签 (返回 tag_id)
    pub async fn find_or_create(&self, name: &str) -> Result<Uuid> {
        let row = sqlx::query!(
            r#"
            INSERT INTO tags (name)
            VALUES ($1)
            ON CONFLICT (name) DO UPDATE SET updated_at = NOW()
            RETURNING tag_id
            "#,
            name
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row.tag_id)
    }

    /// 批量获取或创建
    pub async fn find_or_create_many(&self, names: &[String]) -> Result<Vec<Uuid>> {
        let mut ids = Vec::with_capacity(names.len());
        for name in names {
            ids.push(self.find_or_create(name).await?);
        }
        Ok(ids)
    }

    /// 设置用户标签
    pub async fn set_user_tags(&self, user_id: i64, tag_names: &[String]) -> Result<()> {
        let tag_ids = self.find_or_create_many(tag_names).await?;
        
        let mut tx = self.pool.begin().await?;
        
        sqlx::query!("DELETE FROM user_tags WHERE user_id = $1", user_id)
            .execute(&mut *tx)
            .await?;
            
        for tag_id in tag_ids {
            sqlx::query!(
                "INSERT INTO user_tags (user_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
                user_id,
                tag_id
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

        for tag_id in tag_ids {
            sqlx::query!(
                "INSERT INTO moment_tags (moment_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
                moment_id,
                tag_id
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

    /// 获取瞬间标签名列表
    pub async fn get_moment_tags(&self, moment_id: Uuid) -> Result<Vec<String>> {
        let rows = sqlx::query_scalar!(
            r#"
            SELECT t.name
            FROM tags t
            JOIN moment_tags mt ON t.tag_id = mt.tag_id
            WHERE mt.moment_id = $1
            "#,
            moment_id
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
            map.entry(row.moment_id)
                .or_default()
                .push(row.name);
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
        let result = sqlx::query!(
            "DELETE FROM tags WHERE name = $1",
            name
        )
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

        for tag_id in tag_ids {
            sqlx::query!(
                "INSERT INTO group_tags (group_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
                group_id,
                tag_id
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
}
