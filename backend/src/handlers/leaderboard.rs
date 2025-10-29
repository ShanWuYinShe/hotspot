use axum::{extract::State, Json};

use crate::{
    dto::{request::GetLeaderboardRequest, response::{ApiResponse, LeaderboardResponse}},
    error::AppError,
    services::leaderboard_service::LeaderboardService,
    state::AppState,
};

pub struct LeaderboardHandler;

impl LeaderboardHandler {
    /// 获取排行榜
    pub async fn get_leaderboard(
        State(state): State<AppState>,
        Json(req): Json<GetLeaderboardRequest>,
    ) -> Result<Json<ApiResponse<LeaderboardResponse>>, AppError> {
        let service = LeaderboardService::new(state.db.clone(), state.redis.clone());
        let result = service.get_leaderboard(req).await?;
        Ok(Json(ApiResponse::success_data(result)))
    }
}

