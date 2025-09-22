/// 搜索优化服务
/// 
/// 功能职责：
/// - 提供搜索缓存管理
/// - 实现搜索索引优化
/// - 管理搜索性能监控
/// - 提供搜索结果预加载

use async_trait::async_trait;
use std::sync::Arc;
use std::collections::HashMap;
use tokio::sync::RwLock;
use tracing::{debug, info, warn};
use serde::{Serialize, Deserialize};

use crate::domain::DomainResult;
use crate::application::services::enhanced_search_service::{EnhancedSearchService, SearchStats};
use crate::application::services::search_application_service::UnifiedSearchResult;
use crate::application::queries::SearchResultType;

/// 搜索缓存条目
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchCacheEntry {
    pub query: String,
    pub results: Vec<UnifiedSearchResult>,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub expires_at: chrono::DateTime<chrono::Utc>,
    pub hit_count: u64,
    pub last_accessed: chrono::DateTime<chrono::Utc>,
}

/// 搜索索引条目
#[derive(Debug, Clone)]
pub struct SearchIndexEntry {
    pub id: String,
    pub content_type: SearchResultType,
    pub keywords: Vec<String>,
    pub title: String,
    pub description: Option<String>,
    pub popularity_score: f64,
    pub location: Option<(f64, f64)>, // (latitude, longitude)
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
}

/// 搜索性能指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchPerformanceMetrics {
    pub total_queries: u64,
    pub cache_hit_rate: f64,
    pub average_response_time_ms: f64,
    pub slow_queries_count: u64,
    pub popular_queries: Vec<(String, u64)>,
    pub error_rate: f64,
    pub peak_qps: u64,
    pub current_qps: u64,
}

/// 搜索优化配置
#[derive(Debug, Clone)]
pub struct SearchOptimizationConfig {
    /// 缓存最大条目数
    pub max_cache_entries: usize,
    /// 缓存TTL（秒）
    pub cache_ttl_seconds: u64,
    /// 慢查询阈值（毫秒）
    pub slow_query_threshold_ms: u64,
    /// 索引更新间隔（秒）
    pub index_update_interval_seconds: u64,
    /// 预加载热门查询
    pub preload_popular_queries: bool,
    /// 启用查询建议
    pub enable_query_suggestions: bool,
}

impl Default for SearchOptimizationConfig {
    fn default() -> Self {
        Self {
            max_cache_entries: 10000,
            cache_ttl_seconds: 300, // 5分钟
            slow_query_threshold_ms: 1000, // 1秒
            index_update_interval_seconds: 3600, // 1小时
            preload_popular_queries: true,
            enable_query_suggestions: true,
        }
    }
}

/// 搜索优化服务接口
#[async_trait]
pub trait SearchOptimizationService: Send + Sync {
    /// 获取缓存的搜索结果
    async fn get_cached_results(&self, query: &str) -> Option<Vec<UnifiedSearchResult>>;
    
    /// 缓存搜索结果
    async fn cache_search_results(&self, query: String, results: Vec<UnifiedSearchResult>);
    
    /// 预热搜索缓存
    async fn warm_up_cache(&self) -> DomainResult<u32>;
    
    /// 清理过期缓存
    async fn cleanup_expired_cache(&self) -> DomainResult<u32>;
    
    /// 获取搜索性能指标
    async fn get_performance_metrics(&self) -> DomainResult<SearchPerformanceMetrics>;
    
    /// 记录搜索查询
    async fn record_search_query(&self, query: String, response_time_ms: u64, result_count: usize, is_cache_hit: bool);
    
    /// 获取查询建议
    async fn get_query_suggestions(&self, partial_query: String) -> DomainResult<Vec<String>>;
    
    /// 更新搜索索引
    async fn update_search_index(&self, entry: SearchIndexEntry) -> DomainResult<()>;
    
    /// 批量更新搜索索引
    async fn batch_update_search_index(&self, entries: Vec<SearchIndexEntry>) -> DomainResult<()>;
}

/// 搜索优化服务实现
pub struct SearchOptimizationServiceImpl {
    /// 搜索缓存
    cache: Arc<RwLock<HashMap<String, SearchCacheEntry>>>,
    /// 搜索索引
    index: Arc<RwLock<HashMap<String, SearchIndexEntry>>>,
    /// 查询统计
    query_stats: Arc<RwLock<HashMap<String, (u64, u64)>>>, // (count, total_time_ms)
    /// 性能指标
    performance_metrics: Arc<RwLock<SearchPerformanceMetrics>>,
    /// 配置
    config: SearchOptimizationConfig,
}

impl SearchOptimizationServiceImpl {
    pub fn new() -> Self {
        Self {
            cache: Arc::new(RwLock::new(HashMap::new())),
            index: Arc::new(RwLock::new(HashMap::new())),
            query_stats: Arc::new(RwLock::new(HashMap::new())),
            performance_metrics: Arc::new(RwLock::new(SearchPerformanceMetrics {
                total_queries: 0,
                cache_hit_rate: 0.0,
                average_response_time_ms: 0.0,
                slow_queries_count: 0,
                popular_queries: vec![],
                error_rate: 0.0,
                peak_qps: 0,
                current_qps: 0,
            })),
            config: SearchOptimizationConfig::default(),
        }
    }
    
    pub fn with_config(mut self, config: SearchOptimizationConfig) -> Self {
        self.config = config;
        self
    }
    
    /// 生成缓存键
    fn generate_cache_key(&self, query: &str) -> String {
        // 标准化查询作为缓存键
        query.trim().to_lowercase()
    }
    
    /// 检查缓存条目是否过期
    fn is_cache_entry_expired(&self, entry: &SearchCacheEntry) -> bool {
        chrono::Utc::now() > entry.expires_at
    }
    
    /// 清理单个过期缓存条目
    async fn cleanup_single_expired_entry(&self, key: &str) -> bool {
        let mut cache = self.cache.write().await;
        if let Some(entry) = cache.get(key) {
            if self.is_cache_entry_expired(entry) {
                cache.remove(key);
                return true;
            }
        }
        false
    }
    
    /// 更新性能指标
    async fn update_performance_metrics(&self) {
        let cache = self.cache.read().await;
        let stats = self.query_stats.read().await;
        
        let total_queries = stats.values().map(|(count, _)| count).sum::<u64>();
        let total_time = stats.values().map(|(_, time)| time).sum::<u64>();
        
        let cache_hits = cache.values().map(|entry| entry.hit_count).sum::<u64>();
        let cache_hit_rate = if total_queries > 0 {
            cache_hits as f64 / total_queries as f64
        } else {
            0.0
        };
        
        let average_response_time = if total_queries > 0 {
            total_time as f64 / total_queries as f64
        } else {
            0.0
        };
        
        let slow_queries_count = stats.values()
            .filter(|(_, time)| *time > self.config.slow_query_threshold_ms)
            .count() as u64;
        
        // 获取热门查询
        let mut popular_queries: Vec<(String, u64)> = stats.iter()
            .map(|(query, (count, _))| (query.clone(), *count))
            .collect();
        popular_queries.sort_by(|a, b| b.1.cmp(&a.1));
        popular_queries.truncate(10);
        
        let mut metrics = self.performance_metrics.write().await;
        metrics.total_queries = total_queries;
        metrics.cache_hit_rate = cache_hit_rate;
        metrics.average_response_time_ms = average_response_time;
        metrics.slow_queries_count = slow_queries_count;
        metrics.popular_queries = popular_queries;
    }
    
    /// 提取搜索关键词
    fn extract_keywords(&self, text: &str) -> Vec<String> {
        text.split_whitespace()
            .map(|word| word.to_lowercase())
            .filter(|word| word.len() > 2) // 过滤短词
            .collect()
    }
    
    /// 计算查询相似度
    fn calculate_query_similarity(&self, query1: &str, query2: &str) -> f64 {
        let words1: std::collections::HashSet<&str> = query1.split_whitespace().collect();
        let words2: std::collections::HashSet<&str> = query2.split_whitespace().collect();
        
        let intersection = words1.intersection(&words2).count();
        let union = words1.union(&words2).count();
        
        if union == 0 {
            0.0
        } else {
            intersection as f64 / union as f64
        }
    }
}

#[async_trait]
impl SearchOptimizationService for SearchOptimizationServiceImpl {
    /// 获取缓存的搜索结果
    async fn get_cached_results(&self, query: &str) -> Option<Vec<UnifiedSearchResult>> {
        let cache_key = self.generate_cache_key(query);
        let mut cache = self.cache.write().await;
        
        if let Some(entry) = cache.get_mut(&cache_key) {
            if !self.is_cache_entry_expired(entry) {
                // 更新访问统计
                entry.hit_count += 1;
                entry.last_accessed = chrono::Utc::now();
                
                debug!("缓存命中: 查询='{}', 命中次数={}", query, entry.hit_count);
                return Some(entry.results.clone());
            } else {
                // 移除过期条目
                cache.remove(&cache_key);
                debug!("缓存过期并移除: 查询='{}'", query);
            }
        }
        
        None
    }
    
    /// 缓存搜索结果
    async fn cache_search_results(&self, query: String, results: Vec<UnifiedSearchResult>) {
        let cache_key = self.generate_cache_key(&query);
        let now = chrono::Utc::now();
        let expires_at = now + chrono::Duration::seconds(self.config.cache_ttl_seconds as i64);
        
        let entry = SearchCacheEntry {
            query: query.clone(),
            results,
            created_at: now,
            expires_at,
            hit_count: 0,
            last_accessed: now,
        };
        
        let mut cache = self.cache.write().await;
        
        // 如果缓存已满，移除最旧的条目
        if cache.len() >= self.config.max_cache_entries {
            if let Some((oldest_key, _)) = cache.iter()
                .min_by_key(|(_, entry)| entry.last_accessed)
                .map(|(k, v)| (k.clone(), v.clone())) {
                cache.remove(&oldest_key);
                debug!("缓存已满，移除最旧条目: {}", oldest_key);
            }
        }
        
        cache.insert(cache_key, entry);
        debug!("缓存搜索结果: 查询='{}', 结果数={}", query, results.len());
    }
    
    /// 预热搜索缓存
    async fn warm_up_cache(&self) -> DomainResult<u32> {
        debug!("开始预热搜索缓存");
        
        if !self.config.preload_popular_queries {
            return Ok(0);
        }
        
        // TODO: 实现缓存预热逻辑
        // 1. 获取热门搜索词
        // 2. 执行搜索并缓存结果
        // 3. 预加载常用的搜索组合
        
        let preloaded_count = 0;
        info!("搜索缓存预热完成: 预加载{}个查询", preloaded_count);
        Ok(preloaded_count)
    }
    
    /// 清理过期缓存
    async fn cleanup_expired_cache(&self) -> DomainResult<u32> {
        debug!("开始清理过期搜索缓存");
        
        let mut cache = self.cache.write().await;
        let initial_count = cache.len();
        
        let now = chrono::Utc::now();
        cache.retain(|_, entry| now <= entry.expires_at);
        
        let cleaned_count = initial_count - cache.len();
        info!("搜索缓存清理完成: 清理{}个过期条目", cleaned_count);
        Ok(cleaned_count as u32)
    }
    
    /// 获取搜索性能指标
    async fn get_performance_metrics(&self) -> DomainResult<SearchPerformanceMetrics> {
        self.update_performance_metrics().await;
        let metrics = self.performance_metrics.read().await;
        Ok(metrics.clone())
    }
    
    /// 记录搜索查询
    async fn record_search_query(&self, query: String, response_time_ms: u64, result_count: usize, is_cache_hit: bool) {
        let mut stats = self.query_stats.write().await;
        let entry = stats.entry(query.clone()).or_insert((0, 0));
        entry.0 += 1; // 增加查询次数
        entry.1 += response_time_ms; // 累加响应时间
        
        debug!("记录搜索查询: 查询='{}', 响应时间={}ms, 结果数={}, 缓存命中={}", 
               query, response_time_ms, result_count, is_cache_hit);
        
        // 异步更新性能指标
        let metrics_clone = Arc::clone(&self.performance_metrics);
        tokio::spawn(async move {
            // 这里可以进行更复杂的性能分析
        });
    }
    
    /// 获取查询建议
    async fn get_query_suggestions(&self, partial_query: String) -> DomainResult<Vec<String>> {
        if !self.config.enable_query_suggestions {
            return Ok(vec![]);
        }
        
        debug!("生成查询建议: 部分查询='{}'", partial_query);
        
        let mut suggestions = Vec::new();
        
        // 基于历史查询生成建议
        let stats = self.query_stats.read().await;
        for (query, (count, _)) in stats.iter() {
            if query.starts_with(&partial_query) && query != &partial_query {
                suggestions.push((query.clone(), *count));
            }
        }
        
        // 按查询频率排序
        suggestions.sort_by(|a, b| b.1.cmp(&a.1));
        
        // 返回前10个建议
        let result: Vec<String> = suggestions.into_iter()
            .take(10)
            .map(|(query, _)| query)
            .collect();
        
        Ok(result)
    }
    
    /// 更新搜索索引
    async fn update_search_index(&self, entry: SearchIndexEntry) -> DomainResult<()> {
        debug!("更新搜索索引: ID={}, 类型={:?}", entry.id, entry.content_type);
        
        let mut index = self.index.write().await;
        index.insert(entry.id.clone(), entry);
        
        Ok(())
    }
    
    /// 批量更新搜索索引
    async fn batch_update_search_index(&self, entries: Vec<SearchIndexEntry>) -> DomainResult<()> {
        debug!("批量更新搜索索引: 数量={}", entries.len());
        
        let mut index = self.index.write().await;
        for entry in entries {
            index.insert(entry.id.clone(), entry);
        }
        
        info!("批量索引更新完成: 更新{}个条目", index.len());
        Ok(())
    }
}

impl Clone for SearchOptimizationServiceImpl {
    fn clone(&self) -> Self {
        Self {
            cache: Arc::clone(&self.cache),
            index: Arc::clone(&self.index),
            query_stats: Arc::clone(&self.query_stats),
            performance_metrics: Arc::clone(&self.performance_metrics),
            config: self.config.clone(),
        }
    }
}

/// 搜索优化管理器
/// 
/// 提供搜索优化的统一管理接口
pub struct SearchOptimizationManager {
    optimization_service: Arc<dyn SearchOptimizationService>,
    enhanced_search_service: Arc<dyn EnhancedSearchService>,
}

impl SearchOptimizationManager {
    pub fn new(
        optimization_service: Arc<dyn SearchOptimizationService>,
        enhanced_search_service: Arc<dyn EnhancedSearchService>,
    ) -> Self {
        Self {
            optimization_service,
            enhanced_search_service,
        }
    }
    
    /// 执行优化搜索（带缓存）
    pub async fn optimized_search(
        &self,
        query: String,
        latitude: Option<f64>,
        longitude: Option<f64>,
        radius_km: Option<f64>,
        limit: i32,
    ) -> DomainResult<Vec<UnifiedSearchResult>> {
        let start_time = std::time::Instant::now();
        
        // 尝试从缓存获取结果
        if let Some(cached_results) = self.optimization_service.get_cached_results(&query).await {
            let response_time = start_time.elapsed().as_millis() as u64;
            self.optimization_service.record_search_query(
                query,
                response_time,
                cached_results.len(),
                true, // 缓存命中
            ).await;
            
            return Ok(cached_results);
        }
        
        // 缓存未命中，执行实际搜索
        let results = self.enhanced_search_service.unified_search(
            query.clone(),
            latitude,
            longitude,
            radius_km,
            limit,
        ).await?;
        
        // 缓存搜索结果
        self.optimization_service.cache_search_results(query.clone(), results.clone()).await;
        
        // 记录搜索统计
        let response_time = start_time.elapsed().as_millis() as u64;
        self.optimization_service.record_search_query(
            query,
            response_time,
            results.len(),
            false, // 缓存未命中
        ).await;
        
        Ok(results)
    }
    
    /// 定期维护任务
    pub async fn perform_maintenance(&self) -> DomainResult<()> {
        debug!("执行搜索优化维护任务");
        
        // 清理过期缓存
        let cleaned_count = self.optimization_service.cleanup_expired_cache().await?;
        
        // 预热缓存
        let preloaded_count = self.optimization_service.warm_up_cache().await?;
        
        info!("搜索优化维护完成: 清理{}个过期缓存, 预加载{}个查询", 
              cleaned_count, preloaded_count);
        
        Ok(())
    }
    
    /// 获取优化统计报告
    pub async fn get_optimization_report(&self) -> DomainResult<serde_json::Value> {
        let performance_metrics = self.optimization_service.get_performance_metrics().await?;
        let search_stats = self.enhanced_search_service.get_search_stats().await?;
        
        Ok(serde_json::json!({
            "performance_metrics": performance_metrics,
            "search_stats": search_stats,
            "timestamp": chrono::Utc::now()
        }))
    }
}
