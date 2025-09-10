/// 瞬间相关命令定义
use serde::{Deserialize, Serialize};
use crate::domain::{DomainResult, DomainError};

/// 创建瞬间命令
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateMomentCommand {
    pub user_id: i64,
    pub content: String,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub address: Option<String>,
    pub tags: Option<Vec<String>>,
    pub image_urls: Option<Vec<String>>,
}

/// 删除瞬间命令
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteMomentCommand {
    pub moment_id: i64,
    pub requester_id: i64,
}

/// 点赞瞬间命令
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LikeMomentCommand {
    pub moment_id: i64,
    pub user_id: i64,
}

/// 取消点赞瞬间命令
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnlikeMomentCommand {
    pub moment_id: i64,
    pub user_id: i64,
}

impl CreateMomentCommand {
    pub fn validate(&self) -> DomainResult<()> {
        if self.content.trim().is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "瞬间内容不能为空".to_string(),
            });
        }
        if self.content.len() > 500 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "瞬间内容不能超过500字符".to_string(),
            });
        }
        if let Some(tags) = &self.tags {
            if tags.len() > 10 {
                return Err(DomainError::BusinessRuleViolation {
                    rule: "标签数量不能超过10个".to_string(),
                });
            }
            for tag in tags {
                if tag.len() > 20 {
                    return Err(DomainError::BusinessRuleViolation {
                        rule: "单个标签不能超过20字符".to_string(),
                    });
                }
            }
        }
        if let Some(image_urls) = &self.image_urls {
            if image_urls.len() > 9 {
                return Err(DomainError::BusinessRuleViolation {
                    rule: "图片数量不能超过9张".to_string(),
                });
            }
        }
        Ok(())
    }
}
