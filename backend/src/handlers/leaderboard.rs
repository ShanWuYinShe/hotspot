use axum::{extract::{Request, State}, Json};

use crate::{
    dto::{
        request::{ListLeaderboardRequest, GetMyRankRequest},
        response::{ApiResponse, PaginatedResponse, CreatorRankResponse, MomentRankResponse, GetMyRankResponse},
    },
    error::Result,
    middleware::auth::extract_user_id,
    services::leaderboard_service::LeaderboardService,
    state::AppState,
};

/// POST /api/v1/public/leaderboard/list
pub async fn list(
    State(state): State<AppState>,
    Json(req): Json<ListLeaderboardRequest>,
) -> Result<Json<ApiResponse<PaginatedResponse<serde_json::Value>>>> {
    let service = LeaderboardService::new(&state.db, &state.redis);
    let response = service.list_leaderboard(&req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/leaderboard/my_rank
pub async fn my_rank(
    State(state): State<AppState>,
    req: Request,
    Json(req_body): Json<GetMyRankRequest>,
) -> Result<Json<ApiResponse<GetMyRankResponse>>> {
    let user_id = extract_user_id(&req)?;
    let service = LeaderboardService::new(&state.db, &state.redis);
    let response = service.get_my_rank(user_id, &req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

