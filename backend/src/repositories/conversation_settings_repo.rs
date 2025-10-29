use sqlx::PgPool;
use uuid::Uuid;

use crate::{error::AppError, models::conversation_settings::ConversationSettings};

#[derive(Debug, Clone)]
pub struct ConversationSettingsRepository {
    pool: PgPool,
}

impl ConversationSettingsRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建会话设置
    pub async fn create_settings(
        &self,
        settings: &ConversationSettings,
    ) -> Result<ConversationSettings, AppError> {
        sqlx::query_as!(
            ConversationSettings,
            r#"
            INSERT INTO conversation_settings (id, conversation_id, user_id, mute_notifications, show_preview, pin_conversation, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            RETURNING id, conversation_id, user_id, mute_notifications, show_preview, pin_conversation, created_at, updated_at
            "#,
            settings.id,
            settings.conversation_id,
            settings.user_id,
            settings.mute_notifications,
            settings.show_preview,
            settings.pin_conversation,
            settings.created_at,
            settings.updated_at
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 查找会话设置
    pub async fn find_settings(
        &self,
        conversation_id: Uuid,
        user_id: i64,
    ) -> Result<Option<ConversationSettings>, AppError> {
        sqlx::query_as!(
            ConversationSettings,
            r#"
            SELECT id, conversation_id, user_id, mute_notifications, show_preview, pin_conversation, created_at, updated_at
            FROM conversation_settings
            WHERE conversation_id = $1 AND user_id = $2
            "#,
            conversation_id,
            user_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 更新会话设置
    pub async fn update_settings(
        &self,
        conversation_id: Uuid,
        user_id: i64,
        mute_notifications: Option<bool>,
        show_preview: Option<bool>,
        pin_conversation: Option<bool>,
    ) -> Result<ConversationSettings, AppError> {
        sqlx::query_as!(
            ConversationSettings,
            r#"
            UPDATE conversation_settings
            SET 
                mute_notifications = COALESCE($3, mute_notifications),
                show_preview = COALESCE($4, show_preview),
                pin_conversation = COALESCE($5, pin_conversation),
                updated_at = NOW()
            WHERE conversation_id = $1 AND user_id = $2
            RETURNING id, conversation_id, user_id, mute_notifications, show_preview, pin_conversation, created_at, updated_at
            "#,
            conversation_id,
            user_id,
            mute_notifications,
            show_preview,
            pin_conversation
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 删除会话设置
    pub async fn delete_settings(
        &self,
        conversation_id: Uuid,
        user_id: i64,
    ) -> Result<u64, AppError> {
        sqlx::query!(
            r#"
            DELETE FROM conversation_settings
            WHERE conversation_id = $1 AND user_id = $2
            "#,
            conversation_id,
            user_id
        )
        .execute(&self.pool)
        .await
        .map(|r| r.rows_affected())
        .map_err(AppError::from)
    }

    /// 获取用户所有置顶的会话
    pub async fn find_pinned_conversations(&self, user_id: i64) -> Result<Vec<Uuid>, AppError> {
        sqlx::query_scalar!(
            r#"
            SELECT conversation_id
            FROM conversation_settings
            WHERE user_id = $1 AND pin_conversation = TRUE
            ORDER BY updated_at DESC
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await
        .map_err(AppError::from)
    }
}

