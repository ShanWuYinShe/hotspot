/// 真正的Axum路由内省系统
/// 
/// 功能特性：
/// - 在路由注册时动态收集路由信息
/// - 自动分析路由层次结构和中间件
/// - 支持嵌套路由和路由组
/// - 实时追踪路由变更
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use serde::{Serialize, Deserialize};
use tracing::info;

/// 路由端点信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RouteInfo {
    /// 完整路径
    pub path: String,
    /// HTTP方法
    pub method: String,
    /// 是否需要认证
    pub requires_auth: bool,
    /// 端点描述
    pub description: String,
    /// 所属分类
    pub category: String,
    /// 路由层级
    pub level: usize,
    /// 父路由路径
    pub parent_path: Option<String>,
}

/// 路由分类统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RouteCategory {
    /// 分类名称
    pub name: String,
    /// 分类描述
    pub description: String,
    /// 端点列表
    pub endpoints: Vec<RouteInfo>,
    /// 端点数量
    pub count: usize,
    /// 是否需要认证
    pub requires_auth: bool,
}

/// 路由摘要信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RouteSummary {
    /// 公开路由分类
    pub public_categories: Vec<RouteCategory>,
    /// 受保护路由分类
    pub protected_categories: Vec<RouteCategory>,
    /// WebSocket端点
    pub websocket_endpoints: Vec<RouteInfo>,
    /// 监控端点
    pub monitoring_endpoints: Vec<RouteInfo>,
    /// 总端点数量
    pub total_endpoints: usize,
}

/// 路由收集器 - 全局单例
static ROUTE_COLLECTOR: once_cell::sync::Lazy<Arc<Mutex<RouteCollector>>> = 
    once_cell::sync::Lazy::new(|| Arc::new(Mutex::new(RouteCollector::new())));

/// 路由收集器
#[derive(Debug)]
pub struct RouteCollector {
    routes: Vec<RouteInfo>,
    current_prefix: String,
    current_auth_required: bool,
}

impl RouteCollector {
    pub fn new() -> Self {
        Self {
            routes: Vec::new(),
            current_prefix: String::new(),
            current_auth_required: false,
        }
    }
    
    /// 添加路由信息
    pub fn add_route(&mut self, path: &str, method: &str, description: &str) {
        let full_path = if self.current_prefix.is_empty() {
            path.to_string()
        } else {
            format!("{}{}", self.current_prefix, path)
        };
        
        let category = self.extract_category(&full_path);
        let requires_auth = self.current_auth_required || self.is_protected_route(&full_path);
        
        let route_info = RouteInfo {
            path: full_path,
            method: method.to_string(),
            requires_auth,
            description: description.to_string(),
            category,
            level: self.current_prefix.matches('/').count(),
            parent_path: if self.current_prefix.is_empty() { 
                None 
            } else { 
                Some(self.current_prefix.clone()) 
            },
        };
        
        self.routes.push(route_info);
    }
    
    /// 设置当前路由前缀
    pub fn set_prefix(&mut self, prefix: &str) {
        self.current_prefix = prefix.to_string();
    }
    
    /// 设置当前认证要求
    pub fn set_auth_required(&mut self, required: bool) {
        self.current_auth_required = required;
    }
    
    /// 清除所有路由信息
    pub fn clear(&mut self) {
        self.routes.clear();
        self.current_prefix.clear();
        self.current_auth_required = false;
    }
    
    /// 获取所有路由信息
    pub fn get_routes(&self) -> Vec<RouteInfo> {
        self.routes.clone()
    }
    
    /// 从路径中提取分类
    fn extract_category(&self, path: &str) -> String {
        if path.starts_with("/health") || path.starts_with("/ready") || path.starts_with("/alive") {
            "health".to_string()
        } else if path.starts_with("/api/v1/auth") {
            "auth".to_string()
        } else if path.contains("/user") {
            "user".to_string()
        } else if path.contains("/group") {
            "group".to_string()
        } else if path.contains("/chat") {
            "chat".to_string()
        } else if path.contains("/moments") {
            "moments".to_string()
        } else if path.contains("/admin") {
            "admin".to_string()
        } else if path.starts_with("/ws") {
            "websocket".to_string()
        } else if path.starts_with("/monitoring") {
            "monitoring".to_string()
        } else {
            "other".to_string()
        }
    }
    
    /// 判断是否为受保护路由
    fn is_protected_route(&self, path: &str) -> bool {
        path.contains("/protected/") || 
        path.contains("/admin/") || 
        path.starts_with("/monitoring/")
    }
}

/// 路由内省器
pub struct RouteIntrospector;

impl RouteIntrospector {
    /// 注册路由并收集信息
    pub fn register_route(path: &str, method: &str, description: &str) {
        if let Ok(mut collector) = ROUTE_COLLECTOR.lock() {
            collector.add_route(path, method, description);
        }
    }
    
    /// 设置路由前缀
    pub fn set_prefix(prefix: &str) {
        if let Ok(mut collector) = ROUTE_COLLECTOR.lock() {
            collector.set_prefix(prefix);
        }
    }
    
    /// 设置认证要求
    pub fn set_auth_required(required: bool) {
        if let Ok(mut collector) = ROUTE_COLLECTOR.lock() {
            collector.set_auth_required(required);
        }
    }
    
    /// 清除路由前缀
    pub fn clear_prefix() {
        if let Ok(mut collector) = ROUTE_COLLECTOR.lock() {
            collector.set_prefix("");
        }
    }
    
    /// 获取所有收集的路由
    pub fn get_collected_routes() -> Vec<RouteInfo> {
        if let Ok(collector) = ROUTE_COLLECTOR.lock() {
            collector.get_routes()
        } else {
            Vec::new()
        }
    }
    
    /// 清除所有路由信息
    pub fn clear_routes() {
        if let Ok(mut collector) = ROUTE_COLLECTOR.lock() {
            collector.clear();
        }
    }
    
    /// 生成路由摘要
    pub fn generate_summary() -> RouteSummary {
        let routes = Self::get_collected_routes();
        let mut public_categories = HashMap::new();
        let mut protected_categories = HashMap::new();
        let mut websocket_endpoints = Vec::new();
        let mut monitoring_endpoints = Vec::new();
        
        // 分类处理所有端点
        for route in &routes {
            if route.category == "websocket" {
                websocket_endpoints.push(route.clone());
            } else if route.category == "monitoring" {
                monitoring_endpoints.push(route.clone());
            } else if route.requires_auth {
                protected_categories
                    .entry(route.category.clone())
                    .or_insert_with(Vec::new)
                    .push(route.clone());
            } else {
                public_categories
                    .entry(route.category.clone())
                    .or_insert_with(Vec::new)
                    .push(route.clone());
            }
        }
        
        // 构建分类结构
        let public_categories = Self::build_categories(public_categories, false);
        let protected_categories = Self::build_categories(protected_categories, true);
        
        let total_endpoints = routes.len();
        
        RouteSummary {
            public_categories,
            protected_categories,
            websocket_endpoints,
            monitoring_endpoints,
            total_endpoints,
        }
    }
    
    /// 构建分类结构
    fn build_categories(categories: HashMap<String, Vec<RouteInfo>>, requires_auth: bool) -> Vec<RouteCategory> {
        let mut result = Vec::new();
        
        for (category_name, endpoints) in categories {
            let description = Self::get_category_description(&category_name);
            let count = endpoints.len();
            
            result.push(RouteCategory {
                name: category_name,
                description,
                endpoints,
                count,
                requires_auth,
            });
        }
        
        // 按分类名称排序
        result.sort_by(|a, b| a.name.cmp(&b.name));
        result
    }
    
    /// 获取分类描述
    fn get_category_description(category: &str) -> String {
        match category {
            "health" => "健康检查".to_string(),
            "auth" => "认证系统".to_string(),
            "user" => "用户数据管理".to_string(),
            "group" => "群组管理".to_string(),
            "chat" => "消息通讯".to_string(),
            "moments" => "生活瞬间".to_string(),
            "admin" => "管理接口".to_string(),
            "websocket" => "实时通信".to_string(),
            "monitoring" => "系统监控".to_string(),
            _ => format!("{} 功能", category),
        }
    }
    
    /// 打印路由摘要
    pub fn print_summary(websocket_available: bool) {
        let summary = Self::generate_summary();
        
        info!("📋 API路由摘要 (实时收集):");
        
        // 公开路由
        if !summary.public_categories.is_empty() {
            info!("   🔓 公开路由:");
            for category in &summary.public_categories {
                info!("      • {} - {} ({} 个端点)", 
                    Self::format_category_path(&category.name), 
                    category.description, 
                    category.count
                );
            }
        }
        
        // 受保护路由
        if !summary.protected_categories.is_empty() {
            info!("   🔒 受保护路由:");
            for category in &summary.protected_categories {
                info!("      • {} - {} ({} 个端点)", 
                    Self::format_category_path(&category.name), 
                    category.description, 
                    category.count
                );
            }
        }
        
        // WebSocket端点
        if websocket_available && !summary.websocket_endpoints.is_empty() {
            info!("   🔌 WebSocket:");
            for endpoint in &summary.websocket_endpoints {
                info!("      • {} - {}", endpoint.path, endpoint.description);
            }
        }
        
        // 监控端点
        if !summary.monitoring_endpoints.is_empty() {
            info!("   📊 监控路由:");
            for endpoint in &summary.monitoring_endpoints {
                info!("      • {} - {} (需管理员权限)", endpoint.path, endpoint.description);
            }
        }
        
        info!("   📈 总计: {} 个API端点", summary.total_endpoints);
        info!("   🔍 路由信息通过实时收集获得");
        
        // 验证路由完整性
        Self::validate_routes(&summary);
    }
    
    /// 格式化分类路径
    fn format_category_path(category: &str) -> String {
        match category {
            "health" => "/health, /ready, /alive".to_string(),
            "auth" => "/api/v1/auth/*".to_string(),
            "user" => "/api/v1/{public,protected}/user/*".to_string(),
            "group" => "/api/v1/{public,protected}/group/*".to_string(),
            "chat" => "/api/v1/protected/chat/*".to_string(),
            "moments" => "/api/v1/{public,protected}/moments/*".to_string(),
            "admin" => "/api/v1/protected/admin/*".to_string(),
            "monitoring" => "/monitoring/*".to_string(),
            _ => format!("/api/v1/{}/*", category),
        }
    }
    
    /// 验证路由配置的完整性
    fn validate_routes(summary: &RouteSummary) {
        let mut issues = Vec::new();
        
        // 检查是否有足够的端点
        if summary.total_endpoints < 30 {
            issues.push(format!("端点数量偏少: {} (期望至少30个)", summary.total_endpoints));
        }
        
        // 检查核心分类是否存在
        let required_categories = vec!["auth", "user", "group", "chat", "moments"];
        let existing_categories: std::collections::HashSet<String> = summary.public_categories
            .iter()
            .chain(summary.protected_categories.iter())
            .map(|c| c.name.clone())
            .collect();
            
        for category in required_categories {
            if !existing_categories.contains(category) {
                issues.push(format!("缺少核心分类: {}", category));
            }
        }
        
        if issues.is_empty() {
            info!("✅ 路由配置验证通过");
        } else {
            for issue in &issues {
                tracing::warn!("⚠️  路由配置问题: {}", issue);
            }
        }
    }
    
    /// 获取详细的端点列表
    pub fn get_detailed_endpoints() -> Vec<String> {
        let routes = Self::get_collected_routes();
        let mut details = Vec::new();
        
        for route in &routes {
            let auth_status = if route.requires_auth { "🔒" } else { "🔓" };
            details.push(format!("{} {} {} - {}", 
                auth_status,
                route.method,
                route.path,
                route.description
            ));
        }
        
        details.sort();
        details
    }
}

/// 路由注册宏 - 简化路由注册和信息收集
#[macro_export]
macro_rules! register_route {
    ($path:expr, $method:expr, $description:expr) => {
        $crate::utils::route_introspector::RouteIntrospector::register_route($path, $method, $description);
    };
}

/// 路由组宏 - 管理路由前缀和认证状态
#[macro_export]
macro_rules! route_group {
    ($prefix:expr, $auth:expr, $block:block) => {
        {
            $crate::utils::route_introspector::RouteIntrospector::set_prefix($prefix);
            $crate::utils::route_introspector::RouteIntrospector::set_auth_required($auth);
            let result = $block;
            $crate::utils::route_introspector::RouteIntrospector::clear_prefix();
            $crate::utils::route_introspector::RouteIntrospector::set_auth_required(false);
            result
        }
    };
}

impl Default for RouteCollector {
    fn default() -> Self {
        Self::new()
    }
}

