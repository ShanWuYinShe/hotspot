//! 用户仓库层实现
//!
//! 适配新 Schema: user_id 使用 BIGINT，集成 PostGIS 地理空间功能

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};

use crate::error::Result;
use crate::types::enums::{UserGender, UserStatus, UserType};
use crate::utils::pagination::{decode_cursor, encode_cursor, DistanceCursor};

/// 用户基础行数据
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserRow {
    pub user_id: i64,
    pub username: String,
    pub email: Option<String>,
    pub phone: Option<String>,
    pub nickname: Option<String>,
    pub avatar_url: Option<String>,
    pub bio: Option<String>,
    pub gender: UserGender,
    pub birthday: Option<chrono::NaiveDate>,
    pub user_type: UserType,
    pub status: UserStatus,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub last_login_at: Option<DateTime<Utc>>,
    #[serde(skip)]
    pub password_hash: String,
}

/// 用户统计行
#[derive(Debug, Clone, FromRow)]
pub struct UserStatsRow {
    pub moments_count: i32,
    pub following_count: i32,
    pub followers_count: i32,
    pub groups_count: i32,
}

/// 附近的人搜索结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserSearchRow {
    pub user_id: i64,
    pub nickname: Option<String>,
    pub avatar_url: Option<String>,
    pub bio: Option<String>,
    pub gender: UserGender,
    pub distance: f64,
    pub bearing_degrees: f64,
    pub is_online: bool,
    pub last_active_at: Option<DateTime<Utc>>,
}

/// Feed 推荐用户行
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct FeedUserRow {
    pub user_id: i64,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub bio: String,
    pub user_type: UserType,
    pub tags: Vec<String>,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub distance: f64,
    pub bearing_degrees: f64,
}

/// 搜索建议用户行
#[derive(Debug, Clone, FromRow)]
pub struct UserSuggestRow {
    pub user_id: i64,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub bio: String,
    pub distance: Option<f64>,
}

/// 管理员用户搜索行
#[derive(Debug, Clone, FromRow)]
pub struct AdminUserSearchRow {
    pub user_id: i64,
    pub nickname: Option<String>,
    pub avatar_url: Option<String>,
    pub user_type: UserType,
    pub status: UserStatus,
    pub created_at: DateTime<Utc>,
    pub last_active_at: Option<DateTime<Utc>>,
    pub login_id: String,
    pub is_online: bool,
}

#[derive(Debug, Clone)]
pub struct UserRepository {
    pool: PgPool,
}

impl UserRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建用户
    pub async fn create(
        &self,
        username: String,
        password_hash: String,
        nickname: Option<String>,
        email: Option<String>,
        phone: Option<String>,
    ) -> Result<UserRow> {
        let row = sqlx::query_as!(
            UserRow,
            r#"
            INSERT INTO users (username, password_hash, nickname, email, phone)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING user_id, username, email, phone, nickname, avatar_url, bio, 
                      gender as "gender!: UserGender", birthday, user_type as "user_type!: UserType", status as "status!: UserStatus", 
                      created_at, updated_at, last_login_at, password_hash
            "#,
            username,
            password_hash,
            nickname,
            email,
            phone
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 根据用户 ID 查找
    pub async fn find_by_id(&self, user_id: i64) -> Result<Option<UserRow>> {
        let row = sqlx::query_as!(
            UserRow,
            r#"
            SELECT user_id, username, email, phone, nickname, avatar_url, bio, 
                   gender as "gender!: UserGender", birthday, user_type as "user_type!: UserType", status as "status!: UserStatus", 
                   created_at, updated_at, last_login_at, password_hash
            FROM users
            WHERE user_id = $1 AND status != 'deleted'
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 检查用户是否存在
    pub async fn exists(&self, user_id: i64) -> Result<bool> {
        let row = sqlx::query_scalar!(
            "SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1 AND status != 'deleted') as \"exists!\"",
            user_id
        ).fetch_one(&self.pool).await?;
        Ok(row)
    }

    pub async fn list_user_ids_after(&self, last_user_id: i64, limit: i64) -> Result<Vec<i64>> {
        let rows = sqlx::query_scalar!(
            r#"
            SELECT user_id as "user_id!"
            FROM users
            WHERE user_id > $1 AND status != 'deleted'
            ORDER BY user_id ASC
            LIMIT $2
            "#,
            last_user_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 批量获取用户信息
    pub async fn find_profiles(&self, user_ids: &[i64]) -> Result<Vec<UserRow>> {
        let rows = sqlx::query_as!(
            UserRow,
            r#"
            SELECT user_id, username, email, phone, nickname, avatar_url, bio, 
                   gender as "gender!: UserGender", birthday, user_type as "user_type!: UserType", status as "status!: UserStatus", 
                   created_at, updated_at, last_login_at, password_hash
            FROM users
            WHERE user_id = ANY($1) AND status != 'deleted'
            "#,
            user_ids
        ).fetch_all(&self.pool).await?;
        Ok(rows)
    }

    /// 获取统计信息
    pub async fn get_stats(&self, user_id: i64) -> Result<UserStatsRow> {
        let row = sqlx::query_as!(
            UserStatsRow,
            r#"
            SELECT COALESCE(moments_count, 0)::int as "moments_count!",
                   COALESCE(following_count, 0)::int as "following_count!",
                   COALESCE(followers_count, 0)::int as "followers_count!",
                   COALESCE(groups_count, 0)::int as "groups_count!"
            FROM user_stats WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row.unwrap_or(UserStatsRow {
            moments_count: 0,
            following_count: 0,
            followers_count: 0,
            groups_count: 0,
        }))
    }

    /// 更新个人资料
    pub async fn update_profile(
        &self,
        user_id: i64,
        nickname: Option<&str>,
        avatar_url: Option<&str>,
        bio: Option<&str>,
        gender: Option<UserGender>,
        birthday: Option<&str>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE users SET
                nickname = COALESCE($2, nickname),
                avatar_url = COALESCE($3, avatar_url),
                bio = COALESCE($4, bio),
                gender = COALESCE($5::text::user_gender, gender),
                birthday = COALESCE($6::date, birthday),
                updated_at = NOW()
            WHERE user_id = $1
            "#,
            user_id,
            nickname,
            avatar_url,
            bio,
            gender as Option<UserGender>,
            birthday.and_then(|d| chrono::NaiveDate::parse_from_str(d, "%Y-%m-%d").ok())
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 更新密码
    pub async fn update_password(&self, user_id: i64, password_hash: &str) -> Result<()> {
        sqlx::query!(
            "UPDATE users SET password_hash = $2, updated_at = NOW() WHERE user_id = $1",
            user_id,
            password_hash
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 根据标识符查找
    pub async fn find_by_identifier(&self, identifier: &str) -> Result<Option<UserRow>> {
        let row = sqlx::query_as!(
            UserRow,
            r#"
            SELECT user_id, username, email, phone, nickname, avatar_url, bio, 
                   gender as "gender!: UserGender", birthday, user_type as "user_type!: UserType", status as "status!: UserStatus", 
                   created_at, updated_at, last_login_at, password_hash
            FROM users
            WHERE (username = $1 OR email = $1 OR phone = $1) AND status != 'deleted'
            LIMIT 1
            "#,
            identifier
        ).fetch_optional(&self.pool).await?;
        Ok(row)
    }

    /// 更新登录标识（用户名）
    pub async fn update_login_id(&self, user_id: i64, login_id: &str) -> Result<()> {
        sqlx::query!(
            "UPDATE users SET username = $2, updated_at = NOW() WHERE user_id = $1",
            user_id,
            login_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 更新用户类型
    pub async fn update_user_type(&self, user_id: i64, user_type: UserType) -> Result<()> {
        sqlx::query!(
            "UPDATE users SET user_type = $2, updated_at = NOW() WHERE user_id = $1",
            user_id,
            user_type.to_string()
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 更新用户状态
    pub async fn update_status(&self, user_id: i64, status: UserStatus) -> Result<()> {
        sqlx::query!(
            "UPDATE users SET status = $2, updated_at = NOW() WHERE user_id = $1",
            user_id,
            status as UserStatus
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 更新位置
    pub async fn update_location(&self, user_id: i64, lon: f64, lat: f64) -> Result<()> {
        sqlx::query!(
            "UPDATE users SET location = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, location_updated_at = NOW() WHERE user_id = $1",
            user_id, lon, lat
        ).execute(&self.pool).await?;
        Ok(())
    }

    pub async fn get_current_location(&self, user_id: i64) -> Result<Option<(f64, f64)>> {
        let row = sqlx::query!(
            r#"
            SELECT ST_Y(location::geometry) as "lat?",
                   ST_X(location::geometry) as "lng?"
            FROM users
            WHERE user_id = $1 AND location IS NOT NULL
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row.and_then(|r| match (r.lat, r.lng) {
            (Some(lat), Some(lng)) => Some((lat, lng)),
            _ => None,
        }))
    }

    /// 搜索附近的人
    pub async fn search_nearby(
        &self,
        lon: f64,
        lat: f64,
        radius: f64,
        limit: i64,
        cursor: Option<String>,
        kw: Option<&str>,
    ) -> Result<(Vec<UserSearchRow>, Option<String>)> {
        let (last_dist, last_id) = if let Some(c) = cursor {
            decode_cursor::<DistanceCursor>(&c)?
        } else {
            (0.0, 0)
        };

        let rows = sqlx::query_as!(
            UserSearchRow,
            r#"
            SELECT 
                u.user_id as "user_id!", u.nickname, u.avatar_url, u.bio, u.gender as "gender!: UserGender",
                ST_Distance(u.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) as "distance!",
                degrees(ST_Azimuth(ST_SetSRID(ST_MakePoint($1, $2), 4326), u.location::geometry)) as "bearing_degrees!",
                COALESCE(os.is_online, false) as "is_online!",
                os.last_active_at
            FROM users u
            LEFT JOIN user_online_status os ON u.user_id = os.user_id
            WHERE u.status = 'active' AND u.location IS NOT NULL
              AND ST_DWithin(u.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
              AND (
                  ST_Distance(u.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) > $4
                  OR (ST_Distance(u.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) = $4 AND u.user_id > $5)
              )
              AND ($6::text IS NULL OR u.nickname ILIKE '%' || $6 || '%')
            ORDER BY ST_Distance(u.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) ASC, u.user_id ASC LIMIT $7
            "#,
            lon, lat, radius, last_dist, last_id, kw, limit + 1
        ).fetch_all(&self.pool).await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();
        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.distance, r.user_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// Feed 推荐用户（带标签/距离）
    pub async fn search_for_feed(
        &self,
        lon: f64,
        lat: f64,
        radius: f64,
        limit: i64,
        cursor: Option<String>,
        exclude_user_id: Option<i64>,
    ) -> Result<(Vec<FeedUserRow>, Option<String>)> {
        let (last_dist, last_id) = if let Some(c) = cursor {
            decode_cursor::<DistanceCursor>(&c)?
        } else {
            (0.0, 0)
        };

        let rows: Vec<FeedUserRow> = sqlx::query_as!(
            FeedUserRow,
            r#"
            SELECT 
                u.user_id as "user_id!",
                COALESCE(u.nickname, '') as "nickname!",
                u.avatar_url as "avatar_url?",
                COALESCE(u.bio, '') as "bio!",
                u.user_type as "user_type!: UserType",
                COALESCE(
                    ARRAY_REMOVE(ARRAY_AGG(t.name ORDER BY t.name), NULL),
                    ARRAY[]::text[]
                ) as "tags!",
                ST_Y(u.location::geometry) as "latitude!",
                ST_X(u.location::geometry) as "longitude!",
                COALESCE(u.location_name, '') as "location_name!",
                ST_Distance(u.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) as "distance!",
                degrees(ST_Azimuth(ST_SetSRID(ST_MakePoint($1, $2), 4326), u.location::geometry)) as "bearing_degrees!"
            FROM users u
            LEFT JOIN user_tags ut ON u.user_id = ut.user_id
            LEFT JOIN tags t ON ut.tag_id = t.tag_id
            WHERE u.status = 'active'
              AND u.location IS NOT NULL
              AND ST_DWithin(u.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
              AND ($6::bigint IS NULL OR u.user_id <> $6)
              AND (
                  ST_Distance(u.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) > $4
                  OR (ST_Distance(u.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) = $4 AND u.user_id > $5)
              )
            GROUP BY u.user_id, u.nickname, u.avatar_url, u.bio, u.user_type,
                     u.location, u.location_name
            ORDER BY ST_Distance(u.location, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) ASC, u.user_id ASC
            LIMIT $7
            "#,
            lon,
            lat,
            radius,
            last_dist,
            last_id,
            exclude_user_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();
        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.distance, r.user_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 搜索建议：按昵称前缀匹配
    pub async fn suggest_by_nickname(
        &self,
        prefix: &str,
        limit: i64,
        location: Option<(f64, f64)>,
    ) -> Result<Vec<UserSuggestRow>> {
        let pattern = format!("{}%", prefix);
        let (lat, lon) = location
            .map(|(lat, lon)| (Some(lat), Some(lon)))
            .unwrap_or((None, None));

        let rows = sqlx::query_as!(
            UserSuggestRow,
            r#"
            SELECT
                u.user_id as "user_id!",
                COALESCE(u.nickname, u.username) as "nickname!",
                u.avatar_url as "avatar_url?",
                COALESCE(u.bio, '') as "bio!",
                CASE
                    WHEN $2::float8 IS NULL OR $3::float8 IS NULL OR u.location IS NULL
                        THEN NULL
                    ELSE ST_Distance(u.location, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography)
                END as distance
            FROM users u
            WHERE u.status = 'active'
              AND (u.nickname ILIKE $1 OR u.username ILIKE $1)
            ORDER BY (CASE
                WHEN $2::float8 IS NULL OR $3::float8 IS NULL OR u.location IS NULL THEN NULL
                ELSE ST_Distance(u.location, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography)
            END) ASC NULLS LAST, u.user_id ASC
            LIMIT $4
            "#,
            pattern,
            lat,
            lon,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 按关键字搜索用户
    pub async fn search_by_keyword(
        &self,
        kw: &str,
        lon: f64,
        lat: f64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<UserSearchRow>, Option<String>)> {
        let (last_dist, last_id) = if let Some(c) = cursor {
            decode_cursor::<DistanceCursor>(&c)?
        } else {
            (0.0, 0)
        };

        let pattern = format!("%{}%", kw);
        let rows = sqlx::query_as!(
            UserSearchRow,
            r#"
            SELECT 
                u.user_id as "user_id!", u.nickname, u.avatar_url, u.bio, u.gender as "gender!: UserGender",
                ST_Distance(u.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) as "distance!",
                degrees(ST_Azimuth(ST_SetSRID(ST_MakePoint($2, $3), 4326), u.location::geometry)) as "bearing_degrees!",
                COALESCE(os.is_online, false) as "is_online!",
                os.last_active_at
            FROM users u
            LEFT JOIN user_online_status os ON u.user_id = os.user_id
            WHERE u.status = 'active'
              AND u.location IS NOT NULL
              AND (u.nickname ILIKE $1 OR u.username ILIKE $1)
              AND (
                  ST_Distance(u.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) > $4
                  OR (ST_Distance(u.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) = $4 AND u.user_id > $5)
              )
            ORDER BY ST_Distance(u.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) ASC, u.user_id ASC
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
                .map(|r| encode_cursor(&(r.distance, r.user_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 按标签搜索用户
    pub async fn search_by_tag(
        &self,
        tag: &str,
        lon: f64,
        lat: f64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<UserSearchRow>, Option<String>)> {
        let (last_dist, last_id) = if let Some(c) = cursor {
            decode_cursor::<DistanceCursor>(&c)?
        } else {
            (0.0, 0)
        };

        let rows = sqlx::query_as!(
            UserSearchRow,
            r#"
            SELECT 
                u.user_id as "user_id!", u.nickname, u.avatar_url, u.bio, u.gender as "gender!: UserGender",
                ST_Distance(u.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) as "distance!",
                degrees(ST_Azimuth(ST_SetSRID(ST_MakePoint($2, $3), 4326), u.location::geometry)) as "bearing_degrees!",
                COALESCE(os.is_online, false) as "is_online!",
                os.last_active_at
            FROM users u
            JOIN user_tags ut ON u.user_id = ut.user_id
            JOIN tags t ON t.tag_id = ut.tag_id
            LEFT JOIN user_online_status os ON u.user_id = os.user_id
            WHERE u.status = 'active'
              AND u.location IS NOT NULL
              AND t.name = $1
              AND (
                  ST_Distance(u.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) > $4
                  OR (ST_Distance(u.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) = $4 AND u.user_id > $5)
              )
            ORDER BY ST_Distance(u.location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) ASC, u.user_id ASC
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
                .map(|r| encode_cursor(&(r.distance, r.user_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 管理员用户搜索
    pub async fn admin_search(
        &self,
        user_id: Option<i64>,
        nickname: Option<&str>,
        login_id: Option<&str>,
        status: Option<UserStatus>,
        user_type: Option<UserType>,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<AdminUserSearchRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<crate::utils::pagination::IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let nickname_pattern = nickname.map(|v| format!("%{}%", v));

        let rows = sqlx::query_as!(
            AdminUserSearchRow,
            r#"
            SELECT u.user_id as "user_id!", u.nickname, u.avatar_url,
                   u.user_type as "user_type!: UserType",
                   u.status as "status!: UserStatus",
                   u.created_at as "created_at!",
                   os.last_active_at,
                   u.username as "login_id!",
                   COALESCE(os.is_online, false) as "is_online!"
            FROM users u
            LEFT JOIN user_online_status os ON u.user_id = os.user_id
            WHERE ($1::bigint IS NULL OR u.user_id = $1)
              AND ($2::text IS NULL OR u.nickname ILIKE $2)
              AND ($3::text IS NULL OR u.username = $3 OR u.email = $3 OR u.phone = $3)
              AND ($4::text IS NULL OR u.status = $4::text::user_status)
              AND ($5::text IS NULL OR u.user_type = $5::text)
              AND u.user_id < $6
            ORDER BY u.user_id DESC
            LIMIT $7
            "#,
            user_id,
            nickname_pattern,
            login_id,
            status.map(|s| s.to_string()),
            user_type.map(|t| t.to_string()),
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
                .map(|r| encode_cursor(&r.user_id))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }
}
