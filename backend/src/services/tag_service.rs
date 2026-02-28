//! 标签服务 - 热门标签和搜索建议

use crate::{
    dto::{
        request::{AdminDeleteTagRequest, AdminTagSearchRequest, AdminTagSortBy},
        response::{AdminTagItem, PaginatedData},
    },
    error::{AppError, Result},
    repositories::tag_repo::{AdminTagCursor, AdminTagRow, AdminTagSortOrder, TagRepository},
    utils::{
        pagination::{decode_cursor, encode_cursor, paginate_with_limit_plus_one},
        validation::Validator,
    },
};
use tracing::{info, warn};

#[derive(Clone)]
pub struct TagService {
    tag_repo: TagRepository,
}

impl TagService {
    pub fn new(tag_repo: TagRepository) -> Self {
        Self { tag_repo }
    }

    /// 标签搜索建议
    /// 删除标签（管理员）
    pub async fn admin_delete_tag(&self, req: &AdminDeleteTagRequest) -> Result<()> {
        let started_at = std::time::Instant::now();
        info!(tag_name = %req.name, "admin_tag.delete started");
        let deleted = self.tag_repo.delete(&req.name).await?;
        if !deleted {
            warn!(tag_name = %req.name, "admin_tag.delete rejected: tag not found");
            return Err(AppError::NotFound(format!("Tag '{}' not found", req.name)));
        }
        info!(
            tag_name = %req.name,
            elapsed_ms = started_at.elapsed().as_millis(),
            "admin_tag.delete completed"
        );
        Ok(())
    }

    /// 管理员标签搜索
    pub async fn admin_search_tags(
        &self,
        req: &AdminTagSearchRequest,
    ) -> Result<PaginatedData<AdminTagItem>> {
        let pagination = req.pagination.as_ref();
        let limit_input = pagination.and_then(|p| p.limit).unwrap_or(20);
        let limit = Validator::validate_limit(Some(limit_input), 50)? as i32;
        let cursor_str = pagination.and_then(|p| p.cursor.as_deref());
        let cursor = cursor_str
            .map(decode_cursor::<AdminTagCursor>)
            .transpose()?;

        let sort_by = req.sort_by.unwrap_or(AdminTagSortBy::UsageCountDesc);
        let sort_order = match sort_by {
            AdminTagSortBy::UsageCountDesc => AdminTagSortOrder::UsageCountDesc,
            AdminTagSortBy::UsageCountAsc => AdminTagSortOrder::UsageCountAsc,
            AdminTagSortBy::CreatedAtDesc => AdminTagSortOrder::CreatedAtDesc,
            AdminTagSortBy::LastUsedAtDesc => AdminTagSortOrder::LastUsedAtDesc,
        };

        let fetch_limit = (limit + 1) as i64;
        let rows = self
            .tag_repo
            .admin_search(req.keyword.as_deref(), sort_order, fetch_limit, cursor)
            .await?;

        let encode_fn = |row: &AdminTagRow| -> Result<String> {
            let payload = match sort_by {
                AdminTagSortBy::UsageCountDesc => AdminTagCursor::UsageCountDesc {
                    usage_count: row.usage_count,
                    tag_id: row.tag_id,
                },
                AdminTagSortBy::UsageCountAsc => AdminTagCursor::UsageCountAsc {
                    usage_count: row.usage_count,
                    tag_id: row.tag_id,
                },
                AdminTagSortBy::CreatedAtDesc => AdminTagCursor::CreatedAtDesc {
                    created_at: row.created_at,
                    tag_id: row.tag_id,
                },
                AdminTagSortBy::LastUsedAtDesc => AdminTagCursor::LastUsedAtDesc {
                    last_used_at: row.last_used_at,
                    tag_id: row.tag_id,
                },
            };
            encode_cursor(&payload)
        };

        let (rows, next_cursor) = paginate_with_limit_plus_one(rows, limit, encode_fn)?;
        let items = rows
            .into_iter()
            .map(|row| AdminTagItem {
                name: row.name,
                usage_count: row.usage_count,
                created_at: row.created_at.to_rfc3339(),
                last_used_at: Some(row.last_used_at.to_rfc3339()),
            })
            .collect();

        Ok(PaginatedData { items, next_cursor })
    }
}
