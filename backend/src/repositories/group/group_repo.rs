//! 群组仓库层实现
//!
//! 适配新 Schema: group_id (BIGINT), owner_id (BIGINT), LBS 支持

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};

use crate::error::Result;
use crate::types::enums::GroupJoinMethod;
use crate::utils::pagination::{
    decode_cursor, decode_cursor_or, encode_cursor, DistanceCursor, Paginated,
};

/// 群组基础行数据
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupRow {
    pub group_id: i64,
    pub name: String,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub owner_id: i64,
    pub group_type: String,
    pub max_members: i32,
    pub is_muted: bool,
    pub join_type: GroupJoinMethod,
    pub metadata: serde_json::Value,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub location_name: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub dismissed_at: Option<DateTime<Utc>>,
}

/// 群组搜索结果（带距离和成员数）
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupSearchRow {
    pub group_id: i64,
    pub name: String,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub owner_id: i64,
    pub member_count: i32,
    pub distance: Option<f64>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub location_name: Option<String>,
    pub bearing_degrees: Option<f64>,
    pub created_at: DateTime<Utc>,
}

/// Feed 流群组行（带标签/距离）
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct FeedGroupRow {
    pub group_id: i64,
    pub name: String,
    pub description: String,
    pub avatar_url: Option<String>,
    pub member_count: i32,
    pub distance: f64,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub bearing_degrees: f64,
    pub has_password: bool,
    pub tags: Vec<String>,
}

/// 搜索建议群组行
#[derive(Debug, Clone, FromRow)]
pub struct GroupSuggestRow {
    pub group_id: i64,
    pub name: String,
    pub avatar_url: Option<String>,
    pub description: String,
    pub member_count: i32,
    pub distance: Option<f64>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub location_name: Option<String>,
    pub bearing_degrees: Option<f64>,
}

/// 管理员群组搜索行
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct AdminGroupSearchRow {
    pub group_id: i64,
    pub name: String,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub owner_id: i64,
    pub member_count: i32,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub location_name: Option<String>,
    pub messages_count: i64,
    pub max_members: i32,
    pub created_at: DateTime<Utc>,
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
        owner_id: i64,
        lon: Option<f64>,
        lat: Option<f64>,
        group_type: String,
        join_type: String,
    ) -> Result<GroupRow> {
        let row = sqlx::query_as!(
            GroupRow,
            r#"
            INSERT INTO groups (name, owner_id, location, group_type, join_type)
            VALUES ($1, $2, 
                    CASE WHEN $3::float8 IS NOT NULL AND $4::float8 IS NOT NULL 
                         THEN ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography 
                         ELSE NULL END,
                    $5, $6)
            RETURNING group_id as "group_id!", name as "name!", description, avatar_url, owner_id as "owner_id!",
                      group_type as "group_type!", max_members as "max_members!", is_muted as "is_muted!",
                      join_type as "join_type!: GroupJoinMethod", metadata as "metadata!",
                      ST_Y(location::geometry) as "latitude?",
                      ST_X(location::geometry) as "longitude?",
                      location_name as "location_name?",
                      created_at as "created_at!", updated_at as "updated_at!", dismissed_at
            "#,
            name,
            owner_id,
            lon,
            lat,
            group_type,
            join_type
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 根据 ID 查找
    pub async fn find_by_id(&self, group_id: i64) -> Result<Option<GroupRow>> {
        let row = sqlx::query_as!(
            GroupRow,
            r#"
            SELECT group_id, name as "name!", description, avatar_url, owner_id,
                   group_type as "group_type!", max_members as "max_members!",
                   is_muted as "is_muted!", join_type as "join_type!: GroupJoinMethod", metadata as "metadata!",
                   ST_Y(location::geometry) as "latitude?",
                   ST_X(location::geometry) as "longitude?",
                   location_name as "location_name?",
                   created_at as "created_at!", updated_at as "updated_at!", dismissed_at
            FROM groups
            WHERE group_id = $1 AND dismissed_at IS NULL
            "#,
            group_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn exists(&self, group_id: i64) -> Result<bool> {
        let row = sqlx::query_scalar!(
            "SELECT EXISTS(SELECT 1 FROM groups WHERE group_id = $1 AND dismissed_at IS NULL)",
            group_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row.unwrap_or(false))
    }

    pub async fn touch_updated_at(&self, group_id: i64) -> Result<()> {
        sqlx::query!(
            "UPDATE groups SET updated_at = NOW() WHERE group_id = $1",
            group_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn update_profile(
        &self,
        group_id: i64,
        name: Option<&str>,
        description: Option<&str>,
        avatar_url: Option<&str>,
        max_members: Option<i32>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE groups SET
                name = COALESCE($2, name),
                description = COALESCE($3, description),
                avatar_url = COALESCE($4, avatar_url),
                max_members = COALESCE($5, max_members),
                updated_at = NOW()
            WHERE group_id = $1
            "#,
            group_id,
            name,
            description,
            avatar_url,
            max_members
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn update_location(
        &self,
        group_id: i64,
        lon: f64,
        lat: f64,
        location_name: Option<&str>,
    ) -> Result<()> {
        let loc_name = location_name.unwrap_or("Unknown");
        sqlx::query!(
            r#"
            UPDATE groups SET
                location = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography,
                location_name = $4,
                updated_at = NOW()
            WHERE group_id = $1
            "#,
            group_id,
            lon,
            lat,
            loc_name
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn update_owner(&self, group_id: i64, owner_id: i64) -> Result<()> {
        sqlx::query!(
            "UPDATE groups SET owner_id = $2, updated_at = NOW() WHERE group_id = $1",
            group_id,
            owner_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 搜索附近的群组
    pub async fn search_nearby(
        &self,
        lon: f64,
        lat: f64,
        radius_meters: f64,
        limit: i64,
        cursor: Option<String>,
        kw: Option<&str>,
    ) -> Paginated<GroupSearchRow> {
        let (last_dist, last_id): DistanceCursor = decode_cursor_or(cursor, || (0.0, 0))?;

        let rows = sqlx::query_as!(
            GroupSearchRow,
            r#"
            SELECT 
                g.group_id as "group_id!", g.name as "name!", g.description, g.avatar_url, g.owner_id as "owner_id!",
                COALESCE(gs.member_count, 0)::int as "member_count!",
                ST_Distance(g.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) as distance,
                ST_Y(g.location::geometry) as "latitude?",
                ST_X(g.location::geometry) as "longitude?",
                COALESCE(g.location_name, '') as "location_name?",
                degrees(ST_Azimuth(ST_SetSRID(ST_MakePoint($1, $2), 4326), g.location::geometry)) as "bearing_degrees?",
                g.created_at as "created_at!"
            FROM groups g
            LEFT JOIN group_stats gs ON g.group_id = gs.group_id
            WHERE g.dismissed_at IS NULL
              AND g.location IS NOT NULL
              AND ST_DWithin(g.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
              AND (
                  ST_Distance(g.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) > $4
                  OR (ST_Distance(g.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) = $4 AND g.group_id > $5)
              )
              AND ($6::text IS NULL OR g.name ILIKE '%' || $6 || '%')
            ORDER BY distance ASC, g.group_id ASC
            LIMIT $7
            "#,
            lon,
            lat,
            radius_meters,
            last_dist,
            last_id,
            kw,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.distance.unwrap_or(0.0), r.group_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// Feed: 按位置搜索群组（带标签/密码/距离）
    pub async fn search_by_location(
        &self,
        kw: Option<&str>,
        lon: f64,
        lat: f64,
        radius_meters: f64,
        limit: i32,
        cursor: Option<String>,
        dismiss_user_id: Option<i64>,
    ) -> Paginated<FeedGroupRow> {
        let (last_dist, last_id): DistanceCursor = decode_cursor_or(cursor, || (0.0, 0))?;

        let pattern = kw.map(|v| format!("%{}%", v));

        let rows: Vec<FeedGroupRow> = sqlx::query_as!(
            FeedGroupRow,
            r#"
            SELECT 
                g.group_id as "group_id!",
                g.name as "name!",
                COALESCE(g.description, '') as "description!",
                g.avatar_url as "avatar_url?",
                COALESCE(gs.member_count, 0)::int as "member_count!",
                ST_Distance(g.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) as "distance!",
                ST_Y(g.location::geometry) as "latitude!",
                ST_X(g.location::geometry) as "longitude!",
                COALESCE(g.location_name, '') as "location_name!",
                degrees(ST_Azimuth(ST_SetSRID(ST_MakePoint($1, $2), 4326), g.location::geometry)) as "bearing_degrees!",
                COALESCE(gs_setting.password_hash IS NOT NULL, false) as "has_password!",
                COALESCE(
                    ARRAY_REMOVE(ARRAY_AGG(t.name ORDER BY t.name), NULL),
                    ARRAY[]::text[]
                ) as "tags!"
            FROM groups g
            LEFT JOIN group_stats gs ON g.group_id = gs.group_id
            LEFT JOIN group_settings gs_setting ON g.group_id = gs_setting.group_id
            LEFT JOIN group_tags gt ON g.group_id = gt.group_id
            LEFT JOIN tags t ON gt.tag_id = t.tag_id
            WHERE g.dismissed_at IS NULL
              AND g.location IS NOT NULL
              AND ST_DWithin(g.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
              AND ($4::text IS NULL OR g.name ILIKE $4)
              AND (
                  ST_Distance(g.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) > $5
                  OR (ST_Distance(g.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) = $5 AND g.group_id > $6)
              )
              AND (
                  $8::bigint IS NULL
                  OR feed_entity_not_dismissed($8, 'group'::dismiss_target_type, g.group_id::text)
              )
            GROUP BY g.group_id, g.name, g.description, g.avatar_url, gs.member_count,
                     g.location, g.location_name, gs_setting.password_hash
            ORDER BY ST_Distance(g.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) ASC, g.group_id ASC
            LIMIT $7
            "#,
            lon,
            lat,
            radius_meters,
            pattern,
            last_dist,
            last_id,
            (limit as i64) + 1,
            dismiss_user_id,
        )
        .fetch_all(&self.pool)
        .await?;
        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.distance, r.group_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    pub async fn search_by_keyword(
        &self,
        kw: &str,
        lon: f64,
        lat: f64,
        limit: i64,
        cursor: Option<String>,
    ) -> Paginated<GroupSearchRow> {
        let (last_dist, last_id): DistanceCursor = decode_cursor_or(cursor, || (0.0, 0))?;

        let pattern = format!("%{}%", kw);
        let rows = sqlx::query_as!(
            GroupSearchRow,
            r#"
            SELECT 
                g.group_id as "group_id!", g.name as "name!", g.description, g.avatar_url, g.owner_id as "owner_id!",
                COALESCE(gs.member_count, 0)::int as "member_count!",
                ST_Distance(g.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) as distance,
                ST_Y(g.location::geometry) as "latitude?",
                ST_X(g.location::geometry) as "longitude?",
                COALESCE(g.location_name, '') as "location_name?",
                degrees(ST_Azimuth(ST_SetSRID(ST_MakePoint($2, $3), 4326), g.location::geometry)) as "bearing_degrees?",
                g.created_at as "created_at!"
            FROM groups g
            LEFT JOIN group_stats gs ON g.group_id = gs.group_id
            WHERE g.dismissed_at IS NULL
              AND g.name ILIKE $1
              AND g.location IS NOT NULL
              AND (
                  ST_Distance(g.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) > $4
                  OR (ST_Distance(g.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) = $4 AND g.group_id > $5)
              )
            ORDER BY distance ASC, g.group_id ASC
            LIMIT $6
            "#,
            pattern,
            lon,
            lat,
            last_dist,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();
        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.distance.unwrap_or(0.0), r.group_id)))
                .transpose()?
        } else {
            None
        };
        Ok((items, next_cursor))
    }

    /// 搜索建议：按群组名匹配
    pub async fn suggest_by_name(
        &self,
        prefix: &str,
        limit: i64,
        location: Option<(f64, f64)>,
        radius: Option<f64>,
    ) -> Result<Vec<GroupSuggestRow>> {
        let pattern = format!("{}%", prefix);
        let (lat, lon) = location
            .map(|(lat, lon)| (Some(lat), Some(lon)))
            .unwrap_or((None, None));
        let radius = radius;

        let rows = sqlx::query_as!(
            GroupSuggestRow,
            r#"
            SELECT
                g.group_id as "group_id!",
                g.name as "name!",
                g.avatar_url as "avatar_url?",
                COALESCE(g.description, '') as "description!",
                COALESCE(gs.member_count, 0)::int as "member_count!",
                CASE
                    WHEN $2::float8 IS NULL OR $3::float8 IS NULL OR g.location IS NULL
                        THEN NULL
                    ELSE ST_Distance(g.location, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography)
                END as distance,
                ST_Y(g.location::geometry) as "latitude?",
                ST_X(g.location::geometry) as "longitude?",
                g.location_name as "location_name?",
                CASE WHEN $2::float8 IS NOT NULL AND $3::float8 IS NOT NULL AND g.location IS NOT NULL
                    THEN degrees(ST_Azimuth(ST_SetSRID(ST_MakePoint($3, $2), 4326), g.location::geometry))
                    ELSE NULL
                END as "bearing_degrees?"
            FROM groups g
            LEFT JOIN group_stats gs ON g.group_id = gs.group_id
            WHERE g.dismissed_at IS NULL
              AND g.name ILIKE $1
              AND (
                    $4::float8 IS NULL
                 OR (
                        g.location IS NOT NULL
                    AND $2::float8 IS NOT NULL
                    AND $3::float8 IS NOT NULL
                    AND ST_Distance(g.location, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography) <= $4
                 )
              )
            ORDER BY distance ASC NULLS LAST, g.group_id ASC
            LIMIT $5
            "#,
            pattern,
            lat,
            lon,
            radius,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    pub async fn search_by_tag(
        &self,
        tag: &str,
        lon: f64,
        lat: f64,
        limit: i64,
        cursor: Option<String>,
    ) -> Paginated<GroupSearchRow> {
        let (last_dist, last_id): DistanceCursor = decode_cursor_or(cursor, || (0.0, 0))?;

        let rows = sqlx::query_as!(
            GroupSearchRow,
            r#"
            SELECT 
                g.group_id as "group_id!", g.name as "name!", g.description, g.avatar_url, g.owner_id as "owner_id!",
                COALESCE(gs.member_count, 0)::int as "member_count!",
                ST_Distance(g.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) as distance,
                ST_Y(g.location::geometry) as "latitude?",
                ST_X(g.location::geometry) as "longitude?",
                COALESCE(g.location_name, '') as "location_name?",
                degrees(ST_Azimuth(ST_SetSRID(ST_MakePoint($2, $3), 4326), g.location::geometry)) as "bearing_degrees?",
                g.created_at as "created_at!"
            FROM groups g
            INNER JOIN group_tags gt ON g.group_id = gt.group_id
            INNER JOIN tags t ON t.tag_id = gt.tag_id
            LEFT JOIN group_stats gs ON g.group_id = gs.group_id
            WHERE g.dismissed_at IS NULL
              AND t.name = $1
              AND g.location IS NOT NULL
              AND (
                  ST_Distance(g.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) > $4
                  OR (ST_Distance(g.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) = $4 AND g.group_id > $5)
              )
            ORDER BY distance ASC, g.group_id ASC
            LIMIT $6
            "#,
            tag,
            lon,
            lat,
            last_dist,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();
        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.distance.unwrap_or(0.0), r.group_id)))
                .transpose()?
        } else {
            None
        };
        Ok((items, next_cursor))
    }

    pub async fn list_group_ids_after(
        &self,
        last_group_id: i64,
        limit: i64,
    ) -> Result<Vec<(i64, i64)>> {
        let rows = sqlx::query!(
            r#"
            SELECT group_id as "row_id!", group_id as "group_id!"
            FROM groups
            WHERE group_id > $1 AND dismissed_at IS NULL
            ORDER BY group_id ASC
            LIMIT $2
            "#,
            last_group_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.row_id, r.group_id)).collect())
    }

    pub async fn list_by_ids(&self, group_ids: &[i64]) -> Result<Vec<GroupRow>> {
        if group_ids.is_empty() {
            return Ok(vec![]);
        }
        let rows = sqlx::query_as!(
            GroupRow,
            r#"
            SELECT group_id, name as "name!", description, avatar_url, owner_id,
                   group_type as "group_type!", max_members as "max_members!",
                   is_muted as "is_muted!", join_type as "join_type!: GroupJoinMethod", metadata as "metadata!",
                   ST_Y(location::geometry) as "latitude?",
                   ST_X(location::geometry) as "longitude?",
                   location_name as "location_name?",
                   created_at as "created_at!", updated_at as "updated_at!", dismissed_at
            FROM groups
            WHERE group_id = ANY($1)
            "#,
            group_ids
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

    pub async fn admin_search(
        &self,
        group_id: Option<i64>,
        name: Option<String>,
        owner_id: Option<i64>,
        limit: i64,
        cursor: Option<String>,
    ) -> Paginated<AdminGroupSearchRow> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<i64>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query_as!(
            AdminGroupSearchRow,
            r#"
            SELECT g.group_id, g.name as "name!", g.description as "description?",
                   g.avatar_url as "avatar_url?", g.owner_id,
                   COALESCE(gs.member_count, 0)::int as "member_count!",
                   ST_Y(g.location::geometry) as "latitude?",
                   ST_X(g.location::geometry) as "longitude?",
                   g.location_name as "location_name?",
                   COALESCE(gs.message_count, 0)::bigint as "messages_count!",
                   g.max_members as "max_members!",
                   g.created_at
            FROM groups g
            LEFT JOIN group_stats gs ON g.group_id = gs.group_id
            WHERE g.dismissed_at IS NULL
              AND ($1::bigint IS NULL OR g.group_id = $1)
              AND ($2::text IS NULL OR g.name ILIKE $2)
              AND ($3::bigint IS NULL OR g.owner_id = $3)
              AND g.group_id < $4
            ORDER BY g.group_id DESC
            LIMIT $5
            "#,
            group_id,
            name.map(|n| format!("%{}%", n)),
            owner_id,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();
        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&r.group_id))
                .transpose()?
        } else {
            None
        };
        Ok((items, next_cursor))
    }

    pub async fn get_message_count(&self, group_id: i64) -> Result<i64> {
        let count = sqlx::query_scalar!(
            r#"
            SELECT COALESCE(message_count, 0)::bigint as "count!"
            FROM group_stats
            WHERE group_id = $1
            "#,
            group_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(count.unwrap_or(0))
    }

    /// 解散群组
    pub async fn dismiss(&self, group_id: i64) -> Result<()> {
        sqlx::query!(
            "UPDATE groups SET dismissed_at = NOW() WHERE group_id = $1",
            group_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
