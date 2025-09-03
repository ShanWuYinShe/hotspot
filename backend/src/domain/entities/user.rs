/// 用户领域实体
/// 
/// 包含用户的核心业务逻辑和状态

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// 用户类型枚举
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum UserType {
    /// 正式用户
    Regular,
    /// 临时用户
    Temporary,
}

impl UserType {
    pub fn as_str(&self) -> &'static str {
        match self {
            UserType::Regular => "regular",
            UserType::Temporary => "temporary",
        }
    }
    
    pub fn is_temporary(&self) -> bool {
        matches!(self, UserType::Temporary)
    }
}

/// 用户实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    /// 数据库自增主键
    pub id: i64,
    /// 业务用户ID
    pub user_id: i64,
    /// 登录ID（用户名）
    pub login_id: Option<String>,
    /// 用户昵称
    pub nickname: String,
    /// 密码哈希
    pub password_hash: Option<String>,
    /// 用户类型（通过expires_at判断）
    pub user_type: UserType,
    /// 邮箱
    pub email: Option<String>,
    /// 头像URL
    pub avatar_url: Option<String>,
    /// 过期时间（临时用户使用）
    pub expires_at: Option<DateTime<Utc>>,
    /// 创建时间
    pub created_at: DateTime<Utc>,
    /// 更新时间
    pub updated_at: DateTime<Utc>,
    /// 最后活跃时间
    pub last_active_at: DateTime<Utc>,
}

impl User {
    /// 创建新的正式用户
    pub fn new_regular(user_id: i64, login_id: String, nickname: String, password_hash: String) -> Self {
        let now = Utc::now();
        Self {
            id: 0, // 将由数据库生成
            user_id,
            login_id: Some(login_id),
            nickname,
            password_hash: Some(password_hash),
            user_type: UserType::Regular,
            email: None,
            avatar_url: None,
            expires_at: None,
            created_at: now,
            updated_at: now,
            last_active_at: now,
        }
    }
    
    /// 更新用户昵称
    pub fn update_nickname(&mut self, nickname: String) -> Result<(), String> {
        if nickname.is_empty() {
            return Err("昵称不能为空".to_string());
        }
        if nickname.len() > 32 {
            return Err("昵称长度不能超过32个字符".to_string());
        }
        self.nickname = nickname;
        self.updated_at = Utc::now();
        Ok(())
    }
    
    /// 创建新的临时用户
    pub fn new_temporary(user_id: i64, nickname: String, expires_at: DateTime<Utc>) -> Self {
        let now = Utc::now();
        Self {
            id: 0, // 将由数据库生成
            user_id,
            login_id: None,
            nickname,
            password_hash: None,
            user_type: UserType::Temporary,
            email: None,
            avatar_url: None,
            expires_at: Some(expires_at),
            created_at: now,
            updated_at: now,
            last_active_at: now,
        }
    }
    
    /// 检查用户是否有密码
    pub fn has_password(&self) -> bool {
        self.password_hash.is_some()
    }
    
    /// 检查用户是否为临时用户
    pub fn is_temporary(&self) -> bool {
        self.user_type.is_temporary()
    }
    
    /// 检查用户是否已过期（仅对临时用户有效）
    pub fn is_expired(&self) -> bool {
        if let Some(expires_at) = self.expires_at {
            Utc::now() > expires_at
        } else {
            false
        }
    }
    
    /// 更新最后活跃时间
    pub fn update_last_active(&mut self) {
        self.last_active_at = Utc::now();
    }
    
    /// 更新邮箱
    pub fn update_email(&mut self, email: Option<String>) -> Result<(), String> {
        if let Some(ref email) = email {
            if !email.contains('@') {
                return Err("邮箱格式无效".to_string());
            }
            if email.len() > 255 {
                return Err("邮箱长度不能超过255个字符".to_string());
            }
        }
        self.email = email;
        self.updated_at = Utc::now();
        Ok(())
    }
    
    /// 更新头像URL
    pub fn update_avatar_url(&mut self, avatar_url: Option<String>) {
        self.avatar_url = avatar_url;
        self.updated_at = Utc::now();
    }
    
    /// 升级临时用户为正式用户
    pub fn upgrade_to_regular(&mut self, login_id: String, password_hash: String) -> Result<(), String> {
        if !self.is_temporary() {
            return Err("只有临时用户可以升级为正式用户".to_string());
        }
        
        self.login_id = Some(login_id);
        self.password_hash = Some(password_hash);
        self.user_type = UserType::Regular;
        self.updated_at = Utc::now();
        
        Ok(())
    }
    
    /// 设置用户密码
    pub fn set_password(&mut self, password_hash: String) -> Result<(), String> {
        self.password_hash = Some(password_hash);
        self.updated_at = Utc::now();
        Ok(())
    }
    
    /// 验证用户数据的完整性
    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        
        if self.nickname.is_empty() {
            errors.push("昵称不能为空".to_string());
        }
        
        if self.nickname.len() > 32 {
            errors.push("昵称长度不能超过32个字符".to_string());
        }
        
        if self.user_type == UserType::Regular {
            if self.login_id.is_none() {
                errors.push("正式用户必须有登录ID".to_string());
            }
            
            if self.password_hash.is_none() {
                errors.push("正式用户必须有密码".to_string());
            }
        }
        
        if let Some(ref login_id) = self.login_id {
            if login_id.is_empty() || login_id.len() > 100 {
                errors.push("登录ID长度必须在1-100个字符之间".to_string());
            }
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
}

