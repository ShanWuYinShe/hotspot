use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{SuggestTagsRequest, TrendingTagsRequest},
        response::{ApiResponse, TagSuggestData, TrendingTagsData},
    },
    error::Result,
    state::AppState,
};

/// 获取热门标签
pub async fn trending(
    State(state): State<AppState>,
    Json(req_body): Json<TrendingTagsRequest>,
) -> Result<Json<ApiResponse<TrendingTagsData>>> {
    let response = state.tag_service.get_trending(&req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 标签搜索建议
pub async fn suggest(
    State(state): State<AppState>,
    Json(req_body): Json<SuggestTagsRequest>,
) -> Result<Json<ApiResponse<TagSuggestData>>> {
    let response = state.tag_service.suggest(&req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}
