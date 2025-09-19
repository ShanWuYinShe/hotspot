/// PostgreSQL查询优化器
/// 
/// 提供统一的游标分页、空间查询优化和性能监控功能
use crate::domain::{DomainResult, DomainError};
use sqlx::PgPool;
use std::collections::HashMap;
use serde::{Serialize, Deserialize};

/// 游标分页参数
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CursorPagination {
    /// 游标值（上一页最后一条记录的ID）
    pub cursor: Option<i64>,
    /// 每页记录数（1-100）
    pub limit: i32,
    /// 排序方向
    pub order: PaginationOrder,
}

/// 分页排序方向
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum PaginationOrder {
    /// 升序（ID递增）
    Ascending,
    /// 降序（ID递减，默认）
    Descending,
}

impl Default for CursorPagination {
    fn default() -> Self {
        Self {
            cursor: None,
            limit: 20,
            order: PaginationOrder::Descending,
        }
    }
}

impl CursorPagination {
    /// 创建新的分页参数
    pub fn new(cursor: Option<i64>, limit: Option<i32>) -> Self {
        Self {
            cursor,
            limit: limit.unwrap_or(20).clamp(1, 100),
            order: PaginationOrder::Descending,
        }
    }
    
    /// 设置排序方向
    pub fn with_order(mut self, order: PaginationOrder) -> Self {
        self.order = order;
        self
    }
    
    /// 获取SQL WHERE条件
    pub fn get_where_condition(&self) -> String {
        match (&self.cursor, &self.order) {
            (Some(_cursor), PaginationOrder::Descending) => format!("($1 = 0 OR id < $1)"),
            (Some(_cursor), PaginationOrder::Ascending) => format!("($1 = 0 OR id > $1)"),
            (None, _) => "TRUE".to_string(),
        }
    }
    
    /// 获取SQL ORDER BY子句
    pub fn get_order_clause(&self) -> String {
        match self.order {
            PaginationOrder::Descending => "ORDER BY id DESC".to_string(),
            PaginationOrder::Ascending => "ORDER BY id ASC".to_string(),
        }
    }
    
    /// 获取游标值（用于参数绑定）
    pub fn get_cursor_value(&self) -> i64 {
        self.cursor.unwrap_or(0)
    }
    
    /// 获取LIMIT值（查询limit+1条以判断是否有更多数据）
    pub fn get_limit_value(&self) -> i64 {
        (self.limit + 1) as i64
    }
}

/// 分页查询结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaginatedQueryResult<T> {
    /// 查询结果
    pub items: Vec<T>,
    /// 是否还有更多数据
    pub has_more: bool,
    /// 下一页游标
    pub next_cursor: Option<i64>,
}

impl<T> PaginatedQueryResult<T> {
    /// 从查询结果创建分页结果
    pub fn from_query_result(mut items: Vec<T>, pagination: &CursorPagination, get_id: impl Fn(&T) -> i64) -> Self {
        let has_more = items.len() > pagination.limit as usize;
        
        // 移除多查询的一条记录
        if has_more {
            items.pop();
        }
        
        // 获取下一页游标
        let next_cursor = if has_more && !items.is_empty() {
            Some(get_id(items.last().unwrap()))
        } else {
            None
        };
        
        Self {
            items,
            has_more,
            next_cursor,
        }
    }
}

/// 地理位置查询参数
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GeoQueryParams {
    /// 中心点纬度
    pub latitude: f64,
    /// 中心点经度
    pub longitude: f64,
    /// 搜索半径（米）
    pub radius_meters: f64,
    /// 分页参数
    pub pagination: CursorPagination,
}

impl GeoQueryParams {
    /// 创建新的地理位置查询参数
    pub fn new(latitude: f64, longitude: f64, radius_meters: f64) -> DomainResult<Self> {
        // 验证坐标范围
        if latitude < -90.0 || latitude > 90.0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("Invalid latitude: {}, must be between -90 and 90", latitude),
            });
        }
        
        if longitude < -180.0 || longitude > 180.0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("Invalid longitude: {}, must be between -180 and 180", longitude),
            });
        }
        
        // 验证半径范围
        if radius_meters <= 0.0 || radius_meters > 100_000.0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("Invalid radius: {}, must be between 0 and 100000 meters", radius_meters),
            });
        }
        
        Ok(Self {
            latitude,
            longitude,
            radius_meters,
            pagination: CursorPagination::default(),
        })
    }
    
    /// 设置分页参数
    pub fn with_pagination(mut self, pagination: CursorPagination) -> Self {
        self.pagination = pagination;
        self
    }
    
    /// 获取PostGIS地理位置查询的WHERE条件
    pub fn get_geo_where_condition(&self) -> String {
        format!(
            "ST_DWithin(geom::geography, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, $4) AND {}",
            self.pagination.get_where_condition()
        )
    }
    
    /// 获取距离计算的SELECT子句
    pub fn get_distance_select(&self) -> String {
        "ST_Distance(geom::geography, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography) as \"distance!\"".to_string()
    }
}

/// 查询性能监控
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueryPerformanceMetrics {
    /// 查询类型
    pub query_type: String,
    /// 执行时间（毫秒）
    pub execution_time_ms: u64,
    /// 返回记录数
    pub result_count: usize,
    /// 是否使用了索引
    pub used_index: bool,
    /// 查询参数摘要
    pub params_summary: HashMap<String, String>,
}

/// PostgreSQL查询优化器
#[derive(Clone)]
pub struct PgQueryOptimizer {
    pool: PgPool,
    /// 是否启用性能监控
    performance_monitoring: bool,
}

impl PgQueryOptimizer {
    /// 创建新的查询优化器
    pub fn new(pool: PgPool) -> Self {
        Self {
            pool,
            performance_monitoring: true,
        }
    }
    
    /// 启用/禁用性能监控
    pub fn with_performance_monitoring(mut self, enabled: bool) -> Self {
        self.performance_monitoring = enabled;
        self
    }
    
    /// 执行带性能监控的查询
    pub async fn execute_with_monitoring<F, T>(
        &self,
        query_type: &str,
        params_summary: HashMap<String, String>,
        query_fn: F,
    ) -> DomainResult<(T, Option<QueryPerformanceMetrics>)>
    where
        F: std::future::Future<Output = DomainResult<T>>,
    {
        let start_time = std::time::Instant::now();
        
        let result = query_fn.await?;
        
        let metrics = if self.performance_monitoring {
            let execution_time = start_time.elapsed();
            Some(QueryPerformanceMetrics {
                query_type: query_type.to_string(),
                execution_time_ms: execution_time.as_millis() as u64,
                result_count: 0, // 需要在具体查询中设置
                used_index: true, // 需要通过EXPLAIN分析
                params_summary,
            })
        } else {
            None
        };
        
        Ok((result, metrics))
    }
    
    /// 分析查询计划
    pub async fn analyze_query_plan(&self, sql: &str, _params: &[&str]) -> DomainResult<QueryPlan> {
        let explain_sql = format!("EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) {}", sql);
        
        let row = sqlx::query_scalar::<_, serde_json::Value>(&explain_sql)
            .fetch_one(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Failed to analyze query plan: {}", e),
            })?;
        
        Ok(QueryPlan::from_json(row))
    }
    
    /// 检查空间索引使用情况
    pub async fn check_spatial_index_usage(&self, table_name: &str) -> DomainResult<SpatialIndexInfo> {
        let rows = sqlx::query!(
            r#"
            SELECT 
                schemaname as "schema_name!",
                tablename as "table_name!",
                indexname as "index_name!",
                indexdef as "index_definition!"
            FROM pg_indexes 
            WHERE tablename = $1 
            AND indexdef LIKE '%USING gist%'
            AND indexdef LIKE '%geom%'
            "#,
            table_name
        )
            .fetch_all(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Failed to check spatial indexes: {}", e),
            })?;
        
        let indexes: Vec<SpatialIndex> = rows.into_iter().map(|row| SpatialIndex {
            schema_name: row.schema_name,
            table_name: row.table_name,
            index_name: row.index_name,
            definition: row.index_definition,
        }).collect();
        
        let has_spatial_index = !indexes.is_empty();
        Ok(SpatialIndexInfo {
            table_name: table_name.to_string(),
            indexes,
            has_spatial_index,
        })
    }
    
    /// 优化地理位置查询
    pub async fn optimize_geo_query_performance(&self) -> DomainResult<Vec<OptimizationSuggestion>> {
        let mut suggestions = Vec::new();
        
        // 检查主要表的空间索引
        let tables = vec!["users", "groups", "moments", "messages", "conversations"];
        
        for table in tables {
            let index_info = self.check_spatial_index_usage(table).await?;
            
            if !index_info.has_spatial_index {
                suggestions.push(OptimizationSuggestion {
                    category: "Spatial Index".to_string(),
                    priority: OptimizationPriority::High,
                    description: format!("Table '{}' missing spatial index on geom column", table),
                    sql_command: Some(format!(
                        "CREATE INDEX CONCURRENTLY idx_{}_geom_gist ON {} USING GIST (geom);",
                        table, table
                    )),
                });
            }
        }
        
        // 检查统计信息更新
        suggestions.push(OptimizationSuggestion {
            category: "Statistics".to_string(),
            priority: OptimizationPriority::Medium,
            description: "Update table statistics for better query planning".to_string(),
            sql_command: Some("ANALYZE;".to_string()),
        });
        
        Ok(suggestions)
    }
}

/// 查询计划分析结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueryPlan {
    pub total_cost: f64,
    pub execution_time: f64,
    pub used_indexes: Vec<String>,
    pub node_types: Vec<String>,
}

impl QueryPlan {
    fn from_json(_json: serde_json::Value) -> Self {
        // 简化实现，实际应该解析完整的查询计划JSON
        Self {
            total_cost: 0.0,
            execution_time: 0.0,
            used_indexes: vec![],
            node_types: vec![],
        }
    }
}

/// 空间索引信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SpatialIndexInfo {
    pub table_name: String,
    pub indexes: Vec<SpatialIndex>,
    pub has_spatial_index: bool,
}

/// 空间索引详情
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SpatialIndex {
    pub schema_name: String,
    pub table_name: String,
    pub index_name: String,
    pub definition: String,
}

/// 优化建议
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OptimizationSuggestion {
    pub category: String,
    pub priority: OptimizationPriority,
    pub description: String,
    pub sql_command: Option<String>,
}

/// 优化优先级
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum OptimizationPriority {
    Low,
    Medium,
    High,
    Critical,
}

/// 查询优化助手函数
pub mod query_helpers {
    use super::*;
    
    /// 构建用户附近查询SQL（🔒 安全修复：只查询公开信息）
    pub fn build_nearby_users_query(geo_params: &GeoQueryParams) -> String {
        format!(
            r#"
            SELECT 
                u.user_id as "user_id!",
                u.nickname as "nickname!",
                u.avatar_url,
                u.last_active_at,
                u.expires_at,
                ul.latitude as "latitude!",
                ul.longitude as "longitude!",
                ul.location_name as "location_name!",
                {}
            FROM users u
            INNER JOIN user_locations ul ON u.user_id = ul.user_id
            WHERE {} 
            {}
            LIMIT $5
            "#,
            geo_params.get_distance_select(),
            geo_params.get_geo_where_condition(),
            geo_params.pagination.get_order_clause()
        )
    }
    
    /// 构建群组附近查询SQL
    pub fn build_nearby_groups_query(geo_params: &GeoQueryParams) -> String {
        format!(
            r#"
            SELECT 
                g.id as "id!",
                g.group_id as "group_id!",
                g.name as "name!",
                g.description,
                g.owner_id as "owner_id!",
                g.latitude as "latitude!",
                g.longitude as "longitude!",
                g.location_name as "location_name!",
                g.member_count as "member_count!",
                g.max_members as "max_members!",
                g.is_public as "is_public!",
                g.created_at as "created_at!",
                g.last_active_at as "last_active_at!",
                {}
            FROM groups g
            WHERE {} 
            {}
            LIMIT $5
            "#,
            geo_params.get_distance_select(),
            geo_params.get_geo_where_condition(),
            geo_params.pagination.get_order_clause()
        )
    }
    
    /// 构建生活瞬间附近查询SQL
    pub fn build_nearby_moments_query(geo_params: &GeoQueryParams) -> String {
        format!(
            r#"
            SELECT 
                m.id as "id!",
                m.moment_id as "moment_id!",
                m.user_id as "user_id!",
                m.content,
                m.latitude as "latitude!",
                m.longitude as "longitude!",
                m.location_name as "location_name!",
                m.created_at as "created_at!",
                COALESCE(like_counts.like_count, 0) as "like_count!",
                COALESCE(comment_counts.comment_count, 0) as "comment_count!",
                {}
            FROM moments m
            LEFT JOIN (
                SELECT moment_id, COUNT(*) as like_count
                FROM moment_likes
                GROUP BY moment_id
            ) like_counts ON m.id = like_counts.moment_id
            LEFT JOIN (
                SELECT moment_id, COUNT(*) as comment_count
                FROM comments
                WHERE target_type = 'moment'
                GROUP BY moment_id
            ) comment_counts ON m.id = comment_counts.moment_id
            WHERE {} 
            {}
            LIMIT $5
            "#,
            geo_params.get_distance_select(),
            geo_params.get_geo_where_condition(),
            geo_params.pagination.get_order_clause()
        )
    }
    
    /// 验证分页参数
    pub fn validate_pagination_params(cursor: Option<i64>, limit: Option<i32>) -> DomainResult<CursorPagination> {
        let limit = limit.unwrap_or(20);
        
        if limit < 1 || limit > 100 {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("Invalid limit: {}, must be between 1 and 100", limit),
            });
        }
        
        if let Some(cursor) = cursor {
            if cursor < 0 {
                return Err(DomainError::BusinessRuleViolation {
                    rule: format!("Invalid cursor: {}, must be non-negative", cursor),
                });
            }
        }
        
        Ok(CursorPagination::new(cursor, Some(limit)))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_cursor_pagination_where_condition() {
        let pagination = CursorPagination::new(Some(100), Some(20));
        assert_eq!(pagination.get_where_condition(), "($1 = 0 OR id < $1)");
        
        let pagination = CursorPagination::new(None, Some(20));
        assert_eq!(pagination.get_where_condition(), "TRUE");
    }
    
    #[test]
    fn test_geo_query_params_validation() {
        // 有效参数
        let result = GeoQueryParams::new(39.9042, 116.4074, 1000.0);
        assert!(result.is_ok());
        
        // 无效纬度
        let result = GeoQueryParams::new(91.0, 116.4074, 1000.0);
        assert!(result.is_err());
        
        // 无效经度
        let result = GeoQueryParams::new(39.9042, 181.0, 1000.0);
        assert!(result.is_err());
        
        // 无效半径
        let result = GeoQueryParams::new(39.9042, 116.4074, -100.0);
        assert!(result.is_err());
        
        let result = GeoQueryParams::new(39.9042, 116.4074, 200_000.0);
        assert!(result.is_err());
    }
    
    #[test]
    fn test_paginated_result_creation() {
        let items = vec![1, 2, 3, 4, 5, 6]; // 6个项目，limit=5
        let pagination = CursorPagination::new(None, Some(5));
        
        let result = PaginatedQueryResult::from_query_result(items, &pagination, |item| *item as i64);
        
        assert_eq!(result.items.len(), 5); // 应该只返回5个
        assert!(result.has_more); // 应该有更多数据
        assert_eq!(result.next_cursor, Some(5)); // 下一页游标应该是最后一个项目的ID
    }
}
