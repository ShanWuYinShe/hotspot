use std::f64::consts::PI;

use crate::{
    dto::{
        request::{Bounds, MapRenderRequest},
        response::{MapClusterData, MapEntityData, MapPoint, MapRenderData, MapRenderItem},
    },
    error::{AppError, Result},
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

        let zoom = req.viewport.zoom;
        let bounds = req.viewport.bounds;
        let filter_type = req
            .filter
            .as_ref()
            .and_then(|f| f.data_type)
            .unwrap_or(MapDataType::All);

        let (start_time, end_time) = if let Some(f) = &req.filter {
            let _ = f.only_friends; // public 渲染忽略 only_friends
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
            if filter_type == MapDataType::All || filter_type == MapDataType::User {
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
                        .map(|c| self.cluster_to_item(c, MapDataType::Moment.as_str())),
                );
            }
        } else {
            // ==================== 实体模式 ====================
            let limit = 200; // 每个类型的最大返回数

            if filter_type == MapDataType::All || filter_type == MapDataType::User {
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
                        .map(|m| self.entity_to_item(m, MapDataType::Moment.as_str())),
                );
            }
        }

        Ok(MapRenderData {
            items,
            toast_message: None, // 如果 zoom 太小或者是 limit 截断，可以给提示
        })
    }

    /// Map render with social filtering (only_friends)
    pub async fn render_map_with_social_filter(
        &self,
        user_id: i64,
        req: MapRenderRequest,
    ) -> Result<MapRenderData> {
        Self::validate_bounds(req.viewport.bounds)?;

        let zoom = req.viewport.zoom;
        let bounds = req.viewport.bounds;
        let filter_type = req
            .filter
            .as_ref()
            .and_then(|f| f.data_type)
            .unwrap_or(MapDataType::All);

        let only_friends = req
            .filter
            .as_ref()
            .and_then(|f| f.only_friends)
            .unwrap_or(false);

        let (start_time, end_time) = if let Some(f) = &req.filter {
            if let Some(tr) = &f.time_range {
                (tr.start_time.clone(), tr.end_time.clone())
            } else {
                (None, None)
            }
        } else {
            (None, None)
        };

        let mut items = Vec::new();
        let cluster_zoom_threshold = 12;

        if zoom <= cluster_zoom_threshold {
            // Cluster mode
            let grid_meters = Self::grid_meters(bounds, zoom, None);

            if filter_type == MapDataType::All || filter_type == MapDataType::User {
                let users = if only_friends {
                    self.map_repo
                        .cluster_friend_users_by_bounds(
                            user_id,
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
                let moments = if only_friends {
                    self.map_repo
                        .cluster_friend_moments_by_bounds(
                            user_id,
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
            // Entity mode
            let limit = 200;

            if filter_type == MapDataType::All || filter_type == MapDataType::User {
                let users = if only_friends {
                    self.map_repo
                        .search_friend_users_by_bounds(
                            user_id,
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
                let moments = if only_friends {
                    self.map_repo
                        .search_friend_moments_by_bounds(
                            user_id,
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

        Ok(MapRenderData {
            items,
            toast_message: None,
        })
    }

    fn cluster_to_item(&self, row: MapClusterRow, type_name: &str) -> MapRenderItem {
        // 将预览 URL 转换为 Option<Vec<String>>
        let preview_urls = if row.preview_urls.is_empty() {
            None
        } else {
            Some(row.preview_urls)
        };

        MapRenderItem {
            id: format!("cluster:{}:{}", type_name, row.key),
            item_type: MapItemType::Cluster,
            location: MapPoint {
                latitude: row.latitude,
                longitude: row.longitude,
            },
            cluster: Some(MapClusterData {
                count: row.count,
                label: Some(format!("{} {}", row.count, type_name)),
                preview_urls,
                boundary: None,
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
            id: format!("{}:{}", type_name, row.id),
            item_type,
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
