//! 标签服务 - 热门标签和搜索建议

use crate::{
    dto::{
        request::{SuggestTagsRequest, TrendingTagsRequest},
        response::{TagItem, TagSuggestData, TagSuggestItem, TrendingTagsData},
    },
    error::{AppError, Result},
    repositories::tag_repo::TagRepository,
    utils::{
        cache::{cache_keys, CacheManager},
        validation::Validator,
    },
};

#[derive(Clone)]
pub struct TagService {
    tag_repo: TagRepository,
    cache_manager: CacheManager,
}

impl TagService {
    pub fn new(tag_repo: TagRepository, cache_manager: CacheManager) -> Self {
        Self {
            tag_repo,
            cache_manager,
        }
    }

    /// 获取热门标签
    pub async fn get_trending(&self, req: &TrendingTagsRequest) -> Result<TrendingTagsData> {
        use crate::types::enums::TagTimeframe;

        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let timeframe = req.timeframe.unwrap_or(TagTimeframe::Week);
        let cache_key = format!("{}:{}", timeframe.as_str(), limit);

        let cached =
            self.cache_manager
                .get_or_set_with_lock_with_ttl(
                    cache_keys::TRENDING_TAGS,
                    &cache_key,
                    300,
                    60,
                    || async {
                        // 从 tags 表获取热门标签
                        let tag_rows =
                            self.tag_repo.get_trending(limit).await.map_err(|e| {
                                Box::new(e) as Box<dyn std::error::Error + Send + Sync>
                            })?;

                        let items = tag_rows
                            .into_iter()
                            .map(|r| TagItem {
                                name: r.name,
                                usage_count: r.usage_count.unwrap_or(0),
                                trend: None,
                            })
                            .collect();

                        Ok(Some(TrendingTagsData { items }))
                    },
                )
                .await
                .map_err(|e| AppError::InternalError(e.to_string()))?;

        if let Some(data) = cached {
            return Ok(data);
        }

        Ok(TrendingTagsData { items: vec![] })
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
                usage_count: r.usage_count.unwrap_or(0),
            })
            .collect();

        Ok(TagSuggestData { items })
    }

    /// 删除标签（管理员）
    pub async fn admin_delete_tag(
        &self,
        req: &crate::dto::request::AdminDeleteTagRequest,
    ) -> Result<()> {
        let deleted = self.tag_repo.delete(&req.tag_name).await?;
        if !deleted {
            return Err(AppError::NotFound(format!(
                "Tag '{}' not found",
                req.tag_name
            )));
        }
        Ok(())
    }
}
