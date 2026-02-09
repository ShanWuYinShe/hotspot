use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{BatchPresignUploadRequest, CompleteUploadRequest, PresignUploadRequest},
        response::{ApiResponse, PresignUploadData, UploadMediaData},
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    state::AppState,
};

pub async fn presign_upload(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<PresignUploadRequest>,
) -> Result<Json<ApiResponse<PresignUploadData>>> {
    let data = state
        .media_service
        .presign_upload(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(data)))
}

pub async fn batch_presign_upload(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<BatchPresignUploadRequest>,
) -> Result<Json<ApiResponse<Vec<PresignUploadData>>>> {
    let mut results = Vec::with_capacity(req.items.len());
    for item in req.items {
        let data = state
            .media_service
            .presign_upload(user.user_id, item)
            .await?;
        results.push(data);
    }
    Ok(Json(ApiResponse::success(results)))
}

pub async fn complete_upload(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<CompleteUploadRequest>,
) -> Result<Json<ApiResponse<UploadMediaData>>> {
    let data = state
        .media_service
        .complete_upload(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(data)))
}
