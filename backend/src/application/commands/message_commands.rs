use crate::domain::{DomainError, DomainResult};
/// 消息相关命令定义
use serde::{Deserialize, Serialize};

/// 发送私聊消息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendDirectMessageCommand {
    pub sender_id: i64,
    pub recipient_id: i64,
    pub content: String,
    pub message_type: String, // "text" | "image" | "system"
}

/// 删除私聊消息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteMessageCommand {
    pub message_id: i64,
    pub requester_id: i64,
}

/// 发送群组消息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendGroupMessageCommand {
    pub sender_id: i64,
    pub group_id: i64,
    pub content: String,
    pub message_type: String, // "text" | "image" | "system"
}

/// 删除群组消息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteGroupMessageCommand {
    pub message_id: i64,
    pub requester_id: i64,
}

/// 标记私聊消息已读（可选指定至某条消息）
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarkDirectMessagesReadCommand {
    pub conversation_id: i64,
    pub user_id: i64,
    pub message_id: Option<i64>,
}

/// 标记群组消息已读（可选指定至某条消息）
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarkGroupMessagesReadCommand {
    pub group_id: i64,
    pub user_id: i64,
    pub message_id: Option<i64>,
}

impl SendDirectMessageCommand {
    pub fn validate(&self) -> DomainResult<()> {
        if self.sender_id <= 0 || self.recipient_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效的用户ID".to_string(),
            });
        }
        if self.content.trim().is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "消息内容不能为空".to_string(),
            });
        }
        Ok(())
    }
}

impl DeleteMessageCommand {
    pub fn validate(&self) -> DomainResult<()> {
        if self.message_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效消息ID".to_string(),
            });
        }
        if self.requester_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效用户ID".to_string(),
            });
        }
        Ok(())
    }
}

impl SendGroupMessageCommand {
    pub fn validate(&self) -> DomainResult<()> {
        if self.sender_id <= 0 || self.group_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效的用户或群组ID".to_string(),
            });
        }
        if self.content.trim().is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "消息内容不能为空".to_string(),
            });
        }
        Ok(())
    }
}

impl DeleteGroupMessageCommand {
    pub fn validate(&self) -> DomainResult<()> {
        if self.message_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效消息ID".to_string(),
            });
        }
        if self.requester_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效用户ID".to_string(),
            });
        }
        Ok(())
    }
}

impl MarkDirectMessagesReadCommand {
    pub fn validate(&self) -> DomainResult<()> {
        if self.conversation_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效会话ID".to_string(),
            });
        }
        if self.user_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效用户ID".to_string(),
            });
        }
        if let Some(mid) = self.message_id {
            if mid <= 0 {
                return Err(DomainError::BusinessRuleViolation {
                    rule: "无效消息ID".to_string(),
                });
            }
        }
        Ok(())
    }
}

impl MarkGroupMessagesReadCommand {
    pub fn validate(&self) -> DomainResult<()> {
        if self.group_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效群组ID".to_string(),
            });
        }
        if self.user_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效用户ID".to_string(),
            });
        }
        if let Some(mid) = self.message_id {
            if mid <= 0 {
                return Err(DomainError::BusinessRuleViolation {
                    rule: "无效消息ID".to_string(),
                });
            }
        }
        Ok(())
    }
}
