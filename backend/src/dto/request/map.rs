//! Map 聚合请求类型（统一 canvas 接口）

use serde::Deserialize;

use crate::dto::request::common::{MapCanvasFilter, MapViewport, PaginationParams};

/// 地图画布请求 - 对应 spec 中的 MapCanvasRequest
#[derive(Debug, Deserialize)]
pub struct MapCanvasRequest {
    pub viewport: MapViewport,
    pub filter: Option<MapCanvasFilter>,
    pub pagination: Option<PaginationParams>,
}
