use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    repositories::{
        private_chat_repo::{PrivateChatRepository, PrivateChatRow, PrivateChatSettingsRow},
        user_settings_repo::UserSettingsRepository,
        user_blocks_repo::UserBlocksRepository,
    },
};

/// 私聊服务
#[derive(Clone)]
pub struct PrivateChatService {
    repo: PrivateChatRepository,
    settings_repo: UserSettingsRepository,
    blocks_repo: UserBlocksRepository,
}

impl PrivateChatService {
    pub fn new(pool: PgPool) -> Self {
        Self {
            repo: PrivateChatRepository::new(pool.clone()),
            settings_repo: UserSettingsRepository::new(pool.clone()),
            blocks_repo: UserBlocksRepository::new(pool),
        }
    }

    /// 创建或获取私聊
    /// 
    /// 权限验证逻辑（根据 API 规范 3.5.1）：
    /// 1. 目标用户的 allow_private_chat = true（否则返回错误码 1309）
    /// 2. 目标用户没有将当前用户拉黑（否则返回错误码 1310）
    pub async fn create_or_get(
        &self,
        current_user_id: i64,
        target_user_id: i64,
    ) -> Result<PrivateChatRow> {
        if current_user_id == target_user_id {
            return Err(AppError::BadRequest("不能与自己创建私聊".to_string()));
        }

        // 尝试查找现有私聊（如果已存在则直接返回，无需权限验证）
        if let Some(chat) = self.repo.find_by_users(current_user_id, target_user_id).await? {
            return Ok(chat);
        }

        // 创建新私聊前进行权限验证
        
        // 1. 检查目标用户是否允许接收私聊
        let target_settings = self.settings_repo.find_by_user_id(target_user_id).await?;
        if let Some(settings) = target_settings {
            match settings.allow_private_chat.as_str() {
                "nobody" => {
                    return Err(AppError::PrivateChatNotAllowed(
                        "对方不允许接收私聊".to_string()
                    ));
                }
                "friends" => {
                    // 检查是否是好友（互相关注）
                    // TODO: 添加好友关系检查逻辑
                    // 暂时允许，后续可以添加更严格的检查
                }
                _ => {
                    // "everyone" 或其他值，允许私聊
                }
            }
        }
        // 如果目标用户没有设置记录，使用默认值（allow_private_chat = 'everyone'）
        
        // 2. 检查当前用户是否被目标用户拉黑
        if self.blocks_repo.is_blocked(target_user_id, current_user_id).await? {
            return Err(AppError::BlockedByUser(
                "已被对方拉黑".to_string()
            ));
        }

        // 创建新私聊
        let chat = self.repo.create(current_user_id, target_user_id).await?;
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
