/// PostgreSQL好友仓储实现
/// 
/// 功能职责：
/// - 实现好友关系的数据库CRUD操作
/// - 提供好友请求管理功能
/// - 支持好友关系查询和状态检查
/// - 管理好友请求的生命周期
use async_trait::async_trait;
use chrono::Utc;
use sqlx::PgPool;

use crate::{
    domain::{DomainResult, DomainError, UserId},
    domain::entities::{FriendRequest, Friendship, FriendProfile, FriendshipStatus, FriendRequestStatus},
    ports::repositories::FriendRepository,
};
use super::query_optimizer::PgQueryOptimizer;


/// PostgreSQL好友仓储
#[derive(Clone)]
pub struct PgFriendRepository {
    pool: PgPool,
    query_optimizer: PgQueryOptimizer,
}

impl PgFriendRepository {
    /// 创建新的好友仓储实例
    pub fn new(pool: PgPool) -> Self {
        // 强制启用查询优化器
        let query_optimizer = PgQueryOptimizer::new(pool.clone());
        Self { 
            pool,
            query_optimizer,
        }
    }
    
    /// 执行查询并进行性能监控
    async fn execute_query_with_monitoring<F, T>(
        &self,
        operation_name: &str,
        query_fn: F,
    ) -> DomainResult<T>
    where
        F: std::future::Future<Output = DomainResult<T>>,
    {
        // 强制使用查询优化器进行监控和优化
        let params_summary = std::collections::HashMap::new();
        
        let (result, metrics) = self.query_optimizer.execute_with_monitoring(
            operation_name,
            params_summary,
            query_fn
        ).await?;
        
        // 记录性能指标
        if let Some(ref perf_metrics) = metrics {
            if perf_metrics.execution_time_ms > 100 {
                tracing::warn!(
                    operation = operation_name,
                    duration_ms = perf_metrics.execution_time_ms,
                    query_type = perf_metrics.query_type,
                    used_index = perf_metrics.used_index,
                    repository = "friend_repository",
                    "慢查询检测"
                );
            } else {
                tracing::debug!(
                    operation = operation_name,
                    duration_ms = perf_metrics.execution_time_ms,
                    query_type = perf_metrics.query_type,
                    used_index = perf_metrics.used_index,
                    repository = "friend_repository",
                    "查询完成"
                );
            }
        }
        
        Ok(result)
    }


}

#[async_trait]
impl FriendRepository for PgFriendRepository {
    /// 发送好友请求
    async fn send_friend_request(
        &self,
        requester_id: i64,
        target_id: i64,
        message: Option<&str>,
    ) -> DomainResult<FriendRequest> {
        self.execute_query_with_monitoring("send_friend_request", async {
            // 检查是否已存在待处理的请求
            let existing = sqlx::query!(
                "SELECT id FROM friendships 
                 WHERE user_id = $1 AND friend_id = $2 AND status = 'pending'",
                requester_id,
                target_id
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("数据库错误: {}", e),
            })?;

            if existing.is_some() {
                return Err(DomainError::BusinessRuleViolation {
                    rule: "已存在待处理的好友请求".to_string(),
                });
            }

        // 创建新的好友请求
        let now = Utc::now();

        let result = sqlx::query!(
            "INSERT INTO friendships (user_id, friend_id, status, notes, created_at)
             VALUES ($1, $2, 'pending', $3, $4) RETURNING id",
            requester_id,
            target_id,
            message,
            now
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

            Ok(FriendRequest {
                request_id: result.id.to_string(),
                requester_id,
                target_id,
                status: FriendRequestStatus::Pending,
                message: message.map(|s| s.to_string()),
                created_at: now,
                updated_at: now,
                expires_at: now + chrono::Duration::days(7),
            })
        }).await
    }

    /// 接受好友请求
    async fn accept_friend_request(&self, request_id: &str, target_id: i64) -> DomainResult<Friendship> {
        let request_id_num: i64 = request_id.parse()
            .map_err(|_| DomainError::BusinessRuleViolation {
                rule: "无效的请求ID格式".to_string(),
            })?;

        let mut tx = self.pool.begin().await
            .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        // 更新好友请求状态
        let request = sqlx::query!(
            "UPDATE friendships 
             SET status = 'accepted', updated_at = NOW()
             WHERE id = $1 AND friend_id = $2 AND status = 'pending'
             RETURNING user_id, friend_id",
            request_id_num,
            target_id
        )
        .fetch_optional(&mut *tx)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        let request = request.ok_or_else(|| 
            DomainError::BusinessRuleViolation {
                rule: "好友请求不存在或已处理".to_string(),
            })?;

        tx.commit().await
            .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        Ok(Friendship {
            friendship_id: request_id_num.to_string(),
            user1_id: request.user_id.min(request.friend_id),
            user2_id: request.user_id.max(request.friend_id),
            created_at: Utc::now(),
        })
    }

    /// 拒绝好友请求
    async fn reject_friend_request(&self, request_id: &str, target_id: i64) -> DomainResult<()> {
        let request_id_num: i64 = request_id.parse()
            .map_err(|_| DomainError::BusinessRuleViolation {
                rule: "无效的请求ID格式".to_string(),
            })?;

        let result = sqlx::query!(
            "UPDATE friendships 
             SET status = 'declined', updated_at = NOW()
             WHERE id = $1 AND friend_id = $2 AND status = 'pending'",
            request_id_num,
            target_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        if result.rows_affected() == 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "好友请求不存在或已处理".to_string(),
            });
        }

        Ok(())
    }

    /// 移除好友关系
    async fn remove_friendship(&self, user_id: i64, friend_id: i64) -> DomainResult<()> {
        let result = sqlx::query!(
            "DELETE FROM friendships 
             WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)",
            user_id,
            friend_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        if result.rows_affected() == 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "好友关系不存在".to_string(),
            });
        }

        Ok(())
    }

    /// 获取用户的好友列表 - 优化版本使用UNION避免复杂CASE语句
    async fn get_friends(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendProfile>> {
        let friends = sqlx::query!(
            r#"
            SELECT u.id as "user_id!", u.nickname as "nickname!", u.avatar_url, f.created_at as "created_at!"
            FROM (
                SELECT f.friend_id, f.created_at
                FROM friendships f
                WHERE f.user_id = $1 AND f.status = 'accepted'
                UNION
                SELECT f.user_id as friend_id, f.created_at
                FROM friendships f
                WHERE f.friend_id = $1 AND f.status = 'accepted'
            ) f
            JOIN users u ON u.id = f.friend_id
            WHERE ($2 = 0 OR u.id > $2)
            ORDER BY u.id ASC
            LIMIT $3
            "#,
            user_id,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        let friend_profiles: Vec<FriendProfile> = friends
            .into_iter()
            .map(|row| FriendProfile {
                user_id: row.user_id,
                nickname: row.nickname,
                avatar_url: row.avatar_url,
                last_active_at: None, // TODO: 实现在线状态检测
                friendship_created_at: row.created_at,
                is_online: false, // TODO: 实现在线状态检测
            })
            .collect();

        Ok(friend_profiles)
    }

    /// 获取收到的好友请求
    async fn get_received_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendRequest>> {
        let requests = sqlx::query!(
            "SELECT id, user_id, friend_id, status, notes, created_at, updated_at
             FROM friendships
             WHERE friend_id = $1 AND status = 'pending'
               AND ($2 = 0 OR user_id > $2)
             ORDER BY user_id ASC
             LIMIT $3",
            user_id,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        let friend_requests: Vec<FriendRequest> = requests
            .into_iter()
            .map(|row| FriendRequest {
                request_id: row.id.to_string(),
                requester_id: row.user_id,
                target_id: row.friend_id,
                status: FriendRequestStatus::Pending,
                message: row.notes,
                created_at: row.created_at.unwrap_or_else(|| Utc::now()),
                updated_at: row.updated_at.unwrap_or_else(|| row.created_at.unwrap_or_else(|| Utc::now())),
                expires_at: row.created_at.unwrap_or_else(|| Utc::now()) + chrono::Duration::days(7),
            })
            .collect();

        Ok(friend_requests)
    }

    /// 获取发送的好友请求
    async fn get_sent_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendRequest>> {
        let requests = sqlx::query!(
            "SELECT id, user_id, friend_id, status, notes, created_at, updated_at
             FROM friendships
             WHERE user_id = $1 AND status = 'pending'
               AND ($2 = 0 OR friend_id > $2)
             ORDER BY friend_id ASC
             LIMIT $3",
            user_id,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        let friend_requests: Vec<FriendRequest> = requests
            .into_iter()
            .map(|row| FriendRequest {
                request_id: row.id.to_string(),
                requester_id: row.user_id,
                target_id: row.friend_id,
                status: FriendRequestStatus::Pending,
                message: row.notes,
                created_at: row.created_at.unwrap_or_else(|| Utc::now()),
                updated_at: row.updated_at.unwrap_or_else(|| row.created_at.unwrap_or_else(|| Utc::now())),
                expires_at: row.created_at.unwrap_or_else(|| Utc::now()) + chrono::Duration::days(7),
            })
            .collect();

        Ok(friend_requests)
    }

    /// 检查好友关系状态
    async fn check_friendship(&self, user_id: i64, other_user_id: i64) -> DomainResult<FriendshipStatus> {
        // 检查是否是好友
        let friendship: Option<String> = sqlx::query_scalar!(
            "SELECT status FROM friendships 
             WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)",
            user_id,
            other_user_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?
        .flatten();

        if let Some(status) = friendship {
            return match status.as_str() {
                "accepted" => Ok(FriendshipStatus::Friends),
                "blocked" => Ok(FriendshipStatus::Blocked),
                "pending" => Ok(FriendshipStatus::RequestSent),
                "declined" => Ok(FriendshipStatus::NotFriends),
                _ => Ok(FriendshipStatus::NotFriends),
            };
        }

        Ok(FriendshipStatus::NotFriends)
    }

    /// 获取待处理的好友请求数量
    async fn get_pending_requests_count(&self, user_id: i64) -> DomainResult<i64> {
        let count = sqlx::query_scalar!(
            "SELECT COUNT(*)::bigint as \"count!\" FROM friendships WHERE friend_id = $1 AND status = 'pending'",
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        Ok(count)
    }

    // === 缺失的 trait 方法实现 ===
    
    async fn save(&self, friendship: &Friendship) -> DomainResult<()> {
        sqlx::query!(
            "INSERT INTO friendships (user_id, friend_id, status, created_at)
             VALUES ($1, $2, 'accepted', $3)
             ON CONFLICT (user_id, friend_id) 
             DO UPDATE SET status = 'accepted', updated_at = NOW()",
            friendship.user1_id,
            friendship.user2_id,
            friendship.created_at
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        Ok(())
    }

    async fn find_friends_by_user_id(&self, user_id: UserId) -> DomainResult<Vec<Friendship>> {
        let friendships = sqlx::query!(
            "SELECT id, user_id, friend_id, created_at
             FROM friendships
             WHERE (user_id = $1 OR friend_id = $1) AND status = 'accepted'",
            *user_id
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        let result: Vec<Friendship> = friendships
            .into_iter()
            .map(|row| Friendship {
                friendship_id: row.id.to_string(),
                user1_id: row.user_id.min(row.friend_id),
                user2_id: row.user_id.max(row.friend_id),
                created_at: row.created_at.unwrap_or_else(|| Utc::now()),
            })
            .collect();

        Ok(result)
    }
}
