/// 数据库优化工具
/// 
/// 提供数据库性能分析、索引优化、查询优化等功能

use sqlx::PgPool;
use serde::{Serialize, Deserialize};
use std::collections::HashMap;
use tracing::{info, warn, error};

/// 数据库优化报告
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseOptimizationReport {
    pub database_info: DatabaseInfo,
    pub index_analysis: IndexAnalysis,
    pub query_performance: QueryPerformanceAnalysis,
    pub table_statistics: Vec<TableStatistics>,
    pub optimization_recommendations: Vec<OptimizationRecommendation>,
    pub generated_at: chrono::DateTime<chrono::Utc>,
}

/// 数据库基本信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseInfo {
    pub version: String,
    pub size_mb: f64,
    pub connection_count: i32,
    pub cache_hit_ratio: f64,
    pub extensions: Vec<String>,
}

/// 索引分析结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IndexAnalysis {
    pub total_indexes: i32,
    pub unused_indexes: Vec<UnusedIndex>,
    pub missing_indexes: Vec<MissingIndex>,
    pub duplicate_indexes: Vec<DuplicateIndex>,
    pub index_usage_stats: Vec<IndexUsageStats>,
}

/// 未使用的索引
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnusedIndex {
    pub schema_name: String,
    pub table_name: String,
    pub index_name: String,
    pub size_mb: f64,
    pub scan_count: i64,
}

/// 缺失的索引建议
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MissingIndex {
    pub table_name: String,
    pub columns: Vec<String>,
    pub query_pattern: String,
    pub estimated_benefit: f64,
}

/// 重复索引
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DuplicateIndex {
    pub table_name: String,
    pub index1: String,
    pub index2: String,
    pub columns: Vec<String>,
}

/// 索引使用统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IndexUsageStats {
    pub schema_name: String,
    pub table_name: String,
    pub index_name: String,
    pub scans: i64,
    pub tuples_read: i64,
    pub tuples_fetched: i64,
    pub size_mb: f64,
}

/// 查询性能分析
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueryPerformanceAnalysis {
    pub slow_queries: Vec<SlowQuery>,
    pub most_frequent_queries: Vec<FrequentQuery>,
    pub query_cache_stats: QueryCacheStats,
}

/// 慢查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SlowQuery {
    pub query: String,
    pub calls: i64,
    pub total_time_ms: f64,
    pub mean_time_ms: f64,
    pub rows: i64,
}

/// 频繁查询
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FrequentQuery {
    pub query: String,
    pub calls: i64,
    pub total_time_ms: f64,
    pub mean_time_ms: f64,
}

/// 查询缓存统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueryCacheStats {
    pub hit_ratio: f64,
    pub cache_size_mb: f64,
    pub evictions: i64,
}

/// 表统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TableStatistics {
    pub schema_name: String,
    pub table_name: String,
    pub row_count: i64,
    pub size_mb: f64,
    pub index_size_mb: f64,
    pub seq_scans: i64,
    pub index_scans: i64,
    pub inserts: i64,
    pub updates: i64,
    pub deletes: i64,
    pub last_vacuum: Option<chrono::DateTime<chrono::Utc>>,
    pub last_analyze: Option<chrono::DateTime<chrono::Utc>>,
}

/// 优化建议
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OptimizationRecommendation {
    pub category: String,
    pub priority: String, // High, Medium, Low
    pub title: String,
    pub description: String,
    pub sql_commands: Vec<String>,
    pub estimated_impact: String,
}

/// 数据库优化工具
pub struct DatabaseOptimizerTool {
    pool: PgPool,
}

impl DatabaseOptimizerTool {
    /// 创建新的优化工具实例
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
    
    /// 生成完整的数据库优化报告
    pub async fn generate_optimization_report(&self) -> Result<DatabaseOptimizationReport, sqlx::Error> {
        info!("开始生成数据库优化报告");
        
        let database_info = self.get_database_info().await?;
        let index_analysis = self.analyze_indexes().await?;
        let query_performance = self.analyze_query_performance().await?;
        let table_statistics = self.get_table_statistics().await?;
        let optimization_recommendations = self.generate_recommendations(&index_analysis, &query_performance, &table_statistics).await;
        
        let report = DatabaseOptimizationReport {
            database_info,
            index_analysis,
            query_performance,
            table_statistics,
            optimization_recommendations,
            generated_at: chrono::Utc::now(),
        };
        
        info!("数据库优化报告生成完成");
        Ok(report)
    }
    
    /// 获取数据库基本信息
    async fn get_database_info(&self) -> Result<DatabaseInfo, sqlx::Error> {
        // 获取数据库版本
        let version_row = sqlx::query!("SELECT version()")
            .fetch_one(&self.pool)
            .await?;
        
        // 获取数据库大小
        let size_row = sqlx::query!(
            "SELECT pg_size_pretty(pg_database_size(current_database())) as size, 
             pg_database_size(current_database()) / 1024.0 / 1024.0 as size_mb"
        )
        .fetch_one(&self.pool)
        .await?;
        
        // 获取连接数
        let conn_row = sqlx::query!(
            "SELECT count(*) as connection_count FROM pg_stat_activity"
        )
        .fetch_one(&self.pool)
        .await?;
        
        // 获取缓存命中率
        let cache_row = sqlx::query!(
            r#"
            SELECT 
                round(
                    100.0 * sum(blks_hit) / NULLIF(sum(blks_hit + blks_read), 0), 2
                ) as cache_hit_ratio
            FROM pg_stat_database
            WHERE datname = current_database()
            "#
        )
        .fetch_one(&self.pool)
        .await?;
        
        // 获取已安装的扩展
        let extensions = sqlx::query!("SELECT extname FROM pg_extension ORDER BY extname")
            .fetch_all(&self.pool)
            .await?
            .into_iter()
            .map(|row| row.extname)
            .collect();
        
        Ok(DatabaseInfo {
            version: version_row.version.unwrap_or_default(),
            size_mb: size_row.size_mb.unwrap_or(0.0),
            connection_count: conn_row.connection_count.unwrap_or(0) as i32,
            cache_hit_ratio: cache_row.cache_hit_ratio.unwrap_or(0.0),
            extensions,
        })
    }
    
    /// 分析索引使用情况
    async fn analyze_indexes(&self) -> Result<IndexAnalysis, sqlx::Error> {
        // 获取索引使用统计
        let index_stats = sqlx::query!(
            r#"
            SELECT 
                schemaname,
                tablename,
                indexname,
                idx_scan as scans,
                idx_tup_read as tuples_read,
                idx_tup_fetch as tuples_fetched,
                pg_size_pretty(pg_relation_size(indexrelid)) as size_pretty,
                pg_relation_size(indexrelid) / 1024.0 / 1024.0 as size_mb
            FROM pg_stat_user_indexes
            ORDER BY idx_scan DESC
            "#
        )
        .fetch_all(&self.pool)
        .await?;
        
        let index_usage_stats: Vec<IndexUsageStats> = index_stats
            .into_iter()
            .map(|row| IndexUsageStats {
                schema_name: row.schemaname.unwrap_or_default(),
                table_name: row.tablename.unwrap_or_default(),
                index_name: row.indexname.unwrap_or_default(),
                scans: row.scans.unwrap_or(0),
                tuples_read: row.tuples_read.unwrap_or(0),
                tuples_fetched: row.tuples_fetched.unwrap_or(0),
                size_mb: row.size_mb.unwrap_or(0.0),
            })
            .collect();
        
        // 查找未使用的索引
        let unused_indexes: Vec<UnusedIndex> = index_usage_stats
            .iter()
            .filter(|stat| stat.scans == 0 && stat.size_mb > 1.0) // 大于1MB且未使用
            .map(|stat| UnusedIndex {
                schema_name: stat.schema_name.clone(),
                table_name: stat.table_name.clone(),
                index_name: stat.index_name.clone(),
                size_mb: stat.size_mb,
                scan_count: stat.scans,
            })
            .collect();
        
        // 生成缺失索引建议（基于常见查询模式）
        let missing_indexes = self.suggest_missing_indexes().await?;
        
        // 查找重复索引
        let duplicate_indexes = self.find_duplicate_indexes().await?;
        
        Ok(IndexAnalysis {
            total_indexes: index_usage_stats.len() as i32,
            unused_indexes,
            missing_indexes,
            duplicate_indexes,
            index_usage_stats,
        })
    }
    
    /// 建议缺失的索引
    async fn suggest_missing_indexes(&self) -> Result<Vec<MissingIndex>, sqlx::Error> {
        let mut suggestions = Vec::new();
        
        // 检查用户表是否有适当的索引
        let user_table_check = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM pg_indexes 
                WHERE tablename = 'users' 
                AND indexdef LIKE '%nickname%'
            ) as has_nickname_index
            "#
        )
        .fetch_one(&self.pool)
        .await?;
        
        if !user_table_check.has_nickname_index.unwrap_or(false) {
            suggestions.push(MissingIndex {
                table_name: "users".to_string(),
                columns: vec!["nickname".to_string()],
                query_pattern: "用户昵称搜索查询".to_string(),
                estimated_benefit: 0.8,
            });
        }
        
        // 检查地理位置相关索引
        let geo_index_check = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM pg_indexes 
                WHERE tablename = 'user_locations' 
                AND indexdef LIKE '%gist%'
            ) as has_geo_index
            "#
        )
        .fetch_one(&self.pool)
        .await?;
        
        if !geo_index_check.has_geo_index.unwrap_or(false) {
            suggestions.push(MissingIndex {
                table_name: "user_locations".to_string(),
                columns: vec!["geom".to_string()],
                query_pattern: "地理位置范围查询".to_string(),
                estimated_benefit: 0.9,
            });
        }
        
        Ok(suggestions)
    }
    
    /// 查找重复索引
    async fn find_duplicate_indexes(&self) -> Result<Vec<DuplicateIndex>, sqlx::Error> {
        let duplicates = sqlx::query!(
            r#"
            WITH index_columns AS (
                SELECT 
                    schemaname,
                    tablename,
                    indexname,
                    array_agg(attname ORDER BY attnum) as columns
                FROM pg_indexes 
                JOIN pg_index ON pg_indexes.indexname = pg_class.relname
                JOIN pg_class ON pg_index.indexrelid = pg_class.oid
                JOIN pg_attribute ON pg_attribute.attrelid = pg_index.indrelid 
                    AND pg_attribute.attnum = ANY(pg_index.indkey)
                WHERE schemaname = 'public'
                GROUP BY schemaname, tablename, indexname
            )
            SELECT 
                i1.tablename,
                i1.indexname as index1,
                i2.indexname as index2,
                i1.columns
            FROM index_columns i1
            JOIN index_columns i2 ON i1.tablename = i2.tablename 
                AND i1.columns = i2.columns 
                AND i1.indexname < i2.indexname
            "#
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(duplicates
            .into_iter()
            .map(|row| DuplicateIndex {
                table_name: row.tablename.unwrap_or_default(),
                index1: row.index1.unwrap_or_default(),
                index2: row.index2.unwrap_or_default(),
                columns: row.columns.unwrap_or_default(),
            })
            .collect())
    }
    
    /// 分析查询性能
    async fn analyze_query_performance(&self) -> Result<QueryPerformanceAnalysis, sqlx::Error> {
        // 检查是否启用了pg_stat_statements扩展
        let has_pg_stat_statements = sqlx::query!(
            "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements') as exists"
        )
        .fetch_one(&self.pool)
        .await?
        .exists
        .unwrap_or(false);
        
        let (slow_queries, most_frequent_queries) = if has_pg_stat_statements {
            // 获取慢查询
            let slow_queries_data = sqlx::query!(
                r#"
                SELECT 
                    query,
                    calls,
                    total_time,
                    mean_time,
                    rows
                FROM pg_stat_statements 
                WHERE mean_time > 100  -- 超过100ms的查询
                ORDER BY mean_time DESC
                LIMIT 10
                "#
            )
            .fetch_all(&self.pool)
            .await?;
            
            let slow_queries: Vec<SlowQuery> = slow_queries_data
                .into_iter()
                .map(|row| SlowQuery {
                    query: row.query.unwrap_or_default(),
                    calls: row.calls.unwrap_or(0),
                    total_time_ms: row.total_time.unwrap_or(0.0),
                    mean_time_ms: row.mean_time.unwrap_or(0.0),
                    rows: row.rows.unwrap_or(0),
                })
                .collect();
            
            // 获取最频繁的查询
            let frequent_queries_data = sqlx::query!(
                r#"
                SELECT 
                    query,
                    calls,
                    total_time,
                    mean_time
                FROM pg_stat_statements 
                ORDER BY calls DESC
                LIMIT 10
                "#
            )
            .fetch_all(&self.pool)
            .await?;
            
            let frequent_queries: Vec<FrequentQuery> = frequent_queries_data
                .into_iter()
                .map(|row| FrequentQuery {
                    query: row.query.unwrap_or_default(),
                    calls: row.calls.unwrap_or(0),
                    total_time_ms: row.total_time.unwrap_or(0.0),
                    mean_time_ms: row.mean_time.unwrap_or(0.0),
                })
                .collect();
            
            (slow_queries, frequent_queries)
        } else {
            warn!("pg_stat_statements扩展未启用，无法获取详细的查询统计信息");
            (Vec::new(), Vec::new())
        };
        
        // 查询缓存统计（简化版本）
        let cache_stats = QueryCacheStats {
            hit_ratio: 95.0, // 默认值
            cache_size_mb: 128.0, // 默认值
            evictions: 0, // 默认值
        };
        
        Ok(QueryPerformanceAnalysis {
            slow_queries,
            most_frequent_queries,
            query_cache_stats: cache_stats,
        })
    }
    
    /// 获取表统计信息
    async fn get_table_statistics(&self) -> Result<Vec<TableStatistics>, sqlx::Error> {
        let stats = sqlx::query!(
            r#"
            SELECT 
                schemaname,
                tablename,
                n_tup_ins as inserts,
                n_tup_upd as updates,
                n_tup_del as deletes,
                seq_scan as seq_scans,
                idx_scan as index_scans,
                last_vacuum,
                last_analyze,
                pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as total_size_pretty,
                pg_total_relation_size(schemaname||'.'||tablename) / 1024.0 / 1024.0 as size_mb,
                pg_size_pretty(pg_indexes_size(schemaname||'.'||tablename)) as index_size_pretty,
                pg_indexes_size(schemaname||'.'||tablename) / 1024.0 / 1024.0 as index_size_mb
            FROM pg_stat_user_tables
            ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
            "#
        )
        .fetch_all(&self.pool)
        .await?;
        
        let mut table_stats = Vec::new();
        
        for row in stats {
            // 获取行数估计
            let row_count_result = sqlx::query!(
                "SELECT reltuples::bigint as estimated_rows FROM pg_class WHERE relname = $1",
                row.tablename
            )
            .fetch_optional(&self.pool)
            .await?;
            
            let row_count = row_count_result
                .and_then(|r| r.estimated_rows)
                .unwrap_or(0);
            
            table_stats.push(TableStatistics {
                schema_name: row.schemaname.unwrap_or_default(),
                table_name: row.tablename.unwrap_or_default(),
                row_count,
                size_mb: row.size_mb.unwrap_or(0.0),
                index_size_mb: row.index_size_mb.unwrap_or(0.0),
                seq_scans: row.seq_scans.unwrap_or(0),
                index_scans: row.index_scans.unwrap_or(0),
                inserts: row.inserts.unwrap_or(0),
                updates: row.updates.unwrap_or(0),
                deletes: row.deletes.unwrap_or(0),
                last_vacuum: row.last_vacuum,
                last_analyze: row.last_analyze,
            });
        }
        
        Ok(table_stats)
    }
    
    /// 生成优化建议
    async fn generate_recommendations(
        &self,
        index_analysis: &IndexAnalysis,
        query_performance: &QueryPerformanceAnalysis,
        table_statistics: &[TableStatistics],
    ) -> Vec<OptimizationRecommendation> {
        let mut recommendations = Vec::new();
        
        // 未使用索引建议
        if !index_analysis.unused_indexes.is_empty() {
            let total_unused_size: f64 = index_analysis.unused_indexes.iter().map(|idx| idx.size_mb).sum();
            recommendations.push(OptimizationRecommendation {
                category: "索引优化".to_string(),
                priority: "Medium".to_string(),
                title: "删除未使用的索引".to_string(),
                description: format!("发现 {} 个未使用的索引，总大小 {:.2} MB", 
                                   index_analysis.unused_indexes.len(), total_unused_size),
                sql_commands: index_analysis.unused_indexes.iter()
                    .map(|idx| format!("DROP INDEX IF EXISTS {};", idx.index_name))
                    .collect(),
                estimated_impact: format!("节省 {:.2} MB 存储空间，提高写入性能", total_unused_size),
            });
        }
        
        // 缺失索引建议
        if !index_analysis.missing_indexes.is_empty() {
            for missing in &index_analysis.missing_indexes {
                recommendations.push(OptimizationRecommendation {
                    category: "索引优化".to_string(),
                    priority: "High".to_string(),
                    title: format!("为 {} 表添加索引", missing.table_name),
                    description: format!("建议为 {} 查询添加索引", missing.query_pattern),
                    sql_commands: vec![
                        format!("CREATE INDEX CONCURRENTLY idx_{}_{} ON {} ({});",
                               missing.table_name,
                               missing.columns.join("_"),
                               missing.table_name,
                               missing.columns.join(", "))
                    ],
                    estimated_impact: format!("预计性能提升 {:.0}%", missing.estimated_benefit * 100.0),
                });
            }
        }
        
        // 慢查询优化建议
        if !query_performance.slow_queries.is_empty() {
            recommendations.push(OptimizationRecommendation {
                category: "查询优化".to_string(),
                priority: "High".to_string(),
                title: "优化慢查询".to_string(),
                description: format!("发现 {} 个平均执行时间超过 100ms 的查询", 
                                   query_performance.slow_queries.len()),
                sql_commands: vec![
                    "-- 分析慢查询执行计划".to_string(),
                    "-- EXPLAIN (ANALYZE, BUFFERS) <your_slow_query>;".to_string(),
                ],
                estimated_impact: "显著提高查询响应时间".to_string(),
            });
        }
        
        // 表维护建议
        for table in table_statistics {
            if table.seq_scans > table.index_scans * 2 && table.row_count > 1000 {
                recommendations.push(OptimizationRecommendation {
                    category: "表优化".to_string(),
                    priority: "Medium".to_string(),
                    title: format!("优化 {} 表的查询模式", table.table_name),
                    description: format!("表 {} 的顺序扫描次数 ({}) 远超索引扫描次数 ({})", 
                                       table.table_name, table.seq_scans, table.index_scans),
                    sql_commands: vec![
                        format!("ANALYZE {};", table.table_name),
                        format!("-- 考虑为 {} 添加适当的索引", table.table_name),
                    ],
                    estimated_impact: "减少全表扫描，提高查询效率".to_string(),
                });
            }
        }
        
        recommendations
    }
    
    /// 应用优化建议
    pub async fn apply_optimization(&self, recommendation: &OptimizationRecommendation) -> Result<(), sqlx::Error> {
        info!("应用优化建议: {}", recommendation.title);
        
        for sql_command in &recommendation.sql_commands {
            if sql_command.starts_with("--") {
                // 跳过注释
                continue;
            }
            
            match sqlx::query(sql_command).execute(&self.pool).await {
                Ok(_) => info!("成功执行: {}", sql_command),
                Err(e) => {
                    error!("执行失败: {} - 错误: {}", sql_command, e);
                    return Err(e);
                }
            }
        }
        
        Ok(())
    }
}
