use std::collections::HashSet;

use crate::{
    constants::MAX_SEARCH_RADIUS_I32,
    dto::{
        request::{SearchSuggestRequest, SearchSuggestType},
        response::{
            GroupSuggestItem, MomentSuggestItem, RelativeLocation, SearchSuggestItem,
            SearchSuggestions, TagSuggestItem, UserInfo, UserSuggestItem,
        },
    },
    error::AppError,
    repositories::{
        group_repo::GroupRepository, moment_repo::MomentRepository, tag_repo::TagRepository,
        user_repo::UserRepository,
    },
    utils::validation::Validator,
};

#[derive(Clone)]
pub struct SearchService {
    user_repo: UserRepository,
    group_repo: GroupRepository,
    tag_repo: TagRepository,
    moment_repo: MomentRepository,
    media_service: crate::services::media_service::MediaService,
}

impl SearchService {
    pub fn new(
        user_repo: UserRepository,
        group_repo: GroupRepository,
        tag_repo: TagRepository,
        moment_repo: MomentRepository,
        media_service: crate::services::media_service::MediaService,
    ) -> Self {
        Self {
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

        // 解析位置参数（reference_point 必填）
        Validator::validate_coordinates(
            req.reference_point.latitude,
            req.reference_point.longitude,
        )?;
        let location = Some((req.reference_point.latitude, req.reference_point.longitude));
        let radius_opt = if let Some(r) = req.radius {
            Some(Validator::validate_radius(Some(r), MAX_SEARCH_RADIUS_I32)? as f64)
        } else {
            None
        };

        let type_set: Option<HashSet<SearchSuggestType>> = req
            .types
            .as_ref()
            .map(|list| list.iter().copied().collect());
        let include_tag = should_include_type(&type_set, SearchSuggestType::Tag);
        let include_user = should_include_type(&type_set, SearchSuggestType::User);
        let include_group = should_include_type(&type_set, SearchSuggestType::Group);
        let include_moment = should_include_type(&type_set, SearchSuggestType::Moment);

        let tag_rows = if include_tag {
            self.tag_repo.suggest(&req.prefix, limit).await?
        } else {
            Vec::new()
        };
        let user_rows = if include_user {
            self.user_repo
                .suggest_by_nickname(&req.prefix, limit, location, radius_opt)
                .await?
        } else {
            Vec::new()
        };
        let group_rows = if include_group {
            self.group_repo
                .suggest_by_name(&req.prefix, limit, location, radius_opt)
                .await?
        } else {
            Vec::new()
        };
        let moment_rows = if include_moment {
            self.moment_repo
                .suggest_by_keyword(&req.prefix, viewer_id, limit as i32, location, radius_opt)
                .await?
        } else {
            Vec::new()
        };

        let mut items: Vec<SearchSuggestItem> = Vec::new();

        // 1. 标签建议
        for tag in tag_rows {
            items.push(SearchSuggestItem::Tag(TagSuggestItem {
                name: tag.name,
                usage_count: tag.usage_count.unwrap_or(0),
            }));
        }

        // 2. 用户建议
        for user in user_rows {
            let avatar_url = self
                .media_service
                .sign_url_if_path(user.avatar_url.as_deref())
                .await;
            let bio = if user.bio.trim().is_empty() {
                None
            } else {
                Some(user.bio)
            };

            let location = build_relative_location(
                user.distance,
                user.latitude,
                user.longitude,
                user.location_name.as_deref(),
                user.bearing_degrees,
                req.reference_point.latitude,
                req.reference_point.longitude,
            );
            items.push(SearchSuggestItem::User(UserSuggestItem {
                user_id: user.user_id,
                nickname: user.nickname,
                avatar_url,
                bio,
                location,
            }));
        }

        // 3. 群组建议
        for group in group_rows {
            let avatar_url = self
                .media_service
                .sign_url_if_path(group.avatar_url.as_deref())
                .await;
            let description = if group.description.trim().is_empty() {
                None
            } else {
                Some(group.description)
            };
            let location = build_relative_location(
                group.distance,
                group.latitude,
                group.longitude,
                group.location_name.as_deref(),
                group.bearing_degrees,
                req.reference_point.latitude,
                req.reference_point.longitude,
            );

            items.push(SearchSuggestItem::Group(GroupSuggestItem {
                group_id: group.group_id.to_string(),
                name: group.name,
                avatar_url,
                description,
                members_count: group.member_count,
                location,
            }));
        }

        // 4. 瞬间建议
        for moment in moment_rows {
            let author_avatar_url = self
                .media_service
                .sign_url_if_path(moment.avatar_url.as_deref())
                .await;

            // 限制内容长度为前100个字符
            let description_preview = if moment.description.chars().count() > 100 {
                moment.description.chars().take(100).collect::<String>() + "..."
            } else {
                moment.description
            };

            let author = UserInfo::new(moment.user_id, moment.nickname, author_avatar_url);

            let location = build_relative_location(
                moment.distance,
                moment.latitude,
                moment.longitude,
                moment.location_name.as_deref(),
                moment.bearing_degrees,
                req.reference_point.latitude,
                req.reference_point.longitude,
            );

            items.push(SearchSuggestItem::Moment(MomentSuggestItem {
                moment_id: moment.moment_id.to_string(),
                description_preview,
                author,
                location,
            }));
        }

        // 排序逻辑：按 distance 排序
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
}

fn get_distance(item: &SearchSuggestItem) -> Option<f64> {
    match item {
        SearchSuggestItem::User(u) => u.location.as_ref().and_then(|l| l.distance),
        SearchSuggestItem::Group(g) => g.location.as_ref().and_then(|l| l.distance),
        SearchSuggestItem::Moment(m) => m.location.as_ref().and_then(|l| l.distance),
        SearchSuggestItem::Tag(_) => None,
    }
}

fn build_relative_location(
    distance: Option<f64>,
    latitude: Option<f64>,
    longitude: Option<f64>,
    location_name: Option<&str>,
    bearing_degrees: Option<f64>,
    ref_lat: f64,
    ref_lon: f64,
) -> Option<RelativeLocation> {
    distance.map(|d| {
        let lat = latitude.unwrap_or(ref_lat);
        let lon = longitude.unwrap_or(ref_lon);
        let name = location_name.unwrap_or("附近");
        RelativeLocation::with_distance(lat, lon, name, d, bearing_degrees.unwrap_or(0.0))
    })
}

fn should_include_type(set: &Option<HashSet<SearchSuggestType>>, ty: SearchSuggestType) -> bool {
    set.as_ref()
        .map_or(true, |s| s.is_empty() || s.contains(&ty))
}
