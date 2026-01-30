use crate::{
    dto::{
        request::SearchSuggestRequest,
        response::{
            GroupSuggestItem, MomentSuggestItem, SearchSuggestItem, SearchSuggestions,
            TagSuggestItem, UserSuggestItem,
        },
    },
    error::AppError,
    repositories::{
        group_repo::GroupRepository, moment_repo::MomentRepository, tag_repo::TagRepository,
        user_repo::UserRepository,
    },
    utils::validation::Validator,
};
use redis::aio::ConnectionManager;

#[derive(Clone)]
pub struct SearchService {
    redis: ConnectionManager,
    user_repo: UserRepository,
    group_repo: GroupRepository,
    tag_repo: TagRepository,
    moment_repo: MomentRepository,
    media_service: crate::services::media_service::MediaService,
}

impl SearchService {
    pub fn new(
        redis: ConnectionManager,
        user_repo: UserRepository,
        group_repo: GroupRepository,
        tag_repo: TagRepository,
        moment_repo: MomentRepository,
        media_service: crate::services::media_service::MediaService,
    ) -> Self {
        Self {
            redis,
            user_repo,
            group_repo,
            tag_repo,
            moment_repo,
            media_service,
        }
    }

    /// 综合搜索建议（用户/群组/瞬间/标签）
    pub async fn suggest(
        &self,
        viewer_id: Option<i64>,
        req: SearchSuggestRequest,
    ) -> Result<SearchSuggestions, AppError> {
        Validator::validate_not_empty(&req.prefix, "prefix")?;
        let limit = Validator::validate_limit(req.limit, 20)? as i64;

        // 解析位置参数
        let location = if let Some(point) = req.location {
            Validator::validate_coordinates(point.latitude, point.longitude)?;
            Some((point.latitude, point.longitude))
        } else {
            None
        };

        // 并发执行查询
        let (tag_rows, user_rows, group_rows, moment_rows) = tokio::try_join!(
            self.tag_repo.suggest(&req.prefix, limit),
            self.user_repo
                .suggest_by_nickname(&req.prefix, limit, location),
            self.group_repo
                .suggest_by_name(&req.prefix, limit, location),
            self.moment_repo
                .suggest_by_keyword(&req.prefix, viewer_id, limit as i32, location)
        )?;

        let user_ids: Vec<_> = user_rows.iter().map(|u| u.user_id).collect();
        let group_ids: Vec<_> = group_rows.iter().map(|g| g.group_id).collect();
        let moment_ids: Vec<_> = moment_rows.iter().map(|m| m.moment_id).collect();

        // 批量获取标签（标签建议本身不需要额外标签）
        let (user_tags, group_tags, moment_tags) = tokio::try_join!(
            self.tag_repo.get_user_tags_batch(&user_ids),
            self.tag_repo.get_group_tags_batch(&group_ids),
            self.tag_repo.get_moment_tags_batch(&moment_ids)
        )?;

        let mut items: Vec<SearchSuggestItem> = Vec::new();

        // 1. 标签建议
        for tag in tag_rows {
            items.push(SearchSuggestItem::Tag(TagSuggestItem {
                name: tag.name,
                usage_count: tag.usage_count,
            }));
        }

        // 2. 用户建议
        for user in user_rows {
            let avatar_url = self
                .media_service
                .sign_url_if_path(user.avatar_url.as_deref())
                .await;
            let tags = user_tags
                .get(&user.user_id)
                .cloned()
                .filter(|t| !t.is_empty());
            let bio = if user.bio.trim().is_empty() {
                None
            } else {
                Some(user.bio)
            };

            items.push(SearchSuggestItem::User(UserSuggestItem {
                user: crate::dto::response::UserInfo {
                    user_id: user.user_id,
                    nickname: user.nickname,
                    avatar_url,
                    user_type: user.user_type,
                },
                bio,
                tags,
                distance: user.distance,
            }));
        }

        // 3. 群组建议
        for group in group_rows {
            let avatar_url = self
                .media_service
                .sign_url_if_path(group.avatar_url.as_deref())
                .await;
            let tags = group_tags
                .get(&group.group_id)
                .cloned()
                .filter(|t| !t.is_empty());
            let description = if group.description.trim().is_empty() {
                None
            } else {
                Some(group.description)
            };

            items.push(SearchSuggestItem::Group(GroupSuggestItem {
                group: crate::dto::response::GroupInfo {
                    group_chat_id: group.group_id.to_string(),
                    name: group.name,
                    avatar_url,
                    members_count: group.member_count,
                },
                description,
                tags,
                has_password: group.has_password,
                distance: group.distance,
            }));
        }

        // 4. 瞬间建议
        for moment in moment_rows {
            let avatar_url = self
                .media_service
                .sign_url_if_path(moment.avatar_url.as_deref())
                .await;
            let images = self.sign_moment_images(&moment.images).await;
            let tags = moment_tags
                .get(&moment.moment_id)
                .cloned()
                .filter(|t| !t.is_empty());

            items.push(SearchSuggestItem::Moment(MomentSuggestItem {
                moment_id: moment.moment_id.to_string(),
                description: moment.description,
                images,
                user: crate::dto::response::UserInfo {
                    user_id: moment.user_id,
                    nickname: moment.nickname,
                    avatar_url,
                    user_type: moment.user_type,
                },
                tags,
                distance: moment.distance,
            }));
        }

        // 排序逻辑：
        // 1. 如果有 location，按 distance 排序 (混合排序)
        // 2. 如果没有 location，保持各类别的默认顺序 (或者可以按热度/匹配度混合)
        // 目前简单实现：如果有 location，对所有带 distance 的 item 进行混合排序
        if location.is_some() {
            items.sort_by(|a, b| {
                let dist_a = get_distance(a).unwrap_or(f64::MAX);
                let dist_b = get_distance(b).unwrap_or(f64::MAX);
                dist_a
                    .partial_cmp(&dist_b)
                    .unwrap_or(std::cmp::Ordering::Equal)
            });
        }

        Ok(SearchSuggestions { items })
    }

    async fn sign_moment_images(
        &self,
        images: &[String],
    ) -> Option<Vec<crate::dto::response::MomentImage>> {
        if images.is_empty() {
            return None;
        }
        let mut signed_images = Vec::with_capacity(images.len());
        for path in images {
            let url = match self.media_service.sign_url(path).await {
                Ok(u) => u,
                Err(e) => {
                    tracing::warn!("Failed to sign url for path {}: {}", path, e);
                    path.clone()
                }
            };
            signed_images.push(crate::dto::response::MomentImage {
                url,
                width: None,
                height: None,
                content_type: None,
            });
        }
        Some(signed_images)
    }
}

fn get_distance(item: &SearchSuggestItem) -> Option<f64> {
    match item {
        SearchSuggestItem::User(u) => u.distance,
        SearchSuggestItem::Group(g) => g.distance,
        SearchSuggestItem::Moment(m) => m.distance,
        SearchSuggestItem::Tag(_) => None, // Tags don't have distance
    }
}
