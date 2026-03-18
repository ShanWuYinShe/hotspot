//! 共享类型（GeoPoint, Location, Map 视口等）

use serde::{Deserialize, Serialize};

use crate::types::enums::MapDataType;

#[derive(Debug, Deserialize, Serialize, Clone, Copy)]
pub struct GeoPoint {
    pub latitude: f64,
    pub longitude: f64,
}

#[derive(Debug, Clone, Deserialize, serde::Serialize)]
pub struct Location {
    pub point: GeoPoint,
    pub location_name: String,
}

#[derive(Debug, Deserialize, Clone, Copy)]
pub struct Bounds {
    pub north: f64,
    pub south: f64,
    pub east: f64,
    pub west: f64,
}

#[derive(Debug, Deserialize, Clone)]
pub struct MapViewport {
    pub bounds: Bounds,
    pub zoom: f64,
}

#[derive(Debug, Deserialize)]
pub struct TimeRangeParams {
    pub start_time: Option<String>,
    pub end_time: Option<String>,
}

/// 地图过滤条件：统一用于 heatmap / hotspots / canvas
#[derive(Debug, Deserialize)]
pub struct MapCanvasFilter {
    pub data_type: Option<MapDataType>,
    pub time_range: Option<TimeRangeParams>,
}

#[derive(Debug, Deserialize, Clone)]
pub struct PaginationParams {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}
