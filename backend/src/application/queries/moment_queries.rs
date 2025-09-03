/// 瞬间相关查询定义

use serde::{Deserialize, Serialize};
use crate::domain::{DomainResult, DomainError};
use crate::application::queries::PaginationQuery;

/// 按标签搜索瞬间查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchMomentsByTagsQuery {
    pub tags: Vec<String>,
    pub requester_id: i64,
    pub pagination: PaginationQuery,
}

/// 按ID获取瞬间查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetMomentByIdQuery {
    pub moment_id: i64,
    pub requester_id: i64,
}

/// 按位置搜索瞬间查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchMomentsByLocationQuery {
    pub latitude: f64,
    pub longitude: f64,
    pub radius_km: f64,
    pub requester_id: i64,
    pub pagination: PaginationQuery,
}

/// 获取用户瞬间历史查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetUserMomentsQuery {
    pub user_id: i64,
    pub requester_id: i64,
    pub pagination: PaginationQuery,
}

impl SearchMomentsByTagsQuery {
    pub fn validate(&self) -> DomainResult<()> {
        if self.tags.is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "至少需要一个标签".to_string(),
            });
        }
        if self.tags.len() > 10 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "标签数量不能超过10个".to_string(),
            });
        }
        for tag in &self.tags {
            if tag.trim().is_empty() {
                return Err(DomainError::BusinessRuleViolation {
                    rule: "标签不能为空".to_string(),
                });
            }
            if tag.len() > 20 {
                return Err(DomainError::BusinessRuleViolation {
                    rule: "单个标签不能超过20字符".to_string(),
                });
            }
        }
        Ok(())
    }
}

impl SearchMomentsByLocationQuery {
    pub fn validate(&self) -> DomainResult<()> {
        if self.latitude < -90.0 || self.latitude > 90.0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "纬度必须在-90到90之间".to_string(),
            });
        }
        if self.longitude < -180.0 || self.longitude > 180.0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "经度必须在-180到180之间".to_string(),
            });
        }
        if self.radius_km <= 0.0 || self.radius_km > 100.0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "搜索半径必须在0到100公里之间".to_string(),
            });
        }
        Ok(())
    }
}
