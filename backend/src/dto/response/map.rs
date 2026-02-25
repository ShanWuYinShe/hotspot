//! 地图渲染相关响应类型

use crate::{dto::response::common::GeoPoint, types::enums::MapItemType};
use serde::Serialize;

/// 地图聚合数据 - API 文档: MapClusterData
#[derive(Debug, Serialize)]
pub struct MapClusterData {
    pub count: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub label: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub preview_urls: Option<Vec<String>>,
}

/// 地图实体数据 - API 文档: MapEntityData
#[derive(Debug, Serialize)]
pub struct MapEntityData {
    pub title: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub subtitle: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cover_url: Option<String>,
    pub payload_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub category: Option<String>,
}

/// 地图渲染项 - API 文档: MapRenderItem
#[derive(Debug, Serialize)]
pub struct MapRenderItem {
    pub render_id: String,
    #[serde(rename = "type")]
    pub item_type: MapItemType,
    pub location: GeoPoint,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cluster: Option<MapClusterData>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub entity: Option<MapEntityData>,
}

/// 地图画布数据 - API 文档: MapCanvas
#[derive(Debug, Serialize)]
pub struct MapCanvas {
    pub items: Vec<MapRenderItem>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub toast_message: Option<String>,
}
