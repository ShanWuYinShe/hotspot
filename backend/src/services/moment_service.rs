use redis::aio::ConnectionManager as RedisConnectionManager;
use sqlx::PgPool;
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CheckFavoritedRequest, CheckLikedRequest, CreateMomentRequest, DeleteMomentRequest,
            FavoriteMomentRequest, LikeMomentRequest, MomentLikesRequest,
            MomentOpenTheDoorRequest, MyFavoritesRequest, MyMomentsRequest,
            SearchMomentsByKeywordRequest, SearchMomentsByLocationRequest,
            SearchMomentsByTagRequest, UnfavoriteMomentRequest, UnlikeMomentRequest,
            UpdateMomentRequest, UserMomentsRequest,
        },
        response::{
            RelativeLocation, MomentLikeItem, MomentItem, PaginatedData,
            PaginationMeta, UserInfo,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::file_repo::FileRepository,
    repositories::moment_repo::{MomentRepository, MomentSearchRow},
    repositories::tag_repo::TagRepository,
    repositories::user_repo::UserRepository,
    repositories::user_settings_repo::UserSettingsRepository,
    types::enums::PrivacyLevel,
    utils::validation::Validator,
    EventBus,
};

#[derive(Clone)]
pub struct MomentService {
    db: PgPool,
    _redis: RedisConnectionManager,
    event_bus: Arc<EventBus>,
}

impl MomentService {
    pub fn new(db: &PgPool, redis: &RedisConnectionManager, event_bus: Arc<EventBus>) -> Self {
        Self {
            db: db.clone(),
            _redis: redis.clone(),
            event_bus,
        }
    }

    /// 创建瞬间
    pub async fn create_moment(&self, user_id: i64, req: &CreateMomentRequest) -> Result<MomentItem> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let tag_repo = TagRepository::new(self.db.clone());
        let file_repo = FileRepository::new(self.db.clone());
        let user_repo = UserRepository::new(self.db.clone());

        // 验证坐标
        if req.location.latitude < -90.0 || req.location.latitude > 90.0 {
            return Err(AppError::InvalidInput("纬度必须在-90到90之间".to_string()));
        }
        if req.location.longitude < -180.0 || req.location.longitude > 180.0 {
            return Err(AppError::InvalidInput(
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

        // 解析并验证图片 file_id -> url
        // 重要：不要在 moment 创建成功前标记 is_used，避免“先占用、后失败”导致脏数据。
        let (images, image_uuids) = if let Some(file_ids) = &req.image_file_ids {
            if file_ids.len() > 9 {
                return Err(AppError::BadRequest("最多只能上传9张图片".to_string()));
            }
            let mut urls = Vec::with_capacity(file_ids.len());
            let mut uuids = Vec::with_capacity(file_ids.len());
            for file_id in file_ids {
                let uuid = Uuid::parse_str(file_id)
                    .map_err(|_| AppError::BadRequest("无效的图片 file_id".to_string()))?;
                let url = file_repo
                    .validate_file(uuid, user_id, "moment_image")
                    .await?;
                uuids.push(uuid);
                urls.push(url);
            }
            (Some(urls), uuids)
        } else {
            (None, Vec::new())
        };

        // 创建瞬间
        let moment = moment_repo
            .create(
                user_id,
                req.description.clone(),
                images.clone(),
                req.location.latitude,
                req.location.longitude,
                req.location.location_name.clone(),
            )
            .await?;

        // moment 已创建成功后再标记文件已使用（避免创建失败导致 is_used 脏数据）
        for file_id in image_uuids {
            if let Err(e) = file_repo.mark_as_used(file_id).await {
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
                tag_repo.set_moment_tags(moment.moment_id, tags).await?;
            }
            tags.clone()
        } else {
            Vec::new()
        };

        // 获取创建者信息
        let users = user_repo.find_profiles(&[user_id]).await?;
        let user_info = users.first().ok_or_else(|| AppError::UserNotFound("用户不存在".to_string()))?;

        // 构造响应
        use crate::dto::response::MomentImage;
        let moment_images = images.map(|urls| {
            urls.into_iter()
                .map(|url| MomentImage {
                    url,
                    width: None,
                    height: None,
                    content_type: None,
                })
                .collect()
        });

        Ok(MomentItem {
            moment_id: moment.moment_id.to_string(),
            user: UserInfo {
                user_id: user_info.user_id,
                nickname: user_info.nickname.clone(),
                avatar_url: user_info.avatar_url.clone(),
                user_type: user_info.user_type,
            },
            description: moment.description,
            images: moment_images,
            tags,
            created_at: moment.created_at.to_rfc3339(),
            likes_count: 0,
            comments_count: 0,
            is_liked: Some(false),
            is_favorited: Some(false),
            location: Some(RelativeLocation {
                latitude: req.location.latitude,
                longitude: req.location.longitude,
                location_name: req.location.location_name.clone(),
                distance: 0,
                bearing_degrees: 0.0,
            }),
        })
    }

    /// 编辑瞬间
    pub async fn update_moment(&self, user_id: i64, req: &UpdateMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let tag_repo = TagRepository::new(self.db.clone());
        let file_repo = FileRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        // 验证所有权
        let owner_id = moment_repo
            .get_owner(moment_id)
            .await?
            .ok_or_else(|| AppError::MomentNotFound("瞬间不存在".to_string()))?;

        if owner_id != user_id {
            return Err(AppError::Forbidden("无权编辑该瞬间".to_string()));
        }

        // 更新瞬间基本信息
        // 重要：不要在 moment 更新成功前标记 is_used，避免“先占用、后失败”导致脏数据。
        let (images, image_uuids) = if let Some(file_ids) = &req.image_file_ids {
            if file_ids.len() > 9 {
                return Err(AppError::BadRequest("最多只能上传9张图片".to_string()));
            }
            let mut urls = Vec::with_capacity(file_ids.len());
            let mut uuids = Vec::with_capacity(file_ids.len());
            for file_id in file_ids {
                let uuid = Uuid::parse_str(file_id)
                    .map_err(|_| AppError::BadRequest("无效的图片 file_id".to_string()))?;
                let url = file_repo
                    .validate_file(uuid, user_id, "moment_image")
                    .await?;
                uuids.push(uuid);
                urls.push(url);
            }
            (Some(urls), uuids)
        } else {
            (None, Vec::new())
        };

        moment_repo
            .update(moment_id, req.description.clone(), images)
            .await?;

        // moment 更新成功后再标记文件已使用（避免更新失败导致 is_used 脏数据）
        for file_id in image_uuids {
            if let Err(e) = file_repo.mark_as_used(file_id).await {
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
            moment_repo
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
            tag_repo.set_moment_tags(moment_id, tags).await?;
        }

        Ok(())
    }

    /// 删除瞬间
    pub async fn delete_moment(&self, user_id: i64, req: &DeleteMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        // 验证所有权
        let owner_id = moment_repo
            .get_owner(moment_id)
            .await?
            .ok_or_else(|| AppError::MomentNotFound("瞬间不存在".to_string()))?;

        if owner_id != user_id {
            return Err(AppError::Forbidden("无权删除该瞬间".to_string()));
        }

        moment_repo.soft_delete(moment_id).await?;

        Ok(())
    }

    /// 按关键字搜索生活瞬间（全局）
    pub async fn search_by_keyword(
        &self,
        req: &SearchMomentsByKeywordRequest,
    ) -> Result<PaginatedData<MomentItem>> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor = req.cursor.clone();

        // 按关键字搜索
        let (moments, next_cursor) = moment_repo
            .search_by_keyword(&req.kw, req.reference_point.latitude, req.reference_point.longitude, limit, cursor)
            .await?;

        self.build_search_results_from_rows(moments, next_cursor)
            .await
    }

    /// 按标签搜索生活瞬间（全局）
    pub async fn search_by_tag(
        &self,
        req: &SearchMomentsByTagRequest,
    ) -> Result<PaginatedData<MomentItem>> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor = req.cursor.clone();

        // search_by_tag 返回包含 distance/bearing_degrees 的 MomentSearchRow
        let (moments, next_cursor) = moment_repo
            .search_by_tag(&req.tag, req.reference_point.latitude, req.reference_point.longitude, limit, cursor)
            .await?;

        self.build_search_results_from_rows(moments, next_cursor)
            .await
    }

    /// 按位置搜索生活瞬间（范围过滤，距离由 PostGIS 计算）
    pub async fn search_by_location(
        &self,
        req: &SearchMomentsByLocationRequest,
    ) -> Result<PaginatedData<MomentItem>> {
        let moment_repo = MomentRepository::new(self.db.clone());

        Validator::validate_coordinates(req.reference_point.latitude, req.reference_point.longitude)?;
        Validator::validate_radius_required(req.radius, 50000.0)?;

        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor = req.cursor.clone();

        let (moments, next_cursor) = moment_repo
            .search_by_location(
                req.kw.as_deref(),
                req.reference_point.latitude,
                req.reference_point.longitude,
                req.radius,
                limit,
                cursor,
            )
            .await?;

        self.build_search_results_from_rows(moments, next_cursor)
            .await
    }

    /// 获取瞬间详情（通过 ID 获取指定瞬间的详情）
    pub async fn open_the_door(&self, req: &MomentOpenTheDoorRequest) -> Result<MomentItem> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let tag_repo = TagRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        Validator::validate_coordinates(req.reference_point.latitude, req.reference_point.longitude)?;

        // 使用 find_by_ids_with_distance 获取带距离的详情（包含用户信息和统计数据）
        // 这样可以避免多次查询，且复用 PostGIS 距离计算
        let moments = moment_repo
            .find_by_ids_with_distance(&[moment_id], req.reference_point.latitude, req.reference_point.longitude)
            .await?;

        let moment_data = moments
            .first()
            .ok_or_else(|| AppError::MomentNotFound("瞬间不存在".to_string()))?;

        // 获取标签（find_by_ids_with_distance 目前不包含标签，因为是一对多关系）
        let tags = tag_repo.get_moment_tags(moment_id).await?;

        use crate::dto::response::MomentImage;
        let images = if moment_data.images.is_empty() {
            None
        } else {
            Some(moment_data.images.iter().map(|url| MomentImage {
                url: url.clone(),
                width: None,
                height: None,
                content_type: None,
            }).collect())
        };

        Ok(MomentItem {
            moment_id: moment_data.moment_id.to_string(),
            user: UserInfo {
                user_id: moment_data.user_id,
                nickname: moment_data.nickname.clone(),
                avatar_url: moment_data.avatar_url.clone(),
                user_type: moment_data.user_type,
            },
            description: moment_data.description.clone(),
            images,
            tags,
            created_at: moment_data.created_at.to_rfc3339(),
            likes_count: moment_data.likes_count as i32,
            comments_count: moment_data.comments_count as i32,
            is_liked: None,
            is_favorited: None,
            location: Some(RelativeLocation {
                latitude: moment_data.latitude,
                longitude: moment_data.longitude,
                location_name: moment_data.location_name.clone(),
                distance: moment_data.distance as i32,
                bearing_degrees: moment_data.bearing_degrees,
            }),
        })
    }


    /// 获取指定用户生活瞬间历史（公开 API，受 moments_visible 隐私设置控制）
    pub async fn get_user_moments_list(
        &self,
        req: &UserMomentsRequest,
    ) -> Result<PaginatedData<MomentItem>> {
        let target_user_id = req.user_id;

        // 检查 moments_visible 隐私设置
        let user_settings_repo = UserSettingsRepository::new(self.db.clone());
        let settings = user_settings_repo
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

        self.get_moments_by_user(
            target_user_id,
            req.reference_point.latitude,
            req.reference_point.longitude,
            req.limit,
            req.cursor.as_deref(),
        )
        .await
    }

    /// 获取当前用户生活瞬间历史（/me/ API）
    pub async fn get_my_moments_list(
        &self,
        user_id: i64,
        req: &MyMomentsRequest,
    ) -> Result<PaginatedData<MomentItem>> {
        self.get_moments_by_user(
            user_id,
            req.reference_point.latitude,
            req.reference_point.longitude,
            req.limit,
            req.cursor.as_deref(),
        )
        .await
    }

    async fn get_moments_by_user(
        &self,
        user_id: i64,
        _latitude: f64,
        _longitude: f64,
        limit: Option<i32>,
        cursor: Option<&str>,
    ) -> Result<PaginatedData<MomentItem>> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let limit = Validator::validate_limit(limit, 100)? as i32;
        let cursor_string = cursor.map(|s| s.to_string());

        // list_by_user now returns MomentSearchRow
        let (moments, next_cursor) = moment_repo
            .list_by_user(user_id, limit, cursor_string)
            .await?;

        self.build_search_results_from_rows(moments, next_cursor)
            .await
    }

    /// 点赞瞬间
    pub async fn like_moment(&self, user_id: i64, req: &LikeMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        if !moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }

        let inserted = moment_repo.like_moment(moment_id, user_id).await?;
        if !inserted {
            return Err(AppError::Conflict("已点赞过该瞬间".to_string()));
        }

        // 发布“瞬间被点赞”事件（由 EventHandler 负责偏好+订阅过滤与推送）
        if let Some(owner_id) = moment_repo.get_owner(moment_id).await? {
            // 自己点赞自己的瞬间，不需要通知
            if owner_id != user_id {
                let summary_snapshot = moment_repo.get_summary_snapshot(moment_id).await?;
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
        Ok(())
    }

    /// 取消点赞瞬间
    pub async fn unlike_moment(&self, user_id: i64, req: &UnlikeMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        if !moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }

        let deleted = moment_repo.unlike_moment(moment_id, user_id).await?;
        if !deleted {
            return Err(AppError::Conflict("尚未点赞该瞬间".to_string()));
        }
        Ok(())
    }

    /// 检查是否已点赞
    pub async fn check_liked(&self, user_id: i64, req: &CheckLikedRequest) -> Result<bool> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        moment_repo.is_liked_by(moment_id, user_id).await
    }

    /// 收藏瞬间
    pub async fn favorite_moment(&self, user_id: i64, req: &FavoriteMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        if !moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }

        let inserted = moment_repo.add_favorite(moment_id, user_id).await?;
        if !inserted {
            return Err(AppError::Conflict("已收藏过该瞬间".to_string()));
        }
        Ok(())
    }

    /// 取消收藏瞬间
    pub async fn unfavorite_moment(
        &self,
        user_id: i64,
        req: &UnfavoriteMomentRequest,
    ) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        if !moment_repo.exists(moment_id).await? {
            return Err(AppError::MomentNotFound("瞬间不存在".to_string()));
        }

        let deleted = moment_repo.remove_favorite(moment_id, user_id).await?;
        if !deleted {
            return Err(AppError::Conflict("尚未收藏该瞬间".to_string()));
        }
        Ok(())
    }

    /// 检查是否已收藏
    pub async fn check_favorited(&self, user_id: i64, req: &CheckFavoritedRequest) -> Result<bool> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        moment_repo.is_favorited_by(moment_id, user_id).await
    }

    /// 获取用户收藏列表
    pub async fn get_favorites_list(
        &self,
        user_id: i64,
        req: &MyFavoritesRequest,
    ) -> Result<PaginatedData<MomentItem>> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor = req.cursor.clone();

        // list_favorites returns (MomentSearchRow, favorited_at)
        let (results, next_cursor) = moment_repo
            .list_favorites(user_id, req.reference_point.latitude, req.reference_point.longitude, limit, cursor)
            .await?;

        let search_rows: Vec<MomentSearchRow> = results.into_iter().map(|(m, _)| m).collect();

        self.build_search_results_from_rows(search_rows, next_cursor)
            .await
    }

    /// 获取瞬间点赞列表
    pub async fn get_likes_list(
        &self,
        req: &MomentLikesRequest,
    ) -> Result<PaginatedData<MomentLikeItem>> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let user_repo = UserRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)? as i32;
        let cursor = req.cursor.clone();

        let (likes, next_cursor) = moment_repo
            .list_likes(moment_id, limit as i64, cursor)
            .await?;

        let user_ids: Vec<i64> = likes.iter().map(|l| l.user_id).collect();
        let users = user_repo.find_profiles(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        let items = likes
            .into_iter()
            .filter_map(|like| {
                let user = user_map.get(&like.user_id)?;
                Some(MomentLikeItem {
                    user: UserInfo {
                        user_id: user.user_id,
                        nickname: user.nickname.clone(),
                        avatar_url: user.avatar_url.clone(),
                        user_type: user.user_type,
                    },
                    liked_at: like.created_at.to_rfc3339(),
                })
            })
            .collect();

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
    ) -> Result<PaginatedData<MomentItem>> {
        let tag_repo = TagRepository::new(self.db.clone());

        // 批量获取标签
        let moment_ids: Vec<_> = moments.iter().map(|m| m.moment_id).collect();
        let tags_map = tag_repo.get_moment_tags_batch(&moment_ids).await?;

        use crate::dto::response::MomentImage;
        let items = moments
            .into_iter()
            .map(|m| {
                let tags = tags_map.get(&m.moment_id).cloned().unwrap_or_default();
                let images = if m.images.is_empty() {
                    None
                } else {
                    Some(m.images.iter().map(|url| MomentImage {
                        url: url.clone(),
                        width: None,
                        height: None,
                        content_type: None,
                    }).collect())
                };

                MomentItem {
                    moment_id: m.moment_id.to_string(),
                    user: UserInfo {
                        user_id: m.user_id,
                        nickname: m.nickname,
                        avatar_url: m.avatar_url,
                        user_type: m.user_type,
                    },
                    description: m.description,
                    images,
                    tags,
                    created_at: m.created_at.to_rfc3339(),
                    likes_count: m.likes_count as i32,
                    comments_count: m.comments_count as i32,
                    is_liked: None,
                    is_favorited: None,
                    location: RelativeLocation {
                        latitude: m.latitude,
                        longitude: m.longitude,
                        location_name: m.location_name,
                        distance: m.distance as i32,
                        bearing_degrees: m.bearing_degrees,
                    }),
                }
            })
            .collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }
}
