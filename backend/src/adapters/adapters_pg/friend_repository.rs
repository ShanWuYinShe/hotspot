/// PostgreSQL好友仓储实现
/// 
/// 功能职责：
/// - 实现好友关系的数据库CRUD操作
/// - 提供好友请求管理功能
/// - 支持好友关系查询和状态检查
/// - 管理好友请求的生命周期
use async_trait::async_trait;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    domain::{DomainResult, DomainError, FriendRequest, Friendship, FriendProfile, FriendshipStatus, FriendRequestStatus, DbFriendshipStatus, UserId},
    ports::repository_ports::FriendRepository,
};
use super::query_optimizer::PgQueryOptimizer;

/// 好友请求数据库行结构
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FriendRequestRow {
    pub request_id: Uuid,
    pub requester_id: i64,
    pub target_id: i64,
    pub status: String, // 对应FriendRequestStatus
    pub message: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
}

/// 好友关系数据库行结构
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FriendshipRow {
    pub friendship_id: Uuid,
    pub requester_id: i64,
    pub addressee_id: i64,
    pub status: String, // 对应FriendshipStatus
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

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

    /// 将数据库行转换为好友请求领域模型
    fn row_to_friend_request(&self, row: FriendRequestRow) -> DomainResult<FriendRequest> {
        let status = match row.status.as_str() {
            "pending" => FriendRequestStatus::Pending,
            "accepted" => FriendRequestStatus::Accepted,
            "rejected" => FriendRequestStatus::Rejected,
            "expired" => FriendRequestStatus::Expired,
            _ => return Err(DomainError::BusinessRuleViolation {
                rule: "无效的好友请求状态".to_string(),
            }),
        };

        Ok(FriendRequest {
            request_id: row.request_id,
            requester_id: row.requester_id,
            target_id: row.target_id,
            status,
            message: row.message,
            created_at: row.created_at,
            updated_at: row.updated_at.unwrap_or(row.created_at),
            expires_at: row.created_at + chrono::Duration::days(7), // 7天过期
        })
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
                "SELECT request_id FROM friend_requests 
                 WHERE requester_id = $1 AND target_id = $2 AND status = 'pending'",
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
        let request_id = Uuid::new_v4();
        let now = Utc::now();

        sqlx::query!(
            "INSERT INTO friend_requests (request_id, requester_id, target_id, status, message, created_at)
             VALUES ($1, $2, $3, 'pending', $4, $5)",
            request_id,
            requester_id,
            target_id,
            message,
            now
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

            Ok(FriendRequest {
                request_id,
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
        let request_uuid = Uuid::parse_str(request_id)
            .map_err(|_| DomainError::BusinessRuleViolation {
                rule: "无效的请求ID格式".to_string(),
            })?;

        let mut tx = self.pool.begin().await
            .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        // 更新好友请求状态
        let request = sqlx::query!(
            "UPDATE friend_requests 
             SET status = 'accepted', updated_at = NOW()
             WHERE request_id = $1 AND target_id = $2 AND status = 'pending'
             RETURNING requester_id, target_id",
            request_uuid,
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

        // 创建好友关系
        let friendship_id = Uuid::new_v4();
        let now = Utc::now();

        sqlx::query!(
            "INSERT INTO friendships (friendship_id, requester_id, addressee_id, status, created_at, updated_at)
             VALUES ($1, $2, $3, 'accepted', $4, $5)",
            friendship_id,
            request.requester_id.min(request.target_id),
            request.requester_id.max(request.target_id),
            now,
            now
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        tx.commit().await
            .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        Ok(Friendship {
            friendship_id,
            user1_id: request.requester_id.min(request.target_id),
            user2_id: request.requester_id.max(request.target_id),
            created_at: now,
        })
    }

    /// 拒绝好友请求
    async fn reject_friend_request(&self, request_id: &str, target_id: i64) -> DomainResult<()> {
        let request_uuid = Uuid::parse_str(request_id)
            .map_err(|_| DomainError::BusinessRuleViolation {
                rule: "无效的请求ID格式".to_string(),
            })?;

        let result = sqlx::query!(
            "UPDATE friend_requests 
             SET status = 'rejected', updated_at = NOW()
             WHERE request_id = $1 AND target_id = $2 AND status = 'pending'",
            request_uuid,
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
             WHERE (requester_id = $1 AND addressee_id = $2) OR (requester_id = $2 AND addressee_id = $1)",
            user_id.min(friend_id),
            user_id.max(friend_id)
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
            SELECT u.user_id as "user_id!", u.nickname as "nickname!", u.avatar_url, u.last_active_at as "last_active_at!", f.created_at as "created_at!"
            FROM (
                SELECT f.addressee_id as friend_id, f.created_at
                FROM friendships f
                WHERE f.requester_id = $1
                UNION
                SELECT f.requester_id as friend_id, f.created_at
                FROM friendships f
                WHERE f.addressee_id = $1
            ) f
            JOIN users u ON u.user_id = f.friend_id
            WHERE ($2 = 0 OR u.user_id > $2)
            ORDER BY u.user_id ASC
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
                last_active_at: Some(row.last_active_at),
                friendship_created_at: row.created_at,
                is_online: false, // TODO: 实现在线状态检测
            })
            .collect();

        Ok(friend_profiles)
    }

    /// 获取收到的好友请求
    async fn get_received_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendRequest>> {
        let requests = sqlx::query_as!(
            FriendRequestRow,
            "SELECT request_id, requester_id, target_id, status, message, created_at, updated_at
             FROM friend_requests
             WHERE target_id = $1 AND status = 'pending'
               AND ($2 = 0 OR requester_id > $2)
             ORDER BY requester_id ASC
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

        let mut friend_requests = Vec::new();
        for row in requests {
            friend_requests.push(self.row_to_friend_request(row)?);
        }

        Ok(friend_requests)
    }

    /// 获取发送的好友请求
    async fn get_sent_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendRequest>> {
        let requests = sqlx::query_as!(
            FriendRequestRow,
            "SELECT request_id, requester_id, target_id, status, message, created_at, updated_at
             FROM friend_requests
             WHERE requester_id = $1 AND status = 'pending'
               AND ($2 = 0 OR target_id > $2)
             ORDER BY target_id ASC
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

        let mut friend_requests = Vec::new();
        for row in requests {
            friend_requests.push(self.row_to_friend_request(row)?);
        }

        Ok(friend_requests)
    }

    /// 检查好友关系状态
    async fn check_friendship(&self, user_id: i64, other_user_id: i64) -> DomainResult<FriendshipStatus> {
        // 检查是否是好友
        let friendship: Option<DbFriendshipStatus> = sqlx::query_scalar!(
            "SELECT status as \"status: DbFriendshipStatus\" FROM friendships 
             WHERE (requester_id = $1 AND addressee_id = $2) OR (requester_id = $2 AND addressee_id = $1)",
            user_id.min(other_user_id),
            user_id.max(other_user_id)
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        if let Some(db_status) = friendship {
            return match db_status {
                DbFriendshipStatus::Accepted => Ok(FriendshipStatus::Friends),
                DbFriendshipStatus::Blocked => Ok(FriendshipStatus::Blocked),
                _ => Ok(FriendshipStatus::NotFriends),
            };
        }

        // 检查是否有待处理的好友请求
        let request = sqlx::query!(
            "SELECT requester_id FROM friend_requests 
             WHERE ((requester_id = $1 AND target_id = $2) OR (requester_id = $2 AND target_id = $1))
               AND status = 'pending'",
            user_id,
            other_user_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("数据库错误: {}", e),
        })?;

        if let Some(request) = request {
            if request.requester_id == user_id {
                return Ok(FriendshipStatus::RequestSent);
            } else {
                return Ok(FriendshipStatus::RequestReceived);
            }
        }

        Ok(FriendshipStatus::NotFriends)
    }

    /// 获取待处理的好友请求数量
    async fn get_pending_requests_count(&self, user_id: i64) -> DomainResult<i64> {
        let count = sqlx::query_scalar!(
            "SELECT COUNT(*)::bigint as \"count!\" FROM friend_requests WHERE target_id = $1 AND status = 'pending'",
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
        // TODO: 实现好友关系保存逻辑
        todo!("Implement save method")
    }

    async fn find_friends_by_user_id(&self, user_id: UserId) -> DomainResult<Vec<Friendship>> {
        // TODO: 实现根据用户ID查找好友关系逻辑
        todo!("Implement find_friends_by_user_id method")
    }
}
