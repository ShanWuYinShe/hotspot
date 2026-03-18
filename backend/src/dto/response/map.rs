//! 地图渲染相关响应类型（heatmap / hotspots）

use crate::dto::response::common::GeoPoint;
use serde::Serialize;

/// 热力统计拆分 - 对应 spec 中的 MapHeatBreakdown
#[derive(Debug, Serialize)]
pub struct MapHeatBreakdown {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub group: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub moment: Option<i64>,
}

/// 热力统计 bucket - 对应 spec 中的 MapHeatBucket
#[derive(Debug, Serialize)]
pub struct MapHeatBucket {
    pub location: GeoPoint,
    pub total_count: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub breakdown: Option<MapHeatBreakdown>,
}

/// 热力统计响应 - 对应 spec 中的 MapHeatCanvas
#[derive(Debug, Serialize)]
pub struct MapHeatCanvas {
    pub items: Vec<MapHeatBucket>,
}

/// 热点对象类型 - 对应 spec 中的 HotspotObjectType
#[derive(Debug, Serialize)]
pub struct MapHotspotItem {
    pub object_type: String,
    pub object_id: String,
    pub title: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub subtitle: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cover_url: Option<String>,
    pub location: GeoPoint,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub created_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub distance_meters: Option<f64>,
}

/// 热点明细响应 - 对应 spec 中的 MapHotspotsCanvas
#[derive(Debug, Serialize)]
pub struct MapHotspotsCanvas {
    pub items: Vec<MapHotspotItem>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_cursor: Option<String>,
}

/// 统一地图画布响应 - 对应 spec 中的 MapCanvasResponse
#[derive(Debug, Serialize)]
pub struct MapCanvasResponse {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub heatmap: Option<MapHeatCanvas>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub hotspots: Option<MapHotspotsCanvas>,
}
