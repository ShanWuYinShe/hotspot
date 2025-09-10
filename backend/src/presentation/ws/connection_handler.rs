/// WebSocket连接处理器
/// 
/// 处理WebSocket连接的生命周期管理
use axum::extract::ws::{WebSocket, Message};
use tokio::sync::mpsc;
use crate::presentation::ws::{WSMessageType, ConnectionState, WSError};
// use std::sync::Arc; // 未使用

/// WebSocket连接处理器
pub struct WebSocketConnectionHandler {
    user_id: Option<i64>,
    state: ConnectionState,
    sender: mpsc::UnboundedSender<Message>,
    receiver: mpsc::UnboundedReceiver<Message>,
}

impl WebSocketConnectionHandler {
    /// 创建新的连接处理器
    pub fn new() -> Self {
        let (sender, receiver) = mpsc::unbounded_channel();
        
        Self {
            user_id: None,
            state: ConnectionState::Connected,
            sender,
            receiver,
        }
    }
    
    /// 处理WebSocket连接
    pub async fn handle_connection(&mut self, mut socket: WebSocket) -> Result<(), WSError> {
        log::info!("新的WebSocket连接建立");
        
        loop {
            tokio::select! {
                // 处理来自客户端的消息
                msg = socket.recv() => {
                    match msg {
                        Some(Ok(message)) => {
                            if let Err(e) = self.handle_client_message(message).await {
                                log::error!("处理客户端消息失败: {}", e);
                                let _ = socket.send(Message::Text(
                                    serde_json::to_string(&e.to_ws_message()).unwrap_or_default().into()
                                )).await;
                            }
                        }
                        Some(Err(e)) => {
                            log::error!("WebSocket接收错误: {}", e);
                            break;
                        }
                        None => {
                            log::info!("WebSocket连接关闭");
                            break;
                        }
                    }
                }
                
                // 处理需要发送给客户端的消息
                msg = self.receiver.recv() => {
                    if let Some(message) = msg {
                        if let Err(e) = socket.send(message).await {
                            log::error!("发送消息到WebSocket失败: {}", e);
                            break;
                        }
                    }
                }
            }
        }
        
        self.state = ConnectionState::Closed;
        Ok(())
    }
    
    /// 处理来自客户端的消息
    async fn handle_client_message(&mut self, message: Message) -> Result<(), WSError> {
        match message {
            Message::Text(text) => {
                let ws_message: WSMessageType = serde_json::from_str(&text)
                    .map_err(|_| WSError::InvalidMessageFormat { message: text.to_string() })?;
                
                self.handle_ws_message(ws_message).await
            }
            Message::Binary(_) => {
                Err(WSError::InvalidMessageFormat {
                    message: "不支持二进制消息".to_string(),
                })
            }
            Message::Ping(data) => {
                // 响应Ping消息
                let pong_msg = Message::Pong(data);
                self.sender.send(pong_msg)
                    .map_err(|_| WSError::MessageSendFailed {
                        reason: "发送Pong消息失败".to_string(),
                    })?;
                Ok(())
            }
            Message::Pong(_) => {
                // 收到Pong响应
                log::debug!("收到客户端Pong响应");
                Ok(())
            }
            Message::Close(_) => {
                log::info!("客户端请求关闭连接");
                self.state = ConnectionState::Closing;
                Ok(())
            }
        }
    }
    
    /// 处理WebSocket业务消息
    async fn handle_ws_message(&mut self, message: WSMessageType) -> Result<(), WSError> {
        match message {
            WSMessageType::Auth { token } => {
                self.handle_auth(token).await
            }
            WSMessageType::Ping => {
                self.send_pong().await
            }
            WSMessageType::Subscribe { room_id } => {
                self.handle_subscribe(room_id).await
            }
            WSMessageType::Unsubscribe { room_id } => {
                self.handle_unsubscribe(room_id).await
            }
            WSMessageType::SendMessage { room_id, content } => {
                self.handle_send_message(room_id, content).await
            }
            _ => {
                Err(WSError::InvalidMessageFormat {
                    message: "不支持的消息类型".to_string(),
                })
            }
        }
    }
    
    /// 处理认证
    async fn handle_auth(&mut self, token: String) -> Result<(), WSError> {
        // 验证JWT token
        let jwt_secret = std::env::var("JWT_SECRET")
            .unwrap_or_else(|_| "your-secret-key-change-in-production".to_string());
        
        match crate::utils::jwt::verify_jwt_token(&token, &jwt_secret) {
            Ok(claims) => {
                let user_id = claims.sub;
                self.user_id = Some(user_id);
                self.state = ConnectionState::Authenticated { user_id };
                
                log::info!("用户 {} 认证成功", user_id);
                Ok(())
            }
            Err(e) => {
                log::warn!("JWT验证失败: {}", e);
                Err(WSError::AuthenticationFailed {
                    reason: "无效的token".to_string(),
                })
            }
        }
    }
    
    /// 发送Pong响应
    async fn send_pong(&self) -> Result<(), WSError> {
        let pong_message = WSMessageType::Pong;
        let message_text = serde_json::to_string(&pong_message)
            .map_err(|_| WSError::MessageSendFailed {
                reason: "序列化Pong消息失败".to_string(),
            })?;
        
        self.sender.send(Message::Text(message_text.into()))
            .map_err(|_| WSError::MessageSendFailed {
                reason: "发送Pong消息失败".to_string(),
            })?;
        
        Ok(())
    }
    
    /// 处理房间订阅
    async fn handle_subscribe(&self, room_id: String) -> Result<(), WSError> {
        if let Some(user_id) = self.user_id {
            log::info!("用户 {} 订阅房间: {}", user_id, room_id);
            // 简单实现：记录订阅操作
            // 在实际实现中，这里会调用WebSocket管理器的订阅方法
            Ok(())
        } else {
            Err(WSError::AuthenticationRequired {
                operation: "订阅房间".to_string(),
            })
        }
    }
    
    /// 处理取消订阅
    async fn handle_unsubscribe(&self, room_id: String) -> Result<(), WSError> {
        if let Some(user_id) = self.user_id {
            log::info!("用户 {} 取消订阅房间: {}", user_id, room_id);
            // 简单实现：记录取消订阅操作
            // 在实际实现中，这里会调用WebSocket管理器的取消订阅方法
            Ok(())
        } else {
            Err(WSError::AuthenticationRequired {
                operation: "取消订阅房间".to_string(),
            })
        }
    }
    
    /// 处理发送消息
    async fn handle_send_message(&self, room_id: String, content: String) -> Result<(), WSError> {
        if let Some(user_id) = self.user_id {
            log::info!("用户 {} 在房间 {} 发送消息: {}", user_id, room_id, content);
            // 简单实现：记录消息发送操作
            // 在实际实现中，这里会调用WebSocket管理器的消息发送方法
            Ok(())
        } else {
            Err(WSError::AuthenticationRequired {
                operation: "发送消息".to_string(),
            })
        }
    }
}

impl Default for WebSocketConnectionHandler {
    fn default() -> Self {
        Self::new()
    }
}

