//! 标签服务 - 热门标签和搜索建议

use crate::{
    constants::{TRENDING_DAILY_DAYS, TRENDING_MONTHLY_DAYS, TRENDING_WEEKLY_DAYS},
    dto::{
        request::{SuggestTagsRequest, TrendingTagsRequest},
        response::{TagData, TagSuggestData, TagSuggestItem, TrendingTagsData},
    },
    error::{AppError, Result},
    repositories::tag_repo::TagRepository,
    utils::validation::Validator,
};

#[derive(Clone)]
pub struct TagService {
    tag_repo: TagRepository,
}

impl TagService {
    pub fn new(tag_repo: TagRepository) -> Self {
        Self { tag_repo }
    }

    /// 获取热门标签
    pub async fn get_trending(&self, req: &TrendingTagsRequest) -> Result<TrendingTagsData> {
        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let timeframe = req.timeframe.as_deref().unwrap_or("weekly");

        // 根据时间范围计算开始时间
        let since = match timeframe {
            "daily" => chrono::Utc::now() - chrono::Duration::days(TRENDING_DAILY_DAYS),
            "monthly" => chrono::Utc::now() - chrono::Duration::days(TRENDING_MONTHLY_DAYS),
            _ => chrono::Utc::now() - chrono::Duration::days(TRENDING_WEEKLY_DAYS),
        };

        // 从 tags 表获取热门标签
        let tag_rows = self.tag_repo.get_trending(limit, since).await?;

        let items = tag_rows
            .into_iter()
            .map(|r| TagData {
                name: r.name,
                usage_count: r.usage_count,
                trend: None,
            })
            .collect();

        Ok(TrendingTagsData { items })
    }

    /// 标签搜索建议
    pub async fn suggest(&self, req: &SuggestTagsRequest) -> Result<TagSuggestData> {
        // 验证prefix不为空
        if req.prefix.trim().is_empty() {
            return Err(AppError::BadRequest("标签前缀不能为空".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 50)? as i64;

        // 从 tags 表获取匹配前缀的标签
        let rows = self.tag_repo.suggest(&req.prefix, limit).await?;

        let items = rows
            .into_iter()
            .map(|r| TagSuggestItem {
                name: r.name,
                usage_count: r.usage_count,
            })
            .collect();

        Ok(TagSuggestData { items })
    }
}
