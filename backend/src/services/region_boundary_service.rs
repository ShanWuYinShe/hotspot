//! 行政区边界服务：负责坐标归属和视口跨度计算

use crate::{
	dto::request::Bounds,
	repositories::system::region_boundary_repo::{RegionBoundaryRepo, RegionBoundaryRow},
};

#[derive(Debug, Clone)]
pub struct RegionBoundaryService {
	repo: RegionBoundaryRepo,
}

impl RegionBoundaryService {
	pub fn new(repo: RegionBoundaryRepo) -> Self {
		Self { repo }
	}

	/// 根据坐标返回主行政区 ID
	pub async fn assign_region_for_point(&self, lat: f64, lng: f64) -> Option<i64> {
		self
			.repo
			.find_region_for_point(lat, lng)
			.await
			.ok()
			.flatten()
			.map(|r: RegionBoundaryRow| r.region_id)
	}

	/// 计算视口跨度（米），取经度和纬度方向最大值（目前仅用于近似 UI 逻辑）
	pub fn viewport_span_meters(&self, bounds: Bounds) -> f64 {
		let center_lat = (bounds.north + bounds.south) / 2.0;
		let lat_span = (bounds.north - bounds.south).abs() * 111_320.0;
		let lng_span = (bounds.east - bounds.west).abs() * 111_320.0
			* (center_lat.to_radians().cos().max(0.01));
		lat_span.max(lng_span)
	}

	/// 基于视口 bounds 计算 region_min / region_max
	///
	/// - region_max 始终来自与视口有交集的行政区（IntersectSet）中 region_level 最大值；
	/// - region_min 优先从 InsideSet 获取，其次是 CoverSet，最后退化为 IntersectSet。
	/// - 当 IntersectSet 为空时，返回 Ok(None)。
	pub async fn compute_region_min_max(&self, bounds: Bounds) -> crate::error::Result<Option<(i16, i16)>> {
		let samples = self
			.repo
			.list_region_levels_for_viewport(bounds.west, bounds.south, bounds.east, bounds.north)
			.await?;
		let intersect_levels: Vec<i16> = samples.iter().filter(|s| s.intersects).map(|s| s.region_level).collect();
		tracing::debug!(
			north = bounds.north,
			south = bounds.south,
			east = bounds.east,
			west = bounds.west,
			intersect_count = intersect_levels.len(),
			"map.canvas.region_levels.intersect_loaded",
		);
		if intersect_levels.is_empty() {
			tracing::info!(
				north = bounds.north,
				south = bounds.south,
				east = bounds.east,
				west = bounds.west,
				"map.canvas.region_levels.no_intersect_regions_for_viewport",
			);
			return Ok(None);
		}

		let region_max = *intersect_levels
			.iter()
			.max()
			.expect("non-empty intersect_levels has a max");

		let inside_levels: Vec<i16> = samples.iter().filter(|s| s.inside).map(|s| s.region_level).collect();
		let cover_levels: Vec<i16> = samples.iter().filter(|s| s.covers).map(|s| s.region_level).collect();

		// region_min 的语义：在“完全在视口内”或“完全覆盖视口”的行政区中，选择 region_level 最小（区域最大）的那个；
		// 若两者都为空，则退化为使用 intersect 集合中的最小 region_level。
		let region_min = if !inside_levels.is_empty() {
			*inside_levels
				.iter()
				.min()
				.expect("non-empty inside_levels has a min")
		} else if !cover_levels.is_empty() {
			*cover_levels
				.iter()
				.min()
				.expect("non-empty cover_levels has a min")
		} else {
			*intersect_levels
				.iter()
				.min()
				.expect("non-empty intersect_levels has a min")
		};

		// 为调试方便，直接从 samples 中采样 region_min/region_max 对应的一小部分行政区名称
		let sample_min_names: Vec<String> = samples
			.iter()
			.filter(|s| s.region_level == region_min)
			.take(3)
			.map(|s| s.name.clone())
			.collect();
		let sample_max_names: Vec<String> = samples
			.iter()
			.filter(|s| s.region_level == region_max)
			.take(3)
			.map(|s| s.name.clone())
			.collect();

		tracing::info!(
			north = bounds.north,
			south = bounds.south,
			east = bounds.east,
			west = bounds.west,
			intersect_count = intersect_levels.len(),
			inside_count = inside_levels.len(),
			cover_count = cover_levels.len(),
			region_min,
			region_max,
			region_min_sample_names = ?sample_min_names,
			region_max_sample_names = ?sample_max_names,
			"map.canvas.region_levels.resolved_region_min_max",
		);

		Ok(Some((region_min, region_max)))
	}
}
