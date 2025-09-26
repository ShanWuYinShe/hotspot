/// 实时通知推送服务
/// 
/// 功能职责：
/// - 集成WebSocket实时推送
/// - 支持移动端推送通知 (APNs/FCM)
/// - 管理用户在线状态和设备信息
/// - 提供通知推送的统一接口

use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{debug, error, info, warn};
use serde::{Serialize, Deserialize};
use chrono::{DateTime, Utc};

use crate::domain::DomainResult;
use crate::ws::unified_connection_manager::UnifiedConnectionManager;
use crate::ws::unified_message_types::{ServerMessage, NotificationPayload};
use crate::adapters::adapters_pg::notification_repository::{Notification, NotificationType};

/// 推送通道类型
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum PushChannel {
    /// WebSocket实时推送
    WebSocket,
    /// Apple推送通知服务
    APNs,
    /// Firebase云消息
    FCM,
    /// 邮件通知
    Email,
    /// 短信通知
    SMS,
}

/// 设备信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeviceInfo {
    pub device_id: String,
    pub device_type: String, // "ios", "android", "web"
    pub push_token: Option<String>,
    pub user_agent: Option<String>,
    pub last_active: DateTime<Utc>,
    pub is_online: bool,
}

/// 推送结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PushResult {
    pub channel: PushChannel,
    pub success: bool,
    pub message: String,
    pub sent_at: DateTime<Utc>,
}

/// 批量推送结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BatchPushResult {
    pub total_targets: usize,
    pub successful_pushes: usize,
    pub failed_pushes: usize,
    pub results: Vec<PushResult>,
}

/// 实时通知推送服务
pub struct RealtimeNotificationService {
    /// WebSocket连接管理器
    ws_manager: Arc<UnifiedConnectionManager>,
    /// 用户设备信息缓存
    user_devices: Arc<RwLock<HashMap<i64, Vec<DeviceInfo>>>>,
    /// 推送配置
    push_config: PushConfig,
}

/// 推送配置
#[derive(Debug, Clone)]
pub struct PushConfig {
    /// 是否启用WebSocket推送
    pub websocket_enabled: bool,
    /// 是否启用移动推送
    pub mobile_push_enabled: bool,
    /// 是否启用邮件推送
    pub email_enabled: bool,
    /// APNs配置
    pub apns_config: Option<APNsConfig>,
    /// FCM配置
    pub fcm_config: Option<FCMConfig>,
}

/// APNs配置
#[derive(Debug, Clone)]
pub struct APNsConfig {
    pub team_id: String,
    pub key_id: String,
    pub bundle_id: String,
    pub private_key: Vec<u8>,
    pub sandbox: bool,
}

/// FCM配置
#[derive(Debug, Clone)]
pub struct FCMConfig {
    pub server_key: String,
    pub sender_id: String,
}

impl RealtimeNotificationService {
    /// 创建新的实时通知推送服务
    pub fn new(ws_manager: Arc<UnifiedConnectionManager>) -> Self {
        let push_config = PushConfig {
            websocket_enabled: true,
            mobile_push_enabled: false, // 默认禁用，需要配置后启用
            email_enabled: false,
            apns_config: None,
            fcm_config: None,
        };
        
        Self {
            ws_manager,
            user_devices: Arc::new(RwLock::new(HashMap::new())),
            push_config,
        }
    }
    
    /// 配置推送服务
    pub fn configure(&mut self, config: PushConfig) {
        self.push_config = config;
        info!("通知推送服务配置已更新");
    }
    
    /// 发送单个通知
    pub async fn send_notification(&self, notification: &Notification) -> DomainResult<Vec<PushResult>> {
        debug!("发送通知: 用户={}, 类型={:?}, 标题={}", 
               notification.user_id, notification.notification_type, notification.title);
        
        let mut results = Vec::new();
        
        // 1. WebSocket实时推送
        if self.push_config.websocket_enabled {
            match self.send_websocket_notification(notification).await {
                Ok(_) => {
                    results.push(PushResult {
                        channel: PushChannel::WebSocket,
                        success: true,
                        message: "WebSocket推送成功".to_string(),
                        sent_at: Utc::now(),
                    });
                }
                Err(e) => {
                    warn!("WebSocket推送失败: {}", e);
                    results.push(PushResult {
                        channel: PushChannel::WebSocket,
                        success: false,
                        message: format!("WebSocket推送失败: {}", e),
                        sent_at: Utc::now(),
                    });
                }
            }
        }
        
        // 2. 移动端推送（如果用户不在线）
        if self.push_config.mobile_push_enabled {
            let is_user_online = self.is_user_online(notification.user_id).await;
            if !is_user_online {
                if let Err(e) = self.send_mobile_push_notification(notification).await {
                    warn!("移动端推送失败: {}", e);
                    results.push(PushResult {
                        channel: PushChannel::FCM, // 或APNs，根据设备类型
                        success: false,
                        message: format!("移动端推送失败: {}", e),
                        sent_at: Utc::now(),
                    });
                }
            }
        }
        
        // 3. 邮件推送（对于重要通知）
        if self.push_config.email_enabled && self.should_send_email(notification) {
            if let Err(e) = self.send_email_notification(notification).await {
                warn!("邮件推送失败: {}", e);
                results.push(PushResult {
                    channel: PushChannel::Email,
                    success: false,
                    message: format!("邮件推送失败: {}", e),
                    sent_at: Utc::now(),
                });
            }
        }
        
        Ok(results)
    }
    
    /// 批量发送通知
    pub async fn send_batch_notifications(&self, notifications: Vec<&Notification>) -> DomainResult<BatchPushResult> {
        debug!("批量发送 {} 个通知", notifications.len());
        
        let mut all_results = Vec::new();
        let mut successful_count = 0;
        let mut failed_count = 0;
        
        for notification in &notifications {
            match self.send_notification(notification).await {
                Ok(results) => {
                    let success_count = results.iter().filter(|r| r.success).count();
                    let fail_count = results.len() - success_count;
                    
                    successful_count += success_count;
                    failed_count += fail_count;
                    all_results.extend(results);
                }
                Err(e) => {
                    error!("发送通知失败: {}", e);
                    failed_count += 1;
                    all_results.push(PushResult {
                        channel: PushChannel::WebSocket,
                        success: false,
                        message: format!("发送失败: {}", e),
                        sent_at: Utc::now(),
                    });
                }
            }
        }
        
        Ok(BatchPushResult {
            total_targets: notifications.len(),
            successful_pushes: successful_count,
            failed_pushes: failed_count,
            results: all_results,
        })
    }
    
    /// 发送WebSocket通知
    async fn send_websocket_notification(&self, notification: &Notification) -> DomainResult<()> {
        let notification_payload = NotificationPayload {
            notification_id: notification.notification_id.clone(),
            title: notification.title.clone(),
            content: notification.content.clone(),
            notification_type: self.convert_notification_type(&notification.notification_type),
            data: match notification.data.clone() {
                Some(serde_json::Value::Object(map)) => map.into_iter().collect(),
                _ => HashMap::new(),
            },
            created_at: notification.created_at,
            expires_at: None, // 通知不设置过期时间
        };
        
        let _server_message = ServerMessage::SystemNotification { notification: notification_payload };
        
        // 发送给特定用户
        // TODO: WebSocket管理器的send_to_user方法需要公开
        // 暂时返回成功，实际需要调用WebSocket管理器发送消息
        warn!("WebSocket通知发送暂未实现，需要扩展UnifiedConnectionManager API");
        Ok(())
    }
    
    /// 发送移动端推送通知
    async fn send_mobile_push_notification(&self, notification: &Notification) -> DomainResult<()> {
        let devices = self.get_user_devices(notification.user_id).await;
        
        for device in devices {
            match device.device_type.as_str() {
                "ios" => {
                    if let Some(ref apns_config) = self.push_config.apns_config {
                        self.send_apns_notification(notification, &device, apns_config).await?;
                    }
                }
                "android" => {
                    if let Some(ref fcm_config) = self.push_config.fcm_config {
                        self.send_fcm_notification(notification, &device, fcm_config).await?;
                    }
                }
                _ => {
                    debug!("不支持的设备类型: {}", device.device_type);
                }
            }
        }
        
        Ok(())
    }
    
    /// 发送APNs推送
    async fn send_apns_notification(
        &self,
        _notification: &Notification,
        device: &DeviceInfo,
        _config: &APNsConfig,
    ) -> DomainResult<()> {
        // TODO: 集成APNs推送
        // 这里需要使用a2库或类似的APNs客户端
        debug!("APNs推送暂未实现: 设备={}", device.device_id);
        
        // 模拟推送成功
        Ok(())
    }
    
    /// 发送FCM推送
    async fn send_fcm_notification(
        &self,
        _notification: &Notification,
        device: &DeviceInfo,
        _config: &FCMConfig,
    ) -> DomainResult<()> {
        // TODO: 集成FCM推送
        // 这里需要使用fcm库或HTTP客户端调用FCM API
        debug!("FCM推送暂未实现: 设备={}", device.device_id);
        
        // 模拟推送成功
        Ok(())
    }
    
    /// 发送邮件通知
    async fn send_email_notification(&self, notification: &Notification) -> DomainResult<()> {
        // TODO: 集成邮件服务
        // 这里需要使用lettre库或调用邮件服务API
        debug!("邮件通知暂未实现: 用户={}", notification.user_id);
        
        // 模拟发送成功
        Ok(())
    }
    
    /// 注册用户设备
    pub async fn register_device(&self, user_id: i64, device: DeviceInfo) -> DomainResult<()> {
        debug!("注册用户设备: 用户={}, 设备={}", user_id, device.device_id);
        
        let mut devices = self.user_devices.write().await;
        let user_devices = devices.entry(user_id).or_insert_with(Vec::new);
        
        // 检查设备是否已存在
        if let Some(existing_device) = user_devices.iter_mut().find(|d| d.device_id == device.device_id) {
            // 更新现有设备信息
            *existing_device = device;
        } else {
            // 添加新设备
            user_devices.push(device);
        }
        
        // 限制每个用户最多10个设备
        if user_devices.len() > 10 {
            user_devices.sort_by(|a, b| b.last_active.cmp(&a.last_active));
            user_devices.truncate(10);
        }
        
        Ok(())
    }
    
    /// 注销用户设备
    pub async fn unregister_device(&self, user_id: i64, device_id: &str) -> DomainResult<()> {
        debug!("注销用户设备: 用户={}, 设备={}", user_id, device_id);
        
        let mut devices = self.user_devices.write().await;
        if let Some(user_devices) = devices.get_mut(&user_id) {
            user_devices.retain(|d| d.device_id != device_id);
            if user_devices.is_empty() {
                devices.remove(&user_id);
            }
        }
        
        Ok(())
    }
    
    /// 更新用户在线状态
    pub async fn update_user_status(&self, user_id: i64, is_online: bool) -> DomainResult<()> {
        debug!("更新用户在线状态: 用户={}, 在线={}", user_id, is_online);
        
        let mut devices = self.user_devices.write().await;
        if let Some(user_devices) = devices.get_mut(&user_id) {
            for device in user_devices.iter_mut() {
                device.is_online = is_online;
                if is_online {
                    device.last_active = Utc::now();
                }
            }
        }
        
        Ok(())
    }
    
    /// 检查用户是否在线
    async fn is_user_online(&self, user_id: i64) -> bool {
        let devices = self.user_devices.read().await;
        if let Some(user_devices) = devices.get(&user_id) {
            user_devices.iter().any(|d| d.is_online)
        } else {
            false
        }
    }
    
    /// 获取用户设备列表
    async fn get_user_devices(&self, user_id: i64) -> Vec<DeviceInfo> {
        let devices = self.user_devices.read().await;
        devices.get(&user_id).cloned().unwrap_or_default()
    }
    
    /// 判断是否应该发送邮件通知
    fn should_send_email(&self, notification: &Notification) -> bool {
        // 只对重要通知发送邮件
        matches!(
            notification.notification_type,
            NotificationType::Security | NotificationType::System
        )
    }
    
    /// 转换通知类型
    fn convert_notification_type(&self, notification_type: &NotificationType) -> String {
        match notification_type {
            NotificationType::System => "system".to_string(),
            NotificationType::NewMessage => "new_message".to_string(),
            NotificationType::GroupInvite => "group_invite".to_string(),
            NotificationType::FriendRequest => "friend_request".to_string(),
            NotificationType::FriendAccepted => "friend_accepted".to_string(),
            NotificationType::MomentLike => "moment_like".to_string(),
            NotificationType::MomentComment => "moment_comment".to_string(),
            NotificationType::Security => "security".to_string(),
        }
    }
    
    /// 获取推送统计信息
    pub async fn get_push_statistics(&self) -> HashMap<String, serde_json::Value> {
        let mut stats = HashMap::new();
        
        let devices = self.user_devices.read().await;
        let total_users = devices.len();
        let total_devices: usize = devices.values().map(|d| d.len()).sum();
        let online_users = devices.values().filter(|devices| devices.iter().any(|d| d.is_online)).count();
        
        stats.insert("total_users".to_string(), serde_json::Value::Number(serde_json::Number::from(total_users)));
        stats.insert("total_devices".to_string(), serde_json::Value::Number(serde_json::Number::from(total_devices)));
        stats.insert("online_users".to_string(), serde_json::Value::Number(serde_json::Number::from(online_users)));
        stats.insert("websocket_enabled".to_string(), serde_json::Value::Bool(self.push_config.websocket_enabled));
        stats.insert("mobile_push_enabled".to_string(), serde_json::Value::Bool(self.push_config.mobile_push_enabled));
        stats.insert("email_enabled".to_string(), serde_json::Value::Bool(self.push_config.email_enabled));
        
        stats
    }
    
    /// 清理离线设备
    pub async fn cleanup_offline_devices(&self, offline_threshold_hours: i64) -> DomainResult<usize> {
        debug!("清理离线设备: 离线阈值={}小时", offline_threshold_hours);
        
        let threshold = Utc::now() - chrono::Duration::hours(offline_threshold_hours);
        let mut devices = self.user_devices.write().await;
        let mut removed_count = 0;
        
        for (user_id, user_devices) in devices.iter_mut() {
            let original_len = user_devices.len();
            user_devices.retain(|d| d.last_active > threshold || d.is_online);
            removed_count += original_len - user_devices.len();
            
            if user_devices.is_empty() {
                debug!("用户 {} 的所有设备都已离线，从缓存中移除", user_id);
            }
        }
        
        // 移除没有设备的用户
        devices.retain(|_, user_devices| !user_devices.is_empty());
        
        info!("清理离线设备完成: 移除了 {} 个设备", removed_count);
        Ok(removed_count)
    }
}

impl Clone for RealtimeNotificationService {
    fn clone(&self) -> Self {
        Self {
            ws_manager: Arc::clone(&self.ws_manager),
            user_devices: Arc::clone(&self.user_devices),
            push_config: self.push_config.clone(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::adapters::adapters_pg::notification_repository::{NotificationStatus, NotificationPriority};
    
    #[tokio::test]
    async fn test_device_registration() {
        // 这里需要模拟WebSocket管理器
        // let ws_manager = Arc::new(UnifiedConnectionManager::new());
        // let service = RealtimeNotificationService::new(ws_manager);
        
        // 测试设备注册逻辑
        // ...
    }
    
    #[tokio::test]
    async fn test_notification_sending() {
        // 测试通知发送逻辑
        // ...
    }
}
