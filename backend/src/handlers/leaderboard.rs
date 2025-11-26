use axum::{extract::State, Extension, Json};

use crate::{
    dto::{
        request::{
            GetCreatorLeaderboardRequest, GetMomentLeaderboardRequest,
            GetMyCreatorRankRequest, GetMyMomentRankRequest,
        },
        response::{
            ApiResponse, GetCreatorLeaderboardData, GetMomentLeaderboardData,
            MyCreatorRankData, MyMomentRankData,
        },
    },
    error::Result,
    services::leaderboard_service::LeaderboardService,
    state::AppState,
};

/// POST /api/v1/public/leaderboard/get_creators
pub async fn get_creators(
    State(state): State<AppState>,
    Json(req): Json<GetCreatorLeaderboardRequest>,
) -> Result<Json<ApiResponse<GetCreatorLeaderboardData>>> {
    let service = LeaderboardService::new(state.db.clone(), state.redis.clone());
    let response = service.get_creator_leaderboard(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/leaderboard/get_moments
pub async fn get_moments(
    State(state): State<AppState>,
    Json(req): Json<GetMomentLeaderboardRequest>,
) -> Result<Json<ApiResponse<GetMomentLeaderboardData>>> {
    let service = LeaderboardService::new(state.db.clone(), state.redis.clone());
    let response = service.get_moment_leaderboard(req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/leaderboard/my_creator_rank
pub async fn my_creator_rank(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetMyCreatorRankRequest>,
) -> Result<Json<ApiResponse<MyCreatorRankData>>> {
    let service = LeaderboardService::new(state.db.clone(), state.redis.clone());
    let response = service.get_my_creator_rank(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/leaderboard/my_moment_rank
pub async fn my_moment_rank(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetMyMomentRankRequest>,
) -> Result<Json<ApiResponse<MyMomentRankData>>> {
    let service = LeaderboardService::new(state.db.clone(), state.redis.clone());
    let response = service.get_my_moment_rank(user_id, req).await?;
    Ok(Json(ApiResponse::success(response)))
}
