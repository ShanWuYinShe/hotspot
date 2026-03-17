//! Map 聚合请求类型（heatmap / hotspots）

use serde::Deserialize;

use crate::dto::request::common::{MapHeatFilter, MapHotspotFilter, MapViewport, PaginationParams};

/// 热力统计请求 - 对应 spec 中的 MapHeatRequest
#[derive(Debug, Deserialize)]
pub struct MapHeatRequest {
    pub viewport: MapViewport,
    pub filter: Option<MapHeatFilter>,
}

/// 热点明细请求 - 对应 spec 中的 MapHotspotsRequest
#[derive(Debug, Deserialize)]
pub struct MapHotspotsRequest {
    pub viewport: MapViewport,
    pub filter: Option<MapHotspotFilter>,
    pub pagination: Option<PaginationParams>,
}
