use crate::{
    utils::response::AppError,
    utils::constants::validation::{
        MIN_PASSWORD_LENGTH, MAX_PASSWORD_LENGTH,
        MAX_LOGIN_ID_LENGTH, MAX_NICKNAME_LENGTH
    }
};

/// 验证登录ID格式
pub fn validate_login_id(login_id: &str) -> Result<(), AppError> {
    if login_id.is_empty() || login_id.len() > MAX_LOGIN_ID_LENGTH {
        return Err(AppError::BadRequest(format!(
            "登录ID长度应在1-{}个字符之间", 
            MAX_LOGIN_ID_LENGTH
        )));
    }
    
    if !login_id.chars().all(|c| c.is_alphanumeric() || c == '_' || c == '-' || c == '.') {
        return Err(AppError::BadRequest("登录ID只能包含字母、数字和_-.符号".to_string()));
    }
    
    Ok(())
}

/// 验证密码格式
pub fn validate_password(password: &str) -> Result<(), AppError> {
    if password.len() < MIN_PASSWORD_LENGTH || password.len() > MAX_PASSWORD_LENGTH {
        return Err(AppError::BadRequest(format!(
            "密码长度应在{}-{}个字符之间", 
            MIN_PASSWORD_LENGTH, 
            MAX_PASSWORD_LENGTH
        )));
    }
    
    // 可以添加更多密码复杂度要求
    // 比如必须包含数字、字母、特殊字符等
    
    Ok(())
}

/// 验证用户注册信息
pub fn validate_user_registration(login_id: &str, password: Option<&str>, nickname: Option<&str>) -> Result<(), AppError> {
    // 验证登录ID（复用专门的登录ID验证函数）
    validate_login_id(login_id)?;
    
    // 验证密码（复用专门的密码验证函数）
    if let Some(pwd) = password {
        validate_password(pwd)?;
    }
    
    // 验证昵称
    if let Some(name) = nickname {
        validate_nickname(name)?;
    }
    
    Ok(())
}

/// 验证昵称
pub fn validate_nickname(nickname: &str) -> Result<(), AppError> {
    if nickname.is_empty() || nickname.len() > MAX_NICKNAME_LENGTH {
        return Err(AppError::BadRequest(format!(
            "昵称长度应在1-{}个字符之间", 
            MAX_NICKNAME_LENGTH
        )));
    }
    Ok(())
}

