/// 用户设置领域实体
/// 
/// 管理用户的个性化设置和通知偏好
use serde::{Deserialize, Serialize};

/// 用户通知级别枚举
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum UserNotificationLevel {
    /// 接收所有通知
    All,
    /// 仅接收提及通知
    Mention,
    /// 仅接收重要通知
    Important,
    /// 静音所有通知
    Muted,
}

impl UserNotificationLevel {
    pub fn as_str(&self) -> &'static str {
        match self {
            UserNotificationLevel::All => "all",
            UserNotificationLevel::Mention => "mention",
            UserNotificationLevel::Important => "important",
            UserNotificationLevel::Muted => "muted",
        }
    }
    
    pub fn parse(s: &str) -> Option<Self> {
        match s {
            "all" => Some(UserNotificationLevel::All),
            "mention" => Some(UserNotificationLevel::Mention),
            "important" => Some(UserNotificationLevel::Important),
            "muted" => Some(UserNotificationLevel::Muted),
            _ => None,
        }
    }
}

/// 用户设置实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserSettings {
    /// 数据库自增主键
    pub id: i64,
    /// 关联的用户ID
    pub user_id: i64,
    /// 通知级别
    pub notification_level: UserNotificationLevel,
    /// 消息通知开关
    pub message_notifications: bool,
    /// 群组通知开关
    pub group_notifications: bool,
    /// 系统通知开关
    pub system_notifications: bool,
    /// 位置通知开关
    pub location_notifications: bool,
    /// 免打扰时间开关
    pub quiet_hours_enabled: bool,
    /// 免打扰开始时间（HH:MM:SS格式）
    pub quiet_hours_start: Option<String>,
    /// 免打扰结束时间（HH:MM:SS格式）
    pub quiet_hours_end: Option<String>,
    /// 通知半径（公里）
    pub notification_radius_km: f64,
    /// 声音开关
    pub sound_enabled: bool,
    /// 震动开关
    pub vibration_enabled: bool,
}

impl UserSettings {
    /// 创建默认用户设置
    pub fn new_default(user_id: i64) -> Self {
        Self {
            id: 0, // 将由数据库生成
            user_id,
            notification_level: UserNotificationLevel::All,
            message_notifications: true,
            group_notifications: true,
            system_notifications: true,
            location_notifications: true,
            quiet_hours_enabled: false,
            quiet_hours_start: Some("22:00:00".to_string()),
            quiet_hours_end: Some("08:00:00".to_string()),
            notification_radius_km: 1.0,
            sound_enabled: true,
            vibration_enabled: true,
        }
    }
    
    /// 更新通知级别
    pub fn update_notification_level(&mut self, level: UserNotificationLevel) {
        self.notification_level = level;
    }
    
    /// 更新通知开关
    pub fn update_notification_switches(
        &mut self,
        message: Option<bool>,
        group: Option<bool>,
        system: Option<bool>,
        location: Option<bool>,
    ) {
        if let Some(message) = message {
            self.message_notifications = message;
        }
        if let Some(group) = group {
            self.group_notifications = group;
        }
        if let Some(system) = system {
            self.system_notifications = system;
        }
        if let Some(location) = location {
            self.location_notifications = location;
        }
    }
    
    /// 更新免打扰设置
    pub fn update_quiet_hours(
        &mut self,
        enabled: bool,
        start: Option<String>,
        end: Option<String>,
    ) -> Result<(), String> {
        self.quiet_hours_enabled = enabled;
        
        if let Some(start) = start {
            if !Self::is_valid_time_format(&start) {
                return Err("开始时间格式无效，应为HH:MM:SS".to_string());
            }
            self.quiet_hours_start = Some(start);
        }
        
        if let Some(end) = end {
            if !Self::is_valid_time_format(&end) {
                return Err("结束时间格式无效，应为HH:MM:SS".to_string());
            }
            self.quiet_hours_end = Some(end);
        }
        
        Ok(())
    }
    
    /// 更新通知半径
    pub fn update_notification_radius(&mut self, radius_km: f64) -> Result<(), String> {
        if radius_km <= 0.0 {
            return Err("通知半径必须大于0".to_string());
        }
        if radius_km > 100.0 {
            return Err("通知半径不能超过100公里".to_string());
        }
        self.notification_radius_km = radius_km;
        Ok(())
    }
    
    /// 更新声音和震动设置
    pub fn update_feedback_settings(&mut self, sound: Option<bool>, vibration: Option<bool>) {
        if let Some(sound) = sound {
            self.sound_enabled = sound;
        }
        if let Some(vibration) = vibration {
            self.vibration_enabled = vibration;
        }
    }
    
    /// 验证时间格式（HH:MM:SS）
    fn is_valid_time_format(time: &str) -> bool {
        let parts: Vec<&str> = time.split(':').collect();
        if parts.len() != 3 {
            return false;
        }
        
        // 验证小时（00-23）
        if let Ok(hour) = parts[0].parse::<u32>() {
            if hour > 23 {
                return false;
            }
        } else {
            return false;
        }
        
        // 验证分钟（00-59）
        if let Ok(minute) = parts[1].parse::<u32>() {
            if minute > 59 {
                return false;
            }
        } else {
            return false;
        }
        
        // 验证秒（00-59）
        if let Ok(second) = parts[2].parse::<u32>() {
            if second > 59 {
                return false;
            }
        } else {
            return false;
        }
        
        true
    }
    
    /// 验证设置数据的有效性
    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        
        if self.notification_radius_km <= 0.0 {
            errors.push("通知半径必须大于0".to_string());
        }
        
        if self.notification_radius_km > 100.0 {
            errors.push("通知半径不能超过100公里".to_string());
        }
        
        if let Some(ref start) = self.quiet_hours_start {
            if !Self::is_valid_time_format(start) {
                errors.push("免打扰开始时间格式无效".to_string());
            }
        }
        
        if let Some(ref end) = self.quiet_hours_end {
            if !Self::is_valid_time_format(end) {
                errors.push("免打扰结束时间格式无效".to_string());
            }
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
}