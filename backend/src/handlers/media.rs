use axum::{
    body::Bytes,
    extract::{Multipart, Request, State},
    Json,
};

use crate::{
    dto::response::{ApiResponse, UploadResponse},
    error::{AppError, Result},
    middleware::auth::extract_user_id,
    services::media_service::MediaService,
    state::AppState,
};

/// POST /api/v1/protected/media/upload_image
pub async fn upload_image(
    State(state): State<AppState>,
    req: Request,
    mut multipart: Multipart,
) -> Result<Json<ApiResponse<UploadResponse>>> {
    let user_id = extract_user_id(&req)?;

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
    req: Request,
    mut multipart: Multipart,
) -> Result<Json<ApiResponse<UploadResponse>>> {
    let user_id = extract_user_id(&req)?;

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
    req: Request,
) -> Result<Json<ApiResponse<serde_json::Value>>> {
    let user_id = extract_user_id(&req)?;

    let service = MediaService::new(&state.db, &state.redis);
    let token = service.get_upload_token(user_id).await?;

    Ok(Json(ApiResponse::success(serde_json::json!({
        "token": token,
        "expires_in": 600, // 10分钟
    }))))
}

