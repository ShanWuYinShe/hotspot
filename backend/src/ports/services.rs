/// 外部服务接口定义 - 基础设施服务抽象
/// 
/// 服务特征：
/// - 外部系统集成
/// - 技术能力提供
/// - 可替换实现
/// - 强类型接口

use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use uuid;
use crate::domain::{DomainResult, UserId};

// === 密码服务接口 ===

/// 密码服务接口
#[async_trait]
pub trait PasswordService: Send + Sync {
    /// 哈希密码
    async fn hash_password(&self, password: &str) -> DomainResult<String>;
    
    /// 验证密码
    async fn verify_password(&self, password: &str, hash: &str) -> DomainResult<bool>;
    
    /// 生成随机密码
    async fn generate_random_password(&self, length: usize) -> DomainResult<String>;
}

// === 认证服务接口 ===

/// JWT令牌服务接口
#[async_trait]
pub trait TokenService: Send + Sync + Clone {
    /// 生成访问令牌
    async fn generate_access_token(&self, user_id: UserId, expires_in_seconds: u64) -> DomainResult<String>;
    
    /// 生成刷新令牌
    async fn generate_refresh_token(&self, user_id: UserId) -> DomainResult<String>;
    
    /// 验证访问令牌
    async fn verify_access_token(&self, token: &str) -> DomainResult<TokenClaims>;
    
    /// 验证刷新令牌
    async fn verify_refresh_token(&self, token: &str) -> DomainResult<TokenClaims>;
    
    /// 撤销令牌
    async fn revoke_token(&self, token: &str) -> DomainResult<()>;
}

/// 令牌声明
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct TokenClaims {
    pub user_id: UserId,
    pub issued_at: u64,
    pub expires_at: u64,
    pub token_type: TokenType,
}

/// 令牌类型
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub enum TokenType {
    Access,
    Refresh,
}

// === 通知服务接口 ===

/// 通知传输服务接口
#[async_trait]
pub trait NotificationTransport: Send + Sync {
    /// 发送通知给单个用户
    async fn send_to_user(&self, user_id: UserId, notification: &Notification) -> DomainResult<()>;
    
    /// 发送通知给多个用户
    async fn send_to_users(&self, user_ids: &[UserId], notification: &Notification) -> DomainResult<()>;
    
    /// 发送通知给群组
    async fn send_to_group(&self, group_id: &str, notification: &Notification) -> DomainResult<()>;
    
    /// 广播通知（谨慎使用）
    async fn broadcast(&self, notification: &Notification) -> DomainResult<()>;
    
    /// 发送单个通知
    async fn send_notification(&self, user_id: i64, notification: &Notification) -> DomainResult<()>;
    
    /// 发送多个通知
    async fn send_notifications(&self, notifications: &[(i64, Notification)]) -> DomainResult<()>;
    
    /// 广播通知（别名方法）
    async fn broadcast_notification(&self, notification: &Notification) -> DomainResult<()>;
    
    /// 获取传输统计
    async fn get_transport_stats(&self) -> DomainResult<TransportStats>;
}

/// 通知消息
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct Notification {
    pub notification_id: String,
    pub user_id: Option<UserId>,
    pub notification_type: NotificationType,
    pub title: String,
    pub content: String,
    pub data: Option<serde_json::Value>,
    pub priority: NotificationPriority,
}

impl Notification {
    pub fn new(
        user_id: Option<UserId>,
        notification_type: NotificationType,
        title: String,
        content: String,
    ) -> Self {
        Self {
            notification_id: uuid::Uuid::new_v4().to_string(),
            user_id,
            notification_type,
            title,
            content,
            data: None,
            priority: NotificationPriority::Normal,
        }
    }

    pub fn with_data(mut self, data: serde_json::Value) -> Self {
        self.data = Some(data);
        self
    }

    pub fn with_priority(mut self, priority: NotificationPriority) -> Self {
        self.priority = priority;
        self
    }
}

/// 通知类型
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub enum NotificationType {
    System,
    Message,
    GroupInvite,
    GroupMessage,
    GroupInvitation,
    FriendRequest,
    SystemAnnouncement,
    MomentLike,
    MomentComment,
    GroupJoin,
    GroupLeave,
    LocationUpdate,
}

/// 通知优先级
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub enum NotificationPriority {
    Low,
    Normal,
    High,
    Critical,
}

/// 传输统计信息
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct TransportStats {
    pub total_sent: u64,
    pub total_failed: u64,
    pub active_connections: u64,
    pub last_activity: Option<chrono::DateTime<chrono::Utc>>,
}

// === 邮件服务接口 ===

/// 邮件服务接口
#[async_trait]
pub trait EmailService: Send + Sync + Clone {
    /// 发送欢迎邮件
    async fn send_welcome_email(&self, to: &str, user_name: &str) -> DomainResult<()>;
    
    /// 发送密码重置邮件
    async fn send_password_reset_email(&self, to: &str, reset_token: &str) -> DomainResult<()>;
    
    /// 发送验证邮件
    async fn send_verification_email(&self, to: &str, verification_token: &str) -> DomainResult<()>;
    
    /// 发送通知邮件
    async fn send_notification_email(&self, to: &str, subject: &str, content: &str) -> DomainResult<()>;
}

// === 文件存储服务接口 ===

/// 文件存储服务接口
#[async_trait]
pub trait FileStorageService: Send + Sync + Clone {
    /// 上传文件
    async fn upload_file(&self, file_data: &[u8], file_name: &str, content_type: &str) -> DomainResult<String>;
    
    /// 删除文件
    async fn delete_file(&self, file_url: &str) -> DomainResult<()>;
    
    /// 获取文件URL
    async fn get_file_url(&self, file_path: &str) -> DomainResult<String>;
    
    /// 生成预签名上传URL
    async fn generate_upload_url(&self, file_name: &str, content_type: &str) -> DomainResult<UploadUrl>;
}

/// 上传URL信息
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct UploadUrl {
    pub upload_url: String,
    pub file_url: String,
    pub expires_at: chrono::DateTime<chrono::Utc>,
}

// === 地理位置服务接口 ===

/// 地理位置服务接口
#[async_trait]
pub trait GeolocationService: Send + Sync + Clone {
    /// 根据坐标获取地址
    async fn reverse_geocode(&self, latitude: f64, longitude: f64) -> DomainResult<Address>;
    
    /// 根据地址获取坐标
    async fn geocode(&self, address: &str) -> DomainResult<Coordinates>;
    
    /// 计算两点之间的距离
    async fn calculate_distance(&self, from: &Coordinates, to: &Coordinates) -> DomainResult<f64>;
    
    /// 获取附近的兴趣点
    async fn get_nearby_pois(&self, location: &Coordinates, radius: f64) -> DomainResult<Vec<PointOfInterest>>;
}

/// 地址信息
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct Address {
    pub formatted_address: String,
    pub country: Option<String>,
    pub province: Option<String>,
    pub city: Option<String>,
    pub district: Option<String>,
    pub street: Option<String>,
}

/// 兴趣点
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PointOfInterest {
    pub name: String,
    pub category: String,
    pub coordinates: Coordinates,
    pub distance: f64,
}

// === 短信服务接口 ===

/// 短信服务接口
#[async_trait]
pub trait SmsService: Send + Sync + Clone {
    /// 发送验证码
    async fn send_verification_code(&self, phone: &str, code: &str) -> DomainResult<()>;
    
    /// 发送通知短信
    async fn send_notification(&self, phone: &str, message: &str) -> DomainResult<()>;
    
    /// 验证手机号格式
    async fn validate_phone_number(&self, phone: &str) -> DomainResult<bool>;
}

// === 推送服务接口 ===

/// 推送服务接口
#[async_trait]
pub trait PushService: Send + Sync + Clone {
    /// 发送推送通知
    async fn send_push_notification(&self, device_token: &str, notification: &PushNotification) -> DomainResult<()>;
    
    /// 批量发送推送通知
    async fn send_batch_push_notifications(&self, notifications: &[(String, PushNotification)]) -> DomainResult<()>;
    
    /// 订阅主题
    async fn subscribe_to_topic(&self, device_token: &str, topic: &str) -> DomainResult<()>;
    
    /// 取消订阅主题
    async fn unsubscribe_from_topic(&self, device_token: &str, topic: &str) -> DomainResult<()>;
}

/// 推送通知
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PushNotification {
    pub title: String,
    pub body: String,
    pub data: Option<serde_json::Value>,
    pub badge: Option<u32>,
    pub sound: Option<String>,
}

// === 服务工厂接口 ===

/// 服务工厂 - 创建和管理服务实例
pub trait ServiceFactory: Send + Sync + Clone {
    type PasswordSvc: PasswordService;
    type TokenSvc: TokenService;
    type NotificationTransport: NotificationTransport;
    type EmailSvc: EmailService;
    type FileStorageSvc: FileStorageService;
    type GeolocationSvc: GeolocationService;
    type SmsSvc: SmsService;
    type PushSvc: PushService;
    
    /// 创建密码服务
    fn password_service(&self) -> Self::PasswordSvc;
    
    /// 创建令牌服务
    fn token_service(&self) -> Self::TokenSvc;
    
    /// 创建通知传输服务
    fn notification_transport(&self) -> Self::NotificationTransport;
    
    /// 创建邮件服务
    fn email_service(&self) -> Self::EmailSvc;
    
    /// 创建文件存储服务
    fn file_storage_service(&self) -> Self::FileStorageSvc;
    
    /// 创建地理位置服务
    fn geolocation_service(&self) -> Self::GeolocationSvc;
    
    /// 创建短信服务
    fn sms_service(&self) -> Self::SmsSvc;
    
    /// 创建推送服务
    fn push_service(&self) -> Self::PushSvc;
}

// === 坐标类型重新导出 ===
use crate::domain::Coordinates;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_notification_creation() {
        let notification = Notification {
            notification_id: "test-123".to_string(),
            notification_type: NotificationType::Message,
            title: "Test Notification".to_string(),
            content: "This is a test".to_string(),
            data: None,
            priority: NotificationPriority::Normal,
        };
        
        assert_eq!(notification.title, "Test Notification");
        assert!(matches!(notification.notification_type, NotificationType::Message));
    }
    
    #[test]
    fn test_token_claims() {
        let user_id = UserId::new(1).unwrap();
        let claims = TokenClaims {
            user_id,
            issued_at: 1234567890,
            expires_at: 1234567890 + 3600,
            token_type: TokenType::Access,
        };
        
        assert_eq!(claims.user_id, user_id);
        assert!(matches!(claims.token_type, TokenType::Access));
    }
}

// === 类型别名（向后兼容） ===

/// 密码服务端口（向后兼容别名）
pub type PasswordServicePort = dyn PasswordService;

/// JWT令牌服务trait
#[async_trait]
pub trait JwtService: Send + Sync {
    /// 生成访问令牌
    async fn generate_access_token(&self, user_id: UserId, expires_in_seconds: u64) -> DomainResult<String>;
    
    /// 生成刷新令牌
    async fn generate_refresh_token(&self, user_id: UserId) -> DomainResult<String>;
    
    /// 验证令牌
    async fn verify_token(&self, token: &str) -> DomainResult<UserId>;
    
    /// 刷新令牌
    async fn refresh_token(&self, refresh_token: &str) -> DomainResult<(String, String)>;
}

/// JWT令牌服务端口（向后兼容别名）
pub type JwtServicePort = dyn JwtService;

/// 邮件服务端口（向后兼容别名）
pub type EmailServicePort = dyn EmailService;

/// 通知传输端口（向后兼容别名）
pub type NotificationTransportPort = dyn NotificationTransport;

/// 通知策略端口（向后兼容别名）
pub type NotificationPolicyPort = dyn NotificationPolicy;

// === 缺失的通知相关类型 ===

/// 通知策略trait
#[async_trait::async_trait]
pub trait NotificationPolicy: Send + Sync {
    /// 确定是否应该发送通知
    async fn should_notify(&self, context: &NotificationContext) -> DomainResult<bool>;
    
    /// 获取通知接收者列表
    async fn get_recipients(&self, context: &NotificationContext) -> DomainResult<Vec<i64>>;
    
    /// 过滤通知内容
    async fn filter_notification(&self, notification: &Notification, context: &NotificationContext) -> DomainResult<Option<Notification>>;
}

/// 通知上下文
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationContext {
    /// 用户ID
    pub user_id: crate::domain::UserId,
    /// 上下文数据
    pub context_data: std::collections::HashMap<String, String>,
}


/// 策略统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PolicyStats {
    /// 应用次数
    pub applied_count: u64,
    /// 跳过次数
    pub skipped_count: u64,
    /// 成功率
    pub success_rate: f64,
}

/// 通知渠道（向后兼容）
pub type NotificationChannel = dyn NotificationTransport;

/// 发布器统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PublisherStats {
    /// 发布成功数量
    pub published_count: u64,
    /// 发布失败数量
    pub failed_count: u64,
    /// 平均发布时间（毫秒）
    pub avg_publish_time_ms: f64,
}

/// 位置上下文
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LocationContext {
    pub latitude: f64,
    pub longitude: f64,
    pub accuracy: Option<f64>,
    pub timestamp: chrono::DateTime<chrono::Utc>,
}
