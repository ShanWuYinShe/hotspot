/// PostgreSQL好友关系仓储实现
/// 
/// 功能职责：
/// - 实现好友关系的数据库持久化操作
/// - 处理好友请求的创建、查询和状态更新
/// - 管理好友关系的建立和删除
/// - 提供好友列表和好友请求列表的分页查询
use async_trait::async_trait;
use chrono::{DateTime, Utc};
use sqlx::PgPool;
use tracing::{debug, error};

use crate::domain::entities::friend::{
    Friendship, FriendRequest, FriendProfile, FriendRequestStatus, FriendshipStatus
};
use crate::domain::{DomainError, DomainResult};
use crate::ports::repositories::FriendRepository;

/// PostgreSQL好友关系仓储
pub struct PgFriendRepository {
    pool: PgPool,
}

impl PgFriendRepository {
    /// 创建新的PostgreSQL好友关系仓储实例
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl FriendRepository for PgFriendRepository {
    /// 保存好友关系
    async fn save(&self, friendship: &Friendship) -> DomainResult<()> {
        debug!("保存好友关系: 用户1={}, 用户2={}", 
               friendship.user1_id, friendship.user2_id);
        
        let mut tx = self.pool.begin().await.map_err(|e| {
            error!("开始事务失败: {}", e);
            DomainError::DatabaseError { 
                operation: "begin_transaction".to_string(),
                details: e.to_string()
            }
        })?;
        
        // 检查关系是否已存在
        let existing = sqlx::query!(
            "SELECT relationship_id FROM user_relationships 
             WHERE (from_user_id = $1 AND to_user_id = $2 AND relationship_type = 'friend') 
                OR (from_user_id = $2 AND to_user_id = $1 AND relationship_type = 'friend')",
            friendship.user1_id,
            friendship.user2_id
        )
        .fetch_optional(&mut *tx)
        .await
        .map_err(|e| {
            error!("查询现有关系失败: {}", e);
            DomainError::DatabaseError { 
                operation: "check_existing_friendship".to_string(),
                details: e.to_string()
            }
        })?;
        
        if existing.is_some() {
            // 更新现有关系
            sqlx::query!(
                "UPDATE user_relationships 
                 SET relationship_status = 'accepted', updated_at = NOW()
                 WHERE (from_user_id = $1 AND to_user_id = $2 AND relationship_type = 'friend')
                    OR (from_user_id = $2 AND to_user_id = $1 AND relationship_type = 'friend')",
                friendship.user1_id,
                friendship.user2_id
            )
            .execute(&mut *tx)
            .await
            .map_err(|e| {
                error!("更新好友关系失败: {}", e);
                DomainError::DatabaseError { 
                    operation: "update_friendship".to_string(),
                    details: e.to_string()
                }
            })?;
        } else {
            // 插入新关系
            sqlx::query!(
                "INSERT INTO user_relationships (from_user_id, to_user_id, relationship_type, relationship_status, created_at, updated_at)
                 VALUES ($1, $2, 'friend', 'accepted', $3, $3)",
                friendship.user1_id,
                friendship.user2_id,
                friendship.created_at
            )
            .execute(&mut *tx)
            .await
            .map_err(|e| {
                error!("插入好友关系失败: {}", e);
                DomainError::DatabaseError { 
                    operation: "insert_friendship".to_string(),
                    details: e.to_string()
                }
            })?;
        }
        
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {}", e);
            DomainError::DatabaseError { 
                operation: "commit_transaction".to_string(),
                details: e.to_string()
            }
        })?;
        
        debug!("好友关系保存成功");
        Ok(())
    }

    /// 根据用户ID查找好友关系
    async fn find_friends_by_user_id(&self, user_id: i64) -> DomainResult<Vec<Friendship>> {
        debug!("查找用户好友关系: user_id={}", user_id);
        
        let rows = sqlx::query!(
            "SELECT from_user_id, to_user_id, created_at
             FROM user_relationships 
             WHERE (from_user_id = $1 OR to_user_id = $1) 
               AND relationship_type = 'friend' 
               AND relationship_status = 'accepted'",
            user_id        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("查询好友关系失败: {}", e);
            DomainError::DatabaseError { 
                operation: "find_friends_by_user_id".to_string(),
                details: e.to_string()
            }
        })?;
        
        let friendships: Vec<_> = rows.into_iter().map(|row| {
            // 确定friend_id（对方用户ID）
            let friend_id = if row.from_user_id == user_id {
                row.to_user_id
            } else {
                row.from_user_id
            };
            
            // 确保user1_id < user2_id
            let (user1_id, user2_id) = if user_id < friend_id {
                (user_id, friend_id)
            } else {
                (friend_id, user_id)
            };
            
            Friendship {
                friendship_id: format!("{}_{}", user1_id, user2_id),
                user1_id,
                user2_id,
                created_at: row.created_at,
            }
        }).collect();
        
        debug!("找到{}个好友关系", friendships.len());
        Ok(friendships)
    }

    /// 发送好友请求
    async fn send_friend_request(&self, requester_id: i64, target_id: i64, message: Option<&str>) -> DomainResult<FriendRequest> {
        debug!("发送好友请求: 发送者={}, 目标={}", requester_id, target_id);
        
        // 检查是否是自己
        if requester_id == target_id {
            return Err(DomainError::ValidationError {
                field: "target_id".to_string(),
                reason: "不能向自己发送好友请求".to_string(),
            });
        }
        
        let mut tx = self.pool.begin().await.map_err(|e| {
            error!("开始事务失败: {}", e);
            DomainError::DatabaseError { 
                operation: "begin_transaction".to_string(),
                details: e.to_string()
            }
        })?;
        
        // 检查目标用户是否存在
        let target_exists = sqlx::query!(
            "SELECT id FROM users WHERE id = $1",
            target_id        )
        .fetch_optional(&mut *tx)
        .await
        .map_err(|e| {
            error!("检查目标用户失败: {}", e);
            DomainError::DatabaseError { 
                operation: "check_target_user".to_string(),
                details: e.to_string()
            }
        })?;
        
        if target_exists.is_none() {
            return Err(DomainError::NotFound {
                resource: "目标用户".to_string(),
            });
        }
        
        // 检查是否已经是好友或有待处理的请求
        let existing_relation = sqlx::query!(
            "SELECT relationship_status::text as \"relationship_status!\" FROM user_relationships 
             WHERE ((from_user_id = $1 AND to_user_id = $2) OR (from_user_id = $2 AND to_user_id = $1))
               AND relationship_type = 'friend'",
            requester_id,
            target_id        )
        .fetch_optional(&mut *tx)
        .await
        .map_err(|e| {
            error!("检查现有关系失败: {}", e);
            DomainError::DatabaseError { 
                operation: "check_existing_relation".to_string(),
                details: e.to_string()
            }
        })?;
        
        if let Some(relation) = existing_relation {
            match relation.relationship_status.as_str() {
                "accepted" => return Err(DomainError::ValidationError {
                    field: "target_id".to_string(),
                    reason: "已经是好友关系".to_string(),
                }),
                "pending" => return Err(DomainError::ValidationError {
                    field: "target_id".to_string(),
                    reason: "已有待处理的好友请求".to_string(),
                }),
                "blocked" => return Err(DomainError::ValidationError {
                    field: "target_id".to_string(),
                    reason: "用户已被屏蔽".to_string(),
                }),
                _ => {}
            }
        }
        
        // 插入好友请求
        sqlx::query!(
            "INSERT INTO user_relationships (from_user_id, to_user_id, relationship_type, relationship_status, message)
             VALUES ($1, $2, 'friend', 'pending', $3)",
            requester_id,
            target_id,
            message
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("插入好友请求失败: {}", e);
            DomainError::DatabaseError { 
                operation: "insert_friend_request".to_string(),
                details: e.to_string()
            }
        })?;
        
        // 查询请求详情
        let row = sqlx::query!(
            "SELECT relationship_id, created_at FROM user_relationships 
             WHERE from_user_id = $1 AND to_user_id = $2 AND relationship_type = 'friend' AND relationship_status = 'pending'",
            requester_id,
            target_id        )
        .fetch_one(&mut *tx)
        .await
        .map_err(|e| {
            error!("查询好友请求失败: {}", e);
            DomainError::DatabaseError { 
                operation: "fetch_friend_request".to_string(),
                details: e.to_string()
            }
        })?;
        
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {}", e);
            DomainError::DatabaseError { 
                operation: "commit_transaction".to_string(),
                details: e.to_string()
            }
        })?;
        
        Ok(FriendRequest {
            request_id: row.relationship_id,
            requester_id,
            target_id,
            message: message.map(|s| s.to_string()),
            status: crate::domain::entities::friend::FriendRequestStatus::Pending,
            created_at: row.created_at,
            updated_at: row.created_at,
            expires_at: row.created_at + chrono::Duration::days(30),
        })
    }

    /// 接受好友请求
    async fn accept_friend_request(&self, request_id: &str, target_id: i64) -> DomainResult<Friendship> {
        debug!("接受好友请求: request_id={}, user={}", request_id, target_id);
        
        let row = sqlx::query!(
            "UPDATE user_relationships 
             SET relationship_status = 'accepted', updated_at = NOW()
             WHERE relationship_id = $1 AND to_user_id = $2 AND relationship_type = 'friend' AND relationship_status = 'pending'
             RETURNING from_user_id, to_user_id, created_at",
            request_id,
            target_id        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("更新好友请求失败: {}", e);
            DomainError::DatabaseError { 
                operation: "accept_friend_request".to_string(),
                details: e.to_string()
            }
        })?;
        
        // 确保 user1_id < user2_id
        let (user1, user2) = if row.from_user_id < row.to_user_id { (row.from_user_id, row.to_user_id) } else { (row.to_user_id, row.from_user_id) };
        Ok(Friendship {
            friendship_id: format!("{}_{}", user1, user2),
            user1_id: user1,
            user2_id: user2,
            created_at: row.created_at,
        })
    }
     
    /// 拒绝好友请求
    async fn reject_friend_request(&self, request_id: &str, target_id: i64) -> DomainResult<()> {
        debug!("拒绝好友请求: request_id={}, user={}", request_id, target_id);
        
        sqlx::query!(
            "UPDATE user_relationships 
             SET relationship_status = 'rejected', updated_at = NOW()
             WHERE relationship_id = $1 AND to_user_id = $2 AND relationship_type = 'friend' AND relationship_status = 'pending'",
            request_id,
            target_id        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("拒绝好友请求失败: {}", e);
            DomainError::DatabaseError { 
                operation: "reject_friend_request".to_string(),
                details: e.to_string()
            }
        })?;
        Ok(())
    }
     
    /// 移除好友关系
    async fn remove_friendship(&self, user_id: i64, friend_id: i64) -> DomainResult<()> {
        debug!("移除好友关系: {} <-> {}", user_id, friend_id);
        sqlx::query!(
            "UPDATE user_relationships 
             SET is_deleted = true, updated_at = NOW()
             WHERE ((from_user_id = $1 AND to_user_id = $2) OR (from_user_id = $2 AND to_user_id = $1))
               AND relationship_type = 'friend'",
            user_id,
            friend_id        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("移除好友关系失败: {}", e);
            DomainError::DatabaseError { 
                operation: "remove_friendship".to_string(),
                details: e.to_string()
            }
        })?;
        Ok(())
    }

    /// 获取好友列表
    async fn get_friends(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendProfile>> {
        debug!("获取好友列表: user_id={}, cursor={}, limit={}", user_id, cursor, limit);
        
        let rows = sqlx::query!(
            r#"
            SELECT 
                r.id,
                r.relationship_id,
                u.id as "user_id!",
                u.nickname as "nickname!",
                u.avatar_url,
                u.updated_at as "last_active_at!",
                r.created_at as friendship_created_at
            FROM user_relationships r
            JOIN users u ON (CASE WHEN r.from_user_id = $1 THEN r.to_user_id ELSE r.from_user_id END) = u.id
            WHERE (r.from_user_id = $1 OR r.to_user_id = $1)
              AND r.relationship_type = 'friend'
              AND r.relationship_status = 'accepted'
              AND r.is_deleted = false
              AND ($2 = 0 OR r.id < $2)
            ORDER BY r.id DESC
            LIMIT $3
            "#,
            user_id,
            cursor as i32,
            limit as i64
        )
            .fetch_all(&self.pool)
            .await
            .map_err(|e| {
                error!("查询好友列表失败: {}", e);
                DomainError::DatabaseError { 
                    operation: "get_friends".to_string(),
                    details: e.to_string()
                }
            })?;
        
        let profiles = rows.into_iter().map(|row| FriendProfile {
            user_id: row.user_id,
            nickname: row.nickname,
            avatar_url: row.avatar_url,
            last_active_at: Some(row.last_active_at),
            friendship_created_at: row.friendship_created_at,
            is_online: false,
            relationship_id: row.relationship_id,
            relationship_row_id: row.id,
            block_reason: None,
        }).collect();
        
        Ok(profiles)
    }
 
    /// 关注用户
    async fn follow(&self, follower_id: i64, target_id: i64) -> DomainResult<String> {
        debug!("关注用户: {} -> {}", follower_id, target_id);
        let relationship_id = sqlx::query_scalar!(
            r#"
            INSERT INTO user_relationships (from_user_id, to_user_id, relationship_type, relationship_status, is_deleted, created_at, updated_at)
            VALUES ($1, $2, 'follow', 'active', false, NOW(), NOW())
            ON CONFLICT (from_user_id, to_user_id, relationship_type)
            DO UPDATE SET relationship_status = 'active', is_deleted = false, updated_at = NOW()
            RETURNING relationship_id
            "#,
            follower_id,
            target_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("关注失败: {}", e);
            DomainError::DatabaseError { operation: "follow".to_string(), details: e.to_string() }
        })?;
        Ok(relationship_id)
    }

    /// 取消关注
    async fn unfollow(&self, follower_id: i64, target_id: i64) -> DomainResult<()> {
        debug!("取消关注: {} -> {}", follower_id, target_id);
        sqlx::query!(
            r#"UPDATE user_relationships SET is_deleted = true, updated_at = NOW()
               WHERE from_user_id = $1 AND to_user_id = $2 AND relationship_type = 'follow' AND is_deleted = false"#,
            follower_id,
            target_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| {
            error!("取消关注失败: {}", e);
            DomainError::DatabaseError { operation: "unfollow".to_string(), details: e.to_string() }
        })?;
        Ok(())
    }

    /// 拉黑用户
    async fn block_user(&self, blocker_id: i64, blocked_id: i64, reason: Option<&str>) -> DomainResult<String> {
        debug!("拉黑用户: {} -> {}", blocker_id, blocked_id);
        let mut tx = self.pool.begin().await.map_err(|e| {
            error!("开始事务失败: {}", e);
            DomainError::DatabaseError { operation: "begin_tx_block".to_string(), details: e.to_string() }
        })?;

        // 解除好友关系
        sqlx::query!(
            r#"UPDATE user_relationships SET is_deleted = true, updated_at = NOW()
               WHERE ((from_user_id = $1 AND to_user_id = $2) OR (from_user_id = $2 AND to_user_id = $1))
                 AND relationship_type = 'friend' AND is_deleted = false"#,
            blocker_id,
            blocked_id
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("移除好友关系失败: {}", e);
            DomainError::DatabaseError { operation: "remove_friend_on_block".to_string(), details: e.to_string() }
        })?;

        // 创建/激活拉黑关系
        let relationship_id = sqlx::query_scalar!(
            r#"INSERT INTO user_relationships (from_user_id, to_user_id, relationship_type, relationship_status, message, is_deleted, created_at, updated_at)
                VALUES ($1, $2, 'block', 'active', $3, false, NOW(), NOW())
                ON CONFLICT (from_user_id, to_user_id, relationship_type)
                DO UPDATE SET relationship_status = 'active', is_deleted = false, message = EXCLUDED.message, updated_at = NOW()
                RETURNING relationship_id"#,
            blocker_id,
            blocked_id,
            reason
        )
        .fetch_one(&mut *tx)
        .await
        .map_err(|e| {
            error!("创建拉黑关系失败: {}", e);
            DomainError::DatabaseError { operation: "block_user".to_string(), details: e.to_string() }
        })?;

        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {}", e);
            DomainError::DatabaseError { operation: "commit_block".to_string(), details: e.to_string() }
        })?;
        Ok(relationship_id)
    }

    /// 获取拉黑列表
    async fn get_blocked_users(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendProfile>> {
        debug!("获取拉黑列表: user_id={}, cursor={}, limit={}", user_id, cursor, limit);
        let rows = sqlx::query!(
            r#"SELECT r.id, r.relationship_id, r.message as block_reason, u.id as "user_id!", u.nickname as "nickname!", u.avatar_url, u.updated_at as "last_active_at!", r.created_at as friendship_created_at
                FROM user_relationships r
                JOIN users u ON u.id = r.to_user_id
                WHERE r.from_user_id = $1 AND r.relationship_type = 'block' AND r.is_deleted = false
                  AND ($2 = 0 OR r.id < $2)
                ORDER BY r.id DESC
                LIMIT $3"#,
            user_id,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("查询拉黑列表失败: {}", e);
            DomainError::DatabaseError { operation: "get_blocked_users".to_string(), details: e.to_string() }
        })?;

        let profiles = rows.into_iter().map(|row| FriendProfile {
            user_id: row.user_id,
            nickname: row.nickname,
            avatar_url: row.avatar_url,
            last_active_at: Some(row.last_active_at),
            friendship_created_at: row.friendship_created_at,
            is_online: false,
            relationship_id: row.relationship_id,
            relationship_row_id: row.id,
            block_reason: row.block_reason,
        }).collect();
        Ok(profiles)
    }

    /// 获取收到的好友请求
    async fn get_received_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendRequest>> {
        debug!("获取收到的好友请求: user_id={}, cursor={}, limit={}", user_id, cursor, limit);
        // 查询收到的好友请求（to_user_id = user_id 且状态为 pending）
        let rows = sqlx::query!(
            "SELECT ur.relationship_id, ur.from_user_id as requester_id, ur.message, 
                    ur.created_at, ur.updated_at
             FROM user_relationships ur
             WHERE ur.to_user_id = $1 
               AND ur.relationship_type = 'friend' 
               AND ur.relationship_status = 'pending'
               AND ur.is_deleted = false
               AND ($2 = 0 OR ur.id < $2)
             ORDER BY ur.relationship_id ASC
             LIMIT $3",
            user_id,
            cursor as i32,
            limit as i64
        )
            .fetch_all(&self.pool)
            .await
            .map_err(|e| {
                error!("查询好友请求失败: {}", e);
                DomainError::DatabaseError { 
                    operation: "get_received_friend_requests".to_string(),
                    details: e.to_string()
                }
            })?;
        
        let requests: Vec<FriendRequest> = rows.into_iter().map(|row| {
            let created_at: DateTime<Utc> = row.created_at;
            let expires_at = created_at + chrono::Duration::days(30);
            
            FriendRequest {
                request_id: row.relationship_id.to_string(),
                requester_id: row.requester_id,
                target_id: user_id,
                message: row.message,
                status: FriendRequestStatus::Pending,
                created_at,
                updated_at: row.updated_at,
                expires_at,
            }
        }).collect();
        
        debug!("获取到{}个好友请求", requests.len());
        Ok(requests)
    }
    
    /// 获取发送的好友请求
    async fn get_sent_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendRequest>> {
        debug!("获取发送的好友请求: user_id={}, cursor={}, limit={}", user_id, cursor, limit);
        
        let rows = sqlx::query!(
            r#"
            SELECT 
                r.id as relationship_id,
                r.from_user_id as requester_id,
                r.to_user_id as target_id,
                COALESCE(r.message, '') as message,
                r.created_at,
                r.updated_at
            FROM user_relationships r
            WHERE r.from_user_id = $1 
              AND r.relationship_type = 'friend'
              AND r.relationship_status = 'pending'
              AND ($2 = 0 OR r.id > $2)
            ORDER BY r.created_at DESC
            LIMIT $3
            "#,
            user_id,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("获取发送的好友请求失败: {:?}", e);
            DomainError::DatabaseError {
                operation: "get_sent_friend_requests".to_string(),
                details: e.to_string()
            }
        })?;
        
        let requests: Vec<FriendRequest> = rows.into_iter().map(|row| {
            let created_at: DateTime<Utc> = row.created_at;
            let expires_at = created_at + chrono::Duration::days(30);
            FriendRequest {
                request_id: row.relationship_id.to_string(),
                requester_id: row.requester_id,
                target_id: row.target_id,
                message: row.message,
                created_at,
                updated_at: row.updated_at,
                expires_at,
                status: FriendRequestStatus::Pending,
            }
        }).collect();
        
        debug!("获取到{}个发送的好友请求", requests.len());
        Ok(requests)
    }
    
    /// 检查好友关系
    async fn check_friendship(&self, user_id: i64, friend_id: i64) -> DomainResult<FriendshipStatus> {
        debug!("检查好友关系: user_id={}, friend_id={}", user_id, friend_id);
        
        if user_id == friend_id {
            return Ok(FriendshipStatus::NotFriends); // 不能与自己成为好友
        }
        
        let row = sqlx::query!(
            r#"
            SELECT relationship_status::text as status, from_user_id, to_user_id
            FROM user_relationships
            WHERE ((from_user_id = $1 AND to_user_id = $2) OR (from_user_id = $2 AND to_user_id = $1))
              AND relationship_type = 'friend'
            ORDER BY created_at DESC
            LIMIT 1
            "#,
            user_id, friend_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::DatabaseError {
            operation: "check_friendship".to_string(),
            details: e.to_string()
        })?;
        
        match row {
            Some(row) => {
                match row.status.as_deref().unwrap_or("") {
                    "accepted" => Ok(FriendshipStatus::Friends),
                    "pending" => {
                        if row.from_user_id == user_id {
                            Ok(FriendshipStatus::RequestSent)
                        } else {
                            Ok(FriendshipStatus::RequestReceived)
                        }
                    },
                    "rejected" => Ok(FriendshipStatus::NotFriends),
                    _ => Ok(FriendshipStatus::NotFriends),
                }
            },
            None => {
                // 检查是否被拉黑
                let blocked = sqlx::query_scalar!(
                    "SELECT EXISTS(
                        SELECT 1 FROM user_relationships 
                     WHERE ((from_user_id = $1 AND to_user_id = $2) OR (from_user_id = $2 AND to_user_id = $1))
                          AND relationship_type = 'block' AND relationship_status = 'active'
                    ) as \"exists!\"",
                    user_id, friend_id
                )
                .fetch_one(&self.pool)
                .await
                .map_err(|e| DomainError::DatabaseError {
                    operation: "check_friendship_block".to_string(),
                    details: e.to_string()
                })?;
                
                if blocked {
                    Ok(FriendshipStatus::Blocked)
                } else {
                    Ok(FriendshipStatus::NotFriends)
                }
            }
        }
    }
    
    /// 获取待处理请求数量
    async fn get_pending_requests_count(&self, user_id: i64) -> DomainResult<i64> {
        debug!("获取待处理请求数量: user_id={}", user_id);
        
        let count = sqlx::query_scalar!(
            "SELECT COUNT(*) FROM user_relationships 
             WHERE to_user_id = $1 AND relationship_type = 'friend' AND relationship_status = 'pending'",
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::DatabaseError {
            operation: "get_pending_requests_count".to_string(),
            details: e.to_string()
        })?;
        
        Ok(count.unwrap_or(0))
    }
}