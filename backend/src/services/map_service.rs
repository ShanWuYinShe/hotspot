use chrono::{DateTime, FixedOffset, NaiveDate};

use crate::{
	dto::{
		request::{Bounds, MapHeatRequest, MapHotspotsRequest},
		response::{
			common::GeoPoint,
			MapHeatBreakdown,
			MapHeatBucket,
			MapHeatCanvas,
			MapHotspotItem,
			MapHotspotsCanvas,
		},
	},
	error::{ApiError, Result},
		repositories::system::{
			map_repo::{GroupHotspotRow, MapRepository, MomentHotspotRow, UserHotspotRow},
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

	pub async fn render_heatmap(&self, req: MapHeatRequest) -> Result<MapHeatCanvas> {
		let bounds = req.viewport.bounds;
		Self::validate_bounds(bounds)?;

		let zoom = req.viewport.zoom;
		if !(1.0..=20.0).contains(&zoom) {
			return Err(ApiError::BadRequest("zoom 必须在 1 到 20 之间".to_string()));
		}

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

		let span_m = self.region_boundary_service.viewport_span_meters(bounds);
		if span_m <= 3_000.0 {
			return Ok(MapHeatCanvas {
				items: Vec::new(),
				toast_message: Some("当前视野已足够细，请使用热点视图查看具体对象".to_string()),
			});
		}

		// 数据类型过滤：默认 All
		let (include_users, include_groups, include_moments) = match req
			.filter
			.as_ref()
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
				name: Some(row.name),
			});
		}

		Ok(MapHeatCanvas {
			items: buckets,
			toast_message: None,
		})
	}

	pub async fn render_hotspots(
		&self,
		req: MapHotspotsRequest,
		auth_user_id: Option<i64>,
	) -> Result<MapHotspotsCanvas> {
		let bounds = req.viewport.bounds;
		Self::validate_bounds(bounds)?;

		let zoom = req.viewport.zoom;
		if !(1.0..=20.0).contains(&zoom) {
			return Err(ApiError::BadRequest("zoom 必须在 1 到 20 之间".to_string()));
		}

		let span_m = self.region_boundary_service.viewport_span_meters(bounds);
		if span_m > 3_000.0 || zoom < 13.0 {
			return Ok(MapHotspotsCanvas {
				items: Vec::new(),
				next_cursor: None,
				toast_message: Some("当前视野仍然较大，请放大后再查看详细对象".to_string()),
			});
		}

		let _user_id = auth_user_id; // 预留未来按用户过滤

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

		let pagination = req.pagination.as_ref();
		let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 200)? as i32;
		let cursor = pagination.and_then(|p| p.cursor.clone());

		let data_type = req
			.filter
			.as_ref()
			.and_then(|f| f.data_type)
			.unwrap_or(MapDataType::All);

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
				Ok(MapHotspotsCanvas {
					items,
					next_cursor,
					toast_message: None,
				})
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
				Ok(MapHotspotsCanvas {
					items,
					next_cursor,
					toast_message: None,
				})
			}
			MapDataType::Moment | MapDataType::All => {
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
				Ok(MapHotspotsCanvas {
					items,
					next_cursor,
					toast_message: None,
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
