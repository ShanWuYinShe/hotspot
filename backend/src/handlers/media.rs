use axum::{
    extract::{Multipart, Query, State},
    Extension, Json,
};

use crate::{
    dto::{
        request::{
            InitiateUploadRequest, CompleteUploadRequest, GetUploadStatusRequest, AbortUploadRequest,
            UploadChunkRequest,
        },
        response::{ApiResponse, UploadMediaData, InitiateUploadData, UploadStatusData, UploadChunkData},
    },
    error::{AppError, Result},
    services::media_service::MediaService,
    state::AppState,
};


pub async fn upload_file(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    mut multipart: Multipart,
) -> Result<Json<ApiResponse<UploadMediaData>>> {
    let mut filename: Option<String> = None;
    let mut category: Option<String> = None;
    let mut file_data: Option<Vec<u8>> = None;

    // 解析 multipart 表单
    while let Some(field) = multipart
        .next_field()
        .await
        .map_err(|e| AppError::BadRequest(format!("读取表单失败: {}", e)))?
    {
        let field_name = field.name().unwrap_or("").to_string();
        
        match field_name.as_str() {
            "file" => {
                filename = field.file_name().map(|s| s.to_string());
                file_data = Some(field.bytes().await
                    .map_err(|e| AppError::BadRequest(format!("读取文件数据失败: {}", e)))?
                    .to_vec());
            }
            "category" => {
                category = Some(field.text().await
                    .map_err(|e| AppError::BadRequest(format!("读取 category 失败: {}", e)))?);
            }
            _ => {}
        }
    }

    let filename = filename.ok_or_else(|| AppError::BadRequest("缺少文件".to_string()))?;
    let category = category.ok_or_else(|| AppError::BadRequest("缺少 category 参数".to_string()))?;
    let data = file_data.ok_or_else(|| AppError::BadRequest("缺少文件数据".to_string()))?;

    // 验证 category 是否有效
    let valid_categories = ["avatar", "group_avatar", "moment_image", "chat_image", "chat_video", "chat_audio", "chat_file"];
    if !valid_categories.contains(&category.as_str()) {
        return Err(AppError::BadRequest(format!("无效的 category: {}", category)));
    }

    let service = MediaService::new(&state.db, &state.redis);
    let response = service.upload_file(user_id, &filename, &category, data).await?;

    Ok(Json(ApiResponse::success(response)))
}


pub async fn initiate_upload(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<InitiateUploadRequest>,
) -> Result<Json<ApiResponse<InitiateUploadData>>> {
    let service = MediaService::new(&state.db, &state.redis);
    let response = service.initiate_upload(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn complete_upload(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<CompleteUploadRequest>,
) -> Result<Json<ApiResponse<UploadMediaData>>> {
    let service = MediaService::new(&state.db, &state.redis);
    let response = service.complete_upload(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn get_upload_status(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<GetUploadStatusRequest>,
) -> Result<Json<ApiResponse<UploadStatusData>>> {
    let service = MediaService::new(&state.db, &state.redis);
    let response = service.get_upload_status(user_id, &req).await?;
    Ok(Json(ApiResponse::success(response)))
}


pub async fn abort_upload(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Json(req): Json<AbortUploadRequest>,
) -> Result<Json<ApiResponse<()>>> {
    let service = MediaService::new(&state.db, &state.redis);
    service.abort_upload(user_id, &req).await?;
    Ok(Json(ApiResponse::success_empty()))
}


pub async fn upload_chunk(
    State(state): State<AppState>,
    Extension(user_id): Extension<i64>,
    Query(req): Query<UploadChunkRequest>,
    mut multipart: Multipart,
) -> Result<Json<ApiResponse<UploadChunkData>>> {
    // 获取分块数据
    let field = multipart
        .next_field()
        .await
        .map_err(|e| AppError::BadRequest(format!("读取分块数据失败: {}", e)))?
        .ok_or_else(|| AppError::BadRequest("没有分块数据".to_string()))?;

    let data = field
        .bytes()
        .await
        .map_err(|e| AppError::BadRequest(format!("读取分块数据失败: {}", e)))?
        .to_vec();

    let service = MediaService::new(&state.db, &state.redis);
    let response = service.upload_chunk(user_id, &req.upload_id, req.part_number, data).await?;
    Ok(Json(ApiResponse::success(response)))
}
