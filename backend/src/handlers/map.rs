use axum::{extract::State, Json};

use crate::{
    dto::{
        request::MapRenderRequest,
        response::{ApiResponse, MapRenderData},
    },
    error::Result,
    state::AppState,
};

pub async fn render(
    State(state): State<AppState>,
    Json(req): Json<MapRenderRequest>,
) -> Result<Json<ApiResponse<MapRenderData>>> {
    let data = state.map_service.render_map(req).await?;
    Ok(Json(ApiResponse::success(data)))
}
