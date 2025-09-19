/// 数据库查询性能优化器
/// 
/// 功能职责：
/// - 监控和分析数据库查询性能
/// - 提供查询优化建议
/// - 实现查询缓存和预处理
/// - 管理连接池性能
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;
use sqlx::PgPool;
use serde::{Serialize, Deserialize};
use tracing::{info, warn, debug, error};

use crate::domain::DomainResult;
use crate::domain::DomainError;

/// 查询性能监控器
#[derive(Clone)]
pub struct QueryPerformanceOptimizer {
    pool: PgPool,
    query_stats: Arc<RwLock<HashMap<String, QueryStats>>>,
    config: OptimizationConfig,
}

/// 查询统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueryStats {
    pub query_hash: String,
    pub execution_count: u64,
    pub total_duration: Duration,
    pub avg_duration: Duration,
    pub min_duration: Duration,
    pub max_duration: Duration,
    pub last_executed: chrono::DateTime<chrono::Utc>,
    pub slow_query_count: u64,
}

/// 优化配置
#[derive(Debug, Clone)]
pub struct OptimizationConfig {
    pub slow_query_threshold: Duration,
    pub cache_enabled: bool,
    pub max_cached_queries: usize,
    pub stats_retention_hours: u64,
}

impl Default for OptimizationConfig {
    fn default() -> Self {
        Self {
            slow_query_threshold: Duration::from_millis(100),
            cache_enabled: true,
            max_cached_queries: 1000,
            stats_retention_hours: 24,
        }
    }
}

/// 查询优化建议
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OptimizationSuggestion {
    pub query_hash: String,
    pub suggestion_type: SuggestionType,
    pub description: String,
    pub impact: ImpactLevel,
    pub recommended_action: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SuggestionType {
    AddIndex,
    OptimizeQuery,
    UseCache,
    PartitionTable,
    UpdateStatistics,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ImpactLevel {
    Low,
    Medium,
    High,
    Critical,
}

impl QueryPerformanceOptimizer {
    /// 创建新的查询性能优化器
    pub fn new(pool: PgPool, config: OptimizationConfig) -> Self {
        Self {
            pool,
            query_stats: Arc::new(RwLock::new(HashMap::new())),
            config,
        }
    }

    /// 执行带监控的查询
    pub async fn execute_monitored_query<T, F, Fut>(&self, query_name: &str, query_fn: F) -> DomainResult<T>
    where
        F: FnOnce() -> Fut,
        Fut: std::future::Future<Output = DomainResult<T>>,
    {
        let start_time = Instant::now();
        let result = query_fn().await;
        let duration = start_time.elapsed();

        // 记录查询统计
        self.record_query_stats(query_name, duration, result.is_ok()).await;

        result
    }

    /// 记录查询统计信息
    async fn record_query_stats(&self, query_name: &str, duration: Duration, success: bool) {
        let query_hash = self.hash_query_name(query_name);
        let mut stats = self.query_stats.write().await;
        
        let entry = stats.entry(query_hash.clone()).or_insert_with(|| QueryStats {
            query_hash: query_hash.clone(),
            execution_count: 0,
            total_duration: Duration::ZERO,
            avg_duration: Duration::ZERO,
            min_duration: duration,
            max_duration: duration,
            last_executed: chrono::Utc::now(),
            slow_query_count: 0,
        });

        entry.execution_count += 1;
        entry.total_duration += duration;
        entry.avg_duration = entry.total_duration / entry.execution_count as u32;
        entry.min_duration = entry.min_duration.min(duration);
        entry.max_duration = entry.max_duration.max(duration);
        entry.last_executed = chrono::Utc::now();

        if duration > self.config.slow_query_threshold {
            entry.slow_query_count += 1;
            warn!(
                "慢查询检测: {} 执行时间 {:?} 超过阈值 {:?}",
                query_name, duration, self.config.slow_query_threshold
            );
        }

        if !success {
            error!("查询执行失败: {}", query_name);
        }

        debug!(
            "查询统计更新: {} - 执行次数: {}, 平均耗时: {:?}",
            query_name, entry.execution_count, entry.avg_duration
        );
    }

    /// 获取查询统计信息
    pub async fn get_query_stats(&self) -> HashMap<String, QueryStats> {
        self.query_stats.read().await.clone()
    }

    /// 获取慢查询列表
    pub async fn get_slow_queries(&self) -> Vec<QueryStats> {
        let stats = self.query_stats.read().await;
        stats.values()
            .filter(|stat| stat.slow_query_count > 0)
            .cloned()
            .collect()
    }

    /// 生成优化建议
    pub async fn generate_optimization_suggestions(&self) -> DomainResult<Vec<OptimizationSuggestion>> {
        let mut suggestions = Vec::new();
        let stats = self.query_stats.read().await;

        for (_, stat) in stats.iter() {
            // 检查慢查询
            if stat.avg_duration > self.config.slow_query_threshold {
                suggestions.push(OptimizationSuggestion {
                    query_hash: stat.query_hash.clone(),
                    suggestion_type: SuggestionType::OptimizeQuery,
                    description: format!(
                        "查询平均执行时间 {:?} 超过阈值，建议优化",
                        stat.avg_duration
                    ),
                    impact: self.calculate_impact_level(stat),
                    recommended_action: "分析查询计划，考虑添加索引或重写查询".to_string(),
                });
            }

            // 检查高频查询
            if stat.execution_count > 1000 && stat.avg_duration > Duration::from_millis(50) {
                suggestions.push(OptimizationSuggestion {
                    query_hash: stat.query_hash.clone(),
                    suggestion_type: SuggestionType::UseCache,
                    description: format!(
                        "高频查询（{}次）建议使用缓存",
                        stat.execution_count
                    ),
                    impact: ImpactLevel::Medium,
                    recommended_action: "为此查询实现结果缓存".to_string(),
                });
            }
        }

        Ok(suggestions)
    }

    /// 计算影响级别
    fn calculate_impact_level(&self, stat: &QueryStats) -> ImpactLevel {
        let slow_ratio = stat.slow_query_count as f64 / stat.execution_count as f64;
        
        if stat.avg_duration > Duration::from_secs(1) || slow_ratio > 0.5 {
            ImpactLevel::Critical
        } else if stat.avg_duration > Duration::from_millis(500) || slow_ratio > 0.2 {
            ImpactLevel::High
        } else if stat.avg_duration > Duration::from_millis(200) || slow_ratio > 0.1 {
            ImpactLevel::Medium
        } else {
            ImpactLevel::Low
        }
    }

    /// 检查数据库连接池状态
    pub async fn check_connection_pool_health(&self) -> DomainResult<ConnectionPoolHealth> {
        let pool_size = self.pool.size();
        let idle_connections = self.pool.num_idle();
        
        Ok(ConnectionPoolHealth {
            total_connections: pool_size,
            idle_connections,
            active_connections: pool_size as usize - idle_connections,
            utilization_percent: ((pool_size as usize - idle_connections) as f64 / pool_size as f64) * 100.0,
            is_healthy: idle_connections > 0,
        })
    }

    /// 优化地理位置查询
    pub async fn optimize_geo_queries(&self) -> DomainResult<Vec<GeoQueryOptimization>> {
        let mut optimizations = Vec::new();

        // 检查空间索引使用情况
        let spatial_indexes = self.check_spatial_indexes().await?;
        
        for table in &["user_locations", "group_locations", "moments"] {
            if !spatial_indexes.contains_key(*table) {
                optimizations.push(GeoQueryOptimization {
                    table_name: table.to_string(),
                    optimization_type: GeoOptimizationType::MissingSpatialIndex,
                    description: format!("表 {} 缺少空间索引", table),
                    sql_command: Some(format!(
                        "CREATE INDEX idx_{}_location_gist ON {} USING GIST(location);",
                        table, table
                    )),
                });
            }
        }

        Ok(optimizations)
    }

    /// 检查空间索引
    async fn check_spatial_indexes(&self) -> DomainResult<HashMap<String, bool>> {
        let rows = sqlx::query!(
            r#"
            SELECT 
                t.table_name as "table_name!",
                COUNT(i.indexname) > 0 as "has_spatial_index!"
            FROM information_schema.tables t
            LEFT JOIN pg_indexes i ON t.table_name = i.tablename 
                AND i.indexdef LIKE '%USING gist%'
                AND i.indexdef LIKE '%location%'
            WHERE t.table_schema = 'public'
                AND t.table_name IN ('user_locations', 'group_locations', 'moments')
            GROUP BY t.table_name
            "#
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("检查空间索引失败: {}", e),
        })?;

        Ok(rows.into_iter()
            .map(|row| (row.table_name, row.has_spatial_index))
            .collect())
    }

    /// 清理过期统计信息
    pub async fn cleanup_expired_stats(&self) {
        let cutoff_time = chrono::Utc::now() - chrono::Duration::hours(self.config.stats_retention_hours as i64);
        let mut stats = self.query_stats.write().await;
        
        stats.retain(|_, stat| stat.last_executed > cutoff_time);
        
        info!("清理过期查询统计信息，保留 {} 条记录", stats.len());
    }

    /// 生成查询名称哈希
    fn hash_query_name(&self, query_name: &str) -> String {
        use std::collections::hash_map::DefaultHasher;
        use std::hash::{Hash, Hasher};
        
        let mut hasher = DefaultHasher::new();
        query_name.hash(&mut hasher);
        format!("{:x}", hasher.finish())
    }
}

/// 连接池健康状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConnectionPoolHealth {
    pub total_connections: u32,
    pub idle_connections: usize,
    pub active_connections: usize,
    pub utilization_percent: f64,
    pub is_healthy: bool,
}

/// 地理位置查询优化
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GeoQueryOptimization {
    pub table_name: String,
    pub optimization_type: GeoOptimizationType,
    pub description: String,
    pub sql_command: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum GeoOptimizationType {
    MissingSpatialIndex,
    SuboptimalQuery,
    LargeRadius,
    TooManyResults,
}

/// 查询缓存（简单实现）
#[derive(Clone)]
pub struct QueryCache {
    cache: Arc<RwLock<HashMap<String, CacheEntry>>>,
    max_size: usize,
}

#[derive(Debug, Clone)]
struct CacheEntry {
    data: String,
    created_at: Instant,
    ttl: Duration,
    hit_count: u64,
}

impl QueryCache {
    pub fn new(max_size: usize) -> Self {
        Self {
            cache: Arc::new(RwLock::new(HashMap::new())),
            max_size,
        }
    }

    pub async fn get(&self, key: &str) -> Option<String> {
        let mut cache = self.cache.write().await;
        
        if let Some(entry) = cache.get_mut(key) {
            if entry.created_at.elapsed() < entry.ttl {
                entry.hit_count += 1;
                return Some(entry.data.clone());
            } else {
                cache.remove(key);
            }
        }
        
        None
    }

    pub async fn set(&self, key: String, value: String, ttl: Duration) {
        let mut cache = self.cache.write().await;
        
        // 如果缓存已满，移除最旧的条目
        if cache.len() >= self.max_size {
            if let Some(oldest_key) = cache.keys().next().cloned() {
                cache.remove(&oldest_key);
            }
        }
        
        cache.insert(key, CacheEntry {
            data: value,
            created_at: Instant::now(),
            ttl,
            hit_count: 0,
        });
    }

    pub async fn clear(&self) {
        self.cache.write().await.clear();
    }

    pub async fn size(&self) -> usize {
        self.cache.read().await.len()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_query_cache() {
        let cache = QueryCache::new(10);
        
        // 测试设置和获取
        cache.set("key1".to_string(), "value1".to_string(), Duration::from_secs(60)).await;
        assert_eq!(cache.get("key1").await, Some("value1".to_string()));
        
        // 测试过期
        cache.set("key2".to_string(), "value2".to_string(), Duration::from_millis(1)).await;
        tokio::time::sleep(Duration::from_millis(10)).await;
        assert_eq!(cache.get("key2").await, None);
    }
}
