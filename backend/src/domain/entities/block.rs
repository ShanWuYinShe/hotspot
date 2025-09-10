/// 屏蔽系统领域实体
/// 
/// 功能职责：
/// - 定义用户屏蔽关系的核心业务实体
/// - 实现屏蔽和取消屏蔽的业务规则
/// - 管理屏蔽关系对消息和通知的影响
/// - 提供屏蔽状态的查询和验证逻辑
/// 
/// 业务规则：
/// - 屏蔽关系的单向性（A屏蔽B不等于B屏蔽A）
/// - 屏蔽对消息、通知、搜索结果的过滤影响
/// - 屏蔽关系的优先级高于好友关系
/// - 屏蔽用户的自动好友关系解除
/// - 屏蔽设置的隐私保护
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// 屏蔽类型
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum BlockType {
    /// 完全屏蔽（屏蔽所有交互）
    Complete,
    /// 仅屏蔽消息
    MessagesOnly,
    /// 仅屏蔽通知
    NotificationsOnly,
}

/// 屏蔽原因
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum BlockReason {
    /// 骚扰
    Harassment,
    /// 垃圾信息
    Spam,
    /// 不当内容
    InappropriateContent,
    /// 其他
    Other,
}

/// 屏蔽关系实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Block {
    /// 屏蔽关系ID
    pub block_id: Uuid,
    /// 屏蔽者ID
    pub blocker_id: i64,
    /// 被屏蔽者ID
    pub blocked_id: i64,
    /// 屏蔽类型
    pub block_type: BlockType,
    /// 屏蔽原因
    pub reason: Option<BlockReason>,
    /// 创建时间
    pub created_at: DateTime<Utc>,
}

/// 屏蔽设置实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockSettings {
    /// 用户ID
    pub user_id: i64,
    /// 是否屏蔽陌生人消息
    pub block_stranger_messages: bool,
    /// 是否屏蔽陌生人通知
    pub block_stranger_notifications: bool,
    /// 是否屏蔽群组邀请
    pub block_group_invitations: bool,
    /// 是否屏蔽好友请求
    pub block_friend_requests: bool,
    /// 关键词屏蔽列表
    pub blocked_keywords: Vec<String>,
    /// 更新时间
    pub updated_at: DateTime<Utc>,
}

/// 屏蔽统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockStats {
    /// 用户ID
    pub user_id: i64,
    /// 屏蔽的用户数量
    pub blocked_users_count: i64,
    /// 被屏蔽的次数
    pub blocked_by_count: i64,
    /// 最后更新时间
    pub updated_at: DateTime<Utc>,
}

impl Block {
    /// 创建新的屏蔽关系
    pub fn new(
        blocker_id: i64,
        blocked_id: i64,
        block_type: BlockType,
        reason: Option<BlockReason>,
    ) -> Result<Self, String> {
        if blocker_id == blocked_id {
            return Err("不能屏蔽自己".to_string());
        }
        
        Ok(Self {
            block_id: Uuid::new_v4(),
            blocker_id,
            blocked_id,
            block_type,
            reason,
            created_at: Utc::now(),
        })
    }
    
    /// 检查是否屏蔽消息
    pub fn blocks_messages(&self) -> bool {
        matches!(self.block_type, BlockType::Complete | BlockType::MessagesOnly)
    }
    
    /// 检查是否屏蔽通知
    pub fn blocks_notifications(&self) -> bool {
        matches!(self.block_type, BlockType::Complete | BlockType::NotificationsOnly)
    }
}

impl BlockSettings {
    /// 创建默认屏蔽设置
    pub fn new(user_id: i64) -> Self {
        Self {
            user_id,
            block_stranger_messages: false,
            block_stranger_notifications: false,
            block_group_invitations: false,
            block_friend_requests: false,
            blocked_keywords: Vec::new(),
            updated_at: Utc::now(),
        }
    }
    
    /// 添加屏蔽关键词
    pub fn add_blocked_keyword(&mut self, keyword: String) -> Result<(), String> {
        let keyword = keyword.trim().to_lowercase();
        
        if keyword.is_empty() {
            return Err("关键词不能为空".to_string());
        }
        
        if keyword.len() > 50 {
            return Err("关键词长度不能超过50个字符".to_string());
        }
        
        if self.blocked_keywords.len() >= 100 {
            return Err("屏蔽关键词不能超过100个".to_string());
        }
        
        if !self.blocked_keywords.contains(&keyword) {
            self.blocked_keywords.push(keyword);
            self.updated_at = Utc::now();
        }
        
        Ok(())
    }
    
    /// 移除屏蔽关键词
    pub fn remove_blocked_keyword(&mut self, keyword: &str) {
        let keyword = keyword.trim().to_lowercase();
        if let Some(pos) = self.blocked_keywords.iter().position(|k| k == &keyword) {
            self.blocked_keywords.remove(pos);
            self.updated_at = Utc::now();
        }
    }
    
    /// 检查内容是否包含屏蔽关键词
    pub fn contains_blocked_keyword(&self, content: &str) -> bool {
        let content = content.to_lowercase();
        self.blocked_keywords.iter().any(|keyword| content.contains(keyword))
    }
    
    /// 更新设置
    pub fn update_settings(
        &mut self,
        block_stranger_messages: Option<bool>,
        block_stranger_notifications: Option<bool>,
        block_group_invitations: Option<bool>,
        block_friend_requests: Option<bool>,
    ) {
        if let Some(value) = block_stranger_messages {
            self.block_stranger_messages = value;
        }
        if let Some(value) = block_stranger_notifications {
            self.block_stranger_notifications = value;
        }
        if let Some(value) = block_group_invitations {
            self.block_group_invitations = value;
        }
        if let Some(value) = block_friend_requests {
            self.block_friend_requests = value;
        }
        
        self.updated_at = Utc::now();
    }
}

impl BlockStats {
    /// 创建新的屏蔽统计
    pub fn new(user_id: i64) -> Self {
        Self {
            user_id,
            blocked_users_count: 0,
            blocked_by_count: 0,
            updated_at: Utc::now(),
        }
    }
    
    /// 增加屏蔽用户数量
    pub fn increment_blocked_users(&mut self) {
        self.blocked_users_count += 1;
        self.updated_at = Utc::now();
    }
    
    /// 减少屏蔽用户数量
    pub fn decrement_blocked_users(&mut self) {
        if self.blocked_users_count > 0 {
            self.blocked_users_count -= 1;
            self.updated_at = Utc::now();
        }
    }
    
    /// 增加被屏蔽次数
    pub fn increment_blocked_by(&mut self) {
        self.blocked_by_count += 1;
        self.updated_at = Utc::now();
    }
    
    /// 减少被屏蔽次数
    pub fn decrement_blocked_by(&mut self) {
        if self.blocked_by_count > 0 {
            self.blocked_by_count -= 1;
            self.updated_at = Utc::now();
        }
    }
}
