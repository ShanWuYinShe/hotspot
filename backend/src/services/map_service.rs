use chrono::{DateTime, FixedOffset};
use std::f64::consts::PI;

use crate::{
    constants::{MAP_CLUSTER_GRID_PX, MAP_CLUSTER_ZOOM_THRESHOLD, MAP_DETAIL_ENTITY_LIMIT},
    dto::{
        request::{Bounds, MapRenderRequest},
        response::{common::GeoPoint, MapCanvas, MapClusterData, MapEntityData, MapRenderItem},
    },
    error::{ApiError, Result},
    repositories::map_repo::{MapClusterRow, MapEntityRow, MapRepository},
    types::enums::{MapDataType, MapItemType},
    utils::validation::Validator,
};

#[derive(Debug, Clone)]
pub struct MapService {
    map_repo: MapRepository,
}

impl MapService {
    pub fn new(map_repo: MapRepository) -> Self {
        Self { map_repo }
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

    fn grid_meters(bounds: Bounds, zoom: f64, _grid_size: Option<i32>) -> f64 {
        let grid_px = MAP_CLUSTER_GRID_PX;
        let z = zoom.max(0.0).min(22.0);

        // 使用 bbox 中心纬度做 cos 修正
        let center_lat = ((bounds.north + bounds.south) / 2.0).max(-85.0).min(85.0);
        let cos_lat = (center_lat * PI / 180.0).cos().max(0.01);

        // WebMercator：156543.03392 m/px @ zoom=0（赤道）
        let meters_per_px = 156543.03392 * cos_lat / 2_f64.powf(z);
        (grid_px * meters_per_px).max(10.0)
    }

    /// Map render. When `social_filter_user_id` is `Some(user_id)`, filters results to
    /// friends of that user (bidirectional follows) for User and Moment types.
    pub async fn render_map(
        &self,
        req: MapRenderRequest,
        social_filter_user_id: Option<i64>,
    ) -> Result<MapCanvas> {
        Self::validate_bounds(req.viewport.bounds)?;

        let zoom = req.viewport.zoom;
        if !(1.0..=20.0).contains(&zoom) {
            return Err(ApiError::BadRequest("zoom 必须在 1 到 20 之间".to_string()));
        }
        let bounds = req.viewport.bounds;
        let filter_type = req
            .filter
            .as_ref()
            .and_then(|f| f.data_type)
            .unwrap_or(MapDataType::All);

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

        let mut items = Vec::new();

        if zoom <= MAP_CLUSTER_ZOOM_THRESHOLD {
            let grid_meters = Self::grid_meters(bounds, zoom, None);

            if filter_type == MapDataType::All || filter_type == MapDataType::User {
                let users = if let Some(uid) = social_filter_user_id {
                    self.map_repo
                        .cluster_friend_users_by_bounds(
                            uid,
                            bounds.west,
                            bounds.south,
                            bounds.east,
                            bounds.north,
                            grid_meters,
                        )
                        .await?
                } else {
                    self.map_repo
                        .cluster_users_by_bounds(
                            bounds.west,
                            bounds.south,
                            bounds.east,
                            bounds.north,
                            grid_meters,
                        )
                        .await?
                };
                items.extend(
                    users
                        .into_iter()
                        .map(|c| self.cluster_to_item(c, MapDataType::User.as_str())),
                );
            }

            if filter_type == MapDataType::All || filter_type == MapDataType::Group {
                let groups = self
                    .map_repo
                    .cluster_groups_by_bounds(
                        bounds.west,
                        bounds.south,
                        bounds.east,
                        bounds.north,
                        grid_meters,
                    )
                    .await?;
                items.extend(
                    groups
                        .into_iter()
                        .map(|c| self.cluster_to_item(c, MapDataType::Group.as_str())),
                );
            }

            if filter_type == MapDataType::All || filter_type == MapDataType::Moment {
                let moments = if let Some(uid) = social_filter_user_id {
                    self.map_repo
                        .cluster_friend_moments_by_bounds(
                            uid,
                            bounds.west,
                            bounds.south,
                            bounds.east,
                            bounds.north,
                            grid_meters,
                            start_time.clone(),
                            end_time.clone(),
                        )
                        .await?
                } else {
                    self.map_repo
                        .cluster_moments_by_bounds(
                            bounds.west,
                            bounds.south,
                            bounds.east,
                            bounds.north,
                            grid_meters,
                            start_time.clone(),
                            end_time.clone(),
                        )
                        .await?
                };
                items.extend(
                    moments
                        .into_iter()
                        .map(|c| self.cluster_to_item(c, MapDataType::Moment.as_str())),
                );
            }
        } else {
            let limit = MAP_DETAIL_ENTITY_LIMIT;

            if filter_type == MapDataType::All || filter_type == MapDataType::User {
                let users = if let Some(uid) = social_filter_user_id {
                    self.map_repo
                        .search_friend_users_by_bounds(
                            uid,
                            bounds.west,
                            bounds.south,
                            bounds.east,
                            bounds.north,
                            limit,
                        )
                        .await?
                } else {
                    self.map_repo
                        .search_users_by_bounds(
                            bounds.west,
                            bounds.south,
                            bounds.east,
                            bounds.north,
                            limit,
                        )
                        .await?
                };
                items.extend(
                    users
                        .into_iter()
                        .map(|u| self.entity_to_item(u, MapDataType::User.as_str())),
                );
            }

            if filter_type == MapDataType::All || filter_type == MapDataType::Group {
                let groups = self
                    .map_repo
                    .search_groups_by_bounds(
                        bounds.west,
                        bounds.south,
                        bounds.east,
                        bounds.north,
                        limit,
                    )
                    .await?;
                items.extend(
                    groups
                        .into_iter()
                        .map(|g| self.entity_to_item(g, MapDataType::Group.as_str())),
                );
            }

            if filter_type == MapDataType::All || filter_type == MapDataType::Moment {
                let moments = if let Some(uid) = social_filter_user_id {
                    self.map_repo
                        .search_friend_moments_by_bounds(
                            uid,
                            bounds.west,
                            bounds.south,
                            bounds.east,
                            bounds.north,
                            limit,
                            start_time.clone(),
                            end_time.clone(),
                        )
                        .await?
                } else {
                    self.map_repo
                        .search_moments_by_bounds(
                            bounds.west,
                            bounds.south,
                            bounds.east,
                            bounds.north,
                            limit,
                            start_time.clone(),
                            end_time.clone(),
                        )
                        .await?
                };
                items.extend(
                    moments
                        .into_iter()
                        .map(|m| self.entity_to_item(m, MapDataType::Moment.as_str())),
                );
            }
        }

        Ok(MapCanvas {
            items,
            toast_message: None,
        })
    }

    fn geo_point(latitude: f64, longitude: f64) -> GeoPoint {
        GeoPoint {
            latitude,
            longitude,
        }
    }

    fn cluster_to_item(&self, row: MapClusterRow, type_name: &str) -> MapRenderItem {
        // 将预览 URL 转换为 Option<Vec<String>>
        let preview_urls = if row.preview_urls.is_empty() {
            None
        } else {
            Some(row.preview_urls)
        };

        MapRenderItem {
            render_id: format!("cluster:{}:{}", type_name, row.key),
            item_type: MapItemType::Cluster,
            location: Self::geo_point(row.latitude, row.longitude),
            cluster: Some(MapClusterData {
                count: row.count,
                label: Some(format!("{} {}", row.count, type_name)),
                preview_urls,
            }),
            entity: None,
        }
    }

    fn entity_to_item(&self, row: MapEntityRow, type_name: &str) -> MapRenderItem {
        let item_type = match type_name {
            "user" => MapItemType::User,
            "group" => MapItemType::Group,
            "moment" => MapItemType::Moment,
            _ => MapItemType::User, // Default fallback
        };
        MapRenderItem {
            render_id: format!("{}:{}", type_name, row.id),
            item_type,
            location: Self::geo_point(row.latitude, row.longitude),
            cluster: None,
            entity: Some(MapEntityData {
                title: row.title,
                subtitle: row.subtitle,
                cover_url: row.cover_url,
                payload_id: row.id,
                category: Some(type_name.to_string()),
            }),
        }
    }
}
