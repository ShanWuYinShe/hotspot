use serde::{Deserialize, Serialize};

// TODO: 完整实现所有 WebSocket 事件类型
// 参考 api_spec.mdc 第 4 章

#[derive(Debug, Serialize, Deserialize)]
#[serde(tag = "type", content = "payload")]
pub enum WsMessage {
    // ========== 服务端推送事件 (Server → Client) ==========
    
    // 连接管理事件
    ConnectionEstablished {
        connection_id: String,
        user_id: String,
        server_time: String,
    },
    Heartbeat {
        timestamp: String,
    },
    
    // 聊天消息事件
    ChatMessageNew {
        id: String,
        target_id: String,
        target_type: String,
        sender: UserInfo,
        message_type: String,
        content: String,
        sent_at: String,
    },
    
    // 在线状态事件
    UserOnline {
        user_id: String,
        nickname: String,
        online_at: String,
    },
    UserOffline {
        user_id: String,
        offline_at: String,
    },
    
    // 收件箱事件
    InboxItem {
        id: String,
        category: String,
        item_type: String,
        content: String,
        created_at: String,
    },
    
    // ========== 客户端发送事件 (Client → Server) ==========
    
    Ping {
        timestamp: String,
    },
    Pong {
        timestamp: String,
    },
    
    Subscribe {
        channels: Vec<String>,
    },
    Unsubscribe {
        channels: Vec<String>,
    },
    
    // TODO: 添加所有其他事件类型
}

#[derive(Debug, Serialize, Deserialize)]
pub struct UserInfo {
    pub user_id: String,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
}

