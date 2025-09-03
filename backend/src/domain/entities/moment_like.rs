/// 动态点赞领域实体
/// 
/// 表示用户对动态的点赞记录

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// 动态点赞实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MomentLike {
    pub id: i64,
    pub moment_id: String, // UUID as String
    pub user_id: i64,
    pub created_at: DateTime<Utc>,
}

impl MomentLike {
    /// 创建新的动态点赞记录
    pub fn new(moment_id: String, user_id: i64) -> Self {
        Self {
            id: 0, // 将由数据库生成
            moment_id,
            user_id,
            created_at: Utc::now(),
        }
    }
    
    /// 验证点赞数据的完整性
    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        
        if self.moment_id.is_empty() {
            errors.push("动态ID不能为空".to_string());
        }
        
        if self.user_id <= 0 {
            errors.push("用户ID必须为正数".to_string());
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
}

