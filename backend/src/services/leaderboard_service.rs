use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use sqlx::PgPool;

use crate::{
    dto::{
        request::{GetLeaderboardRequest, UpdateUserStatsRequest},
        response::{LeaderboardEntry, LeaderboardResponse},
    },
    error::AppError,
    repositories::user_repo::UserRepository,
};

#[derive(Clone)]
pub struct LeaderboardService {
    redis: ConnectionManager,
    user_repo: UserRepository,
}

impl LeaderboardService {
    pub fn new(pool: PgPool, redis: ConnectionManager) -> Self {
        Self {
            redis,
            user_repo: UserRepository::new(pool),
        }
    }

    /// 获取排行榜
    pub async fn get_leaderboard(
        &self,
        req: GetLeaderboardRequest,
    ) -> Result<LeaderboardResponse, AppError> {
        let category = req.category.as_str();
        let limit = req.limit.unwrap_or(20).min(100) as isize;
        let offset = 0 as isize; // 暂时使用0，后续可根据timeframe调整

        // 验证排行榜类型
        let valid_categories = vec!["creators", "moments"];
        if !valid_categories.contains(&category) {
            return Err(AppError::InvalidInput(format!(
                "无效的排行榜类别: {}",
                category
            )));
        }

        // Redis key
        let redis_key = format!("leaderboard:{}", category);

        // 从Redis获取排行榜数据
        let mut conn = self.redis.clone();
        let results: Vec<(String, f64)> = conn
            .zrevrange_withscores(&redis_key, offset, offset + limit - 1)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        // 获取总数
        let total: isize = conn
            .zcard(&redis_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        // 转换为LeaderboardEntry
        let mut entries = Vec::new();
        for (rank_idx, (user_id_str, score)) in results.iter().enumerate() {
            let user_id: i64 = user_id_str
                .parse()
                .map_err(|_| AppError::InternalError("无效的用户ID".to_string()))?;

            // 获取用户信息
            let user = self.user_repo.find_by_id(user_id).await?;

            if let Some(user) = user {
                entries.push(LeaderboardEntry {
                    rank: (offset + rank_idx as isize + 1) as i32,
                    user_id: user.user_id.to_string(),
                    nickname: user.nickname,
                    avatar_url: user.avatar_url,
                    score: *score,
                    stats: None,
                    distance: None,
                });
            }
        }

        let has_more = (offset + limit) < total;
        
        // 计算next_cursor（如果有更多数据）
        let next_cursor = if has_more {
            // 使用最后一个条目的排名作为cursor
            entries.last().map(|e| e.rank.to_string())
        } else {
            None
        };

        Ok(LeaderboardResponse {
            category: category.to_string(),
            timeframe: req.timeframe.clone(),
            scope: req.scope.clone(),
            items: entries,
            has_more,
            next_cursor,
        })
    }

    /// 更新用户统计数据（内部使用）
    pub async fn update_user_stats(
        &self,
        req: UpdateUserStatsRequest,
    ) -> Result<(), AppError> {
        let mut conn = self.redis.clone();

        // 根据stats_type更新对应的排行榜
        let redis_key = format!("leaderboard:{}", req.stats_type);
        let _: () = conn
            .zadd(&redis_key, &req.user_id, req.value as f64)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        Ok(())
    }

    /// 增加用户活跃度分数（内部使用）
    pub async fn increment_active_score(
        &self,
        user_id: i64,
        increment: f64,
    ) -> Result<(), AppError> {
        let mut conn = self.redis.clone();
        let redis_key = "leaderboard:active";
        let _: () = conn
            .zadd(redis_key, user_id.to_string(), increment)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        Ok(())
    }

    /// 增加用户人气分数（内部使用）
    pub async fn increment_popular_score(
        &self,
        user_id: i64,
        increment: f64,
    ) -> Result<(), AppError> {
        let mut conn = self.redis.clone();
        let redis_key = "leaderboard:popular";
        // 使用 zadd with INCR 选项
        let _: f64 = redis::cmd("ZINCRBY")
            .arg(redis_key)
            .arg(increment)
            .arg(user_id.to_string())
            .query_async(&mut conn)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        Ok(())
    }

    /// 增加用户贡献度分数（内部使用）
    pub async fn increment_contributor_score(
        &self,
        user_id: i64,
        increment: f64,
    ) -> Result<(), AppError> {
        let mut conn = self.redis.clone();
        let redis_key = "leaderboard:contributor";
        // 使用 zadd with INCR 选项
        let _: f64 = redis::cmd("ZINCRBY")
            .arg(redis_key)
            .arg(increment)
            .arg(user_id.to_string())
            .query_async(&mut conn)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        Ok(())
    }

    /// 获取用户排名（内部使用）
    pub async fn get_user_rank(
        &self,
        user_id: i64,
        leaderboard_type: &str,
    ) -> Result<Option<i64>, AppError> {
        let mut conn = self.redis.clone();
        let redis_key = format!("leaderboard:{}", leaderboard_type);
        let rank: Option<isize> = conn
            .zrevrank(&redis_key, user_id.to_string())
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        Ok(rank.map(|r| (r + 1) as i64))
    }

    /// 获取用户分数（内部使用）
    pub async fn get_user_score(
        &self,
        user_id: i64,
        leaderboard_type: &str,
    ) -> Result<Option<f64>, AppError> {
        let mut conn = self.redis.clone();
        let redis_key = format!("leaderboard:{}", leaderboard_type);
        let score: Option<f64> = conn
            .zscore(&redis_key, user_id.to_string())
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        Ok(score)
    }

    /// 重置排行榜（管理员使用）
    pub async fn reset_leaderboard(&self, leaderboard_type: &str) -> Result<(), AppError> {
        let mut conn = self.redis.clone();
        let redis_key = format!("leaderboard:{}", leaderboard_type);
        let _: () = conn
            .del(&redis_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        Ok(())
    }

    /// 获取排行榜（别名方法）
    pub async fn list_leaderboard(
        &self,
        req: crate::dto::request::GetLeaderboardRequest,
    ) -> Result<crate::dto::response::LeaderboardResponse, AppError> {
        self.get_leaderboard(req).await
    }

    /// 获取我的排名（别名方法）
    pub async fn get_my_rank(
        &self,
        _user_id: i64,
        _req: crate::dto::request::GetMyRankRequest,
    ) -> Result<crate::dto::response::GetMyRankResponse, AppError> {
        // TODO: 实现获取我的排名逻辑
        Err(AppError::NotImplemented("功能开发中".to_string()))
    }
}

