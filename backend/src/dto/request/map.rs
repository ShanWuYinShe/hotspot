//! Map 聚合

use serde::Deserialize;

use crate::dto::request::common::{MapFilter, MapViewport};

#[derive(Debug, Deserialize)]
pub struct MapRenderRequest {
    pub viewport: MapViewport,
    pub filter: Option<MapFilter>,
}
