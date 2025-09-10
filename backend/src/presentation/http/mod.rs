/// HotSpot HTTP 表现层模块
/// 
/// 功能职责：
/// - 处理所有 HTTP 协议相关的请求和响应
/// - 组织控制器、路由、中间件和验证器
/// - 提供统一的 HTTP API 接口
/// - 管理 WebSocket 路由和连接
/// 
/// 模块组织：
/// - controllers: HTTP 请求控制器，处理业务逻辑调用
/// - routes: HTTP 路由定义和中间件应用
/// - auth: 统一认证系统和JWT管理
/// - middleware: HTTP 中间件（CORS、限流等）
/// - validators: 请求参数验证和数据校验
/// - response_mappers: 响应格式化和错误映射
/// - models: HTTP 请求/响应数据模型
/// - websocket_routes: WebSocket 连接路由
pub mod controllers;
pub mod routes;  
pub mod auth;
pub mod middleware;
pub mod validators;
pub mod response_mappers;
pub mod models;
pub mod websocket_routes;

// 重新导出HTTP表现层类型
pub use controllers::*;
pub use routes::*;
pub use middleware::*;
pub use validators::*;
pub use response_mappers::*;
pub use websocket_routes::*;
// 注意：models模块不全局导出，避免与controllers冲突
// 使用时通过 presentation::http::models::* 导入