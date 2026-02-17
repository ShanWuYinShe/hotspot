use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{BatchPresignUploadRequest, CompleteUploadRequest, PresignUploadRequest},
        response::{ApiResponse, BatchPresignedUpload, PresignUploadData, UploadMediaData},
    },
    error::{AppError, Result},
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
) -> Result<Json<ApiResponse<BatchPresignedUpload>>> {
    const MAX_BATCH_FILES: usize = 9;
    if req.files.len() > MAX_BATCH_FILES {
        return Err(AppError::OutOfRange(format!(
            "单次批量上传最多 {} 个文件",
            MAX_BATCH_FILES
        )));
    }
    let mut results = Vec::with_capacity(req.files.len());
    for item in req.files {
        let data = state
            .media_service
            .presign_upload(user.user_id, item)
            .await?;
        results.push(data);
    }
    Ok(Json(ApiResponse::success(BatchPresignedUpload {
        uploads: results,
    })))
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
