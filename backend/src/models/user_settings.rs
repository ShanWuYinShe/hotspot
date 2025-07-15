use serde::{Deserialize, Serialize};
use sqlx::Type;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Type)]
#[sqlx(type_name = "user_notification_level", rename_all = "snake_case")]
pub enum NotificationLevel {
    All,
    Mention,
    Important,
    Muted,
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct UserSettings {
    pub id: i64,
    pub user_id: i64,
    pub notification_level: NotificationLevel,
    pub message_notifications: bool,
    pub group_notifications: bool,
    pub system_notifications: bool,
    pub location_notifications: bool,
    pub quiet_hours_enabled: bool,
    pub quiet_hours_start: Option<String>, // Store as "HH:MM" format
    pub quiet_hours_end: Option<String>,   // Store as "HH:MM" format
    pub notification_radius_km: f64,
    pub sound_enabled: bool,
    pub vibration_enabled: bool,
}

impl Default for UserSettings {
    fn default() -> Self {
        Self {
            id: 0,
            user_id: 0,
            notification_level: NotificationLevel::All,
            message_notifications: true,
            group_notifications: true,
            system_notifications: true,
            location_notifications: true,
            quiet_hours_enabled: false,
            quiet_hours_start: Some("22:00".to_string()),
            quiet_hours_end: Some("08:00".to_string()),
            notification_radius_km: 5.0,
            sound_enabled: true,
            vibration_enabled: true,
        }
    }
}

impl UserSettings {
    pub fn default_for_user(user_id: i64) -> Self {
        Self {
            user_id,
            ..Default::default()
        }
    }
}
