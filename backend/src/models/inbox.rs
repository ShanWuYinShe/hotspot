use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use crate::types::enums::InboxCategory;

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct InboxItem {
    pub id: i64,
    pub item_id: uuid::Uuid,
    pub user_id: i64,
    pub category: InboxCategory,
    pub item_type: String,     // friend_request, comment, like, mention, etc.
    pub title: Option<String>,
    pub content: String,
    pub data: Option<serde_json::Value>, // 通知数据（JSON格式）
    pub action_url: Option<String>,
    pub severity: Option<String>, // info, warning, error
    pub created_at: DateTime<Utc>,
    pub is_read: bool,
}

impl InboxItem {
    /// 创建新的收件箱项
    pub fn new(
        user_id: i64,
        category: InboxCategory,
        item_type: String,
        content: String,
    ) -> Self {
        Self {
            id: 0, // 由数据库生成
            item_id: uuid::Uuid::new_v4(),
            user_id,
            category,
            item_type,
            title: None,
            content,
            data: None,
            action_url: None,
            severity: None,
            created_at: Utc::now(),
            is_read: false,
        }
    }

    /// 构建器：设置标题
    pub fn with_title(mut self, title: String) -> Self {
        self.title = Some(title);
        self
    }

    /// 构建器：设置数据
    pub fn with_data(mut self, data: serde_json::Value) -> Self {
        self.data = Some(data);
        self
    }

    /// 构建器：设置动作URL
    pub fn with_action_url(mut self, url: String) -> Self {
        self.action_url = Some(url);
        self
    }

    /// 构建器：设置严重程度
    pub fn with_severity(mut self, severity: String) -> Self {
        self.severity = Some(severity);
        self
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, FromRow)]
pub struct DeviceToken {
    pub id: i64,
    pub user_id: i64,
    pub token: String,
    pub platform: String, // ios, android, web
    pub created_at: DateTime<Utc>,
}

