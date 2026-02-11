use crate::{
    dto::{
        request::{
            MyDeviceTokensRequest, RegisterDeviceTokenRequest, UnregisterDeviceTokenRequest,
        },
        response::{DeviceTokenItem, PaginatedData, PaginationMeta},
    },
    error::AppError,
    repositories::device_token_repo::DeviceTokenRepository,
    utils::validation::Validator,
};

/// 推送设备管理服务
///
/// 注：通知是实时推送机制，不存储。冷启动通过各业务 API 获取待处理数据。
#[derive(Debug, Clone)]
pub struct DeviceTokensService {
    device_token_repo: DeviceTokenRepository,
}

impl DeviceTokensService {
    pub fn new(device_token_repo: DeviceTokenRepository) -> Self {
        Self { device_token_repo }
    }

    /// 注册设备Token
    pub async fn register_device_token(
        &self,
        user_id: i64,
        req: RegisterDeviceTokenRequest,
    ) -> Result<(), AppError> {
        // platform 已是强类型枚举，无需验证

        if req.device_token.is_empty() {
            return Err(AppError::InvalidFormat("token不能为空".to_string()));
        }

        self.device_token_repo
            .upsert(user_id, req.platform.as_str(), &req.device_token)
            .await?;

        Ok(())
    }

    /// 注销设备Token
    pub async fn unregister_device_token(
        &self,
        user_id: i64,
        req: UnregisterDeviceTokenRequest,
    ) -> Result<(), AppError> {
        // platform 已是强类型枚举，无需验证

        self.device_token_repo
            .unregister(user_id, req.platform, &req.device_token)
            .await?;

        Ok(())
    }

    /// 获取设备Token列表
    pub async fn get_device_tokens(
        &self,
        user_id: i64,
        req: MyDeviceTokensRequest,
    ) -> Result<PaginatedData<DeviceTokenItem>, AppError> {
        let _limit = Validator::validate_limit(req.limit, 100)? as i32;
        let _cursor = req.cursor;

        let (tokens, next_cursor) = (self.device_token_repo.list_by_user(user_id).await?, None);

        let response_items: Vec<DeviceTokenItem> = tokens
            .into_iter()
            .map(|token| DeviceTokenItem {
                device_token_id: token.token_id.to_string(),
                platform: token.platform,
                device_token: token.device_token,
                created_at: token.created_at.to_rfc3339(),
            })
            .collect();

        Ok(PaginatedData {
            items: response_items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }
}
