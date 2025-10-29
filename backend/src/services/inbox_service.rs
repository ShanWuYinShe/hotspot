use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            GetInboxRequest, MarkInboxReadRequest, DeleteInboxItemsRequest,
            RegisterDeviceTokenRequest, UnregisterDeviceTokenRequest,
        },
        response::{InboxItemResponse, PaginatedResponse},
    },
    error::AppError,
    models::inbox::InboxItem,
    repositories::inbox_repo::InboxRepository,
};

#[derive(Debug, Clone)]
pub struct InboxService {
    inbox_repo: InboxRepository,
}

impl InboxService {
    pub fn new(pool: PgPool) -> Self {
        Self {
            inbox_repo: InboxRepository::new(pool),
        }
    }

    /// 获取收件箱列表
    pub async fn get_inbox(
        &self,
        user_id: i64,
        req: GetInboxRequest,
    ) -> Result<PaginatedResponse<InboxItemResponse>, AppError> {
        let offset = req.cursor.unwrap_or(0);
        let limit = req.limit.min(100); // 最多100条

        let items = self
            .inbox_repo
            .find_inbox_items_by_user(
                user_id,
                req.category,
                req.is_read,
                offset,
                limit + 1, // 多查一条判断是否有更多
            )
            .await?;

        let has_more = items.len() > limit as usize;
        let items: Vec<InboxItem> = if has_more {
            items.into_iter().take(limit as usize).collect()
        } else {
            items
        };

        let next_cursor = if has_more {
            Some(offset + limit)
        } else {
            None
        };

        let response_items: Vec<InboxItemResponse> = items
            .into_iter()
            .map(|item| InboxItemResponse {
                id: item.id,
                category: item.category,
                item_type: item.item_type,
                data: item.data,
                is_read: item.is_read,
                created_at: item.created_at,
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
        if req.item_ids.is_empty() {
            return Err(AppError::InvalidInput("item_ids不能为空".to_string()));
        }

        self.inbox_repo.mark_as_read(user_id, req.item_ids).await
    }

    /// 删除收件箱项
    pub async fn delete_items(
        &self,
        user_id: i64,
        req: DeleteInboxItemsRequest,
    ) -> Result<u64, AppError> {
        if req.item_ids.is_empty() {
            return Err(AppError::InvalidInput("item_ids不能为空".to_string()));
        }

        self.inbox_repo
            .delete_inbox_items(user_id, req.item_ids)
            .await
    }

    /// 获取未读数量
    pub async fn get_unread_count(&self, user_id: i64) -> Result<i64, AppError> {
        self.inbox_repo.get_unread_count(user_id).await
    }

    /// 创建收件箱项（内部使用，供其他模块调用）
    pub async fn create_inbox_item(
        &self,
        user_id: i64,
        category: String,
        item_type: String,
        data: serde_json::Value,
    ) -> Result<InboxItem, AppError> {
        let item = InboxItem::new(user_id, category, item_type, data);
        self.inbox_repo.create_inbox_item(&item).await
    }

    /// 注册设备Token（用于推送通知）
    /// 注意：device_tokens表在004_inbox_subscription.sql中定义
    pub async fn register_device_token(
        &self,
        _user_id: i64,
        _req: RegisterDeviceTokenRequest,
    ) -> Result<(), AppError> {
        // TODO: 实现设备Token注册逻辑
        // 1. 检查token是否已存在
        // 2. 插入或更新device_tokens表
        // 3. 记录设备信息（platform, device_id等）
        tracing::warn!("register_device_token not implemented yet");
        Ok(())
    }

    /// 注销设备Token
    pub async fn unregister_device_token(
        &self,
        _user_id: i64,
        _req: UnregisterDeviceTokenRequest,
    ) -> Result<(), AppError> {
        // TODO: 实现设备Token注销逻辑
        // 1. 从device_tokens表中删除
        tracing::warn!("unregister_device_token not implemented yet");
        Ok(())
    }
}

