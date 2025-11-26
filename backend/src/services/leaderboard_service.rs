use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use sqlx::PgPool;

use crate::{
    dto::{
        request::{GetCreatorLeaderboardRequest, GetMomentLeaderboardRequest, GetMyCreatorRankRequest, GetMyMomentRankRequest},
        response::{
            GetCreatorLeaderboardData, GetMomentLeaderboardData,
            CreatorRankItem, MomentRankItem,
            MyCreatorRankData, MyMomentRankData, UserInfo,
        },
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

    /// 获取创作者排行榜
    pub async fn get_creator_leaderboard(
        &self,
        req: GetCreatorLeaderboardRequest,
    ) -> Result<GetCreatorLeaderboardData, AppError> {
        let timeframe = req.timeframe.as_str();
        let limit = req.limit.unwrap_or(20).min(100) as isize;
        let offset = 0 as isize;

        // Redis key: leaderboard:creators:{timeframe}
        let redis_key = format!("leaderboard:creators:{}", timeframe);

        // 从Redis获取排行榜数据
        let mut conn = self.redis.clone();
        let results: Vec<(String, f64)> = conn
            .zrevrange_withscores(&redis_key, offset, offset + limit - 1)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        if results.is_empty() {
            return Ok(GetCreatorLeaderboardData { items: vec![] });
        }

        // 收集所有用户ID
        let user_ids: Vec<i64> = results.iter()
            .filter_map(|(user_id_str, _)| user_id_str.parse().ok())
            .collect();

        // 批量获取用户信息
        let users = self.user_repo.find_by_ids(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> = users.into_iter()
            .map(|u| (u.user_id, u))
            .collect();

        // 转换为CreatorRankItem
        let items: Vec<CreatorRankItem> = results.iter().enumerate().filter_map(|(rank_idx, (user_id_str, score))| {
            let user_id: i64 = user_id_str.parse().ok()?;
            let user = user_map.get(&user_id)?;
            Some(CreatorRankItem {
                rank: (offset + rank_idx as isize + 1) as i32,
                user: UserInfo {
                    user_id: user.user_id.to_string(),
                    nickname: user.nickname.clone(),
                    avatar_url: user.avatar_url.clone(),
                },
                score: *score as i32,
            })
        }).collect();

        Ok(GetCreatorLeaderboardData {
            items,
        })
    }

    /// 获取生活瞬间排行榜
    pub async fn get_moment_leaderboard(
        &self,
        req: GetMomentLeaderboardRequest,
    ) -> Result<GetMomentLeaderboardData, AppError> {
        let timeframe = req.timeframe.as_str();
        let limit = req.limit.unwrap_or(20).min(100) as isize;
        let offset = 0 as isize;

        // Redis key: leaderboard:moments:{timeframe}
        let redis_key = format!("leaderboard:moments:{}", timeframe);

        // 从Redis获取排行榜数据 (member is moment_id)
        let mut conn = self.redis.clone();
        let results: Vec<(String, f64)> = conn
            .zrevrange_withscores(&redis_key, offset, offset + limit - 1)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        // 转换为MomentRankItem
        let mut items = Vec::new();
        for (rank_idx, (moment_id_str, score)) in results.iter().enumerate() {
            items.push(MomentRankItem {
                rank: (offset + rank_idx as isize + 1) as i32,
                moment_id: moment_id_str.clone(),
                score: *score as i32,
            });
        }

        Ok(GetMomentLeaderboardData {
            items,
        })
    }

    /// 获取我的创作者排名
    pub async fn get_my_creator_rank(
        &self,
        user_id: i64,
        req: GetMyCreatorRankRequest,
    ) -> Result<MyCreatorRankData, AppError> {
        let timeframe = req.timeframe.as_str();
        let redis_key = format!("leaderboard:creators:{}", timeframe);
        let mut conn = self.redis.clone();

        // 获取我的排名
        let rank: Option<isize> = conn
            .zrevrank(&redis_key, user_id.to_string())
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        
        let score: Option<f64> = conn
            .zscore(&redis_key, user_id.to_string())
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        let total_participants: isize = conn
            .zcard(&redis_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        // 获取前3名
        let top_results: Vec<(String, f64)> = conn
            .zrevrange_withscores(&redis_key, 0, 2)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        // 批量获取用户信息
        let user_ids: Vec<i64> = top_results.iter()
            .filter_map(|(uid_str, _)| uid_str.parse().ok())
            .collect();
        let users = self.user_repo.find_by_ids(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> = users.into_iter()
            .map(|u| (u.user_id, u))
            .collect();

        let top_3: Vec<CreatorRankItem> = top_results.iter().enumerate().filter_map(|(rank_idx, (uid_str, sc))| {
            let uid: i64 = uid_str.parse().ok()?;
            let user = user_map.get(&uid)?;
            Some(CreatorRankItem {
                rank: (rank_idx + 1) as i32,
                user: UserInfo {
                    user_id: user.user_id.to_string(),
                    nickname: user.nickname.clone(),
                    avatar_url: user.avatar_url.clone(),
                },
                score: *sc as i32,
            })
        }).collect();

        Ok(MyCreatorRankData {
            timeframe: req.timeframe,
            scope: req.scope,
            my_rank: rank.map(|r| (r + 1) as i32),
            my_score: score.unwrap_or(0.0) as i32,
            total_participants: total_participants as i32,
            top_3,
        })
    }

    /// 获取我的瞬间排名
    /// 注意：瞬间排行榜是按瞬间排名，不是按用户排名
    /// 这里返回用户最佳瞬间的排名（暂未实现完整逻辑）
    pub async fn get_my_moment_rank(
        &self,
        _user_id: i64, // 暂未实现：需要根据用户ID查找其瞬间在排行榜中的位置
        req: GetMyMomentRankRequest,
    ) -> Result<MyMomentRankData, AppError> {
        // 暂未实现：需要在 Redis 中建立用户到瞬间的索引才能高效查询
        
        Ok(MyMomentRankData {
            timeframe: req.timeframe,
            scope: req.scope,
            my_rank: None,
            my_score: 0,
            total_participants: 0,
            top_3: vec![],
        })
    }

    /// 增加用户分数 (Internal)
    pub async fn increment_score(
        &self,
        category: &str, // "creators" or "moments"
        id: &str, // user_id or moment_id
        timeframe: &str, // "daily", "weekly", etc.
        increment: f64,
    ) -> Result<(), AppError> {
        let mut conn = self.redis.clone();
        let redis_key = format!("leaderboard:{}:{}", category, timeframe);
        let _: f64 = redis::cmd("ZINCRBY")
            .arg(&redis_key)
            .arg(increment)
            .arg(id)
            .query_async(&mut conn)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        Ok(())
    }
}
