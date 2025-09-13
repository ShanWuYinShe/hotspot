/// 领域服务 - 不属于特定实体的业务逻辑
/// 
/// 领域服务特征：
/// - 无状态
/// - 纯业务逻辑
/// - 操作多个聚合或实体
/// - 不依赖外部技术实现

use crate::domain::{
    values::*,
    entities::*,
    DomainError, DomainResult,
};

/// 用户领域服务
pub struct UserDomainService;

impl UserDomainService {
    /// 验证用户是否可以创建群组
    pub fn can_create_group(user: &User) -> DomainResult<()> {
        // 检查用户是否已过期
        if user.is_expired() {
            return Err(DomainError::TempUserExpired);
        }
        
        // 临时用户可以创建群组，但有限制
        if user.user_type == UserType::Temporary {
            // 可以添加临时用户的特殊限制
            // 例如：临时用户只能创建小型群组
        }
        
        Ok(())
    }
    
    /// 验证用户是否可以加入群组
    pub fn can_join_group(user: &User, group: &Group) -> DomainResult<()> {
        // 检查用户是否已过期
        if user.is_expired() {
            return Err(DomainError::TempUserExpired);
        }
        
        // 检查群组是否已满
        if group.member_count >= group.max_members {
            return Err(DomainError::GroupFull);
        }
        
        // 检查用户是否已是成员
        if group.is_member(user.id) {
            return Err(DomainError::AlreadyGroupMember);
        }
        
        Ok(())
    }
    
    /// 验证用户是否可以发送消息
    pub fn can_send_message(user: &User) -> DomainResult<()> {
        // 检查用户是否已过期
        if user.is_expired() {
            return Err(DomainError::TempUserExpired);
        }
        
        Ok(())
    }
    
    /// 验证用户是否可以创建瞬间
    pub fn can_create_moment(user: &User) -> DomainResult<()> {
        // 检查用户是否已过期
        if user.is_expired() {
            return Err(DomainError::TempUserExpired);
        }
        
        Ok(())
    }
    
    /// 计算用户之间的关系强度（用于通知策略）
    pub fn calculate_relationship_strength(user1: &User, user2: &User) -> f64 {
        let mut strength = 0.0;
        
        // 基础关系强度
        match (user1.user_type, user2.user_type) {
            (UserType::Regular, UserType::Regular) => strength += 1.0,
            (UserType::Regular, UserType::Temporary) | 
            (UserType::Temporary, UserType::Regular) => strength += 0.5,
            (UserType::Temporary, UserType::Temporary) => strength += 0.3,
        }
        
        // 位置相近性加成
        if let (Some(loc1), Some(loc2)) = (&user1.location, &user2.location) {
            let distance = loc1.coordinates.distance_to(&loc2.coordinates);
            if distance.meters() < 1000.0 {
                strength += 0.5; // 1公里内
            } else if distance.meters() < 5000.0 {
                strength += 0.3; // 5公里内
            } else if distance.meters() < 10000.0 {
                strength += 0.1; // 10公里内
            }
        }
        
        strength
    }
}

/// 群组领域服务
pub struct GroupDomainService;

impl GroupDomainService {
    /// 验证群组创建参数
    pub fn validate_group_creation(
        name: &GroupName,
        description: &Option<String>,
        location: &Coordinates,
        radius: f64,
    ) -> DomainResult<()> {
        // 验证半径
        if radius <= 0.0 || radius > 100_000.0 {
            return Err(DomainError::validation("radius", "must be between 0 and 100,000 meters"));
        }
        
        // 验证描述
        if let Some(desc) = description {
            if desc.trim().len() > 300 {
                return Err(DomainError::validation("description", "must not exceed 300 characters"));
            }
        }
        
        Ok(())
    }
    
    /// 计算群组活跃度分数
    pub fn calculate_activity_score(group: &Group) -> f64 {
        let now = chrono::Utc::now();
        let days_since_last_active = (now - group.last_active_at).num_days() as f64;
        
        // 基础分数基于成员数量
        let member_score = (group.member_count as f64).ln() + 1.0;
        
        // 活跃度衰减
        let activity_decay = (-days_since_last_active / 7.0).exp(); // 一周衰减
        
        member_score * activity_decay
    }
    
    /// 检查用户是否在群组活动范围内
    pub fn is_user_in_group_range(user_location: &Coordinates, group: &Group) -> bool {
        let distance = user_location.distance_to(&group.location.coordinates);
        distance.meters() <= group.location.radius
    }
    
    /// 计算群组推荐分数
    pub fn calculate_recommendation_score(
        user: &User,
        group: &Group,
        common_members: usize,
    ) -> f64 {
        let mut score = 0.0;
        
        // 基础活跃度分数
        score += Self::calculate_activity_score(group) * 0.3;
        
        // 地理位置相关性
        if let Some(user_location) = &user.location {
            let distance = user_location.coordinates.distance_to(&group.location.coordinates);
            let location_score = (1.0 / (1.0 + distance.meters() / 1000.0)).max(0.1); // 距离越近分数越高
            score += location_score * 0.4;
        }
        
        // 共同成员加成
        let social_score = (common_members as f64 * 0.1).min(1.0);
        score += social_score * 0.3;
        
        score
    }
}

/// 消息领域服务
pub struct MessageDomainService;

impl MessageDomainService {
    /// 验证消息内容
    pub fn validate_message_content(content: &MessageContent) -> DomainResult<()> {
        if content.content().is_empty() {
            return Err(DomainError::EmptyMessageContent);
        }
        
        if content.content().len() > 2000 {
            return Err(DomainError::MessageTooLong);
        }
        
        Ok(())
    }
    
    /// 检查消息是否需要审核
    pub fn requires_moderation(content: &MessageContent, sender: &User) -> bool {
        // 临时用户的消息可能需要更严格的审核
        if sender.user_type == UserType::Temporary {
            return true;
        }
        
        // 检查消息长度（过长的消息可能需要审核）
        if content.content().len() > 1000 {
            return true;
        }
        
        // 可以添加更多审核规则，如敏感词检测等
        false
    }
    
    /// 计算消息优先级（用于推送排序）
    pub fn calculate_message_priority(
        sender: &User,
        recipient_type: &str,
        is_mention: bool,
    ) -> u8 {
        let mut priority = 5; // 基础优先级
        
        // 注册用户的消息优先级更高
        if sender.user_type == UserType::Regular {
            priority += 2;
        }
        
        // 私聊消息优先级更高
        if recipient_type == "user" {
            priority += 3;
        }
        
        // 提及消息优先级最高
        if is_mention {
            priority += 5;
        }
        
        priority.min(10) // 最高优先级为10
    }
}

/// 瞬间领域服务
pub struct MomentDomainService;

impl MomentDomainService {
    /// 验证瞬间创建参数
    pub fn validate_moment_creation(
        description: &str,
        tags: &[String],
        image_urls: &[String],
    ) -> DomainResult<()> {
        // 验证描述
        let trimmed_desc = description.trim();
        if trimmed_desc.is_empty() {
            return Err(DomainError::EmptyMomentContent);
        }
        
        if trimmed_desc.len() > 500 {
            return Err(DomainError::validation("description", "must not exceed 500 characters"));
        }
        
        // 验证标签数量
        if tags.len() > 10 {
            return Err(DomainError::TooManyTags);
        }
        
        // 验证每个标签
        for tag in tags {
            let trimmed_tag = tag.trim();
            if trimmed_tag.is_empty() {
                return Err(DomainError::validation("tag", "cannot be empty"));
            }
            if trimmed_tag.len() > 20 {
                return Err(DomainError::validation("tag", "must not exceed 20 characters"));
            }
        }
        
        // 验证图片数量
        if image_urls.len() > 9 {
            return Err(DomainError::TooManyImages);
        }
        
        Ok(())
    }
    
    /// 计算瞬间热度分数
    pub fn calculate_hotness_score(
        likes_count: u32,
        comments_count: u32,
        created_at: chrono::DateTime<chrono::Utc>,
    ) -> f64 {
        let now = chrono::Utc::now();
        let hours_since_creation = (now - created_at).num_hours() as f64;
        
        // 基础分数：点赞数 + 评论数 * 2（评论权重更高）
        let base_score = likes_count as f64 + comments_count as f64 * 2.0;
        
        // 时间衰减：24小时内保持高分，之后逐渐衰减
        let time_decay = if hours_since_creation <= 24.0 {
            1.0
        } else {
            (-((hours_since_creation - 24.0) / 48.0)).exp() // 48小时后衰减到约37%
        };
        
        base_score * time_decay
    }
    
    /// 检查用户是否可以查看瞬间
    pub fn can_view_moment(
        viewer: &User,
        moment_location: &Coordinates,
        max_distance: f64,
    ) -> bool {
        // 如果用户没有位置信息，不能查看基于位置的瞬间
        let Some(user_location) = &viewer.location else {
            return false;
        };
        
        // 检查距离
        let distance = user_location.coordinates.distance_to(moment_location);
        distance.meters() <= max_distance
    }
}

/// 地理位置领域服务
pub struct GeolocationDomainService;

impl GeolocationDomainService {
    /// 计算两点之间的距离（米）
    pub fn calculate_distance(point1: &Coordinates, point2: &Coordinates) -> f64 {
        point1.distance_to(point2).meters()
    }
    
    /// 检查坐标是否在指定区域内
    pub fn is_within_area(
        point: &Coordinates,
        center: &Coordinates,
        radius: f64,
    ) -> bool {
        Self::calculate_distance(point, center) <= radius
    }
    
    /// 计算地理围栏
    pub fn create_geofence(
        center: &Coordinates,
        radius: f64,
    ) -> DomainResult<GeofenceArea> {
        if radius <= 0.0 || radius > 100_000.0 {
            return Err(DomainError::SearchRadiusExceeded);
        }
        
        Ok(GeofenceArea {
            center: *center,
            radius,
        })
    }
    
    /// 验证搜索半径是否合理
    pub fn validate_search_radius(radius: f64) -> DomainResult<()> {
        if radius <= 0.0 {
            return Err(DomainError::validation("radius", "must be positive"));
        }
        
        if radius > 100_000.0 {
            return Err(DomainError::SearchRadiusExceeded);
        }
        
        Ok(())
    }
}

/// 地理围栏区域
#[derive(Debug, Clone)]
pub struct GeofenceArea {
    pub center: Coordinates,
    pub radius: f64,
}

impl GeofenceArea {
    /// 检查点是否在围栏内
    pub fn contains(&self, point: &Coordinates) -> bool {
        GeolocationDomainService::is_within_area(point, &self.center, self.radius)
    }
    
    /// 计算到围栏边界的距离
    pub fn distance_to_boundary(&self, point: &Coordinates) -> f64 {
        let distance_to_center = GeolocationDomainService::calculate_distance(point, &self.center);
        (distance_to_center - self.radius).abs()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_user_can_create_group() {
        let nickname = Nickname::new("TestUser".to_string()).unwrap();
        let user_id = UserId::new(1).unwrap();
        let user = User::new(user_id, nickname, UserType::Regular);
        
        assert!(UserDomainService::can_create_group(&user).is_ok());
    }
    
    #[test]
    fn test_expired_user_cannot_create_group() {
        let nickname = Nickname::new("TestUser".to_string()).unwrap();
        let user_id = UserId::new(2).unwrap();
        let mut user = User::new(user_id, nickname, UserType::Temporary);
        
        // 手动设置过期时间为过去
        user.expires_at = Some(chrono::Utc::now() - chrono::Duration::days(1));
        
        assert!(UserDomainService::can_create_group(&user).is_err());
    }
    
    #[test]
    fn test_group_activity_score() {
        let group_name = GroupName::new("Test Group".to_string()).unwrap();
        let owner_id = UserId::new(1).unwrap();
        let coordinates = Coordinates::new(39.9, 116.4).unwrap();
        let group_id = GroupId::new("test-group-123".to_string()).unwrap();
        let location = GroupLocation::new(coordinates, "Beijing".to_string(), 1000.0);
        
        let group = Group::new(
            group_id,
            group_name,
            owner_id,
            location,
            GroupType::Public,
            GroupVisibility::Public,
        );
        
        let score = GroupDomainService::calculate_activity_score(&group);
        assert!(score > 0.0);
    }
    
    #[test]
    fn test_message_priority_calculation() {
        let nickname = Nickname::new("TestUser".to_string()).unwrap();
        let user_id1 = UserId::new(3).unwrap();
        let registered_user = User::new(user_id1, nickname.clone(), UserType::Regular);
        
        let user_id2 = UserId::new(4).unwrap();
        let temp_user = User::new(user_id2, nickname, UserType::Temporary);
        
        let registered_priority = MessageDomainService::calculate_message_priority(
            &registered_user,
            "user",
            true,
        );
        
        let temp_priority = MessageDomainService::calculate_message_priority(
            &temp_user,
            "user",
            true,
        );
        
        assert!(registered_priority > temp_priority);
    }
    
    #[test]
    fn test_geofence_area() {
        let center = Coordinates::new(39.9, 116.4).unwrap();
        let geofence = GeolocationDomainService::create_geofence(&center, 1000.0).unwrap();
        
        let inside_point = Coordinates::new(39.901, 116.401).unwrap();
        let outside_point = Coordinates::new(40.0, 117.0).unwrap();
        
        assert!(geofence.contains(&inside_point));
        assert!(!geofence.contains(&outside_point));
    }
}
