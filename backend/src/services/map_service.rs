use chrono::{DateTime, FixedOffset, NaiveDate};

use crate::{
	dto::{
		request::{Bounds, MapCanvasRequest},
		response::{
			common::GeoPoint,
			MapCanvasResponse,
			MapHeatBreakdown,
			MapHeatBucket,
			MapHeatCanvas,
			MapHotspotItem,
			MapHotspotsCanvas,
		},
	},
	error::{ApiError, Result},
		repositories::system::{
			map_repo::{GroupHotspotRow, MapRepository, MomentHotspotRow, UnifiedHotspotRow, UserHotspotRow},
			region_boundary_stats_repo::{RegionBoundaryStatsRepository, RegionHeatRow},
		},
		services::region_boundary_service::RegionBoundaryService,
		types::enums::MapDataType,
		utils::validation::Validator,
	};

#[derive(Debug, Clone)]
pub struct MapService {
		region_boundary_service: RegionBoundaryService,
		region_boundary_stats_repo: RegionBoundaryStatsRepository,
	map_repo: MapRepository,
}

impl MapService {
	pub fn new(
			region_boundary_service: RegionBoundaryService,
			region_boundary_stats_repo: RegionBoundaryStatsRepository,
			map_repo: MapRepository,
		) -> Self {
			Self {
				region_boundary_service,
				region_boundary_stats_repo,
				map_repo,
			}
		}

	fn validate_bounds(bounds: Bounds) -> Result<()> {
		Validator::validate_coordinates(bounds.north, bounds.east)?;
		Validator::validate_coordinates(bounds.south, bounds.west)?;

		if bounds.north < bounds.south {
			return Err(ApiError::BadRequest(
				"bounds.north 必须 >= bounds.south".to_string(),
			));
		}
		if bounds.east < bounds.west {
			return Err(ApiError::BadRequest(
				"bounds.east 必须 >= bounds.west".to_string(),
			));
		}
		Ok(())
	}

	fn validate_time_range(start_time: Option<&str>, end_time: Option<&str>) -> Result<()> {
		let parsed_start = match start_time {
			Some(v) => Some(DateTime::<FixedOffset>::parse_from_rfc3339(v).map_err(|_| {
				ApiError::BadRequest(
					"filter.time_range.start_time 格式错误，必须为 UTC ISO 8601".to_string(),
				)
			})?),
			None => None,
		};

		let parsed_end = match end_time {
			Some(v) => Some(DateTime::<FixedOffset>::parse_from_rfc3339(v).map_err(|_| {
				ApiError::BadRequest(
					"filter.time_range.end_time 格式错误，必须为 UTC ISO 8601".to_string(),
				)
			})?),
			None => None,
		};

		if let (Some(start), Some(end)) = (parsed_start, parsed_end) {
			if start > end {
				return Err(ApiError::BadRequest(
					"filter.time_range.start_time 必须 <= end_time".to_string(),
				));
			}
		}

		Ok(())
	}

	async fn build_heatmap(
		&self,
		bounds: Bounds,
		filter: Option<&crate::dto::request::MapCanvasFilter>,
		start_time: Option<String>,
		end_time: Option<String>,
	) -> Result<MapHeatCanvas> {
		let (include_users, include_groups, include_moments) = match filter
			.and_then(|f| f.data_type)
		{
			Some(MapDataType::User) => (true, false, false),
			Some(MapDataType::Group) => (false, true, false),
			Some(MapDataType::Moment) => (false, false, true),
			Some(MapDataType::All) | None => (true, true, true),
		};

		let (start_date, end_date): (Option<NaiveDate>, Option<NaiveDate>) = (
			start_time
				.as_deref()
				.and_then(|s| DateTime::<FixedOffset>::parse_from_rfc3339(s).ok())
				.map(|dt| dt.date_naive()),
			end_time
				.as_deref()
				.and_then(|s| DateTime::<FixedOffset>::parse_from_rfc3339(s).ok())
				.map(|dt| dt.date_naive()),
		);

		let center_lat = (bounds.north + bounds.south) / 2.0;
		let center_lng = (bounds.east + bounds.west) / 2.0;

		let rows: Vec<RegionHeatRow> = self
			.region_boundary_stats_repo
			.aggregate_for_heatmap(
				bounds.west,
				bounds.south,
				bounds.east,
				bounds.north,
				center_lat,
				center_lng,
				start_date,
				end_date,
			)
			.await?;

		let mut buckets: Vec<MapHeatBucket> = Vec::with_capacity(rows.len());
		for row in rows {
			let mut user_count = row.user_count;
			let mut group_count = row.group_count;
			let mut moment_count = row.moment_count;
			if !include_users {
				user_count = 0;
			}
			if !include_groups {
				group_count = 0;
			}
			if !include_moments {
				moment_count = 0;
			}

			let total = user_count + group_count + moment_count;
			if total == 0 {
				continue;
			}

			let breakdown = if include_users || include_groups || include_moments {
				Some(MapHeatBreakdown {
					user: if include_users && user_count > 0 {
						Some(user_count)
					} else {
						None
					},
					group: if include_groups && group_count > 0 {
						Some(group_count)
					} else {
						None
					},
					moment: if include_moments && moment_count > 0 {
						Some(moment_count)
					} else {
						None
					},
				})
			} else {
				None
			};

			buckets.push(MapHeatBucket {
				location: GeoPoint {
					latitude: row.centroid_lat,
					longitude: row.centroid_lng,
				},
				total_count: total,
				breakdown,
			});
		}

		const MAX_REGIONS: usize = 500;
		if buckets.len() > MAX_REGIONS {
			buckets.sort_by(|a, b| b.total_count.cmp(&a.total_count));
			buckets.truncate(MAX_REGIONS);
		}

		Ok(MapHeatCanvas { items: buckets })
	}

	async fn build_hotspots(
		&self,
		bounds: Bounds,
		filter: Option<&crate::dto::request::MapCanvasFilter>,
		pagination: Option<&crate::dto::request::PaginationParams>,
		auth_user_id: Option<i64>,
		start_time: Option<String>,
		end_time: Option<String>,
	) -> Result<MapHotspotsCanvas> {
		let _user_id = auth_user_id; // 预留未来按用户过滤

		let data_type = filter
			.and_then(|f| f.data_type)
			.unwrap_or(MapDataType::All);

		let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 200)? as i32;
		let cursor = pagination.and_then(|p| p.cursor.clone());

		match data_type {
			MapDataType::User => {
				let (rows, next_cursor): (Vec<UserHotspotRow>, Option<String>) = self
					.map_repo
					.list_user_hotspots(
						bounds.west,
						bounds.south,
						bounds.east,
						bounds.north,
						start_time,
						end_time,
						limit,
						cursor,
					)
					.await?;
				let items: Vec<MapHotspotItem> = rows
					.into_iter()
					.map(|r: UserHotspotRow| MapHotspotItem {
						object_type: "user".to_string(),
						object_id: r.user_id.to_string(),
						title: r.nickname.unwrap_or_default(),
						subtitle: r.location_name,
						cover_url: r.avatar_url,
						location: GeoPoint {
							latitude: r.latitude,
							longitude: r.longitude,
						},
						created_at: Some(r.created_at.to_rfc3339()),
						distance_meters: Some(r.distance),
					})
					.collect();
				Ok(MapHotspotsCanvas { items, next_cursor })
			}
			MapDataType::Group => {
				let (rows, next_cursor): (Vec<GroupHotspotRow>, Option<String>) = self
					.map_repo
					.list_group_hotspots(
						bounds.west,
						bounds.south,
						bounds.east,
						bounds.north,
						start_time,
						end_time,
						limit,
						cursor,
					)
					.await?;
				let items: Vec<MapHotspotItem> = rows
					.into_iter()
					.map(|r: GroupHotspotRow| MapHotspotItem {
						object_type: "group".to_string(),
						object_id: r.group_id.to_string(),
						title: r.name,
						subtitle: Some(r.location_name),
						cover_url: r.avatar_url,
						location: GeoPoint {
							latitude: r.latitude,
							longitude: r.longitude,
						},
						created_at: Some(r.created_at.to_rfc3339()),
						distance_meters: Some(r.distance),
					})
					.collect();
				Ok(MapHotspotsCanvas { items, next_cursor })
			}
			MapDataType::Moment => {
				let (rows, next_cursor): (Vec<MomentHotspotRow>, Option<String>) = self
					.map_repo
					.list_moment_hotspots(
						bounds.west,
						bounds.south,
						bounds.east,
						bounds.north,
						start_time,
						end_time,
						limit,
						cursor,
					)
					.await?;
				let items: Vec<MapHotspotItem> = rows
					.into_iter()
					.map(|r: MomentHotspotRow| MapHotspotItem {
						object_type: "moment".to_string(),
						object_id: r.moment_id.to_string(),
						title: r
							.description
							.as_deref()
							.unwrap_or("")
							.chars()
							.take(40)
							.collect(),
						subtitle: r.location_name,
						cover_url: None,
						location: GeoPoint {
							latitude: r.latitude,
							longitude: r.longitude,
						},
						created_at: Some(r.created_at.to_rfc3339()),
						distance_meters: Some(r.distance),
					})
					.collect();
				Ok(MapHotspotsCanvas { items, next_cursor })
			}
			MapDataType::All => {
				let (rows, next_cursor): (Vec<UnifiedHotspotRow>, Option<String>) = self
					.map_repo
					.list_all_hotspots(
						bounds.west,
						bounds.south,
						bounds.east,
						bounds.north,
						start_time,
						end_time,
						limit,
						cursor,
					)
					.await?;
				let items: Vec<MapHotspotItem> = rows
					.into_iter()
					.map(|r: UnifiedHotspotRow| MapHotspotItem {
						object_type: r.object_type,
						object_id: r.object_id,
						title: r.title,
						subtitle: r.subtitle,
						cover_url: r.cover_url,
						location: GeoPoint {
							latitude: r.latitude,
							longitude: r.longitude,
						},
						created_at: Some(r.created_at.to_rfc3339()),
						distance_meters: Some(r.distance),
					})
					.collect();
				Ok(MapHotspotsCanvas { items, next_cursor })
			}
		}
	}

	/// 统一地图画布渲染：根据 region_level 跨度自动选择返回统计或明细
	pub async fn render_canvas(
		&self,
		req: MapCanvasRequest,
		auth_user_id: Option<i64>,
	) -> Result<MapCanvasResponse> {
		let bounds = req.viewport.bounds;
		Self::validate_bounds(bounds)?;

		let data_type_str = req
			.filter
			.as_ref()
			.and_then(|f| f.data_type)
			.map(|t| match t {
				MapDataType::User => "user",
				MapDataType::Group => "group",
				MapDataType::Moment => "moment",
				MapDataType::All => "all",
			});
		tracing::info!(
			north = bounds.north,
			south = bounds.south,
			east = bounds.east,
			west = bounds.west,
			data_type = data_type_str,
			has_pagination = req.pagination.is_some(),
			"map.canvas.request.received",
		);

		let (start_time, end_time) = if let Some(f) = &req.filter {
			if let Some(tr) = &f.time_range {
				(tr.start_time.clone(), tr.end_time.clone())
			} else {
				(None, None)
			}
		} else {
			(None, None)
		};
		Self::validate_time_range(start_time.as_deref(), end_time.as_deref())?;

		let region_levels = self
			.region_boundary_service
			.compute_region_min_max(bounds)
			.await?;

		let Some((region_min, region_max)) = region_levels else {
			tracing::info!(
				north = bounds.north,
				south = bounds.south,
				east = bounds.east,
				west = bounds.west,
				"map.canvas.response.empty_no_admin_region",
			);
			return Ok(MapCanvasResponse {
				heatmap: None,
				hotspots: None,
			});
		};

		let diff = region_max - region_min;
		tracing::info!(
			north = bounds.north,
			south = bounds.south,
			east = bounds.east,
			west = bounds.west,
			region_min,
			region_max,
			region_diff = diff,
			"map.canvas.region_diff.resolved",
		);

		if diff > 2 {
			let canvas = self
				.build_heatmap(bounds, req.filter.as_ref(), start_time, end_time)
				.await?;
			if canvas.items.is_empty() {
				tracing::info!(
					north = bounds.north,
					south = bounds.south,
					east = bounds.east,
					west = bounds.west,
					region_min,
					region_max,
					region_diff = diff,
					mode = "heatmap",
					"map.canvas.response.empty_after_heatmap",
				);
				Ok(MapCanvasResponse {
					heatmap: None,
					hotspots: None,
				})
			} else {
				tracing::info!(
					north = bounds.north,
					south = bounds.south,
					east = bounds.east,
					west = bounds.west,
					region_min,
					region_max,
					region_diff = diff,
					items = canvas.items.len(),
					mode = "heatmap",
					"map.canvas.response.heatmap_non_empty",
				);
				Ok(MapCanvasResponse {
					heatmap: Some(canvas),
					hotspots: None,
				})
			}
		} else {
			let canvas = self
				.build_hotspots(
					bounds,
					req.filter.as_ref(),
					req.pagination.as_ref(),
					auth_user_id,
					start_time,
					end_time,
				)
				.await?;
			if canvas.items.is_empty() {
				tracing::info!(
					north = bounds.north,
					south = bounds.south,
					east = bounds.east,
					west = bounds.west,
					region_min,
					region_max,
					region_diff = diff,
					mode = "hotspots",
					"map.canvas.response.empty_after_hotspots",
				);
				Ok(MapCanvasResponse {
					heatmap: None,
					hotspots: None,
				})
			} else {
				tracing::info!(
					north = bounds.north,
					south = bounds.south,
					east = bounds.east,
					west = bounds.west,
					region_min,
					region_max,
					region_diff = diff,
					items = canvas.items.len(),
					mode = "hotspots",
					"map.canvas.response.hotspots_non_empty",
				);
				Ok(MapCanvasResponse {
					heatmap: None,
					hotspots: Some(canvas),
				})
			}
		}
	}

	/// 供定时任务/维护脚本调用：重算某一天的行政区统计
	pub async fn rebuild_stats_for_date(&self, date: NaiveDate) -> Result<()> {
		self
			.region_boundary_stats_repo
			.rebuild_stats_for_date(date)
			.await
	}
}
