use axum::{extract::State, Json};

use crate::{
	dto::{
		request::{MapHeatRequest, MapHotspotsRequest},
		response::{ApiResponse, MapHeatCanvas, MapHotspotsCanvas},
	},
	error::Result,
	middleware::auth::OptionalAuthenticatedUser,
	state::AppState,
};

/// 热力统计（按行政区聚合）
pub async fn heatmap(
	State(state): State<AppState>,
	_user: OptionalAuthenticatedUser,
	Json(req): Json<MapHeatRequest>,
) -> Result<Json<ApiResponse<MapHeatCanvas>>> {
	let data = state.map_service.render_heatmap(req).await?;
	Ok(Json(ApiResponse::success(data)))
}

/// 热点明细（当前视野内对象列表）
pub async fn hotspots(
	State(state): State<AppState>,
	user: OptionalAuthenticatedUser,
	Json(req): Json<MapHotspotsRequest>,
) -> Result<Json<ApiResponse<MapHotspotsCanvas>>> {
	let data = state
		.map_service
		.render_hotspots(req, user.user_id)
		.await?;
	Ok(Json(ApiResponse::success(data)))
}
