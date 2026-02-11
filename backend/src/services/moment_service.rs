//! 动态瞬间服务层实现
//!
//! 核心业务逻辑：发布动态、搜索、互动状态管理

use std::sync::Arc;

use redis::aio::ConnectionManager;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CheckFavoritedRequest, CheckLikedRequest, CreateMomentRequest, DeleteMomentRequest,
            EditMomentRequest, FavoriteMomentRequest, GetMomentInteractionStateRequest,
            LikeMomentRequest, MomentLikesRequest, MomentOpenTheDoorRequest, MyFavoritesRequest,
            MyMomentsRequest, SearchMomentsByKeywordRequest, SearchMomentsByLocationRequest,
            SearchMomentsByTagRequest, UnfavoriteMomentRequest, UnlikeMomentRequest,
            UserMomentsRequest,
        },
        response::{
            MomentImage, MomentInteractionStateData,
            MomentLikeItem, MomentSearchItem, MyMomentState, PaginatedData, PaginationMeta,
            RelativeLocation, UserInfo, UserMomentListItem,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::{
        moment_repo::{MomentDetailRow, MomentImageRow, MomentLikeRow, MomentListRow, MomentRepository},
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

    /// 按关键字搜索生活瞬间（全局）
    pub async fn search_by_keyword(
        &self,
        viewer_id: Option<i64>,
        req: &SearchMomentsByKeywordRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        Validator::validate_not_empty(&req.kw, "kw")?;
        Validator::validate_coordinates(req.reference_point.latitude, req.reference_point.longitude)?;
        let limit = Validator::validate_limit(req.limit, 50)? as i64;

        let (rows, next_cursor) = self
            .moment_repo
            .search_by_keyword(
                &req.kw,
                req.reference_point.longitude,
                req.reference_point.latitude,
                limit,
                req.cursor.clone(),
            )
            .await?;

        self.build_search_list(rows, viewer_id, next_cursor).await
    }

    /// 按标签搜索生活瞬间（全局）
    pub async fn search_by_tag(
        &self,
        viewer_id: Option<i64>,
        req: &SearchMomentsByTagRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        Validator::validate_not_empty(&req.tag, "tag")?;
        Validator::validate_coordinates(req.reference_point.latitude, req.reference_point.longitude)?;
        let limit = Validator::validate_limit(req.limit, 50)? as i64;

        let (rows, next_cursor) = self
            .moment_repo
            .search_by_tag(
                &req.tag,
                req.reference_point.longitude,
                req.reference_point.latitude,
                limit,
                req.cursor.clone(),
            )
            .await?;

        self.build_search_list(rows, viewer_id, next_cursor).await
    }

    /// 按位置搜索生活瞬间（范围过滤）
    pub async fn search_by_location(
        &self,
        viewer_id: Option<i64>,
        req: &SearchMomentsByLocationRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        Validator::validate_coordinates(req.reference_point.latitude, req.reference_point.longitude)?;
        let radius = Validator::validate_radius_required(req.radius, 50000.0)?;
        let limit = Validator::validate_limit(req.limit, 50)? as i64;

        let (rows, next_cursor) = self
            .moment_repo
            .search_by_location(
                req.kw.as_deref(),
                req.reference_point.longitude,
                req.reference_point.latitude,
                radius,
                limit,
                req.cursor.clone(),
            )
            .await?;

        self.build_search_list(rows, viewer_id, next_cursor).await
    }

    /// 获取当前用户生活瞬间历史（/me/ API）
    pub async fn get_my_moments_list(
        &self,
        user_id: i64,
        req: &MyMomentsRequest,
    ) -> Result<PaginatedData<UserMomentListItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let (rows, next_cursor) = self
            .moment_repo
            .list_by_user(user_id, limit, req.cursor.clone())
            .await?;

        self.build_user_moment_list(rows, next_cursor).await
    }

    /// 获取指定用户生活瞬间历史（公开 API）
    pub async fn get_user_moments_list(
        &self,
        req: &UserMomentsRequest,
    ) -> Result<PaginatedData<UserMomentListItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let (rows, next_cursor) = self
            .moment_repo
            .list_by_user(req.user_id, limit, req.cursor.clone())
            .await?;

        self.build_user_moment_list(rows, next_cursor).await
    }

    /// 获取瞬间详情
    pub async fn open_the_door(
        &self,
        viewer_id: Option<i64>,
        req: &MomentOpenTheDoorRequest,
    ) -> Result<MomentSearchItem> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::InvalidFormat("无效的 moment_id".to_string()))?;

        let detail = self
            .moment_repo
            .get_detail(moment_id, None, None)
            .await?
            .ok_or_else(|| AppError::MomentNotFound("瞬间不存在".to_string()))?;

        let tags_map = self
            .tag_repo
            .get_moment_tags_batch(&[detail.moment_id])
            .await?;
        let images_map = self
            .moment_repo
            .list_images_batch(&[detail.moment_id])
            .await?;

        let liked = if let Some(uid) = viewer_id {
            self.moment_repo.is_liked(moment_id, uid).await?
        } else {
            false
        };
        let favorited = if let Some(uid) = viewer_id {
            self.moment_repo.is_favorited(moment_id, uid).await?
        } else {
            false
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
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::InvalidFormat("无效的 moment_id".to_string()))?;
        let limit = Validator::validate_limit(req.limit, 50)? as i64;

        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }

        let (rows, next_cursor) = self
            .moment_repo
            .list_likes(moment_id, limit, req.cursor.clone())
            .await?;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            items.push(self.build_like_item(row).await);
        }

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    /// 创建瞬间
    pub async fn create_moment(
        &self,
        user_id: i64,
        req: &CreateMomentRequest,
    ) -> Result<UserMomentListItem> {
        Validator::validate_not_empty(&req.description, "description")?;
        Validator::validate_coordinates(req.location.latitude, req.location.longitude)?;

        let privacy_level = req
            .privacy_level
            .unwrap_or(PrivacyLevel::Public)
            .to_string();

        let moment = self
            .moment_repo
            .create(
                user_id,
                Some(req.description.clone()),
                req.mentions.clone(),
                Some(req.location.longitude),
                Some(req.location.latitude),
                Some(req.location.location_name.clone()),
                privacy_level,
            )
            .await?;

        let mut image_rows: Vec<(Uuid, String, Option<i32>, Option<i32>, i32)> = Vec::new();
        if let Some(image_ids) = &req.image_file_ids {
            for (idx, fid) in image_ids.iter().enumerate() {
                let file_id = Uuid::parse_str(fid)
                    .map_err(|_| AppError::InvalidFormat("无效的 image_file_id".to_string()))?;
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
                    width: *width,
                    height: *height,
                    content_type: None,
                });
            }
            Some(list)
        };

        Ok(UserMomentListItem {
            moment_id: moment.moment_id.to_string(),
            description: req.description.clone(),
            images,
            tags,
            mentions: req.mentions.clone(),
            created_at: moment.created_at.to_rfc3339(),
            likes_count: moment.like_count,
            comments_count: moment.comments_count,
            location: RelativeLocation {
                latitude: req.location.latitude,
                longitude: req.location.longitude,
                location_name: req.location.location_name.clone(),
                distance: None,
                bearing_degrees: None,
            },
        })
    }

    /// 编辑瞬间
    pub async fn update_moment(&self, user_id: i64, req: &EditMomentRequest) -> Result<()> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::InvalidFormat("无效的 moment_id".to_string()))?;

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
            Validator::validate_coordinates(loc.latitude, loc.longitude)?;
            (Some(loc.longitude), Some(loc.latitude), Some(loc.location_name.clone()))
        } else {
            (None, None, None)
        };

        let privacy_level = req.privacy_level.map(|p| p.to_string());

        self.moment_repo
            .update(
                moment_id,
                req.description.clone(),
                req.mentions.clone(),
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
                let file_id = Uuid::parse_str(fid)
                    .map_err(|_| AppError::InvalidFormat("无效的 image_file_id".to_string()))?;
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

        Ok(())
    }

    /// 删除瞬间
    pub async fn delete_moment(&self, user_id: i64, req: &DeleteMomentRequest) -> Result<()> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::InvalidFormat("无效的 moment_id".to_string()))?;
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
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::InvalidFormat("无效的 moment_id".to_string()))?;
        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }
        let _ = self.moment_repo.insert_like(moment_id, user_id).await?;
        Ok(())
    }

    /// 取消点赞
    pub async fn unlike_moment(&self, user_id: i64, req: &UnlikeMomentRequest) -> Result<()> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::InvalidFormat("无效的 moment_id".to_string()))?;
        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }
        self.moment_repo.delete_like(moment_id, user_id).await?;
        Ok(())
    }

    pub async fn check_liked(&self, user_id: i64, req: &CheckLikedRequest) -> Result<bool> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::InvalidFormat("无效的 moment_id".to_string()))?;
        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }
        self.moment_repo.is_liked(moment_id, user_id).await
    }

    /// 收藏
    pub async fn favorite_moment(&self, user_id: i64, req: &FavoriteMomentRequest) -> Result<()> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::InvalidFormat("无效的 moment_id".to_string()))?;
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
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::InvalidFormat("无效的 moment_id".to_string()))?;
        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }
        self.moment_repo.delete_favorite(moment_id, user_id).await?;
        Ok(())
    }

    pub async fn check_favorited(
        &self,
        user_id: i64,
        req: &CheckFavoritedRequest,
    ) -> Result<bool> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::InvalidFormat("无效的 moment_id".to_string()))?;
        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }
        self.moment_repo.is_favorited(moment_id, user_id).await
    }

    /// 获取瞬间互动状态（点赞+收藏）
    pub async fn get_interaction_state(
        &self,
        user_id: i64,
        req: &GetMomentInteractionStateRequest,
    ) -> Result<MomentInteractionStateData> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::InvalidFormat("无效的 moment_id".to_string()))?;
        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }

        let is_liked = self.moment_repo.is_liked(moment_id, user_id).await?;
        let is_favorited = self.moment_repo.is_favorited(moment_id, user_id).await?;
        Ok(MomentInteractionStateData {
            is_liked,
            is_favorited,
        })
    }

    pub async fn get_favorites_list(
        &self,
        user_id: i64,
        req: &MyFavoritesRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;
        let (rows, next_cursor) = self
            .moment_repo
            .list_favorited_moments(user_id, limit, req.cursor.clone())
            .await?;

        self.build_search_list(rows, Some(user_id), next_cursor).await
    }

    fn build_location(
        latitude: Option<f64>,
        longitude: Option<f64>,
        location_name: Option<String>,
        distance: Option<f64>,
    ) -> RelativeLocation {
        RelativeLocation {
            latitude: latitude.unwrap_or(0.0),
            longitude: longitude.unwrap_or(0.0),
            location_name: location_name.unwrap_or_default(),
            distance,
            bearing_degrees: None,
        }
    }

    async fn build_like_item(&self, row: MomentLikeRow) -> MomentLikeItem {
        MomentLikeItem {
            user: UserInfo {
                user_id: row.user_id,
                nickname: row.user_nickname.unwrap_or_default(),
                avatar_url: self
                    .media_service
                    .sign_url_if_path(row.user_avatar_url.as_deref())
                    .await,
                user_type: crate::types::enums::UserType::Normal,
            },
            liked_at: row.created_at.to_rfc3339(),
        }
    }

    async fn build_search_list(
        &self,
        rows: Vec<MomentDetailRow>,
        viewer_id: Option<i64>,
        next_cursor: Option<String>,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        let moment_ids: Vec<Uuid> = rows.iter().map(|r| r.moment_id).collect();
        let tags_map = self.tag_repo.get_moment_tags_batch(&moment_ids).await?;
        let images_map = self.moment_repo.list_images_batch(&moment_ids).await?;

        let (liked_set, favorited_set) = if let Some(uid) = viewer_id {
            let liked = self.moment_repo.batch_check_liked(&moment_ids, uid).await?;
            let favorited = self.moment_repo.batch_check_favorited(&moment_ids, uid).await?;
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

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    async fn build_search_item(
        &self,
        detail: MomentDetailRow,
        tags: Vec<String>,
        images: Option<&Vec<MomentImageRow>>,
        my_state: Option<MyMomentState>,
    ) -> MomentSearchItem {
        let image_list = match images {
            Some(rows) => self.sign_images(rows).await,
            None => None,
        };

        MomentSearchItem {
            moment_id: detail.moment_id.to_string(),
            user: UserInfo {
                user_id: detail.user_id,
                nickname: detail.user_nickname.unwrap_or_default(),
                avatar_url: self
                    .media_service
                    .sign_url_if_path(detail.user_avatar_url.as_deref())
                    .await,
                user_type: crate::types::enums::UserType::Normal,
            },
            description: detail.description.unwrap_or_default(),
            images: image_list,
            tags,
            mentions: detail.mentions,
            created_at: detail.created_at.to_rfc3339(),
            likes_count: detail.like_count,
            comments_count: detail.comments_count,
            location: Self::build_location(
                detail.latitude,
                detail.longitude,
                detail.location_name,
                detail.distance,
            ),
            my_state,
        }
    }

    async fn build_user_moment_list(
        &self,
        rows: Vec<MomentListRow>,
        next_cursor: Option<String>,
    ) -> Result<PaginatedData<UserMomentListItem>> {
        let moment_ids: Vec<Uuid> = rows.iter().map(|r| r.moment_id).collect();
        let tags_map = self.tag_repo.get_moment_tags_batch(&moment_ids).await?;
        let images_map = self.moment_repo.list_images_batch(&moment_ids).await?;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            let tags = tags_map.get(&row.moment_id).cloned().unwrap_or_default();
            let images = images_map.get(&row.moment_id);
            let image_list = match images {
                Some(rows) => self.sign_images(rows).await,
                None => None,
            };

            items.push(UserMomentListItem {
                moment_id: row.moment_id.to_string(),
                description: row.description.unwrap_or_default(),
                images: image_list,
                tags,
                mentions: row.mentions,
                created_at: row.created_at.to_rfc3339(),
                likes_count: row.like_count,
                comments_count: row.comments_count,
                location: Self::build_location(
                    row.latitude,
                    row.longitude,
                    row.location_name,
                    None,
                ),
            });
        }

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
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
                width: row.width,
                height: row.height,
                content_type: None,
            });
        }

        Some(images)
    }
}
