use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{AdminTagSearchRequest, SuggestTagsRequest, TrendingTagsRequest},
        response::{ApiResponse, TagSuggestData, TrendingTagsData},
    },
    error::Result,
    state::AppState,
};

// ==================== Token 可选 ====================

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

// ==================== Admin APIs ====================

/// 搜索标签（管理员）
pub async fn admin_search(
    State(state): State<AppState>,
    Json(req): Json<AdminTagSearchRequest>,
) -> Result<Json<ApiResponse<TagSuggestData>>> {
    // 复用标签建议搜索逻辑
    let suggest_req = SuggestTagsRequest {
        prefix: req.keyword.unwrap_or_default(),
        limit: req.limit,
    };
    let response = state.tag_service.suggest(&suggest_req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// 删除/屏蔽标签（管理员）
pub async fn admin_delete(
    State(state): State<AppState>,
    Json(req): Json<crate::dto::request::AdminDeleteTagRequest>,
) -> Result<Json<ApiResponse<()>>> {
    state.tag_service.admin_delete_tag(&req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
