use std::f64::consts::PI;

use crate::{
    dto::{
        request::{Bounds, MapRenderRequest},
        response::{MapClusterData, MapEntityData, MapPoint, MapRenderData, MapRenderItem},
    },
    error::{AppError, Result},
    repositories::map_repo::{MapClusterRow, MapEntityRow, MapRepository},
    types::enums::MapResourceType,
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
            return Err(AppError::BadRequest(
                "bounds.north 必须 >= bounds.south".to_string(),
            ));
        }
        if bounds.east < bounds.west {
            return Err(AppError::BadRequest(
                "bounds.east 必须 >= bounds.west".to_string(),
            ));
        }
        Ok(())
    }

    fn grid_meters(bounds: Bounds, zoom: i32, _grid_size: Option<i32>) -> f64 {
        // 默认一个 cluster 格子按 60px 算
        let grid_px = 60.0;
        let z = zoom.max(0).min(22) as f64;

        // 使用 bbox 中心纬度做 cos 修正
        let center_lat = ((bounds.north + bounds.south) / 2.0).max(-85.0).min(85.0);
        let cos_lat = (center_lat * PI / 180.0).cos().max(0.01);

        // WebMercator：156543.03392 m/px @ zoom=0（赤道）
        let meters_per_px = 156543.03392 * cos_lat / 2_f64.powf(z);
        (grid_px * meters_per_px).max(10.0)
    }

    pub async fn render_map(&self, req: MapRenderRequest) -> Result<MapRenderData> {
        Self::validate_bounds(req.viewport.bounds)?;

        use std::str::FromStr;

        let zoom = req.viewport.zoom;
        let bounds = req.viewport.bounds;
        let filter_type_str = req
            .filter
            .as_ref()
            .and_then(|f| f.data_type.as_deref())
            .unwrap_or("all");

        let filter_type =
            MapResourceType::from_str(filter_type_str).unwrap_or(MapResourceType::All);

        let (start_time, end_time) = if let Some(f) = &req.filter {
            let _ = f.only_friends; // TODO: Implement only_friends filter
            if let Some(tr) = &f.time_range {
                (tr.start_time.clone(), tr.end_time.clone())
            } else {
                (None, None)
            }
        } else {
            (None, None)
        };

        let mut items = Vec::new();

        // 阈值：Zoom <= 12 显示聚合点，Zoom > 12 显示具体实体
        // 参考高德/谷歌地图逻辑
        let cluster_zoom_threshold = 12;

        if zoom <= cluster_zoom_threshold {
            // ==================== 聚合模式 ====================
            let grid_meters = Self::grid_meters(bounds, zoom, None);

            // Parallel execution could optimize this
            if filter_type == MapResourceType::All || filter_type == MapResourceType::User {
                let users = self
                    .map_repo
                    .cluster_users_by_bounds(
                        bounds.west,
                        bounds.south,
                        bounds.east,
                        bounds.north,
                        grid_meters,
                    )
                    .await?;
                items.extend(
                    users
                        .into_iter()
                        .map(|c| self.cluster_to_item(c, MapResourceType::User.as_str())),
                );
            }

            if filter_type == MapResourceType::All || filter_type == MapResourceType::Group {
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
                        .map(|c| self.cluster_to_item(c, MapResourceType::Group.as_str())),
                );
            }

            if filter_type == MapResourceType::All || filter_type == MapResourceType::Moment {
                let moments = self
                    .map_repo
                    .cluster_moments_by_bounds(
                        bounds.west,
                        bounds.south,
                        bounds.east,
                        bounds.north,
                        grid_meters,
                        start_time.clone(),
                        end_time.clone(),
                    )
                    .await?;
                items.extend(
                    moments
                        .into_iter()
                        .map(|c| self.cluster_to_item(c, MapResourceType::Moment.as_str())),
                );
            }
        } else {
            // ==================== 实体模式 ====================
            let limit = 200; // 每个类型的最大返回数

            if filter_type == MapResourceType::All || filter_type == MapResourceType::User {
                let users = self
                    .map_repo
                    .search_users_by_bounds(
                        bounds.west,
                        bounds.south,
                        bounds.east,
                        bounds.north,
                        limit,
                    )
                    .await?;
                items.extend(
                    users
                        .into_iter()
                        .map(|u| self.entity_to_item(u, MapResourceType::User.as_str())),
                );
            }

            if filter_type == MapResourceType::All || filter_type == MapResourceType::Group {
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
                        .map(|g| self.entity_to_item(g, MapResourceType::Group.as_str())),
                );
            }

            if filter_type == MapResourceType::All || filter_type == MapResourceType::Moment {
                let moments = self
                    .map_repo
                    .search_moments_by_bounds(
                        bounds.west,
                        bounds.south,
                        bounds.east,
                        bounds.north,
                        limit,
                        start_time.clone(),
                        end_time.clone(),
                    )
                    .await?;
                items.extend(
                    moments
                        .into_iter()
                        .map(|m| self.entity_to_item(m, MapResourceType::Moment.as_str())),
                );
            }
        }

        Ok(MapRenderData {
            items,
            toast_message: None, // 如果 zoom 太小或者是 limit 截断，可以给提示
        })
    }

    fn cluster_to_item(&self, row: MapClusterRow, type_name: &str) -> MapRenderItem {
        MapRenderItem {
            id: format!("cluster:{}:{}", type_name, row.key),
            r#type: "cluster".to_string(), // Frontend distinguishes clusters
            location: MapPoint {
                latitude: row.latitude,
                longitude: row.longitude,
            },
            cluster: Some(MapClusterData {
                count: row.count,
                label: Some(format!("{} {}", row.count, type_name)),
                preview_urls: None, // TODO: query preview avatars
                boundary: None,
            }),
            entity: None,
        }
    }

    fn entity_to_item(&self, row: MapEntityRow, type_name: &str) -> MapRenderItem {
        MapRenderItem {
            id: format!("{}:{}", type_name, row.id),
            r#type: type_name.to_string(),
            location: MapPoint {
                latitude: row.latitude,
                longitude: row.longitude,
            },
            cluster: None,
            entity: Some(MapEntityData {
                title: row.title,
                subtitle: row.subtitle,
                cover_url: row.cover_url,
                payload_id: row.id,
                is_highlighted: None,
            }),
        }
    }
}
