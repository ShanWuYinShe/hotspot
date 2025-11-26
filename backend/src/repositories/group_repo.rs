use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::Result;

/// 群组Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupRow {
    pub id: i64,
    pub group_id: Uuid, // Renamed from group_chat_id to match Service usage
    pub cursor_id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub owner_id: i64,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub location: Option<String>,  // PostGIS geography type stored as WKT string
    pub location_name: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
    pub is_deleted: bool,
    pub is_dismissed: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct GroupRepository {
    pool: PgPool,
}

impl GroupRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建群组
    pub async fn create(
        &self,
        name: String,
        description: Option<String>,
        avatar_url: Option<String>,
        owner_id: i64,
        latitude: Option<f64>,
        longitude: Option<f64>,
        location_name: Option<String>,
    ) -> Result<GroupRow> {
        let row = sqlx::query_as!(
            GroupRow,
            r#"
            INSERT INTO group_chats (
                name, description, avatar_url, owner_id,
                latitude, longitude, location, location_name
            )
            VALUES (
                $1, $2, $3, $4, $5, $6,
                CASE WHEN $5::float8 IS NOT NULL AND $6::float8 IS NOT NULL 
                     THEN ST_SetSRID(ST_MakePoint($6, $5), 4326)::geography 
                     ELSE NULL END,
                $7
            )
            RETURNING id, group_chat_id as group_id, cursor_id, name, description, avatar_url, owner_id,
                      latitude, longitude, ST_AsText(location) as location, location_name,
                      created_at, updated_at, is_deleted, is_dismissed, deleted_at
            "#,
            name,
            description,
            avatar_url,
            owner_id,
            latitude,
            longitude,
            location_name
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按ID查找群组
    pub async fn find_by_id(&self, group_id: Uuid) -> Result<Option<GroupRow>> {
        let row = sqlx::query_as!(
            GroupRow,
            r#"
            SELECT id, group_chat_id as group_id, cursor_id, name, description, avatar_url, owner_id,
                   latitude, longitude, ST_AsText(location) as location, location_name,
                   created_at, updated_at, is_deleted, is_dismissed, deleted_at
            FROM group_chats
            WHERE group_chat_id = $1 AND is_deleted = FALSE AND is_dismissed = FALSE
            "#,
            group_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 更新群组信息
    pub async fn update(
        &self,
        group_id: Uuid,
        name: Option<String>,
        description: Option<String>,
        avatar_url: Option<String>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_chats
            SET name = COALESCE($2, name),
                description = COALESCE($3, description),
                avatar_url = COALESCE($4, avatar_url),
                updated_at = NOW()
            WHERE group_chat_id = $1
            "#,
            group_id,
            name,
            description,
            avatar_url
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 软删除群组
    pub async fn soft_delete(&self, group_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_chats
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE group_chat_id = $1
            "#,
            group_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 将 UUID cursor 转换为自增主键
    async fn resolve_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(c) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM group_chats WHERE cursor_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 按名称搜索群组
    pub async fn search_by_name(&self, keyword: &str, limit: i32, cursor: Option<Uuid>) -> Result<Vec<GroupRow>> {
        let search_pattern = format!("%{}%", keyword);
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let groups = sqlx::query_as!(
            GroupRow,
            r#"
            SELECT id, group_chat_id as group_id, cursor_id, name, description, avatar_url, owner_id,
                   latitude, longitude, ST_AsText(location) as location, location_name,
                   created_at, updated_at, is_deleted, is_dismissed, deleted_at
            FROM group_chats
            WHERE name ILIKE $1 AND id < $2 AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
            search_pattern,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(groups)
    }

    /// 搜索附近群组
    pub async fn search_nearby(&self, latitude: f64, longitude: f64, radius_meters: f64, limit: i32) -> Result<Vec<(Uuid, f64)>> {
        // 使用 PostGIS 查询附近群组
        // 返回 (group_id, distance)
        let rows = sqlx::query!(
            r#"
            SELECT group_chat_id, 
                   ST_Distance(
                       location, 
                       ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                   ) as "distance!"
            FROM group_chats
            WHERE is_deleted = FALSE 
              AND is_dismissed = FALSE
              AND location IS NOT NULL
              AND ST_DWithin(
                  location,
                  ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
                  $3
              )
            ORDER BY ST_Distance(
                       location, 
                       ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                   ) ASC
            LIMIT $4
            "#,
            latitude,
            longitude,
            radius_meters,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.group_chat_id, r.distance)).collect())
    }

    /// 获取成员数量
    pub async fn get_member_count(&self, group_id: Uuid) -> Result<i32> {
        let count = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM group_members
            WHERE group_chat_id = $1 AND is_deleted = FALSE
            "#,
            group_id
        )
        .fetch_one(&self.pool)
        .await?
        .count as i32;
        Ok(count)
    }

    /// 批量获取成员数量
    pub async fn get_member_counts(&self, group_ids: &[Uuid]) -> Result<std::collections::HashMap<Uuid, i32>> {
        if group_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT group_chat_id, COUNT(*) as "count!"
            FROM group_members
            WHERE group_chat_id = ANY($1) AND is_deleted = FALSE
            GROUP BY group_chat_id
            "#,
            group_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.group_chat_id, r.count as i32)).collect())
    }

    /// 批量获取群组
    pub async fn find_by_ids(&self, group_ids: &[Uuid]) -> Result<Vec<GroupRow>> {
        if group_ids.is_empty() {
            return Ok(Vec::new());
        }

        let groups = sqlx::query_as!(
            GroupRow,
            r#"
            SELECT id, group_chat_id as group_id, cursor_id, name, description, avatar_url, owner_id,
                   latitude, longitude, ST_AsText(location) as location, location_name,
                   created_at, updated_at, is_deleted, is_dismissed, deleted_at
            FROM group_chats
            WHERE group_chat_id = ANY($1) AND is_deleted = FALSE AND is_dismissed = FALSE
            "#,
            group_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(groups)
    }

    /// 搜索附近群组（返回完整信息，包含成员数量）
    pub async fn search_nearby_with_details(
        &self,
        latitude: f64,
        longitude: f64,
        radius_meters: f64,
        keyword: Option<&str>,
        limit: i32,
    ) -> Result<Vec<(GroupRow, f64, i32)>> {
        // 使用 JOIN 一次性获取群组信息和成员数量
        let search_pattern = keyword.map(|k| format!("%{}%", k));
        
        let rows = sqlx::query!(
            r#"
            SELECT g.id, g.group_chat_id, g.cursor_id, g.name, g.description, g.avatar_url, g.owner_id,
                   g.latitude, g.longitude, ST_AsText(g.location) as location, g.location_name,
                   g.created_at, g.updated_at, g.is_deleted, g.is_dismissed, g.deleted_at,
                   ST_Distance(
                       g.location, 
                       ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                   ) as "distance!",
                   COALESCE(m.member_count, 0) as "member_count!"
            FROM group_chats g
            LEFT JOIN (
                SELECT group_chat_id, COUNT(*) as member_count
                FROM group_members
                WHERE is_deleted = FALSE
                GROUP BY group_chat_id
            ) m ON g.group_chat_id = m.group_chat_id
            WHERE g.is_deleted = FALSE 
              AND g.is_dismissed = FALSE
              AND g.location IS NOT NULL
              AND ST_DWithin(
                  g.location,
                  ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
                  $3
              )
              AND ($4::text IS NULL OR g.name ILIKE $4 OR g.description ILIKE $4)
            ORDER BY "distance!" ASC
            LIMIT $5
            "#,
            latitude,
            longitude,
            radius_meters,
            search_pattern,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| {
            (
                GroupRow {
                    id: r.id,
                    group_id: r.group_chat_id,
                    cursor_id: r.cursor_id,
                    name: r.name,
                    description: r.description,
                    avatar_url: r.avatar_url,
                    owner_id: r.owner_id,
                    latitude: r.latitude,
                    longitude: r.longitude,
                    location: r.location,
                    location_name: r.location_name,
                    created_at: r.created_at,
                    updated_at: r.updated_at,
                    is_deleted: r.is_deleted,
                    is_dismissed: r.is_dismissed,
                    deleted_at: r.deleted_at,
                },
                r.distance,
                r.member_count as i32,
            )
        }).collect())
    }

    /// 按名称搜索群组（带成员数量）
    pub async fn search_by_name_with_member_count(
        &self,
        keyword: &str,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<Vec<(GroupRow, i32)>> {
        let search_pattern = format!("%{}%", keyword);
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let rows = sqlx::query!(
            r#"
            SELECT g.id, g.group_chat_id, g.cursor_id, g.name, g.description, g.avatar_url, g.owner_id,
                   g.latitude, g.longitude, ST_AsText(g.location) as location, g.location_name,
                   g.created_at, g.updated_at, g.is_deleted, g.is_dismissed, g.deleted_at,
                   COALESCE(m.member_count, 0) as "member_count!"
            FROM group_chats g
            LEFT JOIN (
                SELECT group_chat_id, COUNT(*) as member_count
                FROM group_members
                WHERE is_deleted = FALSE
                GROUP BY group_chat_id
            ) m ON g.group_chat_id = m.group_chat_id
            WHERE g.name ILIKE $1 AND g.id < $2 AND g.is_deleted = FALSE
            ORDER BY g.id DESC
            LIMIT $3
            "#,
            search_pattern,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(rows.into_iter().map(|r| {
            (
                GroupRow {
                    id: r.id,
                    group_id: r.group_chat_id,
                    cursor_id: r.cursor_id,
                    name: r.name,
                    description: r.description,
                    avatar_url: r.avatar_url,
                    owner_id: r.owner_id,
                    latitude: r.latitude,
                    longitude: r.longitude,
                    location: r.location,
                    location_name: r.location_name,
                    created_at: r.created_at,
                    updated_at: r.updated_at,
                    is_deleted: r.is_deleted,
                    is_dismissed: r.is_dismissed,
                    deleted_at: r.deleted_at,
                },
                r.member_count as i32,
            )
        }).collect())
    }

    /// 转让群主
    pub async fn transfer_ownership(&self, group_id: Uuid, new_owner_id: i64) -> Result<()> {
        // 将原群主降为管理员
        sqlx::query!(
            r#"
            UPDATE group_members
            SET role = 'admin'
            WHERE group_chat_id = $1 AND role = 'owner'
            "#,
            group_id
        )
        .execute(&self.pool)
        .await?;

        // 将新用户提升为群主
        sqlx::query!(
            r#"
            UPDATE group_members
            SET role = 'owner'
            WHERE group_chat_id = $1 AND user_id = $2
            "#,
            group_id,
            new_owner_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}
