/// 全局搜索查询
/// 
/// 提供统一的搜索接口，支持搜索用户、群组、生活瞬间等内容
use serde::{Deserialize, Serialize};
use crate::domain::{DomainError, DomainResult};
use super::PaginationQuery;

/// 搜索类型
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum SearchType {
    /// 搜索所有类型
    All,
    /// 只搜索用户
    Users,
    /// 只搜索群组
    Groups,
    /// 只搜索生活瞬间
    Moments,
}

/// 全局搜索查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchByGlobalQuery {
    /// 搜索关键词
    pub query: String,
    /// 搜索类型
    pub search_type: Option<SearchType>,
    /// 分页参数
    #[serde(flatten)]
    pub pagination: PaginationQuery,
    /// 用户位置（用于地理位置相关的搜索排序）
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
}

/// 搜索结果项
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchResultItem {
    /// 结果ID
    pub id: String,
    /// 结果类型
    pub result_type: SearchResultType,
    /// 标题
    pub title: String,
    /// 描述
    pub description: Option<String>,
    /// 头像或图片URL
    pub avatar_url: Option<String>,
    /// 徽章文本
    pub badge: Option<String>,
    /// 徽章颜色
    pub badge_color: Option<String>,
    /// 距离（如果有位置信息）
    pub distance_km: Option<f64>,
    /// 相关性评分
    pub relevance_score: f64,
    /// 创建时间
    pub created_at: chrono::DateTime<chrono::Utc>,
}

/// 搜索结果类型
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum SearchResultType {
    User,
    Group,
    Moment,
}

/// 搜索结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchResult {
    /// 搜索结果列表
    pub items: Vec<SearchResultItem>,
    /// 分页信息
    pub has_more: bool,
    pub next_cursor: Option<i64>,
    /// 搜索统计
    pub total_found: Option<i64>,
    /// 搜索用时（毫秒）
    pub search_time_ms: u64,
}

impl SearchByGlobalQuery {
    /// 验证搜索查询
    pub fn validate(&self) -> DomainResult<()> {
        let mut errors = Vec::new();
        
        // 验证搜索关键词
        let query = self.query.trim();
        if query.is_empty() {
            errors.push("搜索关键词不能为空".to_string());
        } else if query.len() < 2 {
            errors.push("搜索关键词至少需要2个字符".to_string());
        } else if query.len() > 100 {
            errors.push("搜索关键词不能超过100个字符".to_string());
        }
        
        // 验证位置信息（如果提供）
        if let (Some(lat), Some(lng)) = (self.latitude, self.longitude) {
            if lat < -90.0 || lat > 90.0 {
                errors.push("纬度必须在-90到90之间".to_string());
            }
            if lng < -180.0 || lng > 180.0 {
                errors.push("经度必须在-180到180之间".to_string());
            }
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(DomainError::BusinessRuleViolation {
                rule: errors.join("; "),
            })
        }
    }
    
    /// 获取清理后的搜索关键词
    pub fn get_clean_query(&self) -> String {
        self.query.trim().to_lowercase()
    }
    
    /// 是否包含位置信息
    pub fn has_location(&self) -> bool {
        self.latitude.is_some() && self.longitude.is_some()
    }
}

impl Default for SearchType {
    fn default() -> Self {
        SearchType::All
    }
}

impl SearchResultItem {
    /// 创建用户搜索结果
    pub fn from_user(
        user_id: i64,
        nickname: String,
        avatar_url: Option<String>,
        distance_km: Option<f64>,
        is_online: bool,
        created_at: chrono::DateTime<chrono::Utc>,
    ) -> Self {
        let description = match distance_km {
            Some(dist) if dist < 1.0 => format!("附近 {}m", (dist * 1000.0) as i32),
            Some(dist) => format!("附近 {:.1}km", dist),
            None => "位置未知".to_string(),
        };
        
        let (badge, badge_color) = if is_online {
            (Some("在线".to_string()), Some("green".to_string()))
        } else {
            (None, None)
        };
        
        Self {
            id: user_id.to_string(),
            result_type: SearchResultType::User,
            title: nickname,
            description: Some(description),
            avatar_url,
            badge,
            badge_color,
            distance_km,
            relevance_score: 1.0, // 可以根据匹配度计算
            created_at,
        }
    }
    
    /// 创建群组搜索结果
    pub fn from_group(
        group_id: String,
        name: String,
        _description: Option<String>,
        member_count: i64,
        distance_km: Option<f64>,
        is_active: bool,
        created_at: chrono::DateTime<chrono::Utc>,
    ) -> Self {
        let desc = match distance_km {
            Some(dist) => format!("{} 名成员 · 距离 {:.1}km", member_count, dist),
            None => format!("{} 名成员", member_count),
        };
        
        let (badge, badge_color) = if is_active {
            (Some("活跃".to_string()), Some("blue".to_string()))
        } else {
            (None, None)
        };
        
        Self {
            id: group_id,
            result_type: SearchResultType::Group,
            title: name,
            description: Some(desc),
            avatar_url: None, // 群组头像可以后续添加
            badge,
            badge_color,
            distance_km,
            relevance_score: 1.0,
            created_at,
        }
    }
    
    /// 创建生活瞬间搜索结果
    pub fn from_moment(
        moment_id: String,
        content: String,
        user_nickname: String,
        likes_count: i64,
        distance_km: Option<f64>,
        is_popular: bool,
        created_at: chrono::DateTime<chrono::Utc>,
    ) -> Self {
        let time_ago = format_time_ago(created_at);
        let description = match distance_km {
            Some(dist) => format!("{} · {} · 距离 {:.1}km", time_ago, user_nickname, dist),
            None => format!("{} · {}", time_ago, user_nickname),
        };
        
        let (badge, badge_color) = if is_popular {
            (Some("热门".to_string()), Some("orange".to_string()))
        } else if likes_count > 10 {
            (Some("受欢迎".to_string()), Some("pink".to_string()))
        } else {
            (None, None)
        };
        
        Self {
            id: moment_id,
            result_type: SearchResultType::Moment,
            title: content.chars().take(50).collect::<String>() + if content.len() > 50 { "..." } else { "" },
            description: Some(description),
            avatar_url: None, // 可以是瞬间的第一张图片
            badge,
            badge_color,
            distance_km,
            relevance_score: 1.0,
            created_at,
        }
    }
}

/// 格式化时间为相对时间
fn format_time_ago(time: chrono::DateTime<chrono::Utc>) -> String {
    let now = chrono::Utc::now();
    let duration = now.signed_duration_since(time);
    
    if let Ok(std_duration) = duration.to_std() {
        let seconds = std_duration.as_secs();
        
        if seconds < 60 {
            "刚刚".to_string()
        } else if seconds < 3600 {
            format!("{}分钟前", seconds / 60)
        } else if seconds < 86400 {
            format!("{}小时前", seconds / 3600)
        } else if seconds < 2592000 {
            format!("{}天前", seconds / 86400)
        } else {
            format!("{}个月前", seconds / 2592000)
        }
    } else {
        "未知时间".to_string()
    }
}
