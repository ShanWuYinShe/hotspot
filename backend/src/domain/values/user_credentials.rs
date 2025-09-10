/// 用户凭证值对象
/// 
/// 封装用户认证相关的数据和验证逻辑
use serde::{Deserialize, Serialize};
use std::fmt;

/// 登录ID值对象
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct LoginId(String);

impl LoginId {
    /// 创建新的登录ID
    pub fn new(id: String) -> Result<Self, String> {
        Self::validate(&id)?;
        Ok(Self(id))
    }
    
    /// 验证登录ID格式
    fn validate(id: &str) -> Result<(), String> {
        if id.is_empty() {
            return Err("登录ID不能为空".to_string());
        }
        
        if id.len() > 100 {
            return Err("登录ID长度不能超过100个字符".to_string());
        }
        
        // 简单的邮箱格式验证
        if id.contains('@') {
            if !Self::is_valid_email(id) {
                return Err("邮箱格式不正确".to_string());
            }
        } else {
            // 手机号格式验证
            if !Self::is_valid_phone(id) {
                return Err("手机号格式不正确".to_string());
            }
        }
        
        Ok(())
    }
    
    /// 简单的邮箱格式验证
    fn is_valid_email(email: &str) -> bool {
        let parts: Vec<&str> = email.split('@').collect();
        if parts.len() != 2 {
            return false;
        }
        
        let local = parts[0];
        let domain = parts[1];
        
        !local.is_empty() && !domain.is_empty() && domain.contains('.')
    }
    
    /// 简单的手机号格式验证（支持中国手机号）
    fn is_valid_phone(phone: &str) -> bool {
        if phone.len() != 11 {
            return false;
        }
        
        phone.chars().all(|c| c.is_ascii_digit()) && phone.starts_with('1')
    }
    
    /// 获取内部值
    pub fn as_str(&self) -> &str {
        &self.0
    }
    
    /// 检查是否为邮箱
    pub fn is_email(&self) -> bool {
        self.0.contains('@')
    }
    
    /// 检查是否为手机号
    pub fn is_phone(&self) -> bool {
        !self.is_email()
    }
    
    /// 获取脱敏显示（用于日志等）
    pub fn masked(&self) -> String {
        if self.is_email() {
            let parts: Vec<&str> = self.0.split('@').collect();
            if parts.len() == 2 {
                let local = parts[0];
                let domain = parts[1];
                let masked_local = if local.len() > 2 {
                    format!("{}***{}", &local[..1], &local[local.len()-1..])
                } else {
                    "***".to_string()
                };
                format!("{}@{}", masked_local, domain)
            } else {
                "***@***.com".to_string()
            }
        } else {
            // 手机号脱敏
            if self.0.len() == 11 {
                format!("{}****{}", &self.0[..3], &self.0[7..])
            } else {
                "***".to_string()
            }
        }
    }
}

impl fmt::Display for LoginId {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl From<LoginId> for String {
    fn from(login_id: LoginId) -> Self {
        login_id.0
    }
}

/// 密码值对象
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Password {
    plaintext: Option<String>, // 仅在创建时保存，验证后清除
    #[serde(skip_serializing_if = "Option::is_none")]
    hash: Option<String>,
}

impl Password {
    /// 从明文创建密码
    pub fn from_plaintext(plaintext: String) -> Result<Self, String> {
        Self::validate_plaintext(&plaintext)?;
        Ok(Self {
            plaintext: Some(plaintext),
            hash: None,
        })
    }
    
    /// 从哈希创建密码
    pub fn from_hash(hash: String) -> Self {
        Self {
            plaintext: None,
            hash: Some(hash),
        }
    }
    
    /// 验证明文密码格式
    fn validate_plaintext(password: &str) -> Result<(), String> {
        if password.len() < 8 {
            return Err("密码长度不能少于8个字符".to_string());
        }
        
        if password.len() > 64 {
            return Err("密码长度不能超过64个字符".to_string());
        }
        
        // 检查密码强度（至少包含字母和数字）
        let has_letter = password.chars().any(|c| c.is_alphabetic());
        let has_digit = password.chars().any(|c| c.is_ascii_digit());
        
        if !has_letter || !has_digit {
            return Err("密码必须包含字母和数字".to_string());
        }
        
        Ok(())
    }
    
    /// 获取明文密码（仅在创建时可用）
    pub fn take_plaintext(&mut self) -> Option<String> {
        self.plaintext.take()
    }
    
    /// 获取密码哈希
    pub fn hash(&self) -> Option<&str> {
        self.hash.as_deref()
    }
    
    /// 设置密码哈希
    pub fn set_hash(&mut self, hash: String) {
        self.hash = Some(hash);
        self.plaintext = None; // 清除明文
    }
    
    /// 检查是否已哈希
    pub fn is_hashed(&self) -> bool {
        self.hash.is_some()
    }
}

/// 用户昵称值对象
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Nickname(String);

impl Nickname {
    /// 创建新的昵称
    pub fn new(nickname: String) -> Result<Self, String> {
        Self::validate(&nickname)?;
        Ok(Self(nickname))
    }
    
    /// 验证昵称格式
    fn validate(nickname: &str) -> Result<(), String> {
        if nickname.is_empty() {
            return Err("昵称不能为空".to_string());
        }
        
        if nickname.len() > 32 {
            return Err("昵称长度不能超过32个字符".to_string());
        }
        
        // 检查是否包含不允许的字符
        if nickname.chars().any(|c| c.is_control()) {
            return Err("昵称不能包含控制字符".to_string());
        }
        
        Ok(())
    }
    
    /// 获取内部值
    pub fn as_str(&self) -> &str {
        &self.0
    }
    
    /// 创建默认昵称
    pub fn default_for_regular() -> Self {
        Self("用户".to_string())
    }
    
    /// 创建默认临时用户昵称
    pub fn default_for_temporary() -> Self {
        Self("临时用户".to_string())
    }
}

impl fmt::Display for Nickname {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl From<Nickname> for String {
    fn from(nickname: Nickname) -> Self {
        nickname.0
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_login_id_validation() {
        // 有效的邮箱
        assert!(LoginId::new("test@example.com".to_string()).is_ok());
        
        // 有效的手机号
        assert!(LoginId::new("13812345678".to_string()).is_ok());
        
        // 无效的邮箱
        assert!(LoginId::new("invalid-email".to_string()).is_err());
        
        // 无效的手机号
        assert!(LoginId::new("123".to_string()).is_err());
        
        // 空字符串
        assert!(LoginId::new("".to_string()).is_err());
    }

    #[test]
    fn test_login_id_masking() {
        let email = LoginId::new("test@example.com".to_string()).unwrap();
        assert_eq!(email.masked(), "t***t@example.com");
        
        let phone = LoginId::new("13812345678".to_string()).unwrap();
        assert_eq!(phone.masked(), "138****5678");
    }

    #[test]
    fn test_password_validation() {
        // 有效密码
        assert!(Password::from_plaintext("password123".to_string()).is_ok());
        
        // 太短
        assert!(Password::from_plaintext("123".to_string()).is_err());
        
        // 太长
        let long_password = "a".repeat(65);
        assert!(Password::from_plaintext(long_password).is_err());
        
        // 缺少数字
        assert!(Password::from_plaintext("passwordonly".to_string()).is_err());
        
        // 缺少字母
        assert!(Password::from_plaintext("12345678".to_string()).is_err());
    }

    #[test]
    fn test_nickname_validation() {
        // 有效昵称
        assert!(Nickname::new("测试用户".to_string()).is_ok());
        
        // 空昵称
        assert!(Nickname::new("".to_string()).is_err());
        
        // 太长的昵称
        let long_name = "a".repeat(33);
        assert!(Nickname::new(long_name).is_err());
    }
}

