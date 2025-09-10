/// API限流中间件
/// 
/// 提供基于用户和IP的请求限流功能，防止API滥用
use axum::{
    extract::{Request, State},
    http::{HeaderMap, StatusCode},
    middleware::Next,
    response::{Response, IntoResponse},
    Json,
};
use std::collections::HashMap;
use std::net::IpAddr;
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};
use serde_json::json;
use tracing::{debug, warn, info};

/// 限流配置
#[derive(Debug, Clone)]
pub struct RateLimitConfig {
    /// 每分钟最大请求数（按用户）
    pub requests_per_minute_per_user: u32,
    /// 每分钟最大请求数（按IP）
    pub requests_per_minute_per_ip: u32,
    /// 窗口大小（秒）
    pub window_size_seconds: u64,
    /// 是否启用限流
    pub enabled: bool,
    /// 白名单IP列表
    pub whitelist_ips: Vec<IpAddr>,
    /// 管理员用户ID列表（不受限流限制）
    pub admin_user_ids: Vec<i64>,
}

impl Default for RateLimitConfig {
    fn default() -> Self {
        Self {
            requests_per_minute_per_user: 100,
            requests_per_minute_per_ip: 200,
            window_size_seconds: 60,
            enabled: true,
            whitelist_ips: vec![
                "127.0.0.1".parse().unwrap(),
                "::1".parse().unwrap(),
            ],
            admin_user_ids: Vec::new(),
        }
    }
}

/// 限流记录
#[derive(Debug, Clone)]
struct RateLimitRecord {
    requests: Vec<Instant>,
    blocked_until: Option<Instant>,
    total_requests: u64,
    blocked_requests: u64,
}

impl RateLimitRecord {
    fn new() -> Self {
        Self {
            requests: Vec::new(),
            blocked_until: None,
            total_requests: 0,
            blocked_requests: 0,
        }
    }

    fn add_request(&mut self, now: Instant, window_duration: Duration) {
        // 清理过期的请求记录
        self.requests.retain(|&time| now.duration_since(time) < window_duration);
        
        // 添加新请求
        self.requests.push(now);
        self.total_requests += 1;
    }

    fn is_rate_limited(&mut self, now: Instant, max_requests: u32, window_duration: Duration) -> bool {
        // 检查是否在阻塞期内
        if let Some(blocked_until) = self.blocked_until {
            if now < blocked_until {
                self.blocked_requests += 1;
                return true;
            } else {
                self.blocked_until = None;
            }
        }

        // 清理过期的请求记录
        self.requests.retain(|&time| now.duration_since(time) < window_duration);

        // 检查是否超过限制
        if self.requests.len() >= max_requests as usize {
            // 设置阻塞时间（窗口剩余时间）
            if let Some(&oldest_request) = self.requests.first() {
                let remaining_window = window_duration - now.duration_since(oldest_request);
                self.blocked_until = Some(now + remaining_window);
            }
            self.blocked_requests += 1;
            return true;
        }

        false
    }

    #[allow(dead_code)]
    fn get_remaining_requests(&self, max_requests: u32) -> u32 {
        max_requests.saturating_sub(self.requests.len() as u32)
    }

    fn get_reset_time(&self, window_duration: Duration) -> Option<Instant> {
        self.requests.first().map(|&first| first + window_duration)
    }
}

/// 限流器状态
pub struct RateLimiterState {
    config: RateLimitConfig,
    user_records: Arc<Mutex<HashMap<i64, RateLimitRecord>>>,
    ip_records: Arc<Mutex<HashMap<IpAddr, RateLimitRecord>>>,
    stats: Arc<Mutex<RateLimitStats>>,
}

/// 限流统计
#[derive(Debug, Clone, Default)]
pub struct RateLimitStats {
    pub total_requests: u64,
    pub blocked_requests: u64,
    pub user_blocks: u64,
    pub ip_blocks: u64,
    pub active_user_limits: usize,
    pub active_ip_limits: usize,
}

impl RateLimiterState {
    pub fn new(config: RateLimitConfig) -> Self {
        Self {
            config,
            user_records: Arc::new(Mutex::new(HashMap::new())),
            ip_records: Arc::new(Mutex::new(HashMap::new())),
            stats: Arc::new(Mutex::new(RateLimitStats::default())),
        }
    }

    /// 检查用户限流
    fn check_user_rate_limit(&self, user_id: i64) -> Result<(), RateLimitResponse> {
        if !self.config.enabled {
            return Ok(());
        }

        // 检查是否为管理员用户
        if self.config.admin_user_ids.contains(&user_id) {
            return Ok(());
        }

        let now = Instant::now();
        let window_duration = Duration::from_secs(self.config.window_size_seconds);
        let mut user_records = self.user_records.lock().unwrap();
        
        let record = user_records.entry(user_id).or_insert_with(RateLimitRecord::new);
        
        if record.is_rate_limited(now, self.config.requests_per_minute_per_user, window_duration) {
            let mut stats = self.stats.lock().unwrap();
            stats.blocked_requests += 1;
            stats.user_blocks += 1;
            
            warn!("用户{}触发限流", user_id);
            
            return Err(RateLimitResponse {
                error: "用户请求频率过高".to_string(),
                retry_after: record.get_reset_time(window_duration)
                    .map(|t| t.duration_since(now).as_secs())
                    .unwrap_or(60),
                limit: self.config.requests_per_minute_per_user,
                remaining: 0,
            });
        }

        record.add_request(now, window_duration);
        Ok(())
    }

    /// 检查IP限流
    fn check_ip_rate_limit(&self, ip: IpAddr) -> Result<(), RateLimitResponse> {
        if !self.config.enabled {
            return Ok(());
        }

        // 检查是否为白名单IP
        if self.config.whitelist_ips.contains(&ip) {
            return Ok(());
        }

        let now = Instant::now();
        let window_duration = Duration::from_secs(self.config.window_size_seconds);
        let mut ip_records = self.ip_records.lock().unwrap();
        
        let record = ip_records.entry(ip).or_insert_with(RateLimitRecord::new);
        
        if record.is_rate_limited(now, self.config.requests_per_minute_per_ip, window_duration) {
            let mut stats = self.stats.lock().unwrap();
            stats.blocked_requests += 1;
            stats.ip_blocks += 1;
            
            warn!("IP{}触发限流", ip);
            
            return Err(RateLimitResponse {
                error: "IP请求频率过高".to_string(),
                retry_after: record.get_reset_time(window_duration)
                    .map(|t| t.duration_since(now).as_secs())
                    .unwrap_or(60),
                limit: self.config.requests_per_minute_per_ip,
                remaining: 0,
            });
        }

        record.add_request(now, window_duration);
        Ok(())
    }

    /// 获取限流统计
    pub fn get_stats(&self) -> RateLimitStats {
        let mut stats = self.stats.lock().unwrap();
        
        // 更新活跃限制数量
        stats.active_user_limits = self.user_records.lock().unwrap().len();
        stats.active_ip_limits = self.ip_records.lock().unwrap().len();
        
        stats.clone()
    }

    /// 清理过期记录
    pub fn cleanup_expired_records(&self) {
        let now = Instant::now();
        let window_duration = Duration::from_secs(self.config.window_size_seconds);
        
        // 清理用户记录
        let mut user_records = self.user_records.lock().unwrap();
        user_records.retain(|_, record| {
            record.requests.retain(|&time| now.duration_since(time) < window_duration);
            !record.requests.is_empty() || record.blocked_until.map_or(false, |t| now < t)
        });
        
        // 清理IP记录
        let mut ip_records = self.ip_records.lock().unwrap();
        ip_records.retain(|_, record| {
            record.requests.retain(|&time| now.duration_since(time) < window_duration);
            !record.requests.is_empty() || record.blocked_until.map_or(false, |t| now < t)
        });
        
        debug!("清理限流记录完成，剩余用户记录: {}, IP记录: {}", 
               user_records.len(), ip_records.len());
    }

    /// 重置用户限流
    pub fn reset_user_limit(&self, user_id: i64) {
        let mut user_records = self.user_records.lock().unwrap();
        user_records.remove(&user_id);
        info!("重置用户{}的限流状态", user_id);
    }

    /// 重置IP限流
    pub fn reset_ip_limit(&self, ip: IpAddr) {
        let mut ip_records = self.ip_records.lock().unwrap();
        ip_records.remove(&ip);
        info!("重置IP{}的限流状态", ip);
    }
}

/// 限流响应
#[derive(Debug)]
struct RateLimitResponse {
    error: String,
    retry_after: u64,
    limit: u32,
    remaining: u32,
}

/// 提取客户端IP地址
fn extract_client_ip(headers: &HeaderMap) -> Option<IpAddr> {
    // 尝试从各种头部获取真实IP
    let ip_headers = [
        "x-forwarded-for",
        "x-real-ip", 
        "cf-connecting-ip",
        "x-client-ip",
    ];
    
    for &header_name in &ip_headers {
        if let Some(header_value) = headers.get(header_name) {
            if let Ok(header_str) = header_value.to_str() {
                // X-Forwarded-For 可能包含多个IP，取第一个
                let ip_str = header_str.split(',').next().unwrap_or("").trim();
                if let Ok(ip) = ip_str.parse::<IpAddr>() {
                    return Some(ip);
                }
            }
        }
    }
    
    None
}

/// 限流中间件
pub async fn rate_limit_middleware(
    State(rate_limiter): State<Arc<RateLimiterState>>,
    headers: HeaderMap,
    request: Request,
    next: Next,
) -> Result<Response, Response> {
    // 更新总请求统计
    {
        let mut stats = rate_limiter.stats.lock().unwrap();
        stats.total_requests += 1;
    }

    // 提取客户端IP
    let client_ip = extract_client_ip(&headers)
        .unwrap_or_else(|| "127.0.0.1".parse().unwrap());

    // 检查IP限流
    if let Err(rate_limit_response) = rate_limiter.check_ip_rate_limit(client_ip) {
        let response_body = json!({
            "code": 1007,
            "msg": rate_limit_response.error,
            "data": {
                "retry_after": rate_limit_response.retry_after,
                "limit": rate_limit_response.limit,
                "remaining": rate_limit_response.remaining
            }
        });
        
        return Err((StatusCode::OK, Json(response_body)).into_response());
    }

    // 尝试从请求中提取用户ID（如果有JWT token）
    if let Some(auth_header) = headers.get("authorization") {
        if let Ok(auth_str) = auth_header.to_str() {
            if auth_str.starts_with("Bearer ") {
                let token = &auth_str[7..];
                // 这里应该解析JWT获取用户ID，简化处理
                // 在实际实现中，应该使用JWT解析库
                if let Ok(user_id) = extract_user_id_from_token(token) {
                    // 检查用户限流
                    if let Err(rate_limit_response) = rate_limiter.check_user_rate_limit(user_id) {
                        let response_body = json!({
                            "code": 1007,
                            "msg": rate_limit_response.error,
                            "data": {
                                "retry_after": rate_limit_response.retry_after,
                                "limit": rate_limit_response.limit,
                                "remaining": rate_limit_response.remaining
                            }
                        });
                        
                        return Err((StatusCode::OK, Json(response_body)).into_response());
                    }
                }
            }
        }
    }

    debug!("请求通过限流检查: IP={}", client_ip);
    
    // 继续处理请求
    Ok(next.run(request).await)
}

/// 从JWT token中提取用户ID（简化实现）
fn extract_user_id_from_token(_token: &str) -> Result<i64, &'static str> {
    // 这里应该使用实际的JWT解析逻辑
    // 为了演示，我们返回一个错误，表示需要实际实现
    Err("JWT parsing not implemented")
}

/// 创建限流中间件状态
pub fn create_rate_limiter(config: RateLimitConfig) -> Arc<RateLimiterState> {
    let rate_limiter = Arc::new(RateLimiterState::new(config));
    
    // 启动清理任务
    let cleanup_limiter = rate_limiter.clone();
    tokio::spawn(async move {
        let mut interval = tokio::time::interval(Duration::from_secs(300)); // 每5分钟清理一次
        loop {
            interval.tick().await;
            cleanup_limiter.cleanup_expired_records();
        }
    });
    
    info!("限流中间件初始化完成");
    rate_limiter
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_rate_limit_record() {
        let mut record = RateLimitRecord::new();
        let now = Instant::now();
        let window = Duration::from_secs(60);
        
        // 添加请求
        record.add_request(now, window);
        assert_eq!(record.requests.len(), 1);
        assert_eq!(record.total_requests, 1);
        
        // 测试限流
        assert!(!record.is_rate_limited(now, 5, window));
        
        // 添加更多请求直到触发限流
        for _ in 0..5 {
            record.add_request(now, window);
        }
        
        assert!(record.is_rate_limited(now, 5, window));
        assert!(record.blocked_requests > 0);
    }

    #[test]
    fn test_rate_limiter_config() {
        let config = RateLimitConfig::default();
        assert_eq!(config.requests_per_minute_per_user, 100);
        assert_eq!(config.requests_per_minute_per_ip, 200);
        assert!(config.enabled);
        assert!(!config.whitelist_ips.is_empty());
    }

    #[test]
    fn test_client_ip_extraction() {
        let mut headers = HeaderMap::new();
        
        // 测试X-Forwarded-For
        headers.insert("x-forwarded-for", "192.168.1.1, 10.0.0.1".parse().unwrap());
        let ip = extract_client_ip(&headers);
        assert_eq!(ip, Some("192.168.1.1".parse().unwrap()));
        
        // 测试X-Real-IP
        headers.clear();
        headers.insert("x-real-ip", "203.0.113.1".parse().unwrap());
        let ip = extract_client_ip(&headers);
        assert_eq!(ip, Some("203.0.113.1".parse().unwrap()));
    }
}
