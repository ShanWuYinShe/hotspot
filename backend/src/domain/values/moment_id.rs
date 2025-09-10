/// 瞬间ID值对象
/// 
/// 提供类型安全的瞬间ID，防止原始类型的误用
use serde::{Deserialize, Serialize};
use std::fmt::{self, Display};

use crate::domain::DomainError;

/// 瞬间ID值对象
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct MomentId(String);

impl MomentId {
    /// 创建新的瞬间ID
    pub fn new(id: String) -> Result<Self, DomainError> {
        if id.is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "瞬间ID不能为空".to_string(),
            });
        }
        Ok(Self(id))
    }
    
    /// 生成新的瞬间ID
    pub fn generate() -> Self {
        Self(uuid::Uuid::new_v4().to_string())
    }
    
    /// 获取内部值
    pub fn value(&self) -> &str {
        &self.0
    }
    
    /// 转换为字符串
    pub fn into_string(self) -> String {
        self.0
    }
}

impl Display for MomentId {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl From<String> for MomentId {
    fn from(id: String) -> Self {
        Self(id)
    }
}

impl From<MomentId> for String {
    fn from(id: MomentId) -> Self {
        id.0
    }
}


