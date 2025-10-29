use redis::Client as RedisClient;
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreateMomentRequest, UpdateMomentRequest, DeleteMomentRequest,
            SearchMomentByLocationRequest, SearchMomentByTagsRequest,
            ListMyMomentsRequest, LikeMomentRequest, UnlikeMomentRequest,
            CollectMomentRequest, UncollectMomentRequest, ListCollectionsRequest,
        },
        response::{MomentResponse, PaginatedResponse},
    },
    error::{AppError, Result},
    models::moment::Moment,
    repositories::moment_repo::MomentRepository,
};

pub struct MomentService<'a> {
    db: &'a PgPool,
    #[allow(dead_code)]
    redis: &'a RedisClient,
}

impl<'a> MomentService<'a> {
    pub fn new(db: &'a PgPool, redis: &'a RedisClient) -> Self {
        Self { db, redis }
    }

    /// 创建瞬间
    pub async fn create_moment(&self, user_id: i64, req: &CreateMomentRequest) -> Result<String> {
        let moment_repo = MomentRepository::new(self.db);

        // 验证坐标
        if let (Some(lat), Some(lon)) = (req.latitude, req.longitude) {
            crate::utils::validation::Validator::validate_coordinates(lat, lon)?;
        }

        // 验证内容长度
        if req.content.is_empty() {
            return Err(AppError::BadRequest("内容不能为空".to_string()));
        }

        if req.content.len() > 5000 {
            return Err(AppError::BadRequest("内容过长".to_string()));
        }

        // 验证图片数量
        if req.images.len() > 9 {
            return Err(AppError::BadRequest("最多只能上传9张图片".to_string()));
        }

        let moment_id = Uuid::new_v4();
        let moment = Moment {
            moment_id,
            user_id,
            content: req.content.clone(),
            images: req.images.clone(),
            tags: req.tags.clone().unwrap_or_default(),
            latitude: req.latitude,
            longitude: req.longitude,
            visibility: req.visibility.clone().unwrap_or_else(|| "public".to_string()),
            created_at: chrono::Utc::now().naive_utc(),
            edited_at: None,
            deleted_at: None,
        };

        moment_repo.create(&moment).await?;

        // TODO: 触发WebSocket推送（订阅了瞬间更新的用户）
        // TODO: 更新信息流缓存

        Ok(moment_id.to_string())
    }

    /// 更新瞬间
    pub async fn update_moment(
        &self,
        user_id: i64,
        req: &UpdateMomentRequest,
    ) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db);

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        let mut moment = moment_repo
            .find_by_id(moment_id)
            .await?
            .ok_or_else(|| AppError::NotFound("瞬间不存在".to_string()))?;

        // 只能编辑自己的瞬间
        if moment.user_id != user_id {
            return Err(AppError::Forbidden("无权编辑该瞬间".to_string()));
        }

        // 更新字段
        if let Some(content) = &req.content {
            if content.is_empty() {
                return Err(AppError::BadRequest("内容不能为空".to_string()));
            }
            if content.len() > 5000 {
                return Err(AppError::BadRequest("内容过长".to_string()));
            }
            moment.content = content.clone();
        }

        if let Some(images) = &req.images {
            if images.len() > 9 {
                return Err(AppError::BadRequest("最多只能上传9张图片".to_string()));
            }
            moment.images = images.clone();
        }

        if let Some(tags) = &req.tags {
            moment.tags = tags.clone();
        }

        if let Some(visibility) = &req.visibility {
            moment.visibility = visibility.clone();
        }

        moment_repo.update(&moment).await?;

        // TODO: 触发WebSocket推送

        Ok(())
    }

    /// 删除瞬间
    pub async fn delete_moment(&self, user_id: i64, req: &DeleteMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db);

        let moment_id = Uuid::parse_str(&req.moment_id)
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

    /// 获取我的瞬间列表
    pub async fn list_my_moments(
        &self,
        user_id: i64,
        req: &ListMyMomentsRequest,
    ) -> Result<PaginatedResponse<MomentResponse>> {
        let moment_repo = MomentRepository::new(self.db);

        let limit = req.limit.unwrap_or(20).min(100);
        let offset = req.cursor.unwrap_or(0);

        let moments = moment_repo.list_by_user(user_id, limit + 1, offset).await?;

        let has_more = moments.len() > limit as usize;
        let items = self.build_moment_responses(moments.into_iter().take(limit as usize).collect(), Some(user_id)).await?;

        let next_cursor = if has_more {
            Some(offset + limit)
        } else {
            None
        };

        Ok(PaginatedResponse {
            items,
            has_more,
            next_cursor,
        })
    }

    /// 根据位置搜索瞬间
    pub async fn search_by_location(
        &self,
        req: &SearchMomentByLocationRequest,
        current_user_id: Option<i64>,
    ) -> Result<PaginatedResponse<MomentResponse>> {
        let moment_repo = MomentRepository::new(self.db);

        crate::utils::validation::Validator::validate_coordinates(req.latitude, req.longitude)?;
        crate::utils::validation::Validator::validate_radius(req.radius_meters)?;

        let limit = req.limit.unwrap_or(20).min(100);
        let offset = req.cursor.unwrap_or(0);

        let moments_with_distance = moment_repo
            .search_by_location(
                req.latitude,
                req.longitude,
                req.radius_meters,
                limit + 1,
                offset,
            )
            .await?;

        let has_more = moments_with_distance.len() > limit as usize;
        let moments: Vec<Moment> = moments_with_distance
            .into_iter()
            .take(limit as usize)
            .map(|(moment, _distance)| moment)
            .collect();

        let items = self.build_moment_responses(moments, current_user_id).await?;

        let next_cursor = if has_more {
            Some(offset + limit)
        } else {
            None
        };

        Ok(PaginatedResponse {
            items,
            has_more,
            next_cursor,
        })
    }

    /// 根据标签搜索瞬间
    pub async fn search_by_tags(
        &self,
        req: &SearchMomentByTagsRequest,
        current_user_id: Option<i64>,
    ) -> Result<PaginatedResponse<MomentResponse>> {
        let moment_repo = MomentRepository::new(self.db);

        let limit = req.limit.unwrap_or(20).min(100);
        let offset = req.cursor.unwrap_or(0);

        let moments = moment_repo.search_by_tags(&req.tags, limit + 1, offset).await?;

        let has_more = moments.len() > limit as usize;
        let items = self.build_moment_responses(moments.into_iter().take(limit as usize).collect(), current_user_id).await?;

        let next_cursor = if has_more {
            Some(offset + limit)
        } else {
            None
        };

        Ok(PaginatedResponse {
            items,
            has_more,
            next_cursor,
        })
    }

    /// 点赞瞬间
    pub async fn like_moment(&self, user_id: i64, req: &LikeMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db);

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        // 验证瞬间存在
        if moment_repo.find_by_id(moment_id).await?.is_none() {
            return Err(AppError::NotFound("瞬间不存在".to_string()));
        }

        moment_repo.like_moment(moment_id, user_id).await?;

        // TODO: 触发通知（如果不是自己的瞬间）

        Ok(())
    }

    /// 取消点赞
    pub async fn unlike_moment(&self, user_id: i64, req: &UnlikeMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db);

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        moment_repo.unlike_moment(moment_id, user_id).await?;

        Ok(())
    }

    /// 收藏瞬间
    pub async fn collect_moment(&self, user_id: i64, req: &CollectMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db);

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        // 验证瞬间存在
        if moment_repo.find_by_id(moment_id).await?.is_none() {
            return Err(AppError::NotFound("瞬间不存在".to_string()));
        }

        moment_repo.collect_moment(moment_id, user_id).await?;

        Ok(())
    }

    /// 取消收藏
    pub async fn uncollect_moment(&self, user_id: i64, req: &UncollectMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db);

        let moment_id = Uuid::parse_str(&req.moment_id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        moment_repo.uncollect_moment(moment_id, user_id).await?;

        Ok(())
    }

    /// 获取收藏列表
    pub async fn list_collections(
        &self,
        user_id: i64,
        req: &ListCollectionsRequest,
    ) -> Result<PaginatedResponse<MomentResponse>> {
        let moment_repo = MomentRepository::new(self.db);

        let limit = req.limit.unwrap_or(20).min(100);
        let offset = req.cursor.unwrap_or(0);

        let moments = moment_repo.list_collections(user_id, limit + 1, offset).await?;

        let has_more = moments.len() > limit as usize;
        let items = self.build_moment_responses(moments.into_iter().take(limit as usize).collect(), Some(user_id)).await?;

        let next_cursor = if has_more {
            Some(offset + limit)
        } else {
            None
        };

        Ok(PaginatedResponse {
            items,
            has_more,
            next_cursor,
        })
    }

    /// 构建瞬间响应（包含点赞数、评论数、是否已点赞等）
    async fn build_moment_responses(
        &self,
        moments: Vec<Moment>,
        current_user_id: Option<i64>,
    ) -> Result<Vec<MomentResponse>> {
        let moment_repo = MomentRepository::new(self.db);
        let mut responses = Vec::new();

        for moment in moments {
            let like_count = moment_repo.get_like_count(moment.moment_id).await?;
            let comment_count = moment_repo.get_comment_count(moment.moment_id).await?;

            let is_liked = if let Some(user_id) = current_user_id {
                moment_repo.is_liked(moment.moment_id, user_id).await?
            } else {
                false
            };

            let is_collected = if let Some(user_id) = current_user_id {
                moment_repo.is_collected(moment.moment_id, user_id).await?
            } else {
                false
            };

            responses.push(MomentResponse {
                id: moment.moment_id.to_string(),
                user_id: moment.user_id.to_string(),
                content: moment.content,
                images: moment.images,
                tags: moment.tags,
                latitude: moment.latitude,
                longitude: moment.longitude,
                visibility: moment.visibility,
                like_count,
                comment_count,
                is_liked,
                is_collected,
                created_at: moment.created_at.and_utc().timestamp(),
                edited_at: moment.edited_at.map(|dt| dt.and_utc().timestamp()),
            });
        }

        Ok(responses)
    }
}

