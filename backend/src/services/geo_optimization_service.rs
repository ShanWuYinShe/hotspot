/// 地理位置查询优化服务
/// 
/// 提供高性能的地理位置查询优化，包括空间索引管理、查询缓存和性能监控
use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tracing::{debug, info, warn};

use crate::domain::{DomainResult, DomainError};
use crate::domain::errors::LocationDomainError;

/// 地理位置查询优化配置
#[derive(Debug, Clone)]
pub struct GeoOptimizationConfig {
    /// 启用空间索引优化
    pub enable_spatial_index: bool,
    /// 启用查询缓存
    pub enable_query_cache: bool,
    /// 缓存TTL（秒）
    pub cache_ttl_seconds: u64,
    /// 最大缓存大小
    pub max_cache_size: usize,
    /// 慢查询阈值（毫秒）
    pub slow_query_threshold_ms: u64,
    /// 最大查询半径（米）
    pub max_query_radius_meters: f64,
    /// 默认查询半径（米）
    pub default_query_radius_meters: f64,
}

impl Default for GeoOptimizationConfig {
    fn default() -> Self {
        Self {
            enable_spatial_index: true,
            enable_query_cache: true,
            cache_ttl_seconds: 300, // 5分钟
            max_cache_size: 1000,
            slow_query_threshold_ms: 100,
            max_query_radius_meters: 50000.0, // 50公里
            default_query_radius_meters: 5000.0, // 5公里
        }
    }
}

/// 地理位置查询参数
#[derive(Debug, Clone, PartialEq)]
pub struct GeoQueryParams {
    pub latitude: f64,
    pub longitude: f64,
    pub radius_meters: f64,
    pub limit: i32,
    pub cursor: i64,
}

impl GeoQueryParams {
    /// 创建新的地理查询参数
    pub fn new(latitude: f64, longitude: f64, radius_meters: f64, limit: i32, cursor: i64) -> DomainResult<Self> {
        // 验证纬度范围
        if latitude < -90.0 || latitude > 90.0 {
            return Err(DomainError::InvalidCoordinates);
        }

        // 验证经度范围
        if longitude < -180.0 || longitude > 180.0 {
            return Err(DomainError::InvalidCoordinates);
        }

        // 验证半径
        if radius_meters <= 0.0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "Radius must be greater than 0".to_string(),
            });
        }

        // 验证限制
        if limit <= 0 || limit > 100 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "Limit must be between 1 and 100".to_string(),
            });
        }

        Ok(Self {
            latitude,
            longitude,
            radius_meters,
            limit,
            cursor,
        })
    }

    /// 生成缓存键
    pub fn cache_key(&self, query_type: &str) -> String {
        format!(
            "geo:{}:{:.6}:{:.6}:{:.0}:{}:{}",
            query_type, self.latitude, self.longitude, self.radius_meters, self.limit, self.cursor
        )
    }

    /// 检查是否在合理范围内
    pub fn is_reasonable_radius(&self, max_radius: f64) -> bool {
        self.radius_meters <= max_radius
    }

    /// 获取边界框（用于空间索引优化）
    pub fn bounding_box(&self) -> BoundingBox {
        // 地球半径（米）
        const EARTH_RADIUS: f64 = 6371000.0;
        
        // 计算纬度偏移
        let lat_offset = (self.radius_meters / EARTH_RADIUS) * (180.0 / std::f64::consts::PI);
        
        // 计算经度偏移（考虑纬度影响）
        let lng_offset = lat_offset / self.latitude.to_radians().cos();

        BoundingBox {
            min_lat: self.latitude - lat_offset,
            max_lat: self.latitude + lat_offset,
            min_lng: self.longitude - lng_offset,
            max_lng: self.longitude + lng_offset,
        }
    }
}

/// 地理边界框
#[derive(Debug, Clone)]
pub struct BoundingBox {
    pub min_lat: f64,
    pub max_lat: f64,
    pub min_lng: f64,
    pub max_lng: f64,
}

impl BoundingBox {
    /// 生成SQL WHERE条件
    pub fn to_sql_condition(&self, lat_column: &str, lng_column: &str) -> String {
        format!(
            "{} BETWEEN {} AND {} AND {} BETWEEN {} AND {}",
            lat_column, self.min_lat, self.max_lat,
            lng_column, self.min_lng, self.max_lng
        )
    }
}

/// 地理查询性能指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GeoQueryMetrics {
    pub query_type: String,
    pub execution_time_ms: u64,
    pub result_count: usize,
    pub cache_hit: bool,
    pub used_spatial_index: bool,
    pub radius_meters: f64,
    pub timestamp: chrono::DateTime<chrono::Utc>,
}

/// 地理查询统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GeoQueryStats {
    pub total_queries: u64,
    pub cache_hits: u64,
    pub cache_misses: u64,
    pub avg_execution_time_ms: f64,
    pub slow_queries: u64,
    pub spatial_index_usage: u64,
    pub most_common_radius: f64,
    pub query_distribution: HashMap<String, u64>,
}

impl GeoQueryStats {
    pub fn new() -> Self {
        Self {
            total_queries: 0,
            cache_hits: 0,
            cache_misses: 0,
            avg_execution_time_ms: 0.0,
            slow_queries: 0,
            spatial_index_usage: 0,
            most_common_radius: 0.0,
            query_distribution: HashMap::new(),
        }
    }

    pub fn cache_hit_rate(&self) -> f64 {
        if self.total_queries == 0 {
            0.0
        } else {
            self.cache_hits as f64 / self.total_queries as f64
        }
    }

    pub fn spatial_index_usage_rate(&self) -> f64 {
        if self.total_queries == 0 {
            0.0
        } else {
            self.spatial_index_usage as f64 / self.total_queries as f64
        }
    }
}

/// 地理位置查询优化服务接口
#[async_trait]
pub trait GeoOptimizationService: Send + Sync {
    /// 优化地理查询参数
    async fn optimize_query_params(&self, params: &GeoQueryParams) -> DomainResult<GeoQueryParams>;
    
    /// 记录查询指标
    async fn record_query_metrics(&self, metrics: GeoQueryMetrics) -> DomainResult<()>;
    
    /// 获取查询统计
    async fn get_query_stats(&self) -> DomainResult<GeoQueryStats>;
    
    /// 建议最优查询半径
    async fn suggest_optimal_radius(&self, lat: f64, lng: f64, expected_results: usize) -> DomainResult<f64>;
    
    /// 检查空间索引状态
    async fn check_spatial_index_status(&self) -> DomainResult<SpatialIndexStatus>;
    
    /// 清理过期缓存
    async fn cleanup_expired_cache(&self) -> DomainResult<u64>;
}

/// 空间索引状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SpatialIndexStatus {
    pub table_name: String,
    pub index_name: String,
    pub index_exists: bool,
    pub index_size_mb: f64,
    pub last_analyzed: Option<chrono::DateTime<chrono::Utc>>,
    pub estimated_rows: u64,
}

/// 默认地理位置查询优化服务实现
pub struct DefaultGeoOptimizationService {
    config: GeoOptimizationConfig,
    stats: Arc<tokio::sync::RwLock<GeoQueryStats>>,
    query_cache: Arc<tokio::sync::RwLock<HashMap<String, (Instant, Vec<u8>)>>>,
}

impl DefaultGeoOptimizationService {
    pub fn new(config: GeoOptimizationConfig) -> Self {
        Self {
            config,
            stats: Arc::new(tokio::sync::RwLock::new(GeoQueryStats::new())),
            query_cache: Arc::new(tokio::sync::RwLock::new(HashMap::new())),
        }
    }

    /// 清理过期缓存项
    async fn cleanup_cache_internal(&self) -> u64 {
        let mut cache = self.query_cache.write().await;
        let now = Instant::now();
        let ttl = Duration::from_secs(self.config.cache_ttl_seconds);
        
        let initial_size = cache.len();
        cache.retain(|_, (timestamp, _)| now.duration_since(*timestamp) < ttl);
        let final_size = cache.len();
        
        (initial_size - final_size) as u64
    }

    /// 检查缓存大小并清理
    async fn ensure_cache_size(&self) {
        let cache_size = {
            let cache = self.query_cache.read().await;
            cache.len()
        };

        if cache_size > self.config.max_cache_size {
            // 清理过期项
            self.cleanup_cache_internal().await;
            
            // 如果还是太大，清理最老的项
            let mut cache = self.query_cache.write().await;
            if cache.len() > self.config.max_cache_size {
                let excess = cache.len() - self.config.max_cache_size;
                let mut entries: Vec<_> = cache.iter().map(|(k, (t, _))| (k.clone(), *t)).collect();
                entries.sort_by_key(|(_, timestamp)| *timestamp);
                
                for (key, _) in entries.iter().take(excess) {
                    cache.remove(key);
                }
                
                debug!(
                    removed = excess,
                    remaining = cache.len(),
                    "Cleaned up excess cache entries"
                );
            }
        }
    }
}

#[async_trait]
impl GeoOptimizationService for DefaultGeoOptimizationService {
    async fn optimize_query_params(&self, params: &GeoQueryParams) -> DomainResult<GeoQueryParams> {
        let mut optimized = params.clone();

        // 限制最大查询半径
        if optimized.radius_meters > self.config.max_query_radius_meters {
            warn!(
                original_radius = optimized.radius_meters,
                max_radius = self.config.max_query_radius_meters,
                "Query radius exceeds maximum, capping to maximum allowed"
            );
            optimized.radius_meters = self.config.max_query_radius_meters;
        }

        // 调整查询限制以提高性能
        if optimized.limit > 50 && optimized.radius_meters > 10000.0 {
            warn!(
                original_limit = optimized.limit,
                radius = optimized.radius_meters,
                "Large radius with high limit may cause performance issues"
            );
        }

        debug!(
            original_radius = params.radius_meters,
            optimized_radius = optimized.radius_meters,
            limit = optimized.limit,
            "Query parameters optimized"
        );

        Ok(optimized)
    }

    async fn record_query_metrics(&self, metrics: GeoQueryMetrics) -> DomainResult<()> {
        let mut stats = self.stats.write().await;
        
        stats.total_queries += 1;
        
        if metrics.cache_hit {
            stats.cache_hits += 1;
        } else {
            stats.cache_misses += 1;
        }

        if metrics.used_spatial_index {
            stats.spatial_index_usage += 1;
        }

        if metrics.execution_time_ms > self.config.slow_query_threshold_ms {
            stats.slow_queries += 1;
            warn!(
                query_type = %metrics.query_type,
                execution_time_ms = metrics.execution_time_ms,
                threshold_ms = self.config.slow_query_threshold_ms,
                "Slow geo query detected"
            );
        }

        // 更新平均执行时间
        let total_time = stats.avg_execution_time_ms * (stats.total_queries - 1) as f64;
        stats.avg_execution_time_ms = (total_time + metrics.execution_time_ms as f64) / stats.total_queries as f64;

        // 更新查询类型分布
        *stats.query_distribution.entry(metrics.query_type.clone()).or_insert(0) += 1;

        debug!(
            query_type = %metrics.query_type,
            execution_time_ms = metrics.execution_time_ms,
            result_count = metrics.result_count,
            cache_hit = metrics.cache_hit,
            "Geo query metrics recorded"
        );

        Ok(())
    }

    async fn get_query_stats(&self) -> DomainResult<GeoQueryStats> {
        let stats = self.stats.read().await;
        Ok(stats.clone())
    }

    async fn suggest_optimal_radius(&self, lat: f64, lng: f64, expected_results: usize) -> DomainResult<f64> {
        // 基于历史数据和预期结果数量建议最优半径
        let stats = self.stats.read().await;
        
        // 如果没有历史数据，返回默认半径
        if stats.total_queries == 0 {
            return Ok(self.config.default_query_radius_meters);
        }

        // 基于预期结果数量调整半径
        let base_radius = if expected_results <= 10 {
            1000.0 // 1公里
        } else if expected_results <= 50 {
            3000.0 // 3公里
        } else {
            5000.0 // 5公里
        };

        // 考虑地理位置密度（简化实现）
        let density_factor = if lat.abs() > 60.0 {
            // 高纬度地区，密度较低
            1.5
        } else if lat.abs() < 30.0 {
            // 低纬度地区，密度较高
            0.8
        } else {
            1.0
        };

        let suggested_radius: f64 = base_radius * density_factor;
        let capped_radius = suggested_radius.min(self.config.max_query_radius_meters);

        debug!(
            lat = lat,
            lng = lng,
            expected_results = expected_results,
            base_radius = base_radius,
            density_factor = density_factor,
            suggested_radius = capped_radius,
            "Optimal radius suggested"
        );

        Ok(capped_radius)
    }

    async fn check_spatial_index_status(&self) -> DomainResult<SpatialIndexStatus> {
        // 这里应该查询数据库获取实际的空间索引状态
        // 简化实现，返回模拟状态
        Ok(SpatialIndexStatus {
            table_name: "user_locations".to_string(),
            index_name: "idx_user_locations_geom".to_string(),
            index_exists: true,
            index_size_mb: 15.2,
            last_analyzed: Some(chrono::Utc::now() - chrono::Duration::hours(2)),
            estimated_rows: 10000,
        })
    }

    async fn cleanup_expired_cache(&self) -> DomainResult<u64> {
        self.ensure_cache_size().await;
        let cleaned = self.cleanup_cache_internal().await;
        
        info!(
            cleaned_entries = cleaned,
            "Geo query cache cleanup completed"
        );
        
        Ok(cleaned)
    }
}

/// 空的地理位置查询优化服务（用于禁用优化的场景）
pub struct NullGeoOptimizationService;

#[async_trait]
impl GeoOptimizationService for NullGeoOptimizationService {
    async fn optimize_query_params(&self, params: &GeoQueryParams) -> DomainResult<GeoQueryParams> {
        Ok(params.clone())
    }

    async fn record_query_metrics(&self, _metrics: GeoQueryMetrics) -> DomainResult<()> {
        Ok(())
    }

    async fn get_query_stats(&self) -> DomainResult<GeoQueryStats> {
        Ok(GeoQueryStats::new())
    }

    async fn suggest_optimal_radius(&self, _lat: f64, _lng: f64, _expected_results: usize) -> DomainResult<f64> {
        Ok(5000.0) // 默认5公里
    }

    async fn check_spatial_index_status(&self) -> DomainResult<SpatialIndexStatus> {
        Ok(SpatialIndexStatus {
            table_name: "disabled".to_string(),
            index_name: "disabled".to_string(),
            index_exists: false,
            index_size_mb: 0.0,
            last_analyzed: None,
            estimated_rows: 0,
        })
    }

    async fn cleanup_expired_cache(&self) -> DomainResult<u64> {
        Ok(0)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_geo_query_params_validation() {
        // 有效参数
        let params = GeoQueryParams::new(39.916527, 116.397128, 1000.0, 20, 0);
        assert!(params.is_ok());

        // 无效纬度
        let params = GeoQueryParams::new(91.0, 116.397128, 1000.0, 20, 0);
        assert!(params.is_err());

        // 无效经度
        let params = GeoQueryParams::new(39.916527, 181.0, 1000.0, 20, 0);
        assert!(params.is_err());

        // 无效半径
        let params = GeoQueryParams::new(39.916527, 116.397128, -1000.0, 20, 0);
        assert!(params.is_err());

        // 无效限制
        let params = GeoQueryParams::new(39.916527, 116.397128, 1000.0, 0, 0);
        assert!(params.is_err());
    }

    #[test]
    fn test_bounding_box_calculation() {
        let params = GeoQueryParams::new(39.916527, 116.397128, 1000.0, 20, 0).unwrap();
        let bbox = params.bounding_box();
        
        // 边界框应该围绕中心点
        assert!(bbox.min_lat < params.latitude);
        assert!(bbox.max_lat > params.latitude);
        assert!(bbox.min_lng < params.longitude);
        assert!(bbox.max_lng > params.longitude);
    }

    #[test]
    fn test_cache_key_generation() {
        let params = GeoQueryParams::new(39.916527, 116.397128, 1000.0, 20, 0).unwrap();
        let key = params.cache_key("users");
        
        assert!(key.contains("geo:users"));
        assert!(key.contains("39.916527"));
        assert!(key.contains("116.397128"));
        assert!(key.contains("1000"));
    }

    #[tokio::test]
    async fn test_geo_optimization_service() {
        let config = GeoOptimizationConfig::default();
        let service = DefaultGeoOptimizationService::new(config);
        
        // 测试参数优化
        let params = GeoQueryParams::new(39.916527, 116.397128, 100000.0, 20, 0).unwrap();
        let optimized = service.optimize_query_params(&params).await.unwrap();
        
        // 半径应该被限制
        assert!(optimized.radius_meters <= 50000.0);
        
        // 测试指标记录
        let metrics = GeoQueryMetrics {
            query_type: "test".to_string(),
            execution_time_ms: 50,
            result_count: 10,
            cache_hit: false,
            used_spatial_index: true,
            radius_meters: 1000.0,
            timestamp: chrono::Utc::now(),
        };
        
        service.record_query_metrics(metrics).await.unwrap();
        
        // 获取统计信息
        let stats = service.get_query_stats().await.unwrap();
        assert_eq!(stats.total_queries, 1);
        assert_eq!(stats.spatial_index_usage, 1);
    }
}
