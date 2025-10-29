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

#[derive(Debug, Clone)]
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
        let leaderboard_type = req.leaderboard_type.as_str();
        let limit = req.limit.min(100) as isize;
        let offset = req.offset.unwrap_or(0) as isize;

        // 验证排行榜类型
        let valid_types = vec!["active", "popular", "contributor"];
        if !valid_types.contains(&leaderboard_type) {
            return Err(AppError::InvalidInput(format!(
                "无效的排行榜类型: {}",
                leaderboard_type
            )));
        }

        // Redis key
        let redis_key = format!("leaderboard:{}", leaderboard_type);

        // 从Redis获取排行榜数据
        let mut conn = self.redis.clone();
        let results: Vec<(String, f64)> = conn
            .zrevrange_withscores(&redis_key, offset, offset + limit - 1)
            .await
            .map_err(|e| AppError::Internal(format!("Redis错误: {}", e)))?;

        // 获取总数
        let total: isize = conn
            .zcard(&redis_key)
            .await
            .map_err(|e| AppError::Internal(format!("Redis错误: {}", e)))?;

        // 转换为LeaderboardEntry
        let mut entries = Vec::new();
        for (rank_idx, (user_id_str, score)) in results.iter().enumerate() {
            let user_id: i64 = user_id_str
                .parse()
                .map_err(|_| AppError::Internal("无效的用户ID".to_string()))?;

            // 获取用户信息
            let user = self.user_repo.find_user_by_id(user_id).await?;

            if let Some(user) = user {
                entries.push(LeaderboardEntry {
                    rank: (offset + rank_idx as isize + 1) as i32,
                    user_id: user.id,
                    username: user.username,
                    nickname: user.nickname,
                    avatar_url: user.avatar_url,
                    score: *score as i64,
                });
            }
        }

        let has_more = (offset + limit) < total;

        Ok(LeaderboardResponse {
            leaderboard_type: leaderboard_type.to_string(),
            entries,
            has_more,
            total: total as i64,
        })
    }

    /// 更新用户统计数据（内部使用）
    pub async fn update_user_stats(
        &self,
        req: UpdateUserStatsRequest,
    ) -> Result<(), AppError> {
        let mut conn = self.redis.clone();

        // 更新活跃度排行榜
        if let Some(active_score) = req.active_score {
            let redis_key = "leaderboard:active";
            let _: () = conn
                .zadd(redis_key, req.user_id.to_string(), active_score)
                .await
                .map_err(|e| AppError::Internal(format!("Redis错误: {}", e)))?;
        }

        // 更新人气排行榜
        if let Some(popular_score) = req.popular_score {
            let redis_key = "leaderboard:popular";
            let _: () = conn
                .zadd(redis_key, req.user_id.to_string(), popular_score)
                .await
                .map_err(|e| AppError::Internal(format!("Redis错误: {}", e)))?;
        }

        // 更新贡献度排行榜
        if let Some(contributor_score) = req.contributor_score {
            let redis_key = "leaderboard:contributor";
            let _: () = conn
                .zadd(redis_key, req.user_id.to_string(), contributor_score)
                .await
                .map_err(|e| AppError::Internal(format!("Redis错误: {}", e)))?;
        }

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
            .map_err(|e| AppError::Internal(format!("Redis错误: {}", e)))?;
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
        let _: () = conn
            .zincrby(redis_key, increment, user_id.to_string())
            .await
            .map_err(|e| AppError::Internal(format!("Redis错误: {}", e)))?;
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
        let _: () = conn
            .zincrby(redis_key, increment, user_id.to_string())
            .await
            .map_err(|e| AppError::Internal(format!("Redis错误: {}", e)))?;
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
            .map_err(|e| AppError::Internal(format!("Redis错误: {}", e)))?;

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
            .map_err(|e| AppError::Internal(format!("Redis错误: {}", e)))?;

        Ok(score)
    }

    /// 重置排行榜（管理员使用）
    pub async fn reset_leaderboard(&self, leaderboard_type: &str) -> Result<(), AppError> {
        let mut conn = self.redis.clone();
        let redis_key = format!("leaderboard:{}", leaderboard_type);
        let _: () = conn
            .del(&redis_key)
            .await
            .map_err(|e| AppError::Internal(format!("Redis错误: {}", e)))?;
        Ok(())
    }
}

