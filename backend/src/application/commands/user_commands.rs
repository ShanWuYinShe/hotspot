/// 用户相关命令定义和处理器
/// 
/// 包含所有用户操作的命令模型
use crate::domain::{UserEvent, DomainResult, DomainError};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// 用户注册命令
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegisterUserCommand {
    pub login_id: String,
    pub nickname: Option<String>,
    pub password: String,
}

/// 创建临时用户命令
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateTempUserCommand {
    pub nickname: Option<String>,
}

/// 升级用户账户命令
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpgradeUserCommand {
    pub user_id: i64,
    pub login_id: String,
    pub password: String,
}

/// 更新用户位置命令
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateUserLocationCommand {
    pub user_id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub address: Option<String>,
}

/// 更新用户资料命令
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateUserProfileCommand {
    pub user_id: i64,
    pub nickname: Option<String>,
}

/// 设置用户密码命令
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetUserPasswordCommand {
    pub user_id: i64,
    pub password: String,
}

/// 删除用户账户命令
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteUserAccountCommand {
    pub user_id: i64,
    pub requester_id: i64, // 请求删除的用户ID（可能是本人或管理员）
}

/// 用户命令执行结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserCommandResult {
    pub user_id: i64,
    pub success: bool,
    pub message: Option<String>,
    pub events: Vec<UserEvent>,
}

impl UserCommandResult {
    /// 创建成功结果
    pub fn success(user_id: i64, events: Vec<UserEvent>) -> Self {
        Self {
            user_id,
            success: true,
            message: None,
            events,
        }
    }
    
    /// 创建成功结果（带消息）
    pub fn success_with_message(user_id: i64, message: String, events: Vec<UserEvent>) -> Self {
        Self {
            user_id,
            success: true,
            message: Some(message),
            events,
        }
    }
    
    /// 创建失败结果
    pub fn failure(user_id: i64, message: String) -> Self {
        Self {
            user_id,
            success: false,
            message: Some(message),
            events: vec![],
        }
    }
}

/// 用户注册结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserRegistrationResult {
    pub user: crate::domain::UserProfile,
    pub events: Vec<UserEvent>,
}

/// 用户认证结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserAuthResult {
    pub user: crate::domain::UserProfile,
    pub access_token: String,
    pub refresh_token: String,
    pub expires_at: DateTime<Utc>,
}

/// 用户位置更新结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LocationUpdateResult {
    pub user_id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub address: Option<String>,
    pub updated_at: DateTime<Utc>,
}

// 命令验证函数

impl RegisterUserCommand {
    /// 验证注册命令
    pub fn validate(&self) -> DomainResult<()> {
        let mut errors = Vec::new();
        
        if self.login_id.is_empty() {
            errors.push("登录ID不能为空".to_string());
        }
        
        if self.login_id.len() > 100 {
            errors.push("登录ID长度不能超过100个字符".to_string());
        }
        
        if self.password.len() < 8 {
            errors.push("密码长度不能少于8个字符".to_string());
        }
        
        if self.password.len() > 64 {
            errors.push("密码长度不能超过64个字符".to_string());
        }
        
        if let Some(ref nickname) = self.nickname {
            if nickname.is_empty() {
                errors.push("昵称不能为空".to_string());
            }
            if nickname.len() > 32 {
                errors.push("昵称长度不能超过32个字符".to_string());
            }
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(DomainError::ValidationError { 
                field: "user_validation".to_string(),
                reason: format!("Validation errors: {:?}", errors),
            })
        }
    }
}

impl CreateTempUserCommand {
    /// 验证创建临时用户命令
    pub fn validate(&self) -> DomainResult<()> {
        if let Some(ref nickname) = self.nickname {
            if nickname.is_empty() {
                return Err(DomainError::ValidationError {
                    field: "nickname".to_string(),
                    reason: "昵称不能为空".to_string(),
                });
            }
            if nickname.len() > 32 {
                return Err(DomainError::ValidationError {
                    field: "nickname".to_string(),
                    reason: "昵称长度不能超过32个字符".to_string(),
                });
            }
        }
        Ok(())
    }
}

impl UpgradeUserCommand {
    /// 验证升级用户命令
    pub fn validate(&self) -> DomainResult<()> {
        let mut errors = Vec::new();
        
        if self.user_id <= 0 {
            errors.push("用户ID必须为正数".to_string());
        }
        
        if self.login_id.is_empty() {
            errors.push("登录ID不能为空".to_string());
        }
        
        if self.login_id.len() > 100 {
            errors.push("登录ID长度不能超过100个字符".to_string());
        }
        
        if self.password.len() < 8 {
            errors.push("密码长度不能少于8个字符".to_string());
        }
        
        if self.password.len() > 64 {
            errors.push("密码长度不能超过64个字符".to_string());
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(DomainError::ValidationError { 
                field: "user_validation".to_string(),
                reason: format!("Validation errors: {:?}", errors),
            })
        }
    }
}

impl UpdateUserLocationCommand {
    /// 验证位置更新命令
    pub fn validate(&self) -> DomainResult<()> {
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
        
        if let Some(ref address) = self.address
            && address.len() > 200 {
                errors.push("地址长度不能超过200个字符".to_string());
            }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(DomainError::ValidationError { 
                field: "user_validation".to_string(),
                reason: format!("Validation errors: {:?}", errors),
            })
        }
    }
}

impl UpdateUserProfileCommand {
    /// 验证资料更新命令
    pub fn validate(&self) -> DomainResult<()> {
        let mut errors = Vec::new();
        
        if self.user_id <= 0 {
            errors.push("用户ID必须为正数".to_string());
        }
        
        if let Some(ref nickname) = self.nickname {
            if nickname.is_empty() {
                errors.push("昵称不能为空".to_string());
            }
            if nickname.len() > 32 {
                errors.push("昵称长度不能超过32个字符".to_string());
            }
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(DomainError::ValidationError { 
                field: "user_validation".to_string(),
                reason: format!("Validation errors: {:?}", errors),
            })
        }
    }
}

