/// 活动应用服务
/// 
/// 提供系统活动数据的查询和管理功能
use std::sync::Arc;
use tracing::{info, warn};
use chrono::{DateTime, Utc, Duration};

use crate::{
    domain::DomainResult,
    ports::{UserRepository, GroupRepository, MomentRepository},
    application::queries::{
        GetRecentActivitiesQuery, ActivitiesResult, ActivityItem, ActivityType,
    },
};

/// 活动应用服务
#[derive(Clone)]
pub struct ActivityApplicationService<U, G, M> 
where
    U: UserRepository,
    G: GroupRepository,
    M: MomentRepository,
{
    user_repository: Arc<U>,
    group_repository: Arc<G>,
    moment_repository: Arc<M>,
}

impl<U, G, M> ActivityApplicationService<U, G, M>
where
    U: UserRepository,
    G: GroupRepository,
    M: MomentRepository,
{
    /// 创建新的活动服务
    pub fn new(
        user_repository: Arc<U>,
        group_repository: Arc<G>,
        moment_repository: Arc<M>,
    ) -> Self {
        Self {
            user_repository,
            group_repository,
            moment_repository,
        }
    }
    
    /// 获取最近活动
    pub async fn get_recent_activities(&self, query: GetRecentActivitiesQuery) -> DomainResult<ActivitiesResult> {
        info!("获取最近活动，时间范围: {}小时", query.get_hours_range());
        
        // 验证查询参数
        query.validate()?;
        
        let mut all_activities = Vec::new();
        
        // 计算时间范围
        let now = Utc::now();
        let start_time = now - Duration::hours(query.get_hours_range() as i64);
        
        // 收集各种类型的活动
        if query.activity_types.is_none() || 
           query.activity_types.as_ref().unwrap().contains(&ActivityType::UserRegistered) {
            let user_activities = self.collect_user_activities(start_time, now).await?;
            all_activities.extend(user_activities);
        }
        
        if query.activity_types.is_none() || 
           query.activity_types.as_ref().unwrap().contains(&ActivityType::GroupCreated) ||
           query.activity_types.as_ref().unwrap().contains(&ActivityType::GroupJoined) {
            let group_activities = self.collect_group_activities(start_time, now).await?;
            all_activities.extend(group_activities);
        }
        
        if query.activity_types.is_none() || 
           query.activity_types.as_ref().unwrap().contains(&ActivityType::MomentCreated) {
            let moment_activities = self.collect_moment_activities(start_time, now).await?;
            all_activities.extend(moment_activities);
        }
        
        // 按时间倒序排序
        all_activities.sort_by(|a, b| b.time.cmp(&a.time));
        
        // 应用分页
        let limit = query.pagination.get_limit() as usize;
        let cursor = query.pagination.get_cursor() as usize;
        
        // 简单的偏移分页（对于活动数据来说足够了）
        let start_index = cursor;
        let end_index = (start_index + limit).min(all_activities.len());
        
        let page_activities = if start_index < all_activities.len() {
            all_activities[start_index..end_index].to_vec()
        } else {
            Vec::new()
        };
        
        let has_more = end_index < all_activities.len();
        let next_cursor = if has_more {
            Some(end_index as i64)
        } else {
            None
        };
        
        info!("找到{}个活动，返回{}个", all_activities.len(), page_activities.len());
        
        Ok(ActivitiesResult {
            activities: page_activities,
            has_more,
            next_cursor,
            total_count: Some(all_activities.len() as i64),
        })
    }
    
    /// 收集用户相关活动
    async fn collect_user_activities(&self, start_time: DateTime<Utc>, _end_time: DateTime<Utc>) -> DomainResult<Vec<ActivityItem>> {
        let mut activities = Vec::new();
        
        // 获取最近注册的用户
        // 注意：这里简化实现，实际应该有专门的查询方法
        match self.user_repository.search_user_profiles_by_name("", 0, 50).await {
            Ok(users) => {
                for user in users {
                    if user.last_active_at >= start_time {
                        activities.push(ActivityItem::user_registered(
                            user.user_id,
                            user.nickname,
                            user.last_active_at,
                        ));
                    }
                }
            }
            Err(e) => {
                warn!("获取用户活动失败: {:?}", e);
            }
        }
        
        Ok(activities)
    }
    
    /// 收集群组相关活动
    async fn collect_group_activities(&self, start_time: DateTime<Utc>, _end_time: DateTime<Utc>) -> DomainResult<Vec<ActivityItem>> {
        let mut activities = Vec::new();
        
        // 获取最近创建的群组
        match self.group_repository.search_groups_by_name("", 0, 50).await {
            Ok(groups) => {
                for group in groups {
                    if group.last_active_at >= start_time {
                        // GroupSearchResult没有creator_name，使用占位符
                        activities.push(ActivityItem::group_created(
                            group.group_id.to_string(),
                            group.name,
                            "群主".to_string(), // 占位符，实际应该查询用户信息
                            group.last_active_at,
                        ));
                    }
                }
            }
            Err(e) => {
                warn!("获取群组活动失败: {:?}", e);
            }
        }
        
        Ok(activities)
    }
    
    /// 收集生活瞬间相关活动
    async fn collect_moment_activities(&self, start_time: DateTime<Utc>, _end_time: DateTime<Utc>) -> DomainResult<Vec<ActivityItem>> {
        let mut activities = Vec::new();
        
        // 获取最近创建的生活瞬间（使用默认位置进行搜索）
        match self.moment_repository.find_nearby_moments(
            35.16,    // 默认纬度
            112.68,   // 默认经度
            100.0,    // 100公里半径
            0,        // 游标
            50        // 限制50条
        ).await {
            Ok(moments) => {
                for moment in moments {
                    if moment.created_at >= start_time {
                        activities.push(ActivityItem::moment_created(
                            moment.moment_id,
                            moment.user_id.to_string(), // 使用user_id作为用户名占位符
                            moment.content.clone(),
                            moment.created_at,
                        ));
                    }
                }
            }
            Err(e) => {
                warn!("获取生活瞬间活动失败: {:?}", e);
            }
        }
        
        Ok(activities)
    }
    
    /// 生成模拟活动数据（用于演示）
    pub async fn generate_demo_activities(&self) -> DomainResult<ActivitiesResult> {
        info!("生成演示活动数据");
        
        let now = Utc::now();
        let mut activities = Vec::new();
        
        // 生成一些示例活动
        activities.push(ActivityItem::user_registered(
            12345,
            "新用户小明".to_string(),
            now - Duration::minutes(30),
        ));
        
        activities.push(ActivityItem::group_created(
            "group-uuid-123".to_string(),
            "技术交流群".to_string(),
            "群主老王".to_string(),
            now - Duration::hours(2),
        ));
        
        activities.push(ActivityItem::group_joined(
            "group-uuid-123".to_string(),
            "技术交流群".to_string(),
            "开发者小李".to_string(),
            now - Duration::hours(1),
        ));
        
        activities.push(ActivityItem::moment_created(
            "moment-uuid-456".to_string(),
            "摄影师小张".to_string(),
            "今天的日落真美，分享给大家看看！".to_string(),
            now - Duration::minutes(45),
        ));
        
        activities.push(ActivityItem::moment_liked(
            "moment-uuid-456".to_string(),
            "点赞达人小赵".to_string(),
            "摄影师小张".to_string(),
            now - Duration::minutes(20),
        ));
        
        activities.push(ActivityItem::location_updated(
            "旅行者小刘".to_string(),
            "北京市朝阳区".to_string(),
            now - Duration::minutes(10),
        ));
        
        // 按时间倒序排序
        activities.sort_by(|a, b| b.time.cmp(&a.time));
        
        Ok(ActivitiesResult {
            activities,
            has_more: false,
            next_cursor: None,
            total_count: Some(6),
        })
    }
}
