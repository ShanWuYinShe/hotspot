use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{GetTrendingTagsRequest, SuggestTagsRequest},
        response::{ApiResponse, TagSuggestData, TrendingTagsData},
    },
    error::Result,
    services::tag_service::TagService,
    state::AppState,
};

/// POST /api/v1/public/tags/get_trending - 获取热门标签
pub async fn get_trending(
    State(state): State<AppState>,
    Json(req_body): Json<GetTrendingTagsRequest>,
) -> Result<Json<ApiResponse<TrendingTagsData>>> {
    let service = TagService::new(&state.db);
    let response = service.get_trending(&req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/public/tags/suggest - 标签搜索建议
pub async fn suggest(
    State(state): State<AppState>,
    Json(req_body): Json<SuggestTagsRequest>,
) -> Result<Json<ApiResponse<TagSuggestData>>> {
    let service = TagService::new(&state.db);
    let response = service.suggest(&req_body).await?;
    Ok(Json(ApiResponse::success(response)))
}


