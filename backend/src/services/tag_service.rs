//! 标签服务 - 热门标签和搜索建议

use crate::{
    dto::{
        request::SuggestTagsRequest,
        response::{TagSuggestData, TagSuggestItem},
    },
    error::{AppError, Result},
    repositories::tag_repo::TagRepository,
    utils::{
        cache::CacheManager,
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
