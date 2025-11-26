use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    repositories::private_chat_repo::{PrivateChatRepository, PrivateChatRow, PrivateChatSettingsRow},
};

/// 私聊服务
#[derive(Clone)]
pub struct PrivateChatService {
    repo: PrivateChatRepository,
}

impl PrivateChatService {
    pub fn new(pool: PgPool) -> Self {
        Self {
            repo: PrivateChatRepository::new(pool),
        }
    }

    /// 创建或获取私聊
    pub async fn create_or_get(
        &self,
        user_id_1: i64,
        user_id_2: i64,
    ) -> Result<PrivateChatRow> {
        if user_id_1 == user_id_2 {
            return Err(AppError::BadRequest("不能与自己创建私聊".to_string()));
        }

        // 尝试查找现有私聊
        if let Some(chat) = self.repo.find_by_users(user_id_1, user_id_2).await? {
            return Ok(chat);
        }

        // 创建新私聊
        let chat = self.repo.create(user_id_1, user_id_2).await?;
        Ok(chat)
    }

    /// 根据 private_chat_id 查找私聊
    pub async fn find_by_id(
        &self,
        private_chat_id: Uuid,
    ) -> Result<Option<PrivateChatRow>> {
        self.repo.find_by_id(private_chat_id).await
    }

    /// 获取用户的所有私聊列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<Vec<PrivateChatRow>> {
        let cursor_uuid = if let Some(c) = cursor {
            Some(Uuid::parse_str(&c).map_err(|_| AppError::BadRequest("Invalid cursor".to_string()))?)
        } else {
            None
        };
        
        self.repo.list_by_user(user_id, limit, cursor_uuid).await
    }

    /// 删除私聊
    pub async fn delete(
        &self,
        private_chat_id: Uuid,
        user_id: i64,
    ) -> Result<()> {
        // 检查私聊是否存在
        let _chat = self.repo.find_by_id(private_chat_id)
            .await?
            .ok_or_else(|| AppError::NotFound("私聊不存在".to_string()))?;

        // 检查用户是否是私聊成员
        let is_member = self.repo.is_user_in_chat(private_chat_id, user_id).await?;
        if !is_member {
            return Err(AppError::Forbidden("无权删除此私聊".to_string()));
        }

        // 软删除私聊
        self.repo.soft_delete(private_chat_id).await?;

        Ok(())
    }

    /// 获取私聊设置
    pub async fn get_settings(
        &self,
        private_chat_id: Uuid,
        user_id: i64,
    ) -> Result<Option<PrivateChatSettingsRow>> {
        self.repo.get_settings(private_chat_id, user_id).await
    }

    /// 更新私聊设置
    pub async fn update_settings(
        &self,
        private_chat_id: Uuid,
        user_id: i64,
        is_muted: Option<bool>,
        is_pinned: Option<bool>,
        is_archived: Option<bool>,
    ) -> Result<()> {
        // 检查私聊是否存在
        let _chat = self.repo.find_by_id(private_chat_id)
            .await?
            .ok_or_else(|| AppError::NotFound("私聊不存在".to_string()))?;

        // 检查用户是否是私聊成员
        let is_member = self.repo.is_user_in_chat(private_chat_id, user_id).await?;
        if !is_member {
            return Err(AppError::Forbidden("无权修改此私聊设置".to_string()));
        }

        // 更新设置
        self.repo.upsert_settings(
            private_chat_id,
            user_id,
            is_muted,
            is_pinned,
            is_archived,
        ).await?;

        Ok(())
    }
}
