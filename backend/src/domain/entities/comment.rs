/// 评论领域实体
/// 
/// 表示用户对生活瞬间的评论
/// 评论领域实体
/// 
/// 功能职责：
/// - 定义评论核心业务实体和属性
/// - 实现评论发布、编辑、删除的业务规则
/// - 管理评论的层级结构和回复关系
/// - 提供评论内容的验证和过滤逻辑
/// 
/// 业务规则：
/// - 评论内容长度和格式限制
/// - 评论回复的层级深度控制
/// - 评论权限和可见性管理
/// - 评论举报和审核机制
/// - 评论删除的级联处理
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// 评论实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Comment {
    /// 评论ID (UUID)
    pub comment_id: Uuid,
    /// 生活瞬间ID
    pub moment_id: Uuid,
    /// 评论用户ID
    pub user_id: i64,
    /// 评论内容
    pub content: String,
    /// 创建时间
    pub created_at: DateTime<Utc>,
    /// 更新时间
    pub updated_at: DateTime<Utc>,
}

/// 评论详情（包含用户信息）
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CommentDetail {
    /// 评论ID
    pub comment_id: Uuid,
    /// 生活瞬间ID
    pub moment_id: Uuid,
    /// 评论用户ID
    pub user_id: i64,
    /// 评论用户昵称
    pub user_nickname: String,
    /// 评论用户头像
    pub user_avatar: Option<String>,
    /// 评论内容
    pub content: String,
    /// 创建时间
    pub created_at: DateTime<Utc>,
    /// 更新时间
    pub updated_at: DateTime<Utc>,
}

impl Comment {
    /// 创建新评论
    pub fn new(moment_id: Uuid, user_id: i64, content: String) -> Self {
        let now = Utc::now();
        Self {
            comment_id: Uuid::new_v4(),
            moment_id,
            user_id,
            content,
            created_at: now,
            updated_at: now,
        }
    }
    
    /// 验证评论内容
    pub fn validate_content(content: &str) -> Result<(), String> {
        let content = content.trim();
        
        if content.is_empty() {
            return Err("评论内容不能为空".to_string());
        }
        
        if content.len() > 1000 {
            return Err("评论内容不能超过1000个字符".to_string());
        }
        
        Ok(())
    }
    
    /// 更新评论内容
    pub fn update_content(&mut self, new_content: String) -> Result<(), String> {
        Self::validate_content(&new_content)?;
        self.content = new_content.trim().to_string();
        self.updated_at = Utc::now();
        Ok(())
    }
}
