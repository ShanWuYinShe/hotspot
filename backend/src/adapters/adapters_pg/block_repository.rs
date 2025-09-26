/// PostgreSQL屏蔽仓储实现
/// 
/// 功能职责：
/// - 实现用户屏蔽关系的数据库CRUD操作
/// - 提供屏蔽设置管理功能
/// - 支持屏蔽状态查询和验证
/// - 管理屏蔽关系对消息和通知的影响
use async_trait::async_trait;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    domain::{DomainResult, DomainError, Block, BlockSettings, BlockStats, BlockType, BlockReason, UserId},
    ports::repository_ports::BlockRepository,
};

/// 屏蔽关系数据库行结构
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockRow {
    pub block_id: Uuid,
    pub blocker_id: i64,
    pub blocked_id: i64,
    // block_type 字段在数据库中不存在，移除
    pub reason: Option<String>, // 对应BlockReason，可能为空
    pub created_at: DateTime<Utc>,
}

/// 屏蔽设置数据库行结构
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockSettingsRow {
    pub user_id: i64,
    pub auto_block_spam: bool,
    pub auto_block_inappropriate: bool,
    pub block_strangers: bool,
    pub allow_friend_requests: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// PostgreSQL屏蔽仓储
#[derive(Clone)]
pub struct PgBlockRepository {
    pool: PgPool,
}

impl PgBlockRepository {
    /// 创建新的屏蔽仓储实例
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 将数据库行转换为屏蔽关系领域模型
    fn row_to_block(&self, row: BlockRow) -> DomainResult<Block> {
        // 数据库中没有 block_type 字段，默认为完全屏蔽
        let block_type = BlockType::Complete;

        let reason = match row.reason {
            Some(reason_str) if !reason_str.is_empty() => {
                match reason_str.as_str() {
                    "harassment" => Some(BlockReason::Harassment),
                    "spam" => Some(BlockReason::Spam),
                    "inappropriate_content" => Some(BlockReason::InappropriateContent),
                    "other" => Some(BlockReason::Other),
                    _ => Some(BlockReason::Other), // 默认为其他
                }
            }
            _ => None,
        };

        Ok(Block {
            block_id: row.block_id,
            blocker_id: row.blocker_id,
            blocked_id: row.blocked_id,
            block_type,
            reason,
            created_at: row.created_at,
        })
    }

    /// 将数据库行转换为屏蔽设置领域模型
    fn row_to_block_settings(&self, row: BlockSettingsRow) -> DomainResult<BlockSettings> {
        Ok(BlockSettings {
            user_id: row.user_id,
            block_stranger_messages: row.auto_block_spam,
            block_stranger_notifications: row.auto_block_inappropriate,
            block_group_invitations: row.block_strangers,
            block_friend_requests: !row.allow_friend_requests,
            blocked_keywords: Vec::new(), // 数据库中没有此字段
            updated_at: row.updated_at,
        })
    }
}

#[async_trait]
impl BlockRepository for PgBlockRepository {
    /// 屏蔽用户
    async fn block_user(
        &self,
        blocker_id: i64,
        blocked_id: i64,
        block_type: BlockType,
        reason: BlockReason,
    ) -> DomainResult<Block> {
        // 检查是否已存在屏蔽关系
        let existing = sqlx::query!(
            "SELECT block_id FROM blocks 
             WHERE blocker_id = $1 AND blocked_id = $2",
            blocker_id,
            blocked_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;

        if existing.is_some() {
            return Err(DomainError::BusinessRuleViolation {
            rule: "用户已被屏蔽".to_string(),
            });
        }

        let block_id = Uuid::new_v4();
        let now = Utc::now();
        // block_type 字段在数据库中不存在，移除相关代码
        let reason_str = match reason {
            BlockReason::Harassment => "harassment",
            BlockReason::Spam => "spam",
            BlockReason::InappropriateContent => "inappropriate_content",
            BlockReason::Other => "other",
        };

        let mut tx = self.pool.begin().await
            .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;

        // 创建屏蔽关系
        sqlx::query!(
            "INSERT INTO blocks (block_id, blocker_id, blocked_id, reason, created_at)
             VALUES ($1, $2, $3, $4, $5)",
            block_id,
            blocker_id,
            blocked_id,
            reason_str,
            now
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;

        // 如果是完全屏蔽，同时移除好友关系
        if matches!(block_type, BlockType::Complete) {
            sqlx::query!(
                "DELETE FROM friendships 
                 WHERE (requester_id = $1 AND addressee_id = $2) OR (requester_id = $2 AND addressee_id = $1)",
                blocker_id.min(blocked_id),
                blocker_id.max(blocked_id)
            )
            .execute(&mut *tx)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;
        }

        tx.commit().await
            .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;

        Ok(Block {
            block_id,
            blocker_id,
            blocked_id,
            block_type,
            reason: Some(reason),
            created_at: now,
        })
    }

    /// 取消屏蔽用户
    async fn unblock_user(&self, blocker_id: i64, blocked_id: i64) -> DomainResult<()> {
        let result = sqlx::query!(
            "DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2",
            blocker_id,
            blocked_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;

        if result.rows_affected() == 0 {
            return Err(DomainError::BusinessRuleViolation {
            rule: "屏蔽关系不存在".to_string(),
            });
        }

        Ok(())
    }

    /// 检查是否被屏蔽
    async fn is_blocked(&self, blocker_id: i64, blocked_id: i64) -> DomainResult<bool> {
        let count = sqlx::query_scalar!(
            "SELECT COUNT(*)::bigint as \"count!\" FROM blocks WHERE blocker_id = $1 AND blocked_id = $2",
            blocker_id,
            blocked_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;

        Ok(count > 0)
    }

    /// 获取屏蔽关系详情
    async fn get_block_details(&self, blocker_id: i64, blocked_id: i64) -> DomainResult<Option<Block>> {
        let block_row = sqlx::query_as!(
            BlockRow,
            "SELECT block_id, blocker_id, blocked_id, reason, created_at
             FROM blocks
             WHERE blocker_id = $1 AND blocked_id = $2",
            blocker_id,
            blocked_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;

        if let Some(row) = block_row {
            Ok(Some(self.row_to_block(row)?))
        } else {
            Ok(None)
        }
    }

    /// 获取用户屏蔽的用户列表
    async fn get_blocked_users(&self, blocker_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<Block>> {
        let blocks = sqlx::query_as!(
            BlockRow,
            "SELECT block_id, blocker_id, blocked_id, reason, created_at
             FROM blocks
             WHERE blocker_id = $1
               AND ($2 = 0 OR blocked_id > $2)
             ORDER BY blocked_id ASC
             LIMIT $3",
            blocker_id,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;

        let mut block_list = Vec::new();
        for row in blocks {
            block_list.push(self.row_to_block(row)?);
        }

        Ok(block_list)
    }

    /// 获取屏蔽当前用户的用户列表
    async fn get_blocking_users(&self, blocked_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<Block>> {
        let blocks = sqlx::query_as!(
            BlockRow,
            "SELECT block_id, blocker_id, blocked_id, reason, created_at
             FROM blocks
             WHERE blocked_id = $1
               AND ($2 = 0 OR blocker_id > $2)
             ORDER BY blocker_id ASC
             LIMIT $3",
            blocked_id,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;

        let mut block_list = Vec::new();
        for row in blocks {
            block_list.push(self.row_to_block(row)?);
        }

        Ok(block_list)
    }

    /// 获取用户屏蔽设置
    async fn get_block_settings(&self, user_id: i64) -> DomainResult<Option<BlockSettings>> {
        let settings_row = sqlx::query_as!(
            BlockSettingsRow,
            "SELECT user_id, auto_block_spam, auto_block_inappropriate, block_strangers, allow_friend_requests, 
                    created_at, updated_at
             FROM block_settings
             WHERE user_id = $1",
            user_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;

        if let Some(row) = settings_row {
            Ok(Some(self.row_to_block_settings(row)?))
        } else {
            Ok(None)
        }
    }

    /// 更新用户屏蔽设置
    async fn update_block_settings(&self, user_id: i64, settings: &BlockSettings) -> DomainResult<()> {
        let now = Utc::now();

        sqlx::query!(
            "INSERT INTO block_settings 
             (user_id, auto_block_spam, auto_block_inappropriate, block_strangers, allow_friend_requests, created_at, updated_at)
             VALUES ($1, $2, $3, $4, $5, $6, $7)
             ON CONFLICT (user_id) DO UPDATE SET
                 auto_block_spam = EXCLUDED.auto_block_spam,
                 auto_block_inappropriate = EXCLUDED.auto_block_inappropriate,
                 block_strangers = EXCLUDED.block_strangers,
                 allow_friend_requests = EXCLUDED.allow_friend_requests,
                 updated_at = EXCLUDED.updated_at",
            user_id,
            settings.block_stranger_messages,
            settings.block_stranger_notifications,
            settings.block_group_invitations,
            !settings.block_friend_requests,
            now,
            now
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;

        Ok(())
    }

    /// 获取屏蔽统计信息
    async fn get_block_stats(&self, user_id: i64) -> DomainResult<BlockStats> {
        let blocked_count = sqlx::query_scalar!(
            "SELECT COUNT(*)::bigint as \"count!\" FROM blocks WHERE blocker_id = $1",
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;

        let blocking_count = sqlx::query_scalar!(
            "SELECT COUNT(*)::bigint as \"count!\" FROM blocks WHERE blocked_id = $1",
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;

        Ok(BlockStats {
            user_id,
            blocked_users_count: blocked_count,
            blocked_by_count: blocking_count,
            updated_at: Utc::now(),
        })
    }

    /// 批量检查屏蔽状态
    async fn batch_check_blocked(&self, blocker_id: i64, user_ids: &[i64]) -> DomainResult<Vec<(i64, bool)>> {
        if user_ids.is_empty() {
            return Ok(Vec::new());
        }

        let blocked_ids = sqlx::query_scalar!(
            "SELECT blocked_id FROM blocks WHERE blocker_id = $1 AND blocked_id = ANY($2)",
            blocker_id,
            user_ids
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: e.to_string(),
        })?;

        // 将结果转换为 (user_id, is_blocked) 格式
        let mut result = Vec::new();
        for &user_id in user_ids {
            let is_blocked = blocked_ids.contains(&user_id);
            result.push((user_id, is_blocked));
        }

        Ok(result)
    }

    // === 缺失的 trait 方法实现 ===
    
    async fn save(&self, block: &Block) -> DomainResult<()> {
        let reason_str = block.reason.as_ref().map(|r| match r {
            BlockReason::Harassment => "harassment",
            BlockReason::Spam => "spam",
            BlockReason::InappropriateContent => "inappropriate_content",
            BlockReason::Other => "other",
        });

        sqlx::query!(
            "INSERT INTO blocks (block_id, blocker_id, blocked_id, reason, created_at)
             VALUES ($1, $2, $3, $4, $5)
             ON CONFLICT (blocker_id, blocked_id) DO UPDATE SET reason = EXCLUDED.reason",
            block.block_id,
            block.blocker_id,
            block.blocked_id,
            reason_str,
            block.created_at
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: e.to_string() })?;

        Ok(())
    }

    async fn find_blocks_by_user_id(&self, user_id: UserId) -> DomainResult<Vec<Block>> {
        let rows = sqlx::query_as!(
            BlockRow,
            r#"
            SELECT block_id, blocker_id, blocked_id, reason, created_at
            FROM blocks
            WHERE blocker_id = $1
            ORDER BY created_at DESC
            "#,
            user_id.value()
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: e.to_string() })?;

        let mut blocks = Vec::new();
        for row in rows {
            blocks.push(self.row_to_block(row)?);
        }
        Ok(blocks)
    }
}
