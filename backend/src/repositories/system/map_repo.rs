//! 地图仓库层实现
//!
//! 基于 PostGIS 提供高性能的 LBS 聚类和实体检索

use crate::error::Result;
use crate::utils::pagination::{decode_cursor_or, encode_cursor, DistanceCursor, DistanceUuidCursor, Paginated};
use chrono::{DateTime, Utc};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

#[derive(Debug, Clone)]
pub struct MapRepository {
	    pool: PgPool,
}

/// 热点列表行（用户）
#[derive(Debug, FromRow)]
pub struct UserHotspotRow {
	pub user_id: i64,
	pub nickname: Option<String>,
	pub location_name: Option<String>,
	pub avatar_url: Option<String>,
	pub latitude: f64,
	pub longitude: f64,
	pub created_at: DateTime<Utc>,
	pub distance: f64,
}

/// 热点列表行（群组）
#[derive(Debug, FromRow)]
pub struct GroupHotspotRow {
	pub group_id: i64,
	pub name: String,
	pub location_name: String,
	pub avatar_url: Option<String>,
	pub latitude: f64,
	pub longitude: f64,
	pub created_at: DateTime<Utc>,
	pub distance: f64,
}

/// 热点列表行（动态）
#[derive(Debug, FromRow)]
pub struct MomentHotspotRow {
	pub moment_id: Uuid,
	pub description: Option<String>,
	pub location_name: Option<String>,
	pub latitude: f64,
	pub longitude: f64,
	pub created_at: DateTime<Utc>,
	pub distance: f64,
}

/// 热点列表行（混合：用户 / 群组 / 动态）
#[derive(Debug, FromRow)]
pub struct UnifiedHotspotRow {
	pub object_type: String,
	pub object_id: String,
	pub title: String,
	pub subtitle: Option<String>,
	pub cover_url: Option<String>,
	pub latitude: f64,
	pub longitude: f64,
	pub created_at: DateTime<Utc>,
	pub distance: f64,
}

impl MapRepository {
	    pub fn new(pool: PgPool) -> Self {
	        Self { pool }
	    }

		/// 视口内用户热点（按距离排序，游标分页）
		pub async fn list_user_hotspots(
		&self,
		west: f64,
		south: f64,
		east: f64,
		north: f64,
		start_time: Option<String>,
		end_time: Option<String>,
		limit: i32,
		cursor: Option<String>,
	) -> Paginated<UserHotspotRow> {
		let (last_dist, last_id): DistanceCursor =
			decode_cursor_or(cursor, || (0.0, 0_i64))?;

		let rows: Vec<UserHotspotRow> = sqlx::query_as!(
			UserHotspotRow,
			r#"
			WITH env AS (
			  SELECT ST_MakeEnvelope($1, $2, $3, $4, 4326) AS bbox,
			         ST_Centroid(ST_MakeEnvelope($1, $2, $3, $4, 4326)) AS center
			)
			SELECT
			  u.user_id        as "user_id!",
			  u.nickname       as "nickname?",
			  u.location_name  as "location_name?",
			  u.avatar_url     as "avatar_url?",
			  ST_Y(u.location::geometry) as "latitude!",
			  ST_X(u.location::geometry) as "longitude!",
			  u.created_at     as "created_at!",
			  ST_Distance(u.location, env.center::geography) as "distance!"
			FROM users u, env
			WHERE u.status = 'normal'
			  AND u.deleted_at IS NULL
			  AND u.location IS NOT NULL
			  AND ST_Intersects(u.location, env.bbox::geography)
			  AND ($5::text IS NULL OR u.created_at >= ($5::text)::timestamptz)
			  AND ($6::text IS NULL OR u.created_at <= ($6::text)::timestamptz)
			  AND (
			    ST_Distance(u.location, env.center::geography) > $7
			    OR (ST_Distance(u.location, env.center::geography) = $7 AND u.user_id > $8)
			  )
			ORDER BY ST_Distance(u.location, env.center::geography) ASC, u.user_id ASC
			LIMIT $9
			"#,
			west,
			south,
			east,
			north,
			start_time,
			end_time,
			last_dist,
			last_id,
			(limit + 1) as i64,
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

	/// 视口内群组热点（按距离排序，游标分页）
	pub async fn list_group_hotspots(
		&self,
		west: f64,
		south: f64,
		east: f64,
		north: f64,
		start_time: Option<String>,
		end_time: Option<String>,
		limit: i32,
		cursor: Option<String>,
	) -> Paginated<GroupHotspotRow> {
		let (last_dist, last_id): DistanceCursor =
			decode_cursor_or(cursor, || (0.0, 0_i64))?;

		let rows: Vec<GroupHotspotRow> = sqlx::query_as!(
			GroupHotspotRow,
			r#"
			WITH env AS (
			  SELECT ST_MakeEnvelope($1, $2, $3, $4, 4326) AS bbox,
			         ST_Centroid(ST_MakeEnvelope($1, $2, $3, $4, 4326)) AS center
			)
			SELECT
			  g.group_id      as "group_id!",
			  g.name          as "name!",
			  g.location_name as "location_name!",
			  g.avatar_url    as "avatar_url?",
			  ST_Y(g.location::geometry) as "latitude!",
			  ST_X(g.location::geometry) as "longitude!",
			  g.created_at    as "created_at!",
			  ST_Distance(g.location, env.center::geography) as "distance!"
			FROM groups g, env
			WHERE g.dismissed_at IS NULL
			  AND g.location IS NOT NULL
			  AND ST_Intersects(g.location, env.bbox::geography)
			  AND ($5::text IS NULL OR g.created_at >= ($5::text)::timestamptz)
			  AND ($6::text IS NULL OR g.created_at <= ($6::text)::timestamptz)
			  AND (
			    ST_Distance(g.location, env.center::geography) > $7
			    OR (ST_Distance(g.location, env.center::geography) = $7 AND g.group_id > $8)
			  )
			ORDER BY ST_Distance(g.location, env.center::geography) ASC, g.group_id ASC
			LIMIT $9
			"#,
			west,
			south,
			east,
			north,
			start_time,
			end_time,
			last_dist,
			last_id,
			(limit + 1) as i64,
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

	/// 视口内动态热点（按距离排序，游标分页）
	pub async fn list_moment_hotspots(
		&self,
		west: f64,
		south: f64,
		east: f64,
		north: f64,
		start_time: Option<String>,
		end_time: Option<String>,
		limit: i32,
		cursor: Option<String>,
	) -> Paginated<MomentHotspotRow> {
		let (last_dist, last_id): DistanceUuidCursor =
			decode_cursor_or(cursor, || (0.0, Uuid::nil()))?;

		let rows: Vec<MomentHotspotRow> = sqlx::query_as!(
			MomentHotspotRow,
			r#"
			WITH env AS (
			  SELECT ST_MakeEnvelope($1, $2, $3, $4, 4326) AS bbox,
			         ST_Centroid(ST_MakeEnvelope($1, $2, $3, $4, 4326)) AS center
			)
			SELECT
			  m.moment_id     as "moment_id!",
			  m.description   as "description?",
			  m.location_name as "location_name?",
			  ST_Y(m.location::geometry) as "latitude!",
			  ST_X(m.location::geometry) as "longitude!",
			  m.created_at    as "created_at!",
			  ST_Distance(m.location, env.center::geography) as "distance!"
			FROM moments m, env
			WHERE m.deleted_at IS NULL
			  AND m.location IS NOT NULL
			  AND m.privacy_level = 'public'
			  AND ST_Intersects(m.location, env.bbox::geography)
			  AND ($5::text IS NULL OR m.created_at >= ($5::text)::timestamptz)
			  AND ($6::text IS NULL OR m.created_at <= ($6::text)::timestamptz)
			  AND (
			    ST_Distance(m.location, env.center::geography) > $7
			    OR (ST_Distance(m.location, env.center::geography) = $7 AND m.moment_id > $8)
			  )
			ORDER BY ST_Distance(m.location, env.center::geography) ASC, m.moment_id ASC
			LIMIT $9
			"#,
			west,
			south,
			east,
			north,
			start_time,
			end_time,
			last_dist,
			last_id,
			(limit + 1) as i64,
		)
		.fetch_all(&self.pool)
		.await?;

		let has_more = rows.len() > limit as usize;
		let items: Vec<_> = rows.into_iter().take(limit as usize).collect();
		let next_cursor = if has_more {
			items
				.last()
				.map(|r| encode_cursor(&(r.distance, r.moment_id)))
				.transpose()?
		} else {
			None
		};

		Ok((items, next_cursor))
	}

	/// 视口内所有类型热点（用户 / 群组 / 动态，按距离排序，游标分页）
	pub async fn list_all_hotspots(
		&self,
		west: f64,
		south: f64,
		east: f64,
		north: f64,
		start_time: Option<String>,
		end_time: Option<String>,
		limit: i32,
		cursor: Option<String>,
	) -> Paginated<UnifiedHotspotRow> {
		let (last_dist, last_id): (f64, String) =
			decode_cursor_or(cursor, || (0.0_f64, String::new()))?;

		let rows: Vec<UnifiedHotspotRow> = sqlx::query_as!(
			UnifiedHotspotRow,
			r#"
			WITH env AS (
			  SELECT ST_MakeEnvelope($1, $2, $3, $4, 4326) AS bbox,
			         ST_Centroid(ST_MakeEnvelope($1, $2, $3, $4, 4326)) AS center
			),
			users_query AS (
			  SELECT
			    'user'::text AS object_type,
			    u.user_id::text AS object_id,
			    COALESCE(u.nickname, '') AS title,
			    u.location_name AS subtitle,
			    u.avatar_url AS cover_url,
			    ST_Y(u.location::geometry) AS latitude,
			    ST_X(u.location::geometry) AS longitude,
			    u.created_at,
			    ST_Distance(u.location, env.center::geography) AS distance
			  FROM users u, env
			  WHERE u.status = 'normal'
			    AND u.deleted_at IS NULL
			    AND u.location IS NOT NULL
			    AND ST_Intersects(u.location, env.bbox::geography)
			    AND ($5::text IS NULL OR u.created_at >= ($5::text)::timestamptz)
			    AND ($6::text IS NULL OR u.created_at <= ($6::text)::timestamptz)
			),
			groups_query AS (
			  SELECT
			    'group'::text AS object_type,
			    g.group_id::text AS object_id,
			    g.name AS title,
			    g.location_name AS subtitle,
			    g.avatar_url AS cover_url,
			    ST_Y(g.location::geometry) AS latitude,
			    ST_X(g.location::geometry) AS longitude,
			    g.created_at,
			    ST_Distance(g.location, env.center::geography) AS distance
			  FROM groups g, env
			  WHERE g.dismissed_at IS NULL
			    AND g.location IS NOT NULL
			    AND ST_Intersects(g.location, env.bbox::geography)
			    AND ($5::text IS NULL OR g.created_at >= ($5::text)::timestamptz)
			    AND ($6::text IS NULL OR g.created_at <= ($6::text)::timestamptz)
			),
			moments_query AS (
			  SELECT
			    'moment'::text AS object_type,
			    m.moment_id::text AS object_id,
			    COALESCE(m.description, '') AS title,
			    m.location_name AS subtitle,
			    NULL::text AS cover_url,
			    ST_Y(m.location::geometry) AS latitude,
			    ST_X(m.location::geometry) AS longitude,
			    m.created_at,
			    ST_Distance(m.location, env.center::geography) AS distance
			  FROM moments m, env
			  WHERE m.deleted_at IS NULL
			    AND m.location IS NOT NULL
			    AND m.privacy_level = 'public'
			    AND ST_Intersects(m.location, env.bbox::geography)
			    AND ($5::text IS NULL OR m.created_at >= ($5::text)::timestamptz)
			    AND ($6::text IS NULL OR m.created_at <= ($6::text)::timestamptz)
			),
			all_items AS (
			  SELECT * FROM users_query
			  UNION ALL
			  SELECT * FROM groups_query
			  UNION ALL
			  SELECT * FROM moments_query
			)
			SELECT
			  object_type as "object_type!",
			  object_id as "object_id!",
			  title as "title!",
			  subtitle as "subtitle?",
			  cover_url as "cover_url?",
			  latitude as "latitude!",
			  longitude as "longitude!",
			  created_at as "created_at!",
			  distance as "distance!"
			FROM all_items
			WHERE
			  (distance > $7)
			  OR (distance = $7 AND object_id > $8)
			ORDER BY distance ASC, object_id ASC
			LIMIT $9
			"#,
			west,
			south,
			east,
			north,
			start_time,
			end_time,
			last_dist,
			last_id,
			(limit + 1) as i64,
		)
		.fetch_all(&self.pool)
		.await?;

		let has_more = rows.len() > limit as usize;
		let items: Vec<_> = rows.into_iter().take(limit as usize).collect();
		let next_cursor = if has_more {
			items
				.last()
				.map(|r| encode_cursor(&(r.distance, r.object_id.clone())))
				.transpose()?
		} else {
			None
		};

		Ok((items, next_cursor))
	}
}
