use redis::aio::ConnectionManager as RedisConnectionManager;
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            CreateMomentRequest, EditMomentRequest, DeleteMomentRequest,
            SearchMomentsByLocationRequest, SearchMomentsByTagsRequest,
            ListMomentsRequest, LikeMomentRequest, UpdateFavoriteMomentRequest,
            ListFavoritesRequest, OpenTheDoorRequest, GetMomentLikesRequest,
        },
        response::{
            MomentInfo, MomentDetailData, MomentLikeItem, NearbyMomentItem, 
            PaginatedData, PaginationMeta, UserInfo, LocationInfo,
        },
    },
    error::{AppError, Result},
    repositories::moment_repo::{MomentRepository, MomentRow},
    repositories::user_repo::UserRepository,
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

        // moment_repo.create 签名: (user_id, description, images, tags, latitude, longitude, location_name)
        let moment = moment_repo.create(
            user_id,
            req.description.clone(),
            req.images.clone(),
            req.tags.clone().unwrap_or_default(),
            req.location.latitude,
            req.location.longitude,
            req.location.name.clone(),
        ).await?;

        Ok(moment.moment_id.to_string())
    }

    /// 编辑瞬间
    pub async fn update_moment(
        &self,
        user_id: i64,
        req: &EditMomentRequest,
    ) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

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

        // moment_repo.update 签名: (moment_id, description?, images?, tags?)
        moment_repo.update(
            moment_id,
            req.description.clone(),
            req.images.clone(),
            req.tags.clone(),
        ).await?;

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

    /// 获取我的瞬间列表 (list_my_moments -> list_moments with user_id?)
    /// Spec 3.3.8 is get_list (ListMomentsRequest).
    /// Here we implement logic for ListMomentsRequest.
    pub async fn list_moments(
        &self,
        _req_user_id: Option<i64>, // The user making the request (for auth/hidden logic if any)
        req: &ListMomentsRequest,
    ) -> Result<PaginatedData<MomentInfo>> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let user_repo = UserRepository::new(self.db.clone());

        let limit = req.limit.unwrap_or(20).min(100) as i64;
        
        // Target user_id to list moments from
        let target_user_id = if let Some(uid_str) = &req.user_id {
             Some(uid_str.parse::<i64>().map_err(|_| AppError::BadRequest("Invalid user_id".to_string()))?)
        } else {
             None
        };

        // 使用游标分页：多取一条判断has_more
        let cursor = req.cursor.as_ref().and_then(|c| Uuid::parse_str(c).ok());
        let moments = if let Some(uid) = target_user_id {
            moment_repo.list_by_user(uid, (limit + 1) as i32, cursor).await?
        } else {
            // 如果没有指定用户，返回所有公开的瞬间（或者空列表）
            vec![]
        };

        let has_more = moments.len() > limit as usize;
        let moments_to_return: Vec<MomentRow> = moments.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            moments_to_return.last().map(|m| m.moment_id.to_string())
        } else {
            None
        };

        let items = self.build_moment_infos(moments_to_return, &moment_repo, &user_repo).await?;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 根据位置搜索瞬间
    pub async fn search_by_location(
        &self,
        req: &SearchMomentsByLocationRequest,
    ) -> Result<PaginatedData<NearbyMomentItem>> {
        let moment_repo = MomentRepository::new(self.db.clone());

        // 验证坐标
        if req.latitude < -90.0 || req.latitude > 90.0 {
            return Err(AppError::InvalidInput("纬度必须在-90到90之间".to_string()));
        }
        if req.longitude < -180.0 || req.longitude > 180.0 {
            return Err(AppError::InvalidInput("经度必须在-180到180之间".to_string()));
        }
        
        let radius = req.radius.unwrap_or(5000.0);
        let limit = req.limit.unwrap_or(20).min(100) as i64;

        // 使用优化后的批量查询方法，一次性获取瞬间、用户信息、统计数据和距离
        let cursor = req.cursor.as_ref().and_then(|c| Uuid::parse_str(c).ok());
        let results = moment_repo
            .search_by_location_with_details(
                req.latitude,
                req.longitude,
                radius,
                (limit + 1) as i32,
                cursor,
            )
            .await?;

        let has_more = results.len() > limit as usize;
        let results_to_return: Vec<_> = results.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            results_to_return.last().map(|m| m.moment.moment_id.to_string())
        } else {
            None
        };

        let items = results_to_return.into_iter().map(|m| {
            NearbyMomentItem {
                moment_info: MomentInfo {
                    id: m.moment.moment_id.to_string(),
                    user: UserInfo {
                        user_id: m.moment.user_id.to_string(),
                        nickname: m.nickname,
                        avatar_url: m.avatar_url,
                    },
                    description: m.moment.description,
                    images: m.moment.images,
                    location_name: m.moment.location_name,
                    created_at: m.moment.created_at.to_rfc3339(),
                    likes_count: m.likes_count as i32,
                    comments_count: m.comments_count as i32,
                },
                distance: m.distance,
            }
        }).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 根据标签搜索瞬间
    pub async fn search_by_tags(
        &self,
        req: &SearchMomentsByTagsRequest,
    ) -> Result<PaginatedData<MomentInfo>> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let user_repo = UserRepository::new(self.db.clone());

        let limit = req.limit.unwrap_or(20).min(100) as i64;

        // search_by_tags 签名: (tags, limit, cursor)
        let cursor = req.cursor.as_ref().and_then(|c| Uuid::parse_str(c).ok());
        let moments = moment_repo.search_by_tags(&req.tags, (limit + 1) as i32, cursor).await?;

        let has_more = moments.len() > limit as usize;
        let moments_to_return: Vec<MomentRow> = moments.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            moments_to_return.last().map(|m| m.moment_id.to_string())
        } else {
            None
        };

        let items = self.build_moment_infos(moments_to_return, &moment_repo, &user_repo).await?;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 点赞/取消点赞瞬间 (Toggle Like)
    pub async fn toggle_like(&self, user_id: i64, req: &LikeMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        if moment_repo.find_by_id(moment_id).await?.is_none() {
            return Err(AppError::NotFound("瞬间不存在".to_string()));
        }

        match req.action.as_str() {
            "like" => moment_repo.like_moment(moment_id, user_id).await?,
            "unlike" => moment_repo.unlike_moment(moment_id, user_id).await?,
            _ => return Err(AppError::BadRequest("Invalid action".to_string())),
        }

        Ok(())
    }

    /// 收藏/取消收藏瞬间 (Toggle Favorite)
    pub async fn toggle_favorite(&self, user_id: i64, req: &UpdateFavoriteMomentRequest) -> Result<()> {
        let moment_repo = MomentRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        if moment_repo.find_by_id(moment_id).await?.is_none() {
            return Err(AppError::NotFound("瞬间不存在".to_string()));
        }

        match req.action.as_str() {
            "add" => moment_repo.collect_moment(moment_id, user_id).await?,
            "remove" => moment_repo.uncollect_moment(moment_id, user_id).await?,
            _ => return Err(AppError::BadRequest("Invalid action".to_string())),
        }

        Ok(())
    }

    /// 获取收藏列表 (List Favorites)
    pub async fn list_favorites(
        &self,
        user_id: i64,
        req: &ListFavoritesRequest,
    ) -> Result<PaginatedData<MomentInfo>> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let user_repo = UserRepository::new(self.db.clone());

        let limit = req.limit.unwrap_or(20).min(100) as i64;

        let cursor = req.cursor.as_ref().and_then(|c| Uuid::parse_str(c).ok());
        let moments = moment_repo.list_collections(user_id, (limit + 1) as i32, cursor).await?;

        let has_more = moments.len() > limit as usize;
        let moments_to_return: Vec<MomentRow> = moments.into_iter().take(limit as usize).collect();
        
        let next_cursor = if has_more {
            moments_to_return.last().map(|m| m.moment_id.to_string())
        } else {
            None
        };

        let items = self.build_moment_infos(moments_to_return, &moment_repo, &user_repo).await?;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// Helper: 构建 MomentInfo 列表（使用批量查询避免 N+1 问题）
    async fn build_moment_infos(
        &self,
        moments: Vec<MomentRow>,
        moment_repo: &MomentRepository,
        user_repo: &UserRepository,
    ) -> Result<Vec<MomentInfo>> {
        if moments.is_empty() {
            return Ok(Vec::new());
        }

        // 批量获取用户信息
        let user_ids: Vec<i64> = moments.iter().map(|m| m.user_id).collect();
        let users = user_repo.find_by_ids(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> = users.into_iter()
            .map(|u| (u.user_id, u))
            .collect();

        // 批量获取统计数据
        let moment_ids: Vec<Uuid> = moments.iter().map(|m| m.moment_id).collect();
        let stats = moment_repo.get_stats_batch(&moment_ids).await?;

        // 构建结果
        let infos = moments.into_iter().filter_map(|moment| {
            let user = user_map.get(&moment.user_id)?;
            let (likes_count, comments_count) = stats.get(&moment.moment_id).copied().unwrap_or((0, 0));

            Some(MomentInfo {
                id: moment.moment_id.to_string(),
                user: UserInfo {
                    user_id: user.user_id.to_string(),
                    nickname: user.nickname.clone(),
                    avatar_url: user.avatar_url.clone(),
                },
                description: moment.description,
                images: moment.images,
                location_name: moment.location_name,
                created_at: moment.created_at.to_rfc3339(),
                likes_count: likes_count as i32,
                comments_count: comments_count as i32,
            })
        }).collect();

        Ok(infos)
    }

    /// Helper: 构建单个 MomentInfo（用于单个瞬间详情查询）
    async fn build_single_moment_info(
        &self,
        moment: MomentRow,
        moment_repo: &MomentRepository,
        user_repo: &UserRepository,
    ) -> Result<MomentInfo> {
        let like_count = moment_repo.get_like_count(moment.moment_id).await?;
        let comment_count = moment_repo.get_comment_count(moment.moment_id).await?;
        
        let user = user_repo.find_by_id(moment.user_id).await?
            .ok_or_else(|| AppError::NotFound("用户不存在".to_string()))?;

        Ok(MomentInfo {
            id: moment.moment_id.to_string(),
            user: UserInfo {
                user_id: user.user_id.to_string(),
                nickname: user.nickname,
                avatar_url: user.avatar_url,
            },
            description: moment.description,
            images: moment.images,
            location_name: moment.location_name,
            created_at: moment.created_at.to_rfc3339(),
            likes_count: like_count as i32,
            comments_count: comment_count as i32,
        })
    }

    /// 获取瞬间详情
    pub async fn get_detail(
        &self,
        req: &OpenTheDoorRequest,
        current_user_id: Option<i64>,
    ) -> Result<MomentDetailData> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let user_repo = UserRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        let moment = moment_repo.find_by_id(moment_id).await?
            .ok_or_else(|| AppError::NotFound("瞬间不存在".to_string()))?;

        let user = user_repo.find_by_id(moment.user_id).await?
            .ok_or_else(|| AppError::NotFound("用户不存在".to_string()))?;

        let like_count = moment_repo.get_like_count(moment.moment_id).await?;
        let comment_count = moment_repo.get_comment_count(moment.moment_id).await?;

        // 检查当前用户是否点赞/收藏
        let (liked_by_me, favorited_by_me) = if let Some(uid) = current_user_id {
            let liked = moment_repo.is_liked_by(moment.moment_id, uid).await?;
            let favorited = moment_repo.is_favorited_by(moment.moment_id, uid).await?;
            (Some(liked), Some(favorited))
        } else {
            (None, None)
        };

        Ok(MomentDetailData {
            id: moment.moment_id.to_string(),
            user: UserInfo {
                user_id: user.user_id.to_string(),
                nickname: user.nickname,
                avatar_url: user.avatar_url,
            },
            description: moment.description,
            images: moment.images,
            tags: moment.tags,
            location: LocationInfo {
                latitude: moment.latitude,
                longitude: moment.longitude,
                name: moment.location_name.clone(),
            },
            created_at: moment.created_at.to_rfc3339(),
            updated_at: moment.updated_at.map(|t| t.to_rfc3339()),
            likes_count: like_count as i32,
            comments_count: comment_count as i32,
            liked_by_me,
            favorited_by_me,
            distance: None,
        })
    }

    /// 获取瞬间点赞列表
    pub async fn get_moment_likes(
        &self,
        req: &GetMomentLikesRequest,
    ) -> Result<PaginatedData<MomentLikeItem>> {
        let moment_repo = MomentRepository::new(self.db.clone());
        let user_repo = UserRepository::new(self.db.clone());

        let moment_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的瞬间ID".to_string()))?;

        let limit = req.limit.unwrap_or(20).min(100) as i64;
        let cursor = req.cursor.as_ref().and_then(|c| Uuid::parse_str(c).ok());

        let likes = moment_repo.list_likes(moment_id, limit + 1, cursor).await?;

        let has_more = likes.len() > limit as usize;
        let likes_to_return: Vec<_> = likes.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            likes_to_return.last().map(|l| l.cursor_id.to_string())
        } else {
            None
        };

        let mut items = Vec::new();
        for like in likes_to_return {
            if let Some(user) = user_repo.find_by_id(like.user_id).await? {
                items.push(MomentLikeItem {
                    user: UserInfo {
                        user_id: user.user_id.to_string(),
                        nickname: user.nickname,
                        avatar_url: user.avatar_url,
                    },
                    liked_at: like.created_at.to_rfc3339(),
                });
            }
        }

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }
}
