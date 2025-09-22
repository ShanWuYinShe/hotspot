/// 增强搜索服务
/// 
/// 功能职责：
/// - 提供高性能的搜索功能
/// - 实现智能相关性算法
/// - 支持多种搜索策略和优化
/// - 提供搜索结果缓存和分析

use async_trait::async_trait;
use std::sync::Arc;
use std::collections::HashMap;
use tracing::{debug, info, warn};
use serde::{Serialize, Deserialize};

use crate::domain::{DomainResult, DomainError, UserProfile, GroupProfile, Moment};
use crate::ports::{UserRepository, GroupRepository, MomentRepository};
use crate::application::queries::SearchResultType;
use crate::application::services::search_application_service::{SearchApplicationService, UnifiedSearchResult};

/// 搜索配置
#[derive(Debug, Clone)]
pub struct SearchConfig {
    /// 最大搜索结果数量
    pub max_results: i32,
    /// 搜索超时时间（毫秒）
    pub timeout_ms: u64,
    /// 是否启用缓存
    pub enable_cache: bool,
    /// 缓存过期时间（秒）
    pub cache_ttl_seconds: u64,
    /// 最小搜索词长度
    pub min_query_length: usize,
    /// 相关性阈值
    pub relevance_threshold: f64,
}

impl Default for SearchConfig {
    fn default() -> Self {
        Self {
            max_results: 50,
            timeout_ms: 5000,
            enable_cache: true,
            cache_ttl_seconds: 300, // 5分钟
            min_query_length: 2,
            relevance_threshold: 0.1,
        }
    }
}

/// 搜索统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchStats {
    pub total_searches: u64,
    pub cache_hits: u64,
    pub cache_misses: u64,
    pub average_response_time_ms: f64,
    pub popular_queries: Vec<(String, u64)>,
    pub search_types_distribution: HashMap<String, u64>,
}

/// 相关性因子
#[derive(Debug, Clone)]
pub struct RelevanceFactors {
    /// 完全匹配权重
    pub exact_match_weight: f64,
    /// 前缀匹配权重
    pub prefix_match_weight: f64,
    /// 包含匹配权重
    pub contains_match_weight: f64,
    /// 模糊匹配权重
    pub fuzzy_match_weight: f64,
    /// 位置相关性权重
    pub location_weight: f64,
    /// 时间相关性权重
    pub time_weight: f64,
    /// 流行度权重
    pub popularity_weight: f64,
}

impl Default for RelevanceFactors {
    fn default() -> Self {
        Self {
            exact_match_weight: 100.0,
            prefix_match_weight: 80.0,
            contains_match_weight: 60.0,
            fuzzy_match_weight: 40.0,
            location_weight: 20.0,
            time_weight: 10.0,
            popularity_weight: 15.0,
        }
    }
}

/// 增强搜索服务接口
#[async_trait]
pub trait EnhancedSearchService: SearchApplicationService {
    /// 智能搜索（带相关性优化）
    async fn intelligent_search(
        &self,
        query: String,
        search_types: Vec<SearchResultType>,
        latitude: Option<f64>,
        longitude: Option<f64>,
        radius_km: Option<f64>,
        limit: i32,
    ) -> DomainResult<Vec<UnifiedSearchResult>>;
    
    /// 搜索建议（自动补全）
    async fn search_suggestions(&self, partial_query: String, limit: i32) -> DomainResult<Vec<String>>;
    
    /// 热门搜索词
    async fn popular_searches(&self, limit: i32) -> DomainResult<Vec<String>>;
    
    /// 搜索统计
    async fn get_search_stats(&self) -> DomainResult<SearchStats>;
    
    /// 清理搜索缓存
    async fn clear_search_cache(&self) -> DomainResult<()>;
}

/// 增强搜索服务实现
pub struct EnhancedSearchServiceImpl {
    user_repository: Arc<dyn UserRepository>,
    group_repository: Arc<dyn GroupRepository>,
    moment_repository: Arc<dyn MomentRepository>,
    config: SearchConfig,
    relevance_factors: RelevanceFactors,
    // TODO: 添加缓存服务
    // cache_service: Arc<dyn CacheService>,
    // TODO: 添加搜索统计服务
    // stats_service: Arc<dyn SearchStatsService>,
}

impl EnhancedSearchServiceImpl {
    pub fn new(
        user_repository: Arc<dyn UserRepository>,
        group_repository: Arc<dyn GroupRepository>,
        moment_repository: Arc<dyn MomentRepository>,
    ) -> Self {
        Self {
            user_repository,
            group_repository,
            moment_repository,
            config: SearchConfig::default(),
            relevance_factors: RelevanceFactors::default(),
        }
    }
    
    pub fn with_config(mut self, config: SearchConfig) -> Self {
        self.config = config;
        self
    }
    
    pub fn with_relevance_factors(mut self, factors: RelevanceFactors) -> Self {
        self.relevance_factors = factors;
        self
    }
    
    /// 预处理搜索查询
    fn preprocess_query(&self, query: &str) -> String {
        let processed = query
            .trim()
            .to_lowercase()
            // 移除特殊字符
            .chars()
            .filter(|c| c.is_alphanumeric() || c.is_whitespace() || *c == '_' || *c == '-')
            .collect::<String>()
            // 标准化空白字符
            .split_whitespace()
            .collect::<Vec<&str>>()
            .join(" ");
        
        processed
    }
    
    /// 计算增强相关性分数
    fn calculate_enhanced_relevance_score(
        &self,
        query: &str,
        title: &str,
        description: Option<&str>,
        location_distance: Option<f64>,
        created_at: Option<chrono::DateTime<chrono::Utc>>,
        popularity_score: Option<f64>,
    ) -> f64 {
        let query_lower = query.to_lowercase();
        let title_lower = title.to_lowercase();
        
        let mut total_score = 0.0;
        
        // 1. 文本匹配分数
        let text_score = self.calculate_text_match_score(&query_lower, &title_lower, description);
        total_score += text_score;
        
        // 2. 位置相关性分数
        if let Some(distance) = location_distance {
            let location_score = self.calculate_location_score(distance);
            total_score += location_score * self.relevance_factors.location_weight;
        }
        
        // 3. 时间相关性分数
        if let Some(created) = created_at {
            let time_score = self.calculate_time_score(created);
            total_score += time_score * self.relevance_factors.time_weight;
        }
        
        // 4. 流行度分数
        if let Some(popularity) = popularity_score {
            total_score += popularity * self.relevance_factors.popularity_weight;
        }
        
        total_score
    }
    
    /// 计算文本匹配分数
    fn calculate_text_match_score(&self, query: &str, title: &str, description: Option<&str>) -> f64 {
        let mut score = 0.0;
        
        // 完全匹配
        if title == query {
            score += self.relevance_factors.exact_match_weight;
        }
        // 前缀匹配
        else if title.starts_with(query) {
            score += self.relevance_factors.prefix_match_weight;
        }
        // 包含匹配
        else if title.contains(query) {
            score += self.relevance_factors.contains_match_weight;
            
            // 计算匹配位置权重（越靠前权重越高）
            if let Some(pos) = title.find(query) {
                let position_weight = 1.0 - (pos as f64 / title.len() as f64);
                score += position_weight * 10.0;
            }
        }
        // 模糊匹配
        else {
            let fuzzy_score = self.calculate_fuzzy_match_score(query, title);
            score += fuzzy_score * self.relevance_factors.fuzzy_match_weight;
        }
        
        // 描述匹配
        if let Some(desc) = description {
            let desc_lower = desc.to_lowercase();
            if desc_lower.contains(query) {
                score += 20.0; // 描述匹配得分较低
            }
        }
        
        // 查询词覆盖率
        let query_words: Vec<&str> = query.split_whitespace().collect();
        if query_words.len() > 1 {
            let covered_words = query_words.iter()
                .filter(|word| title.contains(*word))
                .count();
            let coverage_ratio = covered_words as f64 / query_words.len() as f64;
            score += coverage_ratio * 30.0;
        }
        
        score
    }
    
    /// 计算模糊匹配分数（简单的编辑距离）
    fn calculate_fuzzy_match_score(&self, query: &str, title: &str) -> f64 {
        let max_len = query.len().max(title.len());
        if max_len == 0 {
            return 1.0;
        }
        
        let distance = self.levenshtein_distance(query, title);
        let similarity = 1.0 - (distance as f64 / max_len as f64);
        
        // 只有相似度超过阈值才给分
        if similarity > 0.6 {
            similarity
        } else {
            0.0
        }
    }
    
    /// 计算编辑距离
    fn levenshtein_distance(&self, s1: &str, s2: &str) -> usize {
        let len1 = s1.len();
        let len2 = s2.len();
        
        if len1 == 0 {
            return len2;
        }
        if len2 == 0 {
            return len1;
        }
        
        let mut matrix = vec![vec![0; len2 + 1]; len1 + 1];
        
        // 初始化第一行和第一列
        for i in 0..=len1 {
            matrix[i][0] = i;
        }
        for j in 0..=len2 {
            matrix[0][j] = j;
        }
        
        let s1_chars: Vec<char> = s1.chars().collect();
        let s2_chars: Vec<char> = s2.chars().collect();
        
        for i in 1..=len1 {
            for j in 1..=len2 {
                let cost = if s1_chars[i - 1] == s2_chars[j - 1] { 0 } else { 1 };
                matrix[i][j] = (matrix[i - 1][j] + 1)
                    .min(matrix[i][j - 1] + 1)
                    .min(matrix[i - 1][j - 1] + cost);
            }
        }
        
        matrix[len1][len2]
    }
    
    /// 计算位置相关性分数
    fn calculate_location_score(&self, distance_km: f64) -> f64 {
        // 距离越近分数越高，使用指数衰减
        let max_distance = 50.0; // 50公里内有效
        if distance_km > max_distance {
            return 0.0;
        }
        
        (-distance_km / max_distance * 2.0).exp()
    }
    
    /// 计算时间相关性分数
    fn calculate_time_score(&self, created_at: chrono::DateTime<chrono::Utc>) -> f64 {
        let now = chrono::Utc::now();
        let age_days = (now - created_at).num_days() as f64;
        
        // 越新的内容分数越高
        let max_age_days = 365.0; // 一年内有效
        if age_days > max_age_days {
            return 0.0;
        }
        
        1.0 - (age_days / max_age_days)
    }
    
    /// 验证搜索查询
    fn validate_search_query(&self, query: &str) -> DomainResult<()> {
        if query.trim().is_empty() {
            return Err(DomainError::ValidationError {
                field: "query".to_string(),
                reason: "搜索查询不能为空".to_string(),
            });
        }
        
        if query.len() < self.config.min_query_length {
            return Err(DomainError::ValidationError {
                field: "query".to_string(),
                reason: format!("搜索查询长度不能少于{}个字符", self.config.min_query_length),
            });
        }
        
        if query.len() > 100 {
            return Err(DomainError::ValidationError {
                field: "query".to_string(),
                reason: "搜索查询长度不能超过100个字符".to_string(),
            });
        }
        
        Ok(())
    }
    
    /// 过滤和排序搜索结果
    fn filter_and_sort_results(&self, mut results: Vec<UnifiedSearchResult>) -> Vec<UnifiedSearchResult> {
        // 过滤低相关性结果
        results.retain(|result| result.score >= self.relevance_factors.relevance_threshold);
        
        // 按相关性分数排序
        results.sort_by(|a, b| b.score.partial_cmp(&a.score).unwrap_or(std::cmp::Ordering::Equal));
        
        // 限制结果数量
        results.truncate(self.config.max_results as usize);
        
        results
    }
    
    /// 记录搜索统计
    async fn record_search_stats(&self, query: &str, result_count: usize, response_time_ms: u64) {
        // TODO: 实现搜索统计记录
        debug!("搜索统计: 查询='{}', 结果数={}, 响应时间={}ms", query, result_count, response_time_ms);
    }
}

#[async_trait]
impl SearchApplicationService for EnhancedSearchServiceImpl {
    /// 统一搜索
    async fn unified_search(
        &self,
        query: String,
        latitude: Option<f64>,
        longitude: Option<f64>,
        radius_km: Option<f64>,
        limit: i32,
    ) -> DomainResult<Vec<UnifiedSearchResult>> {
        let start_time = std::time::Instant::now();
        
        // 验证查询
        self.validate_search_query(&query)?;
        
        // 预处理查询
        let processed_query = self.preprocess_query(&query);
        
        debug!("开始统一搜索: 查询='{}', 处理后='{}'", query, processed_query);
        
        let mut results = Vec::new();
        
        // 搜索用户（限制数量）
        let user_limit = std::cmp::min(limit / 3, 15);
        if let Ok(users) = self.search_users(processed_query.clone(), user_limit).await {
            for user in users {
                let score = self.calculate_enhanced_relevance_score(
                    &processed_query,
                    &user.nickname,
                    None,
                    None, // TODO: 计算用户位置距离
                    Some(user.last_active_at),
                    None, // TODO: 计算用户流行度
                );
                
                results.push(UnifiedSearchResult {
                    result_type: SearchResultType::User,
                    id: user.user_id.to_string(),
                    title: user.nickname.clone(),
                    description: None,
                    score,
                    data: serde_json::to_value(&user).unwrap_or_default(),
                });
            }
        }
        
        // 搜索群组（限制数量）
        let group_limit = std::cmp::min(limit / 3, 15);
        if let Ok(groups) = self.search_groups(processed_query.clone(), latitude, longitude, radius_km, group_limit).await {
            for group in groups {
                // 计算位置距离
                let distance = if let (Some(lat), Some(lng)) = (latitude, longitude) {
                    let group_lat = group.latitude;
                    let group_lng = group.longitude;
                    Some(self.calculate_distance(lat, lng, group_lat, group_lng))
                } else {
                    None
                };
                
                let score = self.calculate_enhanced_relevance_score(
                    &processed_query,
                    &group.name,
                    group.description.as_deref(),
                    distance,
                    Some(group.last_active_at),
                    Some(group.member_count as f64 / 100.0), // 成员数作为流行度指标
                );
                
                results.push(UnifiedSearchResult {
                    result_type: SearchResultType::Group,
                    id: group.group_id.clone(),
                    title: group.name.clone(),
                    description: group.description.clone(),
                    score,
                    data: serde_json::to_value(&group).unwrap_or_default(),
                });
            }
        }
        
        // 搜索生活瞬间（限制数量）
        let moment_limit = std::cmp::min(limit / 3, 15);
        if let Ok(moments) = self.search_moments(processed_query.clone(), latitude, longitude, radius_km, moment_limit).await {
            for moment in moments {
                let score = self.calculate_enhanced_relevance_score(
                    &processed_query,
                    &moment.content,
                    None,
                    None, // TODO: 计算瞬间位置距离
                    Some(moment.created_at),
                    None, // TODO: 计算瞬间流行度（点赞数、评论数等）
                );
                
                results.push(UnifiedSearchResult {
                    result_type: SearchResultType::Moment,
                    id: moment.moment_id.clone(),
                    title: moment.content.clone(),
                    description: None,
                    score,
                    data: serde_json::to_value(&moment).unwrap_or_default(),
                });
            }
        }
        
        // 过滤和排序结果
        let filtered_results = self.filter_and_sort_results(results);
        
        // 记录统计信息
        let response_time = start_time.elapsed().as_millis() as u64;
        self.record_search_stats(&query, filtered_results.len(), response_time).await;
        
        info!("统一搜索完成: 查询='{}', 结果数={}, 响应时间={}ms", 
              query, filtered_results.len(), response_time);
        
        Ok(filtered_results)
    }
    
    /// 搜索用户
    async fn search_users(&self, query: String, limit: i32) -> DomainResult<Vec<UserProfile>> {
        let search_results = self.user_repository.search_user_profiles_by_name(&query, 0, limit).await?;
        let profiles = search_results.into_iter().map(|result| UserProfile {
            user_id: result.user_id,
            nickname: result.nickname,
            avatar_url: result.avatar_url,
            user_type: result.user_type,
            last_active_at: result.last_active_at,
        }).collect();
        Ok(profiles)
    }
    
    /// 搜索群组
    async fn search_groups(
        &self,
        query: String,
        latitude: Option<f64>,
        longitude: Option<f64>,
        radius_km: Option<f64>,
        limit: i32,
    ) -> DomainResult<Vec<GroupProfile>> {
        let search_results = if let (Some(lat), Some(lng), Some(radius)) = (latitude, longitude, radius_km) {
            self.group_repository.find_nearby_groups_with_distance(lat, lng, radius * 1000.0, 0, limit).await?
        } else {
            self.group_repository.search_groups_by_name(&query, 0, limit).await?
        };
        
        let profiles = search_results.into_iter().map(|result| GroupProfile {
            group_id: result.group_id,
            name: result.name,
            description: result.description,
            owner_id: result.owner_id,
            latitude: result.latitude,
            longitude: result.longitude,
            location_name: result.location_name,
            member_count: result.member_count,
            group_type: result.group_type,
            visibility: result.visibility,
            last_active_at: result.last_active_at,
            user_role: None,
        }).collect();
        Ok(profiles)
    }
    
    /// 搜索生活瞬间
    async fn search_moments(
        &self,
        query: String,
        latitude: Option<f64>,
        longitude: Option<f64>,
        radius_km: Option<f64>,
        limit: i32,
    ) -> DomainResult<Vec<Moment>> {
        if let (Some(lat), Some(lng), Some(radius)) = (latitude, longitude, radius_km) {
            self.moment_repository.find_nearby_moments(lat, lng, radius, 0, limit).await
        } else {
            let tags = vec![query];
            self.moment_repository.search_by_tags(&tags, 0, limit).await
        }
    }
    
    /// 全局搜索（兼容现有控制器）
    async fn global_search(
        &self,
        query: crate::application::SearchByGlobalQuery,
        user_id: i64,
    ) -> DomainResult<serde_json::Value> {
        // 使用统一搜索实现全局搜索
        let results = self.unified_search(
            query.query,
            query.latitude,
            query.longitude,
            query.radius_km,
            query.limit.unwrap_or(20),
        ).await?;
        
        Ok(serde_json::json!({
            "results": results,
            "total": results.len(),
            "user_id": user_id
        }))
    }
}

#[async_trait]
impl EnhancedSearchService for EnhancedSearchServiceImpl {
    /// 智能搜索（带相关性优化）
    async fn intelligent_search(
        &self,
        query: String,
        search_types: Vec<SearchResultType>,
        latitude: Option<f64>,
        longitude: Option<f64>,
        radius_km: Option<f64>,
        limit: i32,
    ) -> DomainResult<Vec<UnifiedSearchResult>> {
        debug!("智能搜索: 查询='{}', 类型={:?}", query, search_types);
        
        // 如果没有指定搜索类型，使用统一搜索
        if search_types.is_empty() {
            return self.unified_search(query, latitude, longitude, radius_km, limit).await;
        }
        
        let mut results = Vec::new();
        
        // 根据指定的搜索类型进行搜索
        for search_type in search_types {
            match search_type {
                SearchResultType::User => {
                    if let Ok(users) = self.search_users(query.clone(), limit).await {
                        for user in users {
                            results.push(UnifiedSearchResult {
                                result_type: SearchResultType::User,
                                id: user.user_id.to_string(),
                                title: user.nickname.clone(),
                                description: None,
                                score: self.calculate_enhanced_relevance_score(
                                    &query,
                                    &user.nickname,
                                    None,
                                    None,
                                    Some(user.last_active_at),
                                    None,
                                ),
                                data: serde_json::to_value(&user).unwrap_or_default(),
                            });
                        }
                    }
                }
                SearchResultType::Group => {
                    if let Ok(groups) = self.search_groups(query.clone(), latitude, longitude, radius_km, limit).await {
                        for group in groups {
                            results.push(UnifiedSearchResult {
                                result_type: SearchResultType::Group,
                                id: group.group_id.clone(),
                                title: group.name.clone(),
                                description: group.description.clone(),
                                score: self.calculate_enhanced_relevance_score(
                                    &query,
                                    &group.name,
                                    group.description.as_deref(),
                                    None,
                                    Some(group.last_active_at),
                                    Some(group.member_count as f64 / 100.0),
                                ),
                                data: serde_json::to_value(&group).unwrap_or_default(),
                            });
                        }
                    }
                }
                SearchResultType::Moment => {
                    if let Ok(moments) = self.search_moments(query.clone(), latitude, longitude, radius_km, limit).await {
                        for moment in moments {
                            results.push(UnifiedSearchResult {
                                result_type: SearchResultType::Moment,
                                id: moment.moment_id.clone(),
                                title: moment.content.clone(),
                                description: None,
                                score: self.calculate_enhanced_relevance_score(
                                    &query,
                                    &moment.content,
                                    None,
                                    None,
                                    Some(moment.created_at),
                                    None,
                                ),
                                data: serde_json::to_value(&moment).unwrap_or_default(),
                            });
                        }
                    }
                }
            }
        }
        
        Ok(self.filter_and_sort_results(results))
    }
    
    /// 搜索建议（自动补全）
    async fn search_suggestions(&self, partial_query: String, limit: i32) -> DomainResult<Vec<String>> {
        debug!("搜索建议: 部分查询='{}'", partial_query);
        
        if partial_query.len() < 2 {
            return Ok(vec![]);
        }
        
        let mut suggestions = Vec::new();
        
        // TODO: 实现基于历史搜索的建议
        // TODO: 实现基于热门搜索的建议
        // TODO: 实现基于用户画像的个性化建议
        
        // 简单实现：基于用户昵称的前缀匹配
        if let Ok(users) = self.user_repository.search_user_profiles_by_name(&partial_query, 0, limit).await {
            for user in users {
                if user.nickname.to_lowercase().starts_with(&partial_query.to_lowercase()) {
                    suggestions.push(user.nickname);
                }
            }
        }
        
        // 去重并限制数量
        suggestions.sort();
        suggestions.dedup();
        suggestions.truncate(limit as usize);
        
        Ok(suggestions)
    }
    
    /// 热门搜索词
    async fn popular_searches(&self, limit: i32) -> DomainResult<Vec<String>> {
        debug!("获取热门搜索词: 限制={}", limit);
        
        // 使用搜索分析服务获取真实的热门搜索词
        let analytics_service = SearchAnalyticsService::new(self.pool.clone());
        let popular_searches = analytics_service.get_popular_searches(limit, None).await?;
        
        let search_terms = popular_searches
            .into_iter()
            .map(|ps| ps.query)
            .collect();
        
        Ok(search_terms)
    }
    
    /// 搜索统计
    async fn get_search_stats(&self) -> DomainResult<SearchStats> {
        debug!("获取搜索统计");
        
        // TODO: 实现真实的搜索统计
        Ok(SearchStats {
            total_searches: 0,
            cache_hits: 0,
            cache_misses: 0,
            average_response_time_ms: 0.0,
            popular_queries: vec![],
            search_types_distribution: HashMap::new(),
        })
    }
    
    /// 清理搜索缓存
    async fn clear_search_cache(&self) -> DomainResult<()> {
        debug!("清理搜索缓存");
        
        // TODO: 实现缓存清理
        Ok(())
    }
}

impl EnhancedSearchServiceImpl {
    /// 计算两点间距离（公里）
    fn calculate_distance(&self, lat1: f64, lng1: f64, lat2: f64, lng2: f64) -> f64 {
        let earth_radius = 6371.0; // 地球半径（公里）
        
        let lat1_rad = lat1.to_radians();
        let lat2_rad = lat2.to_radians();
        let delta_lat = (lat2 - lat1).to_radians();
        let delta_lng = (lng2 - lng1).to_radians();
        
        let a = (delta_lat / 2.0).sin().powi(2) +
                lat1_rad.cos() * lat2_rad.cos() * (delta_lng / 2.0).sin().powi(2);
        let c = 2.0 * a.sqrt().atan2((1.0 - a).sqrt());
        
        earth_radius * c
    }
}
