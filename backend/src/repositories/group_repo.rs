use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor, DistanceCursor};

/// 群组Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupRow {
    pub id: i64,
    pub group_id: Uuid, // Renamed from group_chat_id to match Service usage
    pub name: String,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub owner_id: i64,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub location: Option<String>, // PostGIS geography type stored as WKT string
    pub location_name: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
    pub is_deleted: bool,
    pub is_dismissed: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

/// 管理员搜索群组结果
#[derive(Debug, Clone, FromRow)]
pub struct GroupAdminSearchRow {
    pub id: i64, // Added for pagination
    pub group_id: Uuid,
    pub name: String,
    pub avatar_url: Option<String>,
    pub member_count: i64,
    pub owner_id: i64,
    pub owner_nickname: String,
    pub owner_avatar_url: Option<String>,
    pub owner_user_type: crate::types::enums::UserType,
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
    #[allow(clippy::too_many_arguments)]
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
            RETURNING id, group_chat_id as group_id, name, description, avatar_url, owner_id,
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

    /// 检查群组是否存在
    pub async fn exists(&self, group_id: Uuid) -> Result<bool> {
        let result = sqlx::query!(
            r#"SELECT EXISTS(SELECT 1 FROM group_chats WHERE group_chat_id = $1 AND is_deleted = FALSE AND is_dismissed = FALSE) as "exists!""#,
            group_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 获取群组所有者ID
    pub async fn get_owner(&self, group_id: Uuid) -> Result<Option<i64>> {
        let row = sqlx::query!(
            r#"SELECT owner_id FROM group_chats WHERE group_chat_id = $1 AND is_deleted = FALSE AND is_dismissed = FALSE"#,
            group_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row.map(|r| r.owner_id))
    }

    /// 按ID查找群组
    pub async fn find_by_id(&self, group_id: Uuid) -> Result<Option<GroupRow>> {
        let row = sqlx::query_as!(
            GroupRow,
            r#"
            SELECT id, group_chat_id as group_id, name, description, avatar_url, owner_id,
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

    /// 按ID列表查找群组（带距离计算，使用 PostGIS）
    pub async fn find_by_ids_with_distance(
        &self,
        group_ids: &[Uuid],
        latitude: f64,
        longitude: f64,
    ) -> Result<Vec<GroupSearchRow>> {
        if group_ids.is_empty() {
            return Ok(vec![]);
        }

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
                (SELECT COUNT(*)::int4 FROM group_members gm WHERE gm.group_chat_id = g.group_chat_id AND gm.is_deleted = FALSE) as "member_count!",
                (gs.password_hash IS NOT NULL) as "has_password!",
                ST_Distance(
                    ST_SetSRID(ST_MakePoint(COALESCE(g.longitude, 112.682524), COALESCE(g.latitude, 35.154922)), 4326)::geography,
                    ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($2, $3), 4326),
                    ST_SetSRID(ST_MakePoint(COALESCE(g.longitude, 112.682524), COALESCE(g.latitude, 35.154922)), 4326)
                )), 0) as "bearing_degrees!",
                g.created_at as "created_at!",
                g.updated_at as "updated_at"
            FROM group_chats g
            LEFT JOIN group_settings gs ON g.group_chat_id = gs.group_chat_id
            WHERE g.group_chat_id = ANY($1) AND g.is_deleted = FALSE AND g.is_dismissed = FALSE
            ORDER BY "distance!" ASC
            "#,
            group_ids,
            longitude,
            latitude
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows
            .into_iter()
            .map(|r| GroupSearchRow {
                group_id: r.group_chat_id,
                name: r.name,
                description: r.description,
                avatar_url: r.avatar_url,
                latitude: r.latitude,
                longitude: r.longitude,
                location_name: r.location_name,
                member_count: r.member_count,
                has_password: r.has_password,
                tags: vec![], // Tags fetched separately usually
                distance: r.distance,
                bearing_degrees: r.bearing_degrees,
                created_at: r.created_at,
                updated_at: r.updated_at,
            })
            .collect())
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

    /// 更新群组位置
    pub async fn update_location(
        &self,
        group_id: Uuid,
        latitude: f64,
        longitude: f64,
        location_name: String,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_chats
            SET latitude = $2,
                longitude = $3,
                location_name = $4,
                location = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography,
                updated_at = NOW()
            WHERE group_chat_id = $1
            "#,
            group_id,
            latitude,
            longitude,
            location_name
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

    /// 发现附近群组（按位置范围过滤，Keyset Pagination）
    /// 位置参数用于：1. 过滤指定范围内的群组  2. 计算距离用于排序
    pub async fn search_by_location(
        &self,
        kw: Option<&str>,
        latitude: f64,
        longitude: f64,
        radius_meters: f64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<GroupSearchRow>, Option<String>)> {
        // Decode cursor: (distance, id)
        let (last_dist, last_id) = if let Some(c) = cursor {
            decode_cursor::<DistanceCursor>(&c)?
        } else {
            (0.0, 0)
        };

        let rows = sqlx::query!(
            r#"
            SELECT g.id, g.group_chat_id, g.name, g.description, g.avatar_url, g.owner_id,
                   g.latitude, g.longitude, ST_AsText(g.location) as location, g.location_name,
                   g.created_at as "created_at!", g.updated_at as "updated_at", g.is_deleted, g.is_dismissed, g.deleted_at,
                   ST_Distance(
                       g.location::geography, 
                       ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                   ) as "distance!",
                   COALESCE(degrees(ST_Azimuth(
                       ST_SetSRID(ST_MakePoint($2, $1), 4326),
                       g.location::geometry
                   )), 0) as "bearing_degrees!",
                   (SELECT COUNT(*) FROM group_members gm WHERE gm.group_chat_id = g.group_chat_id AND gm.is_deleted = FALSE) as "member_count!",
                   (s.password_hash IS NOT NULL) as "has_password!"
            FROM group_chats g
            LEFT JOIN group_settings s ON g.group_chat_id = s.group_chat_id
            WHERE g.is_deleted = FALSE 
              AND g.is_dismissed = FALSE
              AND g.location IS NOT NULL
              AND (
                $7::text IS NULL
                OR g.name ILIKE ('%' || $7 || '%')
                OR COALESCE(g.description, '') ILIKE ('%' || $7 || '%')
              )
              AND ST_DWithin(
                  g.location,
                  ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
                  $3
              )
              -- Keyset Pagination: (distance, id) > (last_dist, last_id)
              AND (
                  ST_Distance(
                      g.location::geography, 
                      ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                  ) > $4
                  OR (
                      ST_Distance(
                          g.location::geography, 
                          ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                      ) = $4
                      AND g.id > $5
                  )
              )
            ORDER BY "distance!" ASC, g.id ASC
            LIMIT $6
            "#,
            latitude,
            longitude,
            radius_meters,
            last_dist,
            last_id,
            limit as i64,
            kw
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|r| encode_cursor(&(r.distance, r.id)))
            .transpose()?;

        Ok((
            rows.into_iter()
            .map(|r| GroupSearchRow {
                group_id: r.group_chat_id,
                name: r.name,
                description: r.description,
                avatar_url: r.avatar_url,
                latitude: r.latitude.unwrap_or(0.0),
                longitude: r.longitude.unwrap_or(0.0),
                location_name: r.location_name.unwrap_or_default(),
                member_count: r.member_count as i32,
                has_password: r.has_password,
                tags: vec![],
                distance: r.distance,
                bearing_degrees: r.bearing_degrees,
                created_at: r.created_at,
                updated_at: r.updated_at,
            })
            .collect(),
            next_cursor,
        ))
    }

    /// 全局搜索群组（不限范围，按关键词匹配，Keyset Pagination）
    /// 位置参数仅用于：计算距离用于排序和显示，不做范围过滤
    pub async fn search_by_name(
        &self,
        keyword: &str,
        latitude: f64,
        longitude: f64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<GroupSearchRow>, Option<String>)> {
        let search_pattern = format!("%{}%", keyword);
        // Decode cursor: (distance, id)
        let (last_dist, last_id) = if let Some(c) = cursor {
            decode_cursor::<DistanceCursor>(&c)?
        } else {
            (0.0, 0)
        };

        let rows = sqlx::query!(
            r#"
            SELECT g.id, g.group_chat_id, g.name, g.description, g.avatar_url, g.owner_id,
                   g.latitude, g.longitude, ST_AsText(g.location) as location, g.location_name,
                   g.created_at as "created_at!", g.updated_at as "updated_at", g.is_deleted, g.is_dismissed, g.deleted_at,
                   ST_Distance(
                       g.location::geography, 
                       ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                   ) as "distance!",
                   COALESCE(degrees(ST_Azimuth(
                       ST_SetSRID(ST_MakePoint($2, $1), 4326),
                       g.location::geometry
                   )), 0) as "bearing_degrees!",
                   (SELECT COUNT(*) FROM group_members gm WHERE gm.group_chat_id = g.group_chat_id AND gm.is_deleted = FALSE) as "member_count!",
                   (s.password_hash IS NOT NULL) as "has_password!"
            FROM group_chats g
            LEFT JOIN group_settings s ON g.group_chat_id = s.group_chat_id
            WHERE g.is_deleted = FALSE 
              AND g.is_dismissed = FALSE
              AND g.location IS NOT NULL
              AND (g.name ILIKE $3 OR g.description ILIKE $3)
              -- Keyset Pagination: (distance, id) > (last_dist, last_id)
              AND (
                  ST_Distance(
                      g.location::geography, 
                      ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                  ) > $4
                  OR (
                      ST_Distance(
                          g.location::geography, 
                          ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                      ) = $4
                      AND g.id > $5
                  )
              )
            ORDER BY "distance!" ASC, g.id ASC
            LIMIT $6
            "#,
            latitude,
            longitude,
            search_pattern,
            last_dist,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|r| encode_cursor(&(r.distance, r.id)))
            .transpose()?;

        Ok((
            rows.into_iter()
            .map(|r| GroupSearchRow {
                group_id: r.group_chat_id,
                name: r.name,
                description: r.description,
                avatar_url: r.avatar_url,
                latitude: r.latitude.unwrap_or(0.0),
                longitude: r.longitude.unwrap_or(0.0),
                location_name: r.location_name.unwrap_or_default(),
                member_count: r.member_count as i32,
                has_password: r.has_password,
                tags: vec![],
                distance: r.distance,
                bearing_degrees: r.bearing_degrees,
                created_at: r.created_at,
                updated_at: r.updated_at,
            })
            .collect(),
            next_cursor,
        ))
    }

    /// 按标签搜索群组（带距离计算，Keyset Pagination）
    pub async fn search_by_tag(
        &self,
        tag_name: &str,
        latitude: f64,
        longitude: f64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<GroupSearchRow>, Option<String>)> {
        // Decode cursor: (distance, id)
        let (last_dist, last_id) = if let Some(c) = cursor {
            decode_cursor::<DistanceCursor>(&c)?
        } else {
            (0.0, 0)
        };

        let rows = sqlx::query!(
            r#"
            SELECT g.id, g.group_chat_id, g.name, g.description, g.avatar_url, g.owner_id,
                   g.latitude, g.longitude, ST_AsText(g.location) as location, g.location_name,
                   g.created_at as "created_at!", g.updated_at as "updated_at", g.is_deleted, g.is_dismissed, g.deleted_at,
                   ST_Distance(
                       g.location::geography, 
                       ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                   ) as "distance!",
                   COALESCE(degrees(ST_Azimuth(
                       ST_SetSRID(ST_MakePoint($2, $1), 4326),
                       g.location::geometry
                   )), 0) as "bearing_degrees!",
                   (SELECT COUNT(*) FROM group_members gm WHERE gm.group_chat_id = g.group_chat_id AND gm.is_deleted = FALSE) as "member_count!",
                   (s.password_hash IS NOT NULL) as "has_password!"
            FROM group_chats g
            INNER JOIN group_tags gt ON g.group_chat_id = gt.group_chat_id
            INNER JOIN tags t ON gt.tag_id = t.id
            LEFT JOIN group_settings s ON g.group_chat_id = s.group_chat_id
            WHERE g.is_deleted = FALSE 
              AND g.is_dismissed = FALSE
              AND g.location IS NOT NULL
              AND t.name = $3
              -- Keyset Pagination: (distance, id) > (last_dist, last_id)
              AND (
                  ST_Distance(
                      g.location::geography, 
                      ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                  ) > $4
                  OR (
                      ST_Distance(
                          g.location::geography, 
                          ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                      ) = $4
                      AND g.id > $5
                  )
              )
            ORDER BY "distance!" ASC, g.id ASC
            LIMIT $6
            "#,
            latitude,
            longitude,
            tag_name,
            last_dist,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|r| encode_cursor(&(r.distance, r.id)))
            .transpose()?;

        Ok((
            rows.into_iter()
            .map(|r| GroupSearchRow {
                group_id: r.group_chat_id,
                name: r.name,
                description: r.description,
                avatar_url: r.avatar_url,
                latitude: r.latitude.unwrap_or(0.0),
                longitude: r.longitude.unwrap_or(0.0),
                location_name: r.location_name.unwrap_or_default(),
                member_count: r.member_count as i32,
                has_password: r.has_password,
                tags: vec![],
                distance: r.distance,
                bearing_degrees: r.bearing_degrees,
                created_at: r.created_at,
                updated_at: r.updated_at,
            })
            .collect(),
            next_cursor,
        ))
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

    /// 管理员搜索群组
    pub async fn admin_search(
        &self,
        group_id: Option<Uuid>,
        name: Option<&str>,
        owner_id: Option<i64>,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<GroupAdminSearchRow>, Option<String>)> {
        // Decode cursor: (created_at, id)
        let (last_created_at, last_id) = if let Some(c) = cursor {
            decode_cursor::<(DateTime<Utc>, i64)>(&c)?
        } else {
            (Utc::now(), i64::MAX) // Default to newest first
        };

        let rows = sqlx::query_as!(
            GroupAdminSearchRow,
            r#"
            SELECT 
                g.id,
                g.group_chat_id as group_id,
                g.name,
                g.avatar_url,
                (SELECT COUNT(*) FROM group_members gm WHERE gm.group_chat_id = g.group_chat_id AND gm.is_deleted = FALSE) as "member_count!",
                u.id as owner_id,
                u.nickname as owner_nickname,
                u.avatar_url as owner_avatar_url,
                u.user_type as "owner_user_type!: crate::types::enums::UserType",
                g.created_at
            FROM group_chats g
            JOIN users u ON g.owner_id = u.id
            WHERE g.is_deleted = FALSE
              AND g.is_dismissed = FALSE
              AND ($1::uuid IS NULL OR g.group_chat_id = $1)
              AND ($2::text IS NULL OR g.name ILIKE ('%' || $2 || '%'))
              AND ($3::bigint IS NULL OR g.owner_id = $3)
              -- Keyset Pagination: (created_at, id) < (last_created_at, last_id) for DESC
              AND (g.created_at < $4 OR (g.created_at = $4 AND g.id < $5))
            ORDER BY g.created_at DESC, g.id DESC
            LIMIT $6
            "#,
            group_id,
            name,
            owner_id,
            last_created_at,
            last_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|r| encode_cursor::<(DateTime<Utc>, i64)>(&(r.created_at, r.id)))
            .transpose()?;

        Ok((rows, next_cursor))
    }

    /// 解散群组（管理员）
    pub async fn dismiss(&self, group_id: Uuid, reason: Option<&str>) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_chats
            SET is_dismissed = TRUE, deleted_at = NOW(), dismiss_reason = $2
            WHERE group_chat_id = $1
            "#,
            group_id,
            reason
        )
        .execute(&self.pool)
        .await?;

        Ok(())
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
    pub bearing_degrees: f64,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
}
