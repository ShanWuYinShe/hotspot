/// HTTP表现层模块
/// 
/// 处理HTTP协议相关的请求响应，包括控制器、路由、中间件等

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

