use crate::config::app_config::AppConfig;

/// API路径配置
#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct ApiPathConfig {
    /// 公开路径（不需要认证）
    pub public_paths: Vec<String>,
    /// API基础路径
    pub api_base_path: String,
    /// WebSocket路径
    pub websocket_path: String,
    /// 健康检查路径
    pub health_check_path: String,
    /// 监控指标路径
    pub metrics_path: String,
}

impl Default for ApiPathConfig {
    fn default() -> Self {
        Self {
            public_paths: vec![
                "/api/v1/user/register".to_string(),
                "/api/v1/user/login".to_string(),
                "/api/v1/user/temp".to_string(),
                "/api/v1/user/refresh_token".to_string(),
                "/ws/".to_string(),
                "/health".to_string(),
                "/metrics".to_string(),
            ],
            api_base_path: "/api".to_string(),
            websocket_path: "/ws".to_string(),
            health_check_path: "/health".to_string(),
            metrics_path: "/metrics".to_string(),
        }
    }
}

/// CORS配置
#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct CorsConfig {
    /// 是否为开发模式CORS（完全开放）
    pub permissive_mode: bool,
    /// 允许的HTTP方法
    pub allowed_methods: Vec<String>,
    /// 允许的域名
    pub allowed_origins: Vec<String>,
    /// 允许的请求头
    pub allowed_headers: Vec<String>,
}

impl Default for CorsConfig {
    fn default() -> Self {
        Self {
            permissive_mode: cfg!(debug_assertions),
            allowed_methods: vec![
                "POST".to_string(),
                "OPTIONS".to_string(), // 预检请求必须保留
            ],
            allowed_origins: vec!["*".to_string()],
            allowed_headers: vec!["*".to_string()],
        }
    }
}

#[derive(Debug, Clone)]
pub struct ConfigService {
    pub config: AppConfig,
    pub api_paths: ApiPathConfig,
    pub cors: CorsConfig,
}

impl ConfigService {
    pub fn new(config: AppConfig) -> Self {
        Self { 
            config,
            api_paths: ApiPathConfig::default(),
            cors: CorsConfig::default(),
        }
    }
    
    #[allow(dead_code)]
    pub fn with_api_paths(mut self, api_paths: ApiPathConfig) -> Self {
        self.api_paths = api_paths;
        self
    }
    
    #[allow(dead_code)]
    pub fn with_cors_config(mut self, cors: CorsConfig) -> Self {
        self.cors = cors;
        self
    }

    pub fn get_config(&self) -> AppConfig {
        self.config.clone()
    }
    
    /// 获取公开路径列表
    #[allow(dead_code)]
    pub fn get_public_paths(&self) -> &Vec<String> {
        &self.api_paths.public_paths
    }
    
    /// 检查路径是否为公开路径
    pub fn is_public_path(&self, path: &str) -> bool {
        self.api_paths.public_paths
            .iter()
            .any(|prefix| path.starts_with(prefix))
    }
    
    /// 获取API基础路径
    #[allow(dead_code)]
    pub fn get_api_base_path(&self) -> &str {
        &self.api_paths.api_base_path
    }
    
    /// 获取WebSocket路径
    #[allow(dead_code)]
    pub fn get_websocket_path(&self) -> &str {
        &self.api_paths.websocket_path
    }
}
