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

	/// 计算视口跨度（米），取经度和纬度方向最大值
	pub fn viewport_span_meters(&self, bounds: Bounds) -> f64 {
		let center_lat = (bounds.north + bounds.south) / 2.0;
		let lat_span = (bounds.north - bounds.south).abs() * 111_320.0;
		let lng_span = (bounds.east - bounds.west).abs() * 111_320.0 * (center_lat.to_radians().cos().max(0.01));
		lat_span.max(lng_span)
	}

}
