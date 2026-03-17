//! 行政区边界日级统计仓库：负责 region_boundary_stats_daily 的重算与聚合

use chrono::NaiveDate;
use sqlx::PgPool;

use crate::error::Result;

#[derive(Debug, Clone)]
pub struct RegionBoundaryStatsRepository {
	pool: PgPool,
}

#[derive(Debug)]
pub struct RegionHeatRow {
	pub region_id: i64,
	pub region_level: i16,
	pub name: String,
	pub centroid_lat: f64,
	pub centroid_lng: f64,
	pub user_count: i64,
	pub group_count: i64,
	pub moment_count: i64,
}

impl RegionBoundaryStatsRepository {
	pub fn new(pool: PgPool) -> Self {
		Self { pool }
	}

	/// 幂等重算某一天的 region 统计（以 created_at::date 为维度）
	pub async fn rebuild_stats_for_date(&self, date: NaiveDate) -> Result<()> {
		let mut tx = self.pool.begin().await?;

		// 先删除当日旧记录，保证重算幂等
		sqlx::query!(
			"DELETE FROM region_boundary_stats_daily WHERE date = $1",
			date
		)
		.execute(&mut *tx)
		.await?;

		// 按用户新增量聚合
		sqlx::query!(
			r#"
			INSERT INTO region_boundary_stats_daily (date, region_id, user_count, group_count, moment_count)
			SELECT
				$1 as date,
				region_id,
				COUNT(*)::bigint as user_count,
				0::bigint as group_count,
				0::bigint as moment_count
			FROM users
			WHERE deleted_at IS NULL
			  AND region_id IS NOT NULL
			  AND created_at::date = $1
			GROUP BY region_id
			ON CONFLICT (date, region_id)
			DO UPDATE SET user_count = region_boundary_stats_daily.user_count + EXCLUDED.user_count
			"#,
			date
		)
		.execute(&mut *tx)
		.await?;

		// 按群组新增量聚合
		sqlx::query!(
			r#"
			INSERT INTO region_boundary_stats_daily (date, region_id, user_count, group_count, moment_count)
			SELECT
				$1 as date,
				region_id,
				0::bigint as user_count,
				COUNT(*)::bigint as group_count,
				0::bigint as moment_count
			FROM groups
			WHERE dismissed_at IS NULL
			  AND region_id IS NOT NULL
			  AND created_at::date = $1
			GROUP BY region_id
			ON CONFLICT (date, region_id)
			DO UPDATE SET group_count = region_boundary_stats_daily.group_count + EXCLUDED.group_count
			"#,
			date
		)
		.execute(&mut *tx)
		.await?;

		// 按动态新增量聚合（使用 moments 视图，而非分区底表）
		sqlx::query!(
			r#"
			INSERT INTO region_boundary_stats_daily (date, region_id, user_count, group_count, moment_count)
			SELECT
				$1 as date,
				region_id,
				0::bigint as user_count,
				0::bigint as group_count,
				COUNT(*)::bigint as moment_count
			FROM moments_partitioned
			WHERE deleted_at IS NULL
			  AND region_id IS NOT NULL
			  AND created_at::date = $1
			GROUP BY region_id
			ON CONFLICT (date, region_id)
			DO UPDATE SET moment_count = region_boundary_stats_daily.moment_count + EXCLUDED.moment_count
			"#,
			date
		)
		.execute(&mut *tx)
		.await?;

		tx.commit().await?;
		Ok(())
	}

	/// 按行政区聚合一段时间内的统计，用于 /map/heatmap
	///
	/// 设计：
	/// - region_id 始终挂在“最小行政区”（assign_region_for_point 已保证）；
	/// - 统计写入 region_boundary_stats_daily 也是按最小行政区写；
	/// - 在 heatmap 聚合时，根据视口和中心位置选择一批“要展示的行政区”（display_regions），
	///   然后对每个展示行政区，汇总其几何范围内所有子区域（包含自身以外）的统计：
	///   - 子区域判断只依赖几何（ST_Contains(parent.geom, child.centroid)），不强依赖 admin_level；
	///   - 排除 child.region_id = parent.region_id，避免把自己当成自己的子区域；
	pub async fn aggregate_for_heatmap(
		&self,
		west: f64,
		south: f64,
		east: f64,
		north: f64,
		center_lat: f64,
		center_lng: f64,
		start_date: Option<NaiveDate>,
		end_date: Option<NaiveDate>,
	) -> Result<Vec<RegionHeatRow>> {
		let rows: Vec<_> = sqlx::query!(
				 r#"
			WITH bbox AS (
			  SELECT ST_MakeEnvelope($1, $2, $3, $4, 4326) AS geom
			),
			center AS (
			  SELECT ST_SetSRID(ST_MakePoint($5, $6), 4326) AS geom
			),
			visible_regions AS (
			  SELECT
			    rb.region_id,
			    rb.admin_level AS region_level,
			    rb.name,
			    rb.centroid,
			    rb.geom
			  FROM region_boundary rb, bbox
			  WHERE ST_Intersects(rb.geom, bbox.geom)
			),
			global_level AS (
			  SELECT MIN(region_level) AS level FROM visible_regions
			),
			center_region AS (
			  SELECT vr.region_level
			  FROM visible_regions vr, center
			  WHERE ST_Contains(vr.geom, center.geom)
			  ORDER BY vr.region_level ASC
			  LIMIT 1
			),
			levels AS (
			  SELECT
			    gl.level AS global_level,
			    (
			      SELECT MIN(vr.region_level)
			      FROM visible_regions vr
			      WHERE center_region.region_level IS NOT NULL
			        AND vr.region_level > gl.level
			    ) AS center_level
			  FROM global_level gl
			  LEFT JOIN center_region ON TRUE
			),
			display_regions AS (
			  SELECT
			    vr.region_id,
			    vr.region_level,
			    vr.name,
			    vr.centroid,
			    vr.geom
			  FROM visible_regions vr, levels l
			  WHERE
			    -- 中心附近：若存在更细一级 center_level，则使用该级别；否则使用 global_level
			    vr.region_level = COALESCE(l.center_level, l.global_level)
			    OR
			    -- 其他区域：使用 global_level
			    vr.region_level = l.global_level
			),
			children AS (
			  SELECT
			    p.region_id AS parent_region_id,
			    c.region_id AS child_region_id
			  FROM display_regions p
			  JOIN region_boundary c
			    ON ST_Contains(p.geom, c.centroid)
			   AND c.region_id <> p.region_id
			)
			SELECT
			  p.region_id,
			  p.region_level,
			  p.name,
			  ST_Y(p.centroid::geometry) AS "centroid_lat!",
			  ST_X(p.centroid::geometry) AS "centroid_lng!",
			  COALESCE(SUM(s.user_count), 0)::bigint   AS "user_count!",
			  COALESCE(SUM(s.group_count), 0)::bigint  AS "group_count!",
			  COALESCE(SUM(s.moment_count), 0)::bigint AS "moment_count!"
			FROM display_regions p
			LEFT JOIN children ch ON ch.parent_region_id = p.region_id
			LEFT JOIN region_boundary_stats_daily s
			  ON s.region_id = ch.child_region_id
			 AND ($7::date IS NULL OR s.date >= $7)
			 AND ($8::date IS NULL OR s.date <= $8)
			GROUP BY p.region_id, p.region_level, p.name, p.centroid
			"#,
			west,
			south,
			east,
			north,
			center_lng,
			center_lat,
			start_date,
			end_date
		)
		.fetch_all(&self.pool)
		.await?;

		Ok(rows
			.into_iter()
			.map(|r| RegionHeatRow {
				region_id: r.region_id,
				region_level: r.region_level,
				name: r.name,
				centroid_lat: r.centroid_lat,
				centroid_lng: r.centroid_lng,
				user_count: r.user_count,
				group_count: r.group_count,
				moment_count: r.moment_count,
			})
			.collect())
	}
}
