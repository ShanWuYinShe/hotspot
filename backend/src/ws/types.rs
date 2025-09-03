// WebSocket 消息类型、事件、协议常量等统一定义
use serde::{Deserialize, Serialize};
use crate::utils::JsSafeI64;

// 业务相关消息体可继续扩展
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChatPayload {
    pub sender_id: JsSafeI64,
    pub content: String,
    pub timestamp: JsSafeI64,
}

// ...其他 payload struct
