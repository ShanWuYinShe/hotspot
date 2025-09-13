// 消息应用服务
// 
// 处理消息相关的业务逻辑，使用新的实体和仓库架构
// 暂时禁用，等待重新设计
/*
use crate::application::handlers::PaginatedResult;
use crate::domain::{
    DirectMessage, DirectMessageType, DomainResult, GroupMessage, 
    GroupMessageType
};
// use crate::adapters::adapters_pg::message_repository::PgMessageRepository; // 暂时禁用
*/
// 消息应用服务
// 
// 功能职责：
// - 实现消息通讯相关的业务用例和应用逻辑
// - 协调消息领域实体和仓储层的交互
// - 管理私聊和群聊消息的发送、接收、存储
// - 处理消息历史查询和分页显示
// - 管理消息状态和已读回执机制
// 
// 主要用例：
// - 私聊消息的发送和接收
// - 群组消息的广播和分发
// - 消息历史的查询和分页加载
// - 会话列表的管理和维护
// - 消息已读状态的更新和同步
// - 未读消息数量的统计和查询

// 暂时禁用，等待重新设计
// 实时通信：与WebSocket系统集成实现实时消息推送
// #[derive(Clone)]
// pub struct MessageApplicationService {
//     base: super::BaseApplicationService,
//     pg_message_repository: Arc<PgMessageRepository>,
// }

// 暂时禁用，等待重新设计
/*
impl MessageApplicationService {
    pub fn new(
        pg_message_repository: Arc<PgMessageRepository>,
        user_repository: Arc<dyn crate::ports::UserRepository>,
        event_publisher: Arc<dyn crate::ports::EventPublisher>,
    ) -> Self {
        Self { 
            base: super::BaseApplicationService::new("MessageApplicationService")
                .with_user_repository(user_repository)
                .with_event_publisher(event_publisher),
            pg_message_repository,
        }
    }
    
    /// 获取PG消息仓库的引用（用于控制器直接访问）
    pub fn get_pg_repository(&self) -> &Arc<PgMessageRepository> {
        &self.pg_message_repository
    }

    // ===== 私聊消息业务逻辑 =====
    
    /// 创建私聊消息
    pub async fn create_direct_message(
        &self,
        sender_id: i64,
        recipient_id: i64,
        content: String,
        _message_type: DirectMessageType,
        latitude: f64,
        longitude: f64,
    ) -> DomainResult<DirectMessage> {
        // 使用基类验证用户存在
        self.base.validate_users_exist(&[sender_id, recipient_id]).await?;
        
        // 验证消息内容
        super::ValidationRules::validate_text_content(&content, 1000, "消息内容")?;
        
        // 记录操作日志
        self.base.log_operation("create_direct_message", Some(sender_id), Some(&format!("to: {}", recipient_id)));
        // 获取或创建会话
        let conversation = self.pg_message_repository
            .get_or_create_conversation(sender_id, recipient_id)
            .await?;
        
        // 创建消息
        let message = DirectMessage {
            id: 0, // 将由数据库生成
            message_id: Uuid::new_v4().to_string(),
            conversation_id: conversation.conversation_id,
            sender_id,
            recipient_id,
            content,
            message_type: DirectMessageType::Text, // 暂时固定为文本
            created_at: Utc::now(),
            read_at: None,
            latitude,
            longitude,
        };
        
        self.pg_message_repository.create_direct_message(&message).await
    }
    
    /// 获取私聊消息历史
    pub async fn get_direct_messages_by_conversation(
        &self,
        conversation_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<PaginatedResult<DirectMessage>> {
        let limit = limit.clamp(1, 100);
        let messages = self.pg_message_repository
            .find_direct_messages_by_conversation(conversation_id, cursor, limit + 1)
            .await?;
        
        let has_more = messages.len() > limit as usize;
        let mut items = messages;
        if has_more {
            items.pop();
        }
        
        let next_cursor = if has_more && !items.is_empty() {
            Some(items.last().unwrap().id)
        } else {
            None
        };
        
        Ok(PaginatedResult::new(items, has_more, next_cursor))
    }

    // ===== 群组消息业务逻辑 =====
    
    /// 创建群组消息
    pub async fn create_group_message(
        &self,
        group_id: String,
        sender_id: i64,
        content: String,
        _message_type: GroupMessageType,
        latitude: f64,
        longitude: f64,
    ) -> DomainResult<GroupMessage> {
        let message = GroupMessage {
            id: 0, // 将由数据库生成
            message_id: Uuid::new_v4().to_string(),
            group_id,
            sender_id,
            content,
            message_type: GroupMessageType::Text, // 暂时固定为文本
            created_at: Utc::now(),
            latitude,
            longitude,
        };
        
        self.pg_message_repository.create_group_message(&message).await
    }
    
    /// 获取群组消息历史
    pub async fn get_group_messages_by_group(
        &self,
        group_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<PaginatedResult<GroupMessage>> {
        let limit = limit.clamp(1, 100);
        let messages = self.pg_message_repository
            .find_group_messages(group_id, cursor, limit + 1)
            .await?;
        
        let has_more = messages.len() > limit as usize;
        let mut items = messages;
        if has_more {
            items.pop();
        }
        
        let next_cursor = if has_more && !items.is_empty() {
            Some(items.last().unwrap().id)
        } else {
            None
        };
        
        Ok(PaginatedResult::new(items, has_more, next_cursor))
    }
}
*/