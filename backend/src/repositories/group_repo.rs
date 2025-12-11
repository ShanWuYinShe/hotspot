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

    /// 发现附近群组（按位置范围过滤）
    /// 位置参数用于：1. 过滤指定范围内的群组  2. 计算距离用于排序
    pub async fn discover(
        &self,
        latitude: f64,
        longitude: f64,
        radius_meters: f64,
        keyword: Option<&str>,
        limit: i32,
    ) -> Result<Vec<(GroupRow, f64, i32, bool)>> {
        let search_pattern = keyword.map(|k| format!("%{}%", k));
        
        let rows = sqlx::query!(
            r#"
            SELECT g.id, g.group_chat_id, g.cursor_id, g.name, g.description, g.avatar_url, g.owner_id,
                   g.latitude, g.longitude, ST_AsText(g.location) as location, g.location_name,
                   g.created_at, g.updated_at, g.is_deleted, g.is_dismissed, g.deleted_at,
                   ST_Distance(
                       g.location::geography, 
                       ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                   ) as "distance!",
                   COALESCE(m.member_count, 0) as "member_count!",
                   (s.password_hash IS NOT NULL) as "has_password!"
            FROM group_chats g
            LEFT JOIN (
                SELECT group_chat_id, COUNT(*) as member_count
                FROM group_members
                WHERE is_deleted = FALSE
                GROUP BY group_chat_id
            ) m ON g.group_chat_id = m.group_chat_id
            LEFT JOIN group_settings s ON g.group_chat_id = s.group_chat_id
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
                r.has_password,
            )
        }).collect())
    }

    /// 全局搜索群组（不限范围，按关键词匹配）
    /// 位置参数仅用于：计算距离用于排序和显示，不做范围过滤
    pub async fn search_by_name(
        &self,
        keyword: &str,
        latitude: f64,
        longitude: f64,
        limit: i32,
    ) -> Result<Vec<(GroupRow, f64, i32, bool)>> {
        let search_pattern = format!("%{}%", keyword);
        
        let rows = sqlx::query!(
            r#"
            SELECT g.id, g.group_chat_id, g.cursor_id, g.name, g.description, g.avatar_url, g.owner_id,
                   g.latitude, g.longitude, ST_AsText(g.location) as location, g.location_name,
                   g.created_at, g.updated_at, g.is_deleted, g.is_dismissed, g.deleted_at,
                   ST_Distance(
                       g.location::geography, 
                       ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                   ) as "distance!",
                   COALESCE(m.member_count, 0) as "member_count!",
                   (s.password_hash IS NOT NULL) as "has_password!"
            FROM group_chats g
            LEFT JOIN (
                SELECT group_chat_id, COUNT(*) as member_count
                FROM group_members
                WHERE is_deleted = FALSE
                GROUP BY group_chat_id
            ) m ON g.group_chat_id = m.group_chat_id
            LEFT JOIN group_settings s ON g.group_chat_id = s.group_chat_id
            WHERE g.is_deleted = FALSE 
              AND g.is_dismissed = FALSE
              AND g.location IS NOT NULL
              AND (g.name ILIKE $3 OR g.description ILIKE $3)
            ORDER BY "distance!" ASC
            LIMIT $4
            "#,
            latitude,
            longitude,
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
                r.has_password,
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

    /// 按位置搜索群组（PostGIS 计算距离）
    pub async fn search_by_location(
        &self,
        user_lat: f64,
        user_lng: f64,
        radius: f64,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<GroupSearchRow>> {
        let rows = sqlx::query!(
            r#"
            SELECT 
                g.group_chat_id,
                g.name,
                g.description,
                g.avatar_url,
                COALESCE(g.latitude, 35.154922) as "latitude!",
                COALESCE(g.longitude, 112.682524) as "longitude!",
                COALESCE(g.location_name, '南天门') as "location_name!",
                COALESCE(m.member_count, 0)::int4 as "member_count!",
                (gs.password_hash IS NOT NULL) as "has_password!",
                ST_Distance(
                    ST_SetSRID(ST_MakePoint(COALESCE(g.longitude, 112.682524), COALESCE(g.latitude, 35.154922)), 4326)::geography,
                    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($1, $2), 4326),
                    ST_SetSRID(ST_MakePoint(COALESCE(g.longitude, 112.682524), COALESCE(g.latitude, 35.154922)), 4326)
                )), 0) as "degrees!"
            FROM group_chats g
            LEFT JOIN group_settings gs ON g.group_chat_id = gs.group_chat_id
            LEFT JOIN (
                SELECT group_chat_id, COUNT(*) as member_count
                FROM group_members
                WHERE is_deleted = FALSE
                GROUP BY group_chat_id
            ) m ON g.group_chat_id = m.group_chat_id
            WHERE g.is_deleted = FALSE
              AND g.is_dismissed = FALSE
              AND (gs.is_searchable IS NULL OR gs.is_searchable = TRUE)
              AND ST_DWithin(
                  ST_SetSRID(ST_MakePoint(COALESCE(g.longitude, 112.682524), COALESCE(g.latitude, 35.154922)), 4326)::geography,
                  ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                  $3
              )
            ORDER BY "distance!" ASC
            LIMIT $4 OFFSET $5
            "#,
            user_lng,
            user_lat,
            radius,
            limit,
            offset
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| GroupSearchRow {
            group_id: r.group_chat_id,
            name: r.name,
            description: r.description,
            avatar_url: r.avatar_url,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            member_count: r.member_count,
            has_password: r.has_password,
            tags: vec![], // TODO: 从 group_tags 获取
            distance: r.distance,
            degrees: r.degrees,
        }).collect())
    }
}

/// 群组搜索结果Row（统一结构，包含距离和方位角）
#[derive(Debug, Clone)]
pub struct GroupSearchRow {
    pub group_id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub member_count: i32,
    pub has_password: bool,
    pub tags: Vec<String>,
    /// PostGIS 计算的距离（米）
    pub distance: f64,
    /// PostGIS 计算的方位角（0-360度）
    pub degrees: f64,
}
