use sqlx::PgPool;

use crate::{
    dto::{
        request::{
            ListInboxItemsRequest, MarkInboxItemsReadRequest, DeleteInboxItemsRequest,
            RegisterDeviceTokenRequest, UnregisterDeviceTokenRequest, RemoveInboxBatchRequest,
        },
        response::{
            InboxItemData, PaginatedData, PaginationMeta, MarkInboxReadData, UnreadInboxSummaryData,
            UnreadCategoryStats, BatchResultData,
        },
    },
    error::AppError,
    repositories::{
        inbox_repo::{InboxRepository, InboxItemRow},
        device_token_repo::DeviceTokenRepository,
    },
    types::enums::NotificationCategory,
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
    pub async fn get_list(
        &self,
        user_id: i64,
        req: ListInboxItemsRequest,
    ) -> Result<PaginatedData<InboxItemData>, AppError> {
        let limit = req.limit.unwrap_or(20).min(100) as i32;

        let category = req.category.as_deref().and_then(|s| {
            match s {
                "social" => Some(NotificationCategory::Social),
                "content" => Some(NotificationCategory::Content),
                "system" => Some(NotificationCategory::System),
                _ => None,
            }
        });
        
        let cursor = if let Some(c) = req.cursor {
            Some(uuid::Uuid::parse_str(&c).map_err(|_| AppError::BadRequest("无效的游标".to_string()))?)
        } else {
            None
        };

        let items = self
            .inbox_repo
            .list_by_user(
                user_id,
                category,
                req.is_read,
                limit + 1,
                cursor,
            )
            .await?;

        let has_more = items.len() > limit as usize;
        let items_to_return: Vec<InboxItemRow> = items.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            items_to_return.last().map(|item| item.inbox_item_id.to_string())
        } else {
            None
        };

        let response_items: Vec<InboxItemData> = items_to_return
            .into_iter()
            .map(|item| InboxItemData {
                id: item.inbox_item_id.to_string(),
                category: item.category.as_str().to_string(),
                r#type: item.r#type,
                title: item.title,
                content: item.content,
                data: item.data,
                action_url: item.action_url,
                severity: item.severity.map(|s| s.as_str().to_string()),
                is_read: item.is_read,
                created_at: item.created_at.to_rfc3339(),
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

    /// 标记已读 (All or Batch)
    pub async fn mark_all_as_read(
        &self,
        user_id: i64,
        req: MarkInboxItemsReadRequest,
    ) -> Result<MarkInboxReadData, AppError> {
        let ids = req.ids.unwrap_or_default();
        
        let count = if ids.is_empty() {
            self.inbox_repo.mark_all_as_read(user_id, None).await?
        } else {
            let inbox_item_ids: Result<Vec<uuid::Uuid>, _> = ids
                .iter()
                .map(|id| uuid::Uuid::parse_str(id))
                .collect();

            let inbox_item_ids = inbox_item_ids
                .map_err(|_| AppError::InvalidInput("无效的inbox_item_id格式".to_string()))?;

            self.inbox_repo.mark_batch_as_read(user_id, &inbox_item_ids).await?
        };

        Ok(MarkInboxReadData {
            marked_count: count as i32,
        })
    }

    /// 删除收件箱项
    pub async fn remove(
        &self,
        user_id: i64,
        req: DeleteInboxItemsRequest,
    ) -> Result<(), AppError> {
        if req.ids.is_empty() {
            return Err(AppError::InvalidInput("ids不能为空".to_string()));
        }

        let inbox_item_ids: Result<Vec<uuid::Uuid>, _> = req.ids
            .iter()
            .map(|id| uuid::Uuid::parse_str(id))
            .collect();

        let inbox_item_ids = inbox_item_ids
            .map_err(|_| AppError::InvalidInput("无效的inbox_item_id格式".to_string()))?;

        self.inbox_repo
            .delete_batch(user_id, &inbox_item_ids)
            .await?;
            
        Ok(())
    }

    /// 批量删除
    pub async fn remove_batch(
        &self,
        user_id: i64,
        req: RemoveInboxBatchRequest,
    ) -> Result<BatchResultData, AppError> {
        let inbox_item_ids: Result<Vec<uuid::Uuid>, _> = req.ids
            .iter()
            .map(|id| uuid::Uuid::parse_str(id))
            .collect();

        let inbox_item_ids = inbox_item_ids
            .map_err(|_| AppError::InvalidInput("无效的inbox_item_id格式".to_string()))?;

        let count = self.inbox_repo
            .delete_batch(user_id, &inbox_item_ids)
            .await?;

        Ok(BatchResultData {
            total: req.ids.len() as i32,
            succeeded: count as i32,
            failed: req.ids.len() as i32 - count as i32,
            results: vec![]
        })
    }

    /// 获取未读数量
    pub async fn get_unread_count(&self, user_id: i64) -> Result<UnreadInboxSummaryData, AppError> {
        let total = self.inbox_repo.get_unread_count(user_id).await? as i32;
        
        // Placeholder for categorization
        Ok(UnreadInboxSummaryData {
            total,
            by_category: UnreadCategoryStats {
                social: 0,
                system: 0,
                content: 0,
            },
        })
    }

    /// 注册设备Token
    pub async fn register_device_token(
        &self,
        user_id: i64,
        req: RegisterDeviceTokenRequest,
    ) -> Result<(), AppError> {
        if !["ios", "android", "web"].contains(&req.platform.as_str()) {
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
        if !["ios", "android", "web"].contains(&req.platform.as_str()) {
            return Err(AppError::InvalidInput(
                format!("无效的平台类型: {}", req.platform)
            ));
        }

        self.device_token_repo
            .deactivate(user_id, &req.device_token)
            .await?;
        
        Ok(())
    }
}
