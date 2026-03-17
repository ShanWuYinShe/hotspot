//! 行政区边界仓库：封装 region_boundary 相关查询

use sqlx::PgPool;

use crate::error::Result;

#[derive(Debug, Clone)]
pub struct RegionBoundaryRepo {
	pool: PgPool,
}

#[derive(Debug)]
pub struct RegionBoundaryRow {
	pub region_id: i64,
	pub region_level: i16,
	pub name: String,
	pub centroid_lat: f64,
	pub centroid_lng: f64,
}

impl RegionBoundaryRepo {
	pub fn new(pool: PgPool) -> Self {
		Self { pool }
	}

	/// 根据坐标查找最合适的行政区
	/// 始终选择包含该点的“最小行政区”：
	/// - 使用 PostGIS ST_Contains 判断几何包含；
	/// - 在所有包含该点的 region_boundary 中，按 admin_level 降序选一条，
	///   保证拿到 admin_level 最大（通常范围最小）的那一行。
	pub async fn find_region_for_point(
		&self,
		lat: f64,
		lng: f64,
	) -> Result<Option<RegionBoundaryRow>> {
		let row: Option<_> = sqlx::query!(
				"SELECT region_id, admin_level AS region_level, name,
				        ST_Y(centroid::geometry) AS \"centroid_lat!\",
				        ST_X(centroid::geometry) AS \"centroid_lng!\"\n\
				 FROM region_boundary\n\
				 WHERE ST_Contains(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326))\n\
				 ORDER BY admin_level DESC\n\
				 LIMIT 1",
			lng,
			lat
		)
		.fetch_optional(&self.pool)
			.await?;

		Ok(row.map(|r| RegionBoundaryRow {
			region_id: r.region_id,
			region_level: r.region_level,
			name: r.name,
			centroid_lat: r.centroid_lat,
			centroid_lng: r.centroid_lng,
		}))
	}
}
