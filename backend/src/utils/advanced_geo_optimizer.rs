/// 高级地理位置查询优化器
/// 
/// 提供智能的地理位置查询优化策略，包括：
/// - 边界框预过滤
/// - 索引提示
/// - 查询计划优化
/// - 缓存策略
use std::collections::HashMap;
use serde::{Serialize, Deserialize};
use sqlx::PgPool;
use tracing::{info, warn, debug};

/// 地理查询优化配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GeoQueryConfig {
    /// 是否启用边界框预过滤
    pub enable_bbox_filter: bool,
    /// 是否强制使用空间索引
    pub force_spatial_index: bool,
    /// 最大查询半径（公里）
    pub max_radius_km: f64,
    /// 是否启用查询缓存
    pub enable_query_cache: bool,
    /// 缓存TTL（秒）
    pub cache_ttl_seconds: u64,
}

impl Default for GeoQueryConfig {
    fn default() -> Self {
        Self {
            enable_bbox_filter: true,
            force_spatial_index: true,
            max_radius_km: 50.0,
            enable_query_cache: true,
            cache_ttl_seconds: 300, // 5分钟
        }
    }
}

/// 边界框
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BoundingBox {
    pub min_lat: f64,
    pub max_lat: f64,
    pub min_lng: f64,
    pub max_lng: f64,
}

impl BoundingBox {
    /// 根据中心点和半径计算边界框
    pub fn from_center_radius(lat: f64, lng: f64, radius_km: f64) -> Self {
        // 1度纬度约等于111公里
        let lat_delta = radius_km / 111.0;
        // 经度随纬度变化，使用余弦修正
        let lng_delta = radius_km / (111.0 * lat.to_radians().cos());
        
        Self {
            min_lat: lat - lat_delta,
            max_lat: lat + lat_delta,
            min_lng: lng - lng_delta,
            max_lng: lng + lng_delta,
        }
    }
    
    /// 检查点是否在边界框内
    pub fn contains(&self, lat: f64, lng: f64) -> bool {
        lat >= self.min_lat && lat <= self.max_lat &&
        lng >= self.min_lng && lng <= self.max_lng
    }
}

/// 查询性能统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueryPerformanceStats {
    pub query_type: String,
    pub execution_count: u64,
    pub total_duration_ms: u64,
    pub average_duration_ms: f64,
    pub min_duration_ms: u64,
    pub max_duration_ms: u64,
    pub cache_hit_count: u64,
    pub cache_miss_count: u64,
    pub last_executed_at: chrono::DateTime<chrono::Utc>,
}

/// 高级地理位置查询优化器
pub struct AdvancedGeoOptimizer {
    config: GeoQueryConfig,
    performance_stats: std::sync::Arc<std::sync::Mutex<HashMap<String, QueryPerformanceStats>>>,
    query_cache: std::sync::Arc<std::sync::Mutex<HashMap<String, (String, chrono::DateTime<chrono::Utc>)>>>,
}

impl AdvancedGeoOptimizer {
    /// 创建新的优化器
    pub fn new(config: GeoQueryConfig) -> Self {
        Self {
            config,
            performance_stats: std::sync::Arc::new(std::sync::Mutex::new(HashMap::new())),
            query_cache: std::sync::Arc::new(std::sync::Mutex::new(HashMap::new())),
        }
    }
    
    /// 优化附近用户查询
    pub fn optimize_nearby_users_query(
        &self,
        latitude: f64,
        longitude: f64,
        radius_km: f64,
        cursor: i64,
        limit: i32,
    ) -> OptimizedQuery {
        let start_time = std::time::Instant::now();
        
        // 检查半径限制
        let effective_radius = if radius_km > self.config.max_radius_km {
            warn!("查询半径 {}km 超过最大限制 {}km，已调整", radius_km, self.config.max_radius_km);
            self.config.max_radius_km
        } else {
            radius_km
        };
        
        // 计算边界框
        let bbox = BoundingBox::from_center_radius(latitude, longitude, effective_radius);
        
        // 构建优化的SQL查询
        let sql = if self.config.enable_bbox_filter {
            self.build_bbox_optimized_user_query(&bbox, effective_radius)
        } else {
            self.build_standard_user_query(effective_radius)
        };
        
        // 生成查询提示
        let hints = self.generate_query_hints("nearby_users", effective_radius, limit);
        
        let duration = start_time.elapsed();
        self.record_query_performance("nearby_users_optimization", duration);
        
        OptimizedQuery {
            sql,
            hints,
            bbox: Some(bbox),
            estimated_cost: self.estimate_query_cost(effective_radius, limit),
            optimization_notes: vec![
                format!("使用边界框预过滤: {}", self.config.enable_bbox_filter),
                format!("有效查询半径: {:.2}km", effective_radius),
                format!("预计返回结果数: {}", self.estimate_result_count(effective_radius)),
            ],
        }
    }
    
    /// 优化附近群组查询
    pub fn optimize_nearby_groups_query(
        &self,
        latitude: f64,
        longitude: f64,
        radius_km: f64,
        cursor: i64,
        limit: i32,
    ) -> OptimizedQuery {
        let start_time = std::time::Instant::now();
        
        let effective_radius = radius_km.min(self.config.max_radius_km);
        let bbox = BoundingBox::from_center_radius(latitude, longitude, effective_radius);
        
        let sql = if self.config.enable_bbox_filter {
            self.build_bbox_optimized_group_query(&bbox, effective_radius)
        } else {
            self.build_standard_group_query(effective_radius)
        };
        
        let hints = self.generate_query_hints("nearby_groups", effective_radius, limit);
        
        let duration = start_time.elapsed();
        self.record_query_performance("nearby_groups_optimization", duration);
        
        OptimizedQuery {
            sql,
            hints,
            bbox: Some(bbox),
            estimated_cost: self.estimate_query_cost(effective_radius, limit),
            optimization_notes: vec![
                format!("群组查询优化: 半径 {:.2}km", effective_radius),
                format!("边界框过滤: ({:.6}, {:.6}) - ({:.6}, {:.6})", 
                    bbox.min_lat, bbox.min_lng, bbox.max_lat, bbox.max_lng),
            ],
        }
    }
    
    /// 构建边界框优化的用户查询
    fn build_bbox_optimized_user_query(&self, bbox: &BoundingBox, radius_km: f64) -> String {
        format!(r#"
            SELECT u.id, u.user_id, u.login_id, u.nickname, u.password_hash, u.expires_at,
                   u.email, u.avatar_url, u.created_at, u.updated_at, u.last_active_at,
                   ST_Distance(
                       ST_Point(COALESCE(ul.longitude, 0.0), COALESCE(ul.latitude, 0.0))::geography,
                       ST_Point($2, $1)::geography
                   ) / 1000.0 as distance_km
            FROM users u
            LEFT JOIN user_locations ul ON u.user_id = ul.user_id
            WHERE ul.latitude IS NOT NULL
              AND ul.longitude IS NOT NULL
              AND ul.latitude BETWEEN {} AND {}
              AND ul.longitude BETWEEN {} AND {}
              AND ($4 = 0 OR u.user_id < $4)
              AND ST_DWithin(
                  ST_Point(COALESCE(ul.longitude, 0.0), COALESCE(ul.latitude, 0.0))::geography,
                  ST_Point($2, $1)::geography,
                  $3 * 1000
              )
            ORDER BY u.user_id DESC
            LIMIT $5
        "#, bbox.min_lat, bbox.max_lat, bbox.min_lng, bbox.max_lng)
    }
    
    /// 构建标准用户查询
    fn build_standard_user_query(&self, radius_km: f64) -> String {
        r#"
            SELECT u.id, u.user_id, u.login_id, u.nickname, u.password_hash, u.expires_at,
                   u.email, u.avatar_url, u.created_at, u.updated_at, u.last_active_at,
                   ST_Distance(
                       ST_Point(COALESCE(ul.longitude, 0.0), COALESCE(ul.latitude, 0.0))::geography,
                       ST_Point($2, $1)::geography
                   ) / 1000.0 as distance_km
            FROM users u
            LEFT JOIN user_locations ul ON u.user_id = ul.user_id
            WHERE ul.latitude IS NOT NULL
              AND ul.longitude IS NOT NULL
              AND ($4 = 0 OR u.user_id < $4)
              AND ST_DWithin(
                  ST_Point(COALESCE(ul.longitude, 0.0), COALESCE(ul.latitude, 0.0))::geography,
                  ST_Point($2, $1)::geography,
                  $3 * 1000
              )
            ORDER BY u.user_id DESC
            LIMIT $5
        "#.to_string()
    }
    
    /// 构建边界框优化的群组查询
    fn build_bbox_optimized_group_query(&self, bbox: &BoundingBox, radius_km: f64) -> String {
        format!(r#"
            SELECT
                id as "id!",
                group_id as "group_id!",
                name as "name!",
                description,
                owner_id as "owner_id!",
                latitude as "latitude!",
                longitude as "longitude!",
                location_name as "location_name!",
                password_hash,
                created_at as "created_at!",
                updated_at as "updated_at!",
                last_active_at as "last_active_at!",
                ST_Distance(
                    geom,
                    ST_Point($2, $1)::geography
                ) / 1000.0 as distance_km
            FROM groups
            WHERE latitude BETWEEN {} AND {}
              AND longitude BETWEEN {} AND {}
              AND ($4::bigint = 0 OR id < $4::bigint)
              AND ST_DWithin(
                    geom,
                    ST_Point($2, $1)::geography,
                    $3 * 1000
              )
            ORDER BY id DESC
            LIMIT $5::bigint
        "#, bbox.min_lat, bbox.max_lat, bbox.min_lng, bbox.max_lng)
    }
    
    /// 构建标准群组查询
    fn build_standard_group_query(&self, radius_km: f64) -> String {
        r#"
            SELECT
                id as "id!",
                group_id as "group_id!",
                name as "name!",
                description,
                owner_id as "owner_id!",
                latitude as "latitude!",
                longitude as "longitude!",
                location_name as "location_name!",
                password_hash,
                created_at as "created_at!",
                updated_at as "updated_at!",
                last_active_at as "last_active_at!",
                ST_Distance(
                    geom,
                    ST_Point($2, $1)::geography
                ) / 1000.0 as distance_km
            FROM groups
            WHERE ($4::bigint = 0 OR id < $4::bigint)
              AND ST_DWithin(
                    geom,
                    ST_Point($2, $1)::geography,
                    $3 * 1000
              )
            ORDER BY id DESC
            LIMIT $5::bigint
        "#.to_string()
    }
    
    /// 生成查询提示
    fn generate_query_hints(&self, query_type: &str, radius_km: f64, limit: i32) -> Vec<String> {
        let mut hints = Vec::new();
        
        if self.config.force_spatial_index {
            hints.push("/*+ USE_INDEX(spatial_index) */".to_string());
        }
        
        if radius_km > 10.0 {
            hints.push("/*+ PARALLEL(4) */".to_string());
        }
        
        if limit > 100 {
            hints.push("/*+ FIRST_ROWS */".to_string());
        } else {
            hints.push("/*+ ALL_ROWS */".to_string());
        }
        
        hints.push(format!("-- Query type: {}, radius: {:.2}km, limit: {}", 
                          query_type, radius_km, limit));
        
        hints
    }
    
    /// 估算查询成本
    fn estimate_query_cost(&self, radius_km: f64, limit: i32) -> f64 {
        // 简化的成本估算模型
        let area_factor = (radius_km * radius_km * std::f64::consts::PI) / 100.0; // 标准化面积
        let limit_factor = (limit as f64).log10();
        let bbox_discount = if self.config.enable_bbox_filter { 0.7 } else { 1.0 };
        
        area_factor * limit_factor * bbox_discount
    }
    
    /// 估算结果数量
    fn estimate_result_count(&self, radius_km: f64) -> u32 {
        // 基于半径的简化估算
        let density_per_km2 = 10.0; // 假设每平方公里10个用户
        let area = radius_km * radius_km * std::f64::consts::PI;
        (area * density_per_km2) as u32
    }
    
    /// 记录查询性能
    fn record_query_performance(&self, query_type: &str, duration: std::time::Duration) {
        let duration_ms = duration.as_millis() as u64;
        let mut stats = self.performance_stats.lock().unwrap();
        
        let stat = stats.entry(query_type.to_string()).or_insert_with(|| QueryPerformanceStats {
            query_type: query_type.to_string(),
            execution_count: 0,
            total_duration_ms: 0,
            average_duration_ms: 0.0,
            min_duration_ms: u64::MAX,
            max_duration_ms: 0,
            cache_hit_count: 0,
            cache_miss_count: 0,
            last_executed_at: chrono::Utc::now(),
        });
        
        stat.execution_count += 1;
        stat.total_duration_ms += duration_ms;
        stat.average_duration_ms = stat.total_duration_ms as f64 / stat.execution_count as f64;
        stat.min_duration_ms = stat.min_duration_ms.min(duration_ms);
        stat.max_duration_ms = stat.max_duration_ms.max(duration_ms);
        stat.last_executed_at = chrono::Utc::now();
        
        if duration_ms > 1000 {
            warn!("慢查询检测: {} 耗时 {}ms", query_type, duration_ms);
        } else {
            debug!("查询性能: {} 耗时 {}ms", query_type, duration_ms);
        }
    }
    
    /// 获取性能统计
    pub fn get_performance_stats(&self) -> HashMap<String, QueryPerformanceStats> {
        self.performance_stats.lock().unwrap().clone()
    }
    
    /// 清理过期缓存
    pub fn cleanup_expired_cache(&self) {
        let mut cache = self.query_cache.lock().unwrap();
        let now = chrono::Utc::now();
        let ttl = chrono::Duration::seconds(self.config.cache_ttl_seconds as i64);
        
        cache.retain(|_, (_, timestamp)| now.signed_duration_since(*timestamp) < ttl);
    }
    
    /// 分析查询计划
    pub async fn analyze_query_plan(&self, pool: &PgPool, sql: &str) -> Result<QueryPlanAnalysis, sqlx::Error> {
        let explain_sql = format!("EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) {}", sql);
        
        let row = sqlx::query_scalar::<_, serde_json::Value>(&explain_sql)
            .fetch_one(pool)
            .await?;
        
        Ok(QueryPlanAnalysis {
            plan_json: row,
            recommendations: self.generate_plan_recommendations(&row),
        })
    }
    
    /// 生成查询计划建议
    fn generate_plan_recommendations(&self, plan: &serde_json::Value) -> Vec<String> {
        let mut recommendations = Vec::new();
        
        // 检查是否使用了空间索引
        if let Some(plan_array) = plan.as_array() {
            if let Some(plan_obj) = plan_array.get(0) {
                if let Some(execution_time) = plan_obj.get("Execution Time") {
                    if let Some(time) = execution_time.as_f64() {
                        if time > 100.0 {
                            recommendations.push("查询执行时间较长，建议优化索引策略".to_string());
                        }
                    }
                }
                
                // 检查是否使用了顺序扫描
                if plan_obj.to_string().contains("Seq Scan") {
                    recommendations.push("检测到顺序扫描，建议添加适当的索引".to_string());
                }
                
                // 检查是否使用了空间索引
                if !plan_obj.to_string().contains("Index Scan") && !plan_obj.to_string().contains("Bitmap") {
                    recommendations.push("建议使用空间索引以提高地理位置查询性能".to_string());
                }
            }
        }
        
        if recommendations.is_empty() {
            recommendations.push("查询计划看起来不错，无需特别优化".to_string());
        }
        
        recommendations
    }
}

/// 优化后的查询结果
#[derive(Debug, Clone)]
pub struct OptimizedQuery {
    pub sql: String,
    pub hints: Vec<String>,
    pub bbox: Option<BoundingBox>,
    pub estimated_cost: f64,
    pub optimization_notes: Vec<String>,
}

/// 查询计划分析结果
#[derive(Debug, Clone)]
pub struct QueryPlanAnalysis {
    pub plan_json: serde_json::Value,
    pub recommendations: Vec<String>,
}

/// 全局优化器实例
use std::sync::OnceLock;
static GLOBAL_GEO_OPTIMIZER: OnceLock<AdvancedGeoOptimizer> = OnceLock::new();

/// 获取全局地理位置优化器
pub fn get_geo_optimizer() -> &'static AdvancedGeoOptimizer {
    GLOBAL_GEO_OPTIMIZER.get_or_init(|| {
        AdvancedGeoOptimizer::new(GeoQueryConfig::default())
    })
}

/// 优化器配置宏
#[macro_export]
macro_rules! optimize_geo_query {
    ($optimizer:expr, $query_type:expr, $lat:expr, $lng:expr, $radius:expr, $cursor:expr, $limit:expr) => {
        match $query_type {
            "nearby_users" => $optimizer.optimize_nearby_users_query($lat, $lng, $radius, $cursor, $limit),
            "nearby_groups" => $optimizer.optimize_nearby_groups_query($lat, $lng, $radius, $cursor, $limit),
            _ => panic!("不支持的查询类型: {}", $query_type),
        }
    };
}
