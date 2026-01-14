use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{
            CheckChunkedUploadRequest, CompleteUploadRequest, FinishChunkedUploadRequest,
            InitChunkedUploadRequest, PresignUploadRequest,
        },
        response::{
            ApiResponse, CheckChunkedUploadData, InitiateUploadData, PresignUploadData,
            UploadMediaData,
        },
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

pub async fn init_chunked_upload(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<InitChunkedUploadRequest>,
) -> Result<Json<ApiResponse<InitiateUploadData>>> {
    let data = state
        .media_service
        .init_chunked_upload(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(data)))
}

pub async fn finish_chunked_upload(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<FinishChunkedUploadRequest>,
) -> Result<Json<ApiResponse<UploadMediaData>>> {
    let data = state
        .media_service
        .finish_chunked_upload(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(data)))
}

pub async fn check_chunked_upload(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<CheckChunkedUploadRequest>,
) -> Result<Json<ApiResponse<CheckChunkedUploadData>>> {
    let data = state
        .media_service
        .check_chunked_upload(user.user_id, req)
        .await?;
    Ok(Json(ApiResponse::success(data)))
}
