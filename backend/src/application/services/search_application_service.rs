/// 搜索应用服务
/// 
/// 提供跨模块的统一搜索功能

use async_trait::async_trait;
use std::sync::Arc;
use serde::{Serialize, Deserialize};

use crate::domain::{DomainResult, User, Group, Moment};
use crate::ports::{UserRepositoryPort, GroupRepositoryPort, MomentRepositoryPort};

/// 搜索结果类型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SearchResultType {
    User,
    Group,
    Moment,
}

/// 统一搜索结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnifiedSearchResult {
    pub result_type: SearchResultType,
    pub id: String,
    pub title: String,
    pub description: Option<String>,
    pub score: f64,
    pub data: serde_json::Value,
}

/// 搜索应用服务接口
#[async_trait]
pub trait SearchApplicationService: Send + Sync {
    /// 统一搜索
    async fn unified_search(
        &self,
        query: String,
        latitude: Option<f64>,
        longitude: Option<f64>,
        radius_km: Option<f64>,
        limit: i32,
    ) -> DomainResult<Vec<UnifiedSearchResult>>;
    
    /// 搜索用户
    async fn search_users(
        &self,
        query: String,
        limit: i32,
    ) -> DomainResult<Vec<User>>;
    
    /// 搜索群组
    async fn search_groups(
        &self,
        query: String,
        latitude: Option<f64>,
        longitude: Option<f64>,
        radius_km: Option<f64>,
        limit: i32,
    ) -> DomainResult<Vec<Group>>;
    
    /// 搜索生活瞬间
    async fn search_moments(
        &self,
        query: String,
        latitude: Option<f64>,
        longitude: Option<f64>,
        radius_km: Option<f64>,
        limit: i32,
    ) -> DomainResult<Vec<Moment>>;
    
    /// 全局搜索（兼容现有控制器）
    async fn global_search(
        &self,
        query: crate::application::GlobalSearchQuery,
        user_id: i64,
    ) -> DomainResult<serde_json::Value>;
}

/// 搜索应用服务实现
pub struct SearchApplicationServiceImpl {
    user_repository: Arc<dyn UserRepositoryPort>,
    group_repository: Arc<dyn GroupRepositoryPort>,
    moment_repository: Arc<dyn MomentRepositoryPort>,
}

impl SearchApplicationServiceImpl {
    /// 创建新的搜索应用服务
    pub fn new(
        user_repository: Arc<dyn UserRepositoryPort>,
        group_repository: Arc<dyn GroupRepositoryPort>,
        moment_repository: Arc<dyn MomentRepositoryPort>,
    ) -> Self {
        Self {
            user_repository,
            group_repository,
            moment_repository,
        }
    }
    
    /// 计算搜索结果的相关性分数
    fn calculate_relevance_score(&self, query: &str, title: &str, description: Option<&str>) -> f64 {
        let query_lower = query.to_lowercase();
        let title_lower = title.to_lowercase();
        
        let mut score = 0.0;
        
        // 标题完全匹配得分最高
        if title_lower == query_lower {
            score += 100.0;
        } else if title_lower.contains(&query_lower) {
            score += 50.0;
        }
        
        // 描述匹配得分较低
        if let Some(desc) = description {
            let desc_lower = desc.to_lowercase();
            if desc_lower.contains(&query_lower) {
                score += 20.0;
            }
        }
        
        // 根据匹配长度调整分数
        let match_ratio = query_lower.len() as f64 / title_lower.len() as f64;
        score *= match_ratio;
        
        score
    }

    /// 将用户转换为统一搜索结果
    fn user_to_search_result(&self, user: User, query: &str) -> UnifiedSearchResult {
        let score = self.calculate_relevance_score(query, &user.nickname, None);
        
        UnifiedSearchResult {
            result_type: SearchResultType::User,
            id: user.user_id.to_string(),
            title: user.nickname.clone(),
            description: None,
            score,
            data: serde_json::to_value(&user).unwrap_or_default(),
        }
    }

    /// 将群组转换为统一搜索结果
    fn group_to_search_result(&self, group: Group, query: &str) -> UnifiedSearchResult {
        let score = self.calculate_relevance_score(query, &group.name, group.description.as_deref());
        
        UnifiedSearchResult {
            result_type: SearchResultType::Group,
            id: group.group_id.to_string(),
            title: group.name.clone(),
            description: group.description.clone(),
            score,
            data: serde_json::to_value(&group).unwrap_or_default(),
        }
    }

    /// 将生活瞬间转换为统一搜索结果
    fn moment_to_search_result(&self, moment: Moment, query: &str) -> UnifiedSearchResult {
        let title = moment.content.clone().unwrap_or_else(|| "无标题".to_string());
        let score = self.calculate_relevance_score(query, &title, None);
        
        UnifiedSearchResult {
            result_type: SearchResultType::Moment,
            id: moment.moment_id.to_string(),
            title,
            description: Some(moment.location_name.clone()),
            score,
            data: serde_json::to_value(&moment).unwrap_or_default(),
        }
    }
}

#[async_trait]
impl SearchApplicationService for SearchApplicationServiceImpl {
    async fn unified_search(
        &self,
        query: String,
        latitude: Option<f64>,
        longitude: Option<f64>,
        radius_km: Option<f64>,
        limit: i32,
    ) -> DomainResult<Vec<UnifiedSearchResult>> {
        let mut results = Vec::new();
        
        // 搜索用户（限制数量）
        let user_limit = std::cmp::min(limit / 3, 10);
        if let Ok(users) = self.search_users(query.clone(), user_limit).await {
            for user in users {
                results.push(self.user_to_search_result(user, &query));
            }
        }
        
        // 搜索群组（限制数量）
        let group_limit = std::cmp::min(limit / 3, 10);
        if let Ok(groups) = self.search_groups(query.clone(), latitude, longitude, radius_km, group_limit).await {
            for group in groups {
                results.push(self.group_to_search_result(group, &query));
            }
        }
        
        // 搜索生活瞬间（限制数量）
        let moment_limit = std::cmp::min(limit / 3, 10);
        if let Ok(moments) = self.search_moments(query.clone(), latitude, longitude, radius_km, moment_limit).await {
            for moment in moments {
                results.push(self.moment_to_search_result(moment, &query));
            }
        }
        
        // 按相关性分数排序
        results.sort_by(|a, b| b.score.partial_cmp(&a.score).unwrap_or(std::cmp::Ordering::Equal));
        
        // 限制返回结果数量
        results.truncate(limit as usize);
        
        Ok(results)
    }

    async fn search_users(&self, query: String, limit: i32) -> DomainResult<Vec<User>> {
        self.user_repository.search_users_by_nickname(&query, 0, limit).await
    }

    async fn search_groups(
        &self,
        query: String,
        latitude: Option<f64>,
        longitude: Option<f64>,
        radius_km: Option<f64>,
        limit: i32,
    ) -> DomainResult<Vec<Group>> {
        if let (Some(lat), Some(lng), Some(radius)) = (latitude, longitude, radius_km) {
            // 如果提供了位置信息，优先进行地理位置搜索
            self.group_repository.find_nearby_groups(lat, lng, radius, 0, limit).await
        } else {
            // 否则按名称搜索
            self.group_repository.search_groups_by_name(&query, 0, limit).await
        }
    }

    async fn search_moments(
        &self,
        query: String,
        latitude: Option<f64>,
        longitude: Option<f64>,
        radius_km: Option<f64>,
        limit: i32,
    ) -> DomainResult<Vec<Moment>> {
        if let (Some(lat), Some(lng), Some(radius)) = (latitude, longitude, radius_km) {
            // 如果提供了位置信息，进行地理位置搜索
            self.moment_repository.find_nearby_moments(lat, lng, radius, 0, limit).await
        } else {
            // 否则按标签搜索（将查询词作为标签）
            let tags = vec![query];
            self.moment_repository.search_by_tags(tags, 0, limit).await
        }
    }
    
    async fn global_search(
        &self,
        query: crate::application::GlobalSearchQuery,
        _user_id: i64,
    ) -> DomainResult<serde_json::Value> {
        // 简化实现，返回统一搜索结果
        let results = self.unified_search(
            query.query,
            query.latitude,
            query.longitude,
            Some(5.0), // 默认搜索半径5公里
            query.pagination.limit.unwrap_or(20),
        ).await?;
        
        Ok(serde_json::json!({
            "results": results,
            "total": results.len()
        }))
    }
}

/// 搜索管理器
/// 
/// 提供搜索相关的管理功能
#[derive(Clone)]
pub struct SearchManager {
    search_service: Arc<SearchApplicationServiceImpl>,
}

impl SearchManager {
    /// 创建新的搜索管理器
    pub fn new(
        user_repository: Arc<dyn UserRepositoryPort>,
        group_repository: Arc<dyn GroupRepositoryPort>,
        moment_repository: Arc<dyn MomentRepositoryPort>,
    ) -> Self {
        let search_service = Arc::new(SearchApplicationServiceImpl::new(
            user_repository,
            group_repository,
            moment_repository,
        ));
        
        Self { search_service }
    }

    /// 获取搜索应用服务
    pub fn get_search_service(&self) -> Arc<SearchApplicationServiceImpl> {
        self.search_service.clone()
    }

    /// 健康检查
    pub async fn health_check(&self) -> DomainResult<bool> {
        // 尝试执行一个简单的搜索来验证搜索服务是否正常工作
        match self.search_service.search_users("test".to_string(), 1).await {
            Ok(_) => Ok(true),
            Err(_) => Ok(false),
        }
    }
}