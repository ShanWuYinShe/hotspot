use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{MapClusterRequest, MapSearchByBoundsRequest},
        response::{ApiResponse, MapClustersData, MapItemsData},
    },
    error::Result,
    state::AppState,
    utils::validation::Validator,
};

pub async fn cluster_users(
    State(state): State<AppState>,
    Json(req): Json<MapClusterRequest>,
) -> Result<Json<ApiResponse<MapClustersData>>> {
    let data = state.map_service.cluster_users(&req).await?;
    Ok(Json(ApiResponse::success(data)))
}

pub async fn cluster_groups(
    State(state): State<AppState>,
    Json(req): Json<MapClusterRequest>,
) -> Result<Json<ApiResponse<MapClustersData>>> {
    let data = state.map_service.cluster_groups(&req).await?;
    Ok(Json(ApiResponse::success(data)))
}

pub async fn cluster_moments(
    State(state): State<AppState>,
    Json(req): Json<MapClusterRequest>,
) -> Result<Json<ApiResponse<MapClustersData>>> {
    let data = state.map_service.cluster_moments(&req).await?;
    Ok(Json(ApiResponse::success(data)))
}

// 视口内明细：先提供接口壳子（用于前端替代 search_by_location），具体实现由各业务 service 承担
pub async fn search_users_by_bounds(
    State(state): State<AppState>,
    Json(req): Json<MapSearchByBoundsRequest>,
) -> Result<Json<ApiResponse<MapItemsData>>> {
    Validator::validate_coordinates(req.latitude, req.longitude)?;
    // 这里复用 user_service 的“搜索结果 DTO + 隐私控制”构造逻辑
    let items = state
        .user_service
        .search_by_bounds(req.bounds, req.latitude, req.longitude, req.limit)
        .await?;
    Ok(Json(ApiResponse::success(MapItemsData::Users(items))))
}

pub async fn search_groups_by_bounds(
    State(state): State<AppState>,
    Json(req): Json<MapSearchByBoundsRequest>,
) -> Result<Json<ApiResponse<MapItemsData>>> {
    Validator::validate_coordinates(req.latitude, req.longitude)?;
    let items = state
        .group_service
        .search_by_bounds(req.bounds, req.latitude, req.longitude, req.limit)
        .await?;
    Ok(Json(ApiResponse::success(MapItemsData::Groups(items))))
}

pub async fn search_moments_by_bounds(
    State(state): State<AppState>,
    Json(req): Json<MapSearchByBoundsRequest>,
) -> Result<Json<ApiResponse<MapItemsData>>> {
    Validator::validate_coordinates(req.latitude, req.longitude)?;
    let items = state
        .moment_service
        .search_by_bounds(req.bounds, req.latitude, req.longitude, req.limit)
        .await?;
    Ok(Json(ApiResponse::success(MapItemsData::Moments(items))))
}

