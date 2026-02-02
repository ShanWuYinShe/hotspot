use redis::aio::ConnectionManager as RedisConnectionManager;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    constants::{DEFAULT_LATITUDE, DEFAULT_LONGITUDE},
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
            MomentInteractionStateData, MomentLikeItem, MomentSearchItem, MyMomentState,
            PaginatedData, PaginationMeta, RelativeLocation, UserInfo, UserMomentListItem,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::moment_repo::{MomentRepository, MomentSearchRow},
    repositories::tag_repo::TagRepository,
    repositories::user_repo::UserRepository,
    repositories::user_settings_repo::UserSettingsRepository,
    services::media_service::MediaService,
    types::enums::PrivacyLevel,
    utils::{
        bloom_filter::ResourceExistenceChecker,
        cache::{cache_keys, CacheManager},
        validation::Validator,
    },
    EventBus,
};

#[derive(Clone)]
pub struct MomentService {
    _redis: RedisConnectionManager,
    event_bus: Arc<EventBus>,
    moment_repo: MomentRepository,
    tag_repo: TagRepository,
    user_repo: UserRepository,
    user_settings_repo: UserSettingsRepository,
    media_service: MediaService,
    cache_manager: CacheManager,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct MomentDetailCache {
    moment_id: String,
    user_id: i64,
    nickname: String,
    avatar_url: Option<String>,
    user_type: crate::types::enums::UserType,
    description: String,
    images: Vec<String>,
    tags: Vec<String>,
    mentions: Option<Vec<i64>>,
    created_at: String,
    likes_count: i32,
    comments_count: i32,
    latitude: f64,
    longitude: f64,
    location_name: String,
    privacy_level: String,
}

impl MomentService {
    pub fn new(
        redis: RedisConnectionManager,
        event_bus: Arc<EventBus>,
        moment_repo: MomentRepository,
        tag_repo: TagRepository,
        user_repo: UserRepository,
        user_settings_repo: UserSettingsRepository,
        media_service: MediaService,
        cache_manager: CacheManager,
    ) -> Self {
        Self {
            _redis: redis,
            event_bus,
            moment_repo,
            tag_repo,
            user_repo,
            user_settings_repo,
            media_service,
            cache_manager,
        }
    }

    /// 创建瞬间
    pub async fn create_moment(
        &self,
        user_id: i64,
        req: &CreateMomentRequest,
    ) -> Result<UserMomentListItem> {
        // 验证坐标
        if req.location.latitude < -90.0 || req.location.latitude > 90.0 {
            return Err(AppError::InvalidFormat("纬度必须在-90到90之间".to_string()));
        }
        if req.location.longitude < -180.0 || req.location.longitude > 180.0 {
            return Err(AppError::InvalidFormat(
                "经度必须在-180到180之间".to_string(),
            ));
        }

        // 验证内容长度
        if req.description.is_empty() {
            return Err(AppError::BadRequest("内容不能为空".to_string()));
        }

        if req.description.len() > 5000 {
            return Err(AppError::BadRequest("内容过长".to_string()));
        }

        // 验证 mentions（最多 10 个）
        if let Some(mentions) = &req.mentions {
            if mentions.len() > 10 {
                return Err(AppError::BadRequest("最多只能提及10个用户".to_string()));
            }
            // 验证所有被提及的用户是否存在
            for &mentioned_user_id in mentions {
                if !self.user_repo.exists(mentioned_user_id).await? {
                    return Err(AppError::UserNotFound(format!(
                        "被提及的用户 {} 不存在",
                        mentioned_user_id
                    )));
                }
            }
        }

        // 确定隐私级别：未提供时使用用户全局设置 moments_visible
        // privacy_level 已是强类型枚举
        let privacy_level = if let Some(level) = req.privacy_level {
            level
        } else {
            // 获取用户全局设置
            let settings = self.user_settings_repo.find_by_user_id(user_id).await?;
            if let Some(settings) = settings {
                settings.moments_visible
            } else {
                // 默认为 public
                PrivacyLevel::Public
            }
        };

        // 解析并验证图片 file_id -> (url, path)
        // 重要：不要在 moment 创建成功前标记 is_used，避免“先占用、后失败”导致脏数据。
        let (storage_paths, signed_urls, image_uuids) = if let Some(file_ids) = &req.image_file_ids
        {
            if file_ids.len() > 9 {
                return Err(AppError::BadRequest("最多只能上传9张图片".to_string()));
            }
            let mut paths = Vec::with_capacity(file_ids.len());
            let mut urls = Vec::with_capacity(file_ids.len());
            let mut uuids = Vec::with_capacity(file_ids.len());
            for file_id in file_ids {
                let uuid = Uuid::parse_str(file_id)
                    .map_err(|_| AppError::BadRequest("无效的图片 file_id".to_string()))?;
                let file_data = self
                    .media_service
                    .validate_file(uuid, user_id, "moment_image")
                    .await?;
                uuids.push(uuid);
                paths.push(file_data.clone());
                urls.push(file_data);
            }
            (Some(paths), Some(urls), uuids)
        } else {
            (None, None, Vec::new())
        };

        // 创建瞬间 - 将 PrivacyLevel 转换为字符串
        let moment = self
            .moment_repo
            .create(
                user_id,
                req.description.clone(),
                storage_paths.clone(),
                req.mentions.clone(),
                privacy_level.as_str().to_string(),
                Some(req.location.latitude),
                Some(req.location.longitude),
                Some(req.location.location_name.clone()),
            )
            .await?;
        if let Err(e) = ResourceExistenceChecker::new(self._redis.clone())
            .record_moment(&moment.moment_id.to_string())
            .await
        {
            tracing::warn!(moment_id = %moment.moment_id, error = %e, "record_moment bloom failed");
        }

        // moment 已创建成功后再标记文件已使用（避免创建失败导致 is_used 脏数据）
        for file_id in image_uuids {
            if let Err(e) = self.media_service.mark_as_used(file_id).await {
                tracing::warn!(
                    user_id,
                    file_id = %file_id,
                    error = %e,
                    "mark_as_used failed after moment created"
                );
            }
        }

        // 设置标签（通过关联表）
        let tags = if let Some(tags) = &req.tags {
            if !tags.is_empty() {
                self.tag_repo
                    .set_moment_tags(moment.moment_id, tags)
                    .await?;
            }
            tags.clone()
        } else {
            Vec::new()
        };

        // 构造响应
        use crate::dto::response::MomentImage;
        let moment_images: Option<Vec<MomentImage>> = if let Some(urls) = signed_urls {
            let mut images = Vec::with_capacity(urls.len());
            for url in urls {
                let signed_url = match self.media_service.sign_url(&url).await {
                    Ok(u) => u,
                    Err(e) => {
                        tracing::warn!("Failed to sign url for path {}: {}", url, e);
                        url.clone()
                    }
                };
                images.push(MomentImage {
                    url: signed_url,
                    width: None,
                    height: None,
                    content_type: None,
                });
            }
            Some(images)
        } else {
            None
        };

        // 触发 @ 提及事件
        if let Some(mentions) = &req.mentions {
            for &mentioned_user_id in mentions {
                if mentioned_user_id != user_id {
                    // 发送提及通知
                    let _ = self.event_bus.publish(AppEvent::MomentMentioned {
                        moment_id: moment.moment_id,
                        mentioned_user_id,
                        mentioning_user_id: user_id,
                        moment_text_preview: if moment.description.len() > 50 {
                            format!("{}...", &moment.description[..50])
                        } else {
                            moment.description.clone()
                        },
                        moment_cover_url: moment_images.as_ref().and_then(
                            |imgs: &Vec<MomentImage>| imgs.first().map(|img| img.url.clone()),
                        ),
                    });
                }
            }
        }

        Ok(UserMomentListItem {
            moment_id: moment.moment_id.to_string(),
            description: moment.description.clone(),
            images: moment_images,
            mentions: moment.mentions.clone(),
            tags,
            created_at: moment.created_at.to_rfc3339(),
            likes_count: 0,
            comments_count: 0,
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
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        match self.moment_repo.get_delete_status(moment_id).await? {
            None => return Err(AppError::MomentNotFound("瞬间不存在".to_string())),
            Some(true) => return Err(AppError::AlreadyDeleted("瞬间已删除".to_string())),
            Some(false) => {}
        }

        // 验证所有权
        let owner_id = self
            .get_moment_owner_cached(moment_id)
            .await?
            .ok_or_else(|| AppError::MomentNotFound("瞬间不存在".to_string()))?;

        if owner_id != user_id {
            return Err(AppError::Forbidden("无权编辑该瞬间".to_string()));
        }

        // 验证 mentions（最多 10 个）
        if let Some(mentions) = &req.mentions {
            if mentions.len() > 10 {
                return Err(AppError::BadRequest("最多只能提及10个用户".to_string()));
            }
            // 验证所有被提及的用户是否存在
            for &mentioned_user_id in mentions {
                if !self.user_repo.exists(mentioned_user_id).await? {
                    return Err(AppError::UserNotFound(format!(
                        "被提及的用户 {} 不存在",
                        mentioned_user_id
                    )));
                }
            }
        }

        // 验证隐私级别（如果提供）
        if let Some(level) = &req.privacy_level {
            match level.as_str() {
                "public" | "friends" | "private" => {}
                _ => return Err(AppError::BadRequest("无效的隐私级别".to_string())),
            }
        }

        // 更新瞬间基本信息
        // 重要：不要在 moment 更新成功前标记 is_used，避免"先占用、后失败"导致脏数据。
        let (images, image_uuids) = if let Some(file_ids) = &req.image_file_ids {
            if file_ids.len() > 9 {
                return Err(AppError::BadRequest("最多只能上传9张图片".to_string()));
            }
            let mut urls = Vec::with_capacity(file_ids.len());
            let mut uuids = Vec::with_capacity(file_ids.len());
            for file_id in file_ids {
                let uuid = Uuid::parse_str(file_id)
                    .map_err(|_| AppError::BadRequest("无效的图片 file_id".to_string()))?;
                let file_data = self
                    .media_service
                    .validate_file(uuid, user_id, "moment_image")
                    .await?;
                uuids.push(uuid);
                urls.push(file_data);
            }
            (Some(urls), uuids)
        } else {
            (None, Vec::new())
        };

        // privacy_level 已是强类型枚举，直接使用
        let privacy_level = req.privacy_level;

        self.moment_repo
            .update(
                moment_id,
                req.description.clone(),
                images,
                req.mentions.clone(),
                privacy_level,
            )
            .await?;

        // moment 更新成功后再标记文件已使用（避免更新失败导致 is_used 脏数据）
        for file_id in image_uuids {
            if let Err(e) = self.media_service.mark_as_used(file_id).await {
                tracing::warn!(
                    user_id,
                    moment_id = %moment_id,
                    file_id = %file_id,
                    error = %e,
                    "mark_as_used failed after moment updated"
                );
            }
        }

        // 更新位置（可选）
        if let Some(location) = &req.location {
            Validator::validate_coordinates(location.latitude, location.longitude)?;
            self.moment_repo
                .update_location(
                    moment_id,
                    location.latitude,
                    location.longitude,
                    location.location_name.clone(),
                )
                .await?;
        }

        // 更新标签（通过关联表）
        if let Some(tags) = &req.tags {
            self.tag_repo.set_moment_tags(moment_id, tags).await?;
        }

        Ok(())
    }

    /// 删除瞬间
    pub async fn delete_moment(&self, user_id: i64, req: &DeleteMomentRequest) -> Result<()> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }

        // 验证所有权
        let owner_id = self
            .get_moment_owner_cached(moment_id)
            .await?
            .ok_or_else(|| AppError::MomentNotFound("瞬间不存在".to_string()))?;

        if owner_id != user_id {
            return Err(AppError::Forbidden("无权删除该瞬间".to_string()));
        }

        self.moment_repo.soft_delete(moment_id).await?;

        Ok(())
    }

    /// 按关键字搜索生活瞬间（全局）
    pub async fn search_by_keyword(
        &self,
        viewer_id: Option<i64>,
        req: &SearchMomentsByKeywordRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor = req.cursor.clone();

        // 按关键字搜索
        let (moments, next_cursor) = self
            .moment_repo
            .search_by_keyword(
                &req.kw,
                req.reference_point.latitude,
                req.reference_point.longitude,
                viewer_id,
                limit,
                cursor,
            )
            .await?;

        self.build_search_results_from_rows(moments, next_cursor)
            .await
    }

    /// 按标签搜索生活瞬间（全局）
    pub async fn search_by_tag(
        &self,
        viewer_id: Option<i64>,
        req: &SearchMomentsByTagRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor = req.cursor.clone();

        // search_by_tag 返回包含 distance/bearing_degrees 的 MomentSearchRow
        let (moments, next_cursor) = self
            .moment_repo
            .search_by_tag(
                &req.tag,
                req.reference_point.latitude,
                req.reference_point.longitude,
                viewer_id,
                limit,
                cursor,
            )
            .await?;

        self.build_search_results_from_rows(moments, next_cursor)
            .await
    }

    /// 按位置搜索生活瞬间（范围过滤，距离由 PostGIS 计算）
    pub async fn search_by_location(
        &self,
        viewer_id: Option<i64>,
        req: &SearchMomentsByLocationRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        Validator::validate_coordinates(
            req.reference_point.latitude,
            req.reference_point.longitude,
        )?;
        Validator::validate_radius_required(req.radius, 50000.0)?;

        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor = req.cursor.clone();

        let (moments, next_cursor) = self
            .moment_repo
            .search_by_location(
                req.kw.as_deref(),
                req.reference_point.latitude,
                req.reference_point.longitude,
                viewer_id,
                req.radius,
                limit,
                cursor,
            )
            .await?;

        self.build_search_results_from_rows(moments, next_cursor)
            .await
    }

    /// 获取瞬间详情（通过 ID 获取指定瞬间的详情）
    pub async fn open_the_door(
        &self,
        viewer_id: Option<i64>,
        req: &MomentOpenTheDoorRequest,
    ) -> Result<MomentSearchItem> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        let moment_data = self.get_moment_detail_cached(moment_id, viewer_id).await?;
        let images = self.sign_moment_images(&moment_data.images).await;
        let my_state = if let Some(uid) = viewer_id {
            let (is_liked, is_favorited) = tokio::try_join!(
                self.moment_repo.is_liked_by(moment_id, uid),
                self.moment_repo.is_favorited_by(moment_id, uid)
            )?;
            Some(MyMomentState {
                is_liked,
                is_favorited,
            })
        } else {
            None
        };

        Ok(MomentSearchItem {
            moment_id: moment_data.moment_id,
            user: UserInfo {
                user_id: moment_data.user_id,
                nickname: moment_data.nickname.clone(),
                avatar_url: self
                    .media_service
                    .sign_url_if_path(moment_data.avatar_url.as_deref())
                    .await,
                user_type: moment_data.user_type,
            },
            description: moment_data.description.clone(),
            images,
            tags: moment_data.tags.clone(),
            mentions: moment_data.mentions.clone(),
            created_at: moment_data.created_at.clone(),
            likes_count: moment_data.likes_count,
            comments_count: moment_data.comments_count,
            location: RelativeLocation {
                latitude: moment_data.latitude,
                longitude: moment_data.longitude,
                location_name: moment_data.location_name.clone(),
                distance: None, // 详情场景不需要距离
                bearing_degrees: None,
            },
            my_state,
        })
    }

    async fn get_moment_detail_cached(
        &self,
        moment_id: Uuid,
        viewer_id: Option<i64>,
    ) -> Result<MomentDetailCache> {
        let cache_key = moment_id.to_string();
        if let Ok(Some(Some(cached))) = self
            .cache_manager
            .get::<MomentDetailCache>(cache_keys::MOMENT_DETAIL, &cache_key)
            .await
        {
            return Ok(cached);
        }

        let moments = self
            .moment_repo
            .find_by_ids_with_distance(&[moment_id], DEFAULT_LATITUDE, DEFAULT_LONGITUDE, viewer_id)
            .await?;
        let moment_data = moments
            .first()
            .ok_or_else(|| AppError::MomentNotFound("瞬间不存在".to_string()))?;

        let tags = self.tag_repo.get_moment_tags(moment_id).await?;

        let detail = MomentDetailCache {
            moment_id: moment_data.moment_id.to_string(),
            user_id: moment_data.user_id,
            nickname: moment_data.nickname.clone(),
            avatar_url: moment_data.avatar_url.clone(),
            user_type: moment_data.user_type,
            description: moment_data.description.clone(),
            images: moment_data.images.clone(),
            tags,
            mentions: moment_data.mentions.clone(),
            created_at: moment_data.created_at.to_rfc3339(),
            likes_count: moment_data.likes_count as i32,
            comments_count: moment_data.comments_count as i32,
            latitude: moment_data.latitude.unwrap_or(0.0),
            longitude: moment_data.longitude.unwrap_or(0.0),
            location_name: moment_data.location_name.clone().unwrap_or_default(),
            privacy_level: moment_data.privacy_level.clone(),
        };

        if detail.privacy_level == "public" {
            let _ = self
                .cache_manager
                .set_with_ttl(cache_keys::MOMENT_DETAIL, &cache_key, &detail, 30)
                .await;
        }

        Ok(detail)
    }

    /// 获取指定用户生活瞬间历史（公开 API，受 moments_visible 隐私设置控制）
    pub async fn get_user_moments_list(
        &self,
        req: &UserMomentsRequest,
    ) -> Result<PaginatedData<UserMomentListItem>> {
        let target_user_id = req.user_id;

        // 检查 moments_visible 隐私设置
        let settings = self
            .user_settings_repo
            .find_by_user_id(target_user_id)
            .await
            .ok()
            .flatten();
        let moments_visible = settings
            .as_ref()
            .map(|s| s.moments_visible)
            .unwrap_or(PrivacyLevel::Public);

        // 公开 API 只能访问 moments_visible = 'public' 的用户
        if moments_visible != PrivacyLevel::Public {
            return Err(AppError::Forbidden("权限不足".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor_string = req.cursor.clone();

        let (moments, next_cursor) = self
            .moment_repo
            .list_public_by_user(target_user_id, limit, cursor_string)
            .await?;

        self.build_user_moment_list_from_rows(moments, next_cursor)
            .await
    }

    /// 获取当前用户生活瞬间历史（/me/ API）
    pub async fn get_my_moments_list(
        &self,
        user_id: i64,
        req: &MyMomentsRequest,
    ) -> Result<PaginatedData<UserMomentListItem>> {
        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor_string = req.cursor.clone();

        let (moments, next_cursor) = self
            .moment_repo
            .list_by_user(user_id, limit, cursor_string)
            .await?;

        self.build_user_moment_list_from_rows(moments, next_cursor)
            .await
    }

    /// 点赞瞬间
    pub async fn like_moment(&self, user_id: i64, req: &LikeMomentRequest) -> Result<()> {
        use crate::types::enums::LikeAction;
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;
        self.handle_like_action(user_id, moment_id, LikeAction::Like)
            .await
    }

    /// 取消点赞瞬间
    pub async fn unlike_moment(&self, user_id: i64, req: &UnlikeMomentRequest) -> Result<()> {
        use crate::types::enums::LikeAction;
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;
        self.handle_like_action(user_id, moment_id, LikeAction::Unlike)
            .await
    }

    /// 统一处理点赞/取消点赞
    async fn handle_like_action(
        &self,
        user_id: i64,
        moment_id: Uuid,
        action: crate::types::enums::LikeAction,
    ) -> Result<()> {
        use crate::types::enums::LikeAction;

        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }

        match action {
            LikeAction::Like => {
                let inserted = self.moment_repo.like_moment(moment_id, user_id).await?;
                if !inserted {
                    return Err(AppError::Conflict("已点赞过该瞬间".to_string()));
                }
                // 发布事件
                if let Some(owner_id) = self.get_moment_owner_cached(moment_id).await? {
                    if owner_id != user_id {
                        let summary_snapshot =
                            self.moment_repo.get_summary_snapshot(moment_id).await?;
                        if let Some(summary) = summary_snapshot {
                            self.event_bus.publish(AppEvent::MomentLiked {
                                moment_id,
                                moment_summary: summary,
                                user_id: owner_id,
                                liker_id: user_id,
                            });
                        }
                    }
                }
            }
            LikeAction::Unlike => {
                let deleted = self.moment_repo.unlike_moment(moment_id, user_id).await?;
                if !deleted {
                    return Err(AppError::Conflict("尚未点赞该瞬间".to_string()));
                }
            }
        }
        Ok(())
    }

    /// 检查是否已点赞
    pub async fn check_liked(&self, user_id: i64, req: &CheckLikedRequest) -> Result<bool> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        self.moment_repo.is_liked_by(moment_id, user_id).await
    }

    /// 收藏瞬间
    pub async fn favorite_moment(&self, user_id: i64, req: &FavoriteMomentRequest) -> Result<()> {
        use crate::types::enums::FavoriteAction;
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;
        self.handle_favorite_action(user_id, moment_id, FavoriteAction::Add)
            .await
    }

    /// 取消收藏瞬间
    pub async fn unfavorite_moment(
        &self,
        user_id: i64,
        req: &UnfavoriteMomentRequest,
    ) -> Result<()> {
        use crate::types::enums::FavoriteAction;
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;
        self.handle_favorite_action(user_id, moment_id, FavoriteAction::Remove)
            .await
    }

    /// 统一处理收藏/取消收藏
    async fn handle_favorite_action(
        &self,
        user_id: i64,
        moment_id: Uuid,
        action: crate::types::enums::FavoriteAction,
    ) -> Result<()> {
        use crate::types::enums::FavoriteAction;

        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }

        match action {
            FavoriteAction::Add => {
                let inserted = self.moment_repo.add_favorite(moment_id, user_id).await?;
                if !inserted {
                    return Err(AppError::Conflict("已收藏过该瞬间".to_string()));
                }
            }
            FavoriteAction::Remove => {
                let deleted = self.moment_repo.remove_favorite(moment_id, user_id).await?;
                if !deleted {
                    return Err(AppError::Conflict("尚未收藏该瞬间".to_string()));
                }
            }
        }
        Ok(())
    }

    /// 检查是否已收藏
    pub async fn check_favorited(&self, user_id: i64, req: &CheckFavoritedRequest) -> Result<bool> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        self.moment_repo.is_favorited_by(moment_id, user_id).await
    }

    /// 获取瞬间互动状态（点赞+收藏）
    pub async fn get_interaction_state(
        &self,
        user_id: i64,
        req: &GetMomentInteractionStateRequest,
    ) -> Result<MomentInteractionStateData> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        // 验证瞬间是否存在
        if !self.moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }

        // 并行查询点赞和收藏状态
        let (is_liked, is_favorited) = tokio::try_join!(
            self.moment_repo.is_liked_by(moment_id, user_id),
            self.moment_repo.is_favorited_by(moment_id, user_id)
        )?;

        Ok(MomentInteractionStateData {
            is_liked,
            is_favorited,
        })
    }

    /// 获取用户收藏列表
    pub async fn get_favorites_list(
        &self,
        user_id: i64,
        req: &MyFavoritesRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor = req.cursor.clone();

        let user = self
            .user_repo
            .find_by_id(user_id)
            .await?
            .ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;
        let latitude = user.current_latitude.unwrap_or(DEFAULT_LATITUDE);
        let longitude = user.current_longitude.unwrap_or(DEFAULT_LONGITUDE);

        // list_favorites returns (MomentSearchRow, favorited_at)
        let (results, next_cursor) = self
            .moment_repo
            .list_favorites(user_id, latitude, longitude, limit, cursor)
            .await?;

        let search_rows: Vec<MomentSearchRow> = results.into_iter().map(|(m, _)| m).collect();
        let moment_ids: Vec<Uuid> = search_rows.iter().map(|m| m.moment_id).collect();
        let liked_ids = self
            .moment_repo
            .list_liked_ids_by_user(user_id, &moment_ids)
            .await?;
        let liked_set: std::collections::HashSet<Uuid> = liked_ids.into_iter().collect();

        let tags_map = self.tag_repo.get_moment_tags_batch(&moment_ids).await?;

        let mut items = Vec::with_capacity(search_rows.len());
        for m in search_rows {
            let tags = tags_map.get(&m.moment_id).cloned().unwrap_or_default();
            let images = self.sign_moment_images(&m.images).await;
            let my_state = Some(MyMomentState {
                is_liked: liked_set.contains(&m.moment_id),
                is_favorited: true,
            });

            items.push(MomentSearchItem {
                moment_id: m.moment_id.to_string(),
                user: UserInfo {
                    user_id: m.user_id,
                    nickname: m.nickname,
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(m.avatar_url.as_deref())
                        .await,
                    user_type: m.user_type,
                },
                description: m.description,
                images,
                tags,
                mentions: m.mentions.clone(),
                created_at: m.created_at.to_rfc3339(),
                likes_count: m.likes_count as i32,
                comments_count: m.comments_count as i32,
                location: RelativeLocation {
                    latitude: m.latitude.unwrap_or(0.0),
                    longitude: m.longitude.unwrap_or(0.0),
                    location_name: m.location_name.unwrap_or_default(),
                    distance: None,
                    bearing_degrees: None,
                },
                my_state,
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

    /// 获取瞬间点赞列表
    pub async fn get_likes_list(
        &self,
        req: &MomentLikesRequest,
    ) -> Result<PaginatedData<MomentLikeItem>> {
        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor = req.cursor.clone();

        let (likes, next_cursor) = self
            .moment_repo
            .list_likes(moment_id, limit as i64, cursor)
            .await?;

        let user_ids: Vec<i64> = likes.iter().map(|l| l.user_id).collect();
        let users = self.user_repo.find_profiles(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        let mut items = Vec::with_capacity(likes.len());
        for like in likes {
            if let Some(user) = user_map.get(&like.user_id) {
                items.push(MomentLikeItem {
                    user: UserInfo {
                        user_id: user.user_id,
                        nickname: user.nickname.clone(),
                        avatar_url: self
                            .media_service
                            .sign_url_if_path(user.avatar_url.as_deref())
                            .await,
                        user_type: user.user_type,
                    },
                    liked_at: like.created_at.to_rfc3339(),
                });
            }
        }

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    /// Helper: 构建搜索结果 (from MomentSearchRow)
    async fn build_search_results_from_rows(
        &self,
        moments: Vec<MomentSearchRow>,
        next_cursor: Option<String>,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        // 批量获取标签
        let moment_ids: Vec<_> = moments.iter().map(|m| m.moment_id).collect();
        let tags_map = self.tag_repo.get_moment_tags_batch(&moment_ids).await?;

        let mut items = Vec::with_capacity(moments.len());

        for m in moments {
            let tags = tags_map.get(&m.moment_id).cloned().unwrap_or_default();
            let _ = m.privacy_level.as_str();

            let images = self.sign_moment_images(&m.images).await;

            items.push(MomentSearchItem {
                moment_id: m.moment_id.to_string(),
                user: UserInfo {
                    user_id: m.user_id,
                    nickname: m.nickname,
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(m.avatar_url.as_deref())
                        .await,
                    user_type: m.user_type,
                },
                description: m.description,
                images,
                tags,
                mentions: m.mentions.clone(),
                created_at: m.created_at.to_rfc3339(),
                likes_count: m.likes_count as i32,
                comments_count: m.comments_count as i32,
                location: RelativeLocation {
                    latitude: m.latitude.unwrap_or(0.0),
                    longitude: m.longitude.unwrap_or(0.0),
                    location_name: m.location_name.unwrap_or_default(),
                    distance: m.distance,
                    bearing_degrees: m.bearing_degrees,
                },
                my_state: None,
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

    async fn build_user_moment_list_from_rows(
        &self,
        moments: Vec<MomentSearchRow>,
        next_cursor: Option<String>,
    ) -> Result<PaginatedData<UserMomentListItem>> {
        let moment_ids: Vec<_> = moments.iter().map(|m| m.moment_id).collect();
        let tags_map = self.tag_repo.get_moment_tags_batch(&moment_ids).await?;

        let mut items = Vec::with_capacity(moments.len());
        for m in moments {
            let tags = tags_map.get(&m.moment_id).cloned().unwrap_or_default();
            let images = self.sign_moment_images(&m.images).await;

            items.push(UserMomentListItem {
                moment_id: m.moment_id.to_string(),
                description: m.description,
                images,
                tags,
                mentions: m.mentions.clone(),
                created_at: m.created_at.to_rfc3339(),
                likes_count: m.likes_count as i32,
                comments_count: m.comments_count as i32,
                location: RelativeLocation {
                    latitude: m.latitude.unwrap_or(0.0),
                    longitude: m.longitude.unwrap_or(0.0),
                    location_name: m.location_name.unwrap_or_default(),
                    distance: None,
                    bearing_degrees: None,
                },
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

    async fn get_moment_owner_cached(&self, moment_id: Uuid) -> Result<Option<i64>> {
        let key = moment_id.to_string();
        let cached = self
            .cache_manager
            .get_or_set_with_lock_with_ttl(cache_keys::MOMENT_OWNER, &key, 3600, 300, || async {
                let owner = self
                    .moment_repo
                    .get_owner(moment_id)
                    .await
                    .map_err(|e| Box::new(e) as Box<dyn std::error::Error + Send + Sync>)?;
                Ok(owner)
            })
            .await
            .map_err(|e| AppError::InternalError(e.to_string()))?;

        Ok(cached)
    }
}
