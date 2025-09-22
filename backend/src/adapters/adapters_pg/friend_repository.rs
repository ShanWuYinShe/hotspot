/// PostgreSQL好友关系仓储实现
/// 
/// 功能职责：
/// - 实现好友关系的数据库持久化操作
/// - 处理好友请求的创建、查询和状态更新
/// - 管理好友关系的建立和删除
/// - 提供好友列表和好友请求列表的分页查询
use async_trait::async_trait;
use chrono::{DateTime, Utc};
use sqlx::{PgPool, Row};
use tracing::{debug, error, warn};

use crate::domain::entities::friend::{
    Friendship, FriendRequest, FriendProfile, FriendRequestStatus, DbFriendshipStatus
};
use crate::domain::errors::{DomainError, DomainResult};
use crate::domain::values::user_id::UserId;
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
    async fn find_friends_by_user_id(&self, user_id: &UserId) -> DomainResult<Vec<Friendship>> {
        debug!("查找用户好友关系: user_id={}", user_id.value());
        
        let rows = sqlx::query!(
            "SELECT from_user_id, to_user_id, created_at
             FROM user_relationships 
             WHERE (from_user_id = $1 OR to_user_id = $1) 
               AND relationship_type = 'friend' 
               AND relationship_status = 'accepted'",
            user_id.value()
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("查询好友关系失败: {}", e);
            DomainError::DatabaseError { 
                operation: "find_friends_by_user_id".to_string(),
                details: e.to_string()
            }
        })?;
        
        let friendships = rows.into_iter().map(|row| {
            // 确定friend_id（对方用户ID）
            let friend_id = if row.from_user_id == user_id.value() {
                row.to_user_id
            } else {
                row.from_user_id
            };
            
            // 确保user1_id < user2_id
            let (user1_id, user2_id) = if user_id.value() < friend_id {
                (user_id.value(), friend_id)
            } else {
                (friend_id, user_id.value())
            };
            
            Friendship {
                friendship_id: format!("{}_{}", user1_id, user2_id),
                user1_id,
                user2_id,
                created_at: row.created_at.unwrap_or_else(|| Utc::now()),
            }
        }).collect();
        
        debug!("找到{}个好友关系", friendships.len());
        Ok(friendships)
    }

    /// 发送好友请求
    async fn send_friend_request(&self, requester_id: &UserId, target_id: &UserId, message: Option<String>) -> DomainResult<()> {
        debug!("发送好友请求: 发送者={}, 目标={}", requester_id.value(), target_id.value());
        
        // 检查是否是自己
        if requester_id.value() == target_id.value() {
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
            target_id.value()
        )
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
            requester_id.value(),
            target_id.value()
        )
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
            "INSERT INTO user_relationships (from_user_id, to_user_id, relationship_type, relationship_status, message, created_at, updated_at)
             VALUES ($1, $2, 'friend', 'pending', $3, NOW(), NOW())",
            requester_id.value(),
            target_id.value(),
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
        
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {}", e);
            DomainError::DatabaseError { 
                operation: "commit_transaction".to_string(),
                details: e.to_string()
            }
        })?;
        
        debug!("好友请求发送成功");
        Ok(())
    }

    /// 接受好友请求
    async fn accept_friend_request(&self, requester_id: &UserId, target_id: &UserId) -> DomainResult<()> {
        debug!("接受好友请求: 请求者={}, 目标={}", requester_id.value(), target_id.value());
        
        let mut tx = self.pool.begin().await.map_err(|e| {
            error!("开始事务失败: {}", e);
            DomainError::DatabaseError { 
                operation: "begin_transaction".to_string(),
                details: e.to_string()
            }
        })?;
        
        // 检查是否有待处理的好友请求
        let request = sqlx::query!(
            "SELECT relationship_id FROM user_relationships 
             WHERE from_user_id = $1 AND to_user_id = $2 
               AND relationship_type = 'friend' AND relationship_status = 'pending'",
            requester_id.value(),
            target_id.value()
        )
        .fetch_optional(&mut *tx)
        .await
        .map_err(|e| {
            error!("查询好友请求失败: {}", e);
            DomainError::DatabaseError { 
                operation: "find_friend_request".to_string(),
                details: e.to_string()
            }
        })?;
        
        if request.is_none() {
            return Err(DomainError::NotFound {
                resource: "好友请求".to_string(),
            });
        }
        
        // 更新请求状态为已接受
        sqlx::query!(
            "UPDATE user_relationships 
             SET relationship_status = 'accepted', updated_at = NOW()
             WHERE from_user_id = $1 AND to_user_id = $2 
               AND relationship_type = 'friend' AND relationship_status = 'pending'",
            requester_id.value(),
            target_id.value()
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("更新好友请求状态失败: {}", e);
            DomainError::DatabaseError { 
                operation: "accept_friend_request".to_string(),
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
        
        debug!("好友请求接受成功");
        Ok(())
    }

    /// 获取好友列表
    async fn get_friends(&self, user_id: &UserId, cursor: Option<i64>, limit: i32) -> DomainResult<Vec<FriendProfile>> {
        debug!("获取好友列表: user_id={}, cursor={:?}, limit={}", 
               user_id.value(), cursor, limit);
        
        let cursor_condition = cursor.map(|c| format!("AND u.id > {}", c)).unwrap_or_default();
        
        let query = format!(
            "SELECT u.id as user_id, u.nickname, u.avatar_url, u.updated_at as last_active_at,
                    ur.created_at as friendship_created_at
             FROM user_relationships ur
             JOIN users u ON (
                 CASE 
                     WHEN ur.from_user_id = $1 THEN u.id = ur.to_user_id
                     ELSE u.id = ur.from_user_id
                 END
             )
             WHERE (ur.from_user_id = $1 OR ur.to_user_id = $1)
               AND ur.relationship_type = 'friend' 
               AND ur.relationship_status = 'accepted'
               {}
             ORDER BY u.id ASC
             LIMIT $2",
            cursor_condition
        );
        
        let rows = sqlx::query(&query)
            .bind(user_id.value())
            .bind(limit)
            .fetch_all(&self.pool)
            .await
            .map_err(|e| {
                error!("查询好友列表失败: {}", e);
                DomainError::DatabaseError { 
                    operation: "get_friends".to_string(),
                    details: e.to_string()
                }
            })?;
        
        let friends = rows.into_iter().map(|row| {
            FriendProfile {
                user_id: row.get("user_id"),
                nickname: row.get("nickname"),
                avatar_url: row.get("avatar_url"),
                last_active_at: row.get("last_active_at"),
                friendship_created_at: row.get("friendship_created_at"),
                is_online: false, // TODO: 实现在线状态检查
            }
        }).collect();
        
        debug!("获取到{}个好友", friends.len());
        Ok(friends)
    }

    /// 获取收到的好友请求
    async fn get_received_friend_requests(&self, user_id: &UserId, cursor: Option<i64>, limit: i32) -> DomainResult<Vec<FriendRequest>> {
        debug!("获取收到的好友请求: user_id={}, cursor={:?}, limit={}", 
               user_id.value(), cursor, limit);
        
        let cursor_condition = cursor.map(|c| format!("AND ur.relationship_id > {}", c)).unwrap_or_default();
        
        let query = format!(
            "SELECT ur.relationship_id, ur.from_user_id as requester_id, ur.message, 
                    ur.created_at, ur.updated_at
             FROM user_relationships ur
             WHERE ur.to_user_id = $1 
               AND ur.relationship_type = 'friend' 
               AND ur.relationship_status = 'pending'
               {}
             ORDER BY ur.relationship_id ASC
             LIMIT $2",
            cursor_condition
        );
        
        let rows = sqlx::query(&query)
            .bind(user_id.value())
            .bind(limit)
            .fetch_all(&self.pool)
            .await
            .map_err(|e| {
                error!("查询好友请求失败: {}", e);
                DomainError::DatabaseError { 
                    operation: "get_received_friend_requests".to_string(),
                    details: e.to_string()
                }
            })?;
        
        let requests = rows.into_iter().map(|row| {
            let created_at: DateTime<Utc> = row.get("created_at");
            let expires_at = created_at + chrono::Duration::days(30);
            
            FriendRequest {
                request_id: row.get::<i64, _>("relationship_id").to_string(),
                requester_id: row.get("requester_id"),
                target_id: user_id.value(),
                message: row.get("message"),
                status: FriendRequestStatus::Pending,
                created_at,
                updated_at: row.get("updated_at"),
                expires_at,
            }
        }).collect();
        
        debug!("获取到{}个好友请求", requests.len());
        Ok(requests)
    }
}