/// 用户相关查询定义
/// 
/// 包含所有用户查询操作的查询模型

use crate::domain::User;
use super::PaginationQuery;
use serde::{Deserialize, Serialize};

/// 根据ID查询用户
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetUserByIdQuery {
    pub user_id: i64,
}

/// 根据登录ID查询用户
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetUserByLoginIdQuery {
    pub login_id: String,
}

/// 搜索附近用户查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FindNearbyUsersQuery {
    pub user_id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub radius_km: Option<f64>, // 搜索半径（公里），默认5公里
    pub pagination: PaginationQuery,
}

/// 按昵称搜索用户查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchUsersByNicknameQuery {
    pub keyword: String,
    pub requester_id: i64,
    pub pagination: PaginationQuery,
}

/// 获取用户资料查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetUserProfileQuery {
    pub user_id: i64,
    pub requester_id: i64, // 请求者ID，用于权限控制
}

/// 检查用户密码状态查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CheckPasswordStatusQuery {
    pub user_id: i64,
}

/// 获取用户设置查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetUserSettingsQuery {
    pub user_id: i64,
}

/// 用户查询结果

/// 用户基本信息结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserBasicInfo {
    pub user_id: i64,
    pub nickname: String,
    pub user_type: String,
    pub has_location: bool,
    pub created_at: chrono::DateTime<chrono::Utc>,
}

/// 用户详细资料结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserProfile {
    pub user_id: i64,
    pub login_id: Option<String>,
    pub nickname: String,
    pub user_type: String,
    pub location: Option<UserLocation>,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
}

/// 用户位置信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserLocation {
    pub latitude: f64,
    pub longitude: f64,
    pub address: Option<String>,
}

/// 附近用户结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NearbyUser {
    pub user_id: i64,
    pub nickname: String,
    pub location: UserLocation,
    pub distance_km: f64,
    pub created_at: chrono::DateTime<chrono::Utc>,
}

/// 用户搜索结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserSearchResult {
    pub user_id: i64,
    pub nickname: String,
    pub user_type: String,
    pub has_location: bool,
    pub created_at: chrono::DateTime<chrono::Utc>,
}

/// 密码状态结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PasswordStatus {
    pub has_password: bool,
    pub is_temporary: bool,
}

/// 用户设置结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserSettings {
    pub user_id: i64,
    pub privacy_settings: PrivacySettings,
    pub notification_settings: NotificationSettings,
}

/// 隐私设置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PrivacySettings {
    pub allow_location_sharing: bool,
    pub allow_friend_requests: bool,
    pub show_online_status: bool,
}

/// 通知设置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationSettings {
    pub enable_push_notifications: bool,
    pub enable_email_notifications: bool,
    pub enable_message_notifications: bool,
    pub enable_group_notifications: bool,
}

// 查询验证方法
impl FindNearbyUsersQuery {
    /// 验证搜索附近用户查询
    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        
        if self.user_id <= 0 {
            errors.push("用户ID必须为正数".to_string());
        }
        
        if self.latitude < -90.0 || self.latitude > 90.0 {
            errors.push("纬度必须在-90到90之间".to_string());
        }
        
        if self.longitude < -180.0 || self.longitude > 180.0 {
            errors.push("经度必须在-180到180之间".to_string());
        }
        
        if let Some(radius) = self.radius_km {
            if radius <= 0.0 || radius > 50.0 {
                errors.push("搜索半径必须在0-50公里之间".to_string());
            }
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
    
    /// 获取搜索半径
    pub fn get_radius_km(&self) -> f64 {
        self.radius_km.unwrap_or(5.0) // 默认5公里
    }
}

impl SearchUsersByNicknameQuery {
    /// 验证按昵称搜索用户查询
    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        
        if self.keyword.is_empty() {
            errors.push("搜索关键词不能为空".to_string());
        }
        
        if self.keyword.len() < 2 {
            errors.push("搜索关键词至少需要2个字符".to_string());
        }
        
        if self.keyword.len() > 50 {
            errors.push("搜索关键词不能超过50个字符".to_string());
        }
        
        if self.requester_id <= 0 {
            errors.push("请求者ID必须为正数".to_string());
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
}

// 结果转换方法
impl From<User> for UserBasicInfo {
    fn from(user: User) -> Self {
        Self {
            user_id: user.id,
            nickname: user.nickname.clone(),
            user_type: user.user_type.as_str().to_string(),
            has_location: false, // 位置信息现在在单独的表中，需要单独查询
            created_at: user.created_at,
        }
    }
}

impl From<User> for UserProfile {
    fn from(user: User) -> Self {
        let location = None; // 位置信息现在在单独的表中，需要单独查询
        
        Self {
            user_id: user.id,
            login_id: user.login_id,
            nickname: user.nickname,
            user_type: user.user_type.as_str().to_string(),
            location,
            created_at: user.created_at,
            updated_at: user.updated_at,
        }
    }
}

impl From<User> for PasswordStatus {
    fn from(user: User) -> Self {
        Self {
            has_password: user.has_password(),
            is_temporary: user.is_temporary(),
        }
    }
}
