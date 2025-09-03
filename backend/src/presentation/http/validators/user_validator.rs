/// 用户验证器
/// 
/// 迁移自传统validators/user_validator.rs，适配新架构
/// 提供用户相关的参数验证功能

use super::{ValidationError, ValidationResult, validate_string_length, validate_non_empty_string};

/// 验证常量
const MIN_PASSWORD_LENGTH: usize = 8;
const MAX_PASSWORD_LENGTH: usize = 64;
const MAX_LOGIN_ID_LENGTH: usize = 100;
const MAX_NICKNAME_LENGTH: usize = 32;

/// 验证登录ID格式
pub fn validate_login_id(login_id: &str) -> ValidationResult<()> {
    // 验证长度
    validate_string_length(login_id, "登录ID", 1, MAX_LOGIN_ID_LENGTH)?;
    
    // 验证字符格式
    if !login_id.chars().all(|c| c.is_alphanumeric() || c == '_' || c == '-' || c == '.' || c == '@') {
        return Err(ValidationError::InvalidParameter {
            message: "登录ID只能包含字母、数字和_-.@符号".to_string(),
        });
    }
    
    Ok(())
}

/// 验证密码格式
pub fn validate_password(password: &str) -> ValidationResult<()> {
    // 验证长度
    validate_string_length(password, "密码", MIN_PASSWORD_LENGTH, MAX_PASSWORD_LENGTH)?;
    
    // 可以添加更多密码复杂度要求
    // 比如必须包含数字、字母、特殊字符等
    
    Ok(())
}

/// 验证昵称
pub fn validate_nickname(nickname: &str) -> ValidationResult<()> {
    // 验证非空
    validate_non_empty_string(nickname, "昵称")?;
    
    // 验证长度
    validate_string_length(nickname, "昵称", 1, MAX_NICKNAME_LENGTH)?;
    
    // 验证字符格式（允许中文、字母、数字、空格、常见符号）
    if !nickname.chars().all(|c| {
        c.is_alphanumeric() || c.is_whitespace() || 
        c == '_' || c == '-' || c == '.' || c == '(' || c == ')' ||
        c.is_ascii_punctuation() || !c.is_ascii() // 允许非ASCII字符（如中文）
    }) {
        return Err(ValidationError::InvalidParameter {
            message: "昵称包含不允许的字符".to_string(),
        });
    }
    
    Ok(())
}

/// 验证用户注册信息
pub fn validate_user_registration(
    login_id: &str, 
    password: Option<&str>, 
    nickname: Option<&str>
) -> ValidationResult<()> {
    // 验证登录ID
    validate_login_id(login_id)?;
    
    // 验证密码（如果提供）
    if let Some(pwd) = password {
        validate_password(pwd)?;
    }
    
    // 验证昵称（如果提供）
    if let Some(name) = nickname {
        validate_nickname(name)?;
    }
    
    Ok(())
}

/// 验证用户ID
pub fn validate_user_id(user_id: i64) -> ValidationResult<()> {
    if user_id <= 0 {
        return Err(ValidationError::InvalidParameter {
            message: "用户ID必须为正数".to_string(),
        });
    }
    Ok(())
}

/// 验证用户ID字符串
pub fn validate_user_id_string(user_id_str: &str) -> ValidationResult<i64> {
    user_id_str.parse::<i64>()
        .map_err(|_| ValidationError::InvalidParameter {
            message: "无效的用户ID格式".to_string(),
        })
        .and_then(|id| {
            validate_user_id(id)?;
            Ok(id)
        })
}

/// 验证临时用户升级请求
pub fn validate_temp_user_upgrade(login_id: &str, password: &str) -> ValidationResult<()> {
    validate_login_id(login_id)?;
    validate_password(password)?;
    Ok(())
}

/// 验证用户设置更新请求
pub fn validate_user_settings_update(
    notifications_enabled: Option<bool>,
    location_sharing_enabled: Option<bool>,
    privacy_level: Option<&str>,
) -> ValidationResult<()> {
    // 验证隐私级别（如果提供）
    if let Some(level) = privacy_level {
        match level {
            "public" | "friends" | "private" => {}
            _ => return Err(ValidationError::InvalidParameter {
                message: "隐私级别必须是 public、friends 或 private".to_string(),
            }),
        }
    }
    
    // notifications_enabled 和 location_sharing_enabled 是布尔值，不需要额外验证
    let _ = (notifications_enabled, location_sharing_enabled);
    
    Ok(())
}
