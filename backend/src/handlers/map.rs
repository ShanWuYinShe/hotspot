use axum::{extract::State, Json};

use crate::{
	dto::{
		request::MapCanvasRequest,
		response::{ApiResponse, MapCanvasResponse},
	},
	error::Result,
	middleware::auth::OptionalAuthenticatedUser,
	state::AppState,
};

/// 地图画布：根据视口自动选择返回热力统计或热点明细
pub async fn canvas(
	State(state): State<AppState>,
	user: OptionalAuthenticatedUser,
	Json(req): Json<MapCanvasRequest>,
) -> Result<Json<ApiResponse<MapCanvasResponse>>> {
	let data = state
		.map_service
		.render_canvas(req, user.user_id)
		.await?;
	Ok(Json(ApiResponse::success(data)))
}
