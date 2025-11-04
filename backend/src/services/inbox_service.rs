use sqlx::PgPool;

use crate::{
    dto::{
        request::{
            GetInboxRequest, MarkInboxReadRequest, DeleteInboxItemsRequest,
            UpdateDeviceTokenRequest,
        },
        response::{InboxItemResponse, PaginatedResponse},
    },
    error::AppError,
    models::inbox::InboxItem,
    repositories::{
        inbox_repo::InboxRepository,
        device_token_repo::DeviceTokenRepository,
    },
};

#[derive(Debug, Clone)]
pub struct InboxService {
    inbox_repo: InboxRepository,
    device_token_repo: DeviceTokenRepository,
}

impl InboxService {
    pub fn new(pool: PgPool) -> Self {
        Self {
            inbox_repo: InboxRepository::new(pool.clone()),
            device_token_repo: DeviceTokenRepository::new(pool.clone()),
        }
    }

    /// 获取收件箱列表
    pub async fn get_inbox(
        &self,
        user_id: i64,
        req: GetInboxRequest,
    ) -> Result<PaginatedResponse<InboxItemResponse>, AppError> {
        let limit = req.limit.unwrap_or(20).min(100) as i64; // 默认20条，最多100条

        // 将category字符串转换为枚举
        let category = req.category.as_deref().and_then(|s| {
            use crate::types::enums::InboxCategory;
            match s {
                "social" => Some(InboxCategory::Social),
                "content" => Some(InboxCategory::Content),
                "system" => Some(InboxCategory::System),
                _ => None,
            }
        });

        // 使用游标分页：多取一条判断has_more
        let items = self
            .inbox_repo
            .list_by_user(
                user_id,
                category,
                req.cursor.clone(),
                limit + 1,
            )
            .await?;

        let has_more = items.len() > limit as usize;
        let items_to_return: Vec<InboxItem> = items.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的item_id（业务ID）
        let next_cursor = if has_more {
            items_to_return.last().map(|item| item.item_id.to_string())
        } else {
            None
        };

        let response_items: Vec<InboxItemResponse> = items_to_return
            .into_iter()
            .map(|item| InboxItemResponse {
                id: item.item_id.to_string(),
                category: item.category.as_str().to_string(),
                item_type: item.item_type,
                title: item.title,
                content: item.content,
                data: item.data,
                action_url: item.action_url,
                severity: item.severity,
                is_read: item.is_read,
                created_at: item.created_at.to_rfc3339(),
            })
            .collect();

        Ok(PaginatedResponse {
            items: response_items,
            has_more,
            next_cursor,
        })
    }

    /// 标记已读
    pub async fn mark_as_read(
        &self,
        user_id: i64,
        req: MarkInboxReadRequest,
    ) -> Result<u64, AppError> {
        let ids = req.ids.unwrap_or_default();
        if ids.is_empty() {
            // 如果ids为空，表示标记所有未读为已读
            return self.inbox_repo.mark_all_as_read(user_id, None).await;
        }

        // 将字符串ID转换为UUID
        let item_ids: Result<Vec<uuid::Uuid>, _> = ids
            .iter()
            .map(|id| uuid::Uuid::parse_str(id))
            .collect();

        let item_ids = item_ids
            .map_err(|_| AppError::InvalidInput("无效的item_id格式".to_string()))?;

        self.inbox_repo.mark_as_read(user_id, &item_ids).await
    }

    /// 删除收件箱项
    pub async fn delete_items(
        &self,
        user_id: i64,
        req: DeleteInboxItemsRequest,
    ) -> Result<u64, AppError> {
        if req.ids.is_empty() {
            return Err(AppError::InvalidInput("ids不能为空".to_string()));
        }

        // 将字符串ID转换为UUID
        let item_ids: Result<Vec<uuid::Uuid>, _> = req.ids
            .iter()
            .map(|id| uuid::Uuid::parse_str(id))
            .collect();

        let item_ids = item_ids
            .map_err(|_| AppError::InvalidInput("无效的item_id格式".to_string()))?;

        self.inbox_repo
            .delete(user_id, &item_ids)
            .await
    }

    /// 获取未读数量
    pub async fn get_unread_count(&self, user_id: i64) -> Result<i64, AppError> {
        self.inbox_repo.get_unread_count(user_id).await
    }


    /// 更新设备Token（注册或注销）
    pub async fn update_device_token(
        &self,
        user_id: i64,
        req: UpdateDeviceTokenRequest,
    ) -> Result<(), AppError> {
        match req.action.as_str() {
            "register" => {
                // 验证平台类型
                if !["ios", "android", "web"].contains(&req.platform.as_str()) {
                    return Err(AppError::InvalidInput(
                        format!("无效的平台类型: {}", req.platform)
                    ));
                }

                // 验证Token不为空
                if req.token.is_empty() {
                    return Err(AppError::InvalidInput("token不能为空".to_string()));
                }

                // 注册或更新设备Token
                self.device_token_repo
                    .upsert(user_id, &req.platform, &req.token)
                    .await?;

                tracing::info!(
                    "设备Token注册成功: user_id={}, platform={}, token={}...",
                    user_id,
                    req.platform,
                    &req.token[..req.token.len().min(10)]
                );
                Ok(())
            }
            "unregister" => {
                // 如果token为空，注销用户所有设备
                if req.token.is_empty() {
                    let count = self.device_token_repo
                        .deactivate_all(user_id)
                        .await?;
                    
                    tracing::info!(
                        "注销用户所有设备Token: user_id={}, count={}",
                        user_id,
                        count
                    );
                } else {
                    // 注销指定Token
                    let count = self.device_token_repo
                        .deactivate(user_id, &req.token)
                        .await?;
                    
                    tracing::info!(
                        "注销设备Token: user_id={}, token={}..., affected={}",
                        user_id,
                        &req.token[..req.token.len().min(10)],
                        count
                    );
                }
                Ok(())
            }
            _ => Err(AppError::InvalidInput(
                format!("无效的操作类型: {}", req.action)
            )),
        }
    }
}

