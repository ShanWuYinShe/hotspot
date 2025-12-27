use std::f64::consts::PI;

use crate::{
    dto::{
        request::{Bounds, MapClusterRequest},
        response::{MapClusterItem, MapClustersData},
    },
    error::{AppError, Result},
    repositories::map_repo::MapRepository,
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
            return Err(AppError::BadRequest("bounds.north 必须 >= bounds.south".to_string()));
        }
        if bounds.east < bounds.west {
            return Err(AppError::BadRequest("bounds.east 必须 >= bounds.west".to_string()));
        }
        Ok(())
    }

    fn grid_meters(bounds: Bounds, zoom: i32, grid_size: Option<i32>) -> f64 {
        // 默认一个 cluster 格子按 60px 算（前端可覆写）
        let grid_px = grid_size.unwrap_or(60).max(8).min(256) as f64;
        let z = zoom.max(0).min(22) as f64;

        // 使用 bbox 中心纬度做 cos 修正（避免高纬度过密）
        let center_lat = ((bounds.north + bounds.south) / 2.0).max(-85.0).min(85.0);
        let cos_lat = (center_lat * PI / 180.0).cos().max(0.01);

        // WebMercator：156543.03392 m/px @ zoom=0（赤道）
        let meters_per_px = 156543.03392 * cos_lat / 2_f64.powf(z);
        (grid_px * meters_per_px).max(10.0)
    }

    pub async fn cluster_users(&self, req: &MapClusterRequest) -> Result<MapClustersData> {
        Self::validate_bounds(req.bounds)?;
        let grid_meters = Self::grid_meters(req.bounds, req.zoom, req.grid_size);

        let rows = self
            .map_repo
            .cluster_users_by_bounds(
                req.bounds.west,
                req.bounds.south,
                req.bounds.east,
                req.bounds.north,
                grid_meters,
            )
            .await?;

        let items = rows
            .into_iter()
            .map(|r| MapClusterItem {
                key: r.key,
                latitude: r.latitude,
                longitude: r.longitude,
                count: r.count,
            })
            .collect();

        Ok(MapClustersData::Users(items))
    }

    pub async fn cluster_groups(&self, req: &MapClusterRequest) -> Result<MapClustersData> {
        Self::validate_bounds(req.bounds)?;
        let grid_meters = Self::grid_meters(req.bounds, req.zoom, req.grid_size);

        let rows = self
            .map_repo
            .cluster_groups_by_bounds(
                req.bounds.west,
                req.bounds.south,
                req.bounds.east,
                req.bounds.north,
                grid_meters,
            )
            .await?;

        let items = rows
            .into_iter()
            .map(|r| MapClusterItem {
                key: r.key,
                latitude: r.latitude,
                longitude: r.longitude,
                count: r.count,
            })
            .collect();

        Ok(MapClustersData::Groups(items))
    }

    pub async fn cluster_moments(&self, req: &MapClusterRequest) -> Result<MapClustersData> {
        Self::validate_bounds(req.bounds)?;
        let grid_meters = Self::grid_meters(req.bounds, req.zoom, req.grid_size);

        let rows = self
            .map_repo
            .cluster_moments_by_bounds(
                req.bounds.west,
                req.bounds.south,
                req.bounds.east,
                req.bounds.north,
                grid_meters,
            )
            .await?;

        let items = rows
            .into_iter()
            .map(|r| MapClusterItem {
                key: r.key,
                latitude: r.latitude,
                longitude: r.longitude,
                count: r.count,
            })
            .collect();

        Ok(MapClustersData::Moments(items))
    }
}

