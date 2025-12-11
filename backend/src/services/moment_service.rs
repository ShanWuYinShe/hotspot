use redis::aio::ConnectionManager as RedisConnectionManager;
use sqlx::PgPool;
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreateMomentRequest, EditMomentRequest, DeleteMomentRequest,
            SearchMomentsByKeywordRequest, SearchMomentsByTagRequest, SearchMomentsByLocationRequest,
            MomentOpenTheDoorRequest, GetMomentCommentsRequest, GetUserMomentsListRequest, GetMyMomentsListRequest,
            LikeMomentRequest, UnlikeMomentRequest, CheckMomentLikedRequest,
            FavoriteMomentRequest, UnfavoriteMomentRequest, CheckMomentFavoritedRequest,
            GetFavoritesListRequest, GetMomentLikesListRequest, GetCommentsRequest,
        },
        response::{
            MomentSearchItem, CommentListItem, MomentLikeItem, GeoData,
            PaginatedData, PaginationMeta, UserInfo,
        },
    },
    error::{AppError, Result},
    repositories::moment_repo::{MomentRepository, MomentRow},
    repositories::tag_repo::TagRepository,
    repositories::user_repo::UserRepository,
    services::comment_service::CommentService,
    utils::validation::Validator,
    EventBus,
};

pub struct MomentService<'a> {
    db: &'a PgPool,
    event_bus: Arc<EventBus>,
    _redis: &'a RedisConnectionManager,
}

impl<'a> MomentService<'a> {
    pub fn new(db: &'a PgPool, redis: &'a RedisConnectionManager, event_bus: Arc<EventBus>) -> Self {
        Self { db, event_bus, _redis: redis }
    }

    /// 创建瞬间
    pub async fn create_moment(&self, user_id: i64, req: &CreateMomentRequest) -> Result<String> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let tag_repo = TagRepository::new(self.db.clone());

        // 验证坐标
        if req.location.latitude < -90.0 || req.location.latitude > 90.0 {
            return Err(AppError::InvalidInput("纬度必须在-90到90之间".to_string()));
        }
        if req.location.longitude < -180.0 || req.location.longitude > 180.0 {
            return Err(AppError::InvalidInput("经度必须在-180到180之间".to_string()));
        }

        // 验证内容长度
        if req.description.is_empty() {
            return Err(AppError::BadRequest("内容不能为空".to_string()));
        }

        if req.description.len() > 5000 {
            return Err(AppError::BadRequest("内容过长".to_string()));
        }

        // 验证图片数量
        if let Some(images) = &req.images {
            if images.len() > 9 {
                return Err(AppError::BadRequest("最多只能上传9张图片".to_string()));
            }
        }

        // 创建瞬间
        let moment = moment_repo.create(
            user_id,
            req.description.clone(),
            req.images.clone(),
            req.location.latitude,
            req.location.longitude,
            req.location.name.clone(),
        ).await?;

        // 设置标签（通过关联表）
        if let Some(tags) = &req.tags {
            if !tags.is_empty() {
                tag_repo.set_moment_tags(moment.moment_id, tags).await?;
            }
        }

        Ok(moment.moment_id.to_string())
    }

    /// 编辑瞬间
    pub async fn update_moment(
        &self,
        user_id: i64,
        req: &EditMomentRequest,
    ) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let tag_repo = TagRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        let moment = moment_repo
            .find_by_id(moment_id)
            .await?
            .ok_or_else(|| AppError::NotFound("瞬间不存在".to_string()))?;

        // 只能编辑自己的瞬间
        if moment.user_id != user_id {
            return Err(AppError::Forbidden("无权编辑该瞬间".to_string()));
        }

        // 更新瞬间基本信息
        moment_repo.update(
            moment_id,
            req.description.clone(),
            req.images.clone(),
        ).await?;

        // 更新标签（通过关联表）
        if let Some(tags) = &req.tags {
            tag_repo.set_moment_tags(moment_id, tags).await?;
        }

        Ok(())
    }

    /// 删除瞬间
    pub async fn delete_moment(&self, user_id: i64, req: &DeleteMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        let moment = moment_repo
            .find_by_id(moment_id)
            .await?
            .ok_or_else(|| AppError::NotFound("瞬间不存在".to_string()))?;

        // 只能删除自己的瞬间
        if moment.user_id != user_id {
            return Err(AppError::Forbidden("无权删除该瞬间".to_string()));
        }

        moment_repo.soft_delete(moment_id).await?;

        Ok(())
    }

    /// 按关键字搜索生活瞬间（全局）
    /// TODO: 实现真正的关键字搜索（描述文本匹配）
    pub async fn search_by_keyword(
        &self,
        req: &SearchMomentsByKeywordRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;
        
        // 按标签搜索
        let moments = moment_repo.search_by_tag(&req.kw, (limit + 1) as i32, cursor).await?;
        
        self.build_search_results(moments, limit, req.latitude, req.longitude).await
    }

    /// 按标签搜索生活瞬间（全局）
    pub async fn search_by_tag(
        &self,
        req: &SearchMomentsByTagRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;
        
        let moments = moment_repo.search_by_tag(&req.tag, (limit + 1) as i32, cursor).await?;
        
        self.build_search_results(moments, limit, req.latitude, req.longitude).await
    }

    /// 按位置搜索生活瞬间（范围过滤，距离由 PostGIS 计算）
    pub async fn search_by_location(
        &self,
        req: &SearchMomentsByLocationRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        let moment_repo = MomentRepository::new(self.db.clone());

        Validator::validate_coordinates(req.latitude, req.longitude)?;
        Validator::validate_radius_required(req.radius, 50000.0)?;
        
        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;
        
        let results = moment_repo
            .search_by_location(req.latitude, req.longitude, req.radius, (limit + 1) as i32, cursor)
            .await?;

        let has_more = results.len() > limit as usize;
        let results_to_return: Vec<_> = results.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            results_to_return.last().map(|m| m.moment_id.to_string())
        } else {
            None
        };

        // 批量获取标签
        let moment_ids: Vec<_> = results_to_return.iter().map(|m| m.moment_id).collect();
        let tag_repo = TagRepository::new(self.db.clone());
        let tags_map = tag_repo.get_moment_tags_batch(&moment_ids).await?;

        let items = results_to_return.into_iter().map(|m| {
            let tags = tags_map.get(&m.moment_id).cloned().unwrap_or_default();
            MomentSearchItem {
                id: m.moment_id.to_string(),
                user: UserInfo {
                    user_id: m.user_id,
                    nickname: m.nickname,
                    avatar_url: m.avatar_url,
                    user_type: m.user_type,
                },
                description: m.description,
                images: Some(m.images),
                tags,
                created_at: m.created_at.to_rfc3339(),
                likes_count: m.likes_count as i32,
                comments_count: m.comments_count as i32,
                geo: GeoData {
                    latitude: m.latitude,
                    longitude: m.longitude,
                    location_name: m.location_name,
                    distance: m.distance as i32,
                    degrees: m.degrees,
                },
            }
        }).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta { has_more, next_cursor },
        })
    }

    /// 获取瞬间详情（通过 ID 获取指定瞬间的详情）
    pub async fn open_the_door(&self, req: &MomentOpenTheDoorRequest) -> Result<MomentSearchItem> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let user_repo = UserRepository::new(self.db.clone());
        let tag_repo = TagRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        Validator::validate_coordinates(req.latitude, req.longitude)?;

        let moment = moment_repo.find_by_id(moment_id).await?
            .ok_or_else(|| AppError::NotFound("瞬间不存在".to_string()))?;

        let user = user_repo.find_by_id(moment.user_id).await?
            .ok_or_else(|| AppError::NotFound("用户不存在".to_string()))?;

        // 获取标签
        let tags = tag_repo.get_moment_tags(moment_id).await?;

        // TODO: distance 和 degrees 应由 PostGIS 计算
        let distance = 0;
        let degrees = 0.0;

        Ok(MomentSearchItem {
            id: moment.moment_id.to_string(),
            user: UserInfo {
                user_id: user.user_id,
                nickname: user.nickname.clone(),
                avatar_url: user.avatar_url.clone(),
                user_type: user.user_type,
            },
            description: moment.description,
            images: Some(moment.images),
            tags,
            created_at: moment.created_at.to_rfc3339(),
            likes_count: 0, // TODO: 获取点赞计数
            comments_count: 0, // TODO: 获取评论计数
            geo: GeoData {
                latitude: moment.latitude,
                longitude: moment.longitude,
                location_name: moment.location_name,
                distance,
                degrees,
            },
        })
    }

    /// 获取瞬间评论列表
    pub async fn get_comments_list(
        &self,
        req: &GetMomentCommentsRequest,
    ) -> Result<PaginatedData<CommentListItem>> {
        // 将请求转换为 CommentService 的格式
        let comment_req = GetCommentsRequest {
            moment_id: req.moment_id.clone(),
            limit: req.limit,
            cursor: req.cursor.clone(),
        };
        
        // 使用 CommentService 来获取评论
        let comment_service = CommentService::new(self.db, self._redis, self.event_bus.clone());
        comment_service.list_comments(&comment_req).await
    }

    /// 获取指定用户生活瞬间历史（公开 API）
    pub async fn get_user_moments_list(
        &self,
        req: &GetUserMomentsListRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        // TODO: 检查 moments_visible 隐私设置
        self.get_moments_by_user(req.id, req.latitude, req.longitude, req.limit, req.cursor.as_deref()).await
    }

    /// 获取当前用户生活瞬间历史（/me/ API）
    pub async fn get_my_moments_list(
        &self,
        user_id: i64,
        req: &GetMyMomentsListRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        self.get_moments_by_user(user_id, req.latitude, req.longitude, req.limit, req.cursor.as_deref()).await
    }

    async fn get_moments_by_user(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        limit: Option<i32>,
        cursor: Option<&str>,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let limit = Validator::validate_limit(limit, 100)? as i64;
        let cursor = Validator::parse_cursor(cursor)?;
        
        let moments = moment_repo.list_by_user(user_id, (limit + 1) as i32, cursor).await?;
        
        self.build_search_results(moments, limit, latitude, longitude).await
    }

    /// 点赞瞬间
    pub async fn like_moment(&self, user_id: i64, req: &LikeMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        if moment_repo.find_by_id(moment_id).await?.is_none() {
            return Err(AppError::NotFound("瞬间不存在".to_string()));
        }

        moment_repo.like_moment(moment_id, user_id).await?;
        Ok(())
    }

    /// 取消点赞瞬间
    pub async fn unlike_moment(&self, user_id: i64, req: &UnlikeMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        if moment_repo.find_by_id(moment_id).await?.is_none() {
            return Err(AppError::NotFound("瞬间不存在".to_string()));
        }

        moment_repo.unlike_moment(moment_id, user_id).await?;
        Ok(())
    }

    /// 检查是否已点赞
    pub async fn check_liked(&self, user_id: i64, req: &CheckMomentLikedRequest) -> Result<bool> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        moment_repo.is_liked_by(moment_id, user_id).await
    }

    /// 收藏瞬间
    pub async fn favorite_moment(&self, user_id: i64, req: &FavoriteMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        if moment_repo.find_by_id(moment_id).await?.is_none() {
            return Err(AppError::NotFound("瞬间不存在".to_string()));
        }

        moment_repo.add_favorite(moment_id, user_id).await?;
        Ok(())
    }

    /// 取消收藏瞬间
    pub async fn unfavorite_moment(&self, user_id: i64, req: &UnfavoriteMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        if moment_repo.find_by_id(moment_id).await?.is_none() {
            return Err(AppError::NotFound("瞬间不存在".to_string()));
        }

        moment_repo.remove_favorite(moment_id, user_id).await?;
        Ok(())
    }

    /// 检查是否已收藏
    pub async fn check_favorited(&self, user_id: i64, req: &CheckMomentFavoritedRequest) -> Result<bool> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        moment_repo.is_favorited_by(moment_id, user_id).await
    }

    /// 获取收藏列表
    pub async fn get_favorites_list(
        &self,
        user_id: i64,
        req: &GetFavoritesListRequest,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;
        let moments = moment_repo.list_favorites(user_id, (limit + 1) as i32, cursor).await?;

        self.build_search_results(moments, limit, req.latitude, req.longitude).await
    }

    /// 获取瞬间点赞列表
    pub async fn get_likes_list(
        &self,
        req: &GetMomentLikesListRequest,
    ) -> Result<PaginatedData<MomentLikeItem>> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let user_repo = UserRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;
        let likes = moment_repo.list_likes(moment_id, (limit + 1) as i64, cursor).await?;

        let has_more = likes.len() > limit as usize;
        let likes_to_return: Vec<_> = likes.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            likes_to_return.last().map(|l| l.cursor_id.to_string())
        } else {
            None
        };

        let user_ids: Vec<i64> = likes_to_return.iter().map(|l| l.user_id).collect();
        let users = user_repo.find_by_ids(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        let items = likes_to_return.into_iter().filter_map(|like| {
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
        }).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta { has_more, next_cursor },
        })
    }

    /// Helper: 构建搜索结果
    /// 注意：GeoData 中的 distance 和 degrees 应由 Repository 层通过 PostGIS 计算返回
    async fn build_search_results(
        &self,
        moments: Vec<MomentRow>,
        limit: i64,
        _requester_lat: f64,
        _requester_lon: f64,
    ) -> Result<PaginatedData<MomentSearchItem>> {
        let user_repo = UserRepository::new(self.db.clone());
        let tag_repo = TagRepository::new(self.db.clone());
        
        let has_more = moments.len() > limit as usize;
        let moments_to_return: Vec<MomentRow> = moments.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            moments_to_return.last().map(|m| m.moment_id.to_string())
        } else {
            None
        };

        let user_ids: Vec<i64> = moments_to_return.iter().map(|m| m.user_id).collect();
        let users = user_repo.find_by_ids(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        // 批量获取标签
        let moment_ids: Vec<_> = moments_to_return.iter().map(|m| m.moment_id).collect();
        let tags_map = tag_repo.get_moment_tags_batch(&moment_ids).await?;

        let items = moments_to_return.into_iter().filter_map(|m| {
            let user = user_map.get(&m.user_id)?;
            let tags = tags_map.get(&m.moment_id).cloned().unwrap_or_default();
            // TODO: distance 和 degrees 应由 PostGIS 在 SQL 查询中计算
            let distance = 0; // 需要从查询结果获取
            let degrees = 0.0; // 需要从查询结果获取 (ST_Azimuth)
            Some(MomentSearchItem {
                id: m.moment_id.to_string(),
                user: UserInfo {
                    user_id: user.user_id,
                    nickname: user.nickname.clone(),
                    avatar_url: user.avatar_url.clone(),
                    user_type: user.user_type,
                },
                description: m.description,
                images: Some(m.images),
                tags,
                created_at: m.created_at.to_rfc3339(),
                likes_count: 0, // TODO: 获取点赞计数
                comments_count: 0, // TODO: 获取评论计数
                geo: GeoData {
                    latitude: m.latitude,
                    longitude: m.longitude,
                    location_name: m.location_name,
                    distance,
                    degrees,
                },
            })
        }).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta { has_more, next_cursor },
        })
    }

}
