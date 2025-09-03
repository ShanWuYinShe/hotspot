/// 标签领域实体
/// 
/// 表示动态中使用的标签

use serde::{Deserialize, Serialize};

/// 标签实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Tag {
    pub id: i64,
    pub tag_id: String, // UUID as String
    pub name: String,
    pub counter: i32, // 使用次数计数
}

impl Tag {
    /// 创建新的标签
    pub fn new(name: String) -> Self {
        Self {
            id: 0, // 将由数据库生成
            tag_id: uuid::Uuid::new_v4().to_string(),
            name,
            counter: 0,
        }
    }
    
    /// 增加使用计数
    pub fn increment_counter(&mut self) {
        self.counter += 1;
    }
    
    /// 减少使用计数
    pub fn decrement_counter(&mut self) {
        if self.counter > 0 {
            self.counter -= 1;
        }
    }
    
    /// 验证标签数据的完整性
    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        
        if self.name.is_empty() {
            errors.push("标签名称不能为空".to_string());
        }
        
        if self.name.len() > 50 {
            errors.push("标签名称长度不能超过50个字符".to_string());
        }
        
        // 检查标签名称是否包含非法字符
        if self.name.contains(' ') {
            errors.push("标签名称不能包含空格".to_string());
        }
        
        if self.name.contains('#') {
            errors.push("标签名称不能包含#符号".to_string());
        }
        
        if self.counter < 0 {
            errors.push("标签使用计数不能为负数".to_string());
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
}

