use axum::{extract::State, Json};

use crate::{
    dto::{
        request::SearchSuggestRequest,
        response::{ApiResponse, SearchSuggestions},
    },
    error::Result,
    middleware::auth::OptionalAuthenticatedUser,
    state::AppState,
};

/// 综合搜索建议
/// POST /api/v1/search/suggest
pub async fn suggest(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<SearchSuggestRequest>,
) -> Result<Json<ApiResponse<SearchSuggestions>>> {
    let result = state.search_service.suggest(user.user_id, req).await?;
    Ok(Json(ApiResponse::success(result)))
}
