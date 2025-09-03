/// 动态标签关联领域实体
/// 
/// 表示动态和标签之间的多对多关系

use serde::{Deserialize, Serialize};

/// 动态标签关联实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MomentTag {
    pub id: i64,
    pub moment_id: String, // UUID as String
    pub tag_id: String, // UUID as String
}

impl MomentTag {
    /// 创建新的动态标签关联
    pub fn new(moment_id: String, tag_id: String) -> Self {
        Self {
            id: 0, // 将由数据库生成
            moment_id,
            tag_id,
        }
    }
    
    /// 验证关联数据的完整性
    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        
        if self.moment_id.is_empty() {
            errors.push("动态ID不能为空".to_string());
        }
        
        if self.tag_id.is_empty() {
            errors.push("标签ID不能为空".to_string());
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
}

