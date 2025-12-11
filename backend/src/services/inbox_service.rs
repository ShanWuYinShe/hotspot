use sqlx::PgPool;

use crate::{
    dto::{
        request::{
            RegisterDeviceTokenRequest, UnregisterDeviceTokenRequest,
            GetDeviceTokensRequest,
        },
        response::{
            PaginatedData, PaginationMeta, DeviceTokenItem,
        },
    },
    error::AppError,
    repositories::device_token_repo::DeviceTokenRepository,
    utils::validation::Validator,
};

/// 推送设备管理服务
/// 
/// 注：通知是实时推送机制，不存储。冷启动通过各业务 API 获取待处理数据。
#[derive(Debug, Clone)]
pub struct InboxService {
    device_token_repo: DeviceTokenRepository,
}

impl InboxService {
    pub fn new(pool: PgPool) -> Self {
        Self {
            device_token_repo: DeviceTokenRepository::new(pool),
        }
    }

    /// 注册设备Token
    pub async fn register_device_token(
        &self,
        user_id: i64,
        req: RegisterDeviceTokenRequest,
    ) -> Result<(), AppError> {
        if !["ios", "android", "web", "harmony"].contains(&req.platform.as_str()) {
            return Err(AppError::InvalidInput(
                format!("无效的平台类型: {}", req.platform)
            ));
        }

        if req.device_token.is_empty() {
            return Err(AppError::InvalidInput("token不能为空".to_string()));
        }

        self.device_token_repo
            .upsert(user_id, &req.platform, &req.device_token)
            .await?;

        Ok(())
    }

    /// 注销设备Token
    pub async fn unregister_device_token(
        &self,
        user_id: i64,
        req: UnregisterDeviceTokenRequest,
    ) -> Result<(), AppError> {
        if !["ios", "android", "web", "harmony"].contains(&req.platform.as_str()) {
            return Err(AppError::InvalidInput(
                format!("无效的平台类型: {}", req.platform)
            ));
        }

        self.device_token_repo
            .unregister(user_id, &req.device_token)
            .await?;
        
        Ok(())
    }

    /// 获取设备Token列表
    pub async fn get_device_tokens(
        &self,
        user_id: i64,
        req: GetDeviceTokensRequest,
    ) -> Result<PaginatedData<DeviceTokenItem>, AppError> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = req.cursor.as_ref()
            .map(|c| c.parse::<i64>().map_err(|_| AppError::BadRequest("无效的游标".to_string())))
            .transpose()?;

        let tokens = self.device_token_repo.find_by_user(user_id).await?;
        
        // 简单分页实现（基于 id）
        let filtered_tokens: Vec<_> = if let Some(cursor_id) = cursor {
            tokens.into_iter().filter(|t| t.id > cursor_id).take((limit + 1) as usize).collect()
        } else {
            tokens.into_iter().take((limit + 1) as usize).collect()
        };
        
        let has_more = filtered_tokens.len() > limit as usize;
        let items_to_return: Vec<_> = filtered_tokens.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            items_to_return.last().map(|t| t.id.to_string())
        } else {
            None
        };

        let response_items: Vec<DeviceTokenItem> = items_to_return
            .into_iter()
            .map(|token| DeviceTokenItem {
                id: token.id.to_string(),
                platform: token.platform.as_str().to_string(),
                device_token: token.device_token,
                created_at: token.created_at.to_rfc3339(),
            })
            .collect();

        Ok(PaginatedData {
            items: response_items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }
}
