/// 会话领域实体
/// 
/// 表示两个用户之间的私聊会话

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// 会话实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Conversation {
    pub id: i64,
    pub conversation_id: String, // UUID as String
    pub user1_id: i64,
    pub user2_id: i64,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub last_message_at: DateTime<Utc>,
    pub last_message_preview: Option<String>,
}

impl Conversation {
    /// 创建新的会话
    /// 确保user1_id < user2_id以满足数据库约束
    pub fn new(user1_id: i64, user2_id: i64) -> Self {
        let (user1, user2) = if user1_id < user2_id {
            (user1_id, user2_id)
        } else {
            (user2_id, user1_id)
        };
        
        let now = Utc::now();
        Self {
            id: 0, // 将由数据库生成
            conversation_id: uuid::Uuid::new_v4().to_string(),
            user1_id: user1,
            user2_id: user2,
            created_at: now,
            updated_at: now,
            last_message_at: now,
            last_message_preview: None,
        }
    }
    
    /// 更新最后消息信息
    pub fn update_last_message(&mut self, preview: String) {
        self.last_message_preview = Some(preview);
        self.last_message_at = Utc::now();
        self.updated_at = Utc::now();
    }
    
    /// 检查用户是否参与此会话
    pub fn involves_user(&self, user_id: i64) -> bool {
        self.user1_id == user_id || self.user2_id == user_id
    }
    
    /// 获取对方用户ID
    pub fn get_other_user_id(&self, user_id: i64) -> Option<i64> {
        if self.user1_id == user_id {
            Some(self.user2_id)
        } else if self.user2_id == user_id {
            Some(self.user1_id)
        } else {
            None
        }
    }
    
    /// 验证会话数据的完整性
    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        
        if self.user1_id <= 0 {
            errors.push("用户1 ID必须为正数".to_string());
        }
        
        if self.user2_id <= 0 {
            errors.push("用户2 ID必须为正数".to_string());
        }
        
        if self.user1_id == self.user2_id {
            errors.push("不能与自己创建会话".to_string());
        }
        
        if self.user1_id >= self.user2_id {
            errors.push("用户ID顺序错误，user1_id必须小于user2_id".to_string());
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
}

