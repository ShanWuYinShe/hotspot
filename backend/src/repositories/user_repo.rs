use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};

use crate::error::Result;
use crate::types::enums::UserType;

use crate::utils::pagination::{decode_cursor, encode_cursor, DistanceCursor, IdCursor};

/// 用户Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserRow {
    pub user_id: i64,
    pub login_id: Option<String>, // NULL for temp users
    pub nickname: String,
    pub password_hash: Option<String>, // NULL for temp users
    pub avatar_url: Option<String>,    // NULL = use default avatar
    pub bio: String,                   // NOT NULL DEFAULT '' (empty bio = empty string)
    pub email: Option<String>,         // NULL = user didn't provide
    pub phone: Option<String>,         // NULL = user didn't provide
    pub user_type: UserType,
    pub current_latitude: Option<f64>, // NULL = location not set (0.0 ≠ NULL)
    pub current_longitude: Option<f64>, // NULL = location not set
    pub current_location_name: Option<String>, // NULL = location not set
    pub created_at: DateTime<Utc>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>, // NULL = not deleted
}

/// 用户简介Row (用于列表展示，不含敏感信息)
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserProfileRow {
    pub user_id: i64,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub bio: String, // NOT NULL DEFAULT ''
    pub user_type: UserType,
}

/// 用户搜索结果Row - 包含距离和方位角（统一搜索结果结构）
#[derive(Debug, Clone)]
pub struct UserSearchRow {
    pub user_id: i64,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub bio: String, // NOT NULL DEFAULT ''
    pub user_type: UserType,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub tags: Vec<String>,
    /// PostGIS 计算的距离（米）
    pub distance: f64,
    /// PostGIS 计算的方位角（0-360度）
    pub bearing_degrees: f64,
}

/// 管理员搜索用户结果
#[derive(Debug, Clone, FromRow)]
pub struct UserAdminSearchRow {
    pub user_id: i64,
    pub nickname: String,
    pub login_id: Option<String>,
    pub avatar_url: Option<String>,
    pub user_type: UserType,
    pub status: String, // 'normal' | 'banned'
    pub created_at: DateTime<Utc>,
    pub last_active_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct UserRepository {
    pool: PgPool,
}

impl UserRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 获取数据库连接池引用
    pub fn pool(&self) -> &PgPool {
        &self.pool
    }

    /// 创建用户
    pub async fn create(
        &self,
        user_id: i64,
        user_type: UserType,
        login_id: Option<String>,
        nickname: String,
        password_hash: Option<String>,
    ) -> Result<UserRow> {
        let row = sqlx::query_as!(
            UserRow,
            r#"
            INSERT INTO users (user_id, user_type, login_id, nickname, password_hash)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING user_id, login_id, nickname, password_hash, avatar_url, bio,
                      email, phone, user_type as "user_type: UserType",
                      current_latitude, current_longitude, current_location_name,
                      created_at, is_deleted, deleted_at
            "#,
            user_id,
            user_type as UserType,
            login_id,
            nickname,
            password_hash
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按ID查找用户
    pub async fn find_by_id(&self, user_id: i64) -> Result<Option<UserRow>> {
        let row = sqlx::query_as!(
            UserRow,
            r#"
            SELECT user_id, login_id, nickname, password_hash, avatar_url, bio,
                   email, phone, user_type as "user_type: UserType",
                   current_latitude, current_longitude, current_location_name,
                   created_at, is_deleted, deleted_at
            FROM users
            WHERE user_id = $1 AND is_deleted = FALSE
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 批量获取用户简介（不含敏感信息）
    pub async fn find_profiles(&self, user_ids: &[i64]) -> Result<Vec<UserProfileRow>> {
        if user_ids.is_empty() {
            return Ok(vec![]);
        }
        let rows = sqlx::query_as!(
            UserProfileRow,
            r#"
            SELECT user_id, nickname, avatar_url, bio, user_type as "user_type: UserType"
            FROM users
            WHERE user_id = ANY($1) AND is_deleted = FALSE
            "#,
            user_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 检查用户是否存在
    pub async fn exists(&self, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1 AND is_deleted = FALSE) as "exists!""#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 按ID列表查找用户（带距离计算，使用 PostGIS）
    pub async fn find_by_ids_with_distance(
        &self,
        user_ids: &[i64],
        latitude: f64,
        longitude: f64,
    ) -> Result<Vec<UserSearchRow>> {
        if user_ids.is_empty() {
            return Ok(vec![]);
        }
        let rows = sqlx::query!(
            r#"
            SELECT 
                user_id,
                nickname,
                avatar_url,
                bio,
                user_type as "user_type!: UserType",
                COALESCE(current_latitude, 35.154922) as "latitude!",
                COALESCE(current_longitude, 112.682524) as "longitude!",
                COALESCE(current_location_name, '南天门') as "location_name!",
                ST_Distance(
                    COALESCE(location, ST_SetSRID(ST_MakePoint(112.682524, 35.154922), 4326)::geography),
                    ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($2, $3), 4326),
                    COALESCE(location::geometry, ST_SetSRID(ST_MakePoint(112.682524, 35.154922), 4326))
                )), 0) as "bearing_degrees!"
            FROM users
            WHERE user_id = ANY($1) AND is_deleted = FALSE
            ORDER BY "distance!" ASC
            "#,
            user_ids,
            longitude,
            latitude
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows
            .into_iter()
            .map(|r| UserSearchRow {
                user_id: r.user_id,
                nickname: r.nickname,
                avatar_url: r.avatar_url,
                bio: r.bio,
                user_type: r.user_type,
                latitude: r.latitude,
                longitude: r.longitude,
                location_name: r.location_name,
                tags: vec![], // Tags are fetched separately in service layer
                distance: r.distance,
                bearing_degrees: r.bearing_degrees,
            })
            .collect())
    }

    /// 按登录ID查找用户
    pub async fn find_by_login_id(&self, login_id: &str) -> Result<Option<UserRow>> {
        let row = sqlx::query_as!(
            UserRow,
            r#"
            SELECT user_id, login_id, nickname, password_hash, avatar_url, bio,
                   email, phone, user_type as "user_type: UserType",
                   current_latitude, current_longitude, current_location_name,
                   created_at, is_deleted, deleted_at
            FROM users
            WHERE login_id = $1 AND is_deleted = FALSE
            "#,
            login_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 检查登录ID是否存在
    pub async fn exists_by_login_id(&self, login_id: &str) -> Result<bool> {
        let exists = sqlx::query!(
            r#"
            SELECT EXISTS(SELECT 1 FROM users WHERE login_id = $1 AND is_deleted = FALSE) AS "exists!"
            "#,
            login_id
        )
        .fetch_one(&self.pool)
        .await?
        .exists;

        Ok(exists)
    }

    /// 更新用户资料
    pub async fn update_profile(
        &self,
        user_id: i64,
        nickname: Option<&str>,
        avatar_url: Option<&str>,
        bio: Option<&str>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE users
            SET nickname = COALESCE($2, nickname),
                avatar_url = COALESCE($3, avatar_url),
                bio = COALESCE($4, bio)
            WHERE user_id = $1
            "#,
            user_id,
            nickname,
            avatar_url,
            bio
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 更新用户当前位置
    pub async fn update_location(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: Option<&str>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE users
            SET current_latitude = $2,
                current_longitude = $3,
                current_location_name = COALESCE($4, current_location_name)
            WHERE user_id = $1
            "#,
            user_id,
            latitude,
            longitude,
            location_name
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 更新用户密码
    pub async fn update_password(&self, user_id: i64, password_hash: String) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE users
            SET password_hash = $2
            WHERE user_id = $1
            "#,
            user_id,
            password_hash
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 升级账户（临时用户 -> 正式用户）
    pub async fn upgrade_account(
        &self,
        user_id: i64,
        login_id: &str,
        password_hash: &str,
    ) -> Result<()> {
        let normal_type = UserType::Normal.to_string();
        sqlx::query!(
            r#"
            UPDATE users
            SET login_id = $2, 
                password_hash = $3,
                user_type = $4::text::user_type
            WHERE user_id = $1
            "#,
            user_id,
            login_id,
            password_hash,
            normal_type
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 软删除用户
    pub async fn soft_delete(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE users
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 按昵称搜索用户（带距离计算，受隐私设置控制）
    /// - 过滤 searchable = false 的用户
    pub async fn search_by_keyword(
        &self,
        keyword: &str,
        latitude: f64,
        longitude: f64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<UserSearchRow>, Option<String>)> {
        let search_pattern = format!("%{}%", keyword);
        // 解码 cursor: 默认为 id DESC (IdCursor)
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query!(
            r#"
            SELECT 
                u.user_id,
                u.nickname,
                u.avatar_url,
                u.bio,
                u.user_type as "user_type!: UserType",
                COALESCE(u.current_latitude, 35.154922) as "latitude!",
                COALESCE(u.current_longitude, 112.682524) as "longitude!",
                COALESCE(u.current_location_name, '南天门') as "location_name!",
                ST_Distance(
                    COALESCE(u.location, ST_SetSRID(ST_MakePoint(112.682524, 35.154922), 4326)::geography),
                    ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($2, $3), 4326),
                    COALESCE(u.location::geometry, ST_SetSRID(ST_MakePoint(112.682524, 35.154922), 4326))
                )), 0) as "bearing_degrees!"
            FROM users u
            LEFT JOIN user_settings us ON u.user_id = us.user_id
            WHERE u.nickname ILIKE $1 
              AND u.is_deleted = FALSE 
              AND u.user_id < $4
              AND COALESCE(us.searchable, TRUE) = TRUE
            ORDER BY u.user_id DESC
            LIMIT $5
            "#,
            search_pattern,
            longitude,
            latitude,
            last_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows.last().map(|r| encode_cursor(&r.user_id)).transpose()?;

        Ok((
            rows.into_iter()
                .map(|r| UserSearchRow {
                    user_id: r.user_id,
                    nickname: r.nickname,
                    avatar_url: r.avatar_url,
                    bio: r.bio,
                    user_type: r.user_type,
                    latitude: r.latitude,
                    longitude: r.longitude,
                    location_name: r.location_name,
                    tags: vec![], // Tags are fetched separately in service layer
                    distance: r.distance,
                    bearing_degrees: r.bearing_degrees,
                })
                .collect(),
            next_cursor,
        ))
    }

    /// 按位置搜索附近用户（PostGIS 计算距离，受隐私设置控制）
    /// - 过滤 searchable = false 的用户
    /// - 过滤 location_searchable = false 的用户
    pub async fn search_by_location(
        &self,
        kw: Option<&str>,
        latitude: f64,
        longitude: f64,
        radius: f64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<UserSearchRow>, Option<String>)> {
        // Decode cursor: (distance, id)
        let (last_dist, last_id) = if let Some(c) = cursor {
            decode_cursor::<DistanceCursor>(&c)?
        } else {
            (0.0, 0)
        };

        let rows = sqlx::query!(
            r#"
            SELECT 
                u.user_id, 
                u.nickname, 
                u.avatar_url, 
                u.bio,
                u.user_type as "user_type!: UserType",
                u.current_latitude as "latitude!",
                u.current_longitude as "longitude!",
                COALESCE(u.current_location_name, '南天门') as "location_name!",
                calc.dist as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($1, $2), 4326),
                    u.location::geometry
                )), 0) as "bearing_degrees!"
            FROM users u
            CROSS JOIN LATERAL (
                SELECT ST_Distance(
                    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                    u.location
                ) as dist
            ) calc
            LEFT JOIN user_settings us ON u.user_id = us.user_id
            WHERE u.is_deleted = FALSE
              AND u.current_latitude IS NOT NULL
              AND u.current_longitude IS NOT NULL
              AND u.location IS NOT NULL
              AND COALESCE(us.searchable, TRUE) = TRUE
              AND COALESCE(us.location_searchable, TRUE) = TRUE
              AND ($7::text IS NULL OR u.nickname ILIKE ('%' || $7 || '%'))
              AND ST_DWithin(
                  ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                  u.location,
                  $3
              )
              -- Keyset Pagination: (distance, id) > (last_dist, last_id)
              -- Since distance is float, we use approximate comparison for distance, and strict for ID
              AND (
                  calc.dist > $4
                  OR (
                      calc.dist = $4
                      AND u.user_id > $5
                  )
              )
            ORDER BY calc.dist ASC, u.user_id ASC
            LIMIT $6
            "#,
            longitude,
            latitude,
            radius,
            last_dist,
            last_id,
            limit,
            kw
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|r| encode_cursor(&(r.distance, r.user_id)))
            .transpose()?;

        Ok((
            rows.into_iter()
                .map(|r| UserSearchRow {
                    user_id: r.user_id,
                    nickname: r.nickname,
                    avatar_url: r.avatar_url,
                    bio: r.bio,
                    user_type: r.user_type,
                    latitude: r.latitude,
                    longitude: r.longitude,
                    location_name: r.location_name,
                    tags: vec![], // Tags are fetched separately in service layer
                    distance: r.distance,
                    bearing_degrees: r.bearing_degrees,
                })
                .collect(),
            next_cursor,
        ))
    }

    /// 获取用户统计数据（一次 SQL 聚合，减少多次 COUNT 往返）
    pub async fn get_stats_counts(&self, user_id: i64) -> Result<(i64, i64, i64, i64, i64)> {
        let row = sqlx::query!(
            r#"
            SELECT
                (SELECT COUNT(*) FROM moments WHERE user_id = $1 AND is_deleted = FALSE) as "moments_count!",
                (SELECT COUNT(*)
                 FROM user_follows f1
                 INNER JOIN user_follows f2
                   ON f1.following_id = f2.follower_id
                  AND f1.follower_id = f2.following_id
                 WHERE f1.follower_id = $1) as "friends_count!",
                (SELECT COUNT(*) FROM user_follows WHERE following_id = $1) as "followers_count!",
                (SELECT COUNT(*) FROM user_follows WHERE follower_id = $1) as "following_count!",
                (SELECT COUNT(*) FROM group_members WHERE user_id = $1 AND is_deleted = FALSE) as "groups_count!"
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok((
            row.moments_count,
            row.friends_count,
            row.followers_count,
            row.following_count,
            row.groups_count,
        ))
    }

    /// 更新用户最后活跃时间
    pub async fn update_last_active(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE user_online_status
            SET last_active_at = NOW()
            WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 获取用户当前位置
    pub async fn get_location(&self, user_id: i64) -> Result<Option<(f64, f64)>> {
        let row = sqlx::query!(
            r#"
            SELECT current_latitude, current_longitude
            FROM users
            WHERE user_id = $1 AND is_deleted = FALSE
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(
            row.and_then(|r| match (r.current_latitude, r.current_longitude) {
                (Some(lat), Some(lng)) => Some((lat, lng)),
                _ => None,
            }),
        )
    }

    /// 按位置搜索用户（用于 Feed，使用 Keyset Pagination，可排除特定用户）
    pub async fn search_for_feed(
        &self,
        user_lat: f64,
        user_lng: f64,
        radius: f64,
        limit: i64,
        cursor: Option<String>,
        exclude_user_id: Option<i64>,
    ) -> Result<(Vec<UserSearchRow>, Option<String>)> {
        // Decode cursor: (distance, id)
        let (last_dist, last_id) = if let Some(c) = cursor {
            decode_cursor::<DistanceCursor>(&c)?
        } else {
            (0.0, 0)
        };

        let rows = sqlx::query!(
            r#"
            SELECT 
                u.user_id,
                u.nickname,
                u.avatar_url,
                u.bio,
                u.current_latitude as "latitude!",
                u.current_longitude as "longitude!",
                COALESCE(u.current_location_name, '南天门') as "location_name!",
                u.user_type as "user_type!: UserType",
                calc.dist as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($1, $2), 4326),
                    u.location::geometry
                )), 0) as "bearing_degrees!"
            FROM users u
            CROSS JOIN LATERAL (
                SELECT ST_Distance(
                    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                    u.location
                ) as dist
            ) calc
            LEFT JOIN user_settings us ON u.user_id = us.user_id
            WHERE u.is_deleted = FALSE
              AND (us.searchable IS NULL OR us.searchable = TRUE)
              AND (us.location_searchable IS NULL OR us.location_searchable = TRUE)
              AND ($3::BIGINT IS NULL OR u.user_id != $3)
              AND u.current_latitude IS NOT NULL
              AND u.current_longitude IS NOT NULL
              AND u.location IS NOT NULL
              AND ST_DWithin(
                  ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                  u.location,
                  $4
              )
              -- Keyset Pagination: (distance, id) > (last_dist, last_id)
              AND (
                  calc.dist > $5
                  OR (
                      calc.dist = $5
                      AND u.user_id > $6
                  )
              )
            ORDER BY calc.dist ASC, u.user_id ASC
            LIMIT $7
            "#,
            user_lng,
            user_lat,
            exclude_user_id,
            radius,
            last_dist,
            last_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|r| encode_cursor(&(r.distance, r.user_id)))
            .transpose()?;

        Ok((
            rows.into_iter()
                .map(|r| UserSearchRow {
                    user_id: r.user_id,
                    nickname: r.nickname,
                    avatar_url: r.avatar_url,
                    bio: r.bio,
                    user_type: r.user_type,
                    latitude: r.latitude,
                    longitude: r.longitude,
                    location_name: r.location_name,
                    tags: vec![], // Tags are fetched separately in service layer
                    distance: r.distance,
                    bearing_degrees: r.bearing_degrees,
                })
                .collect(),
            next_cursor,
        ))
    }

    /// 管理员搜索用户
    pub async fn admin_search(
        &self,
        user_id: Option<i64>,
        nickname: Option<&str>,
        login_id: Option<&str>,
        status: Option<&str>,
        user_type: Option<UserType>,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<UserAdminSearchRow>, Option<String>)> {
        // 解码 cursor: id DESC
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let nickname_pattern = nickname.map(|s| format!("%{}%", s));
        let login_id_pattern = login_id.map(|s| format!("%{}%", s));
        let is_banned = match status {
            Some("banned") => Some(true),
            Some("normal") => Some(false),
            _ => None,
        };

        let rows = sqlx::query_as!(
            UserAdminSearchRow,
            r#"
            SELECT 
                u.user_id,
                u.nickname,
                u.login_id as "login_id",
                u.avatar_url,
                u.user_type as "user_type: UserType",
                CASE WHEN u.is_banned THEN 'banned' ELSE 'normal' END as "status!",
                u.created_at,
                uos.last_active_at
            FROM users u
            LEFT JOIN user_online_status uos ON u.user_id = uos.user_id
            WHERE u.is_deleted = FALSE
              AND ($1::BIGINT IS NULL OR u.user_id = $1)
              AND ($2::TEXT IS NULL OR u.nickname ILIKE $2)
              AND ($3::TEXT IS NULL OR u.login_id ILIKE $3)
              AND ($4::BOOLEAN IS NULL OR u.is_banned = $4)
              AND ($5::user_type IS NULL OR u.user_type = $5)
              AND u.user_id < $6
            ORDER BY u.user_id DESC
            LIMIT $7
            "#,
            user_id,
            nickname_pattern,
            login_id_pattern,
            is_banned,
            user_type as Option<UserType>,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        // 手动分页处理：截取前 limit 个，并检查是否有更多
        let has_more = rows.len() > limit as usize;
        let items: Vec<UserAdminSearchRow> = rows.into_iter().take(limit as usize).collect();
        let next_cursor = if has_more {
            if let Some(last_item) = items.last() {
                Some(encode_cursor(&last_item.user_id)?)
            } else {
                None
            }
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 封禁用户
    pub async fn ban(
        &self,
        user_id: i64,
        reason: Option<&str>,
        duration_days: Option<i32>,
    ) -> Result<()> {
        let banned_until = duration_days.and_then(|days| {
            if days > 0 {
                Some(Utc::now() + chrono::Duration::days(days as i64))
            } else {
                None
            }
        });

        sqlx::query!(
            r#"
            UPDATE users
            SET is_banned = TRUE, ban_reason = $2, banned_until = $3
            WHERE user_id = $1
            "#,
            user_id,
            reason,
            banned_until
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 解封用户
    pub async fn unban(&self, user_id: i64, _reason: Option<&str>) -> Result<()> {
        // TODO: Record unban reason in audit log if available
        sqlx::query!(
            r#"
            UPDATE users
            SET is_banned = FALSE, ban_reason = NULL, banned_until = NULL
            WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
