/// 领域层错误定义
/// 
/// 纯业务错误，不包含任何技术实现细节

use thiserror::Error;

#[derive(Error, Debug, Clone, PartialEq)]
pub enum DomainError {
    // === 用户领域错误 ===
    #[error("用户不存在")]
    UserNotFound,
    
    #[error("用户已存在")]
    UserAlreadyExists,
    
    #[error("登录ID已存在")]
    LoginIdAlreadyExists,
    
    #[error("无效的用户凭据")]
    InvalidCredentials,
    
    #[error("密码验证失败")]
    PasswordVerificationFailed,
    
    #[error("临时用户已过期")]
    TempUserExpired,
    
    #[error("用户权限不足")]
    InsufficientPermissions,
    
    // === 群组领域错误 ===
    #[error("群组不存在")]
    GroupNotFound,
    
    #[error("群组名称已存在")]
    GroupNameAlreadyExists,
    
    #[error("群组已满")]
    GroupFull,
    
    #[error("用户不是群组成员")]
    NotGroupMember,
    
    #[error("用户已是群组成员")]
    AlreadyGroupMember,
    
    #[error("用户已是群组成员")]
    UserAlreadyMember,
    
    #[error("用户不是群组成员")]
    UserNotMember,
    
    #[error("无法离开自己创建的群组")]
    CannotLeaveOwnGroup,
    
    // === 消息领域错误 ===
    #[error("消息不存在")]
    MessageNotFound,
    
    #[error("消息内容为空")]
    EmptyMessageContent,
    
    #[error("消息内容过长")]
    MessageTooLong,
    
    #[error("会话不存在")]
    ConversationNotFound,
    
    // === 瞬间领域错误 ===
    #[error("瞬间不存在")]
    MomentNotFound,
    
    #[error("瞬间内容为空")]
    EmptyMomentContent,
    
    #[error("标签数量超限")]
    TooManyTags,
    
    #[error("图片数量超限")]
    TooManyImages,
    
    #[error("不是瞬间作者")]
    NotMomentAuthor,
    
    // === 地理位置领域错误 ===
    #[error("无效的地理坐标")]
    InvalidCoordinates,
    
    #[error("搜索半径超出限制")]
    SearchRadiusExceeded,
    
    // === 业务规则错误 ===
    #[error("违反业务规则: {rule}")]
    BusinessRuleViolation { rule: String },
    
    #[error("并发冲突")]
    ConcurrencyConflict,
    
    #[error("资源已被锁定")]
    ResourceLocked,
    
    // === 通用错误 ===
    #[error("资源不存在: {resource}")]
    NotFound { resource: String },
    
    #[error("未授权访问: {reason}")]
    Unauthorized { reason: String },
    
    // === 缓存错误 ===
    #[error("缓存操作失败")]
    CacheError,
    
    #[error("缓存序列化失败")]
    CacheSerializationError,
    
    #[error("缓存反序列化失败")]
    CacheDeserializationError,
    
    // === 验证错误 ===
    #[error("字段验证失败: {field} - {reason}")]
    ValidationError { field: String, reason: String },
    
    #[error("验证失败")]
    ValidationFailed,
    
    #[error("数据格式错误")]
    InvalidFormat,
}

impl DomainError {
    /// 用户相关错误的关联项
    pub const User: DomainError = DomainError::UserNotFound;
    
    /// 群组相关错误的关联项
    pub const Group: DomainError = DomainError::GroupNotFound;
    
    /// 瞬间相关错误的关联项
    pub const Moment: DomainError = DomainError::MomentNotFound;
    
    /// 缓存相关错误的关联项
    pub const Cache: DomainError = DomainError::CacheError;
    
    /// 创建验证错误
    pub fn validation(field: &str, reason: &str) -> Self {
        Self::ValidationError {
            field: field.to_string(),
            reason: reason.to_string(),
        }
    }
    
    /// 创建业务规则违反错误
    pub fn business_rule(rule: &str) -> Self {
        Self::BusinessRuleViolation {
            rule: rule.to_string(),
        }
    }
    
    /// 创建用户友好的错误消息
    pub fn user_message(&self) -> String {
        match self {
            Self::UserNotFound => "用户不存在".to_string(),
            Self::UserAlreadyExists => "用户已存在".to_string(),
            Self::InvalidCredentials => "用户名或密码错误".to_string(),
            Self::GroupNotFound => "群组不存在".to_string(),
            Self::MomentNotFound => "瞬间不存在".to_string(),
            Self::ValidationError { field, reason } => format!("{}: {}", field, reason),
            Self::ValidationFailed => "数据验证失败".to_string(),
            _ => self.to_string(),
        }
    }
    
    /// 检查是否为用户相关错误
    pub fn is_user_error(&self) -> bool {
        matches!(
            self,
            Self::UserNotFound
                | Self::UserAlreadyExists
                | Self::InvalidCredentials
                | Self::TempUserExpired
                | Self::InsufficientPermissions
        )
    }
    
    /// 检查是否为群组相关错误
    pub fn is_group_error(&self) -> bool {
        matches!(
            self,
            Self::GroupNotFound
                | Self::GroupFull
                | Self::NotGroupMember
                | Self::AlreadyGroupMember
                | Self::CannotLeaveOwnGroup
        )
    }
    
    /// 检查是否为验证错误
    pub fn is_validation_error(&self) -> bool {
        matches!(self, Self::ValidationError { .. } | Self::ValidationFailed | Self::InvalidFormat)
    }
    
    /// 检查是否为权限错误
    pub fn is_permission_error(&self) -> bool {
        matches!(self, Self::InsufficientPermissions)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_error_classification() {
        let user_error = DomainError::UserNotFound;
        assert!(user_error.is_user_error());
        assert!(!user_error.is_group_error());
        
        let validation_error = DomainError::validation("email", "invalid format");
        assert!(validation_error.is_validation_error());
        
        let business_error = DomainError::business_rule("maximum groups per user exceeded");
        assert_eq!(
            business_error.to_string(),
            "违反业务规则: maximum groups per user exceeded"
        );
    }
}

// === 特定领域错误类型别名 ===

/// 位置领域错误（向后兼容别名）
pub type LocationDomainError = DomainError;

/// 瞬间领域错误（向后兼容别名）
pub type MomentDomainError = DomainError;

/// 群组领域错误（向后兼容别名）
pub type GroupDomainError = DomainError;

/// 消息领域错误（向后兼容别名）
pub type MessageDomainError = DomainError;

/// 缓存领域错误（向后兼容别名）
pub type CacheDomainError = DomainError;

/// 作业领域错误（向后兼容别名）
pub type JobDomainError = DomainError;

/// 通知领域错误（向后兼容别名）
pub type NotificationDomainError = DomainError;

/// 用户领域错误（向后兼容别名）
pub type UserDomainError = DomainError;
