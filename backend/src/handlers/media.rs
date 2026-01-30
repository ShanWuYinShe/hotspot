use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{CompleteUploadRequest, PresignUploadRequest},
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
