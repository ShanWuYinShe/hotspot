use crate::domain::{DomainError, DomainResult};
/// 群组相关命令定义
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateGroupCommand {
    pub name: String,
    pub description: Option<String>,
    pub owner_id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub password: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateGroupInfoCommand {
    pub group_id: String,
    pub name: Option<String>,
    pub description: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteGroupCommand {
    pub group_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupCommandResult {
    pub group_id: String,
    pub success: bool,
    pub message: Option<String>,
}

impl GroupCommandResult {
    pub fn success(group_id: String) -> Self {
        Self {
            group_id,
            success: true,
            message: None,
        }
    }
    pub fn success_with_message(group_id: String, message: String) -> Self {
        Self {
            group_id,
            success: true,
            message: Some(message),
        }
    }
    pub fn failure(group_id: String, message: String) -> Self {
        Self {
            group_id,
            success: false,
            message: Some(message),
        }
    }
}

impl CreateGroupCommand {
    pub fn validate(&self) -> DomainResult<()> {
        if self.name.is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "群组名称不能为空".to_string(),
            });
        }
        if self.name.len() > 100 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "群组名称长度不能超过100".to_string(),
            });
        }
        if let Some(desc) = &self.description
            && desc.len() > 500 {
                return Err(DomainError::BusinessRuleViolation {
                    rule: "群组描述长度不能超过500".to_string(),
                });
            }
        if self.owner_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效的群主ID".to_string(),
            });
        }
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
        if self.location_name.trim().is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "位置名称不能为空".to_string(),
            });
        }
        Ok(())
    }
}

impl UpdateGroupInfoCommand {
    pub fn validate(&self) -> DomainResult<()> {
        if self.group_id.is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效群组ID".to_string(),
            });
        }
        if let Some(name) = &self.name
            && name.is_empty() {
                return Err(DomainError::BusinessRuleViolation {
                    rule: "群组名称不能为空".to_string(),
                });
            }
        Ok(())
    }
}

impl DeleteGroupCommand {
    pub fn validate(&self) -> DomainResult<()> {
        if self.group_id.is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效群组ID".to_string(),
            });
        }
        Ok(())
    }
}

#[allow(dead_code)]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JoinGroupCommand {
    pub group_id: String,
    pub user_id: i64,
    pub password: Option<String>,
}

#[allow(dead_code)]
impl JoinGroupCommand {
    pub fn validate(&self) -> DomainResult<()> {
        if self.group_id.is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效群组ID".to_string(),
            });
        }
        if self.user_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效用户ID".to_string(),
            });
        }
        if let Some(pwd) = &self.password {
            if pwd.is_empty() {
                return Err(DomainError::BusinessRuleViolation {
                    rule: "群组密码不能为空".to_string(),
                });
            }
        }
        Ok(())
    }
}

#[allow(dead_code)]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LeaveGroupCommand {
    pub group_id: String,
    pub user_id: i64,
}

#[allow(dead_code)]
impl LeaveGroupCommand {
    pub fn validate(&self) -> DomainResult<()> {
        if self.group_id.is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效群组ID".to_string(),
            });
        }
        if self.user_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效用户ID".to_string(),
            });
        }
        Ok(())
    }
}

#[allow(dead_code)]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KickMemberCommand {
    pub group_id: String,
    pub target_user_id: i64,
    pub admin_id: i64,
}

#[allow(dead_code)]
impl KickMemberCommand {
    pub fn validate(&self) -> DomainResult<()> {
        if self.group_id.is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效群组ID".to_string(),
            });
        }
        if self.target_user_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效目标用户ID".to_string(),
            });
        }
        if self.admin_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效管理员用户ID".to_string(),
            });
        }
        if self.target_user_id == self.admin_id {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无法踢出自己，请使用离开群组".to_string(),
            });
        }
        Ok(())
    }
}

#[allow(dead_code)]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransferOwnershipCommand {
    pub group_id: String,
    pub new_owner_id: i64,
    pub current_owner_id: i64,
}

#[allow(dead_code)]
impl TransferOwnershipCommand {
    pub fn validate(&self) -> DomainResult<()> {
        if self.group_id.is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效群组ID".to_string(),
            });
        }
        if self.new_owner_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效新所有者用户ID".to_string(),
            });
        }
        if self.current_owner_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "无效当前所有者用户ID".to_string(),
            });
        }
        if self.new_owner_id == self.current_owner_id {
            return Err(DomainError::BusinessRuleViolation {
                rule: "新旧所有者不能相同".to_string(),
            });
        }
        Ok(())
    }
}

#[allow(dead_code)]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JoinGroupResult {
    pub group_id: i64,
    pub user_id: i64,
    pub role: String,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LeaveGroupResult {
    pub group_id: i64,
    pub user_id: i64,
    pub success: bool,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KickMemberResult {
    pub group_id: i64,
    pub user_id: i64,
    pub success: bool,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransferOwnershipResult {
    pub group_id: i64,
    pub new_owner_id: i64,
    pub success: bool,
}

// 注意：KickMemberCommand、TransferOwnershipCommand、KickMemberResult 已在文件前面定义
