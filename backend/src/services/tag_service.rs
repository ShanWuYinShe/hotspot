use crate::{
    dto::{
        request::{GetTrendingTagsRequest, SuggestTagsRequest},
        response::{TagData, TagSuggestData, TrendingTagsData},
    },
    error::{AppError, Result},
};
use sqlx::PgPool;

pub struct TagService {
    db: PgPool,
}

impl TagService {
    pub fn new(db: &PgPool) -> Self {
        Self { db: db.clone() }
    }

    /// 获取热门标签
    ///
    pub async fn get_trending(&self, req: &GetTrendingTagsRequest) -> Result<TrendingTagsData> {
        let limit = req.limit.unwrap_or(20).min(100).max(1) as i64;
        let timeframe = req.timeframe.as_deref().unwrap_or("weekly");

        // 根据时间范围计算开始时间
        let start_time = match timeframe {
            "daily" => chrono::Utc::now() - chrono::Duration::days(1),
            "monthly" => chrono::Utc::now() - chrono::Duration::days(30),
            _ => chrono::Utc::now() - chrono::Duration::days(7), // weekly
        };

        // 查询热门标签（从moments表的tags字段中统计）
        let records = sqlx::query!(
            r#"
            SELECT tag, COUNT(*) as usage_count
            FROM moments, UNNEST(tags) AS tag
            WHERE created_at >= $1
            GROUP BY tag
            ORDER BY usage_count DESC
            LIMIT $2
            "#,
            start_time,
            limit
        )
        .fetch_all(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        let items = records
            .into_iter()
            .map(|r| TagData {
                name: r.tag.unwrap_or_default(),
                usage_count: r.usage_count.unwrap_or(0) as i32,
                trend: None, // TODO: 实现趋势计算
            })
            .collect();

        Ok(TrendingTagsData { items })
    }

    /// 标签搜索建议
    ///
    pub async fn suggest(&self, req: &SuggestTagsRequest) -> Result<TagSuggestData> {
        // 验证prefix不为空
        if req.prefix.trim().is_empty() {
            return Err(AppError::BadRequest("标签前缀不能为空".to_string()));
        }

        let limit = req.limit.unwrap_or(10).min(50).max(1) as i64;

        // 查询匹配前缀的标签
        let records = sqlx::query!(
            r#"
            SELECT tag, COUNT(*) as usage_count
            FROM moments, UNNEST(tags) AS tag
            WHERE tag ILIKE $1
            GROUP BY tag
            ORDER BY usage_count DESC
            LIMIT $2
            "#,
            format!("{}%", req.prefix),
            limit
        )
        .fetch_all(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        let items = records
            .into_iter()
            .map(|r| r.tag.unwrap_or_default())
            .collect();

        Ok(TagSuggestData { items })
    }
}


