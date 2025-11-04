use redis::aio::ConnectionManager as RedisConnectionManager;
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
        response::{MomentResponse, MomentSummaryResponse, PaginatedResponse},
    },
    error::{AppError, Result},
    models::moment::Moment,
    repositories::moment_repo::MomentRepository,
};

pub struct MomentService<'a> {
    db: &'a PgPool,
    #[allow(dead_code)]
    redis: &'a RedisConnectionManager,
}

impl<'a> MomentService<'a> {
    pub fn new(db: &'a PgPool, redis: &'a RedisConnectionManager) -> Self {
        Self { db, redis }
    }

    /// 创建瞬间
    pub async fn create_moment(&self, user_id: i64, req: &CreateMomentRequest) -> Result<String> {
        let moment_repo = MomentRepository::new(self.db.clone());

        // 验证坐标
        if req.latitude < -90.0 || req.latitude > 90.0 {
            return Err(AppError::InvalidInput("纬度必须在-90到90之间".to_string()));
        }
        if req.longitude < -180.0 || req.longitude > 180.0 {
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

        let moment_id = Uuid::new_v4();
        let moment = Moment {
            moment_id,
            user_id,
            description: req.description.clone(),
            images: req.images.clone(),
            tags: req.tags.clone(),
            latitude: req.latitude,
            longitude: req.longitude,
            location_name: req.location_name.clone(),
            created_at: chrono::Utc::now(),
            updated_at: None,
            is_deleted: false,
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
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
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
        if let Some(content) = &req.description {
            if content.is_empty() {
                return Err(AppError::BadRequest("内容不能为空".to_string()));
            }
            if content.len() > 5000 {
                return Err(AppError::BadRequest("内容过长".to_string()));
            }
            moment.description = content.clone();
        }

        if let Some(images) = &req.images {
            if images.len() > 9 {
                return Err(AppError::BadRequest("最多只能上传9张图片".to_string()));
            }
            moment.images = Some(images.clone());
        }

        if let Some(tags) = &req.tags {
            moment.tags = Some(tags.clone());
        }

        moment.updated_at = Some(chrono::Utc::now());

        moment_repo.update(&moment).await?;

        // TODO: 触发WebSocket推送

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

    /// 获取我的瞬间列表
    pub async fn list_my_moments(
        &self,
        user_id: i64,
        req: &ListMyMomentsRequest,
    ) -> Result<PaginatedResponse<MomentResponse>> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let limit = req.limit.unwrap_or(20).min(100) as i64;

        // 使用游标分页：多取一条判断has_more
        let moments = moment_repo.list_by_user(user_id, req.cursor.clone(), limit + 1).await?;

        let has_more = moments.len() > limit as usize;
        let moments_to_return: Vec<Moment> = moments.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的moment_id（业务ID）
        let next_cursor = if has_more {
            moments_to_return.last().map(|m| m.moment_id.to_string())
        } else {
            None
        };

        let items = self.build_moment_responses(moments_to_return, Some(user_id)).await?;

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
        let moment_repo = MomentRepository::new(self.db.clone());

        // 验证坐标
        if req.latitude < -90.0 || req.latitude > 90.0 {
            return Err(AppError::InvalidInput("纬度必须在-90到90之间".to_string()));
        }
        if req.longitude < -180.0 || req.longitude > 180.0 {
            return Err(AppError::InvalidInput("经度必须在-180到180之间".to_string()));
        }
        
        // 验证半径
        let radius = req.radius.unwrap_or(5000.0);
        if radius <= 0.0 || radius > 50000.0 {
            return Err(AppError::InvalidInput("搜索半径必须在0到50000米之间".to_string()));
        }

        let limit = req.limit.unwrap_or(20).min(100) as i64;

        // 使用游标分页：多取一条判断has_more
        let moments_with_distance = moment_repo
            .search_by_location(
                req.latitude,
                req.longitude,
                radius,
                req.cursor.clone(),
                limit + 1,
            )
            .await?;

        let has_more = moments_with_distance.len() > limit as usize;
        let moments_to_return: Vec<(Moment, f64)> = moments_with_distance
            .into_iter()
            .take(limit as usize)
            .collect();

        // next_cursor是最后一条记录的moment_id（业务ID）
        let next_cursor = if has_more {
            moments_to_return.last().map(|(m, _)| m.moment_id.to_string())
        } else {
            None
        };

        let moments: Vec<Moment> = moments_to_return
            .into_iter()
            .map(|(moment, _distance)| moment)
            .collect();

        let items = self.build_moment_responses(moments, current_user_id).await?;

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
        let moment_repo = MomentRepository::new(self.db.clone());

        let limit = req.limit.unwrap_or(20).min(100) as i64;

        // 使用游标分页：多取一条判断has_more
        let moments = moment_repo.search_by_tags(&req.tags, req.cursor.clone(), limit + 1).await?;

        let has_more = moments.len() > limit as usize;
        let moments_to_return: Vec<Moment> = moments.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的moment_id（业务ID）
        let next_cursor = if has_more {
            moments_to_return.last().map(|m| m.moment_id.to_string())
        } else {
            None
        };

        let items = self.build_moment_responses(moments_to_return, current_user_id).await?;

        Ok(PaginatedResponse {
            items,
            has_more,
            next_cursor,
        })
    }

    /// 点赞瞬间
    pub async fn like_moment(&self, user_id: i64, req: &LikeMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
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
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        moment_repo.unlike_moment(moment_id, user_id).await?;

        Ok(())
    }

    /// 收藏瞬间
    pub async fn collect_moment(&self, user_id: i64, req: &CollectMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
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
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        moment_repo.uncollect_moment(moment_id, user_id).await?;

        Ok(())
    }

    /// 获取收藏列表（使用摘要响应，提升性能）
    pub async fn list_collections(
        &self,
        user_id: i64,
        req: &ListCollectionsRequest,
    ) -> Result<PaginatedResponse<MomentSummaryResponse>> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let limit = req.limit.unwrap_or(20).min(100) as i64;

        // 使用游标分页：多取一条判断has_more
        let moments = moment_repo.list_collections(user_id, req.cursor.clone(), limit + 1).await?;

        let has_more = moments.len() > limit as usize;
        let moments_to_return: Vec<Moment> = moments.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的moment_id（业务ID）
        let next_cursor = if has_more {
            moments_to_return.last().map(|m| m.moment_id.to_string())
        } else {
            None
        };

        let items = self.build_moment_summary_responses(moments_to_return).await?;

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
        let moment_repo = MomentRepository::new(self.db.clone());
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

            // TODO: 获取用户信息
            let user = crate::dto::response::UserResponse {
                user_id: moment.user_id.to_string(),
                nickname: "Unknown".to_string(), // TODO: 从数据库查询
                avatar_url: None,
            };
            
            responses.push(MomentResponse {
                id: moment.moment_id.to_string(),
                user,
                description: moment.description,
                images: moment.images,
                tags: moment.tags.unwrap_or_default(),
                location: crate::dto::response::LocationInfo {
                    latitude: moment.latitude,
                    longitude: moment.longitude,
                    name: moment.location_name,
                },
                created_at: moment.created_at,
                updated_at: moment.updated_at,
                likes_count: like_count as i32,
                comments_count: comment_count as i32,
                liked_by_me: Some(is_liked),
                favorited_by_me: Some(is_collected),
                distance: None, // 我的收藏列表不需要距离信息
            });
        }

        Ok(responses)
    }

    /// 构建瞬间摘要响应（用于列表场景，性能优化）
    /// 相比完整响应，不查询 liked_by_me 和 favorited_by_me，减少 JOIN 查询
    async fn build_moment_summary_responses(
        &self,
        moments: Vec<Moment>,
    ) -> Result<Vec<MomentSummaryResponse>> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let mut responses = Vec::new();

        for moment in moments {
            // 只查询点赞数和评论数，不查询当前用户是否已点赞/收藏
            let like_count = moment_repo.get_like_count(moment.moment_id).await?;
            let comment_count = moment_repo.get_comment_count(moment.moment_id).await?;

            // TODO: 获取用户信息（后续可以批量查询优化）
            let user = crate::dto::response::UserResponse {
                user_id: moment.user_id.to_string(),
                nickname: "Unknown".to_string(), // TODO: 从数据库查询
                avatar_url: None,
            };
            
            responses.push(MomentSummaryResponse {
                id: moment.moment_id.to_string(),
                user,
                description: moment.description,
                images: moment.images,
                location_name: moment.location_name,
                created_at: moment.created_at,
                likes_count: like_count as i32,
                comments_count: comment_count as i32,
            });
        }

        Ok(responses)
    }
}

