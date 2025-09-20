/// 用户领域实体 - 三层数据结构设计
/// 
/// 架构说明：
/// 1. Row结构体：仅在Repository层使用，对应数据库字段
/// 2. Domain模型：核心业务模型，在Application和Presentation层传递
/// 3. API模型：在Presentation层定义，用于HTTP响应序列化
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use crate::domain::{values::*, events::*, DomainError, DomainResult};

// ===== 核心实体 =====


/// 用户实体 - 用户聚合根
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    /// 用户ID（聚合根标识）
    pub id: UserId,
    /// 登录ID（可选，临时用户无登录ID）
    pub login_id: Option<String>,
    /// 用户昵称
    pub nickname: Nickname,
    /// 邮箱地址
    pub email: Option<String>,
    /// 头像URL
    pub avatar_url: Option<String>,
    /// 个人简介
    pub bio: Option<String>,
    /// 密码哈希（可选，临时用户无密码）
    pub password_hash: Option<String>,
    /// 用户类型
    pub user_type: UserType,
    /// 过期时间（临时用户）
    pub expires_at: Option<DateTime<Utc>>,
    /// 创建时间
    pub created_at: DateTime<Utc>,
    /// 最后活跃时间
    pub last_active_at: DateTime<Utc>,
    /// 当前位置
    pub location: Option<UserLocation>,
    /// 领域事件（待发布）
    #[serde(skip)]
    pub events: Vec<DomainEvent>,
}

impl User {
    /// 创建新用户
    pub fn new(
        id: UserId,
        nickname: Nickname,
        user_type: UserType,
    ) -> Self {
        let now = Utc::now();
        
        let mut user = Self {
            id,
            login_id: None,
            nickname,
            email: None,
            avatar_url: None,
            bio: None,
            password_hash: None,
            user_type,
            expires_at: if user_type == UserType::Temporary {
                Some(now + chrono::Duration::days(7)) // 临时用户7天后过期
            } else {
                None
            },
            created_at: now,
            last_active_at: now,
            location: None,
            events: Vec::new(),
        };
        
        // 发布用户创建事件
        user.add_event(DomainEvent::UserCreated {
            user_id: id,
            user_type,
            created_at: now,
        });
        
        user
    }
    
    /// 添加领域事件
    pub fn add_event(&mut self, event: DomainEvent) {
        self.events.push(event);
    }
    
    /// 获取并清空领域事件
    pub fn take_events(&mut self) -> Vec<DomainEvent> {
        std::mem::take(&mut self.events)
    }
    
    /// 更新用户资料
    pub fn update_profile(
        &mut self,
        nickname: Option<Nickname>,
        email: Option<String>,
        bio: Option<String>,
        avatar_url: Option<String>,
    ) -> DomainResult<()> {
        let mut updated = false;
        
        if let Some(new_nickname) = nickname {
            if self.nickname != new_nickname {
                self.nickname = new_nickname;
                updated = true;
            }
        }
        
        if email.is_some() && self.email != email {
            self.email = email;
            updated = true;
        }
        
        if bio.is_some() && self.bio != bio {
            self.bio = bio;
            updated = true;
        }
        
        if avatar_url.is_some() && self.avatar_url != avatar_url {
            self.avatar_url = avatar_url;
            updated = true;
        }
        
        if updated {
            self.add_event(DomainEvent::UserProfileUpdated {
                user_id: self.id,
                updated_at: Utc::now(),
            });
        }
        
        Ok(())
    }
    
    /// 设置密码
    pub fn set_password(&mut self, password_hash: String) -> DomainResult<()> {
        self.password_hash = Some(password_hash);
        
        self.add_event(DomainEvent::UserPasswordSet {
            user_id: self.id,
            set_at: Utc::now(),
        });
        
        Ok(())
    }
    
    /// 更新位置
    pub fn update_location(&mut self, location: UserLocation) -> DomainResult<()> {
        self.location = Some(location.clone());
        self.last_active_at = Utc::now();
        
        self.add_event(DomainEvent::UserLocationUpdated {
            user_id: self.id,
            old_location: None, // TODO: 需要保存旧位置
            new_location: location,
            updated_at: chrono::Utc::now(),
        });
        
        Ok(())
    }
    
    /// 升级临时用户为正式用户
    pub fn upgrade_to_regular(&mut self, login_id: String, password_hash: String) -> DomainResult<()> {
        if self.user_type != UserType::Temporary {
            return Err(DomainError::BusinessRuleViolation {
                rule: "Only temporary users can be upgraded".to_string(),
            });
        }
        
        self.user_type = UserType::Regular;
        self.login_id = Some(login_id);
        self.password_hash = Some(password_hash);
        self.expires_at = None;
        
        self.add_event(DomainEvent::UserUpgraded {
            user_id: self.id,
            upgraded_at: Utc::now(),
        });
        
        Ok(())
    }
    
    /// 检查用户是否过期
    pub fn is_expired(&self) -> bool {
        if let Some(expires_at) = self.expires_at {
            Utc::now() > expires_at
        } else {
            false
        }
    }
    
    /// 检查是否为临时用户
    pub fn is_temporary(&self) -> bool {
        self.user_type == UserType::Temporary
    }
}

/// 用户位置信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserLocation {
    /// 地理坐标
    pub coordinates: Coordinates,
    /// 位置名称
    pub location_name: String,
    /// 更新时间
    pub updated_at: DateTime<Utc>,
}

/// 用户类型枚举
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "user_type", rename_all = "lowercase")]
pub enum UserType {
    /// 临时用户 (类型码: 1)
    Temporary,
    /// 正式用户 (类型码: 2)
    Regular,
    /// 系统用户 (类型码: 3)
    System,
}

impl UserType {
    pub fn as_str(&self) -> &'static str {
        match self {
            UserType::Temporary => "temporary",
            UserType::Regular => "regular",
            UserType::System => "system",
        }
    }
    
    pub fn from_str(s: &str) -> Result<Self, String> {
        match s {
            "temporary" => Ok(UserType::Temporary),
            "regular" => Ok(UserType::Regular),
            "system" => Ok(UserType::System),
            _ => Err(format!("Invalid user type: {}", s)),
        }
    }
    
    /// 获取用户类型编码（用于用户ID编码）
    pub fn type_code(&self) -> u8 {
        match self {
            UserType::Temporary => 1,
            UserType::Regular => 2,
            UserType::System => 3,
        }
    }
    
    /// 从类型编码创建用户类型
    pub fn from_type_code(code: u8) -> Result<Self, String> {
        match code {
            1 => Ok(UserType::Temporary),
            2 => Ok(UserType::Regular),
            3 => Ok(UserType::System),
            _ => Err(format!("Invalid user type code: {}", code)),
        }
    }
    
    pub fn is_temporary(&self) -> bool {
        matches!(self, UserType::Temporary)
    }
    
    pub fn is_regular(&self) -> bool {
        matches!(self, UserType::Regular)
    }
    
    pub fn is_system(&self) -> bool {
        matches!(self, UserType::System)
    }
}

// sqlx类型映射由 derive(sqlx::Type) 自动生成
// PostgreSQL 是核心功能，始终启用

// ===== 用户ID编码工具函数 =====

/// 用户ID编码工具
pub struct UserIdEncoder;

impl UserIdEncoder {
    /// 从用户ID中提取用户类型
    /// 用户ID的高8位表示用户类型
    pub fn get_user_type(user_id: i64) -> Result<UserType, String> {
        let type_code = ((user_id as u64) >> 56) as u8;
        UserType::from_type_code(type_code)
    }
    
    /// 判断是否为临时用户
    pub fn is_temp_user(user_id: i64) -> bool {
        matches!(Self::get_user_type(user_id), Ok(UserType::Temporary))
    }
    
    /// 判断是否为正式用户
    pub fn is_regular_user(user_id: i64) -> bool {
        matches!(Self::get_user_type(user_id), Ok(UserType::Regular))
    }
    
    /// 判断是否为系统用户
    pub fn is_system_user(user_id: i64) -> bool {
        matches!(Self::get_user_type(user_id), Ok(UserType::System))
    }
    
    /// 编码用户ID（将用户类型编码到高8位）
    pub fn encode_user_id(base_id: i64, user_type: UserType) -> i64 {
        let type_code = user_type.type_code() as u64;
        let encoded = (type_code << 56) | (base_id as u64 & 0x00FFFFFFFFFFFFFF);
        encoded as i64
    }
    
    /// 从用户ID中提取基础ID（去除类型编码）
    pub fn extract_base_id(user_id: i64) -> i64 {
        (user_id as u64 & 0x00FFFFFFFFFFFFFF) as i64
    }
}

// ===== 领域模型层 (Domain Models) =====
// 这些是核心业务模型，在Application和Presentation层之间传递

/// 用户基础资料（最常用的轻量级模型）
/// 用途：用户列表、搜索结果、资料展示、聊天显示等90%的场景
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserProfile {
    /// 业务用户ID
    pub user_id: i64,
    /// 用户昵称
    pub nickname: String,
    /// 头像URL
    pub avatar_url: Option<String>,
    /// 用户类型
    pub user_type: UserType,
    /// 最后活跃时间
    pub last_active_at: DateTime<Utc>,
}

/// 用户认证信息（仅用于认证场景）
/// 用途：登录验证、密码检查等认证相关操作
/// 注意：不实现Serialize，防止意外序列化敏感信息
#[derive(Debug, Clone)]
pub struct UserAuth {
    /// 业务用户ID
    pub user_id: i64,
    /// 登录ID（用户名）
    pub login_id: Option<String>,
    /// 用户昵称
    pub nickname: String,
    /// 密码哈希（仅认证时需要）
    pub password_hash: Option<String>,
    /// 用户类型
    pub user_type: UserType,
    /// 头像URL
    pub avatar_url: Option<String>,
}

/// 用户详细信息（包含敏感信息，仅特定场景使用）
/// 用途：用户设置页面、账户管理等需要完整信息的场景
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserDetail {
    /// 业务用户ID
    pub user_id: i64,
    /// 登录ID（用户名）
    pub login_id: Option<String>,
    /// 用户昵称
    pub nickname: String,
    /// 邮箱
    pub email: Option<String>,
    /// 头像URL
    pub avatar_url: Option<String>,
    /// 用户类型
    pub user_type: UserType,
    /// 最后活跃时间
    pub last_active_at: DateTime<Utc>,
    /// 创建时间
    pub created_at: DateTime<Utc>,
    /// 过期时间（临时用户）
    pub expires_at: Option<DateTime<Utc>>,
}

/// 用户搜索结果（专用于搜索场景）
/// 用途：搜索API返回，包含距离等搜索相关字段
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserSearchResult {
    /// 业务用户ID
    pub user_id: i64,
    /// 用户昵称
    pub nickname: String,
    /// 头像URL
    pub avatar_url: Option<String>,
    /// 用户类型
    pub user_type: UserType,
    /// 最后活跃时间
    pub last_active_at: DateTime<Utc>,
    /// 距离（米，仅地理位置搜索时有值）
    pub distance: Option<f64>,
    /// 是否有位置信息
    pub has_location: bool,
}

// ===== 领域模型实现 =====

impl UserProfile {
    /// 创建新的用户资料
    pub fn new(
        user_id: i64,
        nickname: String,
        avatar_url: Option<String>,
        user_type: UserType,
        last_active_at: DateTime<Utc>,
    ) -> Self {
        Self {
            user_id,
            nickname,
            avatar_url,
            user_type,
            last_active_at,
        }
    }

    /// 判断是否为临时用户
    pub fn is_temporary(&self) -> bool {
        self.user_type.is_temporary()
    }
}

impl UserAuth {
    /// 创建新的用户认证信息
    pub fn new(
        user_id: i64,
        login_id: Option<String>,
        nickname: String,
        password_hash: Option<String>,
        user_type: UserType,
        avatar_url: Option<String>,
    ) -> Self {
        Self {
            user_id,
            login_id,
            nickname,
            password_hash,
            user_type,
            avatar_url,
        }
    }

    /// 验证密码哈希是否存在
    pub fn has_password(&self) -> bool {
        self.password_hash.is_some()
    }

    /// 获取密码哈希（仅用于验证）
    pub fn get_password_hash(&self) -> Option<&str> {
        self.password_hash.as_deref()
    }

    /// 转换为用户资料（移除敏感信息）
    pub fn to_profile(&self) -> UserProfile {
        UserProfile {
            user_id: self.user_id,
            nickname: self.nickname.clone(),
            avatar_url: self.avatar_url.clone(),
            user_type: self.user_type,
            last_active_at: Utc::now(), // 认证时更新活跃时间
        }
    }
}

impl UserDetail {
    /// 创建新的用户详细信息
    pub fn new(
        user_id: i64,
        login_id: Option<String>,
        nickname: String,
        email: Option<String>,
        avatar_url: Option<String>,
        user_type: UserType,
        last_active_at: DateTime<Utc>,
        created_at: DateTime<Utc>,
        expires_at: Option<DateTime<Utc>>,
    ) -> Self {
        Self {
            user_id,
            login_id,
            nickname,
            email,
            avatar_url,
            user_type,
            last_active_at,
            created_at,
            expires_at,
        }
    }

    /// 转换为用户资料
    pub fn to_profile(&self) -> UserProfile {
        UserProfile {
            user_id: self.user_id,
            nickname: self.nickname.clone(),
            avatar_url: self.avatar_url.clone(),
            user_type: self.user_type,
            last_active_at: self.last_active_at,
        }
    }

    /// 判断是否已过期（仅临时用户）
    pub fn is_expired(&self) -> bool {
        if let Some(expires_at) = self.expires_at {
            Utc::now() > expires_at
        } else {
            false
        }
    }
}

impl UserSearchResult {
    /// 创建新的用户搜索结果
    pub fn new(
        user_id: i64,
        nickname: String,
        avatar_url: Option<String>,
        user_type: UserType,
        last_active_at: DateTime<Utc>,
        distance: Option<f64>,
        has_location: bool,
    ) -> Self {
        Self {
            user_id,
            nickname,
            avatar_url,
            user_type,
            last_active_at,
            distance,
            has_location,
        }
    }

    /// 从用户资料创建搜索结果
    pub fn from_profile(profile: UserProfile, distance: Option<f64>, has_location: bool) -> Self {
        Self {
            user_id: profile.user_id,
            nickname: profile.nickname,
            avatar_url: profile.avatar_url,
            user_type: profile.user_type,
            last_active_at: profile.last_active_at,
            distance,
            has_location,
        }
    }

    /// 转换为用户资料
    pub fn to_profile(&self) -> UserProfile {
        UserProfile {
            user_id: self.user_id,
            nickname: self.nickname.clone(),
            avatar_url: self.avatar_url.clone(),
            user_type: self.user_type,
            last_active_at: self.last_active_at,
        }
    }
}

// ===== 类型转换实现 =====

impl From<UserDetail> for UserProfile {
    fn from(detail: UserDetail) -> Self {
        detail.to_profile()
    }
}

impl From<UserAuth> for UserProfile {
    fn from(auth: UserAuth) -> Self {
        auth.to_profile()
    }
}

impl From<UserSearchResult> for UserProfile {
    fn from(result: UserSearchResult) -> Self {
        result.to_profile()
    }
}