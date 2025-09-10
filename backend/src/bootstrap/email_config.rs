/// 邮件配置模块
/// 
/// 提供邮件服务的配置管理和环境变量解析
use std::env;
use tracing::{debug, warn};

#[cfg(feature = "email_channel")]
use crate::adapters::{EmailConfig, EmailConfigBuilder};

/// 邮件配置结构
#[derive(Debug, Clone)]
pub struct EmailSettings {
    /// 是否启用邮件功能
    pub enabled: bool,
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

impl Default for EmailSettings {
    fn default() -> Self {
        Self {
            enabled: false,
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

impl EmailSettings {
    /// 从环境变量加载邮件配置
    pub fn from_env() -> Self {
        let mut settings = Self::default();
        
        // 检查是否启用邮件功能
        settings.enabled = env::var("ENABLE_EMAIL")
            .unwrap_or_else(|_| "false".to_string())
            .parse()
            .unwrap_or(false);
            
        if !settings.enabled {
            debug!("邮件功能未启用");
            return settings;
        }
        
        // SMTP服务器配置
        if let Ok(host) = env::var("SMTP_HOST") {
            settings.smtp_host = host;
        }
        
        if let Ok(port) = env::var("SMTP_PORT") {
            settings.smtp_port = port.parse().unwrap_or(587);
        }
        
        // 认证信息
        if let Ok(username) = env::var("SMTP_USERNAME") {
            settings.username = username;
        }
        
        if let Ok(password) = env::var("SMTP_PASSWORD") {
            settings.password = password;
        }
        
        // 发件人信息
        if let Ok(from_email) = env::var("SMTP_FROM_EMAIL") {
            settings.from_email = from_email;
        }
        
        if let Ok(from_name) = env::var("SMTP_FROM_NAME") {
            settings.from_name = from_name;
        }
        
        // 连接池配置
        if let Ok(pool_size) = env::var("SMTP_POOL_SIZE") {
            settings.pool_size = pool_size.parse().unwrap_or(10);
        }
        
        if let Ok(timeout) = env::var("SMTP_TIMEOUT_SECONDS") {
            settings.timeout_seconds = timeout.parse().unwrap_or(30);
        }
        
        // TLS配置
        if let Ok(enable_tls) = env::var("SMTP_ENABLE_TLS") {
            settings.enable_tls = enable_tls.parse().unwrap_or(true);
        }
        
        // 验证必需的配置
        if settings.enabled
            && (settings.username.is_empty() || settings.password.is_empty()) {
            warn!("邮件功能已启用但缺少认证信息，将禁用邮件功能");
            settings.enabled = false;
        }
        
        debug!("邮件配置加载完成: enabled={}, host={}:{}", 
               settings.enabled, settings.smtp_host, settings.smtp_port);
        
        settings
    }
    
    /// 转换为EmailConfig（仅在启用email_channel功能时可用）
    #[cfg(feature = "email_channel")]
    pub fn to_email_config(&self) -> EmailConfig {
        EmailConfigBuilder::new()
            .smtp_host(self.smtp_host.clone())
            .smtp_port(self.smtp_port)
            .credentials(self.username.clone(), self.password.clone())
            .from_email(self.from_email.clone(), self.from_name.clone())
            .pool_size(self.pool_size)
            .timeout(self.timeout_seconds)
            .enable_tls(self.enable_tls)
            .build()
    }
    
    /// 验证配置
    pub fn validate(&self) -> Vec<String> {
        let mut warnings = Vec::new();
        
        if !self.enabled {
            return warnings;
        }
        
        if self.smtp_host.is_empty() {
            warnings.push("SMTP主机地址为空".to_string());
        }
        
        if self.smtp_port == 0 {
            warnings.push("SMTP端口无效".to_string());
        }
        
        if self.username.is_empty() {
            warnings.push("SMTP用户名为空".to_string());
        }
        
        if self.password.is_empty() {
            warnings.push("SMTP密码为空".to_string());
        }
        
        if self.from_email.is_empty() {
            warnings.push("发件人邮箱为空".to_string());
        } else if !self.from_email.contains('@') {
            warnings.push("发件人邮箱格式无效".to_string());
        }
        
        if self.from_name.is_empty() {
            warnings.push("发件人姓名为空".to_string());
        }
        
        if self.pool_size == 0 {
            warnings.push("连接池大小不能为0".to_string());
        }
        
        if self.timeout_seconds == 0 {
            warnings.push("超时时间不能为0".to_string());
        }
        
        warnings
    }
    
    /// 获取配置摘要（用于日志）
    pub fn summary(&self) -> String {
        if !self.enabled {
            return "邮件功能已禁用".to_string();
        }
        
        format!(
            "邮件配置: {}:{} (TLS: {}, 池大小: {}, 超时: {}s)",
            self.smtp_host,
            self.smtp_port,
            self.enable_tls,
            self.pool_size,
            self.timeout_seconds
        )
    }
}

/// 邮件服务工厂
pub struct EmailServiceFactory;

impl EmailServiceFactory {
    /// 创建邮件传输器（仅在启用email_channel功能时可用）
    #[cfg(feature = "email_channel")]
    pub fn create_email_transport(
        settings: &EmailSettings,
    ) -> Result<Option<crate::adapters::EmailNotificationTransport>, Box<dyn std::error::Error>> {
        if !settings.enabled {
            debug!("邮件功能未启用，返回空传输器");
            return Ok(None);
        }
        
        let config = settings.to_email_config();
        let transport = crate::adapters::EmailNotificationTransport::new(config)?;
        
        debug!("邮件传输器创建成功");
        Ok(Some(transport))
    }
    
    /// 创建占位邮件传输器（功能未启用时）
    #[cfg(not(feature = "email_channel"))]
    pub fn create_email_transport(
        _settings: &EmailSettings,
    ) -> Result<Option<()>, Box<dyn std::error::Error>> {
        debug!("email_channel功能未启用，跳过邮件传输器创建");
        Ok(None)
    }
    
    /// 创建复合通知传输器
    #[cfg(all(feature = "ws", feature = "email_channel"))]
    pub fn create_composite_transport(
        email_settings: &EmailSettings,
        websocket_transport: std::sync::Arc<crate::adapters::WebSocketNotificationTransport>,
    ) -> Result<crate::adapters::CompositeNotificationTransport, Box<dyn std::error::Error>> {
        use crate::adapters::NotificationTransportBuilder;
        
        let mut builder = NotificationTransportBuilder::new()
            .with_websocket(websocket_transport);
        
        if let Some(email_transport) = Self::create_email_transport(email_settings)? {
            builder = builder.with_email(std::sync::Arc::new(email_transport));
        }
        
        Ok(builder.build())
    }
    
    /// 创建仅WebSocket的传输器（email功能未启用时）
    #[cfg(all(feature = "ws", not(feature = "email_channel")))]
    pub fn create_composite_transport(
        _email_settings: &EmailSettings,
        websocket_transport: std::sync::Arc<crate::adapters::WebSocketNotificationTransport>,
    ) -> Result<crate::adapters::CompositeNotificationTransport, Box<dyn std::error::Error>> {
        use crate::adapters::NotificationTransportBuilder;
        
        let builder = NotificationTransportBuilder::new()
            .with_websocket(websocket_transport);
        
        Ok(builder.build())
    }
    
    /// 创建邮件任务处理器（仅在启用email_channel功能时可用）
    #[cfg(feature = "email_channel")]
    pub fn create_email_job_handler<T, U>(
        transport: std::sync::Arc<T>,
        user_repository: std::sync::Arc<U>,
    ) -> crate::application::EmailJobHandler<T, U>
    where
        T: crate::ports::NotificationTransportPort,
        U: crate::ports::UserRepositoryPort,
    {
        crate::application::EmailJobHandler::new(transport, user_repository)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_default_email_settings() {
        let settings = EmailSettings::default();
        assert!(!settings.enabled);
        assert_eq!(settings.smtp_host, "localhost");
        assert_eq!(settings.smtp_port, 587);
        assert!(settings.enable_tls);
    }
    
    #[test]
    fn test_email_settings_validation() {
        let mut settings = EmailSettings::default();
        settings.enabled = true;
        
        let warnings = settings.validate();
        assert!(!warnings.is_empty());
        assert!(warnings.iter().any(|w| w.contains("用户名")));
        assert!(warnings.iter().any(|w| w.contains("密码")));
    }
    
    #[test]
    fn test_email_settings_summary() {
        let settings = EmailSettings::default();
        let summary = settings.summary();
        assert!(summary.contains("已禁用"));
        
        let mut enabled_settings = settings;
        enabled_settings.enabled = true;
        let summary = enabled_settings.summary();
        assert!(summary.contains("localhost:587"));
    }
}
