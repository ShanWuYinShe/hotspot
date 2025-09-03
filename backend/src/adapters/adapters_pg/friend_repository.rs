/// PostgreSQL好友仓储实现
use async_trait::async_trait;
use sqlx::PgPool;
use tracing::{debug, error};
use uuid::Uuid;

use crate::domain::{DomainError, DomainResult};
use crate::ports::repository_ports::{FriendRepositoryPort, FriendshipInfo, FriendshipStatus};

/// PostgreSQL好友仓储
#[derive(Clone)]
pub struct PgFriendRepository {
    pool: PgPool,
}

impl PgFriendRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl FriendRepositoryPort for PgFriendRepository {
    async fn send_friend_request(&self, requester_id: i64, addressee_id: i64) -> DomainResult<FriendshipInfo> {
        debug!("发送好友请求: {} -> {}", requester_id, addressee_id);

        // 检查是否已经存在好友关系
        let existing = sqlx::query!(
            r#"
            SELECT status FROM friendships 
            WHERE (requester_id = $1 AND addressee_id = $2) 
               OR (requester_id = $2 AND addressee_id = $1)
            "#,
            requester_id,
            addressee_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| {
            error!("检查现有好友关系失败: {:?}", e);
            DomainError::BusinessRuleViolation { rule: format!("数据库错误: {}", e) }
        })?;

        if existing.is_some() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "好友关系已存在".to_string(),
            });
        }

        // 创建好友请求
        let friendship_id = Uuid::new_v4();
        let result = sqlx::query!(
            r#"
            INSERT INTO friendships (friendship_id, requester_id, addressee_id, status, created_at, updated_at)
            VALUES ($1, $2, $3, 'pending', NOW(), NOW())
            RETURNING friendship_id, requester_id, addressee_id, status, created_at, updated_at
            "#,
            friendship_id,
            requester_id,
            addressee_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("创建好友请求失败: {:?}", e);
            DomainError::BusinessRuleViolation { rule: format!("数据库错误: {}", e) }
        })?;

        Ok(FriendshipInfo {
            friendship_id: result.friendship_id.to_string(),
            requester_id: result.requester_id,
            addressee_id: result.addressee_id,
            status: FriendshipStatus::Pending,
            created_at: result.created_at,
            updated_at: result.updated_at,
        })
    }

    async fn accept_friend_request(&self, requester_id: i64, addressee_id: i64) -> DomainResult<()> {
        debug!("接受好友请求: {} -> {}", requester_id, addressee_id);

        let result = sqlx::query!(
            r#"
            UPDATE friendships 
            SET status = 'accepted', updated_at = NOW()
            WHERE requester_id = $1 AND addressee_id = $2 AND status = 'pending'
            "#,
            requester_id,
            addressee_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("接受好友请求失败: {:?}", e);
            DomainError::BusinessRuleViolation { rule: format!("数据库错误: {}", e) }
        })?;

        if result.rows_affected() == 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "好友请求不存在或已处理".to_string(),
            });
        }

        Ok(())
    }

    async fn reject_friend_request(&self, requester_id: i64, addressee_id: i64) -> DomainResult<()> {
        debug!("拒绝好友请求: {} -> {}", requester_id, addressee_id);

        let result = sqlx::query!(
            r#"
            UPDATE friendships 
            SET status = 'rejected', updated_at = NOW()
            WHERE requester_id = $1 AND addressee_id = $2 AND status = 'pending'
            "#,
            requester_id,
            addressee_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("拒绝好友请求失败: {:?}", e);
            DomainError::BusinessRuleViolation { rule: format!("数据库错误: {}", e) }
        })?;

        if result.rows_affected() == 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "好友请求不存在或已处理".to_string(),
            });
        }

        Ok(())
    }

    async fn remove_friendship(&self, user_id: i64, friend_id: i64) -> DomainResult<()> {
        debug!("删除好友关系: {} <-> {}", user_id, friend_id);

        let result = sqlx::query!(
            r#"
            DELETE FROM friendships 
            WHERE ((requester_id = $1 AND addressee_id = $2) 
                OR (requester_id = $2 AND addressee_id = $1))
              AND status = 'accepted'
            "#,
            user_id,
            friend_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("删除好友关系失败: {:?}", e);
            DomainError::BusinessRuleViolation { rule: format!("数据库错误: {}", e) }
        })?;

        if result.rows_affected() == 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "好友关系不存在".to_string(),
            });
        }

        Ok(())
    }

    async fn get_friends(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<i64>> {
        debug!("获取用户{}的好友列表", user_id);

        let friends = sqlx::query!(
            r#"
            SELECT 
                CASE 
                    WHEN requester_id = $1 THEN addressee_id 
                    ELSE requester_id 
                END as friend_id
            FROM friendships 
            WHERE (requester_id = $1 OR addressee_id = $1) 
              AND status = 'accepted'
              AND id > $2
            ORDER BY id ASC
            LIMIT $3
            "#,
            user_id,
            cursor,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("获取好友列表失败: {:?}", e);
            DomainError::BusinessRuleViolation { rule: format!("数据库错误: {}", e) }
        })?;

        Ok(friends.into_iter().filter_map(|row| row.friend_id).collect())
    }

    async fn get_received_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendshipInfo>> {
        debug!("获取用户{}收到的好友请求", user_id);

        let requests = sqlx::query!(
            r#"
            SELECT friendship_id, requester_id, addressee_id, status, created_at, updated_at
            FROM friendships 
            WHERE addressee_id = $1 AND status = 'pending' AND id > $2
            ORDER BY id ASC
            LIMIT $3
            "#,
            user_id,
            cursor,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("获取收到的好友请求失败: {:?}", e);
            DomainError::BusinessRuleViolation { rule: format!("数据库错误: {}", e) }
        })?;

        Ok(requests.into_iter().map(|row| FriendshipInfo {
            friendship_id: row.friendship_id.to_string(),
            requester_id: row.requester_id,
            addressee_id: row.addressee_id,
            status: match row.status.as_str() {
                "pending" => FriendshipStatus::Pending,
                "accepted" => FriendshipStatus::Accepted,
                "rejected" => FriendshipStatus::Rejected,
                "blocked" => FriendshipStatus::Blocked,
                _ => FriendshipStatus::Pending,
            },
            created_at: row.created_at,
            updated_at: row.updated_at,
        }).collect())
    }

    async fn get_sent_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendshipInfo>> {
        debug!("获取用户{}发送的好友请求", user_id);

        let requests = sqlx::query!(
            r#"
            SELECT friendship_id, requester_id, addressee_id, status, created_at, updated_at
            FROM friendships 
            WHERE requester_id = $1 AND status = 'pending' AND id > $2
            ORDER BY id ASC
            LIMIT $3
            "#,
            user_id,
            cursor,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("获取发送的好友请求失败: {:?}", e);
            DomainError::BusinessRuleViolation { rule: format!("数据库错误: {}", e) }
        })?;

        Ok(requests.into_iter().map(|row| FriendshipInfo {
            friendship_id: row.friendship_id.to_string(),
            requester_id: row.requester_id,
            addressee_id: row.addressee_id,
            status: match row.status.as_str() {
                "pending" => FriendshipStatus::Pending,
                "accepted" => FriendshipStatus::Accepted,
                "rejected" => FriendshipStatus::Rejected,
                "blocked" => FriendshipStatus::Blocked,
                _ => FriendshipStatus::Pending,
            },
            created_at: row.created_at,
            updated_at: row.updated_at,
        }).collect())
    }

    async fn are_friends(&self, user_id1: i64, user_id2: i64) -> DomainResult<bool> {
        debug!("检查用户{}和{}是否是好友", user_id1, user_id2);

        let result = sqlx::query!(
            r#"
            SELECT COUNT(*) as count
            FROM friendships 
            WHERE ((requester_id = $1 AND addressee_id = $2) 
                OR (requester_id = $2 AND addressee_id = $1))
              AND status = 'accepted'
            "#,
            user_id1,
            user_id2
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("检查好友关系失败: {:?}", e);
            DomainError::BusinessRuleViolation { rule: format!("数据库错误: {}", e) }
        })?;

        Ok(result.count.unwrap_or(0) > 0)
    }

    async fn get_pending_requests_count(&self, user_id: i64) -> DomainResult<i64> {
        debug!("获取用户{}的待处理好友请求数量", user_id);

        let result = sqlx::query!(
            r#"
            SELECT COUNT(*) as count
            FROM friendships 
            WHERE addressee_id = $1 AND status = 'pending'
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("获取待处理好友请求数量失败: {:?}", e);
            DomainError::BusinessRuleViolation { rule: format!("数据库错误: {}", e) }
        })?;

        Ok(result.count.unwrap_or(0))
    }
}
