use chrono::{DateTime, FixedOffset};
use std::f64::consts::PI;

use crate::{
    constants::{MAP_CLUSTER_GRID_PX, MAP_RENDER_ITEM_LIMIT},
    dto::{
        request::{Bounds, MapRenderRequest},
        response::{common::GeoPoint, MapCanvas, MapClusterData, MapEntityData, MapRenderItem},
    },
    error::{ApiError, Result},
    repositories::map_repo::{MapClusterRow, MapEntityRow, MapRepository},
    types::enums::{MapDataType, MapItemType},
    utils::validation::Validator,
};

const HIGH_ZOOM_ENTITY_DETAIL_THRESHOLD: f64 = 16.0;

#[derive(Debug, Clone)]
pub struct MapService {
    map_repo: MapRepository,
}

#[derive(Debug, Clone, Copy, Default)]
struct VisibleEntityCounts {
    users: i64,
    groups: i64,
    moments: i64,
}

impl VisibleEntityCounts {
    fn total(&self) -> i64 {
        self.users + self.groups + self.moments
    }
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

    fn cluster_grid_ceiling(bounds: Bounds) -> f64 {
        let center_lat = ((bounds.north + bounds.south) / 2.0).max(-85.0).min(85.0);
        let cos_lat = (center_lat * PI / 180.0).cos().max(0.01);
        let lat_span_meters = (bounds.north - bounds.south).abs() * 111_320.0;
        let lng_span_meters = (bounds.east - bounds.west).abs() * 111_320.0 * cos_lat;
        lat_span_meters.max(lng_span_meters).max(10.0)
    }

    fn grow_cluster_grid(current_grid_meters: f64, ceiling: f64) -> f64 {
        (current_grid_meters * 2.0).min(ceiling).max(10.0)
    }

    fn should_return_entity_items(total_entities: i64, zoom: f64) -> bool {
        total_entities <= MAP_RENDER_ITEM_LIMIT as i64 || zoom >= HIGH_ZOOM_ENTITY_DETAIL_THRESHOLD
    }

    fn capped_entity_counts(counts: VisibleEntityCounts, limit: usize) -> VisibleEntityCounts {
        let limit = limit as i64;
        if counts.total() <= limit {
            return counts;
        }

        let entries = [counts.users, counts.groups, counts.moments];
        let total = counts.total() as f64;
        let active = entries.iter().filter(|count| **count > 0).count() as i64;
        let reserved_floor = limit.min(active);
        let proportional_budget = (limit - reserved_floor).max(0);

        let mut base = [0_i64; 3];
        let mut fractions = [0.0_f64; 3];
        let mut used = 0_i64;

        for (index, count) in entries.iter().enumerate() {
            if *count <= 0 {
                continue;
            }
            base[index] = 1;
            let raw_extra = ((*count as f64 / total) * proportional_budget as f64).max(0.0);
            let extra = raw_extra.floor() as i64;
            base[index] = (base[index] + extra).min(*count);
            fractions[index] = raw_extra - extra as f64;
            used += base[index];
        }

        while used < limit {
            let next_index = (0..entries.len())
                .filter(|index| entries[*index] > base[*index])
                .max_by(|left, right| {
                    fractions[*left]
                        .partial_cmp(&fractions[*right])
                        .unwrap_or(std::cmp::Ordering::Equal)
                });

            let Some(index) = next_index else {
                break;
            };

            base[index] += 1;
            fractions[index] = 0.0;
            used += 1;
        }

        VisibleEntityCounts {
            users: base[0],
            groups: base[1],
            moments: base[2],
        }
    }

    fn entity_limit(count: i64) -> i32 {
        count.clamp(0, i32::MAX as i64) as i32
    }

    async fn count_visible_entities(
        &self,
        bounds: Bounds,
        filter_type: MapDataType,
        social_filter_user_id: Option<i64>,
        start_time: Option<String>,
        end_time: Option<String>,
    ) -> Result<VisibleEntityCounts> {
        let mut counts = VisibleEntityCounts::default();

        if filter_type == MapDataType::All || filter_type == MapDataType::User {
            counts.users = if let Some(uid) = social_filter_user_id {
                self.map_repo
                    .count_friend_users_by_bounds(uid, bounds.west, bounds.south, bounds.east, bounds.north)
                    .await?
            } else {
                self.map_repo
                    .count_users_by_bounds(bounds.west, bounds.south, bounds.east, bounds.north)
                    .await?
            };
        }

        if filter_type == MapDataType::All || filter_type == MapDataType::Group {
            counts.groups = self
                .map_repo
                .count_groups_by_bounds(bounds.west, bounds.south, bounds.east, bounds.north)
                .await?;
        }

        if filter_type == MapDataType::All || filter_type == MapDataType::Moment {
            counts.moments = if let Some(uid) = social_filter_user_id {
                self.map_repo
                    .count_friend_moments_by_bounds(
                        uid,
                        bounds.west,
                        bounds.south,
                        bounds.east,
                        bounds.north,
                        start_time,
                        end_time,
                    )
                    .await?
            } else {
                self.map_repo
                    .count_moments_by_bounds(
                        bounds.west,
                        bounds.south,
                        bounds.east,
                        bounds.north,
                        start_time,
                        end_time,
                    )
                    .await?
            };
        }

        Ok(counts)
    }

    async fn fetch_entity_items(
        &self,
        bounds: Bounds,
        filter_type: MapDataType,
        social_filter_user_id: Option<i64>,
        counts: VisibleEntityCounts,
        start_time: Option<String>,
        end_time: Option<String>,
    ) -> Result<Vec<MapRenderItem>> {
        let mut items = Vec::new();

        if (filter_type == MapDataType::All || filter_type == MapDataType::User) && counts.users > 0 {
            let users = if let Some(uid) = social_filter_user_id {
                self.map_repo
                    .search_friend_users_by_bounds(
                        uid,
                        bounds.west,
                        bounds.south,
                        bounds.east,
                        bounds.north,
                        Self::entity_limit(counts.users),
                    )
                    .await?
            } else {
                self.map_repo
                    .search_users_by_bounds(
                        bounds.west,
                        bounds.south,
                        bounds.east,
                        bounds.north,
                        Self::entity_limit(counts.users),
                    )
                    .await?
            };
            items.extend(
                users
                    .into_iter()
                    .map(|u| self.entity_to_item(u, MapDataType::User.as_str())),
            );
        }

        if (filter_type == MapDataType::All || filter_type == MapDataType::Group) && counts.groups > 0 {
            let groups = self
                .map_repo
                .search_groups_by_bounds(
                    bounds.west,
                    bounds.south,
                    bounds.east,
                    bounds.north,
                    Self::entity_limit(counts.groups),
                )
                .await?;
            items.extend(
                groups
                    .into_iter()
                    .map(|g| self.entity_to_item(g, MapDataType::Group.as_str())),
            );
        }

        if (filter_type == MapDataType::All || filter_type == MapDataType::Moment) && counts.moments > 0 {
            let moments = if let Some(uid) = social_filter_user_id {
                self.map_repo
                    .search_friend_moments_by_bounds(
                        uid,
                        bounds.west,
                        bounds.south,
                        bounds.east,
                        bounds.north,
                        Self::entity_limit(counts.moments),
                        start_time,
                        end_time,
                    )
                    .await?
            } else {
                self.map_repo
                    .search_moments_by_bounds(
                        bounds.west,
                        bounds.south,
                        bounds.east,
                        bounds.north,
                        Self::entity_limit(counts.moments),
                        start_time,
                        end_time,
                    )
                    .await?
            };
            items.extend(
                moments
                    .into_iter()
                    .map(|m| self.entity_to_item(m, MapDataType::Moment.as_str())),
            );
        }

        Ok(items)
    }

    async fn fetch_cluster_items(
        &self,
        bounds: Bounds,
        filter_type: MapDataType,
        social_filter_user_id: Option<i64>,
        grid_meters: f64,
        start_time: Option<String>,
        end_time: Option<String>,
    ) -> Result<Vec<MapRenderItem>> {
        let mut items = Vec::new();

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
                        start_time,
                        end_time,
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
                        start_time,
                        end_time,
                    )
                    .await?
            };
            items.extend(
                moments
                    .into_iter()
                    .map(|c| self.cluster_to_item(c, MapDataType::Moment.as_str())),
            );
        }

        Ok(items)
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

        let counts = self
            .count_visible_entities(
                bounds,
                filter_type,
                social_filter_user_id,
                start_time.clone(),
                end_time.clone(),
            )
            .await?;

        if counts.total() == 0 {
            return Ok(MapCanvas {
                items: Vec::new(),
                toast_message: None,
            });
        }

        if Self::should_return_entity_items(counts.total(), zoom) {
            let capped_counts = Self::capped_entity_counts(counts, MAP_RENDER_ITEM_LIMIT);
            let items = self
                .fetch_entity_items(
                    bounds,
                    filter_type,
                    social_filter_user_id,
                    capped_counts,
                    start_time,
                    end_time,
                )
                .await?;
            return Ok(MapCanvas {
                items,
                toast_message: if counts.total() > MAP_RENDER_ITEM_LIMIT as i64 {
                    Some("当前区域数据较多，已优先展示附近具体内容".to_string())
                } else {
                    None
                },
            });
        }

        let grid_ceiling = Self::cluster_grid_ceiling(bounds);
        let mut grid_meters = Self::grid_meters(bounds, zoom, None).min(grid_ceiling);
        let mut items = self
            .fetch_cluster_items(
                bounds,
                filter_type,
                social_filter_user_id,
                grid_meters,
                start_time.clone(),
                end_time.clone(),
            )
            .await?;

        while items.len() > MAP_RENDER_ITEM_LIMIT && grid_meters < grid_ceiling {
            let next_grid = Self::grow_cluster_grid(grid_meters, grid_ceiling);
            if (next_grid - grid_meters).abs() < f64::EPSILON {
                break;
            }
            grid_meters = next_grid;
            items = self
                .fetch_cluster_items(
                    bounds,
                    filter_type,
                    social_filter_user_id,
                    grid_meters,
                    start_time.clone(),
                    end_time.clone(),
                )
                .await?;
        }

        Ok(MapCanvas {
            items,
            toast_message: Some("当前区域数据较多，已自动聚合显示".to_string()),
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

#[cfg(test)]
mod tests {
    use super::{MapService, VisibleEntityCounts};
    use crate::{constants::MAP_RENDER_ITEM_LIMIT, dto::request::Bounds};

    #[test]
    fn entity_mode_includes_limit_boundary() {
        assert!(MapService::should_return_entity_items(MAP_RENDER_ITEM_LIMIT as i64, 14.0));
        assert!(!MapService::should_return_entity_items((MAP_RENDER_ITEM_LIMIT as i64) + 1, 14.0));
        assert!(MapService::should_return_entity_items((MAP_RENDER_ITEM_LIMIT as i64) + 1, 16.0));
    }

    #[test]
    fn high_zoom_entity_cap_preserves_limit() {
        let capped = MapService::capped_entity_counts(
            VisibleEntityCounts {
                users: 400,
                groups: 300,
                moments: 300,
            },
            MAP_RENDER_ITEM_LIMIT,
        );

        assert_eq!(capped.total(), MAP_RENDER_ITEM_LIMIT as i64);
        assert!(capped.users > 0);
        assert!(capped.groups > 0);
        assert!(capped.moments > 0);
    }

    #[test]
    fn cluster_grid_growth_doubles_until_ceiling() {
        assert_eq!(MapService::grow_cluster_grid(60.0, 500.0), 120.0);
        assert_eq!(MapService::grow_cluster_grid(320.0, 500.0), 500.0);
    }

    #[test]
    fn cluster_grid_ceiling_has_minimum_for_degenerate_bounds() {
        let ceiling = MapService::cluster_grid_ceiling(Bounds {
            north: 39.9,
            south: 39.9,
            east: 116.4,
            west: 116.4,
        });

        assert_eq!(ceiling, 10.0);
    }

    #[test]
    fn cluster_grid_ceiling_covers_viewport_span() {
        let ceiling = MapService::cluster_grid_ceiling(Bounds {
            north: 40.0,
            south: 39.5,
            east: 116.6,
            west: 116.1,
        });

        assert!(ceiling > 40_000.0);
    }
}
