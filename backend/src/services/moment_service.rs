//! 动态瞬间服务层实现
//!
//! 核心业务逻辑：发布动态、搜索、互动状态管理

use std::sync::Arc;

use redis::aio::ConnectionManager;
use tokio::try_join;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreateMomentRequest, DeleteMomentRequest, EditMomentRequest, FavoriteMomentRequest,
            LikeMomentRequest, MomentLikesRequest, MomentOpenTheDoorRequest, MyFavoritesRequest,
            MyMomentsRequest, SearchMomentsRequest, UnfavoriteMomentRequest, UnlikeMomentRequest,
            UserMomentsRequest,
        },
        response::{
            MomentImage, MomentItem, MomentLikeItem, MyMomentState, PaginatedData,
            RelativeLocation, UserInfo,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::{
        moment_repo::{MomentDetailRow, MomentImageRow, MomentLikeRow, MomentRepository},
        tag_repo::TagRepository,
        user_repo::UserRepository,
        user_settings_repo::UserSettingsRepository,
    },
    types::enums::PrivacyLevel,
    utils::{cache::CacheManager, validation::Validator},
    EventBus,
};

#[derive(Clone)]
pub struct MomentService {
    _redis: ConnectionManager,
    event_bus: Arc<EventBus>,
    moment_repo: MomentRepository,
    tag_repo: TagRepository,
    user_repo: UserRepository,
    _user_settings_repo: UserSettingsRepository,
    media_service: crate::services::media_service::MediaService,
    _cache_manager: CacheManager,
}

impl MomentService {
    pub fn new(
        redis: ConnectionManager,
        event_bus: Arc<EventBus>,
        moment_repo: MomentRepository,
        tag_repo: TagRepository,
        user_repo: UserRepository,
        user_settings_repo: UserSettingsRepository,
        media_service: crate::services::media_service::MediaService,
        cache_manager: CacheManager,
    ) -> Self {
        Self {
            _redis: redis,
            event_bus,
            moment_repo,
            tag_repo,
            user_repo,
            _user_settings_repo: user_settings_repo,
            media_service,
            _cache_manager: cache_manager,
        }
    }

    /// 统一搜索生活瞬间（支持关键字/标签/位置）
    pub async fn search(
        &self,
        viewer_id: Option<i64>,
        req: &SearchMomentsRequest,
    ) -> Result<PaginatedData<MomentItem>> {
        Validator::validate_coordinates(
            req.reference_point.latitude,
            req.reference_point.longitude,
        )?;
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i64;
        let cursor = pagination.and_then(|p| p.cursor.clone());

        let (rows, next_cursor) = if let Some(radius) = req.radius {
            let radius =
                Validator::validate_radius_required(radius, crate::constants::MAX_SEARCH_RADIUS)?;
            self.moment_repo
                .search_by_location(
                    req.keyword.as_deref(),
                    req.reference_point.longitude,
                    req.reference_point.latitude,
                    radius,
                    limit,
                    cursor.clone(),
                )
                .await?
        } else if let Some(ref tag) = req.tag {
            Validator::validate_not_empty(tag, "tag")?;
            self.moment_repo
                .search_by_tag(
                    tag,
                    req.reference_point.longitude,
                    req.reference_point.latitude,
                    limit,
                    cursor.clone(),
                )
                .await?
        } else if let Some(ref keyword) = req.keyword {
            Validator::validate_not_empty(keyword, "keyword")?;
            self.moment_repo
                .search_by_keyword(
                    keyword,
                    req.reference_point.longitude,
                    req.reference_point.latitude,
                    limit,
                    cursor.clone(),
                )
                .await?
        } else {
            return Err(AppError::BadRequest(
                "至少需要提供 keyword、tag、radius 中的一个搜索条件".to_string(),
            ));
        };

        self.build_search_list(rows, viewer_id, next_cursor).await
    }

    /// 获取当前用户生活瞬间历史（/me/ API）
    pub async fn get_my_moments_list(
        &self,
        user_id: i64,
        req: &MyMomentsRequest,
    ) -> Result<PaginatedData<MomentItem>> {
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i64;
        let (rows, next_cursor) = self
            .moment_repo
            .list_by_user(user_id, limit, pagination.and_then(|p| p.cursor.clone()))
            .await?;

        self.build_user_moment_list(rows, next_cursor).await
    }

    /// 获取指定用户生活瞬间历史（公开 API）
    pub async fn get_user_moments_list(
        &self,
        req: &UserMomentsRequest,
    ) -> Result<PaginatedData<MomentItem>> {
        let user_id = crate::utils::validation::Validator::parse_user_id(&req.user_id)?;
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i64;
        let (rows, next_cursor) = self
            .moment_repo
            .list_by_user(user_id, limit, pagination.and_then(|p| p.cursor.clone()))
            .await?;

        self.build_user_moment_list(rows, next_cursor).await
    }

    /// 获取瞬间详情
    pub async fn open_the_door(
        &self,
        viewer_id: Option<i64>,
        req: &MomentOpenTheDoorRequest,
    ) -> Result<MomentItem> {
        let moment_id = Validator::parse_uuid(&req.moment_id, "moment_id")?;

        let detail = self
            .moment_repo
            .get_detail(moment_id, None, None)
            .await?
            .ok_or_else(|| AppError::MomentNotFound("瞬间不存在".to_string()))?;

        let ids = [detail.moment_id];
        let (tags_map, images_map) = try_join!(
            self.tag_repo.get_moment_tags_batch(&ids),
            self.moment_repo.list_images_batch(&ids)
        )?;

        let (liked, favorited) = if let Some(uid) = viewer_id {
            try_join!(
                self.moment_repo.is_liked(moment_id, uid),
                self.moment_repo.is_favorited(moment_id, uid)
            )?
        } else {
            (false, false)
        };

        Ok(self
            .build_search_item(
                detail,
                tags_map.get(&moment_id).cloned().unwrap_or_default(),
                images_map.get(&moment_id),
                viewer_id.map(|_| MyMomentState {
                    is_liked: liked,
                    is_favorited: favorited,
                }),
            )
            .await)
    }

    /// 获取生活瞬间点赞列表
    pub async fn get_likes_list(
        &self,
        req: &MomentLikesRequest,
    ) -> Result<PaginatedData<MomentLikeItem>> {
        let moment_id = Validator::parse_uuid(&req.moment_id, "moment_id")?;
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i64;

        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }

        let (rows, next_cursor) = self
            .moment_repo
            .list_likes(moment_id, limit, pagination.and_then(|p| p.cursor.clone()))
            .await?;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            items.push(self.build_like_item(row).await);
        }

        Ok(PaginatedData { items, next_cursor })
    }

    /// 创建瞬间
    pub async fn create_moment(
        &self,
        user_id: i64,
        req: &CreateMomentRequest,
    ) -> Result<MomentItem> {
        Validator::validate_not_empty(&req.description, "description")?;
        Validator::validate_coordinates(req.location.point.latitude, req.location.point.longitude)?;

        let privacy_level = req.privacy_level.unwrap_or(PrivacyLevel::Public);

        let mentions = Validator::parse_mentions(req.mentions.as_deref())?;

        let moment = self
            .moment_repo
            .create(
                user_id,
                Some(req.description.clone()),
                mentions,
                Some(req.location.point.longitude),
                Some(req.location.point.latitude),
                Some(req.location.location_name.clone()),
                privacy_level,
            )
            .await?;

        let mut image_rows: Vec<(Uuid, String, Option<i32>, Option<i32>, i32)> = Vec::new();
        if let Some(image_ids) = &req.image_file_ids {
            for (idx, fid) in image_ids.iter().enumerate() {
                let file_id = Validator::parse_uuid(fid, "image_file_id")?;
                let file = self
                    .media_service
                    .validate_file(file_id, user_id, "moment_image")
                    .await?;
                image_rows.push((file_id, file, None, None, idx as i32));
            }
        }

        if !image_rows.is_empty() {
            self.moment_repo
                .replace_images(moment.moment_id, &image_rows)
                .await?;
        }

        let tags = req.tags.clone().unwrap_or_default();
        if !tags.is_empty() {
            self.tag_repo
                .set_moment_tags(moment.moment_id, &tags)
                .await?;
        }

        self.event_bus.publish(AppEvent::MomentCreated {
            moment_id: moment.moment_id,
            user_id,
        });

        let images = if image_rows.is_empty() {
            None
        } else {
            let mut list = Vec::with_capacity(image_rows.len());
            for (_, url, width, height, _) in &image_rows {
                let signed = self.media_service.sign_url_if_path(Some(url)).await;
                list.push(MomentImage {
                    url: signed.unwrap_or_else(|| url.clone()),
                    thumbnail_url: None,
                    width: *width,
                    height: *height,
                    mime_type: None,
                });
            }
            Some(list)
        };

        let user_row = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

        let user = UserInfo::new(
            user_id,
            user_row.nickname.unwrap_or_default(),
            self.media_service
                .sign_url_if_path(user_row.avatar_url.as_deref())
                .await,
        );

        Ok(MomentItem {
            moment_id: moment.moment_id.to_string(),
            description: req.description.clone(),
            images,
            tags,
            mentions: moment.mentions.clone(),
            created_at: moment.created_at.to_rfc3339(),
            likes_count: moment.like_count,
            comments_count: moment.comments_count,
            shares_count: 0,
            user,
            location: RelativeLocation::without_distance(
                req.location.point.latitude,
                req.location.point.longitude,
                req.location.location_name.clone(),
            ),
            privacy_level: None,
            my_state: None,
            recommendation: None,
        })
    }

    /// 编辑瞬间，返回更新后的完整 MomentItem
    pub async fn update_moment(&self, user_id: i64, req: &EditMomentRequest) -> Result<MomentItem> {
        let moment_id = Validator::parse_uuid(&req.moment_id, "moment_id")?;

        let moment = self
            .moment_repo
            .find_by_id(moment_id)
            .await?
            .ok_or_else(|| AppError::MomentNotFound("瞬间不存在".to_string()))?;

        if moment.user_id != user_id {
            return Err(AppError::Forbidden("无权编辑他人的瞬间".to_string()));
        }

        if let Some(desc) = &req.description {
            if desc.trim().is_empty() {
                return Err(AppError::BadRequest("动态内容不能为空".to_string()));
            }
        }

        let (lon, lat, location_name) = if let Some(loc) = &req.location {
            Validator::validate_coordinates(loc.point.latitude, loc.point.longitude)?;
            (
                Some(loc.point.longitude),
                Some(loc.point.latitude),
                Some(loc.location_name.clone()),
            )
        } else {
            (None, None, None)
        };

        let privacy_level = req.privacy_level;
        let mentions = Validator::parse_mentions(req.mentions.as_deref())?;

        self.moment_repo
            .update(
                moment_id,
                req.description.clone(),
                mentions,
                lon,
                lat,
                location_name,
                privacy_level,
            )
            .await?;

        if let Some(tags) = &req.tags {
            self.tag_repo.set_moment_tags(moment_id, tags).await?;
        }

        if let Some(image_ids) = &req.image_file_ids {
            let mut image_rows: Vec<(Uuid, String, Option<i32>, Option<i32>, i32)> = Vec::new();
            for (idx, fid) in image_ids.iter().enumerate() {
                let file_id = Validator::parse_uuid(fid, "image_file_id")?;
                let file = self
                    .media_service
                    .validate_file(file_id, user_id, "moment_image")
                    .await?;
                image_rows.push((file_id, file, None, None, idx as i32));
            }
            self.moment_repo
                .replace_images(moment_id, &image_rows)
                .await?;
        }

        let detail = self
            .moment_repo
            .get_detail(moment_id, None, None)
            .await?
            .ok_or_else(|| AppError::MomentNotFound("瞬间不存在".to_string()))?;

        let ids = [detail.moment_id];
        let (tags_map, images_map) = try_join!(
            self.tag_repo.get_moment_tags_batch(&ids),
            self.moment_repo.list_images_batch(&ids)
        )?;

        let (liked, favorited) = try_join!(
            self.moment_repo.is_liked(moment_id, user_id),
            self.moment_repo.is_favorited(moment_id, user_id)
        )?;

        Ok(self
            .build_search_item(
                detail,
                tags_map.get(&moment_id).cloned().unwrap_or_default(),
                images_map.get(&moment_id),
                Some(MyMomentState {
                    is_liked: liked,
                    is_favorited: favorited,
                }),
            )
            .await)
    }

    /// 删除瞬间
    pub async fn delete_moment(&self, user_id: i64, req: &DeleteMomentRequest) -> Result<()> {
        let moment_id = Validator::parse_uuid(&req.moment_id, "moment_id")?;
        let moment = self
            .moment_repo
            .find_by_id(moment_id)
            .await?
            .ok_or_else(|| AppError::MomentNotFound("瞬间不存在".to_string()))?;

        if moment.user_id != user_id {
            return Err(AppError::Forbidden("无权删除他人的瞬间".to_string()));
        }

        self.moment_repo.delete(moment_id).await?;
        Ok(())
    }

    /// 点赞
    pub async fn like_moment(&self, user_id: i64, req: &LikeMomentRequest) -> Result<()> {
        let moment_id = Validator::parse_uuid(&req.moment_id, "moment_id")?;
        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }
        let _ = self.moment_repo.insert_like(moment_id, user_id).await?;
        Ok(())
    }

    /// 取消点赞
    pub async fn unlike_moment(&self, user_id: i64, req: &UnlikeMomentRequest) -> Result<()> {
        let moment_id = Validator::parse_uuid(&req.moment_id, "moment_id")?;
        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }
        self.moment_repo.delete_like(moment_id, user_id).await?;
        Ok(())
    }

    /// 收藏
    pub async fn favorite_moment(&self, user_id: i64, req: &FavoriteMomentRequest) -> Result<()> {
        let moment_id = Validator::parse_uuid(&req.moment_id, "moment_id")?;
        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }
        let _ = self.moment_repo.insert_favorite(moment_id, user_id).await?;
        Ok(())
    }

    /// 取消收藏
    pub async fn unfavorite_moment(
        &self,
        user_id: i64,
        req: &UnfavoriteMomentRequest,
    ) -> Result<()> {
        let moment_id = Validator::parse_uuid(&req.moment_id, "moment_id")?;
        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }
        self.moment_repo.delete_favorite(moment_id, user_id).await?;
        Ok(())
    }

    pub async fn get_favorites_list(
        &self,
        user_id: i64,
        req: &MyFavoritesRequest,
    ) -> Result<PaginatedData<MomentItem>> {
        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(pagination.and_then(|p| p.limit), 50)? as i64;
        let (rows, next_cursor) = self
            .moment_repo
            .list_favorited_moments(user_id, limit, pagination.and_then(|p| p.cursor.clone()))
            .await?;

        self.build_search_list(rows, Some(user_id), next_cursor)
            .await
    }

    fn build_location(
        latitude: Option<f64>,
        longitude: Option<f64>,
        location_name: Option<String>,
        distance: Option<f64>,
    ) -> RelativeLocation {
        RelativeLocation {
            point: crate::dto::response::GeoPoint {
                latitude: latitude.unwrap_or(0.0),
                longitude: longitude.unwrap_or(0.0),
            },
            location_name: location_name.unwrap_or_default(),
            distance,
            bearing_degrees: None,
        }
    }

    async fn build_like_item(&self, row: MomentLikeRow) -> MomentLikeItem {
        MomentLikeItem {
            user: UserInfo::new(
                row.user_id,
                row.user_nickname.unwrap_or_default(),
                self.media_service
                    .sign_url_if_path(row.user_avatar_url.as_deref())
                    .await,
            ),
            liked_at: row.created_at.to_rfc3339(),
        }
    }

    async fn build_search_list(
        &self,
        rows: Vec<MomentDetailRow>,
        viewer_id: Option<i64>,
        next_cursor: Option<String>,
    ) -> Result<PaginatedData<MomentItem>> {
        let moment_ids: Vec<Uuid> = rows.iter().map(|r| r.moment_id).collect();
        let (tags_map, images_map) = try_join!(
            self.tag_repo.get_moment_tags_batch(&moment_ids),
            self.moment_repo.list_images_batch(&moment_ids)
        )?;

        let (liked_set, favorited_set) = if let Some(uid) = viewer_id {
            let (liked, favorited) = try_join!(
                self.moment_repo.batch_check_liked(&moment_ids, uid),
                self.moment_repo.batch_check_favorited(&moment_ids, uid)
            )?;
            (Some(liked), Some(favorited))
        } else {
            (None, None)
        };

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            let tags = tags_map.get(&row.moment_id).cloned().unwrap_or_default();
            let images = images_map.get(&row.moment_id);
            let my_state = viewer_id.map(|_| MyMomentState {
                is_liked: liked_set
                    .as_ref()
                    .map(|s| s.contains(&row.moment_id))
                    .unwrap_or(false),
                is_favorited: favorited_set
                    .as_ref()
                    .map(|s| s.contains(&row.moment_id))
                    .unwrap_or(false),
            });

            items.push(self.build_search_item(row, tags, images, my_state).await);
        }

        Ok(PaginatedData { items, next_cursor })
    }

    async fn build_search_item(
        &self,
        detail: MomentDetailRow,
        tags: Vec<String>,
        images: Option<&Vec<MomentImageRow>>,
        my_state: Option<MyMomentState>,
    ) -> MomentItem {
        let image_list = match images {
            Some(rows) => self.sign_images(rows).await,
            None => None,
        };

        MomentItem {
            moment_id: detail.moment_id.to_string(),
            description: detail.description.unwrap_or_default(),
            images: image_list,
            tags,
            mentions: detail.mentions,
            created_at: detail.created_at.to_rfc3339(),
            likes_count: detail.like_count,
            comments_count: detail.comments_count,
            shares_count: 0,
            user: UserInfo::new(
                detail.user_id,
                detail.user_nickname.unwrap_or_default(),
                self.media_service
                    .sign_url_if_path(detail.user_avatar_url.as_deref())
                    .await,
            ),
            location: Self::build_location(
                detail.latitude,
                detail.longitude,
                detail.location_name,
                detail.distance,
            ),
            privacy_level: None,
            my_state,
            recommendation: None,
        }
    }

    async fn build_user_moment_list(
        &self,
        rows: Vec<MomentDetailRow>,
        next_cursor: Option<String>,
    ) -> Result<PaginatedData<MomentItem>> {
        let moment_ids: Vec<Uuid> = rows.iter().map(|r| r.moment_id).collect();
        let (tags_map, images_map) = try_join!(
            self.tag_repo.get_moment_tags_batch(&moment_ids),
            self.moment_repo.list_images_batch(&moment_ids)
        )?;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            let tags = tags_map.get(&row.moment_id).cloned().unwrap_or_default();
            let images = images_map.get(&row.moment_id);
            let image_list = match images {
                Some(img_rows) => self.sign_images(img_rows).await,
                None => None,
            };

            let user = UserInfo::new(
                row.user_id,
                row.user_nickname.unwrap_or_default(),
                self.media_service
                    .sign_url_if_path(row.user_avatar_url.as_deref())
                    .await,
            );

            items.push(MomentItem {
                moment_id: row.moment_id.to_string(),
                description: row.description.unwrap_or_default(),
                images: image_list,
                tags,
                mentions: row.mentions,
                created_at: row.created_at.to_rfc3339(),
                likes_count: row.like_count,
                comments_count: row.comments_count,
                shares_count: 0,
                user,
                location: Self::build_location(
                    row.latitude,
                    row.longitude,
                    row.location_name,
                    row.distance,
                ),
                privacy_level: None,
                my_state: None,
                recommendation: None,
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }

    async fn sign_images(&self, rows: &[MomentImageRow]) -> Option<Vec<MomentImage>> {
        if rows.is_empty() {
            return None;
        }

        let mut images = Vec::with_capacity(rows.len());
        for row in rows {
            let signed = self.media_service.sign_url_if_path(Some(&row.url)).await;
            images.push(MomentImage {
                url: signed.unwrap_or_else(|| row.url.clone()),
                thumbnail_url: None,
                width: row.width,
                height: row.height,
                mime_type: None,
            });
        }

        Some(images)
    }
}
