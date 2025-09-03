/// 领域错误模块
/// 
/// 定义领域层的错误类型，遵循领域语言和业务规则

use thiserror::Error;

/// 用户领域错误
#[derive(Error, Debug, Clone)]
pub enum UserDomainError {
    #[error("用户不存在: {user_id}")]
    UserNotFound { user_id: i64 },
    
    #[error("登录ID已存在: {login_id}")]
    LoginIdAlreadyExists { login_id: String },
    
    #[error("无效的用户类型转换: {reason}")]
    InvalidUserTypeConversion { reason: String },
    
    #[error("临时用户操作限制: {operation}")]
    TemporaryUserRestriction { operation: String },
    
    #[error("用户数据验证失败: {errors:?}")]
    ValidationFailed { errors: Vec<String> },
    
    #[error("密码验证失败")]
    PasswordVerificationFailed,
    
    #[error("无效的凭据")]
    InvalidCredentials,
    
    #[error("用户已被禁用")]
    UserDisabled,
    
    #[error("位置信息无效: {reason}")]
    InvalidLocation { reason: String },
    
    #[error("权限不足: {required}")]
    InsufficientPermissions { required: String },
}

/// 群组领域错误
#[derive(Error, Debug, Clone)]
pub enum GroupDomainError {
    #[error("群组不存在: {group_id}")]
    GroupNotFound { group_id: i64 },
    
    #[error("群组名称已存在: {name}")]
    GroupNameAlreadyExists { name: String },
    
    #[error("权限不足: {required_role}")]
    InsufficientPermission { required_role: String },
    
    #[error("用户不是群组成员: user_id={user_id}, group_id={group_id}")]
    UserNotMember { user_id: i64, group_id: i64 },
    
    #[error("用户已是群组成员: user_id={user_id}, group_id={group_id}")]
    UserAlreadyMember { user_id: i64, group_id: i64 },
    
    #[error("不能移除群主")]
    CannotRemoveOwner,
    
    #[error("群组成员数量已达上限: {max_members}")]
    MemberLimitExceeded { max_members: u32 },
    
    #[error("群组已解散")]
    GroupDissolved,
    
    #[error("群组数据验证失败: {errors:?}")]
    ValidationFailed { errors: Vec<String> },
    
    #[error("不能对相同或更高级别的角色执行操作")]
    CannotManageSameOrHigherRole,
}

/// 消息领域错误
#[derive(Error, Debug, Clone)]
pub enum MessageDomainError {
    #[error("消息不存在: {message_id}")]
    MessageNotFound { message_id: i64 },
    
    #[error("消息已被删除: {message_id}")]
    MessageDeleted { message_id: i64 },
    
    #[error("不是消息发送者")]
    NotMessageSender,
    
    #[error("消息内容为空")]
    EmptyContent,
    
    #[error("消息内容过长: {max_length}")]
    ContentTooLong { max_length: usize },
    
    #[error("不支持的消息类型: {message_type}")]
    UnsupportedMessageType { message_type: String },
    
    #[error("回复的消息不存在: {parent_id}")]
    ParentMessageNotFound { parent_id: i64 },
    
    #[error("消息数据验证失败: {errors:?}")]
    ValidationFailed { errors: Vec<String> },
}

/// 瞬间领域错误
#[derive(Error, Debug, Clone)]
pub enum MomentDomainError {
    #[error("瞬间不存在: {moment_id}")]
    MomentNotFound { moment_id: i64 },
    
    #[error("瞬间已被删除: {moment_id}")]
    MomentDeleted { moment_id: i64 },
    
    #[error("不是瞬间作者")]
    NotMomentAuthor,
    
    #[error("瞬间内容和图片都为空")]
    EmptyContentAndImages,
    
    #[error("图片数量超过限制: {max_images}")]
    TooManyImages { max_images: usize },
    
    #[error("无效的可见性设置: {visibility}")]
    InvalidVisibility { visibility: String },
    
    #[error("瞬间数据验证失败: {errors:?}")]
    ValidationFailed { errors: Vec<String> },
}

/// 地理位置领域错误
#[derive(Error, Debug, Clone)]
pub enum LocationDomainError {
    #[error("无效的坐标: latitude={latitude}, longitude={longitude}")]
    InvalidCoordinates { latitude: f64, longitude: f64 },
    
    #[error("地址格式无效: {address}")]
    InvalidAddress { address: String },
    
    #[error("超出搜索半径限制: {radius} > {max_radius}")]
    RadiusExceeded { radius: f64, max_radius: f64 },
    
    #[error("位置服务不可用")]
    LocationServiceUnavailable,
}

/// 缓存领域错误
#[derive(Error, Debug, Clone)]
pub enum CacheDomainError {
    #[error("缓存键不存在: {key}")]
    KeyNotFound { key: String },
    
    #[error("缓存连接失败")]
    ConnectionFailed,
    
    #[error("序列化错误: {0}")]
    SerializationError(String),
    
    #[error("反序列化错误: {0}")]
    DeserializationError(String),
    
    #[error("缓存已满")]
    CacheFull,
    
    #[error("缓存操作超时")]
    OperationTimeout,
    
    #[error("缓存服务不可用")]
    ServiceUnavailable,
}

/// 通用领域错误
#[derive(Error, Debug, Clone)]
pub enum DomainError {
    #[error("用户错误: {0}")]
    User(#[from] UserDomainError),
    
    #[error("群组错误: {0}")]
    Group(#[from] GroupDomainError),
    
    #[error("消息错误: {0}")]
    Message(#[from] MessageDomainError),
    
    #[error("瞬间错误: {0}")]
    Moment(#[from] MomentDomainError),
    
    #[error("位置错误: {0}")]
    Location(#[from] LocationDomainError),
    
    #[error("缓存错误: {0}")]
    Cache(#[from] CacheDomainError),
    
    #[error("业务规则违反: {rule}")]
    BusinessRuleViolation { rule: String },
    
    #[error("并发冲突: {resource}")]
    ConcurrencyConflict { resource: String },
    
    #[error("操作已取消")]
    OperationCancelled,
    
    #[error("资源已锁定: {resource}")]
    ResourceLocked { resource: String },
}

/// 领域错误结果类型
pub type DomainResult<T> = Result<T, DomainError>;

impl From<String> for DomainError {
    fn from(error: String) -> Self {
        DomainError::BusinessRuleViolation { rule: error }
    }
}

impl DomainError {
    /// 检查是否为未找到错误
    pub fn is_not_found(&self) -> bool {
        matches!(
            self,
            DomainError::User(UserDomainError::UserNotFound { .. })
                | DomainError::Group(GroupDomainError::GroupNotFound { .. })
                | DomainError::Message(MessageDomainError::MessageNotFound { .. })
                | DomainError::Moment(MomentDomainError::MomentNotFound { .. })
        )
    }
    
    /// 检查是否为权限错误
    pub fn is_permission_error(&self) -> bool {
        matches!(
            self,
            DomainError::Group(GroupDomainError::InsufficientPermission { .. })
                | DomainError::Message(MessageDomainError::NotMessageSender)
                | DomainError::Moment(MomentDomainError::NotMomentAuthor)
        )
    }
    
    /// 检查是否为验证错误
    pub fn is_validation_error(&self) -> bool {
        matches!(
            self,
            DomainError::User(UserDomainError::ValidationFailed { .. })
                | DomainError::Group(GroupDomainError::ValidationFailed { .. })
                | DomainError::Message(MessageDomainError::ValidationFailed { .. })
                | DomainError::Moment(MomentDomainError::ValidationFailed { .. })
        )
    }
    
    /// 检查是否为冲突错误
    pub fn is_conflict_error(&self) -> bool {
        matches!(
            self,
            DomainError::User(UserDomainError::LoginIdAlreadyExists { .. })
                | DomainError::Group(GroupDomainError::GroupNameAlreadyExists { .. })
                | DomainError::Group(GroupDomainError::UserAlreadyMember { .. })
                | DomainError::ConcurrencyConflict { .. }
        )
    }
    
    /// 获取错误的用户友好消息
    pub fn user_message(&self) -> String {
        match self {
            // 用户错误的用户友好消息
            DomainError::User(UserDomainError::UserNotFound { .. }) => {
                "用户不存在".to_string()
            }
            DomainError::User(UserDomainError::LoginIdAlreadyExists { .. }) => {
                "该登录账号已被使用".to_string()
            }
            DomainError::User(UserDomainError::PasswordVerificationFailed) => {
                "密码错误".to_string()
            }
            
            // 群组错误的用户友好消息
            DomainError::Group(GroupDomainError::GroupNotFound { .. }) => {
                "群组不存在".to_string()
            }
            DomainError::Group(GroupDomainError::InsufficientPermission { .. }) => {
                "权限不足".to_string()
            }
            DomainError::Group(GroupDomainError::UserAlreadyMember { .. }) => {
                "您已经是群组成员".to_string()
            }
            
            // 其他错误使用原始消息
            _ => self.to_string(),
        }
    }
}
