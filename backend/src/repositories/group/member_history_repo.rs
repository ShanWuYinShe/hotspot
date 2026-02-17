/// 群组成员操作历史仓库
use chrono::{DateTime, Utc};
use sqlx::PgPool;
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::GroupRole;

use std::fmt;

/// 成员操作类型
#[derive(Debug, Clone, Copy, PartialEq, Eq, sqlx::Type)]
#[sqlx(type_name = "member_action", rename_all = "snake_case")]
pub enum MemberAction {
    Join,
    Leave,
    Removed,
    Banned,
    RoleChanged,
    Muted,
    Unmuted,
}

impl fmt::Display for MemberAction {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let s = match self {
            MemberAction::Join => "join",
            MemberAction::Leave => "leave",
            MemberAction::Removed => "removed",
            MemberAction::Banned => "banned",
            MemberAction::RoleChanged => "role_changed",
            MemberAction::Muted => "muted",
            MemberAction::Unmuted => "unmuted",
        };
        write!(f, "{}", s)
    }
}

/// 创建历史记录的参数
pub struct CreateHistoryParams {
    pub group_id: i64,
    pub user_id: i64,
    pub action: MemberAction,
    pub operator_id: Option<i64>,
    pub old_role: Option<GroupRole>,
    pub new_role: Option<GroupRole>,
    pub reason: Option<String>,
    pub muted_until: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct GroupMemberHistoryRepository {
    pool: PgPool,
}

impl GroupMemberHistoryRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 记录成员加入
    pub async fn record_join(
        &self,
        group_id: i64,
        user_id: i64,
        operator_id: Option<i64>,
    ) -> Result<Uuid> {
        self.create(CreateHistoryParams {
            group_id,
            user_id,
            action: MemberAction::Join,
            operator_id,
            old_role: None,
            new_role: Some(GroupRole::Member),
            reason: None,
            muted_until: None,
        })
        .await
    }

    /// 记录成员离开
    pub async fn record_leave(&self, group_id: i64, user_id: i64) -> Result<Uuid> {
        self.create(CreateHistoryParams {
            group_id,
            user_id,
            action: MemberAction::Leave,
            operator_id: Some(user_id), // 自己操作
            old_role: None,
            new_role: None,
            reason: None,
            muted_until: None,
        })
        .await
    }

    /// 记录成员被移除
    pub async fn record_removed(
        &self,
        group_id: i64,
        user_id: i64,
        operator_id: i64,
        reason: Option<String>,
    ) -> Result<Uuid> {
        self.create(CreateHistoryParams {
            group_id,
            user_id,
            action: MemberAction::Removed,
            operator_id: Some(operator_id),
            old_role: None,
            new_role: None,
            reason,
            muted_until: None,
        })
        .await
    }

    /// 记录成员被封禁
    pub async fn record_banned(
        &self,
        group_id: i64,
        user_id: i64,
        operator_id: i64,
        reason: Option<String>,
    ) -> Result<Uuid> {
        self.create(CreateHistoryParams {
            group_id,
            user_id,
            action: MemberAction::Banned,
            operator_id: Some(operator_id),
            old_role: None,
            new_role: None,
            reason,
            muted_until: None,
        })
        .await
    }

    /// 记录角色变更
    pub async fn record_role_changed(
        &self,
        group_id: i64,
        user_id: i64,
        operator_id: i64,
        old_role: GroupRole,
        new_role: GroupRole,
    ) -> Result<Uuid> {
        self.create(CreateHistoryParams {
            group_id,
            user_id,
            action: MemberAction::RoleChanged,
            operator_id: Some(operator_id),
            old_role: Some(old_role),
            new_role: Some(new_role),
            reason: None,
            muted_until: None,
        })
        .await
    }

    /// 记录成员被禁言
    pub async fn record_muted(
        &self,
        group_id: i64,
        user_id: i64,
        operator_id: i64,
        muted_until: DateTime<Utc>,
    ) -> Result<Uuid> {
        self.create(CreateHistoryParams {
            group_id,
            user_id,
            action: MemberAction::Muted,
            operator_id: Some(operator_id),
            old_role: None,
            new_role: None,
            reason: None,
            muted_until: Some(muted_until),
        })
        .await
    }

    /// 记录成员被解除禁言
    pub async fn record_unmuted(
        &self,
        group_id: i64,
        user_id: i64,
        operator_id: i64,
    ) -> Result<Uuid> {
        self.create(CreateHistoryParams {
            group_id,
            user_id,
            action: MemberAction::Unmuted,
            operator_id: Some(operator_id),
            old_role: None,
            new_role: None,
            reason: None,
            muted_until: None,
        })
        .await
    }

    /// 创建历史记录
    async fn create(&self, params: CreateHistoryParams) -> Result<Uuid> {
        let action_str = params.action.to_string();
        let old_role_str = params.old_role.map(|r| r.to_string());
        let new_role_str = params.new_role.map(|r| r.to_string());

        let history_id = sqlx::query_scalar!(
            r#"
            INSERT INTO group_member_history 
                (group_id, user_id, action, operator_id, old_role, new_role, reason, muted_until)
            VALUES ($1, $2, $3::text::member_action, $4, $5::text::group_role, $6::text::group_role, $7, $8)
            RETURNING history_id
            "#,
            params.group_id,
            params.user_id,
            action_str,
            params.operator_id,
            old_role_str,
            new_role_str,
            params.reason,
            params.muted_until
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(history_id)
    }
}
