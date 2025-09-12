/// 通用应用服务基类
/// 
/// 提供所有应用服务的通用功能，减少重复代码
use async_trait::async_trait;
use std::sync::Arc;
use tracing::{info, warn, error};

use crate::domain::{DomainResult, DomainError};
use crate::ports::{UserRepository, EventPublisher};

/// 通用验证规则
pub struct ValidationRules;

impl ValidationRules {
    /// 验证用户ID
    pub fn validate_user_id(user_id: i64) -> DomainResult<()> {
        if user_id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "用户ID必须为正数".to_string(),
            });
        }
        Ok(())
    }

    /// 验证两个用户ID不相同
    pub fn validate_different_users(user1_id: i64, user2_id: i64, context: &str) -> DomainResult<()> {
        if user1_id == user2_id {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("{}: 不能对自己执行此操作", context),
            });
        }
        Ok(())
    }

    /// 验证文本内容
    pub fn validate_text_content(content: &str, max_length: usize, field_name: &str) -> DomainResult<()> {
        if content.trim().is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("{}不能为空", field_name),
            });
        }
        
        if content.len() > max_length {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("{}长度不能超过{}个字符", field_name, max_length),
            });
        }
        
        Ok(())
    }

    /// 验证地理坐标
    pub fn validate_coordinates(latitude: f64, longitude: f64) -> DomainResult<()> {
        if latitude < -90.0 || latitude > 90.0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "纬度必须在-90到90之间".to_string(),
            });
        }
        
        if longitude < -180.0 || longitude > 180.0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "经度必须在-180到180之间".to_string(),
            });
        }
        
        Ok(())
    }

    /// 验证半径
    pub fn validate_radius(radius: f64, max_radius: f64) -> DomainResult<()> {
        if radius <= 0.0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "半径必须大于0".to_string(),
            });
        }
        
        if radius > max_radius {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("半径不能超过{}米", max_radius),
            });
        }
        
        Ok(())
    }

    /// 验证分页参数
    pub fn validate_pagination(cursor: i64, limit: i32) -> DomainResult<()> {
        if cursor < 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "游标不能为负数".to_string(),
            });
        }
        
        if limit <= 0 || limit > 100 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "分页大小必须在1-100之间".to_string(),
            });
        }
        
        Ok(())
    }
}

/// 通用应用服务基类
#[derive(Clone)]
pub struct BaseApplicationService {
    user_repository: Option<Arc<dyn UserRepository>>,
    event_publisher: Option<Arc<dyn EventPublisher>>,
    service_name: &'static str,
}

impl BaseApplicationService {
    /// 创建基础应用服务
    pub fn new(service_name: &'static str) -> Self {
        Self {
            user_repository: None,
            event_publisher: None,
            service_name,
        }
    }

    /// 设置用户仓储
    pub fn with_user_repository(mut self, user_repository: Arc<dyn UserRepository>) -> Self {
        self.user_repository = Some(user_repository);
        self
    }

    /// 设置事件发布器
    pub fn with_event_publisher(mut self, event_publisher: Arc<dyn EventPublisher>) -> Self {
        self.event_publisher = Some(event_publisher);
        self
    }

    /// 验证用户是否存在
    pub async fn validate_user_exists(&self, user_id: i64) -> DomainResult<()> {
        ValidationRules::validate_user_id(user_id)?;
        
        if let Some(ref user_repo) = self.user_repository {
            let user = user_repo.find_user_profile(user_id).await?;
            if user.is_none() {
                return Err(DomainError::BusinessRuleViolation {
                    rule: "用户不存在".to_string(),
                });
            }
        }
        
        Ok(())
    }

    /// 批量验证用户是否存在
    pub async fn validate_users_exist(&self, user_ids: &[i64]) -> DomainResult<()> {
        for &user_id in user_ids {
            self.validate_user_exists(user_id).await?;
        }
        Ok(())
    }

    /// 记录操作日志
    pub fn log_operation(&self, operation: &str, user_id: Option<i64>, details: Option<&str>) {
        match user_id {
            Some(uid) => info!(
                "[{}] 操作: {} | 用户: {} | 详情: {}",
                self.service_name,
                operation,
                uid,
                details.unwrap_or("无")
            ),
            None => info!(
                "[{}] 操作: {} | 详情: {}",
                self.service_name,
                operation,
                details.unwrap_or("无")
            ),
        }
    }

    /// 记录错误日志
    pub fn log_error(&self, operation: &str, error: &DomainError, user_id: Option<i64>) {
        match user_id {
            Some(uid) => error!(
                "[{}] 操作失败: {} | 用户: {} | 错误: {:?}",
                self.service_name, operation, uid, error
            ),
            None => error!(
                "[{}] 操作失败: {} | 错误: {:?}",
                self.service_name, operation, error
            ),
        }
    }

    /// 记录警告日志
    pub fn log_warning(&self, operation: &str, message: &str, user_id: Option<i64>) {
        match user_id {
            Some(uid) => warn!(
                "[{}] 警告: {} | 用户: {} | 消息: {}",
                self.service_name, operation, uid, message
            ),
            None => warn!(
                "[{}] 警告: {} | 消息: {}",
                self.service_name, operation, message
            ),
        }
    }

    /// 发布用户事件
    pub async fn publish_user_event(&self, event: crate::domain::UserEvent) -> DomainResult<()> {
        if let Some(ref publisher) = self.event_publisher {
            publisher.publish(&event).await?;
        }
        Ok(())
    }

    /// 发布群组事件
    pub async fn publish_group_event(&self, event: crate::domain::GroupEvent) -> DomainResult<()> {
        if let Some(ref publisher) = self.event_publisher {
            publisher.publish(&event).await?;
        }
        Ok(())
    }

    /// 发布消息事件
    pub async fn publish_message_event(&self, event: crate::domain::MessageEvent) -> DomainResult<()> {
        if let Some(ref publisher) = self.event_publisher {
            publisher.publish(&event).await?;
        }
        Ok(())
    }

    /// 发布瞬间事件
    pub async fn publish_moment_event(&self, event: crate::domain::MomentEvent) -> DomainResult<()> {
        if let Some(ref publisher) = self.event_publisher {
            publisher.publish(&event).await?;
        }
        Ok(())
    }

    /// 获取服务名称
    pub fn get_service_name(&self) -> &'static str {
        self.service_name
    }
}

/// 应用服务基础trait（扩展版本）
#[async_trait]
pub trait EnhancedApplicationService: Send + Sync {
    /// 服务名称
    fn service_name(&self) -> &'static str;
    
    /// 健康检查
    async fn health_check(&self) -> DomainResult<bool>;
    
    /// 服务初始化
    async fn initialize(&self) -> DomainResult<()> {
        Ok(())
    }
    
    /// 服务清理
    async fn cleanup(&self) -> DomainResult<()> {
        Ok(())
    }
    
    /// 获取服务统计信息
    async fn get_service_stats(&self) -> DomainResult<serde_json::Value> {
        Ok(serde_json::json!({
            "service_name": self.service_name(),
            "status": "healthy"
        }))
    }
}

/// 分页处理助手
pub struct PaginationHelper;

impl PaginationHelper {
    /// 处理分页结果
    pub fn process_pagination_result<T>(
        mut items: Vec<T>,
        requested_limit: i32,
        get_id: impl Fn(&T) -> i64,
    ) -> (Vec<T>, bool, Option<i64>) {
        let has_more = items.len() > requested_limit as usize;
        
        if has_more {
            items.pop(); // 移除多查询的一条记录
        }
        
        let next_cursor = if has_more && !items.is_empty() {
            Some(get_id(items.last().unwrap()))
        } else {
            None
        };
        
        (items, has_more, next_cursor)
    }

    /// 计算分页查询的limit（总是多查询一条）
    pub fn calculate_query_limit(requested_limit: i32) -> i64 {
        (requested_limit + 1) as i64
    }
}

/// 地理位置处理助手
pub struct GeoHelper;

impl GeoHelper {
    /// 计算两点间距离（米）
    pub fn calculate_distance(lat1: f64, lon1: f64, lat2: f64, lon2: f64) -> f64 {
        const EARTH_RADIUS: f64 = 6371000.0; // 地球半径（米）
        
        let lat1_rad = lat1.to_radians();
        let lat2_rad = lat2.to_radians();
        let delta_lat = (lat2 - lat1).to_radians();
        let delta_lon = (lon2 - lon1).to_radians();
        
        let a = (delta_lat / 2.0).sin().powi(2)
            + lat1_rad.cos() * lat2_rad.cos() * (delta_lon / 2.0).sin().powi(2);
        let c = 2.0 * a.sqrt().atan2((1.0 - a).sqrt());
        
        EARTH_RADIUS * c
    }

    /// 验证地理位置是否在指定范围内
    pub fn is_within_radius(
        center_lat: f64,
        center_lon: f64,
        point_lat: f64,
        point_lon: f64,
        radius_meters: f64,
    ) -> bool {
        let distance = Self::calculate_distance(center_lat, center_lon, point_lat, point_lon);
        distance <= radius_meters
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_validation_rules() {
        // 测试用户ID验证
        assert!(ValidationRules::validate_user_id(1).is_ok());
        assert!(ValidationRules::validate_user_id(0).is_err());
        assert!(ValidationRules::validate_user_id(-1).is_err());

        // 测试不同用户验证
        assert!(ValidationRules::validate_different_users(1, 2, "测试").is_ok());
        assert!(ValidationRules::validate_different_users(1, 1, "测试").is_err());

        // 测试文本内容验证
        assert!(ValidationRules::validate_text_content("有效内容", 100, "测试字段").is_ok());
        assert!(ValidationRules::validate_text_content("", 100, "测试字段").is_err());
        assert!(ValidationRules::validate_text_content("   ", 100, "测试字段").is_err());

        // 测试坐标验证
        assert!(ValidationRules::validate_coordinates(39.9, 116.4).is_ok());
        assert!(ValidationRules::validate_coordinates(91.0, 116.4).is_err());
        assert!(ValidationRules::validate_coordinates(39.9, 181.0).is_err());
    }

    #[test]
    fn test_pagination_helper() {
        let items = vec![1, 2, 3, 4, 5];
        let (result, has_more, next_cursor) = PaginationHelper::process_pagination_result(
            items,
            4,
            |&item| item as i64,
        );
        
        assert_eq!(result.len(), 4);
        assert!(has_more);
        assert_eq!(next_cursor, Some(4));
    }

    #[test]
    fn test_geo_helper() {
        // 测试北京天安门到故宫的距离（约1公里）
        let distance = GeoHelper::calculate_distance(39.9042, 116.4074, 39.9163, 116.3972);
        assert!(distance > 900.0 && distance < 1100.0);

        // 测试范围内判断
        assert!(GeoHelper::is_within_radius(39.9042, 116.4074, 39.9163, 116.3972, 2000.0));
        assert!(!GeoHelper::is_within_radius(39.9042, 116.4074, 39.9163, 116.3972, 500.0));
    }
}
