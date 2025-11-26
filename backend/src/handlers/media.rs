use axum::{
    extract::{Multipart, State},
    Extension, Json,
};

use crate::{
    dto::{
        request::{
            InitiateUploadRequest, CompleteUploadRequest, GetUploadStatusRequest, AbortUploadRequest,
        },
        response::{ApiResponse, UploadMediaData, InitiateUploadData, UploadStatusData},
    },
    error::{AppError, Result},
    services::media_service::MediaService,
    state::AppState,
};

/// POST /api/v1/protected/media/upload_image
pub async fn upload_image(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    mut multipart: Multipart,
) -> Result<Json<ApiResponse<UploadMediaData>>> {
    // 获取文件
    let field = multipart
        .next_field()
        .await
        .map_err(|e| AppError::BadRequest(format!("读取文件失败: {}", e)))?
        .ok_or_else(|| AppError::BadRequest("没有文件".to_string()))?;

    let filename = field
        .file_name()
        .ok_or_else(|| AppError::BadRequest("文件名为空".to_string()))?
        .to_string();

    let data = field
        .bytes()
        .await
        .map_err(|e| AppError::BadRequest(format!("读取文件数据失败: {}", e)))?
        .to_vec();

    let service = MediaService::new(&state.db, &state.redis);
    let response = service.upload_image(user_id, &filename, data).await?;

    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/media/upload_file
pub async fn upload_file(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    mut multipart: Multipart,
) -> Result<Json<ApiResponse<UploadMediaData>>> {
    // 获取文件
    let field = multipart
        .next_field()
        .await
        .map_err(|e| AppError::BadRequest(format!("读取文件失败: {}", e)))?
        .ok_or_else(|| AppError::BadRequest("没有文件".to_string()))?;

    let filename = field
        .file_name()
        .ok_or_else(|| AppError::BadRequest("文件名为空".to_string()))?
        .to_string();

    let data = field
        .bytes()
        .await
        .map_err(|e| AppError::BadRequest(format!("读取文件数据失败: {}", e)))?
        .to_vec();

    let service = MediaService::new(&state.db, &state.redis);
    let response = service.upload_file(user_id, &filename, data).await?;

    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/media/upload_token
pub async fn upload_token(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let service = MediaService::new(&state.db, &state.redis);
    let token = service.get_upload_token(user_id).await?;

    Ok(Json(ApiResponse::success(serde_json::json!({
        "token": token,
        "expires_in": 600, // 10分钟
    }))))
}

/// POST /api/v1/protected/media/initiate_upload
pub async fn initiate_upload(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<InitiateUploadRequest>,
) -> Result<Json<ApiResponse<InitiateUploadData>>> {
    let service = MediaService::new(&state.db, &state.redis);
    let response = service.initiate_upload(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/media/complete_upload
pub async fn complete_upload(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CompleteUploadRequest>,
) -> Result<Json<ApiResponse<UploadMediaData>>> {
    let service = MediaService::new(&state.db, &state.redis);
    let response = service.complete_upload(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/media/get_upload_status
pub async fn get_upload_status(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetUploadStatusRequest>,
) -> Result<Json<ApiResponse<UploadStatusData>>> {
    let service = MediaService::new(&state.db, &state.redis);
    let response = service.get_upload_status(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}

/// POST /api/v1/protected/media/abort_upload
pub async fn abort_upload(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<AbortUploadRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = MediaService::new(&state.db, &state.redis);
    service.abort_upload(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}
