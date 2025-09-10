/// 群组事件记录领域实体
/// 
/// 记录群组内的各种事件（创建、加入、离开、更新、踢出、转让）
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value as JsonValue;

/// 群组事件记录类型枚举
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum GroupEventRecordType {
    /// 创建群组
    Create,
    /// 加入群组
    Join,
    /// 离开群组
    Leave,
    /// 更新群组信息
    Update,
    /// 踢出成员
    Kick,
    /// 转让群主
    Transfer,
}

impl GroupEventRecordType {
    pub fn as_str(&self) -> &'static str {
        match self {
            GroupEventRecordType::Create => "create",
            GroupEventRecordType::Join => "join",
            GroupEventRecordType::Leave => "leave",
            GroupEventRecordType::Update => "update",
            GroupEventRecordType::Kick => "kick",
            GroupEventRecordType::Transfer => "transfer",
        }
    }
    
    pub fn from_str(s: &str) -> Option<Self> {
        match s {
            "create" => Some(GroupEventRecordType::Create),
            "join" => Some(GroupEventRecordType::Join),
            "leave" => Some(GroupEventRecordType::Leave),
            "update" => Some(GroupEventRecordType::Update),
            "kick" => Some(GroupEventRecordType::Kick),
            "transfer" => Some(GroupEventRecordType::Transfer),
            _ => None,
        }
    }
}

/// 群组事件记录实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupEventRecord {
    pub id: i64,
    pub group_id: String, // UUID as String
    pub user_id: i64, // 执行事件的用户ID
    pub target_user_id: Option<i64>, // 事件目标用户ID（如被踢出的用户）
    pub event_type: GroupEventRecordType,
    pub event_data: Option<JsonValue>, // 事件相关的额外数据
    pub created_at: DateTime<Utc>,
}

impl GroupEventRecord {
    /// 创建群组创建事件
    pub fn new_create(group_id: String, user_id: i64) -> Self {
        Self {
            id: 0, // 将由数据库生成
            group_id,
            user_id,
            target_user_id: None,
            event_type: GroupEventRecordType::Create,
            event_data: None,
            created_at: Utc::now(),
        }
    }
    
    /// 创建用户加入事件
    pub fn new_join(group_id: String, user_id: i64) -> Self {
        Self {
            id: 0, // 将由数据库生成
            group_id,
            user_id,
            target_user_id: None,
            event_type: GroupEventRecordType::Join,
            event_data: None,
            created_at: Utc::now(),
        }
    }
    
    /// 创建用户离开事件
    pub fn new_leave(group_id: String, user_id: i64) -> Self {
        Self {
            id: 0, // 将由数据库生成
            group_id,
            user_id,
            target_user_id: None,
            event_type: GroupEventRecordType::Leave,
            event_data: None,
            created_at: Utc::now(),
        }
    }
    
    /// 创建群组更新事件
    pub fn new_update(group_id: String, user_id: i64, update_data: JsonValue) -> Self {
        Self {
            id: 0, // 将由数据库生成
            group_id,
            user_id,
            target_user_id: None,
            event_type: GroupEventRecordType::Update,
            event_data: Some(update_data),
            created_at: Utc::now(),
        }
    }
    
    /// 创建踢出成员事件
    pub fn new_kick(group_id: String, user_id: i64, target_user_id: i64) -> Self {
        Self {
            id: 0, // 将由数据库生成
            group_id,
            user_id,
            target_user_id: Some(target_user_id),
            event_type: GroupEventRecordType::Kick,
            event_data: None,
            created_at: Utc::now(),
        }
    }
    
    /// 创建转让群主事件
    pub fn new_transfer(group_id: String, user_id: i64, target_user_id: i64) -> Self {
        Self {
            id: 0, // 将由数据库生成
            group_id,
            user_id,
            target_user_id: Some(target_user_id),
            event_type: GroupEventRecordType::Transfer,
            event_data: None,
            created_at: Utc::now(),
        }
    }
    
    /// 检查事件是否涉及特定用户
    pub fn involves_user(&self, user_id: i64) -> bool {
        self.user_id == user_id || self.target_user_id == Some(user_id)
    }
    
    /// 获取事件描述（用于通知或日志）
    pub fn get_description(&self) -> String {
        match self.event_type {
            GroupEventRecordType::Create => "创建了群组".to_string(),
            GroupEventRecordType::Join => "加入了群组".to_string(),
            GroupEventRecordType::Leave => "离开了群组".to_string(),
            GroupEventRecordType::Update => "更新了群组信息".to_string(),
            GroupEventRecordType::Kick => {
                if let Some(target_id) = self.target_user_id {
                    format!("将用户 {} 踢出了群组", target_id)
                } else {
                    "踢出了成员".to_string()
                }
            }
            GroupEventRecordType::Transfer => {
                if let Some(target_id) = self.target_user_id {
                    format!("将群主转让给了用户 {}", target_id)
                } else {
                    "转让了群主".to_string()
                }
            }
        }
    }
    
    /// 验证事件数据的完整性
    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        
        if self.group_id.is_empty() {
            errors.push("群组ID不能为空".to_string());
        }
        
        if self.user_id <= 0 {
            errors.push("用户ID必须为正数".to_string());
        }
        
        // 检查需要目标用户ID的事件类型
        match self.event_type {
            GroupEventRecordType::Kick | GroupEventRecordType::Transfer => {
                if self.target_user_id.is_none() {
                    errors.push(format!("{:?} 事件必须指定目标用户ID", self.event_type));
                } else if let Some(target_id) = self.target_user_id {
                    if target_id <= 0 {
                        errors.push("目标用户ID必须为正数".to_string());
                    }
                    if target_id == self.user_id {
                        errors.push("目标用户不能是执行操作的用户本身".to_string());
                    }
                }
            }
            _ => {
                // 其他事件类型通常不需要目标用户ID
            }
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
}
