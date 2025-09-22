/// 搜索分析服务
/// 
/// 功能职责：
/// - 记录和分析用户搜索行为
/// - 生成热门搜索词和搜索趋势
/// - 提供搜索建议和自动补全
/// - 支持搜索性能监控

use std::collections::HashMap;
use std::sync::Arc;
use sqlx::PgPool;
use tracing::{debug, error, info};
use serde::{Serialize, Deserialize};
use chrono::{DateTime, Utc, Duration};

use crate::domain::{DomainResult, DomainError};

/// 搜索记录
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchRecord {
    pub search_id: String,
    pub user_id: Option<i64>,
    pub query: String,
    pub search_type: String,
    pub results_count: i32,
    pub clicked_result_id: Option<String>,
    pub search_duration_ms: i64,
    pub created_at: DateTime<Utc>,
}

/// 热门搜索词
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PopularSearch {
    pub query: String,
    pub search_count: i64,
    pub trend_score: f64,
    pub category: String,
}

/// 搜索趋势
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchTrend {
    pub query: String,
    pub current_count: i64,
    pub previous_count: i64,
    pub growth_rate: f64,
    pub trend_direction: TrendDirection,
}

/// 趋势方向
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TrendDirection {
    Rising,
    Falling,
    Stable,
}

/// 搜索建议
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchSuggestion {
    pub suggestion: String,
    pub score: f64,
    pub category: String,
}

/// 搜索分析服务
pub struct SearchAnalyticsService {
    pool: Arc<PgPool>,
}

impl SearchAnalyticsService {
    /// 创建新的搜索分析服务
    pub fn new(pool: Arc<PgPool>) -> Self {
        Self { pool }
    }
    
    /// 记录搜索行为
    pub async fn record_search(
        &self,
        user_id: Option<i64>,
        query: &str,
        search_type: &str,
        results_count: i32,
        search_duration_ms: i64,
    ) -> DomainResult<()> {
        debug!("记录搜索行为: 查询='{}', 类型={}, 结果数={}", query, search_type, results_count);
        
        let search_id = uuid::Uuid::new_v4().to_string();
        let now = Utc::now();
        
        sqlx::query!(
            "INSERT INTO search_logs (
                search_id, user_id, query, search_type, results_count, 
                search_duration_ms, created_at
            ) VALUES ($1, $2, $3, $4, $5, $6, $7)",
            search_id,
            user_id,
            query,
            search_type,
            results_count,
            search_duration_ms,
            now
        )
        .execute(&*self.pool)
        .await
        .map_err(|e| {
            error!("记录搜索行为失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?;
        
        // 异步更新搜索统计
        tokio::spawn({
            let pool = Arc::clone(&self.pool);
            let query = query.to_string();
            async move {
                let _ = Self::update_search_statistics(pool, &query).await;
            }
        });
        
        Ok(())
    }
    
    /// 记录搜索结果点击
    pub async fn record_search_click(
        &self,
        search_id: &str,
        clicked_result_id: &str,
    ) -> DomainResult<()> {
        debug!("记录搜索结果点击: 搜索ID={}, 结果ID={}", search_id, clicked_result_id);
        
        sqlx::query!(
            "UPDATE search_logs SET clicked_result_id = $1 WHERE search_id = $2",
            clicked_result_id,
            search_id
        )
        .execute(&*self.pool)
        .await
        .map_err(|e| {
            error!("记录搜索点击失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?;
        
        Ok(())
    }
    
    /// 获取热门搜索词
    pub async fn get_popular_searches(&self, limit: i32, category: Option<&str>) -> DomainResult<Vec<PopularSearch>> {
        debug!("获取热门搜索词: 限制={}, 分类={:?}", limit, category);
        
        let rows = if let Some(cat) = category {
            sqlx::query!(
                "SELECT query, search_count, trend_score, category
                 FROM popular_searches 
                 WHERE category = $1 AND is_active = true
                 ORDER BY trend_score DESC, search_count DESC
                 LIMIT $2",
                cat,
                limit
            )
            .fetch_all(&*self.pool)
            .await
        } else {
            sqlx::query!(
                "SELECT query, search_count, trend_score, category
                 FROM popular_searches 
                 WHERE is_active = true
                 ORDER BY trend_score DESC, search_count DESC
                 LIMIT $1",
                limit
            )
            .fetch_all(&*self.pool)
            .await
        };
        
        let rows = rows.map_err(|e| {
            error!("查询热门搜索词失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?;
        
        let popular_searches = rows
            .into_iter()
            .map(|row| PopularSearch {
                query: row.query,
                search_count: row.search_count,
                trend_score: row.trend_score,
                category: row.category,
            })
            .collect();
        
        // 如果数据库中没有数据，返回默认的热门搜索词
        if popular_searches.is_empty() {
            return Ok(self.get_default_popular_searches(limit));
        }
        
        Ok(popular_searches)
    }
    
    /// 获取搜索趋势
    pub async fn get_search_trends(&self, limit: i32, days: i32) -> DomainResult<Vec<SearchTrend>> {
        debug!("获取搜索趋势: 限制={}, 天数={}", limit, days);
        
        let end_date = Utc::now();
        let start_date = end_date - Duration::days(days as i64);
        let mid_date = end_date - Duration::days(days as i64 / 2);
        
        let rows = sqlx::query!(
            "WITH recent_searches AS (
                SELECT query, COUNT(*) as recent_count
                FROM search_logs 
                WHERE created_at >= $1 AND created_at < $2
                GROUP BY query
            ),
            older_searches AS (
                SELECT query, COUNT(*) as older_count
                FROM search_logs 
                WHERE created_at >= $3 AND created_at < $1
                GROUP BY query
            )
            SELECT 
                COALESCE(r.query, o.query) as query,
                COALESCE(r.recent_count, 0) as current_count,
                COALESCE(o.older_count, 0) as previous_count
            FROM recent_searches r
            FULL OUTER JOIN older_searches o ON r.query = o.query
            WHERE COALESCE(r.recent_count, 0) + COALESCE(o.older_count, 0) > 0
            ORDER BY COALESCE(r.recent_count, 0) DESC
            LIMIT $4",
            mid_date,
            end_date,
            start_date,
            limit
        )
        .fetch_all(&*self.pool)
        .await
        .map_err(|e| {
            error!("查询搜索趋势失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?;
        
        let trends = rows
            .into_iter()
            .map(|row| {
                let current_count = row.current_count;
                let previous_count = row.previous_count;
                
                let growth_rate = if previous_count > 0 {
                    ((current_count - previous_count) as f64 / previous_count as f64) * 100.0
                } else if current_count > 0 {
                    100.0 // 新出现的搜索词
                } else {
                    0.0
                };
                
                let trend_direction = if growth_rate > 10.0 {
                    TrendDirection::Rising
                } else if growth_rate < -10.0 {
                    TrendDirection::Falling
                } else {
                    TrendDirection::Stable
                };
                
                SearchTrend {
                    query: row.query,
                    current_count,
                    previous_count,
                    growth_rate,
                    trend_direction,
                }
            })
            .collect();
        
        Ok(trends)
    }
    
    /// 获取搜索建议
    pub async fn get_search_suggestions(&self, prefix: &str, limit: i32) -> DomainResult<Vec<SearchSuggestion>> {
        debug!("获取搜索建议: 前缀='{}', 限制={}", prefix, limit);
        
        if prefix.len() < 2 {
            return Ok(vec![]);
        }
        
        // 基于历史搜索记录生成建议
        let rows = sqlx::query!(
            "SELECT query, COUNT(*) as search_count
             FROM search_logs 
             WHERE query ILIKE $1 AND LENGTH(query) >= 2
             AND created_at >= NOW() - INTERVAL '30 days'
             GROUP BY query
             HAVING COUNT(*) >= 2
             ORDER BY COUNT(*) DESC, query
             LIMIT $2",
            format!("{}%", prefix),
            limit
        )
        .fetch_all(&*self.pool)
        .await
        .map_err(|e| {
            error!("查询搜索建议失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?;
        
        let mut suggestions = rows
            .into_iter()
            .map(|row| {
                let score = (row.search_count as f64).ln() + 1.0; // 对数评分
                SearchSuggestion {
                    suggestion: row.query,
                    score,
                    category: "history".to_string(),
                }
            })
            .collect::<Vec<_>>();
        
        // 如果建议不够，添加一些通用建议
        if suggestions.len() < limit as usize {
            let default_suggestions = self.get_default_suggestions(prefix, limit - suggestions.len() as i32);
            suggestions.extend(default_suggestions);
        }
        
        // 按分数排序
        suggestions.sort_by(|a, b| b.score.partial_cmp(&a.score).unwrap_or(std::cmp::Ordering::Equal));
        suggestions.truncate(limit as usize);
        
        Ok(suggestions)
    }
    
    /// 更新搜索统计（内部方法）
    async fn update_search_statistics(pool: Arc<PgPool>, query: &str) -> DomainResult<()> {
        let now = Utc::now();
        
        // 更新或插入搜索统计
        sqlx::query!(
            "INSERT INTO search_statistics (query, search_count, last_searched_at, updated_at)
             VALUES ($1, 1, $2, $3)
             ON CONFLICT (query) 
             DO UPDATE SET 
                search_count = search_statistics.search_count + 1,
                last_searched_at = $2,
                updated_at = $3",
            query,
            now,
            now
        )
        .execute(&*pool)
        .await
        .map_err(|e| {
            error!("更新搜索统计失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?;
        
        Ok(())
    }
    
    /// 计算热门搜索词趋势分数
    pub async fn calculate_trend_scores(&self) -> DomainResult<()> {
        info!("开始计算热门搜索词趋势分数");
        
        let now = Utc::now();
        let one_day_ago = now - Duration::days(1);
        let seven_days_ago = now - Duration::days(7);
        
        // 计算趋势分数并更新popular_searches表
        sqlx::query!(
            "INSERT INTO popular_searches (query, search_count, trend_score, category, is_active, updated_at)
             SELECT 
                s.query,
                s.search_count,
                -- 趋势分数计算：基于搜索频次、时间衰减、增长率
                (s.search_count * 0.6 + 
                 COALESCE(recent.recent_count, 0) * 0.4 + 
                 CASE 
                    WHEN COALESCE(older.older_count, 0) > 0 
                    THEN (COALESCE(recent.recent_count, 0) - COALESCE(older.older_count, 0)) * 0.2
                    ELSE COALESCE(recent.recent_count, 0) * 0.2
                 END) as trend_score,
                CASE 
                    WHEN s.query ~ '[美食|餐厅|小吃]' THEN 'food'
                    WHEN s.query ~ '[运动|健身|跑步]' THEN 'sports'
                    WHEN s.query ~ '[旅行|景点|酒店]' THEN 'travel'
                    WHEN s.query ~ '[电影|音乐|娱乐]' THEN 'entertainment'
                    ELSE 'general'
                END as category,
                true as is_active,
                $1 as updated_at
             FROM search_statistics s
             LEFT JOIN (
                SELECT query, COUNT(*) as recent_count
                FROM search_logs 
                WHERE created_at >= $2
                GROUP BY query
             ) recent ON s.query = recent.query
             LEFT JOIN (
                SELECT query, COUNT(*) as older_count
                FROM search_logs 
                WHERE created_at >= $3 AND created_at < $2
                GROUP BY query
             ) older ON s.query = older.query
             WHERE s.search_count >= 5  -- 至少被搜索5次
             ON CONFLICT (query) 
             DO UPDATE SET 
                search_count = EXCLUDED.search_count,
                trend_score = EXCLUDED.trend_score,
                category = EXCLUDED.category,
                updated_at = EXCLUDED.updated_at",
            now,
            one_day_ago,
            seven_days_ago
        )
        .execute(&*self.pool)
        .await
        .map_err(|e| {
            error!("计算趋势分数失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?;
        
        // 清理过期的热门搜索词
        sqlx::query!(
            "UPDATE popular_searches 
             SET is_active = false 
             WHERE updated_at < $1",
            now - Duration::days(30)
        )
        .execute(&*self.pool)
        .await
        .map_err(|e| {
            error!("清理过期热门搜索词失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?;
        
        info!("热门搜索词趋势分数计算完成");
        Ok(())
    }
    
    /// 获取默认热门搜索词（当数据库为空时使用）
    fn get_default_popular_searches(&self, limit: i32) -> Vec<PopularSearch> {
        let default_searches = vec![
            ("附近的人", "social"),
            ("美食推荐", "food"),
            ("运动健身", "sports"),
            ("旅行攻略", "travel"),
            ("摄影技巧", "photography"),
            ("学习交流", "education"),
            ("电影推荐", "entertainment"),
            ("音乐分享", "music"),
            ("宠物日常", "pets"),
            ("生活技巧", "lifestyle"),
        ];
        
        default_searches
            .into_iter()
            .take(limit as usize)
            .enumerate()
            .map(|(i, (query, category))| PopularSearch {
                query: query.to_string(),
                search_count: 100 - (i as i64 * 10), // 模拟递减的搜索次数
                trend_score: 90.0 - (i as f64 * 5.0), // 模拟递减的趋势分数
                category: category.to_string(),
            })
            .collect()
    }
    
    /// 获取默认搜索建议
    fn get_default_suggestions(&self, prefix: &str, limit: i32) -> Vec<SearchSuggestion> {
        let suggestions_map: HashMap<&str, Vec<&str>> = [
            ("美", vec!["美食", "美景", "美妆"]),
            ("运", vec!["运动", "运动健身", "运动装备"]),
            ("旅", vec!["旅行", "旅游", "旅行攻略"]),
            ("学", vec!["学习", "学生", "学校"]),
            ("工", vec!["工作", "工具", "工程"]),
            ("生", vec!["生活", "生日", "生活技巧"]),
            ("电", vec!["电影", "电视", "电子"]),
            ("音", vec!["音乐", "音响", "音频"]),
        ].iter().cloned().collect();
        
        let first_char = prefix.chars().next().unwrap_or(' ').to_string();
        
        if let Some(suggestions) = suggestions_map.get(first_char.as_str()) {
            suggestions
                .iter()
                .filter(|s| s.starts_with(prefix))
                .take(limit as usize)
                .enumerate()
                .map(|(i, suggestion)| SearchSuggestion {
                    suggestion: suggestion.to_string(),
                    score: 50.0 - (i as f64 * 5.0),
                    category: "default".to_string(),
                })
                .collect()
        } else {
            vec![]
        }
    }
    
    /// 获取搜索统计信息
    pub async fn get_search_statistics(&self, days: i32) -> DomainResult<HashMap<String, serde_json::Value>> {
        debug!("获取搜索统计信息: 天数={}", days);
        
        let start_date = Utc::now() - Duration::days(days as i64);
        
        // 总搜索次数
        let total_searches = sqlx::query_scalar!(
            "SELECT COUNT(*) FROM search_logs WHERE created_at >= $1",
            start_date
        )
        .fetch_one(&*self.pool)
        .await
        .map_err(|e| {
            error!("查询总搜索次数失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?
        .unwrap_or(0);
        
        // 独立用户数
        let unique_users = sqlx::query_scalar!(
            "SELECT COUNT(DISTINCT user_id) FROM search_logs WHERE created_at >= $1 AND user_id IS NOT NULL",
            start_date
        )
        .fetch_one(&*self.pool)
        .await
        .map_err(|e| {
            error!("查询独立用户数失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?
        .unwrap_or(0);
        
        // 平均搜索时长
        let avg_duration = sqlx::query_scalar!(
            "SELECT AVG(search_duration_ms) FROM search_logs WHERE created_at >= $1",
            start_date
        )
        .fetch_one(&*self.pool)
        .await
        .map_err(|e| {
            error!("查询平均搜索时长失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?
        .unwrap_or(0.0);
        
        let mut stats = HashMap::new();
        stats.insert("total_searches".to_string(), serde_json::Value::Number(serde_json::Number::from(total_searches)));
        stats.insert("unique_users".to_string(), serde_json::Value::Number(serde_json::Number::from(unique_users)));
        stats.insert("average_duration_ms".to_string(), serde_json::Value::Number(serde_json::Number::from_f64(avg_duration).unwrap_or(serde_json::Number::from(0))));
        stats.insert("period_days".to_string(), serde_json::Value::Number(serde_json::Number::from(days)));
        
        Ok(stats)
    }
}

impl Clone for SearchAnalyticsService {
    fn clone(&self) -> Self {
        Self {
            pool: Arc::clone(&self.pool),
        }
    }
}
