use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{GetConversationSettingsRequest, UpdateConversationSettingsRequest},
        response::ConversationSettingsResponse,
    },
    error::AppError,
    models::conversation_settings::ConversationSettings,
    repositories::conversation_settings_repo::ConversationSettingsRepository,
};

#[derive(Debug, Clone)]
pub struct ConversationSettingsService {
    settings_repo: ConversationSettingsRepository,
}

impl ConversationSettingsService {
    pub fn new(pool: PgPool) -> Self {
        Self {
            settings_repo: ConversationSettingsRepository::new(pool),
        }
    }

    /// 获取会话设置
    pub async fn get_settings(
        &self,
        user_id: i64,
        req: GetConversationSettingsRequest,
    ) -> Result<ConversationSettingsResponse, AppError> {
        // 查找设置
        let settings = self
            .settings_repo
            .find_settings(req.conversation_id, user_id)
            .await?;

        // 如果不存在，创建默认设置
        let settings = if let Some(s) = settings {
            s
        } else {
            let new_settings = ConversationSettings::new(req.conversation_id, user_id);
            self.settings_repo.create_settings(&new_settings).await?
        };

        Ok(ConversationSettingsResponse {
            conversation_id: settings.conversation_id,
            mute_notifications: settings.mute_notifications,
            show_preview: settings.show_preview,
            pin_conversation: settings.pin_conversation,
        })
    }

    /// 更新会话设置
    pub async fn update_settings(
        &self,
        user_id: i64,
        req: UpdateConversationSettingsRequest,
    ) -> Result<ConversationSettingsResponse, AppError> {
        // 确保设置存在
        let existing = self
            .settings_repo
            .find_settings(req.conversation_id, user_id)
            .await?;

        if existing.is_none() {
            // 如果不存在，先创建
            let new_settings = ConversationSettings::new(req.conversation_id, user_id);
            self.settings_repo.create_settings(&new_settings).await?;
        }

        // 更新设置
        let updated_settings = self
            .settings_repo
            .update_settings(
                req.conversation_id,
                user_id,
                req.mute_notifications,
                req.show_preview,
                req.pin_conversation,
            )
            .await?;

        Ok(ConversationSettingsResponse {
            conversation_id: updated_settings.conversation_id,
            mute_notifications: updated_settings.mute_notifications,
            show_preview: updated_settings.show_preview,
            pin_conversation: updated_settings.pin_conversation,
        })
    }

    /// 获取用户所有置顶的会话ID列表
    pub async fn get_pinned_conversations(&self, user_id: i64) -> Result<Vec<Uuid>, AppError> {
        self.settings_repo.find_pinned_conversations(user_id).await
    }
}

