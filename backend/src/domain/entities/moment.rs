/// 生活瞬间领域实体
/// 
/// 功能职责：
/// - 定义生活瞬间核心业务实体和属性
/// - 实现瞬间发布、编辑、删除的业务规则
/// - 管理瞬间的地理位置和标签系统
/// - 提供瞬间互动（点赞、评论）的业务逻辑
/// 
/// 业务规则：
/// - 瞬间必须包含地理位置信息
/// - 瞬间内容和图片的格式验证
/// - 瞬间标签的数量和格式限制
/// - 瞬间隐私和可见性控制
/// - 瞬间点赞和评论的权限管理
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// 瞬间可见性枚举
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum MomentVisibility {
    /// 公开可见
    Public,
    /// 仅好友可见
    Friends,
    /// 仅自己可见
    Private,
}

impl MomentVisibility {
    pub fn as_str(&self) -> &'static str {
        match self {
            MomentVisibility::Public => "public",
            MomentVisibility::Friends => "friends",
            MomentVisibility::Private => "private",
        }
    }
}

/// 瞬间状态枚举
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum MomentStatus {
    /// 正常
    Active,
    /// 已删除
    Deleted,
}

impl MomentStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            MomentStatus::Active => "active",
            MomentStatus::Deleted => "deleted",
        }
    }
}

/// 生活瞬间实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Moment {
    pub id: i64,
    pub moment_id: String, // UUID as String
    pub user_id: i64,
    pub content: String,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub created_at: DateTime<Utc>,
}

impl Moment {
    /// 创建新的瞬间
    pub fn new(
        user_id: i64,
        content: Option<String>,
        latitude: f64,
        longitude: f64,
        location_name: String,
    ) -> Self {
        Self {
            id: 0, // 将由数据库生成
            moment_id: uuid::Uuid::new_v4().to_string(),
            user_id,
            content,
            latitude,
            longitude,
            location_name,
            created_at: Utc::now(),
        }
    }
    
    /// 检查是否有内容
    pub fn has_content(&self) -> bool {
        self.content.is_some() && !self.content.as_ref().unwrap().is_empty()
    }
    
    /// 更新内容
    pub fn update_content(&mut self, content: Option<String>) {
        self.content = content;
    }
    
    /// 验证瞬间数据的完整性
    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        
        if self.user_id <= 0 {
            errors.push("用户ID必须为正数".to_string());
        }
        
        if let Some(content) = &self.content {
            if content.len() > 1000 {
                errors.push("瞬间内容长度不能超过1000个字符".to_string());
            }
        }
        
        if self.latitude < -90.0 || self.latitude > 90.0 {
            errors.push("纬度必须在-90到90之间".to_string());
        }
        
        if self.longitude < -180.0 || self.longitude > 180.0 {
            errors.push("经度必须在-180到180之间".to_string());
        }
        
        if self.location_name.is_empty() {
            errors.push("位置名称不能为空".to_string());
        }
        
        if self.location_name.len() > 100 {
            errors.push("位置名称长度不能超过100个字符".to_string());
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
}

