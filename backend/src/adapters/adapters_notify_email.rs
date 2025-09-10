/// 邮件通知传输适配器
/// 
/// 基于SMTP的邮件通知发送实现，支持HTML模板和异步发送
use async_trait::async_trait;
use lettre::{
    AsyncSmtpTransport, AsyncTransport, Tokio1Executor,
    message::{header::ContentType, Mailbox, Message},
    transport::smtp::{authentication::Credentials, PoolConfig},
    Address,
};
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;
use tracing::{debug, error, info, warn};

use crate::domain::{DomainResult, DomainError};
use crate::ports::{
    NotificationTransportPort, Notification, NotificationChannel,
    TransportStats,
};

/// 邮件配置
#[derive(Debug, Clone)]
pub struct EmailConfig {
    /// SMTP服务器地址
    pub smtp_host: String,
    /// SMTP端口
    pub smtp_port: u16,
    /// 用户名
    pub username: String,
    /// 密码
    pub password: String,
    /// 发件人邮箱
    pub from_email: String,
    /// 发件人姓名
    pub from_name: String,
    /// 连接池大小
    pub pool_size: u32,
    /// 连接超时时间（秒）
    pub timeout_seconds: u64,
    /// 是否启用TLS
    pub enable_tls: bool,
}

impl Default for EmailConfig {
    fn default() -> Self {
        Self {
            smtp_host: "localhost".to_string(),
            smtp_port: 587,
            username: "".to_string(),
            password: "".to_string(),
            from_email: "noreply@hotspot.com".to_string(),
            from_name: "HotSpot".to_string(),
            pool_size: 10,
            timeout_seconds: 30,
            enable_tls: true,
        }
    }
}

/// 邮件模板类型
#[derive(Debug, Clone, PartialEq)]
pub enum EmailTemplate {
    /// 系统通知
    SystemNotification,
    /// 欢迎邮件
    Welcome,
    /// 密码重置
    PasswordReset,
    /// 群组邀请
    GroupInvite,
    /// 好友请求
    FriendRequest,
    /// 瞬间通知
    MomentNotification,
    /// 通用通知
    Generic,
}

/// 邮件模板数据
#[derive(Debug, Clone)]
pub struct EmailTemplateData {
    /// 收件人姓名
    pub recipient_name: String,
    /// 邮件标题
    pub subject: String,
    /// 邮件内容
    pub content: String,
    /// 额外数据
    pub data: HashMap<String, serde_json::Value>,
}

/// 邮件通知传输实现
#[derive(Clone)]
pub struct EmailNotificationTransport {
    /// SMTP传输器
    smtp_transport: Option<Arc<AsyncSmtpTransport<Tokio1Executor>>>,
    /// 邮件配置
    config: EmailConfig,
    /// 发送统计
    stats: Arc<tokio::sync::Mutex<EmailStats>>,
}

/// 邮件发送统计
#[derive(Debug, Default)]
struct EmailStats {
    sent_count: u64,
    failed_count: u64,
    total_send_time_ms: u64,
}

impl EmailNotificationTransport {
    /// 创建新的邮件传输器
    pub fn new(config: EmailConfig) -> DomainResult<Self> {
        let smtp_transport = Self::create_smtp_transport(&config)?;
        
        Ok(Self {
            smtp_transport: Some(Arc::new(smtp_transport)),
            config,
            stats: Arc::new(tokio::sync::Mutex::new(EmailStats::default())),
        })
    }
    
    /// 创建占位实现（用于测试或禁用邮件功能时）
    pub fn placeholder() -> Self {
        Self {
            smtp_transport: None,
            config: EmailConfig::default(),
            stats: Arc::new(tokio::sync::Mutex::new(EmailStats::default())),
        }
    }
    
    /// 创建SMTP传输器
    fn create_smtp_transport(config: &EmailConfig) -> DomainResult<AsyncSmtpTransport<Tokio1Executor>> {
        let credentials = Credentials::new(config.username.clone(), config.password.clone());
        
        let pool_config = PoolConfig::new()
            .max_size(config.pool_size)
            .min_idle(1);
        
        let transport = if config.enable_tls {
            AsyncSmtpTransport::<Tokio1Executor>::starttls_relay(&config.smtp_host)
                .map_err(|e| DomainError::ExternalServiceError { 
                    service: "SMTP".to_string(), 
                    error: e.to_string() 
                })?
        } else {
            AsyncSmtpTransport::<Tokio1Executor>::builder_dangerous(&config.smtp_host)
        };
        
        let transport = transport
            .port(config.smtp_port)
            .credentials(credentials)
            .pool_config(pool_config)
            .timeout(Some(Duration::from_secs(config.timeout_seconds)))
            .build();
            
        Ok(transport)
    }
    
    /// 获取用户邮箱地址
    async fn get_user_email(&self, user_id: i64) -> DomainResult<Option<String>> {
        // TODO: 这里应该从用户仓库获取用户邮箱
        // 暂时返回占位邮箱用于测试
        debug!("获取用户{}的邮箱地址", user_id);
        Ok(Some(format!("user{}@example.com", user_id)))
    }
    
    /// 根据通知类型选择邮件模板
    fn select_template(&self, notification: &Notification) -> EmailTemplate {
        use crate::ports::NotificationType;
        
        match notification.notification_type {
            NotificationType::System => EmailTemplate::SystemNotification,
            NotificationType::GroupInvite => EmailTemplate::GroupInvite,
            NotificationType::FriendRequest => EmailTemplate::FriendRequest,
            NotificationType::MomentLike | NotificationType::MomentComment => EmailTemplate::MomentNotification,
            _ => EmailTemplate::Generic,
        }
    }
    
    /// 渲染邮件模板
    fn render_template(&self, template: EmailTemplate, data: EmailTemplateData) -> DomainResult<(String, String)> {
        let (subject, html_content) = match template {
            EmailTemplate::SystemNotification => {
                let subject = format!("[HotSpot] {}", data.subject);
                let html = format!(
                    r#"
                    <!DOCTYPE html>
                    <html>
                    <head>
                        <meta charset="UTF-8">
                        <title>{}</title>
                        <style>
                            body {{ font-family: Arial, sans-serif; line-height: 1.6; color: #333; }}
                            .container {{ max-width: 600px; margin: 0 auto; padding: 20px; }}
                            .header {{ background-color: #4CAF50; color: white; padding: 20px; text-align: center; }}
                            .content {{ padding: 20px; background-color: #f9f9f9; }}
                            .footer {{ padding: 10px; text-align: center; font-size: 12px; color: #666; }}
                        </style>
                    </head>
                    <body>
                        <div class="container">
                            <div class="header">
                                <h1>HotSpot 系统通知</h1>
                            </div>
                            <div class="content">
                                <p>亲爱的 {},</p>
                                <p>{}</p>
                            </div>
                            <div class="footer">
                                <p>此邮件由 HotSpot 系统自动发送，请勿回复。</p>
                            </div>
                        </div>
                    </body>
                    </html>
                    "#,
                    data.subject, data.recipient_name, data.content
                );
                (subject, html)
            },
            EmailTemplate::Welcome => {
                let subject = "欢迎加入 HotSpot！".to_string();
                let html = format!(
                    r#"
                    <!DOCTYPE html>
                    <html>
                    <head>
                        <meta charset="UTF-8">
                        <title>欢迎加入 HotSpot</title>
                        <style>
                            body {{ font-family: Arial, sans-serif; line-height: 1.6; color: #333; }}
                            .container {{ max-width: 600px; margin: 0 auto; padding: 20px; }}
                            .header {{ background-color: #2196F3; color: white; padding: 20px; text-align: center; }}
                            .content {{ padding: 20px; }}
                            .welcome-box {{ background-color: #e3f2fd; padding: 15px; border-radius: 5px; margin: 20px 0; }}
                            .footer {{ padding: 10px; text-align: center; font-size: 12px; color: #666; }}
                        </style>
                    </head>
                    <body>
                        <div class="container">
                            <div class="header">
                                <h1>🎉 欢迎加入 HotSpot！</h1>
                            </div>
                            <div class="content">
                                <p>亲爱的 {},</p>
                                <div class="welcome-box">
                                    <h3>欢迎来到 HotSpot 社区！</h3>
                                    <p>感谢您注册 HotSpot。您现在可以：</p>
                                    <ul>
                                        <li>📍 分享您的位置和瞬间</li>
                                        <li>👥 发现附近的朋友和群组</li>
                                        <li>💬 参与实时聊天和讨论</li>
                                        <li>🌟 点赞和评论精彩内容</li>
                                    </ul>
                                </div>
                                <p>开始探索您周围的精彩世界吧！</p>
                            </div>
                            <div class="footer">
                                <p>HotSpot 团队</p>
                            </div>
                        </div>
                    </body>
                    </html>
                    "#,
                    data.recipient_name
                );
                (subject, html)
            },
            EmailTemplate::GroupInvite => {
                let group_name = data.data.get("group_name")
                    .and_then(|v| v.as_str())
                    .unwrap_or("未知群组");
                let inviter_name = data.data.get("inviter_name")
                    .and_then(|v| v.as_str())
                    .unwrap_or("某位用户");
                    
                let subject = format!("您被邀请加入群组：{}", group_name);
                let html = format!(
                    r#"
                    <!DOCTYPE html>
                    <html>
                    <head>
                        <meta charset="UTF-8">
                        <title>群组邀请</title>
                        <style>
                            body {{ font-family: Arial, sans-serif; line-height: 1.6; color: #333; }}
                            .container {{ max-width: 600px; margin: 0 auto; padding: 20px; }}
                            .header {{ background-color: #FF9800; color: white; padding: 20px; text-align: center; }}
                            .content {{ padding: 20px; }}
                            .invite-box {{ background-color: #fff3e0; padding: 15px; border-radius: 5px; margin: 20px 0; }}
                            .footer {{ padding: 10px; text-align: center; font-size: 12px; color: #666; }}
                        </style>
                    </head>
                    <body>
                        <div class="container">
                            <div class="header">
                                <h1>🎊 群组邀请</h1>
                            </div>
                            <div class="content">
                                <p>亲爱的 {},</p>
                                <div class="invite-box">
                                    <h3>您收到了一个群组邀请！</h3>
                                    <p><strong>{}</strong> 邀请您加入群组 <strong>"{}"</strong></p>
                                    <p>{}</p>
                                </div>
                                <p>打开 HotSpot 应用查看详情并回应邀请。</p>
                            </div>
                            <div class="footer">
                                <p>HotSpot 团队</p>
                            </div>
                        </div>
                    </body>
                    </html>
                    "#,
                    data.recipient_name, inviter_name, group_name, data.content
                );
                (subject, html)
            },
            EmailTemplate::FriendRequest => {
                let requester_name = data.data.get("requester_name")
                    .and_then(|v| v.as_str())
                    .unwrap_or("某位用户");
                    
                let subject = format!("您收到了来自 {} 的好友请求", requester_name);
                let html = format!(
                    r#"
                    <!DOCTYPE html>
                    <html>
                    <head>
                        <meta charset="UTF-8">
                        <title>好友请求</title>
                        <style>
                            body {{ font-family: Arial, sans-serif; line-height: 1.6; color: #333; }}
                            .container {{ max-width: 600px; margin: 0 auto; padding: 20px; }}
                            .header {{ background-color: #E91E63; color: white; padding: 20px; text-align: center; }}
                            .content {{ padding: 20px; }}
                            .friend-box {{ background-color: #fce4ec; padding: 15px; border-radius: 5px; margin: 20px 0; }}
                            .footer {{ padding: 10px; text-align: center; font-size: 12px; color: #666; }}
                        </style>
                    </head>
                    <body>
                        <div class="container">
                            <div class="header">
                                <h1>👋 好友请求</h1>
                            </div>
                            <div class="content">
                                <p>亲爱的 {},</p>
                                <div class="friend-box">
                                    <h3>您收到了一个好友请求！</h3>
                                    <p><strong>{}</strong> 想要成为您的好友</p>
                                    <p>{}</p>
                                </div>
                                <p>打开 HotSpot 应用查看详情并回应请求。</p>
                            </div>
                            <div class="footer">
                                <p>HotSpot 团队</p>
                            </div>
                        </div>
                    </body>
                    </html>
                    "#,
                    data.recipient_name, requester_name, data.content
                );
                (subject, html)
            },
            EmailTemplate::MomentNotification => {
                let subject = "您的瞬间有新动态".to_string();
                let html = format!(
                    r#"
                    <!DOCTYPE html>
                    <html>
                    <head>
                        <meta charset="UTF-8">
                        <title>瞬间动态</title>
                        <style>
                            body {{ font-family: Arial, sans-serif; line-height: 1.6; color: #333; }}
                            .container {{ max-width: 600px; margin: 0 auto; padding: 20px; }}
                            .header {{ background-color: #9C27B0; color: white; padding: 20px; text-align: center; }}
                            .content {{ padding: 20px; }}
                            .moment-box {{ background-color: #f3e5f5; padding: 15px; border-radius: 5px; margin: 20px 0; }}
                            .footer {{ padding: 10px; text-align: center; font-size: 12px; color: #666; }}
                        </style>
                    </head>
                    <body>
                        <div class="container">
                            <div class="header">
                                <h1>✨ 瞬间动态</h1>
                            </div>
                            <div class="content">
                                <p>亲爱的 {},</p>
                                <div class="moment-box">
                                    <h3>您的瞬间有新动态！</h3>
                                    <p>{}</p>
                                </div>
                                <p>打开 HotSpot 应用查看详情。</p>
                            </div>
                            <div class="footer">
                                <p>HotSpot 团队</p>
                            </div>
                        </div>
                    </body>
                    </html>
                    "#,
                    data.recipient_name, data.content
                );
                (subject, html)
            },
            EmailTemplate::Generic => {
                let subject = format!("[HotSpot] {}", data.subject);
                let html = format!(
                    r#"
                    <!DOCTYPE html>
                    <html>
                    <head>
                        <meta charset="UTF-8">
                        <title>{}</title>
                        <style>
                            body {{ font-family: Arial, sans-serif; line-height: 1.6; color: #333; }}
                            .container {{ max-width: 600px; margin: 0 auto; padding: 20px; }}
                            .header {{ background-color: #607D8B; color: white; padding: 20px; text-align: center; }}
                            .content {{ padding: 20px; }}
                            .footer {{ padding: 10px; text-align: center; font-size: 12px; color: #666; }}
                        </style>
                    </head>
                    <body>
                        <div class="container">
                            <div class="header">
                                <h1>HotSpot 通知</h1>
                            </div>
                            <div class="content">
                                <p>亲爱的 {},</p>
                                <p>{}</p>
                            </div>
                            <div class="footer">
                                <p>HotSpot 团队</p>
                            </div>
                        </div>
                    </body>
                    </html>
                    "#,
                    data.subject, data.recipient_name, data.content
                );
                (subject, html)
            },
            EmailTemplate::PasswordReset => {
                let reset_link = data.data.get("reset_link")
                    .and_then(|v| v.as_str())
                    .unwrap_or("#");
                    
                let subject = "重置您的 HotSpot 密码".to_string();
                let html = format!(
                    r#"
                    <!DOCTYPE html>
                    <html>
                    <head>
                        <meta charset="UTF-8">
                        <title>密码重置</title>
                        <style>
                            body {{ font-family: Arial, sans-serif; line-height: 1.6; color: #333; }}
                            .container {{ max-width: 600px; margin: 0 auto; padding: 20px; }}
                            .header {{ background-color: #F44336; color: white; padding: 20px; text-align: center; }}
                            .content {{ padding: 20px; }}
                            .reset-box {{ background-color: #ffebee; padding: 15px; border-radius: 5px; margin: 20px 0; }}
                            .button {{ display: inline-block; padding: 10px 20px; background-color: #F44336; color: white; text-decoration: none; border-radius: 5px; }}
                            .footer {{ padding: 10px; text-align: center; font-size: 12px; color: #666; }}
                        </style>
                    </head>
                    <body>
                        <div class="container">
                            <div class="header">
                                <h1>🔒 密码重置</h1>
                            </div>
                            <div class="content">
                                <p>亲爱的 {},</p>
                                <div class="reset-box">
                                    <h3>密码重置请求</h3>
                                    <p>我们收到了您的密码重置请求。点击下面的按钮重置您的密码：</p>
                                    <p style="text-align: center;">
                                        <a href="{}" class="button">重置密码</a>
                                    </p>
                                    <p><strong>注意：</strong>此链接将在24小时后失效。</p>
                                </div>
                                <p>如果您没有请求重置密码，请忽略此邮件。</p>
                            </div>
                            <div class="footer">
                                <p>HotSpot 团队</p>
                            </div>
                        </div>
                    </body>
                    </html>
                    "#,
                    data.recipient_name, reset_link
                );
                (subject, html)
            },
        };
        
        Ok((subject, html_content))
    }
    
    /// 发送单个邮件
    async fn send_email_internal(
        &self,
        to_email: &str,
        to_name: &str,
        subject: &str,
        html_content: &str,
    ) -> DomainResult<()> {
        let Some(smtp_transport) = &self.smtp_transport else {
            warn!("SMTP传输器不可用，跳过邮件发送: {}", to_email);
            return Ok(());
        };
        
        let start_time = std::time::Instant::now();
        
        // 解析邮箱地址
        let to_address: Address = to_email.parse()
            .map_err(|e| DomainError::ValidationError { 
                field: "email".to_string(), 
                message: format!("无效的邮箱地址: {}", e) 
            })?;
            
        let from_address: Address = self.config.from_email.parse()
            .map_err(|e| DomainError::ConfigurationError { 
                message: format!("无效的发件人邮箱地址: {}", e) 
            })?;
        
        // 构建邮件
        let email = Message::builder()
            .from(Mailbox::new(
                Some(self.config.from_name.clone()),
                from_address,
            ))
            .to(Mailbox::new(Some(to_name.to_string()), to_address))
            .subject(subject)
            .header(ContentType::TEXT_HTML)
            .body(html_content.to_string())
            .map_err(|e| DomainError::ExternalServiceError { 
                service: "Email".to_string(), 
                error: format!("构建邮件失败: {}", e) 
            })?;
        
        // 发送邮件
        match smtp_transport.send(email).await {
            Ok(_) => {
                let elapsed = start_time.elapsed();
                let mut stats = self.stats.lock().await;
                stats.sent_count += 1;
                stats.total_send_time_ms += elapsed.as_millis() as u64;
                
                info!(
                    "邮件发送成功: {} -> {} ({}ms)",
                    self.config.from_email,
                    to_email,
                    elapsed.as_millis()
                );
                Ok(())
            },
            Err(e) => {
                let mut stats = self.stats.lock().await;
                stats.failed_count += 1;
                
                error!("邮件发送失败: {} -> {} - {}", self.config.from_email, to_email, e);
                Err(DomainError::ExternalServiceError { 
                    service: "SMTP".to_string(), 
                    error: e.to_string() 
                })
            }
        }
    }
}

#[async_trait]
impl NotificationTransportPort for EmailNotificationTransport {
    async fn send_notification(&self, notification: Notification) -> DomainResult<()> {
        // 检查通知是否包含邮件渠道
        if !notification.channels.contains(&NotificationChannel::Email) {
            debug!("通知不包含邮件渠道，跳过邮件发送: {}", notification.title);
            return Ok(());
        }
        
        // 获取用户邮箱
        let user_email = match self.get_user_email(notification.user_id).await? {
            Some(email) => email,
            None => {
                warn!("用户{}没有邮箱地址，跳过邮件发送", notification.user_id);
                return Ok(());
            }
        };
        
        // 选择邮件模板
        let template = self.select_template(&notification);
        
        // 准备模板数据
        let template_data = EmailTemplateData {
            recipient_name: format!("用户{}", notification.user_id), // TODO: 从用户仓库获取真实姓名
            subject: notification.title.clone(),
            content: notification.content.clone(),
            data: notification.data.clone(),
        };
        
        // 渲染模板
        let (subject, html_content) = self.render_template(template, template_data)?;
        
        // 发送邮件
        self.send_email_internal(
            &user_email,
            &format!("用户{}", notification.user_id),
            &subject,
            &html_content,
        ).await?;
        
        Ok(())
    }
    
    async fn send_notifications(&self, notifications: Vec<Notification>) -> DomainResult<()> {
        let mut errors = Vec::new();
        
        for notification in notifications {
            if let Err(e) = self.send_notification(notification).await {
                errors.push(e);
            }
        }
        
        if !errors.is_empty() {
            warn!("批量邮件发送中有{}个失败", errors.len());
            // 返回第一个错误，但不阻断整个流程
            return Err(errors.into_iter().next().unwrap());
        }
        
        Ok(())
    }
    
    async fn broadcast_notification(&self, notification: Notification) -> DomainResult<()> {
        // 邮件不适合广播，记录警告
        warn!(
            "邮件传输器不支持广播通知: {} - {}",
            notification.title,
            notification.content
        );
        Ok(())
    }
    
    async fn get_transport_stats(&self) -> DomainResult<TransportStats> {
        let stats = self.stats.lock().await;
        let avg_send_time = if stats.sent_count > 0 {
            stats.total_send_time_ms / stats.sent_count
        } else {
            0
        };
        
        Ok(TransportStats {
            sent_notifications: stats.sent_count,
            failed_notifications: stats.failed_count,
            avg_send_time_ms: avg_send_time,
        })
    }
}

/// 邮件配置构建器
pub struct EmailConfigBuilder {
    config: EmailConfig,
}

impl EmailConfigBuilder {
    pub fn new() -> Self {
        Self {
            config: EmailConfig::default(),
        }
    }
    
    pub fn smtp_host(mut self, host: String) -> Self {
        self.config.smtp_host = host;
        self
    }
    
    pub fn smtp_port(mut self, port: u16) -> Self {
        self.config.smtp_port = port;
        self
    }
    
    pub fn credentials(mut self, username: String, password: String) -> Self {
        self.config.username = username;
        self.config.password = password;
        self
    }
    
    pub fn from_email(mut self, email: String, name: String) -> Self {
        self.config.from_email = email;
        self.config.from_name = name;
        self
    }
    
    pub fn pool_size(mut self, size: u32) -> Self {
        self.config.pool_size = size;
        self
    }
    
    pub fn timeout(mut self, seconds: u64) -> Self {
        self.config.timeout_seconds = seconds;
        self
    }
    
    pub fn enable_tls(mut self, enable: bool) -> Self {
        self.config.enable_tls = enable;
        self
    }
    
    pub fn build(self) -> EmailConfig {
        self.config
    }
}

impl Default for EmailConfigBuilder {
    fn default() -> Self {
        Self::new()
    }
}
