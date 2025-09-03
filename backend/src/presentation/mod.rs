/// 表现层模块
/// 
/// 处理外部交互接口，包括HTTP API和WebSocket通信
/// 负责请求解析、响应格式化、协议处理等

pub mod http;
pub mod ws;

// 重新导出表现层类型
pub use http::*;
pub use ws::*;

